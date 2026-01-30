// Package docker provides Docker image backup and registry functionality.
package docker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ImageBackupConfig holds configuration for image backup operations.
type ImageBackupConfig struct {
	// BackupDir is the directory where image backups are stored.
	BackupDir string

	// ExcludePublicImages skips backing up images from public registries.
	ExcludePublicImages bool

	// PublicRegistries is a list of registry prefixes considered public.
	PublicRegistries []string

	// CompressionLevel for tar archives (gzip level 1-9, 0 for none).
	CompressionLevel int

	// MaxConcurrent is the maximum number of concurrent image exports.
	MaxConcurrent int
}

// DefaultImageBackupConfig returns a default configuration.
func DefaultImageBackupConfig() ImageBackupConfig {
	return ImageBackupConfig{
		BackupDir:           "/var/lib/keldris/docker-images",
		ExcludePublicImages: false,
		PublicRegistries: []string{
			"docker.io/",
			"gcr.io/",
			"ghcr.io/",
			"quay.io/",
			"registry.k8s.io/",
			"mcr.microsoft.com/",
			"public.ecr.aws/",
		},
		CompressionLevel: 6,
		MaxConcurrent:    2,
	}
}

// ImageInfo represents information about a Docker image.
type ImageInfo struct {
	ID          string    `json:"id"`
	RepoTags    []string  `json:"repo_tags"`
	RepoDigests []string  `json:"repo_digests,omitempty"`
	Size        int64     `json:"size"`
	Created     time.Time `json:"created"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ImageVersion tracks a specific version of an image for a container.
type ImageVersion struct {
	ContainerID   string    `json:"container_id"`
	ContainerName string    `json:"container_name"`
	ImageID       string    `json:"image_id"`
	ImageTag      string    `json:"image_tag"`
	ImageDigest   string    `json:"image_digest,omitempty"`
	BackupTime    time.Time `json:"backup_time"`
	BackupPath    string    `json:"backup_path"`
	SizeBytes     int64     `json:"size_bytes"`
}

// ImageBackupResult contains the result of an image backup operation.
type ImageBackupResult struct {
	BackupID        uuid.UUID       `json:"backup_id"`
	StartTime       time.Time       `json:"start_time"`
	EndTime         time.Time       `json:"end_time"`
	ImagesBackedUp  int             `json:"images_backed_up"`
	ImagesSkipped   int             `json:"images_skipped"`
	TotalSizeBytes  int64           `json:"total_size_bytes"`
	ImageVersions   []ImageVersion  `json:"image_versions"`
	Errors          []string        `json:"errors,omitempty"`
	DeduplicatedIDs []string        `json:"deduplicated_ids,omitempty"` // Images that were deduplicated
}

// ImageBackupManifest stores metadata about backed up images.
type ImageBackupManifest struct {
	Version       string         `json:"version"`
	BackupID      uuid.UUID      `json:"backup_id"`
	CreatedAt     time.Time      `json:"created_at"`
	Images        []ImageVersion `json:"images"`
	Checksums     map[string]string `json:"checksums"` // imageID -> sha256 of backup file
	Deduplicated  map[string]string `json:"deduplicated,omitempty"` // imageID -> original backup path
}

// RegistryBackupInfo contains information about a private registry backup.
type RegistryBackupInfo struct {
	RegistryURL     string    `json:"registry_url"`
	BackupPath      string    `json:"backup_path"`
	BackupTime      time.Time `json:"backup_time"`
	ImagesCount     int       `json:"images_count"`
	TotalSizeBytes  int64     `json:"total_size_bytes"`
}

// ImageBackupService provides Docker image backup functionality.
type ImageBackupService struct {
	config    ImageBackupConfig
	logger    zerolog.Logger
	mu        sync.Mutex

	// checksumCache stores checksums for deduplication
	checksumCache map[string]string // imageID -> checksum
	// backupPathCache stores backup paths for deduplication
	backupPathCache map[string]string // checksum -> backup path
}

// NewImageBackupService creates a new image backup service.
func NewImageBackupService(config ImageBackupConfig, logger zerolog.Logger) *ImageBackupService {
	return &ImageBackupService{
		config:          config,
		logger:          logger.With().Str("component", "docker_image_backup").Logger(),
		checksumCache:   make(map[string]string),
		backupPathCache: make(map[string]string),
	}
}

// ListImages returns all Docker images on the system.
func (s *ImageBackupService) ListImages(ctx context.Context) ([]ImageInfo, error) {
	s.logger.Debug().Msg("listing Docker images")

	cmd := exec.CommandContext(ctx, "docker", "images", "--format", "{{json .}}", "--no-trunc")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker images failed: %w: %s", err, stderr.String())
	}

	var images []ImageInfo
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var img struct {
			ID         string `json:"ID"`
			Repository string `json:"Repository"`
			Tag        string `json:"Tag"`
			Size       string `json:"Size"`
			CreatedAt  string `json:"CreatedAt"`
		}
		if err := json.Unmarshal([]byte(line), &img); err != nil {
			s.logger.Warn().Str("line", line).Err(err).Msg("failed to parse image info")
			continue
		}

		// Get detailed image info
		info, err := s.inspectImage(ctx, img.ID)
		if err != nil {
			s.logger.Warn().Str("image_id", img.ID).Err(err).Msg("failed to inspect image")
			continue
		}

		images = append(images, *info)
	}

	s.logger.Debug().Int("count", len(images)).Msg("images listed")
	return images, nil
}

// inspectImage gets detailed information about a Docker image.
func (s *ImageBackupService) inspectImage(ctx context.Context, imageID string) (*ImageInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", imageID)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker inspect failed: %w: %s", err, stderr.String())
	}

	var inspectResults []struct {
		ID          string    `json:"Id"`
		RepoTags    []string  `json:"RepoTags"`
		RepoDigests []string  `json:"RepoDigests"`
		Size        int64     `json:"Size"`
		Created     time.Time `json:"Created"`
		Config      struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &inspectResults); err != nil {
		return nil, fmt.Errorf("parse inspect result: %w", err)
	}

	if len(inspectResults) == 0 {
		return nil, fmt.Errorf("no inspect results for image %s", imageID)
	}

	result := inspectResults[0]
	return &ImageInfo{
		ID:          result.ID,
		RepoTags:    result.RepoTags,
		RepoDigests: result.RepoDigests,
		Size:        result.Size,
		Created:     result.Created,
		Labels:      result.Config.Labels,
	}, nil
}

// IsPublicImage checks if an image is from a public registry.
func (s *ImageBackupService) IsPublicImage(image ImageInfo) bool {
	for _, tag := range image.RepoTags {
		isPublic := false
		for _, registry := range s.config.PublicRegistries {
			if strings.HasPrefix(tag, registry) {
				isPublic = true
				break
			}
		}
		// Check if this looks like a private registry
		// Private registries have a domain or localhost with port in the first part
		if strings.Contains(tag, "/") {
			firstPart := strings.Split(tag, "/")[0]
			// If the first part contains a dot (domain) or colon (port), it's a registry URL
			if strings.Contains(firstPart, ".") || strings.Contains(firstPart, ":") {
				// It's a registry URL - only public if we matched a public registry above
				if !isPublic {
					return false // This image is from a private registry
				}
			} else {
				// No dot or colon means it's a Docker Hub user/org (e.g., "library/nginx")
				isPublic = true
			}
		} else {
			// No slash means it's a Docker Hub official image (e.g., "nginx:latest")
			isPublic = true
		}
		if isPublic {
			return true
		}
	}
	return false
}

// ExportImage exports a Docker image to a tar file.
func (s *ImageBackupService) ExportImage(ctx context.Context, imageID string, outputPath string) error {
	s.logger.Info().
		Str("image_id", imageID).
		Str("output_path", outputPath).
		Msg("exporting image")

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	// Run docker save
	cmd := exec.CommandContext(ctx, "docker", "save", imageID)
	cmd.Stdout = outFile

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("docker save failed: %w: %s", err, stderr.String())
	}

	s.logger.Info().
		Str("image_id", imageID).
		Str("output_path", outputPath).
		Msg("image exported successfully")

	return nil
}

// LoadImage loads a Docker image from a tar file.
func (s *ImageBackupService) LoadImage(ctx context.Context, inputPath string) error {
	s.logger.Info().
		Str("input_path", inputPath).
		Msg("loading image")

	// Open input file
	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer inFile.Close()

	// Run docker load
	cmd := exec.CommandContext(ctx, "docker", "load")
	cmd.Stdin = inFile

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker load failed: %w: %s", err, stderr.String())
	}

	s.logger.Info().
		Str("input_path", inputPath).
		Str("output", stdout.String()).
		Msg("image loaded successfully")

	return nil
}

// GetContainerImages returns images used by running containers.
func (s *ImageBackupService) GetContainerImages(ctx context.Context) ([]ImageVersion, error) {
	s.logger.Debug().Msg("getting container images")

	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "{{json .}}", "--no-trunc")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker ps failed: %w: %s", err, stderr.String())
	}

	var versions []ImageVersion
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var container struct {
			ID    string `json:"ID"`
			Names string `json:"Names"`
			Image string `json:"Image"`
		}
		if err := json.Unmarshal([]byte(line), &container); err != nil {
			s.logger.Warn().Str("line", line).Err(err).Msg("failed to parse container info")
			continue
		}

		// Get image ID for the container
		imageInfo, err := s.inspectImage(ctx, container.Image)
		if err != nil {
			s.logger.Warn().
				Str("container_id", container.ID).
				Str("image", container.Image).
				Err(err).
				Msg("failed to get image info for container")
			continue
		}

		version := ImageVersion{
			ContainerID:   container.ID,
			ContainerName: container.Names,
			ImageID:       imageInfo.ID,
			ImageTag:      container.Image,
			BackupTime:    time.Now(),
		}

		if len(imageInfo.RepoDigests) > 0 {
			version.ImageDigest = imageInfo.RepoDigests[0]
		}

		versions = append(versions, version)
	}

	s.logger.Debug().Int("count", len(versions)).Msg("container images retrieved")
	return versions, nil
}

// BackupImages backs up Docker images based on configuration.
func (s *ImageBackupService) BackupImages(ctx context.Context, containerIDs []string) (*ImageBackupResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	startTime := time.Now()
	result := &ImageBackupResult{
		BackupID:  uuid.New(),
		StartTime: startTime,
	}

	s.logger.Info().
		Strs("container_ids", containerIDs).
		Bool("exclude_public", s.config.ExcludePublicImages).
		Msg("starting image backup")

	// Create backup directory
	backupDir := filepath.Join(s.config.BackupDir, result.BackupID.String())
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	// Get images to backup
	var imagesToBackup []ImageInfo
	if len(containerIDs) > 0 {
		// Backup images for specific containers
		for _, containerID := range containerIDs {
			cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.Image}}", containerID)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout
			if err := cmd.Run(); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to get image for container %s: %v", containerID, err))
				continue
			}

			imageID := strings.TrimSpace(stdout.String())
			info, err := s.inspectImage(ctx, imageID)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to inspect image %s: %v", imageID, err))
				continue
			}

			imagesToBackup = append(imagesToBackup, *info)
		}
	} else {
		// Backup all images
		images, err := s.ListImages(ctx)
		if err != nil {
			return nil, fmt.Errorf("list images: %w", err)
		}
		imagesToBackup = images
	}

	// Deduplicate images by ID
	uniqueImages := make(map[string]ImageInfo)
	for _, img := range imagesToBackup {
		uniqueImages[img.ID] = img
	}

	// Process images with concurrency limit
	sem := make(chan struct{}, s.config.MaxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, img := range uniqueImages {
		img := img // capture for closure

		// Check if public and should be excluded
		if s.config.ExcludePublicImages && s.IsPublicImage(img) {
			s.logger.Debug().
				Str("image_id", img.ID).
				Strs("tags", img.RepoTags).
				Msg("skipping public image")
			result.ImagesSkipped++
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check for deduplication
			checksum, duplicatePath := s.checkDeduplication(ctx, img.ID, backupDir)
			if duplicatePath != "" {
				mu.Lock()
				result.DeduplicatedIDs = append(result.DeduplicatedIDs, img.ID)
				mu.Unlock()
				s.logger.Debug().
					Str("image_id", img.ID).
					Str("duplicate_of", duplicatePath).
					Msg("image deduplicated")
				return
			}

			// Export the image
			shortID := img.ID
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			outputPath := filepath.Join(backupDir, fmt.Sprintf("%s.tar", shortID))

			if err := s.ExportImage(ctx, img.ID, outputPath); err != nil {
				mu.Lock()
				result.Errors = append(result.Errors, fmt.Sprintf("failed to export image %s: %v", img.ID, err))
				mu.Unlock()
				return
			}

			// Calculate and store checksum
			if checksum == "" {
				checksum = s.calculateFileChecksum(outputPath)
			}
			if checksum != "" {
				s.mu.Lock()
				s.checksumCache[img.ID] = checksum
				s.backupPathCache[checksum] = outputPath
				s.mu.Unlock()
			}

			// Get file size
			stat, _ := os.Stat(outputPath)
			var size int64
			if stat != nil {
				size = stat.Size()
			}

			// Create version record
			version := ImageVersion{
				ImageID:    img.ID,
				BackupTime: time.Now(),
				BackupPath: outputPath,
				SizeBytes:  size,
			}
			if len(img.RepoTags) > 0 {
				version.ImageTag = img.RepoTags[0]
			}
			if len(img.RepoDigests) > 0 {
				version.ImageDigest = img.RepoDigests[0]
			}

			mu.Lock()
			result.ImageVersions = append(result.ImageVersions, version)
			result.ImagesBackedUp++
			result.TotalSizeBytes += size
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Write manifest
	manifest := ImageBackupManifest{
		Version:   "1.0",
		BackupID:  result.BackupID,
		CreatedAt: time.Now(),
		Images:    result.ImageVersions,
		Checksums: make(map[string]string),
	}

	for _, v := range result.ImageVersions {
		if checksum, ok := s.checksumCache[v.ImageID]; ok {
			manifest.Checksums[v.ImageID] = checksum
		}
	}

	manifestPath := filepath.Join(backupDir, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to marshal manifest: %v", err))
	} else {
		if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to write manifest: %v", err))
		}
	}

	result.EndTime = time.Now()

	s.logger.Info().
		Str("backup_id", result.BackupID.String()).
		Int("images_backed_up", result.ImagesBackedUp).
		Int("images_skipped", result.ImagesSkipped).
		Int64("total_size_bytes", result.TotalSizeBytes).
		Int("errors", len(result.Errors)).
		Dur("duration", result.EndTime.Sub(result.StartTime)).
		Msg("image backup completed")

	return result, nil
}

// checkDeduplication checks if an image can be deduplicated.
func (s *ImageBackupService) checkDeduplication(ctx context.Context, imageID string, currentBackupDir string) (string, string) {
	// Check if we have a checksum for this image
	if checksum, ok := s.checksumCache[imageID]; ok {
		if existingPath, ok := s.backupPathCache[checksum]; ok {
			// Verify the file still exists and is not in current backup
			if !strings.HasPrefix(existingPath, currentBackupDir) {
				if _, err := os.Stat(existingPath); err == nil {
					return checksum, existingPath
				}
			}
		}
	}
	return "", ""
}

// calculateFileChecksum calculates the SHA256 checksum of a file.
func (s *ImageBackupService) calculateFileChecksum(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// RestoreImages restores Docker images from a backup.
func (s *ImageBackupService) RestoreImages(ctx context.Context, backupDir string, imageIDs []string) error {
	s.logger.Info().
		Str("backup_dir", backupDir).
		Strs("image_ids", imageIDs).
		Msg("starting image restore")

	// Read manifest
	manifestPath := filepath.Join(backupDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var manifest ImageBackupManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	// Build list of images to restore
	imagesToRestore := manifest.Images
	if len(imageIDs) > 0 {
		imageIDSet := make(map[string]bool)
		for _, id := range imageIDs {
			imageIDSet[id] = true
		}

		var filtered []ImageVersion
		for _, img := range manifest.Images {
			if imageIDSet[img.ImageID] {
				filtered = append(filtered, img)
			}
		}
		imagesToRestore = filtered
	}

	// Sort by backup time (oldest first) to maintain proper layering
	sort.Slice(imagesToRestore, func(i, j int) bool {
		return imagesToRestore[i].BackupTime.Before(imagesToRestore[j].BackupTime)
	})

	// Restore each image
	var errs []string
	for _, img := range imagesToRestore {
		backupPath := img.BackupPath

		// If path is relative, make it absolute using backup dir
		if !filepath.IsAbs(backupPath) {
			backupPath = filepath.Join(backupDir, filepath.Base(backupPath))
		}

		// Handle deduplication - check if this is a deduplicated image
		if dedupPath, ok := manifest.Deduplicated[img.ImageID]; ok {
			backupPath = dedupPath
		}

		if err := s.LoadImage(ctx, backupPath); err != nil {
			errs = append(errs, fmt.Sprintf("failed to restore image %s: %v", img.ImageID, err))
			continue
		}

		s.logger.Info().
			Str("image_id", img.ImageID).
			Str("image_tag", img.ImageTag).
			Msg("image restored successfully")
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during restore: %s", strings.Join(errs, "; "))
	}

	s.logger.Info().
		Int("images_restored", len(imagesToRestore)).
		Msg("image restore completed")

	return nil
}

// RestoreImagesBeforeContainers ensures all required images are available before container restore.
func (s *ImageBackupService) RestoreImagesBeforeContainers(ctx context.Context, backupDir string, requiredImages []string) error {
	s.logger.Info().
		Str("backup_dir", backupDir).
		Strs("required_images", requiredImages).
		Msg("restoring images before container restore")

	// Check which images are already present
	existingImages, err := s.ListImages(ctx)
	if err != nil {
		return fmt.Errorf("list existing images: %w", err)
	}

	existingIDs := make(map[string]bool)
	for _, img := range existingImages {
		existingIDs[img.ID] = true
		// Also map by tag
		for _, tag := range img.RepoTags {
			existingIDs[tag] = true
		}
	}

	// Determine which images need to be restored
	var imagesToRestore []string
	for _, img := range requiredImages {
		if !existingIDs[img] {
			imagesToRestore = append(imagesToRestore, img)
		}
	}

	if len(imagesToRestore) == 0 {
		s.logger.Info().Msg("all required images already present")
		return nil
	}

	s.logger.Info().
		Strs("images_to_restore", imagesToRestore).
		Msg("restoring missing images")

	return s.RestoreImages(ctx, backupDir, imagesToRestore)
}

// BackupPrivateRegistry backs up data from a private registry.
func (s *ImageBackupService) BackupPrivateRegistry(ctx context.Context, registryURL string, outputPath string) (*RegistryBackupInfo, error) {
	s.logger.Info().
		Str("registry_url", registryURL).
		Str("output_path", outputPath).
		Msg("starting private registry backup")

	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	info := &RegistryBackupInfo{
		RegistryURL: registryURL,
		BackupPath:  outputPath,
		BackupTime:  time.Now(),
	}

	// List images from the registry
	// This uses docker search and pull to get registry contents
	// In a production environment, you might use the registry API directly

	cmd := exec.CommandContext(ctx, "docker", "images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", fmt.Sprintf("reference=%s/*", registryURL))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("list registry images: %w: %s", err, stderr.String())
	}

	images := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(images) == 1 && images[0] == "" {
		images = nil
	}

	info.ImagesCount = len(images)

	if len(images) == 0 {
		s.logger.Warn().
			Str("registry_url", registryURL).
			Msg("no images found in registry")
		return info, nil
	}

	// Export all registry images to a single tar
	args := append([]string{"save", "-o", outputPath}, images...)
	cmd = exec.CommandContext(ctx, "docker", args...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker save registry images: %w: %s", err, stderr.String())
	}

	// Get file size
	stat, err := os.Stat(outputPath)
	if err == nil {
		info.TotalSizeBytes = stat.Size()
	}

	s.logger.Info().
		Str("registry_url", registryURL).
		Int("images_count", info.ImagesCount).
		Int64("size_bytes", info.TotalSizeBytes).
		Msg("private registry backup completed")

	return info, nil
}

// CleanupOldBackups removes image backups older than the specified retention period.
func (s *ImageBackupService) CleanupOldBackups(ctx context.Context, retentionDays int) (int, error) {
	s.logger.Info().
		Int("retention_days", retentionDays).
		Msg("cleaning up old image backups")

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	var removed int

	entries, err := os.ReadDir(s.config.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if this is a valid backup directory (UUID format)
		if _, err := uuid.Parse(entry.Name()); err != nil {
			continue
		}

		// Read manifest to get backup time
		manifestPath := filepath.Join(s.config.BackupDir, entry.Name(), "manifest.json")
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest ImageBackupManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			continue
		}

		if manifest.CreatedAt.Before(cutoff) {
			backupPath := filepath.Join(s.config.BackupDir, entry.Name())
			if err := os.RemoveAll(backupPath); err != nil {
				s.logger.Warn().
					Str("path", backupPath).
					Err(err).
					Msg("failed to remove old backup")
				continue
			}

			// Remove from caches
			for _, img := range manifest.Images {
				delete(s.checksumCache, img.ImageID)
			}

			removed++
			s.logger.Debug().
				Str("backup_id", manifest.BackupID.String()).
				Time("created_at", manifest.CreatedAt).
				Msg("removed old backup")
		}
	}

	s.logger.Info().
		Int("removed", removed).
		Msg("cleanup completed")

	return removed, nil
}

// LoadChecksumCache loads checksum data from existing backups for deduplication.
func (s *ImageBackupService) LoadChecksumCache(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Debug().Msg("loading checksum cache from existing backups")

	entries, err := os.ReadDir(s.config.BackupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read backup directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(s.config.BackupDir, entry.Name(), "manifest.json")
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest ImageBackupManifest
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			continue
		}

		for imageID, checksum := range manifest.Checksums {
			s.checksumCache[imageID] = checksum

			// Find the backup path for this checksum
			for _, img := range manifest.Images {
				if img.ImageID == imageID {
					s.backupPathCache[checksum] = img.BackupPath
					break
				}
			}
		}
	}

	s.logger.Info().
		Int("cached_checksums", len(s.checksumCache)).
		Msg("checksum cache loaded")

	return nil
}
