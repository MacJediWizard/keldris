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

type mockConfigExportStore struct {
	agent     *models.Agent
	agents    []*models.Agent
	schedule  *models.Schedule
	schedules []*models.Schedule
	repo      *models.Repository
	repos     []*models.Repository
	template  *models.ConfigTemplate
	templates []*models.ConfigTemplate
	err       error
}

func (m *mockConfigExportStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func (m *mockConfigExportStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return m.agents, m.err
}

func (m *mockConfigExportStore) CreateAgent(_ context.Context, _ *models.Agent) error {
	return m.err
}

func (m *mockConfigExportStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	return m.schedule, m.err
}

func (m *mockConfigExportStore) GetSchedulesByAgentID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, m.err
}

func (m *mockConfigExportStore) CreateSchedule(_ context.Context, _ *models.Schedule) error {
	return m.err
}

func (m *mockConfigExportStore) UpdateSchedule(_ context.Context, _ *models.Schedule) error {
	return m.err
}

func (m *mockConfigExportStore) SetScheduleRepositories(_ context.Context, _ uuid.UUID, _ []models.ScheduleRepository) error {
	return m.err
}

func (m *mockConfigExportStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	return m.repo, m.err
}

func (m *mockConfigExportStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Repository, error) {
	return m.repos, m.err
}

func (m *mockConfigExportStore) CreateConfigTemplate(_ context.Context, _ *models.ConfigTemplate) error {
	return m.err
}

func (m *mockConfigExportStore) GetConfigTemplateByID(_ context.Context, _ uuid.UUID) (*models.ConfigTemplate, error) {
	return m.template, m.err
}

func (m *mockConfigExportStore) GetConfigTemplatesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.ConfigTemplate, error) {
	return m.templates, m.err
}

func (m *mockConfigExportStore) GetPublicConfigTemplates(_ context.Context) ([]*models.ConfigTemplate, error) {
	return m.templates, m.err
}

func (m *mockConfigExportStore) UpdateConfigTemplate(_ context.Context, _ *models.ConfigTemplate) error {
	return m.err
}

func (m *mockConfigExportStore) DeleteConfigTemplate(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockConfigExportStore) IncrementTemplateUsageCount(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func setupConfigExportTestRouter(store ConfigExportStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewConfigExportHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestConfigExportExportAgent(t *testing.T) {
	user := testUser(uuid.New())

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockConfigExportStore{}
		r := setupConfigExportTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/export/agents/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestConfigExportExportSchedule(t *testing.T) {
	user := testUser(uuid.New())

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockConfigExportStore{}
		r := setupConfigExportTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/export/schedules/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestConfigExportExportBundle(t *testing.T) {
	user := testUser(uuid.New())

	t.Run("invalid json returns 400", func(t *testing.T) {
		store := &mockConfigExportStore{}
		r := setupConfigExportTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/export/bundle", `{invalid`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
