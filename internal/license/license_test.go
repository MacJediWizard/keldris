package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
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

func TestLicense_Generate(t *testing.T) {
	now := time.Now()
	futureExpiry := now.Add(365 * 24 * time.Hour).Unix()
	issuedAt := now.Unix()

	tests := []struct {
		name       string
		tier       LicenseTier
		customerID string
	}{
		{"generate free tier key", TierFree, "cust-free-001"},
		{"generate pro tier key", TierPro, "cust-pro-001"},
		{"generate enterprise tier key", TierEnterprise, "cust-ent-001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := createValidLicenseKey(tt.tier, tt.customerID, futureExpiry, issuedAt)
			if key == "" {
				t.Fatal("createValidLicenseKey() returned empty string")
			}

			parts := strings.SplitN(key, ".", 2)
			if len(parts) != 2 {
				t.Fatal("generated key does not contain dot separator")
			}

			lic, err := ParseLicense(key)
			if err != nil {
				t.Fatalf("ParseLicense() error = %v, want nil", err)
			}
			if lic.Tier != tt.tier {
				t.Errorf("Tier = %v, want %v", lic.Tier, tt.tier)
			}
			if lic.CustomerID != tt.customerID {
				t.Errorf("CustomerID = %v, want %v", lic.CustomerID, tt.customerID)
			}
			if lic.ExpiresAt.Unix() != futureExpiry {
				t.Errorf("ExpiresAt = %v, want %v", lic.ExpiresAt.Unix(), futureExpiry)
			}
			if lic.IssuedAt.Unix() != issuedAt {
				t.Errorf("IssuedAt = %v, want %v", lic.IssuedAt.Unix(), issuedAt)
			}

			expectedLimits := GetLimits(tt.tier)
			if lic.Limits != expectedLimits {
				t.Errorf("Limits = %+v, want %+v", lic.Limits, expectedLimits)
			}
		})
	}

	t.Run("generated keys are unique per customer", func(t *testing.T) {
		key1 := createValidLicenseKey(TierPro, "customer-A", futureExpiry, issuedAt)
		key2 := createValidLicenseKey(TierPro, "customer-B", futureExpiry, issuedAt)
		if key1 == key2 {
			t.Error("different customers produced identical keys")
		}
	})

	t.Run("generated keys are unique per tier", func(t *testing.T) {
		key1 := createValidLicenseKey(TierPro, "customer-123", futureExpiry, issuedAt)
		key2 := createValidLicenseKey(TierEnterprise, "customer-123", futureExpiry, issuedAt)
		if key1 == key2 {
			t.Error("different tiers produced identical keys")
		}
	})
}

func TestLicense_Parse(t *testing.T) {
	now := time.Now()
	futureExpiry := now.Add(365 * 24 * time.Hour).Unix()
	issuedAt := now.Unix()

	t.Run("parse free tier key", func(t *testing.T) {
		key := createValidLicenseKey(TierFree, "free-cust", futureExpiry, issuedAt)
		lic, err := ParseLicense(key)
		if err != nil {
			t.Fatalf("ParseLicense() error = %v", err)
		}
		if lic.Tier != TierFree {
			t.Errorf("Tier = %v, want %v", lic.Tier, TierFree)
		}
		if lic.Limits.MaxAgents != 3 {
			t.Errorf("MaxAgents = %d, want 3", lic.Limits.MaxAgents)
		}
	})

	t.Run("parse enterprise tier key", func(t *testing.T) {
		key := createValidLicenseKey(TierEnterprise, "ent-cust", futureExpiry, issuedAt)
		lic, err := ParseLicense(key)
		if err != nil {
			t.Fatalf("ParseLicense() error = %v", err)
		}
		if lic.Tier != TierEnterprise {
			t.Errorf("Tier = %v, want %v", lic.Tier, TierEnterprise)
		}
		if !IsUnlimited(lic.Limits.MaxAgents) {
			t.Error("enterprise tier should have unlimited agents")
		}
	})

	t.Run("parse preserves timestamps", func(t *testing.T) {
		specificExpiry := int64(1893456000) // 2030-01-01
		specificIssued := int64(1704067200) // 2024-01-01

		key := createValidLicenseKey(TierPro, "ts-cust", specificExpiry, specificIssued)
		lic, err := ParseLicense(key)
		if err != nil {
			t.Fatalf("ParseLicense() error = %v", err)
		}
		if lic.ExpiresAt.Unix() != specificExpiry {
			t.Errorf("ExpiresAt = %d, want %d", lic.ExpiresAt.Unix(), specificExpiry)
		}
		if lic.IssuedAt.Unix() != specificIssued {
			t.Errorf("IssuedAt = %d, want %d", lic.IssuedAt.Unix(), specificIssued)
		}
	})

	t.Run("parse with custom signing key", func(t *testing.T) {
		originalKey := signingKey
		defer SetSigningKey(originalKey)

		customKey := []byte("custom-test-signing-key")
		SetSigningKey(customKey)

		payload := licensePayload{
			Tier:       TierPro,
			CustomerID: "custom-key-cust",
			ExpiresAt:  futureExpiry,
			IssuedAt:   issuedAt,
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

		h := hmac.New(sha256.New, customKey)
		h.Write(payloadJSON)
		sig := h.Sum(nil)
		sigB64 := base64.RawURLEncoding.EncodeToString(sig)

		lic, err := ParseLicense(payloadB64 + "." + sigB64)
		if err != nil {
			t.Fatalf("ParseLicense() error = %v, want nil", err)
		}
		if lic.CustomerID != "custom-key-cust" {
			t.Errorf("CustomerID = %v, want custom-key-cust", lic.CustomerID)
		}
	})

	t.Run("wrong signing key rejects license", func(t *testing.T) {
		originalKey := signingKey
		defer SetSigningKey(originalKey)

		// Generate key with default signing key
		key := createValidLicenseKey(TierPro, "cust", futureExpiry, issuedAt)

		// Switch to different signing key
		SetSigningKey([]byte("different-key"))

		_, err := ParseLicense(key)
		if err == nil {
			t.Error("ParseLicense() error = nil, want error for wrong signing key")
		}
	})
}

func TestLicense_Validate(t *testing.T) {
	now := time.Now()

	t.Run("valid free tier license", func(t *testing.T) {
		lic := &License{
			Tier:       TierFree,
			CustomerID: "free-cust",
			ExpiresAt:  now.Add(365 * 24 * time.Hour),
			IssuedAt:   now,
		}
		if err := ValidateLicense(lic); err != nil {
			t.Errorf("ValidateLicense() error = %v, want nil", err)
		}
	})

	t.Run("valid enterprise tier license", func(t *testing.T) {
		lic := &License{
			Tier:       TierEnterprise,
			CustomerID: "ent-cust",
			ExpiresAt:  now.Add(365 * 24 * time.Hour),
			IssuedAt:   now,
		}
		if err := ValidateLicense(lic); err != nil {
			t.Errorf("ValidateLicense() error = %v, want nil", err)
		}
	})

	t.Run("license expiring in one second is valid", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust",
			ExpiresAt:  now.Add(1 * time.Second),
			IssuedAt:   now,
		}
		if err := ValidateLicense(lic); err != nil {
			t.Errorf("ValidateLicense() error = %v, want nil", err)
		}
	})

	t.Run("error message for expired license", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust",
			ExpiresAt:  now.Add(-1 * time.Hour),
			IssuedAt:   now.Add(-24 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Fatal("ValidateLicense() error = nil, want error")
		}
		if err.Error() != "license has expired" {
			t.Errorf("error message = %q, want %q", err.Error(), "license has expired")
		}
	})

	t.Run("error message for nil license", func(t *testing.T) {
		err := ValidateLicense(nil)
		if err == nil {
			t.Fatal("ValidateLicense() error = nil, want error")
		}
		if err.Error() != "nil license" {
			t.Errorf("error message = %q, want %q", err.Error(), "nil license")
		}
	})

	t.Run("error message for missing customer ID", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "",
			ExpiresAt:  now.Add(365 * 24 * time.Hour),
			IssuedAt:   now,
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Fatal("ValidateLicense() error = nil, want error")
		}
		if err.Error() != "missing customer ID" {
			t.Errorf("error message = %q, want %q", err.Error(), "missing customer ID")
		}
	})
}

func TestLicense_Expired(t *testing.T) {
	now := time.Now()

	t.Run("expired one hour ago", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(-1 * time.Hour),
			IssuedAt:   now.Add(-48 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("want error for license expired 1 hour ago")
		}
	})

	t.Run("expired one day ago", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(-24 * time.Hour),
			IssuedAt:   now.Add(-48 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("want error for license expired 1 day ago")
		}
	})

	t.Run("expired one year ago", func(t *testing.T) {
		lic := &License{
			Tier:       TierEnterprise,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(-365 * 24 * time.Hour),
			IssuedAt:   now.Add(-730 * 24 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("want error for license expired 1 year ago")
		}
	})

	t.Run("not yet expired", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(24 * time.Hour),
			IssuedAt:   now,
		}
		err := ValidateLicense(lic)
		if err != nil {
			t.Errorf("want nil for license not yet expired, got %v", err)
		}
	})

	t.Run("free license never expires in practice", func(t *testing.T) {
		lic := FreeLicense()
		err := ValidateLicense(lic)
		if err != nil {
			t.Errorf("free license should validate, got %v", err)
		}
	})
}

func TestLicense_GracePeriod(t *testing.T) {
	now := time.Now()

	t.Run("expired one second ago is invalid", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(-1 * time.Second),
			IssuedAt:   now.Add(-365 * 24 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("license expired 1 second ago should be invalid")
		}
	})

	t.Run("expired one minute ago is invalid", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(-1 * time.Minute),
			IssuedAt:   now.Add(-365 * 24 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("license expired 1 minute ago should be invalid")
		}
	})

	t.Run("expired 7 days ago is invalid", func(t *testing.T) {
		lic := &License{
			Tier:       TierEnterprise,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(-7 * 24 * time.Hour),
			IssuedAt:   now.Add(-365 * 24 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("license expired 7 days ago should be invalid")
		}
	})

	t.Run("expired 30 days ago is invalid", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(-30 * 24 * time.Hour),
			IssuedAt:   now.Add(-365 * 24 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err == nil {
			t.Error("license expired 30 days ago should be invalid")
		}
	})

	t.Run("valid license about to expire is still valid", func(t *testing.T) {
		lic := &License{
			Tier:       TierPro,
			CustomerID: "cust-123",
			ExpiresAt:  now.Add(5 * time.Minute),
			IssuedAt:   now.Add(-365 * 24 * time.Hour),
		}
		err := ValidateLicense(lic)
		if err != nil {
			t.Errorf("license expiring in 5 minutes should be valid, got %v", err)
		}
	})
}

func TestLicense_InvalidSignature(t *testing.T) {
	now := time.Now()
	futureExpiry := now.Add(365 * 24 * time.Hour).Unix()
	issuedAt := now.Unix()

	t.Run("tampered payload", func(t *testing.T) {
		key := createValidLicenseKey(TierPro, "customer-123", futureExpiry, issuedAt)
		parts := strings.SplitN(key, ".", 2)

		// Decode, tamper, re-encode payload
		payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[0])
		var payload licensePayload
		json.Unmarshal(payloadBytes, &payload)
		payload.CustomerID = "hacker"
		tamperedJSON, _ := json.Marshal(payload)
		tamperedB64 := base64.RawURLEncoding.EncodeToString(tamperedJSON)

		_, err := ParseLicense(tamperedB64 + "." + parts[1])
		if err == nil {
			t.Error("want error for tampered payload")
		}
	})

	t.Run("empty signature", func(t *testing.T) {
		payload := licensePayload{
			Tier:       TierPro,
			CustomerID: "cust",
			ExpiresAt:  futureExpiry,
			IssuedAt:   issuedAt,
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
		emptyB64 := base64.RawURLEncoding.EncodeToString([]byte{})

		_, err := ParseLicense(payloadB64 + "." + emptyB64)
		if err == nil {
			t.Error("want error for empty signature")
		}
	})

	t.Run("signature from different key", func(t *testing.T) {
		payload := licensePayload{
			Tier:       TierPro,
			CustomerID: "cust",
			ExpiresAt:  futureExpiry,
			IssuedAt:   issuedAt,
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

		wrongKey := []byte("wrong-signing-key")
		h := hmac.New(sha256.New, wrongKey)
		h.Write(payloadJSON)
		wrongSig := h.Sum(nil)
		wrongSigB64 := base64.RawURLEncoding.EncodeToString(wrongSig)

		_, err := ParseLicense(payloadB64 + "." + wrongSigB64)
		if err == nil {
			t.Error("want error for signature from different key")
		}
	})

	t.Run("truncated signature", func(t *testing.T) {
		payload := licensePayload{
			Tier:       TierPro,
			CustomerID: "cust",
			ExpiresAt:  futureExpiry,
			IssuedAt:   issuedAt,
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

		h := hmac.New(sha256.New, signingKey)
		h.Write(payloadJSON)
		fullSig := h.Sum(nil)
		// Truncate to half length
		truncatedSig := fullSig[:len(fullSig)/2]
		truncatedB64 := base64.RawURLEncoding.EncodeToString(truncatedSig)

		_, err := ParseLicense(payloadB64 + "." + truncatedB64)
		if err == nil {
			t.Error("want error for truncated signature")
		}
	})

	t.Run("signature with extra bytes", func(t *testing.T) {
		payload := licensePayload{
			Tier:       TierPro,
			CustomerID: "cust",
			ExpiresAt:  futureExpiry,
			IssuedAt:   issuedAt,
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

		h := hmac.New(sha256.New, signingKey)
		h.Write(payloadJSON)
		fullSig := h.Sum(nil)
		extendedSig := append(fullSig, 0xFF, 0xAA)
		extendedB64 := base64.RawURLEncoding.EncodeToString(extendedSig)

		_, err := ParseLicense(payloadB64 + "." + extendedB64)
		if err == nil {
			t.Error("want error for signature with extra bytes")
		}
	})

	t.Run("swapped payload and signature", func(t *testing.T) {
		key := createValidLicenseKey(TierPro, "customer-123", futureExpiry, issuedAt)
		parts := strings.SplitN(key, ".", 2)

		_, err := ParseLicense(parts[1] + "." + parts[0])
		if err == nil {
			t.Error("want error for swapped payload and signature")
		}
	})
}

func TestSetSigningKey(t *testing.T) {
	originalKey := signingKey
	defer SetSigningKey(originalKey)

	newKey := []byte("test-new-signing-key-12345")
	SetSigningKey(newKey)

	// Generate a key with the new signing key
	payload := licensePayload{
		Tier:       TierPro,
		CustomerID: "set-key-cust",
		ExpiresAt:  time.Now().Add(365 * 24 * time.Hour).Unix(),
		IssuedAt:   time.Now().Unix(),
	}
	payloadJSON, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)

	h := hmac.New(sha256.New, newKey)
	h.Write(payloadJSON)
	sig := h.Sum(nil)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	lic, err := ParseLicense(payloadB64 + "." + sigB64)
	if err != nil {
		t.Fatalf("ParseLicense() with new signing key: error = %v", err)
	}
	if lic.CustomerID != "set-key-cust" {
		t.Errorf("CustomerID = %v, want set-key-cust", lic.CustomerID)
	}
}
