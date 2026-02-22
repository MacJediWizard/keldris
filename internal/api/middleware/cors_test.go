package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/gin-gonic/gin"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	mw := CORS([]string{"https://app.keldris.io", "https://admin.keldris.io"}, config.EnvDevelopment)
	mw := CORS([]string{"https://app.keldris.io", "https://admin.keldris.io"})

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.keldris.io")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.keldris.io" {
		t.Fatalf("expected Access-Control-Allow-Origin 'https://app.keldris.io', got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials 'true', got %q", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	mw := CORS([]string{"https://app.keldris.io"}, config.EnvDevelopment)
	mw := CORS([]string{"https://app.keldris.io"})

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w, req)

	// Request should still succeed (CORS doesn't block server-side)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	// But no CORS headers should be set
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no Access-Control-Allow-Origin header, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	mw := CORS([]string{"https://app.keldris.io"}, config.EnvDevelopment)
	mw := CORS([]string{"https://app.keldris.io"})

	r := gin.New()
	r.Use(mw)
	r.OPTIONS("/test", func(c *gin.Context) {
		// This handler should not be reached; middleware aborts first
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://app.keldris.io")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, Authorization")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 for preflight, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.keldris.io" {
		t.Fatalf("expected Access-Control-Allow-Origin 'https://app.keldris.io', got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatal("expected Access-Control-Allow-Methods header to be set")
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatal("expected Access-Control-Allow-Headers header to be set")
	}
	if got := w.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Fatalf("expected Access-Control-Max-Age '86400', got %q", got)
	}
}

func TestCORS_Credentials(t *testing.T) {
	mw := CORS([]string{"https://app.keldris.io"}, config.EnvDevelopment)
	mw := CORS([]string{"https://app.keldris.io"})

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.keldris.io")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected Access-Control-Allow-Credentials 'true', got %q", got)
	}
}

func TestCORS_AllowAllOrigins(t *testing.T) {
	// Empty allowed origins = allow all (dev mode)
	mw := CORS([]string{}, config.EnvDevelopment)
	mw := CORS([]string{})

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("expected Access-Control-Allow-Origin 'http://localhost:5173', got %q", got)
	}
}

func TestCORS_CaseInsensitive(t *testing.T) {
	mw := CORS([]string{"https://app.keldris.io"}, config.EnvDevelopment)
	mw := CORS([]string{"https://app.keldris.io"})

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "HTTPS://APP.KELDRIS.IO")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "HTTPS://APP.KELDRIS.IO" {
		t.Fatalf("expected case-insensitive match, got %q", got)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	mw := CORS([]string{"https://app.keldris.io"}, config.EnvDevelopment)
	mw := CORS([]string{"https://app.keldris.io"})

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// No Origin header (same-origin or non-browser request)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	// No CORS headers should be set when there's no Origin
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS headers without Origin, got %q", got)
	}
}

func TestCORS_PreflightDisallowedOrigin(t *testing.T) {
	mw := CORS([]string{"https://app.keldris.io"}, config.EnvDevelopment)
	mw := CORS([]string{"https://app.keldris.io"})

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	r.ServeHTTP(w, req)

	// Preflight still returns 204 but without CORS headers for the disallowed origin
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS origin for disallowed request, got %q", got)
	}
}

func TestCORS_ProductionPanicsWithoutOrigins(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when CORS_ORIGINS is empty in production")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if msg != "CORS_ORIGINS must be set in production; refusing to start with open CORS policy" {
			t.Fatalf("unexpected panic message: %s", msg)
		}
	}()

	CORS([]string{}, config.EnvProduction)
}

func TestCORS_ProductionWithOrigins(t *testing.T) {
	// Should not panic when origins are provided in production
	mw := CORS([]string{"https://app.keldris.io"}, config.EnvProduction)

	r := gin.New()
	r.Use(mw)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://app.keldris.io")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.keldris.io" {
		t.Fatalf("expected Access-Control-Allow-Origin 'https://app.keldris.io', got %q", got)
	}
}
