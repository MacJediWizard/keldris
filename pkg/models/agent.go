package models

import "time"

// AgentInfo contains lightweight agent information for agent-server communication.
type AgentInfo struct {
	ID       string  `json:"id"`
	Hostname string  `json:"hostname"`
	Status   string  `json:"status"`
	OSInfo   *OSInfo `json:"os_info,omitempty"`
}

// OSInfo contains operating system information from the agent.
type OSInfo struct {
	OS       string `json:"os" example:"linux"`
	Arch     string `json:"arch" example:"amd64"`
	Hostname string `json:"hostname" example:"backup-server-01"`
	Version  string `json:"version,omitempty" example:"Ubuntu 22.04"`
}

// HeartbeatRequest is the request body for agent health reporting.
type HeartbeatRequest struct {
	Status  string            `json:"status" binding:"required,oneof=healthy unhealthy degraded"`
	OSInfo  *OSInfo           `json:"os_info,omitempty"`
	Metrics *HeartbeatMetrics `json:"metrics,omitempty"`
}

// HeartbeatMetrics contains system metrics reported by the agent.
type HeartbeatMetrics struct {
	CPUUsage        float64 `json:"cpu_usage,omitempty"`
	MemoryUsage     float64 `json:"memory_usage,omitempty"`
	DiskUsage       float64 `json:"disk_usage,omitempty"`
	DiskFreeBytes   int64   `json:"disk_free_bytes,omitempty"`
	DiskTotalBytes  int64   `json:"disk_total_bytes,omitempty"`
	NetworkUp       bool    `json:"network_up"`
	UptimeSeconds   int64   `json:"uptime_seconds,omitempty"`
	ResticVersion   string  `json:"restic_version,omitempty"`
	ResticAvailable bool    `json:"restic_available,omitempty"`
}

// HeartbeatResponse is the server response to a heartbeat request.
type HeartbeatResponse struct {
	Acknowledged bool      `json:"acknowledged"`
	ServerTime   time.Time `json:"server_time"`
	AgentID      string    `json:"agent_id"`
}
