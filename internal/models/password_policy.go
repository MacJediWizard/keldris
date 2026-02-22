package models

import (
	"time"

	"github.com/google/uuid"
)

// PasswordPolicy represents an organization's password requirements.
type PasswordPolicy struct {
	ID               uuid.UUID `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	MinLength        int       `json:"min_length"`
	RequireUppercase bool      `json:"require_uppercase"`
	RequireLowercase bool      `json:"require_lowercase"`
	RequireNumber    bool      `json:"require_number"`
	RequireSpecial   bool      `json:"require_special"`
	MaxAgeDays       *int      `json:"max_age_days,omitempty"` // nil means no expiration
	HistoryCount     int       `json:"history_count"`          // Number of previous passwords to remember
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewPasswordPolicy creates a new PasswordPolicy with default values.
func NewPasswordPolicy(orgID uuid.UUID) *PasswordPolicy {
	now := time.Now()
	return &PasswordPolicy{
		ID:               uuid.New(),
		OrgID:            orgID,
		MinLength:        8,
		RequireUppercase: true,
		RequireLowercase: true,
		RequireNumber:    true,
		RequireSpecial:   false,
		MaxAgeDays:       nil,
		HistoryCount:     0,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// HasExpiration returns true if the policy requires password expiration.
func (p *PasswordPolicy) HasExpiration() bool {
	return p.MaxAgeDays != nil && *p.MaxAgeDays > 0
}

// CalculateExpirationDate calculates the expiration date based on the policy.
func (p *PasswordPolicy) CalculateExpirationDate(from time.Time) *time.Time {
	if !p.HasExpiration() {
		return nil
	}
	expires := from.AddDate(0, 0, *p.MaxAgeDays)
	return &expires
}

// PasswordHistory represents a historical password entry for a user.
type PasswordHistory struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	PasswordHash string    `json:"-"` // Never expose hash in JSON
	CreatedAt    time.Time `json:"created_at"`
}

// NewPasswordHistory creates a new PasswordHistory entry.
func NewPasswordHistory(userID uuid.UUID, passwordHash string) *PasswordHistory {
	return &PasswordHistory{
		ID:           uuid.New(),
		UserID:       userID,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
}

// PasswordRequirements describes the password requirements in a human-readable format.
type PasswordRequirements struct {
	MinLength        int    `json:"min_length"`
	RequireUppercase bool   `json:"require_uppercase"`
	RequireLowercase bool   `json:"require_lowercase"`
	RequireNumber    bool   `json:"require_number"`
	RequireSpecial   bool   `json:"require_special"`
	MaxAgeDays       *int   `json:"max_age_days,omitempty"`
	Description      string `json:"description"`
}

// GetRequirements returns the human-readable password requirements.
func (p *PasswordPolicy) GetRequirements() PasswordRequirements {
	req := PasswordRequirements{
		MinLength:        p.MinLength,
		RequireUppercase: p.RequireUppercase,
		RequireLowercase: p.RequireLowercase,
		RequireNumber:    p.RequireNumber,
		RequireSpecial:   p.RequireSpecial,
		MaxAgeDays:       p.MaxAgeDays,
	}

	// Build description
	desc := "Password must be at least " + string(rune('0'+p.MinLength/10)) + string(rune('0'+p.MinLength%10)) + " characters"
	parts := []string{}
	if p.RequireUppercase {
		parts = append(parts, "uppercase letter")
	}
	if p.RequireLowercase {
		parts = append(parts, "lowercase letter")
	}
	if p.RequireNumber {
		parts = append(parts, "number")
	}
	if p.RequireSpecial {
		parts = append(parts, "special character")
	}
	if len(parts) > 0 {
		desc += " and include at least one "
		for i, part := range parts {
			if i > 0 {
				if i == len(parts)-1 {
					desc += ", and "
				} else {
					desc += ", "
				}
			}
			desc += part
		}
	}
	desc += "."
	req.Description = desc

	return req
}

// UpdatePasswordPolicyRequest is the request body for updating a password policy.
type UpdatePasswordPolicyRequest struct {
	MinLength        *int  `json:"min_length,omitempty" binding:"omitempty,min=6,max=128"`
	RequireUppercase *bool `json:"require_uppercase,omitempty"`
	RequireLowercase *bool `json:"require_lowercase,omitempty"`
	RequireNumber    *bool `json:"require_number,omitempty"`
	RequireSpecial   *bool `json:"require_special,omitempty"`
	MaxAgeDays       *int  `json:"max_age_days,omitempty" binding:"omitempty,min=0,max=365"`
	HistoryCount     *int  `json:"history_count,omitempty" binding:"omitempty,min=0,max=24"`
}

// PasswordPolicyResponse is the response for password policy endpoints.
type PasswordPolicyResponse struct {
	Policy       PasswordPolicy       `json:"policy"`
	Requirements PasswordRequirements `json:"requirements"`
}

// ChangePasswordRequest is the request body for changing a user's password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// SetPasswordRequest is the request body for setting a password (first time or reset).
type SetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
}

// PasswordLoginRequest is the request body for password-based login.
type PasswordLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// PasswordExpirationInfo contains information about password expiration status.
type PasswordExpirationInfo struct {
	IsExpired         bool       `json:"is_expired"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	DaysUntilExpiry   *int       `json:"days_until_expiry,omitempty"`
	MustChangeNow     bool       `json:"must_change_now"`
	WarnDaysRemaining int        `json:"warn_days_remaining"` // Show warning when this many days remain
}

// GetExpirationInfo returns expiration information for a user's password.
func GetExpirationInfo(expiresAt *time.Time, mustChange bool) PasswordExpirationInfo {
	info := PasswordExpirationInfo{
		MustChangeNow:     mustChange,
		WarnDaysRemaining: 14, // Default warning threshold
	}

	if expiresAt == nil {
		return info
	}

	now := time.Now()
	info.ExpiresAt = expiresAt
	info.IsExpired = now.After(*expiresAt)

	if !info.IsExpired {
		days := int(expiresAt.Sub(now).Hours() / 24)
		info.DaysUntilExpiry = &days
	}

	return info
}

// UserPasswordInfo holds password-related user information for authentication.
type UserPasswordInfo struct {
	UserID             uuid.UUID  `json:"user_id"`
	PasswordHash       *string    `json:"-"` // Never expose in JSON
	PasswordChangedAt  *time.Time `json:"password_changed_at,omitempty"`
	PasswordExpiresAt  *time.Time `json:"password_expires_at,omitempty"`
	MustChangePassword bool       `json:"must_change_password"`
}
