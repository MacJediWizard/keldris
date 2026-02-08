package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProxmoxConnection represents a Proxmox VE API connection configuration.
type ProxmoxConnection struct {
	ID                   uuid.UUID  `json:"id"`
	OrgID                uuid.UUID  `json:"org_id"`
	Name                 string     `json:"name"`
	Host                 string     `json:"host"`
	Port                 int        `json:"port"`
	Node                 string     `json:"node"`
	Username             string     `json:"username"`
	TokenID              string     `json:"token_id,omitempty"`
	TokenSecretEncrypted []byte     `json:"-"` // Never expose in JSON
	VerifySSL            bool       `json:"verify_ssl"`
	Enabled              bool       `json:"enabled"`
	LastConnectedAt      *time.Time `json:"last_connected_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

// NewProxmoxConnection creates a new ProxmoxConnection with the given details.
func NewProxmoxConnection(orgID uuid.UUID, name, host string, port int, node, username string) *ProxmoxConnection {
	now := time.Now()
	return &ProxmoxConnection{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Host:      host,
		Port:      port,
		Node:      node,
		Username:  username,
		VerifySSL: true,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetTokenAuth sets API token authentication credentials.
func (c *ProxmoxConnection) SetTokenAuth(tokenID string, tokenSecretEncrypted []byte) {
	c.TokenID = tokenID
	c.TokenSecretEncrypted = tokenSecretEncrypted
}

// HasTokenAuth returns true if token authentication is configured.
func (c *ProxmoxConnection) HasTokenAuth() bool {
	return c.TokenID != "" && len(c.TokenSecretEncrypted) > 0
}

// GetAPIURL returns the base URL for the Proxmox API.
func (c *ProxmoxConnection) GetAPIURL() string {
	return fmt.Sprintf("https://%s:%d/api2/json", c.Host, c.Port)
}

// MarkConnected updates the last connected timestamp.
func (c *ProxmoxConnection) MarkConnected() {
	now := time.Now()
	c.LastConnectedAt = &now
	c.UpdatedAt = now
}
