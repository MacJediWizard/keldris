package license

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// offlineLicensePayload is the JSON structure stored in an offline license file.
type offlineLicensePayload struct {
	Tier       LicenseTier      `json:"tier"`
	CustomerID string           `json:"customer_id"`
	ExpiresAt  int64            `json:"expires_at"`
	IssuedAt   int64            `json:"issued_at"`
	Features   []string         `json:"features,omitempty"`
	Limits     map[string]int64 `json:"limits,omitempty"`
}

// OfflineLicenseFile represents the full offline license file format:
// JSON payload + Ed25519 signature.
type OfflineLicenseFile struct {
	Payload   []byte `json:"payload"`
	Signature []byte `json:"signature"`
}

// GenerateOfflineLicense creates a signed offline license using Ed25519.
func GenerateOfflineLicense(customerID string, tier LicenseTier, expiry time.Time, privateKey ed25519.PrivateKey) ([]byte, error) {
	if customerID == "" {
		return nil, errors.New("customer ID is required")
	}
	if !tier.IsValid() {
		return nil, fmt.Errorf("invalid license tier: %s", tier)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key")
	}

	// Build features list for entitlement
	features := make([]string, 0)
	for _, f := range FeaturesForTier(tier) {
		features = append(features, string(f))
	}

	// Build limits map for entitlement
	tierLimits := GetLimits(tier)
	limits := map[string]int64{
		"max_agents":       int64(tierLimits.MaxAgents),
		"max_users":        int64(tierLimits.MaxUsers),
		"max_orgs":         int64(tierLimits.MaxOrgs),
		"max_storage_bytes": tierLimits.MaxStorage,
	}

	payload := offlineLicensePayload{
		Tier:       tier,
		CustomerID: customerID,
		ExpiresAt:  expiry.Unix(),
		IssuedAt:   time.Now().Unix(),
		Features:   features,
		Limits:     limits,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal license payload: %w", err)
	}

	signature := ed25519.Sign(privateKey, payloadBytes)

	licenseFile := OfflineLicenseFile{
		Payload:   payloadBytes,
		Signature: signature,
	}

	data, err := json.Marshal(licenseFile)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal license file: %w", err)
	}

	return data, nil
}

// ValidateOfflineLicense verifies and parses an offline license using Ed25519 public key.
// The returned license includes entitlement data (features + limits) from the offline package.
func ValidateOfflineLicense(licenseData []byte, publicKey []byte) (*License, error) {
	if len(licenseData) == 0 {
		return nil, errors.New("empty license data")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key")
	}

	var licenseFile OfflineLicenseFile
	if err := json.Unmarshal(licenseData, &licenseFile); err != nil {
		return nil, fmt.Errorf("failed to parse license file: %w", err)
	}

	if len(licenseFile.Payload) == 0 {
		return nil, errors.New("license file has empty payload")
	}
	if len(licenseFile.Signature) == 0 {
		return nil, errors.New("license file has empty signature")
	}

	if !ed25519.Verify(ed25519.PublicKey(publicKey), licenseFile.Payload, licenseFile.Signature) {
		return nil, errors.New("invalid license signature")
	}

	var payload offlineLicensePayload
	if err := json.Unmarshal(licenseFile.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse license payload: %w", err)
	}

	if !payload.Tier.IsValid() {
		return nil, fmt.Errorf("unknown license tier: %s", payload.Tier)
	}

	// Use limits from payload if available, otherwise fall back to tier defaults
	limits := GetLimits(payload.Tier)
	if payload.Limits != nil {
		limits = limitsFromMap(payload.Limits)
	}

	lic := &License{
		Tier:       payload.Tier,
		CustomerID: payload.CustomerID,
		ExpiresAt:  time.Unix(payload.ExpiresAt, 0),
		IssuedAt:   time.Unix(payload.IssuedAt, 0),
		Limits:     limits,
	}

	if err := ValidateLicense(lic); err != nil {
		return nil, fmt.Errorf("license validation failed: %w", err)
	}

	return lic, nil
}

// EntitlementFromOfflineLicense creates an Entitlement from an offline license payload.
// The entitlement uses the license expiry (not 24h) since there's no server to refresh from.
func EntitlementFromOfflineLicense(licenseData []byte, publicKey []byte) (*Entitlement, error) {
	if len(licenseData) == 0 {
		return nil, errors.New("empty license data")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key")
	}

	var licenseFile OfflineLicenseFile
	if err := json.Unmarshal(licenseData, &licenseFile); err != nil {
		return nil, fmt.Errorf("failed to parse license file: %w", err)
	}

	if !ed25519.Verify(ed25519.PublicKey(publicKey), licenseFile.Payload, licenseFile.Signature) {
		return nil, errors.New("invalid license signature")
	}

	var payload offlineLicensePayload
	if err := json.Unmarshal(licenseFile.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse license payload: %w", err)
	}

	// Build features list
	features := make([]Feature, 0)
	if payload.Features != nil {
		for _, f := range payload.Features {
			features = append(features, Feature(f))
		}
	} else {
		features = FeaturesForTier(payload.Tier)
	}

	// Build limits
	limits := GetLimits(payload.Tier)
	if payload.Limits != nil {
		limits = limitsFromMap(payload.Limits)
	}

	return &Entitlement{
		Tier:      payload.Tier,
		Features:  features,
		Limits:    limits,
		IssuedAt:  time.Unix(payload.IssuedAt, 0),
		ExpiresAt: time.Unix(payload.ExpiresAt, 0), // Use license expiry, not 24h
	}, nil
}
