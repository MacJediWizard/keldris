package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewRateLimiter(t *testing.T) {
	t.Run("valid configuration", func(t *testing.T) {
		mw, err := NewRateLimiter(10, "1m")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if mw == nil {
			t.Fatal("expected non-nil middleware")
		}
	})

	t.Run("invalid period", func(t *testing.T) {
		_, err := NewRateLimiter(10, "invalid")
		if err == nil {
			t.Fatal("expected error for invalid period")
		}
	})

	t.Run("requests within limit succeed", func(t *testing.T) {
		mw, err := NewRateLimiter(5, "1m")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		r := gin.New()
		r.Use(mw)
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// Make requests within the limit
		for i := 0; i < 5; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "127.0.0.1:12345"
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("request %d: expected status 200, got %d", i+1, w.Code)
			}
		}
	})

	t.Run("requests exceeding limit rejected", func(t *testing.T) {
		mw, err := NewRateLimiter(2, "1m")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		r := gin.New()
		r.Use(mw)
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// Exhaust the limit
		for i := 0; i < 2; i++ {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			r.ServeHTTP(w, req)
		}

		// Next request should be rate limited
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "10.0.0.1:12345"
		r.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("expected status 429, got %d", w.Code)
		}
	})

	t.Run("different IPs have separate limits", func(t *testing.T) {
		mw, err := NewRateLimiter(1, "1m")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		r := gin.New()
		r.Use(mw)
		r.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"ok": true})
		})

		// First IP uses its limit
		w1 := httptest.NewRecorder()
		req1, _ := http.NewRequest("GET", "/test", nil)
		req1.RemoteAddr = "192.168.1.1:12345"
		r.ServeHTTP(w1, req1)
		if w1.Code != http.StatusOK {
			t.Fatalf("first IP: expected status 200, got %d", w1.Code)
		}

		// Second IP should still work
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/test", nil)
		req2.RemoteAddr = "192.168.1.2:12345"
		r.ServeHTTP(w2, req2)
		if w2.Code != http.StatusOK {
			t.Fatalf("second IP: expected status 200, got %d", w2.Code)
		}
	})
}
