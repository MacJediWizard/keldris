package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	// LogLevelDebug indicates debug-level logging.
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo indicates informational logging.
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn indicates warning-level logging.
	LogLevelWarn LogLevel = "warn"
	// LogLevelError indicates error-level logging.
	LogLevelError LogLevel = "error"
)

// AgentLog represents a log entry from an agent.
type AgentLog struct {
	ID        uuid.UUID  `json:"id"`
	AgentID   uuid.UUID  `json:"agent_id"`
	OrgID     uuid.UUID  `json:"org_id"`
	Level     LogLevel   `json:"level"`
	Message   string     `json:"message"`
	Component string     `json:"component,omitempty"`
	Metadata  Metadata   `json:"metadata,omitempty"`
	Timestamp time.Time  `json:"timestamp"`
	CreatedAt time.Time  `json:"created_at"`
}

// Metadata represents arbitrary log metadata.
type Metadata map[string]any

// NewAgentLog creates a new AgentLog with the given details.
func NewAgentLog(agentID, orgID uuid.UUID, level LogLevel, message string) *AgentLog {
	now := time.Now()
	return &AgentLog{
		ID:        uuid.New(),
		AgentID:   agentID,
		OrgID:     orgID,
		Level:     level,
		Message:   message,
		Timestamp: now,
		CreatedAt: now,
	}
}

// SetMetadata sets metadata from JSON bytes.
func (l *AgentLog) SetMetadata(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &l.Metadata)
}

// MetadataJSON returns metadata as JSON bytes for database storage.
func (l *AgentLog) MetadataJSON() ([]byte, error) {
	if l.Metadata == nil {
		return nil, nil
	}
	return json.Marshal(l.Metadata)
}

// AgentLogEntry is the format agents use to submit logs.
type AgentLogEntry struct {
	Level     LogLevel  `json:"level" binding:"required,oneof=debug info warn error"`
	Message   string    `json:"message" binding:"required"`
	Component string    `json:"component,omitempty"`
	Metadata  Metadata  `json:"metadata,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// AgentLogBatch is a batch of log entries submitted by an agent.
type AgentLogBatch struct {
	Logs []AgentLogEntry `json:"logs" binding:"required,min=1,max=100"`
}

// AgentLogsResponse is the response for fetching agent logs.
type AgentLogsResponse struct {
	Logs       []*AgentLog `json:"logs"`
	TotalCount int         `json:"total_count"`
	HasMore    bool        `json:"has_more"`
}

// AgentLogFilter represents filter options for querying agent logs.
type AgentLogFilter struct {
	Level     LogLevel  `json:"level,omitempty"`
	Component string    `json:"component,omitempty"`
	Search    string    `json:"search,omitempty"`
	Since     time.Time `json:"since,omitempty"`
	Until     time.Time `json:"until,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}
