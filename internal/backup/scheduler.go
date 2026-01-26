package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/maintenance"
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

	// GetEnabledBackupScriptsByScheduleID returns all enabled backup scripts for a schedule.
	GetEnabledBackupScriptsByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.BackupScript, error)

	// Checkpoint methods for resumable backups
	CheckpointStore
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
	store             ScheduleStore
	restic            *Restic
	config            SchedulerConfig
	notifier          *notifications.Service
	maintenance       *maintenance.Service
	checkpointManager *CheckpointManager
	largeFileScanner  *LargeFileScanner
	cron              *cron.Cron
	logger            zerolog.Logger
	mu                sync.RWMutex
	entries           map[uuid.UUID]cron.EntryID
	running           bool
}

// NewScheduler creates a new backup scheduler.
// The notifier parameter is optional and can be nil if notifications are not needed.
func NewScheduler(store ScheduleStore, restic *Restic, config SchedulerConfig, notifier *notifications.Service, logger zerolog.Logger) *Scheduler {
	// Initialize checkpoint manager
	checkpointConfig := DefaultCheckpointConfig()
	checkpointManager := NewCheckpointManager(store, checkpointConfig, logger)

	return &Scheduler{
		store:             store,
		restic:            restic,
		config:            config,
		notifier:          notifier,
		checkpointManager: checkpointManager,
		largeFileScanner:  NewLargeFileScanner(logger),
		cron:              cron.New(cron.WithSeconds()),
		logger:            logger.With().Str("component", "scheduler").Logger(),
		entries:           make(map[uuid.UUID]cron.EntryID),
	}
}

// SetMaintenanceService sets the maintenance service for checking maintenance windows.
// This should be called before Start() if maintenance mode checking is desired.
func (s *Scheduler) SetMaintenanceService(maint *maintenance.Service) {
	s.maintenance = maint
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

	// Start checkpoint manager
	if err := s.checkpointManager.Start(ctx); err != nil {
		s.logger.Error().Err(err).Msg("failed to start checkpoint manager")
	}

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

	// Stop checkpoint manager
	s.checkpointManager.Stop()

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

	// Check if maintenance mode is active for the agent's organization
	if s.maintenance != nil {
		agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get agent for maintenance check")
			return
		}
		if s.maintenance.IsMaintenanceActive(agent.OrgID) {
			logger.Info().
				Str("agent_id", agent.ID.String()).
				Str("org_id", agent.OrgID.String()).
				Msg("backup skipped: maintenance mode active")
			return
		}
	}

	// Check network mount availability for scheduled paths
	mountErr := s.checkNetworkMounts(ctx, schedule, logger)
	if mountErr != nil {
		if schedule.OnMountUnavailable == models.MountBehaviorSkip {
			logger.Info().Err(mountErr).Msg("backup skipped: network mount unavailable")
			return
		}
		// Create backup record and mark as failed
		// Get primary repository ID for the backup record
		var repoID *uuid.UUID
		if primaryRepo := schedule.GetPrimaryRepository(); primaryRepo != nil {
			repoID = &primaryRepo.RepositoryID
		}
		backup := models.NewBackup(schedule.ID, schedule.AgentID, repoID)
		if err := s.store.CreateBackup(ctx, backup); err != nil {
			logger.Error().Err(err).Msg("failed to create backup record")
			return
		}
		s.failBackup(ctx, backup, fmt.Sprintf("network mount unavailable: %v", mountErr), logger)
		return
	}

	logger.Info().Msg("starting scheduled backup")

	// Get enabled repositories sorted by priority
	enabledRepos := schedule.GetEnabledRepositories()
	if len(enabledRepos) == 0 {
		logger.Error().Msg("no enabled repositories for schedule")
		return
	}

	// Load backup scripts for this schedule
	scripts, err := s.store.GetEnabledBackupScriptsByScheduleID(ctx, schedule.ID)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to load backup scripts, continuing without scripts")
		scripts = nil
	}

	// Create a temporary backup record for pre-script execution
	preScriptBackup := &models.Backup{
		ID:         uuid.New(),
		ScheduleID: schedule.ID,
		AgentID:    schedule.AgentID,
		Status:     models.BackupStatusRunning,
		StartedAt:  time.Now(),
	}

	// Run pre-backup script if configured
	if err := s.runPreBackupScript(ctx, scripts, preScriptBackup, logger); err != nil {
		// Create a backup record to record the pre-script failure
		preScriptBackup.Fail(fmt.Sprintf("pre-backup script failed: %v", err))
		if createErr := s.store.CreateBackup(ctx, preScriptBackup); createErr != nil {
			logger.Error().Err(createErr).Msg("failed to create backup record for pre-script failure")
		}
		s.runPostBackupScripts(ctx, scripts, preScriptBackup, false, logger)
		// Update the backup record with post-script output
		if updateErr := s.store.UpdateBackup(ctx, preScriptBackup); updateErr != nil {
			logger.Error().Err(updateErr).Msg("failed to update backup record with post-script output")
		}
		s.sendBackupNotification(ctx, schedule, preScriptBackup, false, fmt.Sprintf("pre-backup script failed: %v", err))
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
			backup, stats, resticCfg, err := s.runBackupToRepo(ctx, schedule, schedRepo, scripts, attempt, repoLogger)
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
		// Run post-backup scripts (failure)
		if lastBackup != nil {
			s.runPostBackupScripts(ctx, scripts, lastBackup, false, logger)
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
	scripts []*models.BackupScript,
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

	// Build backup options with bandwidth limit, compression level, and max file size
	var opts *BackupOptions
	if schedule.BandwidthLimitKB != nil || schedule.CompressionLevel != nil || schedule.MaxFileSizeMB != nil {
		opts = &BackupOptions{
			BandwidthLimitKB: schedule.BandwidthLimitKB,
			CompressionLevel: schedule.CompressionLevel,
			MaxFileSizeMB:    schedule.MaxFileSizeMB,
		}
		if schedule.BandwidthLimitKB != nil {
			logger.Debug().Int("bandwidth_limit_kb", *schedule.BandwidthLimitKB).Msg("bandwidth limit applied")
		}
		if schedule.CompressionLevel != nil {
			logger.Debug().Str("compression_level", *schedule.CompressionLevel).Msg("compression level applied")
		}
		if schedule.MaxFileSizeMB != nil && *schedule.MaxFileSizeMB > 0 {
			logger.Debug().Int("max_file_size_mb", *schedule.MaxFileSizeMB).Msg("max file size limit applied")
		}
	}

	// Scan for large files if max file size is configured
	if schedule.MaxFileSizeMB != nil && *schedule.MaxFileSizeMB > 0 {
		scanResult, err := s.largeFileScanner.Scan(ctx, schedule.Paths, schedule.Excludes, *schedule.MaxFileSizeMB)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to scan for large files, proceeding with backup")
		} else if scanResult.TotalExcluded > 0 {
			logger.Warn().
				Int("files_excluded", scanResult.TotalExcluded).
				Int64("total_size_mb", scanResult.TotalSizeMB).
				Msg("large files will be excluded from backup")

			// Convert scan results to model type and record on backup
			excludedFiles := make([]models.ExcludedLargeFile, len(scanResult.LargeFiles))
			for i, f := range scanResult.LargeFiles {
				excludedFiles[i] = models.ExcludedLargeFile{
					Path:      f.Path,
					SizeBytes: f.SizeBytes,
					SizeMB:    f.SizeMB,
				}
			}
			backup.RecordExcludedLargeFiles(excludedFiles)
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

	// Run post-backup scripts (success)
	s.runPostBackupScripts(ctx, scripts, backup, true, logger)

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

// checkNetworkMounts validates that all network mounts required for the backup are available.
// Returns nil if all mounts are available, or an error describing the unavailable mount.
func (s *Scheduler) checkNetworkMounts(ctx context.Context, schedule models.Schedule, logger zerolog.Logger) error {
	// Get the agent to access its network mounts
	agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}

	// If agent has no network mounts reported, skip the check
	if len(agent.NetworkMounts) == 0 {
		return nil
	}

	nd := NewNetworkDrives(s.logger)

	// Check each path in the schedule
	for _, path := range schedule.Paths {
		ok, mount, err := nd.ValidateMountForBackup(ctx, path, agent.NetworkMounts)
		if !ok {
			return err
		}
		if mount != nil {
			logger.Debug().
				Str("path", path).
				Str("mount", mount.Path).
				Str("status", string(mount.Status)).
				Msg("network mount validated")
		}
	}

	return nil
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

			// Refresh maintenance window cache and check for pending notifications
			if s.maintenance != nil {
				if err := s.maintenance.RefreshCache(ctx); err != nil {
					s.logger.Error().Err(err).Msg("failed to refresh maintenance cache")
				}
				s.maintenance.CheckAndSendNotifications(ctx)
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

// GetIncompleteBackups returns all incomplete backups for an agent that can be resumed.
func (s *Scheduler) GetIncompleteBackups(ctx context.Context, agentID uuid.UUID) ([]*models.BackupCheckpoint, error) {
	return s.checkpointManager.GetIncompleteBackups(ctx, agentID)
}

// GetResumeInfo returns information about a resumable backup checkpoint.
func (s *Scheduler) GetResumeInfo(ctx context.Context, checkpointID uuid.UUID) (*ResumeInfo, error) {
	return s.checkpointManager.GetResumeInfo(ctx, checkpointID)
}

// CancelCheckpoint cancels a checkpoint, making it non-resumable.
// The next backup for this schedule will start fresh.
func (s *Scheduler) CancelCheckpoint(ctx context.Context, checkpointID uuid.UUID) error {
	return s.checkpointManager.CancelCheckpoint(ctx, checkpointID)
}

// ResumeBackup resumes an interrupted backup from a checkpoint.
func (s *Scheduler) ResumeBackup(ctx context.Context, checkpointID uuid.UUID) error {
	// Get the checkpoint
	checkpoint, err := s.store.GetCheckpointByID(ctx, checkpointID)
	if err != nil {
		return fmt.Errorf("get checkpoint: %w", err)
	}

	if !checkpoint.IsResumable() {
		return errors.New("checkpoint is not resumable")
	}

	// Get the schedule
	schedules, err := s.store.GetEnabledSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get schedules: %w", err)
	}

	var schedule *models.Schedule
	for i := range schedules {
		if schedules[i].ID == checkpoint.ScheduleID {
			schedule = &schedules[i]
			break
		}
	}

	if schedule == nil {
		return fmt.Errorf("schedule not found: %s", checkpoint.ScheduleID)
	}

	// Prepare the checkpoint for resume
	if err := s.checkpointManager.PrepareResume(ctx, checkpoint); err != nil {
		return fmt.Errorf("prepare resume: %w", err)
	}

	// Execute the resumed backup in background
	go s.executeResumedBackup(*schedule, checkpoint)

	return nil
}

// executeResumedBackup runs a backup that is being resumed from a checkpoint.
func (s *Scheduler) executeResumedBackup(schedule models.Schedule, checkpoint *models.BackupCheckpoint) {
	ctx := context.Background()
	logger := s.logger.With().
		Str("schedule_id", schedule.ID.String()).
		Str("schedule_name", schedule.Name).
		Str("checkpoint_id", checkpoint.ID.String()).
		Int("resume_count", checkpoint.ResumeCount).
		Logger()

	logger.Info().Msg("resuming backup from checkpoint")

	// Get enabled repositories sorted by priority
	enabledRepos := schedule.GetEnabledRepositories()
	if len(enabledRepos) == 0 {
		logger.Error().Msg("no enabled repositories for schedule")
		return
	}

	// Find the repository that matches the checkpoint
	var targetRepo *models.ScheduleRepository
	for i := range enabledRepos {
		if enabledRepos[i].RepositoryID == checkpoint.RepositoryID {
			targetRepo = &enabledRepos[i]
			break
		}
	}

	if targetRepo == nil {
		logger.Error().
			Str("repository_id", checkpoint.RepositoryID.String()).
			Msg("checkpoint repository not found in enabled repositories")
		return
	}

	// Create a resumed backup record
	backup := models.NewResumedBackup(schedule.ID, schedule.AgentID, &targetRepo.RepositoryID, checkpoint.ID, checkpoint.BackupID)
	if err := s.store.CreateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to create resumed backup record")
		return
	}

	// Associate the new backup with the checkpoint
	if err := s.checkpointManager.AssociateBackup(ctx, checkpoint.ID, backup.ID); err != nil {
		logger.Warn().Err(err).Msg("failed to associate backup with checkpoint")
	}

	// Track this backup with the checkpoint manager
	s.checkpointManager.TrackBackup(backup.ID, checkpoint)

	// Get repository configuration
	repo, err := s.store.GetRepository(ctx, targetRepo.RepositoryID)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("get repository: %v", err), logger)
		s.checkpointManager.InterruptBackup(ctx, backup.ID, err.Error())
		return
	}

	// Decrypt repository configuration
	if s.config.DecryptFunc == nil {
		s.failBackup(ctx, backup, "decrypt function not configured", logger)
		s.checkpointManager.InterruptBackup(ctx, backup.ID, "decrypt function not configured")
		return
	}

	configJSON, err := s.config.DecryptFunc(repo.ConfigEncrypted)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("decrypt config: %v", err), logger)
		s.checkpointManager.InterruptBackup(ctx, backup.ID, err.Error())
		return
	}

	// Parse backend configuration
	backend, err := ParseBackend(repo.Type, configJSON)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("parse backend: %v", err), logger)
		s.checkpointManager.InterruptBackup(ctx, backup.ID, err.Error())
		return
	}

	// Get repository password
	if s.config.PasswordFunc == nil {
		s.failBackup(ctx, backup, "password function not configured", logger)
		s.checkpointManager.InterruptBackup(ctx, backup.ID, "password function not configured")
		return
	}

	password, err := s.config.PasswordFunc(repo.ID)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("get password: %v", err), logger)
		s.checkpointManager.InterruptBackup(ctx, backup.ID, err.Error())
		return
	}

	// Build restic config
	resticCfg := backend.ToResticConfig(password)

	// Build tags for the backup
	tags := []string{
		fmt.Sprintf("schedule:%s", schedule.ID.String()),
		fmt.Sprintf("agent:%s", schedule.AgentID.String()),
		"resumed",
	}

	// Build backup options with bandwidth limit and compression level
	var opts *BackupOptions
	if schedule.BandwidthLimitKB != nil || schedule.CompressionLevel != nil {
		opts = &BackupOptions{
			BandwidthLimitKB: schedule.BandwidthLimitKB,
			CompressionLevel: schedule.CompressionLevel,
		}
	}

	// Run the backup with options
	// Note: For a true resume, restic would need to use the --parent flag or other mechanisms
	// to continue from where it left off. This implementation starts a new backup but the
	// checkpoint tracking allows monitoring progress and resuming from failures.
	stats, err := s.restic.BackupWithOptions(ctx, resticCfg, schedule.Paths, schedule.Excludes, tags, opts)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("backup failed: %v", err), logger)
		s.checkpointManager.InterruptBackup(ctx, backup.ID, err.Error())
		return
	}

	// Mark backup as completed
	backup.Complete(stats.SnapshotID, stats.SizeBytes, stats.FilesNew, stats.FilesChanged)

	// Complete the checkpoint
	if err := s.checkpointManager.CompleteBackup(ctx, backup.ID); err != nil {
		logger.Warn().Err(err).Msg("failed to complete checkpoint")
	}

	if err := s.store.UpdateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to update backup record")
	}

	logger.Info().
		Str("snapshot_id", stats.SnapshotID).
		Int("files_new", stats.FilesNew).
		Int("files_changed", stats.FilesChanged).
		Int64("size_bytes", stats.SizeBytes).
		Dur("duration", stats.Duration).
		Msg("resumed backup completed successfully")

	// Send success notification
	s.sendBackupNotification(ctx, schedule, backup, true, "")

	// Run prune if retention policy is set
	if schedule.RetentionPolicy != nil {
		logger.Info().Msg("running prune with retention policy")
		forgetResult, err := s.restic.Prune(ctx, resticCfg, schedule.RetentionPolicy)
		if err != nil {
			logger.Error().Err(err).Msg("prune failed")
			backup.RecordRetention(0, 0, err)
		} else {
			logger.Info().
				Int("snapshots_removed", forgetResult.SnapshotsRemoved).
				Int("snapshots_kept", forgetResult.SnapshotsKept).
				Msg("prune completed")
			backup.RecordRetention(forgetResult.SnapshotsRemoved, forgetResult.SnapshotsKept, nil)
		}
		// Update backup record with retention results
		if err := s.store.UpdateBackup(ctx, backup); err != nil {
			logger.Error().Err(err).Msg("failed to update backup with retention results")
		}
	}
}

// GetCheckpointManager returns the checkpoint manager for external use.
func (s *Scheduler) GetCheckpointManager() *CheckpointManager {
	return s.checkpointManager
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

// runScript executes a backup script with the given timeout.
func (s *Scheduler) runScript(ctx context.Context, script *models.BackupScript, logger zerolog.Logger) (string, error) {
	logger.Info().
		Str("script_type", string(script.Type)).
		Int("timeout_seconds", script.TimeoutSeconds).
		Msg("running backup script")

	// Create context with timeout
	scriptCtx, cancel := context.WithTimeout(ctx, time.Duration(script.TimeoutSeconds)*time.Second)
	defer cancel()

	// Execute the script using sh -c
	cmd := exec.CommandContext(scriptCtx, "sh", "-c", script.Script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Combine stdout and stderr for output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Truncate output if too long (max 64KB)
	const maxOutputLen = 64 * 1024
	if len(output) > maxOutputLen {
		output = output[:maxOutputLen] + "\n... (output truncated)"
	}

	if err != nil {
		if scriptCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("script timed out after %d seconds", script.TimeoutSeconds)
		}
		return output, fmt.Errorf("script failed: %w", err)
	}

	logger.Info().
		Str("script_type", string(script.Type)).
		Msg("backup script completed successfully")

	return output, nil
}

// getScriptsByType returns scripts of the specified type from the scripts list.
func getScriptsByType(scripts []*models.BackupScript, scriptType models.BackupScriptType) *models.BackupScript {
	for _, script := range scripts {
		if script.Type == scriptType {
			return script
		}
	}
	return nil
}

// runPreBackupScript runs the pre-backup script if configured.
func (s *Scheduler) runPreBackupScript(ctx context.Context, scripts []*models.BackupScript, backup *models.Backup, logger zerolog.Logger) error {
	script := getScriptsByType(scripts, models.BackupScriptTypePreBackup)
	if script == nil {
		return nil
	}

	output, err := s.runScript(ctx, script, logger)
	backup.RecordPreScript(output, err)

	if err != nil && script.FailOnError {
		return fmt.Errorf("pre-backup script failed: %w", err)
	}

	return nil
}

// runPostBackupScripts runs the appropriate post-backup scripts based on backup success.
func (s *Scheduler) runPostBackupScripts(ctx context.Context, scripts []*models.BackupScript, backup *models.Backup, success bool, logger zerolog.Logger) {
	var scriptsToRun []*models.BackupScript

	// Always run post_always scripts
	if script := getScriptsByType(scripts, models.BackupScriptTypePostAlways); script != nil {
		scriptsToRun = append(scriptsToRun, script)
	}

	// Run success or failure script based on outcome
	if success {
		if script := getScriptsByType(scripts, models.BackupScriptTypePostSuccess); script != nil {
			scriptsToRun = append(scriptsToRun, script)
		}
	} else {
		if script := getScriptsByType(scripts, models.BackupScriptTypePostFailure); script != nil {
			scriptsToRun = append(scriptsToRun, script)
		}
	}

	if len(scriptsToRun) == 0 {
		return
	}

	var combinedOutput string
	var combinedError error

	for _, script := range scriptsToRun {
		output, err := s.runScript(ctx, script, logger)
		if combinedOutput != "" && output != "" {
			combinedOutput += "\n--- " + string(script.Type) + " ---\n"
		}
		combinedOutput += output

		if err != nil {
			if combinedError == nil {
				combinedError = err
			} else {
				combinedError = fmt.Errorf("%v; %w", combinedError, err)
			}
			logger.Warn().Err(err).Str("script_type", string(script.Type)).Msg("post-backup script failed")
		}
	}

	backup.RecordPostScript(combinedOutput, combinedError)
}

// PriorityQueue manages backup jobs with priority ordering.
// Higher priority jobs (lower number) are processed first.
type PriorityQueue struct {
	store    ScheduleStore
	logger   zerolog.Logger
	mu       sync.RWMutex
	running  map[uuid.UUID]*models.BackupQueueItem // Currently running backups by agent ID
}

// NewPriorityQueue creates a new priority queue.
func NewPriorityQueue(store ScheduleStore, logger zerolog.Logger) *PriorityQueue {
	return &PriorityQueue{
		store:   store,
		logger:  logger.With().Str("component", "priority_queue").Logger(),
		running: make(map[uuid.UUID]*models.BackupQueueItem),
	}
}

// EnqueueBackup adds a backup to the priority queue.
func (pq *PriorityQueue) EnqueueBackup(ctx context.Context, schedule *models.Schedule) (*models.BackupQueueItem, error) {
	item := models.NewBackupQueueItem(schedule.ID, schedule.AgentID, schedule.Priority)

	pq.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("agent_id", schedule.AgentID.String()).
		Int("priority", int(schedule.Priority)).
		Msg("enqueuing backup")

	return item, nil
}

// GetNextPending returns the next pending backup for the given agent, ordered by priority.
func (pq *PriorityQueue) GetNextPending(ctx context.Context, agentID uuid.UUID) (*models.BackupQueueItem, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	// In a full implementation, this would query the backup_queue table
	// ordered by priority ASC, queued_at ASC
	return nil, nil
}

// CanPreempt checks if a new backup can preempt the currently running backup.
func (pq *PriorityQueue) CanPreempt(newPriority, runningPriority models.SchedulePriority, runningPreemptible bool) bool {
	// Only preempt if:
	// 1. The running backup is marked as preemptible
	// 2. The new backup has higher priority (lower number)
	return runningPreemptible && newPriority < runningPriority
}

// StartBackup marks a backup as running.
func (pq *PriorityQueue) StartBackup(ctx context.Context, item *models.BackupQueueItem) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	now := time.Now()
	item.Status = models.QueueStatusRunning
	item.StartedAt = &now
	item.UpdatedAt = now

	pq.running[item.AgentID] = item

	pq.logger.Info().
		Str("item_id", item.ID.String()).
		Str("agent_id", item.AgentID.String()).
		Int("priority", int(item.Priority)).
		Msg("backup started")

	return nil
}

// CompleteBackup marks a backup as completed.
func (pq *PriorityQueue) CompleteBackup(ctx context.Context, item *models.BackupQueueItem, success bool) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	now := time.Now()
	item.CompletedAt = &now
	item.UpdatedAt = now

	if success {
		item.Status = models.QueueStatusCompleted
	} else {
		item.Status = models.QueueStatusFailed
	}

	delete(pq.running, item.AgentID)

	pq.logger.Info().
		Str("item_id", item.ID.String()).
		Str("agent_id", item.AgentID.String()).
		Bool("success", success).
		Msg("backup completed")

	return nil
}

// PreemptBackup preempts a running backup with a higher priority one.
func (pq *PriorityQueue) PreemptBackup(ctx context.Context, runningItem *models.BackupQueueItem, newItem *models.BackupQueueItem) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	now := time.Now()
	runningItem.Status = models.QueueStatusPreempted
	runningItem.PreemptedBy = &newItem.ID
	runningItem.UpdatedAt = now

	pq.logger.Warn().
		Str("preempted_id", runningItem.ID.String()).
		Str("preempting_id", newItem.ID.String()).
		Int("preempted_priority", int(runningItem.Priority)).
		Int("preempting_priority", int(newItem.Priority)).
		Msg("backup preempted by higher priority backup")

	return nil
}

// GetRunningBackup returns the currently running backup for an agent, if any.
func (pq *PriorityQueue) GetRunningBackup(agentID uuid.UUID) *models.BackupQueueItem {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return pq.running[agentID]
}

// GetQueueSummary returns a summary of the backup queue.
func (pq *PriorityQueue) GetQueueSummary(ctx context.Context) (*models.BackupQueueSummary, error) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	summary := &models.BackupQueueSummary{
		TotalRunning: len(pq.running),
	}

	// Count running backups by priority
	for _, item := range pq.running {
		switch item.Priority {
		case models.PriorityHigh:
			summary.HighPriority++
		case models.PriorityMedium:
			summary.MediumPriority++
		case models.PriorityLow:
			summary.LowPriority++
		}
	}

	return summary, nil
}

// GetPriorityLabel returns a human-readable label for a priority level.
func GetPriorityLabel(priority models.SchedulePriority) string {
	switch priority {
	case models.PriorityHigh:
		return "high"
	case models.PriorityMedium:
		return "medium"
	case models.PriorityLow:
		return "low"
	default:
		return "medium"
	}
}
