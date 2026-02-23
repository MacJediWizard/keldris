package models

import (
	"encoding/json"
	"time"

	pkgmodels "github.com/MacJediWizard/keldris/pkg/models"
	"github.com/google/uuid"
)

// OSInfo is a type alias for the shared OSInfo type in pkg/models.
type OSInfo = pkgmodels.OSInfo

// AgentStatus represents the current status of an agent.
type AgentStatus string

const (
	// AgentStatusPending indicates the agent is registered but not yet active.
	AgentStatusPending AgentStatus = "pending"
	// AgentStatusActive indicates the agent is active and communicating.
	AgentStatusActive AgentStatus = "active"
	// AgentStatusOffline indicates the agent has not communicated recently.
	AgentStatusOffline AgentStatus = "offline"
	// AgentStatusDisabled indicates the agent has been manually disabled.
	AgentStatusDisabled AgentStatus = "disabled"
)

// HealthStatus represents the overall health status of an agent.
type HealthStatus string

const (
	// HealthStatusHealthy indicates all metrics are within acceptable ranges.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusWarning indicates some metrics are concerning but not critical.
	HealthStatusWarning HealthStatus = "warning"
	// HealthStatusCritical indicates immediate attention is required.
	HealthStatusCritical HealthStatus = "critical"
	// HealthStatusUnknown indicates health cannot be determined.
	HealthStatusUnknown HealthStatus = "unknown"
)

// HealthMetrics contains system metrics from an agent.
type HealthMetrics struct {
	CPUUsage        float64       `json:"cpu_usage"`
	MemoryUsage     float64       `json:"memory_usage"`
	DiskUsage       float64       `json:"disk_usage"`
	DiskFreeBytes   int64         `json:"disk_free_bytes"`
	DiskTotalBytes  int64         `json:"disk_total_bytes"`
	NetworkUp       bool          `json:"network_up"`
	UptimeSeconds   int64         `json:"uptime_seconds"`
	ResticVersion   string        `json:"restic_version,omitempty"`
	ResticAvailable bool          `json:"restic_available"`
	PiholeInfo      *PiholeInfo   `json:"pihole_info,omitempty"`
	Issues          []HealthIssue `json:"issues,omitempty"`
}

// PiholeInfo contains Pi-hole detection and version information.
type PiholeInfo struct {
	Installed       bool   `json:"installed"`
	Version         string `json:"version,omitempty"`
	FTLVersion      string `json:"ftl_version,omitempty"`
	WebVersion      string `json:"web_version,omitempty"`
	ConfigDir       string `json:"config_dir,omitempty"`
	BlockingEnabled bool   `json:"blocking_enabled"`
}

// HealthIssue represents a specific health issue detected on an agent.
type HealthIssue struct {
	Component string       `json:"component"` // disk, memory, cpu, network, restic, heartbeat
	Severity  HealthStatus `json:"severity"`
	Message   string       `json:"message"`
	Value     float64      `json:"value,omitempty"`
	Threshold float64      `json:"threshold,omitempty"`
}

// DockerContainerInfo represents basic Docker container information.
type DockerContainerInfo struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
	State  string `json:"state"` // running, paused, exited, etc.
}

// DockerVolumeInfo represents basic Docker volume information.
type DockerVolumeInfo struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint,omitempty"`
}

// DockerInfo contains information about Docker on an agent.
type DockerInfo struct {
	Available       bool                  `json:"available"`
	Version         string                `json:"version,omitempty"`
	APIVersion      string                `json:"api_version,omitempty"`
	ContainerCount  int                   `json:"container_count"`
	RunningCount    int                   `json:"running_count"`
	VolumeCount     int                   `json:"volume_count"`
	Containers      []DockerContainerInfo `json:"containers,omitempty"`
	Volumes         []DockerVolumeInfo    `json:"volumes,omitempty"`
	Error           string                `json:"error,omitempty"`
	DetectedAt      *time.Time            `json:"detected_at,omitempty"`
}

// Agent represents an agent registered in the system.
type Agent struct {
	ID                 uuid.UUID        `json:"id"`
	OrgID              uuid.UUID        `json:"org_id"`
	Hostname           string           `json:"hostname"`
	APIKeyHash         string           `json:"-"`
	OSInfo             *OSInfo          `json:"os_info,omitempty"`
	NetworkMounts      []NetworkMount   `json:"network_mounts,omitempty"`
	LastSeen           *time.Time       `json:"last_seen,omitempty"`
	Status             AgentStatus      `json:"status"`
	HealthStatus       HealthStatus     `json:"health_status"`
	HealthMetrics      *HealthMetrics   `json:"health_metrics,omitempty"`
	HealthCheckedAt    *time.Time       `json:"health_checked_at,omitempty"`
	DebugMode          bool             `json:"debug_mode"`
	DebugModeExpiresAt *time.Time       `json:"debug_mode_expires_at,omitempty"`
	DebugModeEnabledAt *time.Time       `json:"debug_mode_enabled_at,omitempty"`
	DebugModeEnabledBy *uuid.UUID       `json:"debug_mode_enabled_by,omitempty"`
	Metadata              map[string]any   `json:"metadata,omitempty"`
	MaxConcurrentBackups  *int             `json:"max_concurrent_backups,omitempty"`
	ProxmoxInfo           *ProxmoxInfo     `json:"proxmox_info,omitempty"`
	CreatedAt             time.Time        `json:"created_at"`
	UpdatedAt             time.Time        `json:"updated_at"`
}

// NewAgent creates a new Agent with the given org, hostname, and API key hash.
func NewAgent(orgID uuid.UUID, hostname, apiKeyHash string) *Agent {
	now := time.Now()
	return &Agent{
		ID:           uuid.New(),
		OrgID:        orgID,
		Hostname:     hostname,
		APIKeyHash:   apiKeyHash,
		Status:       AgentStatusPending,
		HealthStatus: HealthStatusUnknown,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsOnline returns true if the agent has been seen within the given threshold.
func (a *Agent) IsOnline(threshold time.Duration) bool {
	if a.LastSeen == nil {
		return false
	}
	return time.Since(*a.LastSeen) < threshold
}

// MarkSeen updates the agent's last seen time and sets it to active.
func (a *Agent) MarkSeen() {
	now := time.Now()
	a.LastSeen = &now
	a.Status = AgentStatusActive
}

// SetOSInfo sets the OS info from JSON bytes.
func (a *Agent) SetOSInfo(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var info OSInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}
	a.OSInfo = &info
	return nil
}

// OSInfoJSON returns the OS info as JSON bytes for database storage.
func (a *Agent) OSInfoJSON() ([]byte, error) {
	if a.OSInfo == nil {
		return nil, nil
	}
	return json.Marshal(a.OSInfo)
}

// SetNetworkMounts sets the network mounts from JSON bytes.
func (a *Agent) SetNetworkMounts(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &a.NetworkMounts)
}

// NetworkMountsJSON returns the network mounts as JSON bytes for database storage.
func (a *Agent) NetworkMountsJSON() ([]byte, error) {
	if a.NetworkMounts == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(a.NetworkMounts)
}

// GetMountByPath returns the network mount matching the given path, or nil.
func (a *Agent) GetMountByPath(path string) *NetworkMount {
	for i := range a.NetworkMounts {
		if a.NetworkMounts[i].Path == path {
			return &a.NetworkMounts[i]
		}
	}
	return nil
}

// GetConnectedMounts returns all network mounts with connected status.
func (a *Agent) GetConnectedMounts() []NetworkMount {
	var connected []NetworkMount
	for _, m := range a.NetworkMounts {
		if m.Status == MountStatusConnected {
			connected = append(connected, m)
		}
	}
	return connected
}

// SetHealthMetrics sets the health metrics from JSON bytes.
func (a *Agent) SetHealthMetrics(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var metrics HealthMetrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		return err
	}
	a.HealthMetrics = &metrics
	return nil
}

// HealthMetricsJSON returns the health metrics as JSON bytes for database storage.
func (a *Agent) HealthMetricsJSON() ([]byte, error) {
	if a.HealthMetrics == nil {
		return nil, nil
	}
	return json.Marshal(a.HealthMetrics)
}

// SetMetadata sets the metadata from JSON bytes.
func (a *Agent) SetMetadata(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &a.Metadata)
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (a *Agent) MetadataJSON() ([]byte, error) {
	if a.Metadata == nil {
		return nil, nil
	}
	return json.Marshal(a.Metadata)
}

// SetDebugModeRequest is the request body for setting agent debug mode.
type SetDebugModeRequest struct {
	Enabled       bool `json:"enabled"`
	DurationHours int  `json:"duration_hours,omitempty"`
}

// SetDebugModeResponse is the response for setting agent debug mode.
type SetDebugModeResponse struct {
	DebugMode          bool       `json:"debug_mode"`
	DebugModeExpiresAt *time.Time `json:"debug_mode_expires_at,omitempty"`
	Message            string     `json:"message"`
}