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

type mockLegalHoldStore struct {
	user      *models.User
	hold      *models.LegalHold
	holds     []*models.LegalHold
	backup    *models.Backup
	agent     *models.Agent
	statusMap map[string]bool
	onHold    bool
	tier      license.Tier
	err       error
}

func (m *mockLegalHoldStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

func (m *mockLegalHoldStore) CreateLegalHold(_ context.Context, _ *models.LegalHold) error {
	return m.err
}

func (m *mockLegalHoldStore) GetLegalHoldByID(_ context.Context, _ uuid.UUID) (*models.LegalHold, error) {
	return m.hold, m.err
}

func (m *mockLegalHoldStore) GetLegalHoldBySnapshotID(_ context.Context, _ string, _ uuid.UUID) (*models.LegalHold, error) {
	return m.hold, m.err
}

func (m *mockLegalHoldStore) GetLegalHoldsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.LegalHold, error) {
	return m.holds, m.err
}

func (m *mockLegalHoldStore) DeleteLegalHold(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockLegalHoldStore) IsSnapshotOnHold(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
	return m.onHold, m.err
}

func (m *mockLegalHoldStore) GetSnapshotHoldStatus(_ context.Context, _ []string, _ uuid.UUID) (map[string]bool, error) {
	return m.statusMap, m.err
}

func (m *mockLegalHoldStore) GetBackupBySnapshotID(_ context.Context, _ string) (*models.Backup, error) {
	return m.backup, m.err
}

func (m *mockLegalHoldStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	return m.agent, m.err
}

func (m *mockLegalHoldStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return nil
}

// minimal FeatureStore for FeatureChecker
type stubFeatureStore struct {
	tier license.Tier
}

func (s *stubFeatureStore) GetOrgTier(_ context.Context, _ uuid.UUID) (license.Tier, error) {
	return s.tier, nil
}

func (s *stubFeatureStore) SetOrgTier(_ context.Context, _ uuid.UUID, _ license.Tier) error {
	return nil
}

func setupLegalHoldsTestRouter(store LegalHoldStore, user *auth.SessionUser, tier license.Tier) *gin.Engine {
	r := SetupTestRouter(user)
	checker := license.NewFeatureChecker(&stubFeatureStore{tier: tier})
	handler := NewLegalHoldsHandler(store, checker, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func adminUser(orgID uuid.UUID) *auth.SessionUser {
	u := testUser(orgID)
	u.CurrentOrgRole = "admin"
	return u
}

func TestLegalHoldsListLegalHolds(t *testing.T) {
	orgID := uuid.New()
	user := adminUser(orgID)

	t.Run("admin sees holds", func(t *testing.T) {
		store := &mockLegalHoldStore{
			user:  &models.User{ID: user.ID, Role: models.UserRoleAdmin},
			holds: []*models.LegalHold{{ID: uuid.New(), SnapshotID: "snap-1", PlacedBy: user.ID}},
		}
		r := setupLegalHoldsTestRouter(store, user, license.TierEnterprise)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/legal-holds"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		store := &mockLegalHoldStore{
			user: &models.User{ID: user.ID, Role: models.UserRoleViewer},
		}
		r := setupLegalHoldsTestRouter(store, user, license.TierEnterprise)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/legal-holds"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestLegalHoldsCreateLegalHold(t *testing.T) {
	orgID := uuid.New()
	user := adminUser(orgID)

	t.Run("free tier blocked by feature gate", func(t *testing.T) {
		store := &mockLegalHoldStore{}
		r := setupLegalHoldsTestRouter(store, user, license.TierFree)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/snapshots/snap-1/hold", `{"reason":"discovery"}`))
		if resp.Code == http.StatusOK || resp.Code == http.StatusCreated {
			t.Fatalf("expected non-2xx (feature gate), got %d", resp.Code)
		}
	})
}
