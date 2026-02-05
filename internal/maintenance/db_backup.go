package maintenance

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// DatabaseBackupStore defines the interface for database backup persistence.
type DatabaseBackupStore interface {
	CreateDatabaseBackup(ctx context.Context, backup *models.DatabaseBackup) error
	UpdateDatabaseBackup(ctx context.Context, backup *models.DatabaseBackup) error
	GetDatabaseBackupByID(ctx context.Context, id uuid.UUID) (*models.DatabaseBackup, error)
	ListDatabaseBackups(ctx context.Context, limit, offset int) ([]*models.DatabaseBackup, int, error)
	GetLatestDatabaseBackup(ctx context.Context) (*models.DatabaseBackup, error)
	DeleteDatabaseBackup(ctx context.Context, id uuid.UUID) error
	GetDatabaseBackupsOlderThan(ctx context.Context, before time.Time) ([]*models.DatabaseBackup, error)
}

// DatabaseBackupConfig holds configuration for database backups.
type DatabaseBackupConfig struct {
	// DatabaseURL is the PostgreSQL connection string.
	DatabaseURL string

	// BackupDir is the directory where backups are stored.
	BackupDir string

	// RetentionDays is how many days to keep backups.
	RetentionDays int

	// MaxBackups is the maximum number of backups to keep (0 = unlimited).
	MaxBackups int

	// CronExpression for scheduled backups (default: daily at 2 AM).
	CronExpression string

	// IncludeBlobs includes large objects in backup.
	IncludeBlobs bool

	// CompressLevel is the gzip compression level (1-9).
	CompressLevel int
}

// DefaultDatabaseBackupConfig returns a config with sensible defaults.
func DefaultDatabaseBackupConfig() DatabaseBackupConfig {
	return DatabaseBackupConfig{
		BackupDir:      "/var/lib/keldris/backups",
		RetentionDays:  30,
		MaxBackups:     0,
		CronExpression: "0 0 2 * * *", // Daily at 2 AM
		IncludeBlobs:   true,
		CompressLevel:  6,
	}
}

// DatabaseBackupService handles PostgreSQL database backups.
type DatabaseBackupService struct {
	store      DatabaseBackupStore
	keyManager *crypto.KeyManager
	config     DatabaseBackupConfig
	cron       *cron.Cron
	logger     zerolog.Logger
	mu         sync.RWMutex
	running    bool
	entryID    cron.EntryID
}

// NewDatabaseBackupService creates a new database backup service.
func NewDatabaseBackupService(
	store DatabaseBackupStore,
	keyManager *crypto.KeyManager,
	config DatabaseBackupConfig,
	logger zerolog.Logger,
) *DatabaseBackupService {
	return &DatabaseBackupService{
		store:      store,
		keyManager: keyManager,
		config:     config,
		cron:       cron.New(cron.WithSeconds()),
		logger:     logger.With().Str("component", "db_backup").Logger(),
	}
}

// Start starts the backup scheduler.
func (s *DatabaseBackupService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("database backup service already running")
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(s.config.BackupDir, 0750); err != nil {
		return fmt.Errorf("create backup directory: %w", err)
	}

	// Schedule automatic backups
	entryID, err := s.cron.AddFunc(s.config.CronExpression, func() {
		if _, err := s.CreateBackup(context.Background()); err != nil {
			s.logger.Error().Err(err).Msg("scheduled backup failed")
		}
	})
	if err != nil {
		return fmt.Errorf("schedule backup: %w", err)
	}

	s.entryID = entryID
	s.cron.Start()
	s.running = true

	s.logger.Info().
		Str("schedule", s.config.CronExpression).
		Str("backup_dir", s.config.BackupDir).
		Int("retention_days", s.config.RetentionDays).
		Msg("database backup service started")

	return nil
}

// Stop stops the backup scheduler.
func (s *DatabaseBackupService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cron.Stop()
	s.running = false
	s.logger.Info().Msg("database backup service stopped")
}

// CreateBackup creates a new database backup.
func (s *DatabaseBackupService) CreateBackup(ctx context.Context) (*models.DatabaseBackup, error) {
	s.logger.Info().Msg("starting database backup")

	// Create backup record
	backup := models.NewDatabaseBackup()
	if err := s.store.CreateDatabaseBackup(ctx, backup); err != nil {
		return nil, fmt.Errorf("create backup record: %w", err)
	}

	// Mark as running
	backup.Start()
	if err := s.store.UpdateDatabaseBackup(ctx, backup); err != nil {
		s.logger.Error().Err(err).Msg("failed to update backup status")
	}

	// Execute the backup
	result, err := s.executeBackup(ctx, backup)
	if err != nil {
		backup.Fail(err.Error())
		if updateErr := s.store.UpdateDatabaseBackup(ctx, backup); updateErr != nil {
			s.logger.Error().Err(updateErr).Msg("failed to update failed backup status")
		}
		return backup, err
	}

	// Update backup with results
	backup.Complete(result.FilePath, result.SizeBytes, result.Checksum)
	if err := s.store.UpdateDatabaseBackup(ctx, backup); err != nil {
		s.logger.Error().Err(err).Msg("failed to update completed backup status")
	}

	s.logger.Info().
		Str("backup_id", backup.ID.String()).
		Str("file_path", result.FilePath).
		Int64("size_bytes", result.SizeBytes).
		Dur("duration", time.Since(backup.StartedAt)).
		Msg("database backup completed successfully")

	// Run retention cleanup
	go s.cleanupOldBackups(context.Background())

	return backup, nil
}

// backupResult holds the result of a backup operation.
type backupResult struct {
	FilePath  string
	SizeBytes int64
	Checksum  string
}

// executeBackup performs the actual backup operation.
func (s *DatabaseBackupService) executeBackup(ctx context.Context, backup *models.DatabaseBackup) (*backupResult, error) {
	// Generate unique filename
	timestamp := time.Now().Format("20060102-150405")
	baseFilename := fmt.Sprintf("keldris-backup-%s.sql.gz.enc", timestamp)
	filePath := filepath.Join(s.config.BackupDir, baseFilename)

	// Run pg_dump
	dumpData, err := s.runPgDump(ctx)
	if err != nil {
		return nil, fmt.Errorf("pg_dump failed: %w", err)
	}

	s.logger.Debug().
		Int("dump_size", len(dumpData)).
		Msg("pg_dump completed")

	// Compress the dump
	compressedData, err := s.compressData(dumpData)
	if err != nil {
		return nil, fmt.Errorf("compression failed: %w", err)
	}

	s.logger.Debug().
		Int("original_size", len(dumpData)).
		Int("compressed_size", len(compressedData)).
		Float64("ratio", float64(len(compressedData))/float64(len(dumpData))*100).
		Msg("compression completed")

	// Encrypt the compressed data
	encryptedData, err := s.keyManager.Encrypt(compressedData)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	s.logger.Debug().
		Int("encrypted_size", len(encryptedData)).
		Msg("encryption completed")

	// Calculate checksum before writing
	checksum := s.calculateChecksum(encryptedData)

	// Write to file
	if err := os.WriteFile(filePath, encryptedData, 0600); err != nil {
		return nil, fmt.Errorf("write backup file: %w", err)
	}

	return &backupResult{
		FilePath:  filePath,
		SizeBytes: int64(len(encryptedData)),
		Checksum:  checksum,
	}, nil
}

// runPgDump executes pg_dump and returns the output.
func (s *DatabaseBackupService) runPgDump(ctx context.Context) ([]byte, error) {
	args := []string{
		"--format=plain",
		"--no-owner",
		"--no-acl",
		"--clean",
		"--if-exists",
	}

	if s.config.IncludeBlobs {
		args = append(args, "--blobs")
	}

	args = append(args, s.config.DatabaseURL)

	cmd := exec.CommandContext(ctx, "pg_dump", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		return nil, fmt.Errorf("pg_dump error: %s", errMsg)
	}

	return stdout.Bytes(), nil
}

// compressData compresses data using gzip.
func (s *DatabaseBackupService) compressData(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, s.config.CompressLevel)
	if err != nil {
		return nil, fmt.Errorf("create gzip writer: %w", err)
	}

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("write to gzip: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// calculateChecksum calculates SHA256 checksum of data.
func (s *DatabaseBackupService) calculateChecksum(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256Sum(data))
}

// sha256Sum computes SHA256 hash.
func sha256Sum(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

// cleanupOldBackups removes backups that exceed retention policy.
func (s *DatabaseBackupService) cleanupOldBackups(ctx context.Context) {
	s.logger.Debug().Msg("running backup retention cleanup")

	// Get backups older than retention period
	cutoffTime := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	oldBackups, err := s.store.GetDatabaseBackupsOlderThan(ctx, cutoffTime)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to get old backups for cleanup")
		return
	}

	// Delete old backups
	for _, backup := range oldBackups {
		if err := s.deleteBackup(ctx, backup); err != nil {
			s.logger.Error().
				Err(err).
				Str("backup_id", backup.ID.String()).
				Msg("failed to delete old backup")
		} else {
			s.logger.Info().
				Str("backup_id", backup.ID.String()).
				Time("created_at", backup.CreatedAt).
				Msg("deleted old backup")
		}
	}

	// Check max backups limit
	if s.config.MaxBackups > 0 {
		allBackups, total, err := s.store.ListDatabaseBackups(ctx, 0, 0)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to list backups for max limit check")
			return
		}

		if total > s.config.MaxBackups {
			// Sort by creation time (oldest first)
			sort.Slice(allBackups, func(i, j int) bool {
				return allBackups[i].CreatedAt.Before(allBackups[j].CreatedAt)
			})

			// Delete excess backups
			toDelete := total - s.config.MaxBackups
			for i := 0; i < toDelete && i < len(allBackups); i++ {
				if err := s.deleteBackup(ctx, allBackups[i]); err != nil {
					s.logger.Error().
						Err(err).
						Str("backup_id", allBackups[i].ID.String()).
						Msg("failed to delete excess backup")
				} else {
					s.logger.Info().
						Str("backup_id", allBackups[i].ID.String()).
						Msg("deleted excess backup (max limit exceeded)")
				}
			}
		}
	}
}

// deleteBackup deletes a backup file and its database record.
func (s *DatabaseBackupService) deleteBackup(ctx context.Context, backup *models.DatabaseBackup) error {
	// Delete the file if it exists
	if backup.FilePath != "" {
		if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove backup file: %w", err)
		}
	}

	// Delete the database record
	if err := s.store.DeleteDatabaseBackup(ctx, backup.ID); err != nil {
		return fmt.Errorf("delete backup record: %w", err)
	}

	return nil
}

// GetStatus returns the current backup service status.
func (s *DatabaseBackupService) GetStatus(ctx context.Context) (*DatabaseBackupStatus, error) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	status := &DatabaseBackupStatus{
		Enabled:    running,
		BackupDir:  s.config.BackupDir,
		Schedule:   s.config.CronExpression,
		Retention:  s.config.RetentionDays,
		MaxBackups: s.config.MaxBackups,
	}

	// Get latest backup
	latest, err := s.store.GetLatestDatabaseBackup(ctx)
	if err == nil && latest != nil {
		status.LastBackup = latest
		status.LastBackupTime = &latest.CreatedAt
		status.LastBackupStatus = string(latest.Status)
	}

	// Get next scheduled run
	if running {
		entry := s.cron.Entry(s.entryID)
		if entry.Valid() {
			status.NextBackupTime = &entry.Next
		}
	}

	// Count total backups
	_, total, err := s.store.ListDatabaseBackups(ctx, 1, 0)
	if err == nil {
		status.TotalBackups = total
	}

	// Calculate total size
	backups, _, err := s.store.ListDatabaseBackups(ctx, 0, 0)
	if err == nil {
		var totalSize int64
		for _, b := range backups {
			if b.SizeBytes != nil {
				totalSize += *b.SizeBytes
			}
		}
		status.TotalSizeBytes = totalSize
	}

	return status, nil
}

// DatabaseBackupStatus holds the current status of the backup service.
type DatabaseBackupStatus struct {
	Enabled          bool                    `json:"enabled"`
	BackupDir        string                  `json:"backup_dir"`
	Schedule         string                  `json:"schedule"`
	Retention        int                     `json:"retention_days"`
	MaxBackups       int                     `json:"max_backups"`
	LastBackup       *models.DatabaseBackup  `json:"last_backup,omitempty"`
	LastBackupTime   *time.Time              `json:"last_backup_time,omitempty"`
	LastBackupStatus string                  `json:"last_backup_status,omitempty"`
	NextBackupTime   *time.Time              `json:"next_backup_time,omitempty"`
	TotalBackups     int                     `json:"total_backups"`
	TotalSizeBytes   int64                   `json:"total_size_bytes"`
}

// VerifyBackup verifies a backup file's integrity.
func (s *DatabaseBackupService) VerifyBackup(ctx context.Context, backupID uuid.UUID) error {
	backup, err := s.store.GetDatabaseBackupByID(ctx, backupID)
	if err != nil {
		return fmt.Errorf("get backup: %w", err)
	}

	if backup.FilePath == "" {
		return fmt.Errorf("backup has no file path")
	}

	// Read the file
	data, err := os.ReadFile(backup.FilePath)
	if err != nil {
		return fmt.Errorf("read backup file: %w", err)
	}

	// Verify checksum
	if backup.Checksum != "" {
		actualChecksum := s.calculateChecksum(data)
		if actualChecksum != backup.Checksum {
			return fmt.Errorf("checksum mismatch: expected %s, got %s", backup.Checksum, actualChecksum)
		}
	}

	// Try to decrypt
	decrypted, err := s.keyManager.Decrypt(data)
	if err != nil {
		return fmt.Errorf("decrypt backup: %w", err)
	}

	// Try to decompress
	reader, err := gzip.NewReader(bytes.NewReader(decrypted))
	if err != nil {
		return fmt.Errorf("decompress backup: %w", err)
	}
	defer reader.Close()

	// Read a small portion to verify it's valid SQL
	header := make([]byte, 1024)
	n, err := reader.Read(header)
	if err != nil && err != io.EOF {
		return fmt.Errorf("read decompressed data: %w", err)
	}

	// Check for SQL content
	headerStr := string(header[:n])
	if !strings.Contains(headerStr, "PostgreSQL") && !strings.Contains(headerStr, "pg_dump") && !strings.Contains(headerStr, "CREATE") {
		return fmt.Errorf("backup does not appear to contain valid PostgreSQL dump data")
	}

	s.logger.Info().
		Str("backup_id", backupID.String()).
		Msg("backup verification passed")

	return nil
}

// RestoreInstructions returns instructions for restoring from a backup.
func (s *DatabaseBackupService) RestoreInstructions(backupID uuid.UUID) string {
	return fmt.Sprintf(`Database Restore Instructions
=============================

To restore from backup %s:

1. Stop the Keldris server:
   systemctl stop keldris-server

2. Locate the backup file in: %s

3. Decrypt the backup (requires the master encryption key):
   keldris-cli db decrypt-backup --backup-id %s --output backup.sql.gz

4. Decompress:
   gunzip backup.sql.gz

5. Drop and recreate the database:
   dropdb keldris
   createdb keldris

6. Restore the backup:
   psql keldris < backup.sql

7. Run any pending migrations:
   keldris-server migrate

8. Restart the Keldris server:
   systemctl start keldris-server

IMPORTANT NOTES:
- Always verify the backup before restoring
- Create a fresh backup of the current database before restoring
- Test restore procedures in a non-production environment first
- The encryption key must match the one used to create the backup
`, backupID, s.config.BackupDir, backupID)
}

// TriggerBackup triggers an immediate backup (for manual triggering).
func (s *DatabaseBackupService) TriggerBackup(ctx context.Context) (*models.DatabaseBackup, error) {
	return s.CreateBackup(ctx)
}

// IsHealthy returns true if the backup service is healthy.
func (s *DatabaseBackupService) IsHealthy(ctx context.Context) (bool, string) {
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return false, "backup service not running"
	}

	// Check if we have a recent successful backup
	latest, err := s.store.GetLatestDatabaseBackup(ctx)
	if err != nil {
		return false, fmt.Sprintf("failed to get latest backup: %v", err)
	}

	if latest == nil {
		return false, "no backups exist"
	}

	if latest.Status == models.DatabaseBackupStatusFailed {
		return false, fmt.Sprintf("last backup failed: %s", latest.ErrorMessage)
	}

	// Check if backup is within expected interval (last 25 hours for daily backup)
	if time.Since(latest.CreatedAt) > 25*time.Hour {
		return false, fmt.Sprintf("last backup is older than 25 hours: %s", latest.CreatedAt)
	}

	return true, "backup service healthy"
}
