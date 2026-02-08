package license

// TierLimits defines the resource limits for a license tier.
type TierLimits struct {
	MaxAgents  int   `json:"max_agents"`
	MaxUsers   int   `json:"max_users"`
	MaxOrgs    int   `json:"max_orgs"`
	MaxStorage int64 `json:"max_storage_bytes"`
}

// Unlimited is a sentinel value indicating no limit on a resource.
const Unlimited = -1

// tierLimits maps each license tier to its resource limits.
var tierLimits = map[LicenseTier]TierLimits{
	TierFree: {
		MaxAgents:  3,
		MaxUsers:   3,
		MaxOrgs:    1,
		MaxStorage: 10 * 1024 * 1024 * 1024, // 10 GB
	},
	TierPro: {
		MaxAgents:  25,
		MaxUsers:   10,
		MaxOrgs:    3,
		MaxStorage: 100 * 1024 * 1024 * 1024, // 100 GB
	},
	TierEnterprise: {
		MaxAgents:  Unlimited,
		MaxUsers:   Unlimited,
		MaxOrgs:    Unlimited,
		MaxStorage: Unlimited,
	},
}

// GetLimits returns the resource limits for the given license tier.
// Returns free-tier limits for unrecognized tiers.
func GetLimits(tier LicenseTier) TierLimits {
	limits, ok := tierLimits[tier]
	if !ok {
		return tierLimits[TierFree]
	}
	return limits
}

// IsUnlimited returns true if the given limit value represents unlimited.
func IsUnlimited(limit int) bool {
	return limit == Unlimited
}

// IsStorageUnlimited returns true if the storage limit is unlimited.
func IsStorageUnlimited(limit int64) bool {
	return limit == Unlimited
}
