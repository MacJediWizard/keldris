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

type mockSuperuserStore struct {
	user        *models.User
	users       []*models.User
	superusers  []*models.User
	orgs        []*models.Organization
	org         *models.Organization
	memberships []*models.OrgMembership
	setting     *models.SystemSetting
	settings    []*models.SystemSetting
	auditLogs   []*models.SuperuserAuditLogWithUser
	err         error
}

func (m *mockSuperuserStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

func (m *mockSuperuserStore) GetUserByEmail(_ context.Context, _ string) (*models.User, error) {
	return m.user, m.err
}

func (m *mockSuperuserStore) GetAllOrganizations(_ context.Context) ([]*models.Organization, error) {
	return m.orgs, m.err
}

func (m *mockSuperuserStore) GetAllUsers(_ context.Context) ([]*models.User, error) {
	return m.users, m.err
}

func (m *mockSuperuserStore) SetUserSuperuser(_ context.Context, _ uuid.UUID, _ bool) error {
	return m.err
}

func (m *mockSuperuserStore) GetSuperusers(_ context.Context) ([]*models.User, error) {
	return m.superusers, m.err
}

func (m *mockSuperuserStore) CreateSuperuserAuditLog(_ context.Context, _ *models.SuperuserAuditLog) error {
	return nil
}

func (m *mockSuperuserStore) GetSuperuserAuditLogs(_ context.Context, _, _ int) ([]*models.SuperuserAuditLogWithUser, int, error) {
	return m.auditLogs, len(m.auditLogs), m.err
}

func (m *mockSuperuserStore) GetSystemSetting(_ context.Context, _ string) (*models.SystemSetting, error) {
	return m.setting, m.err
}

func (m *mockSuperuserStore) GetSystemSettings(_ context.Context) ([]*models.SystemSetting, error) {
	return m.settings, m.err
}

func (m *mockSuperuserStore) UpdateSystemSetting(_ context.Context, _ string, _ interface{}, _ uuid.UUID) error {
	return m.err
}

func (m *mockSuperuserStore) GetMembershipsByUserID(_ context.Context, _ uuid.UUID) ([]*models.OrgMembership, error) {
	return m.memberships, m.err
}

func (m *mockSuperuserStore) GetOrganizationByID(_ context.Context, _ uuid.UUID) (*models.Organization, error) {
	return m.org, m.err
}

// setupSuperuserTestRouter wires the handler methods directly (bypassing SuperuserMiddleware
// which requires a real SessionStore). The handler's RequireSuperuser still enforces the check
// against the injected SessionUser.
func setupSuperuserTestRouter(store SuperuserStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewSuperuserHandler(store, nil, zerolog.Nop())
	r.GET("/api/v1/superuser/organizations", handler.ListAllOrganizations)
	r.GET("/api/v1/superuser/organizations/:id", handler.GetOrganization)
	r.GET("/api/v1/superuser/users", handler.ListAllUsers)
	r.GET("/api/v1/superuser/superusers", handler.ListSuperusers)
	r.GET("/api/v1/superuser/settings", handler.GetSystemSettings)
	r.GET("/api/v1/superuser/audit-logs", handler.GetAuditLogs)
	return r
}

func TestSuperuserListAllOrganizations(t *testing.T) {
	t.Run("superuser sees all orgs", func(t *testing.T) {
		store := &mockSuperuserStore{orgs: []*models.Organization{{ID: uuid.New(), Name: "Acme"}}}
		r := setupSuperuserTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/organizations"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-superuser forbidden", func(t *testing.T) {
		store := &mockSuperuserStore{}
		r := setupSuperuserTestRouter(store, testUser(uuid.New()))

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/organizations"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestSuperuserGetOrganization(t *testing.T) {
	orgID := uuid.New()
	t.Run("returns org", func(t *testing.T) {
		store := &mockSuperuserStore{org: &models.Organization{ID: orgID, Name: "Acme"}}
		r := setupSuperuserTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/organizations/"+orgID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockSuperuserStore{}
		r := setupSuperuserTestRouter(store, superuserTestUser())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/organizations/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestSuperuserGetSystemSettings(t *testing.T) {
	store := &mockSuperuserStore{settings: []*models.SystemSetting{{Key: "telemetry_enabled"}}}
	r := setupSuperuserTestRouter(store, superuserTestUser())

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/settings"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestSuperuserGetAuditLogs(t *testing.T) {
	store := &mockSuperuserStore{}
	r := setupSuperuserTestRouter(store, superuserTestUser())

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/superuser/audit-logs"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
