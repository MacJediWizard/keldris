package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ScheduleStore defines the interface for schedule persistence operations.
type ScheduleStore interface {
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) error
	UpdateSchedule(ctx context.Context, schedule *models.Schedule) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
}

// SchedulesHandler handles schedule-related HTTP endpoints.
type SchedulesHandler struct {
	store  ScheduleStore
	logger zerolog.Logger
}

// NewSchedulesHandler creates a new SchedulesHandler.
func NewSchedulesHandler(store ScheduleStore, logger zerolog.Logger) *SchedulesHandler {
	return &SchedulesHandler{
		store:  store,
		logger: logger.With().Str("component", "schedules_handler").Logger(),
	}
}

// RegisterRoutes registers schedule routes on the given router group.
func (h *SchedulesHandler) RegisterRoutes(r *gin.RouterGroup) {
	schedules := r.Group("/schedules")
	{
		schedules.GET("", h.List)
		schedules.POST("", h.Create)
		schedules.GET("/:id", h.Get)
		schedules.PUT("/:id", h.Update)
		schedules.DELETE("/:id", h.Delete)
		schedules.POST("/:id/run", h.Run)
	}
}

// CreateScheduleRequest is the request body for creating a schedule.
type CreateScheduleRequest struct {
	AgentID          uuid.UUID               `json:"agent_id" binding:"required"`
	RepositoryID     uuid.UUID               `json:"repository_id" binding:"required"`
	Name             string                  `json:"name" binding:"required,min=1,max=255"`
	CronExpression   string                  `json:"cron_expression" binding:"required"`
	Paths            []string                `json:"paths" binding:"required,min=1"`
	Excludes         []string                `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                    `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int                   `json:"excluded_hours,omitempty"`
	Enabled          *bool                   `json:"enabled,omitempty"`
}

// UpdateScheduleRequest is the request body for updating a schedule.
type UpdateScheduleRequest struct {
	Name             string                  `json:"name,omitempty"`
	CronExpression   string                  `json:"cron_expression,omitempty"`
	Paths            []string                `json:"paths,omitempty"`
	Excludes         []string                `json:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                    `json:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow    `json:"backup_window,omitempty"`
	ExcludedHours    []int                   `json:"excluded_hours,omitempty"`
	Enabled          *bool                   `json:"enabled,omitempty"`
}

// List returns all schedules for agents in the authenticated user's organization.
// GET /api/v1/schedules
// Optional query param: agent_id to filter by agent
func (h *SchedulesHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Optional agent_id filter
	agentIDParam := c.Query("agent_id")
	if agentIDParam != "" {
		agentID, err := uuid.Parse(agentIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent_id"})
			return
		}

		// Verify agent belongs to user's org
		agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}
		if agent.OrgID != user.CurrentOrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
			return
		}

		schedules, err := h.store.GetSchedulesByAgentID(c.Request.Context(), agentID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list schedules")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"schedules": schedules})
		return
	}

	// Get all schedules for all agents in the org
	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list schedules"})
		return
	}

	var allSchedules []*models.Schedule
	for _, agent := range agents {
		schedules, err := h.store.GetSchedulesByAgentID(c.Request.Context(), agent.ID)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to list schedules for agent")
			continue
		}
		allSchedules = append(allSchedules, schedules...)
	}

	c.JSON(http.StatusOK, gin.H{"schedules": allSchedules})
}

// Get returns a specific schedule by ID.
// GET /api/v1/schedules/:id
func (h *SchedulesHandler) Get(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to get schedule")
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Verify schedule's agent belongs to user's org
	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// Create creates a new schedule.
// POST /api/v1/schedules
func (h *SchedulesHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), req.AgentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
		return
	}
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "agent not found"})
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

	// TODO: Validate cron expression using robfig/cron parser
	// For now we accept any string

	schedule := models.NewSchedule(req.AgentID, req.RepositoryID, req.Name, req.CronExpression, req.Paths)

	if req.Excludes != nil {
		schedule.Excludes = req.Excludes
	}

	if req.RetentionPolicy != nil {
		schedule.RetentionPolicy = req.RetentionPolicy
	} else {
		schedule.RetentionPolicy = models.DefaultRetentionPolicy()
	}

	if req.BandwidthLimitKB != nil {
		schedule.BandwidthLimitKB = req.BandwidthLimitKB
	}

	if req.BackupWindow != nil {
		schedule.BackupWindow = req.BackupWindow
	}

	if req.ExcludedHours != nil {
		schedule.ExcludedHours = req.ExcludedHours
	}

	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if err := h.store.CreateSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create schedule"})
		return
	}

	h.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("name", req.Name).
		Str("agent_id", req.AgentID.String()).
		Msg("schedule created")

	c.JSON(http.StatusCreated, schedule)
}

// Update updates an existing schedule.
// PUT /api/v1/schedules/:id
func (h *SchedulesHandler) Update(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var req UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	// Update fields
	if req.Name != "" {
		schedule.Name = req.Name
	}
	if req.CronExpression != "" {
		schedule.CronExpression = req.CronExpression
	}
	if req.Paths != nil {
		schedule.Paths = req.Paths
	}
	if req.Excludes != nil {
		schedule.Excludes = req.Excludes
	}
	if req.RetentionPolicy != nil {
		schedule.RetentionPolicy = req.RetentionPolicy
	}
	if req.BandwidthLimitKB != nil {
		schedule.BandwidthLimitKB = req.BandwidthLimitKB
	}
	if req.BackupWindow != nil {
		schedule.BackupWindow = req.BackupWindow
	}
	if req.ExcludedHours != nil {
		schedule.ExcludedHours = req.ExcludedHours
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if err := h.store.UpdateSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update schedule"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Msg("schedule updated")
	c.JSON(http.StatusOK, schedule)
}

// Delete removes a schedule.
// DELETE /api/v1/schedules/:id
func (h *SchedulesHandler) Delete(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	if err := h.store.DeleteSchedule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to delete schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete schedule"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Msg("schedule deleted")
	c.JSON(http.StatusOK, gin.H{"message": "schedule deleted"})
}

// RunScheduleResponse is the response for running a schedule.
type RunScheduleResponse struct {
	BackupID uuid.UUID `json:"backup_id"`
	Message  string    `json:"message"`
}

// Run triggers an immediate backup for this schedule.
// POST /api/v1/schedules/:id/run
func (h *SchedulesHandler) Run(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyScheduleAccess(c, user.CurrentOrgID, schedule); err != nil {
		return
	}

	// TODO: Implement actual backup trigger when backup package is ready
	// This will create a backup record and dispatch to the agent
	h.logger.Info().
		Str("schedule_id", id.String()).
		Str("agent_id", schedule.AgentID.String()).
		Msg("manual backup run requested")

	c.JSON(http.StatusAccepted, RunScheduleResponse{
		BackupID: uuid.New(), // Placeholder - would be actual backup ID
		Message:  "Backup run not yet implemented. Schedule exists and is accessible.",
	})
}

// verifyScheduleAccess checks if the user has access to the schedule.
// Returns nil if access is granted, or sends an error response and returns error.
func (h *SchedulesHandler) verifyScheduleAccess(c *gin.Context, orgID uuid.UUID, schedule *models.Schedule) error {
	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return err
	}

	if agent.OrgID != orgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return err
	}

	return nil
}
