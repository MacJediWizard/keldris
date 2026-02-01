package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// LicenseTier represents the license tier level.
type LicenseTier string

const (
	// LicenseTierCommunity is the free tier with basic limits.
	LicenseTierCommunity LicenseTier = "community"
	// LicenseTierTeam is the team tier with expanded limits.
	LicenseTierTeam LicenseTier = "team"
	// LicenseTierEnterprise is the enterprise tier with full features.
	LicenseTierEnterprise LicenseTier = "enterprise"
)

// ValidLicenseTiers returns all valid license tiers.
func ValidLicenseTiers() []LicenseTier {
	return []LicenseTier{
		LicenseTierCommunity,
		LicenseTierTeam,
		LicenseTierEnterprise,
	}
}

// IsValid checks if the tier is valid.
func (t LicenseTier) IsValid() bool {
	for _, valid := range ValidLicenseTiers() {
		if t == valid {
			return true
		}
	}
	return false
}

// LicenseStatus represents the current status of a license.
type LicenseStatus string

const (
	// LicenseStatusActive means the license is valid and active.
	LicenseStatusActive LicenseStatus = "active"
	// LicenseStatusExpired means the license has expired.
	LicenseStatusExpired LicenseStatus = "expired"
	// LicenseStatusGracePeriod means the license is expired but within grace period.
	LicenseStatusGracePeriod LicenseStatus = "grace_period"
	// LicenseStatusInvalid means the license signature is invalid.
	LicenseStatusInvalid LicenseStatus = "invalid"
)

// LicenseLimits defines the limits enforced by a license.
type LicenseLimits struct {
	MaxAgents        int `json:"max_agents"`
	MaxUsers         int `json:"max_users"`
	MaxOrganizations int `json:"max_organizations"`
	MaxRepositories  int `json:"max_repositories"`
}

// LicenseFeatures defines which features are enabled by a license.
type LicenseFeatures struct {
	SSO                  bool `json:"sso"`
	RBAC                 bool `json:"rbac"`
	AuditLogs            bool `json:"audit_logs"`
	GeoReplication       bool `json:"geo_replication"`
	RansomwareProtection bool `json:"ransomware_protection"`
	LegalHolds           bool `json:"legal_holds"`
	CustomRetention      bool `json:"custom_retention"`
	APIAccess            bool `json:"api_access"`
	PrioritySupport      bool `json:"priority_support"`
}

// DefaultCommunityLimits returns the default limits for community tier.
func DefaultCommunityLimits() LicenseLimits {
	return LicenseLimits{
		MaxAgents:        5,
		MaxUsers:         3,
		MaxOrganizations: 1,
		MaxRepositories:  5,
	}
}

// DefaultTeamLimits returns the default limits for team tier.
func DefaultTeamLimits() LicenseLimits {
	return LicenseLimits{
		MaxAgents:        25,
		MaxUsers:         15,
		MaxOrganizations: 3,
		MaxRepositories:  25,
	}
}

// DefaultEnterpriseLimits returns the default limits for enterprise tier.
func DefaultEnterpriseLimits() LicenseLimits {
	return LicenseLimits{
		MaxAgents:        -1, // unlimited
		MaxUsers:         -1, // unlimited
		MaxOrganizations: -1, // unlimited
		MaxRepositories:  -1, // unlimited
	}
}

// DefaultCommunityFeatures returns the default features for community tier.
func DefaultCommunityFeatures() LicenseFeatures {
	return LicenseFeatures{
		SSO:                  false,
		RBAC:                 false,
		AuditLogs:            false,
		GeoReplication:       false,
		RansomwareProtection: false,
		LegalHolds:           false,
		CustomRetention:      false,
		APIAccess:            true,
		PrioritySupport:      false,
	}
}

// DefaultTeamFeatures returns the default features for team tier.
func DefaultTeamFeatures() LicenseFeatures {
	return LicenseFeatures{
		SSO:                  true,
		RBAC:                 true,
		AuditLogs:            true,
		GeoReplication:       false,
		RansomwareProtection: true,
		LegalHolds:           false,
		CustomRetention:      true,
		APIAccess:            true,
		PrioritySupport:      false,
	}
}

// DefaultEnterpriseFeatures returns the default features for enterprise tier.
func DefaultEnterpriseFeatures() LicenseFeatures {
	return LicenseFeatures{
		SSO:                  true,
		RBAC:                 true,
		AuditLogs:            true,
		GeoReplication:       true,
		RansomwareProtection: true,
		LegalHolds:           true,
		CustomRetention:      true,
		APIAccess:            true,
		PrioritySupport:      true,
	}
}

// LicensePayload is the data encoded in a license key.
type LicensePayload struct {
	ID         uuid.UUID       `json:"id"`
	CustomerID string          `json:"customer_id"`
	Tier       LicenseTier     `json:"tier"`
	Limits     LicenseLimits   `json:"limits"`
	Features   LicenseFeatures `json:"features"`
	IssuedAt   time.Time       `json:"issued_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

// License represents a stored license record.
type License struct {
	ID             uuid.UUID       `json:"id"`
	LicenseKey     string          `json:"license_key"`
	CustomerID     string          `json:"customer_id"`
	CustomerName   string          `json:"customer_name,omitempty"`
	CustomerEmail  string          `json:"customer_email,omitempty"`
	Tier           LicenseTier     `json:"tier"`
	Limits         LicenseLimits   `json:"limits"`
	Features       LicenseFeatures `json:"features"`
	IssuedAt       time.Time       `json:"issued_at"`
	ExpiresAt      time.Time       `json:"expires_at"`
	ActivatedAt    *time.Time      `json:"activated_at,omitempty"`
	LastValidated  *time.Time      `json:"last_validated,omitempty"`
	IsActive       bool            `json:"is_active"`
	Notes          string          `json:"notes,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// GracePeriodDays is the number of days after expiry before license becomes invalid.
const GracePeriodDays = 30

// NewLicense creates a new License record.
func NewLicense(licenseKey, customerID, customerName string, tier LicenseTier, expiresAt time.Time) *License {
	now := time.Now()
	var limits LicenseLimits
	var features LicenseFeatures

	switch tier {
	case LicenseTierTeam:
		limits = DefaultTeamLimits()
		features = DefaultTeamFeatures()
	case LicenseTierEnterprise:
		limits = DefaultEnterpriseLimits()
		features = DefaultEnterpriseFeatures()
	default:
		limits = DefaultCommunityLimits()
		features = DefaultCommunityFeatures()
	}

	return &License{
		ID:           uuid.New(),
		LicenseKey:   licenseKey,
		CustomerID:   customerID,
		CustomerName: customerName,
		Tier:         tier,
		Limits:       limits,
		Features:     features,
		IssuedAt:     now,
		ExpiresAt:    expiresAt,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Status returns the current status of the license.
func (l *License) Status() LicenseStatus {
	now := time.Now()
	if now.Before(l.ExpiresAt) {
		return LicenseStatusActive
	}
	gracePeriodEnd := l.ExpiresAt.AddDate(0, 0, GracePeriodDays)
	if now.Before(gracePeriodEnd) {
		return LicenseStatusGracePeriod
	}
	return LicenseStatusExpired
}

// IsExpired returns true if the license is expired (past grace period).
func (l *License) IsExpired() bool {
	return l.Status() == LicenseStatusExpired
}

// IsInGracePeriod returns true if the license is in the grace period.
func (l *License) IsInGracePeriod() bool {
	return l.Status() == LicenseStatusGracePeriod
}

// DaysUntilExpiry returns the number of days until the license expires.
// Returns negative values if already expired.
func (l *License) DaysUntilExpiry() int {
	duration := time.Until(l.ExpiresAt)
	return int(duration.Hours() / 24)
}

// DaysRemainingInGrace returns the number of days remaining in grace period.
// Returns 0 if not in grace period.
func (l *License) DaysRemainingInGrace() int {
	if !l.IsInGracePeriod() {
		return 0
	}
	gracePeriodEnd := l.ExpiresAt.AddDate(0, 0, GracePeriodDays)
	duration := time.Until(gracePeriodEnd)
	return int(duration.Hours() / 24)
}

// SetLimits sets the limits from JSON bytes.
func (l *License) SetLimits(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &l.Limits)
}

// LimitsJSON returns the limits as JSON bytes for database storage.
func (l *License) LimitsJSON() ([]byte, error) {
	return json.Marshal(l.Limits)
}

// SetFeatures sets the features from JSON bytes.
func (l *License) SetFeatures(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &l.Features)
}

// FeaturesJSON returns the features as JSON bytes for database storage.
func (l *License) FeaturesJSON() ([]byte, error) {
	return json.Marshal(l.Features)
}

// LicenseValidationResult contains the result of license validation.
type LicenseValidationResult struct {
	Valid         bool          `json:"valid"`
	Status        LicenseStatus `json:"status"`
	Tier          LicenseTier   `json:"tier"`
	Limits        LicenseLimits `json:"limits"`
	Features      LicenseFeatures `json:"features"`
	ExpiresAt     time.Time     `json:"expires_at"`
	DaysRemaining int           `json:"days_remaining"`
	Message       string        `json:"message,omitempty"`
}

// CreateLicenseRequest is the request body for creating a license.
type CreateLicenseRequest struct {
	CustomerID    string      `json:"customer_id" binding:"required"`
	CustomerName  string      `json:"customer_name" binding:"required"`
	CustomerEmail string      `json:"customer_email,omitempty"`
	Tier          LicenseTier `json:"tier" binding:"required"`
	ExpiresAt     time.Time   `json:"expires_at" binding:"required"`
	Notes         string      `json:"notes,omitempty"`
}

// ActivateLicenseRequest is the request body for activating a license.
type ActivateLicenseRequest struct {
	LicenseKey string `json:"license_key" binding:"required"`
}

// LicenseResponse wraps a license for API responses.
type LicenseResponse struct {
	License *License              `json:"license"`
	Status  LicenseStatus         `json:"status"`
	Result  *LicenseValidationResult `json:"validation,omitempty"`
}
