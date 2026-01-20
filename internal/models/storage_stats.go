package models

import (
	"time"

	"github.com/google/uuid"
)

// StorageStats represents a daily snapshot of repository storage statistics.
type StorageStats struct {
	ID             uuid.UUID `json:"id"`
	RepositoryID   uuid.UUID `json:"repository_id"`
	TotalSize      int64     `json:"total_size"`
	TotalFileCount int       `json:"total_file_count"`
	RawDataSize    int64     `json:"raw_data_size"`
	RestoreSize    int64     `json:"restore_size"`
	DedupRatio     float64   `json:"dedup_ratio"`
	SpaceSaved     int64     `json:"space_saved"`
	SpaceSavedPct  float64   `json:"space_saved_pct"`
	SnapshotCount  int       `json:"snapshot_count"`
	CollectedAt    time.Time `json:"collected_at"`
	CreatedAt      time.Time `json:"created_at"`
}

// NewStorageStats creates a new StorageStats record.
func NewStorageStats(repositoryID uuid.UUID) *StorageStats {
	now := time.Now()
	return &StorageStats{
		ID:           uuid.New(),
		RepositoryID: repositoryID,
		CollectedAt:  now,
		CreatedAt:    now,
	}
}

// SetStats populates the storage statistics fields.
func (s *StorageStats) SetStats(totalSize int64, totalFileCount int, rawDataSize, restoreSize int64, snapshotCount int) {
	s.TotalSize = totalSize
	s.TotalFileCount = totalFileCount
	s.RawDataSize = rawDataSize
	s.RestoreSize = restoreSize
	s.SnapshotCount = snapshotCount

	// Calculate dedup metrics
	if rawDataSize > 0 {
		s.DedupRatio = float64(restoreSize) / float64(rawDataSize)
		s.SpaceSaved = restoreSize - rawDataSize
		if restoreSize > 0 {
			s.SpaceSavedPct = (float64(s.SpaceSaved) / float64(restoreSize)) * 100
		}
	}
}

// StorageStatsSummary represents aggregated storage statistics across all repositories.
type StorageStatsSummary struct {
	TotalRawSize    int64   `json:"total_raw_size"`
	TotalRestoreSize int64   `json:"total_restore_size"`
	TotalSpaceSaved int64   `json:"total_space_saved"`
	AvgDedupRatio   float64 `json:"avg_dedup_ratio"`
	RepositoryCount int     `json:"repository_count"`
	TotalSnapshots  int     `json:"total_snapshots"`
}

// StorageGrowthPoint represents a single data point for storage growth over time.
type StorageGrowthPoint struct {
	Date        time.Time `json:"date"`
	RawDataSize int64     `json:"raw_data_size"`
	RestoreSize int64     `json:"restore_size"`
}
