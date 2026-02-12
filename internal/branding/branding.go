// Package branding provides white-label branding logic for Enterprise organizations.
package branding

import (
	"context"
	"fmt"
	"net/url"
	"regexp"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Store defines the persistence interface for branding operations.
type Store interface {
	GetBrandingSettings(ctx context.Context, orgID uuid.UUID) (*models.BrandingSettings, error)
	UpsertBrandingSettings(ctx context.Context, b *models.BrandingSettings) error
	DeleteBrandingSettings(ctx context.Context, orgID uuid.UUID) error
}

// hexColorRegex validates CSS hex color values (#RGB or #RRGGBB).
var hexColorRegex = regexp.MustCompile(`^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// GetDefaultBranding returns the default branding settings (empty values).
func GetDefaultBranding(orgID uuid.UUID) *models.BrandingSettings {
	return models.NewBrandingSettings(orgID)
}

// LoadBranding retrieves the branding settings for an organization,
// returning defaults if none are configured.
func LoadBranding(ctx context.Context, store Store, orgID uuid.UUID) (*models.BrandingSettings, error) {
	b, err := store.GetBrandingSettings(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("load branding: %w", err)
	}
	if b == nil {
		return GetDefaultBranding(orgID), nil
	}
	return b, nil
}

// ValidateColor validates that a color string is a valid hex color.
// Empty strings are allowed (means use default).
func ValidateColor(color string) error {
	if color == "" {
		return nil
	}
	if !hexColorRegex.MatchString(color) {
		return fmt.Errorf("invalid hex color %q: must be #RGB or #RRGGBB format", color)
	}
	return nil
}

// ValidateColors validates both primary and secondary colors.
func ValidateColors(primary, secondary string) error {
	if err := ValidateColor(primary); err != nil {
		return fmt.Errorf("primary color: %w", err)
	}
	if err := ValidateColor(secondary); err != nil {
		return fmt.Errorf("secondary color: %w", err)
	}
	return nil
}

// ValidateURL validates that a URL string is valid.
// Empty strings are allowed (means use default).
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}
	u, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("invalid URL scheme %q: must be http or https", u.Scheme)
	}
	return nil
}
