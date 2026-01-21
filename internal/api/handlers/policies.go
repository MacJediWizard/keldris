package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// PolicyStore defines the interface for policy persistence operations.
type PolicyStore interface {
	GetPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Policy, error)
	GetPolicyByID(ctx context.Context, id uuid.UUID) (*models.Policy, error)
	CreatePolicy(ctx context.Context, policy *models.Policy) error
	UpdatePolicy(ctx context.Context, policy *models.Policy) error
	DeletePolicy(ctx context.Context, id uuid.UUID) error
	GetSchedulesByPolicyID(ctx context.Context, policyID uuid.UUID) ([]*models.Schedule, error)
	// For apply endpoint
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) error
}

// PoliciesHandler handles policy-related HTTP endpoints.
type PoliciesHandler struct {
	store  PolicyStore
	logger zerolog.Logger
}

// NewPoliciesHandler creates a new PoliciesHandler.
func NewPoliciesHandler(store PolicyStore, logger zerolog.Logger) *PoliciesHandler {
	return &PoliciesHandler{
		store:  store,
		logger: logger.With().Str("component", "policies_handler").Logger(),
	}
}

// RegisterRoutes registers policy routes on the given router group.
func (h *PoliciesHandler) RegisterRoutes(r *gin.RouterGroup) {
	policies := r.Group("/policies")
	{
		policies.GET("", h.List)
		policies.POST("", h.Create)
		policies.GET("/:id", h.Get)
		policies.PUT("/:id", h.Update)
		policies.DELETE("/:id", h.Delete)
		policies.GET("/:id/schedules", h.ListSchedules)
		policies.POST("/:id/apply", h.Apply)
	}
}

// CreatePolicyRequest is the request body for creating a policy.
type CreatePolicyRequest struct {
	Name             string                  `json:"name" binding:"required,min=1,max=255"`
	Description      string                  `json:"description,omitempty"`
	Paths            []string                `json:"paths,omitempty"`
	Excludes         []string                `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                    `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int                   `json:"excluded_hours,omitempty"`
	CronExpression   string                  `json:"cron_expression,omitempty"`
}

// UpdatePolicyRequest is the request body for updating a policy.
type UpdatePolicyRequest struct {
	Name             string                  `json:"name,omitempty"`
	Description      string                  `json:"description,omitempty"`
	Paths            []string                `json:"paths,omitempty"`
	Excludes         []string                `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                    `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int                   `json:"excluded_hours,omitempty"`
	CronExpression   string                  `json:"cron_expression,omitempty"`
}

// ApplyPolicyRequest is the request body for applying a policy to agents.
type ApplyPolicyRequest struct {
	AgentIDs     []uuid.UUID `json:"agent_ids" binding:"required,min=1"`
	RepositoryID uuid.UUID   `json:"repository_id" binding:"required"`
	ScheduleName string      `json:"schedule_name,omitempty"` // Optional name template
}

// ApplyPolicyResponse is the response for applying a policy.
type ApplyPolicyResponse struct {
	SchedulesCreated int                `json:"schedules_created"`
	Schedules        []*models.Schedule `json:"schedules"`
}

// List returns all policies for the authenticated user's organization.
// GET /api/v1/policies
func (h *PoliciesHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	policies, err := h.store.GetPoliciesByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list policies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list policies"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// Get returns a specific policy by ID.
// GET /api/v1/policies/:id
func (h *PoliciesHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to get policy")
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Verify policy belongs to user's org
	if policy.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// Create creates a new policy.
// POST /api/v1/policies
func (h *PoliciesHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	policy := models.NewPolicy(user.CurrentOrgID, req.Name)
	policy.Description = req.Description
	policy.CronExpression = req.CronExpression

	if req.Paths != nil {
		policy.Paths = req.Paths
	}
	if req.Excludes != nil {
		policy.Excludes = req.Excludes
	}
	if req.RetentionPolicy != nil {
		policy.RetentionPolicy = req.RetentionPolicy
	}
	if req.BandwidthLimitKB != nil {
		policy.BandwidthLimitKB = req.BandwidthLimitKB
	}
	if req.BackupWindow != nil {
		policy.BackupWindow = req.BackupWindow
	}
	if req.ExcludedHours != nil {
		policy.ExcludedHours = req.ExcludedHours
	}

	if err := h.store.CreatePolicy(c.Request.Context(), policy); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create policy"})
		return
	}

	h.logger.Info().
		Str("policy_id", policy.ID.String()).
		Str("name", req.Name).
		Msg("policy created")

	c.JSON(http.StatusCreated, policy)
}

// Update updates an existing policy.
// PUT /api/v1/policies/:id
func (h *PoliciesHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	policy, err := h.store.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Verify policy belongs to user's org
	if policy.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		policy.Name = req.Name
	}
	if req.Description != "" {
		policy.Description = req.Description
	}
	if req.CronExpression != "" {
		policy.CronExpression = req.CronExpression
	}
	if req.Paths != nil {
		policy.Paths = req.Paths
	}
	if req.Excludes != nil {
		policy.Excludes = req.Excludes
	}
	if req.RetentionPolicy != nil {
		policy.RetentionPolicy = req.RetentionPolicy
	}
	if req.BandwidthLimitKB != nil {
		policy.BandwidthLimitKB = req.BandwidthLimitKB
	}
	if req.BackupWindow != nil {
		policy.BackupWindow = req.BackupWindow
	}
	if req.ExcludedHours != nil {
		policy.ExcludedHours = req.ExcludedHours
	}

	if err := h.store.UpdatePolicy(c.Request.Context(), policy); err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to update policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update policy"})
		return
	}

	h.logger.Info().Str("policy_id", id.String()).Msg("policy updated")
	c.JSON(http.StatusOK, policy)
}

// Delete removes a policy.
// DELETE /api/v1/policies/:id
func (h *PoliciesHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Verify policy belongs to user's org
	if policy.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	if err := h.store.DeletePolicy(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to delete policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete policy"})
		return
	}

	h.logger.Info().Str("policy_id", id.String()).Msg("policy deleted")
	c.JSON(http.StatusOK, gin.H{"message": "policy deleted"})
}

// ListSchedules returns all schedules using a policy.
// GET /api/v1/policies/:id/schedules
func (h *PoliciesHandler) ListSchedules(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Verify policy belongs to user's org
	if policy.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	schedules, err := h.store.GetSchedulesByPolicyID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to list schedules for policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// Apply applies a policy to one or more agents, creating schedules.
// POST /api/v1/policies/:id/apply
func (h *PoliciesHandler) Apply(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	var req ApplyPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	policy, err := h.store.GetPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Verify policy belongs to user's org
	if policy.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	// Verify repository belongs to user's org
	repo, err := h.store.GetRepositoryByID(c.Request.Context(), req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found"})
		return
	}
	if repo.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository not found"})
		return
	}

	// Verify all agents belong to user's org and create schedules
	var createdSchedules []*models.Schedule
	for _, agentID := range req.AgentIDs {
		agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Warn().Err(err).Str("agent_id", agentID.String()).Msg("agent not found, skipping")
			continue
		}
		if agent.OrgID != user.CurrentOrgID {
			h.logger.Warn().Str("agent_id", agentID.String()).Msg("agent not in user's org, skipping")
			continue
		}

		// Create schedule name
		scheduleName := req.ScheduleName
		if scheduleName == "" {
			scheduleName = fmt.Sprintf("%s - %s", policy.Name, agent.Hostname)
		} else {
			scheduleName = fmt.Sprintf("%s - %s", scheduleName, agent.Hostname)
		}

		// Use policy's cron expression, or default to daily at 2am
		cronExpr := policy.CronExpression
		if cronExpr == "" {
			cronExpr = "0 2 * * *"
		}

		// Use policy's paths, or require some default
		paths := policy.Paths
		if len(paths) == 0 {
			paths = []string{"/home"}
		}

		schedule := models.NewSchedule(agentID, scheduleName, cronExpr, paths)

		// Add the repository association
		schedule.Repositories = []models.ScheduleRepository{
			*models.NewScheduleRepository(schedule.ID, req.RepositoryID, 0),
		}

		// Apply policy settings to schedule
		policy.ApplyToSchedule(schedule)

		// Apply default retention if policy doesn't have one
		if schedule.RetentionPolicy == nil {
			schedule.RetentionPolicy = models.DefaultRetentionPolicy()
		}

		if err := h.store.CreateSchedule(c.Request.Context(), schedule); err != nil {
			h.logger.Error().Err(err).
				Str("agent_id", agentID.String()).
				Str("policy_id", id.String()).
				Msg("failed to create schedule from policy")
			continue
		}

		createdSchedules = append(createdSchedules, schedule)
		h.logger.Info().
			Str("schedule_id", schedule.ID.String()).
			Str("policy_id", id.String()).
			Str("agent_id", agentID.String()).
			Msg("schedule created from policy")
	}

	c.JSON(http.StatusCreated, ApplyPolicyResponse{
		SchedulesCreated: len(createdSchedules),
		Schedules:        createdSchedules,
	})
}
