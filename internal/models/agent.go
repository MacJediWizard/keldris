package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

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

// OSInfo contains operating system information from the agent.
type OSInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	Version  string `json:"version,omitempty"`
}

// Agent represents a backup agent installed on a host.
type Agent struct {
	ID         uuid.UUID   `json:"id"`
	OrgID      uuid.UUID   `json:"org_id"`
	Hostname   string      `json:"hostname"`
	APIKeyHash string      `json:"-"` // Never expose in JSON
	OSInfo     *OSInfo     `json:"os_info,omitempty"`
	LastSeen   *time.Time  `json:"last_seen,omitempty"`
	Status     AgentStatus `json:"status"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
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

// AgentStats contains aggregated statistics for an agent.
type AgentStats struct {
	AgentID          uuid.UUID  `json:"agent_id"`
	TotalBackups     int        `json:"total_backups"`
	SuccessfulBackups int       `json:"successful_backups"`
	FailedBackups    int        `json:"failed_backups"`
	SuccessRate      float64    `json:"success_rate"`
	TotalSizeBytes   int64      `json:"total_size_bytes"`
	LastBackupAt     *time.Time `json:"last_backup_at,omitempty"`
	NextScheduledAt  *time.Time `json:"next_scheduled_at,omitempty"`
	ScheduleCount    int        `json:"schedule_count"`
	Uptime           *string    `json:"uptime,omitempty"`
}

// AgentStatsResponse is the response for the agent stats endpoint.
type AgentStatsResponse struct {
	Agent *Agent      `json:"agent"`
	Stats *AgentStats `json:"stats"`
}

// AgentEvent represents an event in the agent's history.
type AgentEvent struct {
	ID          uuid.UUID  `json:"id"`
	AgentID     uuid.UUID  `json:"agent_id"`
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Metadata    string     `json:"metadata,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
