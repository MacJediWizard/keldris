package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockDatabaseBackupStore struct {
	backup  *models.DatabaseBackup
	backups []*models.DatabaseBackup
	summary *models.DatabaseBackupSummary
	err     error
}

func (m *mockDatabaseBackupStore) GetDatabaseBackupByID(_ context.Context, _ uuid.UUID) (*models.DatabaseBackup, error) {
	return m.backup, m.err
}

func (m *mockDatabaseBackupStore) ListDatabaseBackups(_ context.Context, _, _ int) ([]*models.DatabaseBackup, int, error) {
	return m.backups, len(m.backups), m.err
}

func (m *mockDatabaseBackupStore) GetLatestDatabaseBackup(_ context.Context) (*models.DatabaseBackup, error) {
	return m.backup, m.err
}

func (m *mockDatabaseBackupStore) GetDatabaseBackupSummary(_ context.Context) (*models.DatabaseBackupSummary, error) {
	return m.summary, m.err
}

func (m *mockDatabaseBackupStore) MarkDatabaseBackupVerified(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func setupDatabaseBackupTestRouter(store DatabaseBackupStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDatabaseBackupHandler(store, nil, nil, zerolog.Nop())
	// Bypass SuperuserMiddleware (needs real SessionStore); handler's RequireSuperuser still enforces.
	r.GET("/api/v1/superuser/database-backups", handler.ListBackups)
	r.GET("/api/v1/superuser/database-backups/status", handler.GetStatus)
	r.GET("/api/v1/superuser/database-backups/summary", handler.GetSummary)
	r.GET("/api/v1/superuser/database-backups/:id", handler.GetBackup)
	r.GET("/api/v1/superuser/database-backups/:id/restore-instructions", handler.GetRestoreInstructions)
	return r
}

func TestDatabaseBackupListBackups(t *testing.T) {
	t.Run("superuser sees backups", func(t *testing.T) {
		store := &mockDatabaseBackupStore{backups: []*models.DatabaseBackup{}}
		r := setupDatabaseBackupTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/database-backups"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-superuser forbidden", func(t *testing.T) {
		store := &mockDatabaseBackupStore{}
		r := setupDatabaseBackupTestRouter(store, testUser(uuid.New()))

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/database-backups"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestDatabaseBackupGetSummary(t *testing.T) {
	store := &mockDatabaseBackupStore{summary: &models.DatabaseBackupSummary{}}
	r := setupDatabaseBackupTestRouter(store, superuserTestUser())

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/database-backups/summary"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestDatabaseBackupGetBackup(t *testing.T) {
	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockDatabaseBackupStore{}
		r := setupDatabaseBackupTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/database-backups/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
