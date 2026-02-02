package portal

import (
	"github.com/MacJediWizard/keldris/internal/portal/handlers"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Config holds configuration for the portal router.
type Config struct {
	// AllowedOrigins for CORS. Empty means all origins allowed in dev mode.
	AllowedOrigins []string
	// Version information for the version endpoint.
	Version   string
	Commit    string
	BuildDate string
}

// DefaultConfig returns a Config with sensible defaults for development.
func DefaultConfig() Config {
	return Config{
		AllowedOrigins: []string{},
		Version:        "dev",
		Commit:         "unknown",
		BuildDate:      "unknown",
	}
}

// Router wraps a Gin engine with configured middleware and routes.
type Router struct {
	Engine *gin.Engine
	logger zerolog.Logger
	store  Store
}

// NewRouter creates a new Router with the given dependencies.
func NewRouter(cfg Config, store Store, logger zerolog.Logger) (*Router, error) {
	r := &Router{
		Engine: gin.New(),
		logger: logger.With().Str("component", "portal_router").Logger(),
		store:  store,
	}

	// Global middleware
	r.Engine.Use(gin.Recovery())
	r.Engine.Use(requestLogger(logger))
	r.Engine.Use(cors(cfg.AllowedOrigins))

	// Health check endpoint (no auth required)
	r.Engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Version endpoint (no auth required)
	r.Engine.GET("/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":    cfg.Version,
			"commit":     cfg.Commit,
			"build_date": cfg.BuildDate,
		})
	})

	// Public auth routes (no auth required)
	authHandler := handlers.NewAuthHandler(store, logger)
	authHandler.RegisterRoutes(r.Engine.Group("/api/v1"))

	// Protected routes (auth required)
	protected := r.Engine.Group("/api/v1")
	protected.Use(AuthMiddleware(store, logger))

	// Protected auth routes
	authHandler.RegisterProtectedRoutes(protected)

	// License routes
	licensesHandler := handlers.NewLicensesHandler(store, logger)
	licensesHandler.RegisterRoutes(protected)

	// Invoice routes
	invoicesHandler := handlers.NewInvoicesHandler(store, logger)
	invoicesHandler.RegisterRoutes(protected)

	r.logger.Info().Msg("Portal router initialized")
	return r, nil
}

// requestLogger is middleware that logs HTTP requests.
func requestLogger(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Str("client_ip", c.ClientIP()).
			Msg("request")
	}
}

// cors is middleware that handles CORS.
func cors(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// Check if origin is allowed
		allowed := false
		if len(allowedOrigins) == 0 {
			// Allow all in dev mode
			allowed = true
		} else {
			for _, o := range allowedOrigins {
				if o == origin || o == "*" {
					allowed = true
					break
				}
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
