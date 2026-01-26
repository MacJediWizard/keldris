package backup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// TieringStore defines the interface for tiering data operations.
type TieringStore interface {
	// Tier configuration
	GetStorageTierConfigs(ctx context.Context, orgID uuid.UUID) ([]*models.StorageTierConfig, error)
	GetStorageTierConfig(ctx context.Context, id uuid.UUID) (*models.StorageTierConfig, error)
	CreateStorageTierConfig(ctx context.Context, config *models.StorageTierConfig) error
	UpdateStorageTierConfig(ctx context.Context, config *models.StorageTierConfig) error
	CreateDefaultTierConfigs(ctx context.Context, orgID uuid.UUID) error

	// Tier rules
	GetTierRules(ctx context.Context, orgID uuid.UUID) ([]*models.TierRule, error)
	GetTierRule(ctx context.Context, id uuid.UUID) (*models.TierRule, error)
	CreateTierRule(ctx context.Context, rule *models.TierRule) error
	UpdateTierRule(ctx context.Context, rule *models.TierRule) error
	DeleteTierRule(ctx context.Context, id uuid.UUID) error
	GetEnabledTierRules(ctx context.Context, orgID uuid.UUID) ([]*models.TierRule, error)

	// Snapshot tiers
	GetSnapshotTier(ctx context.Context, snapshotID string, repositoryID uuid.UUID) (*models.SnapshotTier, error)
	GetSnapshotTierByID(ctx context.Context, id uuid.UUID) (*models.SnapshotTier, error)
	CreateSnapshotTier(ctx context.Context, tier *models.SnapshotTier) error
	UpdateSnapshotTier(ctx context.Context, tier *models.SnapshotTier) error
	GetSnapshotsForTiering(ctx context.Context, orgID uuid.UUID, currentTier models.StorageTierType, olderThanDays int) ([]*models.SnapshotTier, error)
	GetSnapshotTiersByRepository(ctx context.Context, repositoryID uuid.UUID) ([]*models.SnapshotTier, error)
	GetSnapshotTiersByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.SnapshotTier, error)

	// Tier transitions
	CreateTierTransition(ctx context.Context, transition *models.TierTransition) error
	UpdateTierTransition(ctx context.Context, transition *models.TierTransition) error
	GetPendingTierTransitions(ctx context.Context, orgID uuid.UUID) ([]*models.TierTransition, error)
	GetTierTransitionHistory(ctx context.Context, snapshotID string, repositoryID uuid.UUID, limit int) ([]*models.TierTransition, error)

	// Cold restore requests
	CreateColdRestoreRequest(ctx context.Context, request *models.ColdRestoreRequest) error
	UpdateColdRestoreRequest(ctx context.Context, request *models.ColdRestoreRequest) error
	GetColdRestoreRequest(ctx context.Context, id uuid.UUID) (*models.ColdRestoreRequest, error)
	GetColdRestoreRequestBySnapshot(ctx context.Context, snapshotID string, repositoryID uuid.UUID) (*models.ColdRestoreRequest, error)
	GetPendingColdRestoreRequests(ctx context.Context, orgID uuid.UUID) ([]*models.ColdRestoreRequest, error)
	GetActiveColdRestoreRequests(ctx context.Context, orgID uuid.UUID) ([]*models.ColdRestoreRequest, error)
	ExpireColdRestoreRequests(ctx context.Context) (int, error)

	// Cost reports
	CreateTierCostReport(ctx context.Context, report *models.TierCostReport) error
	GetLatestTierCostReport(ctx context.Context, orgID uuid.UUID) (*models.TierCostReport, error)
	GetTierCostReports(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.TierCostReport, error)

	// Stats
	GetTierStatsSummary(ctx context.Context, orgID uuid.UUID) (*models.TierStatsSummary, error)

	// Organization helper
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
}

// TieringConfig holds configuration for the tiering scheduler.
type TieringConfig struct {
	// ProcessInterval is how often to process tiering rules.
	ProcessInterval time.Duration
	// ReportInterval is how often to generate cost reports.
	ReportInterval time.Duration
	// ColdRestoreCheckInterval is how often to check cold restore status.
	ColdRestoreCheckInterval time.Duration
	// BatchSize is the number of snapshots to process per batch.
	BatchSize int
	// DryRun skips actual tier transitions (for testing).
	DryRun bool
}

// DefaultTieringConfig returns a TieringConfig with sensible defaults.
func DefaultTieringConfig() TieringConfig {
	return TieringConfig{
		ProcessInterval:          6 * time.Hour,
		ReportInterval:           24 * time.Hour,
		ColdRestoreCheckInterval: 15 * time.Minute,
		BatchSize:                100,
		DryRun:                   false,
	}
}

// TieringScheduler manages automatic storage tier transitions.
type TieringScheduler struct {
	store   TieringStore
	restic  *Restic
	config  TieringConfig
	cron    *cron.Cron
	logger  zerolog.Logger
	mu      sync.RWMutex
	running bool
}

// NewTieringScheduler creates a new tiering scheduler.
func NewTieringScheduler(store TieringStore, restic *Restic, config TieringConfig, logger zerolog.Logger) *TieringScheduler {
	return &TieringScheduler{
		store:  store,
		restic: restic,
		config: config,
		cron:   cron.New(cron.WithSeconds()),
		logger: logger.With().Str("component", "tiering_scheduler").Logger(),
	}
}

// Start starts the tiering scheduler.
func (s *TieringScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("tiering scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().Msg("starting tiering scheduler")

	// Process tiering rules every 6 hours (at 1:00, 7:00, 13:00, 19:00)
	_, err := s.cron.AddFunc("0 0 1,7,13,19 * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()
		s.ProcessTieringRules(ctx)
	})
	if err != nil {
		return fmt.Errorf("add tiering cron: %w", err)
	}

	// Generate cost reports daily at 3:00 AM
	_, err = s.cron.AddFunc("0 0 3 * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		s.GenerateCostReports(ctx)
	})
	if err != nil {
		return fmt.Errorf("add report cron: %w", err)
	}

	// Check cold restore requests every 15 minutes
	_, err = s.cron.AddFunc("0 */15 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.ProcessColdRestoreRequests(ctx)
	})
	if err != nil {
		return fmt.Errorf("add cold restore cron: %w", err)
	}

	// Expire old cold restore requests every hour
	_, err = s.cron.AddFunc("0 0 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		s.ExpireColdRestoreRequests(ctx)
	})
	if err != nil {
		return fmt.Errorf("add expire cron: %w", err)
	}

	s.cron.Start()
	s.logger.Info().Msg("tiering scheduler started")

	return nil
}

// Stop stops the tiering scheduler gracefully.
func (s *TieringScheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	s.running = false
	s.logger.Info().Msg("stopping tiering scheduler")

	return s.cron.Stop()
}

// ProcessTieringRules evaluates and applies tier transition rules for all organizations.
func (s *TieringScheduler) ProcessTieringRules(ctx context.Context) {
	s.logger.Info().Msg("processing tiering rules")

	orgs, err := s.store.GetAllOrganizations(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get organizations")
		return
	}

	for _, org := range orgs {
		select {
		case <-ctx.Done():
			s.logger.Warn().Msg("tiering processing cancelled")
			return
		default:
		}

		if err := s.processOrgTiering(ctx, org.ID); err != nil {
			s.logger.Error().
				Err(err).
				Str("org_id", org.ID.String()).
				Msg("failed to process org tiering")
		}
	}

	s.logger.Info().Msg("tiering rules processing completed")
}

// processOrgTiering processes tiering for a single organization.
func (s *TieringScheduler) processOrgTiering(ctx context.Context, orgID uuid.UUID) error {
	logger := s.logger.With().Str("org_id", orgID.String()).Logger()

	// Get enabled tier rules for this org, sorted by priority
	rules, err := s.store.GetEnabledTierRules(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get tier rules: %w", err)
	}

	if len(rules) == 0 {
		logger.Debug().Msg("no enabled tier rules, skipping")
		return nil
	}

	// Process each rule
	for _, rule := range rules {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := s.processRule(ctx, rule); err != nil {
			logger.Error().
				Err(err).
				Str("rule_id", rule.ID.String()).
				Str("rule_name", rule.Name).
				Msg("failed to process tier rule")
			continue
		}
	}

	return nil
}

// processRule applies a single tier rule.
func (s *TieringScheduler) processRule(ctx context.Context, rule *models.TierRule) error {
	logger := s.logger.With().
		Str("rule_id", rule.ID.String()).
		Str("rule_name", rule.Name).
		Str("from_tier", string(rule.FromTier)).
		Str("to_tier", string(rule.ToTier)).
		Int("age_threshold", rule.AgeThresholdDay).
		Logger()

	logger.Debug().Msg("processing tier rule")

	// Get snapshots that match this rule
	snapshots, err := s.store.GetSnapshotsForTiering(ctx, rule.OrgID, rule.FromTier, rule.AgeThresholdDay)
	if err != nil {
		return fmt.Errorf("get snapshots for tiering: %w", err)
	}

	if len(snapshots) == 0 {
		logger.Debug().Msg("no snapshots to tier")
		return nil
	}

	// Filter by repository/schedule if specified
	var filteredSnapshots []*models.SnapshotTier
	for _, snap := range snapshots {
		if rule.RepositoryID != nil && snap.RepositoryID != *rule.RepositoryID {
			continue
		}
		// Schedule filtering would require joining with backups table
		// For now, we process all matching snapshots
		filteredSnapshots = append(filteredSnapshots, snap)
	}

	logger.Info().
		Int("matched_snapshots", len(filteredSnapshots)).
		Msg("found snapshots for tiering")

	// Process in batches
	processed := 0
	for i := 0; i < len(filteredSnapshots) && processed < s.config.BatchSize; i++ {
		snap := filteredSnapshots[i]

		// Keep minimum copies in the source tier
		if rule.MinCopies > 0 {
			// For now, we assume each snapshot is unique
			// More sophisticated logic would count copies across repositories
		}

		// Create transition record
		transition := models.NewTierTransition(snap, rule.ToTier, &rule.ID, fmt.Sprintf("Rule: %s", rule.Name))

		// Calculate estimated savings
		transition.EstimatedSaving = s.calculateMonthlySavings(snap.SizeBytes, rule.FromTier, rule.ToTier)

		if err := s.store.CreateTierTransition(ctx, transition); err != nil {
			logger.Error().
				Err(err).
				Str("snapshot_id", snap.SnapshotID).
				Msg("failed to create tier transition")
			continue
		}

		// Execute the transition (or skip if dry run)
		if !s.config.DryRun {
			if err := s.executeTierTransition(ctx, snap, transition); err != nil {
				transition.Fail(err.Error())
				s.store.UpdateTierTransition(ctx, transition)
				logger.Error().
					Err(err).
					Str("snapshot_id", snap.SnapshotID).
					Msg("tier transition failed")
				continue
			}
		} else {
			logger.Info().
				Str("snapshot_id", snap.SnapshotID).
				Msg("dry run: would transition snapshot")
			transition.Complete()
		}

		if err := s.store.UpdateTierTransition(ctx, transition); err != nil {
			logger.Error().Err(err).Msg("failed to update tier transition")
		}

		processed++
	}

	logger.Info().
		Int("processed", processed).
		Msg("tier rule processing completed")

	return nil
}

// executeTierTransition performs the actual tier transition.
func (s *TieringScheduler) executeTierTransition(ctx context.Context, snap *models.SnapshotTier, transition *models.TierTransition) error {
	transition.Start()
	if err := s.store.UpdateTierTransition(ctx, transition); err != nil {
		return fmt.Errorf("update transition status: %w", err)
	}

	// Update the snapshot's tier
	snap.CurrentTier = transition.ToTier
	snap.TieredAt = time.Now()
	snap.UpdatedAt = time.Now()

	if err := s.store.UpdateSnapshotTier(ctx, snap); err != nil {
		return fmt.Errorf("update snapshot tier: %w", err)
	}

	transition.Complete()

	s.logger.Info().
		Str("snapshot_id", snap.SnapshotID).
		Str("from_tier", string(transition.FromTier)).
		Str("to_tier", string(transition.ToTier)).
		Float64("estimated_saving", transition.EstimatedSaving).
		Msg("tier transition completed")

	return nil
}

// calculateMonthlySavings estimates the monthly cost savings from a tier transition.
func (s *TieringScheduler) calculateMonthlySavings(sizeBytes int64, fromTier, toTier models.StorageTierType) float64 {
	// Default costs per GB per month (approximate S3-like pricing)
	costs := map[models.StorageTierType]float64{
		models.StorageTierHot:     0.023,
		models.StorageTierWarm:    0.0125,
		models.StorageTierCold:    0.004,
		models.StorageTierArchive: 0.00099,
	}

	sizeGB := float64(sizeBytes) / (1024 * 1024 * 1024)
	fromCost := sizeGB * costs[fromTier]
	toCost := sizeGB * costs[toTier]

	return fromCost - toCost
}

// GenerateCostReports generates cost optimization reports for all organizations.
func (s *TieringScheduler) GenerateCostReports(ctx context.Context) {
	s.logger.Info().Msg("generating cost reports")

	orgs, err := s.store.GetAllOrganizations(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get organizations")
		return
	}

	for _, org := range orgs {
		if err := s.generateOrgCostReport(ctx, org.ID); err != nil {
			s.logger.Error().
				Err(err).
				Str("org_id", org.ID.String()).
				Msg("failed to generate cost report")
		}
	}

	s.logger.Info().Msg("cost report generation completed")
}

// generateOrgCostReport generates a cost report for a single organization.
func (s *TieringScheduler) generateOrgCostReport(ctx context.Context, orgID uuid.UUID) error {
	logger := s.logger.With().Str("org_id", orgID.String()).Logger()

	// Get tier statistics
	stats, err := s.store.GetTierStatsSummary(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get tier stats: %w", err)
	}

	if stats.TotalSnapshots == 0 {
		logger.Debug().Msg("no snapshots, skipping cost report")
		return nil
	}

	report := models.NewTierCostReport(orgID)
	report.TotalSize = stats.TotalSizeBytes
	report.CurrentCost = stats.EstimatedMonthlyCost

	// Build tier breakdown
	for tierType, tierStats := range stats.ByTier {
		item := models.TierBreakdownItem{
			TierType:       tierType,
			SnapshotCount:  tierStats.SnapshotCount,
			TotalSizeBytes: tierStats.TotalSizeBytes,
			MonthlyCost:    tierStats.MonthlyCost,
		}
		if stats.TotalSizeBytes > 0 {
			item.Percentage = float64(tierStats.TotalSizeBytes) / float64(stats.TotalSizeBytes) * 100
		}
		report.TierBreakdown = append(report.TierBreakdown, item)
	}

	// Generate optimization suggestions
	// Look for snapshots in hot tier that are old enough for warm
	if hotStats, ok := stats.ByTier[models.StorageTierHot]; ok && hotStats.OldestDays > 30 {
		snapshots, err := s.store.GetSnapshotsForTiering(ctx, orgID, models.StorageTierHot, 30)
		if err == nil {
			for _, snap := range snapshots {
				if len(report.Suggestions) >= 10 {
					break // Limit suggestions
				}
				savings := s.calculateMonthlySavings(snap.SizeBytes, models.StorageTierHot, models.StorageTierWarm)
				report.Suggestions = append(report.Suggestions, models.TierOptSuggestion{
					SnapshotID:     snap.SnapshotID,
					RepositoryID:   snap.RepositoryID,
					CurrentTier:    models.StorageTierHot,
					SuggestedTier:  models.StorageTierWarm,
					AgeDays:        snap.AgeDays(),
					SizeBytes:      snap.SizeBytes,
					MonthlySavings: savings,
					Reason:         fmt.Sprintf("Snapshot is %d days old, consider moving to warm storage", snap.AgeDays()),
				})
			}
		}
	}

	// Calculate optimized cost (if all suggestions were applied)
	report.OptimizedCost = report.CurrentCost
	for _, suggestion := range report.Suggestions {
		report.OptimizedCost -= suggestion.MonthlySavings
	}
	report.PotentialSave = report.CurrentCost - report.OptimizedCost

	if err := s.store.CreateTierCostReport(ctx, report); err != nil {
		return fmt.Errorf("create cost report: %w", err)
	}

	logger.Info().
		Float64("current_cost", report.CurrentCost).
		Float64("potential_savings", report.PotentialSave).
		Int("suggestions", len(report.Suggestions)).
		Msg("cost report generated")

	return nil
}

// ProcessColdRestoreRequests processes pending cold/archive restore requests.
func (s *TieringScheduler) ProcessColdRestoreRequests(ctx context.Context) {
	s.logger.Debug().Msg("processing cold restore requests")

	orgs, err := s.store.GetAllOrganizations(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get organizations")
		return
	}

	for _, org := range orgs {
		requests, err := s.store.GetPendingColdRestoreRequests(ctx, org.ID)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("org_id", org.ID.String()).
				Msg("failed to get pending cold restore requests")
			continue
		}

		for _, req := range requests {
			s.processColdRestoreRequest(ctx, req)
		}
	}
}

// processColdRestoreRequest processes a single cold restore request.
func (s *TieringScheduler) processColdRestoreRequest(ctx context.Context, req *models.ColdRestoreRequest) {
	logger := s.logger.With().
		Str("request_id", req.ID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("from_tier", string(req.FromTier)).
		Logger()

	switch req.Status {
	case "pending":
		// Start warming process
		estimatedReady := time.Now()
		switch req.FromTier {
		case models.StorageTierCold:
			if req.Priority == "expedited" {
				estimatedReady = estimatedReady.Add(1 * time.Hour)
			} else {
				estimatedReady = estimatedReady.Add(5 * time.Hour)
			}
		case models.StorageTierArchive:
			if req.Priority == "expedited" {
				estimatedReady = estimatedReady.Add(3 * time.Hour)
			} else {
				estimatedReady = estimatedReady.Add(12 * time.Hour)
			}
		}

		req.MarkWarming(estimatedReady)
		if err := s.store.UpdateColdRestoreRequest(ctx, req); err != nil {
			logger.Error().Err(err).Msg("failed to update cold restore request")
			return
		}
		logger.Info().
			Time("estimated_ready", estimatedReady).
			Msg("cold restore warming started")

	case "warming":
		// Check if enough time has passed (simulated warming)
		if req.EstimatedReady != nil && time.Now().After(*req.EstimatedReady) {
			// Data is ready, set expiration (24 hours for restored data)
			expiresAt := time.Now().Add(24 * time.Hour)
			req.MarkReady(expiresAt)
			if err := s.store.UpdateColdRestoreRequest(ctx, req); err != nil {
				logger.Error().Err(err).Msg("failed to update cold restore request")
				return
			}
			logger.Info().
				Time("expires_at", expiresAt).
				Msg("cold restore data ready")
		}
	}
}

// ExpireColdRestoreRequests marks expired cold restore requests.
func (s *TieringScheduler) ExpireColdRestoreRequests(ctx context.Context) {
	count, err := s.store.ExpireColdRestoreRequests(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to expire cold restore requests")
		return
	}
	if count > 0 {
		s.logger.Info().Int("expired_count", count).Msg("expired cold restore requests")
	}
}

// InitializeSnapshotTier creates a tier record for a newly created snapshot.
func (s *TieringScheduler) InitializeSnapshotTier(ctx context.Context, snapshotID string, repositoryID, orgID uuid.UUID, sizeBytes int64, snapshotTime time.Time) error {
	tier := models.NewSnapshotTier(snapshotID, repositoryID, orgID, sizeBytes, snapshotTime)
	return s.store.CreateSnapshotTier(ctx, tier)
}

// RequestColdRestore initiates a restore request for cold/archive data.
func (s *TieringScheduler) RequestColdRestore(ctx context.Context, orgID uuid.UUID, snapshotID string, repositoryID, requestedBy uuid.UUID, priority string) (*models.ColdRestoreRequest, error) {
	// Check if snapshot is in cold or archive tier
	tier, err := s.store.GetSnapshotTier(ctx, snapshotID, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("get snapshot tier: %w", err)
	}

	if tier.CurrentTier != models.StorageTierCold && tier.CurrentTier != models.StorageTierArchive {
		return nil, errors.New("snapshot is not in cold or archive tier")
	}

	// Check for existing active request
	existing, err := s.store.GetColdRestoreRequestBySnapshot(ctx, snapshotID, repositoryID)
	if err == nil && existing != nil {
		if existing.Status == "warming" || existing.Status == "ready" {
			return existing, nil // Return existing request
		}
	}

	// Create new request
	req := models.NewColdRestoreRequest(orgID, snapshotID, repositoryID, requestedBy, tier.CurrentTier)
	if priority != "" {
		req.Priority = priority
	}

	// Calculate retrieval cost
	tierConfigs, err := s.store.GetStorageTierConfigs(ctx, orgID)
	if err == nil {
		for _, config := range tierConfigs {
			if config.TierType == tier.CurrentTier {
				sizeGB := float64(tier.SizeBytes) / (1024 * 1024 * 1024)
				req.RetrievalCost = sizeGB * config.RetrievalCost
				break
			}
		}
	}

	if err := s.store.CreateColdRestoreRequest(ctx, req); err != nil {
		return nil, fmt.Errorf("create cold restore request: %w", err)
	}

	s.logger.Info().
		Str("request_id", req.ID.String()).
		Str("snapshot_id", snapshotID).
		Str("priority", req.Priority).
		Float64("retrieval_cost", req.RetrievalCost).
		Msg("cold restore request created")

	return req, nil
}

// GetRestoreStatus returns the status of a cold restore request.
func (s *TieringScheduler) GetRestoreStatus(ctx context.Context, snapshotID string, repositoryID uuid.UUID) (*models.ColdRestoreRequest, error) {
	return s.store.GetColdRestoreRequestBySnapshot(ctx, snapshotID, repositoryID)
}

// ManualTierChange allows manual tier transition for a snapshot.
func (s *TieringScheduler) ManualTierChange(ctx context.Context, snapshotID string, repositoryID uuid.UUID, toTier models.StorageTierType, reason string) error {
	tier, err := s.store.GetSnapshotTier(ctx, snapshotID, repositoryID)
	if err != nil {
		return fmt.Errorf("get snapshot tier: %w", err)
	}

	if tier.CurrentTier == toTier {
		return errors.New("snapshot is already in the target tier")
	}

	// Create transition record
	transition := models.NewTierTransition(tier, toTier, nil, reason)
	transition.EstimatedSaving = s.calculateMonthlySavings(tier.SizeBytes, tier.CurrentTier, toTier)

	if err := s.store.CreateTierTransition(ctx, transition); err != nil {
		return fmt.Errorf("create tier transition: %w", err)
	}

	// Execute transition
	if err := s.executeTierTransition(ctx, tier, transition); err != nil {
		transition.Fail(err.Error())
		s.store.UpdateTierTransition(ctx, transition)
		return fmt.Errorf("execute tier transition: %w", err)
	}

	if err := s.store.UpdateTierTransition(ctx, transition); err != nil {
		return fmt.Errorf("update tier transition: %w", err)
	}

	return nil
}

// TriggerProcessing manually triggers tiering processing (for testing/admin).
func (s *TieringScheduler) TriggerProcessing(ctx context.Context) {
	go s.ProcessTieringRules(ctx)
}

// TriggerCostReport manually triggers cost report generation.
func (s *TieringScheduler) TriggerCostReport(ctx context.Context) {
	go s.GenerateCostReports(ctx)
}
