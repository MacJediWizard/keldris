// Package backup provides automated test restore functionality for verifying backup integrity.
package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// TestRestoreStore defines the interface for test restore persistence operations.
type TestRestoreStore interface {
	// GetEnabledTestRestoreSettings returns all enabled test restore settings.
	GetEnabledTestRestoreSettings(ctx context.Context) ([]*models.TestRestoreSettings, error)

	// GetTestRestoreSettingsByRepoID returns test restore settings for a repository.
	GetTestRestoreSettingsByRepoID(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreSettings, error)

	// CreateTestRestoreSettings creates new test restore settings.
	CreateTestRestoreSettings(ctx context.Context, settings *models.TestRestoreSettings) error

	// UpdateTestRestoreSettings updates existing test restore settings.
	UpdateTestRestoreSettings(ctx context.Context, settings *models.TestRestoreSettings) error

	// DeleteTestRestoreSettings deletes test restore settings.
	DeleteTestRestoreSettings(ctx context.Context, id uuid.UUID) error

	// GetRepository returns a repository by ID.
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)

	// CreateTestRestoreResult creates a new test restore result record.
	CreateTestRestoreResult(ctx context.Context, result *models.TestRestoreResult) error

	// UpdateTestRestoreResult updates an existing test restore result record.
	UpdateTestRestoreResult(ctx context.Context, result *models.TestRestoreResult) error

	// GetTestRestoreResultsByRepoID returns test restore results for a repository.
	GetTestRestoreResultsByRepoID(ctx context.Context, repoID uuid.UUID, limit int) ([]*models.TestRestoreResult, error)

	// GetLatestTestRestoreResultByRepoID returns the most recent test restore result for a repository.
	GetLatestTestRestoreResultByRepoID(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreResult, error)

	// GetConsecutiveFailedTestRestores returns the count of consecutive failed test restores.
	GetConsecutiveFailedTestRestores(ctx context.Context, repoID uuid.UUID) (int, error)
}

// TestRestoreNotifier sends alerts when test restores fail.
type TestRestoreNotifier interface {
	// NotifyTestRestoreFailed sends an alert about a failed test restore.
	NotifyTestRestoreFailed(ctx context.Context, result *models.TestRestoreResult, repo *models.Repository, consecutiveFails int) error
}

// TestRestoreConfig holds configuration for the test restore scheduler.
type TestRestoreConfig struct {
	// RefreshInterval is how often to reload settings from the database.
	RefreshInterval time.Duration

	// TempDir is the directory for test restores.
	TempDir string

	// PasswordFunc retrieves the repository password.
	PasswordFunc func(repoID uuid.UUID) (string, error)

	// DecryptFunc decrypts the repository configuration.
	DecryptFunc DecryptFunc

	// Notifier sends alerts on test restore failure (optional).
	Notifier TestRestoreNotifier

	// AlertAfterConsecutiveFails triggers alerts after this many consecutive failures.
	AlertAfterConsecutiveFails int
}

// DefaultTestRestoreConfig returns a TestRestoreConfig with sensible defaults.
func DefaultTestRestoreConfig() TestRestoreConfig {
	return TestRestoreConfig{
		RefreshInterval:            5 * time.Minute,
		TempDir:                    os.TempDir(),
		AlertAfterConsecutiveFails: 1,
	}
}

// TestRestoreScheduler manages automated test restore schedules using cron.
type TestRestoreScheduler struct {
	store   TestRestoreStore
	restic  *Restic
	config  TestRestoreConfig
	cron    *cron.Cron
	logger  zerolog.Logger
	mu      sync.RWMutex
	entries map[uuid.UUID]cron.EntryID
	running bool
}

// NewTestRestoreScheduler creates a new test restore scheduler.
func NewTestRestoreScheduler(
	store TestRestoreStore,
	restic *Restic,
	config TestRestoreConfig,
	logger zerolog.Logger,
) *TestRestoreScheduler {
	return &TestRestoreScheduler{
		store:   store,
		restic:  restic,
		config:  config,
		cron:    cron.New(cron.WithSeconds()),
		logger:  logger.With().Str("component", "test_restore_scheduler").Logger(),
		entries: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start starts the test restore scheduler and loads initial settings.
func (trs *TestRestoreScheduler) Start(ctx context.Context) error {
	trs.mu.Lock()
	if trs.running {
		trs.mu.Unlock()
		return errors.New("test restore scheduler already running")
	}
	trs.running = true
	trs.mu.Unlock()

	trs.logger.Info().Msg("starting test restore scheduler")

	// Load initial settings
	if err := trs.Reload(ctx); err != nil {
		trs.logger.Error().Err(err).Msg("failed to load initial test restore settings")
	}

	// Start cron scheduler
	trs.cron.Start()

	// Start background refresh goroutine
	go trs.refreshLoop(ctx)

	trs.logger.Info().Msg("test restore scheduler started")
	return nil
}

// Stop stops the test restore scheduler gracefully.
func (trs *TestRestoreScheduler) Stop() context.Context {
	trs.mu.Lock()
	defer trs.mu.Unlock()

	if !trs.running {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx
	}

	trs.running = false
	trs.logger.Info().Msg("stopping test restore scheduler")

	return trs.cron.Stop()
}

// Reload reloads all test restore settings from the database.
func (trs *TestRestoreScheduler) Reload(ctx context.Context) error {
	trs.logger.Debug().Msg("reloading test restore settings from database")

	settings, err := trs.store.GetEnabledTestRestoreSettings(ctx)
	if err != nil {
		return fmt.Errorf("get enabled test restore settings: %w", err)
	}

	trs.mu.Lock()
	defer trs.mu.Unlock()

	// Track which settings we've seen
	seen := make(map[uuid.UUID]bool)

	for _, setting := range settings {
		seen[setting.ID] = true

		// Check if setting already exists
		if entryID, exists := trs.entries[setting.ID]; exists {
			entry := trs.cron.Entry(entryID)
			if entry.Valid() {
				continue
			}
			trs.cron.Remove(entryID)
			delete(trs.entries, setting.ID)
		}

		// Add new setting
		if err := trs.addSetting(setting); err != nil {
			trs.logger.Error().
				Err(err).
				Str("setting_id", setting.ID.String()).
				Msg("failed to add test restore setting")
			continue
		}
	}

	// Remove settings that are no longer enabled
	for id, entryID := range trs.entries {
		if !seen[id] {
			trs.cron.Remove(entryID)
			delete(trs.entries, id)
			trs.logger.Debug().
				Str("setting_id", id.String()).
				Msg("removed disabled test restore setting")
		}
	}

	trs.logger.Info().
		Int("active_settings", len(trs.entries)).
		Msg("test restore settings reloaded")

	return nil
}

// addSetting adds a test restore setting to the cron scheduler.
func (trs *TestRestoreScheduler) addSetting(setting *models.TestRestoreSettings) error {
	s := setting // Create a copy for the closure

	entryID, err := trs.cron.AddFunc(setting.CronExpression, func() {
		trs.executeTestRestore(s)
	})
	if err != nil {
		return fmt.Errorf("add cron entry: %w", err)
	}

	trs.entries[setting.ID] = entryID
	trs.logger.Debug().
		Str("setting_id", setting.ID.String()).
		Str("repository_id", setting.RepositoryID.String()).
		Str("cron_expression", setting.CronExpression).
		Int("sample_percentage", setting.SamplePercentage).
		Msg("added test restore setting")

	return nil
}

// executeTestRestore runs a test restore for the given settings.
func (trs *TestRestoreScheduler) executeTestRestore(setting *models.TestRestoreSettings) {
	ctx := context.Background()
	logger := trs.logger.With().
		Str("setting_id", setting.ID.String()).
		Str("repository_id", setting.RepositoryID.String()).
		Int("sample_percentage", setting.SamplePercentage).
		Logger()

	logger.Info().Msg("starting scheduled test restore")

	// Create test restore result record
	result := models.NewTestRestoreResult(setting.RepositoryID)
	result.SamplePercentage = setting.SamplePercentage
	if err := trs.store.CreateTestRestoreResult(ctx, result); err != nil {
		logger.Error().Err(err).Msg("failed to create test restore result record")
		return
	}

	// Get repository configuration
	repo, err := trs.store.GetRepository(ctx, setting.RepositoryID)
	if err != nil {
		trs.failTestRestore(ctx, result, fmt.Sprintf("get repository: %v", err), nil, logger)
		return
	}

	// Decrypt repository configuration
	if trs.config.DecryptFunc == nil {
		trs.failTestRestore(ctx, result, "decrypt function not configured", nil, logger)
		return
	}

	configJSON, err := trs.config.DecryptFunc(repo.ConfigEncrypted)
	if err != nil {
		trs.failTestRestore(ctx, result, fmt.Sprintf("decrypt config: %v", err), nil, logger)
		return
	}

	// Parse backend configuration
	backend, err := ParseBackend(repo.Type, configJSON)
	if err != nil {
		trs.failTestRestore(ctx, result, fmt.Sprintf("parse backend: %v", err), nil, logger)
		return
	}

	// Get repository password
	if trs.config.PasswordFunc == nil {
		trs.failTestRestore(ctx, result, "password function not configured", nil, logger)
		return
	}

	password, err := trs.config.PasswordFunc(repo.ID)
	if err != nil {
		trs.failTestRestore(ctx, result, fmt.Sprintf("get password: %v", err), nil, logger)
		return
	}

	// Build restic config
	resticCfg := backend.ToResticConfig(password)

	// Execute test restore
	details, err := trs.runTestRestore(ctx, resticCfg, setting.SamplePercentage)
	if err != nil {
		trs.failTestRestore(ctx, result, err.Error(), details, logger)
		trs.checkAndNotify(ctx, result, repo, logger)
		return
	}

	// Mark test restore as passed
	result.Pass(details)
	if err := trs.store.UpdateTestRestoreResult(ctx, result); err != nil {
		logger.Error().Err(err).Msg("failed to update test restore result record")
		return
	}

	// Update last run time on settings
	setting.RecordLastRun(true)
	if err := trs.store.UpdateTestRestoreSettings(ctx, setting); err != nil {
		logger.Warn().Err(err).Msg("failed to update test restore settings last run time")
	}

	logger.Info().
		Dur("duration", result.Duration()).
		Int("files_restored", details.FilesRestored).
		Int("files_verified", details.FilesVerified).
		Int64("bytes_restored", details.BytesRestored).
		Msg("test restore completed successfully")
}

// runTestRestore performs a test restore with checksum verification.
func (trs *TestRestoreScheduler) runTestRestore(ctx context.Context, cfg ResticConfig, samplePercentage int) (*models.TestRestoreDetails, error) {
	details := &models.TestRestoreDetails{}

	// Get the latest snapshot
	snapshots, err := trs.restic.Snapshots(ctx, cfg)
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
	details.SnapshotID = latestSnapshot.ID

	// Get file list from snapshot
	files, err := trs.restic.ListFiles(ctx, cfg, latestSnapshot.ID, "")
	if err != nil {
		return details, fmt.Errorf("list files in snapshot: %w", err)
	}

	// Filter to only regular files
	var regularFiles []SnapshotFile
	for _, f := range files {
		if f.Type == "file" {
			regularFiles = append(regularFiles, f)
		}
	}

	if len(regularFiles) == 0 {
		return details, errors.New("no files found in snapshot")
	}

	// Calculate sample size
	sampleSize := (len(regularFiles) * samplePercentage) / 100
	if sampleSize == 0 {
		sampleSize = 1 // At least one file
	}
	if sampleSize > len(regularFiles) {
		sampleSize = len(regularFiles)
	}

	// Randomly select files to restore
	selectedFiles := selectRandomFiles(regularFiles, sampleSize)
	details.TotalFilesInSnapshot = len(regularFiles)

	// Create a temporary directory for the restore
	tempDir, err := os.MkdirTemp(trs.config.TempDir, "keldris-test-restore-*")
	if err != nil {
		return details, fmt.Errorf("create temp directory: %w", err)
	}
	defer func() {
		// Clean up the temp directory after test
		if err := os.RemoveAll(tempDir); err != nil {
			trs.logger.Warn().Err(err).Str("path", tempDir).Msg("failed to clean up temp restore directory")
		}
	}()

	details.TempDirectory = tempDir

	// Build include patterns for selected files
	var includePatterns []string
	for _, f := range selectedFiles {
		includePatterns = append(includePatterns, f.Path)
	}

	// Restore selected files to temp directory
	restoreOpts := RestoreOptions{
		TargetPath: tempDir,
		Include:    includePatterns,
	}

	if err := trs.restic.Restore(ctx, cfg, latestSnapshot.ID, restoreOpts); err != nil {
		return details, fmt.Errorf("restore failed: %w", err)
	}

	// Verify restored files with checksums
	verificationErrors := make([]string, 0)
	var filesRestored, filesVerified int
	var bytesRestored int64

	for _, expectedFile := range selectedFiles {
		restoredPath := filepath.Join(tempDir, expectedFile.Path)

		// Check if file exists
		info, err := os.Stat(restoredPath)
		if err != nil {
			if os.IsNotExist(err) {
				verificationErrors = append(verificationErrors, fmt.Sprintf("file not restored: %s", expectedFile.Path))
				continue
			}
			verificationErrors = append(verificationErrors, fmt.Sprintf("stat error for %s: %v", expectedFile.Path, err))
			continue
		}

		filesRestored++
		bytesRestored += info.Size()

		// Verify file size matches
		if info.Size() != expectedFile.Size {
			verificationErrors = append(verificationErrors, fmt.Sprintf("size mismatch for %s: expected %d, got %d", expectedFile.Path, expectedFile.Size, info.Size()))
			continue
		}

		// Compute checksum of restored file
		checksum, err := computeFileChecksum(restoredPath)
		if err != nil {
			verificationErrors = append(verificationErrors, fmt.Sprintf("checksum error for %s: %v", expectedFile.Path, err))
			continue
		}

		// Store checksum for reference
		details.VerifiedChecksums = append(details.VerifiedChecksums, models.FileChecksum{
			Path:     expectedFile.Path,
			Checksum: checksum,
			Size:     info.Size(),
		})
		filesVerified++
	}

	details.FilesRestored = filesRestored
	details.FilesVerified = filesVerified
	details.BytesRestored = bytesRestored
	details.VerificationErrors = verificationErrors

	// Check if verification passed
	if len(verificationErrors) > 0 {
		return details, fmt.Errorf("verification failed with %d errors", len(verificationErrors))
	}

	trs.logger.Debug().
		Int("files_restored", filesRestored).
		Int("files_verified", filesVerified).
		Int64("bytes_restored", bytesRestored).
		Str("snapshot_id", latestSnapshot.ID).
		Msg("test restore completed")

	return details, nil
}

// selectRandomFiles randomly selects n files from the given slice.
func selectRandomFiles(files []SnapshotFile, n int) []SnapshotFile {
	if n >= len(files) {
		return files
	}

	// Create a copy to avoid modifying the original
	shuffled := make([]SnapshotFile, len(files))
	copy(shuffled, files)

	// Fisher-Yates shuffle
	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:n]
}

// computeFileChecksum computes the SHA256 checksum of a file.
func computeFileChecksum(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// failTestRestore marks a test restore as failed and logs the error.
func (trs *TestRestoreScheduler) failTestRestore(
	ctx context.Context,
	result *models.TestRestoreResult,
	errMsg string,
	details *models.TestRestoreDetails,
	logger zerolog.Logger,
) {
	result.Fail(errMsg, details)
	if err := trs.store.UpdateTestRestoreResult(ctx, result); err != nil {
		logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update test restore result record")
		return
	}
	logger.Error().Str("error", errMsg).Msg("test restore failed")
}

// checkAndNotify sends an alert if consecutive failures exceed the threshold.
func (trs *TestRestoreScheduler) checkAndNotify(
	ctx context.Context,
	result *models.TestRestoreResult,
	repo *models.Repository,
	logger zerolog.Logger,
) {
	if trs.config.Notifier == nil {
		return
	}

	consecutiveFails, err := trs.store.GetConsecutiveFailedTestRestores(ctx, result.RepositoryID)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get consecutive failure count")
		return
	}

	if consecutiveFails >= trs.config.AlertAfterConsecutiveFails {
		if err := trs.config.Notifier.NotifyTestRestoreFailed(ctx, result, repo, consecutiveFails); err != nil {
			logger.Error().Err(err).Msg("failed to send test restore failure notification")
		} else {
			logger.Info().Int("consecutive_fails", consecutiveFails).Msg("test restore failure notification sent")
		}
	}
}

// refreshLoop periodically reloads settings from the database.
func (trs *TestRestoreScheduler) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(trs.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			trs.mu.RLock()
			running := trs.running
			trs.mu.RUnlock()

			if !running {
				return
			}

			if err := trs.Reload(ctx); err != nil {
				trs.logger.Error().Err(err).Msg("failed to reload test restore settings")
			}
		}
	}
}

// TriggerTestRestore manually triggers a test restore for the given repository.
func (trs *TestRestoreScheduler) TriggerTestRestore(ctx context.Context, repoID uuid.UUID, samplePercentage int) (*models.TestRestoreResult, error) {
	if samplePercentage <= 0 || samplePercentage > 100 {
		samplePercentage = 10 // Default to 10%
	}

	setting := &models.TestRestoreSettings{
		ID:               uuid.New(),
		RepositoryID:     repoID,
		SamplePercentage: samplePercentage,
	}

	// Create result record
	result := models.NewTestRestoreResult(repoID)
	result.SamplePercentage = samplePercentage
	if err := trs.store.CreateTestRestoreResult(ctx, result); err != nil {
		return nil, fmt.Errorf("create test restore result: %w", err)
	}

	// Execute in background
	go trs.executeTestRestore(setting)

	return result, nil
}

// GetRepositoryTestRestoreStatus returns the test restore status for a repository.
func (trs *TestRestoreScheduler) GetRepositoryTestRestoreStatus(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreStatus, error) {
	status := &models.TestRestoreStatus{
		RepositoryID: repoID,
	}

	// Get latest test restore result
	latest, err := trs.store.GetLatestTestRestoreResultByRepoID(ctx, repoID)
	if err == nil {
		status.LastResult = latest
	}

	// Get consecutive failures
	fails, err := trs.store.GetConsecutiveFailedTestRestores(ctx, repoID)
	if err == nil {
		status.ConsecutiveFails = fails
	}

	// Get settings
	settings, err := trs.store.GetTestRestoreSettingsByRepoID(ctx, repoID)
	if err == nil && settings != nil {
		status.Settings = settings

		// Get next scheduled time
		trs.mu.RLock()
		if entryID, exists := trs.entries[settings.ID]; exists {
			entry := trs.cron.Entry(entryID)
			if entry.Valid() {
				t := entry.Next
				status.NextScheduledAt = &t
			}
		}
		trs.mu.RUnlock()
	}

	return status, nil
}

// GetNextRun returns the next scheduled run time for a test restore setting.
func (trs *TestRestoreScheduler) GetNextRun(settingID uuid.UUID) (time.Time, bool) {
	trs.mu.RLock()
	defer trs.mu.RUnlock()

	entryID, exists := trs.entries[settingID]
	if !exists {
		return time.Time{}, false
	}

	entry := trs.cron.Entry(entryID)
	if !entry.Valid() {
		return time.Time{}, false
	}

	return entry.Next, true
}
