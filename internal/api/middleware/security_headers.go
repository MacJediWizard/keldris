package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// cspAPI is a strict Content-Security-Policy for API routes that return JSON.
// No scripts, styles, or other resources should be loaded from API responses.
const cspAPI = "default-src 'none'; frame-ancestors 'none'"

// cspFrontend is the Content-Security-Policy for routes that serve HTML content
// (e.g., Swagger docs, or a future embedded frontend).
//
// 'unsafe-inline' is required for script-src and style-src because:
//   - React/Vite injects inline scripts during development (HMR client)
//   - Vite production builds may emit inline script tags for module preloading
//   - Swagger UI (swaggo) uses inline styles and scripts
//   - Tailwind CSS utilities can generate inline style attributes
//
// TODO: Replace 'unsafe-inline' with nonce-based CSP. This requires:
//   1. Generate a per-request nonce in this middleware
//   2. Pass the nonce to the HTML template via context
//   3. Add nonce attributes to all <script> and <style> tags
//   4. Use 'strict-dynamic' with the nonce for script-src
const cspFrontend = "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'"

// SecurityHeaders returns a middleware that sets security-related HTTP response headers.
func SecurityHeaders() gin.HandlerFunc {
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

		// Apply strict CSP for API routes (JSON-only), relaxed CSP for HTML-serving routes.
		if isAPIRoute(c.Request.URL.Path) {
			c.Header("Content-Security-Policy", cspAPI)
		} else {
			c.Header("Content-Security-Policy", cspFrontend)
		}

		c.Next()
	}
}

// isAPIRoute returns true for paths that only serve JSON responses.
func isAPIRoute(path string) bool {
	return strings.HasPrefix(path, "/api/v1/") ||
		strings.HasPrefix(path, "/auth/")
}
