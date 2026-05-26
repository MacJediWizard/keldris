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

type mockRecentItemsStore struct {
	items     []*models.RecentItem
	item      *models.RecentItem
	listErr   error
	upsertErr error
	deleteErr error
	clearErr  error
	getErr    error
}

func (m *mockRecentItemsStore) CreateOrUpdateRecentItem(_ context.Context, _ *models.RecentItem) error {
	return m.upsertErr
}

func (m *mockRecentItemsStore) GetRecentItemsByUser(_ context.Context, _, _ uuid.UUID, _ int) ([]*models.RecentItem, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *mockRecentItemsStore) GetRecentItemsByUserAndType(_ context.Context, _, _ uuid.UUID, _ models.RecentItemType, _ int) ([]*models.RecentItem, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *mockRecentItemsStore) DeleteRecentItem(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockRecentItemsStore) DeleteRecentItemsForUser(_ context.Context, _, _ uuid.UUID) error {
	return m.clearErr
}

func (m *mockRecentItemsStore) GetRecentItemByID(_ context.Context, _ uuid.UUID) (*models.RecentItem, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.item, nil
}

func setupRecentItemsTestRouter(store RecentItemsStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewRecentItemsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestRecentItemsList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns items", func(t *testing.T) {
		store := &mockRecentItemsStore{items: []*models.RecentItem{{ID: uuid.New(), UserID: user.ID}}}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/recent-items"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid type returns 400", func(t *testing.T) {
		store := &mockRecentItemsStore{}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/recent-items?type=bogus"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockRecentItemsStore{listErr: errors.New("db down")}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/recent-items"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestRecentItemsTrack(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("tracks valid item", func(t *testing.T) {
		store := &mockRecentItemsStore{}
		r := setupRecentItemsTestRouter(store, user)
		body := `{"item_type":"agent","item_id":"` + uuid.New().String() + `","item_name":"agent1","page_path":"/agents/1"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/recent-items", body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid type returns 400", func(t *testing.T) {
		store := &mockRecentItemsStore{}
		r := setupRecentItemsTestRouter(store, user)
		body := `{"item_type":"bogus","item_id":"` + uuid.New().String() + `","item_name":"x","page_path":"/x"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/recent-items", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		store := &mockRecentItemsStore{}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/recent-items", `{"item_type":"agent"}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestRecentItemsDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("deletes own item", func(t *testing.T) {
		store := &mockRecentItemsStore{item: &models.RecentItem{ID: id, UserID: user.ID}}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/recent-items/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("other user's item returns 404", func(t *testing.T) {
		store := &mockRecentItemsStore{item: &models.RecentItem{ID: id, UserID: uuid.New()}}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/recent-items/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockRecentItemsStore{}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/recent-items/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestRecentItemsClearAll(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("clears successfully", func(t *testing.T) {
		store := &mockRecentItemsStore{}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/recent-items"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockRecentItemsStore{clearErr: errors.New("db down")}
		r := setupRecentItemsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/recent-items"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
