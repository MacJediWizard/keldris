package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole defines the role of a user within an organization.
type UserRole string

const (
	// UserRoleAdmin has full access to manage the organization.
	UserRoleAdmin UserRole = "admin"
	// UserRoleUser has standard access to view and manage backups.
	UserRoleUser UserRole = "user"
	// UserRoleViewer has read-only access.
	UserRoleViewer UserRole = "viewer"
)

// User represents a user authenticated via OIDC.
type User struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	OIDCSubject string    `json:"oidc_subject"`
	Email       string    `json:"email"`
	Name        string    `json:"name,omitempty"`
	Role        UserRole  `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewUser creates a new User with the given details.
func NewUser(orgID uuid.UUID, oidcSubject, email, name string, role UserRole) *User {
	now := time.Now()
	return &User{
		ID:          uuid.New(),
		OrgID:       orgID,
		OIDCSubject: oidcSubject,
		Email:       email,
		Name:        name,
		Role:        role,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsAdmin returns true if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}
