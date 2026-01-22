// Package metrics provides metrics collection for the Keldris dashboard.
package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Store defines the interface for metrics persistence operations.
type Store interface {
	// Agent metrics
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)

	// Backup metrics
	GetBackupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Backup, error)
	GetBackupsByOrgIDSince(ctx context.Context, orgID uuid.UUID, since time.Time) ([]*models.Backup, error)
	GetBackupCountsByOrgID(ctx context.Context, orgID uuid.UUID) (total, running, failed24h int, err error)

	// Repository metrics
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)

	// Schedule metrics
	GetSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error)

	// Storage stats
	GetStorageStatsSummary(ctx context.Context, orgID uuid.UUID) (*models.StorageStatsSummary, error)
	GetLatestStatsForAllRepos(ctx context.Context, orgID uuid.UUID) ([]*models.StorageStats, error)

	// Metrics persistence
	CreateMetricsHistory(ctx context.Context, metrics *models.MetricsHistory) error
	GetDashboardStats(ctx context.Context, orgID uuid.UUID) (*models.DashboardStats, error)
	GetBackupSuccessRates(ctx context.Context, orgID uuid.UUID) (*models.BackupSuccessRate, *models.BackupSuccessRate, error)
	GetStorageGrowthTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthTrend, error)
	GetBackupDurationTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.BackupDurationTrend, error)
	GetDailyBackupStats(ctx context.Context, orgID uuid.UUID, days int) ([]*models.DailyBackupStats, error)
}

// Collector collects and aggregates system metrics.
type Collector struct {
	store  Store
	logger zerolog.Logger
}

// NewCollector creates a new Collector.
func NewCollector(store Store, logger zerolog.Logger) *Collector {
	return &Collector{
		store:  store,
		logger: logger.With().Str("component", "metrics_collector").Logger(),
	}
}

// CollectMetrics collects current system metrics for an organization.
func (c *Collector) CollectMetrics(ctx context.Context, orgID uuid.UUID) (*models.MetricsHistory, error) {
	metrics := models.NewMetricsHistory(orgID)

	// Collect agent metrics
	agents, err := c.store.GetAgentsByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get agents: %w", err)
	}

	metrics.AgentTotalCount = len(agents)
	for _, agent := range agents {
		if agent.Status == models.AgentStatusActive {
			metrics.AgentOnlineCount++
		} else if agent.Status == models.AgentStatusOffline {
			metrics.AgentOfflineCount++
		}
	}

	// Collect backup metrics
	backups, err := c.store.GetBackupsByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get backups: %w", err)
	}

	metrics.BackupCount = len(backups)
	for _, backup := range backups {
		if backup.Status == models.BackupStatusCompleted {
			metrics.BackupSuccessCount++
			if backup.SizeBytes != nil {
				metrics.BackupTotalSize += *backup.SizeBytes
			}
		} else if backup.Status == models.BackupStatusFailed {
			metrics.BackupFailedCount++
		}
		if backup.CompletedAt != nil && !backup.StartedAt.IsZero() {
			metrics.BackupTotalDuration += backup.CompletedAt.Sub(backup.StartedAt).Milliseconds()
		}
	}

	// Collect repository metrics
	repos, err := c.store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get repositories: %w", err)
	}
	metrics.RepositoryCount = len(repos)

	// Collect storage stats
	storageSummary, err := c.store.GetStorageStatsSummary(ctx, orgID)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to get storage summary, using defaults")
	} else if storageSummary != nil {
		metrics.StorageUsedBytes = storageSummary.TotalRawSize
		metrics.StorageRawBytes = storageSummary.TotalRestoreSize
		metrics.StorageSpaceSaved = storageSummary.TotalSpaceSaved
		metrics.TotalSnapshots = storageSummary.TotalSnapshots
	}

	return metrics, nil
}

// GetDashboardStats returns aggregated dashboard statistics.
func (c *Collector) GetDashboardStats(ctx context.Context, orgID uuid.UUID) (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	// Get agent counts
	agents, err := c.store.GetAgentsByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get agents: %w", err)
	}
	stats.AgentTotal = len(agents)
	for _, agent := range agents {
		if agent.Status == models.AgentStatusActive {
			stats.AgentOnline++
		} else if agent.Status == models.AgentStatusOffline {
			stats.AgentOffline++
		}
	}

	// Get backup counts
	total, running, failed24h, err := c.store.GetBackupCountsByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get backup counts: %w", err)
	}
	stats.BackupTotal = total
	stats.BackupRunning = running
	stats.BackupFailed24h = failed24h

	// Get repository count
	repos, err := c.store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get repositories: %w", err)
	}
	stats.RepositoryCount = len(repos)

	// Get schedule counts
	schedules, err := c.store.GetSchedulesByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get schedules: %w", err)
	}
	stats.ScheduleCount = len(schedules)
	for _, sched := range schedules {
		if sched.Enabled {
			stats.ScheduleEnabled++
		}
	}

	// Get storage stats
	storageSummary, err := c.store.GetStorageStatsSummary(ctx, orgID)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to get storage summary")
	} else if storageSummary != nil {
		stats.TotalRawSize = storageSummary.TotalRawSize
		stats.TotalBackupSize = storageSummary.TotalRestoreSize
		stats.TotalSpaceSaved = storageSummary.TotalSpaceSaved
		stats.AvgDedupRatio = storageSummary.AvgDedupRatio
	}

	// Get success rates
	rate7d, rate30d, err := c.store.GetBackupSuccessRates(ctx, orgID)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to get success rates")
	} else {
		if rate7d != nil {
			stats.SuccessRate7d = rate7d.SuccessPercent
		}
		if rate30d != nil {
			stats.SuccessRate30d = rate30d.SuccessPercent
		}
	}

	return stats, nil
}

// GetBackupSuccessRates returns success rates for 7-day and 30-day periods.
func (c *Collector) GetBackupSuccessRates(ctx context.Context, orgID uuid.UUID) (*models.BackupSuccessRate, *models.BackupSuccessRate, error) {
	return c.store.GetBackupSuccessRates(ctx, orgID)
}

// GetStorageGrowthTrend returns storage growth over time.
func (c *Collector) GetStorageGrowthTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthTrend, error) {
	return c.store.GetStorageGrowthTrend(ctx, orgID, days)
}

// GetBackupDurationTrend returns backup duration trends over time.
func (c *Collector) GetBackupDurationTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.BackupDurationTrend, error) {
	return c.store.GetBackupDurationTrend(ctx, orgID, days)
}

// GetDailyBackupStats returns daily backup statistics.
func (c *Collector) GetDailyBackupStats(ctx context.Context, orgID uuid.UUID, days int) ([]*models.DailyBackupStats, error) {
	return c.store.GetDailyBackupStats(ctx, orgID, days)
}
