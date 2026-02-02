package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
)

// LicenseStatus defines the status of a license.
type LicenseStatus string

const (
	// LicenseStatusActive is an active, valid license.
	LicenseStatusActive LicenseStatus = "active"
	// LicenseStatusExpired is an expired license.
	LicenseStatusExpired LicenseStatus = "expired"
	// LicenseStatusRevoked is a revoked license.
	LicenseStatusRevoked LicenseStatus = "revoked"
	// LicenseStatusSuspended is a temporarily suspended license.
	LicenseStatusSuspended LicenseStatus = "suspended"
)

// LicenseType defines the type of license.
type LicenseType string

const (
	// LicenseTypeTrial is a trial license with limited duration.
	LicenseTypeTrial LicenseType = "trial"
	// LicenseTypeStandard is a standard paid license.
	LicenseTypeStandard LicenseType = "standard"
	// LicenseTypeProfessional is a professional license with more features.
	LicenseTypeProfessional LicenseType = "professional"
	// LicenseTypeEnterprise is an enterprise license with all features.
	LicenseTypeEnterprise LicenseType = "enterprise"
)

// License represents a product license purchased by a customer.
type License struct {
	ID           uuid.UUID     `json:"id"`
	CustomerID   uuid.UUID     `json:"customer_id"`
	LicenseKey   string        `json:"license_key"`
	LicenseType  LicenseType   `json:"license_type"`
	ProductName  string        `json:"product_name"`
	Status       LicenseStatus `json:"status"`
	MaxAgents    *int          `json:"max_agents,omitempty"`    // nil means unlimited
	MaxRepos     *int          `json:"max_repos,omitempty"`     // nil means unlimited
	MaxStorage   *int64        `json:"max_storage_gb,omitempty"` // nil means unlimited (in GB)
	Features     []string      `json:"features,omitempty"`      // List of enabled features
	IssuedAt     time.Time     `json:"issued_at"`
	ExpiresAt    *time.Time    `json:"expires_at,omitempty"` // nil means perpetual
	ActivatedAt  *time.Time    `json:"activated_at,omitempty"`
	LastVerified *time.Time    `json:"last_verified,omitempty"`
	Notes        string        `json:"notes,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

// NewLicense creates a new License with the given details.
func NewLicense(customerID uuid.UUID, licenseType LicenseType, productName string) *License {
	now := time.Now()
	licenseKey, _ := GenerateLicenseKey()
	return &License{
		ID:          uuid.New(),
		CustomerID:  customerID,
		LicenseKey:  licenseKey,
		LicenseType: licenseType,
		ProductName: productName,
		Status:      LicenseStatusActive,
		IssuedAt:    now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// GenerateLicenseKey generates a unique license key in format XXXX-XXXX-XXXX-XXXX.
func GenerateLicenseKey() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	hex := hex.EncodeToString(bytes)
	// Format as XXXX-XXXX-XXXX-XXXX (uppercase)
	key := hex[0:4] + "-" + hex[4:8] + "-" + hex[8:12] + "-" + hex[12:16]
	return key, nil
}

// IsValid returns true if the license is currently valid.
func (l *License) IsValid() bool {
	if l.Status != LicenseStatusActive {
		return false
	}
	if l.ExpiresAt != nil && time.Now().After(*l.ExpiresAt) {
		return false
	}
	return true
}

// IsExpired returns true if the license has expired.
func (l *License) IsExpired() bool {
	if l.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*l.ExpiresAt)
}

// DaysRemaining returns the number of days remaining until expiration.
// Returns -1 for perpetual licenses.
func (l *License) DaysRemaining() int {
	if l.ExpiresAt == nil {
		return -1
	}
	duration := time.Until(*l.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}

// LicenseWithCustomer includes customer details for display.
type LicenseWithCustomer struct {
	License
	CustomerEmail string `json:"customer_email"`
	CustomerName  string `json:"customer_name"`
}

// CreateLicenseRequest is the request body for creating a license (admin).
type CreateLicenseRequest struct {
	CustomerID  uuid.UUID   `json:"customer_id" binding:"required"`
	LicenseType LicenseType `json:"license_type" binding:"required,oneof=trial standard professional enterprise"`
	ProductName string      `json:"product_name" binding:"required,min=1,max=255"`
	MaxAgents   *int        `json:"max_agents,omitempty"`
	MaxRepos    *int        `json:"max_repos,omitempty"`
	MaxStorage  *int64      `json:"max_storage_gb,omitempty"`
	Features    []string    `json:"features,omitempty"`
	ExpiresAt   *time.Time  `json:"expires_at,omitempty"`
	Notes       string      `json:"notes,omitempty"`
}

// UpdateLicenseRequest is the request body for updating a license (admin).
type UpdateLicenseRequest struct {
	Status     *LicenseStatus `json:"status,omitempty"`
	MaxAgents  *int           `json:"max_agents,omitempty"`
	MaxRepos   *int           `json:"max_repos,omitempty"`
	MaxStorage *int64         `json:"max_storage_gb,omitempty"`
	Features   []string       `json:"features,omitempty"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	Notes      *string        `json:"notes,omitempty"`
}

// LicenseDownloadResponse contains the license key for download.
type LicenseDownloadResponse struct {
	LicenseKey  string      `json:"license_key"`
	LicenseType LicenseType `json:"license_type"`
	ProductName string      `json:"product_name"`
	CustomerID  uuid.UUID   `json:"customer_id"`
	IssuedAt    time.Time   `json:"issued_at"`
	ExpiresAt   *time.Time  `json:"expires_at,omitempty"`
	MaxAgents   *int        `json:"max_agents,omitempty"`
	MaxRepos    *int        `json:"max_repos,omitempty"`
	MaxStorage  *int64      `json:"max_storage_gb,omitempty"`
	Features    []string    `json:"features,omitempty"`
}

// ToDownloadResponse converts a License to LicenseDownloadResponse.
func (l *License) ToDownloadResponse() LicenseDownloadResponse {
	return LicenseDownloadResponse{
		LicenseKey:  l.LicenseKey,
		LicenseType: l.LicenseType,
		ProductName: l.ProductName,
		CustomerID:  l.CustomerID,
		IssuedAt:    l.IssuedAt,
		ExpiresAt:   l.ExpiresAt,
		MaxAgents:   l.MaxAgents,
		MaxRepos:    l.MaxRepos,
		MaxStorage:  l.MaxStorage,
		Features:    l.Features,
	}
}
