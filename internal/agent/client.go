// Package agent provides an HTTP client for agent-to-server communication.
package agent

import (
	"bytes"
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
func (c *Client) ReportBackup(report *BackupReport) error {
	var result map[string]any
	if err := c.post("/api/v1/agent/backups", report, &result); err != nil {
		return fmt.Errorf("report backup: %w", err)
	}
	return nil
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
	Status  string            `json:"status"`
	OSInfo  *OSInfo           `json:"os_info,omitempty"`
	Metrics *HeartbeatMetrics `json:"metrics,omitempty"`
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
	req, err := http.NewRequest("GET", c.serverURL+path, nil)
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

	req, err := http.NewRequest("POST", c.serverURL+path, bytes.NewReader(data))
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
