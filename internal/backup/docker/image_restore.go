package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ImageRestoreOptions contains options for image-aware container restore.
type ImageRestoreOptions struct {
	// RestoreImagesFirst ensures images are restored before containers.
	RestoreImagesFirst bool

	// ImageBackupDir is the directory containing image backups.
	ImageBackupDir string

	// PullMissingImages attempts to pull images that aren't in backup.
	PullMissingImages bool

	// SkipRunningContainers skips restoring containers that are already running.
	SkipRunningContainers bool

	// Force removes existing containers before restore.
	Force bool
}

// DefaultImageRestoreOptions returns default restore options.
func DefaultImageRestoreOptions() ImageRestoreOptions {
	return ImageRestoreOptions{
		RestoreImagesFirst:    true,
		ImageBackupDir:        "/var/lib/keldris/docker-images",
		PullMissingImages:     true,
		SkipRunningContainers: true,
		Force:                 false,
	}
}

// ImageRestoreContainerInfo represents information about a Docker container for image restore.
type ImageRestoreContainerInfo struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	ImageID   string            `json:"image_id"`
	Status    string            `json:"status"`
	State     string            `json:"state"`
	Ports     []string          `json:"ports,omitempty"`
	Volumes   []string          `json:"volumes,omitempty"`
	Networks  []string          `json:"networks,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// ImageRestoreResult contains the result of an image-aware container restore operation.
type ImageRestoreResult struct {
	RestoreID         uuid.UUID `json:"restore_id"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	ImagesRestored    int       `json:"images_restored"`
	ImagesPulled      int       `json:"images_pulled"`
	ContainersCreated int       `json:"containers_created"`
	ContainersStarted int       `json:"containers_started"`
	ContainersSkipped int       `json:"containers_skipped"`
	Errors            []string  `json:"errors,omitempty"`
}

// ImageRestoreService provides image-aware Docker restore functionality.
type ImageRestoreService struct {
	imageService *ImageBackupService
	logger       zerolog.Logger
}

// NewImageRestoreService creates a new image restore service.
func NewImageRestoreService(imageService *ImageBackupService, logger zerolog.Logger) *ImageRestoreService {
	return &ImageRestoreService{
		imageService: imageService,
		logger:       logger.With().Str("component", "docker_image_restore").Logger(),
	}
}

// RestoreContainersWithImages restores Docker containers with image pre-loading.
func (s *ImageRestoreService) RestoreContainersWithImages(
	ctx context.Context,
	containers []ImageRestoreContainerInfo,
	opts ImageRestoreOptions,
) (*ImageRestoreResult, error) {
	startTime := time.Now()
	result := &ImageRestoreResult{
		RestoreID: uuid.New(),
		StartTime: startTime,
	}

	s.logger.Info().
		Int("containers", len(containers)).
		Bool("restore_images_first", opts.RestoreImagesFirst).
		Msg("starting image-aware container restore")

	// Step 1: Restore images first if enabled
	if opts.RestoreImagesFirst {
		imageIDs := make([]string, 0, len(containers))
		for _, c := range containers {
			imageIDs = append(imageIDs, c.ImageID)
		}

		if err := s.restoreRequiredImages(ctx, imageIDs, opts, result); err != nil {
			s.logger.Warn().Err(err).Msg("some images failed to restore")
		}
	}

	// Step 2: Restore containers
	for _, container := range containers {
		if err := s.restoreContainer(ctx, container, opts, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("container %s: %v", container.Name, err))
			continue
		}
	}

	result.EndTime = time.Now()

	s.logger.Info().
		Int("images_restored", result.ImagesRestored).
		Int("images_pulled", result.ImagesPulled).
		Int("containers_created", result.ContainersCreated).
		Int("containers_started", result.ContainersStarted).
		Int("containers_skipped", result.ContainersSkipped).
		Int("errors", len(result.Errors)).
		Dur("duration", result.EndTime.Sub(result.StartTime)).
		Msg("image-aware container restore completed")

	return result, nil
}

// restoreRequiredImages restores images needed for containers.
func (s *ImageRestoreService) restoreRequiredImages(
	ctx context.Context,
	imageIDs []string,
	opts ImageRestoreOptions,
	result *ImageRestoreResult,
) error {
	s.logger.Debug().
		Strs("image_ids", imageIDs).
		Msg("restoring required images")

	// Try to restore from backup first
	err := s.imageService.RestoreImagesBeforeContainers(ctx, opts.ImageBackupDir, imageIDs)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to restore images from backup")
	}

	// Check which images are now available
	existingImages, err := s.imageService.ListImages(ctx)
	if err != nil {
		return fmt.Errorf("list images: %w", err)
	}

	existingIDs := make(map[string]bool)
	for _, img := range existingImages {
		existingIDs[img.ID] = true
		for _, tag := range img.RepoTags {
			existingIDs[tag] = true
		}
	}

	// Count restored images
	for _, id := range imageIDs {
		if existingIDs[id] {
			result.ImagesRestored++
		}
	}

	// Pull missing images if enabled
	if opts.PullMissingImages {
		for _, id := range imageIDs {
			if !existingIDs[id] {
				if err := s.pullImage(ctx, id); err != nil {
					s.logger.Warn().
						Str("image", id).
						Err(err).
						Msg("failed to pull image")
					result.Errors = append(result.Errors, fmt.Sprintf("pull image %s: %v", id, err))
				} else {
					result.ImagesPulled++
				}
			}
		}
	}

	return nil
}

// pullImage pulls a Docker image.
func (s *ImageRestoreService) pullImage(ctx context.Context, image string) error {
	s.logger.Info().Str("image", image).Msg("pulling image")

	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker pull: %w: %s", err, stderr.String())
	}

	return nil
}

// restoreContainer restores a single container.
func (s *ImageRestoreService) restoreContainer(
	ctx context.Context,
	container ImageRestoreContainerInfo,
	opts ImageRestoreOptions,
	result *ImageRestoreResult,
) error {
	s.logger.Debug().
		Str("container_name", container.Name).
		Str("image", container.Image).
		Msg("restoring container")

	// Check if container already exists
	exists, running, err := s.checkContainerStatus(ctx, container.Name)
	if err != nil {
		return fmt.Errorf("check status: %w", err)
	}

	if exists {
		if running && opts.SkipRunningContainers {
			s.logger.Debug().
				Str("container_name", container.Name).
				Msg("skipping running container")
			result.ContainersSkipped++
			return nil
		}

		if opts.Force {
			if err := s.forceRemoveContainer(ctx, container.Name); err != nil {
				return fmt.Errorf("remove existing container: %w", err)
			}
		} else {
			s.logger.Debug().
				Str("container_name", container.Name).
				Msg("container already exists, skipping")
			result.ContainersSkipped++
			return nil
		}
	}

	// Create container
	if err := s.createContainerFromInfo(ctx, container); err != nil {
		return fmt.Errorf("create container: %w", err)
	}
	result.ContainersCreated++

	// Start container
	if container.State == "running" {
		if err := s.startContainerByName(ctx, container.Name); err != nil {
			return fmt.Errorf("start container: %w", err)
		}
		result.ContainersStarted++
	}

	s.logger.Info().
		Str("container_name", container.Name).
		Msg("container restored successfully")

	return nil
}

// checkContainerStatus checks if a container exists and its running state.
func (s *ImageRestoreService) checkContainerStatus(ctx context.Context, name string) (exists bool, running bool, err error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", name)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, false, nil
	}

	runningStr := strings.TrimSpace(stdout.String())
	return true, runningStr == "true", nil
}

// forceRemoveContainer stops and removes a container.
func (s *ImageRestoreService) forceRemoveContainer(ctx context.Context, name string) error {
	s.logger.Debug().Str("container", name).Msg("removing container")

	stopCmd := exec.CommandContext(ctx, "docker", "stop", name)
	stopCmd.Run() // Ignore error if already stopped

	rmCmd := exec.CommandContext(ctx, "docker", "rm", "-f", name)
	var stderr bytes.Buffer
	rmCmd.Stderr = &stderr

	if err := rmCmd.Run(); err != nil {
		return fmt.Errorf("docker rm: %w: %s", err, stderr.String())
	}

	return nil
}

// createContainerFromInfo creates a container from backup info.
func (s *ImageRestoreService) createContainerFromInfo(ctx context.Context, container ImageRestoreContainerInfo) error {
	s.logger.Debug().
		Str("container", container.Name).
		Str("image", container.Image).
		Msg("creating container")

	args := []string{"create", "--name", container.Name}

	for _, port := range container.Ports {
		args = append(args, "-p", port)
	}

	for _, vol := range container.Volumes {
		args = append(args, "-v", vol)
	}

	for _, net := range container.Networks {
		if net != "bridge" && net != "host" && net != "none" {
			args = append(args, "--network", net)
		}
	}

	for key, val := range container.Labels {
		args = append(args, "-l", fmt.Sprintf("%s=%s", key, val))
	}

	args = append(args, container.Image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker create: %w: %s", err, stderr.String())
	}

	return nil
}

// startContainerByName starts a container by name.
func (s *ImageRestoreService) startContainerByName(ctx context.Context, name string) error {
	s.logger.Debug().Str("container", name).Msg("starting container")

	cmd := exec.CommandContext(ctx, "docker", "start", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker start: %w: %s", err, stderr.String())
	}

	return nil
}

// GetContainerInfoForBackup gets information about running containers for backup.
func (s *ImageRestoreService) GetContainerInfoForBackup(ctx context.Context) ([]ImageRestoreContainerInfo, error) {
	s.logger.Debug().Msg("getting container information")

	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--format", "{{json .}}", "--no-trunc")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker ps: %w: %s", err, stderr.String())
	}

	var containers []ImageRestoreContainerInfo
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		var c struct {
			ID      string `json:"ID"`
			Names   string `json:"Names"`
			Image   string `json:"Image"`
			Status  string `json:"Status"`
			State   string `json:"State"`
			Ports   string `json:"Ports"`
			Labels  string `json:"Labels"`
			Created string `json:"CreatedAt"`
		}
		if err := json.Unmarshal([]byte(line), &c); err != nil {
			s.logger.Warn().Str("line", line).Err(err).Msg("failed to parse container info")
			continue
		}

		info, err := s.inspectContainerForImageRestore(ctx, c.ID)
		if err != nil {
			s.logger.Warn().Str("container", c.ID).Err(err).Msg("failed to inspect container")
			continue
		}

		containers = append(containers, *info)
	}

	return containers, nil
}

// inspectContainerForImageRestore gets detailed container information.
func (s *ImageRestoreService) inspectContainerForImageRestore(ctx context.Context, containerID string) (*ImageRestoreContainerInfo, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", containerID)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("docker inspect: %w: %s", err, stderr.String())
	}

	var results []struct {
		ID      string `json:"Id"`
		Name    string `json:"Name"`
		Image   string `json:"Image"`
		Created string `json:"Created"`
		State   struct {
			Status  string `json:"Status"`
			Running bool   `json:"Running"`
		} `json:"State"`
		Config struct {
			Image  string            `json:"Image"`
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
		HostConfig struct {
			Binds        []string `json:"Binds"`
			PortBindings map[string][]struct {
				HostIP   string `json:"HostIp"`
				HostPort string `json:"HostPort"`
			} `json:"PortBindings"`
		} `json:"HostConfig"`
		NetworkSettings struct {
			Networks map[string]interface{} `json:"Networks"`
		} `json:"NetworkSettings"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("parse inspect: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no container found: %s", containerID)
	}

	r := results[0]
	info := &ImageRestoreContainerInfo{
		ID:      r.ID,
		Name:    strings.TrimPrefix(r.Name, "/"),
		Image:   r.Config.Image,
		ImageID: r.Image,
		Status:  r.State.Status,
		Labels:  r.Config.Labels,
		Volumes: r.HostConfig.Binds,
	}

	if r.State.Running {
		info.State = "running"
	} else {
		info.State = "stopped"
	}

	for port, bindings := range r.HostConfig.PortBindings {
		for _, b := range bindings {
			if b.HostPort != "" {
				info.Ports = append(info.Ports, fmt.Sprintf("%s:%s", b.HostPort, port))
			}
		}
	}

	for network := range r.NetworkSettings.Networks {
		info.Networks = append(info.Networks, network)
	}

	if t, err := time.Parse(time.RFC3339Nano, r.Created); err == nil {
		info.CreatedAt = t
	}

	return info, nil
}
