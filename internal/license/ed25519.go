package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ed25519LicensePayload is the JSON structure encoded in an Ed25519-signed license key.
type ed25519LicensePayload struct {
	Product    string      `json:"product"`
	Tier       LicenseTier `json:"tier"`
	CustomerID string      `json:"customer_id"`
	ExpiresAt  int64       `json:"expires_at"`
	IssuedAt   int64       `json:"issued_at"`
}

// ParseLicenseKeyEd25519 decodes and verifies an Ed25519-signed license key.
// The key format is: base64(payload).base64(signature)
// This function only contains verification logic (safe for public repos).
func ParseLicenseKeyEd25519(key string, publicKey ed25519.PublicKey) (*License, error) {
	if key == "" {
		return nil, errors.New("empty license key")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key")
	}

	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid license key format: expected payload.signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode license payload: %w", err)
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode license signature: %w", err)
	}

	if !ed25519.Verify(publicKey, payloadBytes, sigBytes) {
		return nil, errors.New("invalid license signature")
	}

	var payload ed25519LicensePayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse license payload: %w", err)
	}

	if !payload.Tier.IsValid() {
		return nil, fmt.Errorf("unknown license tier: %s", payload.Tier)
	}

	lic := &License{
		Tier:       payload.Tier,
		CustomerID: payload.CustomerID,
		ExpiresAt:  time.Unix(payload.ExpiresAt, 0),
		IssuedAt:   time.Unix(payload.IssuedAt, 0),
		Limits:     GetLimits(payload.Tier),
	}

	return lic, nil
}
