package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// DockerInfo contains information about the Docker installation on an agent.
type DockerInfo struct {
	Available       bool       `json:"available"`
	Version         string     `json:"version,omitempty"`
	APIVersion      string     `json:"api_version,omitempty"`
	ServerVersion   string     `json:"server_version,omitempty"`
	Platform        string     `json:"platform,omitempty"`
	StorageDriver   string     `json:"storage_driver,omitempty"`
	RootDir         string     `json:"root_dir,omitempty"`
	ContainerCount  int        `json:"container_count"`
	RunningCount    int        `json:"running_count"`
	PausedCount     int        `json:"paused_count"`
	StoppedCount    int        `json:"stopped_count"`
	ImageCount      int        `json:"image_count"`
	VolumeCount     int        `json:"volume_count"`
	Containers      []Container `json:"containers,omitempty"`
	Volumes         []Volume   `json:"volumes,omitempty"`
	Error           string     `json:"error,omitempty"`
	DetectedAt      time.Time  `json:"detected_at"`
}

// Detector provides Docker detection functionality.
type Detector struct {
	binary string
	logger zerolog.Logger
}

// NewDetector creates a new Detector instance.
func NewDetector(logger zerolog.Logger) *Detector {
	return &Detector{
		binary: "docker",
		logger: logger.With().Str("component", "docker_detector").Logger(),
	}
}

// NewDetectorWithBinary creates a new Detector with a custom docker binary path.
func NewDetectorWithBinary(binary string, logger zerolog.Logger) *Detector {
	return &Detector{
		binary: binary,
		logger: logger.With().Str("component", "docker_detector").Logger(),
	}
}

// Detect checks if Docker is available and returns information about the installation.
func (d *Detector) Detect(ctx context.Context) (*DockerInfo, error) {
	d.logger.Debug().Msg("detecting Docker")

	info := &DockerInfo{
		DetectedAt: time.Now(),
	}

	// First check if docker binary exists
	if !d.isBinaryAvailable() {
		info.Available = false
		info.Error = "docker binary not found"
		d.logger.Debug().Msg("docker binary not found")
		return info, nil
	}

	// Check if Docker daemon is running and get version
	version, err := d.getVersion(ctx)
	if err != nil {
		info.Available = false
		if err == ErrDockerNotAvailable {
			info.Error = "docker daemon not running"
		} else {
			info.Error = err.Error()
		}
		d.logger.Debug().Err(err).Msg("docker not available")
		return info, nil
	}

	info.Available = true
	info.Version = version.ClientVersion
	info.APIVersion = version.APIVersion
	info.ServerVersion = version.ServerVersion
	info.Platform = version.Platform

	// Get system info
	sysInfo, err := d.getSystemInfo(ctx)
	if err != nil {
		d.logger.Warn().Err(err).Msg("failed to get docker system info")
	} else {
		info.StorageDriver = sysInfo.Driver
		info.RootDir = sysInfo.DockerRootDir
		info.ContainerCount = sysInfo.Containers
		info.RunningCount = sysInfo.ContainersRunning
		info.PausedCount = sysInfo.ContainersPaused
		info.StoppedCount = sysInfo.ContainersStopped
		info.ImageCount = sysInfo.Images
	}

	// Count volumes
	volumes, err := d.listVolumes(ctx)
	if err != nil {
		d.logger.Warn().Err(err).Msg("failed to list volumes")
	} else {
		info.VolumeCount = len(volumes)
		info.Volumes = volumes
	}

	// Get container list
	containers, err := d.listContainers(ctx)
	if err != nil {
		d.logger.Warn().Err(err).Msg("failed to list containers")
	} else {
		info.Containers = containers
	}

	d.logger.Info().
		Str("version", info.Version).
		Int("containers", info.ContainerCount).
		Int("volumes", info.VolumeCount).
		Msg("docker detected")

	return info, nil
}

// IsAvailable checks if Docker is available on the system.
func (d *Detector) IsAvailable(ctx context.Context) bool {
	info, _ := d.Detect(ctx)
	return info != nil && info.Available
}

// GetVersion returns the Docker version information.
func (d *Detector) GetVersion(ctx context.Context) (string, error) {
	version, err := d.getVersion(ctx)
	if err != nil {
		return "", err
	}
	return version.ClientVersion, nil
}

// dockerVersion represents version information from docker version --format json.
type dockerVersion struct {
	ClientVersion string `json:"Client.Version"`
	APIVersion    string `json:"Client.ApiVersion"`
	ServerVersion string `json:"Server.Version"`
	Platform      string `json:"Client.Os"`
}

func (d *Detector) getVersion(ctx context.Context) (*dockerVersion, error) {
	args := []string{"version", "--format", "json"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, err
	}

	// Docker outputs different JSON format, parse carefully
	var rawVersion struct {
		Client struct {
			Version    string `json:"Version"`
			APIVersion string `json:"ApiVersion"`
			Os         string `json:"Os"`
			Arch       string `json:"Arch"`
		} `json:"Client"`
		Server struct {
			Version    string `json:"Version"`
			APIVersion string `json:"ApiVersion"`
		} `json:"Server"`
	}

	if err := json.Unmarshal(output, &rawVersion); err != nil {
		// Try parsing line by line for older docker versions
		version := &dockerVersion{}
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Version:") {
				version.ClientVersion = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
			}
		}
		if version.ClientVersion != "" {
			return version, nil
		}
		return nil, fmt.Errorf("parse version: %w", err)
	}

	return &dockerVersion{
		ClientVersion: rawVersion.Client.Version,
		APIVersion:    rawVersion.Client.APIVersion,
		ServerVersion: rawVersion.Server.Version,
		Platform:      rawVersion.Client.Os + "/" + rawVersion.Client.Arch,
	}, nil
}

// dockerSystemInfo represents system information from docker system info.
type dockerSystemInfo struct {
	Driver             string `json:"Driver"`
	DockerRootDir      string `json:"DockerRootDir"`
	Containers         int    `json:"Containers"`
	ContainersRunning  int    `json:"ContainersRunning"`
	ContainersPaused   int    `json:"ContainersPaused"`
	ContainersStopped  int    `json:"ContainersStopped"`
	Images             int    `json:"Images"`
}

func (d *Detector) getSystemInfo(ctx context.Context) (*dockerSystemInfo, error) {
	args := []string{"system", "info", "--format", "json"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, err
	}

	var info dockerSystemInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("parse system info: %w", err)
	}

	return &info, nil
}

func (d *Detector) listVolumes(ctx context.Context) ([]Volume, error) {
	args := []string{"volume", "ls", "--format", "{{json .}}"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, err
	}

	var volumes []Volume
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var rawVolume struct {
			Name       string `json:"Name"`
			Driver     string `json:"Driver"`
			Mountpoint string `json:"Mountpoint"`
			Scope      string `json:"Scope"`
		}
		if err := json.Unmarshal(line, &rawVolume); err != nil {
			continue
		}

		volumes = append(volumes, Volume{
			Name:       rawVolume.Name,
			Driver:     rawVolume.Driver,
			Mountpoint: rawVolume.Mountpoint,
			Scope:      rawVolume.Scope,
		})
	}

	return volumes, nil
}

func (d *Detector) listContainers(ctx context.Context) ([]Container, error) {
	args := []string{"ps", "-a", "--format", "{{json .}}"}
	output, err := d.run(ctx, args)
	if err != nil {
		return nil, err
	}

	var containers []Container
	lines := bytes.Split(output, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var rawContainer struct {
			ID      string `json:"ID"`
			Names   string `json:"Names"`
			Image   string `json:"Image"`
			Status  string `json:"Status"`
			State   string `json:"State"`
			Ports   string `json:"Ports"`
		}
		if err := json.Unmarshal(line, &rawContainer); err != nil {
			continue
		}

		container := Container{
			ID:     rawContainer.ID,
			Name:   strings.TrimPrefix(rawContainer.Names, "/"),
			Image:  rawContainer.Image,
			Status: rawContainer.Status,
			State:  rawContainer.State,
		}

		if rawContainer.Ports != "" {
			container.Ports = strings.Split(rawContainer.Ports, ", ")
		}

		containers = append(containers, container)
	}

	return containers, nil
}

func (d *Detector) isBinaryAvailable() bool {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("where", d.binary)
	default:
		cmd = exec.Command("which", d.binary)
	}

	return cmd.Run() == nil
}

func (d *Detector) run(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		if strings.Contains(errMsg, "Cannot connect to the Docker daemon") ||
			strings.Contains(errMsg, "Is the docker daemon running") ||
			strings.Contains(errMsg, "permission denied") {
			return nil, ErrDockerNotAvailable
		}
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(errMsg))
	}

	return stdout.Bytes(), nil
}
