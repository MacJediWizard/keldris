package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockDockerStore struct {
	user         *models.User
	agent        *models.Agent
	health       *models.AgentDockerHealth
	orgHealth    []*models.AgentDockerHealth
	summary      *models.DockerHealthSummary
	restarts     []*models.ContainerRestartEvent
	userErr      error
	agentErr     error
	healthErr    error
	orgHealthErr error
	summaryErr   error
	restartsErr  error
}

func (m *mockDockerStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	if m.userErr != nil {
		return nil, m.userErr
	}
	return m.user, nil
}

func (m *mockDockerStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func (m *mockDockerStore) GetAgentDockerHealth(_ context.Context, _ uuid.UUID) (*models.AgentDockerHealth, error) {
	if m.healthErr != nil {
		return nil, m.healthErr
	}
	return m.health, nil
}

func (m *mockDockerStore) GetAgentDockerHealthByOrgID(_ context.Context, _ uuid.UUID) ([]*models.AgentDockerHealth, error) {
	if m.orgHealthErr != nil {
		return nil, m.orgHealthErr
	}
	return m.orgHealth, nil
}

func (m *mockDockerStore) GetDockerHealthSummary(_ context.Context, _ uuid.UUID) (*models.DockerHealthSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summary, nil
}

func (m *mockDockerStore) GetRecentContainerRestartEvents(_ context.Context, _ uuid.UUID, _ int) ([]*models.ContainerRestartEvent, error) {
	if m.restartsErr != nil {
		return nil, m.restartsErr
	}
	return m.restarts, nil
}

func setupDockerTestRouter(store DockerStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDockerHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDockerGetSummary(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns summary", func(t *testing.T) {
		store := &mockDockerStore{summary: &models.DockerHealthSummary{}}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/summary"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		store := &mockDockerStore{summary: &models.DockerHealthSummary{}}
		r := setupDockerTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/summary"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("error returns 500", func(t *testing.T) {
		store := &mockDockerStore{summaryErr: errors.New("db down")}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/summary"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestDockerGetDashboardWidget(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns widget data", func(t *testing.T) {
		store := &mockDockerStore{summary: &models.DockerHealthSummary{}}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/widget"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		store := &mockDockerStore{summary: &models.DockerHealthSummary{}}
		r := setupDockerTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/widget"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestDockerListAgentDockerHealth(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns list", func(t *testing.T) {
		store := &mockDockerStore{orgHealth: []*models.AgentDockerHealth{}}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/agents"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var body map[string]interface{}
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["docker_health"]; !ok {
			t.Errorf("expected docker_health key in response")
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		store := &mockDockerStore{}
		r := setupDockerTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/agents"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestDockerGetAgentDockerHealth(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	user := testUser(orgID)

	t.Run("returns agent docker health", func(t *testing.T) {
		store := &mockDockerStore{
			agent: &models.Agent{ID: agentID, OrgID: orgID},
			health: &models.AgentDockerHealth{
				AgentID:      agentID,
				DockerHealth: &models.DockerHealth{},
			},
		}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/agents/"+agentID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid agent ID", func(t *testing.T) {
		store := &mockDockerStore{}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/agents/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("agent not in org", func(t *testing.T) {
		store := &mockDockerStore{
			agent: &models.Agent{ID: agentID, OrgID: uuid.New()},
		}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/agents/"+agentID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestDockerGetRecentRestarts(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns restarts", func(t *testing.T) {
		store := &mockDockerStore{restarts: []*models.ContainerRestartEvent{}}
		r := setupDockerTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/restarts"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		store := &mockDockerStore{}
		r := setupDockerTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/docker/restarts"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
