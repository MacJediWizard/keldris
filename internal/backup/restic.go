// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// ErrRepositoryNotInitialized is returned when the repository has not been initialized.
var ErrRepositoryNotInitialized = errors.New("repository not initialized")

// ErrSnapshotNotFound is returned when a snapshot cannot be found.
var ErrSnapshotNotFound = errors.New("snapshot not found")

// ResticConfig is an alias to backends.ResticConfig for backwards compatibility.
type ResticConfig = backends.ResticConfig

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

// DryRunFile represents a file that would be backed up in a dry run.
type DryRunFile struct {
	Path   string `json:"path"`
	Type   string `json:"type"` // "file" or "dir"
	Size   int64  `json:"size"`
	Action string `json:"action"` // "new", "changed", or "unchanged"
}

// DryRunExcluded represents a file that was excluded from backup.
type DryRunExcluded struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// DryRunResult contains the results of a dry run backup operation.
type DryRunResult struct {
	FilesToBackup   []DryRunFile     `json:"files_to_backup"`
	ExcludedFiles   []DryRunExcluded `json:"excluded_files"`
	TotalFiles      int              `json:"total_files"`
	TotalSize       int64            `json:"total_size"`
	NewFiles        int              `json:"new_files"`
	ChangedFiles    int              `json:"changed_files"`
	UnchangedFiles  int              `json:"unchanged_files"`
	Duration        time.Duration    `json:"duration"`
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

// BackupOptions contains optional parameters for backup operations.
type BackupOptions struct {
	BandwidthLimitKB *int    // Upload bandwidth limit in KB/s (nil = unlimited)
	CompressionLevel *string // Compression level: off, auto, max (nil = restic default "auto")
	MaxFileSizeMB    *int    // Maximum file size in MB to include (nil/0 = no limit)
}

// Backup runs a backup operation with the given paths and excludes.
func (r *Restic) Backup(ctx context.Context, cfg ResticConfig, paths, excludes []string, tags []string) (*BackupStats, error) {
	return r.BackupWithOptions(ctx, cfg, paths, excludes, tags, nil)
}

// BackupWithOptions runs a backup operation with additional options.
func (r *Restic) BackupWithOptions(ctx context.Context, cfg ResticConfig, paths, excludes []string, tags []string, opts *BackupOptions) (*BackupStats, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths specified for backup")
	}

	logEvent := r.logger.Info().
		Strs("paths", paths).
		Strs("excludes", excludes).
		Strs("tags", tags)

	if opts != nil && opts.BandwidthLimitKB != nil {
		logEvent = logEvent.Int("bandwidth_limit_kb", *opts.BandwidthLimitKB)
	}
	if opts != nil && opts.CompressionLevel != nil {
		logEvent = logEvent.Str("compression_level", *opts.CompressionLevel)
	}
	if opts != nil && opts.MaxFileSizeMB != nil && *opts.MaxFileSizeMB > 0 {
		logEvent = logEvent.Int("max_file_size_mb", *opts.MaxFileSizeMB)
	}
	logEvent.Msg("starting backup")

	start := time.Now()

	args := []string{"backup", "--repo", cfg.Repository, "--json"}

	// Add bandwidth limit if specified
	if opts != nil && opts.BandwidthLimitKB != nil && *opts.BandwidthLimitKB > 0 {
		// Restic expects --limit-upload in KiB/s
		args = append(args, "--limit-upload", fmt.Sprintf("%d", *opts.BandwidthLimitKB))
	}

	// Add compression level if specified
	if opts != nil && opts.CompressionLevel != nil && *opts.CompressionLevel != "" {
		args = append(args, "--compression", *opts.CompressionLevel)
	}

	// Add max file size exclusion if specified
	if opts != nil && opts.MaxFileSizeMB != nil && *opts.MaxFileSizeMB > 0 {
		args = append(args, "--exclude-larger-than", fmt.Sprintf("%dM", *opts.MaxFileSizeMB))
	}

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

// DryRun performs a dry run backup operation to preview what would be backed up.
func (r *Restic) DryRun(ctx context.Context, cfg ResticConfig, paths, excludes []string) (*DryRunResult, error) {
	if len(paths) == 0 {
		return nil, errors.New("no paths specified for dry run")
	}

	r.logger.Info().
		Strs("paths", paths).
		Strs("excludes", excludes).
		Msg("starting dry run backup")

	start := time.Now()

	args := []string{"backup", "--repo", cfg.Repository, "--json", "--dry-run"}

	for _, exclude := range excludes {
		args = append(args, "--exclude", exclude)
	}

	args = append(args, paths...)

	output, err := r.run(ctx, cfg, args)
	if err != nil {
		return nil, fmt.Errorf("dry run failed: %w", err)
	}

	result, err := parseDryRunOutput(output, excludes)
	if err != nil {
		return nil, fmt.Errorf("parse dry run output: %w", err)
	}

	result.Duration = time.Since(start)

	r.logger.Info().
		Int("total_files", result.TotalFiles).
		Int("new_files", result.NewFiles).
		Int("changed_files", result.ChangedFiles).
		Int64("total_size", result.TotalSize).
		Dur("duration", result.Duration).
		Msg("dry run completed")

	return result, nil
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
	DryRun     bool     // If true, only preview what would be restored
}

// RestorePreviewFile represents a file that would be restored.
type RestorePreviewFile struct {
	Path       string    `json:"path"`
	Type       string    `json:"type"` // "file" or "dir"
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mtime"`
	Mode       uint32    `json:"mode"`
	HasConflict bool     `json:"has_conflict,omitempty"` // True if file exists at target
}

// RestorePreview contains the preview results from a dry-run restore.
type RestorePreview struct {
	SnapshotID     string               `json:"snapshot_id"`
	TargetPath     string               `json:"target_path"`
	TotalFiles     int                  `json:"total_files"`
	TotalDirs      int                  `json:"total_dirs"`
	TotalSize      int64                `json:"total_size"`
	ConflictCount  int                  `json:"conflict_count"`
	Files          []RestorePreviewFile `json:"files"`
}

// Restore restores a snapshot to the given target path.
func (r *Restic) Restore(ctx context.Context, cfg ResticConfig, snapshotID string, opts RestoreOptions) error {
	r.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("target_path", opts.TargetPath).
		Strs("include", opts.Include).
		Strs("exclude", opts.Exclude).
		Bool("dry_run", opts.DryRun).
		Msg("starting restore")

	args := []string{
		"restore",
		"--repo", cfg.Repository,
		"--target", opts.TargetPath,
		"--json",
	}

	if opts.DryRun {
		args = append(args, "--dry-run")
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

// RestorePreviewResult returns a preview of what would be restored.
// This uses restic's --dry-run flag combined with file listing to generate a preview.
func (r *Restic) RestorePreviewResult(ctx context.Context, cfg ResticConfig, snapshotID string, opts RestoreOptions) (*RestorePreview, error) {
	r.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("target_path", opts.TargetPath).
		Strs("include", opts.Include).
		Strs("exclude", opts.Exclude).
		Msg("generating restore preview")

	// First, list all files that would be restored using ls command
	// This gives us the complete file information
	files, err := r.ListFiles(ctx, cfg, snapshotID, "")
	if err != nil {
		return nil, fmt.Errorf("list files for preview: %w", err)
	}

	// Filter files based on include/exclude patterns
	filteredFiles := filterFilesByPatterns(files, opts.Include, opts.Exclude)

	// Build preview result
	preview := &RestorePreview{
		SnapshotID: snapshotID,
		TargetPath: opts.TargetPath,
		Files:      make([]RestorePreviewFile, 0, len(filteredFiles)),
	}

	for _, f := range filteredFiles {
		previewFile := RestorePreviewFile{
			Path:    f.Path,
			Type:    f.Type,
			Size:    f.Size,
			ModTime: f.ModTime,
			Mode:    f.Mode,
		}

		if f.Type == "file" {
			preview.TotalFiles++
			preview.TotalSize += f.Size
		} else if f.Type == "dir" {
			preview.TotalDirs++
		}

		// Check if file would conflict with existing file at target
		targetPath := opts.TargetPath + f.Path
		if fileExists(targetPath) {
			previewFile.HasConflict = true
			preview.ConflictCount++
		}

		preview.Files = append(preview.Files, previewFile)
	}

	r.logger.Info().
		Int("total_files", preview.TotalFiles).
		Int("total_dirs", preview.TotalDirs).
		Int64("total_size", preview.TotalSize).
		Int("conflict_count", preview.ConflictCount).
		Msg("restore preview generated")

	return preview, nil
}

// filterFilesByPatterns filters files based on include/exclude patterns.
func filterFilesByPatterns(files []SnapshotFile, include, exclude []string) []SnapshotFile {
	if len(include) == 0 && len(exclude) == 0 {
		return files
	}

	var result []SnapshotFile
	for _, f := range files {
		// Check exclude patterns first
		excluded := false
		for _, pattern := range exclude {
			if matchPattern(f.Path, pattern) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		// If include patterns are specified, check them
		if len(include) > 0 {
			included := false
			for _, pattern := range include {
				if matchPattern(f.Path, pattern) {
					included = true
					break
				}
			}
			if !included {
				continue
			}
		}

		result = append(result, f)
	}
	return result
}

// matchPattern checks if a path matches a pattern (simple prefix/suffix matching).
func matchPattern(path, pattern string) bool {
	// Simple pattern matching: prefix or exact match
	if strings.HasPrefix(path, pattern) {
		return true
	}
	// Check if pattern is a parent directory
	if strings.HasPrefix(path, pattern+"/") {
		return true
	}
	return path == pattern
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Forget removes old snapshots according to the retention policy and returns stats.
func (r *Restic) Forget(ctx context.Context, cfg ResticConfig, retention *models.RetentionPolicy) (*ForgetResult, error) {
	if retention == nil {
		return nil, errors.New("retention policy required for forget")
	}

	r.logger.Info().
		Interface("retention", retention).
		Msg("starting forget with retention policy")

	forgetArgs := r.buildRetentionArgs(cfg.Repository, retention)
	// Remove --prune from forget args - we'll call prune separately if needed
	prunelessArgs := make([]string, 0, len(forgetArgs))
	for _, arg := range forgetArgs {
		if arg != "--prune" {
			prunelessArgs = append(prunelessArgs, arg)
		}
	}

	output, err := r.run(ctx, cfg, prunelessArgs)
	if err != nil {
		return nil, fmt.Errorf("forget failed: %w", err)
	}

	result, err := parseForgetOutput(output)
	if err != nil {
		return nil, fmt.Errorf("parse forget output: %w", err)
	}

	r.logger.Info().
		Int("snapshots_removed", result.SnapshotsRemoved).
		Int("snapshots_kept", result.SnapshotsKept).
		Msg("forget completed")

	return result, nil
}

// Prune removes old snapshots according to the retention policy.
func (r *Restic) Prune(ctx context.Context, cfg ResticConfig, retention *models.RetentionPolicy) (*ForgetResult, error) {
	if retention == nil {
		return nil, errors.New("retention policy required for prune")
	}

	r.logger.Info().
		Interface("retention", retention).
		Msg("starting prune with retention policy")

	// First, apply forget with retention policy
	forgetArgs := r.buildRetentionArgs(cfg.Repository, retention)
	output, err := r.run(ctx, cfg, forgetArgs)
	if err != nil {
		return nil, fmt.Errorf("forget failed: %w", err)
	}

	result, err := parseForgetOutput(output)
	if err != nil {
		// If we can't parse the output, still return a minimal result
		r.logger.Warn().Err(err).Msg("failed to parse forget output")
		result = &ForgetResult{}
	}

	// Then, prune unreferenced data
	pruneArgs := []string{"prune", "--repo", cfg.Repository, "--json"}
	_, err = r.run(ctx, cfg, pruneArgs)
	if err != nil {
		return nil, fmt.Errorf("prune failed: %w", err)
	}

	r.logger.Info().
		Int("snapshots_removed", result.SnapshotsRemoved).
		Int("snapshots_kept", result.SnapshotsKept).
		Msg("prune completed successfully")

	return result, nil
}

// Copy copies a snapshot from one repository to another.
// This is used for replicating backups to secondary repositories.
func (r *Restic) Copy(ctx context.Context, sourceCfg, targetCfg ResticConfig, snapshotID string) error {
	r.logger.Info().
		Str("source_repo", sourceCfg.Repository).
		Str("target_repo", targetCfg.Repository).
		Str("snapshot_id", snapshotID).
		Msg("starting snapshot copy")

	args := []string{
		"copy",
		"--from-repo", sourceCfg.Repository,
		"--repo", targetCfg.Repository,
		"--json",
		snapshotID,
	}

	// Build environment with both passwords
	env := make(map[string]string)
	for k, v := range targetCfg.Env {
		env[k] = v
	}
	env["RESTIC_FROM_PASSWORD"] = sourceCfg.Password

	// Use target config with modified env
	copyCfg := ResticConfig{
		Repository: targetCfg.Repository,
		Password:   targetCfg.Password,
		Env:        env,
	}

	_, err := r.run(ctx, copyCfg, args)
	if err != nil {
		return fmt.Errorf("copy snapshot: %w", err)
	}

	r.logger.Info().
		Str("snapshot_id", snapshotID).
		Msg("snapshot copy completed")

	return nil
}

// CheckOptions configures the repository check operation.
type CheckOptions struct {
	// ReadData verifies the data of all pack files in the repository.
	// This is more thorough but much slower.
	ReadData bool
	// ReadDataSubset reads only a subset of pack files (e.g., "2.5%" or "5G").
	// Ignored if ReadData is false.
	ReadDataSubset string
}

// CheckResult contains the results of a repository check.
type CheckResult struct {
	// Errors contains any errors found during the check.
	Errors []string
	// Duration is how long the check took.
	Duration time.Duration
}

// Check verifies the repository integrity.
func (r *Restic) Check(ctx context.Context, cfg ResticConfig) error {
	_, err := r.CheckWithOptions(ctx, cfg, CheckOptions{})
	return err
}

// CheckWithOptions verifies the repository integrity with the given options.
func (r *Restic) CheckWithOptions(ctx context.Context, cfg ResticConfig, opts CheckOptions) (*CheckResult, error) {
	r.logger.Debug().
		Bool("read_data", opts.ReadData).
		Str("read_data_subset", opts.ReadDataSubset).
		Msg("checking repository integrity")

	start := time.Now()

	args := []string{"check", "--repo", cfg.Repository, "--json"}

	if opts.ReadData {
		if opts.ReadDataSubset != "" {
			args = append(args, "--read-data-subset", opts.ReadDataSubset)
		} else {
			args = append(args, "--read-data")
		}
	}

	output, err := r.run(ctx, cfg, args)
	result := &CheckResult{
		Duration: time.Since(start),
	}

	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		return result, fmt.Errorf("check failed: %w", err)
	}

	// Parse output for any warnings
	if len(output) > 0 {
		r.logger.Debug().Str("output", string(output)).Msg("check output")
	}

	r.logger.Debug().
		Dur("duration", result.Duration).
		Msg("repository check passed")

	return result, nil
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

// ForgetResult contains the results of a forget operation.
type ForgetResult struct {
	SnapshotsRemoved int      `json:"snapshots_removed"`
	SnapshotsKept    int      `json:"snapshots_kept"`
	RemovedIDs       []string `json:"removed_ids,omitempty"`
}

// forgetGroupOutput represents the JSON output from restic forget --json.
type forgetGroupOutput struct {
	Tags   []string          `json:"tags,omitempty"`
	Host   string            `json:"host,omitempty"`
	Paths  []string          `json:"paths,omitempty"`
	Keep   []forgetSnapshot  `json:"keep"`
	Remove []forgetSnapshot  `json:"remove"`
	Reason string            `json:"reasons,omitempty"`
}

type forgetSnapshot struct {
	ID       string    `json:"id"`
	ShortID  string    `json:"short_id"`
	Time     time.Time `json:"time"`
	Hostname string    `json:"hostname"`
	Paths    []string  `json:"paths"`
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

// dryRunStatusMessage represents the JSON output from restic backup --dry-run --json.
type dryRunStatusMessage struct {
	MessageType  string   `json:"message_type"`
	Action       string   `json:"action,omitempty"`       // "new", "modified", "unchanged"
	Item         string   `json:"item,omitempty"`         // file path
	DataSize     int64    `json:"data_size,omitempty"`    // size of the item
	TotalFiles   int      `json:"total_files,omitempty"`  // summary fields
	TotalBytes   int64    `json:"total_bytes,omitempty"`
	FilesNew     int      `json:"files_new,omitempty"`
	FilesChanged int      `json:"files_changed,omitempty"`
}

// parseDryRunOutput parses the JSON output from restic backup --dry-run.
func parseDryRunOutput(output []byte, excludes []string) (*DryRunResult, error) {
	result := &DryRunResult{
		FilesToBackup: make([]DryRunFile, 0),
		ExcludedFiles: make([]DryRunExcluded, 0),
	}

	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var msg dryRunStatusMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		switch msg.MessageType {
		case "status":
			// Status messages contain file information during dry run
			if msg.Item != "" {
				file := DryRunFile{
					Path: msg.Item,
					Type: "file",
					Size: msg.DataSize,
				}

				switch msg.Action {
				case "new":
					file.Action = "new"
					result.NewFiles++
				case "modified":
					file.Action = "changed"
					result.ChangedFiles++
				case "unchanged":
					file.Action = "unchanged"
					result.UnchangedFiles++
				default:
					file.Action = "new"
					result.NewFiles++
				}

				result.FilesToBackup = append(result.FilesToBackup, file)
				result.TotalFiles++
				result.TotalSize += msg.DataSize
			}
		case "summary":
			// Summary message at the end provides totals
			if msg.TotalFiles > 0 {
				result.TotalFiles = msg.TotalFiles
				result.TotalSize = msg.TotalBytes
				result.NewFiles = msg.FilesNew
				result.ChangedFiles = msg.FilesChanged
				result.UnchangedFiles = msg.TotalFiles - msg.FilesNew - msg.FilesChanged
			}
		}
	}

	// Add excluded patterns as excluded files info
	for _, pattern := range excludes {
		result.ExcludedFiles = append(result.ExcludedFiles, DryRunExcluded{
			Path:   pattern,
			Reason: "matched exclude pattern",
		})
	}

	return result, nil
}

// parseForgetOutput parses the JSON output from restic forget.
func parseForgetOutput(output []byte) (*ForgetResult, error) {
	// Try to parse as an array of forget group outputs
	var groups []forgetGroupOutput
	if err := json.Unmarshal(output, &groups); err == nil && len(groups) > 0 {
		result := &ForgetResult{}
		for _, group := range groups {
			result.SnapshotsKept += len(group.Keep)
			result.SnapshotsRemoved += len(group.Remove)
			for _, snap := range group.Remove {
				result.RemovedIDs = append(result.RemovedIDs, snap.ShortID)
			}
		}
		return result, nil
	}

	// If array parsing fails, try parsing line by line (restic sometimes outputs multiple JSON objects)
	lines := bytes.Split(output, []byte("\n"))
	result := &ForgetResult{}
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var group forgetGroupOutput
		if err := json.Unmarshal(line, &group); err != nil {
			continue
		}

		result.SnapshotsKept += len(group.Keep)
		result.SnapshotsRemoved += len(group.Remove)
		for _, snap := range group.Remove {
			result.RemovedIDs = append(result.RemovedIDs, snap.ShortID)
		}
	}

	return result, nil
}
