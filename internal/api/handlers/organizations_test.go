package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockOrgStore struct {
	orgs            []*models.Organization
	orgByID         map[uuid.UUID]*models.Organization
	orgBySlug       map[string]*models.Organization
	memberships     map[string]*models.OrgMembership // key: "userID|orgID"
	membersByOrg    map[uuid.UUID][]*models.OrgMembershipWithUser
	membersByUser   []*models.OrgMembership
	invitations     []*models.OrgInvitationWithDetails
	invByToken      map[string]*models.OrgInvitation
	user            *models.User
	createOrgErr    error
	updateOrgErr    error
	deleteOrgErr    error
	createMemberErr error
	updateMemberErr error
	deleteMemberErr error
	createInvErr    error
	deleteInvErr    error
	acceptInvErr    error
	listErr         error
}

func membershipKey(userID, orgID uuid.UUID) string {
	return userID.String() + "|" + orgID.String()
}

func (m *mockOrgStore) GetOrganizationByID(_ context.Context, id uuid.UUID) (*models.Organization, error) {
	if o, ok := m.orgByID[id]; ok {
		return o, nil
	}
	return nil, errors.New("organization not found")
}

func (m *mockOrgStore) GetOrganizationBySlug(_ context.Context, slug string) (*models.Organization, error) {
	if o, ok := m.orgBySlug[slug]; ok {
		return o, nil
	}
	return nil, errors.New("organization not found")
}

func (m *mockOrgStore) CreateOrganization(_ context.Context, _ *models.Organization) error {
	return m.createOrgErr
}

func (m *mockOrgStore) UpdateOrganization(_ context.Context, _ *models.Organization) error {
	return m.updateOrgErr
}

func (m *mockOrgStore) DeleteOrganization(_ context.Context, _ uuid.UUID) error {
	return m.deleteOrgErr
}

func (m *mockOrgStore) GetUserOrganizations(_ context.Context, _ uuid.UUID) ([]*models.Organization, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.orgs, nil
}

func (m *mockOrgStore) GetMembershipByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	key := membershipKey(userID, orgID)
	if mem, ok := m.memberships[key]; ok {
		return mem, nil
	}
	return nil, errors.New("membership not found")
}

func (m *mockOrgStore) GetMembershipsByUserID(_ context.Context, _ uuid.UUID) ([]*models.OrgMembership, error) {
	return m.membersByUser, nil
}

func (m *mockOrgStore) GetMembershipsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.OrgMembershipWithUser, error) {
	if members, ok := m.membersByOrg[orgID]; ok {
		return members, nil
	}
	return nil, nil
}

func (m *mockOrgStore) CreateMembership(_ context.Context, _ *models.OrgMembership) error {
	return m.createMemberErr
}

func (m *mockOrgStore) UpdateMembership(_ context.Context, _ *models.OrgMembership) error {
	return m.updateMemberErr
}

func (m *mockOrgStore) DeleteMembership(_ context.Context, _, _ uuid.UUID) error {
	return m.deleteMemberErr
}

func (m *mockOrgStore) CreateInvitation(_ context.Context, _ *models.OrgInvitation) error {
	return m.createInvErr
}

func (m *mockOrgStore) GetInvitationByToken(_ context.Context, token string) (*models.OrgInvitation, error) {
	if inv, ok := m.invByToken[token]; ok {
		return inv, nil
	}
	return nil, errors.New("invitation not found")
}

func (m *mockOrgStore) GetPendingInvitationsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.OrgInvitationWithDetails, error) {
	return m.invitations, nil
}

func (m *mockOrgStore) GetPendingInvitationsByEmail(_ context.Context, _ string) ([]*models.OrgInvitationWithDetails, error) {
	return nil, nil
}

func (m *mockOrgStore) AcceptInvitation(_ context.Context, _ uuid.UUID) error {
	return m.acceptInvErr
}

func (m *mockOrgStore) DeleteInvitation(_ context.Context, _ uuid.UUID) error {
	return m.deleteInvErr
}

func (m *mockOrgStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func setupOrgTestRouter(store *mockOrgStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	rbac := auth.NewRBAC(store)
	handler := NewOrganizationsHandler(store, nil, rbac, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListOrganizations(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test-org"}

	store := &mockOrgStore{
		orgs:    []*models.Organization{org},
		orgByID: map[uuid.UUID]*models.Organization{orgID: org},
		memberships: map[string]*models.OrgMembership{
			membershipKey(userID, orgID): {ID: uuid.New(), UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
		},
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["organizations"]; !ok {
			t.Fatal("expected 'organizations' key")
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupOrgTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockOrgStore{
			orgs:    []*models.Organization{org},
			orgByID: map[uuid.UUID]*models.Organization{orgID: org},
			memberships: map[string]*models.OrgMembership{
				membershipKey(userID, orgID): {ID: uuid.New(), UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
			},
			listErr: errors.New("db error"),
		}
		r := setupOrgTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestCreateOrganization(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()

	store := &mockOrgStore{
		orgByID:   map[uuid.UUID]*models.Organization{},
		orgBySlug: map[string]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(userID, orgID): {ID: uuid.New(), UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
		},
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"New Org","slug":"neworg"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("slug exists", func(t *testing.T) {
		existingOrg := &models.Organization{ID: uuid.New(), Name: "Existing", Slug: "existing"}
		slugStore := &mockOrgStore{
			orgByID:     map[uuid.UUID]*models.Organization{},
			orgBySlug:   map[string]*models.Organization{"existing": existingOrg},
			memberships: map[string]*models.OrgMembership{},
		}
		r := setupOrgTestRouter(slugStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"Another","slug":"existing"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupOrgTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"Fail","slug":"fail"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockOrgStore{
			orgByID:   map[uuid.UUID]*models.Organization{},
			orgBySlug: map[string]*models.Organization{},
			memberships: map[string]*models.OrgMembership{
				membershipKey(userID, orgID): {ID: uuid.New(), UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
			},
			createOrgErr: errors.New("db error"),
		}
		r := setupOrgTestRouter(errStore, user)
		w := httptest.NewRecorder()
		body := `{"name":"FailOrg","slug":"failorg"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestGetCurrentOrganization(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Current Org", Slug: "current"}

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{orgID: org},
		memberships: map[string]*models.OrgMembership{
			membershipKey(userID, orgID): {ID: uuid.New(), UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
		},
	}

	t.Run("success", func(t *testing.T) {
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID, CurrentOrgRole: "owner"}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/current", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: userID}
		r := setupOrgTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/current", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestGetOrganization(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Test Org", Slug: "test"}

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{orgID: org},
		memberships: map[string]*models.OrgMembership{
			membershipKey(userID, orgID): {ID: uuid.New(), UserID: userID, OrgID: orgID, Role: models.OrgRoleMember},
		},
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not a member", func(t *testing.T) {
		otherUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupOrgTestRouter(store, otherUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})
}

func TestUpdateOrganization(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()
	org := &models.Organization{ID: orgID, Name: "Old Name", Slug: "oldslug"}

	store := &mockOrgStore{
		orgByID:   map[uuid.UUID]*models.Organization{orgID: org},
		orgBySlug: map[string]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID):  {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
			membershipKey(memberID, orgID): {ID: uuid.New(), UserID: memberID, OrgID: orgID, Role: models.OrgRoleMember},
		},
	}

	t.Run("success as owner", func(t *testing.T) {
		ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		body := `{"name":"New Name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied for member", func(t *testing.T) {
		memberUser := &auth.SessionUser{ID: memberID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, memberUser)
		w := httptest.NewRecorder()
		body := `{"name":"Should Fail"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		body := `{"name":"Test"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		missingOrgID := uuid.New()
		ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		notFoundStore := &mockOrgStore{
			orgByID:   map[uuid.UUID]*models.Organization{orgID: org},
			orgBySlug: map[string]*models.Organization{},
			memberships: map[string]*models.OrgMembership{
				membershipKey(ownerID, orgID):        {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
				membershipKey(ownerID, missingOrgID): {ID: uuid.New(), UserID: ownerID, OrgID: missingOrgID, Role: models.OrgRoleOwner},
			},
		}
		r := setupOrgTestRouter(notFoundStore, ownerUser)
		w := httptest.NewRecorder()
		body := `{"name":"Test"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+missingOrgID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("store error", func(t *testing.T) {
		ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		errStore := &mockOrgStore{
			orgByID:   map[uuid.UUID]*models.Organization{orgID: org},
			orgBySlug: map[string]*models.Organization{},
			memberships: map[string]*models.OrgMembership{
				membershipKey(ownerID, orgID): {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
			},
			updateOrgErr: errors.New("db error"),
		}
		r := setupOrgTestRouter(errStore, ownerUser)
		w := httptest.NewRecorder()
		body := `{"name":"Updated"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestDeleteOrganization(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	org := &models.Organization{ID: orgID, Name: "Delete Me", Slug: "deleteme"}

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{orgID: org},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID):  {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
			membershipKey(memberID, orgID): {ID: uuid.New(), UserID: memberID, OrgID: orgID, Role: models.OrgRoleMember},
		},
	}

	t.Run("success as owner", func(t *testing.T) {
		ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied for member", func(t *testing.T) {
		memberUser := &auth.SessionUser{ID: memberID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, memberUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockOrgStore{
			orgByID: map[uuid.UUID]*models.Organization{orgID: org},
			memberships: map[string]*models.OrgMembership{
				membershipKey(ownerID, orgID): {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
			},
			deleteOrgErr: errors.New("db error"),
		}
		ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(errStore, ownerUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestListMembers(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID): {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
		},
		membersByOrg: map[uuid.UUID][]*models.OrgMembershipWithUser{
			orgID: {{ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner, Email: "owner@test.com"}},
		},
	}

	t.Run("success", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/members", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		nonMember := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupOrgTestRouter(store, nonMember)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/members", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/bad-uuid/members", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestRemoveMember(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID):  {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
			membershipKey(memberID, orgID): {ID: uuid.New(), UserID: memberID, OrgID: orgID, Role: models.OrgRoleMember},
		},
		membersByOrg: map[uuid.UUID][]*models.OrgMembershipWithUser{
			orgID: {
				{ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
				{ID: uuid.New(), UserID: memberID, OrgID: orgID, Role: models.OrgRoleMember},
			},
		},
	}
	ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/members/"+memberID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("cannot remove last owner", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		// Another owner user trying to remove the only owner
		// The owner is trying to leave (self-removal), and they are the last owner
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/members/"+ownerID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid org id", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/bad-uuid/members/"+memberID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid user id", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/members/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestListInvitations(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID): {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
		},
		invitations: []*models.OrgInvitationWithDetails{},
	}

	t.Run("success", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/invitations", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		nonMember := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupOrgTestRouter(store, nonMember)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/invitations", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})
}

func TestCreateInvitation(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID): {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
		},
	}

	t.Run("success", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"email":"invite@test.com","role":"member"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/invitations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		nonMember := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupOrgTestRouter(store, nonMember)
		w := httptest.NewRecorder()
		body := `{"email":"invite@test.com","role":"member"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/invitations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/invitations", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestDeleteInvitation(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	invID := uuid.New()

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID): {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
		},
	}

	t.Run("success", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/invitations/"+invID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid org id", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/bad-uuid/invitations/"+invID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid invitation id", func(t *testing.T) {
		user := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/invitations/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		nonMember := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupOrgTestRouter(store, nonMember)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/invitations/"+invID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})
}

func TestAcceptInvitation(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	invID := uuid.New()

	inv := &models.OrgInvitation{
		ID:        invID,
		OrgID:     orgID,
		Email:     "user@test.com",
		Role:      models.OrgRoleMember,
		Token:     "valid-token",
		InvitedBy: uuid.New(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	org := &models.Organization{ID: orgID, Name: "Test", Slug: "test"}

	store := &mockOrgStore{
		orgByID:     map[uuid.UUID]*models.Organization{orgID: org},
		memberships: map[string]*models.OrgMembership{},
		invByToken:  map[string]*models.OrgInvitation{"valid-token": inv},
	}
	user := &auth.SessionUser{ID: userID, Email: "user@test.com", CurrentOrgID: uuid.New()}

	t.Run("not found", func(t *testing.T) {
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{"token":"invalid-token"}`
		req, _ := http.NewRequest("POST", "/api/v1/invitations/accept", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong email", func(t *testing.T) {
		wrongEmailUser := &auth.SessionUser{ID: userID, Email: "other@test.com", CurrentOrgID: uuid.New()}
		r := setupOrgTestRouter(store, wrongEmailUser)
		w := httptest.NewRecorder()
		body := `{"token":"valid-token"}`
		req, _ := http.NewRequest("POST", "/api/v1/invitations/accept", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupOrgTestRouter(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/invitations/accept", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupOrgTestRouter(store, nil)
		w := httptest.NewRecorder()
		body := `{"token":"valid-token"}`
		req, _ := http.NewRequest("POST", "/api/v1/invitations/accept", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("success", func(t *testing.T) {
		successStore := &mockOrgStore{
			orgByID:     map[uuid.UUID]*models.Organization{orgID: org},
			memberships: map[string]*models.OrgMembership{},
			invByToken:  map[string]*models.OrgInvitation{"valid-token": inv},
		}
		r := setupOrgTestRouterWithSessions(successStore, user)
		w := httptest.NewRecorder()
		body := `{"token":"valid-token"}`
		req, _ := http.NewRequest("POST", "/api/v1/invitations/accept", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("expired invitation", func(t *testing.T) {
		expiredInv := &models.OrgInvitation{
			ID:        uuid.New(),
			OrgID:     orgID,
			Email:     "user@test.com",
			Role:      models.OrgRoleMember,
			Token:     "expired-token",
			InvitedBy: uuid.New(),
			ExpiresAt: time.Now().Add(-24 * time.Hour),
		}
		expiredStore := &mockOrgStore{
			orgByID:     map[uuid.UUID]*models.Organization{orgID: org},
			memberships: map[string]*models.OrgMembership{},
			invByToken:  map[string]*models.OrgInvitation{"expired-token": expiredInv},
		}
		r := setupOrgTestRouter(expiredStore, user)
		w := httptest.NewRecorder()
		body := `{"token":"expired-token"}`
		req, _ := http.NewRequest("POST", "/api/v1/invitations/accept", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusGone {
			t.Fatalf("expected 410, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestUpdateMember(t *testing.T) {
	orgID := uuid.New()
	ownerID := uuid.New()
	targetID := uuid.New()

	store := &mockOrgStore{
		orgByID: map[uuid.UUID]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(ownerID, orgID):  {ID: uuid.New(), UserID: ownerID, OrgID: orgID, Role: models.OrgRoleOwner},
			membershipKey(targetID, orgID): {ID: uuid.New(), UserID: targetID, OrgID: orgID, Role: models.OrgRoleMember},
		},
	}
	ownerUser := &auth.SessionUser{ID: ownerID, CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		body := `{"role":"admin"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/members/"+targetID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/members/"+targetID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid org id", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		body := `{"role":"admin"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/bad-uuid/members/"+targetID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("invalid user id", func(t *testing.T) {
		r := setupOrgTestRouter(store, ownerUser)
		w := httptest.NewRecorder()
		body := `{"role":"admin"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/members/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func setupOrgTestRouterWithSessions(store *mockOrgStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	rbac := auth.NewRBAC(store)
	sessionStore, _ := auth.NewSessionStore(auth.SessionConfig{
		Secret:     make([]byte, 32),
		MaxAge:     3600,
		CookiePath: "/",
	}, zerolog.Nop())
	handler := NewOrganizationsHandler(store, sessionStore, rbac, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestSwitchOrganization(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	org := &models.Organization{ID: orgID, Name: "Target Org", Slug: "target"}

	store := &mockOrgStore{
		orgByID:   map[uuid.UUID]*models.Organization{orgID: org},
		orgBySlug: map[string]*models.Organization{},
		memberships: map[string]*models.OrgMembership{
			membershipKey(userID, orgID): {ID: uuid.New(), UserID: userID, OrgID: orgID, Role: models.OrgRoleMember},
		},
	}

	t.Run("success", func(t *testing.T) {
		user := &auth.SessionUser{ID: userID, CurrentOrgID: uuid.New()}
		r := setupOrgTestRouterWithSessions(store, user)
		w := httptest.NewRecorder()
		body := `{"org_id":"` + orgID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/switch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		user := &auth.SessionUser{ID: userID, CurrentOrgID: uuid.New()}
		r := setupOrgTestRouterWithSessions(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/switch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not a member", func(t *testing.T) {
		otherUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupOrgTestRouterWithSessions(store, otherUser)
		w := httptest.NewRecorder()
		body := `{"org_id":"` + orgID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/switch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", w.Code)
		}
	})

	t.Run("org not found", func(t *testing.T) {
		unknownOrgID := uuid.New()
		storeWithMembership := &mockOrgStore{
			orgByID:   map[uuid.UUID]*models.Organization{},
			orgBySlug: map[string]*models.Organization{},
			memberships: map[string]*models.OrgMembership{
				membershipKey(userID, unknownOrgID): {ID: uuid.New(), UserID: userID, OrgID: unknownOrgID, Role: models.OrgRoleMember},
			},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: uuid.New()}
		r := setupOrgTestRouterWithSessions(storeWithMembership, user)
		w := httptest.NewRecorder()
		body := `{"org_id":"` + unknownOrgID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/switch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		r := setupOrgTestRouterWithSessions(store, nil)
		w := httptest.NewRecorder()
		body := `{"org_id":"` + orgID.String() + `"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/switch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}
