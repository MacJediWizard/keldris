package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// RequestLogger returns a middleware that logs HTTP requests using zerolog.
func RequestLogger(logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "http").Logger()

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		event := log.Info()
		if status >= 400 && status < 500 {
			event = log.Warn()
		} else if status >= 500 {
			event = log.Error()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", status).
			Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Int("body_size", c.Writer.Size()).
			Msg("request")
	}
}
