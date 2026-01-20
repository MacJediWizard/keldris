// Package api provides the HTTP API for the Keldris server.
package api

import (
	"github.com/MacJediWizard/keldris/internal/api/handlers"
	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
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
}

// DefaultConfig returns a Config with sensible defaults for development.
func DefaultConfig() Config {
	return Config{
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
	Engine   *gin.Engine
	logger   zerolog.Logger
	sessions *auth.SessionStore
	db       *db.DB
}

// NewRouter creates a new Router with the given dependencies.
func NewRouter(
	cfg Config,
	database *db.DB,
	oidc *auth.OIDC,
	sessions *auth.SessionStore,
	logger zerolog.Logger,
) (*Router, error) {
	r := &Router{
		Engine:   gin.New(),
		logger:   logger.With().Str("component", "router").Logger(),
		sessions: sessions,
		db:       database,
	}

	// Global middleware
	r.Engine.Use(gin.Recovery())
	r.Engine.Use(middleware.RequestLogger(logger))
	r.Engine.Use(middleware.CORS(cfg.AllowedOrigins))

	// Rate limiting
	rateLimiter, err := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitPeriod)
	if err != nil {
		return nil, err
	}
	r.Engine.Use(rateLimiter)

	// Health check endpoint (no auth required)
	r.Engine.GET("/health", r.healthCheck)

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

	// Register API handlers
	versionHandler.RegisterRoutes(apiV1)

	agentsHandler := handlers.NewAgentsHandler(database, logger)
	agentsHandler.RegisterRoutes(apiV1)

	reposHandler := handlers.NewRepositoriesHandler(database, logger)
	reposHandler.RegisterRoutes(apiV1)

	schedulesHandler := handlers.NewSchedulesHandler(database, logger)
	schedulesHandler.RegisterRoutes(apiV1)

	backupsHandler := handlers.NewBackupsHandler(database, logger)
	backupsHandler.RegisterRoutes(apiV1)

	r.logger.Info().Msg("API router initialized")
	return r, nil
}

// healthCheck returns basic health information.
func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"db":     r.db.Health(),
	})
}
