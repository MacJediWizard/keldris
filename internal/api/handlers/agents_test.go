package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockAgentStore implements AgentStore for testing.
type mockAgentStore struct {
	agents           []*models.Agent
	agentByID        map[uuid.UUID]*models.Agent
	user             *models.User
	stats            *models.AgentStats
	backups          []*models.Backup
	schedules        []*models.Schedule
	healthHistory    []*models.AgentHealthHistory
	fleetHealth      *models.FleetHealthSummary
	createErr        error
	deleteErr        error
	updateErr        error
	getErr           error
	updateAPIKeyErr  error
	revokeAPIKeyErr  error
}

func (m *mockAgentStore) GetAgentsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Agent, error) {
	var result []*models.Agent
	for _, a := range m.agents {
		if a.OrgID == orgID {
			result = append(result, a)
		}
	}
	return result, nil
}

func (m *mockAgentStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if a, ok := m.agentByID[id]; ok {
		return a, nil
	}
	return nil, errors.New("agent not found")
}

func (m *mockAgentStore) CreateAgent(_ context.Context, agent *models.Agent) error {
	return m.createErr
}

func (m *mockAgentStore) UpdateAgent(_ context.Context, agent *models.Agent) error {
	return m.updateErr
}

func (m *mockAgentStore) DeleteAgent(_ context.Context, id uuid.UUID) error {
	return m.deleteErr
}

func (m *mockAgentStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockAgentStore) GetAgentByAPIKeyHash(_ context.Context, hash string) (*models.Agent, error) {
	return nil, errors.New("not found")
}

func (m *mockAgentStore) UpdateAgentAPIKeyHash(_ context.Context, id uuid.UUID, hash string) error {
	return m.updateAPIKeyErr
}

func (m *mockAgentStore) RevokeAgentAPIKey(_ context.Context, id uuid.UUID) error {
	return m.revokeAPIKeyErr
}

func (m *mockAgentStore) GetAgentStats(_ context.Context, agentID uuid.UUID) (*models.AgentStats, error) {
	if m.stats != nil {
		return m.stats, nil
	}
	return nil, errors.New("no stats")
}

func (m *mockAgentStore) GetBackupsByAgentID(_ context.Context, agentID uuid.UUID) ([]*models.Backup, error) {
	return m.backups, nil
}

func (m *mockAgentStore) GetSchedulesByAgentID(_ context.Context, agentID uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, nil
}

func (m *mockAgentStore) GetAgentHealthHistory(_ context.Context, agentID uuid.UUID, limit int) ([]*models.AgentHealthHistory, error) {
	return m.healthHistory, nil
}

func (m *mockAgentStore) GetFleetHealthSummary(_ context.Context, orgID uuid.UUID) (*models.FleetHealthSummary, error) {
	if m.fleetHealth != nil {
		return m.fleetHealth, nil
	}
	return nil, errors.New("no fleet health")
}

func setupAgentTestRouter(store AgentStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Inject user into context
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})

	handler := NewAgentsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListAgents(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	agent := &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "test-host",
		Status:   models.AgentStatusActive,
	}

	store := &mockAgentStore{
		agents:    []*models.Agent{agent},
		agentByID: map[uuid.UUID]*models.Agent{agentID: agent},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/agents", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if _, ok := resp["agents"]; !ok {
			t.Fatal("expected 'agents' key in response")
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupAgentTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/agents", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupAgentTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/agents", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})
}

func TestCreateAgent(t *testing.T) {
	orgID := uuid.New()
	store := &mockAgentStore{
		agentByID: map[uuid.UUID]*models.Agent{},
	}
	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"hostname":"new-agent"}`
		req, _ := http.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp CreateAgentResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Hostname != "new-agent" {
			t.Fatalf("expected hostname 'new-agent', got %q", resp.Hostname)
		}
		if resp.APIKey == "" {
			t.Fatal("expected non-empty API key")
		}
		if !strings.HasPrefix(resp.APIKey, "kld_") {
			t.Fatalf("expected API key to start with 'kld_', got %q", resp.APIKey)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"hostname":""}`
		req, _ := http.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("missing body", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockAgentStore{
			agentByID: map[uuid.UUID]*models.Agent{},
			createErr: errors.New("db error"),
		}
		r := setupAgentTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"hostname":"fail-agent"}`
		req, _ := http.NewRequest("POST", "/api/v1/agents", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}

func TestDeleteAgent(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	otherOrgID := uuid.New()

	agent := &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "delete-me",
		Status:   models.AgentStatusActive,
	}

	store := &mockAgentStore{
		agents:    []*models.Agent{agent},
		agentByID: map[uuid.UUID]*models.Agent{agentID: agent},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/agents/not-a-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/agents/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{
			ID:           uuid.New(),
			CurrentOrgID: otherOrgID,
		}
		r := setupAgentTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockAgentStore{
			agents:    []*models.Agent{agent},
			agentByID: map[uuid.UUID]*models.Agent{agentID: agent},
			deleteErr: errors.New("db error"),
		}
		r := setupAgentTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})
}

func TestGetAgent(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()

	agent := &models.Agent{
		ID:       agentID,
		OrgID:    orgID,
		Hostname: "test-host",
		Status:   models.AgentStatusActive,
	}

	store := &mockAgentStore{
		agentByID: map[uuid.UUID]*models.Agent{agentID: agent},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/agents/bad-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{
			ID:           uuid.New(),
			CurrentOrgID: uuid.New(),
		}
		r := setupAgentTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/agents/"+agentID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
	})
}

func TestFleetHealth(t *testing.T) {
	orgID := uuid.New()
	store := &mockAgentStore{
		agentByID: map[uuid.UUID]*models.Agent{},
		fleetHealth: &models.FleetHealthSummary{
			TotalAgents:  5,
			HealthyCount: 3,
			WarningCount: 1,
			CriticalCount: 1,
		},
	}

	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupAgentTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/agents/fleet-health", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}
