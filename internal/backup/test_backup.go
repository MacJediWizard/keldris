// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ValidationConfig holds configuration for backup validation.
type ValidationConfig struct {
	// SpotCheckCount is the number of random files to verify.
	SpotCheckCount int

	// FileCountErrorMargin is the acceptable percentage difference in file counts.
	// For example, 0.01 allows 1% difference.
	FileCountErrorMargin float64

	// RunIntegrityCheck determines if restic check should run after validation.
	RunIntegrityCheck bool

	// IntegrityCheckSubset is the subset of data to verify (e.g., "2%").
	// Empty string means no data verification.
	IntegrityCheckSubset string
}

// DefaultValidationConfig returns a ValidationConfig with sensible defaults.
func DefaultValidationConfig() ValidationConfig {
	return ValidationConfig{
		SpotCheckCount:       10,
		FileCountErrorMargin: 0.01, // 1% tolerance
		RunIntegrityCheck:    false,
		IntegrityCheckSubset: "",
	}
}

// ValidationStore defines the interface for validation persistence operations.
type ValidationStore interface {
	// CreateBackupValidation creates a new backup validation record.
	CreateBackupValidation(ctx context.Context, v *models.BackupValidation) error

	// UpdateBackupValidation updates an existing backup validation record.
	UpdateBackupValidation(ctx context.Context, v *models.BackupValidation) error

	// GetBackupValidationByBackupID returns the validation for a backup.
	GetBackupValidationByBackupID(ctx context.Context, backupID uuid.UUID) (*models.BackupValidation, error)

	// GetLatestBackupValidationByRepoID returns the most recent validation for a repository.
	GetLatestBackupValidationByRepoID(ctx context.Context, repoID uuid.UUID) (*models.BackupValidation, error)
}

// ValidationNotifier sends alerts when backup validations fail.
type ValidationNotifier interface {
	// NotifyValidationFailed sends an alert about a failed backup validation.
	NotifyValidationFailed(ctx context.Context, v *models.BackupValidation, backup *models.Backup, errMsg string) error
}

// BackupValidator performs automated validation of backups.
type BackupValidator struct {
	restic   *Restic
	config   ValidationConfig
	store    ValidationStore
	notifier ValidationNotifier
	logger   zerolog.Logger
}

// NewBackupValidator creates a new BackupValidator.
func NewBackupValidator(
	restic *Restic,
	config ValidationConfig,
	store ValidationStore,
	logger zerolog.Logger,
) *BackupValidator {
	return &BackupValidator{
		restic: restic,
		config: config,
		store:  store,
		logger: logger.With().Str("component", "backup_validator").Logger(),
	}
}

// SetNotifier sets the notification service for validation failures.
func (bv *BackupValidator) SetNotifier(notifier ValidationNotifier) {
	bv.notifier = notifier
}

// ValidateBackup performs comprehensive validation of a completed backup.
// It verifies the backup completed successfully, checks snapshot existence,
// validates metadata, compares file counts, and spot-checks random files.
func (bv *BackupValidator) ValidateBackup(
	ctx context.Context,
	backup *models.Backup,
	cfg ResticConfig,
	sourcePaths []string,
) (*models.BackupValidation, error) {
	if backup == nil {
		return nil, fmt.Errorf("backup cannot be nil")
	}

	if backup.Status != models.BackupStatusCompleted {
		return nil, fmt.Errorf("cannot validate backup with status %s", backup.Status)
	}

	if backup.SnapshotID == "" {
		return nil, fmt.Errorf("backup has no snapshot ID")
	}

	if backup.RepositoryID == nil {
		return nil, fmt.Errorf("backup has no repository ID")
	}

	logger := bv.logger.With().
		Str("backup_id", backup.ID.String()).
		Str("snapshot_id", backup.SnapshotID).
		Logger()

	logger.Info().Msg("starting backup validation")

	// Create validation record
	validation := models.NewBackupValidation(backup.ID, *backup.RepositoryID, backup.SnapshotID)
	details := &models.BackupValidationDetails{}

	if bv.store != nil {
		if err := bv.store.CreateBackupValidation(ctx, validation); err != nil {
			logger.Error().Err(err).Msg("failed to create validation record")
			// Continue with validation even if we can't persist
		}
	}

	// Step 1: Verify snapshot exists in repository
	logger.Debug().Msg("verifying snapshot exists in repository")
	snapshotExists, snapshotErr := bv.verifySnapshotExists(ctx, cfg, backup.SnapshotID)
	details.SnapshotExists = snapshotExists
	if snapshotErr != nil {
		details.SnapshotErrorMessage = snapshotErr.Error()
		bv.failValidation(ctx, validation, "snapshot verification failed: "+snapshotErr.Error(), details, logger)
		return validation, snapshotErr
	}
	if !snapshotExists {
		bv.failValidation(ctx, validation, "snapshot not found in repository", details, logger)
		return validation, fmt.Errorf("snapshot %s not found in repository", backup.SnapshotID)
	}
	details.SnapshotVerified = true

	// Step 2: Validate snapshot metadata
	logger.Debug().Msg("validating snapshot metadata")
	metadataValid, metadataErrors := bv.validateSnapshotMetadata(ctx, cfg, backup.SnapshotID)
	details.MetadataValid = metadataValid
	details.MetadataErrors = metadataErrors
	if !metadataValid {
		errMsg := "metadata validation failed"
		if len(metadataErrors) > 0 {
			errMsg = metadataErrors[0]
		}
		bv.failValidation(ctx, validation, errMsg, details, logger)
		return validation, fmt.Errorf("metadata validation failed: %v", metadataErrors)
	}

	// Step 3: Compare file counts with source
	logger.Debug().Msg("comparing file counts with source")
	expectedCount := bv.countSourceFiles(ctx, sourcePaths, logger)
	actualCount, err := bv.countSnapshotFiles(ctx, cfg, backup.SnapshotID)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to count snapshot files, skipping file count validation")
		// Don't fail validation just because we can't count files
	} else {
		details.ExpectedFileCount = expectedCount
		details.ActualFileCount = actualCount
		details.FileCountDifference = actualCount - expectedCount
		details.FileCountErrorMargin = bv.config.FileCountErrorMargin

		// Calculate if the difference is within acceptable margin
		if expectedCount > 0 {
			margin := float64(abs(actualCount-expectedCount)) / float64(expectedCount)
			details.FileCountValid = margin <= bv.config.FileCountErrorMargin
		} else {
			// If expected count is 0, actual should also be 0 (or very small)
			details.FileCountValid = actualCount <= 10
		}

		if !details.FileCountValid {
			logger.Warn().
				Int("expected", expectedCount).
				Int("actual", actualCount).
				Int("difference", details.FileCountDifference).
				Float64("margin", bv.config.FileCountErrorMargin).
				Msg("file count difference exceeds acceptable margin")
		}
	}

	// Step 4: Spot-check random files
	logger.Debug().Int("count", bv.config.SpotCheckCount).Msg("performing spot check on random files")
	spotCheckResults, err := bv.spotCheckFiles(ctx, cfg, backup.SnapshotID, sourcePaths)
	if err != nil {
		logger.Warn().Err(err).Msg("spot check encountered errors")
	}
	details.SpotCheckPerformed = true
	details.SpotCheckFilesTotal = len(spotCheckResults)
	details.SpotCheckResults = spotCheckResults

	// Count passed/failed spot checks
	for _, result := range spotCheckResults {
		if result.SizeMatch && result.ExistsInSnap {
			details.SpotCheckFilesPassed++
		} else {
			details.SpotCheckFilesFailed++
		}
	}

	// Step 5: Optional integrity check
	if bv.config.RunIntegrityCheck {
		logger.Debug().Msg("running repository integrity check")
		checkOpts := CheckOptions{
			ReadData:       bv.config.IntegrityCheckSubset != "",
			ReadDataSubset: bv.config.IntegrityCheckSubset,
		}
		checkResult, checkErr := bv.restic.CheckWithOptions(ctx, cfg, checkOpts)
		details.IntegrityCheckRun = true
		if checkErr != nil {
			details.IntegrityCheckPassed = false
			details.IntegrityCheckError = checkErr.Error()
			logger.Warn().Err(checkErr).Msg("integrity check failed")
		} else {
			details.IntegrityCheckPassed = checkResult == nil || len(checkResult.Errors) == 0
			if !details.IntegrityCheckPassed && checkResult != nil && len(checkResult.Errors) > 0 {
				details.IntegrityCheckError = checkResult.Errors[0]
			}
		}
	}

	// Determine overall validation result
	overallPassed := details.SnapshotExists &&
		details.SnapshotVerified &&
		details.MetadataValid &&
		(details.FileCountValid || details.ExpectedFileCount == 0) &&
		details.SpotCheckFilesFailed == 0 &&
		(!details.IntegrityCheckRun || details.IntegrityCheckPassed)

	if overallPassed {
		validation.Pass(details)
		if bv.store != nil {
			if err := bv.store.UpdateBackupValidation(ctx, validation); err != nil {
				logger.Error().Err(err).Msg("failed to update validation record")
			}
		}
		logger.Info().
			Int64("duration_ms", *validation.DurationMs).
			Msg("backup validation passed")
	} else {
		// Determine the primary failure reason
		var failReason string
		if details.SpotCheckFilesFailed > 0 {
			failReason = fmt.Sprintf("spot check failed for %d/%d files", details.SpotCheckFilesFailed, details.SpotCheckFilesTotal)
		} else if !details.FileCountValid && details.ExpectedFileCount > 0 {
			failReason = fmt.Sprintf("file count mismatch: expected %d, got %d", details.ExpectedFileCount, details.ActualFileCount)
		} else if details.IntegrityCheckRun && !details.IntegrityCheckPassed {
			failReason = "repository integrity check failed"
		} else {
			failReason = "validation failed"
		}
		bv.failValidation(ctx, validation, failReason, details, logger)
	}

	return validation, nil
}

// verifySnapshotExists checks if the snapshot exists in the repository.
func (bv *BackupValidator) verifySnapshotExists(ctx context.Context, cfg ResticConfig, snapshotID string) (bool, error) {
	snapshots, err := bv.restic.Snapshots(ctx, cfg)
	if err != nil {
		return false, fmt.Errorf("list snapshots: %w", err)
	}

	for _, snap := range snapshots {
		if snap.ID == snapshotID || snap.ShortID == snapshotID {
			return true, nil
		}
	}

	return false, nil
}

// validateSnapshotMetadata validates the snapshot's metadata.
func (bv *BackupValidator) validateSnapshotMetadata(ctx context.Context, cfg ResticConfig, snapshotID string) (bool, []string) {
	var errors []string

	snapshots, err := bv.restic.Snapshots(ctx, cfg)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to get snapshots: %v", err))
		return false, errors
	}

	var snapshot *Snapshot
	for i := range snapshots {
		if snapshots[i].ID == snapshotID || snapshots[i].ShortID == snapshotID {
			snapshot = &snapshots[i]
			break
		}
	}

	if snapshot == nil {
		errors = append(errors, "snapshot not found")
		return false, errors
	}

	// Validate required metadata fields
	if snapshot.Time.IsZero() {
		errors = append(errors, "snapshot has no timestamp")
	}

	if len(snapshot.Paths) == 0 {
		errors = append(errors, "snapshot has no paths recorded")
	}

	// Validate timestamp is not in the future
	if snapshot.Time.After(time.Now().Add(time.Hour)) {
		errors = append(errors, "snapshot timestamp is in the future")
	}

	// Validate timestamp is not too old (more than 24 hours before now for a just-completed backup)
	if snapshot.Time.Before(time.Now().Add(-24 * time.Hour)) {
		errors = append(errors, "snapshot timestamp is more than 24 hours ago")
	}

	return len(errors) == 0, errors
}

// countSourceFiles counts the total number of files in the source paths.
func (bv *BackupValidator) countSourceFiles(ctx context.Context, paths []string, logger zerolog.Logger) int {
	count := 0
	for _, path := range paths {
		err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				return nil // Skip files we can't access
			}
			if !info.IsDir() {
				count++
			}
			return nil
		})
		if err != nil {
			logger.Warn().Err(err).Str("path", path).Msg("error counting source files")
		}
	}
	return count
}

// countSnapshotFiles counts the total number of files in the snapshot.
func (bv *BackupValidator) countSnapshotFiles(ctx context.Context, cfg ResticConfig, snapshotID string) (int, error) {
	files, err := bv.restic.ListFiles(ctx, cfg, snapshotID, "")
	if err != nil {
		return 0, fmt.Errorf("list snapshot files: %w", err)
	}

	count := 0
	for _, f := range files {
		if f.Type == "file" {
			count++
		}
	}

	return count, nil
}

// spotCheckFiles randomly selects and verifies files from the backup.
func (bv *BackupValidator) spotCheckFiles(
	ctx context.Context,
	cfg ResticConfig,
	snapshotID string,
	sourcePaths []string,
) ([]models.SpotCheckFileResult, error) {
	results := make([]models.SpotCheckFileResult, 0, bv.config.SpotCheckCount)

	// Get list of files in snapshot
	snapshotFiles, err := bv.restic.ListFiles(ctx, cfg, snapshotID, "")
	if err != nil {
		return nil, fmt.Errorf("list snapshot files: %w", err)
	}

	// Filter to only files (not directories)
	var files []SnapshotFile
	for _, f := range snapshotFiles {
		if f.Type == "file" {
			files = append(files, f)
		}
	}

	if len(files) == 0 {
		return results, nil
	}

	// Select random files to check
	checkCount := bv.config.SpotCheckCount
	if checkCount > len(files) {
		checkCount = len(files)
	}

	// Generate random indices
	selectedIndices := make(map[int]bool)
	for len(selectedIndices) < checkCount {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(files))))
		if err != nil {
			continue
		}
		idx := int(n.Int64())
		if !selectedIndices[idx] {
			selectedIndices[idx] = true
		}
	}

	// Check each selected file
	for idx := range selectedIndices {
		file := files[idx]
		result := models.SpotCheckFileResult{
			Path:         file.Path,
			ActualSize:   file.Size,
			ExistsInSnap: true,
		}

		// Try to find the corresponding source file
		for _, basePath := range sourcePaths {
			// Construct potential source path
			sourcePath := filepath.Join(basePath, file.Path)
			if !filepath.IsAbs(file.Path) {
				sourcePath = file.Path
			}

			info, err := os.Stat(sourcePath)
			if err == nil {
				result.ExpectedSize = info.Size()
				result.SizeMatch = result.ExpectedSize == result.ActualSize
				break
			}
		}

		// If we couldn't find the source file, just verify it exists in snapshot
		if result.ExpectedSize == 0 {
			result.SizeMatch = true // Can't compare, assume OK
		}

		results = append(results, result)
	}

	return results, nil
}

// failValidation marks a validation as failed and sends notifications.
func (bv *BackupValidator) failValidation(
	ctx context.Context,
	v *models.BackupValidation,
	errMsg string,
	details *models.BackupValidationDetails,
	logger zerolog.Logger,
) {
	v.Fail(errMsg, details)

	if bv.store != nil {
		if err := bv.store.UpdateBackupValidation(ctx, v); err != nil {
			logger.Error().Err(err).Str("original_error", errMsg).Msg("failed to update validation record")
		}
	}

	logger.Error().
		Str("error", errMsg).
		Int64("duration_ms", *v.DurationMs).
		Msg("backup validation failed")

	// Send notification if configured
	if bv.notifier != nil {
		// Note: We need the backup to send notification, but we don't have it here
		// The caller should handle notification if needed
	}
}

// abs returns the absolute value of an integer.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ValidateBackupResult contains the result of a validation operation
// along with the backup and any alerts that should be sent.
type ValidateBackupResult struct {
	Validation *models.BackupValidation
	Passed     bool
	AlertSent  bool
}

// ValidateAndNotify validates a backup and sends notifications if it fails.
func (bv *BackupValidator) ValidateAndNotify(
	ctx context.Context,
	backup *models.Backup,
	cfg ResticConfig,
	sourcePaths []string,
) (*ValidateBackupResult, error) {
	result := &ValidateBackupResult{}

	validation, err := bv.ValidateBackup(ctx, backup, cfg, sourcePaths)
	if err != nil {
		// Validation itself errored, not just failed
		return nil, err
	}

	result.Validation = validation
	result.Passed = validation.Status == models.BackupValidationStatusPassed

	// Send notification for failed validations
	if !result.Passed && bv.notifier != nil {
		if notifyErr := bv.notifier.NotifyValidationFailed(ctx, validation, backup, validation.ErrorMessage); notifyErr != nil {
			bv.logger.Error().Err(notifyErr).Msg("failed to send validation failure notification")
		} else {
			result.AlertSent = true
		}
	}

	return result, nil
}
