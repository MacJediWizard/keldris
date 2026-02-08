package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// DockerClient wraps the Docker CLI for container and volume operations.
type DockerClient struct {
	binary string
	logger zerolog.Logger
}

// NewDockerClient creates a new DockerClient.
func NewDockerClient(logger zerolog.Logger) *DockerClient {
	return &DockerClient{
		binary: "docker",
		logger: logger.With().Str("component", "docker").Logger(),
	}
}

// NewDockerClientWithBinary creates a new DockerClient with a custom binary path.
func NewDockerClientWithBinary(binary string, logger zerolog.Logger) *DockerClient {
	return &DockerClient{
		binary: binary,
		logger: logger.With().Str("component", "docker").Logger(),
	}
}

// ListContainers returns all containers (including stopped ones).
func (d *DockerClient) ListContainers(ctx context.Context) ([]Container, error) {
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

// ListVolumes returns all Docker volumes.
func (d *DockerClient) ListVolumes(ctx context.Context) ([]Volume, error) {
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

	d.logger.Debug().Int("count", len(volumes)).Msg("volumes listed")
	return volumes, nil
}

// InspectContainer returns detailed information about a container.
func (d *DockerClient) InspectContainer(ctx context.Context, id string) (*ContainerInfo, error) {
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

	info := &ContainerInfo{
		ID:    raw.ID,
		Name:  strings.TrimPrefix(raw.Name, "/"),
		Image: raw.Config.Image,
		State: ContainerState{
			Status:   raw.State.Status,
			Running:  raw.State.Running,
			Paused:   raw.State.Paused,
			ExitCode: raw.State.ExitCode,
		},
		Config: ContainerConfig{
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
func (d *DockerClient) PauseContainer(ctx context.Context, id string) error {
	d.logger.Info().Str("container_id", id).Msg("pausing container")

	args := []string{"pause", id}
	if _, err := d.run(ctx, args); err != nil {
		return fmt.Errorf("pause container %s: %w", id, err)
	}

	d.logger.Info().Str("container_id", id).Msg("container paused")
	return nil
}

// UnpauseContainer unpauses a paused container.
func (d *DockerClient) UnpauseContainer(ctx context.Context, id string) error {
	d.logger.Info().Str("container_id", id).Msg("unpausing container")

	args := []string{"unpause", id}
	if _, err := d.run(ctx, args); err != nil {
		return fmt.Errorf("unpause container %s: %w", id, err)
	}

	d.logger.Info().Str("container_id", id).Msg("container unpaused")
	return nil
}

// run executes a docker command and returns the output.
func (d *DockerClient) run(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.binary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	d.logger.Debug().
		Str("command", d.binary).
		Strs("args", args).
		Msg("executing docker command")

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = stdout.String()
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
