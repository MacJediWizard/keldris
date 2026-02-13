package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimitMiddleware returns a Gin middleware that limits the size of request bodies.
// Requests exceeding maxBytes will receive a 413 Request Entity Too Large response.
func BodyLimitMiddleware(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
