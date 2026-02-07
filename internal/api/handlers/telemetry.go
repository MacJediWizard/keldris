package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// CurrentConsentVersion is the current privacy policy version for telemetry.
const CurrentConsentVersion = "1.0"

// TelemetryStore defines the interface for telemetry persistence operations.
type TelemetryStore interface {
	GetTelemetrySettings(ctx context.Context) (*telemetry.Settings, error)
	UpdateTelemetrySettings(ctx context.Context, settings *telemetry.Settings) error
	EnableTelemetry(ctx context.Context, consentVersion string) error
	DisableTelemetry(ctx context.Context) error
	UpdateTelemetryLastSent(ctx context.Context, sentAt time.Time, data *telemetry.TelemetryData) error
	CollectTelemetryData(ctx context.Context) (*telemetry.TelemetryCounts, *telemetry.TelemetryFeatures, error)
}

// TelemetryHandler handles telemetry-related HTTP endpoints.
type TelemetryHandler struct {
	store   TelemetryStore
	service *telemetry.Service
	logger  zerolog.Logger
}

// NewTelemetryHandler creates a new TelemetryHandler.
func NewTelemetryHandler(store TelemetryStore, service *telemetry.Service, logger zerolog.Logger) *TelemetryHandler {
	return &TelemetryHandler{
		store:   store,
		service: service,
		logger:  logger.With().Str("component", "telemetry_handler").Logger(),
	}
}

// RegisterRoutes registers telemetry routes on the given router group.
func (h *TelemetryHandler) RegisterRoutes(r *gin.RouterGroup) {
	telem := r.Group("/telemetry")
	{
		// Get current telemetry status and settings
		telem.GET("", h.GetStatus)

		// Update telemetry settings (enable/disable)
		telem.PUT("", h.UpdateSettings)

		// Preview what telemetry data would be sent
		telem.GET("/preview", h.Preview)

		// Get privacy explanation
		telem.GET("/privacy", h.GetPrivacyExplanation)

		// Manually trigger telemetry send (for testing)
		telem.POST("/send", h.TriggerSend)
	}
}

// GetStatus returns the current telemetry settings and status.
//
//	@Summary		Get telemetry status
//	@Description	Returns the current telemetry settings, last data sent, and privacy explanation
//	@Tags			Telemetry
//	@Produce		json
//	@Success		200	{object}	telemetry.TelemetryStatusResponse
//	@Failure		403	{object}	gin.H
//	@Failure		500	{object}	gin.H
//	@Router			/api/v1/telemetry [get]
func (h *TelemetryHandler) GetStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins/owners can view telemetry settings
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	settings, err := h.store.GetTelemetrySettings(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get telemetry settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get telemetry settings"})
		return
	}

	c.JSON(http.StatusOK, telemetry.TelemetryStatusResponse{
		Settings:    *settings,
		Explanation: telemetry.GetPrivacyExplanation(),
	})
}

// UpdateSettings updates the telemetry settings.
//
//	@Summary		Update telemetry settings
//	@Description	Enable or disable telemetry collection
//	@Tags			Telemetry
//	@Accept			json
//	@Produce		json
//	@Param			body	body		telemetry.UpdateSettingsRequest	true	"Telemetry settings update"
//	@Success		200		{object}	telemetry.Settings
//	@Failure		400		{object}	gin.H
//	@Failure		403		{object}	gin.H
//	@Failure		500		{object}	gin.H
//	@Router			/api/v1/telemetry [put]
func (h *TelemetryHandler) UpdateSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins/owners can modify telemetry settings
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	var req telemetry.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Enabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "enabled field is required"})
		return
	}

	if *req.Enabled {
		// Enable telemetry with consent
		if err := h.store.EnableTelemetry(c.Request.Context(), CurrentConsentVersion); err != nil {
			h.logger.Error().Err(err).Msg("failed to enable telemetry")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable telemetry"})
			return
		}
		h.logger.Info().
			Str("user_id", user.ID.String()).
			Msg("telemetry enabled by user")
	} else {
		// Disable telemetry
		if err := h.store.DisableTelemetry(c.Request.Context()); err != nil {
			h.logger.Error().Err(err).Msg("failed to disable telemetry")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable telemetry"})
			return
		}
		h.logger.Info().
			Str("user_id", user.ID.String()).
			Msg("telemetry disabled by user")
	}

	// Return updated settings
	settings, err := h.store.GetTelemetrySettings(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get updated telemetry settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get updated settings"})
		return
	}

	// Update the service settings if available
	if h.service != nil {
		h.service.SetSettings(*settings)
	}

	c.JSON(http.StatusOK, settings)
}

// Preview returns what telemetry data would be sent without actually sending it.
//
//	@Summary		Preview telemetry data
//	@Description	Shows exactly what telemetry data would be collected and sent
//	@Tags			Telemetry
//	@Produce		json
//	@Success		200	{object}	telemetry.TelemetryPreviewResponse
//	@Failure		403	{object}	gin.H
//	@Failure		500	{object}	gin.H
//	@Router			/api/v1/telemetry/preview [get]
func (h *TelemetryHandler) Preview(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins/owners can view telemetry preview
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	// Get current settings for install_id
	settings, err := h.store.GetTelemetrySettings(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get telemetry settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get telemetry settings"})
		return
	}

	// Collect the data that would be sent
	counts, features, err := h.store.CollectTelemetryData(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to collect telemetry data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to collect telemetry data"})
		return
	}

	// Build preview data
	data := &telemetry.TelemetryData{
		InstallID:   settings.InstallID,
		CollectedAt: time.Now().UTC(),
		Version:     "preview",
		Counts:      *counts,
		Features:    *features,
	}

	// If service is available, use its version info
	if h.service != nil {
		svcSettings := h.service.GetSettings()
		data.InstallID = svcSettings.InstallID
	}

	c.JSON(http.StatusOK, telemetry.TelemetryPreviewResponse{
		Data:        data,
		Explanation: telemetry.GetPrivacyExplanation(),
	})
}

// GetPrivacyExplanation returns the privacy explanation for telemetry.
//
//	@Summary		Get privacy explanation
//	@Description	Returns a detailed explanation of what telemetry collects and why
//	@Tags			Telemetry
//	@Produce		json
//	@Success		200	{object}	gin.H
//	@Router			/api/v1/telemetry/privacy [get]
func (h *TelemetryHandler) GetPrivacyExplanation(c *gin.Context) {
	// This endpoint is public - anyone can view the privacy explanation
	c.JSON(http.StatusOK, gin.H{
		"explanation":      telemetry.GetPrivacyExplanation(),
		"consent_version":  CurrentConsentVersion,
		"default_enabled":  false,
		"opt_in_required":  true,
	})
}

// TriggerSend manually triggers telemetry collection and sending.
//
//	@Summary		Trigger telemetry send
//	@Description	Manually triggers telemetry collection and sending (admin only)
//	@Tags			Telemetry
//	@Produce		json
//	@Success		200	{object}	gin.H
//	@Failure		400	{object}	gin.H
//	@Failure		403	{object}	gin.H
//	@Failure		500	{object}	gin.H
//	@Router			/api/v1/telemetry/send [post]
func (h *TelemetryHandler) TriggerSend(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins/owners can trigger telemetry send
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	// Check if telemetry is enabled
	settings, err := h.store.GetTelemetrySettings(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get telemetry settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get telemetry settings"})
		return
	}

	if !settings.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "telemetry is not enabled",
			"message": "Enable telemetry first to send data",
		})
		return
	}

	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "telemetry service not available"})
		return
	}

	// Collect and send
	data, err := h.service.Collect(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to collect telemetry data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to collect telemetry data"})
		return
	}

	if err := h.service.Send(c.Request.Context(), data); err != nil {
		h.logger.Error().Err(err).Msg("failed to send telemetry data")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to send telemetry data",
			"message": err.Error(),
		})
		return
	}

	// Update last sent in database
	if err := h.store.UpdateTelemetryLastSent(c.Request.Context(), time.Now(), data); err != nil {
		h.logger.Warn().Err(err).Msg("failed to update telemetry last sent")
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Msg("telemetry manually sent by user")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "telemetry data sent successfully",
		"data":    data,
	})
}
