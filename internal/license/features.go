package license

// Feature represents a gated feature that requires a specific license tier.
// Package license provides feature gating based on license tiers.
package license

// Package license provides feature gating based on license tiers.
package license

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Tier represents a license tier level.
type Tier string

const (
	// TierFree is the free tier with basic features.
	TierFree Tier = "free"
	// TierPro is the professional tier with advanced features.
	TierPro Tier = "pro"
	// TierEnterprise is the enterprise tier with all features.
	TierEnterprise Tier = "enterprise"
)

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
	// FeatureAuditLogs enables comprehensive audit logging (Pro+).
	FeatureAuditLogs Feature = "audit_logs"
	// FeatureMultiOrg enables multiple organizations (Enterprise).
	FeatureMultiOrg Feature = "multi_org"
	// FeatureSLATracking enables SLA tracking and compliance (Enterprise).
	FeatureSLATracking Feature = "sla_tracking"
	// FeatureWhiteLabel enables white-labeling capabilities (Enterprise).
	FeatureWhiteLabel Feature = "white_label"
)

// featureTierMap defines which tier is required for each feature.
var featureTierMap = map[Feature]Tier{
	FeatureOIDC:        TierPro,
	FeatureAuditLogs:   TierPro,
	FeatureMultiOrg:    TierEnterprise,
	FeatureSLATracking: TierEnterprise,
	FeatureWhiteLabel:  TierEnterprise,
}

// AllFeatures returns all defined features.
func AllFeatures() []Feature {
	return []Feature{
		FeatureOIDC,
		FeatureAuditLogs,
		FeatureMultiOrg,
		FeatureSLATracking,
		FeatureWhiteLabel,
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
	case FeatureMultiOrg:
		info.DisplayName = "Multiple Organizations"
		info.Description = "Manage multiple organizations from a single account"
	case FeatureSLATracking:
		info.DisplayName = "SLA Tracking"
		info.Description = "Service level agreement monitoring and compliance reporting"
	case FeatureWhiteLabel:
		info.DisplayName = "White-Label"
		info.Description = "Custom branding and white-labeling capabilities"
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
