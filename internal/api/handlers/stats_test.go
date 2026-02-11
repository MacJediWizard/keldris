package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockStatsStore struct {
	user         *models.User
	repos        []*models.Repository
	repo         *models.Repository
	summary      *models.StorageStatsSummary
	stats        []*models.StorageStats
	latestStats  *models.StorageStats
	growth       []*models.StorageGrowthPoint
	getUserErr   error
	getRepoErr   error
	getReposErr  error
	summaryErr   error
	growthErr    error
	statsErr     error
	latestErr    error
}

func (m *mockStatsStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	if m.getUserErr != nil {
		return nil, m.getUserErr
	}
	return m.user, nil
}

func (m *mockStatsStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	if m.repo != nil && m.repo.ID == id {
		return m.repo, nil
	}
	return nil, errors.New("not found")
}

func (m *mockStatsStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Repository, error) {
	if m.getReposErr != nil {
		return nil, m.getReposErr
	}
	return m.repos, nil
}

func (m *mockStatsStore) GetLatestStorageStats(_ context.Context, _ uuid.UUID) (*models.StorageStats, error) {
	if m.latestErr != nil {
		return nil, m.latestErr
	}
	return m.latestStats, nil
}

func (m *mockStatsStore) GetStorageStatsByRepositoryID(_ context.Context, _ uuid.UUID, _ int) ([]*models.StorageStats, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.stats, nil
}

func (m *mockStatsStore) GetStorageStatsSummary(_ context.Context, _ uuid.UUID) (*models.StorageStatsSummary, error) {
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	return m.summary, nil
}

func (m *mockStatsStore) GetStorageGrowth(_ context.Context, _ uuid.UUID, _ int) ([]*models.StorageGrowthPoint, error) {
	if m.growthErr != nil {
		return nil, m.growthErr
	}
	return m.growth, nil
}

func (m *mockStatsStore) GetAllStorageGrowth(_ context.Context, _ uuid.UUID, _ int) ([]*models.StorageGrowthPoint, error) {
	if m.growthErr != nil {
		return nil, m.growthErr
	}
	return m.growth, nil
}

func (m *mockStatsStore) GetLatestStatsForAllRepos(_ context.Context, _ uuid.UUID) ([]*models.StorageStats, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.stats, nil
}

func setupStatsTestRouter(store StatsStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewStatsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestStatsGetSummary(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockStatsStore{
			user:    dbUser,
			summary: &models.StorageStatsSummary{TotalRawSize: 1024},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/summary"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("user not found", func(t *testing.T) {
		store := &mockStatsStore{getUserErr: errors.New("not found")}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/summary"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("summary error", func(t *testing.T) {
		store := &mockStatsStore{user: dbUser, summaryErr: errors.New("db error")}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/summary"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockStatsStore{}
		r := setupStatsTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/summary"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestStatsGetGrowth(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}

	t.Run("success default days", func(t *testing.T) {
		store := &mockStatsStore{
			user:   dbUser,
			growth: []*models.StorageGrowthPoint{},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/growth"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("custom days", func(t *testing.T) {
		store := &mockStatsStore{
			user:   dbUser,
			growth: []*models.StorageGrowthPoint{},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/growth?days=7"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("growth error", func(t *testing.T) {
		store := &mockStatsStore{user: dbUser, growthErr: errors.New("db error")}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/growth"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestStatsListRepositoryStats(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repo := &models.Repository{ID: uuid.New(), OrgID: orgID, Name: "repo1"}

	t.Run("success", func(t *testing.T) {
		store := &mockStatsStore{
			user:  dbUser,
			stats: []*models.StorageStats{{RepositoryID: repo.ID, TotalSize: 1024}},
			repos: []*models.Repository{repo},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["stats"]; !ok {
			t.Fatal("expected 'stats' key")
		}
	})

	t.Run("stats error", func(t *testing.T) {
		store := &mockStatsStore{user: dbUser, statsErr: errors.New("db error")}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("repos error", func(t *testing.T) {
		store := &mockStatsStore{
			user:        dbUser,
			stats:       []*models.StorageStats{},
			getReposErr: errors.New("db error"),
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestStatsGetRepositoryStats(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "repo1"}

	t.Run("success", func(t *testing.T) {
		store := &mockStatsStore{
			user:        dbUser,
			repo:        repo,
			latestStats: &models.StorageStats{RepositoryID: repoID, TotalSize: 2048},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+repoID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockStatsStore{user: dbUser}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/bad-id"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("repo not found", func(t *testing.T) {
		store := &mockStatsStore{user: dbUser, getRepoErr: errors.New("not found")}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherRepo := &models.Repository{ID: uuid.New(), OrgID: uuid.New(), Name: "other"}
		store := &mockStatsStore{user: dbUser, repo: otherRepo}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+otherRepo.ID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("no stats yet", func(t *testing.T) {
		store := &mockStatsStore{
			user:      dbUser,
			repo:      repo,
			latestErr: errors.New("no stats"),
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+repoID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200 with message, got %d", resp.Code)
		}
	})
}

func TestStatsGetRepositoryGrowth(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "repo1"}

	t.Run("success", func(t *testing.T) {
		store := &mockStatsStore{
			user:   dbUser,
			repo:   repo,
			growth: []*models.StorageGrowthPoint{},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+repoID.String()+"/growth"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("growth error", func(t *testing.T) {
		store := &mockStatsStore{
			user:      dbUser,
			repo:      repo,
			growthErr: errors.New("db error"),
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+repoID.String()+"/growth"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestStatsGetRepositoryHistory(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	dbUser := &models.User{ID: user.ID, OrgID: orgID}
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "repo1"}

	t.Run("success", func(t *testing.T) {
		store := &mockStatsStore{
			user:  dbUser,
			repo:  repo,
			stats: []*models.StorageStats{},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+repoID.String()+"/history"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("custom limit", func(t *testing.T) {
		store := &mockStatsStore{
			user:  dbUser,
			repo:  repo,
			stats: []*models.StorageStats{},
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+repoID.String()+"/history?limit=10"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("history error", func(t *testing.T) {
		store := &mockStatsStore{
			user:     dbUser,
			repo:     repo,
			statsErr: errors.New("db error"),
		}
		r := setupStatsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/stats/repositories/"+repoID.String()+"/history"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
