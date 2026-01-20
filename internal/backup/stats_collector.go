package backup

import (
	"context"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// StatsStore defines the interface for storage stats persistence operations.
type StatsStore interface {
	// GetRepositoriesByOrgID returns all repositories for an organization.
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)

	// GetRepositoryByID returns a repository by ID.
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)

	// CreateStorageStats creates a new storage stats record.
	CreateStorageStats(ctx context.Context, stats *models.StorageStats) error

	// GetAllOrganizations returns all organizations (for collecting stats across all orgs).
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
}

// StatsCollectorConfig holds configuration for the stats collector.
type StatsCollectorConfig struct {
	// CronSchedule is the cron expression for when to collect stats (default: daily at 2am).
	CronSchedule string

	// PasswordFunc retrieves the repository password.
	PasswordFunc func(repoID uuid.UUID) (string, error)

	// DecryptFunc decrypts the repository configuration.
	DecryptFunc DecryptFunc
}

// DefaultStatsCollectorConfig returns a StatsCollectorConfig with sensible defaults.
func DefaultStatsCollectorConfig() StatsCollectorConfig {
	return StatsCollectorConfig{
		CronSchedule: "0 0 2 * * *", // Daily at 2:00 AM
	}
}

// StatsCollector collects repository storage statistics on a schedule.
type StatsCollector struct {
	store   StatsStore
	restic  *Restic
	config  StatsCollectorConfig
	cron    *cron.Cron
	logger  zerolog.Logger
	mu      sync.RWMutex
	running bool
	entryID cron.EntryID
}

// NewStatsCollector creates a new StatsCollector.
func NewStatsCollector(store StatsStore, restic *Restic, config StatsCollectorConfig, logger zerolog.Logger) *StatsCollector {
	return &StatsCollector{
		store:  store,
		restic: restic,
		config: config,
		cron:   cron.New(cron.WithSeconds()),
		logger: logger.With().Str("component", "stats_collector").Logger(),
	}
}

// Start starts the stats collector scheduler.
func (c *StatsCollector) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = true
	c.mu.Unlock()

	c.logger.Info().
		Str("schedule", c.config.CronSchedule).
		Msg("starting stats collector")

	entryID, err := c.cron.AddFunc(c.config.CronSchedule, func() {
		c.collectAllStats()
	})
	if err != nil {
		return err
	}
	c.entryID = entryID

	c.cron.Start()
	c.logger.Info().Msg("stats collector started")

	return nil
}

// Stop stops the stats collector scheduler.
func (c *StatsCollector) Stop() context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	c.running = false
	c.logger.Info().Msg("stopping stats collector")

	return c.cron.Stop()
}

// CollectNow triggers an immediate stats collection for all repositories.
func (c *StatsCollector) CollectNow(ctx context.Context) error {
	c.logger.Info().Msg("manual stats collection triggered")
	return c.collectAllStatsWithContext(ctx)
}

// CollectForRepository collects stats for a single repository.
func (c *StatsCollector) CollectForRepository(ctx context.Context, repositoryID uuid.UUID) error {
	c.logger.Info().
		Str("repository_id", repositoryID.String()).
		Msg("collecting stats for repository")

	repo, err := c.store.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return err
	}

	return c.collectRepoStats(ctx, repo)
}

// collectAllStats collects stats for all repositories across all organizations.
func (c *StatsCollector) collectAllStats() {
	ctx := context.Background()
	if err := c.collectAllStatsWithContext(ctx); err != nil {
		c.logger.Error().Err(err).Msg("failed to collect stats")
	}
}

// collectAllStatsWithContext collects stats for all repositories with a given context.
func (c *StatsCollector) collectAllStatsWithContext(ctx context.Context) error {
	c.logger.Info().Msg("starting scheduled stats collection")

	orgs, err := c.store.GetAllOrganizations(ctx)
	if err != nil {
		return err
	}

	var totalRepos, successCount, failCount int

	for _, org := range orgs {
		repos, err := c.store.GetRepositoriesByOrgID(ctx, org.ID)
		if err != nil {
			c.logger.Error().
				Err(err).
				Str("org_id", org.ID.String()).
				Msg("failed to get repositories for org")
			continue
		}

		totalRepos += len(repos)

		for _, repo := range repos {
			if err := c.collectRepoStats(ctx, repo); err != nil {
				c.logger.Error().
					Err(err).
					Str("repository_id", repo.ID.String()).
					Str("repository_name", repo.Name).
					Msg("failed to collect stats for repository")
				failCount++
			} else {
				successCount++
			}
		}
	}

	c.logger.Info().
		Int("total_repos", totalRepos).
		Int("success", successCount).
		Int("failed", failCount).
		Msg("stats collection completed")

	return nil
}

// collectRepoStats collects stats for a single repository.
func (c *StatsCollector) collectRepoStats(ctx context.Context, repo *models.Repository) error {
	logger := c.logger.With().
		Str("repository_id", repo.ID.String()).
		Str("repository_name", repo.Name).
		Logger()

	logger.Debug().Msg("collecting repository stats")

	// Decrypt repository configuration
	if c.config.DecryptFunc == nil {
		logger.Warn().Msg("decrypt function not configured, skipping repository")
		return nil
	}

	configJSON, err := c.config.DecryptFunc(repo.ConfigEncrypted)
	if err != nil {
		return err
	}

	// Parse backend configuration
	backend, err := ParseBackend(repo.Type, configJSON)
	if err != nil {
		return err
	}

	// Get repository password
	if c.config.PasswordFunc == nil {
		logger.Warn().Msg("password function not configured, skipping repository")
		return nil
	}

	password, err := c.config.PasswordFunc(repo.ID)
	if err != nil {
		return err
	}

	// Build restic config
	resticCfg := backend.ToResticConfig(password)

	// Get extended stats
	extStats, err := c.restic.GetExtendedStats(ctx, resticCfg)
	if err != nil {
		return err
	}

	// Create storage stats record
	stats := models.NewStorageStats(repo.ID)
	stats.SetStats(
		extStats.TotalSize,
		extStats.TotalFileCount,
		extStats.RawDataSize,
		extStats.RestoreSize,
		extStats.SnapshotCount,
	)

	// Save to database
	if err := c.store.CreateStorageStats(ctx, stats); err != nil {
		return err
	}

	logger.Info().
		Int64("raw_data_size", stats.RawDataSize).
		Int64("restore_size", stats.RestoreSize).
		Float64("dedup_ratio", stats.DedupRatio).
		Int64("space_saved", stats.SpaceSaved).
		Msg("stats collected and saved")

	return nil
}

// GetNextRun returns the next scheduled stats collection time.
func (c *StatsCollector) GetNextRun() (time.Time, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.running {
		return time.Time{}, false
	}

	entry := c.cron.Entry(c.entryID)
	if !entry.Valid() {
		return time.Time{}, false
	}

	return entry.Next, true
}

// IsRunning returns whether the stats collector is currently running.
func (c *StatsCollector) IsRunning() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running
}
