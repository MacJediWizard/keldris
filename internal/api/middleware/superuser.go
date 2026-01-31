package middleware

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// SuperuserMiddleware returns a Gin middleware that requires superuser privileges.
func SuperuserMiddleware(sessions *auth.SessionStore, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "superuser_middleware").Logger()

	return func(c *gin.Context) {
		user := GetUser(c)
		if user == nil {
			log.Debug().Str("path", c.Request.URL.Path).Msg("unauthenticated request to superuser endpoint")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		if !user.IsSuperuser {
			log.Warn().
				Str("user_id", user.ID.String()).
				Str("path", c.Request.URL.Path).
				Msg("non-superuser attempted to access superuser endpoint")
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "superuser privileges required"})
			return
		}

		log.Debug().
			Str("user_id", user.ID.String()).
			Str("path", c.Request.URL.Path).
			Msg("superuser access granted")

		c.Next()
	}
}

// RequireSuperuser is a helper that checks superuser status or aborts with 403.
func RequireSuperuser(c *gin.Context) *auth.SessionUser {
	user := GetUser(c)
	if user == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil
	}
	if !user.IsSuperuser {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "superuser privileges required"})
		return nil
	}
	return user
}

// IsSuperuser returns true if the current user is a superuser.
func IsSuperuser(c *gin.Context) bool {
	user := GetUser(c)
	return user != nil && user.IsSuperuser
}

// IsImpersonating returns true if the current session is impersonating another user.
func IsImpersonating(c *gin.Context) bool {
	user := GetUser(c)
	return user != nil && user.ImpersonatingID != [16]byte{}
}
