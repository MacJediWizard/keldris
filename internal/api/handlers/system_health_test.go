package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockSystemHealthStore struct {
	pingErr     error
	dbSize      int64
	dbSizeErr   error
	activeConns int
	connsErr    error
	pending     int
	pendingErr  error
	running     int
	runningErr  error
	recentErrs  []*db.ServerError
	errsErr     error
	history     []*db.HealthHistoryRecord
	historyErr  error
}

func (m *mockSystemHealthStore) Ping(_ context.Context) error { return m.pingErr }
func (m *mockSystemHealthStore) Health() map[string]any {
	return map[string]any{"max_connections": int32(50), "total_connections": int32(5)}
}
func (m *mockSystemHealthStore) GetDatabaseSize(_ context.Context) (int64, error) {
	return m.dbSize, m.dbSizeErr
}
func (m *mockSystemHealthStore) GetActiveConnections(_ context.Context) (int, error) {
	return m.activeConns, m.connsErr
}
func (m *mockSystemHealthStore) GetPendingBackupsCount(_ context.Context, _ uuid.UUID) (int, error) {
	return m.pending, m.pendingErr
}
func (m *mockSystemHealthStore) GetRunningBackupsCount(_ context.Context, _ uuid.UUID) (int, error) {
	return m.running, m.runningErr
}
func (m *mockSystemHealthStore) GetRecentServerErrors(_ context.Context, _ int) ([]*db.ServerError, error) {
	return m.recentErrs, m.errsErr
}
func (m *mockSystemHealthStore) GetHealthHistoryRecords(_ context.Context, _ time.Time) ([]*db.HealthHistoryRecord, error) {
	return m.history, m.historyErr
}
func (m *mockSystemHealthStore) SaveHealthHistoryRecord(_ context.Context, _ *db.HealthHistoryRecord) error {
	return nil
}

func setupSystemHealthTestRouter(store SystemHealthStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewSystemHealthHandler(store, nil, zerolog.Nop())
	// Bypass SuperuserMiddleware (requires real SessionStore); RequireSuperuser inside the handler still enforces the check.
	r.GET("/api/v1/admin/health", handler.GetSystemHealth)
	r.GET("/api/v1/admin/health/history", handler.GetHealthHistory)
	return r
}

func superuserTestUser() *auth.SessionUser {
	u := testUser(uuid.New())
	u.IsSuperuser = true
	return u
}

func TestSystemHealthGetSystemHealth(t *testing.T) {
	t.Run("superuser sees healthy status", func(t *testing.T) {
		store := &mockSystemHealthStore{dbSize: 1024 * 1024 * 50}
		r := setupSystemHealthTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/health"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-superuser forbidden", func(t *testing.T) {
		store := &mockSystemHealthStore{}
		r := setupSystemHealthTestRouter(store, testUser(uuid.New()))

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/health"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestSystemHealthGetHealthHistory(t *testing.T) {
	t.Run("returns history records", func(t *testing.T) {
		store := &mockSystemHealthStore{history: []*db.HealthHistoryRecord{{ID: uuid.New().String(), Status: "healthy"}}}
		r := setupSystemHealthTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/health/history"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSystemHealthStore{historyErr: context.DeadlineExceeded}
		r := setupSystemHealthTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/health/history"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
