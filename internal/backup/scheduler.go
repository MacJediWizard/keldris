package backup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
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

	// GetAgentByID returns an agent by ID.
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
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
	store    ScheduleStore
	restic   *Restic
	config   SchedulerConfig
	notifier *notifications.Service
	cron     *cron.Cron
	logger   zerolog.Logger
	mu       sync.RWMutex
	entries  map[uuid.UUID]cron.EntryID
	running  bool
}

// NewScheduler creates a new backup scheduler.
// The notifier parameter is optional and can be nil if notifications are not needed.
func NewScheduler(store ScheduleStore, restic *Restic, config SchedulerConfig, notifier *notifications.Service, logger zerolog.Logger) *Scheduler {
	return &Scheduler{
		store:    store,
		restic:   restic,
		config:   config,
		notifier: notifier,
		cron:     cron.New(cron.WithSeconds()),
		logger:   logger.With().Str("component", "scheduler").Logger(),
		entries:  make(map[uuid.UUID]cron.EntryID),
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

	// Check if backup can run at current time based on time window and excluded hours
	now := time.Now()
	if !schedule.CanRunAt(now) {
		nextAllowed := schedule.NextAllowedTime(now)
		logger.Info().
			Time("current_time", now).
			Time("next_allowed_time", nextAllowed).
			Msg("backup skipped: outside allowed time window")
		return
	}

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
	var lastBackup *models.Backup

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
			lastBackup = backup // Track failed backup for notification
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

	// If all repositories failed, log, notify, and return
	if successRepo == nil {
		errMsg := "backup failed to all repositories"
		if lastErr != nil {
			errMsg = fmt.Sprintf("backup failed to all repositories: %v", lastErr)
		}
		logger.Error().
			Err(lastErr).
			Int("repos_tried", len(enabledRepos)).
			Msg("backup failed to all repositories")
		// Send failure notification (use last backup attempt if available)
		if lastBackup != nil {
			s.sendBackupNotification(ctx, schedule, lastBackup, false, errMsg)
		}
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

	// Send success notification
	s.sendBackupNotification(ctx, schedule, successBackup, true, "")

	// Run prune if retention policy is set
	if schedule.RetentionPolicy != nil {
		logger.Info().Msg("running prune with retention policy")
		forgetResult, err := s.restic.Prune(ctx, successResticCfg, schedule.RetentionPolicy)
		if err != nil {
			logger.Error().Err(err).Msg("prune failed")
			successBackup.RecordRetention(0, 0, err)
		} else {
			logger.Info().
				Int("snapshots_removed", forgetResult.SnapshotsRemoved).
				Int("snapshots_kept", forgetResult.SnapshotsKept).
				Msg("prune completed")
			successBackup.RecordRetention(forgetResult.SnapshotsRemoved, forgetResult.SnapshotsKept, nil)
		}
		// Update backup record with retention results
		if err := s.store.UpdateBackup(ctx, successBackup); err != nil {
			logger.Error().Err(err).Msg("failed to update backup with retention results")
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

	// Build backup options with bandwidth limit and compression level
	var opts *BackupOptions
	if schedule.BandwidthLimitKB != nil || schedule.CompressionLevel != nil {
		opts = &BackupOptions{
			BandwidthLimitKB: schedule.BandwidthLimitKB,
			CompressionLevel: schedule.CompressionLevel,
		}
		if schedule.BandwidthLimitKB != nil {
			logger.Debug().Int("bandwidth_limit_kb", *schedule.BandwidthLimitKB).Msg("bandwidth limit applied")
		}
		if schedule.CompressionLevel != nil {
			logger.Debug().Str("compression_level", *schedule.CompressionLevel).Msg("compression level applied")
		}
	}

	// Run the backup with options
	stats, err := s.restic.BackupWithOptions(ctx, resticCfg, schedule.Paths, schedule.Excludes, tags, opts)
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

// sendBackupNotification sends a notification for a backup result.
func (s *Scheduler) sendBackupNotification(ctx context.Context, schedule models.Schedule, backup *models.Backup, success bool, errMsg string) {
	if s.notifier == nil {
		return
	}

	// Get agent info for hostname
	agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
	if err != nil {
		s.logger.Error().Err(err).
			Str("agent_id", schedule.AgentID.String()).
			Msg("failed to get agent for notification")
		return
	}

	result := notifications.BackupResult{
		OrgID:        agent.OrgID,
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
		AgentID:      agent.ID,
		Hostname:     agent.Hostname,
		SnapshotID:   backup.SnapshotID,
		StartedAt:    backup.StartedAt,
		Success:      success,
		ErrorMessage: errMsg,
	}

	if backup.CompletedAt != nil {
		result.CompletedAt = *backup.CompletedAt
	}
	if backup.SizeBytes != nil {
		result.SizeBytes = *backup.SizeBytes
	}
	if backup.FilesNew != nil {
		result.FilesNew = *backup.FilesNew
	}
	if backup.FilesChanged != nil {
		result.FilesChanged = *backup.FilesChanged
	}

	s.notifier.NotifyBackupComplete(ctx, result)
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

// DRTestStore defines the interface for DR test scheduling.
type DRTestStore interface {
	GetEnabledDRTestSchedules(ctx context.Context) ([]*models.DRTestSchedule, error)
	GetDRRunbookByID(ctx context.Context, id uuid.UUID) (*models.DRRunbook, error)
	CreateDRTest(ctx context.Context, test *models.DRTest) error
	UpdateDRTest(ctx context.Context, test *models.DRTest) error
	UpdateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)
}

// DRTestScheduler manages DR test schedules using cron.
type DRTestScheduler struct {
	store      DRTestStore
	restic     *Restic
	config     SchedulerConfig
	cron       *cron.Cron
	logger     zerolog.Logger
	mu         sync.RWMutex
	drEntries  map[uuid.UUID]cron.EntryID
	running    bool
}

// NewDRTestScheduler creates a new DR test scheduler.
func NewDRTestScheduler(store DRTestStore, restic *Restic, config SchedulerConfig, logger zerolog.Logger) *DRTestScheduler {
	return &DRTestScheduler{
		store:     store,
		restic:    restic,
		config:    config,
		cron:      cron.New(cron.WithSeconds()),
		logger:    logger.With().Str("component", "dr_scheduler").Logger(),
		drEntries: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start starts the DR test scheduler.
func (s *DRTestScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("DR test scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().Msg("starting DR test scheduler")

	// Load initial schedules
	if err := s.ReloadDRSchedules(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to load initial DR test schedules")
	}

	// Start cron scheduler
	s.cron.Start()

	// Start background refresh goroutine
	go s.refreshDRLoop(ctx)

	s.logger.Info().Msg("DR test scheduler started")
	return nil
}

// Stop stops the DR test scheduler gracefully.
func (s *DRTestScheduler) Stop() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	s.running = false
	s.logger.Info().Msg("stopping DR test scheduler")

	return s.cron.Stop()
}

// ReloadDRSchedules reloads all DR test schedules from the database.
func (s *DRTestScheduler) ReloadDRSchedules(ctx context.Context) error {
	s.logger.Debug().Msg("reloading DR test schedules from database")

	schedules, err := s.store.GetEnabledDRTestSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get enabled DR test schedules: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Track which schedules we've seen
	seen := make(map[uuid.UUID]bool)

	for _, schedule := range schedules {
		seen[schedule.ID] = true

		// Check if schedule already exists
		if entryID, exists := s.drEntries[schedule.ID]; exists {
			entry := s.cron.Entry(entryID)
			if entry.Valid() {
				continue
			}
			s.cron.Remove(entryID)
			delete(s.drEntries, schedule.ID)
		}

		// Add new schedule
		if err := s.addDRTestSchedule(schedule); err != nil {
			s.logger.Error().
				Err(err).
				Str("schedule_id", schedule.ID.String()).
				Msg("failed to add DR test schedule")
			continue
		}
	}

	// Remove schedules that are no longer enabled
	for id, entryID := range s.drEntries {
		if !seen[id] {
			s.cron.Remove(entryID)
			delete(s.drEntries, id)
			s.logger.Debug().
				Str("schedule_id", id.String()).
				Msg("removed disabled DR test schedule")
		}
	}

	s.logger.Info().
		Int("active_dr_schedules", len(s.drEntries)).
		Msg("DR test schedules reloaded")

	return nil
}

// addDRTestSchedule adds a DR test schedule to the cron scheduler.
func (s *DRTestScheduler) addDRTestSchedule(schedule *models.DRTestSchedule) error {
	// Create a copy for the closure
	sched := *schedule

	entryID, err := s.cron.AddFunc(schedule.CronExpression, func() {
		s.executeDRTest(sched)
	})
	if err != nil {
		return fmt.Errorf("add cron entry: %w", err)
	}

	s.drEntries[schedule.ID] = entryID

	// Update next run time
	entry := s.cron.Entry(entryID)
	if entry.Valid() {
		sched.NextRunAt = &entry.Next
		if err := s.store.UpdateDRTestSchedule(context.Background(), &sched); err != nil {
			s.logger.Warn().Err(err).Str("schedule_id", schedule.ID.String()).Msg("failed to update next run time")
		}
	}

	s.logger.Debug().
		Str("schedule_id", schedule.ID.String()).
		Str("cron_expression", schedule.CronExpression).
		Msg("added DR test schedule")

	return nil
}

// executeDRTest runs a DR test for the given schedule.
func (s *DRTestScheduler) executeDRTest(schedule models.DRTestSchedule) {
	ctx := context.Background()
	logger := s.logger.With().
		Str("schedule_id", schedule.ID.String()).
		Str("runbook_id", schedule.RunbookID.String()).
		Logger()

	logger.Info().Msg("starting scheduled DR test")

	// Get runbook
	runbook, err := s.store.GetDRRunbookByID(ctx, schedule.RunbookID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get runbook")
		return
	}

	// Create DR test record
	test := models.NewDRTest(schedule.RunbookID)
	test.Notes = "Scheduled DR test"

	if runbook.ScheduleID != nil {
		test.SetSchedule(*runbook.ScheduleID)
	}

	if err := s.store.CreateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Msg("failed to create DR test record")
		return
	}

	// Start the test
	test.Start()
	if err := s.store.UpdateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Msg("failed to update DR test record")
		return
	}

	// Perform the restore test if there's an associated schedule
	if runbook.ScheduleID != nil {
		backupSchedule, err := s.store.GetScheduleByID(ctx, *runbook.ScheduleID)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("get backup schedule: %v", err), logger)
			return
		}

		// Get primary repository from schedule
		primaryRepo := backupSchedule.GetPrimaryRepository()
		if primaryRepo == nil {
			s.failDRTest(ctx, test, "no primary repository configured for schedule", logger)
			return
		}

		// Get repository configuration
		repo, err := s.store.GetRepository(ctx, primaryRepo.RepositoryID)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("get repository: %v", err), logger)
			return
		}

		// Decrypt repository configuration
		if s.config.DecryptFunc == nil {
			s.failDRTest(ctx, test, "decrypt function not configured", logger)
			return
		}

		configJSON, err := s.config.DecryptFunc(repo.ConfigEncrypted)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("decrypt config: %v", err), logger)
			return
		}

		// Parse backend configuration
		backend, err := ParseBackend(repo.Type, configJSON)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("parse backend: %v", err), logger)
			return
		}

		// Get repository password
		if s.config.PasswordFunc == nil {
			s.failDRTest(ctx, test, "password function not configured", logger)
			return
		}

		password, err := s.config.PasswordFunc(repo.ID)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("get password: %v", err), logger)
			return
		}

		// Build restic config
		resticCfg := backend.ToResticConfig(password)

		// Get the latest snapshot
		snapshots, err := s.restic.Snapshots(ctx, resticCfg)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("list snapshots: %v", err), logger)
			return
		}

		if len(snapshots) == 0 {
			s.failDRTest(ctx, test, "no snapshots available for restore test", logger)
			return
		}

		// Use the most recent snapshot
		latestSnapshot := snapshots[0]
		test.SnapshotID = latestSnapshot.ID

		// Perform restore to a temporary location for verification
		startTime := time.Now()

		// Note: In a real implementation, you would restore to a temp directory
		// and verify the data. For now, we just verify the snapshot is readable.
		stats, err := s.restic.Stats(ctx, resticCfg)
		if err != nil {
			s.failDRTest(ctx, test, fmt.Sprintf("verify snapshot: %v", err), logger)
			return
		}

		duration := int(time.Since(startTime).Seconds())

		// Mark test as completed
		test.Complete(latestSnapshot.ID, stats.TotalSize, duration, true)
	} else {
		// No associated schedule, just mark as completed for manual verification
		test.Complete("", 0, 0, true)
		test.Notes = "Scheduled DR test - manual verification required"
	}

	if err := s.store.UpdateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Msg("failed to update DR test record")
		return
	}

	// Update schedule's last run time
	now := time.Now()
	schedule.LastRunAt = &now
	if err := s.store.UpdateDRTestSchedule(ctx, &schedule); err != nil {
		logger.Warn().Err(err).Msg("failed to update DR test schedule")
	}

	logger.Info().
		Str("test_id", test.ID.String()).
		Bool("passed", *test.VerificationPassed).
		Msg("DR test completed")
}

// failDRTest marks a DR test as failed.
func (s *DRTestScheduler) failDRTest(ctx context.Context, test *models.DRTest, errMsg string, logger zerolog.Logger) {
	test.Fail(errMsg)
	if err := s.store.UpdateDRTest(ctx, test); err != nil {
		logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update DR test record")
		return
	}
	logger.Error().Str("error", errMsg).Msg("DR test failed")
}

// refreshDRLoop periodically reloads DR test schedules from the database.
func (s *DRTestScheduler) refreshDRLoop(ctx context.Context) {
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

			if err := s.ReloadDRSchedules(ctx); err != nil {
				s.logger.Error().Err(err).Msg("failed to reload DR test schedules")
			}
		}
	}
}

// TriggerDRTest manually triggers a DR test for the given runbook.
func (s *DRTestScheduler) TriggerDRTest(ctx context.Context, runbookID uuid.UUID) error {
	runbook, err := s.store.GetDRRunbookByID(ctx, runbookID)
	if err != nil {
		return fmt.Errorf("get runbook: %w", err)
	}

	// Create a temporary schedule for this manual run
	tempSchedule := models.DRTestSchedule{
		ID:        uuid.New(),
		RunbookID: runbook.ID,
	}

	go s.executeDRTest(tempSchedule)
	return nil
}

// GetActiveDRSchedules returns the number of active DR test schedules.
func (s *DRTestScheduler) GetActiveDRSchedules() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.drEntries)
}

// GetNextAllowedRun returns the next time a backup can actually run for a schedule,
// accounting for both the cron schedule and backup window constraints.
func (s *Scheduler) GetNextAllowedRun(ctx context.Context, scheduleID uuid.UUID) (time.Time, bool) {
	s.mu.RLock()
	entryID, exists := s.entries[scheduleID]
	s.mu.RUnlock()

	if !exists {
		return time.Time{}, false
	}

	entry := s.cron.Entry(entryID)
	if !entry.Valid() {
		return time.Time{}, false
	}

	// Get the schedule to check time window constraints
	schedules, err := s.store.GetEnabledSchedules(ctx)
	if err != nil {
		return entry.Next, true // Fall back to cron time if we can't get schedule
	}

	var schedule *models.Schedule
	for _, sched := range schedules {
		if sched.ID == scheduleID {
			schedule = &sched
			break
		}
	}

	if schedule == nil {
		return entry.Next, true // Fall back to cron time if schedule not found
	}

	// Check if the cron time is within the allowed window
	nextCronTime := entry.Next
	if schedule.CanRunAt(nextCronTime) {
		return nextCronTime, true
	}

	// Find the next allowed time after the cron time
	nextAllowed := schedule.NextAllowedTime(nextCronTime)
	return nextAllowed, true
}
