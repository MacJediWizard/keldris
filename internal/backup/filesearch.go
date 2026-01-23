// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FileSearchResult represents a file found in a snapshot.
type FileSearchResult struct {
	SnapshotID   string    `json:"snapshot_id"`
	SnapshotTime time.Time `json:"snapshot_time"`
	Hostname     string    `json:"hostname"`
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"file_path"`
	FileSize     int64     `json:"file_size"`
	FileType     string    `json:"file_type"` // "file" or "dir"
	ModTime      time.Time `json:"mod_time"`
}

// FileSearchFilter contains filter options for file search.
type FileSearchFilter struct {
	Query       string     // Filename pattern to search for
	PathPrefix  string     // Optional path prefix to filter results
	SnapshotIDs []string   // Optional list of snapshot IDs to search in
	DateFrom    *time.Time // Optional date range start
	DateTo      *time.Time // Optional date range end
	SizeMin     *int64     // Optional minimum file size
	SizeMax     *int64     // Optional maximum file size
	Limit       int        // Maximum number of results (0 = unlimited)
}

// FileSearchResponse contains grouped search results.
type FileSearchResponse struct {
	Query      string              `json:"query"`
	TotalCount int                 `json:"total_count"`
	Snapshots  []SnapshotFileGroup `json:"snapshots"`
}

// SnapshotFileGroup groups files by snapshot.
type SnapshotFileGroup struct {
	SnapshotID   string             `json:"snapshot_id"`
	SnapshotTime time.Time          `json:"snapshot_time"`
	Hostname     string             `json:"hostname"`
	FileCount    int                `json:"file_count"`
	Files        []FileSearchResult `json:"files"`
}

// resticFindMatch represents a single match from restic find --json output.
type resticFindMatch struct {
	Matches    []resticFindEntry `json:"matches"`
	Snapshot   string            `json:"snapshot"`
	Hits       int               `json:"hits"`
	SnapshotID string            // populated from Snapshot for convenience
}

// resticFindEntry represents a file entry in restic find output.
type resticFindEntry struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	Mode       uint32    `json:"mode"`
	ModTime    time.Time `json:"mtime"`
	AccessTime time.Time `json:"atime"`
	ChangeTime time.Time `json:"ctime"`
}

// SearchFiles searches for files matching a pattern across all snapshots using restic find.
func (r *Restic) SearchFiles(ctx context.Context, cfg ResticConfig, filter FileSearchFilter) (*FileSearchResponse, error) {
	r.logger.Debug().
		Str("query", filter.Query).
		Str("path_prefix", filter.PathPrefix).
		Int("limit", filter.Limit).
		Msg("searching files across snapshots")

	if filter.Query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	// Build restic find command
	args := []string{"find", "--repo", cfg.Repository, "--json"}

	// Add snapshot filter if specified
	for _, snapshotID := range filter.SnapshotIDs {
		args = append(args, "--snapshot", snapshotID)
	}

	// Add path filter if specified
	if filter.PathPrefix != "" {
		args = append(args, "--path", filter.PathPrefix)
	}

	// Add the search pattern - use glob pattern for partial matching
	pattern := filter.Query
	if !strings.Contains(pattern, "*") {
		// If no wildcard, wrap in wildcards for partial matching
		pattern = "*" + pattern + "*"
	}
	args = append(args, pattern)

	output, err := r.run(ctx, cfg, args)
	if err != nil {
		// Check for common errors
		if strings.Contains(err.Error(), "repository does not exist") {
			return nil, ErrRepositoryNotInitialized
		}
		return nil, fmt.Errorf("find files: %w", err)
	}

	// Parse the output
	results, err := parseResticFindOutput(output)
	if err != nil {
		return nil, fmt.Errorf("parse find output: %w", err)
	}

	// Get snapshot details for time information
	snapshots, err := r.Snapshots(ctx, cfg)
	if err != nil {
		r.logger.Warn().Err(err).Msg("failed to get snapshot details for file search")
		// Continue without snapshot details
	}

	// Build snapshot lookup map
	snapshotMap := make(map[string]Snapshot)
	for _, snap := range snapshots {
		snapshotMap[snap.ID] = snap
		// Also map by short ID
		if len(snap.ID) >= 8 {
			snapshotMap[snap.ID[:8]] = snap
		}
	}

	// Group results by snapshot and apply filters
	response := &FileSearchResponse{
		Query:     filter.Query,
		Snapshots: make([]SnapshotFileGroup, 0),
	}

	totalCount := 0
	for _, match := range results {
		// Apply date filter if specified
		snap, hasSnap := snapshotMap[match.Snapshot]
		if hasSnap {
			if filter.DateFrom != nil && snap.Time.Before(*filter.DateFrom) {
				continue
			}
			if filter.DateTo != nil && snap.Time.After(*filter.DateTo) {
				continue
			}
		}

		group := SnapshotFileGroup{
			SnapshotID: match.Snapshot,
			Files:      make([]FileSearchResult, 0),
		}

		if hasSnap {
			group.SnapshotTime = snap.Time
			group.Hostname = snap.Hostname
		}

		for _, entry := range match.Matches {
			// Apply size filters
			if filter.SizeMin != nil && entry.Size < *filter.SizeMin {
				continue
			}
			if filter.SizeMax != nil && entry.Size > *filter.SizeMax {
				continue
			}

			// Check limit
			if filter.Limit > 0 && totalCount >= filter.Limit {
				break
			}

			result := FileSearchResult{
				SnapshotID:   match.Snapshot,
				SnapshotTime: group.SnapshotTime,
				Hostname:     group.Hostname,
				FileName:     entry.Name,
				FilePath:     entry.Path,
				FileSize:     entry.Size,
				FileType:     entry.Type,
				ModTime:      entry.ModTime,
			}

			group.Files = append(group.Files, result)
			totalCount++
		}

		if len(group.Files) > 0 {
			group.FileCount = len(group.Files)
			response.Snapshots = append(response.Snapshots, group)
		}

		// Check if we've hit the limit
		if filter.Limit > 0 && totalCount >= filter.Limit {
			break
		}
	}

	response.TotalCount = totalCount

	r.logger.Debug().
		Int("total_count", totalCount).
		Int("snapshot_count", len(response.Snapshots)).
		Msg("file search completed")

	return response, nil
}

// parseResticFindOutput parses the JSON output from restic find.
func parseResticFindOutput(output []byte) ([]resticFindMatch, error) {
	var results []resticFindMatch

	// Restic find outputs one JSON object per line
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var match resticFindMatch
		if err := json.Unmarshal(line, &match); err != nil {
			// Skip lines that don't parse (could be status messages)
			continue
		}

		if match.Hits > 0 {
			results = append(results, match)
		}
	}

	return results, nil
}
