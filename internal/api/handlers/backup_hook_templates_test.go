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

type mockBackupHookTemplateStore struct {
	templates []*models.BackupHookTemplate
	template  *models.BackupHookTemplate
	schedule  *models.Schedule
	agent     *models.Agent
	script    *models.BackupScript
	listErr   error
	getErr    error
	createErr error
	updateErr error
	deleteErr error
	scheduErr error
	agentErr  error
	scriptErr error
	scriptGet error
	scriptUpd error
	incrErr   error
}

func (m *mockBackupHookTemplateStore) GetBackupHookTemplatesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.BackupHookTemplate, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.templates, nil
}

func (m *mockBackupHookTemplateStore) GetBackupHookTemplateByID(_ context.Context, _ uuid.UUID) (*models.BackupHookTemplate, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.template, nil
}

func (m *mockBackupHookTemplateStore) GetBackupHookTemplatesByServiceType(_ context.Context, _ uuid.UUID, _ string) ([]*models.BackupHookTemplate, error) {
	return m.templates, nil
}

func (m *mockBackupHookTemplateStore) GetBackupHookTemplatesByVisibility(_ context.Context, _ uuid.UUID, _ models.BackupHookTemplateVisibility) ([]*models.BackupHookTemplate, error) {
	return m.templates, nil
}

func (m *mockBackupHookTemplateStore) CreateBackupHookTemplate(_ context.Context, _ *models.BackupHookTemplate) error {
	return m.createErr
}

func (m *mockBackupHookTemplateStore) UpdateBackupHookTemplate(_ context.Context, _ *models.BackupHookTemplate) error {
	return m.updateErr
}

func (m *mockBackupHookTemplateStore) DeleteBackupHookTemplate(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockBackupHookTemplateStore) IncrementBackupHookTemplateUsage(_ context.Context, _ uuid.UUID) error {
	return m.incrErr
}

func (m *mockBackupHookTemplateStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	if m.scheduErr != nil {
		return nil, m.scheduErr
	}
	return m.schedule, nil
}

func (m *mockBackupHookTemplateStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.agentErr != nil {
		return nil, m.agentErr
	}
	return m.agent, nil
}

func (m *mockBackupHookTemplateStore) CreateBackupScript(_ context.Context, _ *models.BackupScript) error {
	return m.scriptErr
}

func (m *mockBackupHookTemplateStore) GetBackupScriptByScheduleAndType(_ context.Context, _ uuid.UUID, _ models.BackupScriptType) (*models.BackupScript, error) {
	if m.scriptGet != nil {
		return nil, m.scriptGet
	}
	return m.script, nil
}

func (m *mockBackupHookTemplateStore) UpdateBackupScript(_ context.Context, _ *models.BackupScript) error {
	return m.scriptUpd
}

func setupBackupHookTemplatesTestRouter(store BackupHookTemplateStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := SetupTestRouter(user)
	handler := NewBackupHookTemplatesHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestBackupHookTemplatesList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns templates", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{templates: []*models.BackupHookTemplate{}}
		r := setupBackupHookTemplatesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-hook-templates"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{}
		r := setupBackupHookTemplatesTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-hook-templates"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{listErr: errors.New("db down")}
		r := setupBackupHookTemplatesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-hook-templates"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestBackupHookTemplatesCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates template", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{}
		r := setupBackupHookTemplatesTestRouter(store, user)
		body := `{"name":"My Template","service_type":"postgres"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/backup-hook-templates", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid visibility returns 400", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{}
		r := setupBackupHookTemplatesTestRouter(store, user)
		body := `{"name":"x","service_type":"y","visibility":"bogus"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/backup-hook-templates", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{}
		r := setupBackupHookTemplatesTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/backup-hook-templates", `{"name":"x","service_type":"y"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestBackupHookTemplatesGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{}
		r := setupBackupHookTemplatesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-hook-templates/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("not found returns 404", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{getErr: errors.New("not found")}
		r := setupBackupHookTemplatesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/backup-hook-templates/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestBackupHookTemplatesDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("deletes template", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{template: &models.BackupHookTemplate{
			ID:          id,
			OrgID:       orgID,
			CreatedByID: user.ID,
			Visibility:  models.BackupHookTemplateVisibilityPrivate,
		}}
		r := setupBackupHookTemplatesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/backup-hook-templates/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("wrong org returns 404", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{template: &models.BackupHookTemplate{
			ID:          id,
			OrgID:       uuid.New(),
			CreatedByID: user.ID,
		}}
		r := setupBackupHookTemplatesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/backup-hook-templates/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockBackupHookTemplateStore{}
		r := setupBackupHookTemplatesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/backup-hook-templates/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
