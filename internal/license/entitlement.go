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

// Entitlement represents a parsed and verified entitlement token.
type Entitlement struct {
	Tier      LicenseTier      `json:"tier"`
	Features  []Feature        `json:"features"`
	Limits    TierLimits       `json:"limits"`
	Nonce     string           `json:"nonce"`
	IssuedAt  time.Time        `json:"issued_at"`
	ExpiresAt time.Time        `json:"expires_at"`
}

// entitlementPayload is the JSON structure in a signed entitlement token.
type entitlementPayload struct {
	Tier      LicenseTier      `json:"tier"`
	Features  []string         `json:"features"`
	Limits    map[string]int64 `json:"limits"`
	Nonce     string           `json:"nonce,omitempty"`
	IssuedAt  int64            `json:"issued_at"`
	ExpiresAt int64            `json:"expires_at"`
}

// ParseEntitlementToken decodes and verifies a signed entitlement token.
// Format: base64url(payload).base64url(signature)
func ParseEntitlementToken(token string, publicKey ed25519.PublicKey) (*Entitlement, error) {
	if token == "" {
		return nil, errors.New("empty entitlement token")
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, errors.New("invalid Ed25519 public key")
	}

	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("invalid entitlement token format")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode entitlement payload: %w", err)
	}

	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode entitlement signature: %w", err)
	}

	if !ed25519.Verify(publicKey, payloadBytes, sigBytes) {
		return nil, errors.New("invalid entitlement token signature")
	}

	var payload entitlementPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("parse entitlement payload: %w", err)
	}

	features := make([]Feature, len(payload.Features))
	for i, f := range payload.Features {
		features[i] = Feature(f)
	}

	limits := limitsFromMap(payload.Limits)

	return &Entitlement{
		Tier:      payload.Tier,
		Features:  features,
		Limits:    limits,
		Nonce:     payload.Nonce,
		IssuedAt:  time.Unix(payload.IssuedAt, 0),
		ExpiresAt: time.Unix(payload.ExpiresAt, 0),
	}, nil
}

// IsExpired returns true if the entitlement token has expired.
func (e *Entitlement) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// HasFeature checks if the entitlement includes a specific feature.
func (e *Entitlement) HasFeature(feature Feature) bool {
	for _, f := range e.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// limitsFromMap converts a map of limit keys to TierLimits.
func limitsFromMap(m map[string]int64) TierLimits {
	limits := TierLimits{}
	if v, ok := m["max_agents"]; ok {
		limits.MaxAgents = int(v)
	} else if v, ok := m["agents"]; ok {
		limits.MaxAgents = int(v)
	}
	if v, ok := m["max_users"]; ok {
		limits.MaxUsers = int(v)
	} else if v, ok := m["users"]; ok {
		limits.MaxUsers = int(v)
	}
	if v, ok := m["max_orgs"]; ok {
		limits.MaxOrgs = int(v)
	} else if v, ok := m["orgs"]; ok {
		limits.MaxOrgs = int(v)
	}
	return limits
}
