package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// SetupStore defines the interface for checking setup status.
type SetupStore interface {
	IsSetupComplete(ctx context.Context) (bool, error)
}

// SetupRequiredMiddleware returns a Gin middleware that blocks all non-setup routes
// when server setup has not been completed. This ensures users are redirected to
// the setup wizard on fresh installs.
func SetupRequiredMiddleware(store SetupStore, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "setup_middleware").Logger()

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Always allow setup endpoints
		if strings.HasPrefix(path, "/api/v1/setup") {
			c.Next()
			return
		}

		// Always allow health and branding endpoints
		if path == "/api/v1/health" || path == "/health" || path == "/api/v1/branding" {
			c.Next()
			return
		}

		// Always allow auth endpoints (needed for post-setup login)
		if strings.HasPrefix(path, "/auth/") {
			c.Next()
			return
		}

		// Always allow static assets and frontend routes
		if strings.HasPrefix(path, "/assets/") || strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".css") ||
			strings.HasSuffix(path, ".ico") || strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".svg") {
			c.Next()
			return
		}

		// Always allow frontend SPA routes (served as index.html by the catch-all)
		if path == "/" || path == "/setup" || path == "/login" || !strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		// Check if setup is complete
		isComplete, err := store.IsSetupComplete(c.Request.Context())
		if err != nil {
			log.Error().Err(err).Msg("failed to check setup status")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check setup status"})
			c.Abort()
			return
		}

		if !isComplete {
			log.Debug().Str("path", path).Msg("setup not complete, redirecting to setup")
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":    "server setup required",
				"redirect": "/setup",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SetupLockMiddleware returns a Gin middleware that prevents accessing setup endpoints
// after setup has been completed (except for superuser re-run endpoints).
func SetupLockMiddleware(store SetupStore, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "setup_lock_middleware").Logger()

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Always allow status checks
		if path == "/api/v1/setup/status" {
			c.Next()
			return
		}

		// Check if setup is complete
		isComplete, err := store.IsSetupComplete(c.Request.Context())
		if err != nil {
			log.Error().Err(err).Msg("failed to check setup status")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check setup status"})
			c.Abort()
			return
		}

		if !isComplete {
			// Setup not complete, allow access to setup endpoints
			c.Next()
			return
		}

		// Setup is complete - only allow superuser re-run endpoints
		if strings.HasPrefix(path, "/api/v1/setup/rerun") {
			c.Next()
			return
		}

		log.Warn().Str("path", path).Msg("setup already complete, blocking access")
		c.JSON(http.StatusForbidden, gin.H{"error": "server setup already completed"})
		c.Abort()
	}
}
