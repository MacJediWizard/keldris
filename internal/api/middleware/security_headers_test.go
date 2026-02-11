package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders_AllHeadersSet(t *testing.T) {
	mw := SecurityHeaders()

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	expected := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"X-XSS-Protection":      "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Permissions-Policy":     "geolocation=(), microphone=(), camera=()",
		"Content-Security-Policy": cspFrontend,
	}

	for header, want := range expected {
		if got := w.Header().Get(header); got != want {
			t.Errorf("expected %s %q, got %q", header, want, got)
		}
	}

	// HSTS should NOT be set without TLS
	if got := w.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("expected no Strict-Transport-Security without TLS, got %q", got)
	}
}

func TestSecurityHeaders_HSTSOnlyWithTLS(t *testing.T) {
	mw := SecurityHeaders()

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if got := w.Header().Get("Strict-Transport-Security"); got != "max-age=31536000; includeSubDomains" {
		t.Errorf("expected HSTS header with TLS, got %q", got)
	}
}

func TestSecurityHeaders_APIRouteStrictCSP(t *testing.T) {
	mw := SecurityHeaders()

	r := gin.New()
	r.Use(mw)
	r.GET("/api/v1/agents", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"agents": []string{}})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/agents", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if got := w.Header().Get("Content-Security-Policy"); got != cspAPI {
		t.Errorf("expected strict API CSP %q, got %q", cspAPI, got)
	}
}

func TestSecurityHeaders_AuthRouteStrictCSP(t *testing.T) {
	mw := SecurityHeaders()

	r := gin.New()
	r.Use(mw)
	r.GET("/auth/callback", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/auth/callback", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	if got := w.Header().Get("Content-Security-Policy"); got != cspAPI {
		t.Errorf("expected strict API CSP %q, got %q", cspAPI, got)
	}
}

func TestSecurityHeaders_FrontendRouteRelaxedCSP(t *testing.T) {
	mw := SecurityHeaders()

	r := gin.New()
	r.Use(mw)
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "<html></html>")
	})
	r.GET("/api/docs/index.html", func(c *gin.Context) {
		c.String(http.StatusOK, "<html></html>")
	})

	tests := []struct {
		name string
		path string
	}{
		{"root", "/"},
		{"swagger docs", "/api/docs/index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", tt.path, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}

			if got := w.Header().Get("Content-Security-Policy"); got != cspFrontend {
				t.Errorf("expected frontend CSP %q, got %q", cspFrontend, got)
			}
		})
	}
}

func TestIsAPIRoute(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/agents", true},
		{"/api/v1/agents/123", true},
		{"/auth/login", true},
		{"/auth/callback", true},
		{"/", false},
		{"/api/docs/index.html", false},
		{"/health", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isAPIRoute(tt.path); got != tt.want {
				t.Errorf("isAPIRoute(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
