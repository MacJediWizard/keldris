package models

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of action that was audited.
type AuditAction string

const (
	// User actions
	AuditActionLogin  AuditAction = "login"
	AuditActionLogout AuditAction = "logout"

	// CRUD actions
	AuditActionCreate AuditAction = "create"
	AuditActionRead   AuditAction = "read"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"

	// Agent actions
	AuditActionBackup  AuditAction = "backup"
	AuditActionRestore AuditAction = "restore"
)

// AuditResult represents the outcome of an audited action.
type AuditResult string

const (
	// AuditResultSuccess indicates the action completed successfully.
	AuditResultSuccess AuditResult = "success"
	// AuditResultFailure indicates the action failed.
	AuditResultFailure AuditResult = "failure"
	// AuditResultDenied indicates the action was denied due to authorization.
	AuditResultDenied AuditResult = "denied"
)

// AuditLog represents a single audit log entry for compliance tracking.
type AuditLog struct {
	ID           uuid.UUID   `json:"id"`
	OrgID        uuid.UUID   `json:"org_id"`
	UserID       *uuid.UUID  `json:"user_id,omitempty"`
	AgentID      *uuid.UUID  `json:"agent_id,omitempty"`
	Action       AuditAction `json:"action"`
	ResourceType string      `json:"resource_type"`
	ResourceID   *uuid.UUID  `json:"resource_id,omitempty"`
	Result       AuditResult `json:"result"`
	IPAddress    string      `json:"ip_address,omitempty"`
	UserAgent    string      `json:"user_agent,omitempty"`
	Details      string      `json:"details,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
}

// NewAuditLog creates a new AuditLog entry.
func NewAuditLog(orgID uuid.UUID, action AuditAction, resourceType string, result AuditResult) *AuditLog {
	return &AuditLog{
		ID:           uuid.New(),
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		Result:       result,
		CreatedAt:    time.Now(),
	}
}

// WithUser sets the user context for the audit log.
func (a *AuditLog) WithUser(userID uuid.UUID) *AuditLog {
	a.UserID = &userID
	return a
}

// WithAgent sets the agent context for the audit log.
func (a *AuditLog) WithAgent(agentID uuid.UUID) *AuditLog {
	a.AgentID = &agentID
	return a
}

// WithResource sets the resource being acted upon.
func (a *AuditLog) WithResource(resourceID uuid.UUID) *AuditLog {
	a.ResourceID = &resourceID
	return a
}

// WithRequestInfo sets HTTP request information.
func (a *AuditLog) WithRequestInfo(ipAddress, userAgent string) *AuditLog {
	a.IPAddress = ipAddress
	a.UserAgent = userAgent
	return a
}

// WithDetails sets additional details about the action.
func (a *AuditLog) WithDetails(details string) *AuditLog {
	a.Details = details
	return a
}

// IsSuccess returns true if the action was successful.
func (a *AuditLog) IsSuccess() bool {
	return a.Result == AuditResultSuccess
}

// IsUserAction returns true if the action was performed by a user.
func (a *AuditLog) IsUserAction() bool {
	return a.UserID != nil
}

// IsAgentAction returns true if the action was performed by an agent.
func (a *AuditLog) IsAgentAction() bool {
	return a.AgentID != nil
}
