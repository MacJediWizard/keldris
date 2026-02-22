package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// NotificationStore defines the interface for notification persistence operations.
type NotificationStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetNotificationChannelsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationChannel, error)
	GetNotificationChannelByID(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error)
	CreateNotificationChannel(ctx context.Context, channel *models.NotificationChannel) error
	UpdateNotificationChannel(ctx context.Context, channel *models.NotificationChannel) error
	DeleteNotificationChannel(ctx context.Context, id uuid.UUID) error
	GetNotificationPreferencesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.NotificationPreference, error)
	GetNotificationPreferencesByChannelID(ctx context.Context, channelID uuid.UUID) ([]*models.NotificationPreference, error)
	CreateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error
	UpdateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error
	DeleteNotificationPreference(ctx context.Context, id uuid.UUID) error
	GetNotificationLogsByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.NotificationLog, error)
}

// NotificationsHandler handles notification-related HTTP endpoints.
type NotificationsHandler struct {
	store      NotificationStore
	keyManager *crypto.KeyManager
	logger     zerolog.Logger
	env        config.Environment
	service NotificationService
}

// NewNotificationsHandler creates a new NotificationsHandler.
func NewNotificationsHandler(store NotificationStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *NotificationsHandler {
	return &NotificationsHandler{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "notifications_handler").Logger(),
		env:        config.EnvDevelopment,
	}
}

// NewNotificationsHandlerWithEnv creates a new NotificationsHandler with an explicit environment.
func NewNotificationsHandlerWithEnv(store NotificationStore, keyManager *crypto.KeyManager, logger zerolog.Logger, env config.Environment) *NotificationsHandler {
	return &NotificationsHandler{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "notifications_handler").Logger(),
		env:        env,
	}
}

// RegisterRoutes registers notification routes available on all tiers.
// These allow free-tier users to manage email notification channels and preferences.
// SetNotificationService sets the notification service for testing channels.
func (h *NotificationsHandler) SetNotificationService(service NotificationService) {
	h.service = service
}

// NotificationService defines the interface for testing notification channels.
type NotificationService interface {
	TestChannel(channel *models.NotificationChannel) error
}

	store  NotificationStore
	logger zerolog.Logger
}

// NewNotificationsHandler creates a new NotificationsHandler.
func NewNotificationsHandler(store NotificationStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *NotificationsHandler {
	return &NotificationsHandler{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "notifications_handler").Logger(),
	}
}

// RegisterRoutes registers notification routes on the given router group.
func (h *NotificationsHandler) RegisterRoutes(r *gin.RouterGroup) {
	notifications := r.Group("/notifications")
	{
		// Channels (read + create/update/delete for email)
		// Channels
		notifications.GET("/channels", h.ListChannels)
		notifications.POST("/channels", h.CreateChannel)
		notifications.GET("/channels/:id", h.GetChannel)
		notifications.PUT("/channels/:id", h.UpdateChannel)
		notifications.DELETE("/channels/:id", h.DeleteChannel)
		notifications.POST("/channels/:id/test", h.TestChannel)

		// Channel types info
		notifications.GET("/channel-types", h.ListChannelTypes)

		// Preferences
		notifications.GET("/preferences", h.ListPreferences)
		notifications.POST("/preferences", h.CreatePreference)
		notifications.PUT("/preferences/:id", h.UpdatePreference)
		notifications.DELETE("/preferences/:id", h.DeletePreference)

		// Logs
		notifications.GET("/logs", h.ListLogs)
	}
}

// ListChannels returns all notification channels for the authenticated user's organization.
// GET /api/v1/notifications/channels
func (h *NotificationsHandler) ListChannels(c *gin.Context) {
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

	channels, err := h.store.GetNotificationChannelsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list notification channels")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notification channels"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"channels": channels})
}

// GetChannel returns a specific notification channel by ID.
// GET /api/v1/notifications/channels/:id
func (h *NotificationsHandler) GetChannel(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", id.String()).Msg("failed to get notification channel")
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if channel.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	// Get preferences for this channel
	prefs, err := h.store.GetNotificationPreferencesByChannelID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", id.String()).Msg("failed to get channel preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get channel preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"channel":     channel,
		"preferences": prefs,
	})
}

// CreateChannelRequest is the request body for creating a notification channel.
type CreateChannelRequest struct {
	Name   string                      `json:"name" binding:"required,min=1,max=255"`
	Type   models.NotificationChannelType `json:"type" binding:"required"`
	Config json.RawMessage             `json:"config" binding:"required"`
}

// CreateChannel creates a new notification channel.
// POST /api/v1/notifications/channels
func (h *NotificationsHandler) CreateChannel(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate channel type
	if !isValidChannelType(req.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel type"})
		return
	}

	// Validate webhook URL for SSRF protection
	if req.Type == models.ChannelTypeWebhook {
		if err := h.validateWebhookConfig(req.Config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	configEncrypted, err := h.keyManager.Encrypt([]byte(req.Config))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt channel config")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt config"})
		return
	}

	channel := models.NewNotificationChannel(dbUser.OrgID, req.Name, req.Type, configEncrypted)
	// TODO: Encrypt config before storing
	// For now, store as-is (should use crypto/aes.go when implemented)
	channel := models.NewNotificationChannel(dbUser.OrgID, req.Name, req.Type, req.Config)

	if err := h.store.CreateNotificationChannel(c.Request.Context(), channel); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create notification channel")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification channel"})
		return
	}

	h.logger.Info().
		Str("channel_id", channel.ID.String()).
		Str("name", req.Name).
		Str("type", string(req.Type)).
		Str("org_id", dbUser.OrgID.String()).
		Msg("notification channel created")

	c.JSON(http.StatusCreated, channel)
}

// UpdateChannelRequest is the request body for updating a notification channel.
type UpdateChannelRequest struct {
	Name    *string         `json:"name,omitempty"`
	Config  json.RawMessage `json:"config,omitempty"`
	Enabled *bool           `json:"enabled,omitempty"`
}

// UpdateChannel updates an existing notification channel.
// PUT /api/v1/notifications/channels/:id
func (h *NotificationsHandler) UpdateChannel(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if channel.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Config != nil {
		// Validate webhook URL for SSRF protection on config updates
		if channel.Type == models.ChannelTypeWebhook {
			if err := h.validateWebhookConfig(req.Config); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}

		configEncrypted, err := h.keyManager.Encrypt([]byte(req.Config))
		if err != nil {
			h.logger.Error().Err(err).Msg("failed to encrypt channel config")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt config"})
			return
		}
		channel.ConfigEncrypted = configEncrypted
		channel.ConfigEncrypted = req.Config
	}
	if req.Enabled != nil {
		channel.Enabled = *req.Enabled
	}

	if err := h.store.UpdateNotificationChannel(c.Request.Context(), channel); err != nil {
		h.logger.Error().Err(err).Str("channel_id", id.String()).Msg("failed to update notification channel")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification channel"})
		return
	}

	h.logger.Info().Str("channel_id", id.String()).Msg("notification channel updated")
	c.JSON(http.StatusOK, channel)
}

// DeleteChannel removes a notification channel.
// DELETE /api/v1/notifications/channels/:id
func (h *NotificationsHandler) DeleteChannel(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if channel.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	if err := h.store.DeleteNotificationChannel(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("channel_id", id.String()).Msg("failed to delete notification channel")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete notification channel"})
		return
	}

	h.logger.Info().Str("channel_id", id.String()).Msg("notification channel deleted")
	c.JSON(http.StatusOK, gin.H{"message": "notification channel deleted"})
}

// ListPreferences returns all notification preferences for the authenticated user's organization.
// GET /api/v1/notifications/preferences
func (h *NotificationsHandler) ListPreferences(c *gin.Context) {
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

	prefs, err := h.store.GetNotificationPreferencesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list notification preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notification preferences"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"preferences": prefs})
}

// CreatePreferenceRequest is the request body for creating a notification preference.
type CreatePreferenceRequest struct {
	ChannelID uuid.UUID                  `json:"channel_id" binding:"required"`
	EventType models.NotificationEventType `json:"event_type" binding:"required"`
	Enabled   bool                       `json:"enabled"`
}

// CreatePreference creates a new notification preference.
// POST /api/v1/notifications/preferences
func (h *NotificationsHandler) CreatePreference(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if !isValidEventType(req.EventType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event type"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Verify channel belongs to org
	channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), req.ChannelID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification channel not found"})
		return
	}
	if channel.OrgID != dbUser.OrgID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "notification channel not found"})
		return
	}

	pref := models.NewNotificationPreference(dbUser.OrgID, req.ChannelID, req.EventType)
	pref.Enabled = req.Enabled

	if err := h.store.CreateNotificationPreference(c.Request.Context(), pref); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", req.ChannelID.String()).
			Str("event_type", string(req.EventType)).
			Msg("failed to create notification preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification preference"})
		return
	}

	h.logger.Info().
		Str("preference_id", pref.ID.String()).
		Str("channel_id", req.ChannelID.String()).
		Str("event_type", string(req.EventType)).
		Msg("notification preference created")

	c.JSON(http.StatusCreated, pref)
}

// UpdatePreferenceRequest is the request body for updating a notification preference.
type UpdatePreferenceRequest struct {
	Enabled bool `json:"enabled"`
}

// UpdatePreference updates an existing notification preference.
// PUT /api/v1/notifications/preferences/:id
func (h *NotificationsHandler) UpdatePreference(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid preference ID"})
		return
	}

	var req UpdatePreferenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Get all preferences to find this one and verify ownership
	prefs, err := h.store.GetNotificationPreferencesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	var pref *models.NotificationPreference
	for _, p := range prefs {
		if p.ID == id {
			pref = p
			break
		}
	}

	if pref == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification preference not found"})
		return
	}

	pref.Enabled = req.Enabled

	if err := h.store.UpdateNotificationPreference(c.Request.Context(), pref); err != nil {
		h.logger.Error().Err(err).Str("preference_id", id.String()).Msg("failed to update notification preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update notification preference"})
		return
	}

	h.logger.Info().Str("preference_id", id.String()).Bool("enabled", req.Enabled).Msg("notification preference updated")
	c.JSON(http.StatusOK, pref)
}

// DeletePreference removes a notification preference.
// DELETE /api/v1/notifications/preferences/:id
func (h *NotificationsHandler) DeletePreference(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid preference ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	// Get all preferences to verify ownership
	prefs, err := h.store.GetNotificationPreferencesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get preferences")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	found := false
	for _, p := range prefs {
		if p.ID == id {
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification preference not found"})
		return
	}

	if err := h.store.DeleteNotificationPreference(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("preference_id", id.String()).Msg("failed to delete notification preference")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete notification preference"})
		return
	}

	h.logger.Info().Str("preference_id", id.String()).Msg("notification preference deleted")
	c.JSON(http.StatusOK, gin.H{"message": "notification preference deleted"})
}

// ListLogs returns notification logs for the authenticated user's organization.
// GET /api/v1/notifications/logs
func (h *NotificationsHandler) ListLogs(c *gin.Context) {
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

	// Default limit of 100 logs
	limit := 100
	logs, err := h.store.GetNotificationLogsByOrgID(c.Request.Context(), dbUser.OrgID, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list notification logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list notification logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// validateWebhookConfig parses webhook config and validates the URL for SSRF protection.
func (h *NotificationsHandler) validateWebhookConfig(rawConfig json.RawMessage) error {
	var webhookConfig models.WebhookChannelConfig
	if err := json.Unmarshal(rawConfig, &webhookConfig); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}

	requireHTTPS := h.env == config.EnvProduction || h.env == config.EnvStaging
	return notifications.ValidateWebhookURL(webhookConfig.URL, requireHTTPS)
}

// TestChannel sends a test notification to verify the channel configuration.
// POST /api/v1/notifications/channels/:id/test
func (h *NotificationsHandler) TestChannel(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel ID"})
		return
	}

	channel, err := h.store.GetNotificationChannelByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("channel_id", id.String()).Msg("failed to get notification channel")
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if channel.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "notification channel not found"})
		return
	}

	if h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "notification service not configured"})
		return
	}

	if err := h.service.TestChannel(channel); err != nil {
		h.logger.Error().Err(err).
			Str("channel_id", id.String()).
			Str("channel_type", string(channel.Type)).
			Msg("test notification failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "test notification failed",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("channel_id", id.String()).
		Str("channel_type", string(channel.Type)).
		Msg("test notification sent successfully")

	c.JSON(http.StatusOK, gin.H{
		"message": "test notification sent successfully",
	})
}

// ChannelTypeInfo represents information about a notification channel type.
type ChannelTypeInfo struct {
	Type        models.NotificationChannelType `json:"type"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	ConfigFields []ChannelConfigField          `json:"config_fields"`
}

// ChannelConfigField represents a configuration field for a channel type.
type ChannelConfigField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Placeholder string `json:"placeholder,omitempty"`
}

// ListChannelTypes returns information about available notification channel types.
// GET /api/v1/notifications/channel-types
func (h *NotificationsHandler) ListChannelTypes(c *gin.Context) {
	channelTypes := []ChannelTypeInfo{
		{
			Type:        models.ChannelTypeEmail,
			Name:        "Email (SMTP)",
			Description: "Send notifications via email using SMTP",
			ConfigFields: []ChannelConfigField{
				{Name: "host", Type: "string", Required: true, Description: "SMTP server hostname", Placeholder: "smtp.example.com"},
				{Name: "port", Type: "number", Required: true, Description: "SMTP server port", Placeholder: "587"},
				{Name: "username", Type: "string", Required: false, Description: "SMTP username for authentication"},
				{Name: "password", Type: "password", Required: false, Description: "SMTP password for authentication"},
				{Name: "from", Type: "email", Required: true, Description: "From email address", Placeholder: "alerts@example.com"},
				{Name: "tls", Type: "boolean", Required: false, Description: "Enable TLS encryption"},
			},
		},
		{
			Type:        models.ChannelTypeSlack,
			Name:        "Slack",
			Description: "Send notifications to Slack channels via webhooks",
			ConfigFields: []ChannelConfigField{
				{Name: "webhook_url", Type: "url", Required: true, Description: "Slack Incoming Webhook URL", Placeholder: "https://hooks.slack.com/services/..."},
				{Name: "channel", Type: "string", Required: false, Description: "Override default channel (e.g., #alerts)"},
				{Name: "username", Type: "string", Required: false, Description: "Bot username displayed in Slack", Placeholder: "Keldris Backup"},
				{Name: "icon_emoji", Type: "string", Required: false, Description: "Emoji to use as icon (e.g., :shield:)"},
			},
		},
		{
			Type:        models.ChannelTypeTeams,
			Name:        "Microsoft Teams",
			Description: "Send notifications to Microsoft Teams channels via webhooks",
			ConfigFields: []ChannelConfigField{
				{Name: "webhook_url", Type: "url", Required: true, Description: "Teams Incoming Webhook URL", Placeholder: "https://outlook.office.com/webhook/..."},
			},
		},
		{
			Type:        models.ChannelTypeDiscord,
			Name:        "Discord",
			Description: "Send notifications to Discord channels via webhooks",
			ConfigFields: []ChannelConfigField{
				{Name: "webhook_url", Type: "url", Required: true, Description: "Discord Webhook URL", Placeholder: "https://discord.com/api/webhooks/..."},
				{Name: "username", Type: "string", Required: false, Description: "Bot username displayed in Discord", Placeholder: "Keldris Backup"},
				{Name: "avatar_url", Type: "url", Required: false, Description: "Avatar image URL for the bot"},
			},
		},
		{
			Type:        models.ChannelTypePagerDuty,
			Name:        "PagerDuty",
			Description: "Send alerts to PagerDuty for incident management",
			ConfigFields: []ChannelConfigField{
				{Name: "routing_key", Type: "string", Required: true, Description: "PagerDuty Events API v2 routing key (integration key)"},
				{Name: "severity", Type: "select", Required: false, Description: "Default severity level (critical, error, warning, info)"},
				{Name: "component", Type: "string", Required: false, Description: "Component name for the alert"},
				{Name: "group", Type: "string", Required: false, Description: "Logical grouping of alerts"},
				{Name: "class", Type: "string", Required: false, Description: "Class/type of alert"},
			},
		},
		{
			Type:        models.ChannelTypeWebhook,
			Name:        "Generic Webhook",
			Description: "Send notifications to any HTTP endpoint",
			ConfigFields: []ChannelConfigField{
				{Name: "url", Type: "url", Required: true, Description: "Webhook URL endpoint", Placeholder: "https://api.example.com/webhooks/..."},
				{Name: "method", Type: "select", Required: false, Description: "HTTP method (default: POST)"},
				{Name: "auth_type", Type: "select", Required: false, Description: "Authentication type (none, bearer, basic)"},
				{Name: "auth_token", Type: "password", Required: false, Description: "Bearer token for authentication"},
				{Name: "basic_user", Type: "string", Required: false, Description: "Username for basic authentication"},
				{Name: "basic_pass", Type: "password", Required: false, Description: "Password for basic authentication"},
				{Name: "content_type", Type: "string", Required: false, Description: "Content-Type header (default: application/json)"},
				{Name: "template", Type: "textarea", Required: false, Description: "Custom payload template (Go template syntax)"},
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{"channel_types": channelTypes})
}

// isValidChannelType checks if a channel type is valid.
func isValidChannelType(t models.NotificationChannelType) bool {
	switch t {
	case models.ChannelTypeEmail, models.ChannelTypeSlack, models.ChannelTypeWebhook, models.ChannelTypePagerDuty, models.ChannelTypeTeams, models.ChannelTypeDiscord:
// isValidChannelType checks if a channel type is valid.
func isValidChannelType(t models.NotificationChannelType) bool {
	switch t {
	case models.ChannelTypeEmail, models.ChannelTypeSlack, models.ChannelTypeWebhook, models.ChannelTypePagerDuty:
		return true
	default:
		return false
	}
}

// isValidEventType checks if an event type is valid.
func isValidEventType(t models.NotificationEventType) bool {
	switch t {
	case models.EventBackupSuccess, models.EventBackupFailed, models.EventAgentOffline, models.EventMaintenanceScheduled:
	case models.EventBackupSuccess, models.EventBackupFailed, models.EventAgentOffline:
		return true
	default:
		return false
	}
}
