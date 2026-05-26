package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockRepositoryImportStore struct {
	repo      *models.Repository
	repos     []*models.Repository
	agent     *models.Agent
	agents    []*models.Agent
	snapshots []*models.ImportedSnapshot
	err       error
}

func (m *mockRepositoryImportStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Repository, error) {
	return m.repos, m.err
}

func (m *mockRepositoryImportStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	return m.repo, m.err
}

func (m *mockRepositoryImportStore) CreateRepository(_ context.Context, _ *models.Repository) error {
	return m.err
}

func (m *mockRepositoryImportStore) CreateRepositoryKey(_ context.Context, _ *models.RepositoryKey) error {
	return m.err
}

func (m *mockRepositoryImportStore) CreateImportedSnapshot(_ context.Context, _ *models.ImportedSnapshot) error {
	return m.err
}

func (m *mockRepositoryImportStore) CreateImportedSnapshots(_ context.Context, _ []*models.ImportedSnapshot) error {
	return m.err
}

func (m *mockRepositoryImportStore) MarkRepositoryAsImported(_ context.Context, _ uuid.UUID, _ int) error {
	return m.err
}

func (m *mockRepositoryImportStore) GetImportedSnapshotsByRepositoryID(_ context.Context, _ uuid.UUID) ([]*models.ImportedSnapshot, error) {
	return m.snapshots, m.err
}

func (m *mockRepositoryImportStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func (m *mockRepositoryImportStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return m.agents, m.err
}

func setupRepositoryImportTestRouter(store RepositoryImportStore, user *auth.SessionUser, km *crypto.KeyManager) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewRepositoryImportHandler(store, km, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestRepositoryImportVerifyAccess(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)

	t.Run("missing body returns 400", func(t *testing.T) {
		store := &mockRepositoryImportStore{}
		r := setupRepositoryImportTestRouter(store, user, km)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/repositories/import/verify", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		store := &mockRepositoryImportStore{}
		r := setupRepositoryImportTestRouter(store, user, km)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/repositories/import/verify", `{invalid`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestRepositoryImportPreview(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)

	t.Run("missing body returns 400", func(t *testing.T) {
		store := &mockRepositoryImportStore{}
		r := setupRepositoryImportTestRouter(store, user, km)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/repositories/import/preview", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
