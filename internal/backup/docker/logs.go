// Package docker provides Docker container log backup and restoration functionality.
package docker

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// LogBackupConfig holds configuration for the log backup service.
type LogBackupConfig struct {
	// BackupDir is the base directory for storing log backups.
	BackupDir string
	// TempDir is the directory for temporary files during backup.
	TempDir string
}

// DefaultLogBackupConfig returns sensible defaults.
func DefaultLogBackupConfig() LogBackupConfig {
	return LogBackupConfig{
		BackupDir: "/var/lib/keldris/docker-logs",
		TempDir:   "/tmp/keldris-docker-logs",
	}
}

// LogBackupService handles Docker container log backup operations.
type LogBackupService struct {
	config LogBackupConfig
	logger zerolog.Logger
}

// NewLogBackupService creates a new LogBackupService.
func NewLogBackupService(config LogBackupConfig, logger zerolog.Logger) *LogBackupService {
	return &LogBackupService{
		config: config,
		logger: logger.With().Str("component", "docker_log_backup").Logger(),
	}
}

// RawLogLine represents a raw Docker log line with metadata.
type RawLogLine struct {
	Timestamp time.Time `json:"timestamp"`
	Stream    string    `json:"stream"` // stdout or stderr
	Message   string    `json:"message"`
}

// BackupContainerLogs backs up logs from a container log reader.
// The reader should provide Docker JSON log format lines.
func (s *LogBackupService) BackupContainerLogs(
	ctx context.Context,
	backup *models.DockerLogBackup,
	reader io.Reader,
	policy models.DockerLogRetentionPolicy,
) error {
	backup.MarkRunning()

	// Create backup directory structure: {base}/{agent_id}/{container_id}/{date}/
	backupDir := filepath.Join(
		s.config.BackupDir,
		backup.AgentID.String(),
		backup.ContainerID,
		time.Now().Format("2006-01-02"),
	)

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		backup.Fail(fmt.Sprintf("failed to create backup directory: %v", err))
		return fmt.Errorf("create backup directory: %w", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("150405") // HHMMSS
	var filename string
	if policy.CompressEnabled {
		filename = fmt.Sprintf("%s_%s.log.gz", backup.ContainerName, timestamp)
	} else {
		filename = fmt.Sprintf("%s_%s.log", backup.ContainerName, timestamp)
	}
	backupPath := filepath.Join(backupDir, filename)

	// Create output file
	outFile, err := os.Create(backupPath)
	if err != nil {
		backup.Fail(fmt.Sprintf("failed to create backup file: %v", err))
		return fmt.Errorf("create backup file: %w", err)
	}

	var writer io.WriteCloser = outFile
	var gzWriter *gzip.Writer

	if policy.CompressEnabled {
		gzWriter, err = gzip.NewWriterLevel(outFile, policy.CompressLevel)
		if err != nil {
			outFile.Close()
			backup.Fail(fmt.Sprintf("failed to create gzip writer: %v", err))
			return fmt.Errorf("create gzip writer: %w", err)
		}
		writer = gzWriter
	}

	// Process logs
	var lineCount int64
	var originalSize int64
	var startTime, endTime time.Time
	scanner := bufio.NewScanner(reader)

	// Increase scanner buffer for long lines
	const maxScanTokenSize = 1024 * 1024 // 1MB
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			writer.Close()
			if gzWriter != nil {
				gzWriter.Close()
			}
			outFile.Close()
			os.Remove(backupPath)
			backup.Fail("backup cancelled")
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		originalSize += int64(len(line)) + 1 // +1 for newline

		// Parse Docker JSON log format
		var logLine RawLogLine
		if err := json.Unmarshal(line, &logLine); err != nil {
			// If not JSON, treat as raw log line
			logLine = RawLogLine{
				Timestamp: time.Now(),
				Stream:    "stdout",
				Message:   string(line),
			}
		}

		// Track time range
		if lineCount == 0 || logLine.Timestamp.Before(startTime) {
			startTime = logLine.Timestamp
		}
		if logLine.Timestamp.After(endTime) {
			endTime = logLine.Timestamp
		}

		// Write to backup file in JSON Lines format
		backupLine, err := json.Marshal(logLine)
		if err != nil {
			s.logger.Warn().Err(err).Msg("failed to marshal log line")
			continue
		}

		if _, err := writer.Write(append(backupLine, '\n')); err != nil {
			writer.Close()
			if gzWriter != nil {
				gzWriter.Close()
			}
			outFile.Close()
			backup.Fail(fmt.Sprintf("failed to write log line: %v", err))
			return fmt.Errorf("write log line: %w", err)
		}

		lineCount++

		// Check size limit
		if policy.MaxSizeBytes > 0 && originalSize > policy.MaxSizeBytes {
			s.logger.Info().
				Int64("size", originalSize).
				Int64("max_size", policy.MaxSizeBytes).
				Msg("reached max size limit, stopping backup")
			break
		}
	}

	if err := scanner.Err(); err != nil {
		writer.Close()
		if gzWriter != nil {
			gzWriter.Close()
		}
		outFile.Close()
		backup.Fail(fmt.Sprintf("failed to read logs: %v", err))
		return fmt.Errorf("read logs: %w", err)
	}

	// Close writers
	if gzWriter != nil {
		if err := gzWriter.Close(); err != nil {
			outFile.Close()
			backup.Fail(fmt.Sprintf("failed to close gzip writer: %v", err))
			return fmt.Errorf("close gzip writer: %w", err)
		}
	}

	if err := outFile.Close(); err != nil {
		backup.Fail(fmt.Sprintf("failed to close backup file: %v", err))
		return fmt.Errorf("close backup file: %w", err)
	}

	// Get compressed size
	var compressedSize int64
	if fileInfo, err := os.Stat(backupPath); err == nil {
		compressedSize = fileInfo.Size()
	}

	if !policy.CompressEnabled {
		compressedSize = 0
	}

	backup.Complete(backupPath, originalSize, compressedSize, lineCount, startTime, endTime, policy.CompressEnabled)

	s.logger.Info().
		Str("container_id", backup.ContainerID).
		Str("container_name", backup.ContainerName).
		Int64("line_count", lineCount).
		Int64("original_size", originalSize).
		Int64("compressed_size", compressedSize).
		Float64("compression_ratio", backup.CompressionRatio()).
		Str("path", backupPath).
		Msg("container logs backed up successfully")

	return nil
}

// RestoreContainerLogs reads a backed up log file and returns its contents.
func (s *LogBackupService) RestoreContainerLogs(
	ctx context.Context,
	backup *models.DockerLogBackup,
	offset, limit int64,
) (*models.DockerLogViewResponse, error) {
	if backup.LogPath == "" {
		return nil, fmt.Errorf("backup has no log path")
	}

	file, err := os.Open(backup.LogPath)
	if err != nil {
		return nil, fmt.Errorf("open backup file: %w", err)
	}
	defer file.Close()

	var reader io.Reader = file

	// Handle compressed files
	if backup.Compressed || strings.HasSuffix(backup.LogPath, ".gz") {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("create gzip reader: %w", err)
		}
		defer gzReader.Close()
		reader = gzReader
	}

	response := &models.DockerLogViewResponse{
		BackupID:      backup.ID,
		ContainerID:   backup.ContainerID,
		ContainerName: backup.ContainerName,
		TotalLines:    backup.LineCount,
		Offset:        offset,
		Limit:         limit,
		StartTime:     backup.StartTime,
		EndTime:       backup.EndTime,
	}

	scanner := bufio.NewScanner(reader)
	const maxScanTokenSize = 1024 * 1024
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	var lineNum int64
	var entries []models.DockerLogEntry

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		lineNum++

		// Skip lines before offset
		if lineNum <= offset {
			continue
		}

		// Stop if we've collected enough lines
		if limit > 0 && int64(len(entries)) >= limit {
			break
		}

		var rawLine RawLogLine
		if err := json.Unmarshal(scanner.Bytes(), &rawLine); err != nil {
			// If not JSON, treat as raw text
			entries = append(entries, models.DockerLogEntry{
				Timestamp: time.Time{},
				Stream:    "unknown",
				Message:   scanner.Text(),
				LineNum:   lineNum,
			})
			continue
		}

		entries = append(entries, models.DockerLogEntry{
			Timestamp: rawLine.Timestamp,
			Stream:    rawLine.Stream,
			Message:   rawLine.Message,
			LineNum:   lineNum,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read backup file: %w", err)
	}

	response.Entries = entries
	return response, nil
}

// ApplyRetentionPolicy removes old backups based on the retention policy.
func (s *LogBackupService) ApplyRetentionPolicy(
	ctx context.Context,
	agentID uuid.UUID,
	containerID string,
	policy models.DockerLogRetentionPolicy,
) (int, int64, error) {
	containerDir := filepath.Join(s.config.BackupDir, agentID.String(), containerID)

	if _, err := os.Stat(containerDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	cutoffDate := time.Now().AddDate(0, 0, -policy.MaxAgeDays)
	var removedCount int
	var removedBytes int64

	err := filepath.Walk(containerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Skip directories
		if info.IsDir() {
			// Check if date directory is old enough to remove
			if path != containerDir {
				dirName := filepath.Base(path)
				if dirDate, err := time.Parse("2006-01-02", dirName); err == nil {
					if dirDate.Before(cutoffDate) {
						s.logger.Info().
							Str("path", path).
							Time("date", dirDate).
							Msg("removing old backup directory")
						// Remove entire directory
						dirSize, _ := s.dirSize(path)
						if err := os.RemoveAll(path); err != nil {
							s.logger.Warn().Err(err).Str("path", path).Msg("failed to remove old directory")
						} else {
							removedBytes += dirSize
							removedCount++
						}
						return filepath.SkipDir
					}
				}
			}
			return nil
		}

		// Check file age
		if info.ModTime().Before(cutoffDate) {
			s.logger.Debug().
				Str("path", path).
				Time("mod_time", info.ModTime()).
				Msg("removing old backup file")
			removedBytes += info.Size()
			if err := os.Remove(path); err != nil {
				s.logger.Warn().Err(err).Str("path", path).Msg("failed to remove old backup")
			} else {
				removedCount++
			}
		}

		return nil
	})

	if err != nil {
		return removedCount, removedBytes, fmt.Errorf("walk backup directory: %w", err)
	}

	s.logger.Info().
		Str("agent_id", agentID.String()).
		Str("container_id", containerID).
		Int("removed_count", removedCount).
		Int64("removed_bytes", removedBytes).
		Msg("retention policy applied")

	return removedCount, removedBytes, nil
}

// dirSize calculates the total size of a directory.
func (s *LogBackupService) dirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// ListBackups returns information about all backups for a container.
func (s *LogBackupService) ListBackups(
	ctx context.Context,
	agentID uuid.UUID,
	containerID string,
) ([]BackupFileInfo, error) {
	containerDir := filepath.Join(s.config.BackupDir, agentID.String(), containerID)

	if _, err := os.Stat(containerDir); os.IsNotExist(err) {
		return nil, nil
	}

	var backups []BackupFileInfo

	err := filepath.Walk(containerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		// Only process log files
		if !strings.HasSuffix(info.Name(), ".log") && !strings.HasSuffix(info.Name(), ".log.gz") {
			return nil
		}

		backups = append(backups, BackupFileInfo{
			Path:       path,
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			Compressed: strings.HasSuffix(info.Name(), ".gz"),
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}

	return backups, nil
}

// BackupFileInfo contains information about a backup file on disk.
type BackupFileInfo struct {
	Path       string    `json:"path"`
	Size       int64     `json:"size"`
	ModTime    time.Time `json:"mod_time"`
	Compressed bool      `json:"compressed"`
}

// DeleteBackup deletes a specific backup file.
func (s *LogBackupService) DeleteBackup(ctx context.Context, backup *models.DockerLogBackup) error {
	if backup.LogPath == "" {
		return nil
	}

	if _, err := os.Stat(backup.LogPath); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(backup.LogPath); err != nil {
		return fmt.Errorf("delete backup file: %w", err)
	}

	s.logger.Info().
		Str("backup_id", backup.ID.String()).
		Str("path", backup.LogPath).
		Msg("backup file deleted")

	return nil
}

// GetStorageStats returns storage statistics for an agent's docker log backups.
func (s *LogBackupService) GetStorageStats(ctx context.Context, agentID uuid.UUID) (*StorageStats, error) {
	agentDir := filepath.Join(s.config.BackupDir, agentID.String())

	if _, err := os.Stat(agentDir); os.IsNotExist(err) {
		return &StorageStats{}, nil
	}

	stats := &StorageStats{
		ContainerStats: make(map[string]*ContainerStorageStats),
	}

	err := filepath.Walk(agentDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if info.IsDir() {
			return nil
		}

		// Extract container ID from path
		relPath, _ := filepath.Rel(agentDir, path)
		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) < 1 {
			return nil
		}
		containerID := parts[0]

		// Update stats
		stats.TotalSize += info.Size()
		stats.TotalFiles++

		if _, ok := stats.ContainerStats[containerID]; !ok {
			stats.ContainerStats[containerID] = &ContainerStorageStats{}
		}
		stats.ContainerStats[containerID].Size += info.Size()
		stats.ContainerStats[containerID].Files++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("calculate storage stats: %w", err)
	}

	return stats, nil
}

// StorageStats represents storage usage statistics.
type StorageStats struct {
	TotalSize      int64                            `json:"total_size"`
	TotalFiles     int                              `json:"total_files"`
	ContainerStats map[string]*ContainerStorageStats `json:"container_stats"`
}

// ContainerStorageStats represents storage stats for a single container.
type ContainerStorageStats struct {
	Size  int64 `json:"size"`
	Files int   `json:"files"`
}
