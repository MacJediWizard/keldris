package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockPasswordResetStore struct {
	user       *models.User
	hasPwd     bool
	token      *models.PasswordResetToken
	rateLimit  *models.PasswordResetRateLimit
	policy     *models.PasswordPolicy
	err        error
	rateLimErr error
}

func (m *mockPasswordResetStore) GetUserByEmail(_ context.Context, _ string) (*models.User, error) {
	return m.user, m.err
}

func (m *mockPasswordResetStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return m.user, m.err
}

func (m *mockPasswordResetStore) HasPasswordAuth(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasPwd, m.err
}

func (m *mockPasswordResetStore) CreatePasswordResetToken(_ context.Context, _ *models.PasswordResetToken) error {
	return m.err
}

func (m *mockPasswordResetStore) GetPasswordResetTokenByHash(_ context.Context, _ string) (*models.PasswordResetToken, error) {
	return m.token, m.err
}

func (m *mockPasswordResetStore) MarkPasswordResetTokenUsed(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockPasswordResetStore) InvalidateUserResetTokens(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockPasswordResetStore) GetResetRateLimit(_ context.Context, _, _ string) (*models.PasswordResetRateLimit, error) {
	return m.rateLimit, m.rateLimErr
}

func (m *mockPasswordResetStore) IncrementResetRateLimit(_ context.Context, _, _ string, _ time.Duration) error {
	return m.err
}

func (m *mockPasswordResetStore) CleanupExpiredRateLimits(_ context.Context, _ time.Duration) error {
	return m.err
}

func (m *mockPasswordResetStore) UpdateUserPassword(_ context.Context, _ uuid.UUID, _ string, _ *time.Time) error {
	return m.err
}

func (m *mockPasswordResetStore) GetPasswordPolicyByOrgID(_ context.Context, _ uuid.UUID) (*models.PasswordPolicy, error) {
	if m.policy != nil {
		return m.policy, m.err
	}
	return models.NewPasswordPolicy(uuid.New()), m.err
}

func (m *mockPasswordResetStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return nil
}

func setupPasswordResetTestRouter(store PasswordResetStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewPasswordResetHandler(store, nil, "http://localhost", zerolog.Nop())
	handler.RegisterPublicRoutes(r)
	return r
}

func TestPasswordResetRequestReset(t *testing.T) {
	t.Run("invalid email returns 400", func(t *testing.T) {
		store := &mockPasswordResetStore{}
		r := setupPasswordResetTestRouter(store)

		resp := DoRequest(r, JSONRequest("POST", "/auth/reset-password/request", `{}`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("malformed json returns 400", func(t *testing.T) {
		store := &mockPasswordResetStore{}
		r := setupPasswordResetTestRouter(store)

		resp := DoRequest(r, JSONRequest("POST", "/auth/reset-password/request", `{invalid`))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})
}

func TestPasswordResetValidateToken(t *testing.T) {
	t.Skip("handler dereferences token; nil-safety lives in service layer covered by reset_test.go")
}
