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

type mockKomodoStore struct {
	integrations []*models.KomodoIntegration
	integration  *models.KomodoIntegration
	containers   []*models.KomodoContainer
	container    *models.KomodoContainer
	stacks       []*models.KomodoStack
	stack        *models.KomodoStack
	events       []*models.KomodoWebhookEvent
	schedules    []*models.Schedule
	err          error
}

func (m *mockKomodoStore) GetKomodoIntegrationsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.KomodoIntegration, error) {
	return m.integrations, m.err
}

func (m *mockKomodoStore) GetKomodoIntegrationByID(_ context.Context, _ uuid.UUID) (*models.KomodoIntegration, error) {
	return m.integration, m.err
}

func (m *mockKomodoStore) CreateKomodoIntegration(_ context.Context, _ *models.KomodoIntegration) error {
	return m.err
}

func (m *mockKomodoStore) UpdateKomodoIntegration(_ context.Context, _ *models.KomodoIntegration) error {
	return m.err
}

func (m *mockKomodoStore) DeleteKomodoIntegration(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockKomodoStore) GetKomodoContainersByOrgID(_ context.Context, _ uuid.UUID) ([]*models.KomodoContainer, error) {
	return m.containers, m.err
}

func (m *mockKomodoStore) GetKomodoContainersByIntegrationID(_ context.Context, _ uuid.UUID) ([]*models.KomodoContainer, error) {
	return m.containers, m.err
}

func (m *mockKomodoStore) GetKomodoContainerByID(_ context.Context, _ uuid.UUID) (*models.KomodoContainer, error) {
	return m.container, m.err
}

func (m *mockKomodoStore) GetKomodoContainerByKomodoID(_ context.Context, _ uuid.UUID, _ string) (*models.KomodoContainer, error) {
	return m.container, m.err
}

func (m *mockKomodoStore) CreateKomodoContainer(_ context.Context, _ *models.KomodoContainer) error {
	return m.err
}

func (m *mockKomodoStore) UpdateKomodoContainer(_ context.Context, _ *models.KomodoContainer) error {
	return m.err
}

func (m *mockKomodoStore) DeleteKomodoContainer(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockKomodoStore) UpsertKomodoContainer(_ context.Context, _ *models.KomodoContainer) error {
	return m.err
}

func (m *mockKomodoStore) GetKomodoStacksByOrgID(_ context.Context, _ uuid.UUID) ([]*models.KomodoStack, error) {
	return m.stacks, m.err
}

func (m *mockKomodoStore) GetKomodoStacksByIntegrationID(_ context.Context, _ uuid.UUID) ([]*models.KomodoStack, error) {
	return m.stacks, m.err
}

func (m *mockKomodoStore) GetKomodoStackByID(_ context.Context, _ uuid.UUID) (*models.KomodoStack, error) {
	return m.stack, m.err
}

func (m *mockKomodoStore) UpsertKomodoStack(_ context.Context, _ *models.KomodoStack) error {
	return m.err
}

func (m *mockKomodoStore) CreateKomodoWebhookEvent(_ context.Context, _ *models.KomodoWebhookEvent) error {
	return m.err
}

func (m *mockKomodoStore) GetKomodoWebhookEventsByOrgID(_ context.Context, _ uuid.UUID, _ int) ([]*models.KomodoWebhookEvent, error) {
	return m.events, m.err
}

func (m *mockKomodoStore) UpdateKomodoWebhookEvent(_ context.Context, _ *models.KomodoWebhookEvent) error {
	return m.err
}

func (m *mockKomodoStore) GetSchedulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, m.err
}

func setupKomodoTestRouter(store KomodoStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewKomodoHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestKomodoListIntegrations(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockKomodoStore{integrations: []*models.KomodoIntegration{}}
	r := setupKomodoTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/integrations/komodo"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestKomodoGetIntegration(t *testing.T) {
	user := testUser(uuid.New())

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockKomodoStore{}
		r := setupKomodoTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/integrations/komodo/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
