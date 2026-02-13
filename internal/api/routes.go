// Package api provides the HTTP API for the Keldris server.
package api

import (
	"github.com/MacJediWizard/keldris/internal/api/handlers"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/license"
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
	// VerificationTrigger for manually triggering verifications (optional).
	VerificationTrigger handlers.VerificationTrigger
	// ReportScheduler for report generation and sending (optional).
	ReportScheduler *reports.Scheduler
	// DRTestRunner for triggering DR test execution (optional).
	DRTestRunner handlers.DRTestRunner
	// License is the current server license for feature gating (optional).
	License *license.License
}

// DefaultConfig returns a Config with sensible defaults for development.
func DefaultConfig() Config {
	return Config{
		Environment:       config.EnvDevelopment,
		AllowedOrigins:    []string{},
		RateLimitRequests: 100,
		RateLimitPeriod:   "1m",
		Version:           "dev",
		Commit:            "unknown",
		BuildDate:         "unknown",
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

	// Auth routes (no auth required)
	authGroup := r.Engine.Group("/auth")
	authHandler := handlers.NewAuthHandler(oidc, sessions, database, logger)
	authHandler.RegisterRoutes(authGroup)

	// API v1 routes (auth required)
	apiV1 := r.Engine.Group("/api/v1")
	apiV1.Use(middleware.AuthMiddleware(sessions, logger))
	apiV1.Use(middleware.AuditMiddleware(database, logger))

	// License middleware for feature gating
	lic := cfg.License
	if lic == nil {
		lic = license.FreeLicense()
	}
	apiV1.Use(middleware.LicenseMiddleware(lic, logger))

	// Create RBAC for permission checks
	rbac := auth.NewRBAC(database)

	// Register API handlers
	versionHandler.RegisterRoutes(apiV1)

	orgsHandler := handlers.NewOrganizationsHandler(database, sessions, rbac, logger)
	orgsHandler.RegisterRoutes(apiV1)

	agentsHandler := handlers.NewAgentsHandler(database, logger)
	agentsHandler.RegisterRoutes(apiV1)

	agentGroupsHandler := handlers.NewAgentGroupsHandler(database, logger)
	agentGroupsHandler.RegisterRoutes(apiV1)

	reposHandler := handlers.NewRepositoriesHandler(database, keyManager, logger)
	reposHandler.RegisterRoutes(apiV1)

	schedulesHandler := handlers.NewSchedulesHandler(database, logger)
	schedulesHandler.RegisterRoutes(apiV1)

	backupScriptsHandler := handlers.NewBackupScriptsHandler(database, logger)
	backupScriptsHandler.RegisterRoutes(apiV1)

	policiesHandler := handlers.NewPoliciesHandler(database, logger)
	policiesHandler.RegisterRoutes(apiV1)

	backupsHandler := handlers.NewBackupsHandler(database, logger)
	backupsHandler.RegisterRoutes(apiV1)

	snapshotsHandler := handlers.NewSnapshotsHandler(database, logger)
	snapshotsHandler.RegisterRoutes(apiV1)

	fileHistoryHandler := handlers.NewFileHistoryHandler(database, logger)
	fileHistoryHandler.RegisterRoutes(apiV1)

	auditLogsHandler := handlers.NewAuditLogsHandler(database, logger)
	auditLogsHandler.RegisterRoutes(apiV1)

	alertsHandler := handlers.NewAlertsHandler(database, logger)
	alertsHandler.RegisterRoutes(apiV1)

	notificationsGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureNotificationSlack, logger))
	notificationsHandler := handlers.NewNotificationsHandler(database, keyManager, logger)
	notificationsHandler.RegisterRoutes(notificationsGroup)

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

	// DR Runbook routes (Enterprise)
	drRunbooksGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureDRRunbooks, logger))
	drRunbooksHandler := handlers.NewDRRunbooksHandler(database, logger)
	drRunbooksHandler.RegisterRoutes(drRunbooksGroup)

	// DR Test routes (Enterprise)
	drTestsGroup := apiV1.Group("", middleware.FeatureMiddleware(license.FeatureDRTests, logger))
	drTestsHandler := handlers.NewDRTestsHandler(database, cfg.DRTestRunner, logger)
	drTestsHandler.RegisterRoutes(drTestsGroup)

	// Agent API routes (API key auth required)
	// These endpoints are for agents to communicate with the server
	apiKeyValidator := auth.NewAPIKeyValidator(database, logger)
	agentAPI := r.Engine.Group("/api/v1/agent")
	agentAPI.Use(middleware.APIKeyMiddleware(apiKeyValidator, logger))

	agentAPIHandler := handlers.NewAgentAPIHandler(database, logger)
	agentAPIHandler.RegisterRoutes(agentAPI)

	r.logger.Info().Msg("API router initialized")
	return r, nil
}

