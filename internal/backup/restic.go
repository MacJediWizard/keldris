// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// ErrRepositoryNotInitialized is returned when the repository has not been initialized.
var ErrRepositoryNotInitialized = errors.New("repository not initialized")

// ErrSnapshotNotFound is returned when a snapshot cannot be found.
var ErrSnapshotNotFound = errors.New("snapshot not found")

// ResticConfig holds configuration for a Restic backup operation.
type ResticConfig struct {
	Repository string
	Password   string
	Env        map[string]string
}

// Snapshot represents a Restic snapshot.
type Snapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Username string    `json:"username"`
	Paths    []string  `json:"paths"`
	Tags     []string  `json:"tags,omitempty"`
}

// BackupStats contains statistics from a backup operation.
type BackupStats struct {
	SnapshotID   string
	FilesNew     int
	FilesChanged int
	SizeBytes    int64
	Duration     time.Duration
}

// Restic wraps the restic CLI for backup operations.
type Restic struct {
	binary string
	logger zerolog.Logger
}

// NewRestic creates a new Restic wrapper.
func NewRestic(logger zerolog.Logger) *Restic {
	return &Restic{
		binary: "restic",
		logger: logger.With().Str("component", "restic").Logger(),
	}
}

// NewResticWithBinary creates a new Restic wrapper with a custom binary path.
func NewResticWithBinary(binary string, logger zerolog.Logger) *Restic {
	return &Restic{
		binary: binary,
		logger: logger.With().Str("component", "restic").Logger(),
	}
}

// Init initializes a new Restic repository.
func (r *Restic) Init(ctx context.Context, cfg ResticConfig) error {
	r.logger.Info().Str("repository", cfg.Repository).Msg("initializing repository")

	args := []string{"init", "--repo", cfg.Repository, "--json"}
	_, err := r.run(ctx, cfg, args)
	if err != nil {
		// Check if already initialized
		if strings.Contains(err.Error(), "already exists") ||
			strings.Contains(err.Error(), "already initialized") {
			r.logger.Debug().Msg("repository already initialized")
			return nil
		}
		return fmt.Errorf("init repository: %w", err)
	}

	r.logger.Info().Msg("repository initialized successfully")
	return nil
}

// Backup runs a backup operation with the given paths and excludes.
func (r *Restic) Backup(ctx context.Context, cfg ResticConfig, paths, excludes []string, tags []string) (*BackupStats, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths specified for backup")
	}

	r.logger.Info().
		Strs("paths", paths).
		Strs("excludes", excludes).
		Strs("tags", tags).
		Msg("starting backup")

	start := time.Now()

	args := []string{"backup", "--repo", cfg.Repository, "--json"}

	for _, exclude := range excludes {
		args = append(args, "--exclude", exclude)
	}

	for _, tag := range tags {
		args = append(args, "--tag", tag)
	}

	args = append(args, paths...)

	output, err := r.run(ctx, cfg, args)
	if err != nil {
		return nil, fmt.Errorf("backup failed: %w", err)
	}

	stats, err := parseBackupOutput(output)
	if err != nil {
		return nil, fmt.Errorf("parse backup output: %w", err)
	}

	stats.Duration = time.Since(start)

	r.logger.Info().
		Str("snapshot_id", stats.SnapshotID).
		Int("files_new", stats.FilesNew).
		Int("files_changed", stats.FilesChanged).
		Int64("size_bytes", stats.SizeBytes).
		Dur("duration", stats.Duration).
		Msg("backup completed")

	return stats, nil
}

// Snapshots lists all snapshots in the repository.
func (r *Restic) Snapshots(ctx context.Context, cfg ResticConfig) ([]Snapshot, error) {
	r.logger.Debug().Msg("listing snapshots")

	args := []string{"snapshots", "--repo", cfg.Repository, "--json"}
	output, err := r.run(ctx, cfg, args)
	if err != nil {
		if strings.Contains(err.Error(), "repository does not exist") {
			return nil, ErrRepositoryNotInitialized
		}
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	var snapshots []Snapshot
	if err := json.Unmarshal(output, &snapshots); err != nil {
		return nil, fmt.Errorf("parse snapshots: %w", err)
	}

	r.logger.Debug().Int("count", len(snapshots)).Msg("snapshots listed")
	return snapshots, nil
}

// SnapshotFile represents a file or directory in a snapshot.
type SnapshotFile struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"` // "file" or "dir"
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Mode       uint32    `json:"mode"`
	ModTime    time.Time `json:"mtime"`
	AccessTime time.Time `json:"atime"`
	ChangeTime time.Time `json:"ctime"`
}

// ListFiles lists files in a snapshot, optionally filtered by path prefix.
func (r *Restic) ListFiles(ctx context.Context, cfg ResticConfig, snapshotID, pathPrefix string) ([]SnapshotFile, error) {
	r.logger.Debug().
		Str("snapshot_id", snapshotID).
		Str("path_prefix", pathPrefix).
		Msg("listing files in snapshot")

	args := []string{"ls", "--repo", cfg.Repository, "--json", snapshotID}
	if pathPrefix != "" {
		args = append(args, pathPrefix)
	}

	output, err := r.run(ctx, cfg, args)
	if err != nil {
		if strings.Contains(err.Error(), "no matching ID") {
			return nil, ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("list files: %w", err)
	}

	var files []SnapshotFile
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var file SnapshotFile
		if err := json.Unmarshal(line, &file); err != nil {
			// Skip snapshot metadata line (first line)
			continue
		}
		// Only include files and directories
		if file.Type == "file" || file.Type == "dir" {
			files = append(files, file)
		}
	}

	r.logger.Debug().Int("count", len(files)).Msg("files listed")
	return files, nil
}

// RestoreOptions configures a restore operation.
type RestoreOptions struct {
	TargetPath string   // Destination path for restore
	Include    []string // Paths to include (empty = all)
	Exclude    []string // Paths to exclude
}

// Restore restores a snapshot to the given target path.
func (r *Restic) Restore(ctx context.Context, cfg ResticConfig, snapshotID string, opts RestoreOptions) error {
	r.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("target_path", opts.TargetPath).
		Strs("include", opts.Include).
		Strs("exclude", opts.Exclude).
		Msg("starting restore")

	args := []string{
		"restore",
		"--repo", cfg.Repository,
		"--target", opts.TargetPath,
		"--json",
	}

	for _, include := range opts.Include {
		args = append(args, "--include", include)
	}

	for _, exclude := range opts.Exclude {
		args = append(args, "--exclude", exclude)
	}

	args = append(args, snapshotID)

	_, err := r.run(ctx, cfg, args)
	if err != nil {
		if strings.Contains(err.Error(), "no matching ID") {
			return ErrSnapshotNotFound
		}
		return fmt.Errorf("restore failed: %w", err)
	}

	r.logger.Info().Msg("restore completed successfully")
	return nil
}

// Prune removes old snapshots according to the retention policy.
func (r *Restic) Prune(ctx context.Context, cfg ResticConfig, retention *models.RetentionPolicy) error {
	if retention == nil {
		return errors.New("retention policy required for prune")
	}

	r.logger.Info().
		Interface("retention", retention).
		Msg("starting prune with retention policy")

	// First, apply forget with retention policy
	forgetArgs := r.buildRetentionArgs(cfg.Repository, retention)
	_, err := r.run(ctx, cfg, forgetArgs)
	if err != nil {
		return fmt.Errorf("forget failed: %w", err)
	}

	// Then, prune unreferenced data
	pruneArgs := []string{"prune", "--repo", cfg.Repository, "--json"}
	_, err = r.run(ctx, cfg, pruneArgs)
	if err != nil {
		return fmt.Errorf("prune failed: %w", err)
	}

	r.logger.Info().Msg("prune completed successfully")
	return nil
}

// Check verifies the repository integrity.
func (r *Restic) Check(ctx context.Context, cfg ResticConfig) error {
	r.logger.Debug().Msg("checking repository integrity")

	args := []string{"check", "--repo", cfg.Repository, "--json"}
	_, err := r.run(ctx, cfg, args)
	if err != nil {
		return fmt.Errorf("check failed: %w", err)
	}

	r.logger.Debug().Msg("repository check passed")
	return nil
}

// Stats returns statistics about the repository.
func (r *Restic) Stats(ctx context.Context, cfg ResticConfig) (*RepoStats, error) {
	r.logger.Debug().Msg("getting repository stats")

	args := []string{"stats", "--repo", cfg.Repository, "--json"}
	output, err := r.run(ctx, cfg, args)
	if err != nil {
		return nil, fmt.Errorf("stats failed: %w", err)
	}

	var stats RepoStats
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("parse stats: %w", err)
	}

	return &stats, nil
}

// RepoStats contains repository statistics.
type RepoStats struct {
	TotalSize      int64 `json:"total_size"`
	TotalFileCount int   `json:"total_file_count"`
}

// buildRetentionArgs builds the forget command arguments from a retention policy.
func (r *Restic) buildRetentionArgs(repository string, retention *models.RetentionPolicy) []string {
	args := []string{"forget", "--repo", repository, "--json", "--prune"}

	if retention.KeepLast > 0 {
		args = append(args, "--keep-last", fmt.Sprintf("%d", retention.KeepLast))
	}
	if retention.KeepHourly > 0 {
		args = append(args, "--keep-hourly", fmt.Sprintf("%d", retention.KeepHourly))
	}
	if retention.KeepDaily > 0 {
		args = append(args, "--keep-daily", fmt.Sprintf("%d", retention.KeepDaily))
	}
	if retention.KeepWeekly > 0 {
		args = append(args, "--keep-weekly", fmt.Sprintf("%d", retention.KeepWeekly))
	}
	if retention.KeepMonthly > 0 {
		args = append(args, "--keep-monthly", fmt.Sprintf("%d", retention.KeepMonthly))
	}
	if retention.KeepYearly > 0 {
		args = append(args, "--keep-yearly", fmt.Sprintf("%d", retention.KeepYearly))
	}

	return args
}

// run executes a restic command and returns the output.
func (r *Restic) run(ctx context.Context, cfg ResticConfig, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, r.binary, args...)

	// Set environment variables
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("RESTIC_PASSWORD=%s", cfg.Password))
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	r.logger.Debug().
		Str("command", r.binary).
		Strs("args", redactArgs(args)).
		Msg("executing restic command")

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
	}

	return stdout.Bytes(), nil
}

// redactArgs removes sensitive information from args for logging.
func redactArgs(args []string) []string {
	redacted := make([]string, len(args))
	copy(redacted, args)
	// Repository passwords are passed via env, so args should be safe to log
	return redacted
}

// backupSummary represents the JSON output from restic backup --json.
type backupSummary struct {
	MessageType      string  `json:"message_type"`
	SnapshotID       string  `json:"snapshot_id"`
	FilesNew         int     `json:"files_new"`
	FilesChanged     int     `json:"files_changed"`
	FilesUnmodified  int     `json:"files_unmodified"`
	DirsNew          int     `json:"dirs_new"`
	DirsChanged      int     `json:"dirs_changed"`
	DirsUnmodified   int     `json:"dirs_unmodified"`
	DataBlobs        int     `json:"data_blobs"`
	TreeBlobs        int     `json:"tree_blobs"`
	DataAdded        int64   `json:"data_added"`
	TotalFilesProc   int     `json:"total_files_processed"`
	TotalBytesProc   int64   `json:"total_bytes_processed"`
	TotalDuration    float64 `json:"total_duration"`
	SnapshotFileSize int64   `json:"snapshot_file_size,omitempty"`
}

// parseBackupOutput parses the JSON output from restic backup.
func parseBackupOutput(output []byte) (*BackupStats, error) {
	// Restic outputs multiple JSON lines, find the summary
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var msg struct {
			MessageType string `json:"message_type"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.MessageType == "summary" {
			var summary backupSummary
			if err := json.Unmarshal(line, &summary); err != nil {
				return nil, fmt.Errorf("parse summary: %w", err)
			}

			return &BackupStats{
				SnapshotID:   summary.SnapshotID,
				FilesNew:     summary.FilesNew,
				FilesChanged: summary.FilesChanged,
				SizeBytes:    summary.DataAdded,
			}, nil
		}
	}

	return nil, errors.New("no backup summary found in output")
}
