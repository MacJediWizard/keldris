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
