package license

// Feature represents a gated feature that requires a specific license tier.
type Feature string

const (
	// FeatureOIDC enables OIDC authentication (Pro+).
	FeatureOIDC Feature = "oidc"
	// FeatureAuditLogs enables audit logging (Pro+).
	FeatureAuditLogs Feature = "audit_logs"
	// FeatureMultiOrg enables multiple organizations (Enterprise).
	FeatureMultiOrg Feature = "multi_org"
	// FeatureSLATracking enables SLA tracking (Enterprise).
	FeatureSLATracking Feature = "sla_tracking"
	// FeatureWhiteLabel enables white-label branding (Enterprise).
	FeatureWhiteLabel Feature = "white_label"
	// FeatureAirGap enables air-gapped deployment (Enterprise).
	FeatureAirGap Feature = "air_gap"
)

// featureAccess maps each license tier to the features it unlocks.
var featureAccess = map[LicenseTier][]Feature{
	TierFree: {},
	TierPro: {
		FeatureOIDC,
		FeatureAuditLogs,
	},
	TierEnterprise: {
		FeatureOIDC,
		FeatureAuditLogs,
		FeatureMultiOrg,
		FeatureSLATracking,
		FeatureWhiteLabel,
		FeatureAirGap,
	},
}

// HasFeature returns true if the given tier has access to the specified feature.
func HasFeature(tier LicenseTier, feature Feature) bool {
	features, ok := featureAccess[tier]
	if !ok {
		return false
	}
	for _, f := range features {
		if f == feature {
			return true
		}
	}
	return false
}

// FeaturesForTier returns all features available for the given tier.
func FeaturesForTier(tier LicenseTier) []Feature {
	features, ok := featureAccess[tier]
	if !ok {
		return nil
	}
	result := make([]Feature, len(features))
	copy(result, features)
	return result
}
