// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// LargeFile represents a file that exceeds the size limit.
type LargeFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"size_bytes"`
	SizeMB    int64  `json:"size_mb"`
}

// LargeFileScanner scans paths to identify files exceeding a size limit.
type LargeFileScanner struct {
	logger zerolog.Logger
}

// NewLargeFileScanner creates a new LargeFileScanner.
func NewLargeFileScanner(logger zerolog.Logger) *LargeFileScanner {
	return &LargeFileScanner{
		logger: logger.With().Str("component", "largefile_scanner").Logger(),
	}
}

// ScanResult contains the results of scanning for large files.
type ScanResult struct {
	LargeFiles    []LargeFile `json:"large_files"`
	TotalExcluded int         `json:"total_excluded"`
	TotalSizeMB   int64       `json:"total_size_mb"`
}

// Scan scans the given paths for files exceeding maxSizeMB.
// It respects the exclude patterns when scanning.
// Returns a list of large files that will be excluded by restic's --exclude-larger-than flag.
func (s *LargeFileScanner) Scan(ctx context.Context, paths []string, excludes []string, maxSizeMB int) (*ScanResult, error) {
	if maxSizeMB <= 0 {
		return &ScanResult{}, nil
	}

	maxSizeBytes := int64(maxSizeMB) * 1024 * 1024
	result := &ScanResult{
		LargeFiles: make([]LargeFile, 0),
	}

	// Build a map of exclude patterns for quick lookup
	excludeSet := make(map[string]bool)
	for _, ex := range excludes {
		excludeSet[ex] = true
	}

	for _, path := range paths {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		if err := s.scanPath(ctx, path, maxSizeBytes, excludeSet, result); err != nil {
			s.logger.Warn().Err(err).Str("path", path).Msg("error scanning path for large files")
			// Continue scanning other paths
		}
	}

	s.logger.Info().
		Int("total_excluded", result.TotalExcluded).
		Int64("total_size_mb", result.TotalSizeMB).
		Msg("large file scan completed")

	return result, nil
}

// scanPath scans a single path recursively for large files.
func (s *LargeFileScanner) scanPath(ctx context.Context, basePath string, maxSizeBytes int64, excludeSet map[string]bool, result *ScanResult) error {
	return filepath.WalkDir(basePath, func(path string, d os.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			// Log and continue on permission errors
			s.logger.Debug().Err(err).Str("path", path).Msg("error accessing path")
			return nil
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if path matches any exclude pattern (simple check)
		for pattern := range excludeSet {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				return nil
			}
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			s.logger.Debug().Err(err).Str("path", path).Msg("error getting file info")
			return nil
		}

		// Check if file exceeds size limit
		if info.Size() > maxSizeBytes {
			sizeMB := info.Size() / (1024 * 1024)
			result.LargeFiles = append(result.LargeFiles, LargeFile{
				Path:      path,
				SizeBytes: info.Size(),
				SizeMB:    sizeMB,
			})
			result.TotalExcluded++
			result.TotalSizeMB += sizeMB

			s.logger.Warn().
				Str("path", path).
				Int64("size_mb", sizeMB).
				Int64("max_size_mb", maxSizeBytes/(1024*1024)).
				Msg("file exceeds size limit and will be excluded")
		}

		return nil
	})
}

// FormatExcludedFiles formats the list of excluded files for display/logging.
// Returns a truncated list if there are too many files.
func (s *LargeFileScanner) FormatExcludedFiles(result *ScanResult, maxFiles int) []string {
	if result == nil || len(result.LargeFiles) == 0 {
		return nil
	}

	files := make([]string, 0, min(len(result.LargeFiles), maxFiles))
	for i, f := range result.LargeFiles {
		if i >= maxFiles {
			break
		}
		files = append(files, f.Path)
	}

	return files
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
