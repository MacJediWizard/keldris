package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"context"

	"github.com/MacJediWizard/keldris/internal/auth"
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

func setupHealthTestRouter(db DatabaseHealthChecker, oidcProvider *auth.OIDCProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewHealthHandler(db, oidcProvider, zerolog.Nop())
	handler.RegisterPublicRoutes(r)
	return r
}

// nilOIDC returns an OIDCProvider wrapper with no inner provider (not configured).
func nilOIDC() *auth.OIDCProvider {
	return auth.NewOIDCProvider(nil, zerolog.Nop())
}

func TestHealthOverall(t *testing.T) {
	t.Run("all healthy", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{health: map[string]any{"total_conns": int32(10)}}
		r := setupHealthTestRouter(db, nilOIDC())

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
		r := setupHealthTestRouter(db, nilOIDC())

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("nil db", func(t *testing.T) {
		r := setupHealthTestRouter(nil, nilOIDC())

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status 503, got %d", w.Code)
		}
	})

	t.Run("nil oidc provider is ok", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{health: map[string]any{}}
		r := setupHealthTestRouter(db, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("unconfigured oidc provider is ok", func(t *testing.T) {
		db := &mockDatabaseHealthChecker{health: map[string]any{}}
		r := setupHealthTestRouter(db, nilOIDC())

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		oidcCheck := resp.Checks["oidc"]
		if oidcCheck == nil {
			t.Fatal("expected oidc check in response")
		}
		if oidcCheck.Details["configured"] != false {
			t.Fatalf("expected configured=false, got %v", oidcCheck.Details["configured"])
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
	t.Run("not configured", func(t *testing.T) {
		r := setupHealthTestRouter(nil, nilOIDC())

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health/oidc", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp HealthResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if resp.Status != HealthStatusHealthy {
			t.Fatalf("expected healthy, got %q", resp.Status)
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
