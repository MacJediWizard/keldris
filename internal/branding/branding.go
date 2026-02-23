// Package branding provides white-label branding logic for Enterprise organizations.
package branding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"time"

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

// SettingKey is the system settings key for branding configuration.
const SettingKey = "branding"

// BrandingSettings holds white-label branding configuration for an organization.
type BrandingSettings struct {
	Enabled        bool   `json:"enabled"`
	ProductName    string `json:"product_name"`     // Custom product name
	CompanyName    string `json:"company_name"`     // Company name for footer
	LogoURL        string `json:"logo_url"`         // URL to custom logo
	LogoDarkURL    string `json:"logo_dark_url"`    // URL to logo for dark mode
	FaviconURL     string `json:"favicon_url"`      // URL to custom favicon
	PrimaryColor   string `json:"primary_color"`    // Primary brand color (hex)
	SecondaryColor string `json:"secondary_color"`  // Secondary brand color (hex)
	AccentColor    string `json:"accent_color"`     // Accent color (hex)
	SupportURL     string `json:"support_url"`      // Custom support URL
	SupportEmail   string `json:"support_email"`    // Custom support email
	PrivacyURL     string `json:"privacy_url"`      // Privacy policy URL
	TermsURL       string `json:"terms_url"`        // Terms of service URL
	FooterText     string `json:"footer_text"`      // Custom footer text
	LoginTitle     string `json:"login_title"`      // Custom login page title
	LoginSubtitle  string `json:"login_subtitle"`   // Custom login page subtitle
	LoginBgURL     string `json:"login_bg_url"`     // Login page background image URL
	HidePoweredBy  bool   `json:"hide_powered_by"`  // Hide "Powered by" text
	CustomCSS      string `json:"custom_css"`       // Custom CSS overrides
}

// DefaultBrandingSettings returns BrandingSettings with sensible defaults.
func DefaultBrandingSettings() BrandingSettings {
	return BrandingSettings{
		Enabled:        false,
		ProductName:    "Keldris",
		CompanyName:    "",
		LogoURL:        "",
		LogoDarkURL:    "",
		FaviconURL:     "",
		PrimaryColor:   "#4f46e5", // Indigo-600
		SecondaryColor: "#64748b", // Slate-500
		AccentColor:    "#06b6d4", // Cyan-500
		SupportURL:     "",
		SupportEmail:   "",
		PrivacyURL:     "",
		TermsURL:       "",
		FooterText:     "",
		LoginTitle:     "",
		LoginSubtitle:  "",
		LoginBgURL:     "",
		HidePoweredBy:  false,
		CustomCSS:      "",
	}
}

// Validate validates the branding settings.
func (s *BrandingSettings) Validate() error {
	if !s.Enabled {
		return nil // Skip validation if disabled
	}

	// Validate product name length
	if len(s.ProductName) > 100 {
		return errors.New("product name must be 100 characters or less")
	}

	// Validate company name length
	if len(s.CompanyName) > 200 {
		return errors.New("company name must be 200 characters or less")
	}

	// Validate URLs
	if s.LogoURL != "" {
		if _, err := url.Parse(s.LogoURL); err != nil {
			return fmt.Errorf("invalid logo URL: %w", err)
		}
	}

	if s.LogoDarkURL != "" {
		if _, err := url.Parse(s.LogoDarkURL); err != nil {
			return fmt.Errorf("invalid dark mode logo URL: %w", err)
		}
	}

	if s.FaviconURL != "" {
		if _, err := url.Parse(s.FaviconURL); err != nil {
			return fmt.Errorf("invalid favicon URL: %w", err)
		}
	}

	if s.SupportURL != "" {
		if _, err := url.Parse(s.SupportURL); err != nil {
			return fmt.Errorf("invalid support URL: %w", err)
		}
	}

	if s.PrivacyURL != "" {
		if _, err := url.Parse(s.PrivacyURL); err != nil {
			return fmt.Errorf("invalid privacy URL: %w", err)
		}
	}

	if s.TermsURL != "" {
		if _, err := url.Parse(s.TermsURL); err != nil {
			return fmt.Errorf("invalid terms URL: %w", err)
		}
	}

	if s.LoginBgURL != "" {
		if _, err := url.Parse(s.LoginBgURL); err != nil {
			return fmt.Errorf("invalid login background URL: %w", err)
		}
	}

	// Validate hex colors
	if s.PrimaryColor != "" && !hexColorRegex.MatchString(s.PrimaryColor) {
		return errors.New("primary color must be a valid hex color (e.g., #4f46e5)")
	}

	if s.SecondaryColor != "" && !hexColorRegex.MatchString(s.SecondaryColor) {
		return errors.New("secondary color must be a valid hex color (e.g., #64748b)")
	}

	if s.AccentColor != "" && !hexColorRegex.MatchString(s.AccentColor) {
		return errors.New("accent color must be a valid hex color (e.g., #06b6d4)")
	}

	// Validate footer text length
	if len(s.FooterText) > 500 {
		return errors.New("footer text must be 500 characters or less")
	}

	// Validate login text lengths
	if len(s.LoginTitle) > 100 {
		return errors.New("login title must be 100 characters or less")
	}

	if len(s.LoginSubtitle) > 200 {
		return errors.New("login subtitle must be 200 characters or less")
	}

	// Validate custom CSS length (limit to prevent abuse)
	if len(s.CustomCSS) > 10000 {
		return errors.New("custom CSS must be 10000 characters or less")
	}

	return nil
}

// UpdateBrandingSettingsRequest is the request for updating branding settings.
type UpdateBrandingSettingsRequest struct {
	Enabled        *bool   `json:"enabled,omitempty"`
	ProductName    *string `json:"product_name,omitempty" binding:"omitempty,max=100"`
	CompanyName    *string `json:"company_name,omitempty" binding:"omitempty,max=200"`
	LogoURL        *string `json:"logo_url,omitempty" binding:"omitempty,url,max=500"`
	LogoDarkURL    *string `json:"logo_dark_url,omitempty" binding:"omitempty,url,max=500"`
	FaviconURL     *string `json:"favicon_url,omitempty" binding:"omitempty,url,max=500"`
	PrimaryColor   *string `json:"primary_color,omitempty" binding:"omitempty,max=10"`
	SecondaryColor *string `json:"secondary_color,omitempty" binding:"omitempty,max=10"`
	AccentColor    *string `json:"accent_color,omitempty" binding:"omitempty,max=10"`
	SupportURL     *string `json:"support_url,omitempty" binding:"omitempty,url,max=500"`
	SupportEmail   *string `json:"support_email,omitempty" binding:"omitempty,email,max=255"`
	PrivacyURL     *string `json:"privacy_url,omitempty" binding:"omitempty,url,max=500"`
	TermsURL       *string `json:"terms_url,omitempty" binding:"omitempty,url,max=500"`
	FooterText     *string `json:"footer_text,omitempty" binding:"omitempty,max=500"`
	LoginTitle     *string `json:"login_title,omitempty" binding:"omitempty,max=100"`
	LoginSubtitle  *string `json:"login_subtitle,omitempty" binding:"omitempty,max=200"`
	LoginBgURL     *string `json:"login_bg_url,omitempty" binding:"omitempty,url,max=500"`
	HidePoweredBy  *bool   `json:"hide_powered_by,omitempty"`
	CustomCSS      *string `json:"custom_css,omitempty" binding:"omitempty,max=10000"`
}

// BrandingSettingsResponse is the response containing branding settings.
type BrandingSettingsResponse struct {
	Branding  BrandingSettings `json:"branding"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// BrandingAuditLog records changes to branding settings.
type BrandingAuditLog struct {
	ID             uuid.UUID       `json:"id"`
	OrgID          uuid.UUID       `json:"org_id"`
	OldValue       json.RawMessage `json:"old_value,omitempty"`
	NewValue       json.RawMessage `json:"new_value"`
	ChangedBy      uuid.UUID       `json:"changed_by"`
	ChangedByEmail string          `json:"changed_by_email,omitempty"`
	ChangedAt      time.Time       `json:"changed_at"`
	IPAddress      string          `json:"ip_address,omitempty"`
}

// NewBrandingAuditLog creates a new audit log entry for branding changes.
func NewBrandingAuditLog(orgID uuid.UUID, oldValue, newValue json.RawMessage, changedBy uuid.UUID, ipAddress string) *BrandingAuditLog {
	return &BrandingAuditLog{
		ID:        uuid.New(),
		OrgID:     orgID,
		OldValue:  oldValue,
		NewValue:  newValue,
		ChangedBy: changedBy,
		ChangedAt: time.Now(),
		IPAddress: ipAddress,
	}
}

// PublicBrandingSettings contains only the branding fields that are safe to expose publicly
// (e.g., on the login page before authentication).
type PublicBrandingSettings struct {
	Enabled        bool   `json:"enabled"`
	ProductName    string `json:"product_name"`
	LogoURL        string `json:"logo_url"`
	LogoDarkURL    string `json:"logo_dark_url"`
	FaviconURL     string `json:"favicon_url"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	AccentColor    string `json:"accent_color"`
	SupportURL     string `json:"support_url"`
	PrivacyURL     string `json:"privacy_url"`
	TermsURL       string `json:"terms_url"`
	LoginTitle     string `json:"login_title"`
	LoginSubtitle  string `json:"login_subtitle"`
	LoginBgURL     string `json:"login_bg_url"`
	HidePoweredBy  bool   `json:"hide_powered_by"`
}

// ToPublic converts full branding settings to public-safe settings.
func (s *BrandingSettings) ToPublic() PublicBrandingSettings {
	return PublicBrandingSettings{
		Enabled:        s.Enabled,
		ProductName:    s.ProductName,
		LogoURL:        s.LogoURL,
		LogoDarkURL:    s.LogoDarkURL,
		FaviconURL:     s.FaviconURL,
		PrimaryColor:   s.PrimaryColor,
		SecondaryColor: s.SecondaryColor,
		AccentColor:    s.AccentColor,
		SupportURL:     s.SupportURL,
		PrivacyURL:     s.PrivacyURL,
		TermsURL:       s.TermsURL,
		LoginTitle:     s.LoginTitle,
		LoginSubtitle:  s.LoginSubtitle,
		LoginBgURL:     s.LoginBgURL,
		HidePoweredBy:  s.HidePoweredBy,
	}
}
