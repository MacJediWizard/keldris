package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ServerSetupStore defines the interface for server setup persistence operations.
type ServerSetupStore interface {
	// Setup state
	GetServerSetup(ctx context.Context) (*models.ServerSetup, error)
	IsSetupComplete(ctx context.Context) (bool, error)
	CompleteSetupStep(ctx context.Context, step models.ServerSetupStep) error
	FinalizeSetup(ctx context.Context, userID uuid.UUID) error

	// User/Superuser
	HasAnySuperuser(ctx context.Context) (bool, error)
	CreateSuperuserWithPassword(ctx context.Context, email, password, name string) (*models.User, *models.Organization, error)

	// License
	GetActiveLicense(ctx context.Context) (*models.LicenseKey, error)
	ActivateLicense(ctx context.Context, licenseKey string, activatedBy *uuid.UUID) (*models.LicenseKey, error)
	CreateTrialLicense(ctx context.Context, companyName, contactEmail string, activatedBy *uuid.UUID) (*models.LicenseKey, error)

	// Organization
	HasAnyOrganization(ctx context.Context) (bool, error)
	CreateFirstOrganization(ctx context.Context, name string, createdBy uuid.UUID) (*models.Organization, error)

	// Settings (for SMTP/OIDC)
	GetSMTPSettings(ctx context.Context, orgID uuid.UUID) (*settings.SMTPSettings, error)
	UpdateSMTPSettings(ctx context.Context, orgID uuid.UUID, smtp *settings.SMTPSettings) error
	GetOIDCSettings(ctx context.Context, orgID uuid.UUID) (*settings.OIDCSettings, error)
	UpdateOIDCSettings(ctx context.Context, orgID uuid.UUID, oidc *settings.OIDCSettings) error
	EnsureSystemSettingsExist(ctx context.Context, orgID uuid.UUID) error

	// Audit
	CreateServerSetupAuditLog(ctx context.Context, log *models.ServerSetupAuditLog) error
}

// DBPinger defines the interface for testing database connectivity.
type DBPinger interface {
	Ping(ctx context.Context) error
}

// ServerSetupHandler handles first-time server setup HTTP endpoints.
type ServerSetupHandler struct {
	store    ServerSetupStore
	db       DBPinger
	sessions *auth.SessionStore
	logger   zerolog.Logger
}

// NewServerSetupHandler creates a new ServerSetupHandler.
func NewServerSetupHandler(store ServerSetupStore, db DBPinger, sessions *auth.SessionStore, logger zerolog.Logger) *ServerSetupHandler {
	return &ServerSetupHandler{
		store:    store,
		db:       db,
		sessions: sessions,
		logger:   logger.With().Str("component", "server_setup_handler").Logger(),
	}
}

// RegisterRoutes registers setup routes on the given router group.
// These routes are available during initial setup (no auth required).
// Setup is simplified to database verification + superuser creation only.
func (h *ServerSetupHandler) RegisterRoutes(r *gin.RouterGroup) {
	setup := r.Group("/setup")
	setup.Use(middleware.SetupLockMiddleware(h.store, h.logger))
	{
		// Status endpoint (always available)
		setup.GET("/status", h.GetStatus)

		// Setup steps (DB + superuser only)
		setup.POST("/database/test", h.TestDatabaseConnection)
		setup.POST("/superuser", h.CreateSuperuser)
		setup.POST("/complete", h.CompleteSetup)
	}
}

// RegisterSuperuserRoutes registers superuser-only re-run endpoints.
func (h *ServerSetupHandler) RegisterSuperuserRoutes(r *gin.RouterGroup) {
	setup := r.Group("/setup/rerun")
	setup.Use(middleware.SuperuserMiddleware(h.sessions, h.logger))
	{
		setup.GET("", h.GetRerunStatus)
		setup.POST("/smtp", h.RerunConfigureSMTP)
		setup.POST("/oidc", h.RerunConfigureOIDC)
		setup.POST("/license", h.RerunUpdateLicense)
	}
}

// getClientIP extracts the client IP address from the request.
func (h *ServerSetupHandler) getClientIP(c *gin.Context) string {
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}
	return c.ClientIP()
}

// logAction creates a setup audit log entry.
func (h *ServerSetupHandler) logAction(c *gin.Context, action, step string, userID *uuid.UUID, details interface{}) {
	log := models.NewServerSetupAuditLog(action, step).
		WithRequestInfo(h.getClientIP(c), c.GetHeader("User-Agent"))
	if userID != nil {
		log = log.WithPerformedBy(*userID)
	}
	if details != nil {
		log = log.WithDetails(details)
	}

	if err := h.store.CreateServerSetupAuditLog(c.Request.Context(), log); err != nil {
		h.logger.Warn().Err(err).Str("action", action).Msg("failed to create setup audit log")
	}
}

// GetStatus returns the current server setup status.
// GET /api/v1/setup/status
func (h *ServerSetupHandler) GetStatus(c *gin.Context) {
	ctx := c.Request.Context()

	setup, err := h.store.GetServerSetup(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get server setup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get setup status"})
		return
	}

	// Check database connectivity
	databaseOK := h.db.Ping(ctx) == nil

	// Check if superuser exists
	hasSuperuser, _ := h.store.HasAnySuperuser(ctx)

	response := &models.SetupStatusResponse{
		NeedsSetup:     !setup.SetupCompleted,
		SetupCompleted: setup.SetupCompleted,
		CurrentStep:    setup.CurrentStep,
		CompletedSteps: setup.CompletedSteps,
		DatabaseOK:     databaseOK,
		HasSuperuser:   hasSuperuser,
	}

	c.JSON(http.StatusOK, response)
}

// TestDatabaseConnection tests the database connection.
// POST /api/v1/setup/database/test
func (h *ServerSetupHandler) TestDatabaseConnection(c *gin.Context) {
	ctx := c.Request.Context()

	err := h.db.Ping(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("database connection test failed")
		h.logAction(c, "database_test_failed", "database", nil, map[string]string{"error": err.Error()})
		c.JSON(http.StatusOK, gin.H{
			"ok":      false,
			"message": "Database connection failed: " + err.Error(),
		})
		return
	}

	// Mark database step as complete
	if err := h.store.CompleteSetupStep(ctx, models.SetupStepDatabase); err != nil {
		h.logger.Error().Err(err).Msg("failed to complete database step")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update setup progress"})
		return
	}

	h.logAction(c, "database_test_success", "database", nil, nil)

	c.JSON(http.StatusOK, gin.H{
		"ok":      true,
		"message": "Database connection successful",
	})
}

// CreateSuperuser creates the first superuser account.
// POST /api/v1/setup/superuser
func (h *ServerSetupHandler) CreateSuperuser(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CreateSuperuserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password length
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	// Check if superuser already exists
	hasSuperuser, err := h.store.HasAnySuperuser(ctx)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to check for existing superuser")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check superuser status"})
		return
	}
	if hasSuperuser {
		c.JSON(http.StatusConflict, gin.H{"error": "superuser already exists"})
		return
	}

	// Create superuser
	user, org, err := h.store.CreateSuperuserWithPassword(ctx, req.Email, req.Password, req.Name)
	if err != nil {
		h.logger.Error().Err(err).Str("email", req.Email).Msg("failed to create superuser")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create superuser"})
		return
	}

	// Mark superuser step as complete
	if err := h.store.CompleteSetupStep(ctx, models.SetupStepSuperuser); err != nil {
		h.logger.Error().Err(err).Msg("failed to complete superuser step")
	}

	h.logAction(c, "superuser_created", "superuser", &user.ID, map[string]string{
		"email":  req.Email,
		"org_id": org.ID.String(),
	})

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
		"org_id":  org.ID,
		"message": "Superuser created successfully",
	})
}

// CompleteSetup finalizes the setup process.
// POST /api/v1/setup/complete
func (h *ServerSetupHandler) CompleteSetup(c *gin.Context) {
	ctx := c.Request.Context()

	setup, err := h.store.GetServerSetup(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get setup state"})
		return
	}

	// Verify all required steps are complete (DB + superuser only)
	requiredSteps := []models.ServerSetupStep{
		models.SetupStepDatabase,
		models.SetupStepSuperuser,
	}

	for _, step := range requiredSteps {
		if !setup.HasCompletedStep(step) {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"error":        "not all required steps completed",
				"missing_step": step,
			})
			return
		}
	}

	// Finalize setup (we don't have a user ID at this point)
	if err := h.store.FinalizeSetup(ctx, uuid.Nil); err != nil {
		h.logger.Error().Err(err).Msg("failed to finalize setup")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete setup"})
		return
	}

	h.logAction(c, "setup_completed", "complete", nil, nil)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Server setup completed successfully",
		"redirect": "/login",
	})
}

// Superuser re-run endpoints

// GetRerunStatus returns the current configuration status for re-run.
// GET /api/v1/setup/rerun
func (h *ServerSetupHandler) GetRerunStatus(c *gin.Context) {
	ctx := c.Request.Context()

	license, _ := h.store.GetActiveLicense(ctx)

	status := gin.H{
		"setup_completed": true,
		"can_configure":   []string{"smtp", "oidc", "license"},
	}

	if license != nil {
		status["license"] = &models.LicenseInfo{
			LicenseType: license.LicenseType,
			Status:      string(license.Status),
			ExpiresAt:   license.ExpiresAt,
			CompanyName: license.CompanyName,
		}
	}

	c.JSON(http.StatusOK, status)
}

// RerunConfigureSMTP allows superuser to reconfigure SMTP.
// POST /api/v1/setup/rerun/smtp
func (h *ServerSetupHandler) RerunConfigureSMTP(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	ctx := c.Request.Context()

	var req settings.SMTPSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateSMTPSettings(ctx, user.CurrentOrgID, &req); err != nil {
		h.logger.Error().Err(err).Msg("failed to update SMTP settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save SMTP settings"})
		return
	}

	h.logAction(c, "smtp_reconfigured", "smtp", &user.ID, map[string]string{"host": req.Host})

	c.JSON(http.StatusOK, gin.H{"message": "SMTP settings updated"})
}

// RerunConfigureOIDC allows superuser to reconfigure OIDC.
// POST /api/v1/setup/rerun/oidc
func (h *ServerSetupHandler) RerunConfigureOIDC(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	ctx := c.Request.Context()

	var req settings.OIDCSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := req.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateOIDCSettings(ctx, user.CurrentOrgID, &req); err != nil {
		h.logger.Error().Err(err).Msg("failed to update OIDC settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save OIDC settings"})
		return
	}

	h.logAction(c, "oidc_reconfigured", "oidc", &user.ID, map[string]string{"issuer": req.Issuer})

	c.JSON(http.StatusOK, gin.H{"message": "OIDC settings updated"})
}

// RerunUpdateLicense allows superuser to update the license.
// POST /api/v1/setup/rerun/license
func (h *ServerSetupHandler) RerunUpdateLicense(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	ctx := c.Request.Context()

	var req models.ActivateLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	license, err := h.store.ActivateLicense(ctx, req.LicenseKey, &user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to update license")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to update license: " + err.Error()})
		return
	}

	h.logAction(c, "license_updated", "license", &user.ID, map[string]string{
		"license_type": string(license.LicenseType),
	})

	c.JSON(http.StatusOK, gin.H{
		"license_type": license.LicenseType,
		"expires_at":   license.ExpiresAt,
		"message":      "License updated successfully",
	})
}
