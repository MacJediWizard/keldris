package license

// Feature represents a gated feature that requires a specific license tier.
type Feature string

const (
	// FeatureOIDC enables OIDC authentication (Pro+).
	FeatureOIDC Feature = "oidc"
	// FeatureAuditLogs enables audit logging (Pro+).
	FeatureAuditLogs Feature = "audit_logs"
	// FeatureNotificationSlack enables Slack notifications (Pro+).
	FeatureNotificationSlack Feature = "notification_slack"
	// FeatureNotificationTeams enables Microsoft Teams notifications (Pro+).
	FeatureNotificationTeams Feature = "notification_teams"
	// FeatureNotificationPagerDuty enables PagerDuty notifications (Pro+).
	FeatureNotificationPagerDuty Feature = "notification_pagerduty"
	// FeatureNotificationDiscord enables Discord notifications (Pro+).
	FeatureNotificationDiscord Feature = "notification_discord"
	// FeatureStorageS3 enables S3-compatible storage backends (Pro+).
	FeatureStorageS3 Feature = "storage_s3"
	// FeatureStorageB2 enables Backblaze B2 storage backend (Pro+).
	FeatureStorageB2 Feature = "storage_b2"
	// FeatureStorageSFTP enables SFTP storage backend (Pro+).
	FeatureStorageSFTP Feature = "storage_sftp"
	// FeatureDockerBackup enables Docker container backups (Pro+).
	FeatureDockerBackup Feature = "docker_backup"
	// FeatureMultiRepo enables multiple backup repositories (Pro+).
	FeatureMultiRepo Feature = "multi_repo"
	// FeatureAPIAccess enables programmatic API access (Pro+).
	FeatureAPIAccess Feature = "api_access"
	// FeatureMultiOrg enables multiple organizations (Enterprise).
	FeatureMultiOrg Feature = "multi_org"
	// FeatureSLATracking enables SLA tracking (Enterprise).
	FeatureSLATracking Feature = "sla_tracking"
	// FeatureWhiteLabel enables white-label branding (Enterprise).
	FeatureWhiteLabel Feature = "white_label"
	// FeatureAirGap enables air-gapped deployment (Enterprise).
	FeatureAirGap Feature = "air_gap"
	// FeatureDRRunbooks enables disaster recovery runbooks (Enterprise).
	FeatureDRRunbooks Feature = "dr_runbooks"
	// FeatureDRTests enables disaster recovery testing (Enterprise).
	FeatureDRTests Feature = "dr_tests"
)

// featureAccess maps each license tier to the features it unlocks.
var featureAccess = map[LicenseTier][]Feature{
	TierFree: {},
	TierPro: {
		FeatureOIDC,
		FeatureAuditLogs,
		FeatureNotificationSlack,
		FeatureNotificationTeams,
		FeatureNotificationPagerDuty,
		FeatureNotificationDiscord,
		FeatureStorageS3,
		FeatureStorageB2,
		FeatureStorageSFTP,
		FeatureDockerBackup,
		FeatureMultiRepo,
		FeatureAPIAccess,
	},
	TierEnterprise: {
		FeatureOIDC,
		FeatureAuditLogs,
		FeatureNotificationSlack,
		FeatureNotificationTeams,
		FeatureNotificationPagerDuty,
		FeatureNotificationDiscord,
		FeatureStorageS3,
		FeatureStorageB2,
		FeatureStorageSFTP,
		FeatureDockerBackup,
		FeatureMultiRepo,
		FeatureAPIAccess,
		FeatureMultiOrg,
		FeatureSLATracking,
		FeatureWhiteLabel,
		FeatureAirGap,
		FeatureDRRunbooks,
		FeatureDRTests,
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
