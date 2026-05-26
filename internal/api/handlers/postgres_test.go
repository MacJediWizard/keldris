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

type mockPostgresStore struct {
	agent  *models.Agent
	agents []*models.Agent
	err    error
}

func (m *mockPostgresStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func (m *mockPostgresStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return m.agents, m.err
}

func setupPostgresTestRouter(store PostgresStore, user *auth.SessionUser, km *crypto.KeyManager) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewPostgresHandler(store, km, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestPostgresTestConnection(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)
	store := &mockPostgresStore{}
	r := setupPostgresTestRouter(store, user, km)

	t.Run("missing body returns 400", func(t *testing.T) {
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/postgres/test-connection", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/postgres/test-connection", `{invalid`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestPostgresGetRestoreInstructions(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)
	store := &mockPostgresStore{}
	r := setupPostgresTestRouter(store, user, km)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/postgres/restore-instructions"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestPostgresEncryptPassword(t *testing.T) {
	user := testUser(uuid.New())
	km := newTestKeyManager(t)
	store := &mockPostgresStore{}
	r := setupPostgresTestRouter(store, user, km)

	t.Run("encrypts password", func(t *testing.T) {
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/postgres/encrypt-password", `{"password":"hunter2"}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing password returns 400", func(t *testing.T) {
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/postgres/encrypt-password", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
