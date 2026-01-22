// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"context"
	"time"
)

// FileVersion represents a single version of a file across snapshots.
type FileVersion struct {
	SnapshotID   string    `json:"snapshot_id"`
	SnapshotTime time.Time `json:"snapshot_time"`
	FilePath     string    `json:"file_path"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Mode         uint32    `json:"mode"`
}

// FileHistory contains the history of a file across all snapshots.
type FileHistory struct {
	FilePath string        `json:"file_path"`
	Versions []FileVersion `json:"versions"`
}

// GetFileHistory queries restic for all snapshots containing a specific file
// and returns the file's metadata from each snapshot.
func (r *Restic) GetFileHistory(ctx context.Context, cfg ResticConfig, filePath string) (*FileHistory, error) {
	r.logger.Debug().
		Str("file_path", filePath).
		Msg("getting file history")

	// Get all snapshots in the repository
	snapshots, err := r.Snapshots(ctx, cfg)
	if err != nil {
		return nil, err
	}

	history := &FileHistory{
		FilePath: filePath,
		Versions: make([]FileVersion, 0),
	}

	// For each snapshot, check if the file exists and get its metadata
	for _, snap := range snapshots {
		files, err := r.ListFiles(ctx, cfg, snap.ID, filePath)
		if err != nil {
			// If snapshot not found or other error, skip this snapshot
			r.logger.Debug().
				Err(err).
				Str("snapshot_id", snap.ID).
				Msg("skipping snapshot due to error")
			continue
		}

		// Find the exact file match
		for _, file := range files {
			if file.Path == filePath && file.Type == "file" {
				history.Versions = append(history.Versions, FileVersion{
					SnapshotID:   snap.ID,
					SnapshotTime: snap.Time,
					FilePath:     file.Path,
					Size:         file.Size,
					ModTime:      file.ModTime,
					Mode:         file.Mode,
				})
				break
			}
		}
	}

	r.logger.Debug().
		Int("version_count", len(history.Versions)).
		Msg("file history retrieved")

	return history, nil
}

// FindFileInSnapshots searches for files matching a path pattern across all snapshots.
// This is useful for discovering files when the exact path is not known.
func (r *Restic) FindFileInSnapshots(ctx context.Context, cfg ResticConfig, pathPrefix string) ([]FileVersion, error) {
	r.logger.Debug().
		Str("path_prefix", pathPrefix).
		Msg("finding files in snapshots")

	snapshots, err := r.Snapshots(ctx, cfg)
	if err != nil {
		return nil, err
	}

	var results []FileVersion

	for _, snap := range snapshots {
		files, err := r.ListFiles(ctx, cfg, snap.ID, pathPrefix)
		if err != nil {
			continue
		}

		for _, file := range files {
			if file.Type == "file" {
				results = append(results, FileVersion{
					SnapshotID:   snap.ID,
					SnapshotTime: snap.Time,
					FilePath:     file.Path,
					Size:         file.Size,
					ModTime:      file.ModTime,
					Mode:         file.Mode,
				})
			}
		}
	}

	r.logger.Debug().
		Int("result_count", len(results)).
		Msg("files found in snapshots")

	return results, nil
}

// RestoreFileVersion restores a specific file version from a snapshot.
func (r *Restic) RestoreFileVersion(ctx context.Context, cfg ResticConfig, snapshotID, filePath, targetPath string) error {
	r.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("file_path", filePath).
		Str("target_path", targetPath).
		Msg("restoring file version")

	opts := RestoreOptions{
		TargetPath: targetPath,
		Include:    []string{filePath},
	}

	return r.Restore(ctx, cfg, snapshotID, opts)
}
