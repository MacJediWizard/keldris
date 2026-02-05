package middleware

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig holds configuration for security headers.
// All fields are optional; sensible defaults are applied when empty.
type SecurityHeadersConfig struct {
	// ContentSecurityPolicy sets the Content-Security-Policy header.
	// If empty, a restrictive default policy is used.
	ContentSecurityPolicy string

	// FrameOptions sets the X-Frame-Options header.
	// Defaults to "DENY" if empty.
	FrameOptions string

	// ContentTypeOptions sets the X-Content-Type-Options header.
	// Defaults to "nosniff" if empty.
	ContentTypeOptions string

	// StrictTransportSecurity sets the Strict-Transport-Security header.
	// Only applied if EnableHSTS is true. Defaults to max-age=31536000.
	StrictTransportSecurity string

	// EnableHSTS determines whether HSTS header is added.
	// Should be true in production with HTTPS.
	EnableHSTS bool

	// ReferrerPolicy sets the Referrer-Policy header.
	// Defaults to "strict-origin-when-cross-origin" if empty.
	ReferrerPolicy string

	// PermissionsPolicy sets the Permissions-Policy header.
	// Defaults to a restrictive policy if empty.
	PermissionsPolicy string

	// CrossOriginOpenerPolicy sets the Cross-Origin-Opener-Policy header.
	// Defaults to "same-origin" if empty.
	CrossOriginOpenerPolicy string

	// CrossOriginResourcePolicy sets the Cross-Origin-Resource-Policy header.
	// Defaults to "same-origin" if empty.
	CrossOriginResourcePolicy string

	// CrossOriginEmbedderPolicy sets the Cross-Origin-Embedder-Policy header.
	// Defaults to "require-corp" if empty.
	CrossOriginEmbedderPolicy string

	// AdditionalCSPDirectives allows adding extra CSP directives for white-label.
	// These are appended to the default policy.
	AdditionalCSPDirectives map[string]string
}

// DefaultSecurityHeadersConfig returns a SecurityHeadersConfig with secure defaults.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:     defaultCSP(),
		FrameOptions:              "DENY",
		ContentTypeOptions:        "nosniff",
		StrictTransportSecurity:   "max-age=31536000; includeSubDomains",
		EnableHSTS:                true,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         defaultPermissionsPolicy(),
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		CrossOriginEmbedderPolicy: "require-corp",
	}
}

// DevelopmentSecurityHeadersConfig returns a less restrictive config for development.
func DevelopmentSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:     developmentCSP(),
		FrameOptions:              "DENY",
		ContentTypeOptions:        "nosniff",
		EnableHSTS:                false, // Don't enable HSTS in development
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         defaultPermissionsPolicy(),
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		CrossOriginEmbedderPolicy: "credentialless", // Less restrictive for dev
	}
}

// defaultCSP returns the default Content-Security-Policy.
func defaultCSP() string {
	directives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self' 'unsafe-inline'", // inline styles often needed for UI frameworks
		"img-src 'self' data: https:",
		"font-src 'self'",
		"connect-src 'self'",
		"frame-ancestors 'none'",
		"form-action 'self'",
		"base-uri 'self'",
		"object-src 'none'",
		"upgrade-insecure-requests",
	}
	return strings.Join(directives, "; ")
}

// developmentCSP returns a more permissive CSP for development.
func developmentCSP() string {
	directives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'", // Allow for hot reload, dev tools
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: https: http:",
		"font-src 'self' data:",
		"connect-src 'self' ws: wss: http: https:", // Allow WebSocket for hot reload
		"frame-ancestors 'self'",
		"form-action 'self'",
		"base-uri 'self'",
		"object-src 'none'",
	}
	return strings.Join(directives, "; ")
}

// defaultPermissionsPolicy returns restrictive permissions policy.
func defaultPermissionsPolicy() string {
	// Disable potentially dangerous browser features
	policies := []string{
		"accelerometer=()",
		"ambient-light-sensor=()",
		"autoplay=()",
		"battery=()",
		"camera=()",
		"display-capture=()",
		"document-domain=()",
		"encrypted-media=()",
		"fullscreen=(self)",
		"geolocation=()",
		"gyroscope=()",
		"magnetometer=()",
		"microphone=()",
		"midi=()",
		"payment=()",
		"picture-in-picture=()",
		"publickey-credentials-get=()",
		"screen-wake-lock=()",
		"usb=()",
		"xr-spatial-tracking=()",
	}
	return strings.Join(policies, ", ")
}

// SecurityHeaders returns middleware that sets security headers on all responses.
func SecurityHeaders(cfg SecurityHeadersConfig) gin.HandlerFunc {
	// Build CSP with any additional directives
	csp := buildCSP(cfg)

	// Apply defaults for empty values
	frameOptions := cfg.FrameOptions
	if frameOptions == "" {
		frameOptions = "DENY"
	}

	contentTypeOptions := cfg.ContentTypeOptions
	if contentTypeOptions == "" {
		contentTypeOptions = "nosniff"
	}

	hsts := cfg.StrictTransportSecurity
	if hsts == "" {
		hsts = "max-age=31536000; includeSubDomains"
	}

	referrerPolicy := cfg.ReferrerPolicy
	if referrerPolicy == "" {
		referrerPolicy = "strict-origin-when-cross-origin"
	}

	permissionsPolicy := cfg.PermissionsPolicy
	if permissionsPolicy == "" {
		permissionsPolicy = defaultPermissionsPolicy()
	}

	coop := cfg.CrossOriginOpenerPolicy
	if coop == "" {
		coop = "same-origin"
	}

	corp := cfg.CrossOriginResourcePolicy
	if corp == "" {
		corp = "same-origin"
	}

	coep := cfg.CrossOriginEmbedderPolicy
	if coep == "" {
		coep = "require-corp"
	}

	return func(c *gin.Context) {
		// Set security headers before handling request
		c.Header("X-Frame-Options", frameOptions)
		c.Header("X-Content-Type-Options", contentTypeOptions)
		c.Header("Referrer-Policy", referrerPolicy)
		c.Header("Permissions-Policy", permissionsPolicy)
		c.Header("Cross-Origin-Opener-Policy", coop)
		c.Header("Cross-Origin-Resource-Policy", corp)
		c.Header("Cross-Origin-Embedder-Policy", coep)

		// Content-Security-Policy
		if csp != "" {
			c.Header("Content-Security-Policy", csp)
		}

		// HSTS - only set if explicitly enabled (should be HTTPS)
		if cfg.EnableHSTS {
			c.Header("Strict-Transport-Security", hsts)
		}

		// Legacy headers for older browsers
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Permitted-Cross-Domain-Policies", "none")

		c.Next()
	}
}

// buildCSP builds the Content-Security-Policy header value from config.
func buildCSP(cfg SecurityHeadersConfig) string {
	if cfg.ContentSecurityPolicy == "" && len(cfg.AdditionalCSPDirectives) == 0 {
		return defaultCSP()
	}

	baseCSP := cfg.ContentSecurityPolicy
	if baseCSP == "" {
		baseCSP = defaultCSP()
	}

	// If no additional directives, return base
	if len(cfg.AdditionalCSPDirectives) == 0 {
		return baseCSP
	}

	// Parse existing directives
	existingDirectives := parseCSP(baseCSP)

	// Merge additional directives (additional values are appended)
	for directive, value := range cfg.AdditionalCSPDirectives {
		if existing, ok := existingDirectives[directive]; ok {
			// Append to existing directive
			existingDirectives[directive] = existing + " " + value
		} else {
			// Add new directive
			existingDirectives[directive] = value
		}
	}

	// Rebuild CSP string
	return buildCSPString(existingDirectives)
}

// parseCSP parses a CSP header into directive-value pairs.
func parseCSP(csp string) map[string]string {
	directives := make(map[string]string)
	parts := strings.Split(csp, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		// Split on first space to get directive name and value
		spaceIdx := strings.Index(part, " ")
		if spaceIdx == -1 {
			// Directive with no value (e.g., upgrade-insecure-requests)
			directives[part] = ""
		} else {
			directive := part[:spaceIdx]
			value := part[spaceIdx+1:]
			directives[directive] = value
		}
	}
	return directives
}

// buildCSPString builds a CSP header string from directive-value pairs.
func buildCSPString(directives map[string]string) string {
	var parts []string
	for directive, value := range directives {
		if value == "" {
			parts = append(parts, directive)
		} else {
			parts = append(parts, fmt.Sprintf("%s %s", directive, value))
		}
	}
	return strings.Join(parts, "; ")
}

// GetSecurityHeadersInfo returns information about the current security headers.
// This is useful for testing and verification endpoints.
type SecurityHeadersInfo struct {
	ContentSecurityPolicy     string `json:"content_security_policy"`
	XFrameOptions             string `json:"x_frame_options"`
	XContentTypeOptions       string `json:"x_content_type_options"`
	StrictTransportSecurity   string `json:"strict_transport_security,omitempty"`
	ReferrerPolicy            string `json:"referrer_policy"`
	PermissionsPolicy         string `json:"permissions_policy"`
	CrossOriginOpenerPolicy   string `json:"cross_origin_opener_policy"`
	CrossOriginResourcePolicy string `json:"cross_origin_resource_policy"`
	CrossOriginEmbedderPolicy string `json:"cross_origin_embedder_policy"`
	XSSProtection             string `json:"x_xss_protection"`
	PermittedCrossDomain      string `json:"x_permitted_cross_domain_policies"`
}

// GetSecurityHeadersFromContext extracts security header values from a gin context.
func GetSecurityHeadersFromContext(c *gin.Context) SecurityHeadersInfo {
	return SecurityHeadersInfo{
		ContentSecurityPolicy:     c.Writer.Header().Get("Content-Security-Policy"),
		XFrameOptions:             c.Writer.Header().Get("X-Frame-Options"),
		XContentTypeOptions:       c.Writer.Header().Get("X-Content-Type-Options"),
		StrictTransportSecurity:   c.Writer.Header().Get("Strict-Transport-Security"),
		ReferrerPolicy:            c.Writer.Header().Get("Referrer-Policy"),
		PermissionsPolicy:         c.Writer.Header().Get("Permissions-Policy"),
		CrossOriginOpenerPolicy:   c.Writer.Header().Get("Cross-Origin-Opener-Policy"),
		CrossOriginResourcePolicy: c.Writer.Header().Get("Cross-Origin-Resource-Policy"),
		CrossOriginEmbedderPolicy: c.Writer.Header().Get("Cross-Origin-Embedder-Policy"),
		XSSProtection:             c.Writer.Header().Get("X-XSS-Protection"),
		PermittedCrossDomain:      c.Writer.Header().Get("X-Permitted-Cross-Domain-Policies"),
	}
}
