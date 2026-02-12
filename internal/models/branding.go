package models

import (
	"time"

	"github.com/google/uuid"
)

// BrandingSettings holds white-label branding configuration for an organization.
type BrandingSettings struct {
	ID             uuid.UUID `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	LogoURL        string    `json:"logo_url"`
	FaviconURL     string    `json:"favicon_url"`
	ProductName    string    `json:"product_name"`
	PrimaryColor   string    `json:"primary_color"`
	SecondaryColor string    `json:"secondary_color"`
	SupportURL     string    `json:"support_url"`
	CustomCSS      string    `json:"custom_css"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewBrandingSettings creates a new BrandingSettings with defaults.
func NewBrandingSettings(orgID uuid.UUID) *BrandingSettings {
	now := time.Now()
	return &BrandingSettings{
		ID:        uuid.New(),
		OrgID:     orgID,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
