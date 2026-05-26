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

type mockRateLimitConfigStore struct {
	configs []*models.RateLimitConfig
	config  *models.RateLimitConfig
	stats   *models.RateLimitStats
	blocked []*models.BlockedRequest
	bans    []*models.IPBan
	err     error
}

func (m *mockRateLimitConfigStore) ListRateLimitConfigs(_ context.Context, _ uuid.UUID) ([]*models.RateLimitConfig, error) {
	return m.configs, m.err
}

func (m *mockRateLimitConfigStore) GetRateLimitConfigByID(_ context.Context, _ uuid.UUID) (*models.RateLimitConfig, error) {
	return m.config, m.err
}

func (m *mockRateLimitConfigStore) GetRateLimitConfigByEndpoint(_ context.Context, _ uuid.UUID, _ string) (*models.RateLimitConfig, error) {
	return m.config, m.err
}

func (m *mockRateLimitConfigStore) CreateRateLimitConfig(_ context.Context, _ *models.RateLimitConfig) error {
	return m.err
}

func (m *mockRateLimitConfigStore) UpdateRateLimitConfig(_ context.Context, _ *models.RateLimitConfig) error {
	return m.err
}

func (m *mockRateLimitConfigStore) DeleteRateLimitConfig(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockRateLimitConfigStore) GetRateLimitStats(_ context.Context, _ uuid.UUID) (*models.RateLimitStats, error) {
	return m.stats, m.err
}

func (m *mockRateLimitConfigStore) ListRecentBlockedRequests(_ context.Context, _ uuid.UUID, _ int) ([]*models.BlockedRequest, error) {
	return m.blocked, m.err
}

func (m *mockRateLimitConfigStore) ListIPBans(_ context.Context, _ uuid.UUID) ([]*models.IPBan, error) {
	return m.bans, m.err
}

func (m *mockRateLimitConfigStore) ListActiveIPBans(_ context.Context, _ uuid.UUID) ([]*models.IPBan, error) {
	return m.bans, m.err
}

func (m *mockRateLimitConfigStore) CreateIPBan(_ context.Context, _ *models.IPBan) error {
	return m.err
}

func (m *mockRateLimitConfigStore) DeleteIPBan(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func setupRateLimitsTestRouter(store RateLimitConfigStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewRateLimitsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestRateLimitsList(t *testing.T) {
	orgID := uuid.New()

	t.Run("admin sees configs", func(t *testing.T) {
		store := &mockRateLimitConfigStore{configs: []*models.RateLimitConfig{}}
		r := setupRateLimitsTestRouter(store, adminUser(orgID))

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/rate-limit-configs"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		viewer := testUser(orgID)
		viewer.CurrentOrgRole = "viewer"
		store := &mockRateLimitConfigStore{}
		r := setupRateLimitsTestRouter(store, viewer)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/rate-limit-configs"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestRateLimitsGetStats(t *testing.T) {
	orgID := uuid.New()
	store := &mockRateLimitConfigStore{stats: &models.RateLimitStats{}}
	r := setupRateLimitsTestRouter(store, adminUser(orgID))

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/rate-limit-configs/stats"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestRateLimitsListBans(t *testing.T) {
	orgID := uuid.New()
	store := &mockRateLimitConfigStore{bans: []*models.IPBan{}}
	r := setupRateLimitsTestRouter(store, adminUser(orgID))

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/admin/ip-bans"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
