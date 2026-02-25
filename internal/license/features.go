// Package license provides feature gating based on license tiers.
package license

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Tier is a type alias for LicenseTier, allowing interchangeable use
// in the feature gating system. The canonical type is LicenseTier
// defined in license.go.
type Tier = LicenseTier

// Feature represents a gated feature name.
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
	// FeatureStorageDropbox enables Dropbox storage backend (Pro+).
	FeatureStorageDropbox Feature = "storage_dropbox"
	// FeatureStorageRest enables REST server storage backend (Pro+).
	FeatureStorageRest Feature = "storage_rest"
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
	// FeatureCustomReports enables custom report generation (Pro+).
	FeatureCustomReports Feature = "custom_reports"
	// FeatureSSOSync enables SSO directory sync (Enterprise).
	FeatureSSOSync Feature = "sso_sync"
	// FeatureRBAC enables role-based access control (Enterprise).
	FeatureRBAC Feature = "rbac"
	// FeatureGeoReplication enables cross-region replication (Enterprise).
	FeatureGeoReplication Feature = "geo_replication"
	// FeatureRansomwareProtect enables ransomware protection (Enterprise).
	FeatureRansomwareProtect Feature = "ransomware_protection"
	// FeatureLegalHolds enables legal hold capabilities (Enterprise).
	FeatureLegalHolds Feature = "legal_holds"
	// FeatureCustomRetention enables custom retention policies (Pro+).
	FeatureCustomRetention Feature = "custom_retention"
	// FeaturePrioritySupport enables priority support access (Enterprise).
	FeaturePrioritySupport Feature = "priority_support"
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
		FeatureStorageDropbox,
		FeatureStorageRest,
		FeatureDockerBackup,
		FeatureMultiRepo,
		FeatureAPIAccess,
		FeatureCustomReports,
		FeatureCustomRetention,
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
		FeatureStorageDropbox,
		FeatureStorageRest,
		FeatureDockerBackup,
		FeatureMultiRepo,
		FeatureAPIAccess,
		FeatureCustomReports,
		FeatureCustomRetention,
		FeatureMultiOrg,
		FeatureSLATracking,
		FeatureWhiteLabel,
		FeatureAirGap,
		FeatureDRRunbooks,
		FeatureDRTests,
		FeatureSSOSync,
		FeatureRBAC,
		FeatureGeoReplication,
		FeatureRansomwareProtect,
		FeatureLegalHolds,
		FeaturePrioritySupport,
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

// featureTierMap defines which tier is required for each feature.
var featureTierMap = map[Feature]Tier{
	// Pro tier features
	FeatureOIDC:                 TierPro,
	FeatureAuditLogs:            TierPro,
	FeatureNotificationSlack:    TierPro,
	FeatureNotificationTeams:    TierPro,
	FeatureNotificationPagerDuty: TierPro,
	FeatureNotificationDiscord:  TierPro,
	FeatureStorageS3:            TierPro,
	FeatureStorageB2:            TierPro,
	FeatureStorageSFTP:          TierPro,
	FeatureStorageDropbox:       TierPro,
	FeatureStorageRest:          TierPro,
	FeatureDockerBackup:         TierPro,
	FeatureMultiRepo:            TierPro,
	FeatureAPIAccess:            TierPro,
	FeatureCustomReports:        TierPro,
	FeatureCustomRetention:      TierPro,
	// Enterprise tier features
	FeatureMultiOrg:          TierEnterprise,
	FeatureSLATracking:       TierEnterprise,
	FeatureWhiteLabel:        TierEnterprise,
	FeatureAirGap:            TierEnterprise,
	FeatureDRRunbooks:        TierEnterprise,
	FeatureDRTests:           TierEnterprise,
	FeatureSSOSync:           TierEnterprise,
	FeatureRBAC:              TierEnterprise,
	FeatureGeoReplication:    TierEnterprise,
	FeatureRansomwareProtect: TierEnterprise,
	FeatureLegalHolds:        TierEnterprise,
	FeaturePrioritySupport:   TierEnterprise,
}

// AllFeatures returns all defined features.
func AllFeatures() []Feature {
	return []Feature{
		// Pro tier features
		FeatureOIDC,
		FeatureAuditLogs,
		FeatureNotificationSlack,
		FeatureNotificationTeams,
		FeatureNotificationPagerDuty,
		FeatureNotificationDiscord,
		FeatureStorageS3,
		FeatureStorageB2,
		FeatureStorageSFTP,
		FeatureStorageDropbox,
		FeatureStorageRest,
		FeatureDockerBackup,
		FeatureMultiRepo,
		FeatureAPIAccess,
		FeatureCustomReports,
		FeatureCustomRetention,
		// Enterprise tier features
		FeatureMultiOrg,
		FeatureSLATracking,
		FeatureWhiteLabel,
		FeatureAirGap,
		FeatureDRRunbooks,
		FeatureDRTests,
		FeatureSSOSync,
		FeatureRBAC,
		FeatureGeoReplication,
		FeatureRansomwareProtect,
		FeatureLegalHolds,
		FeaturePrioritySupport,
	}
}

// TierOrder returns the tier hierarchy for comparison.
// Higher values mean more access.
func TierOrder(t Tier) int {
	switch t {
	case TierFree:
		return 0
	case TierPro:
		return 1
	case TierEnterprise:
		return 2
	default:
		return -1
	}
}

// IsValidTier checks if the given tier is valid.
func IsValidTier(t Tier) bool {
	return TierOrder(t) >= 0
}

// GetRequiredTier returns the minimum tier required for a feature.
func GetRequiredTier(f Feature) Tier {
	if tier, ok := featureTierMap[f]; ok {
		return tier
	}
	return TierFree // Unknown features default to free
}

// CanAccessFeature checks if a given tier can access a feature.
func CanAccessFeature(userTier Tier, feature Feature) bool {
	requiredTier := GetRequiredTier(feature)
	return TierOrder(userTier) >= TierOrder(requiredTier)
}

// GetTierFeatures returns all features available for a tier.
func GetTierFeatures(tier Tier) []Feature {
	var features []Feature
	for _, f := range AllFeatures() {
		if CanAccessFeature(tier, f) {
			features = append(features, f)
		}
	}
	return features
}

// FeatureInfo provides metadata about a feature.
type FeatureInfo struct {
	Name         Feature `json:"name"`
	DisplayName  string  `json:"display_name"`
	Description  string  `json:"description"`
	RequiredTier Tier    `json:"required_tier"`
}

// GetFeatureInfo returns metadata for a feature.
func GetFeatureInfo(f Feature) FeatureInfo {
	info := FeatureInfo{
		Name:         f,
		RequiredTier: GetRequiredTier(f),
	}

	switch f {
	case FeatureOIDC:
		info.DisplayName = "OIDC Authentication"
		info.Description = "Single sign-on with OpenID Connect providers"
	case FeatureAuditLogs:
		info.DisplayName = "Audit Logs"
		info.Description = "Comprehensive logging of all user and system actions"
	case FeatureNotificationSlack:
		info.DisplayName = "Slack Notifications"
		info.Description = "Send backup notifications to Slack channels"
	case FeatureNotificationTeams:
		info.DisplayName = "Teams Notifications"
		info.Description = "Send backup notifications to Microsoft Teams channels"
	case FeatureNotificationPagerDuty:
		info.DisplayName = "PagerDuty Notifications"
		info.Description = "Send backup alerts to PagerDuty for incident management"
	case FeatureNotificationDiscord:
		info.DisplayName = "Discord Notifications"
		info.Description = "Send backup notifications to Discord channels"
	case FeatureStorageS3:
		info.DisplayName = "S3 Storage"
		info.Description = "Use S3-compatible object storage as backup destination"
	case FeatureStorageB2:
		info.DisplayName = "Backblaze B2 Storage"
		info.Description = "Use Backblaze B2 cloud storage as backup destination"
	case FeatureStorageSFTP:
		info.DisplayName = "SFTP Storage"
		info.Description = "Use SFTP servers as backup destination"
	case FeatureStorageDropbox:
		info.DisplayName = "Dropbox Storage"
		info.Description = "Use Dropbox as backup destination"
	case FeatureStorageRest:
		info.DisplayName = "REST Server Storage"
		info.Description = "Use restic REST server as backup destination"
	case FeatureDockerBackup:
		info.DisplayName = "Docker Backup"
		info.Description = "Back up Docker containers and volumes"
	case FeatureMultiRepo:
		info.DisplayName = "Multiple Repositories"
		info.Description = "Configure multiple backup repositories for redundancy"
	case FeatureAPIAccess:
		info.DisplayName = "API Access"
		info.Description = "Programmatic access via REST API"
	case FeatureCustomReports:
		info.DisplayName = "Custom Reports"
		info.Description = "Custom report generation and scheduling"
	case FeatureCustomRetention:
		info.DisplayName = "Custom Retention"
		info.Description = "Custom data retention policies and schedules"
	case FeatureMultiOrg:
		info.DisplayName = "Multiple Organizations"
		info.Description = "Manage multiple organizations from a single account"
	case FeatureSLATracking:
		info.DisplayName = "SLA Tracking"
		info.Description = "Service level agreement monitoring and compliance reporting"
	case FeatureWhiteLabel:
		info.DisplayName = "White-Label"
		info.Description = "Custom branding and white-labeling capabilities"
	case FeatureAirGap:
		info.DisplayName = "Air-Gapped Deployment"
		info.Description = "Deploy in air-gapped environments without internet access"
	case FeatureDRRunbooks:
		info.DisplayName = "DR Runbooks"
		info.Description = "Disaster recovery runbooks and automation workflows"
	case FeatureDRTests:
		info.DisplayName = "DR Testing"
		info.Description = "Automated disaster recovery testing and validation"
	case FeatureSSOSync:
		info.DisplayName = "SSO Directory Sync"
		info.Description = "Synchronize users and groups from SSO directory providers"
	case FeatureRBAC:
		info.DisplayName = "Role-Based Access Control"
		info.Description = "Fine-grained role-based access control for users and teams"
	case FeatureGeoReplication:
		info.DisplayName = "Geo-Replication"
		info.Description = "Cross-region backup replication for disaster recovery"
	case FeatureRansomwareProtect:
		info.DisplayName = "Ransomware Protection"
		info.Description = "Ransomware detection and immutable backup protection"
	case FeatureLegalHolds:
		info.DisplayName = "Legal Holds"
		info.Description = "Place legal holds on backups to prevent deletion"
	case FeaturePrioritySupport:
		info.DisplayName = "Priority Support"
		info.Description = "Priority email and chat support with faster response times"
	default:
		info.DisplayName = string(f)
		info.Description = "Unknown feature"
	}

	return info
}

// GetAllFeatureInfo returns metadata for all features.
func GetAllFeatureInfo() []FeatureInfo {
	features := AllFeatures()
	infos := make([]FeatureInfo, len(features))
	for i, f := range features {
		infos[i] = GetFeatureInfo(f)
	}
	return infos
}

// TierInfo provides metadata about a tier.
type TierInfo struct {
	Name        Tier      `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Features    []Feature `json:"features"`
}

// GetTierInfo returns metadata for a tier.
func GetTierInfo(t Tier) TierInfo {
	info := TierInfo{
		Name:     t,
		Features: GetTierFeatures(t),
	}

	switch t {
	case TierFree:
		info.DisplayName = "Free"
		info.Description = "Basic backup functionality"
	case TierPro:
		info.DisplayName = "Professional"
		info.Description = "Advanced features for growing teams"
	case TierEnterprise:
		info.DisplayName = "Enterprise"
		info.Description = "Full feature set for large organizations"
	default:
		info.DisplayName = string(t)
		info.Description = "Unknown tier"
	}

	return info
}

// GetAllTierInfo returns metadata for all tiers.
func GetAllTierInfo() []TierInfo {
	return []TierInfo{
		GetTierInfo(TierFree),
		GetTierInfo(TierPro),
		GetTierInfo(TierEnterprise),
	}
}

// FeatureStore defines the interface for feature/tier persistence.
type FeatureStore interface {
	GetOrgTier(ctx context.Context, orgID uuid.UUID) (Tier, error)
	SetOrgTier(ctx context.Context, orgID uuid.UUID, tier Tier) error
}

// FeatureChecker provides runtime feature checking for organizations.
type FeatureChecker struct {
	store FeatureStore
	cache map[uuid.UUID]Tier
	mu    sync.RWMutex
}

// NewFeatureChecker creates a new FeatureChecker.
func NewFeatureChecker(store FeatureStore) *FeatureChecker {
	return &FeatureChecker{
		store: store,
		cache: make(map[uuid.UUID]Tier),
	}
}

// GetOrgTier retrieves the tier for an organization, using cache when available.
func (fc *FeatureChecker) GetOrgTier(ctx context.Context, orgID uuid.UUID) (Tier, error) {
	// Check cache first
	fc.mu.RLock()
	if tier, ok := fc.cache[orgID]; ok {
		fc.mu.RUnlock()
		return tier, nil
	}
	fc.mu.RUnlock()

	// Fetch from store
	tier, err := fc.store.GetOrgTier(ctx, orgID)
	if err != nil {
		return TierFree, err
	}

	// Update cache
	fc.mu.Lock()
	fc.cache[orgID] = tier
	fc.mu.Unlock()

	return tier, nil
}

// CheckFeature checks if an organization has access to a feature.
func (fc *FeatureChecker) CheckFeature(ctx context.Context, orgID uuid.UUID, feature Feature) (bool, error) {
	tier, err := fc.GetOrgTier(ctx, orgID)
	if err != nil {
		return false, err
	}
	return CanAccessFeature(tier, feature), nil
}

// InvalidateCache removes an organization from the cache.
func (fc *FeatureChecker) InvalidateCache(orgID uuid.UUID) {
	fc.mu.Lock()
	delete(fc.cache, orgID)
	fc.mu.Unlock()
}

// ClearCache removes all organizations from the cache.
func (fc *FeatureChecker) ClearCache() {
	fc.mu.Lock()
	fc.cache = make(map[uuid.UUID]Tier)
	fc.mu.Unlock()
}

// FeatureCheckResult contains the result of a feature check.
type FeatureCheckResult struct {
	Feature      Feature      `json:"feature"`
	Enabled      bool         `json:"enabled"`
	CurrentTier  Tier         `json:"current_tier"`
	RequiredTier Tier         `json:"required_tier"`
	UpgradeInfo  *UpgradeInfo `json:"upgrade_info,omitempty"`
}

// UpgradeInfo provides information about upgrading to access a feature.
type UpgradeInfo struct {
	RequiredTier Tier   `json:"required_tier"`
	DisplayName  string `json:"display_name"`
	Message      string `json:"message"`
}

// CheckFeatureWithInfo checks if an organization has access to a feature and returns detailed info.
func (fc *FeatureChecker) CheckFeatureWithInfo(ctx context.Context, orgID uuid.UUID, feature Feature) (*FeatureCheckResult, error) {
	tier, err := fc.GetOrgTier(ctx, orgID)
	if err != nil {
		return nil, err
	}

	requiredTier := GetRequiredTier(feature)
	enabled := CanAccessFeature(tier, feature)

	result := &FeatureCheckResult{
		Feature:      feature,
		Enabled:      enabled,
		CurrentTier:  tier,
		RequiredTier: requiredTier,
	}

	if !enabled {
		tierInfo := GetTierInfo(requiredTier)
		result.UpgradeInfo = &UpgradeInfo{
			RequiredTier: requiredTier,
			DisplayName:  tierInfo.DisplayName,
			Message:      "Upgrade to " + tierInfo.DisplayName + " to access this feature",
		}
	}

	return result, nil
}
