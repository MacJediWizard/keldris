package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockIPAllowlistStore struct {
	allowlists  []*models.IPAllowlist
	allowlist   *models.IPAllowlist
	settings    *models.IPAllowlistSettings
	attempts    []*models.IPBlockedAttempt
	total       int
	getErr      error
	listErr     error
	createErr   error
	updateErr   error
	deleteErr   error
	settingsErr error
	attemptsErr error
}

func (m *mockIPAllowlistStore) GetIPAllowlistByID(_ context.Context, _ uuid.UUID) (*models.IPAllowlist, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.allowlist, nil
}

func (m *mockIPAllowlistStore) ListIPAllowlistsByOrg(_ context.Context, _ uuid.UUID) ([]*models.IPAllowlist, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.allowlists, nil
}

func (m *mockIPAllowlistStore) CreateIPAllowlist(_ context.Context, _ *models.IPAllowlist) error {
	return m.createErr
}

func (m *mockIPAllowlistStore) UpdateIPAllowlist(_ context.Context, _ *models.IPAllowlist) error {
	return m.updateErr
}

func (m *mockIPAllowlistStore) DeleteIPAllowlist(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockIPAllowlistStore) GetOrCreateIPAllowlistSettings(_ context.Context, orgID uuid.UUID) (*models.IPAllowlistSettings, error) {
	if m.settingsErr != nil {
		return nil, m.settingsErr
	}
	if m.settings != nil {
		return m.settings, nil
	}
	return models.NewIPAllowlistSettings(orgID), nil
}

func (m *mockIPAllowlistStore) UpdateIPAllowlistSettings(_ context.Context, _ *models.IPAllowlistSettings) error {
	return m.updateErr
}

func (m *mockIPAllowlistStore) ListIPBlockedAttemptsByOrg(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.IPBlockedAttempt, int, error) {
	if m.attemptsErr != nil {
		return nil, 0, m.attemptsErr
	}
	return m.attempts, m.total, nil
}

type mockIPFilterInvalidator struct {
	invalidated bool
}

func (m *mockIPFilterInvalidator) InvalidateCache(_ uuid.UUID) {
	m.invalidated = true
}

func setupIPAllowlistsTestRouter(store IPAllowlistStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewIPAllowlistsHandler(store, &mockIPFilterInvalidator{}, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestIPAllowlistsList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("admin lists allowlists", func(t *testing.T) {
		store := &mockIPAllowlistStore{allowlists: []*models.IPAllowlist{{ID: uuid.New(), OrgID: orgID, CIDR: "10.0.0.0/8", Type: models.IPAllowlistTypeUI}}}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlists"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, nonAdmin)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlists"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockIPAllowlistStore{listErr: errors.New("db error")}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlists"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestIPAllowlistsCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates valid allowlist", func(t *testing.T) {
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, user)
		body := `{"cidr":"10.0.0.0/8","type":"ui"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/ip-allowlists", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid CIDR returns 400", func(t *testing.T) {
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, user)
		body := `{"cidr":"not-an-ip","type":"ui"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/ip-allowlists", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, nonAdmin)
		body := `{"cidr":"10.0.0.0/8","type":"ui"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/ip-allowlists", body))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestIPAllowlistsGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("gets own allowlist", func(t *testing.T) {
		store := &mockIPAllowlistStore{allowlist: &models.IPAllowlist{ID: id, OrgID: orgID}}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlists/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		store := &mockIPAllowlistStore{allowlist: &models.IPAllowlist{ID: id, OrgID: uuid.New()}}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlists/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlists/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestIPAllowlistsUpdate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("updates own allowlist", func(t *testing.T) {
		store := &mockIPAllowlistStore{allowlist: &models.IPAllowlist{ID: id, OrgID: orgID, CIDR: "10.0.0.0/8"}}
		r := setupIPAllowlistsTestRouter(store, user)
		body := `{"description":"updated"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/ip-allowlists/"+id.String(), body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		store := &mockIPAllowlistStore{allowlist: &models.IPAllowlist{ID: id, OrgID: uuid.New()}}
		r := setupIPAllowlistsTestRouter(store, user)
		body := `{"description":"updated"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/ip-allowlists/"+id.String(), body))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestIPAllowlistsDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("deletes own allowlist", func(t *testing.T) {
		store := &mockIPAllowlistStore{allowlist: &models.IPAllowlist{ID: id, OrgID: orgID}}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/ip-allowlists/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other org returns 404", func(t *testing.T) {
		store := &mockIPAllowlistStore{allowlist: &models.IPAllowlist{ID: id, OrgID: uuid.New()}}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/ip-allowlists/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestIPAllowlistsGetSettings(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("admin gets settings", func(t *testing.T) {
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlist-settings"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, nonAdmin)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-allowlist-settings"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestIPAllowlistsUpdateSettings(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("admin updates settings", func(t *testing.T) {
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, user)
		body := `{"enabled":true}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/ip-allowlist-settings", body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		nonAdmin := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID, CurrentOrgRole: "member"}
		store := &mockIPAllowlistStore{}
		r := setupIPAllowlistsTestRouter(store, nonAdmin)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/ip-allowlist-settings", `{"enabled":true}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestIPAllowlistsListBlockedAttempts(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("admin lists attempts", func(t *testing.T) {
		store := &mockIPAllowlistStore{attempts: []*models.IPBlockedAttempt{{ID: uuid.New(), OrgID: orgID}}, total: 1}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-blocked-attempts"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockIPAllowlistStore{attemptsErr: errors.New("db down")}
		r := setupIPAllowlistsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/ip-blocked-attempts"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
