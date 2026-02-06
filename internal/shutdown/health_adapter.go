package shutdown

import (
	"time"
)

// HealthStatus is the status structure expected by the health handler.
type HealthStatus struct {
	State             string        `json:"state"`
	StartedAt         *time.Time    `json:"started_at,omitempty"`
	TimeRemaining     time.Duration `json:"time_remaining,omitempty"`
	RunningBackups    int           `json:"running_backups"`
	CheckpointedCount int           `json:"checkpointed_count"`
	AcceptingNewJobs  bool          `json:"accepting_new_jobs"`
	Message           string        `json:"message,omitempty"`
}

// HealthAdapter adapts the shutdown Manager to the ShutdownStatusProvider interface
// expected by the health handler.
type HealthAdapter struct {
	manager *Manager
}

// NewHealthAdapter creates a new health adapter for the shutdown manager.
func NewHealthAdapter(manager *Manager) *HealthAdapter {
	return &HealthAdapter{manager: manager}
}

// GetStatus returns the current shutdown status in the format expected by the health handler.
func (a *HealthAdapter) GetStatus() HealthStatus {
	status := a.manager.GetStatus()
	return HealthStatus{
		State:             string(status.State),
		StartedAt:         status.StartedAt,
		TimeRemaining:     status.TimeRemaining,
		RunningBackups:    status.RunningBackups,
		CheckpointedCount: status.CheckpointedCount,
		AcceptingNewJobs:  status.AcceptingNewJobs,
		Message:           status.Message,
	}
}

// IsAcceptingJobs returns true if the server is accepting new backup jobs.
func (a *HealthAdapter) IsAcceptingJobs() bool {
	return a.manager.IsAcceptingJobs()
}
