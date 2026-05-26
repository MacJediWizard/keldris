package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockRansomwareStore struct {
	settings    []*models.RansomwareSettings
	scheduleSet *models.RansomwareSettings
	alerts      []*models.RansomwareAlert
	alert       *models.RansomwareAlert
	count       int
	schedule    *models.Schedule
	agent       *models.Agent
	err         error
}

func (m *mockRansomwareStore) GetRansomwareSettingsByScheduleID(_ context.Context, _ uuid.UUID) (*models.RansomwareSettings, error) {
	return m.scheduleSet, m.err
}

func (m *mockRansomwareStore) GetRansomwareSettingsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.RansomwareSettings, error) {
	return m.settings, m.err
}

func (m *mockRansomwareStore) CreateRansomwareSettings(_ context.Context, _ *models.RansomwareSettings) error {
	return m.err
}

func (m *mockRansomwareStore) UpdateRansomwareSettings(_ context.Context, _ *models.RansomwareSettings) error {
	return m.err
}

func (m *mockRansomwareStore) DeleteRansomwareSettings(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockRansomwareStore) GetRansomwareAlertsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.RansomwareAlert, error) {
	return m.alerts, m.err
}

func (m *mockRansomwareStore) GetActiveRansomwareAlertsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.RansomwareAlert, error) {
	return m.alerts, m.err
}

func (m *mockRansomwareStore) GetActiveRansomwareAlertCountByOrgID(_ context.Context, _ uuid.UUID) (int, error) {
	return m.count, m.err
}

func (m *mockRansomwareStore) GetRansomwareAlertByID(_ context.Context, _ uuid.UUID) (*models.RansomwareAlert, error) {
	return m.alert, m.err
}

func (m *mockRansomwareStore) UpdateRansomwareAlert(_ context.Context, _ *models.RansomwareAlert) error {
	return m.err
}

func (m *mockRansomwareStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	return m.schedule, m.err
}

func (m *mockRansomwareStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func (m *mockRansomwareStore) ResumeSchedule(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func setupRansomwareTestRouter(store RansomwareStore, user *auth.SessionUser, tier license.Tier) *gin.Engine {
	r := SetupTestRouter(user)
	checker := license.NewFeatureChecker(&stubFeatureStore{tier: tier})
	handler := NewRansomwareHandler(store, checker, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestRansomwareListSettings(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockRansomwareStore{}
	r := setupRansomwareTestRouter(store, user, license.TierEnterprise)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ransomware/settings"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestRansomwareListAlerts(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockRansomwareStore{alerts: []*models.RansomwareAlert{{ID: uuid.New(), OrgID: orgID}}}
	r := setupRansomwareTestRouter(store, user, license.TierEnterprise)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ransomware/alerts"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestRansomwareCountActiveAlerts(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockRansomwareStore{count: 3}
	r := setupRansomwareTestRouter(store, user, license.TierEnterprise)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ransomware/alerts/count"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
