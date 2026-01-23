// Package backup provides Restic backup functionality and scheduling.
package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/pmezard/go-difflib/difflib"
)

// FileDiffResult represents the result of comparing a file between two snapshots.
type FileDiffResult struct {
	Path        string `json:"path"`
	IsBinary    bool   `json:"is_binary"`
	ChangeType  string `json:"change_type"` // "modified", "added", "removed"
	OldSize     int64  `json:"old_size,omitempty"`
	NewSize     int64  `json:"new_size,omitempty"`
	OldHash     string `json:"old_hash,omitempty"`
	NewHash     string `json:"new_hash,omitempty"`
	UnifiedDiff string `json:"unified_diff,omitempty"` // For text files
	OldContent  string `json:"old_content,omitempty"`  // For side-by-side view
	NewContent  string `json:"new_content,omitempty"`  // For side-by-side view
}

// MaxFileSizeForDiff is the maximum file size (in bytes) we'll attempt to diff.
// Files larger than this will only show metadata comparison.
const MaxFileSizeForDiff = 10 * 1024 * 1024 // 10 MB

// DiffFile extracts a file from two snapshots and generates a diff.
func (r *Restic) DiffFile(ctx context.Context, cfg ResticConfig, snapshotID1, snapshotID2, filePath string) (*FileDiffResult, error) {
	r.logger.Info().
		Str("snapshot_id_1", snapshotID1).
		Str("snapshot_id_2", snapshotID2).
		Str("file_path", filePath).
		Msg("generating file diff")

	// Create temp directory for extracted files
	tempDir, err := os.MkdirTemp("", "restic-diff-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Extract file from both snapshots
	file1Path := filepath.Join(tempDir, "snapshot1", filepath.Base(filePath))
	file2Path := filepath.Join(tempDir, "snapshot2", filepath.Base(filePath))

	err1 := r.extractFile(ctx, cfg, snapshotID1, filePath, filepath.Join(tempDir, "snapshot1"))
	err2 := r.extractFile(ctx, cfg, snapshotID2, filePath, filepath.Join(tempDir, "snapshot2"))

	result := &FileDiffResult{
		Path: filePath,
	}

	// Determine change type based on extraction results
	if err1 != nil && err2 == nil {
		result.ChangeType = "added"
	} else if err1 == nil && err2 != nil {
		result.ChangeType = "removed"
	} else if err1 != nil && err2 != nil {
		return nil, fmt.Errorf("file not found in either snapshot: %s", filePath)
	} else {
		result.ChangeType = "modified"
	}

	// Get file info and content
	if err1 == nil {
		info, err := os.Stat(file1Path)
		if err == nil {
			result.OldSize = info.Size()
		}
		hash, err := hashFile(file1Path)
		if err == nil {
			result.OldHash = hash
		}
	}

	if err2 == nil {
		info, err := os.Stat(file2Path)
		if err == nil {
			result.NewSize = info.Size()
		}
		hash, err := hashFile(file2Path)
		if err == nil {
			result.NewHash = hash
		}
	}

	// Check if files are identical (for modified case)
	if result.ChangeType == "modified" && result.OldHash == result.NewHash {
		// Files are identical, no diff needed
		return result, nil
	}

	// Determine if files are binary
	if err1 == nil && isBinaryFile(file1Path) {
		result.IsBinary = true
	}
	if err2 == nil && isBinaryFile(file2Path) {
		result.IsBinary = true
	}

	// Skip diff for binary files
	if result.IsBinary {
		r.logger.Debug().Str("file_path", filePath).Msg("skipping diff for binary file")
		return result, nil
	}

	// Skip diff for files that are too large
	if result.OldSize > MaxFileSizeForDiff || result.NewSize > MaxFileSizeForDiff {
		r.logger.Debug().
			Str("file_path", filePath).
			Int64("old_size", result.OldSize).
			Int64("new_size", result.NewSize).
			Msg("skipping diff for large file")
		return result, nil
	}

	// Read file contents for text diff
	var oldContent, newContent string
	if err1 == nil {
		content, err := os.ReadFile(file1Path)
		if err == nil {
			oldContent = string(content)
			result.OldContent = oldContent
		}
	}
	if err2 == nil {
		content, err := os.ReadFile(file2Path)
		if err == nil {
			newContent = string(content)
			result.NewContent = newContent
		}
	}

	// Generate unified diff
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(oldContent),
		B:        difflib.SplitLines(newContent),
		FromFile: fmt.Sprintf("a/%s", filePath),
		ToFile:   fmt.Sprintf("b/%s", filePath),
		Context:  3,
	}
	unifiedDiff, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		r.logger.Warn().Err(err).Msg("failed to generate unified diff")
	} else {
		result.UnifiedDiff = unifiedDiff
	}

	r.logger.Info().
		Str("file_path", filePath).
		Str("change_type", result.ChangeType).
		Bool("is_binary", result.IsBinary).
		Msg("file diff generated")

	return result, nil
}

// extractFile extracts a specific file from a snapshot to a target directory.
func (r *Restic) extractFile(ctx context.Context, cfg ResticConfig, snapshotID, filePath, targetDir string) error {
	r.logger.Debug().
		Str("snapshot_id", snapshotID).
		Str("file_path", filePath).
		Str("target_dir", targetDir).
		Msg("extracting file from snapshot")

	// Create target directory
	if err := os.MkdirAll(targetDir, 0750); err != nil {
		return fmt.Errorf("create target dir: %w", err)
	}

	// Use restic dump to extract the file content
	args := []string{"dump", "--repo", cfg.Repository, snapshotID, filePath}

	cmd := exec.CommandContext(ctx, r.binary, args...)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("RESTIC_PASSWORD=%s", cfg.Password))
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "no matching ID") {
			return ErrSnapshotNotFound
		}
		if strings.Contains(errMsg, "path not found") || strings.Contains(errMsg, "no such file") {
			return fmt.Errorf("file not found: %s", filePath)
		}
		return fmt.Errorf("dump failed: %w: %s", err, errMsg)
	}

	// Write content to target file
	targetPath := filepath.Join(targetDir, filepath.Base(filePath))
	if err := os.WriteFile(targetPath, stdout.Bytes(), 0600); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// hashFile computes the SHA-256 hash of a file.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// isBinaryFile checks if a file appears to be binary.
func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// Read first 8KB to check for binary content
	buf := make([]byte, 8192)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}

	// Check if content is valid UTF-8 and doesn't contain null bytes
	if !utf8.Valid(buf[:n]) {
		return true
	}

	// Check for null bytes (common in binary files)
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}

	return false
}

// GetFileFromSnapshot extracts a file from a snapshot and returns its content.
func (r *Restic) GetFileFromSnapshot(ctx context.Context, cfg ResticConfig, snapshotID, filePath string) ([]byte, error) {
	r.logger.Debug().
		Str("snapshot_id", snapshotID).
		Str("file_path", filePath).
		Msg("getting file from snapshot")

	args := []string{"dump", "--repo", cfg.Repository, snapshotID, filePath}

	cmd := exec.CommandContext(ctx, r.binary, args...)
	cmd.Env = append(cmd.Environ(), fmt.Sprintf("RESTIC_PASSWORD=%s", cfg.Password))
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "no matching ID") {
			return nil, ErrSnapshotNotFound
		}
		if strings.Contains(errMsg, "path not found") || strings.Contains(errMsg, "no such file") {
			return nil, fmt.Errorf("file not found: %s", filePath)
		}
		return nil, fmt.Errorf("dump failed: %w: %s", err, errMsg)
	}

	return stdout.Bytes(), nil
}
