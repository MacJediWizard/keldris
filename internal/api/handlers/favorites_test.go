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

type mockFavoriteStore struct {
	favorite   *models.Favorite
	favorites  []*models.Favorite
	agent      *models.Agent
	schedule   *models.Schedule
	repository *models.Repository
	exists     bool
	listErr    error
	getErr     error
	createErr  error
	deleteErr  error
	existsErr  error
	entityErr  error
}

func (m *mockFavoriteStore) GetFavoritesByUserAndOrg(_ context.Context, _, _ uuid.UUID, _ string) ([]*models.Favorite, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.favorites, nil
}

func (m *mockFavoriteStore) GetFavoriteByUserAndEntity(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (*models.Favorite, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.favorite, nil
}

func (m *mockFavoriteStore) CreateFavorite(_ context.Context, f *models.Favorite) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.favorite = f
	return nil
}

func (m *mockFavoriteStore) DeleteFavorite(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) error {
	return m.deleteErr
}

func (m *mockFavoriteStore) IsFavorite(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	return m.exists, nil
}

func (m *mockFavoriteStore) GetFavoriteEntityIDs(_ context.Context, _, _ uuid.UUID, _ string) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *mockFavoriteStore) GetAgentByID(_ context.Context, _ uuid.UUID) (*models.Agent, error) {
	if m.entityErr != nil {
		return nil, m.entityErr
	}
	return m.agent, nil
}

func (m *mockFavoriteStore) GetScheduleByID(_ context.Context, _ uuid.UUID) (*models.Schedule, error) {
	if m.entityErr != nil {
		return nil, m.entityErr
	}
	return m.schedule, nil
}

func (m *mockFavoriteStore) GetRepositoryByID(_ context.Context, _ uuid.UUID) (*models.Repository, error) {
	if m.entityErr != nil {
		return nil, m.entityErr
	}
	return m.repository, nil
}

func setupFavoritesTestRouter(store FavoriteStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewFavoritesHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestFavoritesList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns favorites", func(t *testing.T) {
		store := &mockFavoriteStore{favorites: []*models.Favorite{{ID: uuid.New(), OrgID: orgID}}}
		r := setupFavoritesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/favorites"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("no org returns 400", func(t *testing.T) {
		store := &mockFavoriteStore{}
		r := setupFavoritesTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/favorites"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockFavoriteStore{listErr: errors.New("db down")}
		r := setupFavoritesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/favorites"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestFavoritesCreate(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	entityID := uuid.New()

	t.Run("creates favorite for agent", func(t *testing.T) {
		store := &mockFavoriteStore{agent: &models.Agent{ID: entityID, OrgID: orgID}}
		r := setupFavoritesTestRouter(store, user)
		body := `{"entity_type":"agent","entity_id":"` + entityID.String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/favorites", body))
		if resp.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid entity type returns 400", func(t *testing.T) {
		store := &mockFavoriteStore{}
		r := setupFavoritesTestRouter(store, user)
		body := `{"entity_type":"bogus","entity_id":"` + entityID.String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/favorites", body))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("already favorited returns 409", func(t *testing.T) {
		store := &mockFavoriteStore{agent: &models.Agent{ID: entityID, OrgID: orgID}, exists: true}
		r := setupFavoritesTestRouter(store, user)
		body := `{"entity_type":"agent","entity_id":"` + entityID.String() + `"}`
		resp := DoRequest(r, JSONRequest("POST", "/api/v1/favorites", body))
		if resp.Code != http.StatusConflict {
			t.Fatalf("expected 409, got %d", resp.Code)
		}
	})
}

func TestFavoritesDelete(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	entityID := uuid.New()

	t.Run("deletes successfully", func(t *testing.T) {
		store := &mockFavoriteStore{}
		r := setupFavoritesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/favorites/agent/"+entityID.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid uuid returns 400", func(t *testing.T) {
		store := &mockFavoriteStore{}
		r := setupFavoritesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/favorites/agent/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid entity type returns 400", func(t *testing.T) {
		store := &mockFavoriteStore{}
		r := setupFavoritesTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/favorites/bogus/"+entityID.String()))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}
