package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

const (
	// TokenExpiryDuration is the duration before a reset token expires.
	TokenExpiryDuration = 1 * time.Hour
	// TokenLength is the length of the reset token in bytes (before hex encoding).
	TokenLength = 32
	// MaxResetAttemptsPerEmail is the maximum number of reset requests per email per window.
	MaxResetAttemptsPerEmail = 3
	// MaxResetAttemptsPerIP is the maximum number of reset requests per IP per window.
	MaxResetAttemptsPerIP = 5
	// RateLimitWindow is the duration of the rate limit window.
	RateLimitWindow = 15 * time.Minute
)

// Common password reset errors.
var (
	ErrResetTokenExpired  = errors.New("reset token has expired")
	ErrResetTokenUsed     = errors.New("reset token has already been used")
	ErrResetTokenInvalid  = errors.New("invalid reset token")
	ErrResetRateLimited   = errors.New("too many password reset requests")
	ErrUserNoPasswordAuth = errors.New("user does not have password authentication")
	ErrUserOIDCOnly       = errors.New("user uses OIDC authentication only")
)

// PasswordResetStore defines the interface for password reset data access.
type PasswordResetStore interface {
	// User methods
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	HasPasswordAuth(ctx context.Context, userID uuid.UUID) (bool, error)

	// Token methods
	CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error
	GetPasswordResetTokenByHash(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	InvalidateUserResetTokens(ctx context.Context, userID uuid.UUID) error

	// Rate limiting methods
	GetResetRateLimit(ctx context.Context, identifier, identifierType string) (*models.PasswordResetRateLimit, error)
	IncrementResetRateLimit(ctx context.Context, identifier, identifierType string, windowDuration time.Duration) error
	CleanupExpiredRateLimits(ctx context.Context, windowDuration time.Duration) error

	// Password update
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string, expiresAt *time.Time) error
	GetPasswordPolicyByOrgID(ctx context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error)

	// Audit logging
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// PasswordResetService handles password reset operations.
type PasswordResetService struct {
	store  PasswordResetStore
	logger zerolog.Logger
}

// NewPasswordResetService creates a new PasswordResetService.
func NewPasswordResetService(store PasswordResetStore, logger zerolog.Logger) *PasswordResetService {
	return &PasswordResetService{
		store:  store,
		logger: logger.With().Str("component", "password_reset").Logger(),
	}
}

// ResetRequest contains the result of a password reset request.
type ResetRequest struct {
	Token     string    // The plain-text token to send to the user
	ExpiresAt time.Time // When the token expires
	UserID    uuid.UUID // The user's ID
	UserEmail string    // The user's email
	UserName  string    // The user's name
}

// RequestReset initiates a password reset for the given email.
// Returns the reset token and user info for sending the reset email.
// Always returns success to prevent email enumeration, but only generates
// a token if the user exists and has password auth.
func (s *PasswordResetService) RequestReset(ctx context.Context, email, ipAddress, userAgent string) (*ResetRequest, error) {
	// Check rate limits
	if err := s.checkRateLimits(ctx, email, ipAddress); err != nil {
		s.logger.Warn().
			Str("email", email).
			Str("ip", ipAddress).
			Msg("password reset rate limited")
		return nil, err
	}

	// Increment rate limits (do this even if user not found to prevent enumeration timing attacks)
	defer func() {
		_ = s.store.IncrementResetRateLimit(ctx, email, "email", RateLimitWindow)
		_ = s.store.IncrementResetRateLimit(ctx, ipAddress, "ip", RateLimitWindow)
	}()

	// Look up user by email
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		s.logger.Debug().Str("email", email).Msg("password reset requested for unknown email")
		// Return nil without error to prevent email enumeration
		return nil, nil
	}

	// Check if user has OIDC subject (OIDC-only users can't use password reset)
	if user.OIDCSubject != "" {
		s.logger.Debug().Str("email", email).Msg("password reset requested for OIDC user")
		// Return nil without error to prevent auth method enumeration
		return nil, nil
	}

	// Invalidate any existing reset tokens for this user
	if err := s.store.InvalidateUserResetTokens(ctx, user.ID); err != nil {
		s.logger.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to invalidate existing reset tokens")
	}

	// Generate secure token
	token, err := generateSecureToken()
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to generate reset token")
		return nil, fmt.Errorf("generate reset token: %w", err)
	}

	// Hash the token for storage
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(TokenExpiryDuration)

	// Create reset token record
	resetToken := models.NewPasswordResetToken(user.ID, tokenHash, expiresAt, ipAddress, userAgent)
	if err := s.store.CreatePasswordResetToken(ctx, resetToken); err != nil {
		s.logger.Error().Err(err).Msg("failed to create reset token")
		return nil, fmt.Errorf("create reset token: %w", err)
	}

	// Create audit log
	auditLog := models.NewAuditLog(user.OrgID, models.AuditActionCreate, "password_reset_request", models.AuditResultSuccess).
		WithUser(user.ID).
		WithRequestInfo(ipAddress, userAgent).
		WithDetails("Password reset requested")
	if err := s.store.CreateAuditLog(ctx, auditLog); err != nil {
		s.logger.Warn().Err(err).Msg("failed to create audit log for reset request")
	}

	s.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", email).
		Time("expires_at", expiresAt).
		Msg("password reset token created")

	return &ResetRequest{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    user.ID,
		UserEmail: user.Email,
		UserName:  user.Name,
	}, nil
}

// ValidateToken validates a password reset token and returns the associated user.
func (s *PasswordResetService) ValidateToken(ctx context.Context, token string) (*models.User, error) {
	tokenHash := hashToken(token)

	resetToken, err := s.store.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		s.logger.Debug().Msg("invalid password reset token")
		return nil, ErrResetTokenInvalid
	}

	// Check if token has been used
	if resetToken.UsedAt != nil {
		s.logger.Debug().Str("token_id", resetToken.ID.String()).Msg("reset token already used")
		return nil, ErrResetTokenUsed
	}

	// Check if token has expired
	if time.Now().After(resetToken.ExpiresAt) {
		s.logger.Debug().Str("token_id", resetToken.ID.String()).Msg("reset token expired")
		return nil, ErrResetTokenExpired
	}

	// Get the user
	user, err := s.store.GetUserByID(ctx, resetToken.UserID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", resetToken.UserID.String()).Msg("failed to get user for reset token")
		return nil, ErrUserNotFound
	}

	return user, nil
}

// ResetPassword resets the user's password using a valid token.
func (s *PasswordResetService) ResetPassword(ctx context.Context, token, newPassword, ipAddress, userAgent string) error {
	tokenHash := hashToken(token)

	// Get and validate the token
	resetToken, err := s.store.GetPasswordResetTokenByHash(ctx, tokenHash)
	if err != nil {
		s.logger.Debug().Msg("invalid password reset token")
		return ErrResetTokenInvalid
	}

	// Check if token has been used
	if resetToken.UsedAt != nil {
		s.logger.Debug().Str("token_id", resetToken.ID.String()).Msg("reset token already used")
		return ErrResetTokenUsed
	}

	// Check if token has expired
	if time.Now().After(resetToken.ExpiresAt) {
		s.logger.Debug().Str("token_id", resetToken.ID.String()).Msg("reset token expired")
		return ErrResetTokenExpired
	}

	// Get the user
	user, err := s.store.GetUserByID(ctx, resetToken.UserID)
	if err != nil {
		s.logger.Error().Err(err).Str("user_id", resetToken.UserID.String()).Msg("failed to get user for password reset")
		return ErrUserNotFound
	}

	// Get password policy for expiration calculation
	policy, err := s.store.GetPasswordPolicyByOrgID(ctx, user.OrgID)
	if err != nil {
		policy = models.NewPasswordPolicy(user.OrgID) // Use defaults
	}

	// Hash the new password
	passwordHash, err := HashPassword(newPassword)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to hash new password")
		return fmt.Errorf("hash password: %w", err)
	}

	// Calculate password expiration
	var expiresAt *time.Time
	if policy.HasExpiration() {
		exp := policy.CalculateExpirationDate(time.Now())
		expiresAt = exp
	}

	// Update the password
	if err := s.store.UpdateUserPassword(ctx, user.ID, passwordHash, expiresAt); err != nil {
		s.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to update password")
		return fmt.Errorf("update password: %w", err)
	}

	// Mark the token as used
	if err := s.store.MarkPasswordResetTokenUsed(ctx, resetToken.ID); err != nil {
		s.logger.Warn().Err(err).Str("token_id", resetToken.ID.String()).Msg("failed to mark reset token as used")
	}

	// Invalidate any other reset tokens for this user
	if err := s.store.InvalidateUserResetTokens(ctx, user.ID); err != nil {
		s.logger.Warn().Err(err).Str("user_id", user.ID.String()).Msg("failed to invalidate other reset tokens")
	}

	// Create audit log for successful reset
	auditLog := models.NewAuditLog(user.OrgID, models.AuditActionUpdate, "password", models.AuditResultSuccess).
		WithUser(user.ID).
		WithRequestInfo(ipAddress, userAgent).
		WithDetails("Password reset via email token")
	if err := s.store.CreateAuditLog(ctx, auditLog); err != nil {
		s.logger.Warn().Err(err).Msg("failed to create audit log for password reset")
	}

	s.logger.Info().
		Str("user_id", user.ID.String()).
		Str("email", user.Email).
		Msg("password reset successfully")

	return nil
}

// checkRateLimits checks if the request is within rate limits.
func (s *PasswordResetService) checkRateLimits(ctx context.Context, email, ipAddress string) error {
	// Check email rate limit
	emailLimit, err := s.store.GetResetRateLimit(ctx, email, "email")
	if err == nil && emailLimit != nil {
		if emailLimit.RequestCount >= MaxResetAttemptsPerEmail {
			return ErrResetRateLimited
		}
	}

	// Check IP rate limit
	ipLimit, err := s.store.GetResetRateLimit(ctx, ipAddress, "ip")
	if err == nil && ipLimit != nil {
		if ipLimit.RequestCount >= MaxResetAttemptsPerIP {
			return ErrResetRateLimited
		}
	}

	return nil
}

// generateSecureToken generates a cryptographically secure random token.
func generateSecureToken() (string, error) {
	bytes := make([]byte, TokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of the token.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// CleanupExpiredTokens removes expired reset tokens and rate limit entries.
func (s *PasswordResetService) CleanupExpiredTokens(ctx context.Context) error {
	if err := s.store.CleanupExpiredRateLimits(ctx, RateLimitWindow); err != nil {
		s.logger.Warn().Err(err).Msg("failed to cleanup expired rate limits")
	}
	return nil
}
