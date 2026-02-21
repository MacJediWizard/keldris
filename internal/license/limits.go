package license

// TierLimits defines the resource limits for a license tier.
// Only tracks resources monitored by the license server via heartbeat.
type TierLimits struct {
	MaxAgents int `json:"max_agents"`
	MaxUsers  int `json:"max_users"`
	MaxOrgs   int `json:"max_orgs"`
}

// Unlimited is a sentinel value indicating no limit on a resource.
const Unlimited = -1

// tierLimits maps each license tier to its resource limits.
var tierLimits = map[LicenseTier]TierLimits{
	TierFree: {
		MaxAgents: 3,
		MaxUsers:  3,
		MaxOrgs:   1,
	},
	TierPro: {
		MaxAgents: 25,
		MaxUsers:  10,
		MaxOrgs:   3,
	},
	TierEnterprise: {
		MaxAgents: Unlimited,
		MaxUsers:  Unlimited,
		MaxOrgs:   Unlimited,
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
