// Package metering provides usage metering for limits and billing tracking.
package metering

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Store defines the database operations needed by the metering service.
type Store interface {
	// Organization operations
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)

	// Agent counts
	GetAgentCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	GetActiveAgentCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)

	// User counts
	GetUserCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	GetActiveUserCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)

	// Storage counts
	GetTotalStorageByOrgID(ctx context.Context, orgID uuid.UUID) (int64, error)

	// Backup counts
	GetBackupCountByOrgIDForPeriod(ctx context.Context, orgID uuid.UUID, start, end time.Time) (completed, failed int, err error)
	GetBackupsThisMonthByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)

	// Repository counts
	GetRepositoryCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)

	// Schedule counts
	GetScheduleCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)

	// Snapshot counts
	GetSnapshotCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)

	// Usage metrics CRUD
	CreateUsageMetrics(ctx context.Context, metrics *models.UsageMetrics) error
	UpsertUsageMetrics(ctx context.Context, metrics *models.UsageMetrics) error
	GetUsageMetricsByOrgID(ctx context.Context, orgID uuid.UUID, startDate, endDate time.Time) ([]*models.UsageMetrics, error)
	GetLatestUsageMetrics(ctx context.Context, orgID uuid.UUID) (*models.UsageMetrics, error)

	// Usage limits CRUD
	GetOrgUsageLimits(ctx context.Context, orgID uuid.UUID) (*models.OrgUsageLimits, error)
	CreateOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error
	UpdateOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error
	UpsertOrgUsageLimits(ctx context.Context, limits *models.OrgUsageLimits) error

	// Usage alerts CRUD
	CreateUsageAlert(ctx context.Context, alert *models.UsageAlert) error
	GetActiveUsageAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.UsageAlert, error)
	GetActiveUsageAlertByType(ctx context.Context, orgID uuid.UUID, alertType models.UsageAlertType) (*models.UsageAlert, error)
	UpdateUsageAlert(ctx context.Context, alert *models.UsageAlert) error
	AcknowledgeUsageAlert(ctx context.Context, id, userID uuid.UUID) error
	ResolveUsageAlert(ctx context.Context, id uuid.UUID) error

	// Monthly summary CRUD
	UpsertMonthlyUsageSummary(ctx context.Context, summary *models.MonthlyUsageSummary) error
	GetMonthlyUsageSummary(ctx context.Context, orgID uuid.UUID, yearMonth string) (*models.MonthlyUsageSummary, error)
	GetMonthlyUsageSummariesByOrgID(ctx context.Context, orgID uuid.UUID, months int) ([]*models.MonthlyUsageSummary, error)
}

// Config holds configuration for the metering service.
type Config struct {
	// SnapshotInterval is how often to take usage snapshots.
	SnapshotInterval time.Duration
	// AlertCheckInterval is how often to check for limit alerts.
	AlertCheckInterval time.Duration
	// AggregationInterval is how often to aggregate monthly summaries.
	AggregationInterval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		SnapshotInterval:    1 * time.Hour,
		AlertCheckInterval:  15 * time.Minute,
		AggregationInterval: 24 * time.Hour,
	}
}

// Service manages usage metering and billing tracking.
type Service struct {
	store  Store
	config Config
	logger zerolog.Logger
	stopCh chan struct{}
}

// NewService creates a new metering Service.
func NewService(store Store, config Config, logger zerolog.Logger) *Service {
	return &Service{
		store:  store,
		config: config,
		logger: logger.With().Str("component", "metering_service").Logger(),
		stopCh: make(chan struct{}),
	}
}

// Start begins the background metering tasks.
func (s *Service) Start(ctx context.Context) {
	s.logger.Info().Msg("starting metering service")

	go s.runSnapshots(ctx)
	go s.runAlertChecks(ctx)
	go s.runAggregation(ctx)
}

// Stop stops the metering service.
func (s *Service) Stop() {
	s.logger.Info().Msg("stopping metering service")
	close(s.stopCh)
}

// runSnapshots periodically takes usage snapshots.
func (s *Service) runSnapshots(ctx context.Context) {
	ticker := time.NewTicker(s.config.SnapshotInterval)
	defer ticker.Stop()

	// Take initial snapshot
	s.takeAllSnapshots(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.takeAllSnapshots(ctx)
		}
	}
}

// runAlertChecks periodically checks for limit alerts.
func (s *Service) runAlertChecks(ctx context.Context) {
	ticker := time.NewTicker(s.config.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkAllLimits(ctx)
		}
	}
}

// runAggregation periodically aggregates monthly summaries.
func (s *Service) runAggregation(ctx context.Context) {
	ticker := time.NewTicker(s.config.AggregationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.aggregateAllMonthlySummaries(ctx)
		}
	}
}

// takeAllSnapshots takes usage snapshots for all organizations.
func (s *Service) takeAllSnapshots(ctx context.Context) {
	orgs, err := s.store.GetAllOrganizations(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get organizations for snapshots")
		return
	}

	for _, org := range orgs {
		if err := s.TakeSnapshot(ctx, org.ID); err != nil {
			s.logger.Error().Err(err).Str("org_id", org.ID.String()).Msg("failed to take snapshot")
		}
	}
}

// TakeSnapshot takes a usage snapshot for an organization.
func (s *Service) TakeSnapshot(ctx context.Context, orgID uuid.UUID) error {
	today := time.Now().Truncate(24 * time.Hour)
	metrics := models.NewUsageMetrics(orgID, today)

	// Collect agent counts
	agentCount, err := s.store.GetAgentCountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get agent count: %w", err)
	}
	metrics.AgentCount = agentCount

	activeAgentCount, err := s.store.GetActiveAgentCountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get active agent count: %w", err)
	}
	metrics.ActiveAgentCount = activeAgentCount

	// Collect user counts
	userCount, err := s.store.GetUserCountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get user count: %w", err)
	}
	metrics.UserCount = userCount

	activeUserCount, err := s.store.GetActiveUserCountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get active user count: %w", err)
	}
	metrics.ActiveUserCount = activeUserCount

	// Collect storage
	storage, err := s.store.GetTotalStorageByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get storage: %w", err)
	}
	metrics.TotalStorageBytes = storage
	metrics.BackupStorageBytes = storage

	// Collect today's backup counts
	startOfDay := today
	endOfDay := today.Add(24 * time.Hour)
	completed, failed, err := s.store.GetBackupCountByOrgIDForPeriod(ctx, orgID, startOfDay, endOfDay)
	if err != nil {
		return fmt.Errorf("get backup counts: %w", err)
	}
	metrics.BackupsCompleted = completed
	metrics.BackupsFailed = failed
	metrics.BackupsTotal = completed + failed

	// Collect repository count
	repoCount, err := s.store.GetRepositoryCountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get repository count: %w", err)
	}
	metrics.RepositoryCount = repoCount

	// Collect schedule count
	scheduleCount, err := s.store.GetScheduleCountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get schedule count: %w", err)
	}
	metrics.ScheduleCount = scheduleCount

	// Collect snapshot count
	snapshotCount, err := s.store.GetSnapshotCountByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get snapshot count: %w", err)
	}
	metrics.SnapshotCount = snapshotCount

	// Upsert the metrics
	if err := s.store.UpsertUsageMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("upsert usage metrics: %w", err)
	}

	s.logger.Debug().
		Str("org_id", orgID.String()).
		Int("agents", agentCount).
		Int("users", userCount).
		Int64("storage_bytes", storage).
		Msg("usage snapshot recorded")

	return nil
}

// checkAllLimits checks usage limits for all organizations.
func (s *Service) checkAllLimits(ctx context.Context) {
	orgs, err := s.store.GetAllOrganizations(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get organizations for limit checks")
		return
	}

	for _, org := range orgs {
		if err := s.CheckLimits(ctx, org.ID); err != nil {
			s.logger.Error().Err(err).Str("org_id", org.ID.String()).Msg("failed to check limits")
		}
	}
}

// CheckLimits checks usage limits for an organization and creates/resolves alerts.
func (s *Service) CheckLimits(ctx context.Context, orgID uuid.UUID) error {
	limits, err := s.store.GetOrgUsageLimits(ctx, orgID)
	if err != nil {
		// No limits configured, skip
		return nil
	}

	// Check agent limits
	if limits.MaxAgents != nil {
		agentCount, err := s.store.GetAgentCountByOrgID(ctx, orgID)
		if err == nil {
			s.checkResourceLimit(ctx, orgID, models.UsageAlertTypeAgents, int64(agentCount), int64(*limits.MaxAgents), limits.WarningThreshold, limits.CriticalThreshold)
		}
	}

	// Check user limits
	if limits.MaxUsers != nil {
		userCount, err := s.store.GetUserCountByOrgID(ctx, orgID)
		if err == nil {
			s.checkResourceLimit(ctx, orgID, models.UsageAlertTypeUsers, int64(userCount), int64(*limits.MaxUsers), limits.WarningThreshold, limits.CriticalThreshold)
		}
	}

	// Check storage limits
	if limits.MaxStorageBytes != nil {
		storage, err := s.store.GetTotalStorageByOrgID(ctx, orgID)
		if err == nil {
			s.checkResourceLimit(ctx, orgID, models.UsageAlertTypeStorage, storage, *limits.MaxStorageBytes, limits.WarningThreshold, limits.CriticalThreshold)
		}
	}

	// Check backup limits
	if limits.MaxBackupsPerMonth != nil {
		backups, err := s.store.GetBackupsThisMonthByOrgID(ctx, orgID)
		if err == nil {
			s.checkResourceLimit(ctx, orgID, models.UsageAlertTypeBackups, int64(backups), int64(*limits.MaxBackupsPerMonth), limits.WarningThreshold, limits.CriticalThreshold)
		}
	}

	// Check repository limits
	if limits.MaxRepositories != nil {
		repoCount, err := s.store.GetRepositoryCountByOrgID(ctx, orgID)
		if err == nil {
			s.checkResourceLimit(ctx, orgID, models.UsageAlertTypeRepositories, int64(repoCount), int64(*limits.MaxRepositories), limits.WarningThreshold, limits.CriticalThreshold)
		}
	}

	return nil
}

// checkResourceLimit checks a single resource against its limit and creates/resolves alerts.
func (s *Service) checkResourceLimit(ctx context.Context, orgID uuid.UUID, alertType models.UsageAlertType, current, limit int64, warningThreshold, criticalThreshold int) {
	if limit <= 0 {
		return
	}

	percentUsed := float64(current) / float64(limit) * 100

	// Determine severity
	var severity models.UsageAlertSeverity
	var shouldAlert bool

	if percentUsed >= 100 {
		severity = models.UsageAlertSeverityExceeded
		shouldAlert = true
	} else if percentUsed >= float64(criticalThreshold) {
		severity = models.UsageAlertSeverityCritical
		shouldAlert = true
	} else if percentUsed >= float64(warningThreshold) {
		severity = models.UsageAlertSeverityWarning
		shouldAlert = true
	}

	// Check for existing active alert
	existingAlert, err := s.store.GetActiveUsageAlertByType(ctx, orgID, alertType)
	if err != nil && existingAlert == nil {
		// No existing alert
		if shouldAlert {
			// Create new alert
			message := s.formatAlertMessage(alertType, severity, current, limit, percentUsed)
			alert := models.NewUsageAlert(orgID, alertType, severity, current, limit, message)
			if err := s.store.CreateUsageAlert(ctx, alert); err != nil {
				s.logger.Error().Err(err).Str("org_id", orgID.String()).Str("type", string(alertType)).Msg("failed to create usage alert")
			} else {
				s.logger.Info().
					Str("org_id", orgID.String()).
					Str("type", string(alertType)).
					Str("severity", string(severity)).
					Float64("percent_used", percentUsed).
					Msg("usage alert created")
			}
		}
	} else if existingAlert != nil {
		if !shouldAlert {
			// Resolve the alert
			if err := s.store.ResolveUsageAlert(ctx, existingAlert.ID); err != nil {
				s.logger.Error().Err(err).Str("alert_id", existingAlert.ID.String()).Msg("failed to resolve usage alert")
			} else {
				s.logger.Info().
					Str("org_id", orgID.String()).
					Str("type", string(alertType)).
					Msg("usage alert resolved")
			}
		} else if severity != existingAlert.Severity {
			// Update severity if changed
			existingAlert.Severity = severity
			existingAlert.CurrentValue = current
			existingAlert.PercentageUsed = percentUsed
			existingAlert.Message = s.formatAlertMessage(alertType, severity, current, limit, percentUsed)
			if err := s.store.UpdateUsageAlert(ctx, existingAlert); err != nil {
				s.logger.Error().Err(err).Str("alert_id", existingAlert.ID.String()).Msg("failed to update usage alert")
			}
		}
	}
}

// formatAlertMessage creates a human-readable alert message.
func (s *Service) formatAlertMessage(alertType models.UsageAlertType, severity models.UsageAlertSeverity, current, limit int64, percentUsed float64) string {
	typeStr := map[models.UsageAlertType]string{
		models.UsageAlertTypeAgents:       "agents",
		models.UsageAlertTypeUsers:        "users",
		models.UsageAlertTypeStorage:      "storage",
		models.UsageAlertTypeBackups:      "monthly backups",
		models.UsageAlertTypeRepositories: "repositories",
	}[alertType]

	switch severity {
	case models.UsageAlertSeverityExceeded:
		return fmt.Sprintf("Usage limit exceeded: %d/%d %s (%.1f%%)", current, limit, typeStr, percentUsed)
	case models.UsageAlertSeverityCritical:
		return fmt.Sprintf("Critical: approaching %s limit - %d/%d (%.1f%%)", typeStr, current, limit, percentUsed)
	default:
		return fmt.Sprintf("Warning: approaching %s limit - %d/%d (%.1f%%)", typeStr, current, limit, percentUsed)
	}
}

// aggregateAllMonthlySummaries aggregates monthly summaries for all organizations.
func (s *Service) aggregateAllMonthlySummaries(ctx context.Context) {
	orgs, err := s.store.GetAllOrganizations(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get organizations for aggregation")
		return
	}

	yearMonth := time.Now().Format("2006-01")
	for _, org := range orgs {
		if err := s.AggregateMonthly(ctx, org.ID, yearMonth); err != nil {
			s.logger.Error().Err(err).Str("org_id", org.ID.String()).Msg("failed to aggregate monthly summary")
		}
	}
}

// AggregateMonthly aggregates daily metrics into a monthly summary.
func (s *Service) AggregateMonthly(ctx context.Context, orgID uuid.UUID, yearMonth string) error {
	// Parse year-month to get date range
	t, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return fmt.Errorf("parse year-month: %w", err)
	}

	startDate := t
	endDate := t.AddDate(0, 1, 0)

	// Get daily metrics for the month
	metrics, err := s.store.GetUsageMetricsByOrgID(ctx, orgID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("get usage metrics: %w", err)
	}

	if len(metrics) == 0 {
		return nil
	}

	summary := models.NewMonthlyUsageSummary(orgID, yearMonth)

	var totalAgentCount, totalStorageBytes int64
	for _, m := range metrics {
		// Track peaks
		if m.AgentCount > summary.PeakAgentCount {
			summary.PeakAgentCount = m.AgentCount
		}
		if m.UserCount > summary.PeakUserCount {
			summary.PeakUserCount = m.UserCount
		}
		if m.TotalStorageBytes > summary.PeakStorageBytes {
			summary.PeakStorageBytes = m.TotalStorageBytes
		}

		// Accumulate totals
		summary.TotalBackupsCompleted += m.BackupsCompleted
		summary.TotalBackupsFailed += m.BackupsFailed

		// For averages
		totalAgentCount += int64(m.AgentCount)
		totalStorageBytes += m.TotalStorageBytes
	}

	// Calculate averages
	summary.AvgAgentCount = float64(totalAgentCount) / float64(len(metrics))
	summary.AvgStorageBytes = totalStorageBytes / int64(len(metrics))

	// Calculate billable units (simplified - actual billing logic may vary)
	summary.BillableAgentHours = summary.PeakAgentCount * 24 * len(metrics)
	summary.BillableStorageGBHours = (summary.PeakStorageBytes / (1024 * 1024 * 1024)) * 24 * int64(len(metrics))

	if err := s.store.UpsertMonthlyUsageSummary(ctx, summary); err != nil {
		return fmt.Errorf("upsert monthly summary: %w", err)
	}

	s.logger.Debug().
		Str("org_id", orgID.String()).
		Str("year_month", yearMonth).
		Int("peak_agents", summary.PeakAgentCount).
		Int64("peak_storage", summary.PeakStorageBytes).
		Msg("monthly usage summary aggregated")

	return nil
}

// GetCurrentUsage returns the current usage state for an organization.
func (s *Service) GetCurrentUsage(ctx context.Context, orgID uuid.UUID) (*models.CurrentUsage, error) {
	usage := &models.CurrentUsage{
		OrgID: orgID,
	}

	// Get current counts
	agentCount, err := s.store.GetAgentCountByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get agent count: %w", err)
	}
	usage.AgentCount = agentCount

	activeAgentCount, err := s.store.GetActiveAgentCountByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get active agent count: %w", err)
	}
	usage.ActiveAgentCount = activeAgentCount

	userCount, err := s.store.GetUserCountByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get user count: %w", err)
	}
	usage.UserCount = userCount

	storage, err := s.store.GetTotalStorageByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get storage: %w", err)
	}
	usage.StorageBytes = storage

	repoCount, err := s.store.GetRepositoryCountByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get repository count: %w", err)
	}
	usage.RepositoryCount = repoCount

	backups, err := s.store.GetBackupsThisMonthByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get backups this month: %w", err)
	}
	usage.BackupsThisMonth = backups

	// Get limits
	limits, err := s.store.GetOrgUsageLimits(ctx, orgID)
	if err == nil && limits != nil {
		usage.AgentLimit = limits.MaxAgents
		usage.UserLimit = limits.MaxUsers
		usage.StorageLimit = limits.MaxStorageBytes
		usage.RepositoryLimit = limits.MaxRepositories
		usage.BackupLimit = limits.MaxBackupsPerMonth
		usage.BillingTier = limits.BillingTier
		usage.BillingPeriodStart = limits.BillingPeriodStart
		usage.BillingPeriodEnd = limits.BillingPeriodEnd

		// Calculate percentages
		if limits.MaxAgents != nil && *limits.MaxAgents > 0 {
			pct := float64(agentCount) / float64(*limits.MaxAgents) * 100
			usage.AgentUsagePercent = &pct
		}
		if limits.MaxUsers != nil && *limits.MaxUsers > 0 {
			pct := float64(userCount) / float64(*limits.MaxUsers) * 100
			usage.UserUsagePercent = &pct
		}
		if limits.MaxStorageBytes != nil && *limits.MaxStorageBytes > 0 {
			pct := float64(storage) / float64(*limits.MaxStorageBytes) * 100
			usage.StorageUsagePercent = &pct
		}
		if limits.MaxRepositories != nil && *limits.MaxRepositories > 0 {
			pct := float64(repoCount) / float64(*limits.MaxRepositories) * 100
			usage.RepositoryUsagePercent = &pct
		}
		if limits.MaxBackupsPerMonth != nil && *limits.MaxBackupsPerMonth > 0 {
			pct := float64(backups) / float64(*limits.MaxBackupsPerMonth) * 100
			usage.BackupUsagePercent = &pct
		}
	}

	// Get active alerts
	alerts, err := s.store.GetActiveUsageAlertsByOrgID(ctx, orgID)
	if err == nil && len(alerts) > 0 {
		usage.ActiveAlerts = make([]models.UsageAlert, len(alerts))
		for i, a := range alerts {
			usage.ActiveAlerts[i] = *a
		}
	}

	return usage, nil
}

// GetUsageHistory returns usage history for charts.
func (s *Service) GetUsageHistory(ctx context.Context, orgID uuid.UUID, days int) ([]models.UsageHistoryPoint, error) {
	endDate := time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
	startDate := endDate.AddDate(0, 0, -days)

	metrics, err := s.store.GetUsageMetricsByOrgID(ctx, orgID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get usage metrics: %w", err)
	}

	points := make([]models.UsageHistoryPoint, len(metrics))
	for i, m := range metrics {
		points[i] = models.UsageHistoryPoint{
			Date:             m.SnapshotDate,
			AgentCount:       m.AgentCount,
			UserCount:        m.UserCount,
			StorageBytes:     m.TotalStorageBytes,
			BackupsCompleted: m.BackupsCompleted,
			BackupsFailed:    m.BackupsFailed,
		}
	}

	return points, nil
}

// GetBillingReport generates a billing report for an organization.
func (s *Service) GetBillingReport(ctx context.Context, orgID uuid.UUID, yearMonth string) (*models.BillingUsageReport, error) {
	// Get organization
	org, err := s.store.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}

	// Get monthly summary
	summary, err := s.store.GetMonthlyUsageSummary(ctx, orgID, yearMonth)
	if err != nil {
		return nil, fmt.Errorf("get monthly summary: %w", err)
	}

	// Get limits for billing tier
	limits, err := s.store.GetOrgUsageLimits(ctx, orgID)
	billingTier := "free"
	if err == nil && limits != nil {
		billingTier = limits.BillingTier
	}

	// Parse period dates
	t, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return nil, fmt.Errorf("parse year-month: %w", err)
	}

	report := &models.BillingUsageReport{
		OrgID:                  orgID,
		OrgName:                org.Name,
		BillingTier:            billingTier,
		PeriodStart:            t,
		PeriodEnd:              t.AddDate(0, 1, 0).Add(-time.Second),
		PeakAgentCount:         summary.PeakAgentCount,
		PeakUserCount:          summary.PeakUserCount,
		PeakStorageBytes:       summary.PeakStorageBytes,
		TotalBackups:           summary.TotalBackupsCompleted,
		AvgAgentCount:          summary.AvgAgentCount,
		AvgStorageBytes:        summary.AvgStorageBytes,
		BillableAgentHours:     summary.BillableAgentHours,
		BillableStorageGBHours: summary.BillableStorageGBHours,
	}

	return report, nil
}
