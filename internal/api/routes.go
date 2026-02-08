// Package api provides the HTTP API for the Keldris server.
package api

import (
	"io/fs"

	"github.com/MacJediWizard/keldris/internal/activity"
	"github.com/MacJediWizard/keldris/internal/api/handlers"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/backup/docker"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/logs"
	"github.com/MacJediWizard/keldris/internal/maintenance"
	"github.com/MacJediWizard/keldris/internal/metering"
	"github.com/MacJediWizard/keldris/internal/monitoring"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/MacJediWizard/keldris/internal/reports"
	"github.com/MacJediWizard/keldris/internal/telemetry"
	"github.com/MacJediWizard/keldris/internal/updates"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/MacJediWizard/keldris/docs/api"
)

// Config holds configuration for the API router.
type Config struct {
	// AllowedOrigins for CORS. Empty means all origins allowed in dev mode.
	AllowedOrigins []string
	// RateLimitRequests is the number of requests allowed per period.
	RateLimitRequests int64
	// RateLimitPeriod is the duration string for rate limiting (e.g. "1m", "1h").
	RateLimitPeriod string
	// Version information for the version endpoint.
	Version   string
	Commit    string
	BuildDate string
	// ServerURL is the base URL of the server for generating registration links.
	ServerURL string
	// VerificationTrigger for manually triggering verifications (optional).
	VerificationTrigger handlers.VerificationTrigger
	// ReportScheduler for report generation and sending (optional).
	ReportScheduler *reports.Scheduler
	// LogBuffer for server log capture and viewing (optional).
	LogBuffer *logs.LogBuffer
	// ActivityFeed for real-time activity events (optional).
	ActivityFeed *activity.Feed
	// TelemetryService for anonymous usage telemetry (optional).
	TelemetryService *telemetry.Service
	// DatabaseBackupService for PostgreSQL backup management (optional).
	DatabaseBackupService *maintenance.DatabaseBackupService
	// SecurityHeaders configures security headers for hardening.
	// If nil, default production settings are used.
	SecurityHeaders *middleware.SecurityHeadersConfig
	// DocsFS is the filesystem containing documentation markdown files (optional).
	DocsFS fs.FS
	// AirGapManager for air-gapped license management (optional).
	AirGapManager *license.AirGapManager
	// MeteringService for usage tracking and billing (optional).
	MeteringService *metering.Service
	// EmailService for sending emails (optional).
	EmailService *notifications.EmailService
	// UpdateChecker for checking for new Keldris versions (optional).
	UpdateChecker *updates.Checker
}

// DefaultConfig returns a Config with sensible defaults for development.
func DefaultConfig() Config {
	devSecurityHeaders := middleware.DevelopmentSecurityHeadersConfig()
	return Config{
		AllowedOrigins:    []string{},
		RateLimitRequests: 100,
		RateLimitPeriod:   "1m",
		Version:           "dev",
		Commit:            "unknown",
		BuildDate:         "unknown",
		SecurityHeaders:   &devSecurityHeaders,
	}
}

// ProductionConfig returns a Config with secure defaults for production.
func ProductionConfig() Config {
	prodSecurityHeaders := middleware.DefaultSecurityHeadersConfig()
	return Config{
		AllowedOrigins:    []string{},
		RateLimitRequests: 100,
		RateLimitPeriod:   "1m",
		Version:           "unknown",
		Commit:            "unknown",
		BuildDate:         "unknown",
		SecurityHeaders:   &prodSecurityHeaders,
	}
}

// Router wraps a Gin engine with configured middleware and routes.
type Router struct {
	Engine     *gin.Engine
	logger     zerolog.Logger
	sessions   *auth.SessionStore
	db         *db.DB
	keyManager *crypto.KeyManager
}

// NewRouter creates a new Router with the given dependencies.
func NewRouter(
	cfg Config,
	database *db.DB,
	oidc *auth.OIDC,
	sessions *auth.SessionStore,
	keyManager *crypto.KeyManager,
	logger zerolog.Logger,
) (*Router, error) {
	r := &Router{
		Engine:     gin.New(),
		logger:     logger.With().Str("component", "router").Logger(),
		sessions:   sessions,
		db:         database,
		keyManager: keyManager,
	}

	// Global middleware
	r.Engine.Use(gin.Recovery())
	r.Engine.Use(middleware.RequestLogger(logger))
	r.Engine.Use(middleware.CORS(cfg.AllowedOrigins))

	// Security headers middleware
	securityHeadersConfig := middleware.DefaultSecurityHeadersConfig()
	if cfg.SecurityHeaders != nil {
		securityHeadersConfig = *cfg.SecurityHeaders
	}
	r.Engine.Use(middleware.SecurityHeaders(securityHeadersConfig))

	// Rate limiting
	rateLimiter, err := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitPeriod)
	if err != nil {
		return nil, err
	}
	r.Engine.Use(rateLimiter)

	// Health check endpoints (no auth required for basic checks)
	healthHandler := handlers.NewHealthHandler(database, oidc, logger)
	healthHandler.SetSessionStore(sessions)
	if cfg.DatabaseBackupService != nil {
		healthHandler.SetDatabaseBackupService(cfg.DatabaseBackupService)
	}
	healthHandler.RegisterPublicRoutes(r.Engine)

	// Prometheus metrics endpoint (no auth required)
	metricsHandler := handlers.NewMetricsHandler(database, logger)
	metricsHandler.RegisterPublicRoutes(r.Engine)

	// Swagger API documentation (no auth required)
	r.Engine.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/api/docs/doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))

	// Version endpoint (no auth required)
	versionHandler := handlers.NewVersionHandler(cfg.Version, cfg.Commit, cfg.BuildDate, logger)
	versionHandler.RegisterPublicRoutes(r.Engine)

	// Security headers test endpoint (no auth required for verification)
	securityHandler := handlers.NewSecurityHandler(logger)
	securityHandler.RegisterPublicRoutes(r.Engine)

	// Changelog endpoint (no auth required for public access)
	changelogHandler := handlers.NewChangelogHandler("CHANGELOG.md", cfg.Version, logger)
	changelogHandler.RegisterPublicRoutes(r.Engine)

	// Documentation endpoint (no auth required for public access)
	if cfg.DocsFS != nil {
		docsHandler := handlers.NewDocsHandler(cfg.DocsFS, logger)
		docsHandler.RegisterPublicRoutes(r.Engine)
	}

	// Air-gap/license management (public status endpoint, protected management endpoints)
	var airGapHandler *handlers.AirGapHandler
	if cfg.AirGapManager != nil {
		airGapHandler = handlers.NewAirGapHandler(cfg.AirGapManager, logger)
		publicAPI := r.Engine.Group("/api/v1/public")
		airGapHandler.RegisterRoutes(nil, publicAPI) // Will register to apiV1 later
	}

	// Update checker endpoint (no auth required for banner display)
	updatesHandler := handlers.NewUpdatesHandler(cfg.UpdateChecker, logger)
	updatesHandler.RegisterPublicRoutes(r.Engine)

	// Auth routes (no auth required)
	authGroup := r.Engine.Group("/auth")
	authHandler := handlers.NewAuthHandler(oidc, sessions, database, logger)
	authHandler.RegisterRoutes(authGroup)

	// Password reset routes (no auth required)
	passwordResetHandler := handlers.NewPasswordResetHandler(database, cfg.EmailService, cfg.ServerURL, logger)
	passwordResetHandler.RegisterPublicRoutes(r.Engine)

	// API v1 routes (auth required)
	apiV1 := r.Engine.Group("/api/v1")
	apiV1.Use(middleware.AuthMiddleware(sessions, logger))
	apiV1.Use(middleware.AuditMiddleware(database, logger))

	// Create IP filter for IP-based access control
	ipFilter := middleware.NewIPFilter(database, logger)
	apiV1.Use(middleware.IPFilterMiddleware(ipFilter, logger))

	// Create RBAC for permission checks
	rbac := auth.NewRBAC(database)

	// Register API handlers
	versionHandler.RegisterRoutes(apiV1)
	changelogHandler.RegisterRoutes(apiV1)
	healthHandler.RegisterRoutes(apiV1)
	securityHandler.RegisterRoutes(apiV1)
	updatesHandler.RegisterRoutes(apiV1)

	// Documentation routes (authenticated)
	if cfg.DocsFS != nil {
		docsHandler := handlers.NewDocsHandler(cfg.DocsFS, logger)
		docsHandler.RegisterRoutes(apiV1)
	}

	// Register air-gap protected routes (after auth middleware is applied)
	if airGapHandler != nil {
		airGapHandler.RegisterRoutes(apiV1, nil)
	}

	orgsHandler := handlers.NewOrganizationsHandler(database, sessions, rbac, logger)
	orgsHandler.RegisterRoutes(apiV1)

	agentsHandler := handlers.NewAgentsHandler(database, logger)
	agentsHandler.RegisterRoutes(apiV1)

	agentCommandsHandler := handlers.NewAgentCommandsHandler(database, logger)
	agentCommandsHandler.RegisterRoutes(apiV1)

	// Agent registration with 2FA codes
	agentRegistrationHandler := handlers.NewAgentRegistrationHandler(database, logger)
	agentRegistrationHandler.RegisterRoutes(apiV1)
	agentRegistrationHandler.RegisterPublicRoutes(r.Engine)

	agentGroupsHandler := handlers.NewAgentGroupsHandler(database, logger)
	agentGroupsHandler.RegisterRoutes(apiV1)

	// Agent CSV import for fleet deployment
	agentImportHandler := handlers.NewAgentImportHandler(database, cfg.ServerURL, logger)
	agentImportHandler.RegisterRoutes(apiV1)

	reposHandler := handlers.NewRepositoriesHandler(database, keyManager, logger)
	reposHandler.RegisterRoutes(apiV1)

	repoImportHandler := handlers.NewRepositoryImportHandler(database, keyManager, logger)
	repoImportHandler.RegisterRoutes(apiV1)

	schedulesHandler := handlers.NewSchedulesHandler(database, logger)
	schedulesHandler.RegisterRoutes(apiV1)

	backupScriptsHandler := handlers.NewBackupScriptsHandler(database, logger)
	backupScriptsHandler.RegisterRoutes(apiV1)

	containerHooksHandler := handlers.NewContainerHooksHandler(database, logger)
	containerHooksHandler.RegisterRoutes(apiV1)

	backupHookTemplatesHandler := handlers.NewBackupHookTemplatesHandler(database, logger)
	backupHookTemplatesHandler.RegisterRoutes(apiV1)

	policiesHandler := handlers.NewPoliciesHandler(database, logger)
	policiesHandler.RegisterRoutes(apiV1)

	backupsHandler := handlers.NewBackupsHandler(database, logger)
	backupsHandler.RegisterRoutes(apiV1)

	snapshotsHandler := handlers.NewSnapshotsHandler(database, logger)
	snapshotsHandler.RegisterRoutes(apiV1)

	legalHoldsHandler := handlers.NewLegalHoldsHandler(database, logger)
	legalHoldsHandler.RegisterRoutes(apiV1)

	fileHistoryHandler := handlers.NewFileHistoryHandler(database, logger)
	fileHistoryHandler.RegisterRoutes(apiV1)

	fileSearchHandler := handlers.NewFileSearchHandler(database, keyManager, logger)
	fileSearchHandler.RegisterRoutes(apiV1)

	auditLogsHandler := handlers.NewAuditLogsHandler(database, logger)
	auditLogsHandler.RegisterRoutes(apiV1)

	alertsHandler := handlers.NewAlertsHandler(database, logger)
	alertsHandler.RegisterRoutes(apiV1)

	notificationsHandler := handlers.NewNotificationsHandler(database, logger)
	notificationsHandler.RegisterRoutes(apiV1)

	notificationRulesHandler := handlers.NewNotificationRulesHandler(database, logger)
	notificationRulesHandler.RegisterRoutes(apiV1)

	// Register reports handler if scheduler is available
	if cfg.ReportScheduler != nil {
		reportsHandler := handlers.NewReportsHandler(database, cfg.ReportScheduler, logger)
		reportsHandler.RegisterRoutes(apiV1)
	}

	statsHandler := handlers.NewStatsHandler(database, logger)
	statsHandler.RegisterRoutes(apiV1)

	excludePatternsHandler := handlers.NewExcludePatternsHandler(database, logger)
	excludePatternsHandler.RegisterRoutes(apiV1)

	tagsHandler := handlers.NewTagsHandler(database, logger)
	tagsHandler.RegisterRoutes(apiV1)

	filtersHandler := handlers.NewFiltersHandler(database, logger)
	filtersHandler.RegisterRoutes(apiV1)

	favoritesHandler := handlers.NewFavoritesHandler(database, logger)
	favoritesHandler.RegisterRoutes(apiV1)

	searchHandler := handlers.NewSearchHandler(database, logger)
	searchHandler.RegisterRoutes(apiV1)

	dashboardMetricsHandler := handlers.NewDashboardMetricsHandler(database, logger)
	dashboardMetricsHandler.RegisterRoutes(apiV1)

	onboardingHandler := handlers.NewOnboardingHandler(database, logger)
	onboardingHandler.RegisterRoutes(apiV1)

	costHandler := handlers.NewCostEstimationHandler(database, logger)
	costHandler.RegisterRoutes(apiV1)

	// Register verification handler if trigger is available
	if cfg.VerificationTrigger != nil {
		verificationsHandler := handlers.NewVerificationsHandler(database, cfg.VerificationTrigger, logger)
		verificationsHandler.RegisterRoutes(apiV1)
	}

	ssoGroupMappingsHandler := handlers.NewSSOGroupMappingsHandler(database, rbac, logger)
	ssoGroupMappingsHandler.RegisterRoutes(apiV1)

	maintenanceHandler := handlers.NewMaintenanceHandler(database, logger)
	maintenanceHandler.RegisterRoutes(apiV1)

	announcementsHandler := handlers.NewAnnouncementsHandler(database, logger)
	announcementsHandler.RegisterRoutes(apiV1)

	// Password policies handler for non-OIDC authentication
	passwordPoliciesHandler := handlers.NewPasswordPoliciesHandler(database, logger)
	passwordPoliciesHandler.RegisterRoutes(apiV1)

	// Server logs handler for admin (requires LogBuffer)
	if cfg.LogBuffer != nil {
		serverLogsHandler := handlers.NewServerLogsHandler(database, cfg.LogBuffer, logger)
		serverLogsHandler.RegisterRoutes(apiV1)
	}

	ransomwareHandler := handlers.NewRansomwareHandler(database, logger)
	ransomwareHandler.RegisterRoutes(apiV1)

	configExportHandler := handlers.NewConfigExportHandler(database, logger)
	configExportHandler.RegisterRoutes(apiV1)

	// Docker restore routes
	dockerRestoreHandler := handlers.NewDockerRestoreHandler(database, logger)
	dockerRestoreHandler.RegisterRoutes(apiV1)

	// DR Runbook routes
	drRunbooksHandler := handlers.NewDRRunbooksHandler(database, logger)
	drRunbooksHandler.RegisterRoutes(apiV1)

	// DR Test routes (runner is nil for now, will be set up when scheduler is integrated)
	drTestsHandler := handlers.NewDRTestsHandler(database, nil, logger)
	drTestsHandler.RegisterRoutes(apiV1)

	// Geo-Replication routes
	geoReplicationHandler := handlers.NewGeoReplicationHandler(database, logger)
	geoReplicationHandler.RegisterRoutes(apiV1)

	// Classification routes
	classificationsHandler := handlers.NewClassificationsHandler(database, logger)
	classificationsHandler.RegisterRoutes(apiV1)

	// Storage Tiering routes (scheduler is nil for now, will be set up when tiering scheduler is integrated)
	storageTiersHandler := handlers.NewStorageTiersHandler(database, nil, logger)
	storageTiersHandler.RegisterRoutes(apiV1)

	// Support bundle routes
	supportHandler := handlers.NewSupportHandler(cfg.Version, cfg.Commit, cfg.BuildDate, "", logger)
	supportHandler.RegisterRoutes(apiV1)

	// SLA routes
	slaHandler := handlers.NewSLAHandler(database, logger)
	slaHandler.RegisterRoutes(apiV1)

	// Downtime tracking routes
	downtimeService := monitoring.NewDowntimeServiceWithDB(database, monitoring.DefaultDowntimeServiceConfig(), logger)
	downtimeHandler := handlers.NewDowntimeHandler(downtimeService, database, logger)
	downtimeHandler.RegisterRoutes(apiV1)

	// Docker health monitoring routes
	dockerHandler := handlers.NewDockerHandler(database, logger)
	dockerHandler.RegisterRoutes(apiV1)

	// IP allowlists routes
	ipAllowlistsHandler := handlers.NewIPAllowlistsHandler(database, ipFilter, logger)
	ipAllowlistsHandler.RegisterRoutes(apiV1)

	// Rate limit dashboard routes (admin only)
	rateLimitHandler := handlers.NewRateLimitHandler(database, logger)
	rateLimitHandler.RegisterRoutes(apiV1)

	// Rate limit config management routes
	rateLimitsHandler := handlers.NewRateLimitsHandler(database, logger)
	rateLimitsHandler.RegisterRoutes(apiV1)

	// User sessions management routes
	userSessionsHandler := handlers.NewUserSessionsHandler(database, logger)
	userSessionsHandler.RegisterRoutes(apiV1)

	// User management routes (admin)
	usersHandler := handlers.NewUsersHandler(database, sessions, rbac, logger)
	usersHandler.RegisterRoutes(apiV1)

	// Recent items tracking routes
	recentItemsHandler := handlers.NewRecentItemsHandler(database, logger)
	recentItemsHandler.RegisterRoutes(apiV1)

	// Lifecycle policy routes
	lifecyclePoliciesHandler := handlers.NewLifecyclePoliciesHandler(database, logger)
	lifecyclePoliciesHandler.RegisterRoutes(apiV1)

	// Job queue routes
	jobQueueHandler := handlers.NewJobQueueHandler(database, rbac, logger)
	jobQueueHandler.RegisterRoutes(apiV1)

	// System settings routes (admin only)
	systemSettingsHandler := handlers.NewSystemSettingsHandler(database, logger)
	systemSettingsHandler.RegisterRoutes(apiV1)

	// License and feature flags routes
	featureChecker := license.NewFeatureChecker(database)
	licenseHandler := handlers.NewLicenseHandler(database, featureChecker, logger)
	licenseHandler.RegisterRoutes(apiV1)

	// Trial management routes
	trialHandler := handlers.NewTrialHandler(database, logger)
	trialHandler.RegisterRoutes(apiV1)

	// Branding settings routes (Enterprise only)
	brandingHandler := handlers.NewBrandingHandler(database, logger)
	brandingHandler.RegisterRoutes(apiV1)
	brandingHandler.RegisterPublicRoutes(r.Engine.Group("/api/public"))

	// Superuser routes (requires superuser privileges)
	superuserHandler := handlers.NewSuperuserHandler(database, sessions, logger)
	superuserHandler.RegisterRoutes(apiV1)

	// Telemetry routes (opt-in anonymous usage telemetry)
	telemetryHandler := handlers.NewTelemetryHandler(database, cfg.TelemetryService, logger)
	telemetryHandler.RegisterRoutes(apiV1)

	// Database backup routes (requires superuser privileges)
	if cfg.DatabaseBackupService != nil {
		databaseBackupHandler := handlers.NewDatabaseBackupHandler(database, sessions, cfg.DatabaseBackupService, logger)
		databaseBackupHandler.RegisterRoutes(apiV1)
	}

	// System health routes (requires superuser privileges)
	systemHealthHandler := handlers.NewSystemHealthHandler(database, sessions, logger)
	systemHealthHandler.RegisterRoutes(apiV1)

	// Docker backup routes
	dockerDiscoveryConfig := docker.DefaultDiscoveryConfig()
	dockerDiscoveryService := docker.NewDiscoveryService(database, dockerDiscoveryConfig, logger)
	dockerBackupHandler := handlers.NewDockerBackupHandler(database, dockerDiscoveryService, logger)
	dockerBackupHandler.RegisterRoutes(apiV1)

	// Docker container logs backup routes
	dockerLogBackupService := docker.NewLogBackupService(docker.DefaultLogBackupConfig(), logger)
	dockerLogsHandler := handlers.NewDockerLogsHandler(database, dockerLogBackupService, logger)
	dockerLogsHandler.RegisterRoutes(apiV1)

	// Docker registry routes
	dockerRegistriesHandler := handlers.NewDockerRegistriesHandler(database, keyManager, logger)
	dockerRegistriesHandler.RegisterRoutes(apiV1)

	// Database connections routes (MySQL/MariaDB backup)
	databaseConnectionsHandler := handlers.NewDatabaseConnectionsHandler(database, keyManager, logger)
	databaseConnectionsHandler.RegisterRoutes(apiV1)

	// Proxmox VM backup routes
	proxmoxHandler := handlers.NewProxmoxHandler(database, keyManager, logger)
	proxmoxHandler.RegisterRoutes(apiV1)

	// Activity feed routes
	if cfg.ActivityFeed != nil {
		activityHandler := handlers.NewActivityHandler(database, cfg.ActivityFeed, logger)
		activityHandler.RegisterRoutes(apiV1)
		activityHandler.RegisterWebSocketRoute(r.Engine, middleware.AuthMiddleware(sessions, logger))
	}

	// Komodo integration routes
	komodoHandler := handlers.NewKomodoHandler(database, logger)
	komodoHandler.RegisterRoutes(apiV1)
	komodoHandler.RegisterWebhookRoutes(r.Engine)

	// Pi-hole backup routes
	piholeHandler := handlers.NewPiholeHandler(database, logger)
	piholeHandler.RegisterRoutes(apiV1)

	// Usage metering routes (requires MeteringService)
	if cfg.MeteringService != nil {
		usageHandler := handlers.NewUsageHandler(database, cfg.MeteringService, logger)
		usageHandler.RegisterRoutes(apiV1)
	}

	// Agent API routes (API key auth required)
	// These endpoints are for agents to communicate with the server
	apiKeyValidator := auth.NewAPIKeyValidator(database, logger)
	agentAPI := r.Engine.Group("/api/v1/agent")
	agentAPI.Use(middleware.APIKeyMiddleware(apiKeyValidator, logger))
	agentAPI.Use(middleware.IPFilterAgentMiddleware(ipFilter, logger))

	agentAPIHandler := handlers.NewAgentAPIHandler(database, logger)
	agentAPIHandler.RegisterRoutes(agentAPI)

	r.logger.Info().Msg("API router initialized")
	return r, nil
}
