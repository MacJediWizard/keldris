package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

type mockDatabaseHealthChecker struct {
	pingErr error
	health  map[string]any
}

func (m *mockDatabaseHealthChecker) Ping(_ context.Context) error {
	return m.pingErr
}

func (m *mockDatabaseHealthChecker) Health() map[string]any {
	if m.health != nil {
		return m.health
	}
	return map[string]any{}
}

type mockOIDCHealthChecker struct {
	healthErr error
}

func (m *mockOIDCHealthChecker) HealthCheck(_ context.Context) error {
	return m.healthErr
}

func setupHealthTestRouter(db DatabaseHealthChecker, oidc OIDCHealthChecker) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewHealthHandler(db, oidc, zerolog.Nop())
	handler.RegisterPublicRoutes(r)
	return r
}

func TestHealthOverall(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{health: map[string]any{"total_conns": int32(10)}}
		oidc := &mockOIDCHealthChecker{}
		r := setupHealthTestRouter(db, oidc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.Status != HealthStatusHealthy {
			t.Fatalf("expected healthy status, got %q", resp.Status)
		}
	})

	t.Run("database unhealthy", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{pingErr: errors.New("connection refused")}
		oidc := &mockOIDCHealthChecker{}
		r := setupHealthTestRouter(db, oidc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("oidc unhealthy", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{health: map[string]any{}}
		oidc := &mockOIDCHealthChecker{healthErr: errors.New("provider unreachable")}
		r := setupHealthTestRouter(db, oidc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("nil db", func(t *testing.T) {
		oidc := &mockOIDCHealthChecker{}
		r := setupHealthTestRouter(nil, oidc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("nil oidc is ok", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{health: map[string]any{}}
		r := setupHealthTestRouter(db, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})
}

func TestHealthDatabase(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{health: map[string]any{"total_conns": int32(5)}}
		r := setupHealthTestRouter(db, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/db", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{pingErr: errors.New("db down")}
		r := setupHealthTestRouter(db, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/db", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("nil db", func(t *testing.T) {
		r := setupHealthTestRouter(nil, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/db", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})
}

func TestHealthOIDC(t *testing.T) {
	t.Run("healthy", func(t *testing.T) {
		oidc := &mockOIDCHealthChecker{}
		r := setupHealthTestRouter(nil, oidc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/oidc", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("unhealthy", func(t *testing.T) {
		oidc := &mockOIDCHealthChecker{healthErr: errors.New("oidc down")}
		r := setupHealthTestRouter(nil, oidc)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/oidc", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("nil oidc", func(t *testing.T) {
		r := setupHealthTestRouter(nil, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/oidc", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})
}
