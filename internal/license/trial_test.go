package license

import (
	"testing"
	"time"
)

func TestTrial_Start(t *testing.T) {
	t.Run("start pro trial", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust-001")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}
		if trial.License.Tier != TierPro {
			t.Errorf("Tier = %v, want %v", trial.License.Tier, TierPro)
		}
		if trial.License.CustomerID != "trial-cust-001" {
			t.Errorf("CustomerID = %v, want trial-cust-001", trial.License.CustomerID)
		}
		if trial.Duration != DefaultTrialDuration {
			t.Errorf("Duration = %v, want %v", trial.Duration, DefaultTrialDuration)
		}
	})

	t.Run("start enterprise trial", func(t *testing.T) {
		trial, err := StartTrial(TierEnterprise, "trial-cust-002")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}
		if trial.License.Tier != TierEnterprise {
			t.Errorf("Tier = %v, want %v", trial.License.Tier, TierEnterprise)
		}
		if !IsUnlimited(trial.License.Limits.MaxAgents) {
			t.Error("enterprise trial should have unlimited agents")
		}
	})

	t.Run("cannot start free tier trial", func(t *testing.T) {
		_, err := StartTrial(TierFree, "trial-cust")
		if err == nil {
			t.Error("StartTrial() error = nil, want error for free tier")
		}
	})

	t.Run("cannot start trial with empty customer ID", func(t *testing.T) {
		_, err := StartTrial(TierPro, "")
		if err == nil {
			t.Error("StartTrial() error = nil, want error for empty customer ID")
		}
	})

	t.Run("cannot start trial with invalid tier", func(t *testing.T) {
		_, err := StartTrial(LicenseTier("invalid"), "trial-cust")
		if err == nil {
			t.Error("StartTrial() error = nil, want error for invalid tier")
		}
	})

	t.Run("trial sets correct expiry", func(t *testing.T) {
		before := time.Now()
		trial, err := StartTrial(TierPro, "trial-cust")
		after := time.Now()
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}

		expectedMin := before.Add(DefaultTrialDuration)
		expectedMax := after.Add(DefaultTrialDuration)

		if trial.License.ExpiresAt.Before(expectedMin) || trial.License.ExpiresAt.After(expectedMax) {
			t.Errorf("ExpiresAt = %v, want between %v and %v", trial.License.ExpiresAt, expectedMin, expectedMax)
		}
	})

	t.Run("trial has correct limits for tier", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}

		expectedLimits := GetLimits(TierPro)
		if trial.License.Limits != expectedLimits {
			t.Errorf("Limits = %+v, want %+v", trial.License.Limits, expectedLimits)
		}
	})
}

func TestTrial_IsActive(t *testing.T) {
	t.Run("newly created trial is active", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}
		if !trial.IsActive() {
			t.Error("newly created trial should be active")
		}
	})

	t.Run("trial with future expiry is active", func(t *testing.T) {
		trial := &Trial{
			License: &License{
				Tier:       TierPro,
				CustomerID: "cust",
				ExpiresAt:  time.Now().Add(24 * time.Hour),
				IssuedAt:   time.Now(),
			},
			StartedAt: time.Now(),
			Duration:  DefaultTrialDuration,
		}
		if !trial.IsActive() {
			t.Error("trial with future expiry should be active")
		}
	})

	t.Run("trial with past expiry is not active", func(t *testing.T) {
		trial := &Trial{
			License: &License{
				Tier:       TierPro,
				CustomerID: "cust",
				ExpiresAt:  time.Now().Add(-1 * time.Hour),
				IssuedAt:   time.Now().Add(-15 * 24 * time.Hour),
			},
			StartedAt: time.Now().Add(-15 * 24 * time.Hour),
			Duration:  DefaultTrialDuration,
		}
		if trial.IsActive() {
			t.Error("trial with past expiry should not be active")
		}
	})

	t.Run("nil trial is not active", func(t *testing.T) {
		var trial *Trial
		if trial.IsActive() {
			t.Error("nil trial should not be active")
		}
	})

	t.Run("trial with nil license is not active", func(t *testing.T) {
		trial := &Trial{License: nil}
		if trial.IsActive() {
			t.Error("trial with nil license should not be active")
		}
	})
}

func TestTrial_Expired(t *testing.T) {
	t.Run("newly created trial is not expired", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}
		if trial.IsExpired() {
			t.Error("newly created trial should not be expired")
		}
	})

	t.Run("trial past expiry is expired", func(t *testing.T) {
		trial := &Trial{
			License: &License{
				Tier:       TierPro,
				CustomerID: "cust",
				ExpiresAt:  time.Now().Add(-1 * time.Hour),
				IssuedAt:   time.Now().Add(-15 * 24 * time.Hour),
			},
			StartedAt: time.Now().Add(-15 * 24 * time.Hour),
			Duration:  DefaultTrialDuration,
		}
		if !trial.IsExpired() {
			t.Error("trial past expiry should be expired")
		}
	})

	t.Run("nil trial is expired", func(t *testing.T) {
		var trial *Trial
		if !trial.IsExpired() {
			t.Error("nil trial should be expired")
		}
	})

	t.Run("trial expired 30 days ago", func(t *testing.T) {
		trial := &Trial{
			License: &License{
				Tier:       TierEnterprise,
				CustomerID: "cust",
				ExpiresAt:  time.Now().Add(-30 * 24 * time.Hour),
				IssuedAt:   time.Now().Add(-44 * 24 * time.Hour),
			},
			StartedAt: time.Now().Add(-44 * 24 * time.Hour),
			Duration:  DefaultTrialDuration,
		}
		if !trial.IsExpired() {
			t.Error("trial expired 30 days ago should be expired")
		}
	})

	t.Run("IsExpired is inverse of IsActive", func(t *testing.T) {
		trial, _ := StartTrial(TierPro, "cust")
		if trial.IsActive() == trial.IsExpired() {
			t.Error("IsActive and IsExpired should be inverses")
		}
	})
}

func TestTrial_Convert(t *testing.T) {
	t.Run("convert pro trial to pro license", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}

		lic, err := trial.Convert(TierPro)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if lic.Tier != TierPro {
			t.Errorf("Tier = %v, want %v", lic.Tier, TierPro)
		}
		if lic.CustomerID != "trial-cust" {
			t.Errorf("CustomerID = %v, want trial-cust", lic.CustomerID)
		}
		// Converted license should expire in ~1 year
		if time.Until(lic.ExpiresAt) < 364*24*time.Hour {
			t.Error("converted license should expire in approximately 1 year")
		}
	})

	t.Run("convert pro trial to enterprise license", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}

		lic, err := trial.Convert(TierEnterprise)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if lic.Tier != TierEnterprise {
			t.Errorf("Tier = %v, want %v", lic.Tier, TierEnterprise)
		}
		if !IsUnlimited(lic.Limits.MaxAgents) {
			t.Error("enterprise license should have unlimited agents")
		}
	})

	t.Run("cannot convert to free tier", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}

		_, err = trial.Convert(TierFree)
		if err == nil {
			t.Error("Convert() error = nil, want error for free tier")
		}
	})

	t.Run("cannot convert with invalid tier", func(t *testing.T) {
		trial, err := StartTrial(TierPro, "trial-cust")
		if err != nil {
			t.Fatalf("StartTrial() error = %v", err)
		}

		_, err = trial.Convert(LicenseTier("invalid"))
		if err == nil {
			t.Error("Convert() error = nil, want error for invalid tier")
		}
	})

	t.Run("cannot convert nil trial", func(t *testing.T) {
		var trial *Trial
		_, err := trial.Convert(TierPro)
		if err == nil {
			t.Error("Convert() error = nil, want error for nil trial")
		}
	})

	t.Run("cannot convert trial with nil license", func(t *testing.T) {
		trial := &Trial{License: nil}
		_, err := trial.Convert(TierPro)
		if err == nil {
			t.Error("Convert() error = nil, want error for nil license")
		}
	})

	t.Run("converted license preserves customer ID", func(t *testing.T) {
		trial, _ := StartTrial(TierPro, "important-customer-42")
		lic, err := trial.Convert(TierEnterprise)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if lic.CustomerID != "important-customer-42" {
			t.Errorf("CustomerID = %v, want important-customer-42", lic.CustomerID)
		}
	})

	t.Run("converted license has correct limits", func(t *testing.T) {
		trial, _ := StartTrial(TierPro, "cust")
		lic, err := trial.Convert(TierEnterprise)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		expectedLimits := GetLimits(TierEnterprise)
		if lic.Limits != expectedLimits {
			t.Errorf("Limits = %+v, want %+v", lic.Limits, expectedLimits)
		}
	})

	t.Run("converted license validates successfully", func(t *testing.T) {
		trial, _ := StartTrial(TierPro, "cust")
		lic, err := trial.Convert(TierPro)
		if err != nil {
			t.Fatalf("Convert() error = %v", err)
		}
		if err := ValidateLicense(lic); err != nil {
			t.Errorf("ValidateLicense() on converted license: error = %v", err)
		}
	})
}

func TestDefaultTrialDuration(t *testing.T) {
	expected := 14 * 24 * time.Hour
	if DefaultTrialDuration != expected {
		t.Errorf("DefaultTrialDuration = %v, want %v", DefaultTrialDuration, expected)
	}
}
