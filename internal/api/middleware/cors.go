package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
// In production, allowedOrigins must not be empty or the server will panic.
// In non-production environments, empty allowedOrigins allows all origins with a warning.
func CORS(allowedOrigins []string, env config.Environment) gin.HandlerFunc {
	if len(allowedOrigins) == 0 {
		if env == config.EnvProduction {
			panic("CORS_ORIGINS must be set in production; refusing to start with open CORS policy")
		}
		log.Println("WARNING: CORS_ORIGINS is empty, all origins are allowed (not suitable for production)")
	}

	allowAll := len(allowedOrigins) == 0

	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		originSet[strings.ToLower(origin)] = struct{}{}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Determine if origin is allowed
		allowed := allowAll
		if !allowed && origin != "" {
			_, allowed = originSet[strings.ToLower(origin)]
		}

		if allowed && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Max-Age", "86400")
		}

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
