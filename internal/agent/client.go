// Package agent provides an HTTP client for agent-to-server communication.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client is an HTTP client for communicating with the Keldris server.
type Client struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new agent API client.
func NewClient(serverURL, apiKey string) *Client {
	return &Client{
		serverURL: serverURL,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxConnsPerHost: 10,
				MaxIdleConns:    5,
				IdleConnTimeout: 90 * time.Second,
			},
		},
	}
}

// ScheduleConfig holds schedule configuration with decrypted repository credentials.
type ScheduleConfig struct {
	ID                 uuid.UUID         `json:"id"`
	Name               string            `json:"name"`
	CronExpression     string            `json:"cron_expression"`
	Paths              []string          `json:"paths"`
	Excludes           []string          `json:"excludes"`
	Enabled            bool              `json:"enabled"`
	RepositoryID       uuid.UUID         `json:"repository_id"`
	Repository         string            `json:"repository"`
	RepositoryPassword string            `json:"repository_password"`
	RepositoryEnv      map[string]string `json:"repository_env,omitempty"`
}

// GetSchedules retrieves the agent's backup schedules with decrypted repo credentials.
func (c *Client) GetSchedules() ([]ScheduleConfig, error) {
	var schedules []ScheduleConfig
	if err := c.get("/api/v1/agent/schedules", &schedules); err != nil {
		return nil, fmt.Errorf("get schedules: %w", err)
	}
	return schedules, nil
}

// BackupReport contains the results of a backup operation.
type BackupReport struct {
	ScheduleID   uuid.UUID `json:"schedule_id"`
	RepositoryID uuid.UUID `json:"repository_id"`
	SnapshotID   string    `json:"snapshot_id"`
	Status       string    `json:"status"`
	SizeBytes    *int64    `json:"size_bytes,omitempty"`
	FilesNew     *int      `json:"files_new,omitempty"`
	FilesChanged *int      `json:"files_changed,omitempty"`
	ErrorMessage *string   `json:"error_message,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at"`
}

// ReportBackup reports a completed backup to the server.
// It retries up to 3 times on transient (5xx) server errors.
func (c *Client) ReportBackup(report *BackupReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("report backup: marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		req, err := http.NewRequestWithContext(ctx, "POST", c.serverURL+"/api/v1/agent/backups", bytes.NewReader(data))
		if err != nil {
			cancel()
			return fmt.Errorf("report backup: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("report backup: %w", err)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()
		if err != nil {
			lastErr = fmt.Errorf("report backup: read response: %w", err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return nil
		}

		lastErr = fmt.Errorf("report backup: server returned %d: %s", resp.StatusCode, string(body))

		// Retry on 5xx (server error); fail immediately on 4xx (client error)
		if resp.StatusCode < 500 {
			return lastErr
		}
	}
	return lastErr
}

// SnapshotInfo contains snapshot information returned by the server.
type SnapshotInfo struct {
	SnapshotID   string    `json:"snapshot_id"`
	BackupID     string    `json:"backup_id"`
	ScheduleID   string    `json:"schedule_id"`
	RepositoryID string    `json:"repository_id"`
	Status       string    `json:"status"`
	SizeBytes    *int64    `json:"size_bytes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// GetSnapshots retrieves available snapshots for this agent.
func (c *Client) GetSnapshots() ([]SnapshotInfo, error) {
	var snapshots []SnapshotInfo
	if err := c.get("/api/v1/agent/snapshots", &snapshots); err != nil {
		return nil, fmt.Errorf("get snapshots: %w", err)
	}
	return snapshots, nil
}

// HeartbeatRequest is the request body for health reporting.
type HeartbeatRequest struct {
	Status       string            `json:"status"`
	AgentVersion string            `json:"agent_version,omitempty"`
	OSInfo       *OSInfo           `json:"os_info,omitempty"`
	Metrics      *HeartbeatMetrics `json:"metrics,omitempty"`
}

// OSInfo contains operating system information.
type OSInfo struct {
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Hostname string `json:"hostname"`
	Version  string `json:"version,omitempty"`
}

// HeartbeatMetrics contains system metrics.
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

// SendHeartbeat sends a health report to the server.
func (c *Client) SendHeartbeat(req *HeartbeatRequest) error {
	var result map[string]any
	if err := c.post("/api/v1/agent/health", req, &result); err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	return nil
}

func (c *Client) get(path string, result any) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.serverURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, result)
}

func (c *Client) post(path string, payload, result any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.serverURL+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.Unmarshal(body, result)
	}
	return nil
}

// CommandResponse represents a command returned by the server.
type CommandResponse struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Payload   *CommandPayload `json:"payload,omitempty"`
	TimeoutAt string          `json:"timeout_at"`
}

// CommandPayload contains type-specific command parameters.
type CommandPayload struct {
	ScheduleID          *string  `json:"schedule_id,omitempty"`
	TargetVersion       string   `json:"target_version,omitempty"`
	TargetResticVersion string   `json:"target_restic_version,omitempty"`
	DiagnosticTypes     []string `json:"diagnostic_types,omitempty"`
	Purge               bool     `json:"purge,omitempty"`
}

// CommandsResponse is the server response for polling commands.
type CommandsResponse struct {
	Commands []CommandResponse `json:"commands"`
}

// CommandResultReport is the request body for reporting command results.
type CommandResultReport struct {
	Status string               `json:"status"`
	Result *CommandResultDetail `json:"result,omitempty"`
}

// CommandResultDetail contains the result details of a command execution.
type CommandResultDetail struct {
	Output      string              `json:"output,omitempty"`
	Error       string              `json:"error,omitempty"`
	Diagnostics map[string]any      `json:"diagnostics,omitempty"`
	DryRun      *DryRunResultDetail `json:"dry_run,omitempty"`
}

// DryRunResultDetail contains dry run backup preview results.
type DryRunResultDetail struct {
	TotalFiles     int   `json:"total_files"`
	TotalSize      int64 `json:"total_size"`
	NewFiles       int   `json:"new_files"`
	ChangedFiles   int   `json:"changed_files"`
	UnchangedFiles int   `json:"unchanged_files"`
}

// GetCommands polls the server for pending commands.
func (c *Client) GetCommands() ([]CommandResponse, error) {
	var resp CommandsResponse
	if err := c.get("/api/v1/agent/commands", &resp); err != nil {
		return nil, fmt.Errorf("get commands: %w", err)
	}
	return resp.Commands, nil
}

// AcknowledgeCommand acknowledges receipt of a command.
func (c *Client) AcknowledgeCommand(id string) error {
	var result map[string]any
	if err := c.post("/api/v1/agent/commands/"+id+"/ack", struct{}{}, &result); err != nil {
		return fmt.Errorf("acknowledge command %s: %w", id, err)
	}
	return nil
}

// ReportCommandResult reports the execution result of a command.
func (c *Client) ReportCommandResult(id string, report *CommandResultReport) error {
	var result map[string]any
	if err := c.post("/api/v1/agent/commands/"+id+"/result", report, &result); err != nil {
		return fmt.Errorf("report command result %s: %w", id, err)
	}
	return nil
}
