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
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

func TestKeyPairGeneration(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if len(kp.PublicKey) == 0 {
		t.Error("PublicKey is empty")
	}
	if len(kp.PrivateKey) == 0 {
		t.Error("PrivateKey is empty")
	}
}

func TestKeyPairBase64Encoding(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// Test public key encoding/decoding
	publicBase64 := kp.PublicKeyToBase64()
	decodedPublic, err := PublicKeyFromBase64(publicBase64)
	if err != nil {
		t.Fatalf("PublicKeyFromBase64() error = %v", err)
	}
	if string(decodedPublic) != string(kp.PublicKey) {
		t.Error("Decoded public key does not match original")
	}

	// Test private key encoding/decoding
	privateBase64 := kp.PrivateKeyToBase64()
	decodedPrivate, err := PrivateKeyFromBase64(privateBase64)
	if err != nil {
		t.Fatalf("PrivateKeyFromBase64() error = %v", err)
	}
	if string(decodedPrivate) != string(kp.PrivateKey) {
		t.Error("Decoded private key does not match original")
	}
}

func TestLicenseGenerationAndValidation(t *testing.T) {
	// Generate a key pair
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// Create generator and validator
	generator, err := NewGenerator(kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	validator, err := NewValidator(kp.PublicKey)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	// Create a valid license payload
	payload := &models.LicensePayload{
		ID:         uuid.New(),
		CustomerID: "customer-123",
		Tier:       models.LicenseTierTeam,
		Limits:     models.DefaultTeamLimits(),
		Features:   models.DefaultTeamFeatures(),
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().AddDate(1, 0, 0), // 1 year from now
	}

	// Generate license key
	licenseKey, err := generator.Generate(payload)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if licenseKey == "" {
		t.Error("Generated license key is empty")
	}

	// Validate the license key
	result, err := validator.Validate(licenseKey)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !result.Valid {
		t.Errorf("License should be valid, got status: %s", result.Status)
	}

	if result.Status != models.LicenseStatusActive {
		t.Errorf("Expected status Active, got %s", result.Status)
	}

	if result.Tier != models.LicenseTierTeam {
		t.Errorf("Expected tier Team, got %s", result.Tier)
	}
}

func TestExpiredLicenseWithGracePeriod(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	generator, err := NewGenerator(kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	validator, err := NewValidator(kp.PublicKey)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	// Create an expired license (expired 10 days ago, within grace period)
	payload := &models.LicensePayload{
		ID:         uuid.New(),
		CustomerID: "customer-123",
		Tier:       models.LicenseTierTeam,
		Limits:     models.DefaultTeamLimits(),
		Features:   models.DefaultTeamFeatures(),
		IssuedAt:   time.Now().AddDate(-1, 0, 0),
		ExpiresAt:  time.Now().AddDate(0, 0, -10), // 10 days ago
	}

	licenseKey, err := generator.Generate(payload)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result, err := validator.Validate(licenseKey)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if !result.Valid {
		t.Error("License should still be valid within grace period")
	}

	if result.Status != models.LicenseStatusGracePeriod {
		t.Errorf("Expected status GracePeriod, got %s", result.Status)
	}
}

func TestFullyExpiredLicense(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	generator, err := NewGenerator(kp.PrivateKey)
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	validator, err := NewValidator(kp.PublicKey)
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}

	// Create a fully expired license (expired 60 days ago, past grace period)
	payload := &models.LicensePayload{
		ID:         uuid.New(),
		CustomerID: "customer-123",
		Tier:       models.LicenseTierTeam,
		Limits:     models.DefaultTeamLimits(),
		Features:   models.DefaultTeamFeatures(),
		IssuedAt:   time.Now().AddDate(-1, 0, 0),
		ExpiresAt:  time.Now().AddDate(0, 0, -60), // 60 days ago
	}

	licenseKey, err := generator.Generate(payload)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	result, err := validator.Validate(licenseKey)
	if err != ErrLicenseExpired {
		t.Errorf("Expected ErrLicenseExpired, got %v", err)
	}

	if result.Valid {
		t.Error("License should not be valid past grace period")
	}

	if result.Status != models.LicenseStatusExpired {
		t.Errorf("Expected status Expired, got %s", result.Status)
	}
}

func TestInvalidSignature(t *testing.T) {
	// Generate two different key pairs
	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()

	// Create generator with first key pair
	generator, _ := NewGenerator(kp1.PrivateKey)

	// Create validator with second key pair (different public key)
	validator, _ := NewValidator(kp2.PublicKey)

	payload := &models.LicensePayload{
		ID:         uuid.New(),
		CustomerID: "customer-123",
		Tier:       models.LicenseTierTeam,
		Limits:     models.DefaultTeamLimits(),
		Features:   models.DefaultTeamFeatures(),
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().AddDate(1, 0, 0),
	}

	licenseKey, _ := generator.Generate(payload)

	// Validation should fail due to signature mismatch
	_, err := validator.Validate(licenseKey)
	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature, got %v", err)
	}
}

func TestInvalidLicenseKeyFormat(t *testing.T) {
	kp, _ := GenerateKeyPair()
	validator, _ := NewValidator(kp.PublicKey)

	testCases := []struct {
		name       string
		licenseKey string
	}{
		{"empty", ""},
		{"no prefix", "ABC-1.payload.signature"},
		{"wrong prefix", "WRONG-1.payload.signature"},
		{"missing parts", "KLDRS-1.payload"},
		{"too many parts", "KLDRS-1.a.b.c.d"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := validator.Validate(tc.licenseKey)
			if err == nil {
				t.Error("Expected error for invalid license key format")
			}
		})
	}
}

func TestManagerWithLicense(t *testing.T) {
	kp, _ := GenerateKeyPair()
	generator, _ := NewGenerator(kp.PrivateKey)
	manager, err := NewManager(kp.PublicKey)
	if err != nil {
		t.Fatalf("NewManager() error = %v", err)
	}

	// Generate a license
	payload := &models.LicensePayload{
		ID:         uuid.New(),
		CustomerID: "customer-123",
		Tier:       models.LicenseTierEnterprise,
		Limits:     models.DefaultEnterpriseLimits(),
		Features:   models.DefaultEnterpriseFeatures(),
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().AddDate(1, 0, 0),
	}

	licenseKey, _ := generator.Generate(payload)

	// Create a license model
	license := &models.License{
		ID:         payload.ID,
		LicenseKey: licenseKey,
		CustomerID: payload.CustomerID,
		Tier:       payload.Tier,
		Limits:     payload.Limits,
		Features:   payload.Features,
		ExpiresAt:  payload.ExpiresAt,
	}

	// Set license on manager
	result, err := manager.SetLicense(license)
	if err != nil {
		t.Fatalf("SetLicense() error = %v", err)
	}

	if !result.Valid {
		t.Error("License should be valid")
	}

	// Check manager methods
	if manager.GetTier() != models.LicenseTierEnterprise {
		t.Errorf("Expected Enterprise tier, got %s", manager.GetTier())
	}

	limits := manager.GetLimits()
	if limits.MaxAgents != -1 {
		t.Errorf("Expected unlimited agents (-1), got %d", limits.MaxAgents)
	}

	features := manager.GetFeatures()
	if !features.GeoReplication {
		t.Error("Expected geo replication to be enabled for Enterprise tier")
	}
}
