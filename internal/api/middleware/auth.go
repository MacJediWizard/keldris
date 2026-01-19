// Package middleware provides HTTP middleware for the Keldris API.
package middleware

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ContextKey is the type for context keys used by this package.
type ContextKey string

const (
	// UserContextKey is the context key for the authenticated user.
	UserContextKey ContextKey = "user"
)

// AuthMiddleware returns a Gin middleware that requires authentication.
func AuthMiddleware(sessions *auth.SessionStore, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "auth_middleware").Logger()

	return func(c *gin.Context) {
		sessionUser, err := sessions.GetUser(c.Request)
		if err != nil {
			log.Debug().Err(err).Str("path", c.Request.URL.Path).Msg("unauthenticated request")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// Store user in Gin context for handlers to access
		c.Set(string(UserContextKey), sessionUser)

		log.Debug().
			Str("user_id", sessionUser.ID.String()).
			Str("path", c.Request.URL.Path).
			Msg("authenticated request")

		c.Next()
	}
}

// OptionalAuthMiddleware returns a Gin middleware that loads user if present but doesn't require it.
func OptionalAuthMiddleware(sessions *auth.SessionStore, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "auth_middleware").Logger()

	return func(c *gin.Context) {
		sessionUser, err := sessions.GetUser(c.Request)
		if err == nil {
			c.Set(string(UserContextKey), sessionUser)
			log.Debug().
				Str("user_id", sessionUser.ID.String()).
				Str("path", c.Request.URL.Path).
				Msg("authenticated request (optional)")
		}
		c.Next()
	}
}

// GetUser retrieves the authenticated user from the Gin context.
// Returns nil if no user is authenticated.
func GetUser(c *gin.Context) *auth.SessionUser {
	user, exists := c.Get(string(UserContextKey))
	if !exists {
		return nil
	}
	sessionUser, ok := user.(*auth.SessionUser)
	if !ok {
		return nil
	}
	return sessionUser
}

// RequireUser is a helper that gets the authenticated user or aborts with 401.
// Use this in handlers that expect AuthMiddleware to have already run.
func RequireUser(c *gin.Context) *auth.SessionUser {
	user := GetUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil
	}
	return user
}
