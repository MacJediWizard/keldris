package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockTrialStore struct {
	info       *license.TrialInfo
	extensions []*license.TrialExtension
	activity   []*license.TrialActivity
	err        error
}

func (m *mockTrialStore) GetTrialInfo(_ context.Context, orgID uuid.UUID) (*license.TrialInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.info != nil {
		return m.info, nil
	}
	return &license.TrialInfo{OrgID: orgID, PlanTier: license.PlanTierFree, TrialStatus: license.TrialStatusNone}, nil
}

func (m *mockTrialStore) StartTrial(_ context.Context, _ uuid.UUID, _ string) error {
	return m.err
}

func (m *mockTrialStore) ExtendTrial(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int, _ string) (*license.TrialExtension, error) {
	return nil, m.err
}

func (m *mockTrialStore) ConvertTrial(_ context.Context, _ uuid.UUID, _ license.PlanTier) error {
	return m.err
}

func (m *mockTrialStore) ExpireTrials(_ context.Context) (int, error) {
	return 0, m.err
}

func (m *mockTrialStore) GetTrialExtensions(_ context.Context, _ uuid.UUID) ([]*license.TrialExtension, error) {
	return m.extensions, m.err
}

func (m *mockTrialStore) LogTrialActivity(_ context.Context, _ *license.TrialActivity) error {
	return m.err
}

func (m *mockTrialStore) GetTrialActivity(_ context.Context, _ uuid.UUID, _, _ int) ([]*license.TrialActivity, error) {
	return m.activity, m.err
}

func (m *mockTrialStore) GetExpiringTrials(_ context.Context, _ int) ([]*license.TrialInfo, error) {
	return nil, m.err
}

func setupTrialTestRouter(store TrialStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewTrialHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestTrialGetStatus(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns trial info", func(t *testing.T) {
		store := &mockTrialStore{}
		r := setupTrialTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/trial/status"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
		var info license.TrialInfo
		if err := json.Unmarshal(resp.Body.Bytes(), &info); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if info.OrgID != orgID {
			t.Errorf("expected org id %s, got %s", orgID, info.OrgID)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockTrialStore{}
		r := setupTrialTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/trial/status"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestTrialStartTrial(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("non-admin forbidden", func(t *testing.T) {
		viewer := testUser(orgID)
		viewer.CurrentOrgRole = "viewer"
		store := &mockTrialStore{}
		r := setupTrialTestRouter(store, viewer)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/trial/start", `{"email":"test@example.com"}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("invalid body returns 400", func(t *testing.T) {
		store := &mockTrialStore{}
		r := setupTrialTestRouter(store, user)

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/trial/start", `{invalid`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockTrialStore{}
		r := setupTrialTestRouter(store, testUserNoOrg())

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/trial/start", `{"email":"test@example.com"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestTrialGetFeatures(t *testing.T) {
	user := testUser(uuid.New())
	store := &mockTrialStore{}
	r := setupTrialTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/trial/features"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
