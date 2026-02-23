package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// VerificationStore defines the interface for verification persistence operations.
type VerificationStore interface {
	// GetEnabledVerificationSchedules returns all enabled verification schedules.
	GetEnabledVerificationSchedules(ctx context.Context) ([]*models.VerificationSchedule, error)

	// GetVerificationSchedulesByRepoID returns verification schedules for a repository.
	GetVerificationSchedulesByRepoID(ctx context.Context, repoID uuid.UUID) ([]*models.VerificationSchedule, error)

	// GetRepository returns a repository by ID.
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)

	// CreateVerification creates a new verification record.
	CreateVerification(ctx context.Context, v *models.Verification) error

	// UpdateVerification updates an existing verification record.
	UpdateVerification(ctx context.Context, v *models.Verification) error

	// GetLatestVerificationByRepoID returns the most recent verification for a repository.
	GetLatestVerificationByRepoID(ctx context.Context, repoID uuid.UUID) (*models.Verification, error)

	// GetConsecutiveFailedVerifications returns the count of consecutive failed verifications.
	GetConsecutiveFailedVerifications(ctx context.Context, repoID uuid.UUID) (int, error)
}

// VerificationNotifier sends alerts when verifications fail.
type VerificationNotifier interface {
	// NotifyVerificationFailed sends an alert about a failed verification.
	NotifyVerificationFailed(ctx context.Context, v *models.Verification, repo *models.Repository, consecutiveFails int) error
}

// VerificationConfig holds configuration for the verification scheduler.
type VerificationConfig struct {
	// RefreshInterval is how often to reload schedules from the database.
	RefreshInterval time.Duration

	// TempDir is the directory for test restores.
	TempDir string

	// PasswordFunc retrieves the repository password.
	PasswordFunc func(repoID uuid.UUID) (string, error)

	// DecryptFunc decrypts the repository configuration.
	DecryptFunc DecryptFunc

	// Notifier sends alerts on verification failure (optional).
	Notifier VerificationNotifier

	// AlertAfterConsecutiveFails triggers alerts after this many consecutive failures.
	AlertAfterConsecutiveFails int
}

// DefaultVerificationConfig returns a VerificationConfig with sensible defaults.
func DefaultVerificationConfig() VerificationConfig {
	return VerificationConfig{
		RefreshInterval:            5 * time.Minute,
		TempDir:                    os.TempDir(),
		AlertAfterConsecutiveFails: 1,
	}
}

// VerificationScheduler manages verification schedules using cron.
type VerificationScheduler struct {
	store   VerificationStore
	restic  *Restic
	config  VerificationConfig
	cron    *cron.Cron
	logger  zerolog.Logger
	mu      sync.RWMutex
	entries map[uuid.UUID]cron.EntryID
	running bool
}

// NewVerificationScheduler creates a new verification scheduler.
func NewVerificationScheduler(
	store VerificationStore,
	restic *Restic,
	config VerificationConfig,
	logger zerolog.Logger,
) *VerificationScheduler {
	return &VerificationScheduler{
		store:   store,
		restic:  restic,
		config:  config,
		cron:    cron.New(cron.WithSeconds()),
		logger:  logger.With().Str("component", "verification_scheduler").Logger(),
		entries: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start starts the verification scheduler and loads initial schedules.
func (vs *VerificationScheduler) Start(ctx context.Context) error {
	vs.mu.Lock()
	if vs.running {
		vs.mu.Unlock()
		return errors.New("verification scheduler already running")
	}
	vs.running = true
	vs.mu.Unlock()

	vs.logger.Info().Msg("starting verification scheduler")

	// Load initial schedules
	if err := vs.Reload(ctx); err != nil {
		vs.logger.Error().Err(err).Msg("failed to load initial verification schedules")
	}

	// Start cron scheduler
	vs.cron.Start()

	// Start background refresh goroutine
	go vs.refreshLoop(ctx)

	vs.logger.Info().Msg("verification scheduler started")
	return nil
}

// Stop stops the verification scheduler gracefully.
func (vs *VerificationScheduler) Stop() context.Context {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	if !vs.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	vs.running = false
	vs.logger.Info().Msg("stopping verification scheduler")

	return vs.cron.Stop()
}

// Reload reloads all verification schedules from the database.
func (vs *VerificationScheduler) Reload(ctx context.Context) error {
	vs.logger.Debug().Msg("reloading verification schedules from database")

	schedules, err := vs.store.GetEnabledVerificationSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get enabled verification schedules: %w", err)
	}

	vs.mu.Lock()
	defer vs.mu.Unlock()

	// Track which schedules we've seen
	seen := make(map[uuid.UUID]bool)

	for _, schedule := range schedules {
		seen[schedule.ID] = true

		// Check if schedule already exists
		if entryID, exists := vs.entries[schedule.ID]; exists {
			entry := vs.cron.Entry(entryID)
			if entry.Valid() {
				continue
			}
			vs.cron.Remove(entryID)
			delete(vs.entries, schedule.ID)
		}

		// Add new schedule
		if err := vs.addSchedule(schedule); err != nil {
			vs.logger.Error().
				Err(err).
				Str("schedule_id", schedule.ID.String()).
				Msg("failed to add verification schedule")
			continue
		}
	}

	// Remove schedules that are no longer enabled
	for id, entryID := range vs.entries {
		if !seen[id] {
			vs.cron.Remove(entryID)
			delete(vs.entries, id)
			vs.logger.Debug().
				Str("schedule_id", id.String()).
				Msg("removed disabled verification schedule")
		}
	}

	vs.logger.Info().
		Int("active_schedules", len(vs.entries)).
		Msg("verification schedules reloaded")

	return nil
}

// addSchedule adds a verification schedule to the cron scheduler.
func (vs *VerificationScheduler) addSchedule(schedule *models.VerificationSchedule) error {
	sched := schedule // Create a copy for the closure

	entryID, err := vs.cron.AddFunc(schedule.CronExpression, func() {
		vs.executeVerification(sched)
	})
	if err != nil {
		return fmt.Errorf("add cron entry: %w", err)
	}

	vs.entries[schedule.ID] = entryID
	vs.logger.Debug().
		Str("schedule_id", schedule.ID.String()).
		Str("repository_id", schedule.RepositoryID.String()).
		Str("type", string(schedule.Type)).
		Str("cron_expression", schedule.CronExpression).
		Msg("added verification schedule")

	return nil
}

// executeVerification runs a verification for the given schedule.
func (vs *VerificationScheduler) executeVerification(schedule *models.VerificationSchedule) {
	ctx := context.Background()
	logger := vs.logger.With().
		Str("schedule_id", schedule.ID.String()).
		Str("repository_id", schedule.RepositoryID.String()).
		Str("type", string(schedule.Type)).
		Logger()

	logger.Info().Msg("starting scheduled verification")

	// Create verification record
	verification := models.NewVerification(schedule.RepositoryID, schedule.Type)
	if err := vs.store.CreateVerification(ctx, verification); err != nil {
		logger.Error().Err(err).Msg("failed to create verification record")
		return
	}

	// Get repository configuration
	repo, err := vs.store.GetRepository(ctx, schedule.RepositoryID)
	if err != nil {
		vs.failVerification(ctx, verification, fmt.Sprintf("get repository: %v", err), nil, logger)
		return
	}

	// Decrypt repository configuration
	if vs.config.DecryptFunc == nil {
		vs.failVerification(ctx, verification, "decrypt function not configured", nil, logger)
		return
	}

	configJSON, err := vs.config.DecryptFunc(repo.ConfigEncrypted)
	if err != nil {
		vs.failVerification(ctx, verification, fmt.Sprintf("decrypt config: %v", err), nil, logger)
		return
	}

	// Parse backend configuration
	backend, err := ParseBackend(repo.Type, configJSON)
	if err != nil {
		vs.failVerification(ctx, verification, fmt.Sprintf("parse backend: %v", err), nil, logger)
		return
	}

	// Get repository password
	if vs.config.PasswordFunc == nil {
		vs.failVerification(ctx, verification, "password function not configured", nil, logger)
		return
	}

	password, err := vs.config.PasswordFunc(repo.ID)
	if err != nil {
		vs.failVerification(ctx, verification, fmt.Sprintf("get password: %v", err), nil, logger)
		return
	}

	// Build restic config
	resticCfg := backend.ToResticConfig(password)

	// Execute verification based on type
	var details *models.VerificationDetails
	switch schedule.Type {
	case models.VerificationTypeCheck:
		details, err = vs.runCheck(ctx, resticCfg, false, "")
	case models.VerificationTypeCheckReadData:
		details, err = vs.runCheck(ctx, resticCfg, true, schedule.ReadDataSubset)
	case models.VerificationTypeTestRestore:
		details, err = vs.runTestRestore(ctx, resticCfg)
	default:
		err = fmt.Errorf("unknown verification type: %s", schedule.Type)
	}

	if err != nil {
		vs.failVerification(ctx, verification, err.Error(), details, logger)
		vs.checkAndNotify(ctx, verification, repo, logger)
		return
	}

	// Mark verification as passed
	verification.Pass(details)
	if err := vs.store.UpdateVerification(ctx, verification); err != nil {
		logger.Error().Err(err).Msg("failed to update verification record")
		return
	}

	logger.Info().
		Dur("duration", verification.Duration()).
		Msg("verification completed successfully")
}

// runCheck executes a restic check operation.
func (vs *VerificationScheduler) runCheck(ctx context.Context, cfg ResticConfig, readData bool, subset string) (*models.VerificationDetails, error) {
	opts := CheckOptions{
		ReadData:       readData,
		ReadDataSubset: subset,
	}

	result, err := vs.restic.CheckWithOptions(ctx, cfg, opts)
	details := &models.VerificationDetails{
		ReadDataSubset: subset,
	}

	if result != nil && len(result.Errors) > 0 {
		details.ErrorsFound = result.Errors
	}

	return details, err
}

// runTestRestore performs a test restore to verify backup data can be restored.
func (vs *VerificationScheduler) runTestRestore(ctx context.Context, cfg ResticConfig) (*models.VerificationDetails, error) {
	details := &models.VerificationDetails{}

	// Get the latest snapshot
	snapshots, err := vs.restic.Snapshots(ctx, cfg)
	if err != nil {
		return details, fmt.Errorf("list snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return details, errors.New("no snapshots available for test restore")
	}

	// Use the most recent snapshot
	latestSnapshot := snapshots[0]
	for _, s := range snapshots {
		if s.Time.After(latestSnapshot.Time) {
			latestSnapshot = s
		}
	}

	// Create a temporary directory for the restore
	tempDir, err := os.MkdirTemp(vs.config.TempDir, "keldris-test-restore-*")
	if err != nil {
		return details, fmt.Errorf("create temp directory: %w", err)
	}
	defer func() {
		// Clean up the temp directory after verification
		if err := os.RemoveAll(tempDir); err != nil {
			vs.logger.Warn().Err(err).Str("path", tempDir).Msg("failed to clean up temp restore directory")
		}
	}()

	// Restore to temp directory
	if err := vs.restic.Restore(ctx, cfg, latestSnapshot.ID, RestoreOptions{TargetPath: tempDir}); err != nil {
		return details, fmt.Errorf("restore failed: %w", err)
	}

	// Count restored files and size
	var filesRestored int
	var bytesRestored int64
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			filesRestored++
			bytesRestored += info.Size()
		}
		return nil
	})
	if err != nil {
		return details, fmt.Errorf("count restored files: %w", err)
	}

	details.FilesRestored = filesRestored
	details.BytesRestored = bytesRestored

	vs.logger.Debug().
		Int("files_restored", filesRestored).
		Int64("bytes_restored", bytesRestored).
		Str("snapshot_id", latestSnapshot.ID).
		Msg("test restore completed")

	return details, nil
}

// failVerification marks a verification as failed and logs the error.
func (vs *VerificationScheduler) failVerification(
	ctx context.Context,
	v *models.Verification,
	errMsg string,
	details *models.VerificationDetails,
	logger zerolog.Logger,
) {
	v.Fail(errMsg, details)
	if err := vs.store.UpdateVerification(ctx, v); err != nil {
		logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update verification record")
		return
	}
	logger.Error().Str("error", errMsg).Msg("verification failed")
}

// checkAndNotify sends an alert if consecutive failures exceed the threshold.
func (vs *VerificationScheduler) checkAndNotify(
	ctx context.Context,
	v *models.Verification,
	repo *models.Repository,
	logger zerolog.Logger,
) {
	if vs.config.Notifier == nil {
		return
	}

	consecutiveFails, err := vs.store.GetConsecutiveFailedVerifications(ctx, v.RepositoryID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get consecutive failure count")
		return
	}

	if consecutiveFails >= vs.config.AlertAfterConsecutiveFails {
		if err := vs.config.Notifier.NotifyVerificationFailed(ctx, v, repo, consecutiveFails); err != nil {
			logger.Error().Err(err).Msg("failed to send verification failure notification")
		} else {
			logger.Info().Int("consecutive_fails", consecutiveFails).Msg("verification failure notification sent")
		}
	}
}

// refreshLoop periodically reloads schedules from the database.
func (vs *VerificationScheduler) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(vs.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			vs.mu.RLock()
			running := vs.running
			vs.mu.RUnlock()

			if !running {
				return
			}

			if err := vs.Reload(ctx); err != nil {
				vs.logger.Error().Err(err).Msg("failed to reload verification schedules")
			}
		}
	}
}

// TriggerVerification manually triggers a verification for the given repository.
func (vs *VerificationScheduler) TriggerVerification(ctx context.Context, repoID uuid.UUID, verType models.VerificationType) (*models.Verification, error) {
	schedule := &models.VerificationSchedule{
		ID:           uuid.New(),
		RepositoryID: repoID,
		Type:         verType,
	}

	// Create verification record
	verification := models.NewVerification(repoID, verType)
	if err := vs.store.CreateVerification(ctx, verification); err != nil {
		return nil, fmt.Errorf("create verification record: %w", err)
	}

	// Execute in background
	go vs.executeVerification(schedule)

	return verification, nil
}

// GetRepositoryVerificationStatus returns the verification status for a repository.
func (vs *VerificationScheduler) GetRepositoryVerificationStatus(ctx context.Context, repoID uuid.UUID) (*models.RepositoryVerificationStatus, error) {
	status := &models.RepositoryVerificationStatus{
		RepositoryID: repoID,
	}

	// Get latest verification
	latest, err := vs.store.GetLatestVerificationByRepoID(ctx, repoID)
	if err == nil {
		status.LastVerification = latest
	}

	// Get consecutive failures
	fails, err := vs.store.GetConsecutiveFailedVerifications(ctx, repoID)
	if err == nil {
		status.ConsecutiveFails = fails
	}

	// Get next scheduled time
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	schedules, err := vs.store.GetVerificationSchedulesByRepoID(ctx, repoID)
	if err == nil && len(schedules) > 0 {
		var nextTime *time.Time
		for _, sched := range schedules {
			if entryID, exists := vs.entries[sched.ID]; exists {
				entry := vs.cron.Entry(entryID)
				if entry.Valid() {
					if nextTime == nil || entry.Next.Before(*nextTime) {
						t := entry.Next
						nextTime = &t
					}
				}
			}
		}
		status.NextScheduledAt = nextTime
	}

	return status, nil
}

// GetNextRun returns the next scheduled run time for a verification schedule.
func (vs *VerificationScheduler) GetNextRun(scheduleID uuid.UUID) (time.Time, bool) {
	vs.mu.RLock()
	defer vs.mu.RUnlock()

	entryID, exists := vs.entries[scheduleID]
	if !exists {
		return time.Time{}, false
	}

	entry := vs.cron.Entry(entryID)
	if !entry.Valid() {
		return time.Time{}, false
	}

	return entry.Next, true
}
