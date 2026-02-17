// Package models defines the domain models for Keldris.
package models

import (
	"time"

	"github.com/google/uuid"
)

// Organization represents a multi-tenant organization.
// Note: SSO settings (sso_default_role, sso_auto_create_orgs) and org settings
// are managed via dedicated methods (GetOrganizationSSOSettings, etc.) rather
// than being loaded with the core Organization struct.
type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewOrganization creates a new Organization with the given name and slug.
func NewOrganization(name, slug string) *Organization {
	now := time.Now()
	return &Organization{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
