package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SuperuserAction defines actions that can be performed by superusers.
type SuperuserAction string

const (
	// SuperuserActionViewOrgs is viewing all organizations.
	SuperuserActionViewOrgs SuperuserAction = "view_organizations"
	// SuperuserActionViewOrg is viewing a specific organization.
	SuperuserActionViewOrg SuperuserAction = "view_organization"
	// SuperuserActionImpersonate is impersonating a user.
	SuperuserActionImpersonate SuperuserAction = "impersonate_user"
	// SuperuserActionEndImpersonate is ending impersonation.
	SuperuserActionEndImpersonate SuperuserAction = "end_impersonation"
	// SuperuserActionUpdateSettings is updating system settings.
	SuperuserActionUpdateSettings SuperuserAction = "update_system_settings"
	// SuperuserActionViewSettings is viewing system settings.
	SuperuserActionViewSettings SuperuserAction = "view_system_settings"
	// SuperuserActionGrantSuperuser is granting superuser privileges.
	SuperuserActionGrantSuperuser SuperuserAction = "grant_superuser"
	// SuperuserActionRevokeSuperuser is revoking superuser privileges.
	SuperuserActionRevokeSuperuser SuperuserAction = "revoke_superuser"
	// SuperuserActionViewUsers is viewing all users across orgs.
	SuperuserActionViewUsers SuperuserAction = "view_users"
	// SuperuserActionViewAuditLogs is viewing superuser audit logs.
	SuperuserActionViewAuditLogs SuperuserAction = "view_superuser_audit_logs"
)

// SuperuserAuditLog records actions taken by superusers for compliance.
type SuperuserAuditLog struct {
	ID                 uuid.UUID       `json:"id"`
	SuperuserID        uuid.UUID       `json:"superuser_id"`
	Action             SuperuserAction `json:"action"`
	TargetType         string          `json:"target_type"`
	TargetID           *uuid.UUID      `json:"target_id,omitempty"`
	TargetOrgID        *uuid.UUID      `json:"target_org_id,omitempty"`
	ImpersonatedUserID *uuid.UUID      `json:"impersonated_user_id,omitempty"`
	IPAddress          string          `json:"ip_address,omitempty"`
	UserAgent          string          `json:"user_agent,omitempty"`
	Details            json.RawMessage `json:"details,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

// SuperuserAuditLogWithUser includes superuser details for display.
type SuperuserAuditLogWithUser struct {
	SuperuserAuditLog
	SuperuserEmail string `json:"superuser_email"`
	SuperuserName  string `json:"superuser_name"`
}

// NewSuperuserAuditLog creates a new superuser audit log entry.
func NewSuperuserAuditLog(superuserID uuid.UUID, action SuperuserAction, targetType string) *SuperuserAuditLog {
	return &SuperuserAuditLog{
		ID:          uuid.New(),
		SuperuserID: superuserID,
		Action:      action,
		TargetType:  targetType,
		CreatedAt:   time.Now(),
	}
}

// WithTargetID sets the target resource ID.
func (l *SuperuserAuditLog) WithTargetID(id uuid.UUID) *SuperuserAuditLog {
	l.TargetID = &id
	return l
}

// WithTargetOrgID sets the target organization ID.
func (l *SuperuserAuditLog) WithTargetOrgID(orgID uuid.UUID) *SuperuserAuditLog {
	l.TargetOrgID = &orgID
	return l
}

// WithImpersonatedUser sets the impersonated user ID.
func (l *SuperuserAuditLog) WithImpersonatedUser(userID uuid.UUID) *SuperuserAuditLog {
	l.ImpersonatedUserID = &userID
	return l
}

// WithRequestInfo sets request metadata.
func (l *SuperuserAuditLog) WithRequestInfo(ipAddress, userAgent string) *SuperuserAuditLog {
	l.IPAddress = ipAddress
	l.UserAgent = userAgent
	return l
}

// WithDetails sets additional details as JSON.
func (l *SuperuserAuditLog) WithDetails(details interface{}) *SuperuserAuditLog {
	if data, err := json.Marshal(details); err == nil {
		l.Details = data
	}
	return l
}

// SystemSetting represents a global system configuration.
type SystemSetting struct {
	Key         string          `json:"key"`
	Value       json.RawMessage `json:"value"`
	Description string          `json:"description,omitempty"`
	UpdatedBy   *uuid.UUID      `json:"updated_by,omitempty"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// GetBoolValue returns the setting value as a boolean.
func (s *SystemSetting) GetBoolValue() bool {
	var v bool
	_ = json.Unmarshal(s.Value, &v)
	return v
}

// GetStringValue returns the setting value as a string.
func (s *SystemSetting) GetStringValue() string {
	var v string
	_ = json.Unmarshal(s.Value, &v)
	return v
}
