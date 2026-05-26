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

type mockDockerStackStore struct {
	stack      *models.DockerStack
	stackErr   error
	stacks     []*models.DockerStack
	listErr    error
	backup     *models.DockerStackBackup
	backupErr  error
	backups    []*models.DockerStackBackup
	restore    *models.DockerStackRestore
	restoreErr error
	agent      *models.Agent
	agentErr   error
	createErr  error
	updateErr  error
	deleteErr  error
}

func (m *mockDockerStackStore) CreateDockerStack(_ context.Context, _ *models.DockerStack) error {
	return m.createErr
}

func (m *mockDockerStackStore) GetDockerStackByID(_ context.Context, _ uuid.UUID) (*models.DockerStack, error) {
	if m.stackErr != nil {
		return nil, m.stackErr
	}
	return m.stack, nil
}

func (m *mockDockerStackStore) GetDockerStacksByOrgID(_ context.Context, _ uuid.UUID) ([]*models.DockerStack, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.stacks, nil
}

func (m *mockDockerStackStore) GetDockerStacksByAgentID(_ context.Context, _ uuid.UUID) ([]*models.DockerStack, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.stacks, nil
}

func (m *mockDockerStackStore) UpdateDockerStack(_ context.Context, _ *models.DockerStack) error {
	return m.updateErr
}

func (m *mockDockerStackStore) DeleteDockerStack(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockDockerStackStore) CreateDockerStackBackup(_ context.Context, _ *models.DockerStackBackup) error {
	return nil
}

func (m *mockDockerStackStore) GetDockerStackBackupByID(_ context.Context, _ uuid.UUID) (*models.DockerStackBackup, error) {
	if m.backupErr != nil {
		return nil, m.backupErr
	}
	return m.backup, nil
}

func (m *mockDockerStackStore) GetDockerStackBackupsByStackID(_ context.Context, _ uuid.UUID) ([]*models.DockerStackBackup, error) {
	return m.backups, nil
}

func (m *mockDockerStackStore) UpdateDockerStackBackup(_ context.Context, _ *models.DockerStackBackup) error {
	return nil
}

func (m *mockDockerStackStore) DeleteDockerStackBackup(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockDockerStackStore) CreateDockerStackRestore(_ context.Context, _ *models.DockerStackRestore) error {
	return nil
}

func (m *mockDockerStackStore) GetDockerStackRestoreByID(_ context.Context, _ uuid.UUID) (*models.DockerStackRestore, error) {
	if m.restoreErr != nil {
		return nil, m.restoreErr
	}
	return m.restore, nil
}

func (m *mockDockerStackStore) GetDockerStackRestoresByBackupID(_ context.Context, _ uuid.UUID) ([]*models.DockerStackRestore, error) {
	return nil, nil
}

func (m *mockDockerStackStore) UpdateDockerStackRestore(_ context.Context, _ *models.DockerStackRestore) error {
	return nil
}

func (m *mockDockerStackStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func setupDockerStacksTestRouter(store DockerStackStore, svc DockerStackBackupService, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDockerStacksHandler(store, svc, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDockerStacksList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns stacks", func(t *testing.T) {
		store := &mockDockerStackStore{stacks: []*models.DockerStack{}}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-stacks"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid agent_id returns 400", func(t *testing.T) {
		store := &mockDockerStackStore{}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-stacks?agent_id=bogus"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockDockerStackStore{listErr: errors.New("db down")}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-stacks"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestDockerStacksCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	agentID := uuid.New()

	t.Run("creates stack", func(t *testing.T) {
		store := &mockDockerStackStore{agent: &models.Agent{ID: agentID, OrgID: orgID}}
		r := setupDockerStacksTestRouter(store, nil, user)
		body := `{"name":"web","agent_id":"` + agentID.String() + `","compose_path":"/opt/web/docker-compose.yml"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/docker-stacks", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing name returns 400", func(t *testing.T) {
		store := &mockDockerStackStore{}
		r := setupDockerStacksTestRouter(store, nil, user)
		body := `{"agent_id":"` + agentID.String() + `","compose_path":"/x"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/docker-stacks", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid agent_id returns 400", func(t *testing.T) {
		store := &mockDockerStackStore{}
		r := setupDockerStacksTestRouter(store, nil, user)
		body := `{"name":"web","agent_id":"bogus","compose_path":"/x"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/docker-stacks", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestDockerStacksGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("returns stack", func(t *testing.T) {
		store := &mockDockerStackStore{stack: &models.DockerStack{ID: id, OrgID: orgID}}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-stacks/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		store := &mockDockerStackStore{stack: &models.DockerStack{ID: id, OrgID: uuid.New()}}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-stacks/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockDockerStackStore{}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker-stacks/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestDockerStacksDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("deletes stack", func(t *testing.T) {
		store := &mockDockerStackStore{stack: &models.DockerStack{ID: id, OrgID: orgID}}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/docker-stacks/"+id.String()))
		if resp.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", resp.Code)
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		store := &mockDockerStackStore{stack: &models.DockerStack{ID: id, OrgID: uuid.New()}}
		r := setupDockerStacksTestRouter(store, nil, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/docker-stacks/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}
