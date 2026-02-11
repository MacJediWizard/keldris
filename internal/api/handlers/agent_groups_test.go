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

type mockAgentGroupStore struct {
	groups           []*models.AgentGroup
	agents           []*models.Agent
	agentsWithGroups []*models.AgentWithGroups
	getGroupErr      error
	listGroupsErr    error
	createGroupErr   error
	updateGroupErr   error
	deleteGroupErr   error
	listMembersErr   error
	addAgentErr      error
	removeAgentErr   error
	getAgentErr      error
}

func (m *mockAgentGroupStore) GetAgentGroupsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.AgentGroup, error) {
	if m.listGroupsErr != nil {
		return nil, m.listGroupsErr
	}
	return m.groups, nil
}

func (m *mockAgentGroupStore) GetAgentGroupByID(_ context.Context, id uuid.UUID) (*models.AgentGroup, error) {
	if m.getGroupErr != nil {
		return nil, m.getGroupErr
	}
	for _, g := range m.groups {
		if g.ID == id {
			return g, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAgentGroupStore) CreateAgentGroup(_ context.Context, _ *models.AgentGroup) error {
	return m.createGroupErr
}

func (m *mockAgentGroupStore) UpdateAgentGroup(_ context.Context, _ *models.AgentGroup) error {
	return m.updateGroupErr
}

func (m *mockAgentGroupStore) DeleteAgentGroup(_ context.Context, _ uuid.UUID) error {
	return m.deleteGroupErr
}

func (m *mockAgentGroupStore) GetAgentGroupMembers(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	if m.listMembersErr != nil {
		return nil, m.listMembersErr
	}
	return m.agents, nil
}

func (m *mockAgentGroupStore) AddAgentToGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.addAgentErr
}

func (m *mockAgentGroupStore) RemoveAgentFromGroup(_ context.Context, _, _ uuid.UUID) error {
	return m.removeAgentErr
}

func (m *mockAgentGroupStore) GetAgentByID(_ context.Context, id uuid.UUID) (*models.Agent, error) {
	if m.getAgentErr != nil {
		return nil, m.getAgentErr
	}
	for _, a := range m.agents {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockAgentGroupStore) GetAgentsWithGroupsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.AgentWithGroups, error) {
	return m.agentsWithGroups, nil
}

func setupAgentGroupsTestRouter(store AgentGroupStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewAgentGroupsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestAgentGroupsList(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		g1 := models.NewAgentGroup(orgID, "Production", "Prod servers", "#FF0000")
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{g1}}
		r := setupAgentGroupsTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["groups"]; !ok {
			t.Fatal("expected 'groups' key")
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, TestUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockAgentGroupStore{listGroupsErr: errors.New("db error")}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsGet(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	group := models.NewAgentGroup(orgID, "Dev", "", "")

	t.Run("success", func(t *testing.T) {
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{group}}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups/"+group.ID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups/bad-id"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherGroup := models.NewAgentGroup(uuid.New(), "Other", "", "")
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{otherGroup}}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups/"+otherGroup.ID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsCreate(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups", `{"name":"Staging","description":"Staging servers"}`))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups", `{"description":"no name"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, TestUserNoOrg())
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups", `{"name":"Test"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockAgentGroupStore{createGroupErr: errors.New("db error")}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups", `{"name":"Test"}`))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsUpdate(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	group := models.NewAgentGroup(orgID, "Dev", "", "")

	t.Run("success", func(t *testing.T) {
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{group}}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/agent-groups/"+group.ID.String(), `{"name":"Updated"}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/agent-groups/"+uuid.New().String(), `{"name":"x"}`))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherGroup := models.NewAgentGroup(uuid.New(), "Other", "", "")
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{otherGroup}}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/agent-groups/"+otherGroup.ID.String(), `{"name":"x"}`))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockAgentGroupStore{
			groups:         []*models.AgentGroup{group},
			updateGroupErr: errors.New("db error"),
		}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/agent-groups/"+group.ID.String(), `{"name":"x"}`))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsDelete(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	group := models.NewAgentGroup(orgID, "Dev", "", "")

	t.Run("success", func(t *testing.T) {
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{group}}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-groups/"+group.ID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-groups/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockAgentGroupStore{
			groups:         []*models.AgentGroup{group},
			deleteGroupErr: errors.New("db error"),
		}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-groups/"+group.ID.String()))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsListMembers(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	group := models.NewAgentGroup(orgID, "Dev", "", "")

	t.Run("success", func(t *testing.T) {
		agent := &models.Agent{ID: uuid.New(), OrgID: orgID, Hostname: "host1"}
		store := &mockAgentGroupStore{
			groups: []*models.AgentGroup{group},
			agents: []*models.Agent{agent},
		}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups/"+group.ID.String()+"/agents"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("group not found", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups/"+uuid.New().String()+"/agents"))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockAgentGroupStore{
			groups:         []*models.AgentGroup{group},
			listMembersErr: errors.New("db error"),
		}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agent-groups/"+group.ID.String()+"/agents"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsAddAgent(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	group := models.NewAgentGroup(orgID, "Dev", "", "")
	agent := &models.Agent{ID: uuid.New(), OrgID: orgID, Hostname: "host1"}

	t.Run("success", func(t *testing.T) {
		store := &mockAgentGroupStore{
			groups: []*models.AgentGroup{group},
			agents: []*models.Agent{agent},
		}
		r := setupAgentGroupsTestRouter(store, user)
		body := `{"agent_id":"` + agent.ID.String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups/"+group.ID.String()+"/agents", body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid agent_id", func(t *testing.T) {
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{group}}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups/"+group.ID.String()+"/agents", `{"agent_id":"bad"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("agent not found", func(t *testing.T) {
		store := &mockAgentGroupStore{
			groups: []*models.AgentGroup{group},
			agents: []*models.Agent{},
		}
		r := setupAgentGroupsTestRouter(store, user)
		body := `{"agent_id":"` + uuid.New().String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups/"+group.ID.String()+"/agents", body))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: uuid.New(), OrgID: uuid.New(), Hostname: "other"}
		store := &mockAgentGroupStore{
			groups: []*models.AgentGroup{group},
			agents: []*models.Agent{otherAgent},
		}
		r := setupAgentGroupsTestRouter(store, user)
		body := `{"agent_id":"` + otherAgent.ID.String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups/"+group.ID.String()+"/agents", body))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockAgentGroupStore{
			groups:      []*models.AgentGroup{group},
			agents:      []*models.Agent{agent},
			addAgentErr: errors.New("db error"),
		}
		r := setupAgentGroupsTestRouter(store, user)
		body := `{"agent_id":"` + agent.ID.String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agent-groups/"+group.ID.String()+"/agents", body))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsRemoveAgent(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	group := models.NewAgentGroup(orgID, "Dev", "", "")

	t.Run("success", func(t *testing.T) {
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{group}}
		r := setupAgentGroupsTestRouter(store, user)
		agentID := uuid.New()
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-groups/"+group.ID.String()+"/agents/"+agentID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("invalid agent id", func(t *testing.T) {
		store := &mockAgentGroupStore{groups: []*models.AgentGroup{group}}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-groups/"+group.ID.String()+"/agents/bad-id"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("group not found", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-groups/"+uuid.New().String()+"/agents/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockAgentGroupStore{
			groups:         []*models.AgentGroup{group},
			removeAgentErr: errors.New("db error"),
		}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agent-groups/"+group.ID.String()+"/agents/"+uuid.New().String()))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAgentGroupsListAgentsWithGroups(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockAgentGroupStore{
			agentsWithGroups: []*models.AgentWithGroups{},
		}
		r := setupAgentGroupsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/with-groups"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockAgentGroupStore{}
		r := setupAgentGroupsTestRouter(store, TestUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/with-groups"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
