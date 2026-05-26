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

type mockUserSessionStore struct {
	sessions     []*models.UserSession
	session      *models.UserSession
	listErr      error
	getErr       error
	revokeErr    error
	revokeAllErr error
	revokedCount int64
}

func (m *mockUserSessionStore) ListActiveUserSessionsByUserID(_ context.Context, _ uuid.UUID) ([]*models.UserSession, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.sessions, nil
}

func (m *mockUserSessionStore) GetUserSessionByID(_ context.Context, _ uuid.UUID) (*models.UserSession, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.session, nil
}

func (m *mockUserSessionStore) RevokeUserSession(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return m.revokeErr
}

func (m *mockUserSessionStore) RevokeAllUserSessions(_ context.Context, _ uuid.UUID, _ *uuid.UUID) (int64, error) {
	if m.revokeAllErr != nil {
		return 0, m.revokeAllErr
	}
	return m.revokedCount, nil
}

func setupUserSessionsTestRouter(store UserSessionStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewUserSessionsHandler(store, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestUserSessionsList(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns sessions", func(t *testing.T) {
		store := &mockUserSessionStore{sessions: []*models.UserSession{{ID: uuid.New(), UserID: user.ID, SessionTokenHash: "secret"}}}
		r := setupUserSessionsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users/me/sessions"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockUserSessionStore{listErr: errors.New("db down")}
		r := setupUserSessionsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users/me/sessions"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		store := &mockUserSessionStore{}
		r := setupUserSessionsTestRouter(store, nil)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/users/me/sessions"))
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", resp.Code)
		}
	})
}

func TestUserSessionsRevoke(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	id := uuid.New()

	t.Run("revokes session", func(t *testing.T) {
		store := &mockUserSessionStore{}
		r := setupUserSessionsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/me/sessions/"+id.String()))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("invalid id returns 400", func(t *testing.T) {
		store := &mockUserSessionStore{}
		r := setupUserSessionsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/me/sessions/not-a-uuid"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 404", func(t *testing.T) {
		store := &mockUserSessionStore{revokeErr: errors.New("not found")}
		r := setupUserSessionsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/me/sessions/"+id.String()))
		if resp.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", resp.Code)
		}
	})
}

func TestUserSessionsRevokeAll(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("revokes successfully", func(t *testing.T) {
		store := &mockUserSessionStore{revokedCount: 3}
		r := setupUserSessionsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/me/sessions"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockUserSessionStore{revokeAllErr: errors.New("db down")}
		r := setupUserSessionsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("DELETE", "/api/v1/users/me/sessions"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
