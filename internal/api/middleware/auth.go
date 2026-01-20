// Package middleware provides HTTP middleware for the Keldris API.
package middleware

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
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

// SessionOrAPIKeyMiddleware returns a Gin middleware that accepts either session or API key auth.
// Useful for endpoints that can be called by both users and agents.
func SessionOrAPIKeyMiddleware(sessions *auth.SessionStore, validator *auth.APIKeyValidator, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "dual_auth_middleware").Logger()

	return func(c *gin.Context) {
		// Try session auth first
		sessionUser, err := sessions.GetUser(c.Request)
		if err == nil && sessionUser != nil {
			c.Set(string(UserContextKey), sessionUser)
			log.Debug().
				Str("user_id", sessionUser.ID.String()).
				Str("path", c.Request.URL.Path).
				Msg("authenticated via session")
			c.Next()
			return
		}

		// Try API key auth
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			apiKey := auth.ExtractBearerToken(authHeader)
			if apiKey != "" {
				agent, err := validator.ValidateAPIKey(c.Request.Context(), apiKey)
				if err == nil && agent != nil {
					c.Set(string(AgentContextKey), agent)
					log.Debug().
						Str("agent_id", agent.ID.String()).
						Str("hostname", agent.Hostname).
						Str("path", c.Request.URL.Path).
						Msg("authenticated via API key")
					c.Next()
					return
				}
			}
		}

		log.Debug().Str("path", c.Request.URL.Path).Msg("unauthenticated request")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
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
