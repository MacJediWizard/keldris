package auth

import (
	"context"
	"errors"
	"fmt"
	"unicode"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Common password validation errors.
var (
	ErrPasswordTooShort      = errors.New("password is too short")
	ErrPasswordNoUppercase   = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase   = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber      = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial     = errors.New("password must contain at least one special character")
	ErrPasswordInHistory     = errors.New("password has been used recently")
	ErrPasswordMismatch      = errors.New("current password is incorrect")
	ErrPasswordExpired       = errors.New("password has expired")
	ErrPasswordChangeRequired = errors.New("password change is required")
)

// PasswordPolicyStore defines the interface for password policy data access.
type PasswordPolicyStore interface {
	GetPasswordPolicyByOrgID(ctx context.Context, orgID uuid.UUID) (*models.PasswordPolicy, error)
	GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*models.PasswordHistory, error)
	CreatePasswordHistory(ctx context.Context, history *models.PasswordHistory) error
	CleanupPasswordHistory(ctx context.Context, userID uuid.UUID, keepCount int) error
}

// PasswordValidator handles password validation against policies.
type PasswordValidator struct {
	store PasswordPolicyStore
}

// NewPasswordValidator creates a new PasswordValidator.
func NewPasswordValidator(store PasswordPolicyStore) *PasswordValidator {
	return &PasswordValidator{store: store}
}

// ValidationResult contains the result of password validation.
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ValidatePassword validates a password against the organization's policy.
func (v *PasswordValidator) ValidatePassword(ctx context.Context, orgID uuid.UUID, password string) (*ValidationResult, error) {
	policy, err := v.store.GetPasswordPolicyByOrgID(ctx, orgID)
	if err != nil {
		// If no policy exists, use default policy
		policy = models.NewPasswordPolicy(orgID)
	}

	return v.ValidateAgainstPolicy(password, policy), nil
}

// ValidateAgainstPolicy validates a password against a specific policy.
func (v *PasswordValidator) ValidateAgainstPolicy(password string, policy *models.PasswordPolicy) *ValidationResult {
	result := &ValidationResult{Valid: true}

	// Check minimum length
	if len(password) < policy.MinLength {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Password must be at least %d characters long", policy.MinLength))
	}

	// Check character requirements
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasNumber = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		}
	}

	if policy.RequireUppercase && !hasUpper {
		result.Valid = false
		result.Errors = append(result.Errors, "Password must contain at least one uppercase letter")
	}

	if policy.RequireLowercase && !hasLower {
		result.Valid = false
		result.Errors = append(result.Errors, "Password must contain at least one lowercase letter")
	}

	if policy.RequireNumber && !hasNumber {
		result.Valid = false
		result.Errors = append(result.Errors, "Password must contain at least one number")
	}

	if policy.RequireSpecial && !hasSpecial {
		result.Valid = false
		result.Errors = append(result.Errors, "Password must contain at least one special character")
	}

	// Add warnings for weak passwords that still pass policy
	if result.Valid {
		if len(password) < 12 {
			result.Warnings = append(result.Warnings, "Consider using a longer password for better security")
		}
		if !hasSpecial && !policy.RequireSpecial {
			result.Warnings = append(result.Warnings, "Consider adding special characters for better security")
		}
	}

	return result
}

// ValidatePasswordWithHistory validates a password against policy and history.
func (v *PasswordValidator) ValidatePasswordWithHistory(ctx context.Context, orgID, userID uuid.UUID, password string) (*ValidationResult, error) {
	// First validate against policy
	result, err := v.ValidatePassword(ctx, orgID, password)
	if err != nil {
		return nil, err
	}

	// Get the policy for history count
	policy, err := v.store.GetPasswordPolicyByOrgID(ctx, orgID)
	if err != nil {
		policy = models.NewPasswordPolicy(orgID)
	}

	// Check password history if enabled
	if policy.HistoryCount > 0 {
		history, err := v.store.GetPasswordHistory(ctx, userID, policy.HistoryCount)
		if err != nil {
			return nil, fmt.Errorf("check password history: %w", err)
		}

		for _, h := range history {
			if err := bcrypt.CompareHashAndPassword([]byte(h.PasswordHash), []byte(password)); err == nil {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Password has been used in the last %d passwords", policy.HistoryCount))
				break
			}
		}
	}

	return result, nil
}

// HashPassword creates a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword compares a password with its hash.
func VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// RecordPasswordChange records a password change in history.
func (v *PasswordValidator) RecordPasswordChange(ctx context.Context, orgID, userID uuid.UUID, passwordHash string) error {
	// Get policy for history count
	policy, err := v.store.GetPasswordPolicyByOrgID(ctx, orgID)
	if err != nil {
		policy = models.NewPasswordPolicy(orgID)
	}

	// Record the new password in history
	if policy.HistoryCount > 0 {
		history := models.NewPasswordHistory(userID, passwordHash)
		if err := v.store.CreatePasswordHistory(ctx, history); err != nil {
			return fmt.Errorf("record password history: %w", err)
		}

		// Clean up old history entries
		if err := v.store.CleanupPasswordHistory(ctx, userID, policy.HistoryCount); err != nil {
			// Log but don't fail the password change
			// This is a housekeeping operation
		}
	}

	return nil
}

// DefaultPasswordPolicy returns a default password policy for organizations without one.
func DefaultPasswordPolicy() *models.PasswordPolicy {
	return models.NewPasswordPolicy(uuid.Nil)
}
