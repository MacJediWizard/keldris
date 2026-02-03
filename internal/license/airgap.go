// Package license provides air-gapped license management for enterprise deployments.
package license

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// License validation errors.
var (
	ErrLicenseExpired     = errors.New("license has expired")
	ErrLicenseInvalid     = errors.New("license signature is invalid")
	ErrLicenseNotFound    = errors.New("license file not found")
	ErrLicenseMalformed   = errors.New("license file is malformed")
	ErrLicenseRevoked     = errors.New("license has been revoked")
	ErrFeatureNotLicensed = errors.New("feature not included in license")
	ErrAirGapModeRequired = errors.New("operation not available in air-gapped mode")
)

// LicenseType represents the tier of license.
type LicenseType string

const (
	LicenseTypeCommunity  LicenseType = "community"
	LicenseTypePro        LicenseType = "pro"
	LicenseTypeEnterprise LicenseType = "enterprise"
)

// Feature represents a licensable feature.
type Feature string

const (
	FeatureAirGapMode       Feature = "airgap_mode"
	FeatureMultiOrg         Feature = "multi_org"
	FeatureAdvancedReports  Feature = "advanced_reports"
	FeatureGeoReplication   Feature = "geo_replication"
	FeatureRansomwareDetect Feature = "ransomware_detection"
	FeatureLegalHolds       Feature = "legal_holds"
	FeatureDisasterRecovery Feature = "disaster_recovery"
	FeatureCustomBranding   Feature = "custom_branding"
	FeaturePrioritySupport  Feature = "priority_support"
	FeatureUnlimitedAgents  Feature = "unlimited_agents"
)

// License represents a validated license.
type License struct {
	ID             string            `json:"id"`
	Type           LicenseType       `json:"type"`
	Organization   string            `json:"organization"`
	Email          string            `json:"email"`
	MaxAgents      int               `json:"max_agents"`
	Features       []Feature         `json:"features"`
	IssuedAt       time.Time         `json:"issued_at"`
	ExpiresAt      time.Time         `json:"expires_at"`
	AirGapEnabled  bool              `json:"airgap_enabled"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	HardwareID     string            `json:"hardware_id,omitempty"`
	RevocationHash string            `json:"-"`
}

// IsExpired returns true if the license has expired.
func (l *License) IsExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

// DaysUntilExpiry returns the number of days until the license expires.
func (l *License) DaysUntilExpiry() int {
	duration := time.Until(l.ExpiresAt)
	if duration < 0 {
		return 0
	}
	return int(duration.Hours() / 24)
}

// HasFeature checks if the license includes a specific feature.
func (l *License) HasFeature(f Feature) bool {
	for _, feature := range l.Features {
		if feature == f {
			return true
		}
	}
	return false
}

// IsAirGapMode returns true if this is an air-gap license.
func (l *License) IsAirGapMode() bool {
	return l.AirGapEnabled && l.HasFeature(FeatureAirGapMode)
}

// LicenseFile represents the signed license file format.
type LicenseFile struct {
	Version   int    `json:"version"`
	License   string `json:"license"`   // Base64-encoded JSON
	Signature string `json:"signature"` // Base64-encoded Ed25519 signature
}

// AirGapConfig holds configuration for air-gapped operation.
type AirGapConfig struct {
	Enabled                bool   `json:"enabled"`
	LicensePath            string `json:"license_path"`
	UpdatePackagePath      string `json:"update_package_path"`
	DocumentationPath      string `json:"documentation_path"`
	RevocationListPath     string `json:"revocation_list_path"`
	DisableUpdateChecker   bool   `json:"disable_update_checker"`
	DisableTelemetry       bool   `json:"disable_telemetry"`
	DisableExternalLinks   bool   `json:"disable_external_links"`
	OfflineDocsVersion     string `json:"offline_docs_version"`
	LastRevocationCheck    time.Time `json:"last_revocation_check"`
}

// DefaultAirGapConfig returns the default air-gap configuration.
func DefaultAirGapConfig() AirGapConfig {
	return AirGapConfig{
		Enabled:              false,
		LicensePath:          "/etc/keldris/license.json",
		UpdatePackagePath:    "/var/lib/keldris/updates",
		DocumentationPath:    "/var/lib/keldris/docs",
		RevocationListPath:   "/etc/keldris/revocations.json",
		DisableUpdateChecker: true,
		DisableTelemetry:     true,
		DisableExternalLinks: true,
	}
}

// Manager handles license validation and management for air-gapped environments.
type Manager struct {
	config        AirGapConfig
	publicKey     ed25519.PublicKey
	license       *License
	revocationSet map[string]bool
	mu            sync.RWMutex
	logger        zerolog.Logger
}

// NewManager creates a new license manager.
func NewManager(config AirGapConfig, publicKeyBase64 string, logger zerolog.Logger) (*Manager, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}

	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
	}

	return &Manager{
		config:        config,
		publicKey:     ed25519.PublicKey(pubKeyBytes),
		revocationSet: make(map[string]bool),
		logger:        logger.With().Str("component", "license_manager").Logger(),
	}, nil
}

// LoadLicense loads and validates the license from the configured path.
func (m *Manager) LoadLicense(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load revocation list first
	if err := m.loadRevocationList(); err != nil {
		m.logger.Warn().Err(err).Msg("failed to load revocation list, continuing without it")
	}

	// Read license file
	data, err := os.ReadFile(m.config.LicensePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrLicenseNotFound
		}
		return fmt.Errorf("read license file: %w", err)
	}

	// Parse license file
	var licFile LicenseFile
	if err := json.Unmarshal(data, &licFile); err != nil {
		return ErrLicenseMalformed
	}

	// Decode license data
	licenseJSON, err := base64.StdEncoding.DecodeString(licFile.License)
	if err != nil {
		return ErrLicenseMalformed
	}

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(licFile.Signature)
	if err != nil {
		return ErrLicenseMalformed
	}

	// Verify signature
	if !ed25519.Verify(m.publicKey, licenseJSON, signature) {
		return ErrLicenseInvalid
	}

	// Parse license
	var license License
	if err := json.Unmarshal(licenseJSON, &license); err != nil {
		return ErrLicenseMalformed
	}

	// Compute revocation hash
	hash := sha256.Sum256([]byte(license.ID))
	license.RevocationHash = base64.StdEncoding.EncodeToString(hash[:])

	// Check revocation
	if m.revocationSet[license.RevocationHash] {
		return ErrLicenseRevoked
	}

	// Validate expiration
	if license.IsExpired() {
		m.license = &license // Still store for status display
		return ErrLicenseExpired
	}

	// Validate hardware ID if specified
	if license.HardwareID != "" {
		currentHWID, err := m.getHardwareID()
		if err != nil {
			m.logger.Warn().Err(err).Msg("failed to get hardware ID")
		} else if currentHWID != license.HardwareID {
			return ErrLicenseInvalid
		}
	}

	m.license = &license

	m.logger.Info().
		Str("license_id", license.ID).
		Str("type", string(license.Type)).
		Str("organization", license.Organization).
		Int("days_until_expiry", license.DaysUntilExpiry()).
		Bool("airgap_mode", license.IsAirGapMode()).
		Msg("license loaded successfully")

	return nil
}

// GetLicense returns the current license (may be nil).
func (m *Manager) GetLicense() *License {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.license
}

// GetConfig returns the air-gap configuration.
func (m *Manager) GetConfig() AirGapConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// IsAirGapMode returns true if running in air-gapped mode.
func (m *Manager) IsAirGapMode() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}

// IsValid returns true if there is a valid, non-expired license.
func (m *Manager) IsValid() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.license != nil && !m.license.IsExpired()
}

// CheckFeature returns an error if the feature is not licensed.
func (m *Manager) CheckFeature(f Feature) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.license == nil {
		return ErrLicenseNotFound
	}

	if m.license.IsExpired() {
		return ErrLicenseExpired
	}

	if !m.license.HasFeature(f) {
		return ErrFeatureNotLicensed
	}

	return nil
}

// loadRevocationList loads the revocation list from disk.
func (m *Manager) loadRevocationList() error {
	data, err := os.ReadFile(m.config.RevocationListPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No revocation list is fine
		}
		return err
	}

	var revocations struct {
		Version int      `json:"version"`
		Hashes  []string `json:"hashes"`
	}

	if err := json.Unmarshal(data, &revocations); err != nil {
		return err
	}

	m.revocationSet = make(map[string]bool, len(revocations.Hashes))
	for _, hash := range revocations.Hashes {
		m.revocationSet[hash] = true
	}

	m.config.LastRevocationCheck = time.Now()
	return nil
}

// UpdateRevocationList updates the revocation list from a new file.
func (m *Manager) UpdateRevocationList(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate format
	var revocations struct {
		Version   int      `json:"version"`
		Hashes    []string `json:"hashes"`
		Signature string   `json:"signature"`
	}

	if err := json.Unmarshal(data, &revocations); err != nil {
		return fmt.Errorf("invalid revocation list format: %w", err)
	}

	// Verify signature
	hashData, _ := json.Marshal(struct {
		Version int      `json:"version"`
		Hashes  []string `json:"hashes"`
	}{revocations.Version, revocations.Hashes})

	sig, err := base64.StdEncoding.DecodeString(revocations.Signature)
	if err != nil {
		return fmt.Errorf("invalid signature encoding: %w", err)
	}

	if !ed25519.Verify(m.publicKey, hashData, sig) {
		return errors.New("revocation list signature verification failed")
	}

	// Write to disk
	if err := os.WriteFile(m.config.RevocationListPath, data, 0600); err != nil {
		return fmt.Errorf("write revocation list: %w", err)
	}

	// Update in memory
	m.revocationSet = make(map[string]bool, len(revocations.Hashes))
	for _, hash := range revocations.Hashes {
		m.revocationSet[hash] = true
	}

	// Check if current license is now revoked
	if m.license != nil && m.revocationSet[m.license.RevocationHash] {
		m.logger.Warn().Str("license_id", m.license.ID).Msg("current license has been revoked")
		return ErrLicenseRevoked
	}

	m.config.LastRevocationCheck = time.Now()
	m.logger.Info().Int("count", len(revocations.Hashes)).Msg("revocation list updated")

	return nil
}

// ApplyNewLicense validates and applies a new license file.
func (m *Manager) ApplyNewLicense(data []byte) error {
	// Write to a temp file first
	tempPath := m.config.LicensePath + ".new"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("write temp license: %w", err)
	}

	// Backup existing license
	if _, err := os.Stat(m.config.LicensePath); err == nil {
		backupPath := m.config.LicensePath + ".bak"
		if err := os.Rename(m.config.LicensePath, backupPath); err != nil {
			os.Remove(tempPath)
			return fmt.Errorf("backup existing license: %w", err)
		}
	}

	// Move new license into place
	if err := os.Rename(tempPath, m.config.LicensePath); err != nil {
		return fmt.Errorf("install new license: %w", err)
	}

	// Reload
	if err := m.LoadLicense(context.Background()); err != nil {
		// Restore backup if validation fails
		backupPath := m.config.LicensePath + ".bak"
		if _, statErr := os.Stat(backupPath); statErr == nil {
			os.Rename(backupPath, m.config.LicensePath)
			m.LoadLicense(context.Background())
		}
		return fmt.Errorf("validate new license: %w", err)
	}

	m.logger.Info().Msg("new license applied successfully")
	return nil
}

// getHardwareID generates a hardware fingerprint for license binding.
func (m *Manager) getHardwareID() (string, error) {
	// Collect hardware identifiers
	var identifiers []string

	// Machine ID (Linux)
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		identifiers = append(identifiers, string(data))
	}

	// Product UUID (Linux)
	if data, err := os.ReadFile("/sys/class/dmi/id/product_uuid"); err == nil {
		identifiers = append(identifiers, string(data))
	}

	// Fallback: hostname
	if hostname, err := os.Hostname(); err == nil {
		identifiers = append(identifiers, hostname)
	}

	if len(identifiers) == 0 {
		return "", errors.New("no hardware identifiers available")
	}

	// Hash all identifiers
	hash := sha256.New()
	for _, id := range identifiers {
		hash.Write([]byte(id))
	}

	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

// GenerateRenewalRequest creates a license renewal request for manual submission.
func (m *Manager) GenerateRenewalRequest() (*RenewalRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.license == nil {
		return nil, ErrLicenseNotFound
	}

	hwID, _ := m.getHardwareID()

	return &RenewalRequest{
		LicenseID:    m.license.ID,
		Organization: m.license.Organization,
		Email:        m.license.Email,
		CurrentType:  m.license.Type,
		HardwareID:   hwID,
		RequestedAt:  time.Now(),
		ExpiresAt:    m.license.ExpiresAt,
	}, nil
}

// RenewalRequest represents a license renewal request.
type RenewalRequest struct {
	LicenseID    string      `json:"license_id"`
	Organization string      `json:"organization"`
	Email        string      `json:"email"`
	CurrentType  LicenseType `json:"current_type"`
	HardwareID   string      `json:"hardware_id"`
	RequestedAt  time.Time   `json:"requested_at"`
	ExpiresAt    time.Time   `json:"expires_at"`
}

// GetUpdatePackages lists available offline update packages.
func (m *Manager) GetUpdatePackages() ([]UpdatePackage, error) {
	m.mu.RLock()
	updatePath := m.config.UpdatePackagePath
	m.mu.RUnlock()

	entries, err := os.ReadDir(updatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read update directory: %w", err)
	}

	var packages []UpdatePackage
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isUpdatePackage(name) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		packages = append(packages, UpdatePackage{
			Filename:  name,
			Path:      filepath.Join(updatePath, name),
			Size:      info.Size(),
			CreatedAt: info.ModTime(),
		})
	}

	return packages, nil
}

// UpdatePackage represents an offline update package.
type UpdatePackage struct {
	Filename  string    `json:"filename"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	Version   string    `json:"version,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// isUpdatePackage checks if a filename looks like an update package.
func isUpdatePackage(name string) bool {
	// Expected format: keldris-update-vX.Y.Z.tar.gz or keldris-update-vX.Y.Z.zip
	return len(name) > 20 &&
		(filepath.Ext(name) == ".gz" || filepath.Ext(name) == ".zip") &&
		name[:15] == "keldris-update-"
}

// LicenseStatus represents the current license status for API responses.
type LicenseStatus struct {
	Valid           bool              `json:"valid"`
	Type            LicenseType       `json:"type,omitempty"`
	Organization    string            `json:"organization,omitempty"`
	ExpiresAt       *time.Time        `json:"expires_at,omitempty"`
	DaysUntilExpiry int               `json:"days_until_expiry,omitempty"`
	AirGapMode      bool              `json:"airgap_mode"`
	Features        []Feature         `json:"features,omitempty"`
	MaxAgents       int               `json:"max_agents,omitempty"`
	Error           string            `json:"error,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// GetStatus returns the current license status.
func (m *Manager) GetStatus() LicenseStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := LicenseStatus{
		AirGapMode: m.config.Enabled,
	}

	if m.license == nil {
		status.Error = "No license installed"
		return status
	}

	status.Type = m.license.Type
	status.Organization = m.license.Organization
	status.ExpiresAt = &m.license.ExpiresAt
	status.DaysUntilExpiry = m.license.DaysUntilExpiry()
	status.Features = m.license.Features
	status.MaxAgents = m.license.MaxAgents
	status.Metadata = m.license.Metadata

	if m.license.IsExpired() {
		status.Error = "License has expired"
	} else {
		status.Valid = true
	}

	return status
}
