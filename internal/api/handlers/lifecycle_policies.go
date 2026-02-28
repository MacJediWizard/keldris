package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/classification"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/lifecycle"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LifecyclePolicyStore defines the interface for lifecycle policy persistence operations.
type LifecyclePolicyStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	CreateLifecyclePolicy(ctx context.Context, policy *models.LifecyclePolicy) error
	GetLifecyclePolicyByID(ctx context.Context, id uuid.UUID) (*models.LifecyclePolicy, error)
	GetLifecyclePoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.LifecyclePolicy, error)
	GetActiveLifecyclePoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.LifecyclePolicy, error)
	UpdateLifecyclePolicy(ctx context.Context, policy *models.LifecyclePolicy) error
	DeleteLifecyclePolicy(ctx context.Context, id uuid.UUID) error
	CreateLifecycleDeletionEvent(ctx context.Context, event *models.LifecycleDeletionEvent) error
	GetLifecycleDeletionEventsByPolicyID(ctx context.Context, policyID uuid.UUID, limit int) ([]*models.LifecycleDeletionEvent, error)
	GetLifecycleDeletionEventsByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.LifecycleDeletionEvent, error)
	// For dry-run evaluation
	GetBackupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Backup, error)
	GetBackupClassification(ctx context.Context, backupID uuid.UUID) (*models.BackupClassification, error)
	IsSnapshotOnHold(ctx context.Context, snapshotID string, orgID uuid.UUID) (bool, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	// Audit logging
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// LifecyclePoliciesHandler handles lifecycle policy HTTP endpoints.
type LifecyclePoliciesHandler struct {
	store   LifecyclePolicyStore
	checker *license.FeatureChecker
	logger  zerolog.Logger
}

// NewLifecyclePoliciesHandler creates a new LifecyclePoliciesHandler.
func NewLifecyclePoliciesHandler(store LifecyclePolicyStore, checker *license.FeatureChecker, logger zerolog.Logger) *LifecyclePoliciesHandler {
	return &LifecyclePoliciesHandler{
		store:   store,
		checker: checker,
		logger:  logger.With().Str("component", "lifecycle_policies_handler").Logger(),
	}
}

// RegisterRoutes registers lifecycle policy routes on the given router group.
func (h *LifecyclePoliciesHandler) RegisterRoutes(r *gin.RouterGroup) {
	policies := r.Group("/lifecycle-policies")
	{
		policies.GET("", h.ListLifecyclePolicies)
		policies.POST("", h.CreateLifecyclePolicy)
		policies.GET("/:id", h.GetLifecyclePolicy)
		policies.PUT("/:id", h.UpdateLifecyclePolicy)
		policies.DELETE("/:id", h.DeleteLifecyclePolicy)
		policies.POST("/:id/dry-run", h.DryRunPolicy)
		policies.GET("/:id/deletions", h.ListDeletionEvents)
	}

	// Dry run without policy (preview mode)
	r.POST("/lifecycle-policies/preview", h.PreviewDryRun)

	// Deletion events for org
	r.GET("/lifecycle-deletions", h.ListOrgDeletionEvents)
}

// LifecyclePolicyResponse represents a lifecycle policy in API responses.
type LifecyclePolicyResponse struct {
	ID              string                           `json:"id"`
	Name            string                           `json:"name"`
	Description     string                           `json:"description,omitempty"`
	Status          string                           `json:"status"`
	Rules           []models.ClassificationRetention `json:"rules"`
	RepositoryIDs   []string                         `json:"repository_ids,omitempty"`
	ScheduleIDs     []string                         `json:"schedule_ids,omitempty"`
	LastEvaluatedAt *string                          `json:"last_evaluated_at,omitempty"`
	LastDeletionAt  *string                          `json:"last_deletion_at,omitempty"`
	DeletionCount   int64                            `json:"deletion_count"`
	BytesReclaimed  int64                            `json:"bytes_reclaimed"`
	CreatedBy       string                           `json:"created_by"`
	CreatedAt       string                           `json:"created_at"`
	UpdatedAt       string                           `json:"updated_at"`
}

func (h *LifecyclePoliciesHandler) toResponse(p *models.LifecyclePolicy) LifecyclePolicyResponse {
	resp := LifecyclePolicyResponse{
		ID:             p.ID.String(),
		Name:           p.Name,
		Description:    p.Description,
		Status:         string(p.Status),
		Rules:          p.Rules,
		DeletionCount:  p.DeletionCount,
		BytesReclaimed: p.BytesReclaimed,
		CreatedBy:      p.CreatedBy.String(),
		CreatedAt:      p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      p.UpdatedAt.Format(time.RFC3339),
	}

	if p.LastEvaluatedAt != nil {
		ts := p.LastEvaluatedAt.Format(time.RFC3339)
		resp.LastEvaluatedAt = &ts
	}
	if p.LastDeletionAt != nil {
		ts := p.LastDeletionAt.Format(time.RFC3339)
		resp.LastDeletionAt = &ts
	}

	if p.RepositoryIDs != nil {
		resp.RepositoryIDs = make([]string, len(p.RepositoryIDs))
		for i, id := range p.RepositoryIDs {
			resp.RepositoryIDs[i] = id.String()
		}
	}
	if p.ScheduleIDs != nil {
		resp.ScheduleIDs = make([]string, len(p.ScheduleIDs))
		for i, id := range p.ScheduleIDs {
			resp.ScheduleIDs[i] = id.String()
		}
	}

	return resp
}

// ListLifecyclePolicies returns all lifecycle policies for the authenticated user's organization.
//
//	@Summary		List lifecycle policies
//	@Description	Returns all lifecycle policies for the current organization (admin only)
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]LifecyclePolicyResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies [get]
func (h *LifecyclePoliciesHandler) ListLifecyclePolicies(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	policies, err := h.store.GetLifecyclePoliciesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list lifecycle policies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list lifecycle policies"})
		return
	}

	responses := make([]LifecyclePolicyResponse, len(policies))
	for i, p := range policies {
		responses[i] = h.toResponse(p)
	}

	c.JSON(http.StatusOK, gin.H{"policies": responses})
}

// CreateLifecyclePolicy creates a new lifecycle policy.
//
//	@Summary		Create lifecycle policy
//	@Description	Creates a new lifecycle policy for automated snapshot deletion (admin only)
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.CreateLifecyclePolicyRequest	true	"Policy details"
//	@Success		201		{object}	LifecyclePolicyResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies [post]
func (h *LifecyclePoliciesHandler) CreateLifecyclePolicy(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureCustomRetention) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.CreateLifecyclePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if len(req.Rules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one rule is required"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	policy := models.NewLifecyclePolicy(dbUser.OrgID, req.Name, dbUser.ID)
	policy.Description = req.Description
	policy.Rules = req.Rules

	if req.Status != "" {
		policy.Status = req.Status
	}

	// Parse repository IDs
	if len(req.RepositoryIDs) > 0 {
		policy.RepositoryIDs = make([]uuid.UUID, len(req.RepositoryIDs))
		for i, idStr := range req.RepositoryIDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id: " + idStr})
				return
			}
			policy.RepositoryIDs[i] = id
		}
	}

	// Parse schedule IDs
	if len(req.ScheduleIDs) > 0 {
		policy.ScheduleIDs = make([]uuid.UUID, len(req.ScheduleIDs))
		for i, idStr := range req.ScheduleIDs {
			id, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule_id: " + idStr})
				return
			}
			policy.ScheduleIDs[i] = id
		}
	}

	if err := h.store.CreateLifecyclePolicy(c.Request.Context(), policy); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create lifecycle policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create lifecycle policy"})
		return
	}

	// Create audit log
	auditLog := models.NewAuditLog(dbUser.OrgID, models.AuditActionCreate, "lifecycle_policy", models.AuditResultSuccess).
		WithUser(dbUser.ID).
		WithResource(policy.ID).
		WithDetails("Created lifecycle policy: " + policy.Name)
	if err := h.store.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log")
	}

	h.logger.Info().
		Str("policy_id", policy.ID.String()).
		Str("name", policy.Name).
		Str("created_by", dbUser.ID.String()).
		Msg("lifecycle policy created")

	c.JSON(http.StatusCreated, h.toResponse(policy))
}

// GetLifecyclePolicy returns a specific lifecycle policy.
//
//	@Summary		Get lifecycle policy
//	@Description	Returns a specific lifecycle policy by ID
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Policy ID"
//	@Success		200	{object}	LifecyclePolicyResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies/{id} [get]
func (h *LifecyclePoliciesHandler) GetLifecyclePolicy(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	policy, err := h.store.GetLifecyclePolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	// Verify org access
	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	c.JSON(http.StatusOK, h.toResponse(policy))
}

// UpdateLifecyclePolicy updates a lifecycle policy.
//
//	@Summary		Update lifecycle policy
//	@Description	Updates a lifecycle policy (admin only)
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Policy ID"
//	@Param			request	body		models.UpdateLifecyclePolicyRequest	true	"Update details"
//	@Success		200		{object}	LifecyclePolicyResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies/{id} [put]
func (h *LifecyclePoliciesHandler) UpdateLifecyclePolicy(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	var req models.UpdateLifecyclePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	policy, err := h.store.GetLifecyclePolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	// Verify org access
	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		policy.Name = *req.Name
	}
	if req.Description != nil {
		policy.Description = *req.Description
	}
	if req.Status != nil {
		policy.Status = *req.Status
	}
	if req.Rules != nil {
		policy.Rules = *req.Rules
	}
	if req.RepositoryIDs != nil {
		policy.RepositoryIDs = make([]uuid.UUID, len(*req.RepositoryIDs))
		for i, idStr := range *req.RepositoryIDs {
			repoID, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id: " + idStr})
				return
			}
			policy.RepositoryIDs[i] = repoID
		}
	}
	if req.ScheduleIDs != nil {
		policy.ScheduleIDs = make([]uuid.UUID, len(*req.ScheduleIDs))
		for i, idStr := range *req.ScheduleIDs {
			schedID, err := uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule_id: " + idStr})
				return
			}
			policy.ScheduleIDs[i] = schedID
		}
	}

	if err := h.store.UpdateLifecyclePolicy(c.Request.Context(), policy); err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to update lifecycle policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update lifecycle policy"})
		return
	}

	// Create audit log
	auditLog := models.NewAuditLog(dbUser.OrgID, models.AuditActionUpdate, "lifecycle_policy", models.AuditResultSuccess).
		WithUser(dbUser.ID).
		WithResource(policy.ID).
		WithDetails("Updated lifecycle policy: " + policy.Name)
	if err := h.store.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log")
	}

	h.logger.Info().
		Str("policy_id", policy.ID.String()).
		Str("updated_by", dbUser.ID.String()).
		Msg("lifecycle policy updated")

	c.JSON(http.StatusOK, h.toResponse(policy))
}

// DeleteLifecyclePolicy deletes a lifecycle policy.
//
//	@Summary		Delete lifecycle policy
//	@Description	Deletes a lifecycle policy (admin only)
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Policy ID"
//	@Success		200	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies/{id} [delete]
func (h *LifecyclePoliciesHandler) DeleteLifecyclePolicy(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	policy, err := h.store.GetLifecyclePolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	// Verify org access
	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	if err := h.store.DeleteLifecyclePolicy(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to delete lifecycle policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete lifecycle policy"})
		return
	}

	// Create audit log
	auditLog := models.NewAuditLog(dbUser.OrgID, models.AuditActionDelete, "lifecycle_policy", models.AuditResultSuccess).
		WithUser(dbUser.ID).
		WithResource(policy.ID).
		WithDetails("Deleted lifecycle policy: " + policy.Name)
	if err := h.store.CreateAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log")
	}

	h.logger.Info().
		Str("policy_id", id.String()).
		Str("deleted_by", dbUser.ID.String()).
		Msg("lifecycle policy deleted")

	c.JSON(http.StatusOK, gin.H{"message": "lifecycle policy deleted"})
}

// DryRunPolicy performs a dry-run evaluation of a policy.
//
//	@Summary		Dry run lifecycle policy
//	@Description	Preview what snapshots would be deleted by this policy
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Policy ID"
//	@Success		200	{object}	models.LifecycleDryRunResult
//	@Failure		401	{object}	map[string]string
//	@Failure		403	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies/{id}/dry-run [post]
func (h *LifecyclePoliciesHandler) DryRunPolicy(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	policy, err := h.store.GetLifecyclePolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	// Verify org access
	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	result, err := h.performDryRun(c.Request.Context(), dbUser.OrgID, policy.Rules, policy.RepositoryIDs, policy.ScheduleIDs)
	if err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to perform dry run")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform dry run"})
		return
	}

	result.PolicyID = id.String()

	c.JSON(http.StatusOK, result)
}

// PreviewDryRun performs a dry-run without a saved policy.
//
//	@Summary		Preview lifecycle dry run
//	@Description	Preview what snapshots would be deleted with given rules (without saving a policy)
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			request	body		models.LifecycleDryRunRequest	true	"Dry run parameters"
//	@Success		200		{object}	models.LifecycleDryRunResult
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies/preview [post]
func (h *LifecyclePoliciesHandler) PreviewDryRun(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.LifecycleDryRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var rules []models.ClassificationRetention
	var repoIDs []uuid.UUID
	var schedIDs []uuid.UUID

	// If policy ID provided, load the policy
	if req.PolicyID != "" {
		policyID, err := uuid.Parse(req.PolicyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy_id"})
			return
		}

		policy, err := h.store.GetLifecyclePolicyByID(c.Request.Context(), policyID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
			return
		}

		if policy.OrgID != dbUser.OrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
			return
		}

		rules = policy.Rules
		repoIDs = policy.RepositoryIDs
		schedIDs = policy.ScheduleIDs
	} else if len(req.Rules) > 0 {
		rules = req.Rules
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "either policy_id or rules must be provided"})
		return
	}

	// Parse repository IDs from request if provided
	if len(req.RepositoryIDs) > 0 {
		repoIDs = make([]uuid.UUID, len(req.RepositoryIDs))
		for i, idStr := range req.RepositoryIDs {
			repoIDs[i], err = uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id: " + idStr})
				return
			}
		}
	}

	// Parse schedule IDs from request if provided
	if len(req.ScheduleIDs) > 0 {
		schedIDs = make([]uuid.UUID, len(req.ScheduleIDs))
		for i, idStr := range req.ScheduleIDs {
			schedIDs[i], err = uuid.Parse(idStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule_id: " + idStr})
				return
			}
		}
	}

	result, err := h.performDryRun(c.Request.Context(), dbUser.OrgID, rules, repoIDs, schedIDs)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to perform dry run preview")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to perform dry run"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// performDryRun executes the dry-run evaluation logic.
func (h *LifecyclePoliciesHandler) performDryRun(
	ctx context.Context,
	orgID uuid.UUID,
	rules []models.ClassificationRetention,
	repoIDs []uuid.UUID,
	schedIDs []uuid.UUID,
) (*models.LifecycleDryRunResult, error) {
	// Convert model rules to lifecycle rules
	lifecycleRules := make([]lifecycle.ClassificationRule, len(rules))
	for i, r := range rules {
		lifecycleRules[i] = lifecycle.ClassificationRule{
			Level: r.Level,
			Retention: lifecycle.RetentionDuration{
				MinDays: r.Retention.MinDays,
				MaxDays: r.Retention.MaxDays,
			},
		}
		if len(r.DataTypeOverrides) > 0 {
			lifecycleRules[i].DataTypeOverrides = make(map[classification.DataType]lifecycle.RetentionDuration)
			for _, override := range r.DataTypeOverrides {
				lifecycleRules[i].DataTypeOverrides[override.DataType] = lifecycle.RetentionDuration{
					MinDays: override.Retention.MinDays,
					MaxDays: override.Retention.MaxDays,
				}
			}
		}
	}

	evaluator := lifecycle.NewEvaluator(lifecycleRules)

	result := &models.LifecycleDryRunResult{
		EvaluatedAt: time.Now(),
		Evaluations: []models.LifecycleSnapshotEvaluation{},
	}

	// Get all backups for the org
	backups, err := h.store.GetBackupsByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Create filter sets if specified
	repoFilter := make(map[uuid.UUID]bool)
	for _, id := range repoIDs {
		repoFilter[id] = true
	}
	schedFilter := make(map[uuid.UUID]bool)
	for _, id := range schedIDs {
		schedFilter[id] = true
	}

	for _, backup := range backups {
		// Skip if no snapshot
		if backup.SnapshotID == "" {
			continue
		}

		// Filter by repository
		if len(repoFilter) > 0 && backup.RepositoryID != nil && !repoFilter[*backup.RepositoryID] {
			continue
		}

		// Filter by schedule
		if len(schedFilter) > 0 && !schedFilter[backup.ScheduleID] {
			continue
		}

		// Get classification for the backup
		classLevel := classification.LevelPublic
		var dataTypes []classification.DataType
		if classif, err := h.store.GetBackupClassification(ctx, backup.ID); err == nil && classif != nil {
			classLevel = classif.Level
			dataTypes = classif.DataTypes
		}

		// Check legal hold status
		isOnHold, _ := h.store.IsSnapshotOnHold(ctx, backup.SnapshotID, orgID)

		// Evaluate the snapshot
		eval := evaluator.EvaluateSnapshot(
			backup.SnapshotID,
			backup.StartedAt,
			classLevel,
			dataTypes,
			isOnHold,
		)

		// Get schedule name
		scheduleName := ""
		if schedule, err := h.store.GetScheduleByID(ctx, backup.ScheduleID); err == nil && schedule != nil {
			scheduleName = schedule.Name
		}

		// Build the evaluation response
		snapshotEval := models.LifecycleSnapshotEvaluation{
			SnapshotID:          backup.SnapshotID,
			Action:              string(eval.Action),
			Reason:              eval.Reason,
			SnapshotAgeDays:     eval.SnapshotAge,
			MinRetentionDays:    eval.MinRetention,
			MaxRetentionDays:    eval.MaxRetention,
			DaysUntilDeletable:  eval.DaysUntilDeletable,
			DaysUntilAutoDelete: eval.DaysUntilAutoDelete,
			ClassificationLevel: eval.ClassificationLevel,
			IsOnLegalHold:       eval.IsOnLegalHold,
			SnapshotTime:        backup.StartedAt,
			ScheduleName:        scheduleName,
		}

		if backup.RepositoryID != nil {
			snapshotEval.RepositoryID = backup.RepositoryID.String()
		}
		if backup.SizeBytes != nil {
			snapshotEval.SizeBytes = *backup.SizeBytes
		}

		result.TotalSnapshots++
		result.Evaluations = append(result.Evaluations, snapshotEval)

		switch eval.Action {
		case lifecycle.ActionKeep:
			result.KeepCount++
		case lifecycle.ActionCanDelete:
			result.CanDeleteCount++
			if backup.SizeBytes != nil {
				result.TotalSizeToDelete += *backup.SizeBytes
			}
		case lifecycle.ActionMustDelete:
			result.MustDeleteCount++
			if backup.SizeBytes != nil {
				result.TotalSizeToDelete += *backup.SizeBytes
			}
		case lifecycle.ActionHold:
			result.HoldCount++
		}
	}

	return result, nil
}

// ListDeletionEvents returns deletion events for a policy.
//
//	@Summary		List policy deletion events
//	@Description	Returns the deletion events for a specific policy
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Policy ID"
//	@Param			limit	query		int		false	"Maximum number of events to return"
//	@Success		200		{object}	map[string][]models.LifecycleDeletionEvent
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-policies/{id}/deletions [get]
func (h *LifecyclePoliciesHandler) ListDeletionEvents(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	policy, err := h.store.GetLifecyclePolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	// Verify org access
	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle policy not found"})
		return
	}

	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := parseIntParam(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	events, err := h.store.GetLifecycleDeletionEventsByPolicyID(c.Request.Context(), id, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to list deletion events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list deletion events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// ListOrgDeletionEvents returns all deletion events for the organization.
//
//	@Summary		List organization deletion events
//	@Description	Returns all lifecycle deletion events for the organization
//	@Tags			Lifecycle Policies
//	@Accept			json
//	@Produce		json
//	@Param			limit	query		int	false	"Maximum number of events to return"
//	@Success		200		{object}	map[string][]models.LifecycleDeletionEvent
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/lifecycle-deletions [get]
func (h *LifecyclePoliciesHandler) ListOrgDeletionEvents(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Admin-only access
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	limit := 100
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := parseIntParam(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	events, err := h.store.GetLifecycleDeletionEventsByOrgID(c.Request.Context(), dbUser.OrgID, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list deletion events")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list deletion events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}
