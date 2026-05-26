package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockDockerRestoreStore struct {
	agent           *models.Agent
	agentErr        error
	repo            *models.Repository
	repoErr         error
	backup          *models.Backup
	backupErr       error
	restore         *models.DockerRestore
	restoreErr      error
	restoresByOrg   []*models.DockerRestore
	restoresByAgent []*models.DockerRestore
	listErr         error
	createErr       error
	updateErr       error
	createCmdErr    error
}

func (m *mockDockerRestoreStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func (m *mockDockerRestoreStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	if m.repoErr != nil {
		return nil, m.repoErr
	}
	return m.repo, nil
}

func (m *mockDockerRestoreStore) GetBackupBySnapshotID(_ context.Context, _ string) (*models.Backup, error) {
	if m.backupErr != nil {
		return nil, m.backupErr
	}
	return m.backup, nil
}

func (m *mockDockerRestoreStore) CreateDockerRestore(_ context.Context, _ *models.DockerRestore) error {
	return m.createErr
}

func (m *mockDockerRestoreStore) UpdateDockerRestore(_ context.Context, _ *models.DockerRestore) error {
	return m.updateErr
}

func (m *mockDockerRestoreStore) GetDockerRestoreByID(_ context.Context, _ uuid.UUID) (*models.DockerRestore, error) {
	if m.restoreErr != nil {
		return nil, m.restoreErr
	}
	return m.restore, nil
}

func (m *mockDockerRestoreStore) GetDockerRestoresByOrgID(_ context.Context, _ uuid.UUID) ([]*models.DockerRestore, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.restoresByOrg, nil
}

func (m *mockDockerRestoreStore) GetDockerRestoresByAgentID(_ context.Context, _ uuid.UUID) ([]*models.DockerRestore, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.restoresByAgent, nil
}

func (m *mockDockerRestoreStore) CreateAgentCommand(_ context.Context, _ *models.AgentCommand) error {
	return m.createCmdErr
}

func setupDockerRestoreTestRouter(store DockerRestoreStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDockerRestoreHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDockerRestoreList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns restores", func(t *testing.T) {
		store := &mockDockerRestoreStore{restoresByOrg: []*models.DockerRestore{}}
		r := setupDockerRestoreTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-restores"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid agent_id returns 400", func(t *testing.T) {
		store := &mockDockerRestoreStore{}
		r := setupDockerRestoreTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-restores?agent_id=bogus"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockDockerRestoreStore{listErr: errors.New("db down")}
		r := setupDockerRestoreTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-restores"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestDockerRestoreCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	agentID := uuid.New()
	repoID := uuid.New()

	t.Run("creates successfully", func(t *testing.T) {
		store := &mockDockerRestoreStore{
			agent:  &models.Agent{ID: agentID, OrgID: orgID},
			repo:   &models.Repository{ID: repoID, OrgID: orgID},
			backup: &models.Backup{ID: uuid.New()},
		}
		r := setupDockerRestoreTestRouter(store, user)
		body := `{"snapshot_id":"snap1","agent_id":"` + agentID.String() + `","repository_id":"` + repoID.String() + `","container_name":"web"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/docker-restores", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing container and volume returns 400", func(t *testing.T) {
		store := &mockDockerRestoreStore{}
		r := setupDockerRestoreTestRouter(store, user)
		body := `{"snapshot_id":"snap1","agent_id":"` + agentID.String() + `","repository_id":"` + repoID.String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/docker-restores", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("missing required fields returns 400", func(t *testing.T) {
		store := &mockDockerRestoreStore{}
		r := setupDockerRestoreTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/docker-restores", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestDockerRestoreGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("returns restore in org", func(t *testing.T) {
		store := &mockDockerRestoreStore{restore: &models.DockerRestore{ID: id, OrgID: orgID}}
		r := setupDockerRestoreTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-restores/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		store := &mockDockerRestoreStore{restore: &models.DockerRestore{ID: id, OrgID: uuid.New()}}
		r := setupDockerRestoreTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-restores/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockDockerRestoreStore{}
		r := setupDockerRestoreTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-restores/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
