package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockGeoReplicationStore struct {
	configs        []*models.GeoReplicationConfig
	config         *models.GeoReplicationConfig
	configByRepo   *models.GeoReplicationConfig
	events         []*models.ReplicationEvent
	repo           *models.Repository
	repoErr        error
	configErr      error
	configByRepErr error
	listErr        error
	createErr      error
	updateErr      error
	deleteErr      error
	eventsErr      error
	updateRepoErr  error
	lagSnapshots   int
	lagSyncAt      *time.Time
	lagErr         error
}

func (m *mockGeoReplicationStore) CreateGeoReplicationConfig(_ context.Context, _ *models.GeoReplicationConfig) error {
	return m.createErr
}

func (m *mockGeoReplicationStore) GetGeoReplicationConfig(_ context.Context, _ uuid.UUID) (*models.GeoReplicationConfig, error) {
	if m.configErr != nil {
		return nil, m.configErr
	}
	return m.config, nil
}

func (m *mockGeoReplicationStore) GetGeoReplicationConfigByRepository(_ context.Context, _ uuid.UUID) (*models.GeoReplicationConfig, error) {
	if m.configByRepErr != nil {
		return nil, m.configByRepErr
	}
	return m.configByRepo, nil
}

func (m *mockGeoReplicationStore) UpdateGeoReplicationConfig(_ context.Context, _ *models.GeoReplicationConfig) error {
	return m.updateErr
}

func (m *mockGeoReplicationStore) DeleteGeoReplicationConfig(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockGeoReplicationStore) ListGeoReplicationConfigsByOrg(_ context.Context, _ uuid.UUID) ([]*models.GeoReplicationConfig, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.configs, nil
}

func (m *mockGeoReplicationStore) GetReplicationEvents(_ context.Context, _ uuid.UUID, _ int) ([]*models.ReplicationEvent, error) {
	if m.eventsErr != nil {
		return nil, m.eventsErr
	}
	return m.events, nil
}

func (m *mockGeoReplicationStore) GetReplicationLagForConfig(_ context.Context, _ uuid.UUID) (int, *time.Time, error) {
	return m.lagSnapshots, m.lagSyncAt, m.lagErr
}

func (m *mockGeoReplicationStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	if m.repoErr != nil {
		return nil, m.repoErr
	}
	return m.repo, nil
}

func (m *mockGeoReplicationStore) UpdateRepositoryRegion(_ context.Context, _ uuid.UUID, _ string) error {
	return m.updateRepoErr
}

func setupGeoReplicationTestRouter(store GeoReplicationStore, user *auth.SessionUser, checker *license.FeatureChecker) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewGeoReplicationHandler(store, checker, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestGeoReplicationList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns configs", func(t *testing.T) {
		store := &mockGeoReplicationStore{configs: []*models.GeoReplicationConfig{
			models.NewGeoReplicationConfig(orgID, uuid.New(), uuid.New(), "us-east-1", "eu-west-1"),
		}}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/configs"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockGeoReplicationStore{listErr: errors.New("db down")}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/configs"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockGeoReplicationStore{}
		r := setupGeoReplicationTestRouter(store, testUserNoOrg(), nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/configs"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestGeoReplicationCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid region returns 400", func(t *testing.T) {
		store := &mockGeoReplicationStore{}
		r := setupGeoReplicationTestRouter(store, user, nil)
		body := `{"source_repository_id":"` + uuid.New().String() + `","target_repository_id":"` + uuid.New().String() + `","source_region":"bogus","target_region":"bogus"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/geo-replication/configs", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		store := &mockGeoReplicationStore{}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/geo-replication/configs", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("feature flag blocks free tier", func(t *testing.T) {
		store := &mockGeoReplicationStore{}
		// Real FeatureChecker with TierFree blocks FeatureGeoReplication (Enterprise).
		checker := license.NewFeatureChecker(&mockLicenseStore{tier: license.TierFree})
		r := setupGeoReplicationTestRouter(store, user, checker)
		body := `{"source_repository_id":"` + uuid.New().String() + `","target_repository_id":"` + uuid.New().String() + `","source_region":"us-east-1","target_region":"eu-west-1"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/geo-replication/configs", body))
		if resp.Code != http.StatusPaymentRequired {
			t.Fatalf("expected 402 (feature gated), got %d: %s", resp.Code, resp.Body.String())
		}
	})
}

func TestGeoReplicationGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	cfgID := uuid.New()

	t.Run("returns own config", func(t *testing.T) {
		cfg := models.NewGeoReplicationConfig(orgID, uuid.New(), uuid.New(), "us-east-1", "eu-west-1")
		cfg.ID = cfgID
		store := &mockGeoReplicationStore{config: cfg}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/configs/"+cfgID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		cfg := models.NewGeoReplicationConfig(uuid.New(), uuid.New(), uuid.New(), "us-east-1", "eu-west-1")
		cfg.ID = cfgID
		store := &mockGeoReplicationStore{config: cfg}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/configs/"+cfgID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockGeoReplicationStore{}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/configs/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestGeoReplicationDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	cfgID := uuid.New()

	t.Run("deletes own config", func(t *testing.T) {
		cfg := models.NewGeoReplicationConfig(orgID, uuid.New(), uuid.New(), "us-east-1", "eu-west-1")
		cfg.ID = cfgID
		store := &mockGeoReplicationStore{config: cfg}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/geo-replication/configs/"+cfgID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		cfg := models.NewGeoReplicationConfig(uuid.New(), uuid.New(), uuid.New(), "us-east-1", "eu-west-1")
		cfg.ID = cfgID
		store := &mockGeoReplicationStore{config: cfg}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/geo-replication/configs/"+cfgID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestGeoReplicationListRegions(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns regions", func(t *testing.T) {
		store := &mockGeoReplicationStore{}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/regions"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})
}

func TestGeoReplicationGetSummary(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns summary", func(t *testing.T) {
		store := &mockGeoReplicationStore{configs: []*models.GeoReplicationConfig{
			models.NewGeoReplicationConfig(orgID, uuid.New(), uuid.New(), "us-east-1", "eu-west-1"),
		}}
		r := setupGeoReplicationTestRouter(store, user, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/summary"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockGeoReplicationStore{}
		r := setupGeoReplicationTestRouter(store, testUserNoOrg(), nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/geo-replication/summary"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
