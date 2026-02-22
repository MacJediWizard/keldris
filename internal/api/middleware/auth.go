// Package middleware provides HTTP middleware for the Keldris API.
package middleware

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// UserStore is the interface for verifying users exist in the database.
type UserStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

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
	// AgentContextKey is the context key for the authenticated agent.
	AgentContextKey ContextKey = "agent"
	// SessionIDContextKey is the context key for the database session record ID.
	SessionIDContextKey ContextKey = "session_id"
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

		// Refresh last activity timestamp for idle timeout tracking
		if err := sessions.TouchSession(c.Request, c.Writer); err != nil {
			log.Warn().Err(err).Msg("failed to touch session")
		}

		// Store user in Gin context for handlers to access
		c.Set(string(UserContextKey), sessionUser)

		// Store session record ID if present
		if sessionUser.SessionRecordID != uuid.Nil {
			c.Set(string(SessionIDContextKey), sessionUser.SessionRecordID)
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

// UserVerifyMiddleware returns a Gin middleware that verifies the session user exists in the database.
// This catches stale sessions after a database reset. Must run after AuthMiddleware.
func UserVerifyMiddleware(store UserStore, sessions *auth.SessionStore, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "user_verify_middleware").Logger()

	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			c.Next()
			return
		}

		_, err := store.GetUserByID(c.Request.Context(), user.ID)
		if err != nil {
			log.Warn().
				Str("user_id", user.ID.String()).
				Msg("session user not found in database, clearing stale session")
			if clearErr := sessions.ClearUser(c.Request, c.Writer); clearErr != nil {
				log.Warn().Err(clearErr).Msg("failed to clear stale session")
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired, please log in again"})
			return
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

// GetCurrentSessionID retrieves the database session record ID from the Gin context.
// Returns uuid.Nil if no session ID is present.
func GetCurrentSessionID(c *gin.Context) uuid.UUID {
	sessionID, exists := c.Get(string(SessionIDContextKey))
	if !exists {
		return uuid.Nil
	}
	id, ok := sessionID.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}

// APIKeyMiddleware returns a Gin middleware that authenticates requests using API keys.
// This is used for agent-to-server communication.
func APIKeyMiddleware(validator *auth.APIKeyValidator, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "apikey_middleware").Logger()

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Debug().Str("path", c.Request.URL.Path).Msg("missing authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		apiKey := auth.ExtractBearerToken(authHeader)
		if apiKey == "" {
			log.Debug().Str("path", c.Request.URL.Path).Msg("invalid authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		agent, err := validator.ValidateAPIKey(c.Request.Context(), apiKey)
		if err != nil || agent == nil {
			log.Debug().Str("path", c.Request.URL.Path).Msg("invalid API key")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		// Store agent in Gin context for handlers to access
		c.Set(string(AgentContextKey), agent)

		log.Debug().
			Str("agent_id", agent.ID.String()).
			Str("hostname", agent.Hostname).
			Str("path", c.Request.URL.Path).
			Msg("authenticated agent request")

		c.Next()
	}
}

// GetAgent retrieves the authenticated agent from the Gin context.
// Returns nil if no agent is authenticated.
func GetAgent(c *gin.Context) *models.Agent {
	agent, exists := c.Get(string(AgentContextKey))
	if !exists {
		return nil
	}
	a, ok := agent.(*models.Agent)
	if !ok {
		return nil
	}
	return a
}

// RequireAgent is a helper that gets the authenticated agent or aborts with 401.
// Use this in handlers that expect APIKeyMiddleware to have already run.
func RequireAgent(c *gin.Context) *models.Agent {
	agent := GetAgent(c)
	if agent == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "agent authentication required"})
		return nil
	}
	return agent
}
