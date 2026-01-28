// Package docker provides Docker container and volume restore functionality.
package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Common errors for Docker restore operations.
var (
	ErrContainerNotFound      = errors.New("container not found in snapshot")
	ErrVolumeNotFound         = errors.New("volume not found in snapshot")
	ErrContainerAlreadyExists = errors.New("container with same name already exists")
	ErrVolumeAlreadyExists    = errors.New("volume with same name already exists")
	ErrDockerNotAvailable     = errors.New("docker daemon not available")
	ErrInvalidRestoreTarget   = errors.New("invalid restore target")
	ErrContainerStartFailed   = errors.New("container failed to start after restore")
	ErrVolumeDependency       = errors.New("volume dependency not satisfied")
)

// RestoreTargetType represents the type of restore target.
type RestoreTargetType string

const (
	// RestoreTargetLocal restores to the local Docker host.
	RestoreTargetLocal RestoreTargetType = "local"
	// RestoreTargetRemote restores to a remote Docker host.
	RestoreTargetRemote RestoreTargetType = "remote"
)

// ContainerConfig represents the backed up configuration of a Docker container.
type ContainerConfig struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Command      []string          `json:"command,omitempty"`
	Entrypoint   []string          `json:"entrypoint,omitempty"`
	Env          []string          `json:"env,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Volumes      []VolumeMount     `json:"volumes,omitempty"`
	Ports        []PortBinding     `json:"ports,omitempty"`
	Networks     []string          `json:"networks,omitempty"`
	RestartPolicy string           `json:"restart_policy,omitempty"`
	WorkingDir   string            `json:"working_dir,omitempty"`
	User         string            `json:"user,omitempty"`
	Hostname     string            `json:"hostname,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

// VolumeMount represents a volume mount configuration.
type VolumeMount struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode,omitempty"`
	Type        string `json:"type"` // "volume", "bind", "tmpfs"
}

// PortBinding represents a port binding configuration.
type PortBinding struct {
	ContainerPort string `json:"container_port"`
	HostPort      string `json:"host_port,omitempty"`
	HostIP        string `json:"host_ip,omitempty"`
	Protocol      string `json:"protocol,omitempty"` // "tcp" or "udp"
}

// VolumeConfig represents the backed up configuration of a Docker volume.
type VolumeConfig struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Labels     map[string]string `json:"labels,omitempty"`
	Options    map[string]string `json:"options,omitempty"`
	Mountpoint string            `json:"mountpoint"`
	CreatedAt  time.Time         `json:"created_at"`
	SizeBytes  int64             `json:"size_bytes,omitempty"`
}

// RestoreTarget represents the target Docker host for restore.
type RestoreTarget struct {
	Type     RestoreTargetType `json:"type"`
	Host     string            `json:"host,omitempty"`      // For remote: docker host URL
	CertPath string            `json:"cert_path,omitempty"` // For remote: TLS cert path
	TLSVerify bool             `json:"tls_verify,omitempty"`
}

// RestoreOptions configures a Docker restore operation.
type RestoreOptions struct {
	// SnapshotPath is the path to the snapshot data.
	SnapshotPath string `json:"snapshot_path"`
	// ContainerName is the name of the container to restore (empty for volume-only restore).
	ContainerName string `json:"container_name,omitempty"`
	// VolumeName is the name of the volume to restore (empty for container restore with all volumes).
	VolumeName string `json:"volume_name,omitempty"`
	// NewContainerName is the new name for the restored container (optional).
	NewContainerName string `json:"new_container_name,omitempty"`
	// NewVolumeName is the new name for the restored volume (optional).
	NewVolumeName string `json:"new_volume_name,omitempty"`
	// Target is the Docker host target for restore.
	Target RestoreTarget `json:"target"`
	// OverwriteExisting allows overwriting existing containers/volumes.
	OverwriteExisting bool `json:"overwrite_existing"`
	// StartAfterRestore starts the container after restore.
	StartAfterRestore bool `json:"start_after_restore"`
	// VerifyStart verifies the container starts successfully.
	VerifyStart bool `json:"verify_start"`
	// VerifyTimeout is the timeout for container start verification.
	VerifyTimeout time.Duration `json:"verify_timeout,omitempty"`
}

// RestoreProgress tracks the progress of a Docker restore operation.
type RestoreProgress struct {
	mu             sync.RWMutex
	Status         string    `json:"status"` // "preparing", "restoring_volumes", "creating_container", "starting", "verifying", "completed", "failed"
	CurrentStep    string    `json:"current_step"`
	TotalSteps     int       `json:"total_steps"`
	CompletedSteps int       `json:"completed_steps"`
	TotalBytes     int64     `json:"total_bytes"`
	RestoredBytes  int64     `json:"restored_bytes"`
	CurrentVolume  string    `json:"current_volume,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	ErrorMessage   string    `json:"error_message,omitempty"`
}

// Update updates the progress with thread safety.
func (p *RestoreProgress) Update(fn func(*RestoreProgress)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	fn(p)
}

// Get returns a copy of the progress with thread safety.
func (p *RestoreProgress) Get() RestoreProgress {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return RestoreProgress{
		Status:         p.Status,
		CurrentStep:    p.CurrentStep,
		TotalSteps:     p.TotalSteps,
		CompletedSteps: p.CompletedSteps,
		TotalBytes:     p.TotalBytes,
		RestoredBytes:  p.RestoredBytes,
		CurrentVolume:  p.CurrentVolume,
		StartedAt:      p.StartedAt,
		ErrorMessage:   p.ErrorMessage,
	}
}

// PercentComplete returns the restore completion percentage.
func (p *RestoreProgress) PercentComplete() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.TotalSteps == 0 {
		return 0
	}
	return float64(p.CompletedSteps) / float64(p.TotalSteps) * 100
}

// RestoreResult contains the result of a Docker restore operation.
type RestoreResult struct {
	ContainerID   string        `json:"container_id,omitempty"`
	ContainerName string        `json:"container_name,omitempty"`
	RestoredVolumes []string    `json:"restored_volumes,omitempty"`
	Duration      time.Duration `json:"duration"`
	StartVerified bool          `json:"start_verified"`
	Warnings      []string      `json:"warnings,omitempty"`
}

// RestorePlan represents a preview of what will be restored.
type RestorePlan struct {
	Container       *ContainerConfig `json:"container,omitempty"`
	Volumes         []VolumeConfig   `json:"volumes,omitempty"`
	TotalSizeBytes  int64            `json:"total_size_bytes"`
	Conflicts       []RestoreConflict `json:"conflicts,omitempty"`
	Dependencies    []string          `json:"dependencies,omitempty"`
	EstimatedTime   time.Duration     `json:"estimated_time,omitempty"`
}

// RestoreConflict represents a conflict detected during restore planning.
type RestoreConflict struct {
	Type        string `json:"type"` // "container", "volume", "network"
	Name        string `json:"name"`
	ExistingID  string `json:"existing_id,omitempty"`
	Description string `json:"description"`
}

// Restorer handles Docker restore operations.
type Restorer struct {
	logger       zerolog.Logger
	dockerPath   string
	progressChan chan<- RestoreProgress
}

// NewRestorer creates a new Docker Restorer.
func NewRestorer(logger zerolog.Logger) *Restorer {
	dockerPath, _ := exec.LookPath("docker")
	if dockerPath == "" {
		dockerPath = "docker"
	}
	return &Restorer{
		logger:     logger.With().Str("component", "docker_restorer").Logger(),
		dockerPath: dockerPath,
	}
}

// SetProgressChannel sets the channel for progress updates.
func (r *Restorer) SetProgressChannel(ch chan<- RestoreProgress) {
	r.progressChan = ch
}

// CheckDockerAvailable verifies Docker daemon is accessible.
func (r *Restorer) CheckDockerAvailable(ctx context.Context, target RestoreTarget) error {
	args := r.buildDockerArgs(target, "info")
	cmd := exec.CommandContext(ctx, r.dockerPath, args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %v", ErrDockerNotAvailable, err)
	}
	return nil
}

// PreviewRestore generates a restore plan without performing the restore.
func (r *Restorer) PreviewRestore(ctx context.Context, opts RestoreOptions) (*RestorePlan, error) {
	r.logger.Info().
		Str("snapshot_path", opts.SnapshotPath).
		Str("container_name", opts.ContainerName).
		Str("volume_name", opts.VolumeName).
		Msg("generating restore preview")

	plan := &RestorePlan{}

	// Load container config if restoring a container
	if opts.ContainerName != "" {
		containerConfig, err := r.loadContainerConfig(opts.SnapshotPath, opts.ContainerName)
		if err != nil {
			return nil, fmt.Errorf("load container config: %w", err)
		}
		plan.Container = containerConfig

		// Check for container conflicts
		targetName := opts.NewContainerName
		if targetName == "" {
			targetName = containerConfig.Name
		}
		if exists, existingID := r.containerExists(ctx, opts.Target, targetName); exists {
			plan.Conflicts = append(plan.Conflicts, RestoreConflict{
				Type:        "container",
				Name:        targetName,
				ExistingID:  existingID,
				Description: fmt.Sprintf("Container '%s' already exists", targetName),
			})
		}

		// Add volume configs from container
		for _, mount := range containerConfig.Volumes {
			if mount.Type == "volume" {
				volConfig, err := r.loadVolumeConfig(opts.SnapshotPath, mount.Source)
				if err == nil {
					plan.Volumes = append(plan.Volumes, *volConfig)
					plan.TotalSizeBytes += volConfig.SizeBytes
				}
			}
		}
	}

	// Load specific volume if restoring just a volume
	if opts.VolumeName != "" && opts.ContainerName == "" {
		volConfig, err := r.loadVolumeConfig(opts.SnapshotPath, opts.VolumeName)
		if err != nil {
			return nil, fmt.Errorf("load volume config: %w", err)
		}
		plan.Volumes = append(plan.Volumes, *volConfig)
		plan.TotalSizeBytes += volConfig.SizeBytes
	}

	// Check for volume conflicts
	for _, vol := range plan.Volumes {
		targetName := vol.Name
		if opts.NewVolumeName != "" && len(plan.Volumes) == 1 {
			targetName = opts.NewVolumeName
		}
		if exists := r.volumeExists(ctx, opts.Target, targetName); exists {
			plan.Conflicts = append(plan.Conflicts, RestoreConflict{
				Type:        "volume",
				Name:        targetName,
				Description: fmt.Sprintf("Volume '%s' already exists", targetName),
			})
		}
	}

	return plan, nil
}

// Restore performs a Docker container or volume restore.
func (r *Restorer) Restore(ctx context.Context, opts RestoreOptions) (*RestoreResult, error) {
	start := time.Now()
	progress := &RestoreProgress{
		Status:    "preparing",
		StartedAt: start,
	}

	r.logger.Info().
		Str("snapshot_path", opts.SnapshotPath).
		Str("container_name", opts.ContainerName).
		Str("volume_name", opts.VolumeName).
		Bool("overwrite_existing", opts.OverwriteExisting).
		Msg("starting Docker restore")

	// Check Docker is available
	if err := r.CheckDockerAvailable(ctx, opts.Target); err != nil {
		return nil, err
	}

	result := &RestoreResult{}

	// Calculate total steps
	plan, err := r.PreviewRestore(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("preview restore: %w", err)
	}

	totalSteps := len(plan.Volumes)
	if plan.Container != nil {
		totalSteps += 2 // Create container + optional start
	}
	progress.TotalSteps = totalSteps
	progress.TotalBytes = plan.TotalSizeBytes

	// Check for conflicts
	if !opts.OverwriteExisting && len(plan.Conflicts) > 0 {
		var conflictMsgs []string
		for _, c := range plan.Conflicts {
			conflictMsgs = append(conflictMsgs, c.Description)
		}
		return nil, fmt.Errorf("conflicts detected: %s", strings.Join(conflictMsgs, "; "))
	}

	// Handle conflicts if overwriting
	if opts.OverwriteExisting {
		for _, conflict := range plan.Conflicts {
			switch conflict.Type {
			case "container":
				if err := r.removeContainer(ctx, opts.Target, conflict.Name); err != nil {
					r.logger.Warn().Err(err).Str("container", conflict.Name).Msg("failed to remove existing container")
				}
			case "volume":
				if err := r.removeVolume(ctx, opts.Target, conflict.Name); err != nil {
					r.logger.Warn().Err(err).Str("volume", conflict.Name).Msg("failed to remove existing volume")
				}
			}
		}
	}

	// Restore volumes
	progress.Update(func(p *RestoreProgress) {
		p.Status = "restoring_volumes"
	})
	r.sendProgress(progress)

	for _, vol := range plan.Volumes {
		targetName := vol.Name
		if opts.NewVolumeName != "" && len(plan.Volumes) == 1 {
			targetName = opts.NewVolumeName
		}

		progress.Update(func(p *RestoreProgress) {
			p.CurrentVolume = targetName
			p.CurrentStep = fmt.Sprintf("Restoring volume: %s", targetName)
		})
		r.sendProgress(progress)

		if err := r.restoreVolume(ctx, opts, vol, targetName); err != nil {
			return nil, fmt.Errorf("restore volume %s: %w", vol.Name, err)
		}

		result.RestoredVolumes = append(result.RestoredVolumes, targetName)
		progress.Update(func(p *RestoreProgress) {
			p.CompletedSteps++
			p.RestoredBytes += vol.SizeBytes
		})
		r.sendProgress(progress)
	}

	// Restore container if specified
	if plan.Container != nil {
		progress.Update(func(p *RestoreProgress) {
			p.Status = "creating_container"
			p.CurrentStep = "Creating container"
		})
		r.sendProgress(progress)

		targetName := opts.NewContainerName
		if targetName == "" {
			targetName = plan.Container.Name
		}

		// Build volume mappings for new names
		volumeMappings := make(map[string]string)
		if opts.NewVolumeName != "" && len(plan.Volumes) == 1 {
			volumeMappings[plan.Volumes[0].Name] = opts.NewVolumeName
		}

		containerID, err := r.createContainer(ctx, opts, plan.Container, targetName, volumeMappings)
		if err != nil {
			return nil, fmt.Errorf("create container: %w", err)
		}

		result.ContainerID = containerID
		result.ContainerName = targetName
		progress.Update(func(p *RestoreProgress) {
			p.CompletedSteps++
		})
		r.sendProgress(progress)

		// Start container if requested
		if opts.StartAfterRestore {
			progress.Update(func(p *RestoreProgress) {
				p.Status = "starting"
				p.CurrentStep = "Starting container"
			})
			r.sendProgress(progress)

			if err := r.startContainer(ctx, opts.Target, containerID); err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to start container: %v", err))
			} else {
				progress.Update(func(p *RestoreProgress) {
					p.CompletedSteps++
				})

				// Verify start if requested
				if opts.VerifyStart {
					progress.Update(func(p *RestoreProgress) {
						p.Status = "verifying"
						p.CurrentStep = "Verifying container started"
					})
					r.sendProgress(progress)

					timeout := opts.VerifyTimeout
					if timeout == 0 {
						timeout = 30 * time.Second
					}

					if err := r.verifyContainerStarted(ctx, opts.Target, containerID, timeout); err != nil {
						result.Warnings = append(result.Warnings, fmt.Sprintf("Container start verification failed: %v", err))
					} else {
						result.StartVerified = true
					}
				}
			}
		}
	}

	result.Duration = time.Since(start)

	progress.Update(func(p *RestoreProgress) {
		p.Status = "completed"
		p.CurrentStep = "Restore completed"
	})
	r.sendProgress(progress)

	r.logger.Info().
		Str("container_id", result.ContainerID).
		Strs("restored_volumes", result.RestoredVolumes).
		Dur("duration", result.Duration).
		Bool("start_verified", result.StartVerified).
		Msg("Docker restore completed")

	return result, nil
}

// loadContainerConfig loads container configuration from snapshot.
func (r *Restorer) loadContainerConfig(snapshotPath, containerName string) (*ContainerConfig, error) {
	configPath := filepath.Join(snapshotPath, "docker", "containers", containerName, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrContainerNotFound
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config ContainerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &config, nil
}

// loadVolumeConfig loads volume configuration from snapshot.
func (r *Restorer) loadVolumeConfig(snapshotPath, volumeName string) (*VolumeConfig, error) {
	configPath := filepath.Join(snapshotPath, "docker", "volumes", volumeName, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrVolumeNotFound
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config VolumeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &config, nil
}

// containerExists checks if a container with the given name exists.
func (r *Restorer) containerExists(ctx context.Context, target RestoreTarget, name string) (bool, string) {
	args := r.buildDockerArgs(target, "inspect", "--format", "{{.Id}}", name)
	cmd := exec.CommandContext(ctx, r.dockerPath, args...)
	output, err := cmd.Output()
	if err != nil {
		return false, ""
	}
	return true, strings.TrimSpace(string(output))
}

// volumeExists checks if a volume with the given name exists.
func (r *Restorer) volumeExists(ctx context.Context, target RestoreTarget, name string) bool {
	args := r.buildDockerArgs(target, "volume", "inspect", name)
	cmd := exec.CommandContext(ctx, r.dockerPath, args...)
	return cmd.Run() == nil
}

// removeContainer removes an existing container.
func (r *Restorer) removeContainer(ctx context.Context, target RestoreTarget, name string) error {
	// Stop first
	stopArgs := r.buildDockerArgs(target, "stop", name)
	exec.CommandContext(ctx, r.dockerPath, stopArgs...).Run()

	// Remove
	rmArgs := r.buildDockerArgs(target, "rm", "-f", name)
	cmd := exec.CommandContext(ctx, r.dockerPath, rmArgs...)
	return cmd.Run()
}

// removeVolume removes an existing volume.
func (r *Restorer) removeVolume(ctx context.Context, target RestoreTarget, name string) error {
	args := r.buildDockerArgs(target, "volume", "rm", "-f", name)
	cmd := exec.CommandContext(ctx, r.dockerPath, args...)
	return cmd.Run()
}

// restoreVolume restores a volume from the snapshot.
func (r *Restorer) restoreVolume(ctx context.Context, opts RestoreOptions, vol VolumeConfig, targetName string) error {
	r.logger.Info().
		Str("source_volume", vol.Name).
		Str("target_volume", targetName).
		Msg("restoring volume")

	// Create the volume
	createArgs := r.buildDockerArgs(opts.Target, "volume", "create")
	if vol.Driver != "" && vol.Driver != "local" {
		createArgs = append(createArgs, "--driver", vol.Driver)
	}
	for k, v := range vol.Labels {
		createArgs = append(createArgs, "--label", fmt.Sprintf("%s=%s", k, v))
	}
	for k, v := range vol.Options {
		createArgs = append(createArgs, "--opt", fmt.Sprintf("%s=%s", k, v))
	}
	createArgs = append(createArgs, targetName)

	cmd := exec.CommandContext(ctx, r.dockerPath, createArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create volume: %s: %w", string(output), err)
	}

	// Restore volume data using a temporary container
	dataPath := filepath.Join(opts.SnapshotPath, "docker", "volumes", vol.Name, "data")
	if _, err := os.Stat(dataPath); err != nil {
		if os.IsNotExist(err) {
			r.logger.Warn().Str("volume", vol.Name).Msg("no data directory found for volume, skipping data restore")
			return nil
		}
		return fmt.Errorf("check data path: %w", err)
	}

	// Use a temporary alpine container to copy data into the volume
	runArgs := r.buildDockerArgs(opts.Target,
		"run", "--rm",
		"-v", fmt.Sprintf("%s:/backup:ro", dataPath),
		"-v", fmt.Sprintf("%s:/data", targetName),
		"alpine",
		"sh", "-c", "cp -a /backup/. /data/",
	)

	cmd = exec.CommandContext(ctx, r.dockerPath, runArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restore volume data: %s: %w", string(output), err)
	}

	return nil
}

// createContainer creates a container from the backup configuration.
func (r *Restorer) createContainer(ctx context.Context, opts RestoreOptions, config *ContainerConfig, targetName string, volumeMappings map[string]string) (string, error) {
	r.logger.Info().
		Str("original_name", config.Name).
		Str("target_name", targetName).
		Msg("creating container")

	args := r.buildDockerArgs(opts.Target, "create", "--name", targetName)

	// Add environment variables
	for _, env := range config.Env {
		args = append(args, "-e", env)
	}

	// Add labels
	for k, v := range config.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	// Add volume mounts
	for _, mount := range config.Volumes {
		source := mount.Source
		// Apply volume mapping if exists
		if newName, ok := volumeMappings[source]; ok {
			source = newName
		}

		var mountSpec string
		switch mount.Type {
		case "volume":
			mountSpec = fmt.Sprintf("%s:%s", source, mount.Destination)
		case "bind":
			mountSpec = fmt.Sprintf("%s:%s", source, mount.Destination)
		default:
			continue
		}
		if mount.Mode != "" {
			mountSpec += ":" + mount.Mode
		}
		args = append(args, "-v", mountSpec)
	}

	// Add port bindings
	for _, port := range config.Ports {
		portSpec := port.ContainerPort
		if port.Protocol != "" && port.Protocol != "tcp" {
			portSpec += "/" + port.Protocol
		}
		if port.HostPort != "" {
			hostSpec := port.HostPort
			if port.HostIP != "" {
				hostSpec = port.HostIP + ":" + hostSpec
			}
			portSpec = hostSpec + ":" + portSpec
		}
		args = append(args, "-p", portSpec)
	}

	// Add networks
	for _, network := range config.Networks {
		args = append(args, "--network", network)
	}

	// Add restart policy
	if config.RestartPolicy != "" {
		args = append(args, "--restart", config.RestartPolicy)
	}

	// Add working directory
	if config.WorkingDir != "" {
		args = append(args, "-w", config.WorkingDir)
	}

	// Add user
	if config.User != "" {
		args = append(args, "-u", config.User)
	}

	// Add hostname
	if config.Hostname != "" {
		args = append(args, "-h", config.Hostname)
	}

	// Add image
	args = append(args, config.Image)

	// Add command
	if len(config.Command) > 0 {
		args = append(args, config.Command...)
	}

	cmd := exec.CommandContext(ctx, r.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create container: %s: %w", string(output), err)
	}

	containerID := strings.TrimSpace(string(output))
	return containerID, nil
}

// startContainer starts a container.
func (r *Restorer) startContainer(ctx context.Context, target RestoreTarget, containerID string) error {
	args := r.buildDockerArgs(target, "start", containerID)
	cmd := exec.CommandContext(ctx, r.dockerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start container: %s: %w", string(output), err)
	}
	return nil
}

// verifyContainerStarted verifies that a container started successfully.
func (r *Restorer) verifyContainerStarted(ctx context.Context, target RestoreTarget, containerID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		args := r.buildDockerArgs(target, "inspect", "--format", "{{.State.Status}}", containerID)
		cmd := exec.CommandContext(ctx, r.dockerPath, args...)
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("inspect container: %w", err)
		}

		status := strings.TrimSpace(string(output))
		switch status {
		case "running":
			return nil
		case "exited", "dead":
			// Get exit code
			exitArgs := r.buildDockerArgs(target, "inspect", "--format", "{{.State.ExitCode}}", containerID)
			exitCmd := exec.CommandContext(ctx, r.dockerPath, exitArgs...)
			exitOutput, _ := exitCmd.Output()
			return fmt.Errorf("%w: status=%s, exit_code=%s", ErrContainerStartFailed, status, strings.TrimSpace(string(exitOutput)))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}

	return fmt.Errorf("%w: timeout waiting for container to start", ErrContainerStartFailed)
}

// buildDockerArgs builds docker command arguments with target configuration.
func (r *Restorer) buildDockerArgs(target RestoreTarget, args ...string) []string {
	var result []string

	if target.Type == RestoreTargetRemote && target.Host != "" {
		result = append(result, "-H", target.Host)
		if target.CertPath != "" {
			result = append(result, "--tlscacert", filepath.Join(target.CertPath, "ca.pem"))
			result = append(result, "--tlscert", filepath.Join(target.CertPath, "cert.pem"))
			result = append(result, "--tlskey", filepath.Join(target.CertPath, "key.pem"))
		}
		if target.TLSVerify {
			result = append(result, "--tlsverify")
		}
	}

	result = append(result, args...)
	return result
}

// sendProgress sends a progress update if a channel is configured.
func (r *Restorer) sendProgress(progress *RestoreProgress) {
	if r.progressChan == nil {
		return
	}
	select {
	case r.progressChan <- progress.Get():
	default:
		// Don't block if channel is full
	}
}

// ListContainersInSnapshot lists all containers available in a snapshot.
func (r *Restorer) ListContainersInSnapshot(snapshotPath string) ([]ContainerConfig, error) {
	containersDir := filepath.Join(snapshotPath, "docker", "containers")
	entries, err := os.ReadDir(containersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read containers directory: %w", err)
	}

	var containers []ContainerConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		config, err := r.loadContainerConfig(snapshotPath, entry.Name())
		if err != nil {
			r.logger.Warn().Err(err).Str("container", entry.Name()).Msg("failed to load container config")
			continue
		}
		containers = append(containers, *config)
	}

	return containers, nil
}

// ListVolumesInSnapshot lists all volumes available in a snapshot.
func (r *Restorer) ListVolumesInSnapshot(snapshotPath string) ([]VolumeConfig, error) {
	volumesDir := filepath.Join(snapshotPath, "docker", "volumes")
	entries, err := os.ReadDir(volumesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read volumes directory: %w", err)
	}

	var volumes []VolumeConfig
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		config, err := r.loadVolumeConfig(snapshotPath, entry.Name())
		if err != nil {
			r.logger.Warn().Err(err).Str("volume", entry.Name()).Msg("failed to load volume config")
			continue
		}
		volumes = append(volumes, *config)
	}

	return volumes, nil
}

// Ensure io.Reader is available for potential future streaming support
var _ = io.Reader(nil)
