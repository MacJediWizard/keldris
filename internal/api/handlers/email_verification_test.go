package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockEmailVerificationStore struct {
	token      *auth.EmailVerificationToken
	verifiable auth.VerifiableUser
	user       *models.User
	verified   bool
	security   *settings.SecuritySettings
	err        error
}

func (m *mockEmailVerificationStore) CreateEmailVerificationToken(_ context.Context, _ *auth.EmailVerificationToken) error {
	return m.err
}

func (m *mockEmailVerificationStore) GetEmailVerificationTokenByHash(_ context.Context, _ string) (*auth.EmailVerificationToken, error) {
	return m.token, m.err
}

func (m *mockEmailVerificationStore) MarkEmailVerificationTokenUsed(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockEmailVerificationStore) SetUserEmailVerified(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockEmailVerificationStore) InvalidateUserVerificationTokens(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockEmailVerificationStore) IsUserEmailVerified(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.verified, m.err
}

func (m *mockEmailVerificationStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

func (m *mockEmailVerificationStore) GetUserByIDForVerification(_ context.Context, _ uuid.UUID) (auth.VerifiableUser, error) {
	return m.verifiable, m.err
}

func (m *mockEmailVerificationStore) AdminSetUserEmailVerified(_ context.Context, _ uuid.UUID, _ bool) error {
	return m.err
}

func (m *mockEmailVerificationStore) GetSecuritySettings(_ context.Context, _ uuid.UUID) (*settings.SecuritySettings, error) {
	return m.security, m.err
}

func (m *mockEmailVerificationStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return nil
}

func setupEmailVerificationTestRouter(store EmailVerificationStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewEmailVerificationHandler(store, nil, nil, "http://localhost", zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestEmailVerificationVerifyEmail(t *testing.T) {
	t.Run("missing token returns 400", func(t *testing.T) {
		store := &mockEmailVerificationStore{}
		r := setupEmailVerificationTestRouter(store, testUser(uuid.New()))

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/verify", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		store := &mockEmailVerificationStore{}
		r := setupEmailVerificationTestRouter(store, testUser(uuid.New()))

		resp := DoRequest(r, JSONRequest("POST", "/api/v1/verify", `{invalid`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestEmailVerificationGetStatus(t *testing.T) {
	t.Skip("GetVerificationStatus calls *auth.SessionStore.GetUser directly; requires real session store")
}
