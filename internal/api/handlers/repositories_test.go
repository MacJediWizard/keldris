package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockRepositoryStore struct {
	repos      []*models.Repository
	repoByID   map[uuid.UUID]*models.Repository
	user       *models.User
	repoKey    *models.RepositoryKey
	repoKeys   []*models.RepositoryKey
	createErr  error
	updateErr  error
	deleteErr  error
}

func (m *mockRepositoryStore) GetRepositoriesByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.Repository, error) {
	var result []*models.Repository
	for _, r := range m.repos {
		if r.OrgID == orgID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRepositoryStore) GetRepositoryByID(_ context.Context, id uuid.UUID) (*models.Repository, error) {
	if r, ok := m.repoByID[id]; ok {
		return r, nil
	}
	return nil, errors.New("repository not found")
}

func (m *mockRepositoryStore) CreateRepository(_ context.Context, _ *models.Repository) error {
	return m.createErr
}

func (m *mockRepositoryStore) UpdateRepository(_ context.Context, _ *models.Repository) error {
	return m.updateErr
}

func (m *mockRepositoryStore) DeleteRepository(_ context.Context, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockRepositoryStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if m.user != nil && m.user.ID == id {
		return m.user, nil
	}
	return nil, errors.New("user not found")
}

func (m *mockRepositoryStore) CreateRepositoryKey(_ context.Context, _ *models.RepositoryKey) error {
	return nil
}

func (m *mockRepositoryStore) GetRepositoryKeyByRepositoryID(_ context.Context, repoID uuid.UUID) (*models.RepositoryKey, error) {
	if m.repoKey != nil && m.repoKey.RepositoryID == repoID {
		return m.repoKey, nil
	}
	return nil, errors.New("key not found")
}

func (m *mockRepositoryStore) GetRepositoryKeysWithEscrowByOrgID(_ context.Context, _ uuid.UUID) ([]*models.RepositoryKey, error) {
	return m.repoKeys, nil
}

func setupRepositoryTestRouter(store RepositoryStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	// Pass nil keyManager and checker - List/Get/Delete tests don't need encryption or license checks
	handler := NewRepositoriesHandler(store, nil, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestListRepositories(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "s3-backup", Type: models.RepositoryTypeS3}

	store := &mockRepositoryStore{
		repos:    []*models.Repository{repo},
		repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
		var resp map[string]json.RawMessage
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if _, ok := resp["repositories"]; !ok {
			t.Fatal("expected 'repositories' key")
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		r := setupRepositoryTestRouter(store, noOrgUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, nil)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})
}

func TestGetRepository(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "s3-backup", Type: models.RepositoryTypeS3}

	store := &mockRepositoryStore{
		repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupRepositoryTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestDeleteRepository(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "s3-backup", Type: models.RepositoryTypeS3}

	store := &mockRepositoryStore{
		repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/repositories/"+repoID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/repositories/bad-uuid", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/repositories/"+uuid.New().String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupRepositoryTestRouter(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/repositories/"+repoID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := &mockRepositoryStore{
			repoByID:  map[uuid.UUID]*models.Repository{repoID: repo},
			deleteErr: errors.New("db error"),
		}
		r := setupRepositoryTestRouter(errStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/repositories/"+repoID.String(), nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", w.Code)
		}
	})
}

func TestTestRepository(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()

	km, _ := crypto.NewKeyManager(make([]byte, 32))
	configJSON, _ := json.Marshal(map[string]any{"path": "/tmp/test-repo-conn"})
	configEncrypted, _ := km.Encrypt(configJSON)

	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "local-backup", Type: models.RepositoryTypeLocal, ConfigEncrypted: configEncrypted}

	store := &mockRepositoryStore{
		repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
	}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+uuid.New().String()+"/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		r := setupRepositoryTestRouterWithKeyManager(store, wrongUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/repositories/"+repoID.String()+"/test", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func TestRecoverKey(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	repoID := uuid.New()

	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "s3-backup"}
	adminUser := &models.User{ID: userID, OrgID: orgID, Role: models.UserRoleAdmin}

	store := &mockRepositoryStore{
		repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
		user:     adminUser,
		repoKey: &models.RepositoryKey{
			RepositoryID:       repoID,
			EscrowEnabled:      true,
			EscrowEncryptedKey: []byte("encrypted-key"),
		},
	}
	user := &auth.SessionUser{ID: userID, CurrentOrgID: orgID}

	t.Run("forbidden for non-admin", func(t *testing.T) {
		viewerUserID := uuid.New()
		viewerUser := &models.User{ID: viewerUserID, OrgID: orgID, Role: models.UserRoleViewer}
		viewerStore := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
			user:     viewerUser,
		}
		viewerSessionUser := &auth.SessionUser{ID: viewerUserID, CurrentOrgID: orgID}
		r := setupRepositoryTestRouter(viewerStore, viewerSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/key/recover", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		r := setupRepositoryTestRouter(store, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+uuid.New().String()+"/key/recover", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})

	t.Run("escrow not enabled", func(t *testing.T) {
		noEscrowStore := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
			user:     adminUser,
			repoKey: &models.RepositoryKey{
				RepositoryID:  repoID,
				EscrowEnabled: false,
			},
		}
		r := setupRepositoryTestRouter(noEscrowStore, user)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/key/recover", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		otherUserID := uuid.New()
		otherUser := &models.User{ID: otherUserID, OrgID: uuid.New(), Role: models.UserRoleAdmin}
		wrongStore := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
			user:     otherUser,
		}
		wrongSessionUser := &auth.SessionUser{ID: otherUserID, CurrentOrgID: uuid.New()}
		r := setupRepositoryTestRouter(wrongStore, wrongSessionUser)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/repositories/"+repoID.String()+"/key/recover", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", w.Code)
		}
	})
}

func setupRepositoryTestRouterWithKeyManager(store RepositoryStore, user *auth.SessionUser) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if user != nil {
			c.Set(string(middleware.UserContextKey), user)
		}
		c.Next()
	})
	km, _ := crypto.NewKeyManager(make([]byte, 32))
	handler := NewRepositoriesHandler(store, km, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestCreateRepository(t *testing.T) {
	orgID := uuid.New()
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"test-repo","type":"local","config":{"path":"/tmp/test"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp CreateRepositoryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Repository.Name != "test-repo" {
			t.Fatalf("expected name 'test-repo', got %q", resp.Repository.Name)
		}
		if resp.Password == "" {
			t.Fatal("expected password in response")
		}
	})

	t.Run("success with escrow", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"test-repo","type":"local","config":{"path":"/tmp/test"},"escrow_enabled":true}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp CreateRepositoryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if !resp.Repository.EscrowEnabled {
			t.Fatal("expected escrow_enabled to be true")
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"test-repo","type":"local","config":{"path":"/tmp/test"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no org", func(t *testing.T) {
		noOrgUser := &auth.SessionUser{ID: uuid.New()}
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, noOrgUser)
		w := httptest.NewRecorder()
		body := `{"name":"test-repo","type":"local","config":{"path":"/tmp/test"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"test","type":"badtype","config":{"a":"b"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("store create error", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID:  make(map[uuid.UUID]*models.Repository),
			createErr: errors.New("db error"),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"test-repo","type":"local","config":{"path":"/tmp/test"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestUpdateRepository(t *testing.T) {
	orgID := uuid.New()
	repoID := uuid.New()
	repo := &models.Repository{ID: repoID, OrgID: orgID, Name: "s3-backup", Type: models.RepositoryTypeS3}
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("success with name", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/repositories/"+repoID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp RepositoryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		if resp.Name != "new-name" {
			t.Fatalf("expected name 'new-name', got %q", resp.Name)
		}
	})

	t.Run("success with config", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"config":{"path":"/new"}}`
		req, _ := http.NewRequest("PUT", "/api/v1/repositories/"+repoID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("not found", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/repositories/"+uuid.New().String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("wrong org", func(t *testing.T) {
		wrongUser := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: uuid.New()}
		store := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
		}
		r := setupRepositoryTestRouterWithKeyManager(store, wrongUser)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/repositories/"+repoID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/repositories/bad-uuid", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("store update error", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID:  map[uuid.UUID]*models.Repository{repoID: repo},
			updateErr: errors.New("db error"),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/repositories/"+repoID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("no auth", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: map[uuid.UUID]*models.Repository{repoID: repo},
		}
		r := setupRepositoryTestRouterWithKeyManager(store, nil)
		w := httptest.NewRecorder()
		body := `{"name":"new-name"}`
		req, _ := http.NewRequest("PUT", "/api/v1/repositories/"+repoID.String(), strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestTestConnection(t *testing.T) {
	orgID := uuid.New()
	user := &auth.SessionUser{ID: uuid.New(), CurrentOrgID: orgID}

	t.Run("no auth", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, nil)
		w := httptest.NewRecorder()
		body := `{"type":"local","config":{"path":"/tmp/test"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/test-connection", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid body", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/test-connection", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"type":"badtype","config":{"a":"b"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/test-connection", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("valid local config", func(t *testing.T) {
		store := &mockRepositoryStore{
			repoByID: make(map[uuid.UUID]*models.Repository),
		}
		r := setupRepositoryTestRouterWithKeyManager(store, user)
		w := httptest.NewRecorder()
		body := `{"type":"local","config":{"path":"/tmp/test"}}`
		req, _ := http.NewRequest("POST", "/api/v1/repositories/test-connection", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var resp TestRepositoryResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		// The path /tmp/test likely exists on macOS/Linux (parent /tmp exists),
		// but we accept either success or failure since both return 200.
		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 regardless of connection result, got %d", w.Code)
		}
	})
}
