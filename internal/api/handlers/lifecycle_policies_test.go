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

type mockLifecyclePolicyStore struct {
	user           *models.User
	policy         *models.LifecyclePolicy
	policies       []*models.LifecyclePolicy
	events         []*models.LifecycleDeletionEvent
	backups        []*models.Backup
	classification *models.BackupClassification
	schedule       *models.Schedule
	onHold         bool
	err            error
}

func (m *mockLifecyclePolicyStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

func (m *mockLifecyclePolicyStore) CreateLifecyclePolicy(_ context.Context, _ *models.LifecyclePolicy) error {
	return m.err
}

func (m *mockLifecyclePolicyStore) GetLifecyclePolicyByID(_ context.Context, _ uuid.UUID) (*models.LifecyclePolicy, error) {
	return m.policy, m.err
}

func (m *mockLifecyclePolicyStore) GetLifecyclePoliciesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.LifecyclePolicy, error) {
	return m.policies, m.err
}

func (m *mockLifecyclePolicyStore) GetActiveLifecyclePoliciesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.LifecyclePolicy, error) {
	return m.policies, m.err
}

func (m *mockLifecyclePolicyStore) UpdateLifecyclePolicy(_ context.Context, _ *models.LifecyclePolicy) error {
	return m.err
}

func (m *mockLifecyclePolicyStore) DeleteLifecyclePolicy(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockLifecyclePolicyStore) CreateLifecycleDeletionEvent(_ context.Context, _ *models.LifecycleDeletionEvent) error {
	return m.err
}

func (m *mockLifecyclePolicyStore) GetLifecycleDeletionEventsByPolicyID(_ context.Context, _ uuid.UUID, _ int) ([]*models.LifecycleDeletionEvent, error) {
	return m.events, m.err
}

func (m *mockLifecyclePolicyStore) GetLifecycleDeletionEventsByOrgID(_ context.Context, _ uuid.UUID, _ int) ([]*models.LifecycleDeletionEvent, error) {
	return m.events, m.err
}

func (m *mockLifecyclePolicyStore) GetBackupsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Backup, error) {
	return m.backups, m.err
}

func (m *mockLifecyclePolicyStore) GetBackupClassification(_ context.Context, _ uuid.UUID) (*models.BackupClassification, error) {
	return m.classification, m.err
}

func (m *mockLifecyclePolicyStore) IsSnapshotOnHold(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
	return m.onHold, m.err
}

func (m *mockLifecyclePolicyStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	return m.schedule, m.err
}

func (m *mockLifecyclePolicyStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return nil
}

func setupLifecyclePoliciesTestRouter(store LifecyclePolicyStore, user *auth.SessionUser, tier license.Tier) *gin.Engine {
	r := SetupTestRouter(user)
	checker := license.NewFeatureChecker(&stubFeatureStore{tier: tier})
	handler := NewLifecyclePoliciesHandler(store, checker, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestLifecyclePoliciesListLifecyclePolicies(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockLifecyclePolicyStore{
		user:     &models.User{ID: user.ID, Role: models.UserRoleAdmin},
		policies: []*models.LifecyclePolicy{},
	}
	r := setupLifecyclePoliciesTestRouter(store, user, license.TierEnterprise)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/lifecycle-policies"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestLifecyclePoliciesGetLifecyclePolicy(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockLifecyclePolicyStore{user: &models.User{ID: user.ID, Role: models.UserRoleAdmin}}
		r := setupLifecyclePoliciesTestRouter(store, user, license.TierEnterprise)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/lifecycle-policies/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
