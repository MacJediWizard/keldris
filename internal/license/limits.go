package license

// TierLimits defines the resource limits for a license tier.
// Only tracks resources monitored by the license server via heartbeat.
type TierLimits struct {
	MaxAgents int `json:"max_agents"`
	MaxUsers  int `json:"max_users"`
	MaxOrgs   int `json:"max_orgs"`
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
import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	// ErrAgentLimitExceeded indicates the agent limit has been reached.
	ErrAgentLimitExceeded = errors.New("agent limit exceeded for current license tier")
	// ErrUserLimitExceeded indicates the user limit has been reached.
	ErrUserLimitExceeded = errors.New("user limit exceeded for current license tier")
	// ErrOrgLimitExceeded indicates the organization limit has been reached.
	ErrOrgLimitExceeded = errors.New("organization limit exceeded for current license tier")
	// ErrRepositoryLimitExceeded indicates the repository limit has been reached.
	ErrRepositoryLimitExceeded = errors.New("repository limit exceeded for current license tier")
	// ErrFeatureNotAvailable indicates the feature is not available in current tier.
	ErrFeatureNotAvailable = errors.New("feature not available in current license tier")
)

// LimitStore defines the interface for checking resource counts.
type LimitStore interface {
	CountAgents(ctx context.Context) (int, error)
	CountUsers(ctx context.Context) (int, error)
	CountOrganizations(ctx context.Context) (int, error)
	CountRepositories(ctx context.Context, orgID uuid.UUID) (int, error)
}

// Enforcer enforces license limits and feature gates.
type Enforcer struct {
	manager *Manager
	store   LimitStore
}

// NewEnforcer creates a new license limit enforcer.
func NewEnforcer(manager *Manager, store LimitStore) *Enforcer {
	return &Enforcer{
		manager: manager,
		store:   store,
	}
}

// CanAddAgent checks if a new agent can be added within license limits.
func (e *Enforcer) CanAddAgent(ctx context.Context) error {
	limits := e.manager.GetLimits()
	if limits.MaxAgents == -1 {
		return nil // unlimited
	}

	count, err := e.store.CountAgents(ctx)
	if err != nil {
		return fmt.Errorf("count agents: %w", err)
	}

	if count >= limits.MaxAgents {
		return ErrAgentLimitExceeded
	}
	return nil
}

// CanAddUser checks if a new user can be added within license limits.
func (e *Enforcer) CanAddUser(ctx context.Context) error {
	limits := e.manager.GetLimits()
	if limits.MaxUsers == -1 {
		return nil // unlimited
	}

	count, err := e.store.CountUsers(ctx)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	if count >= limits.MaxUsers {
		return ErrUserLimitExceeded
	}
	return nil
}

// CanAddOrganization checks if a new organization can be added within license limits.
func (e *Enforcer) CanAddOrganization(ctx context.Context) error {
	limits := e.manager.GetLimits()
	if limits.MaxOrganizations == -1 {
		return nil // unlimited
	}

	count, err := e.store.CountOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("count organizations: %w", err)
	}

	if count >= limits.MaxOrganizations {
		return ErrOrgLimitExceeded
	}
	return nil
}

// CanAddRepository checks if a new repository can be added within license limits.
func (e *Enforcer) CanAddRepository(ctx context.Context, orgID uuid.UUID) error {
	limits := e.manager.GetLimits()
	if limits.MaxRepositories == -1 {
		return nil // unlimited
	}

	count, err := e.store.CountRepositories(ctx, orgID)
	if err != nil {
		return fmt.Errorf("count repositories: %w", err)
	}

	if count >= limits.MaxRepositories {
		return ErrRepositoryLimitExceeded
	}
	return nil
}

// CheckAgentLimit returns current agent count and limit.
func (e *Enforcer) CheckAgentLimit(ctx context.Context) (current, limit int, err error) {
	limits := e.manager.GetLimits()
	count, err := e.store.CountAgents(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("count agents: %w", err)
	}
	return count, limits.MaxAgents, nil
}

// CheckUserLimit returns current user count and limit.
func (e *Enforcer) CheckUserLimit(ctx context.Context) (current, limit int, err error) {
	limits := e.manager.GetLimits()
	count, err := e.store.CountUsers(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("count users: %w", err)
	}
	return count, limits.MaxUsers, nil
}

// CheckOrgLimit returns current organization count and limit.
func (e *Enforcer) CheckOrgLimit(ctx context.Context) (current, limit int, err error) {
	limits := e.manager.GetLimits()
	count, err := e.store.CountOrganizations(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("count organizations: %w", err)
	}
	return count, limits.MaxOrganizations, nil
}

// CheckRepositoryLimit returns current repository count and limit for an organization.
func (e *Enforcer) CheckRepositoryLimit(ctx context.Context, orgID uuid.UUID) (current, limit int, err error) {
	limits := e.manager.GetLimits()
	count, err := e.store.CountRepositories(ctx, orgID)
	if err != nil {
		return 0, 0, fmt.Errorf("count repositories: %w", err)
	}
	return count, limits.MaxRepositories, nil
}

// Feature gates

// RequireSSO checks if SSO feature is available.
func (e *Enforcer) RequireSSO() error {
	features := e.manager.GetFeatures()
	if !features.SSO {
		return fmt.Errorf("%w: SSO requires Team or Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// RequireRBAC checks if RBAC feature is available.
func (e *Enforcer) RequireRBAC() error {
	features := e.manager.GetFeatures()
	if !features.RBAC {
		return fmt.Errorf("%w: RBAC requires Team or Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// RequireAuditLogs checks if audit logs feature is available.
func (e *Enforcer) RequireAuditLogs() error {
	features := e.manager.GetFeatures()
	if !features.AuditLogs {
		return fmt.Errorf("%w: Audit logs require Team or Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// RequireGeoReplication checks if geo-replication feature is available.
func (e *Enforcer) RequireGeoReplication() error {
	features := e.manager.GetFeatures()
	if !features.GeoReplication {
		return fmt.Errorf("%w: Geo-replication requires Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// RequireRansomwareProtection checks if ransomware protection feature is available.
func (e *Enforcer) RequireRansomwareProtection() error {
	features := e.manager.GetFeatures()
	if !features.RansomwareProtection {
		return fmt.Errorf("%w: Ransomware protection requires Team or Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// RequireLegalHolds checks if legal holds feature is available.
func (e *Enforcer) RequireLegalHolds() error {
	features := e.manager.GetFeatures()
	if !features.LegalHolds {
		return fmt.Errorf("%w: Legal holds require Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// RequireCustomRetention checks if custom retention feature is available.
func (e *Enforcer) RequireCustomRetention() error {
	features := e.manager.GetFeatures()
	if !features.CustomRetention {
		return fmt.Errorf("%w: Custom retention requires Team or Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// RequireAPIAccess checks if API access feature is available.
func (e *Enforcer) RequireAPIAccess() error {
	features := e.manager.GetFeatures()
	if !features.APIAccess {
		return fmt.Errorf("%w: API access not available", ErrFeatureNotAvailable)
	}
	return nil
}

// RequirePrioritySupport checks if priority support feature is available.
func (e *Enforcer) RequirePrioritySupport() error {
	features := e.manager.GetFeatures()
	if !features.PrioritySupport {
		return fmt.Errorf("%w: Priority support requires Enterprise license", ErrFeatureNotAvailable)
	}
	return nil
}

// IsFeatureEnabled checks if a specific feature is enabled.
func (e *Enforcer) IsFeatureEnabled(feature string) bool {
	features := e.manager.GetFeatures()
	switch feature {
	case "sso":
		return features.SSO
	case "rbac":
		return features.RBAC
	case "audit_logs":
		return features.AuditLogs
	case "geo_replication":
		return features.GeoReplication
	case "ransomware_protection":
		return features.RansomwareProtection
	case "legal_holds":
		return features.LegalHolds
	case "custom_retention":
		return features.CustomRetention
	case "api_access":
		return features.APIAccess
	case "priority_support":
		return features.PrioritySupport
	default:
		return false
	}
}

// GetAllFeatures returns a map of all feature flags.
func (e *Enforcer) GetAllFeatures() map[string]bool {
	features := e.manager.GetFeatures()
	return map[string]bool{
		"sso":                   features.SSO,
		"rbac":                  features.RBAC,
		"audit_logs":            features.AuditLogs,
		"geo_replication":       features.GeoReplication,
		"ransomware_protection": features.RansomwareProtection,
		"legal_holds":           features.LegalHolds,
		"custom_retention":      features.CustomRetention,
		"api_access":            features.APIAccess,
		"priority_support":      features.PrioritySupport,
	}
}

// UsageStats contains current resource usage statistics.
type UsageStats struct {
	Agents        UsageStat `json:"agents"`
	Users         UsageStat `json:"users"`
	Organizations UsageStat `json:"organizations"`
	Repositories  UsageStat `json:"repositories"`
}

// UsageStat represents a single usage statistic with current count and limit.
type UsageStat struct {
	Current   int  `json:"current"`
	Limit     int  `json:"limit"`
	Unlimited bool `json:"unlimited"`
}

// GetUsageStats returns current usage statistics.
func (e *Enforcer) GetUsageStats(ctx context.Context) (*UsageStats, error) {
	limits := e.manager.GetLimits()

	agentCount, err := e.store.CountAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("count agents: %w", err)
	}

	userCount, err := e.store.CountUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}

	orgCount, err := e.store.CountOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("count organizations: %w", err)
	}

	return &UsageStats{
		Agents: UsageStat{
			Current:   agentCount,
			Limit:     limits.MaxAgents,
			Unlimited: limits.MaxAgents == -1,
		},
		Users: UsageStat{
			Current:   userCount,
			Limit:     limits.MaxUsers,
			Unlimited: limits.MaxUsers == -1,
		},
		Organizations: UsageStat{
			Current:   orgCount,
			Limit:     limits.MaxOrganizations,
			Unlimited: limits.MaxOrganizations == -1,
		},
		Repositories: UsageStat{
			Current:   0, // Would need org context
			Limit:     limits.MaxRepositories,
			Unlimited: limits.MaxRepositories == -1,
		},
	}, nil
}

// IsStorageUnlimited returns true if the storage limit is unlimited.
func IsStorageUnlimited(limit int64) bool {
	return limit == Unlimited
}
