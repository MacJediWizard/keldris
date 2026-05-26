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

type mockAgentCommandsStore struct {
	agent     *models.Agent
	command   *models.AgentCommand
	commands  []*models.AgentCommand
	agentErr  error
	cmdErr    error
	listErr   error
	createErr error
	cancelErr error
}

func (m *mockAgentCommandsStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func (m *mockAgentCommandsStore) CreateAgentCommand(_ context.Context, cmd *models.AgentCommand) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.command = cmd
	return nil
}

func (m *mockAgentCommandsStore) GetAgentCommandByID(_ context.Context, _ uuid.UUID) (*models.AgentCommand, error) {
	if m.cmdErr != nil {
		return nil, m.cmdErr
	}
	return m.command, nil
}

func (m *mockAgentCommandsStore) GetCommandsByAgentID(_ context.Context, _ uuid.UUID, _ int) ([]*models.AgentCommand, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.commands, nil
}

func (m *mockAgentCommandsStore) CancelAgentCommand(_ context.Context, _ uuid.UUID) error {
	return m.cancelErr
}

func setupAgentCommandsTestRouter(store AgentCommandsStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewAgentCommandsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestAgentCommandsList(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	user := testUser(orgID)

	t.Run("returns commands", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			agent:    &models.Agent{ID: agentID, OrgID: orgID},
			commands: []*models.AgentCommand{},
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/"+agentID.String()+"/commands"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid agent ID returns 400", func(t *testing.T) {
		store := &mockAgentCommandsStore{}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/not-a-uuid/commands"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("agent in different org returns 404", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			agent: &models.Agent{ID: agentID, OrgID: uuid.New()},
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/"+agentID.String()+"/commands"))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockAgentCommandsStore{}
		r := setupAgentCommandsTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/"+agentID.String()+"/commands"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestAgentCommandsCreate(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	user := testUser(orgID)

	t.Run("creates command", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			agent: &models.Agent{ID: agentID, OrgID: orgID},
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agents/"+agentID.String()+"/commands", `{"type":"backup_now"}`))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid type returns 400", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			agent: &models.Agent{ID: agentID, OrgID: orgID},
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agents/"+agentID.String()+"/commands", `{"type":"bogus"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			agent:     &models.Agent{ID: agentID, OrgID: orgID},
			createErr: errors.New("db down"),
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/agents/"+agentID.String()+"/commands", `{"type":"backup_now"}`))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestAgentCommandsGet(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	commandID := uuid.New()
	user := testUser(orgID)

	t.Run("returns command", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			command: &models.AgentCommand{ID: commandID, AgentID: agentID, OrgID: orgID},
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/"+agentID.String()+"/commands/"+commandID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("command in different org returns 404", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			command: &models.AgentCommand{ID: commandID, AgentID: agentID, OrgID: uuid.New()},
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/agents/"+agentID.String()+"/commands/"+commandID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestAgentCommandsCancel(t *testing.T) {
	orgID := uuid.New()
	agentID := uuid.New()
	commandID := uuid.New()
	user := testUser(orgID)

	t.Run("cancels command", func(t *testing.T) {
		store := &mockAgentCommandsStore{
			command: &models.AgentCommand{ID: commandID, AgentID: agentID, OrgID: orgID},
		}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agents/"+agentID.String()+"/commands/"+commandID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("command not found returns 404", func(t *testing.T) {
		store := &mockAgentCommandsStore{cmdErr: errors.New("not found")}
		r := setupAgentCommandsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/agents/"+agentID.String()+"/commands/"+commandID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}
