package models

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// CustomerStatus defines the status of a customer account.
type CustomerStatus string

const (
	// CustomerStatusActive is a normal active customer account.
	CustomerStatusActive CustomerStatus = "active"
	// CustomerStatusDisabled is an administratively disabled account.
	CustomerStatusDisabled CustomerStatus = "disabled"
	// CustomerStatusPending is an account awaiting email verification.
	CustomerStatusPending CustomerStatus = "pending"
)

// Customer represents a customer who purchases licenses.
type Customer struct {
	ID                  uuid.UUID      `json:"id"`
	Email               string         `json:"email"`
	Name                string         `json:"name"`
	Company             string         `json:"company,omitempty"`
	PasswordHash        string         `json:"-"` // Never expose password hash
	Status              CustomerStatus `json:"status"`
	LastLoginAt         *time.Time     `json:"last_login_at,omitempty"`
	LastLoginIP         string         `json:"last_login_ip,omitempty"`
	FailedLoginAttempts int            `json:"failed_login_attempts,omitempty"`
	LockedUntil         *time.Time     `json:"locked_until,omitempty"`
	ResetToken          string         `json:"-"` // Never expose reset token
	ResetTokenExpiresAt *time.Time     `json:"-"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// NewCustomer creates a new Customer with the given details.
func NewCustomer(email, name, passwordHash string) *Customer {
	now := time.Now()
	return &Customer{
		ID:           uuid.New(),
		Email:        email,
		Name:         name,
		PasswordHash: passwordHash,
		Status:       CustomerStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsActive returns true if the customer account is active.
func (c *Customer) IsActive() bool {
	return c.Status == CustomerStatusActive
}

// IsLocked returns true if the customer account is locked.
func (c *Customer) IsLocked() bool {
	if c.LockedUntil != nil && time.Now().Before(*c.LockedUntil) {
		return true
	}
	return false
}

// HashPassword creates a SHA-256 hash of a password for storage.
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// ComparePasswordHash compares a password with a stored hash using constant-time comparison.
func ComparePasswordHash(password, storedHash string) bool {
	computedHash := HashPassword(password)
	return subtle.ConstantTimeCompare([]byte(computedHash), []byte(storedHash)) == 1
}

// CustomerLoginRequest is the request body for customer login.
type CustomerLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// CustomerRegisterRequest is the request body for customer registration.
type CustomerRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=1,max=255"`
	Company  string `json:"company,omitempty"`
	Password string `json:"password" binding:"required,min=8"`
}

// CustomerResetPasswordRequest is the request body for password reset.
type CustomerResetPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// CustomerSetPasswordRequest is the request body for setting a new password.
type CustomerSetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// CustomerChangePasswordRequest is the request body for changing password.
type CustomerChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// CustomerResponse is the public customer response (without sensitive fields).
type CustomerResponse struct {
	ID        uuid.UUID      `json:"id"`
	Email     string         `json:"email"`
	Name      string         `json:"name"`
	Company   string         `json:"company,omitempty"`
	Status    CustomerStatus `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
}

// ToResponse converts a Customer to CustomerResponse.
func (c *Customer) ToResponse() CustomerResponse {
	return CustomerResponse{
		ID:        c.ID,
		Email:     c.Email,
		Name:      c.Name,
		Company:   c.Company,
		Status:    c.Status,
		CreatedAt: c.CreatedAt,
	}
}
