package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSecurityHeaders_DefaultConfig(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig()
	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	tests := []struct {
		header   string
		expected string
		contains bool // if true, check contains; if false, check exact match
	}{
		{"X-Frame-Options", "DENY", false},
		{"X-Content-Type-Options", "nosniff", false},
		{"Referrer-Policy", "strict-origin-when-cross-origin", false},
		{"X-XSS-Protection", "1; mode=block", false},
		{"X-Permitted-Cross-Domain-Policies", "none", false},
		{"Cross-Origin-Opener-Policy", "same-origin", false},
		{"Cross-Origin-Resource-Policy", "same-origin", false},
		{"Cross-Origin-Embedder-Policy", "credentialless", false},
		{"Content-Security-Policy", "default-src", true},
		{"Strict-Transport-Security", "max-age=", true},
		{"Permissions-Policy", "camera=()", true},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			value := w.Header().Get(tt.header)
			if value == "" {
				t.Errorf("Expected %s header to be set", tt.header)
				return
			}
			if tt.contains {
				if !strings.Contains(value, tt.expected) {
					t.Errorf("%s = %q, want to contain %q", tt.header, value, tt.expected)
				}
			} else {
				if value != tt.expected {
					t.Errorf("%s = %q, want %q", tt.header, value, tt.expected)
				}
			}
		})
	}
}

func TestSecurityHeaders_DevelopmentConfig(t *testing.T) {
	cfg := DevelopmentSecurityHeadersConfig()
	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// HSTS should NOT be set in development
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("Strict-Transport-Security should not be set in development mode")
	}

	// CSP should allow unsafe-eval for development
	csp := w.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "unsafe-eval") {
		t.Errorf("Development CSP should contain 'unsafe-eval', got: %s", csp)
	}
}

func TestSecurityHeaders_CustomConfig(t *testing.T) {
	cfg := SecurityHeadersConfig{
		FrameOptions:       "SAMEORIGIN",
		ContentTypeOptions: "nosniff",
		ReferrerPolicy:     "no-referrer",
		EnableHSTS:         false,
	}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if got := w.Header().Get("X-Frame-Options"); got != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "SAMEORIGIN")
	}

	if got := w.Header().Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy = %q, want %q", got, "no-referrer")
	}

	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS should not be set when EnableHSTS is false")
	}
}

func TestSecurityHeaders_AdditionalCSPDirectives(t *testing.T) {
	cfg := SecurityHeadersConfig{
		AdditionalCSPDirectives: map[string]string{
			"script-src": "https://cdn.example.com",
			"frame-src":  "'self' https://embed.example.com",
		},
	}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")

	// script-src should have the additional source appended
	if !strings.Contains(csp, "https://cdn.example.com") {
		t.Errorf("CSP should contain additional script-src, got: %s", csp)
	}

	// frame-src should be added as new directive
	if !strings.Contains(csp, "frame-src") {
		t.Errorf("CSP should contain frame-src directive, got: %s", csp)
	}
}

func TestSecurityHeaders_CustomCSP(t *testing.T) {
	customCSP := "default-src 'none'; script-src 'self'; style-src 'self'"
	cfg := SecurityHeadersConfig{
		ContentSecurityPolicy: customCSP,
	}

	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if got := w.Header().Get("Content-Security-Policy"); got != customCSP {
		t.Errorf("Content-Security-Policy = %q, want %q", got, customCSP)
	}
}

func TestParseCSP(t *testing.T) {
	tests := []struct {
		name     string
		csp      string
		expected map[string]string
	}{
		{
			name: "basic CSP",
			csp:  "default-src 'self'; script-src 'self' 'unsafe-inline'",
			expected: map[string]string{
				"default-src": "'self'",
				"script-src":  "'self' 'unsafe-inline'",
			},
		},
		{
			name: "CSP with directive without value",
			csp:  "default-src 'self'; upgrade-insecure-requests",
			expected: map[string]string{
				"default-src":               "'self'",
				"upgrade-insecure-requests": "",
			},
		},
		{
			name:     "empty CSP",
			csp:      "",
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseCSP(tt.csp)
			for directive, value := range tt.expected {
				if result[directive] != value {
					t.Errorf("parseCSP()[%q] = %q, want %q", directive, result[directive], value)
				}
			}
		})
	}
}

func TestGetSecurityHeadersFromContext(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig()
	router := gin.New()
	router.Use(SecurityHeaders(cfg))

	var capturedInfo SecurityHeadersInfo
	router.GET("/test", func(c *gin.Context) {
		capturedInfo = GetSecurityHeadersFromContext(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if capturedInfo.XFrameOptions != "DENY" {
		t.Errorf("XFrameOptions = %q, want %q", capturedInfo.XFrameOptions, "DENY")
	}

	if capturedInfo.XContentTypeOptions != "nosniff" {
		t.Errorf("XContentTypeOptions = %q, want %q", capturedInfo.XContentTypeOptions, "nosniff")
	}

	if capturedInfo.ContentSecurityPolicy == "" {
		t.Error("ContentSecurityPolicy should be set")
	}
}

func TestDefaultPermissionsPolicy(t *testing.T) {
	policy := defaultPermissionsPolicy()

	// Check that dangerous features are disabled
	disabledFeatures := []string{
		"camera=()",
		"microphone=()",
		"geolocation=()",
		"payment=()",
	}

	for _, feature := range disabledFeatures {
		if !strings.Contains(policy, feature) {
			t.Errorf("Permissions policy should contain %q", feature)
		}
	}
}

func TestSecurityHeaders_ResponseCode(t *testing.T) {
	cfg := DefaultSecurityHeadersConfig()
	router := gin.New()
	router.Use(SecurityHeaders(cfg))
	router.GET("/ok", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
	})

	tests := []struct {
		path       string
		wantStatus int
	}{
		{"/ok", http.StatusOK},
		{"/error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.path, nil)
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			// Headers should be set regardless of response status
			if w.Header().Get("X-Frame-Options") == "" {
				t.Error("Security headers should be set even on error responses")
			}
		})
	}
}
