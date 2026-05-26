package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockUsageStore struct {
	org       *models.Organization
	metrics   []*models.UsageMetrics
	latest    *models.UsageMetrics
	limits    *models.OrgUsageLimits
	alerts    []*models.UsageAlert
	monthly   *models.MonthlyUsageSummary
	summaries []*models.MonthlyUsageSummary
	err       error
}

func (m *mockUsageStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
	return m.org, m.err
}

func (m *mockUsageStore) GetUsageMetricsByOrgID(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*models.UsageMetrics, error) {
	return m.metrics, m.err
}

func (m *mockUsageStore) GetLatestUsageMetrics(_ context.Context, _ uuid.UUID) (*models.UsageMetrics, error) {
	return m.latest, m.err
}

func (m *mockUsageStore) GetOrgUsageLimits(_ context.Context, _ uuid.UUID) (*models.OrgUsageLimits, error) {
	return m.limits, m.err
}

func (m *mockUsageStore) CreateOrgUsageLimits(_ context.Context, _ *models.OrgUsageLimits) error {
	return m.err
}

func (m *mockUsageStore) UpdateOrgUsageLimits(_ context.Context, _ *models.OrgUsageLimits) error {
	return m.err
}

func (m *mockUsageStore) UpsertOrgUsageLimits(_ context.Context, l *models.OrgUsageLimits) error {
	if m.err != nil {
		return m.err
	}
	m.limits = l
	return nil
}

func (m *mockUsageStore) GetActiveUsageAlertsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.UsageAlert, error) {
	return m.alerts, m.err
}

func (m *mockUsageStore) AcknowledgeUsageAlert(_ context.Context, _, _ uuid.UUID) error {
	return m.err
}

func (m *mockUsageStore) ResolveUsageAlert(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockUsageStore) GetMonthlyUsageSummary(_ context.Context, _ uuid.UUID, _ string) (*models.MonthlyUsageSummary, error) {
	return m.monthly, m.err
}

func (m *mockUsageStore) GetMonthlyUsageSummariesByOrgID(_ context.Context, _ uuid.UUID, _ int) ([]*models.MonthlyUsageSummary, error) {
	return m.summaries, m.err
}

func setupUsageTestRouter(store UsageStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewUsageHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestUsageGetLimits(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns existing limits", func(t *testing.T) {
		store := &mockUsageStore{limits: models.NewOrgUsageLimits(orgID)}
		r := setupUsageTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/usage/limits"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("returns defaults when none exist", func(t *testing.T) {
		store := &mockUsageStore{}
		r := setupUsageTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/usage/limits"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})
}

func TestUsageGetAlerts(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockUsageStore{alerts: []*models.UsageAlert{{ID: uuid.New(), OrgID: orgID}}}
	r := setupUsageTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/usage/alerts"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUsageGetMonthlySummaries(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockUsageStore{summaries: []*models.MonthlyUsageSummary{{OrgID: orgID, YearMonth: "2026-05"}}}
	r := setupUsageTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/usage/monthly"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
