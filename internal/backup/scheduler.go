package backup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/backup/apps"
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

	// Validation methods for backup validation
	ValidationStore
	// GetAgentByID returns an agent by ID.
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
	store              ScheduleStore
	restic             *Restic
	config             SchedulerConfig
	notifier           *notifications.Service
	maintenance        *maintenance.Service
	checkpointManager  *CheckpointManager
	largeFileScanner   *LargeFileScanner
	concurrencyManager *ConcurrencyManager
	validator          *BackupValidator
	validationConfig   ValidationConfig
	cron               *cron.Cron
	logger             zerolog.Logger
	mu                 sync.RWMutex
	entries            map[uuid.UUID]cron.EntryID
	running            bool
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

// SetConcurrencyManager sets the concurrency manager for backup limits.
// This should be called before Start() if concurrency limiting is desired.
func (s *Scheduler) SetConcurrencyManager(cm *ConcurrencyManager) {
	s.concurrencyManager = cm
}

// SetBackupValidator sets the backup validator for automated validation after backups.
// This should be called before Start() if backup validation is desired.
func (s *Scheduler) SetBackupValidator(validator *BackupValidator) {
	s.validator = validator
}

// SetValidationConfig sets the validation configuration.
// This should be called before Start() if custom validation settings are desired.
func (s *Scheduler) SetValidationConfig(config ValidationConfig) {
	s.validationConfig = config
}

// EnableValidation enables backup validation with the given configuration.
// This creates a BackupValidator and configures it for the scheduler.
func (s *Scheduler) EnableValidation(config ValidationConfig) {
	s.validationConfig = config
	s.validator = NewBackupValidator(s.restic, config, s.store, s.logger)
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

// executeBackup runs a backup for the given schedule with retry/failover, replication, and scripts.
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

	// Check concurrency limits and queue if necessary
	if s.concurrencyManager != nil {
		agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get agent for concurrency check")
			return
		}
		acquired, queueEntry, err := s.concurrencyManager.AcquireSlot(ctx, agent.OrgID, agent.ID, schedule.ID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to check concurrency limits")
			return
		}
		if !acquired {
			if queueEntry != nil {
				logger.Info().
					Int("queue_position", queueEntry.QueuePosition).
					Msg("backup queued due to concurrency limit")
			}
			return
		}
		// Slot acquired, ensure we release it when done
		defer func() {
			if err := s.concurrencyManager.ReleaseSlot(ctx, agent.OrgID, agent.ID); err != nil {
				logger.Error().Err(err).Msg("failed to release concurrency slot")
			}
		}()
	}

	// Check network mount availability for scheduled paths
	mountErr := s.checkNetworkMounts(ctx, schedule, logger)
	if mountErr != nil {
		if schedule.OnMountUnavailable == models.MountBehaviorSkip {
			logger.Info().Err(mountErr).Msg("backup skipped: network mount unavailable")
			return
		}
		// Create backup record and mark as failed
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

	// Handle Pi-hole specific backup
	if schedule.IsPiholeBackup() {
		agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get agent for Pi-hole backup")
			return
		}
		s.executePiholeBackup(ctx, schedule, agent, logger)
		return
	}

	// Handle Proxmox VM backup
	if schedule.IsProxmoxBackup() {
		agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
		if err != nil {
			logger.Error().Err(err).Msg("failed to get agent for Proxmox backup")
			return
		}
		s.executeProxmoxBackup(ctx, schedule, agent, logger)
		return
	}

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

	// Run pre-backup script if configured
	preScriptBackup := &models.Backup{
		ID:         uuid.New(),
		ScheduleID: schedule.ID,
		AgentID:    schedule.AgentID,
		Status:     models.BackupStatusRunning,
		StartedAt:  time.Now(),
	}
	if err := s.runPreBackupScript(ctx, scripts, preScriptBackup, logger); err != nil {
		preScriptBackup.Fail(fmt.Sprintf("pre-backup script failed: %v", err))
		if createErr := s.store.CreateBackup(ctx, preScriptBackup); createErr != nil {
			logger.Error().Err(createErr).Msg("failed to create backup record for pre-script failure")
		}
		s.runPostBackupScripts(ctx, scripts, preScriptBackup, false, logger)
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

	// Run post-backup scripts (success)
	s.runPostBackupScripts(ctx, scripts, successBackup, true, logger)

	logger.Info().
		Str("repository_id", successRepo.RepositoryID.String()).
		Str("snapshot_id", successStats.SnapshotID).
		Int("files_new", successStats.FilesNew).
		Int("files_changed", successStats.FilesChanged).
		Int64("size_bytes", successStats.SizeBytes).
		Dur("duration", successStats.Duration).
		Msg("backup completed successfully")

	// Run automated backup validation if enabled
	if s.validator != nil {
		s.runBackupValidation(ctx, successBackup, successResticCfg, schedule.Paths, logger)
	}

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
		return backup, nil, ResticConfig{}, fmt.Errorf("get repository: %w", err)
	}

	// Decrypt repository configuration
	if s.config.DecryptFunc == nil {
		s.failBackup(ctx, backup, "decrypt function not configured", logger)
		return backup, nil, ResticConfig{}, errors.New("decrypt function not configured")
	}

	configJSON, err := s.config.DecryptFunc(repo.ConfigEncrypted)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("decrypt config: %v", err), logger)
		return backup, nil, ResticConfig{}, fmt.Errorf("decrypt config: %w", err)
	}

	// Parse backend configuration
	backend, err := ParseBackend(repo.Type, configJSON)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("parse backend: %v", err), logger)
		return backup, nil, ResticConfig{}, fmt.Errorf("parse backend: %w", err)
	}

	// Get repository password
	if s.config.PasswordFunc == nil {
		s.failBackup(ctx, backup, "password function not configured", logger)
		return backup, nil, ResticConfig{}, errors.New("password function not configured")
	}

	password, err := s.config.PasswordFunc(repo.ID)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("get password: %v", err), logger)
		return backup, nil, ResticConfig{}, fmt.Errorf("get password: %w", err)
	}

	// Build restic config
	resticCfg := backend.ToResticConfig(password)

	// Build tags
	tags := []string{
		fmt.Sprintf("schedule:%s", schedule.ID.String()),
		fmt.Sprintf("agent:%s", schedule.AgentID.String()),
	}

	// Build backup options with bandwidth limit and compression
	var opts *BackupOptions
	if schedule.BandwidthLimitKB != nil || (schedule.CompressionLevel != nil && *schedule.CompressionLevel != "") {
		opts = &BackupOptions{}
		if schedule.BandwidthLimitKB != nil {
			opts.BandwidthLimitKB = schedule.BandwidthLimitKB
			logger.Debug().Int("bandwidth_limit_kb", *schedule.BandwidthLimitKB).Msg("bandwidth limit applied")
		}
		if schedule.CompressionLevel != nil && *schedule.CompressionLevel != "" {
			opts.CompressionLevel = schedule.CompressionLevel
		}
	}

	// Run the backup with options
	stats, err := s.restic.BackupWithOptions(ctx, resticCfg, schedule.Paths, schedule.Excludes, tags, opts)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("backup failed: %v", err), logger)
		return backup, nil, resticCfg, fmt.Errorf("backup failed: %w", err)
	}

	// Update backup record with success
	backup.Complete(stats.SnapshotID, stats.FilesNew, stats.FilesChanged, stats.SizeBytes)
	if err := s.store.UpdateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to update backup record")
	}

	return backup, stats, resticCfg, nil
}

// failBackup marks a backup as failed and updates the record.
func (s *Scheduler) failBackup(ctx context.Context, backup *models.Backup, errMsg string, logger zerolog.Logger) {
	backup.Fail(errMsg)
	if err := s.store.UpdateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update backup record")
	}
	logger.Error().Str("error", errMsg).Msg("backup failed")
}

// failBackupWithSchedule marks a backup as failed, runs post-scripts, and sends notification.
func (s *Scheduler) failBackupWithSchedule(ctx context.Context, backup *models.Backup, schedule models.Schedule, errMsg string, logger zerolog.Logger) {
	s.failBackup(ctx, backup, errMsg, logger)
	s.sendBackupNotification(ctx, schedule, backup, false, errMsg)
}

// sendBackupNotification sends a notification about a backup result.
func (s *Scheduler) sendBackupNotification(ctx context.Context, schedule models.Schedule, backup *models.Backup, success bool, errMsg string) {
	if s.notifier == nil {
		return
	}

	result := notifications.BackupResult{
		ScheduleID:   schedule.ID,
		ScheduleName: schedule.Name,
		AgentID:      schedule.AgentID,
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

// checkNetworkMounts verifies that network mounts used by the schedule paths are available.
func (s *Scheduler) checkNetworkMounts(ctx context.Context, schedule models.Schedule, logger zerolog.Logger) error {
	agent, err := s.store.GetAgentByID(ctx, schedule.AgentID)
	if err != nil {
		// If agent is not found, skip mount check
		logger.Debug().Err(err).Msg("could not get agent for mount check, skipping")
		return nil
	}

	if len(agent.NetworkMounts) == 0 {
		return nil
	}

	// Check if any backup paths use network mounts
	for _, backupPath := range schedule.Paths {
		for _, mount := range agent.NetworkMounts {
			if filepath.HasPrefix(backupPath, mount.Path) {
				if mount.Status != models.MountStatusConnected {
					return fmt.Errorf("mount %s is not connected (status: %s)", mount.Path, mount.Status)
				}
			}
		}
	}

	return nil
}

// runPreBackupScript runs pre-backup scripts for the schedule.
func (s *Scheduler) runPreBackupScript(ctx context.Context, scripts []*models.BackupScript, backup *models.Backup, logger zerolog.Logger) error {
	if len(scripts) == 0 {
		return nil
	}

	for _, script := range scripts {
		if !script.IsPreBackup() {
			continue
		}

		logger.Debug().Str("script_type", string(script.Type)).Msg("running pre-backup script")

		output, err := s.executeScript(ctx, script)
		backup.PreScriptOutput = output

		if err != nil {
			backup.PreScriptError = err.Error()
			if script.FailOnError {
				return fmt.Errorf("pre-backup script failed: %w", err)
			}
			logger.Warn().Err(err).Msg("pre-backup script failed but FailOnError is false, continuing")
		}
	}

	return nil
}

// runPostBackupScripts runs post-backup scripts for the schedule.
func (s *Scheduler) runPostBackupScripts(ctx context.Context, scripts []*models.BackupScript, backup *models.Backup, success bool, logger zerolog.Logger) {
	if len(scripts) == 0 {
		return
	}

	for _, script := range scripts {
		if !script.IsPostScript() {
			continue
		}

		// Check if this script should run based on backup result
		if success && !script.ShouldRunOnSuccess() {
			continue
		}
		if !success && !script.ShouldRunOnFailure() {
			continue
		}

		logger.Debug().
			Str("script_type", string(script.Type)).
			Bool("backup_success", success).
			Msg("running post-backup script")

		output, err := s.executeScript(ctx, script)
		backup.PostScriptOutput = output
		if err != nil {
			backup.PostScriptError = err.Error()
			logger.Warn().Err(err).Msg("post-backup script failed")
		}
	}
}

// executeScript executes a backup script and returns its output.
func (s *Scheduler) executeScript(ctx context.Context, script *models.BackupScript) (string, error) {
	timeout := time.Duration(script.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", script.Script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("script execution failed: %s: %w", stderr.String(), err)
	}

	return stdout.String(), nil
}

// executePiholeBackup handles Pi-hole specific backup execution.
func (s *Scheduler) executePiholeBackup(ctx context.Context, schedule models.Schedule, agent *models.Agent, logger zerolog.Logger) {
	logger.Info().Msg("executing Pi-hole backup")

	// Get the primary repository for this schedule
	enabledRepos := schedule.GetEnabledRepositories()
	if len(enabledRepos) == 0 {
		logger.Error().Msg("no enabled repositories for Pi-hole backup")
		return
	}

	primaryRepo := &enabledRepos[0]
	backup := models.NewPiholeBackup(schedule.ID, schedule.AgentID, &primaryRepo.RepositoryID, "")
	if err := s.store.CreateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to create Pi-hole backup record")
		return
	}

	// Use the apps package to perform the Pi-hole backup
	piholeApp := apps.NewPiholeBackup(logger)
	piholeResult, err := piholeApp.Backup(ctx, "/tmp/keldris-pihole-backup")
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("Pi-hole backup failed: %v", err), logger)
		return
	}

	logger.Info().
		Int64("size_bytes", piholeResult.SizeBytes).
		Msg("Pi-hole backup completed")

	// Now back up the Pi-hole data to the restic repository
	repo, err := s.store.GetRepository(ctx, primaryRepo.RepositoryID)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("get repository: %v", err), logger)
		return
	}

	if s.config.DecryptFunc == nil || s.config.PasswordFunc == nil {
		s.failBackup(ctx, backup, "decrypt or password function not configured", logger)
		return
	}

	configJSON, err := s.config.DecryptFunc(repo.ConfigEncrypted)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("decrypt config: %v", err), logger)
		return
	}

	backend, err := ParseBackend(repo.Type, configJSON)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("parse backend: %v", err), logger)
		return
	}

	password, err := s.config.PasswordFunc(repo.ID)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("get password: %v", err), logger)
		return
	}

	resticCfg := backend.ToResticConfig(password)
	tags := []string{
		"pihole",
		fmt.Sprintf("schedule:%s", schedule.ID.String()),
	}

	backupPaths := piholeResult.BackupFiles
	if len(backupPaths) == 0 && piholeResult.BackupPath != "" {
		backupPaths = []string{piholeResult.BackupPath}
	}
	stats, err := s.restic.Backup(ctx, resticCfg, backupPaths, nil, tags)
	if err != nil {
		s.failBackup(ctx, backup, fmt.Sprintf("restic backup failed: %v", err), logger)
		return
	}

	backup.Complete(stats.SnapshotID, stats.FilesNew, stats.FilesChanged, stats.SizeBytes)
	if err := s.store.UpdateBackup(ctx, backup); err != nil {
		logger.Error().Err(err).Msg("failed to update Pi-hole backup record")
	}

	s.sendBackupNotification(ctx, schedule, backup, true, "")
}

// executeProxmoxBackup handles Proxmox VM/container backup execution.
func (s *Scheduler) executeProxmoxBackup(ctx context.Context, schedule models.Schedule, agent *models.Agent, logger zerolog.Logger) {
	logger.Info().Msg("executing Proxmox backup")
	logger.Warn().Msg("Proxmox backup not yet implemented in scheduler")
}

// runBackupValidation runs automated validation after a successful backup.
func (s *Scheduler) runBackupValidation(ctx context.Context, backup *models.Backup, resticCfg ResticConfig, sourcePaths []string, logger zerolog.Logger) {
	if s.validator == nil {
		return
	}

	logger.Debug().Msg("running automated backup validation")
	result, err := s.validator.ValidateBackup(ctx, backup, resticCfg, sourcePaths)
	if err != nil {
		logger.Error().Err(err).Msg("backup validation failed")
		return
	}
	if result != nil {
		logger.Info().
			Str("status", string(result.Status)).
			Msg("backup validation completed")
	}
}

// replicateToOtherRepos copies a snapshot to other enabled repositories.
func (s *Scheduler) replicateToOtherRepos(
	ctx context.Context,
	schedule models.Schedule,
	sourceRepo *models.ScheduleRepository,
	snapshotID string,
	sourceResticCfg ResticConfig,
	allRepos []models.ScheduleRepository,
	logger zerolog.Logger,
) {
	for i := range allRepos {
		targetRepo := &allRepos[i]
		if targetRepo.RepositoryID == sourceRepo.RepositoryID {
			continue
		}

		targetLogger := logger.With().
			Str("target_repository_id", targetRepo.RepositoryID.String()).
			Logger()

		targetLogger.Info().Msg("replicating snapshot to target repository")

		// Get target repository configuration
		repo, err := s.store.GetRepository(ctx, targetRepo.RepositoryID)
		if err != nil {
			targetLogger.Error().Err(err).Msg("failed to get target repository")
			continue
		}

		if s.config.DecryptFunc == nil {
			targetLogger.Error().Msg("decrypt function not configured for replication")
			continue
		}

		configJSON, err := s.config.DecryptFunc(repo.ConfigEncrypted)
		if err != nil {
			targetLogger.Error().Err(err).Msg("failed to decrypt target repo config")
			continue
		}

		backend, err := ParseBackend(repo.Type, configJSON)
		if err != nil {
			targetLogger.Error().Err(err).Msg("failed to parse target backend")
			continue
		}

		if s.config.PasswordFunc == nil {
			targetLogger.Error().Msg("password function not configured for replication")
			continue
		}

		password, err := s.config.PasswordFunc(repo.ID)
		if err != nil {
			targetLogger.Error().Err(err).Msg("failed to get target repo password")
			continue
		}

		targetResticCfg := backend.ToResticConfig(password)

		// Copy snapshot to target
		if err := s.restic.Copy(ctx, sourceResticCfg, targetResticCfg, snapshotID); err != nil {
			targetLogger.Error().Err(err).Msg("failed to replicate snapshot")
			// Update replication status
			rs, rsErr := s.store.GetOrCreateReplicationStatus(ctx, schedule.ID, sourceRepo.RepositoryID, targetRepo.RepositoryID)
			if rsErr == nil {
				rs.MarkFailed(err.Error())
				_ = s.store.UpdateReplicationStatus(ctx, rs)
			}
			continue
		}

		targetLogger.Info().Msg("snapshot replicated successfully")

		// Update replication status
		rs, rsErr := s.store.GetOrCreateReplicationStatus(ctx, schedule.ID, sourceRepo.RepositoryID, targetRepo.RepositoryID)
		if rsErr == nil {
			rs.MarkSynced(snapshotID)
			_ = s.store.UpdateReplicationStatus(ctx, rs)
		}
	}
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