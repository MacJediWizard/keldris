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

type mockSavedFilterStore struct {
	filters   []*models.SavedFilter
	filter    *models.SavedFilter
	listErr   error
	getErr    error
	createErr error
	updateErr error
	deleteErr error
}

func (m *mockSavedFilterStore) GetSavedFiltersByUserAndOrg(_ context.Context, _, _ uuid.UUID, _ string) ([]*models.SavedFilter, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.filters, nil
}

func (m *mockSavedFilterStore) GetSavedFilterByID(_ context.Context, _ uuid.UUID) (*models.SavedFilter, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.filter, nil
}

func (m *mockSavedFilterStore) GetDefaultSavedFilter(_ context.Context, _, _ uuid.UUID, _ string) (*models.SavedFilter, error) {
	return m.filter, nil
}

func (m *mockSavedFilterStore) CreateSavedFilter(_ context.Context, f *models.SavedFilter) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.filter = f
	return nil
}

func (m *mockSavedFilterStore) UpdateSavedFilter(_ context.Context, _ *models.SavedFilter) error {
	return m.updateErr
}

func (m *mockSavedFilterStore) DeleteSavedFilter(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func setupFiltersTestRouter(store SavedFilterStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewFiltersHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestFiltersList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns filters", func(t *testing.T) {
		store := &mockSavedFilterStore{filters: []*models.SavedFilter{{ID: uuid.New(), OrgID: orgID, UserID: user.ID}}}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/filters"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockSavedFilterStore{}
		r := setupFiltersTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/filters"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSavedFilterStore{listErr: errors.New("db down")}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/filters"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestFiltersGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("returns own filter", func(t *testing.T) {
		store := &mockSavedFilterStore{filter: &models.SavedFilter{ID: id, OrgID: orgID, UserID: user.ID}}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/filters/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other user's private filter returns 404", func(t *testing.T) {
		store := &mockSavedFilterStore{filter: &models.SavedFilter{ID: id, OrgID: orgID, UserID: uuid.New(), Shared: false}}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/filters/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockSavedFilterStore{}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/filters/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestFiltersCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("creates filter", func(t *testing.T) {
		store := &mockSavedFilterStore{}
		r := setupFiltersTestRouter(store, user)
		body := `{"name":"my filter","entity_type":"agents","filters":{"status":"active"}}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/filters", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		store := &mockSavedFilterStore{}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/filters", `{"name":"only name"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSavedFilterStore{createErr: errors.New("db down")}
		r := setupFiltersTestRouter(store, user)
		body := `{"name":"my filter","entity_type":"agents","filters":{"status":"active"}}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/filters", body))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestFiltersUpdate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("updates own filter", func(t *testing.T) {
		store := &mockSavedFilterStore{filter: &models.SavedFilter{ID: id, OrgID: orgID, UserID: user.ID}}
		r := setupFiltersTestRouter(store, user)
		body := `{"name":"renamed"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/filters/"+id.String(), body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("other user's filter returns 403", func(t *testing.T) {
		store := &mockSavedFilterStore{filter: &models.SavedFilter{ID: id, OrgID: orgID, UserID: uuid.New()}}
		r := setupFiltersTestRouter(store, user)
		body := `{"name":"renamed"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/filters/"+id.String(), body))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestFiltersDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("deletes own filter", func(t *testing.T) {
		store := &mockSavedFilterStore{filter: &models.SavedFilter{ID: id, OrgID: orgID, UserID: user.ID}}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/filters/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other user's filter returns 403", func(t *testing.T) {
		store := &mockSavedFilterStore{filter: &models.SavedFilter{ID: id, OrgID: orgID, UserID: uuid.New()}}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/filters/"+id.String()))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockSavedFilterStore{}
		r := setupFiltersTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/filters/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
