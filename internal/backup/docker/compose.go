// Package docker provides Docker Compose stack backup and restore functionality.
package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// Errors specific to docker compose backup operations.
// Note: ErrDockerNotAvailable, ErrContainerNotFound, and ErrVolumeNotFound are defined in docker.go
var (
	ErrComposeFileNotFound = errors.New("docker-compose file not found")
	ErrInvalidComposeFile  = errors.New("invalid docker-compose file")
	ErrStackNotRunning     = errors.New("docker stack is not running")
	ErrBackupInProgress    = errors.New("backup already in progress for this stack")
	ErrRestoreInProgress   = errors.New("restore already in progress for this stack")
	ErrCircularDependency  = errors.New("circular dependency detected in services")
)

// ComposeFile represents a parsed docker-compose.yml file.
type ComposeFile struct {
	Version  string                     `yaml:"version,omitempty"`
	Services map[string]ServiceConfig   `yaml:"services"`
	Volumes  map[string]VolumeConfig    `yaml:"volumes,omitempty"`
	Networks map[string]NetworkConfig   `yaml:"networks,omitempty"`
}

// ServiceConfig represents a service definition in docker-compose.
type ServiceConfig struct {
	Image         string            `yaml:"image,omitempty"`
	Build         interface{}       `yaml:"build,omitempty"` // Can be string or BuildConfig
	Volumes       []string          `yaml:"volumes,omitempty"`
	Environment   interface{}       `yaml:"environment,omitempty"` // Can be []string or map[string]string
	EnvFile       interface{}       `yaml:"env_file,omitempty"`    // Can be string or []string
	Ports         []string          `yaml:"ports,omitempty"`
	DependsOn     interface{}       `yaml:"depends_on,omitempty"` // Can be []string or map[string]DependencyCondition
	Networks      interface{}       `yaml:"networks,omitempty"`
	Command       interface{}       `yaml:"command,omitempty"`
	Entrypoint    interface{}       `yaml:"entrypoint,omitempty"`
	Labels        map[string]string `yaml:"labels,omitempty"`
	Restart       string            `yaml:"restart,omitempty"`
	HealthCheck   *HealthCheckConfig `yaml:"healthcheck,omitempty"`
	User          string            `yaml:"user,omitempty"`
	WorkingDir    string            `yaml:"working_dir,omitempty"`
	ContainerName string            `yaml:"container_name,omitempty"`
}

// VolumeConfig represents a volume definition.
type VolumeConfig struct {
	Driver     string            `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	External   interface{}       `yaml:"external,omitempty"` // Can be bool or ExternalConfig
	Name       string            `yaml:"name,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty"`
}

// NetworkConfig represents a network definition.
type NetworkConfig struct {
	Driver     string            `yaml:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty"`
	External   interface{}       `yaml:"external,omitempty"`
	Name       string            `yaml:"name,omitempty"`
	IPAM       *IPAMConfig       `yaml:"ipam,omitempty"`
}

// IPAMConfig represents IP address management configuration.
type IPAMConfig struct {
	Driver string       `yaml:"driver,omitempty"`
	Config []IPAMPoolConfig `yaml:"config,omitempty"`
}

// IPAMPoolConfig represents an IPAM pool configuration.
type IPAMPoolConfig struct {
	Subnet  string `yaml:"subnet,omitempty"`
	Gateway string `yaml:"gateway,omitempty"`
}

// HealthCheckConfig represents a health check configuration.
type HealthCheckConfig struct {
	Test        interface{} `yaml:"test,omitempty"` // Can be string or []string
	Interval    string      `yaml:"interval,omitempty"`
	Timeout     string      `yaml:"timeout,omitempty"`
	Retries     int         `yaml:"retries,omitempty"`
	StartPeriod string      `yaml:"start_period,omitempty"`
}

// DependencyCondition represents a conditional dependency.
type DependencyCondition struct {
	Condition string `yaml:"condition,omitempty"`
}

// ContainerState represents the runtime state of a container.
type ContainerState struct {
	ServiceName string    `json:"service_name"`
	ContainerID string    `json:"container_id"`
	Status      string    `json:"status"` // running, paused, stopped, etc.
	Health      string    `json:"health,omitempty"` // healthy, unhealthy, starting
	Created     time.Time `json:"created"`
	Started     time.Time `json:"started,omitempty"`
	Image       string    `json:"image"`
	ImageID     string    `json:"image_id"`
}

// VolumeBackupInfo contains information about a backed up volume.
type VolumeBackupInfo struct {
	VolumeName   string    `json:"volume_name"`
	ServiceName  string    `json:"service_name,omitempty"` // Service that uses this volume
	MountPath    string    `json:"mount_path"`
	BackupPath   string    `json:"backup_path"`
	SizeBytes    int64     `json:"size_bytes"`
	FileCount    int       `json:"file_count"`
	BackedUpAt   time.Time `json:"backed_up_at"`
	IsNamedVolume bool     `json:"is_named_volume"`
	IsBindMount   bool     `json:"is_bind_mount"`
}

// BindMountBackupInfo contains information about a backed up bind mount.
type BindMountBackupInfo struct {
	HostPath     string    `json:"host_path"`
	ContainerPath string   `json:"container_path"`
	ServiceName  string    `json:"service_name"`
	BackupPath   string    `json:"backup_path"`
	SizeBytes    int64     `json:"size_bytes"`
	FileCount    int       `json:"file_count"`
	BackedUpAt   time.Time `json:"backed_up_at"`
}

// ImageBackupInfo contains information about a backed up Docker image.
type ImageBackupInfo struct {
	ImageName    string    `json:"image_name"`
	ImageID      string    `json:"image_id"`
	Tags         []string  `json:"tags"`
	SizeBytes    int64     `json:"size_bytes"`
	BackupPath   string    `json:"backup_path"`
	BackedUpAt   time.Time `json:"backed_up_at"`
}

// StackBackupManifest contains all the metadata needed to restore a stack.
type StackBackupManifest struct {
	Version           string                `json:"version"`
	StackName         string                `json:"stack_name"`
	ComposeFilePath   string                `json:"compose_file_path"`
	ComposeFileHash   string                `json:"compose_file_hash"`
	BackupTimestamp   time.Time             `json:"backup_timestamp"`
	ContainerStates   []ContainerState      `json:"container_states"`
	Volumes           []VolumeBackupInfo    `json:"volumes"`
	BindMounts        []BindMountBackupInfo `json:"bind_mounts"`
	Images            []ImageBackupInfo     `json:"images,omitempty"`
	EnvFiles          []string              `json:"env_files,omitempty"`
	DependencyOrder   []string              `json:"dependency_order"`
	BackupSizeBytes   int64                 `json:"backup_size_bytes"`
	IncludesImages    bool                  `json:"includes_images"`
}

// StackBackupOptions configures a stack backup operation.
type StackBackupOptions struct {
	ComposePath    string   // Path to docker-compose.yml
	BackupDir      string   // Directory to store backup
	StackName      string   // Optional stack name (derived from directory if not set)
	ExportImages   bool     // Whether to export Docker images (large)
	IncludeEnvFiles bool    // Whether to include .env files
	StopContainers bool     // Whether to stop containers before backup (for consistency)
	ExcludePaths   []string // Paths to exclude from bind mount backups
}

// StackRestoreOptions configures a stack restore operation.
type StackRestoreOptions struct {
	ManifestPath    string            // Path to backup manifest
	TargetDir       string            // Directory to restore to
	RestoreVolumes  bool              // Whether to restore volumes
	RestoreImages   bool              // Whether to restore images
	PathMappings    map[string]string // Map original paths to new paths
	StartContainers bool              // Whether to start containers after restore
}

// ComposeBackup provides Docker Compose stack backup functionality.
type ComposeBackup struct {
	logger zerolog.Logger
}

// NewComposeBackup creates a new ComposeBackup instance.
func NewComposeBackup(logger zerolog.Logger) *ComposeBackup {
	return &ComposeBackup{
		logger: logger.With().Str("component", "compose_backup").Logger(),
	}
}

// CheckDockerAvailable verifies that Docker is installed and accessible.
func (cb *ComposeBackup) CheckDockerAvailable(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		return ErrDockerNotAvailable
	}
	return nil
}

// ParseComposeFile reads and parses a docker-compose.yml file.
func (cb *ComposeBackup) ParseComposeFile(composePath string) (*ComposeFile, error) {
	data, err := os.ReadFile(composePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrComposeFileNotFound
		}
		return nil, fmt.Errorf("read compose file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidComposeFile, err)
	}

	if len(compose.Services) == 0 {
		return nil, fmt.Errorf("%w: no services defined", ErrInvalidComposeFile)
	}

	return &compose, nil
}

// GetServiceDependencyOrder returns services sorted by dependency order.
// Services with no dependencies come first, then services that depend on them.
func (cb *ComposeBackup) GetServiceDependencyOrder(compose *ComposeFile) ([]string, error) {
	// Build dependency graph
	deps := make(map[string][]string)
	allServices := make(map[string]bool)

	for name, svc := range compose.Services {
		allServices[name] = true
		deps[name] = cb.extractDependencies(svc.DependsOn)
	}

	// Topological sort using Kahn's algorithm
	// Count incoming edges for each service
	inDegree := make(map[string]int)
	for name := range allServices {
		inDegree[name] = 0
	}
	for _, depList := range deps {
		for _, dep := range depList {
			if allServices[dep] {
				inDegree[dep]++
			}
		}
	}

	// Start with services that have no dependents (nothing depends on them)
	// Actually, we want services with no dependencies first
	queue := make([]string, 0)
	for name := range allServices {
		if len(deps[name]) == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // Stable ordering

	var order []string
	visited := make(map[string]bool)

	for len(queue) > 0 {
		// Pop from queue
		svc := queue[0]
		queue = queue[1:]

		if visited[svc] {
			continue
		}
		visited[svc] = true
		order = append(order, svc)

		// Find services that depend on this one
		for name, depList := range deps {
			if visited[name] {
				continue
			}
			// Check if all dependencies are satisfied
			allSatisfied := true
			for _, dep := range depList {
				if !visited[dep] {
					allSatisfied = false
					break
				}
			}
			if allSatisfied {
				queue = append(queue, name)
			}
		}
		sort.Strings(queue)
	}

	// Check for circular dependencies
	if len(order) != len(allServices) {
		return nil, ErrCircularDependency
	}

	return order, nil
}

// extractDependencies extracts service dependencies from the depends_on field.
func (cb *ComposeBackup) extractDependencies(dependsOn interface{}) []string {
	if dependsOn == nil {
		return nil
	}

	switch v := dependsOn.(type) {
	case []interface{}:
		deps := make([]string, 0, len(v))
		for _, d := range v {
			if s, ok := d.(string); ok {
				deps = append(deps, s)
			}
		}
		return deps
	case map[string]interface{}:
		deps := make([]string, 0, len(v))
		for name := range v {
			deps = append(deps, name)
		}
		return deps
	default:
		return nil
	}
}

// GetContainerStates retrieves the current state of all containers in the stack.
func (cb *ComposeBackup) GetContainerStates(ctx context.Context, composePath, _ string) ([]ContainerState, error) {
	composeDir := filepath.Dir(composePath)

	// Use docker compose ps to get container info
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "ps", "--format", "json")
	cmd.Dir = composeDir

	output, err := cmd.Output()
	if err != nil {
		// Stack might not be running
		cb.logger.Debug().Err(err).Msg("failed to get container states, stack may not be running")
		return nil, nil
	}

	var states []ContainerState

	// Docker compose ps --format json outputs newline-delimited JSON
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var container struct {
			ID      string `json:"ID"`
			Name    string `json:"Name"`
			Service string `json:"Service"`
			State   string `json:"State"`
			Health  string `json:"Health"`
			Image   string `json:"Image"`
		}

		if err := json.Unmarshal(line, &container); err != nil {
			cb.logger.Warn().Err(err).Str("line", string(line)).Msg("failed to parse container info")
			continue
		}

		state := ContainerState{
			ServiceName: container.Service,
			ContainerID: container.ID,
			Status:      container.State,
			Health:      container.Health,
			Image:       container.Image,
		}

		// Get detailed container info
		inspectCmd := exec.CommandContext(ctx, "docker", "inspect", container.ID, "--format",
			`{"created":"{{.Created}}","started":"{{.State.StartedAt}}","image_id":"{{.Image}}"}`)
		inspectOutput, err := inspectCmd.Output()
		if err == nil {
			var inspect struct {
				Created  string `json:"created"`
				Started  string `json:"started"`
				ImageID  string `json:"image_id"`
			}
			if err := json.Unmarshal(inspectOutput, &inspect); err == nil {
				if t, err := time.Parse(time.RFC3339Nano, inspect.Created); err == nil {
					state.Created = t
				}
				if t, err := time.Parse(time.RFC3339Nano, inspect.Started); err == nil {
					state.Started = t
				}
				state.ImageID = inspect.ImageID
			}
		}

		states = append(states, state)
	}

	return states, nil
}

// ExtractVolumes extracts volume information from a compose file and running containers.
func (cb *ComposeBackup) ExtractVolumes(ctx context.Context, compose *ComposeFile, composePath string) ([]VolumeBackupInfo, []BindMountBackupInfo, error) {
	var namedVolumes []VolumeBackupInfo
	var bindMounts []BindMountBackupInfo

	composeDir := filepath.Dir(composePath)
	stackName := filepath.Base(composeDir)

	// Process each service's volumes
	for serviceName, svc := range compose.Services {
		for _, vol := range svc.Volumes {
			volInfo, bindInfo, err := cb.parseVolumeSpec(ctx, vol, serviceName, stackName, composeDir, compose.Volumes)
			if err != nil {
				cb.logger.Warn().Err(err).Str("volume", vol).Msg("failed to parse volume spec")
				continue
			}
			if volInfo != nil {
				namedVolumes = append(namedVolumes, *volInfo)
			}
			if bindInfo != nil {
				bindMounts = append(bindMounts, *bindInfo)
			}
		}
	}

	return namedVolumes, bindMounts, nil
}

// parseVolumeSpec parses a volume specification string.
func (cb *ComposeBackup) parseVolumeSpec(ctx context.Context, spec, serviceName, stackName, composeDir string, volumeDefs map[string]VolumeConfig) (*VolumeBackupInfo, *BindMountBackupInfo, error) {
	parts := strings.Split(spec, ":")
	if len(parts) < 2 {
		return nil, nil, fmt.Errorf("invalid volume spec: %s", spec)
	}

	source := parts[0]
	target := parts[1]

	// Check if it's a named volume or bind mount
	if strings.HasPrefix(source, "/") || strings.HasPrefix(source, "./") || strings.HasPrefix(source, "..") {
		// It's a bind mount
		hostPath := source
		if !filepath.IsAbs(hostPath) {
			hostPath = filepath.Join(composeDir, source)
		}
		hostPath = filepath.Clean(hostPath)

		// Get size info
		sizeBytes, fileCount := cb.getPathStats(hostPath)

		return nil, &BindMountBackupInfo{
			HostPath:      hostPath,
			ContainerPath: target,
			ServiceName:   serviceName,
			SizeBytes:     sizeBytes,
			FileCount:     fileCount,
		}, nil
	}

	// It's a named volume
	volumeName := source

	// Check if volume has a custom name in the definition
	if def, ok := volumeDefs[volumeName]; ok && def.Name != "" {
		volumeName = def.Name
	} else {
		// Docker compose prefixes volume names with project name
		volumeName = fmt.Sprintf("%s_%s", stackName, source)
	}

	// Get volume info from Docker
	cmd := exec.CommandContext(ctx, "docker", "volume", "inspect", volumeName, "--format",
		`{"name":"{{.Name}}","mountpoint":"{{.Mountpoint}}"}`)
	output, err := cmd.Output()
	if err != nil {
		// Volume might not exist yet
		return &VolumeBackupInfo{
			VolumeName:    volumeName,
			ServiceName:   serviceName,
			MountPath:     target,
			IsNamedVolume: true,
		}, nil, nil
	}

	var volInfo struct {
		Name       string `json:"name"`
		Mountpoint string `json:"mountpoint"`
	}
	if err := json.Unmarshal(output, &volInfo); err != nil {
		return nil, nil, fmt.Errorf("parse volume info: %w", err)
	}

	sizeBytes, fileCount := cb.getPathStats(volInfo.Mountpoint)

	return &VolumeBackupInfo{
		VolumeName:    volumeName,
		ServiceName:   serviceName,
		MountPath:     target,
		SizeBytes:     sizeBytes,
		FileCount:     fileCount,
		IsNamedVolume: true,
	}, nil, nil
}

// getPathStats returns the total size and file count for a path.
func (cb *ComposeBackup) getPathStats(path string) (int64, int) {
	var totalSize int64
	var fileCount int

	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	return totalSize, fileCount
}

// BackupStack performs a full backup of a Docker Compose stack.
func (cb *ComposeBackup) BackupStack(ctx context.Context, opts StackBackupOptions) (*StackBackupManifest, error) {
	cb.logger.Info().
		Str("compose_path", opts.ComposePath).
		Str("backup_dir", opts.BackupDir).
		Bool("export_images", opts.ExportImages).
		Msg("starting stack backup")

	// Verify Docker is available
	if err := cb.CheckDockerAvailable(ctx); err != nil {
		return nil, err
	}

	// Parse compose file
	compose, err := cb.ParseComposeFile(opts.ComposePath)
	if err != nil {
		return nil, err
	}

	// Determine stack name
	stackName := opts.StackName
	if stackName == "" {
		stackName = filepath.Base(filepath.Dir(opts.ComposePath))
	}

	// Create backup directory structure
	timestamp := time.Now()
	backupRoot := filepath.Join(opts.BackupDir, fmt.Sprintf("%s_%s", stackName, timestamp.Format("20060102_150405")))
	volumeBackupDir := filepath.Join(backupRoot, "volumes")
	bindMountBackupDir := filepath.Join(backupRoot, "bind_mounts")
	imageBackupDir := filepath.Join(backupRoot, "images")

	for _, dir := range []string{backupRoot, volumeBackupDir, bindMountBackupDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create backup directory: %w", err)
		}
	}

	if opts.ExportImages {
		if err := os.MkdirAll(imageBackupDir, 0755); err != nil {
			return nil, fmt.Errorf("create image backup directory: %w", err)
		}
	}

	// Get dependency order
	dependencyOrder, err := cb.GetServiceDependencyOrder(compose)
	if err != nil {
		return nil, err
	}

	// Get container states
	containerStates, err := cb.GetContainerStates(ctx, opts.ComposePath, stackName)
	if err != nil {
		cb.logger.Warn().Err(err).Msg("failed to get container states")
	}

	// Stop containers if requested (for consistent backup)
	if opts.StopContainers && len(containerStates) > 0 {
		cb.logger.Info().Msg("stopping containers for consistent backup")
		if err := cb.stopStack(ctx, opts.ComposePath); err != nil {
			return nil, fmt.Errorf("stop stack: %w", err)
		}
		defer func() {
			// Restart containers after backup
			cb.logger.Info().Msg("restarting containers after backup")
			if err := cb.startStack(ctx, opts.ComposePath); err != nil {
				cb.logger.Error().Err(err).Msg("failed to restart stack")
			}
		}()
	}

	// Extract and backup volumes
	namedVolumes, bindMounts, err := cb.ExtractVolumes(ctx, compose, opts.ComposePath)
	if err != nil {
		return nil, fmt.Errorf("extract volumes: %w", err)
	}

	// Backup named volumes
	var backedUpVolumes []VolumeBackupInfo
	for _, vol := range namedVolumes {
		backupPath := filepath.Join(volumeBackupDir, vol.VolumeName+".tar.gz")
		if err := cb.backupVolume(ctx, vol.VolumeName, backupPath); err != nil {
			cb.logger.Warn().Err(err).Str("volume", vol.VolumeName).Msg("failed to backup volume")
			continue
		}
		vol.BackupPath = backupPath
		vol.BackedUpAt = timestamp
		backedUpVolumes = append(backedUpVolumes, vol)
	}

	// Backup bind mounts
	var backedUpBindMounts []BindMountBackupInfo
	for _, mount := range bindMounts {
		// Skip excluded paths
		excluded := false
		for _, excl := range opts.ExcludePaths {
			if strings.HasPrefix(mount.HostPath, excl) {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		backupPath := filepath.Join(bindMountBackupDir, strings.ReplaceAll(mount.HostPath[1:], "/", "_")+".tar.gz")
		if err := cb.backupPath(ctx, mount.HostPath, backupPath); err != nil {
			cb.logger.Warn().Err(err).Str("path", mount.HostPath).Msg("failed to backup bind mount")
			continue
		}
		mount.BackupPath = backupPath
		mount.BackedUpAt = timestamp
		backedUpBindMounts = append(backedUpBindMounts, mount)
	}

	// Backup images if requested
	var backedUpImages []ImageBackupInfo
	if opts.ExportImages {
		images := cb.collectImages(compose, containerStates)
		for _, img := range images {
			backupPath := filepath.Join(imageBackupDir, strings.ReplaceAll(img, "/", "_")+".tar")
			imgInfo, err := cb.exportImage(ctx, img, backupPath)
			if err != nil {
				cb.logger.Warn().Err(err).Str("image", img).Msg("failed to export image")
				continue
			}
			backedUpImages = append(backedUpImages, *imgInfo)
		}
	}

	// Copy compose file
	composeBackupPath := filepath.Join(backupRoot, "docker-compose.yml")
	if err := cb.copyFile(opts.ComposePath, composeBackupPath); err != nil {
		return nil, fmt.Errorf("copy compose file: %w", err)
	}

	// Copy env files if requested
	var envFiles []string
	if opts.IncludeEnvFiles {
		envFiles, err = cb.backupEnvFiles(opts.ComposePath, backupRoot)
		if err != nil {
			cb.logger.Warn().Err(err).Msg("failed to backup env files")
		}
	}

	// Calculate total backup size
	var totalSize int64
	_ = filepath.Walk(backupRoot, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// Create manifest
	manifest := &StackBackupManifest{
		Version:         "1.0",
		StackName:       stackName,
		ComposeFilePath: opts.ComposePath,
		BackupTimestamp: timestamp,
		ContainerStates: containerStates,
		Volumes:         backedUpVolumes,
		BindMounts:      backedUpBindMounts,
		Images:          backedUpImages,
		EnvFiles:        envFiles,
		DependencyOrder: dependencyOrder,
		BackupSizeBytes: totalSize,
		IncludesImages:  opts.ExportImages,
	}

	// Calculate compose file hash
	data, _ := os.ReadFile(opts.ComposePath)
	manifest.ComposeFileHash = fmt.Sprintf("%x", data)

	// Write manifest
	manifestPath := filepath.Join(backupRoot, "manifest.json")
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	cb.logger.Info().
		Str("backup_path", backupRoot).
		Int64("total_size", totalSize).
		Int("volumes", len(backedUpVolumes)).
		Int("bind_mounts", len(backedUpBindMounts)).
		Int("images", len(backedUpImages)).
		Msg("stack backup completed")

	return manifest, nil
}

// backupVolume backs up a Docker volume to a tar.gz file.
func (cb *ComposeBackup) backupVolume(ctx context.Context, volumeName, backupPath string) error {
	cb.logger.Debug().Str("volume", volumeName).Str("path", backupPath).Msg("backing up volume")

	// Use a temporary container to read volume data
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"-v", volumeName+":/source:ro",
		"-v", filepath.Dir(backupPath)+":/backup",
		"alpine:latest",
		"tar", "czf", "/backup/"+filepath.Base(backupPath), "-C", "/source", ".")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("backup volume: %w: %s", err, string(output))
	}

	return nil
}

// backupPath backs up a host path to a tar.gz file.
func (cb *ComposeBackup) backupPath(ctx context.Context, sourcePath, backupPath string) error {
	cb.logger.Debug().Str("source", sourcePath).Str("path", backupPath).Msg("backing up path")

	// Check if path exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", sourcePath)
	}

	cmd := exec.CommandContext(ctx, "tar", "czf", backupPath, "-C", filepath.Dir(sourcePath), filepath.Base(sourcePath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("backup path: %w: %s", err, string(output))
	}

	return nil
}

// collectImages collects unique image names from compose and running containers.
func (cb *ComposeBackup) collectImages(compose *ComposeFile, states []ContainerState) []string {
	imageSet := make(map[string]bool)

	// From compose file
	for _, svc := range compose.Services {
		if svc.Image != "" {
			imageSet[svc.Image] = true
		}
	}

	// From running containers
	for _, state := range states {
		if state.Image != "" {
			imageSet[state.Image] = true
		}
	}

	images := make([]string, 0, len(imageSet))
	for img := range imageSet {
		images = append(images, img)
	}
	sort.Strings(images)

	return images
}

// exportImage exports a Docker image to a tar file.
func (cb *ComposeBackup) exportImage(ctx context.Context, imageName, backupPath string) (*ImageBackupInfo, error) {
	cb.logger.Debug().Str("image", imageName).Str("path", backupPath).Msg("exporting image")

	cmd := exec.CommandContext(ctx, "docker", "save", "-o", backupPath, imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("export image: %w: %s", err, string(output))
	}

	// Get file size
	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("stat image backup: %w", err)
	}

	// Get image ID
	inspectCmd := exec.CommandContext(ctx, "docker", "image", "inspect", imageName, "--format", "{{.Id}}")
	idOutput, _ := inspectCmd.Output()

	return &ImageBackupInfo{
		ImageName:  imageName,
		ImageID:    strings.TrimSpace(string(idOutput)),
		SizeBytes:  info.Size(),
		BackupPath: backupPath,
		BackedUpAt: time.Now(),
	}, nil
}

// backupEnvFiles backs up .env files associated with the compose file.
func (cb *ComposeBackup) backupEnvFiles(composePath, backupRoot string) ([]string, error) {
	composeDir := filepath.Dir(composePath)
	var envFiles []string

	// Look for common env files
	patterns := []string{".env", ".env.local", "*.env"}
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(composeDir, pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			destPath := filepath.Join(backupRoot, "env", filepath.Base(match))
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				continue
			}
			if err := cb.copyFile(match, destPath); err != nil {
				continue
			}
			envFiles = append(envFiles, filepath.Base(match))
		}
	}

	return envFiles, nil
}

// copyFile copies a file from src to dst.
func (cb *ComposeBackup) copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// stopStack stops all containers in the stack.
func (cb *ComposeBackup) stopStack(ctx context.Context, composePath string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "stop")
	cmd.Dir = filepath.Dir(composePath)
	return cmd.Run()
}

// startStack starts all containers in the stack.
func (cb *ComposeBackup) startStack(ctx context.Context, composePath string) error {
	cmd := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "start")
	cmd.Dir = filepath.Dir(composePath)
	return cmd.Run()
}

// RestoreStack restores a Docker Compose stack from a backup.
func (cb *ComposeBackup) RestoreStack(ctx context.Context, opts StackRestoreOptions) error {
	cb.logger.Info().
		Str("manifest_path", opts.ManifestPath).
		Str("target_dir", opts.TargetDir).
		Bool("restore_volumes", opts.RestoreVolumes).
		Bool("restore_images", opts.RestoreImages).
		Msg("starting stack restore")

	// Load manifest
	manifestData, err := os.ReadFile(opts.ManifestPath)
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}

	var manifest StackBackupManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	backupRoot := filepath.Dir(opts.ManifestPath)

	// Restore compose file
	srcCompose := filepath.Join(backupRoot, "docker-compose.yml")
	dstCompose := filepath.Join(opts.TargetDir, "docker-compose.yml")
	if err := os.MkdirAll(opts.TargetDir, 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}
	if err := cb.copyFile(srcCompose, dstCompose); err != nil {
		return fmt.Errorf("restore compose file: %w", err)
	}

	// Restore env files
	envDir := filepath.Join(backupRoot, "env")
	if _, err := os.Stat(envDir); err == nil {
		entries, _ := os.ReadDir(envDir)
		for _, entry := range entries {
			src := filepath.Join(envDir, entry.Name())
			dst := filepath.Join(opts.TargetDir, entry.Name())
			if err := cb.copyFile(src, dst); err != nil {
				cb.logger.Warn().Err(err).Str("file", entry.Name()).Msg("failed to restore env file")
			}
		}
	}

	// Restore images if requested
	if opts.RestoreImages && manifest.IncludesImages {
		for _, img := range manifest.Images {
			if err := cb.loadImage(ctx, img.BackupPath); err != nil {
				cb.logger.Warn().Err(err).Str("image", img.ImageName).Msg("failed to restore image")
			}
		}
	}

	// Create volumes and restore data
	if opts.RestoreVolumes {
		for _, vol := range manifest.Volumes {
			if err := cb.restoreVolume(ctx, vol.VolumeName, vol.BackupPath); err != nil {
				cb.logger.Warn().Err(err).Str("volume", vol.VolumeName).Msg("failed to restore volume")
			}
		}

		// Restore bind mounts
		for _, mount := range manifest.BindMounts {
			targetPath := mount.HostPath
			if mapping, ok := opts.PathMappings[mount.HostPath]; ok {
				targetPath = mapping
			}
			if err := cb.restorePath(ctx, mount.BackupPath, targetPath); err != nil {
				cb.logger.Warn().Err(err).Str("path", targetPath).Msg("failed to restore bind mount")
			}
		}
	}

	// Start containers if requested
	if opts.StartContainers {
		cb.logger.Info().Msg("starting stack")
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", dstCompose, "up", "-d")
		cmd.Dir = opts.TargetDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("start stack: %w: %s", err, string(output))
		}
	}

	cb.logger.Info().
		Str("target_dir", opts.TargetDir).
		Msg("stack restore completed")

	return nil
}

// loadImage loads a Docker image from a tar file.
func (cb *ComposeBackup) loadImage(ctx context.Context, imagePath string) error {
	cb.logger.Debug().Str("path", imagePath).Msg("loading image")
	cmd := exec.CommandContext(ctx, "docker", "load", "-i", imagePath)
	return cmd.Run()
}

// restoreVolume restores a Docker volume from a tar.gz file.
func (cb *ComposeBackup) restoreVolume(ctx context.Context, volumeName, backupPath string) error {
	cb.logger.Debug().Str("volume", volumeName).Str("path", backupPath).Msg("restoring volume")

	// Create volume if it doesn't exist
	createCmd := exec.CommandContext(ctx, "docker", "volume", "create", volumeName)
	if err := createCmd.Run(); err != nil {
		cb.logger.Debug().Err(err).Msg("volume might already exist")
	}

	// Restore data using a temporary container
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm",
		"-v", volumeName+":/target",
		"-v", filepath.Dir(backupPath)+":/backup:ro",
		"alpine:latest",
		"tar", "xzf", "/backup/"+filepath.Base(backupPath), "-C", "/target")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore volume: %w: %s", err, string(output))
	}

	return nil
}

// restorePath restores a host path from a tar.gz file.
func (cb *ComposeBackup) restorePath(ctx context.Context, backupPath, targetPath string) error {
	cb.logger.Debug().Str("backup", backupPath).Str("target", targetPath).Msg("restoring path")

	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "tar", "xzf", backupPath, "-C", targetPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restore path: %w: %s", err, string(output))
	}

	return nil
}

// ListStacks lists Docker Compose stacks that can be backed up.
func (cb *ComposeBackup) ListStacks(ctx context.Context, searchPaths []string) ([]StackInfo, error) {
	var stacks []StackInfo

	for _, searchPath := range searchPaths {
		// Find docker-compose.yml files
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors
			}
			if info.IsDir() {
				return nil
			}

			baseName := filepath.Base(path)
			if baseName == "docker-compose.yml" || baseName == "docker-compose.yaml" || baseName == "compose.yml" || baseName == "compose.yaml" {
				compose, err := cb.ParseComposeFile(path)
				if err != nil {
					cb.logger.Debug().Err(err).Str("path", path).Msg("skipping invalid compose file")
					return nil
				}

				stackName := filepath.Base(filepath.Dir(path))
				states, _ := cb.GetContainerStates(ctx, path, stackName)

				stack := StackInfo{
					Name:         stackName,
					ComposePath:  path,
					ServiceCount: len(compose.Services),
					IsRunning:    len(states) > 0,
				}

				stacks = append(stacks, stack)
			}

			return nil
		})
		if err != nil {
			cb.logger.Warn().Err(err).Str("path", searchPath).Msg("error scanning path")
		}
	}

	return stacks, nil
}

// StackInfo contains basic information about a Docker Compose stack.
type StackInfo struct {
	Name         string `json:"name"`
	ComposePath  string `json:"compose_path"`
	ServiceCount int    `json:"service_count"`
	IsRunning    bool   `json:"is_running"`
}
