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

type mockStorageTiersStore struct {
	configs     []*models.StorageTierConfig
	config      *models.StorageTierConfig
	rules       []*models.TierRule
	rule        *models.TierRule
	tier        *models.SnapshotTier
	tiers       []*models.SnapshotTier
	history     []*models.TierTransition
	cold        *models.ColdRestoreRequest
	colds       []*models.ColdRestoreRequest
	report      *models.TierCostReport
	reports     []*models.TierCostReport
	stats       *models.TierStatsSummary
	listErr     error
	getErr      error
	createErr   error
	updateErr   error
	deleteErr   error
	defaultsErr error
	tierGetErr  error
	historyErr  error
	coldGetErr  error
	statsErr    error
}

func (m *mockStorageTiersStore) GetStorageTierConfigs(_ context.Context, _ uuid.UUID) ([]*models.StorageTierConfig, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.configs, nil
}

func (m *mockStorageTiersStore) GetStorageTierConfig(_ context.Context, _ uuid.UUID) (*models.StorageTierConfig, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.config, nil
}

func (m *mockStorageTiersStore) CreateStorageTierConfig(_ context.Context, _ *models.StorageTierConfig) error {
	return m.createErr
}

func (m *mockStorageTiersStore) UpdateStorageTierConfig(_ context.Context, _ *models.StorageTierConfig) error {
	return m.updateErr
}

func (m *mockStorageTiersStore) CreateDefaultTierConfigs(_ context.Context, _ uuid.UUID) error {
	return m.defaultsErr
}

func (m *mockStorageTiersStore) GetTierRules(_ context.Context, _ uuid.UUID) ([]*models.TierRule, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.rules, nil
}

func (m *mockStorageTiersStore) GetTierRule(_ context.Context, _ uuid.UUID) (*models.TierRule, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.rule, nil
}

func (m *mockStorageTiersStore) CreateTierRule(_ context.Context, _ *models.TierRule) error {
	return m.createErr
}

func (m *mockStorageTiersStore) UpdateTierRule(_ context.Context, _ *models.TierRule) error {
	return m.updateErr
}

func (m *mockStorageTiersStore) DeleteTierRule(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockStorageTiersStore) GetSnapshotTier(_ context.Context, _ string, _ uuid.UUID) (*models.SnapshotTier, error) {
	if m.tierGetErr != nil {
		return nil, m.tierGetErr
	}
	return m.tier, nil
}

func (m *mockStorageTiersStore) GetSnapshotTiersByRepository(_ context.Context, _ uuid.UUID) ([]*models.SnapshotTier, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.tiers, nil
}

func (m *mockStorageTiersStore) GetTierTransitionHistory(_ context.Context, _ string, _ uuid.UUID, _ int) ([]*models.TierTransition, error) {
	if m.historyErr != nil {
		return nil, m.historyErr
	}
	return m.history, nil
}

func (m *mockStorageTiersStore) GetColdRestoreRequest(_ context.Context, _ uuid.UUID) (*models.ColdRestoreRequest, error) {
	if m.coldGetErr != nil {
		return nil, m.coldGetErr
	}
	return m.cold, nil
}

func (m *mockStorageTiersStore) GetActiveColdRestoreRequests(_ context.Context, _ uuid.UUID) ([]*models.ColdRestoreRequest, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.colds, nil
}

func (m *mockStorageTiersStore) GetLatestTierCostReport(_ context.Context, _ uuid.UUID) (*models.TierCostReport, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.report, nil
}

func (m *mockStorageTiersStore) GetTierCostReports(_ context.Context, _ uuid.UUID, _ int) ([]*models.TierCostReport, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.reports, nil
}

func (m *mockStorageTiersStore) GetTierStatsSummary(_ context.Context, _ uuid.UUID) (*models.TierStatsSummary, error) {
	if m.statsErr != nil {
		return nil, m.statsErr
	}
	return m.stats, nil
}

func setupStorageTiersTestRouter(store StorageTiersStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := SetupTestRouter(user)
	// Scheduler is nil — endpoints that require it will return 503.
	handler := NewStorageTiersHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestStorageTiersListConfigs(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns configs", func(t *testing.T) {
		store := &mockStorageTiersStore{configs: []*models.StorageTierConfig{}}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/configs"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/configs"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockStorageTiersStore{listErr: errors.New("db down")}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/configs"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestStorageTiersListRules(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns rules", func(t *testing.T) {
		store := &mockStorageTiersStore{rules: []*models.TierRule{}}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/rules"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/rules"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestStorageTiersCreateRule(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates rule", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, user)
		body := `{"name":"r1","from_tier":"hot","to_tier":"cold","age_threshold_days":30}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/storage-tiers/rules", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid tier returns 400", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, user)
		body := `{"name":"r1","from_tier":"bogus","to_tier":"cold","age_threshold_days":30}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/storage-tiers/rules", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("same from/to returns 400", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, user)
		body := `{"name":"r1","from_tier":"hot","to_tier":"hot","age_threshold_days":30}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/storage-tiers/rules", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestStorageTiersGetRule(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("returns rule", func(t *testing.T) {
		store := &mockStorageTiersStore{rule: &models.TierRule{ID: id, OrgID: orgID}}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/rules/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("wrong org returns 404", func(t *testing.T) {
		store := &mockStorageTiersStore{rule: &models.TierRule{ID: id, OrgID: uuid.New()}}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/rules/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/rules/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestStorageTiersDeleteRule(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("deletes rule", func(t *testing.T) {
		store := &mockStorageTiersStore{rule: &models.TierRule{ID: id, OrgID: orgID}}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/storage-tiers/rules/"+id.String()))
		if resp.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("wrong org returns 404", func(t *testing.T) {
		store := &mockStorageTiersStore{rule: &models.TierRule{ID: id, OrgID: uuid.New()}}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/storage-tiers/rules/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestStorageTiersTransitionSnapshot(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("nil scheduler returns 503", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, user)
		body := `{"to_tier":"cold"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/storage-tiers/snapshots/snap1/repository/"+uuid.New().String()+"/transition", body))
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d: %s", resp.Code, resp.Body.String())
		}
	})
}

func TestStorageTiersRequestColdRestore(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("nil scheduler returns 503", func(t *testing.T) {
		store := &mockStorageTiersStore{}
		r := setupStorageTiersTestRouter(store, user)
		body := `{"snapshot_id":"s1","repository_id":"` + uuid.New().String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/storage-tiers/cold-restore", body))
		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", resp.Code)
		}
	})
}

func TestStorageTiersGetTierStats(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns stats", func(t *testing.T) {
		store := &mockStorageTiersStore{stats: &models.TierStatsSummary{}}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/stats"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockStorageTiersStore{statsErr: errors.New("db down")}
		r := setupStorageTiersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/storage-tiers/stats"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
