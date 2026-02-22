package models

import (
	"encoding/json"
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

// UserStatus defines the status of a user account.
type UserStatus string

const (
	// UserStatusActive is a normal active user account.
	UserStatusActive UserStatus = "active"
	// UserStatusDisabled is an administratively disabled account.
	UserStatusDisabled UserStatus = "disabled"
	// UserStatusPending is an invited but not yet accepted account.
	UserStatusPending UserStatus = "pending"
	// UserStatusLocked is an account locked due to failed login attempts.
	UserStatusLocked UserStatus = "locked"
)

// User represents a user authenticated via OIDC.
type User struct {
	ID                  uuid.UUID  `json:"id"`
	OrgID               uuid.UUID  `json:"org_id"`
	OIDCSubject         string     `json:"oidc_subject"`
	Email               string     `json:"email"`
	Name                string     `json:"name,omitempty"`
	Role                UserRole   `json:"role"`
	Status              UserStatus `json:"status"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP         string     `json:"last_login_ip,omitempty"`
	FailedLoginAttempts int        `json:"failed_login_attempts,omitempty"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	InvitedBy           *uuid.UUID `json:"invited_by,omitempty"`
	InvitedAt           *time.Time `json:"invited_at,omitempty"`
	IsSuperuser         bool       `json:"is_superuser"`
	EmailVerified       bool       `json:"email_verified"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
// User represents a user authenticated via OIDC.
type User struct {
	ID                  uuid.UUID  `json:"id"`
	OrgID               uuid.UUID  `json:"org_id"`
	OIDCSubject         string     `json:"oidc_subject"`
	Email               string     `json:"email"`
	Name                string     `json:"name,omitempty"`
	Role                UserRole   `json:"role"`
	Status              UserStatus `json:"status"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	LastLoginIP         string     `json:"last_login_ip,omitempty"`
	FailedLoginAttempts int        `json:"failed_login_attempts,omitempty"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	InvitedBy           *uuid.UUID `json:"invited_by,omitempty"`
	InvitedAt           *time.Time `json:"invited_at,omitempty"`
	IsSuperuser         bool       `json:"is_superuser"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
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
		Status:      UserStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsAdmin returns true if the user has admin role.
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsActive returns true if the user account is active.
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive
}

// IsLocked returns true if the user account is locked.
func (u *User) IsLocked() bool {
	if u.Status == UserStatusLocked {
		return true
	}
	if u.LockedUntil != nil && time.Now().Before(*u.LockedUntil) {
		return true
	}
	return false
}

// IsEmailVerified returns true if the user's email is verified.
// OIDC users are automatically considered verified.
func (u *User) IsEmailVerified() bool {
	// OIDC users are automatically verified
	if u.OIDCSubject != "" {
		return true
	}
	return u.EmailVerified
}

// RequiresEmailVerification returns true if the user needs to verify their email.
func (u *User) RequiresEmailVerification() bool {
	return u.OIDCSubject == "" && !u.EmailVerified
}

// UserWithMembership includes user details with their org membership.
type UserWithMembership struct {
	User
	OrgRole OrgRole `json:"org_role"`
}

// UserActivityLog represents a log entry for user activity.
type UserActivityLog struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	OrgID        uuid.UUID       `json:"org_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type,omitempty"`
	ResourceID   *uuid.UUID      `json:"resource_id,omitempty"`
	IPAddress    string          `json:"ip_address,omitempty"`
	UserAgent    string          `json:"user_agent,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

// NewUserActivityLog creates a new user activity log entry.
func NewUserActivityLog(userID, orgID uuid.UUID, action string) *UserActivityLog {
	return &UserActivityLog{
		ID:        uuid.New(),
		UserID:    userID,
		OrgID:     orgID,
		Action:    action,
		CreatedAt: time.Now(),
	}
}

// WithResource sets the resource type and ID for the activity log.
func (l *UserActivityLog) WithResource(resourceType string, resourceID uuid.UUID) *UserActivityLog {
	l.ResourceType = resourceType
	l.ResourceID = &resourceID
	return l
}

// WithIPAddress sets the IP address for the activity log.
func (l *UserActivityLog) WithIPAddress(ip string) *UserActivityLog {
	l.IPAddress = ip
	return l
}

// WithUserAgent sets the user agent for the activity log.
func (l *UserActivityLog) WithUserAgent(ua string) *UserActivityLog {
	l.UserAgent = ua
	return l
}

// WithDetails sets the details for the activity log.
func (l *UserActivityLog) WithDetails(details json.RawMessage) *UserActivityLog {
	l.Details = details
	return l
}

// UserActivityLogWithUser includes user details for display.
type UserActivityLogWithUser struct {
	UserActivityLog
	UserEmail string `json:"user_email"`
	UserName  string `json:"user_name"`
}

// UserImpersonationLog represents a log entry for user impersonation.
type UserImpersonationLog struct {
	ID           uuid.UUID  `json:"id"`
	AdminUserID  uuid.UUID  `json:"admin_user_id"`
	TargetUserID uuid.UUID  `json:"target_user_id"`
	OrgID        uuid.UUID  `json:"org_id"`
	Reason       string     `json:"reason,omitempty"`
	StartedAt    time.Time  `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at,omitempty"`
	IPAddress    string     `json:"ip_address,omitempty"`
	UserAgent    string     `json:"user_agent,omitempty"`
}

// NewUserImpersonationLog creates a new impersonation log entry.
func NewUserImpersonationLog(adminUserID, targetUserID, orgID uuid.UUID, reason string) *UserImpersonationLog {
	return &UserImpersonationLog{
		ID:           uuid.New(),
		AdminUserID:  adminUserID,
		TargetUserID: targetUserID,
		OrgID:        orgID,
		Reason:       reason,
		StartedAt:    time.Now(),
	}
}

// UserImpersonationLogWithUsers includes user details for display.
type UserImpersonationLogWithUsers struct {
	UserImpersonationLog
	AdminEmail  string `json:"admin_email"`
	AdminName   string `json:"admin_name"`
	TargetEmail string `json:"target_email"`
	TargetName  string `json:"target_name"`
}

// InviteUserRequest is the request body for inviting a new user.
type InviteUserRequest struct {
	Email string  `json:"email" binding:"required,email"`
	Name  string  `json:"name,omitempty"`
	Role  OrgRole `json:"role" binding:"required,oneof=owner admin member readonly"`
}

// UpdateUserRequest is the request body for updating a user.
type UpdateUserRequest struct {
	Name   string      `json:"name,omitempty"`
	Role   *OrgRole    `json:"role,omitempty"`
	Status *UserStatus `json:"status,omitempty"`
}

// ResetPasswordRequest is the request body for resetting a user's password.
type ResetPasswordRequest struct {
	NewPassword        string `json:"new_password" binding:"required,min=8"`
	RequireChangeOnUse bool   `json:"require_change_on_use"`
}

// ImpersonateUserRequest is the request body for impersonating a user.
type ImpersonateUserRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

// IsSuperAdmin returns true if the user is a global superuser.
// Superusers have system-wide access across all organizations.
func (u *User) IsSuperAdmin() bool {
	return u.IsSuperuser
}
