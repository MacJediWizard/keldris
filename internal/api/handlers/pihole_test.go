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

type mockPiholeStore struct {
	agent  *models.Agent
	agents []*models.Agent
	err    error
}

func (m *mockPiholeStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func (m *mockPiholeStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return m.agents, m.err
}

func setupPiholeTestRouter(store PiholeStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewPiholeHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestPiholeListAgentsWithPihole(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns empty when no pihole agents", func(t *testing.T) {
		store := &mockPiholeStore{agents: []*models.Agent{{ID: uuid.New(), OrgID: orgID}}}
		r := setupPiholeTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pihole/agents"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockPiholeStore{}
		r := setupPiholeTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pihole/agents"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestPiholeGetStatus(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid agent id returns 400", func(t *testing.T) {
		store := &mockPiholeStore{}
		r := setupPiholeTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pihole/agents/not-a-uuid/status"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
