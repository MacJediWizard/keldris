package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// --- Mock stores ---

type mockSSOGroupMappingStore struct {
	mappings          []*models.SSOGroupMapping
	mapping           *models.SSOGroupMapping
	ssoGroups         *models.UserSSOGroups
	defaultRole       *string
	autoCreate        bool
	listErr           error
	getErr            error
	createErr         error
	updateErr         error
	deleteErr         error
	ssoGroupsErr      error
	getSettingsErr    error
	updateSettingsErr error
	auditErr          error
}

func (m *mockSSOGroupMappingStore) GetSSOGroupMappingsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.SSOGroupMapping, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.mappings, nil
}

func (m *mockSSOGroupMappingStore) GetSSOGroupMappingByID(_ context.Context, _ uuid.UUID) (*models.SSOGroupMapping, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.mapping, nil
}

func (m *mockSSOGroupMappingStore) CreateSSOGroupMapping(_ context.Context, _ *models.SSOGroupMapping) error {
	return m.createErr
}

func (m *mockSSOGroupMappingStore) UpdateSSOGroupMapping(_ context.Context, _ *models.SSOGroupMapping) error {
	return m.updateErr
}

func (m *mockSSOGroupMappingStore) DeleteSSOGroupMapping(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockSSOGroupMappingStore) GetUserSSOGroups(_ context.Context, _ uuid.UUID) (*models.UserSSOGroups, error) {
	if m.ssoGroupsErr != nil {
		return nil, m.ssoGroupsErr
	}
	return m.ssoGroups, nil
}

func (m *mockSSOGroupMappingStore) GetOrganizationSSOSettings(_ context.Context, _ uuid.UUID) (*string, bool, error) {
	if m.getSettingsErr != nil {
		return nil, false, m.getSettingsErr
	}
	return m.defaultRole, m.autoCreate, nil
}

func (m *mockSSOGroupMappingStore) UpdateOrganizationSSOSettings(_ context.Context, _ uuid.UUID, _ *string, _ bool) error {
	return m.updateSettingsErr
}

func (m *mockSSOGroupMappingStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return m.auditErr
}

type mockSSOmembershipStore struct {
	membership  *models.OrgMembership
	memberships []*models.OrgMembership
	err         error
}

func (m *mockSSOmembershipStore) GetMembershipByUserAndOrg(_ context.Context, _, _ uuid.UUID) (*models.OrgMembership, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.membership, nil
}

func (m *mockSSOmembershipStore) GetMembershipsByUserID(_ context.Context, _ uuid.UUID) ([]*models.OrgMembership, error) {
	return m.memberships, m.err
}

// --- Setup helper ---

func setupSSOGroupMappingsTestRouter(store SSOGroupMappingStore, memberStore *mockSSOmembershipStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	rbac := auth.NewRBAC(memberStore)
	handler := NewSSOGroupMappingsHandler(store, rbac, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

// --- Test: List SSO Group Mappings ---

func TestListSSOGroupMappings(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	now := time.Now()

	mapping1 := &models.SSOGroupMapping{
		ID:            uuid.New(),
		OrgID:         orgID,
		OIDCGroupName: "engineering",
		Role:          models.OrgRoleAdmin,
		AutoCreateOrg: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	mapping2 := &models.SSOGroupMapping{
		ID:            uuid.New(),
		OrgID:         orgID,
		OIDCGroupName: "viewers",
		Role:          models.OrgRoleReadonly,
		AutoCreateOrg: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	t.Run("success", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			mappings: []*models.SSOGroupMapping{mapping1, mapping2},
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOGroupMappingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if len(resp.Mappings) != 2 {
			t.Fatalf("expected 2 mappings, got %d", len(resp.Mappings))
		}
	})

	t.Run("invalid org_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/bad-uuid/sso-group-mappings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied - readonly role", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleReadonly},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			listErr: errors.New("database error"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: Create SSO Group Mapping ---

func TestCreateSSOGroupMapping(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"engineering","role":"admin","auto_create_org":true}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOGroupMappingResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Mapping == nil {
			t.Fatal("expected mapping in response")
		}
		if resp.Mapping.OIDCGroupName != "engineering" {
			t.Fatalf("expected oidc_group_name 'engineering', got '%s'", resp.Mapping.OIDCGroupName)
		}
		if resp.Mapping.Role != models.OrgRoleAdmin {
			t.Fatalf("expected role 'admin', got '%s'", resp.Mapping.Role)
		}
		if !resp.Mapping.AutoCreateOrg {
			t.Fatal("expected auto_create_org to be true")
		}
	})

	t.Run("invalid org_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"engineering","role":"admin"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/not-a-uuid/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleReadonly},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"engineering","role":"admin"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body - missing required fields", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body - malformed JSON", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `not json`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"engineering","role":"superuser"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("store create error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			createErr: errors.New("database error"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"engineering","role":"admin"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"engineering","role":"admin"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: Get SSO Group Mapping ---

func TestGetSSOGroupMapping(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	mappingID := uuid.New()
	now := time.Now()

	mapping := &models.SSOGroupMapping{
		ID:            mappingID,
		OrgID:         orgID,
		OIDCGroupName: "engineering",
		Role:          models.OrgRoleAdmin,
		AutoCreateOrg: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	t.Run("success", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOGroupMappingResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Mapping == nil {
			t.Fatal("expected mapping in response")
		}
		if resp.Mapping.ID != mappingID {
			t.Fatalf("expected mapping ID %s, got %s", mappingID, resp.Mapping.ID)
		}
	})

	t.Run("invalid org_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/bad-uuid/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid mapping_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/bad-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleReadonly},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found - store error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			getErr: errors.New("not found"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong org - mapping belongs to different org", func(t *testing.T) {
		differentOrgMapping := &models.SSOGroupMapping{
			ID:            mappingID,
			OrgID:         uuid.New(), // Different org
			OIDCGroupName: "engineering",
			Role:          models.OrgRoleAdmin,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		store := &mockSSOGroupMappingStore{mapping: differentOrgMapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: Update SSO Group Mapping ---

func TestUpdateSSOGroupMapping(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	mappingID := uuid.New()
	now := time.Now()

	mapping := &models.SSOGroupMapping{
		ID:            mappingID,
		OrgID:         orgID,
		OIDCGroupName: "engineering",
		Role:          models.OrgRoleAdmin,
		AutoCreateOrg: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	t.Run("success - update role", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOGroupMappingResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Mapping == nil {
			t.Fatal("expected mapping in response")
		}
		if resp.Mapping.Role != models.OrgRoleMember {
			t.Fatalf("expected role 'member', got '%s'", resp.Mapping.Role)
		}
	})

	t.Run("success - update auto_create_org", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"auto_create_org":true}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid org_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/bad-uuid/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid mapping_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleReadonly},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body - malformed JSON", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `not json`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found - store error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			getErr: errors.New("not found"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong org - mapping belongs to different org", func(t *testing.T) {
		differentOrgMapping := &models.SSOGroupMapping{
			ID:            mappingID,
			OrgID:         uuid.New(), // Different org
			OIDCGroupName: "engineering",
			Role:          models.OrgRoleAdmin,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		store := &mockSSOGroupMappingStore{mapping: differentOrgMapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid role in update", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"superuser"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("update error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			mapping:   mapping,
			updateErr: errors.New("database error"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		body := `{"role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: Delete SSO Group Mapping ---

func TestDeleteSSOGroupMapping(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	mappingID := uuid.New()
	now := time.Now()

	mapping := &models.SSOGroupMapping{
		ID:            mappingID,
		OrgID:         orgID,
		OIDCGroupName: "engineering",
		Role:          models.OrgRoleAdmin,
		AutoCreateOrg: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	t.Run("success", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["message"] != "SSO group mapping deleted" {
			t.Fatalf("expected delete confirmation message, got: %s", resp["message"])
		}
	})

	t.Run("invalid org_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/bad-uuid/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid mapping_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/bad-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleReadonly},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found - store error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			getErr: errors.New("not found"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong org - mapping belongs to different org", func(t *testing.T) {
		differentOrgMapping := &models.SSOGroupMapping{
			ID:            mappingID,
			OrgID:         uuid.New(), // Different org
			OIDCGroupName: "engineering",
			Role:          models.OrgRoleAdmin,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		store := &mockSSOGroupMappingStore{mapping: differentOrgMapping}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("delete error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			mapping:   mapping,
			deleteErr: errors.New("database error"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{mapping: mapping}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: GetSSOSettings ---

func TestGetSSOSettings(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		defaultRole := "member"
		store := &mockSSOGroupMappingStore{
			defaultRole: &defaultRole,
			autoCreate:  true,
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-settings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.DefaultRole == nil || *resp.DefaultRole != "member" {
			t.Fatalf("expected default_role 'member', got %v", resp.DefaultRole)
		}
		if !resp.AutoCreateOrgs {
			t.Fatal("expected auto_create_orgs to be true")
		}
	})

	t.Run("success - nil default role", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			defaultRole: nil,
			autoCreate:  false,
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-settings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.DefaultRole != nil {
			t.Fatalf("expected nil default_role, got %v", *resp.DefaultRole)
		}
	})

	t.Run("invalid org_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/bad-uuid/sso-settings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleReadonly},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-settings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			getSettingsErr: errors.New("database error"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-settings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-settings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: UpdateSSOSettings ---

func TestUpdateSSOSettings(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("success - update default role", func(t *testing.T) {
		currentRole := "member"
		store := &mockSSOGroupMappingStore{
			defaultRole: &currentRole,
			autoCreate:  false,
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":"admin"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.DefaultRole == nil || *resp.DefaultRole != "admin" {
			t.Fatalf("expected default_role 'admin', got %v", resp.DefaultRole)
		}
	})

	t.Run("success - update auto_create_orgs", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			defaultRole: nil,
			autoCreate:  false,
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"auto_create_orgs":true}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !resp.AutoCreateOrgs {
			t.Fatal("expected auto_create_orgs to be true")
		}
	})

	t.Run("success - clear default role with empty string", func(t *testing.T) {
		currentRole := "member"
		store := &mockSSOGroupMappingStore{
			defaultRole: &currentRole,
			autoCreate:  false,
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":""}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.DefaultRole != nil {
			t.Fatalf("expected nil default_role after clearing, got %v", *resp.DefaultRole)
		}
	})

	t.Run("invalid org_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/bad-uuid/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleReadonly},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body - malformed JSON", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `not json`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid default role", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":"superuser"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("get settings error", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			getSettingsErr: errors.New("database error"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("update settings error", func(t *testing.T) {
		currentRole := "member"
		store := &mockSSOGroupMappingStore{
			defaultRole:       &currentRole,
			autoCreate:        false,
			updateSettingsErr: errors.New("database error"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":"admin"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		body := `{"default_role":"member"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: GetUserSSOGroups ---

func TestGetUserSSOGroups(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	now := time.Now()

	t.Run("success - own groups", func(t *testing.T) {
		ssoGroups := &models.UserSSOGroups{
			ID:         uuid.New(),
			UserID:     userID,
			OIDCGroups: []string{"engineering", "devops", "platform"},
			SyncedAt:   now,
		}
		store := &mockSSOGroupMappingStore{ssoGroups: ssoGroups}
		memberStore := &mockSSOmembershipStore{}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/users/"+userID.String()+"/sso-groups", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["user_id"] != userID.String() {
			t.Fatalf("expected user_id %s, got %v", userID, resp["user_id"])
		}
		groups, ok := resp["oidc_groups"].([]interface{})
		if !ok {
			t.Fatal("expected oidc_groups to be an array")
		}
		if len(groups) != 3 {
			t.Fatalf("expected 3 groups, got %d", len(groups))
		}
	})

	t.Run("forbidden - different user", func(t *testing.T) {
		otherUserID := uuid.New()
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/users/"+otherUserID.String()+"/sso-groups", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid user_id", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/users/bad-uuid/sso-groups", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no groups - returns empty", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			ssoGroupsErr: errors.New("no groups found"),
		}
		memberStore := &mockSSOmembershipStore{}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/users/"+userID.String()+"/sso-groups", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp["user_id"] != userID.String() {
			t.Fatalf("expected user_id %s, got %v", userID, resp["user_id"])
		}
		groups, ok := resp["oidc_groups"].([]interface{})
		if !ok {
			t.Fatal("expected oidc_groups to be an array")
		}
		if len(groups) != 0 {
			t.Fatalf("expected empty groups, got %d", len(groups))
		}
		if resp["synced_at"] != nil {
			t.Fatalf("expected nil synced_at, got %v", resp["synced_at"])
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/users/"+userID.String()+"/sso-groups", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: Permission with owner role ---

func TestSSOGroupMappingsOwnerPermission(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("owner can list mappings", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			mappings: []*models.SSOGroupMapping{},
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleOwner},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("member role cannot manage SSO", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleMember},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"eng","role":"admin"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: Audit log error does not fail the request ---

func TestSSOGroupMappingsAuditLogError(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	mappingID := uuid.New()
	now := time.Now()

	mapping := &models.SSOGroupMapping{
		ID:            mappingID,
		OrgID:         orgID,
		OIDCGroupName: "engineering",
		Role:          models.OrgRoleAdmin,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	t.Run("create succeeds even when audit log fails", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			auditErr: errors.New("audit log write failed"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"oidc_group_name":"engineering","role":"admin"}`
		req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("delete succeeds even when audit log fails", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			mapping:  mapping,
			auditErr: errors.New("audit log write failed"),
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings/"+mappingID.String(), nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

// --- Test: All valid role types can be used in create ---

func TestCreateSSOGroupMappingAllRoles(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	validRoles := []string{"owner", "admin", "member", "readonly"}

	for _, role := range validRoles {
		t.Run(fmt.Sprintf("create with role %s", role), func(t *testing.T) {
			store := &mockSSOGroupMappingStore{}
			memberStore := &mockSSOmembershipStore{
				membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
			}
			user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
			r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

			w := httptest.NewRecorder()
			body := fmt.Sprintf(`{"oidc_group_name":"group-%s","role":"%s"}`, role, role)
			req, _ := http.NewRequest("POST", "/api/v1/organizations/"+orgID.String()+"/sso-group-mappings", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 for role %s, got %d: %s", role, w.Code, w.Body.String())
			}
		})
	}
}

// --- Test: UpdateSSOSettings preserves unset fields ---

func TestUpdateSSOSettingsPartialUpdate(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	t.Run("only update default_role preserves auto_create", func(t *testing.T) {
		store := &mockSSOGroupMappingStore{
			defaultRole: nil,
			autoCreate:  true,
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"default_role":"admin"}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !resp.AutoCreateOrgs {
			t.Fatal("expected auto_create_orgs to remain true")
		}
		if resp.DefaultRole == nil || *resp.DefaultRole != "admin" {
			t.Fatalf("expected default_role 'admin', got %v", resp.DefaultRole)
		}
	})

	t.Run("only update auto_create preserves default_role", func(t *testing.T) {
		currentRole := "member"
		store := &mockSSOGroupMappingStore{
			defaultRole: &currentRole,
			autoCreate:  false,
		}
		memberStore := &mockSSOmembershipStore{
			membership: &models.OrgMembership{UserID: userID, OrgID: orgID, Role: models.OrgRoleAdmin},
		}
		user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}
		r := setupSSOGroupMappingsTestRouter(store, memberStore, user)

		w := httptest.NewRecorder()
		body := `{"auto_create_orgs":true}`
		req, _ := http.NewRequest("PUT", "/api/v1/organizations/"+orgID.String()+"/sso-settings", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp SSOSettingsResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.DefaultRole == nil || *resp.DefaultRole != "member" {
			t.Fatalf("expected default_role to remain 'member', got %v", resp.DefaultRole)
		}
		if !resp.AutoCreateOrgs {
			t.Fatal("expected auto_create_orgs to be true")
		}
	})
}
