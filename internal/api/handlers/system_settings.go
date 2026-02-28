package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SystemSettingsStore defines the interface for system settings persistence operations.
type SystemSettingsStore interface {
	GetAllSettings(ctx context.Context, orgID uuid.UUID) (*settings.SystemSettingsResponse, error)
	GetSMTPSettings(ctx context.Context, orgID uuid.UUID) (*settings.SMTPSettings, error)
	UpdateSMTPSettings(ctx context.Context, orgID uuid.UUID, smtp *settings.SMTPSettings) error
	GetOIDCSettings(ctx context.Context, orgID uuid.UUID) (*settings.OIDCSettings, error)
	UpdateOIDCSettings(ctx context.Context, orgID uuid.UUID, oidc *settings.OIDCSettings) error
	GetStorageDefaultSettings(ctx context.Context, orgID uuid.UUID) (*settings.StorageDefaultSettings, error)
	UpdateStorageDefaultSettings(ctx context.Context, orgID uuid.UUID, storage *settings.StorageDefaultSettings) error
	GetSecuritySettings(ctx context.Context, orgID uuid.UUID) (*settings.SecuritySettings, error)
	UpdateSecuritySettings(ctx context.Context, orgID uuid.UUID, security *settings.SecuritySettings) error
	CreateSettingsAuditLog(ctx context.Context, log *settings.SettingsAuditLog) error
	GetSettingsAuditLogs(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*settings.SettingsAuditLog, error)
	EnsureSystemSettingsExist(ctx context.Context, orgID uuid.UUID) error
}

// SystemSettingsHandler handles system settings HTTP endpoints.
type SystemSettingsHandler struct {
	store   SystemSettingsStore
	checker *license.FeatureChecker
	logger  zerolog.Logger
}

// NewSystemSettingsHandler creates a new SystemSettingsHandler.
func NewSystemSettingsHandler(store SystemSettingsStore, checker *license.FeatureChecker, logger zerolog.Logger) *SystemSettingsHandler {
	return &SystemSettingsHandler{
		store:   store,
		checker: checker,
		logger:  logger.With().Str("component", "system_settings_handler").Logger(),
	}
}

// RegisterRoutes registers system settings routes on the given router group.
func (h *SystemSettingsHandler) RegisterRoutes(r *gin.RouterGroup) {
	sysSettings := r.Group("/system-settings")
	{
		// All settings at once
		sysSettings.GET("", h.GetAll)

		// Individual setting categories
		sysSettings.GET("/smtp", h.GetSMTP)
		sysSettings.PUT("/smtp", h.UpdateSMTP)
		sysSettings.POST("/smtp/test", h.TestSMTP)

		sysSettings.GET("/oidc", h.GetOIDC)
		sysSettings.PUT("/oidc", h.UpdateOIDC)
		sysSettings.POST("/oidc/test", h.TestOIDC)

		sysSettings.GET("/storage", h.GetStorageDefaults)
		sysSettings.PUT("/storage", h.UpdateStorageDefaults)

		sysSettings.GET("/security", h.GetSecurity)
		sysSettings.PUT("/security", h.UpdateSecurity)

		// Audit log
		sysSettings.GET("/audit-log", h.GetAuditLog)
	}
}

// getClientIP extracts the client IP address from the request.
func getClientIP(c *gin.Context) string {
	// Try X-Forwarded-For header first (for proxied requests)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		return xff
	}
	// Try X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}
	// Fall back to RemoteAddr
	return c.ClientIP()
}

// GetAll returns all system settings for the organization.
// GET /api/v1/system-settings
func (h *SystemSettingsHandler) GetAll(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Only admins/owners can view system settings
	if !isAdmin(user.CurrentOrgRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return
	}

	// Ensure default settings exist
	if err := h.store.EnsureSystemSettingsExist(c.Request.Context(), user.CurrentOrgID); err != nil {
		h.logger.Warn().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to ensure settings exist")
	}

	allSettings, err := h.store.GetAllSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get all settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	// Mask sensitive fields before returning
	allSettings.SMTP.Password = ""
	allSettings.OIDC.ClientSecret = ""

	c.JSON(http.StatusOK, allSettings)
}

// GetSMTP returns SMTP settings for the organization.
// GET /api/v1/system-settings/smtp
func (h *SystemSettingsHandler) GetSMTP(c *gin.Context) {
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

	smtp, err := h.store.GetSMTPSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get SMTP settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get SMTP settings"})
		return
	}

	// Mask password
	smtp.Password = ""

	c.JSON(http.StatusOK, smtp)
}

// UpdateSMTP updates SMTP settings for the organization.
// PUT /api/v1/system-settings/smtp
func (h *SystemSettingsHandler) UpdateSMTP(c *gin.Context) {
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

	var req settings.UpdateSMTPSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current settings for audit log and merging
	current, err := h.store.GetSMTPSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get current SMTP settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get current settings"})
		return
	}

	// Store old value for audit (mask password)
	oldCopy := *current
	oldCopy.Password = "[MASKED]"
	oldValue, _ := json.Marshal(oldCopy)

	// Apply updates
	if req.Host != nil {
		current.Host = *req.Host
	}
	if req.Port != nil {
		current.Port = *req.Port
	}
	if req.Username != nil {
		current.Username = *req.Username
	}
	if req.Password != nil && *req.Password != "" {
		current.Password = *req.Password
	}
	if req.FromEmail != nil {
		current.FromEmail = *req.FromEmail
	}
	if req.FromName != nil {
		current.FromName = *req.FromName
	}
	if req.Encryption != nil {
		current.Encryption = *req.Encryption
	}
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.SkipTLSVerify != nil {
		current.SkipTLSVerify = *req.SkipTLSVerify
	}
	if req.ConnectionTimeout != nil {
		current.ConnectionTimeout = *req.ConnectionTimeout
	}

	// Validate
	if err := current.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save
	if err := h.store.UpdateSMTPSettings(c.Request.Context(), user.CurrentOrgID, current); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update SMTP settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	// Create audit log
	newCopy := *current
	newCopy.Password = "[MASKED]"
	newValue, _ := json.Marshal(newCopy)

	auditLog := settings.NewSettingsAuditLog(
		user.CurrentOrgID,
		settings.SettingKeySMTP,
		oldValue,
		newValue,
		user.ID,
		getClientIP(c),
	)
	if err := h.store.CreateSettingsAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create settings audit log")
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Msg("SMTP settings updated")

	// Return updated settings (masked)
	current.Password = ""
	c.JSON(http.StatusOK, current)
}

// TestSMTP tests the SMTP connection.
// POST /api/v1/system-settings/smtp/test
func (h *SystemSettingsHandler) TestSMTP(c *gin.Context) {
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

	var req settings.TestSMTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	smtp, err := h.store.GetSMTPSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get SMTP settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get SMTP settings"})
		return
	}

	if !smtp.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP is not enabled"})
		return
	}

	if smtp.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SMTP host is not configured"})
		return
	}

	// TODO: Implement actual SMTP test connection
	// For now, return a success response if validation passes
	if err := smtp.Validate(); err != nil {
		c.JSON(http.StatusOK, settings.TestSMTPResponse{
			Success: false,
			Message: "Configuration validation failed: " + err.Error(),
		})
		return
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Str("recipient", req.RecipientEmail).
		Msg("SMTP test requested")

	c.JSON(http.StatusOK, settings.TestSMTPResponse{
		Success: true,
		Message: "SMTP configuration is valid. Test email would be sent to: " + req.RecipientEmail,
	})
}

// GetOIDC returns OIDC settings for the organization.
// GET /api/v1/system-settings/oidc
func (h *SystemSettingsHandler) GetOIDC(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureOIDC) {
		return
	}

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

	oidc, err := h.store.GetOIDCSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get OIDC settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get OIDC settings"})
		return
	}

	// Mask client secret
	oidc.ClientSecret = ""

	c.JSON(http.StatusOK, oidc)
}

// UpdateOIDC updates OIDC settings for the organization.
// PUT /api/v1/system-settings/oidc
func (h *SystemSettingsHandler) UpdateOIDC(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureOIDC) {
		return
	}

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

	var req settings.UpdateOIDCSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current settings
	current, err := h.store.GetOIDCSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get current OIDC settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get current settings"})
		return
	}

	// Store old value for audit (mask secret)
	oldCopy := *current
	oldCopy.ClientSecret = "[MASKED]"
	oldValue, _ := json.Marshal(oldCopy)

	// Apply updates
	if req.Enabled != nil {
		current.Enabled = *req.Enabled
	}
	if req.Issuer != nil {
		current.Issuer = *req.Issuer
	}
	if req.ClientID != nil {
		current.ClientID = *req.ClientID
	}
	if req.ClientSecret != nil && *req.ClientSecret != "" {
		current.ClientSecret = *req.ClientSecret
	}
	if req.RedirectURL != nil {
		current.RedirectURL = *req.RedirectURL
	}
	if req.Scopes != nil {
		current.Scopes = req.Scopes
	}
	if req.AutoCreateUsers != nil {
		current.AutoCreateUsers = *req.AutoCreateUsers
	}
	if req.DefaultRole != nil {
		current.DefaultRole = *req.DefaultRole
	}
	if req.AllowedDomains != nil {
		current.AllowedDomains = req.AllowedDomains
	}
	if req.RequireEmailVerification != nil {
		current.RequireEmailVerification = *req.RequireEmailVerification
	}

	// Validate
	if err := current.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save
	if err := h.store.UpdateOIDCSettings(c.Request.Context(), user.CurrentOrgID, current); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update OIDC settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	// Create audit log
	newCopy := *current
	newCopy.ClientSecret = "[MASKED]"
	newValue, _ := json.Marshal(newCopy)

	auditLog := settings.NewSettingsAuditLog(
		user.CurrentOrgID,
		settings.SettingKeyOIDC,
		oldValue,
		newValue,
		user.ID,
		getClientIP(c),
	)
	if err := h.store.CreateSettingsAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create settings audit log")
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Msg("OIDC settings updated")

	// Return updated settings (masked)
	current.ClientSecret = ""
	c.JSON(http.StatusOK, current)
}

// TestOIDC tests the OIDC provider connection.
// POST /api/v1/system-settings/oidc/test
func (h *SystemSettingsHandler) TestOIDC(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureOIDC) {
		return
	}

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

	oidc, err := h.store.GetOIDCSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get OIDC settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get OIDC settings"})
		return
	}

	if !oidc.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "OIDC is not enabled"})
		return
	}

	// Validate configuration
	if err := oidc.Validate(); err != nil {
		c.JSON(http.StatusOK, settings.TestOIDCResponse{
			Success: false,
			Message: "Configuration validation failed: " + err.Error(),
		})
		return
	}

	// TODO: Implement actual OIDC provider discovery check
	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Str("issuer", oidc.Issuer).
		Msg("OIDC test requested")

	c.JSON(http.StatusOK, settings.TestOIDCResponse{
		Success:       true,
		Message:       "OIDC configuration is valid",
		ProviderName:  oidc.Issuer,
		SupportedFlow: "authorization_code",
	})
}

// GetStorageDefaults returns storage default settings for the organization.
// GET /api/v1/system-settings/storage
func (h *SystemSettingsHandler) GetStorageDefaults(c *gin.Context) {
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

	storage, err := h.store.GetStorageDefaultSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get storage settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get storage settings"})
		return
	}

	c.JSON(http.StatusOK, storage)
}

// UpdateStorageDefaults updates storage default settings for the organization.
// PUT /api/v1/system-settings/storage
func (h *SystemSettingsHandler) UpdateStorageDefaults(c *gin.Context) {
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

	var req settings.UpdateStorageDefaultsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current settings
	current, err := h.store.GetStorageDefaultSettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get current storage settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get current settings"})
		return
	}

	// Store old value for audit
	oldValue, _ := json.Marshal(current)

	// Apply updates
	if req.DefaultRetentionDays != nil {
		current.DefaultRetentionDays = *req.DefaultRetentionDays
	}
	if req.MaxRetentionDays != nil {
		current.MaxRetentionDays = *req.MaxRetentionDays
	}
	if req.DefaultStorageBackend != nil {
		current.DefaultStorageBackend = *req.DefaultStorageBackend
	}
	if req.MaxBackupSizeGB != nil {
		current.MaxBackupSizeGB = *req.MaxBackupSizeGB
	}
	if req.EnableCompression != nil {
		current.EnableCompression = *req.EnableCompression
	}
	if req.CompressionLevel != nil {
		current.CompressionLevel = *req.CompressionLevel
	}
	if req.DefaultEncryptionMethod != nil {
		current.DefaultEncryptionMethod = *req.DefaultEncryptionMethod
	}
	if req.PruneSchedule != nil {
		current.PruneSchedule = *req.PruneSchedule
	}
	if req.AutoPruneEnabled != nil {
		current.AutoPruneEnabled = *req.AutoPruneEnabled
	}

	// Validate
	if err := current.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save
	if err := h.store.UpdateStorageDefaultSettings(c.Request.Context(), user.CurrentOrgID, current); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update storage settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	// Create audit log
	newValue, _ := json.Marshal(current)

	auditLog := settings.NewSettingsAuditLog(
		user.CurrentOrgID,
		settings.SettingKeyStorageDefaults,
		oldValue,
		newValue,
		user.ID,
		getClientIP(c),
	)
	if err := h.store.CreateSettingsAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create settings audit log")
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Msg("storage default settings updated")

	c.JSON(http.StatusOK, current)
}

// GetSecurity returns security settings for the organization.
// GET /api/v1/system-settings/security
func (h *SystemSettingsHandler) GetSecurity(c *gin.Context) {
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

	security, err := h.store.GetSecuritySettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get security settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get security settings"})
		return
	}

	c.JSON(http.StatusOK, security)
}

// UpdateSecurity updates security settings for the organization.
// PUT /api/v1/system-settings/security
func (h *SystemSettingsHandler) UpdateSecurity(c *gin.Context) {
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

	var req settings.UpdateSecuritySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current settings
	current, err := h.store.GetSecuritySettings(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get current security settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get current settings"})
		return
	}

	// Store old value for audit
	oldValue, _ := json.Marshal(current)

	// Apply updates
	if req.SessionTimeoutMinutes != nil {
		current.SessionTimeoutMinutes = *req.SessionTimeoutMinutes
	}
	if req.MaxConcurrentSessions != nil {
		current.MaxConcurrentSessions = *req.MaxConcurrentSessions
	}
	if req.RequireMFA != nil {
		current.RequireMFA = *req.RequireMFA
	}
	if req.MFAGracePeriodDays != nil {
		current.MFAGracePeriodDays = *req.MFAGracePeriodDays
	}
	if req.AllowedIPRanges != nil {
		current.AllowedIPRanges = req.AllowedIPRanges
	}
	if req.BlockedIPRanges != nil {
		current.BlockedIPRanges = req.BlockedIPRanges
	}
	if req.FailedLoginLockoutAttempts != nil {
		current.FailedLoginLockoutAttempts = *req.FailedLoginLockoutAttempts
	}
	if req.FailedLoginLockoutMinutes != nil {
		current.FailedLoginLockoutMinutes = *req.FailedLoginLockoutMinutes
	}
	if req.APIKeyExpirationDays != nil {
		current.APIKeyExpirationDays = *req.APIKeyExpirationDays
	}
	if req.EnableAuditLogging != nil {
		current.EnableAuditLogging = *req.EnableAuditLogging
	}
	if req.AuditLogRetentionDays != nil {
		current.AuditLogRetentionDays = *req.AuditLogRetentionDays
	}
	if req.ForceHTTPS != nil {
		current.ForceHTTPS = *req.ForceHTTPS
	}
	if req.AllowPasswordLogin != nil {
		current.AllowPasswordLogin = *req.AllowPasswordLogin
	}

	// Validate
	if err := current.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save
	if err := h.store.UpdateSecuritySettings(c.Request.Context(), user.CurrentOrgID, current); err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to update security settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	// Create audit log
	newValue, _ := json.Marshal(current)

	auditLog := settings.NewSettingsAuditLog(
		user.CurrentOrgID,
		settings.SettingKeySecurity,
		oldValue,
		newValue,
		user.ID,
		getClientIP(c),
	)
	if err := h.store.CreateSettingsAuditLog(c.Request.Context(), auditLog); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create settings audit log")
	}

	h.logger.Info().
		Str("org_id", user.CurrentOrgID.String()).
		Str("user_id", user.ID.String()).
		Msg("security settings updated")

	c.JSON(http.StatusOK, current)
}

// GetAuditLog returns the settings audit log for the organization.
// GET /api/v1/system-settings/audit-log
func (h *SystemSettingsHandler) GetAuditLog(c *gin.Context) {
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

	limit := 50
	offset := 0
	if l := c.Query("limit"); l != "" {
		if _, err := fmt.Sscanf(l, "%d", &limit); err != nil || limit < 1 || limit > 100 {
			limit = 50
		}
	}
	if o := c.Query("offset"); o != "" {
		if _, err := fmt.Sscanf(o, "%d", &offset); err != nil || offset < 0 {
			offset = 0
		}
	}

	logs, err := h.store.GetSettingsAuditLogs(c.Request.Context(), user.CurrentOrgID, limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to get settings audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get audit logs"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":   logs,
		"limit":  limit,
		"offset": offset,
	})
}
