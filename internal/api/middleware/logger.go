package middleware

import (
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// sensitiveParams lists query parameter names whose values must be redacted from logs.
var sensitiveParams = map[string]bool{
	"token":    true,
	"key":      true,
	"secret":   true,
	"password": true,
	"code":     true,
	"state":    true,
}

// redactQueryString replaces values of known sensitive query parameters with [REDACTED].
func redactQueryString(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}

	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}

	redacted := false
	for name, values := range params {
		if sensitiveParams[strings.ToLower(name)] {
			for i := range values {
				values[i] = "[REDACTED]"
			}
			params[name] = values
			redacted = true
		}
	}

	if !redacted {
		return rawQuery
	}

	return params.Encode()
}

// RequestLogger returns a middleware that logs HTTP requests using zerolog.
func RequestLogger(logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "http").Logger()

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := redactQueryString(c.Request.URL.RawQuery)
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
