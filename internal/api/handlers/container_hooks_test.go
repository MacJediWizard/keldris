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

type mockContainerHookStore struct {
	hooks      []*models.ContainerBackupHook
	hookByID   map[uuid.UUID]*models.ContainerBackupHook
	schedule   *models.Schedule
	agent      *models.Agent
	executions []*models.ContainerHookExecution
	err        error
}

func (m *mockContainerHookStore) GetContainerBackupHooksByScheduleID(_ context.Context, _ uuid.UUID) ([]*models.ContainerBackupHook, error) {
	return m.hooks, m.err
}

func (m *mockContainerHookStore) GetContainerBackupHookByID(_ context.Context, id uuid.UUID) (*models.ContainerBackupHook, error) {
	if m.hookByID == nil {
		return nil, m.err
	}
	return m.hookByID[id], m.err
}

func (m *mockContainerHookStore) CreateContainerBackupHook(_ context.Context, h *models.ContainerBackupHook) error {
	if m.err != nil {
		return m.err
	}
	if m.hookByID == nil {
		m.hookByID = map[uuid.UUID]*models.ContainerBackupHook{}
	}
	m.hookByID[h.ID] = h
	m.hooks = append(m.hooks, h)
	return nil
}

func (m *mockContainerHookStore) UpdateContainerBackupHook(_ context.Context, _ *models.ContainerBackupHook) error {
	return m.err
}

func (m *mockContainerHookStore) DeleteContainerBackupHook(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockContainerHookStore) GetContainerHookExecutionsByBackupID(_ context.Context, _ uuid.UUID) ([]*models.ContainerHookExecution, error) {
	return m.executions, m.err
}

func (m *mockContainerHookStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	return m.schedule, m.err
}

func (m *mockContainerHookStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func setupContainerHooksTestRouter(store ContainerHookStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewContainerHooksHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestContainerHooksListTemplates(t *testing.T) {
	user := testUser(uuid.New())
	store := &mockContainerHookStore{}
	r := setupContainerHooksTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/container-hook-templates"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestContainerHooksList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	scheduleID := uuid.New()

	t.Run("returns hooks for accessible schedule", func(t *testing.T) {
		agentID := uuid.New()
		store := &mockContainerHookStore{
			schedule: &models.Schedule{ID: scheduleID, AgentID: agentID},
			agent:    &models.Agent{ID: agentID, OrgID: orgID},
			hooks:    []*models.ContainerBackupHook{{ID: uuid.New(), ScheduleID: scheduleID}},
		}
		r := setupContainerHooksTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/container-hooks"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid schedule id returns 400", func(t *testing.T) {
		store := &mockContainerHookStore{}
		r := setupContainerHooksTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/not-a-uuid/container-hooks"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockContainerHookStore{}
		r := setupContainerHooksTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/container-hooks"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
