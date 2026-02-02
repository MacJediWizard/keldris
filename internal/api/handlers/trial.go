package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// TrialStore defines the interface for trial data persistence.
type TrialStore interface {
	GetTrialInfo(ctx context.Context, orgID uuid.UUID) (*license.TrialInfo, error)
	StartTrial(ctx context.Context, orgID uuid.UUID, email string) error
	ExtendTrial(ctx context.Context, orgID, extendedBy uuid.UUID, days int, reason string) (*license.TrialExtension, error)
	ConvertTrial(ctx context.Context, orgID uuid.UUID, tier license.PlanTier) error
	ExpireTrials(ctx context.Context) (int, error)
	GetTrialExtensions(ctx context.Context, orgID uuid.UUID) ([]*license.TrialExtension, error)
	LogTrialActivity(ctx context.Context, activity *license.TrialActivity) error
	GetTrialActivity(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*license.TrialActivity, error)
	GetExpiringTrials(ctx context.Context, withinDays int) ([]*license.TrialInfo, error)
}

// TrialHandler handles trial-related HTTP endpoints.
type TrialHandler struct {
	manager *license.Manager
	store   TrialStore
	logger  zerolog.Logger
}

// NewTrialHandler creates a new TrialHandler.
func NewTrialHandler(store TrialStore, logger zerolog.Logger) *TrialHandler {
	return &TrialHandler{
		manager: license.NewManager(store),
		store:   store,
		logger:  logger.With().Str("component", "trial_handler").Logger(),
	}
}

// RegisterRoutes registers trial routes on the given router group.
func (h *TrialHandler) RegisterRoutes(r *gin.RouterGroup) {
	trial := r.Group("/trial")
	{
		// Get current trial status
		trial.GET("/status", h.GetStatus)

		// Start trial (collect email)
		trial.POST("/start", h.StartTrial)

		// Get available Pro features
		trial.GET("/features", h.GetFeatures)

		// Get trial activity log
		trial.GET("/activity", h.GetActivity)

		// Extend trial (admin/superuser only)
		trial.POST("/extend", h.ExtendTrial)

		// Convert trial to paid (marks as converted)
		trial.POST("/convert", h.ConvertTrial)

		// Get extension history
		trial.GET("/extensions", h.GetExtensions)
	}
}

// GetStatus returns the current trial status for the organization.
// GET /api/v1/trial/status
func (h *TrialHandler) GetStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	info, err := h.manager.GetTrialInfo(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get trial info")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trial status"})
		return
	}

	c.JSON(http.StatusOK, info)
}

// StartTrial begins a new trial for the organization.
// POST /api/v1/trial/start
func (h *TrialHandler) StartTrial(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Only admins can start trials
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req license.StartTrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	info, err := h.manager.StartTrial(c.Request.Context(), user.CurrentOrgID, req.Email)
	if err != nil {
		h.logger.Warn().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to start trial")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("email", req.Email).
		Str("user_id", user.ID.String()).
		Msg("trial started")

	c.JSON(http.StatusOK, info)
}

// GetFeatures returns the list of Pro features with availability.
// GET /api/v1/trial/features
func (h *TrialHandler) GetFeatures(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	features, err := h.manager.GetProFeatures(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get features")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get features"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"features": features})
}

// GetActivity returns the trial feature usage log.
// GET /api/v1/trial/activity
func (h *TrialHandler) GetActivity(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	activities, err := h.store.GetTrialActivity(c.Request.Context(), user.CurrentOrgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get trial activity")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"activities": activities})
}

// ExtendTrial extends an active trial (admin/superuser only).
// POST /api/v1/trial/extend
func (h *TrialHandler) ExtendTrial(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Only superusers can extend trials
	if !user.IsSuperuser {
		c.JSON(http.StatusForbidden, gin.H{"error": "superuser access required"})
		return
	}

	var req license.ExtendTrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	extension, err := h.manager.ExtendTrial(c.Request.Context(), user.CurrentOrgID, user.ID, req.ExtensionDays, req.Reason)
	if err != nil {
		h.logger.Warn().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to extend trial")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Int("days", req.ExtensionDays).
		Str("extended_by", user.ID.String()).
		Msg("trial extended")

	c.JSON(http.StatusOK, extension)
}

// ConvertTrial marks the trial as converted (to paid subscription).
// POST /api/v1/trial/convert
func (h *TrialHandler) ConvertTrial(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Only admins can convert trials
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req license.ConvertTrialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.manager.ConvertTrial(c.Request.Context(), user.CurrentOrgID, req.PlanTier); err != nil {
		h.logger.Warn().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to convert trial")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("tier", string(req.PlanTier)).
		Str("user_id", user.ID.String()).
		Msg("trial converted")

	// Return updated status
	info, _ := h.manager.GetTrialInfo(c.Request.Context(), user.CurrentOrgID)
	c.JSON(http.StatusOK, info)
}

// GetExtensions returns the trial extension history.
// GET /api/v1/trial/extensions
func (h *TrialHandler) GetExtensions(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	extensions, err := h.manager.GetTrialExtensions(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get extensions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get extensions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"extensions": extensions})
}
