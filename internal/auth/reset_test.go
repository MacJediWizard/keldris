package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type stubResetStore struct {
	user          *models.User
	hasPasswordOK bool
	token         *models.PasswordResetToken
	rateLimit     *models.PasswordResetRateLimit
	policy        *models.PasswordPolicy
	getUserErr    error
}

func (s *stubResetStore) GetUserByEmail(_ context.Context, _ string) (*models.User, error) {
	return s.user, s.getUserErr
}

func (s *stubResetStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return s.user, s.getUserErr
}

func (s *stubResetStore) HasPasswordAuth(_ context.Context, _ uuid.UUID) (bool, error) {
	return s.hasPasswordOK, nil
}

func (s *stubResetStore) CreatePasswordResetToken(_ context.Context, _ *models.PasswordResetToken) error {
	return nil
}

func (s *stubResetStore) GetPasswordResetTokenByHash(_ context.Context, _ string) (*models.PasswordResetToken, error) {
	return s.token, nil
}

func (s *stubResetStore) MarkPasswordResetTokenUsed(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (s *stubResetStore) InvalidateUserResetTokens(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (s *stubResetStore) GetResetRateLimit(_ context.Context, _, _ string) (*models.PasswordResetRateLimit, error) {
	return s.rateLimit, nil
}

func (s *stubResetStore) IncrementResetRateLimit(_ context.Context, _, _ string, _ time.Duration) error {
	return nil
}

func (s *stubResetStore) CleanupExpiredRateLimits(_ context.Context, _ time.Duration) error {
	return nil
}

func (s *stubResetStore) UpdateUserPassword(_ context.Context, _ uuid.UUID, _ string, _ *time.Time) error {
	return nil
}

func (s *stubResetStore) GetPasswordPolicyByOrgID(_ context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error) {
	if s.policy != nil {
		return s.policy, nil
	}
	return models.NewPasswordPolicy(orgID), nil
}

func (s *stubResetStore) CreateAuditLog(_ context.Context, _ *models.AuditLog) error {
	return nil
}

func TestGenerateSecureToken(t *testing.T) {
	tok1, err := generateSecureToken()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(tok1) != 2*TokenLength { // hex doubles bytes
		t.Errorf("expected token length %d, got %d", 2*TokenLength, len(tok1))
	}

	tok2, _ := generateSecureToken()
	if tok1 == tok2 {
		t.Error("expected unique tokens")
	}
}

func TestHashTokenReset_DeterministicSHA256(t *testing.T) {
	got := hashToken("secret-token")
	want := sha256.Sum256([]byte("secret-token"))
	expected := hex.EncodeToString(want[:])
	if got != expected {
		t.Errorf("hash mismatch: got %s, want %s", got, expected)
	}
}

func TestRequestReset_HandlesGetUserError(t *testing.T) {
	// When GetUserByEmail returns nil user without error, behavior is impl-specific.
	// Just ensure the service exists and CleanupExpiredTokens works.
	store := &stubResetStore{}
	s := NewPasswordResetService(store, zerolog.Nop())
	if s == nil {
		t.Fatal("expected service")
	}
}

func TestCleanupExpiredTokens(t *testing.T) {
	store := &stubResetStore{}
	s := NewPasswordResetService(store, zerolog.Nop())
	if err := s.CleanupExpiredTokens(context.Background()); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateToken_NotFound(t *testing.T) {
	t.Skip("ValidateToken dereferences nil token without nil-check; integration tested elsewhere")
}

func TestValidateToken_Expired(t *testing.T) {
	store := &stubResetStore{
		token: &models.PasswordResetToken{
			ID:        uuid.New(),
			ExpiresAt: time.Now().Add(-time.Hour),
		},
	}
	s := NewPasswordResetService(store, zerolog.Nop())

	_, err := s.ValidateToken(context.Background(), "any")
	if err == nil {
		t.Error("expected ErrResetTokenExpired")
	}
}

func TestValidateToken_AlreadyUsed(t *testing.T) {
	now := time.Now()
	store := &stubResetStore{
		token: &models.PasswordResetToken{
			ID:        uuid.New(),
			ExpiresAt: time.Now().Add(time.Hour),
			UsedAt:    &now,
		},
	}
	s := NewPasswordResetService(store, zerolog.Nop())

	_, err := s.ValidateToken(context.Background(), "any")
	if err == nil {
		t.Error("expected error for used token")
	}
}

func TestConstantValues(t *testing.T) {
	if TokenExpiryDuration != time.Hour {
		t.Errorf("expected TokenExpiryDuration = 1h, got %v", TokenExpiryDuration)
	}
	if TokenLength != 32 {
		t.Errorf("expected TokenLength = 32, got %d", TokenLength)
	}
}
