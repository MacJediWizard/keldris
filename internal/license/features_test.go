package license

import (
	"testing"
)

func TestHasFeature(t *testing.T) {
	tests := []struct {
		name    string
		tier    LicenseTier
		feature Feature
		has     bool
	}{
		// Free tier - no features
		{"free tier has no OIDC", TierFree, FeatureOIDC, false},
		{"free tier has no audit logs", TierFree, FeatureAuditLogs, false},
		{"free tier has no multi-org", TierFree, FeatureMultiOrg, false},

		// Pro tier - OIDC and audit logs
		{"pro tier has OIDC", TierPro, FeatureOIDC, true},
		{"pro tier has audit logs", TierPro, FeatureAuditLogs, true},
		{"pro tier has no multi-org", TierPro, FeatureMultiOrg, false},
		{"pro tier has no SLA tracking", TierPro, FeatureSLATracking, false},
		{"pro tier has no white label", TierPro, FeatureWhiteLabel, false},
		{"pro tier has no air gap", TierPro, FeatureAirGap, false},

		// Enterprise tier - all features
		{"enterprise tier has OIDC", TierEnterprise, FeatureOIDC, true},
		{"enterprise tier has audit logs", TierEnterprise, FeatureAuditLogs, true},
		{"enterprise tier has multi-org", TierEnterprise, FeatureMultiOrg, true},
		{"enterprise tier has SLA tracking", TierEnterprise, FeatureSLATracking, true},
		{"enterprise tier has white label", TierEnterprise, FeatureWhiteLabel, true},
		{"enterprise tier has air gap", TierEnterprise, FeatureAirGap, true},

		// Invalid tier
		{"invalid tier has no OIDC", LicenseTier("invalid"), FeatureOIDC, false},
		{"empty tier has no features", LicenseTier(""), FeatureOIDC, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasFeature(tt.tier, tt.feature); got != tt.has {
				t.Errorf("HasFeature(%v, %v) = %v, want %v", tt.tier, tt.feature, got, tt.has)
			}
		})
	}
}

func TestFeaturesForTier(t *testing.T) {
	t.Run("free tier features", func(t *testing.T) {
		features := FeaturesForTier(TierFree)
		if len(features) != 0 {
			t.Errorf("FeaturesForTier(TierFree) returned %d features, want 0", len(features))
		}
	})

	t.Run("pro tier features", func(t *testing.T) {
		features := FeaturesForTier(TierPro)
		if len(features) != 2 {
			t.Errorf("FeaturesForTier(TierPro) returned %d features, want 2", len(features))
		}
	})

	t.Run("enterprise tier features", func(t *testing.T) {
		features := FeaturesForTier(TierEnterprise)
		if len(features) != 6 {
			t.Errorf("FeaturesForTier(TierEnterprise) returned %d features, want 6", len(features))
		}
	})

	t.Run("invalid tier features", func(t *testing.T) {
		features := FeaturesForTier(LicenseTier("invalid"))
		if features != nil {
			t.Errorf("FeaturesForTier(invalid) = %v, want nil", features)
		}
	})
}

func TestFeatures_HasFeature_AllTiers(t *testing.T) {
	allFeatures := []Feature{
		FeatureOIDC,
		FeatureAuditLogs,
		FeatureMultiOrg,
		FeatureSLATracking,
		FeatureWhiteLabel,
		FeatureAirGap,
	}

	t.Run("free tier has no features at all", func(t *testing.T) {
		for _, feature := range allFeatures {
			if HasFeature(TierFree, feature) {
				t.Errorf("free tier should not have feature %s", feature)
			}
		}
	})

	t.Run("pro tier has exactly OIDC and audit logs", func(t *testing.T) {
		proFeatures := map[Feature]bool{
			FeatureOIDC:     true,
			FeatureAuditLogs: true,
		}
		for _, feature := range allFeatures {
			got := HasFeature(TierPro, feature)
			want := proFeatures[feature]
			if got != want {
				t.Errorf("HasFeature(TierPro, %s) = %v, want %v", feature, got, want)
			}
		}
	})

	t.Run("enterprise tier has all features", func(t *testing.T) {
		for _, feature := range allFeatures {
			if !HasFeature(TierEnterprise, feature) {
				t.Errorf("enterprise tier should have feature %s", feature)
			}
		}
	})

	t.Run("unknown feature returns false for all tiers", func(t *testing.T) {
		unknownFeature := Feature("nonexistent")
		for _, tier := range ValidTiers() {
			if HasFeature(tier, unknownFeature) {
				t.Errorf("HasFeature(%s, %s) = true, want false", tier, unknownFeature)
			}
		}
	})
}

func TestFeatures_FreeTierLimits(t *testing.T) {
	features := FeaturesForTier(TierFree)
	if len(features) != 0 {
		t.Errorf("free tier should have 0 features, got %d", len(features))
	}

	// Verify free tier cannot access any gated feature
	if HasFeature(TierFree, FeatureOIDC) {
		t.Error("free tier should not have OIDC")
	}
	if HasFeature(TierFree, FeatureAuditLogs) {
		t.Error("free tier should not have audit logs")
	}
	if HasFeature(TierFree, FeatureMultiOrg) {
		t.Error("free tier should not have multi-org")
	}
	if HasFeature(TierFree, FeatureSLATracking) {
		t.Error("free tier should not have SLA tracking")
	}
	if HasFeature(TierFree, FeatureWhiteLabel) {
		t.Error("free tier should not have white label")
	}
	if HasFeature(TierFree, FeatureAirGap) {
		t.Error("free tier should not have air gap")
	}
}

func TestFeatures_ProTierLimits(t *testing.T) {
	features := FeaturesForTier(TierPro)
	if len(features) != 2 {
		t.Fatalf("pro tier should have 2 features, got %d", len(features))
	}

	// Verify the exact features
	featureSet := make(map[Feature]bool)
	for _, f := range features {
		featureSet[f] = true
	}

	if !featureSet[FeatureOIDC] {
		t.Error("pro tier features should include OIDC")
	}
	if !featureSet[FeatureAuditLogs] {
		t.Error("pro tier features should include audit logs")
	}

	// Verify enterprise-only features are not included
	if featureSet[FeatureMultiOrg] {
		t.Error("pro tier features should not include multi-org")
	}
	if featureSet[FeatureSLATracking] {
		t.Error("pro tier features should not include SLA tracking")
	}
	if featureSet[FeatureWhiteLabel] {
		t.Error("pro tier features should not include white label")
	}
	if featureSet[FeatureAirGap] {
		t.Error("pro tier features should not include air gap")
	}
}

func TestFeatures_EnterpriseTierLimits(t *testing.T) {
	features := FeaturesForTier(TierEnterprise)
	if len(features) != 6 {
		t.Fatalf("enterprise tier should have 6 features, got %d", len(features))
	}

	// Verify all features are present
	featureSet := make(map[Feature]bool)
	for _, f := range features {
		featureSet[f] = true
	}

	expectedFeatures := []Feature{
		FeatureOIDC,
		FeatureAuditLogs,
		FeatureMultiOrg,
		FeatureSLATracking,
		FeatureWhiteLabel,
		FeatureAirGap,
	}

	for _, expected := range expectedFeatures {
		if !featureSet[expected] {
			t.Errorf("enterprise tier should include feature %s", expected)
		}
	}

	// Verify FeaturesForTier returns a copy (not a reference to the internal slice)
	features[0] = Feature("tampered")
	originalFeatures := FeaturesForTier(TierEnterprise)
	if originalFeatures[0] == Feature("tampered") {
		t.Error("FeaturesForTier should return a copy, not a reference")
	}
}
