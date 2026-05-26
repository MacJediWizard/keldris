package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func makeValidKey(t *testing.T, payload ed25519LicensePayload) (string, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("genkey: %v", err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	sig := ed25519.Sign(priv, body)
	key := base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig)
	return key, pub
}

func TestParseLicenseKeyEd25519_EmptyKey(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseLicenseKeyEd25519("", pub)
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestParseLicenseKeyEd25519_InvalidPublicKey(t *testing.T) {
	_, err := ParseLicenseKeyEd25519("a.b", []byte{1, 2, 3})
	if err == nil {
		t.Error("expected error for invalid public key length")
	}
}

func TestParseLicenseKeyEd25519_InvalidFormat(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseLicenseKeyEd25519("no-dot", pub)
	if err == nil {
		t.Error("expected error for missing dot separator")
	}
}

func TestParseLicenseKeyEd25519_InvalidPayloadBase64(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseLicenseKeyEd25519("@@@.def", pub)
	if err == nil {
		t.Error("expected error for invalid base64 payload")
	}
}

func TestParseLicenseKeyEd25519_InvalidSigBase64(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseLicenseKeyEd25519("YWJj.@@@", pub)
	if err == nil {
		t.Error("expected error for invalid base64 signature")
	}
}

func TestParseLicenseKeyEd25519_BadSignature(t *testing.T) {
	// payload signed with key A, verify with key B
	payload := ed25519LicensePayload{Tier: TierPro, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	body, _ := json.Marshal(payload)
	_, badPriv, _ := ed25519.GenerateKey(rand.Reader)
	sig := ed25519.Sign(badPriv, body)
	key := base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig)

	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, err := ParseLicenseKeyEd25519(key, otherPub)
	if err == nil {
		t.Error("expected error for signature/key mismatch")
	}
}

func TestParseLicenseKeyEd25519_ValidProTier(t *testing.T) {
	payload := ed25519LicensePayload{
		Product:    "keldris",
		Tier:       TierPro,
		CustomerID: "cust-123",
		IssuedAt:   time.Now().Add(-time.Hour).Unix(),
		ExpiresAt:  time.Now().Add(24 * time.Hour).Unix(),
	}
	key, pub := makeValidKey(t, payload)

	lic, err := ParseLicenseKeyEd25519(key, pub)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if lic.Tier != TierPro {
		t.Errorf("expected TierPro, got %s", lic.Tier)
	}
	if lic.CustomerID != "cust-123" {
		t.Errorf("expected cust-123, got %s", lic.CustomerID)
	}
}

func TestParseLicenseKeyEd25519_ValidEnterpriseTier(t *testing.T) {
	payload := ed25519LicensePayload{
		Tier:      TierEnterprise,
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	key, pub := makeValidKey(t, payload)

	lic, err := ParseLicenseKeyEd25519(key, pub)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if lic.Tier != TierEnterprise {
		t.Errorf("expected TierEnterprise, got %s", lic.Tier)
	}
}

func TestParseLicenseKeyEd25519_InvalidTier(t *testing.T) {
	payload := ed25519LicensePayload{
		Tier:      LicenseTier("ultra-mega"),
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	}
	key, pub := makeValidKey(t, payload)

	_, err := ParseLicenseKeyEd25519(key, pub)
	if err == nil {
		t.Error("expected error for unknown tier")
	}
}

func TestParseLicenseKeyEd25519_InvalidJSONPayload(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	body := []byte("not json")
	sig := ed25519.Sign(priv, body)
	key := base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig)
	pub := priv.Public().(ed25519.PublicKey)

	_, err := ParseLicenseKeyEd25519(key, pub)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
