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

type mockExcludePatternStore struct {
	patterns    []*models.ExcludePattern
	builtins    []*models.ExcludePattern
	getErr      error
	listErr     error
	catErr      error
	builtinErr  error
	createErr   error
	updateErr   error
	deleteErr   error
	seedErr     error
}

func (m *mockExcludePatternStore) GetExcludePatternsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.ExcludePattern, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.patterns, nil
}

func (m *mockExcludePatternStore) GetExcludePatternByID(_ context.Context, id uuid.UUID) (*models.ExcludePattern, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	for _, p := range m.patterns {
		if p.ID == id {
			return p, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockExcludePatternStore) GetExcludePatternsByCategory(_ context.Context, _ uuid.UUID, _ string) ([]*models.ExcludePattern, error) {
	if m.catErr != nil {
		return nil, m.catErr
	}
	return m.patterns, nil
}

func (m *mockExcludePatternStore) GetBuiltinExcludePatterns(_ context.Context) ([]*models.ExcludePattern, error) {
	if m.builtinErr != nil {
		return nil, m.builtinErr
	}
	return m.builtins, nil
}

func (m *mockExcludePatternStore) CreateExcludePattern(_ context.Context, _ *models.ExcludePattern) error {
	return m.createErr
}

func (m *mockExcludePatternStore) UpdateExcludePattern(_ context.Context, _ *models.ExcludePattern) error {
	return m.updateErr
}

func (m *mockExcludePatternStore) DeleteExcludePattern(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockExcludePatternStore) SeedBuiltinExcludePatterns(_ context.Context, _ []*models.ExcludePattern) error {
	return m.seedErr
}

func setupExcludePatternsTestRouter(store ExcludePatternStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewExcludePatternsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestExcludePatternsList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		p1 := &models.ExcludePattern{ID: uuid.New(), OrgID: &orgID, Name: "Test", Patterns: []string{"*.tmp"}, Category: "temp"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{p1}}
		r := setupExcludePatternsTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if _, ok := body["patterns"]; !ok {
			t.Fatal("expected 'patterns' key")
		}
	})

	t.Run("with category filter", func(t *testing.T) {
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns?category=os"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("category error", func(t *testing.T) {
		store := &mockExcludePatternStore{catErr: errors.New("db error")}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns?category=os"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, testUserNoOrg())
		r := setupExcludePatternsTestRouter(store, TestUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockExcludePatternStore{listErr: errors.New("db error")}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestExcludePatternsLibrary(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns/library"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})
}

func TestExcludePatternsCategories(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns/categories"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})
}

func TestExcludePatternsGet(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)

	t.Run("success custom pattern", func(t *testing.T) {
		p := &models.ExcludePattern{ID: uuid.New(), OrgID: &orgID, Name: "Test", Patterns: []string{"*.log"}, Category: "logs"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{p}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns/"+p.ID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("success builtin pattern", func(t *testing.T) {
		p := &models.ExcludePattern{ID: uuid.New(), Name: "OS Default", IsBuiltin: true, Patterns: []string{"*.DS_Store"}, Category: "os"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{p}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns/"+p.ID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns/bad-id"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockExcludePatternStore{getErr: errors.New("not found")}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("wrong org non-builtin", func(t *testing.T) {
		otherOrg := uuid.New()
		p := &models.ExcludePattern{ID: uuid.New(), OrgID: &otherOrg, Name: "Other", Patterns: []string{"*.log"}, Category: "logs"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{p}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/exclude-patterns/"+p.ID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestExcludePatternsCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)

	t.Run("success", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, user)
		body := `{"name":"My Pattern","patterns":["*.tmp","*.bak"],"category":"temp"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/exclude-patterns", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing name", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, user)
		body := `{"patterns":["*.tmp"],"category":"temp"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/exclude-patterns", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, user)
		body := `{"name":"Test","patterns":["*.tmp"],"category":"nonexistent"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/exclude-patterns", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("no org", func(t *testing.T) {
		store := &mockExcludePatternStore{}
		r := setupExcludePatternsTestRouter(store, testUserNoOrg())
		r := setupExcludePatternsTestRouter(store, TestUserNoOrg())
		body := `{"name":"Test","patterns":["*.tmp"],"category":"temp"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/exclude-patterns", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockExcludePatternStore{createErr: errors.New("db error")}
		r := setupExcludePatternsTestRouter(store, user)
		body := `{"name":"Test","patterns":["*.tmp"],"category":"temp"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/exclude-patterns", body))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestExcludePatternsUpdate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)
	p := &models.ExcludePattern{ID: uuid.New(), OrgID: &orgID, Name: "Test", Patterns: []string{"*.tmp"}, Category: "temp"}

	t.Run("success", func(t *testing.T) {
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{p}}
		r := setupExcludePatternsTestRouter(store, user)
		body := `{"name":"Updated"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/exclude-patterns/"+p.ID.String(), body))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("builtin forbidden", func(t *testing.T) {
		builtin := &models.ExcludePattern{ID: uuid.New(), Name: "Builtin", IsBuiltin: true, Patterns: []string{"*.DS_Store"}, Category: "os"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{builtin}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/exclude-patterns/"+builtin.ID.String(), `{"name":"hack"}`))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherOrg := uuid.New()
		otherP := &models.ExcludePattern{ID: uuid.New(), OrgID: &otherOrg, Name: "Other", Patterns: []string{"*.log"}, Category: "logs"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{otherP}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/exclude-patterns/"+otherP.ID.String(), `{"name":"hack"}`))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("invalid category", func(t *testing.T) {
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{p}}
		r := setupExcludePatternsTestRouter(store, user)
		body := `{"category":"nonexistent"}`
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/exclude-patterns/"+p.ID.String(), body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockExcludePatternStore{
			patterns:  []*models.ExcludePattern{p},
			updateErr: errors.New("db error"),
		}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, JSONRequest("PUT", "/api/v1/exclude-patterns/"+p.ID.String(), `{"name":"x"}`))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestExcludePatternsDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	user := TestUser(orgID)
	p := &models.ExcludePattern{ID: uuid.New(), OrgID: &orgID, Name: "Test", Patterns: []string{"*.tmp"}, Category: "temp"}

	t.Run("success", func(t *testing.T) {
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{p}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/exclude-patterns/"+p.ID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("builtin forbidden", func(t *testing.T) {
		builtin := &models.ExcludePattern{ID: uuid.New(), Name: "Builtin", IsBuiltin: true, Patterns: []string{"*.DS_Store"}, Category: "os"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{builtin}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/exclude-patterns/"+builtin.ID.String()))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherOrg := uuid.New()
		otherP := &models.ExcludePattern{ID: uuid.New(), OrgID: &otherOrg, Name: "Other", Patterns: []string{"*.log"}, Category: "logs"}
		store := &mockExcludePatternStore{patterns: []*models.ExcludePattern{otherP}}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/exclude-patterns/"+otherP.ID.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockExcludePatternStore{getErr: errors.New("not found")}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/exclude-patterns/"+uuid.New().String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		store := &mockExcludePatternStore{
			patterns:  []*models.ExcludePattern{p},
			deleteErr: errors.New("db error"),
		}
		r := setupExcludePatternsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/exclude-patterns/"+p.ID.String()))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
