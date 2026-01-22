// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/rs/zerolog"
)

// ImportPreview contains information about an existing repository that can be imported.
type ImportPreview struct {
	// SnapshotCount is the total number of snapshots in the repository.
	SnapshotCount int `json:"snapshot_count"`
	// Snapshots is a list of snapshots found in the repository.
	Snapshots []Snapshot `json:"snapshots"`
	// Hostnames contains unique hostnames found in snapshots.
	Hostnames []string `json:"hostnames"`
	// TotalSize is the total deduplicated size of the repository in bytes.
	TotalSize int64 `json:"total_size"`
	// TotalFileCount is the total number of files across all snapshots.
	TotalFileCount int `json:"total_file_count"`
}

// ImportOptions configures which snapshots to import.
type ImportOptions struct {
	// SnapshotIDs specifies which snapshots to import (empty = all).
	SnapshotIDs []string `json:"snapshot_ids,omitempty"`
	// Hostnames filters snapshots by hostname (empty = all).
	Hostnames []string `json:"hostnames,omitempty"`
	// AgentID is the agent to associate imported snapshots with.
	// If empty, snapshots will be imported without an agent association.
	AgentID string `json:"agent_id,omitempty"`
}

// ImportResult contains the results of a repository import operation.
type ImportResult struct {
	// SnapshotsImported is the number of snapshots imported.
	SnapshotsImported int `json:"snapshots_imported"`
	// Snapshots contains details of the imported snapshots.
	Snapshots []Snapshot `json:"snapshots"`
}

// Importer handles importing existing Restic repositories.
type Importer struct {
	restic *Restic
	logger zerolog.Logger
}

// NewImporter creates a new Importer.
func NewImporter(logger zerolog.Logger) *Importer {
	return &Importer{
		restic: NewRestic(logger),
		logger: logger.With().Str("component", "importer").Logger(),
	}
}

// VerifyAccess verifies that the repository can be accessed with the given credentials.
func (i *Importer) VerifyAccess(ctx context.Context, backend backends.Backend, password string) error {
	i.logger.Info().Str("type", string(backend.Type())).Msg("verifying repository access")

	cfg := backend.ToResticConfig(password)

	// Try to list snapshots to verify access
	_, err := i.restic.Snapshots(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to access repository: %w", err)
	}

	i.logger.Info().Msg("repository access verified")
	return nil
}

// Preview retrieves information about an existing repository without modifying it.
func (i *Importer) Preview(ctx context.Context, backend backends.Backend, password string) (*ImportPreview, error) {
	i.logger.Info().Str("type", string(backend.Type())).Msg("previewing repository for import")

	cfg := backend.ToResticConfig(password)

	// Get all snapshots
	snapshots, err := i.restic.Snapshots(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Extract unique hostnames
	hostnameSet := make(map[string]struct{})
	for _, snap := range snapshots {
		hostnameSet[snap.Hostname] = struct{}{}
	}

	hostnames := make([]string, 0, len(hostnameSet))
	for hostname := range hostnameSet {
		hostnames = append(hostnames, hostname)
	}

	// Get repository stats
	stats, err := i.restic.Stats(ctx, cfg)
	if err != nil {
		// Stats might fail on some backends, continue without them
		i.logger.Warn().Err(err).Msg("failed to get repository stats, continuing without size info")
		stats = &RepoStats{}
	}

	preview := &ImportPreview{
		SnapshotCount:  len(snapshots),
		Snapshots:      snapshots,
		Hostnames:      hostnames,
		TotalSize:      stats.TotalSize,
		TotalFileCount: stats.TotalFileCount,
	}

	i.logger.Info().
		Int("snapshot_count", preview.SnapshotCount).
		Int("hostname_count", len(hostnames)).
		Int64("total_size", preview.TotalSize).
		Msg("repository preview completed")

	return preview, nil
}

// FilterSnapshots filters snapshots based on import options.
func (i *Importer) FilterSnapshots(snapshots []Snapshot, opts ImportOptions) []Snapshot {
	if len(opts.SnapshotIDs) == 0 && len(opts.Hostnames) == 0 {
		return snapshots
	}

	// Build lookup maps for efficient filtering
	snapshotIDSet := make(map[string]struct{})
	for _, id := range opts.SnapshotIDs {
		snapshotIDSet[id] = struct{}{}
	}

	hostnameSet := make(map[string]struct{})
	for _, hostname := range opts.Hostnames {
		hostnameSet[hostname] = struct{}{}
	}

	var filtered []Snapshot
	for _, snap := range snapshots {
		// If snapshot IDs specified, check if this snapshot matches
		if len(opts.SnapshotIDs) > 0 {
			if _, ok := snapshotIDSet[snap.ID]; !ok {
				if _, ok := snapshotIDSet[snap.ShortID]; !ok {
					continue
				}
			}
		}

		// If hostnames specified, check if this snapshot's hostname matches
		if len(opts.Hostnames) > 0 {
			if _, ok := hostnameSet[snap.Hostname]; !ok {
				continue
			}
		}

		filtered = append(filtered, snap)
	}

	return filtered
}

// GetSnapshots retrieves snapshots from the repository, optionally filtered by options.
func (i *Importer) GetSnapshots(ctx context.Context, backend backends.Backend, password string, opts ImportOptions) ([]Snapshot, error) {
	i.logger.Info().
		Str("type", string(backend.Type())).
		Strs("hostnames", opts.Hostnames).
		Strs("snapshot_ids", opts.SnapshotIDs).
		Msg("getting snapshots for import")

	cfg := backend.ToResticConfig(password)

	snapshots, err := i.restic.Snapshots(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	filtered := i.FilterSnapshots(snapshots, opts)

	i.logger.Info().
		Int("total_snapshots", len(snapshots)).
		Int("filtered_snapshots", len(filtered)).
		Msg("snapshots retrieved for import")

	return filtered, nil
}

// CheckRepository verifies the integrity of an existing repository.
func (i *Importer) CheckRepository(ctx context.Context, backend backends.Backend, password string) error {
	i.logger.Info().Str("type", string(backend.Type())).Msg("checking repository integrity")

	cfg := backend.ToResticConfig(password)

	if err := i.restic.Check(ctx, cfg); err != nil {
		return fmt.Errorf("repository check failed: %w", err)
	}

	i.logger.Info().Msg("repository integrity check passed")
	return nil
}
