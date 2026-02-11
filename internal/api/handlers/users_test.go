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

type mockUsersStore struct {
	users     []*models.User
	getErr    error
	listErr   error
	updateErr error
	deleteErr error
}

func (m *mockUsersStore) ListUsers(_ context.Context, _ uuid.UUID) ([]*models.User, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.users, nil
}

func (m *mockUsersStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUsersStore) UpdateUser(_ context.Context, _ *models.User) error {
	return m.updateErr
}

func (m *mockUsersStore) DeleteUser(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

type mockMembershipStore struct {
	membership *models.OrgMembership
	err        error
}

func (m *mockMembershipStore) GetMembershipByUserAndOrg(_ context.Context, _, _ uuid.UUID) (*models.OrgMembership, error) {
	return m.membership, m.err
}

func (m *mockMembershipStore) GetMembershipsByUserID(_ context.Context, _ uuid.UUID) ([]*models.OrgMembership, error) {
	if m.membership != nil {
		return []*models.OrgMembership{m.membership}, nil
	}
	return nil, nil
}

func setupUsersTestRouter(store UsersStore, memberStore auth.MembershipStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	rbac := auth.NewRBAC(memberStore)
	handler := NewUsersHandler(store, rbac, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestUsersList(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	membership := &models.OrgMembership{UserID: user.ID, OrgID: orgID, Role: models.OrgRoleAdmin}

	t.Run("success", func(t *testing.T) {
		u1 := models.NewUser(orgID, "sub1", "user1@example.com", "User 1", models.UserRoleAdmin)
		store := &mockUsersStore{users: []*models.User{u1}}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users"))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}

		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["users"]; !ok {
			t.Fatal("expected 'users' key in response")
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockUsersStore{}
		memberStore := &mockMembershipStore{}
		r := setupUsersTestRouter(store, memberStore, nil)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users"))

		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockUsersStore{}
		memberStore := &mockMembershipStore{err: errors.New("no membership")}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users"))

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockUsersStore{listErr: errors.New("db error")}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users"))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestUsersGet(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	membership := &models.OrgMembership{UserID: user.ID, OrgID: orgID, Role: models.OrgRoleAdmin}
	target := models.NewUser(orgID, "sub1", "target@example.com", "Target", models.UserRoleUser)

	t.Run("success", func(t *testing.T) {
		store := &mockUsersStore{users: []*models.User{target}}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users/"+target.ID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockUsersStore{}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users/not-a-uuid"))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockUsersStore{users: []*models.User{}}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users/"+uuid.New().String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUser := models.NewUser(uuid.New(), "sub2", "other@example.com", "Other", models.UserRoleUser)
		store := &mockUsersStore{users: []*models.User{otherUser}}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users/"+otherUser.ID.String()))

		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestUsersUpdate(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	membership := &models.OrgMembership{UserID: user.ID, OrgID: orgID, Role: models.OrgRoleAdmin}
	target := models.NewUser(orgID, "sub1", "target@example.com", "Target", models.UserRoleUser)

	t.Run("success", func(t *testing.T) {
		store := &mockUsersStore{users: []*models.User{target}}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/users/"+target.ID.String(), `{"name":"Updated"}`))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockUsersStore{}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/users/bad-id", `{"name":"x"}`))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockUsersStore{}
		memberStore := &mockMembershipStore{err: errors.New("no access")}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/users/"+target.ID.String(), `{"name":"x"}`))

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockUsersStore{
			users:     []*models.User{target},
			updateErr: errors.New("db error"),
		}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/users/"+target.ID.String(), `{"name":"x"}`))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestUsersDelete(t *testing.T) {
	orgID := uuid.New()
	user := TestUser(orgID)
	membership := &models.OrgMembership{UserID: user.ID, OrgID: orgID, Role: models.OrgRoleAdmin}
	target := models.NewUser(orgID, "sub1", "target@example.com", "Target", models.UserRoleUser)

	t.Run("success", func(t *testing.T) {
		store := &mockUsersStore{users: []*models.User{target}}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/"+target.ID.String()))

		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("last owner error", func(t *testing.T) {
		store := &mockUsersStore{
			users:     []*models.User{target},
			deleteErr: errors.New("last owner of org"),
		}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/"+target.ID.String()))

		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store generic error", func(t *testing.T) {
		store := &mockUsersStore{
			users:     []*models.User{target},
			deleteErr: errors.New("db error"),
		}
		memberStore := &mockMembershipStore{membership: membership}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/"+target.ID.String()))

		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		store := &mockUsersStore{}
		memberStore := &mockMembershipStore{err: errors.New("no access")}
		r := setupUsersTestRouter(store, memberStore, user)

		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/"+target.ID.String()))

		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}
