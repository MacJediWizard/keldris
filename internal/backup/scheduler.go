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

// ScheduleStore defines the interface for loading schedule data.
type ScheduleStore interface {
	// GetEnabledSchedules returns all enabled schedules.
	GetEnabledSchedules(ctx context.Context) ([]models.Schedule, error)

	// GetRepository returns a repository by ID.
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)

	// CreateBackup creates a new backup record.
	CreateBackup(ctx context.Context, backup *models.Backup) error

	// UpdateBackup updates an existing backup record.
	UpdateBackup(ctx context.Context, backup *models.Backup) error

	// GetOrCreateReplicationStatus gets or creates a replication status record.
	GetOrCreateReplicationStatus(ctx context.Context, scheduleID, sourceRepoID, targetRepoID uuid.UUID) (*models.ReplicationStatus, error)

	// UpdateReplicationStatus updates a replication status record.
	UpdateReplicationStatus(ctx context.Context, rs *models.ReplicationStatus) error
}

const (
	// maxRetries is the number of retry attempts per repository.
	maxRetries = 3

	// retryDelay is the delay between retry attempts.
	retryDelay = 5 * time.Second
)

// DecryptFunc is a function that decrypts repository configuration.
type DecryptFunc func(encrypted []byte) ([]byte, error)

// SchedulerConfig holds configuration for the backup scheduler.
type SchedulerConfig struct {
	// RefreshInterval is how often to reload schedules from the database.
	RefreshInterval time.Duration

	// PasswordFunc retrieves the repository password.
	PasswordFunc func(repoID uuid.UUID) (string, error)

	// DecryptFunc decrypts the repository configuration.
	DecryptFunc DecryptFunc
}

// DefaultSchedulerConfig returns a SchedulerConfig with sensible defaults.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		RefreshInterval: 5 * time.Minute,
	}
}

// Scheduler manages backup schedules using cron.
type Scheduler struct {
	store   ScheduleStore
	restic  *Restic
	config  SchedulerConfig
	cron    *cron.Cron
	logger  zerolog.Logger
	mu      sync.RWMutex
	entries map[uuid.UUID]cron.EntryID
	running bool
}

// NewScheduler creates a new backup scheduler.
func NewScheduler(store ScheduleStore, restic *Restic, config SchedulerConfig, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		store:   store,
		restic:  restic,
		config:  config,
		cron:    cron.New(cron.WithSeconds()),
		logger:  logger.With().Str("component", "scheduler").Logger(),
		entries: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start starts the scheduler and loads initial schedules.
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().Msg("starting backup scheduler")

	// Load initial schedules
	if err := s.Reload(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to load initial schedules")
	}

	// Start cron scheduler
	s.cron.Start()

	// Start background refresh goroutine
	go s.refreshLoop(ctx)

	s.logger.Info().Msg("backup scheduler started")
	return nil
}

// Stop stops the scheduler gracefully.
func (s *Scheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	s.running = false
	s.logger.Info().Msg("stopping backup scheduler")

	return s.cron.Stop()
}

// Reload reloads all schedules from the database.
func (s *Scheduler) Reload(ctx context.Context) error {
	s.logger.Debug().Msg("reloading schedules from database")

	schedules, err := s.store.GetEnabledSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get enabled schedules: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Track which schedules we've seen
	seen := make(map[uuid.UUID]bool)

	for _, schedule := range schedules {
		seen[schedule.ID] = true

		// Check if schedule already exists with same cron expression
		if entryID, exists := s.entries[schedule.ID]; exists {
			entry := s.cron.Entry(entryID)
			// If entry is valid and schedule hasn't changed, skip
			if entry.Valid() {
				continue
			}
			// Remove old entry
			s.cron.Remove(entryID)
			delete(s.entries, schedule.ID)
		}

		// Add new schedule
		if err := s.addSchedule(schedule); err != nil {
			s.logger.Error().
				Err(err).
				Str("schedule_id", schedule.ID.String()).
				Str("schedule_name", schedule.Name).
				Msg("failed to add schedule")
			continue
		}
	}

	// Remove schedules that are no longer enabled
	for id, entryID := range s.entries {
		if !seen[id] {
			s.cron.Remove(entryID)
			delete(s.entries, id)
			s.logger.Debug().
				Str("schedule_id", id.String()).
				Msg("removed disabled schedule")
		}
	}

	s.logger.Info().
		Int("active_schedules", len(s.entries)).
		Msg("schedules reloaded")

	return nil
}

// addSchedule adds a schedule to the cron scheduler.
func (s *Scheduler) addSchedule(schedule models.Schedule) error {
	// Create a copy of schedule for the closure
	sched := schedule

	entryID, err := s.cron.AddFunc(schedule.CronExpression, func() {
		s.executeBackup(sched)
	})
	if err != nil {
		return fmt.Errorf("add cron entry: %w", err)
	}

	s.entries[schedule.ID] = entryID
	s.logger.Debug().
		Str("schedule_id", schedule.ID.String()).
		Str("schedule_name", schedule.Name).
		Str("cron_expression", schedule.CronExpression).
		Msg("added schedule")

	return nil
}

// executeBackup runs a backup for the given schedule with retry/failover and replication.
func (s *Scheduler) executeBackup(schedule models.Schedule) {
	ctx := context.Background()
	logger := s.logger.With().
		Str("schedule_id", schedule.ID.String()).
		Str("schedule_name", schedule.Name).
		Logger()

	logger.Info().Msg("starting scheduled backup")

	// Get enabled repositories sorted by priority
	enabledRepos := schedule.GetEnabledRepositories()
	if len(enabledRepos) == 0 {
		logger.Error().Msg("no enabled repositories for schedule")
		return
	}

	// Try backup to each repository with retry logic
	var successRepo *models.ScheduleRepository
	var successBackup *models.Backup
	var successStats *BackupStats
	var successResticCfg ResticConfig
	var lastErr error

	for i := range enabledRepos {
		schedRepo := &enabledRepos[i]
		repoLogger := logger.With().
			Str("repository_id", schedRepo.RepositoryID.String()).
			Int("priority", schedRepo.Priority).
			Logger()

		repoLogger.Info().Msg("attempting backup to repository")

		// Try up to maxRetries times for this repository
		for attempt := 1; attempt <= maxRetries; attempt++ {
			backup, stats, resticCfg, err := s.runBackupToRepo(ctx, schedule, schedRepo, attempt, repoLogger)
			if err == nil {
				successRepo = schedRepo
				successBackup = backup
				successStats = stats
				successResticCfg = resticCfg
				break
			}

			lastErr = err
			repoLogger.Warn().
				Err(err).
				Int("attempt", attempt).
				Int("max_attempts", maxRetries).
				Msg("backup attempt failed")

			if attempt < maxRetries {
				time.Sleep(retryDelay)
			}
		}

		if successRepo != nil {
			break
		}
		repoLogger.Warn().Msg("all retry attempts failed for repository, trying next")
	}

	// If all repositories failed, log and return
	if successRepo == nil {
		logger.Error().
			Err(lastErr).
			Int("repos_tried", len(enabledRepos)).
			Msg("backup failed to all repositories")
		return
	}

	logger.Info().
		Str("repository_id", successRepo.RepositoryID.String()).
		Str("snapshot_id", successStats.SnapshotID).
		Int("files_new", successStats.FilesNew).
		Int("files_changed", successStats.FilesChanged).
		Int64("size_bytes", successStats.SizeBytes).
		Dur("duration", successStats.Duration).
		Msg("backup completed successfully")

	// Run prune if retention policy is set
	if schedule.RetentionPolicy != nil {
		logger.Info().Msg("running prune with retention policy")
		if err := s.restic.Prune(ctx, successResticCfg, schedule.RetentionPolicy); err != nil {
			logger.Error().Err(err).Msg("prune failed")
		} else {
			logger.Info().Msg("prune completed")
		}
	}

	// Replicate to other repositories
	s.replicateToOtherRepos(ctx, schedule, successRepo, successStats.SnapshotID, successResticCfg, enabledRepos, logger)

	_ = successBackup // Backup record already updated in runBackupToRepo
}

// runBackupToRepo attempts a backup to a specific repository.
func (s *Scheduler) runBackupToRepo(
	ctx context.Context,
	schedule models.Schedule,
	schedRepo *models.ScheduleRepository,
	attempt int,
	logger zerolog.Logger,
) (*models.Backup, *BackupStats, ResticConfig, error) {
	// Create backup record
	backup := models.NewBackup(schedule.ID, schedule.AgentID, &schedRepo.RepositoryID)
	if err := s.store.CreateBackup(ctx, backup); err != nil {
		return nil, nil, ResticConfig{}, fmt.Errorf("create backup record: %w", err)
	}

	// Get repository configuration
	repo, err := s.store.GetRepository(ctx, schedRepo.RepositoryID)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("get repository: %v", err), logger)
		return nil, nil, ResticConfig{}, fmt.Errorf("get repository: %w", err)
	}

	// Decrypt repository configuration
	if s.config.DecryptFunc == nil {
		s.failBackup(ctx, backup, "decrypt function not configured", logger)
		return nil, nil, ResticConfig{}, errors.New("decrypt function not configured")
	}

	configJSON, err := s.config.DecryptFunc(repo.ConfigEncrypted)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("decrypt config: %v", err), logger)
		return nil, nil, ResticConfig{}, fmt.Errorf("decrypt config: %w", err)
	}

	// Parse backend configuration
	backend, err := ParseBackend(repo.Type, configJSON)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("parse backend: %v", err), logger)
		return nil, nil, ResticConfig{}, fmt.Errorf("parse backend: %w", err)
	}

	// Get repository password
	if s.config.PasswordFunc == nil {
		s.failBackup(ctx, backup, "password function not configured", logger)
		return nil, nil, ResticConfig{}, errors.New("password function not configured")
	}

	password, err := s.config.PasswordFunc(repo.ID)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("get password: %v", err), logger)
		return nil, nil, ResticConfig{}, fmt.Errorf("get password: %w", err)
	}

	// Build restic config
	resticCfg := backend.ToResticConfig(password)

	// Build tags for the backup
	tags := []string{
		fmt.Sprintf("schedule:%s", schedule.ID.String()),
		fmt.Sprintf("agent:%s", schedule.AgentID.String()),
	}

	// Run the backup
	stats, err := s.restic.Backup(ctx, resticCfg, schedule.Paths, schedule.Excludes, tags)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("backup failed (attempt %d): %v", attempt, err), logger)
		return nil, nil, ResticConfig{}, fmt.Errorf("backup failed: %w", err)
	}

	// Mark backup as completed
	backup.Complete(stats.SnapshotID, stats.SizeBytes, stats.FilesNew, stats.FilesChanged)
	if err := s.store.UpdateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to update backup record")
	}

	return backup, stats, resticCfg, nil
}

// replicateToOtherRepos copies the snapshot to other enabled repositories.
func (s *Scheduler) replicateToOtherRepos(
	ctx context.Context,
	schedule models.Schedule,
	sourceRepo *models.ScheduleRepository,
	snapshotID string,
	sourceCfg ResticConfig,
	allRepos []models.ScheduleRepository,
	logger zerolog.Logger,
) {
	for i := range allRepos {
		targetRepo := &allRepos[i]

		// Skip the source repository
		if targetRepo.RepositoryID == sourceRepo.RepositoryID {
			continue
		}

		replicateLogger := logger.With().
			Str("source_repo", sourceRepo.RepositoryID.String()).
			Str("target_repo", targetRepo.RepositoryID.String()).
			Logger()

		replicateLogger.Info().Msg("starting replication to secondary repository")

		// Get or create replication status
		replStatus, err := s.store.GetOrCreateReplicationStatus(
			ctx, schedule.ID, sourceRepo.RepositoryID, targetRepo.RepositoryID,
		)
		if err != nil {
			replicateLogger.Error().Err(err).Msg("failed to get replication status")
			continue
		}

		// Mark as syncing
		replStatus.MarkSyncing()
		if err := s.store.UpdateReplicationStatus(ctx, replStatus); err != nil {
			replicateLogger.Error().Err(err).Msg("failed to update replication status")
		}

		// Get target repository configuration
		targetRepoObj, err := s.store.GetRepository(ctx, targetRepo.RepositoryID)
		if err != nil {
			replStatus.MarkFailed(fmt.Sprintf("get target repository: %v", err))
			s.store.UpdateReplicationStatus(ctx, replStatus)
			replicateLogger.Error().Err(err).Msg("failed to get target repository")
			continue
		}

		// Decrypt target repository configuration
		targetConfigJSON, err := s.config.DecryptFunc(targetRepoObj.ConfigEncrypted)
		if err != nil {
			replStatus.MarkFailed(fmt.Sprintf("decrypt target config: %v", err))
			s.store.UpdateReplicationStatus(ctx, replStatus)
			replicateLogger.Error().Err(err).Msg("failed to decrypt target repository config")
			continue
		}

		// Parse target backend configuration
		targetBackend, err := ParseBackend(targetRepoObj.Type, targetConfigJSON)
		if err != nil {
			replStatus.MarkFailed(fmt.Sprintf("parse target backend: %v", err))
			s.store.UpdateReplicationStatus(ctx, replStatus)
			replicateLogger.Error().Err(err).Msg("failed to parse target backend")
			continue
		}

		// Get target repository password
		targetPassword, err := s.config.PasswordFunc(targetRepoObj.ID)
		if err != nil {
			replStatus.MarkFailed(fmt.Sprintf("get target password: %v", err))
			s.store.UpdateReplicationStatus(ctx, replStatus)
			replicateLogger.Error().Err(err).Msg("failed to get target password")
			continue
		}

		// Build target restic config
		targetCfg := targetBackend.ToResticConfig(targetPassword)

		// Copy snapshot to target repository
		if err := s.restic.Copy(ctx, sourceCfg, targetCfg, snapshotID); err != nil {
			replStatus.MarkFailed(fmt.Sprintf("copy snapshot: %v", err))
			s.store.UpdateReplicationStatus(ctx, replStatus)
			replicateLogger.Error().Err(err).Msg("failed to copy snapshot")
			continue
		}

		// Mark as synced
		replStatus.MarkSynced(snapshotID)
		if err := s.store.UpdateReplicationStatus(ctx, replStatus); err != nil {
			replicateLogger.Error().Err(err).Msg("failed to update replication status")
		}

		replicateLogger.Info().
			Str("snapshot_id", snapshotID).
			Msg("replication completed successfully")
	}
}

// failBackup marks a backup as failed and logs the error.
func (s *Scheduler) failBackup(ctx context.Context, backup *models.Backup, errMsg string, logger zerolog.Logger) {
	backup.Fail(errMsg)
	if err := s.store.UpdateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update backup record")
		return
	}
	logger.Error().Str("error", errMsg).Msg("backup failed")
}

// refreshLoop periodically reloads schedules from the database.
func (s *Scheduler) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()

			if !running {
				return
			}

			if err := s.Reload(ctx); err != nil {
				s.logger.Error().Err(err).Msg("failed to reload schedules")
			}
		}
	}
}

// TriggerBackup manually triggers a backup for the given schedule.
func (s *Scheduler) TriggerBackup(ctx context.Context, scheduleID uuid.UUID) error {
	schedules, err := s.store.GetEnabledSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get schedules: %w", err)
	}

	for _, schedule := range schedules {
		if schedule.ID == scheduleID {
			go s.executeBackup(schedule)
			return nil
		}
	}

	return fmt.Errorf("schedule not found: %s", scheduleID)
}

// GetActiveSchedules returns the number of active schedules.
func (s *Scheduler) GetActiveSchedules() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entries)
}

// GetNextRun returns the next scheduled run time for a schedule.
func (s *Scheduler) GetNextRun(scheduleID uuid.UUID) (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entryID, exists := s.entries[scheduleID]
	if !exists {
		return time.Time{}, false
	}

	entry := s.cron.Entry(entryID)
	if !entry.Valid() {
		return time.Time{}, false
	}

	return entry.Next, true
}
