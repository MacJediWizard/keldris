package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RetentionPolicy defines how long backups are retained.
type RetentionPolicy struct {
	KeepLast    int `json:"keep_last,omitempty"`
	KeepHourly  int `json:"keep_hourly,omitempty"`
	KeepDaily   int `json:"keep_daily,omitempty"`
	KeepWeekly  int `json:"keep_weekly,omitempty"`
	KeepMonthly int `json:"keep_monthly,omitempty"`
	KeepYearly  int `json:"keep_yearly,omitempty"`
}

// BackupWindow represents a time window during which backups are allowed.
type BackupWindow struct {
	Start string `json:"start,omitempty"` // HH:MM format (e.g., "02:00")
	End   string `json:"end,omitempty"`   // HH:MM format (e.g., "06:00")
}

// SchedulePriority represents the priority level for a backup schedule.
type SchedulePriority int

const (
	// PriorityHigh indicates high priority (1) - runs first, can preempt lower priority.
	PriorityHigh SchedulePriority = 1
	// PriorityMedium indicates medium priority (2) - default priority level.
	PriorityMedium SchedulePriority = 2
	// PriorityLow indicates low priority (3) - runs last, can be preempted.
	PriorityLow SchedulePriority = 3
)

// BackupType represents the type of backup to perform.
type BackupType string

const (
	// BackupTypeFile is a standard file/directory backup.
	BackupTypeFile BackupType = "file"
	// BackupTypeDocker backs up Docker volumes and container configs.
	BackupTypeDocker BackupType = "docker"
)

// DockerBackupOptions contains Docker-specific backup configuration.
type DockerBackupOptions struct {
	// VolumeIDs specifies which Docker volumes to backup. Empty means all volumes.
	VolumeIDs []string `json:"volume_ids,omitempty"`
	// ContainerIDs specifies which container configs to backup. Empty means all containers.
	ContainerIDs []string `json:"container_ids,omitempty"`
	// PauseContainers pauses running containers during volume backup for consistency.
	PauseContainers bool `json:"pause_containers"`
	// IncludeContainerConfigs backs up container configurations as JSON.
	IncludeContainerConfigs bool `json:"include_container_configs"`
}

// Schedule represents a backup schedule configuration.
// A schedule can be assigned to either an individual agent (via AgentID)
// or to an agent group (via AgentGroupID). When AgentGroupID is set,
// the schedule applies to all agents in that group.
type Schedule struct {
	ID                      uuid.UUID              `json:"id"`
	AgentID                 uuid.UUID              `json:"agent_id"`
	AgentGroupID            *uuid.UUID             `json:"agent_group_id,omitempty"` // If set, applies to all agents in the group
	PolicyID                *uuid.UUID             `json:"policy_id,omitempty"`      // Policy this schedule was created from
	Name                    string                 `json:"name"`
	BackupType              BackupType             `json:"backup_type"`              // Type of backup: file, docker
	CronExpression          string                 `json:"cron_expression"`
	Paths                   []string               `json:"paths"`
	Excludes                []string               `json:"excludes,omitempty"`
	RetentionPolicy         *RetentionPolicy       `json:"retention_policy,omitempty"`
	BandwidthLimitKB        *int                   `json:"bandwidth_limit_kb,omitempty"`   // Upload limit in KB/s
	BackupWindow            *BackupWindow          `json:"backup_window,omitempty"`        // Allowed backup time window
	ExcludedHours           []int                  `json:"excluded_hours,omitempty"`       // Hours (0-23) when backups should not run
	CompressionLevel        *string                `json:"compression_level,omitempty"`    // Compression level: off, auto, max
	MaxFileSizeMB           *int                   `json:"max_file_size_mb,omitempty"`     // Max file size in MB (0 = disabled)
	OnMountUnavailable      MountBehavior          `json:"on_mount_unavailable,omitempty"` // Behavior when network mount unavailable
	ClassificationLevel     string                 `json:"classification_level,omitempty"` // Data classification level: public, internal, confidential, restricted
	ClassificationDataTypes []string               `json:"classification_data_types,omitempty"` // Data types: pii, phi, pci, proprietary, general
	Priority                SchedulePriority       `json:"priority"`                       // Backup priority: 1=high, 2=medium, 3=low
	Preemptible             bool                   `json:"preemptible"`                    // Can be preempted by higher priority backups
	DockerOptions           *DockerBackupOptions   `json:"docker_options,omitempty"`       // Docker-specific backup options
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
	Enabled                 bool                   `json:"enabled"`
	Repositories            []ScheduleRepository   `json:"repositories,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
}

// NewSchedule creates a new Schedule with the given details.
func NewSchedule(agentID uuid.UUID, name, cronExpr string, paths []string) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		BackupType:         BackupTypeFile, // Default to file backup
		CronExpression:     cronExpr,
		Paths:              paths,
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium, // Default to medium priority
		Preemptible:        false,          // Not preemptible by default
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// NewDockerSchedule creates a new Schedule for Docker backup.
func NewDockerSchedule(agentID uuid.UUID, name, cronExpr string, opts *DockerBackupOptions) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		BackupType:         BackupTypeDocker,
		CronExpression:     cronExpr,
		Paths:              []string{}, // Docker backups don't use paths
		DockerOptions:      opts,
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium,
		Preemptible:        false,
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// IsDockerBackup returns true if this is a Docker backup schedule.
func (s *Schedule) IsDockerBackup() bool {
	return s.BackupType == BackupTypeDocker
}

// SetDockerOptions sets the Docker backup options from JSON bytes.
func (s *Schedule) SetDockerOptions(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var opts DockerBackupOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return err
	}
	s.DockerOptions = &opts
	return nil
}

// DockerOptionsJSON returns the Docker options as JSON bytes for database storage.
func (s *Schedule) DockerOptionsJSON() ([]byte, error) {
	if s.DockerOptions == nil {
		return nil, nil
	}
	return json.Marshal(s.DockerOptions)
}

// PriorityLabel returns a human-readable label for the priority.
func (s *Schedule) PriorityLabel() string {
	switch s.Priority {
	case PriorityHigh:
		return "high"
	case PriorityMedium:
		return "medium"
	case PriorityLow:
		return "low"
	default:
		return "medium"
	}
}

// IsHigherPriorityThan returns true if this schedule has higher priority than other.
func (s *Schedule) IsHigherPriorityThan(other *Schedule) bool {
	return s.Priority < other.Priority
}

// GetPrimaryRepository returns the primary repository (priority 0), or nil if none.
func (s *Schedule) GetPrimaryRepository() *ScheduleRepository {
	for i := range s.Repositories {
		if s.Repositories[i].Priority == 0 && s.Repositories[i].Enabled {
			return &s.Repositories[i]
		}
	}
	return nil
}

// GetEnabledRepositories returns all enabled repositories sorted by priority.
func (s *Schedule) GetEnabledRepositories() []ScheduleRepository {
	var enabled []ScheduleRepository
	for _, r := range s.Repositories {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	// Sort by priority (already sorted in DB query, but ensure)
	for i := 0; i < len(enabled)-1; i++ {
		for j := i + 1; j < len(enabled); j++ {
			if enabled[i].Priority > enabled[j].Priority {
				enabled[i], enabled[j] = enabled[j], enabled[i]
			}
		}
	}
	return enabled
}

// SetPaths sets the paths from JSON bytes.
func (s *Schedule) SetPaths(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Paths)
}

// PathsJSON returns the paths as JSON bytes for database storage.
func (s *Schedule) PathsJSON() ([]byte, error) {
	if s.Paths == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s.Paths)
}

// SetExcludes sets the excludes from JSON bytes.
func (s *Schedule) SetExcludes(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Excludes)
}

// ExcludesJSON returns the excludes as JSON bytes for database storage.
func (s *Schedule) ExcludesJSON() ([]byte, error) {
	if s.Excludes == nil {
		return nil, nil
	}
	return json.Marshal(s.Excludes)
}

// SetRetentionPolicy sets the retention policy from JSON bytes.
func (s *Schedule) SetRetentionPolicy(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var policy RetentionPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return err
	}
	s.RetentionPolicy = &policy
	return nil
}

// RetentionPolicyJSON returns the retention policy as JSON bytes for database storage.
func (s *Schedule) RetentionPolicyJSON() ([]byte, error) {
	if s.RetentionPolicy == nil {
		return nil, nil
	}
	return json.Marshal(s.RetentionPolicy)
}

// DefaultRetentionPolicy returns a sensible default retention policy.
func DefaultRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		KeepLast:    5,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 6,
	}
}

// SetBackupWindow sets the backup window from JSON bytes.
func (s *Schedule) SetBackupWindow(startTime, endTime *string) {
	if startTime == nil && endTime == nil {
		s.BackupWindow = nil
		return
	}
	s.BackupWindow = &BackupWindow{}
	if startTime != nil {
		s.BackupWindow.Start = *startTime
	}
	if endTime != nil {
		s.BackupWindow.End = *endTime
	}
}

// SetExcludedHours sets the excluded hours from JSON bytes.
func (s *Schedule) SetExcludedHours(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.ExcludedHours)
}

// ExcludedHoursJSON returns the excluded hours as JSON bytes for database storage.
func (s *Schedule) ExcludedHoursJSON() ([]byte, error) {
	if s.ExcludedHours == nil {
		return nil, nil
	}
	return json.Marshal(s.ExcludedHours)
}

// IsWithinBackupWindow checks if the given time is within the allowed backup window.
// Returns true if no window is set (always allowed) or if the time is within the window.
func (s *Schedule) IsWithinBackupWindow(t time.Time) bool {
	if s.BackupWindow == nil || (s.BackupWindow.Start == "" && s.BackupWindow.End == "") {
		return true
	}

	currentTime := t.Format("15:04")
	start := s.BackupWindow.Start
	end := s.BackupWindow.End

	// Handle window that doesn't cross midnight (e.g., 02:00 to 06:00)
	if start <= end {
		return currentTime >= start && currentTime < end
	}

	// Handle window that crosses midnight (e.g., 22:00 to 06:00)
	return currentTime >= start || currentTime < end
}

// IsHourExcluded checks if the given hour is in the excluded hours list.
func (s *Schedule) IsHourExcluded(hour int) bool {
	for _, h := range s.ExcludedHours {
		if h == hour {
			return true
		}
	}
	return false
}

// CanRunAt checks if a backup can run at the given time based on window and excluded hours.
func (s *Schedule) CanRunAt(t time.Time) bool {
	if !s.IsWithinBackupWindow(t) {
		return false
	}
	if s.IsHourExcluded(t.Hour()) {
		return false
	}
	return true
}

// NextAllowedTime finds the next time when a backup can run, starting from the given time.
// Returns the input time if it's already allowed.
func (s *Schedule) NextAllowedTime(t time.Time) time.Time {
	// Check up to 24 hours ahead
	for i := 0; i < 24*60; i++ {
		checkTime := t.Add(time.Duration(i) * time.Minute)
		if s.CanRunAt(checkTime) {
			return checkTime
		}
	}
	// Fallback: return original time if no valid window found
	return t
}

// SetClassificationDataTypes sets the classification data types from JSON bytes.
func (s *Schedule) SetClassificationDataTypes(data []byte) error {
	if len(data) == 0 {
		s.ClassificationDataTypes = []string{"general"}
		return nil
	}
	return json.Unmarshal(data, &s.ClassificationDataTypes)
}

// ClassificationDataTypesJSON returns the classification data types as JSON bytes.
func (s *Schedule) ClassificationDataTypesJSON() ([]byte, error) {
	if len(s.ClassificationDataTypes) == 0 {
		return []byte(`["general"]`), nil
	}
	return json.Marshal(s.ClassificationDataTypes)
}

// SetMetadata sets the metadata from JSON bytes.
func (s *Schedule) SetMetadata(data []byte) error {
	if len(data) == 0 {
		s.Metadata = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(data, &s.Metadata)
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (s *Schedule) MetadataJSON() ([]byte, error) {
	if s.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(s.Metadata)
}

// BackupQueueStatus represents the status of a backup queue item.
type BackupQueueStatus string

const (
	// QueueStatusPending indicates the backup is waiting to run.
	QueueStatusPending BackupQueueStatus = "pending"
	// QueueStatusRunning indicates the backup is currently running.
	QueueStatusRunning BackupQueueStatus = "running"
	// QueueStatusCompleted indicates the backup completed successfully.
	QueueStatusCompleted BackupQueueStatus = "completed"
	// QueueStatusFailed indicates the backup failed.
	QueueStatusFailed BackupQueueStatus = "failed"
	// QueueStatusPreempted indicates the backup was preempted by a higher priority backup.
	QueueStatusPreempted BackupQueueStatus = "preempted"
	// QueueStatusCanceled indicates the backup was canceled.
	QueueStatusCanceled BackupQueueStatus = "canceled"
)

// BackupQueueItem represents a backup in the priority queue.
type BackupQueueItem struct {
	ID          uuid.UUID         `json:"id"`
	ScheduleID  uuid.UUID         `json:"schedule_id"`
	AgentID     uuid.UUID         `json:"agent_id"`
	Priority    SchedulePriority  `json:"priority"`
	Status      BackupQueueStatus `json:"status"`
	QueuedAt    time.Time         `json:"queued_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	PreemptedBy *uuid.UUID        `json:"preempted_by,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewBackupQueueItem creates a new backup queue item.
func NewBackupQueueItem(scheduleID, agentID uuid.UUID, priority SchedulePriority) *BackupQueueItem {
	now := time.Now()
	return &BackupQueueItem{
		ID:         uuid.New(),
		ScheduleID: scheduleID,
		AgentID:    agentID,
		Priority:   priority,
		Status:     QueueStatusPending,
		QueuedAt:   now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// BackupQueueSummary provides a summary of the backup queue for display.
type BackupQueueSummary struct {
	TotalPending   int `json:"total_pending"`
	TotalRunning   int `json:"total_running"`
	HighPriority   int `json:"high_priority"`
	MediumPriority int `json:"medium_priority"`
	LowPriority    int `json:"low_priority"`
}
