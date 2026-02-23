package models

import (
	"time"

	"github.com/google/uuid"
)

// AgentHealthHistory represents a historical health record for an agent.
type AgentHealthHistory struct {
	ID               uuid.UUID    `json:"id"`
	AgentID          uuid.UUID    `json:"agent_id"`
	OrgID            uuid.UUID    `json:"org_id"`
	HealthStatus     HealthStatus `json:"health_status"`
	CPUUsage         float64      `json:"cpu_usage"`
	MemoryUsage      float64      `json:"memory_usage"`
	DiskUsage        float64      `json:"disk_usage"`
	DiskFreeBytes    int64        `json:"disk_free_bytes"`
	DiskTotalBytes   int64        `json:"disk_total_bytes"`
	NetworkUp        bool         `json:"network_up"`
	ResticVersion    string       `json:"restic_version,omitempty"`
	ResticAvailable  bool         `json:"restic_available"`
	Issues           []HealthIssue `json:"issues,omitempty"`
	RecordedAt       time.Time    `json:"recorded_at"`
	CreatedAt        time.Time    `json:"created_at"`
}

// NewAgentHealthHistory creates a new AgentHealthHistory record.
func NewAgentHealthHistory(agentID, orgID uuid.UUID, healthStatus HealthStatus, metrics *HealthMetrics, issues []HealthIssue) *AgentHealthHistory {
	now := time.Now()
	h := &AgentHealthHistory{
		ID:           uuid.New(),
		AgentID:      agentID,
		OrgID:        orgID,
		HealthStatus: healthStatus,
		Issues:       issues,
		RecordedAt:   now,
		CreatedAt:    now,
	}
	if metrics != nil {
		h.CPUUsage = metrics.CPUUsage
		h.MemoryUsage = metrics.MemoryUsage
		h.DiskUsage = metrics.DiskUsage
		h.DiskFreeBytes = metrics.DiskFreeBytes
		h.DiskTotalBytes = metrics.DiskTotalBytes
		h.NetworkUp = metrics.NetworkUp
		h.ResticVersion = metrics.ResticVersion
		h.ResticAvailable = metrics.ResticAvailable
	}
	return h
}

// FleetHealthSummary contains aggregated health stats for all agents in an organization.
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

// AgentStats contains aggregated statistics for an agent.
type AgentStats struct {
	AgentID          uuid.UUID  `json:"agent_id"`
	TotalBackups     int        `json:"total_backups"`
	SuccessfulBackups int       `json:"successful_backups"`
	FailedBackups    int        `json:"failed_backups"`
	TotalSizeBytes   int64      `json:"total_size_bytes"`
	LastBackupAt     *time.Time `json:"last_backup_at,omitempty"`
	SuccessRate      float64    `json:"success_rate"`
	ScheduleCount    int        `json:"schedule_count"`
}

// AgentStatsResponse is the API response for agent statistics.
type AgentStatsResponse struct {
	Agent *Agent      `json:"agent"`
	Stats *AgentStats `json:"stats"`
}
