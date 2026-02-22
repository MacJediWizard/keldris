package handlers

import (
	"errors"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/updates"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// UpdatesHandler handles update checking HTTP endpoints.
type UpdatesHandler struct {
	checker *updates.Checker
	logger  zerolog.Logger
}

// NewUpdatesHandler creates a new UpdatesHandler.
func NewUpdatesHandler(checker *updates.Checker, logger zerolog.Logger) *UpdatesHandler {
	return &UpdatesHandler{
		checker: checker,
		logger:  logger.With().Str("component", "updates_handler").Logger(),
	}
}

// RegisterRoutes registers update routes on the given router group.
func (h *UpdatesHandler) RegisterRoutes(r *gin.RouterGroup) {
	updatesGroup := r.Group("/updates")
	{
		updatesGroup.GET("/check", h.Check)
		updatesGroup.POST("/check", h.ForceCheck)
		updatesGroup.GET("/status", h.Status)
	}
}

// RegisterPublicRoutes registers update routes that don't require authentication.
// The check endpoint is public so the banner can be shown before login.
func (h *UpdatesHandler) RegisterPublicRoutes(r *gin.Engine) {
	r.GET("/updates/check", h.Check)
	r.GET("/updates/status", h.Status)
}

// Check returns the current update status, using cached data if available.
//
//	@Summary		Check for updates
//	@Description	Checks for available Keldris updates. Uses cached result if within check interval.
//	@Tags			Updates
//	@Produce		json
//	@Success		200	{object}	updates.UpdateInfo
//	@Failure		503	{object}	ErrorResponse	"Update checking disabled"
//	@Router			/updates/check [get]
func (h *UpdatesHandler) Check(c *gin.Context) {
	if h.checker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "update checking not configured"})
		return
	}

	info, err := h.checker.Check(c.Request.Context())
	if err != nil {
		if errors.Is(err, updates.ErrCheckDisabled) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "update checking disabled",
				"enabled": false,
			})
			return
		}
		if errors.Is(err, updates.ErrAirGapMode) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":        "update checking disabled in air-gap mode",
				"enabled":      false,
				"air_gap_mode": true,
			})
			return
		}

		h.logger.Warn().Err(err).Msg("failed to check for updates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check for updates"})
		return
	}

	c.JSON(http.StatusOK, info)
}

// ForceCheck forces an update check, bypassing the cache.
//
//	@Summary		Force update check
//	@Description	Forces an update check, bypassing the cache. Requires admin privileges.
//	@Tags			Updates
//	@Produce		json
//	@Success		200	{object}	updates.UpdateInfo
//	@Failure		403	{object}	ErrorResponse	"Admin access required"
//	@Failure		503	{object}	ErrorResponse	"Update checking disabled"
//	@Router			/updates/check [post]
func (h *UpdatesHandler) ForceCheck(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Only admins can force a check
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	if h.checker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "update checking not configured"})
		return
	}

	info, err := h.checker.ForceCheck(c.Request.Context())
	if err != nil {
		if errors.Is(err, updates.ErrCheckDisabled) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "update checking disabled",
				"enabled": false,
			})
			return
		}
		if errors.Is(err, updates.ErrAirGapMode) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":        "update checking disabled in air-gap mode",
				"enabled":      false,
				"air_gap_mode": true,
			})
			return
		}

		h.logger.Warn().Err(err).Msg("failed to force update check")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check for updates"})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Bool("update_available", info.UpdateAvailable).
		Str("latest_version", info.LatestVersion).
		Msg("forced update check")

	c.JSON(http.StatusOK, info)
}

// UpdateStatusResponse contains the update checker status.
type UpdateStatusResponse struct {
	Enabled        bool                `json:"enabled"`
	AirGapMode     bool                `json:"air_gap_mode"`
	UpdateInfo     *updates.UpdateInfo `json:"update_info,omitempty"`
	ChangelogURL   string              `json:"changelog_url"`
	UpgradeDocsURL string              `json:"upgrade_docs_url"`
}

// Status returns the update checker status and cached info.
//
//	@Summary		Get update checker status
//	@Description	Returns the update checker status and cached update info if available.
//	@Tags			Updates
//	@Produce		json
//	@Success		200	{object}	UpdateStatusResponse
//	@Router			/updates/status [get]
func (h *UpdatesHandler) Status(c *gin.Context) {
	if h.checker == nil {
		c.JSON(http.StatusOK, UpdateStatusResponse{
			Enabled:        false,
			ChangelogURL:   updates.ChangelogURL,
			UpgradeDocsURL: updates.UpgradeInstructionsURL,
		})
		return
	}

	c.JSON(http.StatusOK, UpdateStatusResponse{
		Enabled:        h.checker.IsEnabled(),
		AirGapMode:     !h.checker.IsEnabled() && h.checker.GetCachedInfo() == nil,
		UpdateInfo:     h.checker.GetCachedInfo(),
		ChangelogURL:   updates.ChangelogURL,
		UpgradeDocsURL: updates.UpgradeInstructionsURL,
	})
}
