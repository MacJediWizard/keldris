// Package license provides license management and feature gating for Keldris.
package license

import (
	"errors"
	"fmt"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// LicenseTier represents the subscription level.
type LicenseTier string

const (
	// TierFree is the default tier with basic functionality.
	TierFree LicenseTier = "free"
	// TierPro unlocks advanced features like OIDC and audit logs.
	TierPro LicenseTier = "pro"
	// TierEnterprise unlocks all features including multi-org.
	// TierEnterprise unlocks all features including multi-org and air-gap.
	TierEnterprise LicenseTier = "enterprise"
)

// ValidTiers returns all valid license tiers.
func ValidTiers() []LicenseTier {
	return []LicenseTier{TierFree, TierPro, TierEnterprise}
}

// IsValid checks if the tier is a recognized value.
func (t LicenseTier) IsValid() bool {
	for _, valid := range ValidTiers() {
		if t == valid {
			return true
		}
	}
	return false
}

// License represents a Keldris license with tier, limits, and expiry.
type License struct {
	Tier              LicenseTier `json:"tier"`
	CustomerID        string      `json:"customer_id"`
	CustomerName      string      `json:"customer_name,omitempty"`
	ExpiresAt         time.Time   `json:"expires_at"`
	IssuedAt          time.Time   `json:"issued_at"`
	Limits            TierLimits  `json:"limits"`
	IsTrial           bool        `json:"is_trial"`
	TrialDurationDays int         `json:"trial_duration_days,omitempty"`
	TrialStartedAt    time.Time   `json:"trial_started_at,omitempty"`
}

// TrialDaysLeft returns the number of days remaining in the trial, or 0 if not a trial.
func (l *License) TrialDaysLeft() int {
	if !l.IsTrial {
		return 0
	}
	days := int(time.Until(l.ExpiresAt).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// ValidateLicense checks that a license is still valid (not expired).
func ValidateLicense(license *License) error {
	if license == nil {
		return errors.New("nil license")
	}

	if !license.Tier.IsValid() {
		return fmt.Errorf("unknown license tier: %s", license.Tier)
	}

	if time.Now().After(license.ExpiresAt) {
		return errors.New("license has expired")
	}

	if license.CustomerID == "" {
		return errors.New("missing customer ID")
	}

	return nil
}

// FreeLicense returns a default free-tier license with no expiry constraint.
func FreeLicense() *License {
	return &License{
		Tier:       TierFree,
		CustomerID: "free",
		ExpiresAt:  time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC),
		IssuedAt:   time.Now(),
		Limits:     GetLimits(TierFree),
	}
// Package license provides license key generation and validation for Keldris.
package license

import (
// Package license provides license key generation and validation for Keldris.
package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

const (
	// LicenseKeyPrefix is the prefix for all license keys.
	LicenseKeyPrefix = "KLDRS"
	// LicenseKeyVersion is the current license key format version.
	LicenseKeyVersion = "1"
)

var (
	// ErrInvalidLicenseKey indicates the license key format is invalid.
	ErrInvalidLicenseKey = errors.New("invalid license key format")
	// ErrInvalidSignature indicates the license key signature is invalid.
	ErrInvalidSignature = errors.New("invalid license signature")
	// ErrLicenseExpired indicates the license has expired past grace period.
	ErrLicenseExpired = errors.New("license has expired")
	// ErrInvalidPublicKey indicates the public key is invalid.
	ErrInvalidPublicKey = errors.New("invalid public key")
	// ErrInvalidPrivateKey indicates the private key is invalid.
	ErrInvalidPrivateKey = errors.New("invalid private key")
)

// KeyPair holds Ed25519 signing keys.
type KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateKeyPair generates a new Ed25519 key pair for signing licenses.
func GenerateKeyPair() (*KeyPair, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate key pair: %w", err)
	}
	return &KeyPair{
		PublicKey:  public,
		PrivateKey: private,
	}, nil
}

// PublicKeyToBase64 encodes the public key to base64 for storage.
func (kp *KeyPair) PublicKeyToBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PublicKey)
}

// PrivateKeyToBase64 encodes the private key to base64 for storage.
func (kp *KeyPair) PrivateKeyToBase64() string {
	return base64.StdEncoding.EncodeToString(kp.PrivateKey)
}

// PublicKeyFromBase64 decodes a base64-encoded public key.
func PublicKeyFromBase64(encoded string) (ed25519.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(data) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	return ed25519.PublicKey(data), nil
}

// PrivateKeyFromBase64 decodes a base64-encoded private key.
func PrivateKeyFromBase64(encoded string) (ed25519.PrivateKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	if len(data) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPrivateKey
	}
	return ed25519.PrivateKey(data), nil
}

// Generator creates signed license keys.
type Generator struct {
	privateKey ed25519.PrivateKey
}

// NewGenerator creates a new license key generator with the given private key.
func NewGenerator(privateKey ed25519.PrivateKey) (*Generator, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, ErrInvalidPrivateKey
	}
	return &Generator{privateKey: privateKey}, nil
}

// NewGeneratorFromBase64 creates a generator from a base64-encoded private key.
func NewGeneratorFromBase64(encodedKey string) (*Generator, error) {
	privateKey, err := PrivateKeyFromBase64(encodedKey)
	if err != nil {
		return nil, err
	}
	return NewGenerator(privateKey)
}

// Generate creates a new signed license key.
func (g *Generator) Generate(payload *models.LicensePayload) (string, error) {
	if payload.ID == uuid.Nil {
		payload.ID = uuid.New()
	}
	if payload.IssuedAt.IsZero() {
		payload.IssuedAt = time.Now()
	}

	// Serialize payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	// Sign the payload
	signature := ed25519.Sign(g.privateKey, payloadBytes)

	// Encode payload and signature
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)

	// Format: PREFIX-VERSION.PAYLOAD.SIGNATURE
	licenseKey := fmt.Sprintf("%s-%s.%s.%s",
		LicenseKeyPrefix,
		LicenseKeyVersion,
		encodedPayload,
		encodedSignature,
	)

	return licenseKey, nil
}

// Validator validates license keys offline.
type Validator struct {
	publicKey ed25519.PublicKey
}

// NewValidator creates a new license key validator with the given public key.
func NewValidator(publicKey ed25519.PublicKey) (*Validator, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, ErrInvalidPublicKey
	}
	return &Validator{publicKey: publicKey}, nil
}

// NewValidatorFromBase64 creates a validator from a base64-encoded public key.
func NewValidatorFromBase64(encodedKey string) (*Validator, error) {
	publicKey, err := PublicKeyFromBase64(encodedKey)
	if err != nil {
		return nil, err
	}
	return NewValidator(publicKey)
}

// Validate validates a license key and returns the payload if valid.
func (v *Validator) Validate(licenseKey string) (*models.LicenseValidationResult, error) {
	result := &models.LicenseValidationResult{
		Valid:  false,
		Status: models.LicenseStatusInvalid,
	}

	// Parse the license key
	payload, err := v.parseAndVerify(licenseKey)
	if err != nil {
		result.Message = err.Error()
		return result, err
	}

	// Check expiry
	now := time.Now()
	result.Tier = payload.Tier
	result.Limits = payload.Limits
	result.Features = payload.Features
	result.ExpiresAt = payload.ExpiresAt

	if now.Before(payload.ExpiresAt) {
		result.Valid = true
		result.Status = models.LicenseStatusActive
		result.DaysRemaining = int(time.Until(payload.ExpiresAt).Hours() / 24)
		return result, nil
	}

	// Check grace period
	gracePeriodEnd := payload.ExpiresAt.AddDate(0, 0, models.GracePeriodDays)
	if now.Before(gracePeriodEnd) {
		result.Valid = true
		result.Status = models.LicenseStatusGracePeriod
		result.DaysRemaining = int(time.Until(gracePeriodEnd).Hours() / 24)
		result.Message = fmt.Sprintf("License expired, %d days remaining in grace period", result.DaysRemaining)
		return result, nil
	}

	// Expired past grace period
	result.Status = models.LicenseStatusExpired
	result.Message = "License has expired"
	return result, ErrLicenseExpired
}

// ParsePayload extracts the payload from a license key without validating signature.
// Use Validate for full validation.
func (v *Validator) ParsePayload(licenseKey string) (*models.LicensePayload, error) {
	return v.parseAndVerify(licenseKey)
}

// parseAndVerify parses and verifies the license key signature.
func (v *Validator) parseAndVerify(licenseKey string) (*models.LicensePayload, error) {
	// Check prefix
	if !strings.HasPrefix(licenseKey, LicenseKeyPrefix+"-") {
		return nil, ErrInvalidLicenseKey
	}

	// Remove prefix and parse version
	parts := strings.Split(licenseKey, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidLicenseKey
	}

	// Parse header (PREFIX-VERSION)
	header := parts[0]
	expectedHeader := LicenseKeyPrefix + "-" + LicenseKeyVersion
	if header != expectedHeader {
		return nil, ErrInvalidLicenseKey
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	// Decode signature
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode signature: %w", err)
	}

	// Verify signature
	if !ed25519.Verify(v.publicKey, payloadBytes, signature) {
		return nil, ErrInvalidSignature
	}

	// Parse payload
	var payload models.LicensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}

	return &payload, nil
}

// ValidateLicenseKey validates a license key without requiring a Validator instance.
// This is a convenience function for simple validation scenarios.
func ValidateLicenseKey(licenseKey string, publicKey ed25519.PublicKey) (*models.LicenseValidationResult, error) {
	validator, err := NewValidator(publicKey)
	if err != nil {
		return nil, err
	}
	return validator.Validate(licenseKey)
}

// Manager handles license validation and caching.
type Manager struct {
	validator     *Validator
	currentLicense *models.License
	lastValidated  time.Time
}

// NewManager creates a new license manager.
func NewManager(publicKey ed25519.PublicKey) (*Manager, error) {
	validator, err := NewValidator(publicKey)
	if err != nil {
		return nil, err
	}
	return &Manager{
		validator: validator,
	}, nil
}

// NewManagerFromBase64 creates a manager from a base64-encoded public key.
func NewManagerFromBase64(encodedKey string) (*Manager, error) {
	publicKey, err := PublicKeyFromBase64(encodedKey)
	if err != nil {
		return nil, err
	}
	return NewManager(publicKey)
}

// SetLicense sets the current license and validates it.
func (m *Manager) SetLicense(license *models.License) (*models.LicenseValidationResult, error) {
	result, err := m.validator.Validate(license.LicenseKey)
	if err != nil && err != ErrLicenseExpired {
		return result, err
	}

	m.currentLicense = license
	m.lastValidated = time.Now()
	return result, nil
}

// GetCurrentLicense returns the current license.
func (m *Manager) GetCurrentLicense() *models.License {
	return m.currentLicense
}

// GetValidationResult returns the current validation result.
func (m *Manager) GetValidationResult() (*models.LicenseValidationResult, error) {
	if m.currentLicense == nil {
		return &models.LicenseValidationResult{
			Valid:   false,
			Status:  models.LicenseStatusInvalid,
			Message: "No license configured",
		}, nil
	}
	return m.validator.Validate(m.currentLicense.LicenseKey)
}

// IsValid returns true if the current license is valid.
func (m *Manager) IsValid() bool {
	if m.currentLicense == nil {
		return false
	}
	result, err := m.validator.Validate(m.currentLicense.LicenseKey)
	if err != nil {
		return false
	}
	return result.Valid
}

// GetLimits returns the current license limits.
func (m *Manager) GetLimits() models.LicenseLimits {
	if m.currentLicense == nil {
		return models.DefaultCommunityLimits()
	}
	return m.currentLicense.Limits
}

// GetFeatures returns the current license features.
func (m *Manager) GetFeatures() models.LicenseFeatures {
	if m.currentLicense == nil {
		return models.DefaultCommunityFeatures()
	}
	return m.currentLicense.Features
}

// GetTier returns the current license tier.
func (m *Manager) GetTier() models.LicenseTier {
	if m.currentLicense == nil {
		return models.LicenseTierCommunity
	}
	return m.currentLicense.Tier
	Tier       LicenseTier `json:"tier"`
	CustomerID string      `json:"customer_id"`
	ExpiresAt  time.Time   `json:"expires_at"`
	IssuedAt   time.Time   `json:"issued_at"`
	Limits     TierLimits  `json:"limits"`
}

// licensePayload is the JSON structure encoded in a license key.
type licensePayload struct {
	Tier       LicenseTier `json:"tier"`
	CustomerID string      `json:"customer_id"`
	ExpiresAt  int64       `json:"expires_at"`
	IssuedAt   int64       `json:"issued_at"`
}

// signingKey is used to validate license keys. In production this would be
// loaded from configuration or an environment variable.
var signingKey = []byte("keldris-license-signing-key")

// SetSigningKey configures the HMAC key used for license validation.
func SetSigningKey(key []byte) {
	signingKey = key
}

// ParseLicense decodes and verifies a license key string.
// License keys are base64-encoded JSON payloads with an HMAC-SHA256 signature
// appended after a "." separator.
func ParseLicense(key string) (*License, error) {
	if key == "" {
		return nil, errors.New("empty license key")
	}

	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid license key format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode license payload: %w", err)
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode license signature: %w", err)
	}

	// Verify HMAC signature.
	mac := hmac.New(sha256.New, signingKey)
	mac.Write(payloadBytes)
	expectedMAC := mac.Sum(nil)
	if !hmac.Equal(sigBytes, expectedMAC) {
		return nil, errors.New("invalid license signature")
	}

	var payload licensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse license payload: %w", err)
	}

	if !payload.Tier.IsValid() {
		return nil, fmt.Errorf("unknown license tier: %s", payload.Tier)
	}

	license := &License{
		Tier:       payload.Tier,
		CustomerID: payload.CustomerID,
		ExpiresAt:  time.Unix(payload.ExpiresAt, 0),
		IssuedAt:   time.Unix(payload.IssuedAt, 0),
		Limits:     GetLimits(payload.Tier),
	}

	return license, nil
}

// ValidateLicense checks that a license is still valid (not expired).
func ValidateLicense(license *License) error {
	if license == nil {
		return errors.New("nil license")
	}

	if !license.Tier.IsValid() {
		return fmt.Errorf("unknown license tier: %s", license.Tier)
	}

	if time.Now().After(license.ExpiresAt) {
		return errors.New("license has expired")
	}

	if license.CustomerID == "" {
		return errors.New("missing customer ID")
	}

	return nil
}

// FreeLicense returns a default free-tier license with no expiry constraint.
func FreeLicense() *License {
	return &License{
		Tier:       TierFree,
		CustomerID: "free",
		ExpiresAt:  time.Date(2099, 12, 31, 23, 59, 59, 0, time.UTC),
		IssuedAt:   time.Now(),
		Limits:     GetLimits(TierFree),
	}
}
