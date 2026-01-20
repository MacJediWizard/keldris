// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"context"
	"encoding/json"
	"fmt"
)

// RawStats represents the raw statistics from a restic repository.
type RawStats struct {
	TotalSize      int64 `json:"total_size"`
	TotalFileCount int   `json:"total_file_count"`
}

// RawStatsMode represents statistics with mode information from restic stats --mode.
type RawStatsMode struct {
	TotalSize         int64 `json:"total_size"`
	TotalFileCount    int   `json:"total_file_count"`
	TotalBlobCount    int64 `json:"total_blob_count,omitempty"`
	SnapshotsCount    int   `json:"snapshots_count,omitempty"`
	TotalUncompressed int64 `json:"total_uncompressed_size,omitempty"`
}

// ExtendedRepoStats contains comprehensive repository statistics including dedup metrics.
type ExtendedRepoStats struct {
	// Basic stats
	TotalSize      int64 `json:"total_size"`
	TotalFileCount int   `json:"total_file_count"`

	// Raw data mode stats (actual storage used)
	RawDataSize int64 `json:"raw_data_size"`

	// Restore size mode stats (original data size before dedup)
	RestoreSize int64 `json:"restore_size"`

	// Calculated dedup metrics
	DedupRatio    float64 `json:"dedup_ratio"`
	SpaceSaved    int64   `json:"space_saved"`
	SpaceSavedPct float64 `json:"space_saved_pct"`

	// Snapshot count
	SnapshotCount int `json:"snapshot_count"`
}

// StatsWithRawData runs restic stats with --mode raw-data to get actual storage used.
func (r *Restic) StatsWithRawData(ctx context.Context, cfg ResticConfig) (*RawStatsMode, error) {
	r.logger.Debug().Msg("getting repository stats (raw-data mode)")

	args := []string{"stats", "--repo", cfg.Repository, "--mode", "raw-data", "--json"}
	output, err := r.run(ctx, cfg, args)
	if err != nil {
		return nil, fmt.Errorf("stats raw-data failed: %w", err)
	}

	var stats RawStatsMode
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("parse stats raw-data: %w", err)
	}

	return &stats, nil
}

// StatsWithRestoreSize runs restic stats with --mode restore-size to get original data size.
func (r *Restic) StatsWithRestoreSize(ctx context.Context, cfg ResticConfig) (*RawStatsMode, error) {
	r.logger.Debug().Msg("getting repository stats (restore-size mode)")

	args := []string{"stats", "--repo", cfg.Repository, "--mode", "restore-size", "--json"}
	output, err := r.run(ctx, cfg, args)
	if err != nil {
		return nil, fmt.Errorf("stats restore-size failed: %w", err)
	}

	var stats RawStatsMode
	if err := json.Unmarshal(output, &stats); err != nil {
		return nil, fmt.Errorf("parse stats restore-size: %w", err)
	}

	return &stats, nil
}

// GetExtendedStats collects comprehensive repository statistics including dedup metrics.
func (r *Restic) GetExtendedStats(ctx context.Context, cfg ResticConfig) (*ExtendedRepoStats, error) {
	r.logger.Debug().Msg("collecting extended repository stats")

	// Get raw data stats (actual storage used on disk)
	rawStats, err := r.StatsWithRawData(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("get raw-data stats: %w", err)
	}

	// Get restore size stats (original data size before dedup)
	restoreStats, err := r.StatsWithRestoreSize(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("get restore-size stats: %w", err)
	}

	// Get snapshot count
	snapshots, err := r.Snapshots(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	// Calculate dedup metrics
	rawDataSize := rawStats.TotalSize
	restoreSize := restoreStats.TotalSize

	var dedupRatio float64
	var spaceSaved int64
	var spaceSavedPct float64

	if restoreSize > 0 {
		// Dedup ratio: how much the original data was compressed/deduped
		// e.g., 3.0 means data was reduced to 1/3 of original size
		dedupRatio = float64(restoreSize) / float64(rawDataSize)
		spaceSaved = restoreSize - rawDataSize
		spaceSavedPct = (float64(spaceSaved) / float64(restoreSize)) * 100
	}

	return &ExtendedRepoStats{
		TotalSize:      restoreStats.TotalSize,
		TotalFileCount: restoreStats.TotalFileCount,
		RawDataSize:    rawDataSize,
		RestoreSize:    restoreSize,
		DedupRatio:     dedupRatio,
		SpaceSaved:     spaceSaved,
		SpaceSavedPct:  spaceSavedPct,
		SnapshotCount:  len(snapshots),
	}, nil
}

// CalculateDedupRatio calculates the deduplication ratio from raw and restore sizes.
// Returns 0 if restore size is 0 to avoid division by zero.
func CalculateDedupRatio(rawDataSize, restoreSize int64) float64 {
	if rawDataSize == 0 {
		return 0
	}
	return float64(restoreSize) / float64(rawDataSize)
}

// CalculateSpaceSaved calculates the space saved through deduplication.
func CalculateSpaceSaved(rawDataSize, restoreSize int64) int64 {
	if restoreSize <= rawDataSize {
		return 0
	}
	return restoreSize - rawDataSize
}

// CalculateSpaceSavedPercent calculates the percentage of space saved.
func CalculateSpaceSavedPercent(rawDataSize, restoreSize int64) float64 {
	if restoreSize == 0 {
		return 0
	}
	saved := restoreSize - rawDataSize
	return (float64(saved) / float64(restoreSize)) * 100
}
