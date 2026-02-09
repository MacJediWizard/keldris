package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"
)

func TestLicenseTier_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		tier  LicenseTier
		valid bool
	}{
		{"free tier is valid", TierFree, true},
		{"pro tier is valid", TierPro, true},
		{"enterprise tier is valid", TierEnterprise, true},
		{"empty tier is invalid", LicenseTier(""), false},
		{"unknown tier is invalid", LicenseTier("unknown"), false},
		{"random tier is invalid", LicenseTier("basic"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tier.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestValidTiers(t *testing.T) {
	tiers := ValidTiers()

	if len(tiers) != 3 {
		t.Errorf("ValidTiers() returned %d tiers, want 3", len(tiers))
	}

	expected := map[LicenseTier]bool{
		TierFree:       false,
		TierPro:        false,
		TierEnterprise: false,
	}

	for _, tier := range tiers {
		if _, ok := expected[tier]; !ok {
			t.Errorf("ValidTiers() returned unexpected tier: %s", tier)
		}
		expected[tier] = true
	}

	for tier, found := range expected {
		if !found {
			t.Errorf("ValidTiers() missing expected tier: %s", tier)
		}
	}
}

func createValidLicenseKey(tier LicenseTier, customerID string, expiresAt, issuedAt int64) string {
	payload := licensePayload{
		Tier:       tier,
		CustomerID: customerID,
		ExpiresAt:  expiresAt,
		IssuedAt:   issuedAt,
	}

	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	key := []byte("keldris-license-signing-key")
	h := hmac.New(sha256.New, key)
	h.Write(payloadJSON)
	signature := h.Sum(nil)
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return payloadB64 + "." + signatureB64
}

func TestParseLicense(t *testing.T) {
	now := time.Now()
	futureExpiry := now.Add(365 * 24 * time.Hour).Unix()
	issuedAt := now.Unix()

	validKey := createValidLicenseKey(TierPro, "customer-123", futureExpiry, issuedAt)

	t.Run("valid license key", func(t *testing.T) {
		lic, err := ParseLicense(validKey)
		if err != nil {
			t.Fatalf("ParseLicense() error = %v, want nil", err)
		}
		if lic.Tier != TierPro {
			t.Errorf("Tier = %v, want %v", lic.Tier, TierPro)
		}
		if lic.CustomerID != "customer-123" {
			t.Errorf("CustomerID = %v, want customer-123", lic.CustomerID)
		}
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := ParseLicense("")
		if err == nil {
			t.Error("ParseLicense() error = nil, want error")
		}
	})

	t.Run("missing dot separator", func(t *testing.T) {
		_, err := ParseLicense("no-dot-separator")
		if err == nil {
			t.Error("ParseLicense() error = nil, want error")
		}
	})

	t.Run("invalid base64 in payload", func(t *testing.T) {
		_, err := ParseLicense("invalid!!!.validbase64")
		if err == nil {
			t.Error("ParseLicense() error = nil, want error")
		}
	})

	t.Run("invalid base64 in signature", func(t *testing.T) {
		payload := licensePayload{
			Tier:       TierPro,
			CustomerID: "customer-123",
			ExpiresAt:  futureExpiry,
			IssuedAt:   issuedAt,
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

		_, err := ParseLicense(payloadB64 + ".invalid!!!")
		if err == nil {
			t.Error("ParseLicense() error = nil, want error")
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		payload := licensePayload{
			Tier:       TierPro,
			CustomerID: "customer-123",
			ExpiresAt:  futureExpiry,
			IssuedAt:   issuedAt,
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

		wrongSignature := base64.RawURLEncoding.EncodeToString([]byte("wrong-signature-value-here"))
		_, err := ParseLicense(payloadB64 + "." + wrongSignature)
		if err == nil {
			t.Error("ParseLicense() error = nil, want error")
		}
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		invalidJSON := []byte("{invalid json")
		payloadB64 := base64.RawURLEncoding.EncodeToString(invalidJSON)

		key := []byte("keldris-license-signing-key")
		h := hmac.New(sha256.New, key)
		h.Write(invalidJSON)
		signature := h.Sum(nil)
		signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

		_, err := ParseLicense(payloadB64 + "." + signatureB64)
		if err == nil {
			t.Error("ParseLicense() error = nil, want error")
		}
	})

	t.Run("unknown tier", func(t *testing.T) {
		unknownTierKey := createValidLicenseKey(LicenseTier("unknown"), "customer-123", futureExpiry, issuedAt)
		_, err := ParseLicense(unknownTierKey)
		if err == nil {
			t.Error("ParseLicense() error = nil, want error for unknown tier")
		}
	})
}

func TestValidateLicense(t *testing.T) {
	now := time.Now()

	t.Run("nil license", func(t *testing.T) {
		err := ValidateLicense(nil)
		if err == nil {
			t.Error("ValidateLicense() error = nil, want error for nil license")
		}
	})

	t.Run("invalid tier", func(t *testing.T) {
		lic := &License{
			Tier:       LicenseTier("invalid"),
			CustomerID: "customer-123",
			ExpiresAt:  now.Add(365 * 24 * time.Hour),
			IssuedAt:   now,
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("ValidateLicense() error = nil, want error for invalid tier")
		}
	})

	t.Run("expired license", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "customer-123",
			ExpiresAt:  now.Add(-24 * time.Hour),
			IssuedAt:   now.Add(-48 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("ValidateLicense() error = nil, want error for expired license")
		}
	})

	t.Run("empty customer ID", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "",
			ExpiresAt:  now.Add(365 * 24 * time.Hour),
			IssuedAt:   now,
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("ValidateLicense() error = nil, want error for empty customer ID")
		}
	})

	t.Run("valid license", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "customer-123",
			ExpiresAt:  now.Add(365 * 24 * time.Hour),
			IssuedAt:   now,
		}
		err := ValidateLicense(lic)
		if err != nil {
			t.Errorf("ValidateLicense() error = %v, want nil", err)
		}
	})
}

func TestFreeLicense(t *testing.T) {
	lic := FreeLicense()

	if lic == nil {
		t.Fatal("FreeLicense() returned nil")
	}

	if lic.Tier != TierFree {
		t.Errorf("Tier = %v, want %v", lic.Tier, TierFree)
	}

	if lic.CustomerID == "" {
		t.Error("CustomerID is empty, want non-empty")
	}

	if time.Until(lic.ExpiresAt) < 50*365*24*time.Hour {
		t.Error("ExpiresAt is not far enough in the future")
	}

	err := ValidateLicense(lic)
	if err != nil {
		t.Errorf("ValidateLicense() error = %v, want nil for free license", err)
	}
}
