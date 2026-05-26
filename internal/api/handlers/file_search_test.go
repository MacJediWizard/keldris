package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockFileSearchStore struct {
	agent      *models.Agent
	repository *models.Repository
	repoKey    *models.RepositoryKey
	agentErr   error
	repoErr    error
	repoKeyErr error
}

func (m *mockFileSearchStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func (m *mockFileSearchStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	if m.repoErr != nil {
		return nil, m.repoErr
	}
	return m.repository, nil
}

func (m *mockFileSearchStore) GetRepositoryKeyByRepositoryID(_ context.Context, _ uuid.UUID) (*models.RepositoryKey, error) {
	if m.repoKeyErr != nil {
		return nil, m.repoKeyErr
	}
	return m.repoKey, nil
}

func setupFileSearchTestRouter(store FileSearchStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	// 32-byte key for AES-256
	km, err := crypto.NewKeyManager(make([]byte, 32))
	if err != nil {
		panic(err)
	}
	handler := NewFileSearchHandler(store, km, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestFileSearchSearchFiles(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	agentID := uuid.New()
	repoID := uuid.New()

	t.Run("missing query returns 400", func(t *testing.T) {
		store := &mockFileSearchStore{}
		r := setupFileSearchTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/search/files"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("missing agent_id returns 400", func(t *testing.T) {
		store := &mockFileSearchStore{}
		r := setupFileSearchTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/search/files?q=test"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("missing repository_id returns 400", func(t *testing.T) {
		store := &mockFileSearchStore{}
		r := setupFileSearchTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/search/files?q=test&agent_id="+agentID.String()))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid agent_id returns 400", func(t *testing.T) {
		store := &mockFileSearchStore{}
		r := setupFileSearchTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/search/files?q=test&agent_id=not-a-uuid&repository_id="+repoID.String()))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("agent not found returns 404", func(t *testing.T) {
		store := &mockFileSearchStore{agentErr: errors.New("not found")}
		r := setupFileSearchTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/search/files?q=test&agent_id="+agentID.String()+"&repository_id="+repoID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("agent in other org returns 404", func(t *testing.T) {
		store := &mockFileSearchStore{agent: &models.Agent{ID: agentID, OrgID: uuid.New()}}
		r := setupFileSearchTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/search/files?q=test&agent_id="+agentID.String()+"&repository_id="+repoID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("repo in other org returns 404", func(t *testing.T) {
		store := &mockFileSearchStore{
			agent:      &models.Agent{ID: agentID, OrgID: orgID},
			repository: &models.Repository{ID: repoID, OrgID: uuid.New()},
		}
		r := setupFileSearchTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/search/files?q=test&agent_id="+agentID.String()+"&repository_id="+repoID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}
