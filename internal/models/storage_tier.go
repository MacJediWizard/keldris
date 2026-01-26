package models

import (
	"time"

	"github.com/google/uuid"
)

// StorageTierType represents the tier classification for storage.
type StorageTierType string

const (
	// StorageTierHot is for frequently accessed data with immediate availability.
	StorageTierHot StorageTierType = "hot"
	// StorageTierWarm is for less frequently accessed data with fast retrieval.
	StorageTierWarm StorageTierType = "warm"
	// StorageTierCold is for infrequently accessed data with slower retrieval.
	StorageTierCold StorageTierType = "cold"
	// StorageTierArchive is for long-term retention with slowest retrieval.
	StorageTierArchive StorageTierType = "archive"
)

// StorageTierConfig represents organization-level tier configuration.
type StorageTierConfig struct {
	ID             uuid.UUID       `json:"id"`
	OrgID          uuid.UUID       `json:"org_id"`
	TierType       StorageTierType `json:"tier_type"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	CostPerGBMonth float64         `json:"cost_per_gb_month"`
	RetrievalCost  float64         `json:"retrieval_cost"`
	RetrievalTime  string          `json:"retrieval_time"` // e.g., "immediate", "1-5 minutes", "1-12 hours"
	Enabled        bool            `json:"enabled"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// NewStorageTierConfig creates a new tier configuration.
func NewStorageTierConfig(orgID uuid.UUID, tierType StorageTierType, name string) *StorageTierConfig {
	now := time.Now()
	return &StorageTierConfig{
		ID:        uuid.New(),
		OrgID:     orgID,
		TierType:  tierType,
		Name:      name,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// DefaultTierConfigs returns the default tier configurations for an organization.
func DefaultTierConfigs(orgID uuid.UUID) []*StorageTierConfig {
	configs := []*StorageTierConfig{
		{
			ID:             uuid.New(),
			OrgID:          orgID,
			TierType:       StorageTierHot,
			Name:           "Hot Storage",
			Description:    "Frequently accessed data with immediate availability",
			CostPerGBMonth: 0.023,
			RetrievalCost:  0.0,
			RetrievalTime:  "immediate",
			Enabled:        true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			OrgID:          orgID,
			TierType:       StorageTierWarm,
			Name:           "Warm Storage",
			Description:    "Infrequently accessed data with fast retrieval",
			CostPerGBMonth: 0.0125,
			RetrievalCost:  0.01,
			RetrievalTime:  "1-5 minutes",
			Enabled:        true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			OrgID:          orgID,
			TierType:       StorageTierCold,
			Name:           "Cold Storage",
			Description:    "Rarely accessed data with slower retrieval",
			CostPerGBMonth: 0.004,
			RetrievalCost:  0.02,
			RetrievalTime:  "1-5 hours",
			Enabled:        true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             uuid.New(),
			OrgID:          orgID,
			TierType:       StorageTierArchive,
			Name:           "Archive Storage",
			Description:    "Long-term retention with slowest retrieval",
			CostPerGBMonth: 0.00099,
			RetrievalCost:  0.03,
			RetrievalTime:  "1-12 hours",
			Enabled:        true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}
	return configs
}

// TierRule defines when snapshots should transition between tiers.
type TierRule struct {
	ID              uuid.UUID       `json:"id"`
	OrgID           uuid.UUID       `json:"org_id"`
	RepositoryID    *uuid.UUID      `json:"repository_id,omitempty"` // nil means applies to all repos
	ScheduleID      *uuid.UUID      `json:"schedule_id,omitempty"`   // nil means applies to all schedules
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	FromTier        StorageTierType `json:"from_tier"`
	ToTier          StorageTierType `json:"to_tier"`
	AgeThresholdDay int             `json:"age_threshold_days"` // Move after X days in from_tier
	MinCopies       int             `json:"min_copies"`         // Minimum copies to keep in from_tier
	Priority        int             `json:"priority"`           // Lower number = higher priority
	Enabled         bool            `json:"enabled"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewTierRule creates a new tier rule.
func NewTierRule(orgID uuid.UUID, name string, fromTier, toTier StorageTierType, ageDays int) *TierRule {
	now := time.Now()
	return &TierRule{
		ID:              uuid.New(),
		OrgID:           orgID,
		Name:            name,
		FromTier:        fromTier,
		ToTier:          toTier,
		AgeThresholdDay: ageDays,
		MinCopies:       1,
		Priority:        100,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// DefaultTierRules returns default tier transition rules for an organization.
func DefaultTierRules(orgID uuid.UUID) []*TierRule {
	return []*TierRule{
		{
			ID:              uuid.New(),
			OrgID:           orgID,
			Name:            "Hot to Warm (30 days)",
			Description:     "Move snapshots to warm storage after 30 days",
			FromTier:        StorageTierHot,
			ToTier:          StorageTierWarm,
			AgeThresholdDay: 30,
			MinCopies:       1,
			Priority:        10,
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              uuid.New(),
			OrgID:           orgID,
			Name:            "Warm to Cold (90 days)",
			Description:     "Move snapshots to cold storage after 90 days",
			FromTier:        StorageTierWarm,
			ToTier:          StorageTierCold,
			AgeThresholdDay: 90,
			MinCopies:       1,
			Priority:        20,
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              uuid.New(),
			OrgID:           orgID,
			Name:            "Cold to Archive (365 days)",
			Description:     "Move snapshots to archive after 1 year",
			FromTier:        StorageTierCold,
			ToTier:          StorageTierArchive,
			AgeThresholdDay: 365,
			MinCopies:       1,
			Priority:        30,
			Enabled:         true,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}
}

// SnapshotTier tracks the current tier of a snapshot.
type SnapshotTier struct {
	ID           uuid.UUID       `json:"id"`
	SnapshotID   string          `json:"snapshot_id"`
	RepositoryID uuid.UUID       `json:"repository_id"`
	OrgID        uuid.UUID       `json:"org_id"`
	CurrentTier  StorageTierType `json:"current_tier"`
	SizeBytes    int64           `json:"size_bytes"`
	SnapshotTime time.Time       `json:"snapshot_time"`
	TieredAt     time.Time       `json:"tiered_at"` // When it was moved to current tier
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// NewSnapshotTier creates a new snapshot tier record.
func NewSnapshotTier(snapshotID string, repositoryID, orgID uuid.UUID, sizeBytes int64, snapshotTime time.Time) *SnapshotTier {
	now := time.Now()
	return &SnapshotTier{
		ID:           uuid.New(),
		SnapshotID:   snapshotID,
		RepositoryID: repositoryID,
		OrgID:        orgID,
		CurrentTier:  StorageTierHot, // New snapshots start in hot tier
		SizeBytes:    sizeBytes,
		SnapshotTime: snapshotTime,
		TieredAt:     now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// DaysInCurrentTier returns the number of days the snapshot has been in the current tier.
func (st *SnapshotTier) DaysInCurrentTier() int {
	return int(time.Since(st.TieredAt).Hours() / 24)
}

// AgeDays returns the total age of the snapshot in days.
func (st *SnapshotTier) AgeDays() int {
	return int(time.Since(st.SnapshotTime).Hours() / 24)
}

// TierTransition records a tier change event for auditing.
type TierTransition struct {
	ID              uuid.UUID       `json:"id"`
	SnapshotTierID  uuid.UUID       `json:"snapshot_tier_id"`
	SnapshotID      string          `json:"snapshot_id"`
	RepositoryID    uuid.UUID       `json:"repository_id"`
	OrgID           uuid.UUID       `json:"org_id"`
	FromTier        StorageTierType `json:"from_tier"`
	ToTier          StorageTierType `json:"to_tier"`
	TriggerRuleID   *uuid.UUID      `json:"trigger_rule_id,omitempty"`
	TriggerReason   string          `json:"trigger_reason"`
	SizeBytes       int64           `json:"size_bytes"`
	EstimatedSaving float64         `json:"estimated_saving"` // Monthly cost savings
	Status          string          `json:"status"`           // pending, in_progress, completed, failed
	ErrorMessage    string          `json:"error_message,omitempty"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// NewTierTransition creates a new tier transition record.
func NewTierTransition(snapshotTier *SnapshotTier, toTier StorageTierType, ruleID *uuid.UUID, reason string) *TierTransition {
	return &TierTransition{
		ID:             uuid.New(),
		SnapshotTierID: snapshotTier.ID,
		SnapshotID:     snapshotTier.SnapshotID,
		RepositoryID:   snapshotTier.RepositoryID,
		OrgID:          snapshotTier.OrgID,
		FromTier:       snapshotTier.CurrentTier,
		ToTier:         toTier,
		TriggerRuleID:  ruleID,
		TriggerReason:  reason,
		SizeBytes:      snapshotTier.SizeBytes,
		Status:         "pending",
		CreatedAt:      time.Now(),
	}
}

// Start marks the transition as started.
func (tt *TierTransition) Start() {
	now := time.Now()
	tt.StartedAt = &now
	tt.Status = "in_progress"
}

// Complete marks the transition as completed.
func (tt *TierTransition) Complete() {
	now := time.Now()
	tt.CompletedAt = &now
	tt.Status = "completed"
}

// Fail marks the transition as failed.
func (tt *TierTransition) Fail(errMsg string) {
	now := time.Now()
	tt.CompletedAt = &now
	tt.Status = "failed"
	tt.ErrorMessage = errMsg
}

// TierCostReport represents a cost optimization report for storage tiers.
type TierCostReport struct {
	ID            uuid.UUID             `json:"id"`
	OrgID         uuid.UUID             `json:"org_id"`
	ReportDate    time.Time             `json:"report_date"`
	TotalSize     int64                 `json:"total_size_bytes"`
	CurrentCost   float64               `json:"current_monthly_cost"`
	OptimizedCost float64               `json:"optimized_monthly_cost"`
	PotentialSave float64               `json:"potential_monthly_savings"`
	TierBreakdown []TierBreakdownItem   `json:"tier_breakdown"`
	Suggestions   []TierOptSuggestion   `json:"suggestions"`
	CreatedAt     time.Time             `json:"created_at"`
}

// TierBreakdownItem shows storage distribution by tier.
type TierBreakdownItem struct {
	TierType       StorageTierType `json:"tier_type"`
	SnapshotCount  int             `json:"snapshot_count"`
	TotalSizeBytes int64           `json:"total_size_bytes"`
	MonthlyCost    float64         `json:"monthly_cost"`
	Percentage     float64         `json:"percentage"`
}

// TierOptSuggestion represents a cost optimization suggestion.
type TierOptSuggestion struct {
	SnapshotID     string          `json:"snapshot_id"`
	RepositoryID   uuid.UUID       `json:"repository_id"`
	CurrentTier    StorageTierType `json:"current_tier"`
	SuggestedTier  StorageTierType `json:"suggested_tier"`
	AgeDays        int             `json:"age_days"`
	SizeBytes      int64           `json:"size_bytes"`
	MonthlySavings float64         `json:"monthly_savings"`
	Reason         string          `json:"reason"`
}

// NewTierCostReport creates a new cost report.
func NewTierCostReport(orgID uuid.UUID) *TierCostReport {
	return &TierCostReport{
		ID:            uuid.New(),
		OrgID:         orgID,
		ReportDate:    time.Now(),
		TierBreakdown: []TierBreakdownItem{},
		Suggestions:   []TierOptSuggestion{},
		CreatedAt:     time.Now(),
	}
}

// ColdRestoreRequest represents a request to restore data from cold/archive storage.
type ColdRestoreRequest struct {
	ID              uuid.UUID       `json:"id"`
	OrgID           uuid.UUID       `json:"org_id"`
	SnapshotID      string          `json:"snapshot_id"`
	RepositoryID    uuid.UUID       `json:"repository_id"`
	RequestedBy     uuid.UUID       `json:"requested_by"`
	FromTier        StorageTierType `json:"from_tier"`
	TargetPath      string          `json:"target_path,omitempty"`
	Priority        string          `json:"priority"` // standard, expedited, bulk
	Status          string          `json:"status"`   // pending, warming, ready, restoring, completed, failed, expired
	EstimatedReady  *time.Time      `json:"estimated_ready_at,omitempty"`
	ReadyAt         *time.Time      `json:"ready_at,omitempty"`
	ExpiresAt       *time.Time      `json:"expires_at,omitempty"` // Warmed data expires
	ErrorMessage    string          `json:"error_message,omitempty"`
	RetrievalCost   float64         `json:"retrieval_cost"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// NewColdRestoreRequest creates a new cold restore request.
func NewColdRestoreRequest(orgID uuid.UUID, snapshotID string, repositoryID, requestedBy uuid.UUID, fromTier StorageTierType) *ColdRestoreRequest {
	now := time.Now()
	return &ColdRestoreRequest{
		ID:           uuid.New(),
		OrgID:        orgID,
		SnapshotID:   snapshotID,
		RepositoryID: repositoryID,
		RequestedBy:  requestedBy,
		FromTier:     fromTier,
		Priority:     "standard",
		Status:       "pending",
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// MarkWarming updates status to warming (retrieval in progress).
func (cr *ColdRestoreRequest) MarkWarming(estimatedReady time.Time) {
	cr.Status = "warming"
	cr.EstimatedReady = &estimatedReady
	cr.UpdatedAt = time.Now()
}

// MarkReady updates status to ready for restore.
func (cr *ColdRestoreRequest) MarkReady(expiresAt time.Time) {
	now := time.Now()
	cr.Status = "ready"
	cr.ReadyAt = &now
	cr.ExpiresAt = &expiresAt
	cr.UpdatedAt = now
}

// MarkRestoring updates status to restoring.
func (cr *ColdRestoreRequest) MarkRestoring() {
	cr.Status = "restoring"
	cr.UpdatedAt = time.Now()
}

// MarkCompleted updates status to completed.
func (cr *ColdRestoreRequest) MarkCompleted() {
	cr.Status = "completed"
	cr.UpdatedAt = time.Now()
}

// MarkFailed updates status to failed.
func (cr *ColdRestoreRequest) MarkFailed(errMsg string) {
	cr.Status = "failed"
	cr.ErrorMessage = errMsg
	cr.UpdatedAt = time.Now()
}

// IsExpired checks if the warmed data has expired.
func (cr *ColdRestoreRequest) IsExpired() bool {
	if cr.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*cr.ExpiresAt)
}

// TierStatsSummary provides an overview of tier distribution.
type TierStatsSummary struct {
	TotalSnapshots      int                 `json:"total_snapshots"`
	TotalSizeBytes      int64               `json:"total_size_bytes"`
	EstimatedMonthlyCost float64            `json:"estimated_monthly_cost"`
	ByTier              map[StorageTierType]TierStats `json:"by_tier"`
	PotentialSavings    float64             `json:"potential_savings"`
}

// TierStats represents statistics for a single tier.
type TierStats struct {
	SnapshotCount  int     `json:"snapshot_count"`
	TotalSizeBytes int64   `json:"total_size_bytes"`
	MonthlyCost    float64 `json:"monthly_cost"`
	OldestDays     int     `json:"oldest_snapshot_days"`
	NewestDays     int     `json:"newest_snapshot_days"`
}
