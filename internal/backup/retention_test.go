package backup

import (
	"context"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

func TestValidateRetentionPolicy(t *testing.T) {
	t.Run("nil policy", func(t *testing.T) {
		err := ValidateRetentionPolicy(nil)
		if err == nil {
			t.Fatal("expected error for nil policy")
		}
		if err.Error() != "retention policy is nil" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("all zeros", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{})
		if err == nil {
			t.Fatal("expected error for empty policy")
		}
		if err.Error() != "at least one retention rule must be specified" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid daily only", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepDaily: 7})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid weekly only", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepWeekly: 4})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid monthly only", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepMonthly: 6})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid all fields", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{
			KeepLast:    5,
			KeepHourly:  24,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
			KeepYearly:  2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("negative keep_last", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepLast: -1, KeepDaily: 7})
		if err == nil {
			t.Fatal("expected error for negative keep_last")
		}
		if err.Error() != "keep_last cannot be negative" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("negative keep_hourly", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepHourly: -1, KeepDaily: 7})
		if err == nil {
			t.Fatal("expected error for negative keep_hourly")
		}
		if err.Error() != "keep_hourly cannot be negative" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("negative keep_daily", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepDaily: -1, KeepLast: 5})
		if err == nil {
			t.Fatal("expected error for negative keep_daily")
		}
		if err.Error() != "keep_daily cannot be negative" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("negative keep_weekly", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepWeekly: -1, KeepDaily: 7})
		if err == nil {
			t.Fatal("expected error for negative keep_weekly")
		}
		if err.Error() != "keep_weekly cannot be negative" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("negative keep_monthly", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepMonthly: -1, KeepDaily: 7})
		if err == nil {
			t.Fatal("expected error for negative keep_monthly")
		}
		if err.Error() != "keep_monthly cannot be negative" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("negative keep_yearly", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepYearly: -1, KeepDaily: 7})
		if err == nil {
			t.Fatal("expected error for negative keep_yearly")
		}
		if err.Error() != "keep_yearly cannot be negative" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("only keep_last set", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepLast: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("only keep_yearly set", func(t *testing.T) {
		err := ValidateRetentionPolicy(&models.RetentionPolicy{KeepYearly: 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestParseRetentionConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := map[string]int{
			"keep_last":    5,
			"keep_daily":   7,
			"keep_weekly":  4,
			"keep_monthly": 6,
		}

		policy, err := ParseRetentionConfig(cfg)
		if err != nil {
			t.Fatalf("ParseRetentionConfig() error = %v", err)
		}
		if policy.KeepLast != 5 {
			t.Errorf("KeepLast = %v, want 5", policy.KeepLast)
		}
		if policy.KeepDaily != 7 {
			t.Errorf("KeepDaily = %v, want 7", policy.KeepDaily)
		}
		if policy.KeepWeekly != 4 {
			t.Errorf("KeepWeekly = %v, want 4", policy.KeepWeekly)
		}
		if policy.KeepMonthly != 6 {
			t.Errorf("KeepMonthly = %v, want 6", policy.KeepMonthly)
		}
	})

	t.Run("all fields", func(t *testing.T) {
		cfg := map[string]int{
			"keep_last":    3,
			"keep_hourly":  12,
			"keep_daily":   7,
			"keep_weekly":  4,
			"keep_monthly": 6,
			"keep_yearly":  2,
		}

		policy, err := ParseRetentionConfig(cfg)
		if err != nil {
			t.Fatalf("ParseRetentionConfig() error = %v", err)
		}
		if policy.KeepHourly != 12 {
			t.Errorf("KeepHourly = %v, want 12", policy.KeepHourly)
		}
		if policy.KeepYearly != 2 {
			t.Errorf("KeepYearly = %v, want 2", policy.KeepYearly)
		}
	})

	t.Run("invalid config - all zeros", func(t *testing.T) {
		cfg := map[string]int{}

		_, err := ParseRetentionConfig(cfg)
		if err == nil {
			t.Fatal("expected error for empty config")
		}
	})

	t.Run("partial config", func(t *testing.T) {
		cfg := map[string]int{
			"keep_daily": 30,
		}

		policy, err := ParseRetentionConfig(cfg)
		if err != nil {
			t.Fatalf("ParseRetentionConfig() error = %v", err)
		}
		if policy.KeepDaily != 30 {
			t.Errorf("KeepDaily = %v, want 30", policy.KeepDaily)
		}
		if policy.KeepLast != 0 {
			t.Errorf("KeepLast = %v, want 0", policy.KeepLast)
		}
	})
}

func TestMergeRetentionPolicy(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		result := MergeRetentionPolicy(nil, nil)
		if result != nil {
			t.Errorf("expected nil, got: %v", result)
		}
	})

	t.Run("nil base", func(t *testing.T) {
		override := &models.RetentionPolicy{KeepDaily: 7}
		result := MergeRetentionPolicy(nil, override)
		if result != override {
			t.Error("expected override to be returned when base is nil")
		}
	})

	t.Run("nil override", func(t *testing.T) {
		base := &models.RetentionPolicy{KeepDaily: 7}
		result := MergeRetentionPolicy(base, nil)
		if result != base {
			t.Error("expected base to be returned when override is nil")
		}
	})

	t.Run("override replaces base values", func(t *testing.T) {
		base := &models.RetentionPolicy{
			KeepLast:    5,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
		}
		override := &models.RetentionPolicy{
			KeepDaily:  30,
			KeepYearly: 2,
		}

		result := MergeRetentionPolicy(base, override)
		if result.KeepLast != 5 {
			t.Errorf("KeepLast = %v, want 5 (from base)", result.KeepLast)
		}
		if result.KeepDaily != 30 {
			t.Errorf("KeepDaily = %v, want 30 (from override)", result.KeepDaily)
		}
		if result.KeepWeekly != 4 {
			t.Errorf("KeepWeekly = %v, want 4 (from base)", result.KeepWeekly)
		}
		if result.KeepMonthly != 6 {
			t.Errorf("KeepMonthly = %v, want 6 (from base)", result.KeepMonthly)
		}
		if result.KeepYearly != 2 {
			t.Errorf("KeepYearly = %v, want 2 (from override)", result.KeepYearly)
		}
	})

	t.Run("zero override values preserved from base", func(t *testing.T) {
		base := &models.RetentionPolicy{
			KeepLast:   5,
			KeepHourly: 24,
			KeepDaily:  7,
		}
		override := &models.RetentionPolicy{
			KeepLast: 10,
			// KeepHourly: 0 - should keep base value
		}

		result := MergeRetentionPolicy(base, override)
		if result.KeepLast != 10 {
			t.Errorf("KeepLast = %v, want 10", result.KeepLast)
		}
		if result.KeepHourly != 24 {
			t.Errorf("KeepHourly = %v, want 24 (preserved from base)", result.KeepHourly)
		}
		if result.KeepDaily != 7 {
			t.Errorf("KeepDaily = %v, want 7 (preserved from base)", result.KeepDaily)
		}
	})
}

func TestRetentionPolicyDescription(t *testing.T) {
	t.Run("nil policy", func(t *testing.T) {
		desc := RetentionPolicyDescription(nil)
		if desc != "No retention policy" {
			t.Errorf("description = %v, want 'No retention policy'", desc)
		}
	})

	t.Run("empty policy", func(t *testing.T) {
		desc := RetentionPolicyDescription(&models.RetentionPolicy{})
		if desc != "Empty retention policy" {
			t.Errorf("description = %v, want 'Empty retention policy'", desc)
		}
	})

	t.Run("daily only", func(t *testing.T) {
		desc := RetentionPolicyDescription(&models.RetentionPolicy{KeepDaily: 7})
		if desc != "Keep: 7 daily" {
			t.Errorf("description = %v, want 'Keep: 7 daily'", desc)
		}
	})

	t.Run("multiple fields", func(t *testing.T) {
		desc := RetentionPolicyDescription(&models.RetentionPolicy{
			KeepLast:    5,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
		})
		expected := "Keep: last 5, 7 daily, 4 weekly, 6 monthly"
		if desc != expected {
			t.Errorf("description = %v, want %v", desc, expected)
		}
	})

	t.Run("all fields", func(t *testing.T) {
		desc := RetentionPolicyDescription(&models.RetentionPolicy{
			KeepLast:    5,
			KeepHourly:  24,
			KeepDaily:   7,
			KeepWeekly:  4,
			KeepMonthly: 6,
			KeepYearly:  2,
		})
		expected := "Keep: last 5, 24 hourly, 7 daily, 4 weekly, 6 monthly, 2 yearly"
		if desc != expected {
			t.Errorf("description = %v, want %v", desc, expected)
		}
	})

	t.Run("yearly only", func(t *testing.T) {
		desc := RetentionPolicyDescription(&models.RetentionPolicy{KeepYearly: 3})
		if desc != "Keep: 3 yearly" {
			t.Errorf("description = %v, want 'Keep: 3 yearly'", desc)
		}
	})
}

func TestNewRetentionEnforcer(t *testing.T) {
	r := NewRestic(zerolog.Nop())
	enforcer := NewRetentionEnforcer(r, zerolog.Nop())
	if enforcer == nil {
		t.Fatal("NewRetentionEnforcer() returned nil")
	}
	if enforcer.restic != r {
		t.Error("restic reference mismatch")
	}
}

func TestRetentionEnforcer_ApplyPolicy_NilPolicy(t *testing.T) {
	r := NewRestic(zerolog.Nop())
	enforcer := NewRetentionEnforcer(r, zerolog.Nop())

	result, err := enforcer.ApplyPolicy(context.Background(), testResticConfig(), nil, false)
	if err != nil {
		t.Fatalf("ApplyPolicy() error = %v", err)
	}
	if result.Applied {
		t.Error("Applied should be false for nil policy")
	}
}

func TestRetentionEnforcer_ApplyPolicy_InvalidPolicy(t *testing.T) {
	r := NewRestic(zerolog.Nop())
	enforcer := NewRetentionEnforcer(r, zerolog.Nop())

	result, err := enforcer.ApplyPolicy(context.Background(), testResticConfig(), &models.RetentionPolicy{}, false)
	if err == nil {
		t.Fatal("expected error for empty policy")
	}
	if result.Applied {
		t.Error("Applied should be false for invalid policy")
	}
	if result.Error == "" {
		t.Error("Error should be set for invalid policy")
	}
}

func TestRetentionEnforcer_ApplyPolicy_ForgetOnly(t *testing.T) {
	forgetOutput := `[{"keep":[{"id":"s1","short_id":"s1","time":"2024-01-15T00:00:00Z","hostname":"h","paths":["/"]}],"remove":[{"id":"s2","short_id":"s2","time":"2024-01-10T00:00:00Z","hostname":"h","paths":["/"]}]}]`
	r, cleanup := newTestRestic(forgetOutput)
	defer cleanup()

	enforcer := NewRetentionEnforcer(r, zerolog.Nop())

	result, err := enforcer.ApplyPolicy(context.Background(), testResticConfig(), &models.RetentionPolicy{KeepDaily: 7}, false)
	if err != nil {
		t.Fatalf("ApplyPolicy() error = %v", err)
	}
	if !result.Applied {
		t.Error("Applied should be true")
	}
	if result.SnapshotsKept != 1 {
		t.Errorf("SnapshotsKept = %v, want 1", result.SnapshotsKept)
	}
	if result.SnapshotsRemoved != 1 {
		t.Errorf("SnapshotsRemoved = %v, want 1", result.SnapshotsRemoved)
	}
}

func TestRetentionEnforcer_ApplyPolicy_WithPrune(t *testing.T) {
	forgetOutput := `[{"keep":[{"id":"s1","short_id":"s1","time":"2024-01-15T00:00:00Z","hostname":"h","paths":["/"]}],"remove":[]}]`
	r, cleanup := newTestRestic(forgetOutput)
	defer cleanup()

	enforcer := NewRetentionEnforcer(r, zerolog.Nop())

	result, err := enforcer.ApplyPolicy(context.Background(), testResticConfig(), &models.RetentionPolicy{KeepLast: 5}, true)
	if err != nil {
		t.Fatalf("ApplyPolicy() error = %v", err)
	}
	if !result.Applied {
		t.Error("Applied should be true")
	}
}

func TestRetentionEnforcer_ApplyPolicy_Error(t *testing.T) {
	r, cleanup := newTestResticError("forget command failed")
	defer cleanup()

	enforcer := NewRetentionEnforcer(r, zerolog.Nop())

	result, err := enforcer.ApplyPolicy(context.Background(), testResticConfig(), &models.RetentionPolicy{KeepDaily: 7}, false)
	if err == nil {
		t.Fatal("expected error")
	}
	if result.Applied {
		t.Error("Applied should be false on error")
	}
	if result.Error == "" {
		t.Error("Error should be set")
	}
}
