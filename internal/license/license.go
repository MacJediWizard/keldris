// Package license provides license management and feature gating for Keldris.
package license

import (
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
