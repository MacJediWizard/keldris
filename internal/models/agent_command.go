package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CommandType represents the type of command to execute on an agent.
type CommandType string

const (
	// CommandTypeBackupNow triggers an immediate backup.
	CommandTypeBackupNow CommandType = "backup_now"
	// CommandTypeUpdate triggers an agent update.
	CommandTypeUpdate CommandType = "update"
	// CommandTypeRestart triggers an agent restart.
	CommandTypeRestart CommandType = "restart"
	// CommandTypeDiagnostics triggers a diagnostics collection.
	CommandTypeDiagnostics CommandType = "diagnostics"
)

// CommandStatus represents the current status of a command.
type CommandStatus string

const (
	// CommandStatusPending indicates the command is waiting to be picked up.
	CommandStatusPending CommandStatus = "pending"
	// CommandStatusAcknowledged indicates the agent has received the command.
	CommandStatusAcknowledged CommandStatus = "acknowledged"
	// CommandStatusRunning indicates the command is currently executing.
	CommandStatusRunning CommandStatus = "running"
	// CommandStatusCompleted indicates the command finished successfully.
	CommandStatusCompleted CommandStatus = "completed"
	// CommandStatusFailed indicates the command failed.
	CommandStatusFailed CommandStatus = "failed"
	// CommandStatusTimedOut indicates the command timed out.
	CommandStatusTimedOut CommandStatus = "timed_out"
	// CommandStatusCanceled indicates the command was canceled.
	CommandStatusCanceled CommandStatus = "canceled"
)

// CommandPayload contains type-specific command parameters.
type CommandPayload struct {
	// For backup_now command
	ScheduleID *uuid.UUID `json:"schedule_id,omitempty"`
	// For update command
	TargetVersion string `json:"target_version,omitempty"`
	// For diagnostics command
	DiagnosticTypes []string `json:"diagnostic_types,omitempty"`
}

// CommandResult contains the result of a command execution.
type CommandResult struct {
	Output      string         `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
	Diagnostics map[string]any `json:"diagnostics,omitempty"`
	BackupID    *uuid.UUID     `json:"backup_id,omitempty"`
}

// AgentCommand represents a command queued for an agent.
type AgentCommand struct {
	ID             uuid.UUID       `json:"id"`
	AgentID        uuid.UUID       `json:"agent_id"`
	OrgID          uuid.UUID       `json:"org_id"`
	Type           CommandType     `json:"type"`
	Status         CommandStatus   `json:"status"`
	Payload        *CommandPayload `json:"payload,omitempty"`
	Result         *CommandResult  `json:"result,omitempty"`
	CreatedBy      *uuid.UUID      `json:"created_by,omitempty"`
	CreatedByName  string          `json:"created_by_name,omitempty"`
	AcknowledgedAt *time.Time      `json:"acknowledged_at,omitempty"`
	StartedAt      *time.Time      `json:"started_at,omitempty"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	TimeoutAt      time.Time       `json:"timeout_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// DefaultCommandTimeout is the default timeout for commands.
const DefaultCommandTimeout = 5 * time.Minute

// NewAgentCommand creates a new AgentCommand with the given details.
func NewAgentCommand(agentID, orgID uuid.UUID, cmdType CommandType, payload *CommandPayload, createdBy *uuid.UUID) *AgentCommand {
	now := time.Now()
	return &AgentCommand{
		ID:        uuid.New(),
		AgentID:   agentID,
		OrgID:     orgID,
		Type:      cmdType,
		Status:    CommandStatusPending,
		Payload:   payload,
		CreatedBy: createdBy,
		TimeoutAt: now.Add(DefaultCommandTimeout),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetPayload sets the payload from JSON bytes.
func (c *AgentCommand) SetPayload(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var payload CommandPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	c.Payload = &payload
	return nil
}

// PayloadJSON returns the payload as JSON bytes for database storage.
func (c *AgentCommand) PayloadJSON() ([]byte, error) {
	if c.Payload == nil {
		return nil, nil
	}
	return json.Marshal(c.Payload)
}

// SetResult sets the result from JSON bytes.
func (c *AgentCommand) SetResult(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var result CommandResult
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	c.Result = &result
	return nil
}

// ResultJSON returns the result as JSON bytes for database storage.
func (c *AgentCommand) ResultJSON() ([]byte, error) {
	if c.Result == nil {
		return nil, nil
	}
	return json.Marshal(c.Result)
}

// Acknowledge marks the command as acknowledged by the agent.
func (c *AgentCommand) Acknowledge() {
	now := time.Now()
	c.Status = CommandStatusAcknowledged
	c.AcknowledgedAt = &now
	c.UpdatedAt = now
}

// MarkRunning marks the command as running.
func (c *AgentCommand) MarkRunning() {
	now := time.Now()
	c.Status = CommandStatusRunning
	c.StartedAt = &now
	c.UpdatedAt = now
}

// Complete marks the command as completed with the given result.
func (c *AgentCommand) Complete(result *CommandResult) {
	now := time.Now()
	c.Status = CommandStatusCompleted
	c.Result = result
	c.CompletedAt = &now
	c.UpdatedAt = now
}

// Fail marks the command as failed with the given error.
func (c *AgentCommand) Fail(errorMsg string) {
	now := time.Now()
	c.Status = CommandStatusFailed
	c.Result = &CommandResult{Error: errorMsg}
	c.CompletedAt = &now
	c.UpdatedAt = now
}

// MarkTimedOut marks the command as timed out.
func (c *AgentCommand) MarkTimedOut() {
	now := time.Now()
	c.Status = CommandStatusTimedOut
	c.Result = &CommandResult{Error: "command timed out waiting for agent response"}
	c.CompletedAt = &now
	c.UpdatedAt = now
}

// Cancel marks the command as canceled.
func (c *AgentCommand) Cancel() {
	now := time.Now()
	c.Status = CommandStatusCanceled
	c.CompletedAt = &now
	c.UpdatedAt = now
}

// IsTerminal returns true if the command is in a terminal state.
func (c *AgentCommand) IsTerminal() bool {
	switch c.Status {
	case CommandStatusCompleted, CommandStatusFailed, CommandStatusTimedOut, CommandStatusCanceled:
		return true
	}
	return false
}

// IsPending returns true if the command is pending and can be picked up.
func (c *AgentCommand) IsPending() bool {
	return c.Status == CommandStatusPending
}

// AgentCommandResponse is the response format for commands sent to agents.
type AgentCommandResponse struct {
	ID        uuid.UUID       `json:"id"`
	Type      CommandType     `json:"type"`
	Payload   *CommandPayload `json:"payload,omitempty"`
	TimeoutAt time.Time       `json:"timeout_at"`
}

// ToResponse converts an AgentCommand to a response suitable for agents.
func (c *AgentCommand) ToResponse() *AgentCommandResponse {
	return &AgentCommandResponse{
		ID:        c.ID,
		Type:      c.Type,
		Payload:   c.Payload,
		TimeoutAt: c.TimeoutAt,
	}
}
