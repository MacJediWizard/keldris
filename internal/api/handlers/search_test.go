package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockSearchStore struct {
	results   []db.SearchResult
	searchErr error
}

func (m *mockSearchStore) Search(_ context.Context, _ uuid.UUID, _ db.SearchFilter) ([]db.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.results, nil
}

func setupSearchTestRouter(store SearchStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	handler := NewSearchHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestSearch(t *testing.T) {
	orgID := uuid.New()
	store := &mockSearchStore{
		results: []db.SearchResult{
			{Type: "agent", ID: uuid.New().String(), Name: "test-agent"},
		},
	}
	user := &auth.SessionUser{
		ID:           uuid.New(),
		CurrentOrgID: orgID,
	}

	t.Run("success", func(t *testing.T) {
		r := setupSearchTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if _, ok := resp["results"]; !ok {
			t.Fatal("expected 'results' key in response")
		}
		if _, ok := resp["query"]; !ok {
			t.Fatal("expected 'query' key in response")
		}
		if _, ok := resp["total"]; !ok {
			t.Fatal("expected 'total' key in response")
		}
	})

	t.Run("missing query", func(t *testing.T) {
		r := setupSearchTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("no org selected", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupSearchTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupSearchTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockSearchStore{searchErr: errors.New("db error")}
		r := setupSearchTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		r := setupSearchTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test&types=agent,backup&status=active&limit=5", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid limit", func(t *testing.T) {
		r := setupSearchTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test&limit=999", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid tag_ids", func(t *testing.T) {
		r := setupSearchTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test&tag_ids=not-a-uuid", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid date_from", func(t *testing.T) {
		r := setupSearchTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test&date_from=not-a-date", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid size_min", func(t *testing.T) {
		r := setupSearchTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/search?q=test&size_min=abc", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected status 400, got %d", w.Code)
		}
	})
}
