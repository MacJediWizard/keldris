package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DockerBackupSchedule represents how often a container should be backed up.
type DockerBackupSchedule string

const (
	// DockerBackupScheduleHourly indicates hourly backups.
	DockerBackupScheduleHourly DockerBackupSchedule = "hourly"
	// DockerBackupScheduleDaily indicates daily backups.
	DockerBackupScheduleDaily DockerBackupSchedule = "daily"
	// DockerBackupScheduleWeekly indicates weekly backups.
	DockerBackupScheduleWeekly DockerBackupSchedule = "weekly"
	// DockerBackupScheduleMonthly indicates monthly backups.
	DockerBackupScheduleMonthly DockerBackupSchedule = "monthly"
	// DockerBackupScheduleCustom indicates a custom cron expression.
	DockerBackupScheduleCustom DockerBackupSchedule = "custom"
)

// DockerContainerConfig represents backup configuration for a Docker container.
type DockerContainerConfig struct {
	ID               uuid.UUID            `json:"id"`
	AgentID          uuid.UUID            `json:"agent_id"`
	ContainerID      string               `json:"container_id"`      // Docker container ID (short or long)
	ContainerName    string               `json:"container_name"`    // Docker container name
	ImageName        string               `json:"image_name"`        // Docker image name
	Enabled          bool                 `json:"enabled"`           // Whether backup is enabled
	Schedule         DockerBackupSchedule `json:"schedule"`          // Backup schedule
	CronExpression   string               `json:"cron_expression,omitempty"` // Custom cron (when schedule=custom)
	Excludes         []string             `json:"excludes,omitempty"`        // Paths to exclude
	PreHook          string               `json:"pre_hook,omitempty"`        // Command to run before backup
	PostHook         string               `json:"post_hook,omitempty"`       // Command to run after backup
	StopOnBackup     bool                 `json:"stop_on_backup"`            // Stop container during backup
	BackupVolumes    bool                 `json:"backup_volumes"`            // Backup mounted volumes
	BackupBindMounts bool                 `json:"backup_bind_mounts"`        // Backup bind mounts
	Labels           map[string]string    `json:"labels,omitempty"`          // Original Docker labels
	Overrides        *ContainerOverrides  `json:"overrides,omitempty"`       // UI-configured overrides
	DiscoveredAt     time.Time            `json:"discovered_at"`
	LastBackupAt     *time.Time           `json:"last_backup_at,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
}

// ContainerOverrides contains UI-configured overrides for label-based settings.
type ContainerOverrides struct {
	Enabled        *bool                 `json:"enabled,omitempty"`
	Schedule       *DockerBackupSchedule `json:"schedule,omitempty"`
	CronExpression *string               `json:"cron_expression,omitempty"`
	Excludes       []string              `json:"excludes,omitempty"`
	PreHook        *string               `json:"pre_hook,omitempty"`
	PostHook       *string               `json:"post_hook,omitempty"`
	StopOnBackup   *bool                 `json:"stop_on_backup,omitempty"`
	BackupVolumes  *bool                 `json:"backup_volumes,omitempty"`
}

// NewDockerContainerConfig creates a new DockerContainerConfig with defaults.
func NewDockerContainerConfig(agentID uuid.UUID, containerID, containerName, imageName string) *DockerContainerConfig {
	now := time.Now()
	return &DockerContainerConfig{
		ID:               uuid.New(),
		AgentID:          agentID,
		ContainerID:      containerID,
		ContainerName:    containerName,
		ImageName:        imageName,
		Enabled:          true,
		Schedule:         DockerBackupScheduleDaily,
		BackupVolumes:    true,
		BackupBindMounts: false,
		DiscoveredAt:     now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// ApplyOverrides applies UI overrides to the label-based configuration.
func (c *DockerContainerConfig) ApplyOverrides() {
	if c.Overrides == nil {
		return
	}

	if c.Overrides.Enabled != nil {
		c.Enabled = *c.Overrides.Enabled
	}
	if c.Overrides.Schedule != nil {
		c.Schedule = *c.Overrides.Schedule
	}
	if c.Overrides.CronExpression != nil {
		c.CronExpression = *c.Overrides.CronExpression
	}
	if len(c.Overrides.Excludes) > 0 {
		c.Excludes = c.Overrides.Excludes
	}
	if c.Overrides.PreHook != nil {
		c.PreHook = *c.Overrides.PreHook
	}
	if c.Overrides.PostHook != nil {
		c.PostHook = *c.Overrides.PostHook
	}
	if c.Overrides.StopOnBackup != nil {
		c.StopOnBackup = *c.Overrides.StopOnBackup
	}
	if c.Overrides.BackupVolumes != nil {
		c.BackupVolumes = *c.Overrides.BackupVolumes
	}
}

// GetEffectiveCronExpression returns the cron expression based on schedule type.
func (c *DockerContainerConfig) GetEffectiveCronExpression() string {
	if c.Schedule == DockerBackupScheduleCustom && c.CronExpression != "" {
		return c.CronExpression
	}

	switch c.Schedule {
	case DockerBackupScheduleHourly:
		return "0 0 * * * *" // Every hour at minute 0
	case DockerBackupScheduleDaily:
		return "0 0 2 * * *" // Daily at 2:00 AM
	case DockerBackupScheduleWeekly:
		return "0 0 2 * * 0" // Weekly on Sunday at 2:00 AM
	case DockerBackupScheduleMonthly:
		return "0 0 2 1 * *" // Monthly on the 1st at 2:00 AM
	default:
		return "0 0 2 * * *" // Default to daily
	}
}

// SetLabels sets the labels from a map.
func (c *DockerContainerConfig) SetLabels(labels map[string]string) {
	c.Labels = labels
}

// LabelsJSON returns the labels as JSON bytes for database storage.
func (c *DockerContainerConfig) LabelsJSON() ([]byte, error) {
	if c.Labels == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(c.Labels)
}

// SetLabelsFromJSON sets the labels from JSON bytes.
func (c *DockerContainerConfig) SetLabelsFromJSON(data []byte) error {
	if len(data) == 0 {
		c.Labels = make(map[string]string)
		return nil
	}
	return json.Unmarshal(data, &c.Labels)
}

// SetExcludes sets the excludes from JSON bytes.
func (c *DockerContainerConfig) SetExcludes(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &c.Excludes)
}

// ExcludesJSON returns the excludes as JSON bytes for database storage.
func (c *DockerContainerConfig) ExcludesJSON() ([]byte, error) {
	if c.Excludes == nil {
		return nil, nil
	}
	return json.Marshal(c.Excludes)
}

// SetOverrides sets the overrides from JSON bytes.
func (c *DockerContainerConfig) SetOverrides(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var overrides ContainerOverrides
	if err := json.Unmarshal(data, &overrides); err != nil {
		return err
	}
	c.Overrides = &overrides
	return nil
}

// OverridesJSON returns the overrides as JSON bytes for database storage.
func (c *DockerContainerConfig) OverridesJSON() ([]byte, error) {
	if c.Overrides == nil {
		return nil, nil
	}
	return json.Marshal(c.Overrides)
}

// DockerDiscoveryResult represents the result of container discovery.
type DockerDiscoveryResult struct {
	Containers       []*DockerContainerConfig `json:"containers"`
	TotalDiscovered  int                      `json:"total_discovered"`
	TotalEnabled     int                      `json:"total_enabled"`
	NewContainers    int                      `json:"new_containers"`
	RemovedContainers int                     `json:"removed_containers"`
	DiscoveredAt     time.Time                `json:"discovered_at"`
}

// DockerLabelDocs represents documentation for Docker backup labels.
type DockerLabelDocs struct {
	Labels     []DockerLabelDoc `json:"labels"`
	GeneratedAt time.Time       `json:"generated_at"`
}

// DockerLabelDoc represents documentation for a single Docker label.
type DockerLabelDoc struct {
	Label       string   `json:"label"`
	Description string   `json:"description"`
	Type        string   `json:"type"`
	Default     string   `json:"default,omitempty"`
	Examples    []string `json:"examples,omitempty"`
	Required    bool     `json:"required"`
}
