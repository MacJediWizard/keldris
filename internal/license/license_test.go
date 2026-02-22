package license

import (
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
}
