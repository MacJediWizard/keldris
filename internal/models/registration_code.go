package models

import (
	"time"

	"github.com/google/uuid"
)

// RegistrationCode represents a one-time code for agent registration.
type RegistrationCode struct {
	ID            uuid.UUID  `json:"id"`
	OrgID         uuid.UUID  `json:"org_id"`
	CreatedBy     uuid.UUID  `json:"created_by"`
	Code          string     `json:"code"`
	Hostname      *string    `json:"hostname,omitempty"`
	ExpiresAt     time.Time  `json:"expires_at"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	UsedByAgentID *uuid.UUID `json:"used_by_agent_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// NewRegistrationCode creates a new registration code with the given details.
func NewRegistrationCode(orgID, createdBy uuid.UUID, code string, hostname *string, expiresAt time.Time) *RegistrationCode {
	return &RegistrationCode{
		ID:        uuid.New(),
		OrgID:     orgID,
		CreatedBy: createdBy,
		Code:      code,
		Hostname:  hostname,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}
}

// IsExpired returns true if the code has expired.
func (r *RegistrationCode) IsExpired() bool {
	return time.Now().After(r.ExpiresAt)
}

// IsUsed returns true if the code has been used.
func (r *RegistrationCode) IsUsed() bool {
	return r.UsedAt != nil
}

// IsValid returns true if the code is neither expired nor used.
func (r *RegistrationCode) IsValid() bool {
	return !r.IsExpired() && !r.IsUsed()
}

// MarkUsed marks the code as used by the given agent.
func (r *RegistrationCode) MarkUsed(agentID uuid.UUID) {
	now := time.Now()
	r.UsedAt = &now
	r.UsedByAgentID = &agentID
}

// CreateRegistrationCodeRequest is the request body for creating a registration code.
type CreateRegistrationCodeRequest struct {
	Hostname string `json:"hostname,omitempty" binding:"omitempty,max=255"`
}

// CreateRegistrationCodeResponse is the response for creating a registration code.
type CreateRegistrationCodeResponse struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Hostname  *string   `json:"hostname,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

// RegisterWithCodeRequest is the request for registering an agent with a code.
type RegisterWithCodeRequest struct {
	Code     string `json:"code" binding:"required,min=6,max=8"`
	Hostname string `json:"hostname" binding:"required,min=1,max=255"`
}

// RegisterWithCodeResponse is the response for registering an agent with a code.
type RegisterWithCodeResponse struct {
	ID       uuid.UUID `json:"id"`
	Hostname string    `json:"hostname"`
	APIKey   string    `json:"api_key"`
}

// PendingRegistration represents a registration code pending agent connection.
type PendingRegistration struct {
	ID        uuid.UUID `json:"id"`
	Hostname  *string   `json:"hostname,omitempty"`
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"` // User email who created the code
}
