package license

import (
	"errors"
	"time"
)

// DefaultTrialDuration is the default trial period (14 days).
const DefaultTrialDuration = 14 * 24 * time.Hour

// Trial represents a time-limited trial of a license tier.
type Trial struct {
	License   *License
	StartedAt time.Time
	Duration  time.Duration
}

// StartTrial creates a new trial for the given tier and customer.
func StartTrial(tier LicenseTier, customerID string) (*Trial, error) {
	if !tier.IsValid() {
		return nil, errors.New("invalid license tier")
	}
	if customerID == "" {
		return nil, errors.New("missing customer ID")
	}
	if tier == TierFree {
		return nil, errors.New("cannot start trial for free tier")
	}

	now := time.Now()
	return &Trial{
		License: &License{
			Tier:       tier,
			CustomerID: customerID,
			ExpiresAt:  now.Add(DefaultTrialDuration),
			IssuedAt:   now,
			Limits:     GetLimits(tier),
		},
		StartedAt: now,
		Duration:  DefaultTrialDuration,
	}, nil
}

// IsActive returns true if the trial has not yet expired.
func (t *Trial) IsActive() bool {
	if t == nil || t.License == nil {
		return false
	}
	return time.Now().Before(t.License.ExpiresAt)
}

// IsExpired returns true if the trial period has ended.
func (t *Trial) IsExpired() bool {
	return !t.IsActive()
}

// Convert creates a paid license from a trial with the specified tier.
func (t *Trial) Convert(tier LicenseTier) (*License, error) {
	if t == nil || t.License == nil {
		return nil, errors.New("nil trial")
	}
	if !tier.IsValid() {
		return nil, errors.New("invalid license tier")
	}
	if tier == TierFree {
		return nil, errors.New("cannot convert trial to free tier")
	}

	now := time.Now()
	return &License{
		Tier:       tier,
		CustomerID: t.License.CustomerID,
		ExpiresAt:  now.Add(365 * 24 * time.Hour),
		IssuedAt:   now,
		Limits:     GetLimits(tier),
	}, nil
// Package license provides trial and subscription management for Keldris.
package license

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// DefaultTrialDays is the standard trial period duration.
const DefaultTrialDays = 30

// MaxExtensionDays is the maximum days an admin can extend a trial.
const MaxExtensionDays = 90

// PlanTier represents the subscription level.
type PlanTier string

const (
	PlanTierFree       PlanTier = "free"
	PlanTierPro        PlanTier = "pro"
	PlanTierEnterprise PlanTier = "enterprise"
)

// TrialStatus represents the current state of a trial.
type TrialStatus string

const (
	TrialStatusNone      TrialStatus = "none"
	TrialStatusActive    TrialStatus = "active"
	TrialStatusExpired   TrialStatus = "expired"
	TrialStatusConverted TrialStatus = "converted"
)

// TrialInfo holds the trial status for an organization.
type TrialInfo struct {
	OrgID            uuid.UUID   `json:"org_id"`
	PlanTier         PlanTier    `json:"plan_tier"`
	TrialStatus      TrialStatus `json:"trial_status"`
	TrialStartedAt   *time.Time  `json:"trial_started_at,omitempty"`
	TrialEndsAt      *time.Time  `json:"trial_ends_at,omitempty"`
	TrialEmail       string      `json:"trial_email,omitempty"`
	TrialConvertedAt *time.Time  `json:"trial_converted_at,omitempty"`
	DaysRemaining    int         `json:"days_remaining"`
	IsTrialActive    bool        `json:"is_trial_active"`
	HasProFeatures   bool        `json:"has_pro_features"`
}

// TrialExtension records a trial period extension.
type TrialExtension struct {
	ID             uuid.UUID `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	ExtendedBy     uuid.UUID `json:"extended_by"`
	ExtendedByName string    `json:"extended_by_name,omitempty"`
	ExtensionDays  int       `json:"extension_days"`
	Reason         string    `json:"reason,omitempty"`
	PreviousEndsAt time.Time `json:"previous_ends_at"`
	NewEndsAt      time.Time `json:"new_ends_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// TrialActivity logs feature access during trial.
type TrialActivity struct {
	ID          uuid.UUID              `json:"id"`
	OrgID       uuid.UUID              `json:"org_id"`
	UserID      *uuid.UUID             `json:"user_id,omitempty"`
	FeatureName string                 `json:"feature_name"`
	Action      string                 `json:"action"` // accessed, blocked, limit_reached
	Details     map[string]interface{} `json:"details,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
}

// ProFeature defines a feature available in Pro tier.
type ProFeature struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Limit       *int   `json:"limit,omitempty"` // nil means unlimited
}

// Feature names for Pro tier.
const (
	FeatureAdvancedScheduling  = "advanced_scheduling"
	FeatureGeoReplication      = "geo_replication"
	FeatureDRRunbooks          = "dr_runbooks"
	FeatureStorageTiering      = "storage_tiering"
	FeatureClassifications     = "classifications"
	FeatureCustomReports       = "custom_reports"
	FeatureSLAMonitoring       = "sla_monitoring"
	FeatureCostEstimation      = "cost_estimation"
	FeatureRansomwareDetection = "ransomware_detection"
	FeatureUnlimitedAgents     = "unlimited_agents"
	FeatureAPIAccess           = "api_access"
	FeaturePrioritySupport     = "priority_support"
)

// StartTrialRequest is the request to start a trial.
type StartTrialRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ExtendTrialRequest is the request to extend a trial.
type ExtendTrialRequest struct {
	ExtensionDays int    `json:"extension_days" binding:"required,min=1,max=90"`
	Reason        string `json:"reason" binding:"required,max=500"`
}

// ConvertTrialRequest is the request to convert a trial to paid.
type ConvertTrialRequest struct {
	PlanTier PlanTier `json:"plan_tier" binding:"required,oneof=pro enterprise"`
}

// TrialStore defines the interface for trial data persistence.
type TrialStore interface {
	GetTrialInfo(ctx context.Context, orgID uuid.UUID) (*TrialInfo, error)
	StartTrial(ctx context.Context, orgID uuid.UUID, email string) error
	ExtendTrial(ctx context.Context, orgID, extendedBy uuid.UUID, days int, reason string) (*TrialExtension, error)
	ConvertTrial(ctx context.Context, orgID uuid.UUID, tier PlanTier) error
	ExpireTrials(ctx context.Context) (int, error)
	GetTrialExtensions(ctx context.Context, orgID uuid.UUID) ([]*TrialExtension, error)
	LogTrialActivity(ctx context.Context, activity *TrialActivity) error
	GetTrialActivity(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*TrialActivity, error)
	GetExpiringTrials(ctx context.Context, withinDays int) ([]*TrialInfo, error)
}

// TrialManager handles trial-related business logic.
type TrialManager struct {
	store TrialStore
}

// NewTrialManager creates a new trial manager.
func NewTrialManager(store TrialStore) *TrialManager {
	return &TrialManager{store: store}
}

// GetTrialInfo returns the current trial status for an organization.
func (m *TrialManager) GetTrialInfo(ctx context.Context, orgID uuid.UUID) (*TrialInfo, error) {
	info, err := m.store.GetTrialInfo(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Calculate derived fields
	m.calculateDerivedFields(info)
	return info, nil
}

// StartTrial begins a new trial for an organization.
func (m *TrialManager) StartTrial(ctx context.Context, orgID uuid.UUID, email string) (*TrialInfo, error) {
	// Check if already has trial or is paid
	info, err := m.store.GetTrialInfo(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if info.TrialStatus != TrialStatusNone {
		return nil, errors.New("organization already has or had a trial")
	}

	if info.PlanTier != PlanTierFree {
		return nil, errors.New("organization already has a paid subscription")
	}

	if err := m.store.StartTrial(ctx, orgID, email); err != nil {
		return nil, err
	}

	return m.GetTrialInfo(ctx, orgID)
}

// ExtendTrial extends an active trial by the specified number of days.
func (m *TrialManager) ExtendTrial(ctx context.Context, orgID, extendedBy uuid.UUID, days int, reason string) (*TrialExtension, error) {
	if days < 1 || days > MaxExtensionDays {
		return nil, errors.New("extension days must be between 1 and 90")
	}

	// Check if trial is active
	info, err := m.store.GetTrialInfo(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if info.TrialStatus != TrialStatusActive && info.TrialStatus != TrialStatusExpired {
		return nil, errors.New("can only extend active or expired trials")
	}

	return m.store.ExtendTrial(ctx, orgID, extendedBy, days, reason)
}

// ConvertTrial converts a trial to a paid subscription.
func (m *TrialManager) ConvertTrial(ctx context.Context, orgID uuid.UUID, tier PlanTier) error {
	if tier != PlanTierPro && tier != PlanTierEnterprise {
		return errors.New("invalid plan tier for conversion")
	}

	info, err := m.store.GetTrialInfo(ctx, orgID)
	if err != nil {
		return err
	}

	if info.TrialStatus == TrialStatusConverted {
		return errors.New("trial already converted")
	}

	return m.store.ConvertTrial(ctx, orgID, tier)
}

// HasFeatureAccess checks if an organization has access to a Pro feature.
func (m *TrialManager) HasFeatureAccess(ctx context.Context, orgID uuid.UUID, featureName string) (bool, error) {
	info, err := m.GetTrialInfo(ctx, orgID)
	if err != nil {
		return false, err
	}

	return info.HasProFeatures, nil
}

// LogFeatureAccess logs when a user accesses a Pro feature.
func (m *TrialManager) LogFeatureAccess(ctx context.Context, orgID uuid.UUID, userID *uuid.UUID, featureName, action string, details map[string]interface{}) error {
	activity := &TrialActivity{
		ID:          uuid.New(),
		OrgID:       orgID,
		UserID:      userID,
		FeatureName: featureName,
		Action:      action,
		Details:     details,
		CreatedAt:   time.Now(),
	}
	return m.store.LogTrialActivity(ctx, activity)
}

// GetProFeatures returns the list of Pro features with availability status.
func (m *TrialManager) GetProFeatures(ctx context.Context, orgID uuid.UUID) ([]ProFeature, error) {
	info, err := m.GetTrialInfo(ctx, orgID)
	if err != nil {
		return nil, err
	}

	features := []ProFeature{
		{Name: FeatureAdvancedScheduling, Description: "Advanced backup scheduling with complex cron patterns", Available: info.HasProFeatures},
		{Name: FeatureGeoReplication, Description: "Cross-region backup replication", Available: info.HasProFeatures},
		{Name: FeatureDRRunbooks, Description: "Disaster recovery runbooks and automation", Available: info.HasProFeatures},
		{Name: FeatureStorageTiering, Description: "Automatic storage tier management", Available: info.HasProFeatures},
		{Name: FeatureClassifications, Description: "Data classification and compliance tagging", Available: info.HasProFeatures},
		{Name: FeatureCustomReports, Description: "Custom report generation and scheduling", Available: info.HasProFeatures},
		{Name: FeatureSLAMonitoring, Description: "SLA tracking and compliance monitoring", Available: info.HasProFeatures},
		{Name: FeatureCostEstimation, Description: "Storage cost estimation and forecasting", Available: info.HasProFeatures},
		{Name: FeatureRansomwareDetection, Description: "Ransomware detection and alerts", Available: info.HasProFeatures},
		{Name: FeatureUnlimitedAgents, Description: "Unlimited backup agents", Available: info.HasProFeatures},
		{Name: FeatureAPIAccess, Description: "Full REST API access", Available: info.HasProFeatures},
		{Name: FeaturePrioritySupport, Description: "Priority email and chat support", Available: info.HasProFeatures},
	}

	return features, nil
}

// GetTrialExtensions returns the extension history for an organization.
func (m *TrialManager) GetTrialExtensions(ctx context.Context, orgID uuid.UUID) ([]*TrialExtension, error) {
	return m.store.GetTrialExtensions(ctx, orgID)
}

// GetExpiringTrials returns trials expiring within the specified days.
func (m *TrialManager) GetExpiringTrials(ctx context.Context, withinDays int) ([]*TrialInfo, error) {
	trials, err := m.store.GetExpiringTrials(ctx, withinDays)
	if err != nil {
		return nil, err
	}

	for _, t := range trials {
		m.calculateDerivedFields(t)
	}
	return trials, nil
}

// ExpireTrials marks all expired trials as expired.
func (m *TrialManager) ExpireTrials(ctx context.Context) (int, error) {
	return m.store.ExpireTrials(ctx)
}

// calculateDerivedFields computes IsTrialActive, DaysRemaining, and HasProFeatures.
func (m *TrialManager) calculateDerivedFields(info *TrialInfo) {
	now := time.Now()

	// Calculate days remaining
	if info.TrialEndsAt != nil && info.TrialStatus == TrialStatusActive {
		remaining := info.TrialEndsAt.Sub(now)
		info.DaysRemaining = int(remaining.Hours() / 24)
		if info.DaysRemaining < 0 {
			info.DaysRemaining = 0
		}
		info.IsTrialActive = info.DaysRemaining > 0 || remaining > 0
	} else {
		info.DaysRemaining = 0
		info.IsTrialActive = false
	}

	// Determine if org has Pro features
	switch info.PlanTier {
	case PlanTierPro, PlanTierEnterprise:
		info.HasProFeatures = true
	default:
		info.HasProFeatures = info.IsTrialActive
	}
}

// AutoStartTrial automatically starts a trial for new organizations.
// Called during organization creation if auto-trial is enabled.
func (m *TrialManager) AutoStartTrial(ctx context.Context, orgID uuid.UUID, email string) (*TrialInfo, error) {
	// For auto-start, we use the provided email or empty string
	if err := m.store.StartTrial(ctx, orgID, email); err != nil {
		return nil, err
	}
	return m.GetTrialInfo(ctx, orgID)
}
