// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DiffChangeType represents the type of change in a diff.
type DiffChangeType string

const (
	DiffChangeAdded    DiffChangeType = "added"
	DiffChangeRemoved  DiffChangeType = "removed"
	DiffChangeModified DiffChangeType = "modified"
)

// DiffEntry represents a single changed file/directory in the diff.
type DiffEntry struct {
	Path         string         `json:"path"`
	ChangeType   DiffChangeType `json:"change_type"`
	Type         string         `json:"type"` // "file" or "dir"
	OldSize      int64          `json:"old_size,omitempty"`
	NewSize      int64          `json:"new_size,omitempty"`
	SizeChange   int64          `json:"size_change,omitempty"`
	OldModTime   string         `json:"old_mod_time,omitempty"`
	NewModTime   string         `json:"new_mod_time,omitempty"`
}

// DiffStats contains summary statistics for a diff operation.
type DiffStats struct {
	FilesAdded      int   `json:"files_added"`
	FilesRemoved    int   `json:"files_removed"`
	FilesModified   int   `json:"files_modified"`
	DirsAdded       int   `json:"dirs_added"`
	DirsRemoved     int   `json:"dirs_removed"`
	TotalSizeAdded  int64 `json:"total_size_added"`
	TotalSizeRemoved int64 `json:"total_size_removed"`
}

// DiffResult contains the result of comparing two snapshots.
type DiffResult struct {
	SnapshotID1 string      `json:"snapshot_id_1"`
	SnapshotID2 string      `json:"snapshot_id_2"`
	Stats       DiffStats   `json:"stats"`
	Changes     []DiffEntry `json:"changes"`
}

// resticDiffMessage represents a JSON message from restic diff --json output.
type resticDiffMessage struct {
	MessageType string `json:"message_type"`
	// For "change" messages
	SourcePath string `json:"source_path,omitempty"`
	TargetPath string `json:"target_path,omitempty"`
	Modifier   string `json:"modifier,omitempty"` // "+", "-", "M", "T", "U"
	// For "statistics" messages
	SourceSnapshot string `json:"source_snapshot,omitempty"`
	TargetSnapshot string `json:"target_snapshot,omitempty"`
	ChangedFiles   int    `json:"changed_files,omitempty"`
	Added          struct {
		Files int   `json:"files,omitempty"`
		Dirs  int   `json:"dirs,omitempty"`
		Bytes int64 `json:"bytes,omitempty"`
	} `json:"added,omitempty"`
	Removed struct {
		Files int   `json:"files,omitempty"`
		Dirs  int   `json:"dirs,omitempty"`
		Bytes int64 `json:"bytes,omitempty"`
	} `json:"removed,omitempty"`
}

// Diff compares two snapshots and returns the differences.
func (r *Restic) Diff(ctx context.Context, cfg ResticConfig, snapshotID1, snapshotID2 string) (*DiffResult, error) {
	r.logger.Info().
		Str("snapshot_id_1", snapshotID1).
		Str("snapshot_id_2", snapshotID2).
		Msg("comparing snapshots")

	args := []string{"diff", "--repo", cfg.Repository, "--json", snapshotID1, snapshotID2}

	output, err := r.run(ctx, cfg, args)
	if err != nil {
		if strings.Contains(err.Error(), "no matching ID") {
			return nil, ErrSnapshotNotFound
		}
		return nil, fmt.Errorf("diff failed: %w", err)
	}

	result, err := parseDiffOutput(output, snapshotID1, snapshotID2)
	if err != nil {
		return nil, fmt.Errorf("parse diff output: %w", err)
	}

	r.logger.Info().
		Int("files_added", result.Stats.FilesAdded).
		Int("files_removed", result.Stats.FilesRemoved).
		Int("files_modified", result.Stats.FilesModified).
		Msg("diff completed")

	return result, nil
}

// parseDiffOutput parses the JSON output from restic diff --json.
func parseDiffOutput(output []byte, snapshotID1, snapshotID2 string) (*DiffResult, error) {
	result := &DiffResult{
		SnapshotID1: snapshotID1,
		SnapshotID2: snapshotID2,
		Changes:     []DiffEntry{},
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg resticDiffMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			// Skip lines that don't parse as JSON
			continue
		}

		switch msg.MessageType {
		case "change":
			entry := parseDiffChange(msg)
			if entry != nil {
				result.Changes = append(result.Changes, *entry)
				updateStats(&result.Stats, entry)
			}
		case "statistics":
			// Update stats from the statistics message
			result.Stats.FilesAdded = msg.Added.Files
			result.Stats.FilesRemoved = msg.Removed.Files
			result.Stats.DirsAdded = msg.Added.Dirs
			result.Stats.DirsRemoved = msg.Removed.Dirs
			result.Stats.TotalSizeAdded = msg.Added.Bytes
			result.Stats.TotalSizeRemoved = msg.Removed.Bytes
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan output: %w", err)
	}

	return result, nil
}

// parseDiffChange converts a restic diff change message to a DiffEntry.
func parseDiffChange(msg resticDiffMessage) *DiffEntry {
	if msg.SourcePath == "" && msg.TargetPath == "" {
		return nil
	}

	entry := &DiffEntry{
		Type: "file", // Default to file
	}

	// Determine the path
	if msg.TargetPath != "" {
		entry.Path = msg.TargetPath
	} else {
		entry.Path = msg.SourcePath
	}

	// Determine change type based on modifier
	switch msg.Modifier {
	case "+":
		entry.ChangeType = DiffChangeAdded
	case "-":
		entry.ChangeType = DiffChangeRemoved
	case "M", "C", "T", "U":
		entry.ChangeType = DiffChangeModified
	default:
		entry.ChangeType = DiffChangeModified
	}

	return entry
}

// updateStats updates the DiffStats based on a DiffEntry.
func updateStats(stats *DiffStats, entry *DiffEntry) {
	switch entry.ChangeType {
	case DiffChangeAdded:
		if entry.Type == "dir" {
			stats.DirsAdded++
		} else {
			stats.FilesAdded++
			stats.TotalSizeAdded += entry.NewSize
		}
	case DiffChangeRemoved:
		if entry.Type == "dir" {
			stats.DirsRemoved++
		} else {
			stats.FilesRemoved++
			stats.TotalSizeRemoved += entry.OldSize
		}
	case DiffChangeModified:
		stats.FilesModified++
	}
}

// ParseDiffFromText parses restic diff output from text format (non-JSON).
// This is a fallback for when JSON output is not available.
func ParseDiffFromText(output string, snapshotID1, snapshotID2 string) (*DiffResult, error) {
	result := &DiffResult{
		SnapshotID1: snapshotID1,
		SnapshotID2: snapshotID2,
		Changes:     []DiffEntry{},
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse lines like:
		// +    /path/to/added/file
		// -    /path/to/removed/file
		// M    /path/to/modified/file
		var entry *DiffEntry
		if strings.HasPrefix(line, "+") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "+"))
			entry = &DiffEntry{
				Path:       path,
				ChangeType: DiffChangeAdded,
				Type:       inferFileType(path),
			}
		} else if strings.HasPrefix(line, "-") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "-"))
			entry = &DiffEntry{
				Path:       path,
				ChangeType: DiffChangeRemoved,
				Type:       inferFileType(path),
			}
		} else if strings.HasPrefix(line, "M") {
			path := strings.TrimSpace(strings.TrimPrefix(line, "M"))
			entry = &DiffEntry{
				Path:       path,
				ChangeType: DiffChangeModified,
				Type:       inferFileType(path),
			}
		}

		if entry != nil {
			result.Changes = append(result.Changes, *entry)
			updateStats(&result.Stats, entry)
		}
	}

	return result, nil
}

// inferFileType tries to determine if a path is a directory or file.
func inferFileType(path string) string {
	if strings.HasSuffix(path, "/") {
		return "dir"
	}
	return "file"
}

// ParseSizeFromDiffLine attempts to parse file size from a diff line.
// Returns 0 if size cannot be determined.
func ParseSizeFromDiffLine(line string) int64 {
	// Look for size patterns like "1.5 KiB", "500 B", "2.0 MiB"
	parts := strings.Fields(line)
	for i, part := range parts {
		if i+1 < len(parts) {
			unit := parts[i+1]
			if unit == "B" || unit == "KiB" || unit == "MiB" || unit == "GiB" {
				size, err := strconv.ParseFloat(part, 64)
				if err != nil {
					continue
				}
				switch unit {
				case "B":
					return int64(size)
				case "KiB":
					return int64(size * 1024)
				case "MiB":
					return int64(size * 1024 * 1024)
				case "GiB":
					return int64(size * 1024 * 1024 * 1024)
				}
			}
		}
	}
	return 0
}
