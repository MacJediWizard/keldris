package license

import (
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
