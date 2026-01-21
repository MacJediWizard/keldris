package reports

import (
	"context"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ReportStore defines the interface for data access needed by the generator.
type ReportStore interface {
	GetBackupsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Backup, error)
	GetEnabledSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error)
	GetStorageStatsSummary(ctx context.Context, orgID uuid.UUID) (*models.StorageStatsSummary, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetAlertsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Alert, error)
}

// Generator generates report data.
type Generator struct {
	store  ReportStore
	logger zerolog.Logger
}

// NewGenerator creates a new report generator.
func NewGenerator(store ReportStore, logger zerolog.Logger) *Generator {
	return &Generator{
		store:  store,
		logger: logger.With().Str("component", "report_generator").Logger(),
	}
}

// GenerateReport generates a complete report for the given time period.
func (g *Generator) GenerateReport(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) (*models.ReportData, error) {
	g.logger.Debug().
		Str("org_id", orgID.String()).
		Time("period_start", periodStart).
		Time("period_end", periodEnd).
		Msg("generating report")

	report := &models.ReportData{}

	// Generate backup summary
	backupSummary, err := g.generateBackupSummary(ctx, orgID, periodStart, periodEnd)
	if err != nil {
		g.logger.Error().Err(err).Msg("failed to generate backup summary")
	} else {
		report.BackupSummary = *backupSummary
	}

	// Generate storage summary
	storageSummary, err := g.generateStorageSummary(ctx, orgID)
	if err != nil {
		g.logger.Error().Err(err).Msg("failed to generate storage summary")
	} else {
		report.StorageSummary = *storageSummary
	}

	// Generate agent summary
	agentSummary, err := g.generateAgentSummary(ctx, orgID)
	if err != nil {
		g.logger.Error().Err(err).Msg("failed to generate agent summary")
	} else {
		report.AgentSummary = *agentSummary
	}

	// Generate alert summary
	alertSummary, topIssues, err := g.generateAlertSummary(ctx, orgID, periodStart, periodEnd)
	if err != nil {
		g.logger.Error().Err(err).Msg("failed to generate alert summary")
	} else {
		report.AlertSummary = *alertSummary
		report.TopIssues = topIssues
	}

	return report, nil
}

func (g *Generator) generateBackupSummary(ctx context.Context, orgID uuid.UUID, start, end time.Time) (*models.BackupSummary, error) {
	backups, err := g.store.GetBackupsByOrgIDAndDateRange(ctx, orgID, start, end)
	if err != nil {
		return nil, err
	}

	schedules, err := g.store.GetEnabledSchedulesByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	summary := &models.BackupSummary{
		TotalBackups:    len(backups),
		SchedulesActive: len(schedules),
	}

	var totalData int64
	for _, b := range backups {
		if b.Status == models.BackupStatusCompleted {
			summary.SuccessfulBackups++
			if b.SizeBytes != nil {
				totalData += *b.SizeBytes
			}
		} else if b.Status == models.BackupStatusFailed {
			summary.FailedBackups++
		}
	}

	summary.TotalDataBacked = totalData
	if summary.TotalBackups > 0 {
		summary.SuccessRate = float64(summary.SuccessfulBackups) / float64(summary.TotalBackups) * 100
	}

	return summary, nil
}

func (g *Generator) generateStorageSummary(ctx context.Context, orgID uuid.UUID) (*models.StorageSummary, error) {
	stats, err := g.store.GetStorageStatsSummary(ctx, orgID)
	if err != nil {
		return nil, err
	}

	var spaceSavedPct float64
	if stats.TotalRestoreSize > 0 {
		spaceSavedPct = float64(stats.TotalSpaceSaved) / float64(stats.TotalRestoreSize) * 100
	}

	return &models.StorageSummary{
		TotalRawSize:     stats.TotalRawSize,
		TotalRestoreSize: stats.TotalRestoreSize,
		SpaceSaved:       stats.TotalSpaceSaved,
		SpaceSavedPct:    spaceSavedPct,
		RepositoryCount:  stats.RepositoryCount,
		TotalSnapshots:   stats.TotalSnapshots,
	}, nil
}

func (g *Generator) generateAgentSummary(ctx context.Context, orgID uuid.UUID) (*models.AgentSummary, error) {
	agents, err := g.store.GetAgentsByOrgID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	summary := &models.AgentSummary{
		TotalAgents: len(agents),
	}

	for _, a := range agents {
		switch a.Status {
		case models.AgentStatusActive:
			summary.ActiveAgents++
		case models.AgentStatusOffline:
			summary.OfflineAgents++
		case models.AgentStatusPending:
			summary.PendingAgents++
		}
	}

	return summary, nil
}

func (g *Generator) generateAlertSummary(ctx context.Context, orgID uuid.UUID, start, end time.Time) (*models.AlertSummary, []models.ReportIssue, error) {
	alerts, err := g.store.GetAlertsByOrgIDAndDateRange(ctx, orgID, start, end)
	if err != nil {
		return nil, nil, err
	}

	summary := &models.AlertSummary{
		TotalAlerts: len(alerts),
	}

	var issues []models.ReportIssue
	for _, a := range alerts {
		switch a.Severity {
		case models.AlertSeverityCritical:
			summary.CriticalAlerts++
		case models.AlertSeverityWarning:
			summary.WarningAlerts++
		}

		switch a.Status {
		case models.AlertStatusAcknowledged:
			summary.AcknowledgedAlerts++
		case models.AlertStatusResolved:
			summary.ResolvedAlerts++
		}

		// Collect top issues (critical alerts)
		if a.Severity == models.AlertSeverityCritical && len(issues) < 5 {
			issues = append(issues, models.ReportIssue{
				Type:        string(a.Type),
				Severity:    string(a.Severity),
				Title:       a.Title,
				Description: a.Message,
				OccurredAt:  a.CreatedAt,
			})
		}
	}

	return summary, issues, nil
}

// CalculatePeriod calculates the start and end time for a given frequency.
func CalculatePeriod(frequency models.ReportFrequency, timezone string) (start, end time.Time) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)

	switch frequency {
	case models.ReportFrequencyDaily:
		// Previous 24 hours
		end = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
		start = end.AddDate(0, 0, -1)
	case models.ReportFrequencyWeekly:
		// Previous week (Monday to Sunday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		end = time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, loc)
		start = end.AddDate(0, 0, -7)
	case models.ReportFrequencyMonthly:
		// Previous month
		end = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		start = end.AddDate(0, -1, 0)
	default:
		end = now
		start = now.AddDate(0, 0, -7)
	}

	return start, end.Add(-time.Nanosecond)
}
