package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/config"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders_AllHeadersSet(t *testing.T) {
	mw := SecurityHeaders(config.EnvDevelopment)
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

func TestSecurityHeaders_HSTS(t *testing.T) {
	tests := []struct {
		name     string
		env      config.Environment
		tls      bool
		proto    string // X-Forwarded-Proto header value
		wantHSTS bool
	}{
		{"TLS direct connection", config.EnvDevelopment, true, "", true},
		{"X-Forwarded-Proto https", config.EnvDevelopment, false, "https", true},
		{"production without TLS", config.EnvProduction, false, "", true},
		{"staging without TLS", config.EnvStaging, false, "", true},
		{"development without TLS", config.EnvDevelopment, false, "", false},
		{"X-Forwarded-Proto http in dev", config.EnvDevelopment, false, "http", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw := SecurityHeaders(tt.env)

			r := gin.New()
			r.Use(mw)
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"ok": true})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			if tt.tls {
				req.TLS = &tls.ConnectionState{}
			}
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}

			got := w.Header().Get("Strict-Transport-Security")
			if tt.wantHSTS && got != "max-age=31536000; includeSubDomains" {
				t.Errorf("expected HSTS header, got %q", got)
			}
			if !tt.wantHSTS && got != "" {
				t.Errorf("expected no HSTS header, got %q", got)
			}
		})
	}
}

func TestSecurityHeaders_APIRouteStrictCSP(t *testing.T) {
	mw := SecurityHeaders(config.EnvDevelopment)

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
	mw := SecurityHeaders(config.EnvDevelopment)

	r := gin.New()
	r.Use(mw)
	r.GET("/auth/callback", func(c *gin.Context) {
func TestSecurityHeaders_HSTSOnlyWithTLS(t *testing.T) {
	mw := SecurityHeaders()

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
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

func TestSecurityHeaders_SwaggerRouteCSP_Development(t *testing.T) {
	mw := SecurityHeaders(config.EnvDevelopment)

	r := gin.New()
	r.Use(mw)
	r.GET("/api/docs/index.html", func(c *gin.Context) {
		c.String(http.StatusOK, "<html></html>")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/docs/index.html", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	got := w.Header().Get("Content-Security-Policy")
	if got != cspSwaggerDev {
		t.Errorf("expected swagger dev CSP %q, got %q", cspSwaggerDev, got)
	}

	// Dev-only swagger CSP retains unsafe-inline for third-party Swagger UI content.
	if !strings.Contains(got, "'unsafe-inline'") {
		t.Error("swagger dev CSP should contain 'unsafe-inline'")
	}
}

func TestSecurityHeaders_SwaggerRouteCSP_Production(t *testing.T) {
	mw := SecurityHeaders(config.EnvProduction)

	r := gin.New()
	r.Use(mw)

	var capturedNonce string
	r.GET("/api/docs/index.html", func(c *gin.Context) {
		capturedNonce = GetCSPNonce(c)
		c.String(http.StatusOK, "<html></html>")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/docs/index.html", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	got := w.Header().Get("Content-Security-Policy")

	// In production, swagger routes must NOT get unsafe-inline CSP.
	if strings.Contains(got, "'unsafe-inline'") {
		t.Error("production swagger route should not contain 'unsafe-inline'")
	}

	// Should get nonce-based CSP instead (falls through to default).
	if capturedNonce == "" {
		t.Fatal("expected nonce to be set for swagger route in production")
	}
	nonceDirective := "'nonce-" + capturedNonce + "'"
	if !strings.Contains(got, "script-src 'self' "+nonceDirective) {
		t.Errorf("production swagger route missing nonce in script-src: %s", got)
	}
}

func TestSecurityHeaders_FrontendRouteNonceCSP(t *testing.T) {
	mw := SecurityHeaders(config.EnvDevelopment)

	r := gin.New()
	r.Use(mw)

	var capturedNonce string
	r.GET("/", func(c *gin.Context) {
		capturedNonce = GetCSPNonce(c)
		c.String(http.StatusOK, "<html></html>")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req, _ := http.NewRequest("GET", "/test", nil)
	req.TLS = &tls.ConnectionState{}
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	csp := w.Header().Get("Content-Security-Policy")

	// Must NOT contain unsafe-inline.
	if strings.Contains(csp, "'unsafe-inline'") {
		t.Error("frontend CSP should not contain 'unsafe-inline'")
	}

	// Must contain a nonce directive.
	if capturedNonce == "" {
		t.Fatal("expected nonce to be set in context")
	}
	nonceDirective := "'nonce-" + capturedNonce + "'"
	if !strings.Contains(csp, "script-src 'self' "+nonceDirective) {
		t.Errorf("CSP missing nonce in script-src: %s", csp)
	}
	if !strings.Contains(csp, "style-src 'self' "+nonceDirective) {
		t.Errorf("CSP missing nonce in style-src: %s", csp)
	}

	// Standard directives still present.
	if !strings.Contains(csp, "default-src 'self'") {
		t.Errorf("CSP missing default-src: %s", csp)
	}
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Errorf("CSP missing frame-ancestors: %s", csp)
	}
}

func TestSecurityHeaders_NonceUniqueness(t *testing.T) {
	mw := SecurityHeaders(config.EnvDevelopment)

	r := gin.New()
	r.Use(mw)

	var nonces []string
	r.GET("/", func(c *gin.Context) {
		nonces = append(nonces, GetCSPNonce(c))
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		r.ServeHTTP(w, req)
	}

	if len(nonces) != 10 {
		t.Fatalf("expected 10 nonces, got %d", len(nonces))
	}

	seen := make(map[string]bool)
	for _, n := range nonces {
		if n == "" {
			t.Fatal("empty nonce")
		}
		if seen[n] {
			t.Fatalf("duplicate nonce detected: %s", n)
		}
		seen[n] = true
	}
}

func TestSecurityHeaders_NonceNotSetOnAPIRoutes(t *testing.T) {
	mw := SecurityHeaders(config.EnvDevelopment)

	r := gin.New()
	r.Use(mw)

	var capturedNonce string
	r.GET("/api/v1/test", func(c *gin.Context) {
		capturedNonce = GetCSPNonce(c)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	r.ServeHTTP(w, req)

	if capturedNonce != "" {
		t.Errorf("expected no nonce on API routes, got %q", capturedNonce)
	}
}

func TestGetCSPNonce_EmptyWithoutMiddleware(t *testing.T) {
	r := gin.New()

	var capturedNonce string
	r.GET("/", func(c *gin.Context) {
		capturedNonce = GetCSPNonce(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	r.ServeHTTP(w, req)

	if capturedNonce != "" {
		t.Errorf("expected empty nonce without middleware, got %q", capturedNonce)
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

func TestIsSwaggerRoute(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/api/docs/index.html", true},
		{"/api/docs/doc.json", true},
		{"/api/docs/", true},
		{"/api/v1/agents", false},
		{"/", false},
		{"/auth/login", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isSwaggerRoute(tt.path); got != tt.want {
				t.Errorf("isSwaggerRoute(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := generateNonce()
	if err != nil {
		t.Fatalf("generateNonce() error: %v", err)
	}
	if nonce == "" {
		t.Fatal("expected non-empty nonce")
	}
	// 16 bytes base64-encoded = 24 chars (with padding) or 22 (without).
	if len(nonce) < 20 {
		t.Errorf("nonce seems too short: %q (len=%d)", nonce, len(nonce))
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
