package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// mockSetupStore implements SetupStore for testing.
type mockSetupStore struct {
	complete bool
	err      error
}

func (m *mockSetupStore) IsSetupComplete(_ context.Context) (bool, error) {
	return m.complete, m.err
}

func init() {
	gin.SetMode(gin.TestMode)
}

// newSetupRouter creates a gin router with the given middleware and a catch-all 200 handler.
func newSetupRouter(mw gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(mw)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func doRequest(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// SetupRequiredMiddleware tests
// ---------------------------------------------------------------------------

func TestSetupRequired_BlocksAPI(t *testing.T) {
	store := &mockSetupStore{complete: false}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	paths := []string{
		"/api/v1/agents",
		"/api/v1/backups",
		"/api/v1/settings",
		"/api/v1/onboarding/status",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("path %s: expected 503, got %d", path, w.Code)
		}
	}
}

func TestSetupRequired_AllowsSetup(t *testing.T) {
	store := &mockSetupStore{complete: false}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	paths := []string{
		"/api/v1/setup",
		"/api/v1/setup/status",
		"/api/v1/setup/database",
		"/api/v1/setup/superuser",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSetupRequired_AllowsHealth(t *testing.T) {
	store := &mockSetupStore{complete: false}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	paths := []string{
		"/health",
		"/api/v1/health",
		"/api/v1/branding",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSetupRequired_AllowsAuth(t *testing.T) {
	store := &mockSetupStore{complete: false}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	paths := []string{
		"/auth/login/password",
		"/auth/me",
		"/auth/logout",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSetupRequired_AllowsStatic(t *testing.T) {
	store := &mockSetupStore{complete: false}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	paths := []string{
		"/assets/logo.png",
		"/assets/fonts/inter.woff2",
		"/main.js",
		"/style.css",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSetupRequired_AllowsFrontendRoutes(t *testing.T) {
	store := &mockSetupStore{complete: false}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	paths := []string{
		"/",
		"/setup",
		"/login",
		"/dashboard",
		"/agents",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSetupRequired_PassesAfterComplete(t *testing.T) {
	store := &mockSetupStore{complete: true}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	paths := []string{
		"/api/v1/agents",
		"/api/v1/backups",
		"/api/v1/settings",
		"/dashboard",
		"/api/v1/setup/status",
		"/health",
		"/auth/me",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// SetupLockMiddleware tests
// ---------------------------------------------------------------------------

func TestSetupLock_BlocksAfterComplete(t *testing.T) {
	store := &mockSetupStore{complete: true}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupLockMiddleware(store, logger))

	paths := []string{
		"/api/v1/setup",
		"/api/v1/setup/database",
		"/api/v1/setup/superuser",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusForbidden {
			t.Errorf("path %s: expected 403, got %d", path, w.Code)
		}
	}
}

func TestSetupLock_AllowsStatus(t *testing.T) {
	store := &mockSetupStore{complete: true}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupLockMiddleware(store, logger))

	w := doRequest(r, "GET", "/api/v1/setup/status")
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSetupLock_AllowsWhenIncomplete(t *testing.T) {
	store := &mockSetupStore{complete: false}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupLockMiddleware(store, logger))

	paths := []string{
		"/api/v1/setup",
		"/api/v1/setup/database",
		"/api/v1/setup/superuser",
	}

	for _, path := range paths {
		w := doRequest(r, "GET", path)
		if w.Code != http.StatusOK {
			t.Errorf("path %s: expected 200, got %d", path, w.Code)
		}
	}
}

func TestSetupLock_AllowsRerunAfterComplete(t *testing.T) {
	store := &mockSetupStore{complete: true}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupLockMiddleware(store, logger))

	w := doRequest(r, "POST", "/api/v1/setup/rerun")
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestSetupRequired_StoreError(t *testing.T) {
	store := &mockSetupStore{complete: false, err: context.DeadlineExceeded}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupRequiredMiddleware(store, logger))

	// Non-exempt path triggers the store call, which returns an error.
	w := doRequest(r, "GET", "/api/v1/agents")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestSetupLock_StoreError(t *testing.T) {
	store := &mockSetupStore{complete: false, err: context.DeadlineExceeded}
	logger := zerolog.Nop()
	r := newSetupRouter(SetupLockMiddleware(store, logger))

	// Non-status path triggers the store call, which returns an error.
	w := doRequest(r, "GET", "/api/v1/setup/database")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
