package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockPasswordPolicyStore struct {
	policy       *models.PasswordPolicy
	userInfo     *models.UserPasswordInfo
	getOrCreate  error
	updateErr    error
	getInfoErr   error
	updatePwdErr error
	createHist   error
	cleanupHist  error
	history      []*models.PasswordHistory
}

func (m *mockPasswordPolicyStore) GetPasswordPolicyByOrgID(_ context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error) {
	if m.policy != nil {
		return m.policy, nil
	}
	return models.NewPasswordPolicy(orgID), nil
}

func (m *mockPasswordPolicyStore) GetOrCreatePasswordPolicy(_ context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error) {
	if m.getOrCreate != nil {
		return nil, m.getOrCreate
	}
	if m.policy != nil {
		return m.policy, nil
	}
	m.policy = models.NewPasswordPolicy(orgID)
	return m.policy, nil
}

func (m *mockPasswordPolicyStore) UpdatePasswordPolicy(_ context.Context, p *models.PasswordPolicy) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.policy = p
	return nil
}

func (m *mockPasswordPolicyStore) GetPasswordHistory(_ context.Context, _ uuid.UUID, _ int) ([]*models.PasswordHistory, error) {
	return m.history, nil
}

func (m *mockPasswordPolicyStore) CreatePasswordHistory(_ context.Context, _ *models.PasswordHistory) error {
	return m.createHist
}

func (m *mockPasswordPolicyStore) CleanupPasswordHistory(_ context.Context, _ uuid.UUID, _ int) error {
	return m.cleanupHist
}

func (m *mockPasswordPolicyStore) GetUserPasswordInfo(_ context.Context, _ uuid.UUID) (*models.UserPasswordInfo, error) {
	if m.getInfoErr != nil {
		return nil, m.getInfoErr
	}
	if m.userInfo != nil {
		return m.userInfo, nil
	}
	return &models.UserPasswordInfo{}, nil
}

func (m *mockPasswordPolicyStore) UpdateUserPassword(_ context.Context, _ uuid.UUID, _ string, _ *time.Time) error {
	return m.updatePwdErr
}

func setupPasswordPolicyTestRouter(store PasswordPolicyStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewPasswordPoliciesHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestPasswordPoliciesGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns default policy", func(t *testing.T) {
		store := &mockPasswordPolicyStore{}
		r := setupPasswordPolicyTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/password-policies"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var body models.PasswordPolicyResponse
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if body.Policy.MinLength != 8 {
			t.Errorf("expected default MinLength 8, got %d", body.Policy.MinLength)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockPasswordPolicyStore{}
		r := setupPasswordPolicyTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/password-policies"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestPasswordPoliciesUpdate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("admin updates policy", func(t *testing.T) {
		store := &mockPasswordPolicyStore{}
		r := setupPasswordPolicyTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/password-policies", `{"min_length":12,"require_special":true}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		if store.policy.MinLength != 12 {
			t.Errorf("expected MinLength=12 stored, got %d", store.policy.MinLength)
		}
		if !store.policy.RequireSpecial {
			t.Errorf("expected RequireSpecial=true stored")
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		viewer := testUser(orgID)
		viewer.CurrentOrgRole = "viewer"
		store := &mockPasswordPolicyStore{}
		r := setupPasswordPolicyTestRouter(store, viewer)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/password-policies", `{"min_length":12}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("min_length below 6 rejected by binding", func(t *testing.T) {
		store := &mockPasswordPolicyStore{}
		r := setupPasswordPolicyTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/password-policies", `{"min_length":4}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestPasswordPoliciesGetRequirements(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockPasswordPolicyStore{}
	r := setupPasswordPolicyTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/password-policies/requirements"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	var req models.PasswordRequirements
	if err := json.Unmarshal(resp.Body.Bytes(), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.MinLength != 8 {
		t.Errorf("expected default MinLength 8, got %d", req.MinLength)
	}
}

func TestPasswordPoliciesValidatePassword(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockPasswordPolicyStore{}
	r := setupPasswordPolicyTestRouter(store, user)

	t.Run("valid password", func(t *testing.T) {
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/password-policies/validate", `{"password":"GoodPassword123"}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var result auth.ValidationResult
		_ = json.Unmarshal(resp.Body.Bytes(), &result)
		if !result.Valid {
			t.Errorf("expected valid=true, got errors=%v", result.Errors)
		}
	})

	t.Run("too short", func(t *testing.T) {
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/password-policies/validate", `{"password":"short"}`))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		var result auth.ValidationResult
		_ = json.Unmarshal(resp.Body.Bytes(), &result)
		if result.Valid {
			t.Errorf("expected valid=false for short password")
		}
	})
}

func TestPasswordPoliciesGetExpirationInfo(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns empty info when no password set", func(t *testing.T) {
		store := &mockPasswordPolicyStore{
			userInfo: &models.UserPasswordInfo{UserID: user.ID, PasswordHash: nil},
		}
		r := setupPasswordPolicyTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/password/expiration"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var info models.PasswordExpirationInfo
		_ = json.Unmarshal(resp.Body.Bytes(), &info)
		if info.IsExpired || info.MustChangeNow {
			t.Errorf("expected empty info, got %+v", info)
		}
	})
}
