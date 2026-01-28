package docker

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestDefaultImageBackupConfig(t *testing.T) {
	config := DefaultImageBackupConfig()

	if config.BackupDir == "" {
		t.Error("BackupDir should not be empty")
	}

	if config.MaxConcurrent <= 0 {
		t.Error("MaxConcurrent should be positive")
	}

	if config.CompressionLevel < 0 || config.CompressionLevel > 9 {
		t.Error("CompressionLevel should be between 0 and 9")
	}

	if len(config.PublicRegistries) == 0 {
		t.Error("PublicRegistries should contain default registries")
	}
}

func TestNewImageBackupService(t *testing.T) {
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	config := DefaultImageBackupConfig()

	service := NewImageBackupService(config, logger)

	if service == nil {
		t.Fatal("service should not be nil")
	}

	if service.checksumCache == nil {
		t.Error("checksumCache should be initialized")
	}

	if service.backupPathCache == nil {
		t.Error("backupPathCache should be initialized")
	}
}

func TestIsPublicImage(t *testing.T) {
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	config := DefaultImageBackupConfig()
	service := NewImageBackupService(config, logger)

	tests := []struct {
		name     string
		image    ImageInfo
		expected bool
	}{
		{
			name: "docker hub image without registry",
			image: ImageInfo{
				RepoTags: []string{"nginx:latest"},
			},
			expected: true,
		},
		{
			name: "docker.io prefixed image",
			image: ImageInfo{
				RepoTags: []string{"docker.io/library/nginx:latest"},
			},
			expected: true,
		},
		{
			name: "gcr.io image",
			image: ImageInfo{
				RepoTags: []string{"gcr.io/myproject/myimage:v1"},
			},
			expected: true,
		},
		{
			name: "private registry image",
			image: ImageInfo{
				RepoTags: []string{"myregistry.example.com/myimage:v1"},
			},
			expected: false,
		},
		{
			name: "localhost registry",
			image: ImageInfo{
				RepoTags: []string{"localhost:5000/myimage:v1"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsPublicImage(tt.image)
			if result != tt.expected {
				t.Errorf("IsPublicImage() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	config := DefaultImageBackupConfig()
	service := NewImageBackupService(config, logger)

	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content for checksum")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	checksum := service.calculateFileChecksum(tmpFile)
	if checksum == "" {
		t.Error("checksum should not be empty")
	}

	// Same content should produce same checksum
	tmpFile2 := filepath.Join(tmpDir, "test2.txt")
	if err := os.WriteFile(tmpFile2, content, 0644); err != nil {
		t.Fatalf("failed to create temp file 2: %v", err)
	}

	checksum2 := service.calculateFileChecksum(tmpFile2)
	if checksum != checksum2 {
		t.Error("same content should produce same checksum")
	}

	// Different content should produce different checksum
	tmpFile3 := filepath.Join(tmpDir, "test3.txt")
	if err := os.WriteFile(tmpFile3, []byte("different content"), 0644); err != nil {
		t.Fatalf("failed to create temp file 3: %v", err)
	}

	checksum3 := service.calculateFileChecksum(tmpFile3)
	if checksum == checksum3 {
		t.Error("different content should produce different checksum")
	}

	// Non-existent file should return empty
	nonExistent := filepath.Join(tmpDir, "nonexistent.txt")
	checksumNone := service.calculateFileChecksum(nonExistent)
	if checksumNone != "" {
		t.Error("non-existent file should return empty checksum")
	}
}

func TestCheckDeduplication(t *testing.T) {
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	config := DefaultImageBackupConfig()
	config.BackupDir = t.TempDir()
	service := NewImageBackupService(config, logger)

	ctx := context.Background()

	// Initially no deduplication should be found
	checksum, path := service.checkDeduplication(ctx, "image1", config.BackupDir)
	if path != "" {
		t.Error("should not find deduplication for new image")
	}
	if checksum != "" {
		t.Error("should not have checksum for new image")
	}

	// Add to cache
	service.checksumCache["image1"] = "abc123"
	testPath := filepath.Join(t.TempDir(), "test.tar")
	if err := os.WriteFile(testPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	service.backupPathCache["abc123"] = testPath

	// Should find deduplication now
	checksum, path = service.checkDeduplication(ctx, "image1", config.BackupDir)
	if path != testPath {
		t.Errorf("should find deduplication path, got %s", path)
	}
	if checksum != "abc123" {
		t.Errorf("should return checksum, got %s", checksum)
	}

	// Should not deduplicate within same backup directory
	_, path = service.checkDeduplication(ctx, "image1", filepath.Dir(testPath))
	if path != "" {
		t.Error("should not deduplicate within same backup directory")
	}
}

func TestLoadChecksumCache(t *testing.T) {
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	config := DefaultImageBackupConfig()
	config.BackupDir = t.TempDir()
	service := NewImageBackupService(config, logger)

	ctx := context.Background()

	// Should not error on empty directory
	if err := service.LoadChecksumCache(ctx); err != nil {
		t.Errorf("should not error on empty directory: %v", err)
	}

	// Should not error on non-existent directory
	config.BackupDir = filepath.Join(t.TempDir(), "nonexistent")
	service2 := NewImageBackupService(config, logger)
	if err := service2.LoadChecksumCache(ctx); err != nil {
		t.Errorf("should not error on non-existent directory: %v", err)
	}
}

func TestCleanupOldBackups(t *testing.T) {
	logger := zerolog.New(os.Stderr).Level(zerolog.Disabled)
	config := DefaultImageBackupConfig()
	config.BackupDir = t.TempDir()
	service := NewImageBackupService(config, logger)

	ctx := context.Background()

	// Should not error on empty directory
	removed, err := service.CleanupOldBackups(ctx, 30)
	if err != nil {
		t.Errorf("should not error on empty directory: %v", err)
	}
	if removed != 0 {
		t.Error("should not remove anything from empty directory")
	}

	// Should not error on non-existent directory
	config.BackupDir = filepath.Join(t.TempDir(), "nonexistent")
	service2 := NewImageBackupService(config, logger)
	removed, err = service2.CleanupOldBackups(ctx, 30)
	if err != nil {
		t.Errorf("should not error on non-existent directory: %v", err)
	}
	if removed != 0 {
		t.Error("should not remove anything from non-existent directory")
	}
}
