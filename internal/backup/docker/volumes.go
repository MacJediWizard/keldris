package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/rs/zerolog"
)

// VolumeBackup handles backing up Docker volumes via restic.
type VolumeBackup struct {
	docker *DockerCLI
	restic *backup.Restic
	logger zerolog.Logger
}

// NewVolumeBackup creates a new VolumeBackup instance.
func NewVolumeBackup(docker *DockerCLI, restic *backup.Restic, logger zerolog.Logger) *VolumeBackup {
	return &VolumeBackup{
		docker: docker,
		restic: restic,
		logger: logger.With().Str("component", "docker-volume-backup").Logger(),
	}
}

// GetVolumePath returns the host mountpoint path for a Docker volume.
func (vb *VolumeBackup) GetVolumePath(ctx context.Context, volumeName string) (string, error) {
	vb.logger.Debug().Str("volume", volumeName).Msg("getting volume path")

	args := []string{"volume", "inspect", "--format", "{{json .Mountpoint}}", volumeName}
	output, err := vb.docker.run(ctx, args)
	if err != nil {
		return "", fmt.Errorf("inspect volume %s: %w", volumeName, err)
	}

	var mountpoint string
	if err := json.Unmarshal(output, &mountpoint); err != nil {
		return "", fmt.Errorf("parse volume mountpoint: %w", err)
	}

	if mountpoint == "" {
		return "", fmt.Errorf("volume %s has no mountpoint", volumeName)
	}

	return mountpoint, nil
}

// ListVolumeContents lists the files in a Docker volume by running a temporary container.
func (vb *VolumeBackup) ListVolumeContents(ctx context.Context, volumeName string) ([]string, error) {
	vb.logger.Debug().Str("volume", volumeName).Msg("listing volume contents")

	args := []string{
		"run", "--rm",
		"-v", volumeName + ":/data:ro",
		"alpine", "find", "/data", "-type", "f",
	}
	output, err := vb.docker.run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("list volume contents %s: %w", volumeName, err)
	}

	var files []string
	for _, line := range strings.Split(string(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}

	vb.logger.Debug().Str("volume", volumeName).Int("files", len(files)).Msg("volume contents listed")
	return files, nil
}

// BackupVolumeOptions configures a Docker volume backup operation.
type BackupVolumeOptions struct {
	VolumeName       string
	Tags             []string
	Excludes         []string
	PauseContainers  bool
	BandwidthLimitKB *int
	CompressionLevel *string
}

// BackupVolume backs up a Docker volume using restic.
// If PauseContainers is true, all containers using the volume are paused during backup.
func (vb *VolumeBackup) BackupVolume(ctx context.Context, cfg backends.ResticConfig, opts BackupVolumeOptions) (*backup.BackupStats, error) {
	vb.logger.Info().
		Str("volume", opts.VolumeName).
		Bool("pause_containers", opts.PauseContainers).
		Msg("starting volume backup")

	mountpoint, err := vb.GetVolumePath(ctx, opts.VolumeName)
	if err != nil {
		return nil, err
	}

	// Find and optionally pause containers using this volume.
	var pausedContainers []string
	if opts.PauseContainers {
		pausedContainers, err = vb.pauseVolumeContainers(ctx, opts.VolumeName)
		if err != nil {
			return nil, fmt.Errorf("pause containers for volume %s: %w", opts.VolumeName, err)
		}
		defer vb.unpauseContainers(ctx, pausedContainers)
	}

	tags := append(opts.Tags, "docker-volume:"+opts.VolumeName)

	backupOpts := &backup.BackupOptions{
		BandwidthLimitKB: opts.BandwidthLimitKB,
		CompressionLevel: opts.CompressionLevel,
	}

	stats, err := vb.restic.BackupWithOptions(ctx, cfg, []string{mountpoint}, opts.Excludes, tags, backupOpts)
	if err != nil {
		return nil, fmt.Errorf("backup volume %s: %w", opts.VolumeName, err)
	}

	vb.logger.Info().
		Str("volume", opts.VolumeName).
		Str("snapshot_id", stats.SnapshotID).
		Int64("size_bytes", stats.SizeBytes).
		Msg("volume backup completed")

	return stats, nil
}

// pauseVolumeContainers pauses all running containers that use the given volume.
func (vb *VolumeBackup) pauseVolumeContainers(ctx context.Context, volumeName string) ([]string, error) {
	containers, err := vb.docker.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	var paused []string
	for _, c := range containers {
		if c.State != "running" {
			continue
		}

		info, err := vb.docker.InspectContainer(ctx, c.ID)
		if err != nil {
			vb.logger.Warn().Err(err).Str("container", c.Name).Msg("failed to inspect container")
			continue
		}

		for _, m := range info.Mounts {
			if m.Name == volumeName {
				if err := vb.docker.PauseContainer(ctx, c.ID); err != nil {
					// Unpause any already-paused containers before returning.
					vb.unpauseContainers(ctx, paused)
					return nil, fmt.Errorf("pause container %s: %w", c.Name, err)
				}
				paused = append(paused, c.ID)
				vb.logger.Info().Str("container", c.Name).Msg("paused container for volume backup")
				break
			}
		}
	}

	return paused, nil
}

// unpauseContainers unpauses the given containers, logging any errors.
func (vb *VolumeBackup) unpauseContainers(ctx context.Context, containerIDs []string) {
	for _, id := range containerIDs {
		if err := vb.docker.UnpauseContainer(ctx, id); err != nil {
			vb.logger.Error().Err(err).Str("container_id", id).Msg("failed to unpause container")
		}
	}
}
