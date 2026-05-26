package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func makeEntToken(t *testing.T, payload entitlementPayload) (string, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	body, _ := json.Marshal(payload)
	sig := ed25519.Sign(priv, body)
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig), pub
}

func TestParseEntitlementToken_Empty(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseEntitlementToken("", pub)
	if err == nil {
		t.Error("expected error for empty token")
	}
}

func TestParseEntitlementToken_BadPublicKey(t *testing.T) {
	_, err := ParseEntitlementToken("a.b", []byte{1})
	if err == nil {
		t.Error("expected error for invalid public key")
	}
}

func TestParseEntitlementToken_InvalidFormat(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseEntitlementToken("nodelimiter", pub)
	if err == nil {
		t.Error("expected error for missing delimiter")
	}
}

func TestParseEntitlementToken_ValidWithFeatures(t *testing.T) {
	payload := entitlementPayload{
		Tier:      TierPro,
		Features:  []string{"oidc", "api_access"},
		Limits:    map[string]int64{"max_agents": 50, "max_users": 10},
		Nonce:     "abc-nonce",
		IssuedAt:  time.Now().Add(-time.Hour).Unix(),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	token, pub := makeEntToken(t, payload)

	ent, err := ParseEntitlementToken(token, pub)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ent.Tier != TierPro {
		t.Errorf("expected TierPro, got %s", ent.Tier)
	}
	if len(ent.Features) != 2 {
		t.Errorf("expected 2 features, got %d", len(ent.Features))
	}
	if ent.Limits.MaxAgents != 50 {
		t.Errorf("expected MaxAgents=50, got %d", ent.Limits.MaxAgents)
	}
	if ent.Nonce != "abc-nonce" {
		t.Errorf("expected nonce abc-nonce, got %s", ent.Nonce)
	}
}

func TestParseEntitlementToken_BadSignature(t *testing.T) {
	payload := entitlementPayload{Tier: TierPro, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	body, _ := json.Marshal(payload)
	_, badPriv, _ := ed25519.GenerateKey(rand.Reader)
	sig := ed25519.Sign(badPriv, body)
	token := base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig)

	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseEntitlementToken(token, pub)
	if err == nil {
		t.Error("expected error for bad signature")
	}
}

func TestEntitlement_IsExpired(t *testing.T) {
	expired := &Entitlement{ExpiresAt: time.Now().Add(-time.Hour)}
	if !expired.IsExpired() {
		t.Error("expected expired=true for past expiry")
	}

	live := &Entitlement{ExpiresAt: time.Now().Add(time.Hour)}
	if live.IsExpired() {
		t.Error("expected expired=false for future expiry")
	}
}

func TestEntitlement_HasFeature(t *testing.T) {
	ent := &Entitlement{
		Features: []Feature{FeatureOIDC, FeatureAPIAccess},
	}
	if !ent.HasFeature(FeatureOIDC) {
		t.Error("expected to have FeatureOIDC")
	}
	if ent.HasFeature(FeatureMultiOrg) {
		t.Error("expected NOT to have FeatureMultiOrg")
	}
}

func TestLimitsFromMap_PrefersMaxPrefix(t *testing.T) {
	limits := limitsFromMap(map[string]int64{"max_agents": 5, "agents": 99})
	if limits.MaxAgents != 5 {
		t.Errorf("expected max_agents to win, got %d", limits.MaxAgents)
	}
}

func TestLimitsFromMap_FallbackKeys(t *testing.T) {
	limits := limitsFromMap(map[string]int64{"agents": 3, "users": 7, "orgs": 2})
	if limits.MaxAgents != 3 {
		t.Errorf("expected agents=3, got %d", limits.MaxAgents)
	}
	if limits.MaxUsers != 7 {
		t.Errorf("expected users=7, got %d", limits.MaxUsers)
	}
	if limits.MaxOrgs != 2 {
		t.Errorf("expected orgs=2, got %d", limits.MaxOrgs)
	}
}
