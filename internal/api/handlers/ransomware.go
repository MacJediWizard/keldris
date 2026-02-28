package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RansomwareStore defines the interface for ransomware persistence operations.
type RansomwareStore interface {
	GetRansomwareSettingsByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*models.RansomwareSettings, error)
	GetRansomwareSettingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareSettings, error)
	CreateRansomwareSettings(ctx context.Context, settings *models.RansomwareSettings) error
	UpdateRansomwareSettings(ctx context.Context, settings *models.RansomwareSettings) error
	DeleteRansomwareSettings(ctx context.Context, id uuid.UUID) error
	GetRansomwareAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareAlert, error)
	GetActiveRansomwareAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RansomwareAlert, error)
	GetActiveRansomwareAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	GetRansomwareAlertByID(ctx context.Context, id uuid.UUID) (*models.RansomwareAlert, error)
	UpdateRansomwareAlert(ctx context.Context, alert *models.RansomwareAlert) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	ResumeSchedule(ctx context.Context, scheduleID uuid.UUID) error
}

// RansomwareHandler handles ransomware detection HTTP endpoints.
type RansomwareHandler struct {
	store   RansomwareStore
	checker *license.FeatureChecker
	logger  zerolog.Logger
}

// NewRansomwareHandler creates a new RansomwareHandler.
func NewRansomwareHandler(store RansomwareStore, checker *license.FeatureChecker, logger zerolog.Logger) *RansomwareHandler {
	return &RansomwareHandler{
		store:   store,
		checker: checker,
		logger:  logger.With().Str("component", "ransomware_handler").Logger(),
	}
}

// RegisterRoutes registers ransomware routes on the given router group.
func (h *RansomwareHandler) RegisterRoutes(r *gin.RouterGroup) {
	ransomware := r.Group("/ransomware")
	{
		// Settings endpoints
		ransomware.GET("/settings", h.ListSettings)
		ransomware.GET("/settings/schedule/:schedule_id", h.GetSettingsBySchedule)
		ransomware.POST("/settings", h.CreateSettings)
		ransomware.PUT("/settings/:id", h.UpdateSettings)
		ransomware.DELETE("/settings/:id", h.DeleteSettings)

		// Alert endpoints
		ransomware.GET("/alerts", h.ListAlerts)
		ransomware.GET("/alerts/active", h.ListActiveAlerts)
		ransomware.GET("/alerts/count", h.CountActiveAlerts)
		ransomware.GET("/alerts/:id", h.GetAlert)
		ransomware.POST("/alerts/:id/actions/investigate", h.Investigate)
		ransomware.POST("/alerts/:id/actions/false-positive", h.MarkFalsePositive)
		ransomware.POST("/alerts/:id/actions/confirm", h.MarkConfirmed)
		ransomware.POST("/alerts/:id/actions/resolve", h.Resolve)
		ransomware.POST("/alerts/:id/actions/resume-backups", h.ResumeBackups)
	}
}

// ListSettings returns all ransomware settings for the organization.
// GET /api/v1/ransomware/settings
func (h *RansomwareHandler) ListSettings(c *gin.Context) {
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

	settings, err := h.store.GetRansomwareSettingsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list ransomware settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ransomware settings"})
		return
	}

	if settings == nil {
		settings = []*models.RansomwareSettings{}
	}

	c.JSON(http.StatusOK, gin.H{"settings": settings})
}

// GetSettingsBySchedule returns ransomware settings for a specific schedule.
// GET /api/v1/ransomware/settings/schedule/:schedule_id
func (h *RansomwareHandler) GetSettingsBySchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	scheduleIDParam := c.Param("schedule_id")
	scheduleID, err := uuid.Parse(scheduleIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	// Verify schedule belongs to user's org
	schedule, err := h.store.GetScheduleByID(c.Request.Context(), scheduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	settings, err := h.store.GetRansomwareSettingsByScheduleID(c.Request.Context(), scheduleID)
	if err != nil {
		// Return default settings if none exist
		settings = models.DefaultRansomwareSettings(scheduleID)
		settings.ID = uuid.Nil // Indicate these are defaults, not persisted
	}

	c.JSON(http.StatusOK, settings)
}

// CreateSettingsRequest is the request body for creating ransomware settings.
type CreateSettingsRequest struct {
	ScheduleID              uuid.UUID `json:"schedule_id" binding:"required"`
	Enabled                 bool      `json:"enabled"`
	ChangeThresholdPercent  int       `json:"change_threshold_percent"`
	ExtensionsToDetect      []string  `json:"extensions_to_detect,omitempty"`
	EntropyDetectionEnabled bool      `json:"entropy_detection_enabled"`
	EntropyThreshold        float64   `json:"entropy_threshold"`
	AutoPauseOnAlert        bool      `json:"auto_pause_on_alert"`
}

// CreateSettings creates new ransomware settings for a schedule.
// POST /api/v1/ransomware/settings
func (h *RansomwareHandler) CreateSettings(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureRansomwareProtect) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify schedule belongs to user's org
	schedule, err := h.store.GetScheduleByID(c.Request.Context(), req.ScheduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Check if settings already exist
	existing, _ := h.store.GetRansomwareSettingsByScheduleID(c.Request.Context(), req.ScheduleID)
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "ransomware settings already exist for this schedule"})
		return
	}

	// Validate thresholds
	if req.ChangeThresholdPercent <= 0 || req.ChangeThresholdPercent > 100 {
		req.ChangeThresholdPercent = 30 // Default
	}
	if req.EntropyThreshold <= 0 || req.EntropyThreshold > 8 {
		req.EntropyThreshold = 7.5 // Default
	}

	settings := models.NewRansomwareSettings(req.ScheduleID)
	settings.Enabled = req.Enabled
	settings.ChangeThresholdPercent = req.ChangeThresholdPercent
	settings.ExtensionsToDetect = req.ExtensionsToDetect
	settings.EntropyDetectionEnabled = req.EntropyDetectionEnabled
	settings.EntropyThreshold = req.EntropyThreshold
	settings.AutoPauseOnAlert = req.AutoPauseOnAlert

	if err := h.store.CreateRansomwareSettings(c.Request.Context(), settings); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", req.ScheduleID.String()).Msg("failed to create ransomware settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create ransomware settings"})
		return
	}

	h.logger.Info().
		Str("settings_id", settings.ID.String()).
		Str("schedule_id", req.ScheduleID.String()).
		Msg("ransomware settings created")

	c.JSON(http.StatusCreated, settings)
}

// UpdateSettingsRequest is the request body for updating ransomware settings.
type UpdateSettingsRequest struct {
	Enabled                 *bool    `json:"enabled,omitempty"`
	ChangeThresholdPercent  *int     `json:"change_threshold_percent,omitempty"`
	ExtensionsToDetect      []string `json:"extensions_to_detect,omitempty"`
	EntropyDetectionEnabled *bool    `json:"entropy_detection_enabled,omitempty"`
	EntropyThreshold        *float64 `json:"entropy_threshold,omitempty"`
	AutoPauseOnAlert        *bool    `json:"auto_pause_on_alert,omitempty"`
}

// UpdateSettings updates existing ransomware settings.
// PUT /api/v1/ransomware/settings/:id
func (h *RansomwareHandler) UpdateSettings(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureRansomwareProtect) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settings ID"})
		return
	}

	// Get existing settings
	settings, err := h.store.GetRansomwareSettingsByScheduleID(c.Request.Context(), id)
	if err != nil {
		// Try by ID if schedule lookup fails
		// For now, return not found
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware settings not found"})
		return
	}

	// Verify access
	schedule, err := h.store.GetScheduleByID(c.Request.Context(), settings.ScheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware settings not found"})
		return
	}

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.ChangeThresholdPercent != nil {
		if *req.ChangeThresholdPercent > 0 && *req.ChangeThresholdPercent <= 100 {
			settings.ChangeThresholdPercent = *req.ChangeThresholdPercent
		}
	}
	if req.ExtensionsToDetect != nil {
		settings.ExtensionsToDetect = req.ExtensionsToDetect
	}
	if req.EntropyDetectionEnabled != nil {
		settings.EntropyDetectionEnabled = *req.EntropyDetectionEnabled
	}
	if req.EntropyThreshold != nil {
		if *req.EntropyThreshold > 0 && *req.EntropyThreshold <= 8 {
			settings.EntropyThreshold = *req.EntropyThreshold
		}
	}
	if req.AutoPauseOnAlert != nil {
		settings.AutoPauseOnAlert = *req.AutoPauseOnAlert
	}

	if err := h.store.UpdateRansomwareSettings(c.Request.Context(), settings); err != nil {
		h.logger.Error().Err(err).Str("settings_id", id.String()).Msg("failed to update ransomware settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update ransomware settings"})
		return
	}

	h.logger.Info().Str("settings_id", id.String()).Msg("ransomware settings updated")

	c.JSON(http.StatusOK, settings)
}

// DeleteSettings deletes ransomware settings.
// DELETE /api/v1/ransomware/settings/:id
func (h *RansomwareHandler) DeleteSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid settings ID"})
		return
	}

	// Get settings to verify access
	settings, err := h.store.GetRansomwareSettingsByScheduleID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware settings not found"})
		return
	}

	// Verify access
	schedule, err := h.store.GetScheduleByID(c.Request.Context(), settings.ScheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	if agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware settings not found"})
		return
	}

	if err := h.store.DeleteRansomwareSettings(c.Request.Context(), settings.ID); err != nil {
		h.logger.Error().Err(err).Str("settings_id", id.String()).Msg("failed to delete ransomware settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete ransomware settings"})
		return
	}

	h.logger.Info().Str("settings_id", id.String()).Msg("ransomware settings deleted")
	c.JSON(http.StatusOK, gin.H{"message": "ransomware settings deleted"})
}

// Alert Handlers

// ListAlerts returns all ransomware alerts for the organization.
// GET /api/v1/ransomware/alerts
func (h *RansomwareHandler) ListAlerts(c *gin.Context) {
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

	alerts, err := h.store.GetRansomwareAlertsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list ransomware alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ransomware alerts"})
		return
	}

	if alerts == nil {
		alerts = []*models.RansomwareAlert{}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// ListActiveAlerts returns active ransomware alerts for the organization.
// GET /api/v1/ransomware/alerts/active
func (h *RansomwareHandler) ListActiveAlerts(c *gin.Context) {
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

	alerts, err := h.store.GetActiveRansomwareAlertsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list active ransomware alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list ransomware alerts"})
		return
	}

	if alerts == nil {
		alerts = []*models.RansomwareAlert{}
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// CountActiveAlerts returns the count of active ransomware alerts.
// GET /api/v1/ransomware/alerts/count
func (h *RansomwareHandler) CountActiveAlerts(c *gin.Context) {
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

	count, err := h.store.GetActiveRansomwareAlertCountByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to count ransomware alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count ransomware alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// GetAlert returns a specific ransomware alert by ID.
// GET /api/v1/ransomware/alerts/:id
func (h *RansomwareHandler) GetAlert(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	alert, err := h.store.GetRansomwareAlertByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to get ransomware alert")
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// Investigate marks an alert as under investigation.
// POST /api/v1/ransomware/alerts/:id/actions/investigate
func (h *RansomwareHandler) Investigate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	alert, err := h.store.GetRansomwareAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	if !alert.IsActive() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alert is not in an active state"})
		return
	}

	alert.Investigate()

	if err := h.store.UpdateRansomwareAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to update ransomware alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert"})
		return
	}

	h.logger.Info().
		Str("alert_id", id.String()).
		Str("user_id", user.ID.String()).
		Msg("ransomware alert marked as investigating")

	c.JSON(http.StatusOK, alert)
}

// ResolutionRequest is the request body for resolving an alert.
type ResolutionRequest struct {
	Resolution string `json:"resolution" binding:"required,min=1"`
}

// MarkFalsePositive marks an alert as a false positive.
// POST /api/v1/ransomware/alerts/:id/actions/false-positive
func (h *RansomwareHandler) MarkFalsePositive(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	var req ResolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	alert, err := h.store.GetRansomwareAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	alert.MarkFalsePositive(user.ID, req.Resolution)

	if err := h.store.UpdateRansomwareAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to update ransomware alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert"})
		return
	}

	h.logger.Info().
		Str("alert_id", id.String()).
		Str("user_id", user.ID.String()).
		Msg("ransomware alert marked as false positive")

	c.JSON(http.StatusOK, alert)
}

// MarkConfirmed marks an alert as confirmed ransomware.
// POST /api/v1/ransomware/alerts/:id/actions/confirm
func (h *RansomwareHandler) MarkConfirmed(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	var req ResolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	alert, err := h.store.GetRansomwareAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	alert.MarkConfirmed(user.ID, req.Resolution)

	if err := h.store.UpdateRansomwareAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to update ransomware alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert"})
		return
	}

	h.logger.Warn().
		Str("alert_id", id.String()).
		Str("user_id", user.ID.String()).
		Msg("ransomware alert CONFIRMED")

	c.JSON(http.StatusOK, alert)
}

// Resolve marks an alert as resolved.
// POST /api/v1/ransomware/alerts/:id/actions/resolve
func (h *RansomwareHandler) Resolve(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	var req ResolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	alert, err := h.store.GetRansomwareAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	alert.Resolve(user.ID, req.Resolution)

	if err := h.store.UpdateRansomwareAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to update ransomware alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert"})
		return
	}

	h.logger.Info().
		Str("alert_id", id.String()).
		Str("user_id", user.ID.String()).
		Msg("ransomware alert resolved")

	c.JSON(http.StatusOK, alert)
}

// ResumeBackups resumes backups that were paused due to ransomware detection.
// POST /api/v1/ransomware/alerts/:id/actions/resume-backups
func (h *RansomwareHandler) ResumeBackups(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	alert, err := h.store.GetRansomwareAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "ransomware alert not found"})
		return
	}

	if !alert.BackupsPaused {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backups are not paused"})
		return
	}

	// Resume the schedule
	if err := h.store.ResumeSchedule(c.Request.Context(), alert.ScheduleID); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", alert.ScheduleID.String()).Msg("failed to resume schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to resume backups"})
		return
	}

	alert.ResumeBackups()

	if err := h.store.UpdateRansomwareAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to update ransomware alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update alert"})
		return
	}

	h.logger.Info().
		Str("alert_id", id.String()).
		Str("schedule_id", alert.ScheduleID.String()).
		Str("user_id", user.ID.String()).
		Msg("backups resumed after ransomware alert")

	c.JSON(http.StatusOK, alert)
}
