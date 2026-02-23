// Package docker provides Docker container and volume backup functionality.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ErrDockerNotAvailable is returned when Docker is not installed or not running.
var ErrDockerNotAvailable = errors.New("docker is not available")

// ErrContainerNotFound is returned when a container cannot be found.
var ErrContainerNotFound = errors.New("container not found")

// ErrVolumeNotFound is returned when a volume cannot be found.
var ErrVolumeNotFound = errors.New("volume not found")

// DockerCLI wraps the Docker CLI for container and volume operations.
type DockerCLI struct {
	binary string
	logger zerolog.Logger
}

// NewDockerClient creates a new DockerCLI.
func NewDockerClient(logger zerolog.Logger) *DockerCLI {
	return &DockerCLI{
		binary: "docker",
		logger: logger.With().Str("component", "docker").Logger(),
	}
}

// NewDockerClientWithBinary creates a new DockerCLI with a custom binary path.
func NewDockerClientWithBinary(binary string, logger zerolog.Logger) *DockerCLI {
	return &DockerCLI{
		binary: binary,
		logger: logger.With().Str("component", "docker").Logger(),
	}
}

// ContainerMount represents a mount point in a container for backup/restore.
type ContainerMount struct {
	Type        string `json:"type"` // bind, volume, tmpfs
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode,omitempty"` // rw, ro
	VolumeName  string `json:"volume_name,omitempty"`
}

// PortBinding represents a port binding.
type PortBinding struct {
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"` // tcp, udp
}

// NetworkInfo represents network configuration.
type NetworkInfo struct {
	Name      string `json:"name"`
	ID        string `json:"id"`
	IPAddress string `json:"ip_address,omitempty"`
}

// BackupContainerConfig represents the configuration of a Docker container for backup.
type BackupContainerConfig struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Image       string            `json:"image"`
	Hostname    string            `json:"hostname,omitempty"`
	Env         []string          `json:"env,omitempty"`
	Cmd         []string          `json:"cmd,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Mounts      []ContainerMount  `json:"mounts,omitempty"`
	Ports       []PortBinding     `json:"ports,omitempty"`
	Networks    []NetworkInfo     `json:"networks,omitempty"`
	// Full inspect output for complete restore
	InspectData json.RawMessage `json:"inspect_data,omitempty"`
}

// BackupOptions contains options for Docker backup operations.
type BackupOptions struct {
	// PauseContainers pauses running containers during volume backup for consistency.
	PauseContainers bool `json:"pause_containers"`
	// VolumeIDs specifies which volumes to backup. Empty means all volumes.
	VolumeIDs []string `json:"volume_ids,omitempty"`
	// ContainerIDs specifies which container configs to backup. Empty means all containers.
	ContainerIDs []string `json:"container_ids,omitempty"`
	// Tags are added to the restic snapshot.
	Tags []string `json:"tags,omitempty"`
	// IncludeImages backs up container images (docker save).
	IncludeImages bool `json:"include_images"`
}

// BackupResult contains the results of a Docker backup operation.
type BackupResult struct {
	SnapshotID         string        `json:"snapshot_id"`
	VolumesBackedUp    []string      `json:"volumes_backed_up"`
	ContainersBackedUp []string      `json:"containers_backed_up"`
	ContainersPaused   []string      `json:"containers_paused,omitempty"`
	SizeBytes          int64         `json:"size_bytes"`
	Duration           time.Duration `json:"duration"`
	Errors             []string      `json:"errors,omitempty"`
}

// DockerBackup provides Docker backup functionality using restic.
type DockerBackup struct {
	binary string // docker binary path
	restic *backup.Restic
	logger zerolog.Logger
}

// NewDockerBackup creates a new DockerBackup instance.
func NewDockerBackup(restic *backup.Restic, logger zerolog.Logger) *DockerBackup {
	return &DockerBackup{
		binary: "docker",
		restic: restic,
		logger: logger.With().Str("component", "docker_backup").Logger(),
	}
}

// NewDockerBackupWithBinary creates a new DockerBackup with a custom docker binary path.
func NewDockerBackupWithBinary(binary string, restic *backup.Restic, logger zerolog.Logger) *DockerBackup {
	return &DockerBackup{
		binary: binary,
		restic: restic,
		logger: logger.With().Str("component", "docker_backup").Logger(),
	}
}

// ---------------------------------------------------------------------------
// DockerCLI methods
// ---------------------------------------------------------------------------

// ListContainers returns all containers (including stopped ones).
func (d *DockerCLI) ListContainers(ctx context.Context) ([]Container, error) {
	d.logger.Debug().Msg("listing containers")

	args := []string{"ps", "-a", "--no-trunc", "--format", "{{json .}}"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var containers []Container
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var ps dockerPSOutput
		if err := json.Unmarshal(line, &ps); err != nil {
			d.logger.Warn().Err(err).Str("line", string(line)).Msg("failed to parse container")
			continue
		}

		created, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", ps.Created)

		containers = append(containers, Container{
			ID:      ps.ID,
			Name:    strings.TrimPrefix(ps.Names, "/"),
			Image:   ps.Image,
			Status:  ps.Status,
			State:   ps.State,
			Labels:  parseLabels(ps.Labels),
			Created: created,
		})
	}

	d.logger.Debug().Int("count", len(containers)).Msg("containers listed")
	return containers, nil
}

// ListVolumes returns all Docker volumes.
func (d *DockerCLI) ListVolumes(ctx context.Context) ([]Volume, error) {
	d.logger.Debug().Msg("listing volumes")

	args := []string{"volume", "ls", "--format", "{{json .}}"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}

	var volumes []Volume
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var rawVolume dockerVolumeOutput
		if err := json.Unmarshal(line, &rawVolume); err != nil {
			d.logger.Warn().Err(err).Str("line", string(line)).Msg("failed to parse volume")
			continue
		}

		volume := Volume{
			Name:       rawVolume.Name,
			Driver:     rawVolume.Driver,
			Mountpoint: rawVolume.Mountpoint,
			Scope:      rawVolume.Scope,
			Labels:     parseLabels(rawVolume.Labels),
			CreatedAt:  rawVolume.CreatedAt,
		}

		volumes = append(volumes, volume)
	}

	d.logger.Debug().Int("count", len(volumes)).Msg("volumes listed")
	return volumes, nil
}

// InspectContainer returns detailed information about a container.
func (d *DockerCLI) InspectContainer(ctx context.Context, id string) (*InspectContainerInfo, error) {
	d.logger.Debug().Str("container_id", id).Msg("inspecting container")

	args := []string{"inspect", "--format", "{{json .}}", id}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("inspect container %s: %w", id, err)
	}

	var raw dockerInspectOutput
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("parse inspect output: %w", err)
	}

	info := &InspectContainerInfo{
		ID:    raw.ID,
		Name:  strings.TrimPrefix(raw.Name, "/"),
		Image: raw.Config.Image,
		State: InspectContainerState{
			Status:   raw.State.Status,
			Running:  raw.State.Running,
			Paused:   raw.State.Paused,
			ExitCode: raw.State.ExitCode,
		},
		Config: InspectContainerConfig{
			Hostname: raw.Config.Hostname,
			Env:      raw.Config.Env,
			Cmd:      raw.Config.Cmd,
		},
		Labels:     raw.Config.Labels,
		RestartKey: raw.HostConfig.RestartPolicy.Name,
	}

	created, err := time.Parse(time.RFC3339Nano, raw.Created)
	if err == nil {
		info.Created = created
	}

	if raw.State.StartedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw.State.StartedAt); err == nil {
			info.State.StartedAt = &t
		}
	}
	if raw.State.FinishedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw.State.FinishedAt); err == nil {
			info.State.FinishedAt = &t
		}
	}

	for _, m := range raw.Mounts {
		info.Mounts = append(info.Mounts, Mount{
			Type:        m.Type,
			Name:        m.Name,
			Source:      m.Source,
			Destination: m.Destination,
			ReadOnly:    !m.RW,
		})
	}

	return info, nil
}

// PauseContainer pauses a running container.
func (d *DockerCLI) PauseContainer(ctx context.Context, id string) error {
	d.logger.Info().Str("container_id", id).Msg("pausing container")

	args := []string{"pause", id}
	if _, err := d.run(ctx, args); err != nil {
		return fmt.Errorf("pause container %s: %w", id, err)
	}

	d.logger.Info().Str("container_id", id).Msg("container paused")
	return nil
}

// UnpauseContainer unpauses a paused container.
func (d *DockerCLI) UnpauseContainer(ctx context.Context, id string) error {
	d.logger.Info().Str("container_id", id).Msg("unpausing container")

	args := []string{"unpause", id}
	if _, err := d.run(ctx, args); err != nil {
		return fmt.Errorf("unpause container %s: %w", id, err)
	}

	d.logger.Info().Str("container_id", id).Msg("container unpaused")
	return nil
}

// run executes a docker command and returns the output.
func (d *DockerCLI) run(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	d.logger.Debug().
		Str("command", d.binary).
		Strs("args", args).
		Msg("executing docker command")

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		// Check for common docker errors
		if strings.Contains(errMsg, "Cannot connect to the Docker daemon") ||
			strings.Contains(errMsg, "Is the docker daemon running") {
			return nil, ErrDockerNotAvailable
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
	}

	return stdout.Bytes(), nil
}

// ---------------------------------------------------------------------------
// DockerBackup methods
// ---------------------------------------------------------------------------

// ListContainers returns all containers on the system.
func (d *DockerBackup) ListContainers(ctx context.Context) ([]Container, error) {
	d.logger.Debug().Msg("listing containers")

	args := []string{"ps", "-a", "--format", "{{json .}}"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var containers []Container
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var ps dockerPSOutput
		if err := json.Unmarshal(line, &ps); err != nil {
			d.logger.Warn().Err(err).Msg("failed to parse container line")
			continue
		}

		created, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", ps.Created)

		containers = append(containers, Container{
			ID:      ps.ID,
			Name:    strings.TrimPrefix(ps.Names, "/"),
			Image:   ps.Image,
			State:   ps.State,
			Status:  ps.Status,
			Labels:  parseLabels(ps.Labels),
			Created: created,
		})
	}

	d.logger.Debug().Int("count", len(containers)).Msg("containers listed")
	return containers, nil
}

// ListVolumes returns all volumes on the system.
func (d *DockerBackup) ListVolumes(ctx context.Context) ([]Volume, error) {
	d.logger.Debug().Msg("listing volumes")

	args := []string{"volume", "ls", "--format", "{{json .}}"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}

	var volumes []Volume
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var vol dockerVolumeOutput
		if err := json.Unmarshal(line, &vol); err != nil {
			d.logger.Warn().Err(err).Msg("failed to parse volume line")
			continue
		}

		volumes = append(volumes, Volume{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			Labels:     parseLabels(vol.Labels),
			Scope:      vol.Scope,
			CreatedAt:  vol.CreatedAt,
		})
	}

	// Get volume details including size and usage
	for i := range volumes {
		if details, err := d.inspectVolume(ctx, volumes[i].Name); err == nil {
			volumes[i].CreatedAt = details.CreatedAt
			volumes[i].UsedBy = details.UsedBy
		}
	}

	d.logger.Debug().Int("count", len(volumes)).Msg("volumes listed")
	return volumes, nil
}

// inspectVolume gets detailed information about a volume.
func (d *DockerBackup) inspectVolume(ctx context.Context, name string) (*Volume, error) {
	args := []string{"volume", "inspect", name}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("inspect volume: %w", err)
	}

	var inspectResult []struct {
		Name       string            `json:"Name"`
		Driver     string            `json:"Driver"`
		Mountpoint string            `json:"Mountpoint"`
		Scope      string            `json:"Scope"`
		CreatedAt  string            `json:"CreatedAt"`
		Labels     map[string]string `json:"Labels"`
	}
	if err := json.Unmarshal(output, &inspectResult); err != nil {
		return nil, fmt.Errorf("parse volume inspect: %w", err)
	}

	if len(inspectResult) == 0 {
		return nil, ErrVolumeNotFound
	}

	r := inspectResult[0]
	return &Volume{
		Name:       r.Name,
		Driver:     r.Driver,
		Mountpoint: r.Mountpoint,
		Scope:      r.Scope,
		CreatedAt:  r.CreatedAt,
		Labels:     r.Labels,
	}, nil
}

// GetContainerConfig retrieves the full configuration of a container.
func (d *DockerBackup) GetContainerConfig(ctx context.Context, containerID string) (*BackupContainerConfig, error) {
	d.logger.Debug().Str("container_id", containerID).Msg("getting container config")

	args := []string{"inspect", containerID}
	output, err := d.run(ctx, args)
	if err != nil {
		if strings.Contains(err.Error(), "No such container") {
			return nil, ErrContainerNotFound
		}
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	var inspectResult []json.RawMessage
	if err := json.Unmarshal(output, &inspectResult); err != nil {
		return nil, fmt.Errorf("parse container inspect: %w", err)
	}

	if len(inspectResult) == 0 {
		return nil, ErrContainerNotFound
	}

	// Parse basic info from inspect
	var basicInfo struct {
		ID     string `json:"Id"`
		Name   string `json:"Name"`
		Config struct {
			Hostname string            `json:"Hostname"`
			Image    string            `json:"Image"`
			Env      []string          `json:"Env"`
			Cmd      []string          `json:"Cmd"`
			Labels   map[string]string `json:"Labels"`
		} `json:"Config"`
		Mounts []struct {
			Type        string `json:"Type"`
			Source      string `json:"Source"`
			Destination string `json:"Destination"`
			Mode        string `json:"Mode"`
			Name        string `json:"Name"`
		} `json:"Mounts"`
		NetworkSettings struct {
			Ports map[string][]struct {
				HostIP   string `json:"HostIp"`
				HostPort string `json:"HostPort"`
			} `json:"Ports"`
			Networks map[string]struct {
				NetworkID string `json:"NetworkID"`
				IPAddress string `json:"IPAddress"`
			} `json:"Networks"`
		} `json:"NetworkSettings"`
	}
	if err := json.Unmarshal(inspectResult[0], &basicInfo); err != nil {
		return nil, fmt.Errorf("parse container config: %w", err)
	}

	config := &BackupContainerConfig{
		ID:          basicInfo.ID,
		Name:        strings.TrimPrefix(basicInfo.Name, "/"),
		Image:       basicInfo.Config.Image,
		Hostname:    basicInfo.Config.Hostname,
		Env:         basicInfo.Config.Env,
		Cmd:         basicInfo.Config.Cmd,
		Labels:      basicInfo.Config.Labels,
		InspectData: inspectResult[0],
	}

	// Parse mounts
	for _, m := range basicInfo.Mounts {
		config.Mounts = append(config.Mounts, ContainerMount{
			Type:        m.Type,
			Source:      m.Source,
			Destination: m.Destination,
			Mode:        m.Mode,
			VolumeName:  m.Name,
		})
	}

	// Parse ports
	for portProto, bindings := range basicInfo.NetworkSettings.Ports {
		parts := strings.Split(portProto, "/")
		containerPort := parts[0]
		protocol := "tcp"
		if len(parts) > 1 {
			protocol = parts[1]
		}
		for _, binding := range bindings {
			config.Ports = append(config.Ports, PortBinding{
				HostIP:        binding.HostIP,
				HostPort:      binding.HostPort,
				ContainerPort: containerPort,
				Protocol:      protocol,
			})
		}
	}

	// Parse networks
	for name, net := range basicInfo.NetworkSettings.Networks {
		config.Networks = append(config.Networks, NetworkInfo{
			Name:      name,
			ID:        net.NetworkID,
			IPAddress: net.IPAddress,
		})
	}

	return config, nil
}

// PauseContainer pauses a running container.
func (d *DockerBackup) PauseContainer(ctx context.Context, containerID string) error {
	d.logger.Info().Str("container_id", containerID).Msg("pausing container")

	args := []string{"pause", containerID}
	_, err := d.run(ctx, args)
	if err != nil {
		return fmt.Errorf("pause container: %w", err)
	}
	return nil
}

// UnpauseContainer unpauses a paused container.
func (d *DockerBackup) UnpauseContainer(ctx context.Context, containerID string) error {
	d.logger.Info().Str("container_id", containerID).Msg("unpausing container")

	args := []string{"unpause", containerID}
	_, err := d.run(ctx, args)
	if err != nil {
		return fmt.Errorf("unpause container: %w", err)
	}
	return nil
}

// BackupVolumes backs up Docker volumes using restic.
func (d *DockerBackup) BackupVolumes(ctx context.Context, cfg backends.ResticConfig, opts BackupOptions) (*BackupResult, error) {
	d.logger.Info().
		Strs("volume_ids", opts.VolumeIDs).
		Bool("pause_containers", opts.PauseContainers).
		Msg("starting volume backup")

	start := time.Now()
	result := &BackupResult{}

	// Get volumes to backup
	volumes, err := d.ListVolumes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list volumes: %w", err)
	}

	// Filter volumes if specific ones requested
	var targetVolumes []Volume
	if len(opts.VolumeIDs) > 0 {
		volumeSet := make(map[string]bool)
		for _, id := range opts.VolumeIDs {
			volumeSet[id] = true
		}
		for _, v := range volumes {
			if volumeSet[v.Name] {
				targetVolumes = append(targetVolumes, v)
			}
		}
	} else {
		targetVolumes = volumes
	}

	if len(targetVolumes) == 0 {
		return nil, errors.New("no volumes to backup")
	}

	// Pause containers if requested
	var pausedContainers []string
	if opts.PauseContainers {
		containers, err := d.ListContainers(ctx)
		if err != nil {
			d.logger.Warn().Err(err).Msg("failed to list containers for pausing")
		} else {
			for _, c := range containers {
				if c.State == "running" {
					if err := d.PauseContainer(ctx, c.ID); err != nil {
						d.logger.Warn().Err(err).Str("container_id", c.ID).Msg("failed to pause container")
						result.Errors = append(result.Errors, fmt.Sprintf("failed to pause container %s: %v", c.Name, err))
					} else {
						pausedContainers = append(pausedContainers, c.ID)
					}
				}
			}
		}
	}

	// Ensure we unpause containers even on error
	defer func() {
		for _, containerID := range pausedContainers {
			if err := d.UnpauseContainer(ctx, containerID); err != nil {
				d.logger.Error().Err(err).Str("container_id", containerID).Msg("failed to unpause container")
			}
		}
	}()

	// Collect paths to backup
	var paths []string
	for _, v := range targetVolumes {
		if v.Mountpoint != "" {
			paths = append(paths, v.Mountpoint)
			result.VolumesBackedUp = append(result.VolumesBackedUp, v.Name)
		}
	}

	// Add docker volume tag
	tags := append([]string{"docker-volume"}, opts.Tags...)

	// Run restic backup
	stats, err := d.restic.Backup(ctx, cfg, paths, nil, tags)
	if err != nil {
		return nil, fmt.Errorf("restic backup: %w", err)
	}

	result.SnapshotID = stats.SnapshotID
	result.SizeBytes = stats.SizeBytes
	result.ContainersPaused = pausedContainers
	result.Duration = time.Since(start)

	d.logger.Info().
		Str("snapshot_id", result.SnapshotID).
		Int("volumes_backed_up", len(result.VolumesBackedUp)).
		Dur("duration", result.Duration).
		Msg("volume backup completed")

	return result, nil
}

// BackupContainerConfigs backs up container configurations as JSON.
func (d *DockerBackup) BackupContainerConfigs(ctx context.Context, containerIDs []string) ([]BackupContainerConfig, error) {
	d.logger.Info().Strs("container_ids", containerIDs).Msg("backing up container configs")

	var configs []BackupContainerConfig

	// If no specific containers, get all
	if len(containerIDs) == 0 {
		containers, err := d.ListContainers(ctx)
		if err != nil {
			return nil, fmt.Errorf("list containers: %w", err)
		}
		for _, c := range containers {
			containerIDs = append(containerIDs, c.ID)
		}
	}

	for _, containerID := range containerIDs {
		config, err := d.GetContainerConfig(ctx, containerID)
		if err != nil {
			d.logger.Warn().Err(err).Str("container_id", containerID).Msg("failed to get container config")
			continue
		}
		configs = append(configs, *config)
	}

	d.logger.Info().Int("count", len(configs)).Msg("container configs backed up")
	return configs, nil
}

// DockerBackupMetadata contains metadata for a Docker backup snapshot.
type DockerBackupMetadata struct {
	BackupID         uuid.UUID               `json:"backup_id"`
	BackupType       string                  `json:"backup_type"` // "volume", "config", "full"
	Volumes          []Volume                `json:"volumes,omitempty"`
	ContainerConfigs []BackupContainerConfig `json:"container_configs,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
	AgentHostname    string                  `json:"agent_hostname"`
}

// run executes a docker command and returns the output.
func (d *DockerBackup) run(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	d.logger.Debug().
		Str("command", d.binary).
		Strs("args", args).
		Msg("executing docker command")

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		// Check for common docker errors
		if strings.Contains(errMsg, "Cannot connect to the Docker daemon") ||
			strings.Contains(errMsg, "Is the docker daemon running") {
			return nil, ErrDockerNotAvailable
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
	}

	return stdout.Bytes(), nil
}

// parseLabels parses a Docker label string (key=val,key2=val2) into a map.
func parseLabels(labels string) map[string]string {
	if labels == "" {
		return nil
	}

	result := make(map[string]string)
	for _, pair := range strings.Split(labels, ",") {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			result[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return result
}
