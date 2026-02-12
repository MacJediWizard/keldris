package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/gin-gonic/gin"
)

// CSPNonceKey is the gin.Context key where the per-request CSP nonce is stored.
// Use GetCSPNonce to retrieve it from a handler.
const CSPNonceKey = "csp_nonce"

// nonceBytes is the number of random bytes used to generate CSP nonces.
// 16 bytes yields 22 base64 characters, providing 128 bits of entropy.
const nonceBytes = 16

// cspAPI is a strict Content-Security-Policy for API routes that return JSON.
// No scripts, styles, or other resources should be loaded from API responses.
const cspAPI = "default-src 'none'; frame-ancestors 'none'"

// cspSwaggerDev is the Content-Security-Policy for Swagger UI routes in development.
// Swagger UI (swaggo) requires 'unsafe-inline' for its inline styles and scripts
// which we cannot control. This policy is only applied in non-production environments.
// Swagger UI is disabled entirely in production for security.
const cspSwaggerDev = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'"

// GetCSPNonce retrieves the per-request CSP nonce from the gin context.
// Returns an empty string if no nonce is set (e.g., on API routes).
func GetCSPNonce(c *gin.Context) string {
	if v, ok := c.Get(CSPNonceKey); ok {
		return v.(string)
	}
	return ""
}

// generateNonce creates a cryptographically random base64-encoded nonce.
func generateNonce() (string, error) {
	b := make([]byte, nonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate CSP nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// SecurityHeaders returns a middleware that sets security-related HTTP response headers.
// The env parameter controls whether development-only CSP exceptions (e.g., Swagger UI)
// are applied. In production, all routes use strict CSP without 'unsafe-inline'.
func SecurityHeaders(env config.Environment) gin.HandlerFunc {
	isProduction := env == config.EnvProduction

	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HSTS - only in production with TLS
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		path := c.Request.URL.Path

		switch {
		case isAPIRoute(path):
			// Strict CSP for API routes (JSON-only responses).
			c.Header("Content-Security-Policy", cspAPI)
		case !isProduction && isSwaggerRoute(path):
			// Swagger UI requires 'unsafe-inline' for its inline content.
			// Only applied in non-production; Swagger UI is disabled in production.
			c.Header("Content-Security-Policy", cspSwaggerDev)
		default:
			// Frontend routes use nonce-based CSP to avoid 'unsafe-inline'.
			nonce, err := generateNonce()
			if err != nil {
				// Fall back to strict default-src 'self' if nonce generation fails.
				c.Header("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
				c.Next()
				return
			}
			c.Set(CSPNonceKey, nonce)
			csp := fmt.Sprintf(
				"default-src 'self'; script-src 'self' 'nonce-%s'; style-src 'self' 'nonce-%s'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'",
				nonce, nonce,
			)
			c.Header("Content-Security-Policy", csp)
		}

		c.Next()
	}
}

// isAPIRoute returns true for paths that only serve JSON responses.
func isAPIRoute(path string) bool {
	return strings.HasPrefix(path, "/api/v1/") ||
		strings.HasPrefix(path, "/auth/")
}

// isSwaggerRoute returns true for Swagger UI documentation paths.
func isSwaggerRoute(path string) bool {
	return strings.HasPrefix(path, "/api/docs/")
}
