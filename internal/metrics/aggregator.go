package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AggregatorStore defines the interface for aggregator persistence operations.
type AggregatorStore interface {
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GetBackupsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Backup, error)
	CreateOrUpdateDailySummary(ctx context.Context, summary *models.MetricsDailySummary) error
}

// Aggregator aggregates backup metrics into daily summaries.
type Aggregator struct {
	store  AggregatorStore
	logger zerolog.Logger
}

// NewAggregator creates a new Aggregator.
func NewAggregator(store AggregatorStore, logger zerolog.Logger) *Aggregator {
	return &Aggregator{
		store:  store,
		logger: logger.With().Str("component", "metrics_aggregator").Logger(),
	}
}

// AggregateDailyMetrics aggregates backup metrics for a single organization and date.
func (a *Aggregator) AggregateDailyMetrics(ctx context.Context, orgID uuid.UUID, date time.Time) error {
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24*time.Hour - time.Nanosecond)

	backups, err := a.store.GetBackupsByOrgIDAndDateRange(ctx, orgID, dayStart, dayEnd)
	if err != nil {
		return fmt.Errorf("get backups for date %s: %w", dayStart.Format("2006-01-02"), err)
	}

	now := time.Now()
	summary := &models.MetricsDailySummary{
		ID:        uuid.New(),
		OrgID:     orgID,
		Date:      dayStart,
		CreatedAt: now,
		UpdatedAt: now,
	}

	agentSet := make(map[uuid.UUID]struct{})

	for _, b := range backups {
		summary.TotalBackups++

		switch b.Status {
		case models.BackupStatusCompleted:
			summary.SuccessfulBackups++
			if b.SizeBytes != nil {
				summary.TotalSizeBytes += *b.SizeBytes
			}
		case models.BackupStatusFailed:
			summary.FailedBackups++
		}

		if b.CompletedAt != nil && !b.StartedAt.IsZero() {
			summary.TotalDurationSecs += int64(b.CompletedAt.Sub(b.StartedAt).Seconds())
		}

		agentSet[b.AgentID] = struct{}{}
	}

	summary.AgentsActive = len(agentSet)

	if err := a.store.CreateOrUpdateDailySummary(ctx, summary); err != nil {
		return fmt.Errorf("upsert daily summary for date %s: %w", dayStart.Format("2006-01-02"), err)
	}

	a.logger.Debug().
		Str("org_id", orgID.String()).
		Str("date", dayStart.Format("2006-01-02")).
		Int("total_backups", summary.TotalBackups).
		Msg("aggregated daily metrics")

	return nil
}

// AggregateAllOrgs aggregates daily metrics for all organizations.
func (a *Aggregator) AggregateAllOrgs(ctx context.Context, date time.Time) error {
	orgs, err := a.store.GetAllOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("get organizations: %w", err)
	}

	var errs []error
	for _, org := range orgs {
		if err := a.AggregateDailyMetrics(ctx, org.ID, date); err != nil {
			a.logger.Error().Err(err).Str("org_id", org.ID.String()).Msg("failed to aggregate daily metrics")
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to aggregate %d/%d organizations", len(errs), len(orgs))
	}

	return nil
}
