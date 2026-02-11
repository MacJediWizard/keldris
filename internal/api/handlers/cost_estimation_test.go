package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/cost"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockCostEstimationStore implements CostEstimationStore for testing.
type mockCostEstimationStore struct {
	user          *models.User
	repos         []*models.Repository
	repo          *models.Repository
	stats         []*models.StorageStats
	stat          *models.StorageStats
	growth        []*models.StorageGrowthPoint
	pricing       []*models.StoragePricing
	pricingByType *models.StoragePricing
	estimates     []*models.CostEstimateRecord
	alerts        []*models.CostAlert
	alert         *models.CostAlert

	getUserErr       error
	getReposErr      error
	getRepoErr       error
	getStatsErr      error
	getStatErr       error
	getGrowthErr     error
	getPricingErr    error
	createPricingErr error
	updatePricingErr error
	deletePricingErr error
	getEstimatesErr  error
	getAlertsErr     error
	getAlertErr      error
	createAlertErr   error
	updateAlertErr   error
	deleteAlertErr   error
}

func (m *mockCostEstimationStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	return m.user, nil
}

func (m *mockCostEstimationStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Repository, error) {
	if m.getReposErr != nil {
		return nil, m.getReposErr
	}
	return m.repos, nil
}

func (m *mockCostEstimationStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	if m.repo != nil && m.repo.ID == id {
		return m.repo, nil
	}
	return nil, errors.New("not found")
}

func (m *mockCostEstimationStore) GetLatestStorageStats(_ context.Context, _ uuid.UUID) (*models.StorageStats, error) {
	if m.getStatErr != nil {
		return nil, m.getStatErr
	}
	return m.stat, nil
}

func (m *mockCostEstimationStore) GetLatestStatsForAllRepos(_ context.Context, _ uuid.UUID) ([]*models.StorageStats, error) {
	if m.getStatsErr != nil {
		return nil, m.getStatsErr
	}
	return m.stats, nil
}

func (m *mockCostEstimationStore) GetStorageGrowth(_ context.Context, _ uuid.UUID, _ int) ([]*models.StorageGrowthPoint, error) {
	if m.getGrowthErr != nil {
		return nil, m.getGrowthErr
	}
	return m.growth, nil
}

func (m *mockCostEstimationStore) GetAllStorageGrowth(_ context.Context, _ uuid.UUID, _ int) ([]*models.StorageGrowthPoint, error) {
	if m.getGrowthErr != nil {
		return nil, m.getGrowthErr
	}
	return m.growth, nil
}

func (m *mockCostEstimationStore) GetStoragePricingByOrgID(_ context.Context, _ uuid.UUID) ([]*models.StoragePricing, error) {
	if m.getPricingErr != nil {
		return nil, m.getPricingErr
	}
	return m.pricing, nil
}

func (m *mockCostEstimationStore) GetStoragePricingByType(_ context.Context, _ uuid.UUID, _ string) (*models.StoragePricing, error) {
	if m.getPricingErr != nil {
		return nil, m.getPricingErr
	}
	return m.pricingByType, nil
}

func (m *mockCostEstimationStore) CreateStoragePricing(_ context.Context, _ *models.StoragePricing) error {
	return m.createPricingErr
}

func (m *mockCostEstimationStore) UpdateStoragePricing(_ context.Context, _ *models.StoragePricing) error {
	return m.updatePricingErr
}

func (m *mockCostEstimationStore) DeleteStoragePricing(_ context.Context, _ uuid.UUID) error {
	return m.deletePricingErr
}

func (m *mockCostEstimationStore) CreateCostEstimate(_ context.Context, _ *models.CostEstimateRecord) error {
	return nil
}

func (m *mockCostEstimationStore) GetLatestCostEstimates(_ context.Context, _ uuid.UUID) ([]*models.CostEstimateRecord, error) {
	if m.getEstimatesErr != nil {
		return nil, m.getEstimatesErr
	}
	return m.estimates, nil
}

func (m *mockCostEstimationStore) GetCostEstimateHistory(_ context.Context, _ uuid.UUID, _ int) ([]*models.CostEstimateRecord, error) {
	if m.getEstimatesErr != nil {
		return nil, m.getEstimatesErr
	}
	return m.estimates, nil
}

func (m *mockCostEstimationStore) GetCostAlertsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.CostAlert, error) {
	if m.getAlertsErr != nil {
		return nil, m.getAlertsErr
	}
	return m.alerts, nil
}

func (m *mockCostEstimationStore) GetCostAlertByID(_ context.Context, id uuid.UUID) (*models.CostAlert, error) {
	if m.getAlertErr != nil {
		return nil, m.getAlertErr
	}
	if m.alert != nil && m.alert.ID == id {
		return m.alert, nil
	}
	return nil, errors.New("not found")
}

func (m *mockCostEstimationStore) CreateCostAlert(_ context.Context, _ *models.CostAlert) error {
	return m.createAlertErr
}

func (m *mockCostEstimationStore) UpdateCostAlert(_ context.Context, _ *models.CostAlert) error {
	return m.updateAlertErr
}

func (m *mockCostEstimationStore) DeleteCostAlert(_ context.Context, _ uuid.UUID) error {
	return m.deleteAlertErr
}

func setupCostEstimationTestRouter(store CostEstimationStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewCostEstimationHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

// ---------------------------------------------------------------------------
// GetCostSummary tests
// ---------------------------------------------------------------------------

func TestCostEstimationGetCostSummary(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repoID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
			stats: []*models.StorageStats{
				{RepositoryID: repoID, RawDataSize: 1073741824}, // 1 GB
			},
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 536870912},  // 0.5 GB
				{RawDataSize: 1073741824}, // 1 GB
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/summary"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body cost.CostSummary
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.RepositoryCount != 1 {
			t.Fatalf("expected 1 repository, got %d", body.RepositoryCount)
		}
		if body.TotalStorageSizeGB <= 0 {
			t.Fatalf("expected positive storage size, got %f", body.TotalStorageSizeGB)
		}
	})

	t.Run("success no growth data", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
			stats: []*models.StorageStats{
				{RepositoryID: repoID, RawDataSize: 1073741824},
			},
			growth: []*models.StorageGrowthPoint{}, // no growth data
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/summary"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/summary"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("repos error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:        dbUser,
			getReposErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/summary"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("stats error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
			getStatsErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/summary"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/summary"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// ListRepositoryCosts tests
// ---------------------------------------------------------------------------

func TestCostEstimationListRepositoryCosts(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repoID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
			stats: []*models.StorageStats{
				{RepositoryID: repoID, RawDataSize: 2147483648}, // 2 GB
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string][]cost.CostEstimate
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(body["repositories"]) != 1 {
			t.Fatalf("expected 1 repository estimate, got %d", len(body["repositories"]))
		}
	})

	t.Run("success with custom pricing", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
			stats: []*models.StorageStats{
				{RepositoryID: repoID, RawDataSize: 1073741824},
			},
			pricing: []*models.StoragePricing{
				{ID: uuid.New(), OrgID: orgID, RepositoryType: "s3", StoragePerGBMonth: 0.05},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("repos error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:        dbUser,
			getReposErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("stats error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
			getStatsErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GetRepositoryCost tests
// ---------------------------------------------------------------------------

func TestCostEstimationGetRepositoryCost(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3}

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			repo: repo,
			stat: &models.StorageStats{RepositoryID: repoID, RawDataSize: 1073741824},
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 536870912},
				{RawDataSize: 1073741824},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/"+repoID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["estimate"]; !ok {
			t.Fatal("expected 'estimate' in response")
		}
		if _, ok := body["forecasts"]; !ok {
			t.Fatal("expected 'forecasts' in response")
		}
	})

	t.Run("success with custom pricing", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:          dbUser,
			repo:          repo,
			stat:          &models.StorageStats{RepositoryID: repoID, RawDataSize: 1073741824},
			pricingByType: &models.StoragePricing{StoragePerGBMonth: 0.05},
			growth:        []*models.StorageGrowthPoint{{RawDataSize: 500}},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/"+repoID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("success no storage stats", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:       dbUser,
			repo:       repo,
			getStatErr: errors.New("no stats"),
			growth:     []*models.StorageGrowthPoint{},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/"+repoID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/bad-uuid"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/"+repoID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:       dbUser,
			getRepoErr: errors.New("not found"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/"+repoID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherRepo := &models.Repository{ID: repoID, OrgID: uuid.New(), Name: "other", Type: models.RepositoryTypeS3}
		store := &mockCostEstimationStore{
			user: dbUser,
			repo: otherRepo,
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/"+repoID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/repositories/"+repoID.String()))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GetCostForecast tests
// ---------------------------------------------------------------------------

func TestCostEstimationGetCostForecast(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repoID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 536870912},
				{RawDataSize: 1073741824},
			},
			stats: []*models.StorageStats{
				{RepositoryID: repoID, RawDataSize: 1073741824},
			},
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["forecasts"]; !ok {
			t.Fatal("expected 'forecasts' in response")
		}
	})

	t.Run("success with days param", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 536870912},
				{RawDataSize: 1073741824},
			},
			stats: []*models.StorageStats{
				{RepositoryID: repoID, RawDataSize: 1073741824},
			},
			repos: []*models.Repository{
				{ID: repoID, OrgID: orgID, Name: "repo1", Type: models.RepositoryTypeS3},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast?days=60"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("growth error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:         dbUser,
			getGrowthErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("insufficient data", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 1073741824}, // only 1 point
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["message"]; !ok {
			t.Fatal("expected 'message' for insufficient data")
		}
	})

	t.Run("stats error after growth", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 536870912},
				{RawDataSize: 1073741824},
			},
			getStatsErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("repos error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 536870912},
				{RawDataSize: 1073741824},
			},
			stats: []*models.StorageStats{
				{RepositoryID: repoID, RawDataSize: 1073741824},
			},
			getReposErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/forecast"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GetCostHistory tests
// ---------------------------------------------------------------------------

func TestCostEstimationGetCostHistory(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			estimates: []*models.CostEstimateRecord{
				{ID: uuid.New(), OrgID: orgID, MonthlyCost: 10.0},
			},
			growth: []*models.StorageGrowthPoint{
				{RawDataSize: 1073741824},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/history"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["estimates"]; !ok {
			t.Fatal("expected 'estimates' in response")
		}
		if _, ok := body["storage_growth"]; !ok {
			t.Fatal("expected 'storage_growth' in response")
		}
	})

	t.Run("success with days param", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:      dbUser,
			estimates: []*models.CostEstimateRecord{},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/history?days=90"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/history"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("estimates error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:            dbUser,
			getEstimatesErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/history"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/costs/history"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// ListPricing tests
// ---------------------------------------------------------------------------

func TestCostEstimationListPricing(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		pricingID := uuid.New()
		store := &mockCostEstimationStore{
			user: dbUser,
			pricing: []*models.StoragePricing{
				{ID: pricingID, OrgID: orgID, RepositoryType: "s3", StoragePerGBMonth: 0.023},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pricing"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string][]*models.StoragePricing
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(body["pricing"]) != 1 {
			t.Fatalf("expected 1 pricing entry, got %d", len(body["pricing"]))
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pricing"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("pricing error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:          dbUser,
			getPricingErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pricing"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pricing"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GetDefaultPricing tests
// ---------------------------------------------------------------------------

func TestCostEstimationGetDefaultPricing(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pricing/defaults"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string]map[string]cost.StoragePricing
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		defaults := body["defaults"]
		if len(defaults) == 0 {
			t.Fatal("expected non-empty defaults map")
		}
		if _, ok := defaults["s3"]; !ok {
			t.Fatal("expected s3 in defaults")
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/pricing/defaults"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// CreatePricing tests
// ---------------------------------------------------------------------------

func TestCostEstimationCreatePricing(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{user: dbUser}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"repository_type":"s3","storage_per_gb_month":0.05,"egress_per_gb":0.10,"operations_per_k":0.01,"provider_name":"Custom S3","provider_description":"Custom provider"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/pricing", body))

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}

		var pricing models.StoragePricing
		if err := json.Unmarshal(resp.Body.Bytes(), &pricing); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if pricing.RepositoryType != "s3" {
			t.Fatalf("expected repository_type 's3', got %q", pricing.RepositoryType)
		}
		if pricing.OrgID != orgID {
			t.Fatalf("expected org_id %s, got %s", orgID, pricing.OrgID)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockCostEstimationStore{user: dbUser}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05}` // missing required repository_type
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/pricing", body))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"repository_type":"s3","storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/pricing", body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:             dbUser,
			createPricingErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"repository_type":"s3","storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/pricing", body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)

		body := `{"repository_type":"s3","storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/pricing", body))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// UpdatePricing tests
// ---------------------------------------------------------------------------

func TestCostEstimationUpdatePricing(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	pricingID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			pricing: []*models.StoragePricing{
				{ID: pricingID, OrgID: orgID, RepositoryType: "s3", StoragePerGBMonth: 0.023},
			},
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var pricing models.StoragePricing
		if err := json.Unmarshal(resp.Body.Bytes(), &pricing); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if pricing.StoragePerGBMonth != 0.05 {
			t.Fatalf("expected storage_per_gb_month 0.05, got %f", pricing.StoragePerGBMonth)
		}
	})

	t.Run("success with all fields", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			pricing: []*models.StoragePricing{
				{ID: pricingID, OrgID: orgID, RepositoryType: "s3", StoragePerGBMonth: 0.023},
			},
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05,"egress_per_gb":0.10,"operations_per_k":0.01,"provider_name":"Updated","provider_description":"Updated desc"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/bad-uuid", body))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)

		body := `{invalid json}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("pricing not found", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:    dbUser,
			pricing: []*models.StoragePricing{}, // empty, no matching pricing
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("pricing query error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:          dbUser,
			getPricingErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("update error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			pricing: []*models.StoragePricing{
				{ID: pricingID, OrgID: orgID, RepositoryType: "s3", StoragePerGBMonth: 0.023},
			},
			updatePricingErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)

		body := `{"storage_per_gb_month":0.05}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/pricing/"+pricingID.String(), body))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// DeletePricing tests
// ---------------------------------------------------------------------------

func TestCostEstimationDeletePricing(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	pricingID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			pricing: []*models.StoragePricing{
				{ID: pricingID, OrgID: orgID, RepositoryType: "s3"},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/pricing/"+pricingID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/pricing/bad-uuid"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/pricing/"+pricingID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:    dbUser,
			pricing: []*models.StoragePricing{}, // empty
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/pricing/"+pricingID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user: dbUser,
			pricing: []*models.StoragePricing{
				{ID: pricingID, OrgID: orgID, RepositoryType: "s3"},
			},
			deletePricingErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/pricing/"+pricingID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/pricing/"+pricingID.String()))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// ListCostAlerts tests
// ---------------------------------------------------------------------------

func TestCostEstimationListCostAlerts(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		alertID := uuid.New()
		store := &mockCostEstimationStore{
			user: dbUser,
			alerts: []*models.CostAlert{
				{ID: alertID, OrgID: orgID, Name: "Budget Alert", MonthlyThreshold: 100.0, Enabled: true},
			},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string][]*models.CostAlert
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(body["alerts"]) != 1 {
			t.Fatalf("expected 1 alert, got %d", len(body["alerts"]))
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("alerts error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:         dbUser,
			getAlertsErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// GetCostAlert tests
// ---------------------------------------------------------------------------

func TestCostEstimationGetCostAlert(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	alertID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:  dbUser,
			alert: &models.CostAlert{ID: alertID, OrgID: orgID, Name: "Budget Alert", MonthlyThreshold: 100.0},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var alert models.CostAlert
		if err := json.Unmarshal(resp.Body.Bytes(), &alert); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if alert.Name != "Budget Alert" {
			t.Fatalf("expected name 'Budget Alert', got %q", alert.Name)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts/bad-uuid"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("alert not found", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:        dbUser,
			getAlertErr: errors.New("not found"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:  dbUser,
			alert: &models.CostAlert{ID: alertID, OrgID: uuid.New(), Name: "Other Org Alert"},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// CreateCostAlert tests
// ---------------------------------------------------------------------------

func TestCostEstimationCreateCostAlert(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{user: dbUser}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Budget Alert","monthly_threshold":100.0}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/cost-alerts", body))

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}

		var alert models.CostAlert
		if err := json.Unmarshal(resp.Body.Bytes(), &alert); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if alert.Name != "Budget Alert" {
			t.Fatalf("expected name 'Budget Alert', got %q", alert.Name)
		}
		if alert.MonthlyThreshold != 100.0 {
			t.Fatalf("expected threshold 100.0, got %f", alert.MonthlyThreshold)
		}
		if alert.OrgID != orgID {
			t.Fatalf("expected org_id %s, got %s", orgID, alert.OrgID)
		}
		// Defaults from NewCostAlert
		if !alert.Enabled {
			t.Fatal("expected enabled to be true by default")
		}
	})

	t.Run("success with optional fields", func(t *testing.T) {
		store := &mockCostEstimationStore{user: dbUser}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Full Alert","monthly_threshold":200.0,"enabled":false,"notify_on_exceed":false,"notify_on_forecast":true,"forecast_months":6}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/cost-alerts", body))

		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}

		var alert models.CostAlert
		if err := json.Unmarshal(resp.Body.Bytes(), &alert); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if alert.Enabled {
			t.Fatal("expected enabled to be false")
		}
		if alert.NotifyOnExceed {
			t.Fatal("expected notify_on_exceed to be false")
		}
		if !alert.NotifyOnForecast {
			t.Fatal("expected notify_on_forecast to be true")
		}
		if alert.ForecastMonths != 6 {
			t.Fatalf("expected forecast_months 6, got %d", alert.ForecastMonths)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockCostEstimationStore{user: dbUser}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"monthly_threshold":100.0}` // missing required name
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/cost-alerts", body))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Budget Alert","monthly_threshold":100.0}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/cost-alerts", body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("create error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:           dbUser,
			createAlertErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Budget Alert","monthly_threshold":100.0}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/cost-alerts", body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)

		body := `{"name":"Budget Alert","monthly_threshold":100.0}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/cost-alerts", body))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// UpdateCostAlert tests
// ---------------------------------------------------------------------------

func TestCostEstimationUpdateCostAlert(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	alertID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:  dbUser,
			alert: &models.CostAlert{ID: alertID, OrgID: orgID, Name: "Old Name", MonthlyThreshold: 50.0, Enabled: true},
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Updated Name","monthly_threshold":200.0}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var alert models.CostAlert
		if err := json.Unmarshal(resp.Body.Bytes(), &alert); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if alert.Name != "Updated Name" {
			t.Fatalf("expected name 'Updated Name', got %q", alert.Name)
		}
		if alert.MonthlyThreshold != 200.0 {
			t.Fatalf("expected threshold 200.0, got %f", alert.MonthlyThreshold)
		}
	})

	t.Run("success with all fields", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:  dbUser,
			alert: &models.CostAlert{ID: alertID, OrgID: orgID, Name: "Alert", MonthlyThreshold: 50.0, Enabled: true, ForecastMonths: 3},
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Full Update","monthly_threshold":300.0,"enabled":false,"notify_on_exceed":true,"notify_on_forecast":true,"forecast_months":12}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var alert models.CostAlert
		if err := json.Unmarshal(resp.Body.Bytes(), &alert); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if alert.ForecastMonths != 12 {
			t.Fatalf("expected forecast_months 12, got %d", alert.ForecastMonths)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Update"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/bad-uuid", body))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)

		body := `{invalid json}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Update"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("alert not found", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:        dbUser,
			getAlertErr: errors.New("not found"),
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Update"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:  dbUser,
			alert: &models.CostAlert{ID: alertID, OrgID: uuid.New(), Name: "Other Org"},
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Update"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("update error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:           dbUser,
			alert:          &models.CostAlert{ID: alertID, OrgID: orgID, Name: "Alert"},
			updateAlertErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)

		body := `{"name":"Update"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)

		body := `{"name":"Update"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/cost-alerts/"+alertID.String(), body))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteCostAlert tests
// ---------------------------------------------------------------------------

func TestCostEstimationDeleteCostAlert(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	alertID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:  dbUser,
			alert: &models.CostAlert{ID: alertID, OrgID: orgID, Name: "To Delete"},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/cost-alerts/bad-uuid"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockCostEstimationStore{getUserErr: errors.New("not found")}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("alert not found", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:        dbUser,
			getAlertErr: errors.New("not found"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:  dbUser,
			alert: &models.CostAlert{ID: alertID, OrgID: uuid.New(), Name: "Other Org"},
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		store := &mockCostEstimationStore{
			user:           dbUser,
			alert:          &models.CostAlert{ID: alertID, OrgID: orgID, Name: "Alert"},
			deleteAlertErr: errors.New("db error"),
		}
		r := setupCostEstimationTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockCostEstimationStore{}
		r := setupCostEstimationTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/cost-alerts/"+alertID.String()))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}
