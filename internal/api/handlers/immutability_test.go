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

type mockImmutabilityStore struct {
	user     *models.User
	repo     *models.Repository
	repos    []*models.Repository
	lock     *models.SnapshotImmutability
	locks    []*models.SnapshotImmutability
	settings *models.RepositoryImmutabilitySettings
	backup   *models.Backup
	locked   bool
	err      error
}

func (m *mockImmutabilityStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

func (m *mockImmutabilityStore) GetRepository(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	return m.repo, m.err
}

func (m *mockImmutabilityStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Repository, error) {
	return m.repos, m.err
}

func (m *mockImmutabilityStore) CreateSnapshotImmutability(_ context.Context, _ *models.SnapshotImmutability) error {
	return m.err
}

func (m *mockImmutabilityStore) GetSnapshotImmutability(_ context.Context, _ uuid.UUID, _ string) (*models.SnapshotImmutability, error) {
	return m.lock, m.err
}

func (m *mockImmutabilityStore) GetSnapshotImmutabilityByID(_ context.Context, _ uuid.UUID) (*models.SnapshotImmutability, error) {
	return m.lock, m.err
}

func (m *mockImmutabilityStore) UpdateSnapshotImmutability(_ context.Context, _ *models.SnapshotImmutability) error {
	return m.err
}

func (m *mockImmutabilityStore) DeleteExpiredImmutabilityLocks(_ context.Context) (int, error) {
	return 0, m.err
}

func (m *mockImmutabilityStore) GetActiveImmutabilityLocksByRepositoryID(_ context.Context, _ uuid.UUID) ([]*models.SnapshotImmutability, error) {
	return m.locks, m.err
}

func (m *mockImmutabilityStore) GetActiveImmutabilityLocksByOrgID(_ context.Context, _ uuid.UUID) ([]*models.SnapshotImmutability, error) {
	return m.locks, m.err
}

func (m *mockImmutabilityStore) IsSnapshotLocked(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return m.locked, m.err
}

func (m *mockImmutabilityStore) GetRepositoryImmutabilitySettings(_ context.Context, _ uuid.UUID) (*models.RepositoryImmutabilitySettings, error) {
	return m.settings, m.err
}

func (m *mockImmutabilityStore) UpdateRepositoryImmutabilitySettings(_ context.Context, _ uuid.UUID, _ *models.RepositoryImmutabilitySettings) error {
	return m.err
}

func (m *mockImmutabilityStore) GetBackupBySnapshotID(_ context.Context, _ string) (*models.Backup, error) {
	return m.backup, m.err
}

func setupImmutabilityTestRouter(store ImmutabilityStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewImmutabilityHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestImmutabilityListLocks(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockImmutabilityStore{
		user:  &models.User{ID: user.ID, Role: models.UserRoleAdmin},
		locks: []*models.SnapshotImmutability{},
	}
	r := setupImmutabilityTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/immutability"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestImmutabilityGetLock(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockImmutabilityStore{}
		r := setupImmutabilityTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/immutability/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
