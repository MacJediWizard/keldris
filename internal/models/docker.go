package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ContainerHealthStatus represents the health status of a Docker container.
type ContainerHealthStatus string

const (
	// ContainerHealthStatusHealthy indicates the container is running and healthy.
	ContainerHealthStatusHealthy ContainerHealthStatus = "healthy"
	// ContainerHealthStatusUnhealthy indicates the container health check is failing.
	ContainerHealthStatusUnhealthy ContainerHealthStatus = "unhealthy"
	// ContainerHealthStatusStarting indicates the container is starting up.
	ContainerHealthStatusStarting ContainerHealthStatus = "starting"
	// ContainerHealthStatusNone indicates no health check is configured.
	ContainerHealthStatusNone ContainerHealthStatus = "none"
)

// ContainerState represents the running state of a Docker container.
type ContainerState string

const (
	// ContainerStateRunning indicates the container is running.
	ContainerStateRunning ContainerState = "running"
	// ContainerStateStopped indicates the container has stopped.
	ContainerStateStopped ContainerState = "stopped"
	// ContainerStateRestarting indicates the container is restarting.
	ContainerStateRestarting ContainerState = "restarting"
	// ContainerStatePaused indicates the container is paused.
	ContainerStatePaused ContainerState = "paused"
	// ContainerStateExited indicates the container has exited.
	ContainerStateExited ContainerState = "exited"
	// ContainerStateDead indicates the container is in a dead state.
	ContainerStateDead ContainerState = "dead"
)

// ContainerInfo represents information about a Docker container on an agent.
type ContainerInfo struct {
	ContainerID   string                `json:"container_id"`
	Name          string                `json:"name"`
	Image         string                `json:"image"`
	State         ContainerState        `json:"state"`
	HealthStatus  ContainerHealthStatus `json:"health_status"`
	RestartCount  int                   `json:"restart_count"`
	StartedAt     *time.Time            `json:"started_at,omitempty"`
	FinishedAt    *time.Time            `json:"finished_at,omitempty"`
	ExitCode      *int                  `json:"exit_code,omitempty"`
	CPUPercent    float64               `json:"cpu_percent"`
	MemoryUsage   int64                 `json:"memory_usage"`
	MemoryLimit   int64                 `json:"memory_limit"`
	MemoryPercent float64               `json:"memory_percent"`
	NetworkRxBytes int64                `json:"network_rx_bytes"`
	NetworkTxBytes int64                `json:"network_tx_bytes"`
	Labels        map[string]string     `json:"labels,omitempty"`
}

// VolumeInfo represents information about a Docker volume.
type VolumeInfo struct {
	Name       string    `json:"name"`
	Driver     string    `json:"driver"`
	Mountpoint string    `json:"mountpoint"`
	UsedBytes  int64     `json:"used_bytes"`
	TotalBytes int64     `json:"total_bytes"`
	UsagePercent float64 `json:"usage_percent"`
	CreatedAt  time.Time `json:"created_at"`
}

// DockerHealth represents the Docker health status for an agent.
type DockerHealth struct {
	Available        bool              `json:"available"`
	DaemonVersion    string            `json:"daemon_version,omitempty"`
	DaemonAPIVersion string            `json:"daemon_api_version,omitempty"`
	ContainerCount   int               `json:"container_count"`
	RunningCount     int               `json:"running_count"`
	StoppedCount     int               `json:"stopped_count"`
	UnhealthyCount   int               `json:"unhealthy_count"`
	RestartingCount  int               `json:"restarting_count"`
	Containers       []ContainerInfo   `json:"containers,omitempty"`
	Volumes          []VolumeInfo      `json:"volumes,omitempty"`
	LastChecked      time.Time         `json:"last_checked"`
}

// NewDockerHealth creates a new DockerHealth instance.
func NewDockerHealth() *DockerHealth {
	return &DockerHealth{
		Available:   false,
		Containers:  make([]ContainerInfo, 0),
		Volumes:     make([]VolumeInfo, 0),
		LastChecked: time.Now(),
	}
}

// CalculateCounts updates the container counts based on container info.
func (d *DockerHealth) CalculateCounts() {
	d.ContainerCount = len(d.Containers)
	d.RunningCount = 0
	d.StoppedCount = 0
	d.UnhealthyCount = 0
	d.RestartingCount = 0

	for _, c := range d.Containers {
		switch c.State {
		case ContainerStateRunning:
			d.RunningCount++
		case ContainerStateStopped, ContainerStateExited:
			d.StoppedCount++
		case ContainerStateRestarting:
			d.RestartingCount++
		}
		if c.HealthStatus == ContainerHealthStatusUnhealthy {
			d.UnhealthyCount++
		}
	}
}

// GetUnhealthyContainers returns containers that are unhealthy.
func (d *DockerHealth) GetUnhealthyContainers() []ContainerInfo {
	var unhealthy []ContainerInfo
	for _, c := range d.Containers {
		if c.HealthStatus == ContainerHealthStatusUnhealthy {
			unhealthy = append(unhealthy, c)
		}
	}
	return unhealthy
}

// GetRestartingContainers returns containers that are in a restart loop.
func (d *DockerHealth) GetRestartingContainers() []ContainerInfo {
	var restarting []ContainerInfo
	for _, c := range d.Containers {
		if c.State == ContainerStateRestarting {
			restarting = append(restarting, c)
		}
	}
	return restarting
}

// GetHighRestartContainers returns containers with restart count above threshold.
func (d *DockerHealth) GetHighRestartContainers(threshold int) []ContainerInfo {
	var highRestart []ContainerInfo
	for _, c := range d.Containers {
		if c.RestartCount >= threshold {
			highRestart = append(highRestart, c)
		}
	}
	return highRestart
}

// GetLowSpaceVolumes returns volumes with usage above threshold percent.
func (d *DockerHealth) GetLowSpaceVolumes(thresholdPercent float64) []VolumeInfo {
	var lowSpace []VolumeInfo
	for _, v := range d.Volumes {
		if v.UsagePercent >= thresholdPercent {
			lowSpace = append(lowSpace, v)
		}
	}
	return lowSpace
}

// AgentDockerHealth extends Agent with Docker-specific health information.
type AgentDockerHealth struct {
	ID               uuid.UUID      `json:"id"`
	OrgID            uuid.UUID      `json:"org_id"`
	AgentID          uuid.UUID      `json:"agent_id"`
	DockerHealth     *DockerHealth  `json:"docker_health,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// NewAgentDockerHealth creates a new AgentDockerHealth instance.
func NewAgentDockerHealth(orgID, agentID uuid.UUID) *AgentDockerHealth {
	now := time.Now()
	return &AgentDockerHealth{
		ID:           uuid.New(),
		OrgID:        orgID,
		AgentID:      agentID,
		DockerHealth: NewDockerHealth(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// SetDockerHealth sets the Docker health from JSON bytes.
func (a *AgentDockerHealth) SetDockerHealth(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var health DockerHealth
	if err := json.Unmarshal(data, &health); err != nil {
		return err
	}
	a.DockerHealth = &health
	return nil
}

// DockerHealthJSON returns the Docker health as JSON bytes for database storage.
func (a *AgentDockerHealth) DockerHealthJSON() ([]byte, error) {
	if a.DockerHealth == nil {
		return nil, nil
	}
	return json.Marshal(a.DockerHealth)
}

// ContainerRestartEvent represents a container restart event for tracking restart loops.
type ContainerRestartEvent struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	AgentID      uuid.UUID `json:"agent_id"`
	ContainerID  string    `json:"container_id"`
	ContainerName string   `json:"container_name"`
	RestartCount int       `json:"restart_count"`
	ExitCode     *int      `json:"exit_code,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// NewContainerRestartEvent creates a new ContainerRestartEvent.
func NewContainerRestartEvent(orgID, agentID uuid.UUID, containerID, containerName string, restartCount int) *ContainerRestartEvent {
	now := time.Now()
	return &ContainerRestartEvent{
		ID:            uuid.New(),
		OrgID:         orgID,
		AgentID:       agentID,
		ContainerID:   containerID,
		ContainerName: containerName,
		RestartCount:  restartCount,
		OccurredAt:    now,
		CreatedAt:     now,
	}
}

// DockerHealthSummary provides a fleet-wide summary of Docker health.
type DockerHealthSummary struct {
	TotalAgentsWithDocker  int `json:"total_agents_with_docker"`
	AgentsDockerHealthy    int `json:"agents_docker_healthy"`
	AgentsDockerUnhealthy  int `json:"agents_docker_unhealthy"`
	AgentsDockerUnavailable int `json:"agents_docker_unavailable"`
	TotalContainers        int `json:"total_containers"`
	RunningContainers      int `json:"running_containers"`
	StoppedContainers      int `json:"stopped_containers"`
	UnhealthyContainers    int `json:"unhealthy_containers"`
	RestartingContainers   int `json:"restarting_containers"`
	HighRestartContainers  int `json:"high_restart_containers"`
	LowSpaceVolumes        int `json:"low_space_volumes"`
}

// DockerDashboardWidget contains data for the Docker health dashboard widget.
type DockerDashboardWidget struct {
	Summary           *DockerHealthSummary    `json:"summary"`
	RecentRestarts    []ContainerRestartEvent `json:"recent_restarts,omitempty"`
	UnhealthyAgents   []uuid.UUID             `json:"unhealthy_agents,omitempty"`
	VolumeAlerts      []VolumeInfo            `json:"volume_alerts,omitempty"`
}

// DockerMonitoringConfig holds configuration for Docker monitoring.
type DockerMonitoringConfig struct {
	// RestartThreshold is the number of restarts before alerting.
	RestartThreshold int `json:"restart_threshold"`
	// RestartWindowMinutes is the time window for counting restarts.
	RestartWindowMinutes int `json:"restart_window_minutes"`
	// VolumeUsageThresholdPercent is the volume usage percent to alert on.
	VolumeUsageThresholdPercent float64 `json:"volume_usage_threshold_percent"`
	// ContainerMemoryThresholdPercent is the container memory usage to alert on.
	ContainerMemoryThresholdPercent float64 `json:"container_memory_threshold_percent"`
	// ContainerCPUThresholdPercent is the container CPU usage to alert on.
	ContainerCPUThresholdPercent float64 `json:"container_cpu_threshold_percent"`
}

// DefaultDockerMonitoringConfig returns default Docker monitoring configuration.
func DefaultDockerMonitoringConfig() DockerMonitoringConfig {
	return DockerMonitoringConfig{
		RestartThreshold:               5,
		RestartWindowMinutes:           60,
		VolumeUsageThresholdPercent:    85.0,
		ContainerMemoryThresholdPercent: 90.0,
		ContainerCPUThresholdPercent:    90.0,
	}
}
