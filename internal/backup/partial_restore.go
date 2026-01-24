// Package backup provides partial restore functionality for Restic backups.
package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"
)

// Common errors for partial restore operations.
var (
	ErrPathNotInSnapshot = errors.New("path not found in snapshot")
	ErrInvalidPath       = errors.New("invalid path specification")
	ErrTargetExists      = errors.New("target directory already exists")
)

// PartialRestoreOptions configures a partial restore operation.
type PartialRestoreOptions struct {
	// SnapshotID is the ID of the snapshot to restore from.
	SnapshotID string
	// Paths is the list of paths to restore (files or directories).
	Paths []string
	// TargetPath is the destination directory. Empty means restore to original location.
	TargetPath string
	// Overwrite allows overwriting existing files at the target.
	Overwrite bool
}

// PartialRestoreResult contains the results of a partial restore operation.
type PartialRestoreResult struct {
	// RestoredFiles is the number of files successfully restored.
	RestoredFiles int
	// RestoredDirs is the number of directories created.
	RestoredDirs int
	// TotalSize is the total size of restored data in bytes.
	TotalSize int64
	// Skipped contains paths that were skipped (e.g., already exist).
	Skipped []string
	// Errors contains any non-fatal errors encountered.
	Errors []string
}

// PartialRestoreValidator provides validation for partial restore operations.
type PartialRestoreValidator struct {
	restic *Restic
	logger zerolog.Logger
}

// NewPartialRestoreValidator creates a new validator for partial restore operations.
func NewPartialRestoreValidator(restic *Restic, logger zerolog.Logger) *PartialRestoreValidator {
	return &PartialRestoreValidator{
		restic: restic,
		logger: logger.With().Str("component", "partial_restore_validator").Logger(),
	}
}

// ValidatePaths checks that all specified paths exist in the snapshot.
// Returns the validated paths that were found and any paths that were not found.
func (v *PartialRestoreValidator) ValidatePaths(ctx context.Context, cfg ResticConfig, snapshotID string, paths []string) (found []string, notFound []string, err error) {
	if len(paths) == 0 {
		return nil, nil, ErrInvalidPath
	}

	// List all files in the snapshot
	files, err := v.restic.ListFiles(ctx, cfg, snapshotID, "")
	if err != nil {
		return nil, nil, fmt.Errorf("list files in snapshot: %w", err)
	}

	// Build a set of all paths in the snapshot (including parent directories)
	snapshotPaths := make(map[string]bool)
	for _, f := range files {
		snapshotPaths[f.Path] = true
		// Also add all parent directories
		dir := filepath.Dir(f.Path)
		for dir != "/" && dir != "." {
			snapshotPaths[dir] = true
			dir = filepath.Dir(dir)
		}
	}

	// Check each requested path
	for _, path := range paths {
		path = normalizePath(path)
		if pathExistsInSnapshot(path, snapshotPaths, files) {
			found = append(found, path)
		} else {
			notFound = append(notFound, path)
		}
	}

	return found, notFound, nil
}

// pathExistsInSnapshot checks if a path exists in the snapshot.
// Handles both exact matches and directory prefixes.
func pathExistsInSnapshot(path string, pathSet map[string]bool, files []SnapshotFile) bool {
	// Check exact match
	if pathSet[path] {
		return true
	}

	// Check if it's a prefix of any file path (directory match)
	pathWithSlash := path
	if !strings.HasSuffix(pathWithSlash, "/") {
		pathWithSlash = path + "/"
	}

	for _, f := range files {
		if strings.HasPrefix(f.Path, pathWithSlash) {
			return true
		}
	}

	return false
}

// normalizePath ensures consistent path formatting.
func normalizePath(path string) string {
	// Clean the path
	path = filepath.Clean(path)

	// Ensure it starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path
}

// GetPathsInfo returns detailed information about the specified paths in a snapshot.
func (v *PartialRestoreValidator) GetPathsInfo(ctx context.Context, cfg ResticConfig, snapshotID string, paths []string) ([]SnapshotFile, error) {
	// List all files in the snapshot
	allFiles, err := v.restic.ListFiles(ctx, cfg, snapshotID, "")
	if err != nil {
		return nil, fmt.Errorf("list files in snapshot: %w", err)
	}

	if len(paths) == 0 {
		return allFiles, nil
	}

	// Filter to only the requested paths
	var result []SnapshotFile
	for _, f := range allFiles {
		for _, path := range paths {
			path = normalizePath(path)
			// Check exact match or prefix match (for directories)
			if f.Path == path || strings.HasPrefix(f.Path, path+"/") {
				result = append(result, f)
				break
			}
		}
	}

	return result, nil
}

// CalculateRestoreSize calculates the total size of files that would be restored.
func (v *PartialRestoreValidator) CalculateRestoreSize(ctx context.Context, cfg ResticConfig, snapshotID string, paths []string) (totalFiles int, totalDirs int, totalSize int64, err error) {
	files, err := v.GetPathsInfo(ctx, cfg, snapshotID, paths)
	if err != nil {
		return 0, 0, 0, err
	}

	for _, f := range files {
		if f.Type == "file" {
			totalFiles++
			totalSize += f.Size
		} else if f.Type == "dir" {
			totalDirs++
		}
	}

	return totalFiles, totalDirs, totalSize, nil
}

// PrepareTargetDirectory ensures the target directory exists and is ready for restore.
func PrepareTargetDirectory(targetPath string, createIfNotExists bool) error {
	if targetPath == "" || targetPath == "/" {
		// Restoring to original location, no preparation needed
		return nil
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			if createIfNotExists {
				if err := os.MkdirAll(targetPath, 0755); err != nil {
					return fmt.Errorf("create target directory: %w", err)
				}
				return nil
			}
			return fmt.Errorf("target directory does not exist: %s", targetPath)
		}
		return fmt.Errorf("check target directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("target path is not a directory: %s", targetPath)
	}

	return nil
}

// ConvertPathsToIncludePatterns converts user-selected paths to restic --include patterns.
// Restic uses glob-style patterns for --include.
func ConvertPathsToIncludePatterns(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	patterns := make([]string, 0, len(paths))
	for _, path := range paths {
		path = normalizePath(path)
		patterns = append(patterns, path)
		// Also add pattern to include all children for directories
		if !strings.HasSuffix(path, "/*") {
			patterns = append(patterns, path+"/*")
		}
	}

	return patterns
}

// PartialRestore performs a partial restore of specific paths from a snapshot.
func (r *Restic) PartialRestore(ctx context.Context, cfg ResticConfig, opts PartialRestoreOptions) (*PartialRestoreResult, error) {
	r.logger.Info().
		Str("snapshot_id", opts.SnapshotID).
		Strs("paths", opts.Paths).
		Str("target_path", opts.TargetPath).
		Bool("overwrite", opts.Overwrite).
		Msg("starting partial restore")

	// Build restore options
	restoreOpts := RestoreOptions{
		TargetPath: opts.TargetPath,
		Include:    ConvertPathsToIncludePatterns(opts.Paths),
		DryRun:     false,
	}

	// If restoring to original location, use /
	if restoreOpts.TargetPath == "" {
		restoreOpts.TargetPath = "/"
	}

	// Perform the restore
	err := r.Restore(ctx, cfg, opts.SnapshotID, restoreOpts)
	if err != nil {
		return nil, fmt.Errorf("partial restore failed: %w", err)
	}

	// For now, return a basic result since restic doesn't provide detailed stats
	// In a full implementation, we would parse restic's JSON output for details
	result := &PartialRestoreResult{
		RestoredFiles: 0, // Would be populated from restic output
		RestoredDirs:  0,
		TotalSize:     0,
		Skipped:       nil,
		Errors:        nil,
	}

	r.logger.Info().
		Str("snapshot_id", opts.SnapshotID).
		Int("restored_files", result.RestoredFiles).
		Msg("partial restore completed")

	return result, nil
}

// PreviewPartialRestore generates a preview of what would be restored in a partial restore.
func (r *Restic) PreviewPartialRestore(ctx context.Context, cfg ResticConfig, snapshotID string, paths []string, targetPath string) (*RestorePreview, error) {
	r.logger.Info().
		Str("snapshot_id", snapshotID).
		Strs("paths", paths).
		Str("target_path", targetPath).
		Msg("generating partial restore preview")

	// Create restore options with include patterns
	opts := RestoreOptions{
		TargetPath: targetPath,
		Include:    paths,
		Exclude:    nil,
	}

	// If restoring to original location
	if opts.TargetPath == "" {
		opts.TargetPath = "/"
	}

	// Use the existing preview functionality
	return r.RestorePreviewResult(ctx, cfg, snapshotID, opts)
}
