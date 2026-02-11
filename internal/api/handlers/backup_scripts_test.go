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

type mockBackupScriptStore struct {
	scripts       []*models.BackupScript
	schedule      *models.Schedule
	agent         *models.Agent
	getScriptErr  error
	listErr       error
	createErr     error
	updateErr     error
	deleteErr     error
	getSchedErr   error
	getAgentErr   error
}

func (m *mockBackupScriptStore) GetBackupScriptsByScheduleID(_ context.Context, _ uuid.UUID) ([]*models.BackupScript, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.scripts, nil
}

func (m *mockBackupScriptStore) GetBackupScriptByID(_ context.Context, id uuid.UUID) (*models.BackupScript, error) {
	if m.getScriptErr != nil {
		return nil, m.getScriptErr
	}
	for _, s := range m.scripts {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockBackupScriptStore) CreateBackupScript(_ context.Context, _ *models.BackupScript) error {
	return m.createErr
}

func (m *mockBackupScriptStore) UpdateBackupScript(_ context.Context, _ *models.BackupScript) error {
	return m.updateErr
}

func (m *mockBackupScriptStore) DeleteBackupScript(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockBackupScriptStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	if m.getSchedErr != nil {
		return nil, m.getSchedErr
	}
	return m.schedule, nil
}

func (m *mockBackupScriptStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.getAgentErr != nil {
		return nil, m.getAgentErr
	}
	return m.agent, nil
}

func setupBackupScriptsTestRouter(store BackupScriptStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewBackupScriptsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func testBackupScriptSetup(orgID uuid.UUID) (uuid.UUID, *models.Schedule, *models.Agent) {
	agentID := uuid.New()
	scheduleID := uuid.New()
	schedule := &models.Schedule{ID: scheduleID, AgentID: agentID}
	agent := &models.Agent{ID: agentID, OrgID: orgID, Hostname: "host1"}
	return scheduleID, schedule, agent
}

func TestBackupScriptsList(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	scheduleID, schedule, agent := testBackupScriptSetup(orgID)

	t.Run("success", func(t *testing.T) {
		script := models.NewBackupScript(scheduleID, models.BackupScriptTypePreBackup, "echo hello")
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{script},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockBackupScriptStore{schedule: schedule, agent: agent}
		r := setupBackupScriptsTestRouter(store, TestUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid schedule id", func(t *testing.T) {
		store := &mockBackupScriptStore{}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/bad-id/scripts"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("schedule not found", func(t *testing.T) {
		store := &mockBackupScriptStore{getSchedErr: errors.New("not found")}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+uuid.New().String()+"/scripts"))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("agent wrong org", func(t *testing.T) {
		otherAgent := &models.Agent{ID: uuid.New(), OrgID: uuid.New(), Hostname: "other"}
		store := &mockBackupScriptStore{schedule: schedule, agent: otherAgent}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts"))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("list error", func(t *testing.T) {
		store := &mockBackupScriptStore{
			schedule: schedule,
			agent:    agent,
			listErr:  errors.New("db error"),
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockBackupScriptStore{}
		r := setupBackupScriptsTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestBackupScriptsGet(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	scheduleID, schedule, agent := testBackupScriptSetup(orgID)
	script := models.NewBackupScript(scheduleID, models.BackupScriptTypePreBackup, "echo hello")

	t.Run("success", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{script},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+script.ID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("script not found", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("script wrong schedule", func(t *testing.T) {
		wrongScript := models.NewBackupScript(uuid.New(), models.BackupScriptTypePreBackup, "echo wrong")
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{wrongScript},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+wrongScript.ID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestBackupScriptsCreate(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	scheduleID, schedule, agent := testBackupScriptSetup(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockBackupScriptStore{schedule: schedule, agent: agent}
		r := setupBackupScriptsTestRouter(store, user)
		body := `{"type":"pre_backup","script":"echo hello"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/scripts", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		store := &mockBackupScriptStore{schedule: schedule, agent: agent}
		r := setupBackupScriptsTestRouter(store, user)
		body := `{"type":"invalid_type","script":"echo hello"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/scripts", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("missing script", func(t *testing.T) {
		store := &mockBackupScriptStore{schedule: schedule, agent: agent}
		r := setupBackupScriptsTestRouter(store, user)
		body := `{"type":"pre_backup"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/scripts", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		store := &mockBackupScriptStore{schedule: schedule, agent: agent}
		r := setupBackupScriptsTestRouter(store, user)
		body := `{"type":"pre_backup","script":"echo hello","timeout_seconds":9999}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/scripts", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockBackupScriptStore{schedule: schedule, agent: agent, createErr: errors.New("db error")}
		r := setupBackupScriptsTestRouter(store, user)
		body := `{"type":"pre_backup","script":"echo hello"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/schedules/"+scheduleID.String()+"/scripts", body))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestBackupScriptsUpdate(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	scheduleID, schedule, agent := testBackupScriptSetup(orgID)
	script := models.NewBackupScript(scheduleID, models.BackupScriptTypePreBackup, "echo old")

	t.Run("success", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{script},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		body := `{"script":"echo new"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+script.ID.String(), body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("script not found", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+uuid.New().String(), `{"script":"x"}`))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid timeout", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{script},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		body := `{"timeout_seconds":0}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+script.ID.String(), body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:   []*models.BackupScript{script},
			schedule:  schedule,
			agent:     agent,
			updateErr: errors.New("db error"),
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+script.ID.String(), `{"script":"x"}`))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestBackupScriptsDelete(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	scheduleID, schedule, agent := testBackupScriptSetup(orgID)
	script := models.NewBackupScript(scheduleID, models.BackupScriptTypePreBackup, "echo hello")

	t.Run("success", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{script},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+script.ID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("script not found", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:  []*models.BackupScript{},
			schedule: schedule,
			agent:    agent,
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockBackupScriptStore{
			scripts:   []*models.BackupScript{script},
			schedule:  schedule,
			agent:     agent,
			deleteErr: errors.New("db error"),
		}
		r := setupBackupScriptsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/schedules/"+scheduleID.String()+"/scripts/"+script.ID.String()))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
