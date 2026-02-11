package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

func setupMetricsTestRouter(db MetricsStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewMetricsHandler(db, zerolog.Nop())
	handler.RegisterPublicRoutes(r)
	return r
}

func TestMetrics(t *testing.T) {
	t.Run("with healthy db", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{
			health: map[string]any{
				"total_conns":      int32(10),
				"acquired_conns":   int32(2),
				"idle_conns":       int32(8),
				"max_conns":        int32(20),
				"constructing":     int32(0),
				"empty_acquire":    int64(0),
				"canceled_acquire": int64(0),
				"max_lifetime_dest": int64(0),
				"max_idle_dest":    int64(0),
			},
		}
		r := setupMetricsTestRouter(db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, "keldris_info") {
			t.Fatal("expected keldris_info metric")
		}
		if !strings.Contains(body, "keldris_up") {
			t.Fatal("expected keldris_up metric")
		}
		if !strings.Contains(body, "keldris_db_connections_total") {
			t.Fatal("expected keldris_db_connections_total metric")
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/plain") {
			t.Fatalf("expected text/plain content type, got %q", contentType)
		}
	})

	t.Run("with unhealthy db", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{
			pingErr: errors.New("db down"),
			health:  map[string]any{},
		}
		r := setupMetricsTestRouter(db)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, `keldris_up{component="database"} 0`) {
			t.Fatal("expected database unhealthy metric")
		}
	})

	t.Run("nil db", func(t *testing.T) {
		r := setupMetricsTestRouter(nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/metrics", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if !strings.Contains(body, `keldris_up{component="database"} 0`) {
			t.Fatal("expected database unhealthy metric when nil")
		}
	})
}
