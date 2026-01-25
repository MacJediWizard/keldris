package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

// IPAllowlistType represents the type of access the allowlist applies to.
type IPAllowlistType string

const (
	IPAllowlistTypeUI    IPAllowlistType = "ui"
	IPAllowlistTypeAgent IPAllowlistType = "agent"
	IPAllowlistTypeBoth  IPAllowlistType = "both"
)

// IPAllowlist represents an IP address or CIDR range in the allowlist.
type IPAllowlist struct {
	ID          uuid.UUID       `json:"id"`
	OrgID       uuid.UUID       `json:"org_id"`
	CIDR        string          `json:"cidr"`
	Description string          `json:"description,omitempty"`
	Type        IPAllowlistType `json:"type"`
	Enabled     bool            `json:"enabled"`
	CreatedBy   *uuid.UUID      `json:"created_by,omitempty"`
	UpdatedBy   *uuid.UUID      `json:"updated_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// NewIPAllowlist creates a new IPAllowlist with the given details.
func NewIPAllowlist(orgID uuid.UUID, cidr string, allowlistType IPAllowlistType) *IPAllowlist {
	now := time.Now()
	return &IPAllowlist{
		ID:        uuid.New(),
		OrgID:     orgID,
		CIDR:      cidr,
		Type:      allowlistType,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ContainsIP checks if the given IP address is within the CIDR range.
func (a *IPAllowlist) ContainsIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	_, network, err := net.ParseCIDR(a.CIDR)
	if err != nil {
		// If it's not a valid CIDR, try parsing as a single IP
		allowedIP := net.ParseIP(a.CIDR)
		if allowedIP == nil {
			return false
		}
		return ip.Equal(allowedIP)
	}

	return network.Contains(ip)
}

// AppliesToType checks if this allowlist entry applies to the given access type.
func (a *IPAllowlist) AppliesToType(accessType IPAllowlistType) bool {
	if a.Type == IPAllowlistTypeBoth {
		return true
	}
	return a.Type == accessType
}

// IPAllowlistSettings represents the organization-level settings for IP allowlists.
type IPAllowlistSettings struct {
	ID               uuid.UUID `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	Enabled          bool      `json:"enabled"`
	EnforceForUI     bool      `json:"enforce_for_ui"`
	EnforceForAgent  bool      `json:"enforce_for_agent"`
	AllowAdminBypass bool      `json:"allow_admin_bypass"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// NewIPAllowlistSettings creates a new IPAllowlistSettings with default values.
func NewIPAllowlistSettings(orgID uuid.UUID) *IPAllowlistSettings {
	now := time.Now()
	return &IPAllowlistSettings{
		ID:               uuid.New(),
		OrgID:            orgID,
		Enabled:          false,
		EnforceForUI:     true,
		EnforceForAgent:  true,
		AllowAdminBypass: true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// IPBlockedAttempt records a blocked access attempt.
type IPBlockedAttempt struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       uuid.UUID  `json:"org_id"`
	IPAddress   string     `json:"ip_address"`
	RequestType string     `json:"request_type"`
	Path        string     `json:"path,omitempty"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	AgentID     *uuid.UUID `json:"agent_id,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// NewIPBlockedAttempt creates a new IPBlockedAttempt record.
func NewIPBlockedAttempt(orgID uuid.UUID, ipAddress, requestType, path, reason string) *IPBlockedAttempt {
	return &IPBlockedAttempt{
		ID:          uuid.New(),
		OrgID:       orgID,
		IPAddress:   ipAddress,
		RequestType: requestType,
		Path:        path,
		Reason:      reason,
		CreatedAt:   time.Now(),
	}
}

// WithUser sets the user context for the blocked attempt.
func (b *IPBlockedAttempt) WithUser(userID uuid.UUID) *IPBlockedAttempt {
	b.UserID = &userID
	return b
}

// WithAgent sets the agent context for the blocked attempt.
func (b *IPBlockedAttempt) WithAgent(agentID uuid.UUID) *IPBlockedAttempt {
	b.AgentID = &agentID
	return b
}

// CreateIPAllowlistRequest is the request body for creating an IP allowlist entry.
type CreateIPAllowlistRequest struct {
	CIDR        string          `json:"cidr" binding:"required"`
	Description string          `json:"description,omitempty"`
	Type        IPAllowlistType `json:"type" binding:"required,oneof=ui agent both"`
	Enabled     *bool           `json:"enabled,omitempty"`
}

// UpdateIPAllowlistRequest is the request body for updating an IP allowlist entry.
type UpdateIPAllowlistRequest struct {
	CIDR        *string          `json:"cidr,omitempty"`
	Description *string          `json:"description,omitempty"`
	Type        *IPAllowlistType `json:"type,omitempty"`
	Enabled     *bool            `json:"enabled,omitempty"`
}

// UpdateIPAllowlistSettingsRequest is the request body for updating IP allowlist settings.
type UpdateIPAllowlistSettingsRequest struct {
	Enabled          *bool `json:"enabled,omitempty"`
	EnforceForUI     *bool `json:"enforce_for_ui,omitempty"`
	EnforceForAgent  *bool `json:"enforce_for_agent,omitempty"`
	AllowAdminBypass *bool `json:"allow_admin_bypass,omitempty"`
}

// IPAllowlistsResponse is the response for listing IP allowlists.
type IPAllowlistsResponse struct {
	Allowlists []IPAllowlist `json:"allowlists"`
}

// IPBlockedAttemptsResponse is the response for listing blocked attempts.
type IPBlockedAttemptsResponse struct {
	Attempts []IPBlockedAttempt `json:"attempts"`
	Total    int                `json:"total"`
}

// IPAllowlistWithSettingsResponse combines allowlists with settings.
type IPAllowlistWithSettingsResponse struct {
	Settings   IPAllowlistSettings `json:"settings"`
	Allowlists []IPAllowlist       `json:"allowlists"`
}
