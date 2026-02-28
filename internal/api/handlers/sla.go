package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/sla"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SLAStore defines the interface for SLA persistence operations.
type SLAStore interface {
	CreateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error
	GetSLAPolicyByID(ctx context.Context, id uuid.UUID) (*models.SLAPolicy, error)
	ListSLAPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SLAPolicy, error)
	UpdateSLAPolicy(ctx context.Context, policy *models.SLAPolicy) error
	DeleteSLAPolicy(ctx context.Context, id uuid.UUID) error
	CreateSLAStatusSnapshot(ctx context.Context, snapshot *models.SLAStatusSnapshot) error
	GetSLAStatusHistory(ctx context.Context, policyID uuid.UUID, limit int) ([]*models.SLAStatusSnapshot, error)
	GetLatestSLAStatus(ctx context.Context, policyID uuid.UUID) (*models.SLAStatusSnapshot, error)
	GetBackupSuccessRateForOrg(ctx context.Context, orgID uuid.UUID, hours int) (float64, error)
	GetMaxRPOHoursForOrg(ctx context.Context, orgID uuid.UUID) (float64, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	// Tracker methods
	GetSLADefinitionByID(ctx context.Context, id uuid.UUID) (*models.SLADefinition, error)
	ListActiveSLADefinitionsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLADefinition, error)
	ListSLAAssignmentsBySLA(ctx context.Context, slaID uuid.UUID) ([]*models.SLAAssignment, error)
	ListSLAAssignmentsByAgent(ctx context.Context, agentID uuid.UUID) ([]*models.SLAAssignment, error)
	CreateSLACompliance(ctx context.Context, c *models.SLACompliance) error
	CreateSLABreach(ctx context.Context, b *models.SLABreach) error
	ListActiveSLABreachesByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SLABreach, error)
	UpdateSLABreach(ctx context.Context, b *models.SLABreach) error
}

// SLAHandler handles SLA policy HTTP endpoints.
type SLAHandler struct {
	store      SLAStore
	checker    *license.FeatureChecker
	calculator *sla.Calculator
	logger     zerolog.Logger
}

// NewSLAHandler creates a new SLAHandler.
func NewSLAHandler(store SLAStore, checker *license.FeatureChecker, logger zerolog.Logger) *SLAHandler {
	return &SLAHandler{
		store:      store,
		checker:    checker,
		calculator: sla.NewCalculator(store),
		logger:     logger.With().Str("component", "sla_handler").Logger(),
	}
}

// RegisterRoutes registers SLA routes on the given router group.
func (h *SLAHandler) RegisterRoutes(r *gin.RouterGroup) {
	policies := r.Group("/sla/policies")
	{
		policies.GET("", h.List)
		policies.POST("", h.Create)
		policies.GET("/:id", h.Get)
		policies.PUT("/:id", h.Update)
		policies.DELETE("/:id", h.Delete)
		policies.GET("/:id/status", h.GetStatus)
		policies.GET("/:id/history", h.GetHistory)
	}
}

// List returns all SLA policies for the authenticated user's organization.
func (h *SLAHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	policies, err := h.store.ListSLAPoliciesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list SLA policies")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list SLA policies"})
		return
	}

	if policies == nil {
		policies = []*models.SLAPolicy{}
	}

	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// Create creates a new SLA policy.
func (h *SLAHandler) Create(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureSLATracking) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.CreateSLAPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	policy := models.NewSLAPolicy(dbUser.OrgID, req.Name, req.TargetRPOHours, req.TargetRTOHours, req.TargetSuccessRate)
	policy.Description = req.Description

	if err := h.store.CreateSLAPolicy(c.Request.Context(), policy); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create SLA policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create SLA policy"})
		return
	}

	h.logger.Info().
		Str("policy_id", policy.ID.String()).
		Str("name", policy.Name).
		Msg("SLA policy created")

	c.JSON(http.StatusCreated, policy)
}

// Get returns a specific SLA policy by ID.
func (h *SLAHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetSLAPolicyByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to get SLA policy")
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	c.JSON(http.StatusOK, policy)
}

// Update updates an existing SLA policy.
func (h *SLAHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetSLAPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	var req models.UpdateSLAPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Name != nil {
		policy.Name = *req.Name
	}
	if req.Description != nil {
		policy.Description = *req.Description
	}
	if req.TargetRPOHours != nil {
		policy.TargetRPOHours = *req.TargetRPOHours
	}
	if req.TargetRTOHours != nil {
		policy.TargetRTOHours = *req.TargetRTOHours
	}
	if req.TargetSuccessRate != nil {
		policy.TargetSuccessRate = *req.TargetSuccessRate
	}
	if req.Enabled != nil {
		policy.Enabled = *req.Enabled
	}

	if err := h.store.UpdateSLAPolicy(c.Request.Context(), policy); err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to update SLA policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update SLA policy"})
		return
	}

	h.logger.Info().Str("policy_id", id.String()).Msg("SLA policy updated")

	c.JSON(http.StatusOK, policy)
}

// Delete deletes an SLA policy.
func (h *SLAHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetSLAPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	if err := h.store.DeleteSLAPolicy(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to delete SLA policy")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete SLA policy"})
		return
	}

	h.logger.Info().Str("policy_id", id.String()).Msg("SLA policy deleted")
	c.JSON(http.StatusOK, gin.H{"message": "SLA policy deleted"})
}

// GetStatus returns the current SLA compliance status for a policy.
func (h *SLAHandler) GetStatus(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureSLATracking) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetSLAPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	status, err := h.calculator.CalculateSLAStatus(c.Request.Context(), dbUser.OrgID, id)
	if err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to calculate SLA status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to calculate SLA status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetHistory returns the SLA status history for a policy.
func (h *SLAHandler) GetHistory(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid policy ID"})
		return
	}

	policy, err := h.store.GetSLAPolicyByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if policy.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "SLA policy not found"})
		return
	}

	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.store.GetSLAStatusHistory(c.Request.Context(), id, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("policy_id", id.String()).Msg("failed to get SLA history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get SLA history"})
		return
	}

	if history == nil {
		history = []*models.SLAStatusSnapshot{}
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}
