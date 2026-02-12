package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockDashboardMetricsStore struct {
	user           *models.User
	dashStats      *models.DashboardStats
	rate7d         *models.BackupSuccessRate
	rate30d        *models.BackupSuccessRate
	growthTrend    []*models.StorageGrowthTrend
	durationTrend  []*models.BackupDurationTrend
	dailyStats     []*models.DailyBackupStats
	dailySummaries []models.MetricsDailySummary
	getUserErr     error
	dashErr        error
	ratesErr       error
	growthErr      error
	durationErr    error
	dailyErr       error
	summariesErr   error
}

func (m *mockDashboardMetricsStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	return m.user, nil
}

func (m *mockDashboardMetricsStore) GetDashboardStats(_ context.Context, _ uuid.UUID) (*models.DashboardStats, error) {
	if m.dashErr != nil {
		return nil, m.dashErr
	}
	return m.dashStats, nil
}

func (m *mockDashboardMetricsStore) GetBackupSuccessRates(_ context.Context, _ uuid.UUID) (*models.BackupSuccessRate, *models.BackupSuccessRate, error) {
	if m.ratesErr != nil {
		return nil, nil, m.ratesErr
	}
	return m.rate7d, m.rate30d, nil
}

func (m *mockDashboardMetricsStore) GetStorageGrowthTrend(_ context.Context, _ uuid.UUID, _ int) ([]*models.StorageGrowthTrend, error) {
	if m.growthErr != nil {
		return nil, m.growthErr
	}
	return m.growthTrend, nil
}

func (m *mockDashboardMetricsStore) GetBackupDurationTrend(_ context.Context, _ uuid.UUID, _ int) ([]*models.BackupDurationTrend, error) {
	if m.durationErr != nil {
		return nil, m.durationErr
	}
	return m.durationTrend, nil
}

func (m *mockDashboardMetricsStore) GetDailyBackupStats(_ context.Context, _ uuid.UUID, _ int) ([]*models.DailyBackupStats, error) {
	if m.dailyErr != nil {
		return nil, m.dailyErr
	}
	return m.dailyStats, nil
}

func (m *mockDashboardMetricsStore) GetDailySummaries(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]models.MetricsDailySummary, error) {
	if m.summariesErr != nil {
		return nil, m.summariesErr
	}
	return m.dailySummaries, nil
}

func setupDashboardMetricsTestRouter(store DashboardMetricsStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewDashboardMetricsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestDashboardGetStats(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:      dbUser,
			dashStats: &models.DashboardStats{AgentTotal: 5},
			rate7d:    &models.BackupSuccessRate{SuccessPercent: 95.0},
			rate30d:   &models.BackupSuccessRate{SuccessPercent: 90.0},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/stats"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockDashboardMetricsStore{getUserErr: errors.New("not found")}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/stats"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("dash stats error", func(t *testing.T) {
		store := &mockDashboardMetricsStore{user: dbUser, dashErr: errors.New("db error")}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/stats"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("rates error still returns stats", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:      dbUser,
			dashStats: &models.DashboardStats{AgentTotal: 3},
			ratesErr:  errors.New("db error"),
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/stats"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 even with rates error, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockDashboardMetricsStore{}
		r := setupDashboardMetricsTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/stats"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestDashboardGetSuccessRates(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:    dbUser,
			rate7d:  &models.BackupSuccessRate{SuccessPercent: 95.0},
			rate30d: &models.BackupSuccessRate{SuccessPercent: 90.0},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/success-rates"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		store := &mockDashboardMetricsStore{user: dbUser, ratesErr: errors.New("db error")}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/success-rates"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestDashboardGetStorageGrowthTrend(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:        dbUser,
			growthTrend: []*models.StorageGrowthTrend{},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/storage-growth"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("custom days", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:        dbUser,
			growthTrend: []*models.StorageGrowthTrend{},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/storage-growth?days=7"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		store := &mockDashboardMetricsStore{user: dbUser, growthErr: errors.New("db error")}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/storage-growth"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestDashboardGetBackupDurationTrend(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:          dbUser,
			durationTrend: []*models.BackupDurationTrend{},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/backup-duration"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		store := &mockDashboardMetricsStore{user: dbUser, durationErr: errors.New("db error")}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/backup-duration"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestDashboardGetDailyBackupStats(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:       dbUser,
			dailyStats: []*models.DailyBackupStats{},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/daily-backups"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("error", func(t *testing.T) {
		store := &mockDashboardMetricsStore{user: dbUser, dailyErr: errors.New("db error")}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/daily-backups"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("uses summaries for days > 7", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user: dbUser,
			dailySummaries: []models.MetricsDailySummary{
				{
					Date:              time.Now().AddDate(0, 0, -10),
					TotalBackups:      5,
					SuccessfulBackups: 4,
					FailedBackups:     1,
					TotalSizeBytes:    1024,
				},
			},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/daily-backups?days=30"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("falls back to real-time when summaries empty", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:           dbUser,
			dailySummaries: []models.MetricsDailySummary{},
			dailyStats:     []*models.DailyBackupStats{},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/daily-backups?days=30"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("falls back to real-time when summaries error", func(t *testing.T) {
		store := &mockDashboardMetricsStore{
			user:         dbUser,
			summariesErr: errors.New("db error"),
			dailyStats:   []*models.DailyBackupStats{},
		}
		r := setupDashboardMetricsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/dashboard-metrics/daily-backups?days=30"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}
