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

type mockActivityStore struct {
	events     []*models.ActivityEvent
	count      int
	categories map[string]int
	getErr     error
	countErr   error
	catErr     error
	searchErr  error
	recentErr  error
}

func (m *mockActivityStore) GetActivityEvents(_ context.Context, _ uuid.UUID, _ models.ActivityEventFilter) ([]*models.ActivityEvent, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.events, nil
}

func (m *mockActivityStore) GetActivityEventCount(_ context.Context, _ uuid.UUID, _ models.ActivityEventFilter) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.count, nil
}

func (m *mockActivityStore) GetRecentActivityEvents(_ context.Context, _ uuid.UUID, _ int) ([]*models.ActivityEvent, error) {
	if m.recentErr != nil {
		return nil, m.recentErr
	}
	return m.events, nil
}

func (m *mockActivityStore) GetActivityCategories(_ context.Context, _ uuid.UUID) (map[string]int, error) {
	if m.catErr != nil {
		return nil, m.catErr
	}
	return m.categories, nil
}

func (m *mockActivityStore) SearchActivityEvents(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.ActivityEvent, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.events, nil
}

func setupActivityTestRouter(store ActivityStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewActivityHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestActivityList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns events", func(t *testing.T) {
		store := &mockActivityStore{events: []*models.ActivityEvent{}}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockActivityStore{getErr: errors.New("db down")}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no user returns 401", func(t *testing.T) {
		store := &mockActivityStore{}
		r := setupActivityTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestActivityRecent(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns recent events", func(t *testing.T) {
		store := &mockActivityStore{events: []*models.ActivityEvent{}}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/recent?limit=10"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("error returns 500", func(t *testing.T) {
		store := &mockActivityStore{recentErr: errors.New("db down")}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/recent"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestActivityCount(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns count", func(t *testing.T) {
		store := &mockActivityStore{count: 5}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/count"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("error returns 500", func(t *testing.T) {
		store := &mockActivityStore{countErr: errors.New("db down")}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/count"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestActivityCategories(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns categories", func(t *testing.T) {
		store := &mockActivityStore{categories: map[string]int{"backup": 3}}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/categories"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("error returns 500", func(t *testing.T) {
		store := &mockActivityStore{catErr: errors.New("db down")}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/categories"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestActivitySearch(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns search results", func(t *testing.T) {
		store := &mockActivityStore{events: []*models.ActivityEvent{}}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/search?q=backup"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("missing query returns 400", func(t *testing.T) {
		store := &mockActivityStore{}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/search"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("error returns 500", func(t *testing.T) {
		store := &mockActivityStore{searchErr: errors.New("db down")}
		r := setupActivityTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/activity/search?q=test"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
