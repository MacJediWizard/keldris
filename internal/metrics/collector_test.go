package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockStore implements Store for testing.
type mockStore struct {
	agents           []*models.Agent
	backups          []*models.Backup
	backupsSince     []*models.Backup
	repositories     []*models.Repository
	schedules        []*models.Schedule
	storageSummary   *models.StorageStatsSummary
	storageStats     []*models.StorageStats
	metricsHistory   *models.MetricsHistory
	dashboardStats   *models.DashboardStats
	rate7d           *models.BackupSuccessRate
	rate30d          *models.BackupSuccessRate
	storageGrowth    []*models.StorageGrowthTrend
	durationTrend    []*models.BackupDurationTrend
	dailyStats       []*models.DailyBackupStats
	backupTotal      int
	backupRunning    int
	backupFailed24h  int

	agentErr         error
	backupErr        error
	backupSinceErr   error
	backupCountsErr  error
	repoErr          error
	scheduleErr      error
	storageSumErr    error
	storageStatsErr  error
	createMetricsErr error
	dashboardErr     error
	successRateErr   error
	storageGrowthErr error
	durationTrendErr error
	dailyStatsErr    error
}

func (m *mockStore) GetAgentsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Agent, error) {
	return m.agents, m.agentErr
}

func (m *mockStore) GetBackupsByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Backup, error) {
	return m.backups, m.backupErr
}

func (m *mockStore) GetBackupsByOrgIDSince(_ context.Context, _ uuid.UUID, _ time.Time) ([]*models.Backup, error) {
	return m.backupsSince, m.backupSinceErr
}

func (m *mockStore) GetBackupCountsByOrgID(_ context.Context, _ uuid.UUID) (int, int, int, error) {
	return m.backupTotal, m.backupRunning, m.backupFailed24h, m.backupCountsErr
}

func (m *mockStore) GetRepositoriesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Repository, error) {
	return m.repositories, m.repoErr
}

func (m *mockStore) GetSchedulesByOrgID(_ context.Context, _ uuid.UUID) ([]*models.Schedule, error) {
	return m.schedules, m.scheduleErr
}

func (m *mockStore) GetStorageStatsSummary(_ context.Context, _ uuid.UUID) (*models.StorageStatsSummary, error) {
	return m.storageSummary, m.storageSumErr
}

func (m *mockStore) GetLatestStatsForAllRepos(_ context.Context, _ uuid.UUID) ([]*models.StorageStats, error) {
	return m.storageStats, m.storageStatsErr
}

func (m *mockStore) CreateMetricsHistory(_ context.Context, metrics *models.MetricsHistory) error {
	m.metricsHistory = metrics
	return m.createMetricsErr
}

func (m *mockStore) GetDashboardStats(_ context.Context, _ uuid.UUID) (*models.DashboardStats, error) {
	return m.dashboardStats, m.dashboardErr
}

func (m *mockStore) GetBackupSuccessRates(_ context.Context, _ uuid.UUID) (*models.BackupSuccessRate, *models.BackupSuccessRate, error) {
	return m.rate7d, m.rate30d, m.successRateErr
}

func (m *mockStore) GetStorageGrowthTrend(_ context.Context, _ uuid.UUID, _ int) ([]*models.StorageGrowthTrend, error) {
	return m.storageGrowth, m.storageGrowthErr
}

func (m *mockStore) GetBackupDurationTrend(_ context.Context, _ uuid.UUID, _ int) ([]*models.BackupDurationTrend, error) {
	return m.durationTrend, m.durationTrendErr
}

func (m *mockStore) GetDailyBackupStats(_ context.Context, _ uuid.UUID, _ int) ([]*models.DailyBackupStats, error) {
	return m.dailyStats, m.dailyStatsErr
}

func TestCollector_CollectBackupStats(t *testing.T) {
	orgID := uuid.New()
	completedAt := time.Now()
	size := int64(1024)

	t.Run("counts completed and failed backups", func(t *testing.T) {
		store := &mockStore{
			agents: []*models.Agent{},
			backups: []*models.Backup{
				{
					ID:          uuid.New(),
					Status:      models.BackupStatusCompleted,
					SizeBytes:   &size,
					StartedAt:   completedAt.Add(-10 * time.Minute),
					CompletedAt: &completedAt,
				},
				{
					ID:          uuid.New(),
					Status:      models.BackupStatusCompleted,
					SizeBytes:   &size,
					StartedAt:   completedAt.Add(-5 * time.Minute),
					CompletedAt: &completedAt,
				},
				{
					ID:        uuid.New(),
					Status:    models.BackupStatusFailed,
					StartedAt: completedAt.Add(-3 * time.Minute),
				},
			},
			repositories: []*models.Repository{},
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if metrics.BackupCount != 3 {
			t.Errorf("expected 3 backups, got %d", metrics.BackupCount)
		}
		if metrics.BackupSuccessCount != 2 {
			t.Errorf("expected 2 successful, got %d", metrics.BackupSuccessCount)
		}
		if metrics.BackupFailedCount != 1 {
			t.Errorf("expected 1 failed, got %d", metrics.BackupFailedCount)
		}
		if metrics.BackupTotalSize != 2048 {
			t.Errorf("expected total size 2048, got %d", metrics.BackupTotalSize)
		}
	})

	t.Run("handles nil size bytes", func(t *testing.T) {
		store := &mockStore{
			agents: []*models.Agent{},
			backups: []*models.Backup{
				{
					ID:          uuid.New(),
					Status:      models.BackupStatusCompleted,
					SizeBytes:   nil,
					StartedAt:   completedAt.Add(-5 * time.Minute),
					CompletedAt: &completedAt,
				},
			},
			repositories: []*models.Repository{},
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metrics.BackupTotalSize != 0 {
			t.Errorf("expected total size 0 for nil size, got %d", metrics.BackupTotalSize)
		}
	})

	t.Run("calculates total duration", func(t *testing.T) {
		start := time.Now().Add(-10 * time.Minute)
		end := time.Now()
		store := &mockStore{
			agents: []*models.Agent{},
			backups: []*models.Backup{
				{
					ID:          uuid.New(),
					Status:      models.BackupStatusCompleted,
					StartedAt:   start,
					CompletedAt: &end,
				},
			},
			repositories: []*models.Repository{},
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expectedDuration := end.Sub(start).Milliseconds()
		if metrics.BackupTotalDuration != expectedDuration {
			t.Errorf("expected duration %d, got %d", expectedDuration, metrics.BackupTotalDuration)
		}
	})

	t.Run("error fetching backups", func(t *testing.T) {
		store := &mockStore{
			agents:    []*models.Agent{},
			backupErr: errors.New("db error"),
		}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.CollectMetrics(context.Background(), orgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("empty backups", func(t *testing.T) {
		store := &mockStore{
			agents:       []*models.Agent{},
			backups:      []*models.Backup{},
			repositories: []*models.Repository{},
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metrics.BackupCount != 0 {
			t.Errorf("expected 0 backups, got %d", metrics.BackupCount)
		}
	})
}

func TestCollector_CollectAgentStats(t *testing.T) {
	orgID := uuid.New()

	t.Run("counts online and offline agents", func(t *testing.T) {
		store := &mockStore{
			agents: []*models.Agent{
				{ID: uuid.New(), Status: models.AgentStatusActive},
				{ID: uuid.New(), Status: models.AgentStatusActive},
				{ID: uuid.New(), Status: models.AgentStatusOffline},
				{ID: uuid.New(), Status: models.AgentStatusPending},
			},
			backups:      []*models.Backup{},
			repositories: []*models.Repository{},
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metrics.AgentTotalCount != 4 {
			t.Errorf("expected 4 total agents, got %d", metrics.AgentTotalCount)
		}
		if metrics.AgentOnlineCount != 2 {
			t.Errorf("expected 2 online agents, got %d", metrics.AgentOnlineCount)
		}
		if metrics.AgentOfflineCount != 1 {
			t.Errorf("expected 1 offline agent, got %d", metrics.AgentOfflineCount)
		}
	})

	t.Run("error fetching agents", func(t *testing.T) {
		store := &mockStore{
			agentErr: errors.New("db error"),
		}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.CollectMetrics(context.Background(), orgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("no agents", func(t *testing.T) {
		store := &mockStore{
			agents:       []*models.Agent{},
			backups:      []*models.Backup{},
			repositories: []*models.Repository{},
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metrics.AgentTotalCount != 0 {
			t.Errorf("expected 0 agents, got %d", metrics.AgentTotalCount)
		}
	})
}

func TestCollector_CollectStorageStats(t *testing.T) {
	orgID := uuid.New()

	t.Run("collects storage summary", func(t *testing.T) {
		store := &mockStore{
			agents:       []*models.Agent{},
			backups:      []*models.Backup{},
			repositories: []*models.Repository{},
			storageSummary: &models.StorageStatsSummary{
				TotalRawSize:     1000,
				TotalRestoreSize: 5000,
				TotalSpaceSaved:  4000,
				TotalSnapshots:   10,
			},
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metrics.StorageUsedBytes != 1000 {
			t.Errorf("expected storage used 1000, got %d", metrics.StorageUsedBytes)
		}
		if metrics.StorageRawBytes != 5000 {
			t.Errorf("expected storage raw 5000, got %d", metrics.StorageRawBytes)
		}
		if metrics.StorageSpaceSaved != 4000 {
			t.Errorf("expected space saved 4000, got %d", metrics.StorageSpaceSaved)
		}
		if metrics.TotalSnapshots != 10 {
			t.Errorf("expected 10 snapshots, got %d", metrics.TotalSnapshots)
		}
	})

	t.Run("handles storage summary error gracefully", func(t *testing.T) {
		store := &mockStore{
			agents:        []*models.Agent{},
			backups:       []*models.Backup{},
			repositories:  []*models.Repository{},
			storageSumErr: errors.New("storage error"),
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metrics.StorageUsedBytes != 0 {
			t.Errorf("expected 0 storage on error, got %d", metrics.StorageUsedBytes)
		}
	})

	t.Run("handles nil storage summary", func(t *testing.T) {
		store := &mockStore{
			agents:         []*models.Agent{},
			backups:        []*models.Backup{},
			repositories:   []*models.Repository{},
			storageSummary: nil,
		}
		c := NewCollector(store, zerolog.Nop())
		metrics, err := c.CollectMetrics(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if metrics.StorageUsedBytes != 0 {
			t.Errorf("expected 0 storage for nil summary, got %d", metrics.StorageUsedBytes)
		}
	})
}

func TestCollector_Aggregate(t *testing.T) {
	orgID := uuid.New()

	t.Run("GetDashboardStats aggregates all data", func(t *testing.T) {
		store := &mockStore{
			agents: []*models.Agent{
				{ID: uuid.New(), Status: models.AgentStatusActive},
				{ID: uuid.New(), Status: models.AgentStatusOffline},
			},
			backupTotal:     15,
			backupRunning:   2,
			backupFailed24h: 1,
			repositories: []*models.Repository{
				{ID: uuid.New()},
				{ID: uuid.New()},
			},
			schedules: []*models.Schedule{
				{ID: uuid.New(), Enabled: true},
				{ID: uuid.New(), Enabled: false},
				{ID: uuid.New(), Enabled: true},
			},
			storageSummary: &models.StorageStatsSummary{
				TotalRawSize:     2000,
				TotalRestoreSize: 8000,
				TotalSpaceSaved:  6000,
				AvgDedupRatio:    4.0,
			},
			rate7d:  &models.BackupSuccessRate{SuccessPercent: 95.0},
			rate30d: &models.BackupSuccessRate{SuccessPercent: 90.0},
		}
		c := NewCollector(store, zerolog.Nop())
		stats, err := c.GetDashboardStats(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if stats.AgentTotal != 2 {
			t.Errorf("expected 2 total agents, got %d", stats.AgentTotal)
		}
		if stats.AgentOnline != 1 {
			t.Errorf("expected 1 online agent, got %d", stats.AgentOnline)
		}
		if stats.AgentOffline != 1 {
			t.Errorf("expected 1 offline agent, got %d", stats.AgentOffline)
		}
		if stats.BackupTotal != 15 {
			t.Errorf("expected 15 total backups, got %d", stats.BackupTotal)
		}
		if stats.BackupRunning != 2 {
			t.Errorf("expected 2 running, got %d", stats.BackupRunning)
		}
		if stats.BackupFailed24h != 1 {
			t.Errorf("expected 1 failed24h, got %d", stats.BackupFailed24h)
		}
		if stats.RepositoryCount != 2 {
			t.Errorf("expected 2 repos, got %d", stats.RepositoryCount)
		}
		if stats.ScheduleCount != 3 {
			t.Errorf("expected 3 schedules, got %d", stats.ScheduleCount)
		}
		if stats.ScheduleEnabled != 2 {
			t.Errorf("expected 2 enabled schedules, got %d", stats.ScheduleEnabled)
		}
		if stats.TotalRawSize != 2000 {
			t.Errorf("expected raw size 2000, got %d", stats.TotalRawSize)
		}
		if stats.AvgDedupRatio != 4.0 {
			t.Errorf("expected dedup ratio 4.0, got %f", stats.AvgDedupRatio)
		}
		if stats.SuccessRate7d != 95.0 {
			t.Errorf("expected 7d rate 95.0, got %f", stats.SuccessRate7d)
		}
		if stats.SuccessRate30d != 90.0 {
			t.Errorf("expected 30d rate 90.0, got %f", stats.SuccessRate30d)
		}
	})

	t.Run("GetDashboardStats handles agent error", func(t *testing.T) {
		store := &mockStore{
			agentErr: errors.New("db error"),
		}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.GetDashboardStats(context.Background(), orgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("GetDashboardStats handles backup counts error", func(t *testing.T) {
		store := &mockStore{
			agents:          []*models.Agent{},
			backupCountsErr: errors.New("db error"),
		}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.GetDashboardStats(context.Background(), orgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("GetDashboardStats handles repo error", func(t *testing.T) {
		store := &mockStore{
			agents:  []*models.Agent{},
			repoErr: errors.New("db error"),
		}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.GetDashboardStats(context.Background(), orgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("GetDashboardStats handles schedule error", func(t *testing.T) {
		store := &mockStore{
			agents:       []*models.Agent{},
			repositories: []*models.Repository{},
			scheduleErr:  errors.New("db error"),
		}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.GetDashboardStats(context.Background(), orgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("GetDashboardStats handles storage summary error gracefully", func(t *testing.T) {
		store := &mockStore{
			agents:        []*models.Agent{},
			repositories:  []*models.Repository{},
			schedules:     []*models.Schedule{},
			storageSumErr: errors.New("storage error"),
			rate7d:        &models.BackupSuccessRate{SuccessPercent: 99.0},
			rate30d:       &models.BackupSuccessRate{SuccessPercent: 98.0},
		}
		c := NewCollector(store, zerolog.Nop())
		stats, err := c.GetDashboardStats(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.TotalRawSize != 0 {
			t.Errorf("expected 0 raw size on error, got %d", stats.TotalRawSize)
		}
	})

	t.Run("GetDashboardStats handles success rate error gracefully", func(t *testing.T) {
		store := &mockStore{
			agents:         []*models.Agent{},
			repositories:   []*models.Repository{},
			schedules:      []*models.Schedule{},
			successRateErr: errors.New("rate error"),
		}
		c := NewCollector(store, zerolog.Nop())
		stats, err := c.GetDashboardStats(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.SuccessRate7d != 0 {
			t.Errorf("expected 0 rate on error, got %f", stats.SuccessRate7d)
		}
	})

	t.Run("GetDashboardStats handles nil success rates", func(t *testing.T) {
		store := &mockStore{
			agents:       []*models.Agent{},
			repositories: []*models.Repository{},
			schedules:    []*models.Schedule{},
			rate7d:       nil,
			rate30d:      nil,
		}
		c := NewCollector(store, zerolog.Nop())
		stats, err := c.GetDashboardStats(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if stats.SuccessRate7d != 0 {
			t.Errorf("expected 0 rate for nil, got %f", stats.SuccessRate7d)
		}
	})
}

func TestCollector_GetBackupSuccessRates(t *testing.T) {
	orgID := uuid.New()

	t.Run("delegates to store", func(t *testing.T) {
		rate7d := &models.BackupSuccessRate{SuccessPercent: 95.0}
		rate30d := &models.BackupSuccessRate{SuccessPercent: 90.0}
		store := &mockStore{rate7d: rate7d, rate30d: rate30d}
		c := NewCollector(store, zerolog.Nop())

		r7, r30, err := c.GetBackupSuccessRates(context.Background(), orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if r7.SuccessPercent != 95.0 {
			t.Errorf("expected 95.0, got %f", r7.SuccessPercent)
		}
		if r30.SuccessPercent != 90.0 {
			t.Errorf("expected 90.0, got %f", r30.SuccessPercent)
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		store := &mockStore{successRateErr: errors.New("db error")}
		c := NewCollector(store, zerolog.Nop())
		_, _, err := c.GetBackupSuccessRates(context.Background(), orgID)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCollector_GetStorageGrowthTrend(t *testing.T) {
	orgID := uuid.New()

	t.Run("delegates to store", func(t *testing.T) {
		trend := []*models.StorageGrowthTrend{{TotalSize: 100}}
		store := &mockStore{storageGrowth: trend}
		c := NewCollector(store, zerolog.Nop())

		result, err := c.GetStorageGrowthTrend(context.Background(), orgID, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 trend point, got %d", len(result))
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		store := &mockStore{storageGrowthErr: errors.New("db error")}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.GetStorageGrowthTrend(context.Background(), orgID, 30)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCollector_GetBackupDurationTrend(t *testing.T) {
	orgID := uuid.New()

	t.Run("delegates to store", func(t *testing.T) {
		trend := []*models.BackupDurationTrend{{AvgDurationMs: 5000}}
		store := &mockStore{durationTrend: trend}
		c := NewCollector(store, zerolog.Nop())

		result, err := c.GetBackupDurationTrend(context.Background(), orgID, 7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 trend point, got %d", len(result))
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		store := &mockStore{durationTrendErr: errors.New("db error")}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.GetBackupDurationTrend(context.Background(), orgID, 7)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCollector_GetDailyBackupStats(t *testing.T) {
	orgID := uuid.New()

	t.Run("delegates to store", func(t *testing.T) {
		daily := []*models.DailyBackupStats{{Total: 10, Successful: 9, Failed: 1}}
		store := &mockStore{dailyStats: daily}
		c := NewCollector(store, zerolog.Nop())

		result, err := c.GetDailyBackupStats(context.Background(), orgID, 14)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 daily stat, got %d", len(result))
		}
	})

	t.Run("propagates error", func(t *testing.T) {
		store := &mockStore{dailyStatsErr: errors.New("db error")}
		c := NewCollector(store, zerolog.Nop())
		_, err := c.GetDailyBackupStats(context.Background(), orgID, 14)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestCollector_CollectMetrics_RepoError(t *testing.T) {
	store := &mockStore{
		agents:  []*models.Agent{},
		backups: []*models.Backup{},
		repoErr: errors.New("repo error"),
	}
	c := NewCollector(store, zerolog.Nop())
	_, err := c.CollectMetrics(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewCollector(t *testing.T) {
	store := &mockStore{}
	logger := zerolog.Nop()
	c := NewCollector(store, logger)
	if c == nil {
		t.Fatal("expected non-nil collector")
	}
}
