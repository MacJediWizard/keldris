package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DetectDocker checks if Docker is available and returns the version string.
func DetectDocker() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, ""
	}

	version := strings.TrimSpace(stdout.String())
	if version == "" {
		return false, ""
	}

	return true, version
}

// IsDockerRunning checks if the Docker daemon is responding.
func IsDockerRunning() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{.ServerVersion}}")
	return cmd.Run() == nil
}

// GetDockerInfo returns detailed information about the Docker installation.
func GetDockerInfo() (*DockerInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", "info", "--format", "{{json .}}")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
		}
		return nil, fmt.Errorf("docker info: %w: %s", err, strings.TrimSpace(errMsg))
	}

	var raw dockerInfoOutput
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		return nil, fmt.Errorf("parse docker info: %w", err)
	}

	return &DockerInfo{
		ServerVersion:   raw.ServerVersion,
		StorageDriver:   raw.Driver,
		DockerRootDir:   raw.DockerRootDir,
		Containers:      raw.Containers,
		ContRunning:     raw.ContainersRunning,
		ContPaused:      raw.ContainersPaused,
		ContStopped:     raw.ContainersStopped,
		Images:          raw.Images,
		OperatingSystem: raw.OperatingSystem,
		Architecture:    raw.Architecture,
	}, nil
}
