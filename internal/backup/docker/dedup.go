// Package docker provides image deduplication functionality.
package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DeduplicationStore defines the interface for deduplication persistence.
type DeduplicationStore interface {
	// GetDeduplicationEntry returns an existing deduplication entry for an image.
	GetDeduplicationEntry(ctx context.Context, orgID uuid.UUID, imageID string) (*models.DockerImageDeduplicationEntry, error)

	// GetDeduplicationEntryByChecksum returns an entry by checksum.
	GetDeduplicationEntryByChecksum(ctx context.Context, orgID uuid.UUID, checksum string) (*models.DockerImageDeduplicationEntry, error)

	// CreateDeduplicationEntry creates a new deduplication entry.
	CreateDeduplicationEntry(ctx context.Context, entry *models.DockerImageDeduplicationEntry) error

	// UpdateDeduplicationEntry updates a deduplication entry.
	UpdateDeduplicationEntry(ctx context.Context, entry *models.DockerImageDeduplicationEntry) error

	// DeleteDeduplicationEntry deletes a deduplication entry.
	DeleteDeduplicationEntry(ctx context.Context, id uuid.UUID) error

	// ListDeduplicationEntries returns all entries for an organization.
	ListDeduplicationEntries(ctx context.Context, orgID uuid.UUID) ([]*models.DockerImageDeduplicationEntry, error)

	// CleanupUnusedEntries removes entries with zero references.
	CleanupUnusedEntries(ctx context.Context, orgID uuid.UUID) (int, error)
}

// DeduplicationManager manages image deduplication across backups.
type DeduplicationManager struct {
	store  DeduplicationStore
	logger zerolog.Logger
	mu     sync.RWMutex

	// In-memory cache for fast lookups
	checksumIndex map[string]*models.DockerImageDeduplicationEntry // checksum -> entry
	imageIndex    map[string]*models.DockerImageDeduplicationEntry // imageID -> entry
}

// NewDeduplicationManager creates a new deduplication manager.
func NewDeduplicationManager(store DeduplicationStore, logger zerolog.Logger) *DeduplicationManager {
	return &DeduplicationManager{
		store:         store,
		logger:        logger.With().Str("component", "docker_dedup").Logger(),
		checksumIndex: make(map[string]*models.DockerImageDeduplicationEntry),
		imageIndex:    make(map[string]*models.DockerImageDeduplicationEntry),
	}
}

// LoadCache loads deduplication data into memory for an organization.
func (m *DeduplicationManager) LoadCache(ctx context.Context, orgID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := m.store.ListDeduplicationEntries(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}

	for _, entry := range entries {
		m.checksumIndex[entry.Checksum] = entry
		m.imageIndex[entry.ImageID] = entry
	}

	m.logger.Info().
		Int("entries_loaded", len(entries)).
		Msg("deduplication cache loaded")

	return nil
}

// CheckDuplicate checks if an image can be deduplicated.
// Returns the existing entry if duplicate, nil otherwise.
func (m *DeduplicationManager) CheckDuplicate(ctx context.Context, orgID uuid.UUID, imageID, checksum string) (*models.DockerImageDeduplicationEntry, error) {
	m.mu.RLock()

	// Check in-memory cache first
	if entry, ok := m.checksumIndex[checksum]; ok {
		m.mu.RUnlock()
		// Verify the backup file still exists
		if _, err := os.Stat(entry.OriginalPath); err == nil {
			return entry, nil
		}
		// File doesn't exist, need to clean up
		m.mu.Lock()
		delete(m.checksumIndex, checksum)
		delete(m.imageIndex, entry.ImageID)
		m.mu.Unlock()
		return nil, nil
	}
	m.mu.RUnlock()

	// Check database
	entry, err := m.store.GetDeduplicationEntryByChecksum(ctx, orgID, checksum)
	if err != nil {
		return nil, nil // Entry doesn't exist
	}

	if entry != nil {
		// Verify the backup file still exists
		if _, err := os.Stat(entry.OriginalPath); err == nil {
			// Add to cache
			m.mu.Lock()
			m.checksumIndex[checksum] = entry
			m.imageIndex[entry.ImageID] = entry
			m.mu.Unlock()
			return entry, nil
		}
		// File doesn't exist, clean up the entry
		m.store.DeleteDeduplicationEntry(ctx, entry.ID)
	}

	return nil, nil
}

// RegisterImage registers a new image for deduplication tracking.
func (m *DeduplicationManager) RegisterImage(
	ctx context.Context,
	orgID uuid.UUID,
	imageID, checksum string,
	backupID uuid.UUID,
	backupPath string,
	sizeBytes int64,
) (*models.DockerImageDeduplicationEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already exists
	if existing, ok := m.checksumIndex[checksum]; ok {
		existing.IncrementReference()
		if err := m.store.UpdateDeduplicationEntry(ctx, existing); err != nil {
			return nil, fmt.Errorf("update entry: %w", err)
		}
		return existing, nil
	}

	// Create new entry
	entry := models.NewDockerImageDeduplicationEntry(
		orgID,
		imageID,
		checksum,
		backupID,
		backupPath,
		sizeBytes,
	)

	if err := m.store.CreateDeduplicationEntry(ctx, entry); err != nil {
		return nil, fmt.Errorf("create entry: %w", err)
	}

	// Add to cache
	m.checksumIndex[checksum] = entry
	m.imageIndex[imageID] = entry

	m.logger.Debug().
		Str("image_id", imageID).
		Str("checksum", checksum).
		Msg("registered image for deduplication")

	return entry, nil
}

// IncrementReference increments the reference count for an image.
func (m *DeduplicationManager) IncrementReference(ctx context.Context, entry *models.DockerImageDeduplicationEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry.IncrementReference()
	return m.store.UpdateDeduplicationEntry(ctx, entry)
}

// DecrementReference decrements the reference count for an image.
// If the count reaches zero, the entry is marked for cleanup.
func (m *DeduplicationManager) DecrementReference(ctx context.Context, entry *models.DockerImageDeduplicationEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry.DecrementReference()
	return m.store.UpdateDeduplicationEntry(ctx, entry)
}

// CleanupUnused removes entries with zero references and their backup files.
func (m *DeduplicationManager) CleanupUnused(ctx context.Context, orgID uuid.UUID) (int, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entries, err := m.store.ListDeduplicationEntries(ctx, orgID)
	if err != nil {
		return 0, 0, fmt.Errorf("list entries: %w", err)
	}

	var removed int
	var bytesFreed int64

	for _, entry := range entries {
		if entry.ReferenceCount <= 0 {
			// Remove backup file if it exists and no other entries reference it
			if _, err := os.Stat(entry.OriginalPath); err == nil {
				if err := os.Remove(entry.OriginalPath); err == nil {
					bytesFreed += entry.SizeBytes
				}
			}

			// Remove from database
			if err := m.store.DeleteDeduplicationEntry(ctx, entry.ID); err != nil {
				m.logger.Warn().
					Err(err).
					Str("entry_id", entry.ID.String()).
					Msg("failed to delete deduplication entry")
				continue
			}

			// Remove from cache
			delete(m.checksumIndex, entry.Checksum)
			delete(m.imageIndex, entry.ImageID)

			removed++
		}
	}

	m.logger.Info().
		Int("entries_removed", removed).
		Int64("bytes_freed", bytesFreed).
		Msg("cleanup completed")

	return removed, bytesFreed, nil
}

// GetStats returns deduplication statistics for an organization.
func (m *DeduplicationManager) GetStats(ctx context.Context, orgID uuid.UUID) (*DeduplicationStats, error) {
	entries, err := m.store.ListDeduplicationEntries(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}

	stats := &DeduplicationStats{
		TotalImages:      len(entries),
		TotalReferences:  0,
		TotalSizeBytes:   0,
		PotentialSavings: 0,
	}

	for _, entry := range entries {
		stats.TotalReferences += entry.ReferenceCount
		stats.TotalSizeBytes += entry.SizeBytes
		// Savings = size * (references - 1) because the first reference is the actual file
		if entry.ReferenceCount > 1 {
			stats.PotentialSavings += entry.SizeBytes * int64(entry.ReferenceCount-1)
		}
	}

	return stats, nil
}

// DeduplicationStats contains statistics about deduplication.
type DeduplicationStats struct {
	TotalImages      int   `json:"total_images"`
	TotalReferences  int   `json:"total_references"`
	TotalSizeBytes   int64 `json:"total_size_bytes"`
	PotentialSavings int64 `json:"potential_savings"`
}

// DeduplicationReport provides a detailed report of deduplication status.
type DeduplicationReport struct {
	Stats           DeduplicationStats                      `json:"stats"`
	TopDeduplicated []DeduplicationReportEntry              `json:"top_deduplicated"`
	UnusedEntries   []*models.DockerImageDeduplicationEntry `json:"unused_entries"`
}

// DeduplicationReportEntry represents an entry in the deduplication report.
type DeduplicationReportEntry struct {
	ImageID        string `json:"image_id"`
	ReferenceCount int    `json:"reference_count"`
	SizeBytes      int64  `json:"size_bytes"`
	SpaceSaved     int64  `json:"space_saved"`
}

// GenerateReport generates a detailed deduplication report.
func (m *DeduplicationManager) GenerateReport(ctx context.Context, orgID uuid.UUID) (*DeduplicationReport, error) {
	entries, err := m.store.ListDeduplicationEntries(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}

	stats, err := m.GetStats(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	report := &DeduplicationReport{
		Stats:           *stats,
		TopDeduplicated: make([]DeduplicationReportEntry, 0),
		UnusedEntries:   make([]*models.DockerImageDeduplicationEntry, 0),
	}

	// Find top deduplicated images
	for _, entry := range entries {
		if entry.ReferenceCount > 1 {
			report.TopDeduplicated = append(report.TopDeduplicated, DeduplicationReportEntry{
				ImageID:        entry.ImageID,
				ReferenceCount: entry.ReferenceCount,
				SizeBytes:      entry.SizeBytes,
				SpaceSaved:     entry.SizeBytes * int64(entry.ReferenceCount-1),
			})
		}
		if entry.ReferenceCount == 0 {
			report.UnusedEntries = append(report.UnusedEntries, entry)
		}
	}

	// Sort by space saved (descending)
	for i := 0; i < len(report.TopDeduplicated)-1; i++ {
		for j := i + 1; j < len(report.TopDeduplicated); j++ {
			if report.TopDeduplicated[i].SpaceSaved < report.TopDeduplicated[j].SpaceSaved {
				report.TopDeduplicated[i], report.TopDeduplicated[j] = report.TopDeduplicated[j], report.TopDeduplicated[i]
			}
		}
	}

	// Limit to top 20
	if len(report.TopDeduplicated) > 20 {
		report.TopDeduplicated = report.TopDeduplicated[:20]
	}

	return report, nil
}

// ExportIndex exports the deduplication index for backup purposes.
func (m *DeduplicationManager) ExportIndex(ctx context.Context, orgID uuid.UUID, outputPath string) error {
	entries, err := m.store.ListDeduplicationEntries(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal entries: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	m.logger.Info().
		Str("output_path", outputPath).
		Int("entries", len(entries)).
		Msg("deduplication index exported")

	return nil
}

// ImportIndex imports a deduplication index from a backup.
func (m *DeduplicationManager) ImportIndex(ctx context.Context, inputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var entries []*models.DockerImageDeduplicationEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshal entries: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var imported int
	for _, entry := range entries {
		// Verify backup file exists
		if _, err := os.Stat(entry.OriginalPath); err != nil {
			m.logger.Warn().
				Str("image_id", entry.ImageID).
				Str("path", entry.OriginalPath).
				Msg("skipping entry with missing backup file")
			continue
		}

		// Check if entry already exists
		existing, err := m.store.GetDeduplicationEntry(ctx, entry.OrgID, entry.ImageID)
		if err == nil && existing != nil {
			// Update reference count
			existing.ReferenceCount += entry.ReferenceCount
			m.store.UpdateDeduplicationEntry(ctx, existing)
			m.checksumIndex[existing.Checksum] = existing
			m.imageIndex[existing.ImageID] = existing
		} else {
			// Create new entry
			if err := m.store.CreateDeduplicationEntry(ctx, entry); err != nil {
				m.logger.Warn().
					Err(err).
					Str("image_id", entry.ImageID).
					Msg("failed to import entry")
				continue
			}
			m.checksumIndex[entry.Checksum] = entry
			m.imageIndex[entry.ImageID] = entry
		}
		imported++
	}

	m.logger.Info().
		Str("input_path", inputPath).
		Int("entries_imported", imported).
		Msg("deduplication index imported")

	return nil
}
