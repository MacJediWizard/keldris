// Package api provides the HTTP API for the Keldris server.
package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/MacJediWizard/keldris/internal/api/handlers"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/logs"
	"github.com/MacJediWizard/keldris/internal/monitoring"
	"github.com/MacJediWizard/keldris/internal/reports"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/MacJediWizard/keldris/docs/api"
)

// Config holds configuration for the API router.
type Config struct {
	// Environment is the deployment environment (development, staging, production).
	Environment config.Environment
	// AllowedOrigins for CORS. Empty means all origins allowed in dev mode.
	AllowedOrigins []string
	// RateLimitRequests is the number of requests allowed per period.
	RateLimitRequests int64
	// RateLimitPeriod is the duration string for rate limiting (e.g. "1m", "1h").
	RateLimitPeriod string
	// RedisURL enables Redis-backed distributed rate limiting when set.
	RedisURL string
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
	// DRTestRunner for triggering DR test execution (optional).
	DRTestRunner handlers.DRTestRunner
	// License is the current server license for feature gating (optional).
	License *license.License
	// Validator is the license validator for dynamic license checks (optional).
	Validator *license.Validator
	// LicensePublicKey is the Ed25519 public key for validating license keys (optional).
	LicensePublicKey []byte
	// WebDir is the path to the built frontend files (e.g. "web/dist").
	WebDir string
	// LogBuffer for server log capture and viewing (optional).
	LogBuffer *logs.LogBuffer
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
	r.Engine.Use(middleware.BodyLimitMiddleware(10 << 20)) // 10 MB default body limit
	r.Engine.Use(middleware.RequestLogger(logger))
	r.Engine.Use(middleware.SecurityHeaders(cfg.Environment))
	r.Engine.Use(middleware.CORS(cfg.AllowedOrigins, cfg.Environment))

	// Rate limiting
	rateLimiter, err := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitPeriod, cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	r.Engine.Use(rateLimiter)

	// Health check endpoints (no auth required)
	healthHandler := handlers.NewHealthHandler(database, oidc, logger)
	healthHandler.RegisterPublicRoutes(r.Engine)

	// Prometheus metrics endpoint (no auth required)
	metricsHandler := handlers.NewMetricsHandler(database, logger)
	metricsHandler.RegisterPublicRoutes(r.Engine)

	// Swagger API documentation (no auth required, disabled in production for security)
	if cfg.Environment != config.EnvProduction {
		r.Engine.GET("/api/docs/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
			ginSwagger.URL("/api/docs/doc.json"),
			ginSwagger.DefaultModelsExpandDepth(-1),
		))
	} else {
		r.logger.Info().Msg("Swagger UI disabled in production for security")
	}

	// Version endpoint (no auth required)
	versionHandler := handlers.NewVersionHandler(cfg.Version, cfg.Commit, cfg.BuildDate, logger)
	versionHandler.RegisterPublicRoutes(r.Engine)

	// Changelog endpoint (no auth required for public access)
	changelogHandler := handlers.NewChangelogHandler("CHANGELOG.md", cfg.Version, logger)
	changelogHandler.RegisterPublicRoutes(r.Engine)

	// Auth routes (no auth required)
	authGroup := r.Engine.Group("/auth")
	authHandler := handlers.NewAuthHandler(oidc, sessions, database, logger)
	authHandler.RegisterRoutes(authGroup)

	// API v1 routes (auth required)
	apiV1 := r.Engine.Group("/api/v1")
	apiV1.Use(middleware.AuthMiddleware(sessions, logger))
	apiV1.Use(middleware.UserVerifyMiddleware(database, sessions, logger))
	apiV1.Use(middleware.AuditMiddleware(database, logger))

	// License middleware for feature gating
	if cfg.Validator != nil {
		// Dynamic: reads current license from validator (supports downgrades)
		apiV1.Use(middleware.DynamicLicenseMiddleware(cfg.Validator, logger))
	} else {
		// Static: use the license provided at startup
		lic := cfg.License
		if lic == nil {
			lic = license.FreeLicense()
		}
		apiV1.Use(middleware.LicenseMiddleware(lic, logger))
	}
	// Create IP filter for IP-based access control
	ipFilter := middleware.NewIPFilter(database, logger)
	apiV1.Use(middleware.IPFilterMiddleware(ipFilter, logger))

	// Create RBAC for permission checks
	rbac := auth.NewRBAC(database)

	// Register API handlers
	versionHandler.RegisterRoutes(apiV1)
	changelogHandler.RegisterRoutes(apiV1)

	// License info endpoint
	licenseInfoHandler := handlers.NewLicenseInfoHandler(cfg.Validator, logger)
	licenseInfoHandler.RegisterRoutes(apiV1)

	// License management endpoints (activate/deactivate from GUI)
	if cfg.Validator != nil {
		licenseManageHandler := handlers.NewLicenseManageHandler(cfg.Validator, cfg.LicensePublicKey, logger)
		licenseManageHandler.RegisterRoutes(apiV1)
	}

	orgsHandler := handlers.NewOrganizationsHandler(database, sessions, rbac, logger)
	orgsHandler.RegisterRoutes(apiV1)
	orgsGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureMultiOrg, logger))
	orgsHandler.RegisterMultiOrgRoutes(orgsGroup, middleware.LimitMiddleware(database, "organizations", logger))

	agentsHandler := handlers.NewAgentsHandler(database, logger)
	agentsHandler.RegisterRoutes(apiV1, middleware.LimitMiddleware(database, "agents", logger))

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

	policiesHandler := handlers.NewPoliciesHandler(database, logger)
	policiesHandler.RegisterRoutes(apiV1)

	backupsHandler := handlers.NewBackupsHandler(database, logger)
	backupsHandler.RegisterRoutes(apiV1)

	snapshotsHandler := handlers.NewSnapshotsHandler(database, keyManager, logger)
	snapshotsHandler.RegisterRoutes(apiV1)

	legalHoldsHandler := handlers.NewLegalHoldsHandler(database, logger)
	legalHoldsHandler.RegisterRoutes(apiV1)

	fileHistoryHandler := handlers.NewFileHistoryHandler(database, logger)
	fileHistoryHandler.RegisterRoutes(apiV1)

	auditLogsGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureAuditLogs, logger))
	fileSearchHandler := handlers.NewFileSearchHandler(database, keyManager, logger)
	fileSearchHandler.RegisterRoutes(apiV1)

	auditLogsHandler := handlers.NewAuditLogsHandler(database, logger)
	auditLogsHandler.RegisterRoutes(auditLogsGroup)

	alertsHandler := handlers.NewAlertsHandler(database, logger)
	alertsHandler.RegisterRoutes(apiV1)

	notificationsHandler := handlers.NewNotificationsHandlerWithEnv(database, keyManager, logger, cfg.Environment)
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

	usersHandler := handlers.NewUsersHandler(database, rbac, logger)
	usersHandler.RegisterRoutes(apiV1)

	ssoGroupMappingsHandler := handlers.NewSSOGroupMappingsHandler(database, rbac, logger)
	ssoGroupMappingsHandler.RegisterRoutes(apiV1)

	maintenanceHandler := handlers.NewMaintenanceHandler(database, logger)
	maintenanceHandler.RegisterRoutes(apiV1)

	// Docker backup routes (Pro+)
	dockerBackupGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureDockerBackup, logger))
	dockerBackupHandler := handlers.NewDockerBackupHandler(database, logger)
	dockerBackupHandler.RegisterRoutes(dockerBackupGroup)

	// Branding routes (Enterprise - White Label)
	brandingGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureWhiteLabel, logger))
	brandingHandler := handlers.NewBrandingHandler(database, logger)
	brandingHandler.RegisterRoutes(brandingGroup)

	// DR Runbook routes (Enterprise)
	drRunbooksGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureDRRunbooks, logger))
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

	// DR Runbook routes
	drRunbooksHandler := handlers.NewDRRunbooksHandler(database, logger)
	drRunbooksHandler.RegisterRoutes(drRunbooksGroup)

	// DR Test routes (Enterprise)
	drTestsGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureDRTests, logger))
	drTestsHandler := handlers.NewDRTestsHandler(database, cfg.DRTestRunner, logger)
	drTestsHandler.RegisterRoutes(drTestsGroup)

	// SLA Tracking routes (Enterprise)
	slaGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureSLATracking, logger))
	slaHandler := handlers.NewSLAHandler(database, logger)
	slaHandler.RegisterRoutes(slaGroup)

	// Air-gap routes (available to all tiers — air-gap is a deployment mode, not a premium feature)
	airGapHandler := handlers.NewAirGapHandler(logger)
	airGapHandler.RegisterRoutes(apiV1)

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

	// Lifecycle policy routes
	lifecyclePoliciesHandler := handlers.NewLifecyclePoliciesHandler(database, logger)
	lifecyclePoliciesHandler.RegisterRoutes(apiV1)

	// Agent API routes (API key auth required)
	// These endpoints are for agents to communicate with the server
	apiKeyValidator := auth.NewAPIKeyValidator(database, logger)
	agentAPI := r.Engine.Group("/api/v1/agent")
	agentAPI.Use(middleware.APIKeyMiddleware(apiKeyValidator, logger))
	if cfg.Validator != nil {
		agentAPI.Use(middleware.DynamicLicenseMiddleware(cfg.Validator, logger))
	} else {
		agentLic := cfg.License
		if agentLic == nil {
			agentLic = license.FreeLicense()
		}
		agentAPI.Use(middleware.LicenseMiddleware(agentLic, logger))
	}
	agentAPI.Use(middleware.FeatureMiddleware(license.FeatureAPIAccess, logger))
	agentAPI.Use(middleware.IPFilterAgentMiddleware(ipFilter, logger))

	agentAPIHandler := handlers.NewAgentAPIHandler(database, keyManager, logger)
	agentAPIHandler.RegisterRoutes(agentAPI)

	// Serve React SPA static files
	if cfg.WebDir != "" {
		indexPath := filepath.Join(cfg.WebDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			// Serve static assets (JS, CSS, images)
			r.Engine.Static("/assets", filepath.Join(cfg.WebDir, "assets"))

			// SPA fallback: serve index.html for all unmatched routes
			r.Engine.NoRoute(func(c *gin.Context) {
				// Don't serve SPA for API or auth routes
				if strings.HasPrefix(c.Request.URL.Path, "/api/") ||
					strings.HasPrefix(c.Request.URL.Path, "/auth/") ||
					strings.HasPrefix(c.Request.URL.Path, "/health") {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}
				c.File(indexPath)
			})
			r.logger.Info().Str("dir", cfg.WebDir).Msg("serving static frontend files")
		} else {
			r.logger.Warn().Str("dir", cfg.WebDir).Msg("web directory not found, SPA not served")
		}
	}

	r.logger.Info().Msg("API router initialized")
	return r, nil
}

