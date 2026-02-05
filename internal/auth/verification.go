// Package auth provides email verification for non-OIDC users.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Common errors for email verification.
var (
	ErrTokenExpired      = errors.New("verification token has expired")
	ErrTokenAlreadyUsed  = errors.New("verification token has already been used")
	ErrTokenNotFound     = errors.New("verification token not found")
	ErrUserAlreadyVerified = errors.New("user email is already verified")
	ErrUserNotFound      = errors.New("user not found")
)

// EmailVerificationToken represents a verification token for email verification.
type EmailVerificationToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"-"` // Never expose token hash
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// NewEmailVerificationToken creates a new verification token for a user.
func NewEmailVerificationToken(userID uuid.UUID, expiresIn time.Duration) (*EmailVerificationToken, string, error) {
	// Generate a secure random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	// Encode token as URL-safe base64
	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Hash the token for storage
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	now := time.Now()
	return &EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(expiresIn),
		CreatedAt: now,
	}, token, nil
}

// HashToken hashes a raw token for comparison.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// IsExpired returns true if the token has expired.
func (t *EmailVerificationToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has been used.
func (t *EmailVerificationToken) IsUsed() bool {
	return t.UsedAt != nil
}

// VerificationStore defines the interface for verification token persistence.
type VerificationStore interface {
	CreateEmailVerificationToken(ctx context.Context, token *EmailVerificationToken) error
	GetEmailVerificationTokenByHash(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)
	MarkEmailVerificationTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	GetUserByID(ctx context.Context, userID uuid.UUID) (VerifiableUser, error)
	SetUserEmailVerified(ctx context.Context, userID uuid.UUID) error
	InvalidateUserVerificationTokens(ctx context.Context, userID uuid.UUID) error
}

// VerifiableUser represents a user that can be verified.
type VerifiableUser interface {
	GetID() uuid.UUID
	GetEmail() string
	IsEmailVerified() bool
	IsOIDCUser() bool
}

// VerificationConfig holds configuration for the verification service.
type VerificationConfig struct {
	TokenExpiration  time.Duration // How long verification tokens are valid
	ResendCooldown   time.Duration // Minimum time between resend requests
	MaxTokensPerUser int           // Maximum active tokens per user
}

// DefaultVerificationConfig returns a VerificationConfig with sensible defaults.
func DefaultVerificationConfig() VerificationConfig {
	return VerificationConfig{
		TokenExpiration:  24 * time.Hour,   // 24 hours
		ResendCooldown:   1 * time.Minute,  // 1 minute cooldown
		MaxTokensPerUser: 5,                // Max 5 active tokens per user
	}
}

// VerificationService handles email verification operations.
type VerificationService struct {
	store  VerificationStore
	config VerificationConfig
	logger zerolog.Logger
}

// NewVerificationService creates a new verification service.
func NewVerificationService(store VerificationStore, config VerificationConfig, logger zerolog.Logger) *VerificationService {
	return &VerificationService{
		store:  store,
		config: config,
		logger: logger.With().Str("component", "verification_service").Logger(),
	}
}

// GenerateToken creates a new verification token for a user.
// Returns the raw token that should be sent to the user via email.
func (s *VerificationService) GenerateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	// Check if user exists and needs verification
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}

	if user.IsEmailVerified() {
		return "", ErrUserAlreadyVerified
	}

	// OIDC users don't need email verification
	if user.IsOIDCUser() {
		return "", ErrUserAlreadyVerified
	}

	// Create new token
	token, rawToken, err := NewEmailVerificationToken(userID, s.config.TokenExpiration)
	if err != nil {
		return "", fmt.Errorf("create token: %w", err)
	}

	// Store the token
	if err := s.store.CreateEmailVerificationToken(ctx, token); err != nil {
		return "", fmt.Errorf("store token: %w", err)
	}

	s.logger.Info().
		Str("user_id", userID.String()).
		Time("expires_at", token.ExpiresAt).
		Msg("generated email verification token")

	return rawToken, nil
}

// VerifyToken validates a verification token and marks the user's email as verified.
func (s *VerificationService) VerifyToken(ctx context.Context, rawToken string) error {
	// Hash the provided token for lookup
	tokenHash := HashToken(rawToken)

	// Find the token
	token, err := s.store.GetEmailVerificationTokenByHash(ctx, tokenHash)
	if err != nil {
		s.logger.Debug().Str("token_hash", tokenHash[:8]+"...").Msg("token not found")
		return ErrTokenNotFound
	}

	// Check if token is already used
	if token.IsUsed() {
		s.logger.Debug().Str("token_id", token.ID.String()).Msg("token already used")
		return ErrTokenAlreadyUsed
	}

	// Check if token is expired
	if token.IsExpired() {
		s.logger.Debug().Str("token_id", token.ID.String()).Msg("token expired")
		return ErrTokenExpired
	}

	// Mark token as used
	if err := s.store.MarkEmailVerificationTokenUsed(ctx, token.ID); err != nil {
		return fmt.Errorf("mark token used: %w", err)
	}

	// Mark user as verified
	if err := s.store.SetUserEmailVerified(ctx, token.UserID); err != nil {
		return fmt.Errorf("set user verified: %w", err)
	}

	// Invalidate all other tokens for this user
	if err := s.store.InvalidateUserVerificationTokens(ctx, token.UserID); err != nil {
		// Log but don't fail the verification
		s.logger.Warn().Err(err).Str("user_id", token.UserID.String()).Msg("failed to invalidate other tokens")
	}

	s.logger.Info().
		Str("user_id", token.UserID.String()).
		Str("token_id", token.ID.String()).
		Msg("email verified successfully")

	return nil
}

// ResendVerification generates a new verification token for a user.
// This can be used when the original token expires or is lost.
func (s *VerificationService) ResendVerification(ctx context.Context, userID uuid.UUID) (string, error) {
	// Check if user exists and needs verification
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}

	if user.IsEmailVerified() {
		return "", ErrUserAlreadyVerified
	}

	// OIDC users don't need email verification
	if user.IsOIDCUser() {
		return "", ErrUserAlreadyVerified
	}

	// Invalidate existing tokens
	if err := s.store.InvalidateUserVerificationTokens(ctx, userID); err != nil {
		s.logger.Warn().Err(err).Str("user_id", userID.String()).Msg("failed to invalidate existing tokens")
	}

	// Generate new token
	return s.GenerateToken(ctx, userID)
}

// GetUserVerificationStatus returns the verification status for a user.
type UserVerificationStatus struct {
	UserID        uuid.UUID `json:"user_id"`
	Email         string    `json:"email"`
	IsVerified    bool      `json:"is_verified"`
	IsOIDCUser    bool      `json:"is_oidc_user"`
	RequiresEmail bool      `json:"requires_email"` // True if user needs to verify email
}

// GetUserVerificationStatus returns verification status for a user.
func (s *VerificationService) GetUserVerificationStatus(ctx context.Context, userID uuid.UUID) (*UserVerificationStatus, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	return &UserVerificationStatus{
		UserID:        user.GetID(),
		Email:         user.GetEmail(),
		IsVerified:    user.IsEmailVerified(),
		IsOIDCUser:    user.IsOIDCUser(),
		RequiresEmail: !user.IsEmailVerified() && !user.IsOIDCUser(),
	}, nil
}

// BuildVerificationURL builds the verification URL for an email.
func BuildVerificationURL(baseURL, token string) string {
	return fmt.Sprintf("%s/verify-email?token=%s", baseURL, token)
}
