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

// Agent represents a backup agent installed on a host.
type Agent struct {
	ID                   uuid.UUID              `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	OrgID                uuid.UUID              `json:"org_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	Hostname             string                 `json:"hostname" example:"backup-server-01"`
	APIKeyHash           string                 `json:"-"` // Never expose in JSON
	OSInfo               *OSInfo                `json:"os_info,omitempty"`
	DockerInfo           *DockerInfo            `json:"docker_info,omitempty"`
	NetworkMounts        []NetworkMount         `json:"network_mounts,omitempty"`
	LastSeen             *time.Time             `json:"last_seen,omitempty"`
	Status               AgentStatus            `json:"status" example:"active"`
	HealthStatus         HealthStatus           `json:"health_status" example:"healthy"`
	HealthMetrics        *HealthMetrics         `json:"health_metrics,omitempty"`
	HealthCheckedAt      *time.Time             `json:"health_checked_at,omitempty"`
	DebugMode            bool                   `json:"debug_mode" example:"false"`
	DebugModeExpiresAt   *time.Time             `json:"debug_mode_expires_at,omitempty"`
	DebugModeEnabledAt   *time.Time             `json:"debug_mode_enabled_at,omitempty"`
	DebugModeEnabledBy   *uuid.UUID             `json:"debug_mode_enabled_by,omitempty"`
	MaxConcurrentBackups *int                   `json:"max_concurrent_backups,omitempty"` // nil means use org default
	Metadata             map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// NewAgent creates a new Agent with the given details.
func NewAgent(orgID uuid.UUID, hostname, apiKeyHash string) *Agent {
	now := time.Now()
	return &Agent{
		ID:         uuid.New(),
		OrgID:      orgID,
		Hostname:   hostname,
		APIKeyHash: apiKeyHash,
		Status:     AgentStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// SetOSInfo sets the OS information from JSON bytes.
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

// SetDockerInfo sets the Docker information from JSON bytes.
func (a *Agent) SetDockerInfo(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var info DockerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return err
	}
	a.DockerInfo = &info
	return nil
}

// DockerInfoJSON returns the Docker info as JSON bytes for database storage.
func (a *Agent) DockerInfoJSON() ([]byte, error) {
	if a.DockerInfo == nil {
		return nil, nil
	}
	return json.Marshal(a.DockerInfo)
}

// HasDocker returns true if Docker is available on this agent.
func (a *Agent) HasDocker() bool {
	return a.DockerInfo != nil && a.DockerInfo.Available
}

// IsOnline returns true if the agent has been seen within the threshold.
func (a *Agent) IsOnline(threshold time.Duration) bool {
	if a.LastSeen == nil {
		return false
	}
	return time.Since(*a.LastSeen) < threshold
}

// MarkSeen updates the agent's last seen time and sets status to active.
func (a *Agent) MarkSeen() {
	now := time.Now()
	a.LastSeen = &now
	a.Status = AgentStatusActive
	a.UpdatedAt = now
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


// AgentStats contains aggregated statistics for an agent.
type AgentStats struct {
	AgentID           uuid.UUID  `json:"agent_id"`
	TotalBackups      int        `json:"total_backups"`
	SuccessfulBackups int        `json:"successful_backups"`
	FailedBackups     int        `json:"failed_backups"`
	SuccessRate       float64    `json:"success_rate"`
	TotalSizeBytes    int64      `json:"total_size_bytes"`
	LastBackupAt      *time.Time `json:"last_backup_at,omitempty"`
	NextScheduledAt   *time.Time `json:"next_scheduled_at,omitempty"`
	ScheduleCount     int        `json:"schedule_count"`
	Uptime            *string    `json:"uptime,omitempty"`
}

// AgentStatsResponse is the response for the agent stats endpoint.
type AgentStatsResponse struct {
	Agent *Agent      `json:"agent"`
	Stats *AgentStats `json:"stats"`
}

// AgentEvent represents an event in the agent's history.
type AgentEvent struct {
	ID          uuid.UUID `json:"id"`
	AgentID     uuid.UUID `json:"agent_id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Metadata    string    `json:"metadata,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// AgentHealthHistory represents a health metrics record in the history.
type AgentHealthHistory struct {
	ID              uuid.UUID    `json:"id"`
	AgentID         uuid.UUID    `json:"agent_id"`
	OrgID           uuid.UUID    `json:"org_id"`
	HealthStatus    HealthStatus `json:"health_status"`
	CPUUsage        *float64     `json:"cpu_usage,omitempty"`
	MemoryUsage     *float64     `json:"memory_usage,omitempty"`
	DiskUsage       *float64     `json:"disk_usage,omitempty"`
	DiskFreeBytes   *int64       `json:"disk_free_bytes,omitempty"`
	DiskTotalBytes  *int64       `json:"disk_total_bytes,omitempty"`
	NetworkUp       bool         `json:"network_up"`
	ResticVersion   string       `json:"restic_version,omitempty"`
	ResticAvailable bool         `json:"restic_available"`
	Issues          []HealthIssue `json:"issues,omitempty"`
	RecordedAt      time.Time    `json:"recorded_at"`
	CreatedAt       time.Time    `json:"created_at"`
}

// NewAgentHealthHistory creates a new health history record.
func NewAgentHealthHistory(agentID, orgID uuid.UUID, status HealthStatus, metrics *HealthMetrics, issues []HealthIssue) *AgentHealthHistory {
	now := time.Now()
	h := &AgentHealthHistory{
		ID:           uuid.New(),
		AgentID:      agentID,
		OrgID:        orgID,
		HealthStatus: status,
		NetworkUp:    true,
		Issues:       issues,
		RecordedAt:   now,
		CreatedAt:    now,
	}

	if metrics != nil {
		h.CPUUsage = &metrics.CPUUsage
		h.MemoryUsage = &metrics.MemoryUsage
		h.DiskUsage = &metrics.DiskUsage
		h.DiskFreeBytes = &metrics.DiskFreeBytes
		h.DiskTotalBytes = &metrics.DiskTotalBytes
		h.NetworkUp = metrics.NetworkUp
		h.ResticVersion = metrics.ResticVersion
		h.ResticAvailable = metrics.ResticAvailable
	}

	return h
}

// FleetHealthSummary contains aggregated health stats for all agents in an org.
type FleetHealthSummary struct {
	TotalAgents    int     `json:"total_agents"`
	HealthyCount   int     `json:"healthy_count"`
	WarningCount   int     `json:"warning_count"`
	CriticalCount  int     `json:"critical_count"`
	UnknownCount   int     `json:"unknown_count"`
	ActiveCount    int     `json:"active_count"`
	OfflineCount   int     `json:"offline_count"`
	AvgCPUUsage    float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage float64 `json:"avg_memory_usage"`
	AvgDiskUsage   float64 `json:"avg_disk_usage"`
}

// SetHealthMetrics sets health metrics from JSON bytes.
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

// SetDebugMode enables debug mode on the agent with an expiration time.
func (a *Agent) SetDebugMode(enabled bool, expiresAt *time.Time, enabledBy *uuid.UUID) {
	now := time.Now()
	a.DebugMode = enabled
	a.UpdatedAt = now
	if enabled {
		a.DebugModeEnabledAt = &now
		a.DebugModeExpiresAt = expiresAt
		a.DebugModeEnabledBy = enabledBy
	} else {
		a.DebugModeEnabledAt = nil
		a.DebugModeExpiresAt = nil
		a.DebugModeEnabledBy = nil
	}
}

// IsDebugModeExpired returns true if debug mode has expired.
func (a *Agent) IsDebugModeExpired() bool {
	if !a.DebugMode || a.DebugModeExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.DebugModeExpiresAt)
}

// SetMetadata sets the metadata from JSON bytes.
func (a *Agent) SetMetadata(data []byte) error {
	if len(data) == 0 {
		a.Metadata = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(data, &a.Metadata)
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (a *Agent) MetadataJSON() ([]byte, error) {
	if a.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(a.Metadata)
}

// SetDebugModeRequest is the request body for enabling/disabling debug mode.
type SetDebugModeRequest struct {
	Enabled      bool `json:"enabled" binding:"required" example:"true"`
	DurationHours int  `json:"duration_hours,omitempty" example:"4"` // 0 means no auto-disable
}

// SetDebugModeResponse is the response for the debug mode endpoint.
type SetDebugModeResponse struct {
	DebugMode          bool       `json:"debug_mode" example:"true"`
	DebugModeExpiresAt *time.Time `json:"debug_mode_expires_at,omitempty"`
	Message            string     `json:"message" example:"Debug mode enabled for 4 hours"`
}

// HeartbeatResponse is the response for agent heartbeat with debug mode info.
type HeartbeatResponse struct {
	*Agent
	DebugConfig *DebugConfig `json:"debug_config,omitempty"`
}

// DebugConfig contains debug mode configuration for the agent.
type DebugConfig struct {
	Enabled             bool   `json:"enabled" example:"true"`
	LogLevel            string `json:"log_level" example:"debug"`
	IncludeResticOutput bool   `json:"include_restic_output" example:"true"`
	LogFileOperations   bool   `json:"log_file_operations" example:"true"`
}
