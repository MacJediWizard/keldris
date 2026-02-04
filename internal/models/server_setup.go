package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ServerSetupStep represents a step in the server setup wizard.
type ServerSetupStep string

const (
	// SetupStepDatabase is the database connection verification step.
	SetupStepDatabase ServerSetupStep = "database"
	// SetupStepSuperuser is the superuser account creation step.
	SetupStepSuperuser ServerSetupStep = "superuser"
	// SetupStepSMTP is the SMTP configuration step.
	SetupStepSMTP ServerSetupStep = "smtp"
	// SetupStepOIDC is the OIDC configuration step.
	SetupStepOIDC ServerSetupStep = "oidc"
	// SetupStepLicense is the license key activation step.
	SetupStepLicense ServerSetupStep = "license"
	// SetupStepOrganization is the first organization creation step.
	SetupStepOrganization ServerSetupStep = "organization"
	// SetupStepComplete indicates setup is finished.
	SetupStepComplete ServerSetupStep = "complete"
)

// ServerSetupSteps is the ordered list of setup steps.
var ServerSetupSteps = []ServerSetupStep{
	SetupStepDatabase,
	SetupStepSuperuser,
	SetupStepSMTP,
	SetupStepOIDC,
	SetupStepLicense,
	SetupStepOrganization,
	SetupStepComplete,
}

// ServerSetup represents the server-wide setup state.
type ServerSetup struct {
	ID                int               `json:"id"`
	SetupCompleted    bool              `json:"setup_completed"`
	SetupCompletedAt  *time.Time        `json:"setup_completed_at,omitempty"`
	SetupCompletedBy  *uuid.UUID        `json:"setup_completed_by,omitempty"`
	CurrentStep       ServerSetupStep   `json:"current_step"`
	CompletedSteps    []ServerSetupStep `json:"completed_steps"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// HasCompletedStep checks if a specific step has been completed.
func (s *ServerSetup) HasCompletedStep(step ServerSetupStep) bool {
	for _, completed := range s.CompletedSteps {
		if completed == step {
			return true
		}
	}
	return false
}

// NextStep returns the next step after completing the current step.
func (s *ServerSetup) NextStep() ServerSetupStep {
	for i, step := range ServerSetupSteps {
		if step == s.CurrentStep && i+1 < len(ServerSetupSteps) {
			return ServerSetupSteps[i+1]
		}
	}
	return SetupStepComplete
}

// LicenseType represents the type of license.
type LicenseType string

const (
	// LicenseTypeTrial is a 14-day trial license.
	LicenseTypeTrial LicenseType = "trial"
	// LicenseTypeStandard is a standard license.
	LicenseTypeStandard LicenseType = "standard"
	// LicenseTypeProfessional is a professional license.
	LicenseTypeProfessional LicenseType = "professional"
	// LicenseTypeEnterprise is an enterprise license.
	LicenseTypeEnterprise LicenseType = "enterprise"
)

// LicenseStatus represents the status of a license.
type LicenseStatus string

const (
	// LicenseStatusActive is an active license.
	LicenseStatusActive LicenseStatus = "active"
	// LicenseStatusExpired is an expired license.
	LicenseStatusExpired LicenseStatus = "expired"
	// LicenseStatusRevoked is a revoked license.
	LicenseStatusRevoked LicenseStatus = "revoked"
)

// LicenseKey represents a license key record.
type LicenseKey struct {
	ID              uuid.UUID       `json:"id"`
	LicenseKey      string          `json:"license_key"`
	LicenseType     LicenseType     `json:"license_type"`
	Status          LicenseStatus   `json:"status"`
	MaxAgents       *int            `json:"max_agents,omitempty"`
	MaxRepositories *int            `json:"max_repositories,omitempty"`
	MaxStorageGB    *int            `json:"max_storage_gb,omitempty"`
	Features        json.RawMessage `json:"features,omitempty"`
	IssuedAt        time.Time       `json:"issued_at"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"`
	ActivatedAt     *time.Time      `json:"activated_at,omitempty"`
	ActivatedBy     *uuid.UUID      `json:"activated_by,omitempty"`
	CompanyName     string          `json:"company_name,omitempty"`
	ContactEmail    string          `json:"contact_email,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// IsExpired returns true if the license has expired.
func (l *LicenseKey) IsExpired() bool {
	if l.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*l.ExpiresAt)
}

// IsActive returns true if the license is active and not expired.
func (l *LicenseKey) IsActive() bool {
	return l.Status == LicenseStatusActive && !l.IsExpired()
}

// ServerSetupAuditLog records actions during server setup.
type ServerSetupAuditLog struct {
	ID          uuid.UUID       `json:"id"`
	Action      string          `json:"action"`
	Step        string          `json:"step,omitempty"`
	PerformedBy *uuid.UUID      `json:"performed_by,omitempty"`
	IPAddress   string          `json:"ip_address,omitempty"`
	UserAgent   string          `json:"user_agent,omitempty"`
	Details     json.RawMessage `json:"details,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// NewServerSetupAuditLog creates a new server setup audit log entry.
func NewServerSetupAuditLog(action, step string) *ServerSetupAuditLog {
	return &ServerSetupAuditLog{
		ID:        uuid.New(),
		Action:    action,
		Step:      step,
		CreatedAt: time.Now(),
	}
}

// WithPerformedBy sets the user who performed the action.
func (l *ServerSetupAuditLog) WithPerformedBy(userID uuid.UUID) *ServerSetupAuditLog {
	l.PerformedBy = &userID
	return l
}

// WithRequestInfo sets request metadata.
func (l *ServerSetupAuditLog) WithRequestInfo(ipAddress, userAgent string) *ServerSetupAuditLog {
	l.IPAddress = ipAddress
	l.UserAgent = userAgent
	return l
}

// WithDetails sets additional details as JSON.
func (l *ServerSetupAuditLog) WithDetails(details interface{}) *ServerSetupAuditLog {
	if data, err := json.Marshal(details); err == nil {
		l.Details = data
	}
	return l
}

// SetupStatusResponse is the response for setup status endpoint.
type SetupStatusResponse struct {
	NeedsSetup     bool              `json:"needs_setup"`
	SetupCompleted bool              `json:"setup_completed"`
	CurrentStep    ServerSetupStep   `json:"current_step"`
	CompletedSteps []ServerSetupStep `json:"completed_steps"`
	DatabaseOK     bool              `json:"database_ok"`
	HasSuperuser   bool              `json:"has_superuser"`
}

// CreateSuperuserRequest is the request to create the first superuser.
type CreateSuperuserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required"`
}

// ActivateLicenseRequest is the request to activate a license key.
type ActivateLicenseRequest struct {
	LicenseKey string `json:"license_key" binding:"required"`
}

// StartTrialRequest is the request to start a trial license.
type StartTrialRequest struct {
	CompanyName  string `json:"company_name"`
	ContactEmail string `json:"contact_email" binding:"required,email"`
}

// CreateOrganizationRequest is the request to create the first organization.
type CreateOrganizationRequest struct {
	Name string `json:"name" binding:"required"`
}

// TestSMTPRequest is the request to test SMTP configuration.
type TestSMTPRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"`
}

// LicenseInfo is the public license information returned to clients.
type LicenseInfo struct {
	LicenseType     LicenseType `json:"license_type"`
	Status          string      `json:"status"`
	MaxAgents       *int        `json:"max_agents,omitempty"`
	MaxRepositories *int        `json:"max_repositories,omitempty"`
	MaxStorageGB    *int        `json:"max_storage_gb,omitempty"`
	ExpiresAt       *time.Time  `json:"expires_at,omitempty"`
	CompanyName     string      `json:"company_name,omitempty"`
}
