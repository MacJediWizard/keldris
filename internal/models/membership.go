package models

import (
	"time"

	"github.com/google/uuid"
)

// OrgRole defines the role of a user within an organization.
type OrgRole string

const (
	// OrgRoleOwner has full control over the organization.
	OrgRoleOwner OrgRole = "owner"
	// OrgRoleAdmin can manage members and all resources.
	OrgRoleAdmin OrgRole = "admin"
	// OrgRoleMember can create and manage resources.
	OrgRoleMember OrgRole = "member"
	// OrgRoleReadonly has view-only access.
	OrgRoleReadonly OrgRole = "readonly"
)

// ValidOrgRoles returns all valid organization roles.
func ValidOrgRoles() []OrgRole {
	return []OrgRole{OrgRoleOwner, OrgRoleAdmin, OrgRoleMember, OrgRoleReadonly}
}

// IsValidOrgRole checks if the given role is a valid organization role.
func IsValidOrgRole(role string) bool {
	for _, r := range ValidOrgRoles() {
		if string(r) == role {
			return true
		}
	}
	return false
}

// OrgMembership represents a user's membership in an organization.
type OrgMembership struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
	Role      OrgRole   `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrgMembershipWithUser includes user details for display.
type OrgMembershipWithUser struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	OrgID     uuid.UUID `json:"org_id"`
	Role      OrgRole   `json:"role"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewOrgMembership creates a new OrgMembership.
func NewOrgMembership(userID, orgID uuid.UUID, role OrgRole) *OrgMembership {
	now := time.Now()
	return &OrgMembership{
		ID:        uuid.New(),
		UserID:    userID,
		OrgID:     orgID,
		Role:      role,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsOwner returns true if the membership role is owner.
func (m *OrgMembership) IsOwner() bool {
	return m.Role == OrgRoleOwner
}

// IsAdmin returns true if the membership role is admin or owner.
func (m *OrgMembership) IsAdmin() bool {
	return m.Role == OrgRoleAdmin || m.Role == OrgRoleOwner
}

// CanWrite returns true if the membership can create/modify resources.
func (m *OrgMembership) CanWrite() bool {
	return m.Role == OrgRoleOwner || m.Role == OrgRoleAdmin || m.Role == OrgRoleMember
}

// OrgInvitation represents an invitation to join an organization.
type OrgInvitation struct {
	ID         uuid.UUID  `json:"id"`
	OrgID      uuid.UUID  `json:"org_id"`
	Email      string     `json:"email"`
	Role       OrgRole    `json:"role"`
	Token      string     `json:"-"` // Never expose token in JSON
	InvitedBy  uuid.UUID  `json:"invited_by"`
	ExpiresAt  time.Time  `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// OrgInvitationWithDetails includes organization and inviter details.
type OrgInvitationWithDetails struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       uuid.UUID  `json:"org_id"`
	OrgName     string     `json:"org_name"`
	Email       string     `json:"email"`
	Role        OrgRole    `json:"role"`
	InvitedBy   uuid.UUID  `json:"invited_by"`
	InviterName string     `json:"inviter_name"`
	ExpiresAt   time.Time  `json:"expires_at"`
	AcceptedAt  *time.Time `json:"accepted_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NewOrgInvitation creates a new OrgInvitation.
func NewOrgInvitation(orgID uuid.UUID, email string, role OrgRole, token string, invitedBy uuid.UUID, expiresAt time.Time) *OrgInvitation {
	return &OrgInvitation{
		ID:        uuid.New(),
		OrgID:     orgID,
		Email:     email,
		Role:      role,
		Token:     token,
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}

// IsExpired returns true if the invitation has expired.
func (i *OrgInvitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsAccepted returns true if the invitation has been accepted.
func (i *OrgInvitation) IsAccepted() bool {
	return i.AcceptedAt != nil
}
