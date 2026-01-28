package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DockerStackBackupStatus represents the status of a docker stack backup.
type DockerStackBackupStatus string

const (
	// DockerStackBackupStatusPending indicates the backup has not started.
	DockerStackBackupStatusPending DockerStackBackupStatus = "pending"
	// DockerStackBackupStatusRunning indicates the backup is in progress.
	DockerStackBackupStatusRunning DockerStackBackupStatus = "running"
	// DockerStackBackupStatusCompleted indicates the backup completed successfully.
	DockerStackBackupStatusCompleted DockerStackBackupStatus = "completed"
	// DockerStackBackupStatusFailed indicates the backup failed.
	DockerStackBackupStatusFailed DockerStackBackupStatus = "failed"
	// DockerStackBackupStatusCanceled indicates the backup was canceled.
	DockerStackBackupStatusCanceled DockerStackBackupStatus = "canceled"
)

// DockerStackRestoreStatus represents the status of a docker stack restore.
type DockerStackRestoreStatus string

const (
	// DockerStackRestoreStatusPending indicates the restore has not started.
	DockerStackRestoreStatusPending DockerStackRestoreStatus = "pending"
	// DockerStackRestoreStatusRunning indicates the restore is in progress.
	DockerStackRestoreStatusRunning DockerStackRestoreStatus = "running"
	// DockerStackRestoreStatusCompleted indicates the restore completed successfully.
	DockerStackRestoreStatusCompleted DockerStackRestoreStatus = "completed"
	// DockerStackRestoreStatusFailed indicates the restore failed.
	DockerStackRestoreStatusFailed DockerStackRestoreStatus = "failed"
)

// DockerStack represents a registered Docker Compose stack for backup.
type DockerStack struct {
	ID              uuid.UUID  `json:"id"`
	OrgID           uuid.UUID  `json:"org_id"`
	AgentID         uuid.UUID  `json:"agent_id"`
	Name            string     `json:"name"`
	ComposePath     string     `json:"compose_path"`
	Description     string     `json:"description,omitempty"`
	ServiceCount    int        `json:"service_count"`
	IsRunning       bool       `json:"is_running"`
	LastBackupAt    *time.Time `json:"last_backup_at,omitempty"`
	LastBackupID    *uuid.UUID `json:"last_backup_id,omitempty"`
	BackupScheduleID *uuid.UUID `json:"backup_schedule_id,omitempty"` // Associated schedule
	ExportImages    bool       `json:"export_images"`                 // Include images in backup
	IncludeEnvFiles bool       `json:"include_env_files"`             // Include .env files
	StopForBackup   bool       `json:"stop_for_backup"`               // Stop containers during backup
	ExcludePaths    []string   `json:"exclude_paths,omitempty"`       // Paths to exclude from backup
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// NewDockerStack creates a new DockerStack with default values.
func NewDockerStack(orgID, agentID uuid.UUID, name, composePath string) *DockerStack {
	now := time.Now()
	return &DockerStack{
		ID:              uuid.New(),
		OrgID:           orgID,
		AgentID:         agentID,
		Name:            name,
		ComposePath:     composePath,
		IncludeEnvFiles: true,  // Include env files by default
		ExportImages:    false, // Don't export images by default (large)
		StopForBackup:   false, // Don't stop containers by default
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// SetExcludePaths sets the exclude paths from JSON bytes.
func (s *DockerStack) SetExcludePaths(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.ExcludePaths)
}

// ExcludePathsJSON returns the exclude paths as JSON bytes.
func (s *DockerStack) ExcludePathsJSON() ([]byte, error) {
	if s.ExcludePaths == nil {
		return nil, nil
	}
	return json.Marshal(s.ExcludePaths)
}

// DockerStackBackup represents a backup of a Docker Compose stack.
type DockerStackBackup struct {
	ID               uuid.UUID               `json:"id"`
	OrgID            uuid.UUID               `json:"org_id"`
	StackID          uuid.UUID               `json:"stack_id"`
	AgentID          uuid.UUID               `json:"agent_id"`
	ScheduleID       *uuid.UUID              `json:"schedule_id,omitempty"`     // If part of a schedule
	BackupID         *uuid.UUID              `json:"backup_id,omitempty"`       // Associated restic backup ID
	Status           DockerStackBackupStatus `json:"status"`
	BackupPath       string                  `json:"backup_path"`               // Where backup is stored
	ManifestPath     string                  `json:"manifest_path,omitempty"`   // Path to manifest.json
	VolumeCount      int                     `json:"volume_count"`
	BindMountCount   int                     `json:"bind_mount_count"`
	ImageCount       int                     `json:"image_count,omitempty"`
	TotalSizeBytes   int64                   `json:"total_size_bytes"`
	ContainerStates  []DockerContainerState  `json:"container_states,omitempty"`
	DependencyOrder  []string                `json:"dependency_order,omitempty"`
	IncludesImages   bool                    `json:"includes_images"`
	ErrorMessage     string                  `json:"error_message,omitempty"`
	StartedAt        *time.Time              `json:"started_at,omitempty"`
	CompletedAt      *time.Time              `json:"completed_at,omitempty"`
	CreatedAt        time.Time               `json:"created_at"`
	UpdatedAt        time.Time               `json:"updated_at"`
}

// NewDockerStackBackup creates a new DockerStackBackup.
func NewDockerStackBackup(orgID, stackID, agentID uuid.UUID, backupPath string) *DockerStackBackup {
	now := time.Now()
	return &DockerStackBackup{
		ID:         uuid.New(),
		OrgID:      orgID,
		StackID:    stackID,
		AgentID:    agentID,
		BackupPath: backupPath,
		Status:     DockerStackBackupStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// Start marks the backup as started.
func (b *DockerStackBackup) Start() {
	now := time.Now()
	b.Status = DockerStackBackupStatusRunning
	b.StartedAt = &now
	b.UpdatedAt = now
}

// Complete marks the backup as completed successfully.
func (b *DockerStackBackup) Complete(manifestPath string, volumeCount, bindMountCount, imageCount int, totalSize int64) {
	now := time.Now()
	b.Status = DockerStackBackupStatusCompleted
	b.ManifestPath = manifestPath
	b.VolumeCount = volumeCount
	b.BindMountCount = bindMountCount
	b.ImageCount = imageCount
	b.TotalSizeBytes = totalSize
	b.CompletedAt = &now
	b.UpdatedAt = now
}

// Fail marks the backup as failed.
func (b *DockerStackBackup) Fail(errMsg string) {
	now := time.Now()
	b.Status = DockerStackBackupStatusFailed
	b.ErrorMessage = errMsg
	b.CompletedAt = &now
	b.UpdatedAt = now
}

// SetContainerStates sets the container states from JSON bytes.
func (b *DockerStackBackup) SetContainerStates(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &b.ContainerStates)
}

// ContainerStatesJSON returns the container states as JSON bytes.
func (b *DockerStackBackup) ContainerStatesJSON() ([]byte, error) {
	if b.ContainerStates == nil {
		return nil, nil
	}
	return json.Marshal(b.ContainerStates)
}

// SetDependencyOrder sets the dependency order from JSON bytes.
func (b *DockerStackBackup) SetDependencyOrder(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &b.DependencyOrder)
}

// DependencyOrderJSON returns the dependency order as JSON bytes.
func (b *DockerStackBackup) DependencyOrderJSON() ([]byte, error) {
	if b.DependencyOrder == nil {
		return nil, nil
	}
	return json.Marshal(b.DependencyOrder)
}

// DockerContainerState represents the state of a container at backup time.
type DockerContainerState struct {
	ServiceName string    `json:"service_name"`
	ContainerID string    `json:"container_id"`
	Status      string    `json:"status"`       // running, paused, stopped, etc.
	Health      string    `json:"health,omitempty"` // healthy, unhealthy, starting
	Image       string    `json:"image"`
	ImageID     string    `json:"image_id"`
	Created     time.Time `json:"created"`
	Started     time.Time `json:"started,omitempty"`
}

// DockerVolumeBackup represents a backed up volume.
type DockerVolumeBackup struct {
	ID              uuid.UUID  `json:"id"`
	StackBackupID   uuid.UUID  `json:"stack_backup_id"`
	VolumeName      string     `json:"volume_name"`
	ServiceName     string     `json:"service_name,omitempty"`
	MountPath       string     `json:"mount_path"`
	BackupPath      string     `json:"backup_path"`
	SizeBytes       int64      `json:"size_bytes"`
	FileCount       int        `json:"file_count"`
	IsNamedVolume   bool       `json:"is_named_volume"`
	IsBindMount     bool       `json:"is_bind_mount"`
	BackedUpAt      time.Time  `json:"backed_up_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// DockerImageBackup represents a backed up Docker image.
type DockerImageBackup struct {
	ID              uuid.UUID  `json:"id"`
	StackBackupID   uuid.UUID  `json:"stack_backup_id"`
	ImageName       string     `json:"image_name"`
	ImageID         string     `json:"image_id"`
	Tags            []string   `json:"tags,omitempty"`
	SizeBytes       int64      `json:"size_bytes"`
	BackupPath      string     `json:"backup_path"`
	BackedUpAt      time.Time  `json:"backed_up_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SetTags sets the tags from JSON bytes.
func (i *DockerImageBackup) SetTags(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &i.Tags)
}

// TagsJSON returns the tags as JSON bytes.
func (i *DockerImageBackup) TagsJSON() ([]byte, error) {
	if i.Tags == nil {
		return nil, nil
	}
	return json.Marshal(i.Tags)
}

// DockerStackRestore represents a restore operation for a Docker Compose stack.
type DockerStackRestore struct {
	ID              uuid.UUID                `json:"id"`
	OrgID           uuid.UUID                `json:"org_id"`
	StackBackupID   uuid.UUID                `json:"stack_backup_id"`
	AgentID         uuid.UUID                `json:"agent_id"`
	Status          DockerStackRestoreStatus `json:"status"`
	TargetPath      string                   `json:"target_path"`
	RestoreVolumes  bool                     `json:"restore_volumes"`
	RestoreImages   bool                     `json:"restore_images"`
	StartContainers bool                     `json:"start_containers"`
	PathMappings    map[string]string        `json:"path_mappings,omitempty"`
	VolumesRestored int                      `json:"volumes_restored"`
	ImagesRestored  int                      `json:"images_restored"`
	ErrorMessage    string                   `json:"error_message,omitempty"`
	StartedAt       *time.Time               `json:"started_at,omitempty"`
	CompletedAt     *time.Time               `json:"completed_at,omitempty"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"updated_at"`
}

// NewDockerStackRestore creates a new DockerStackRestore.
func NewDockerStackRestore(orgID, stackBackupID, agentID uuid.UUID, targetPath string) *DockerStackRestore {
	now := time.Now()
	return &DockerStackRestore{
		ID:              uuid.New(),
		OrgID:           orgID,
		StackBackupID:   stackBackupID,
		AgentID:         agentID,
		TargetPath:      targetPath,
		Status:          DockerStackRestoreStatusPending,
		RestoreVolumes:  true,
		StartContainers: true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// Start marks the restore as started.
func (r *DockerStackRestore) Start() {
	now := time.Now()
	r.Status = DockerStackRestoreStatusRunning
	r.StartedAt = &now
	r.UpdatedAt = now
}

// Complete marks the restore as completed.
func (r *DockerStackRestore) Complete(volumesRestored, imagesRestored int) {
	now := time.Now()
	r.Status = DockerStackRestoreStatusCompleted
	r.VolumesRestored = volumesRestored
	r.ImagesRestored = imagesRestored
	r.CompletedAt = &now
	r.UpdatedAt = now
}

// Fail marks the restore as failed.
func (r *DockerStackRestore) Fail(errMsg string) {
	now := time.Now()
	r.Status = DockerStackRestoreStatusFailed
	r.ErrorMessage = errMsg
	r.CompletedAt = &now
	r.UpdatedAt = now
}

// SetPathMappings sets the path mappings from JSON bytes.
func (r *DockerStackRestore) SetPathMappings(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &r.PathMappings)
}

// PathMappingsJSON returns the path mappings as JSON bytes.
func (r *DockerStackRestore) PathMappingsJSON() ([]byte, error) {
	if r.PathMappings == nil {
		return nil, nil
	}
	return json.Marshal(r.PathMappings)
}

// DockerStackScheduleConfig contains docker-specific settings for a backup schedule.
type DockerStackScheduleConfig struct {
	StackID         uuid.UUID `json:"stack_id"`
	ExportImages    bool      `json:"export_images"`
	IncludeEnvFiles bool      `json:"include_env_files"`
	StopForBackup   bool      `json:"stop_for_backup"`
}

// DockerStackBackupSummary provides a summary of stack backup status.
type DockerStackBackupSummary struct {
	StackID             uuid.UUID `json:"stack_id"`
	StackName           string    `json:"stack_name"`
	TotalBackups        int       `json:"total_backups"`
	SuccessfulBackups   int       `json:"successful_backups"`
	FailedBackups       int       `json:"failed_backups"`
	TotalSizeBytes      int64     `json:"total_size_bytes"`
	LastBackupAt        *time.Time `json:"last_backup_at,omitempty"`
	LastBackupStatus    string    `json:"last_backup_status,omitempty"`
	NextScheduledBackup *time.Time `json:"next_scheduled_backup,omitempty"`
}

// DockerStackListResponse contains a list of docker stacks.
type DockerStackListResponse struct {
	Stacks []DockerStack `json:"stacks"`
}

// DockerStackBackupListResponse contains a list of docker stack backups.
type DockerStackBackupListResponse struct {
	Backups []DockerStackBackup `json:"backups"`
}

// CreateDockerStackRequest is the request to register a new Docker stack.
type CreateDockerStackRequest struct {
	Name            string   `json:"name"`
	AgentID         string   `json:"agent_id"`
	ComposePath     string   `json:"compose_path"`
	Description     string   `json:"description,omitempty"`
	ExportImages    bool     `json:"export_images"`
	IncludeEnvFiles bool     `json:"include_env_files"`
	StopForBackup   bool     `json:"stop_for_backup"`
	ExcludePaths    []string `json:"exclude_paths,omitempty"`
}

// UpdateDockerStackRequest is the request to update a Docker stack.
type UpdateDockerStackRequest struct {
	Name            *string   `json:"name,omitempty"`
	Description     *string   `json:"description,omitempty"`
	ExportImages    *bool     `json:"export_images,omitempty"`
	IncludeEnvFiles *bool     `json:"include_env_files,omitempty"`
	StopForBackup   *bool     `json:"stop_for_backup,omitempty"`
	ExcludePaths    *[]string `json:"exclude_paths,omitempty"`
}

// TriggerDockerStackBackupRequest is the request to trigger a manual stack backup.
type TriggerDockerStackBackupRequest struct {
	ExportImages  *bool `json:"export_images,omitempty"`  // Override stack setting
	StopForBackup *bool `json:"stop_for_backup,omitempty"` // Override stack setting
}

// RestoreDockerStackRequest is the request to restore a Docker stack.
type RestoreDockerStackRequest struct {
	BackupID        string            `json:"backup_id"`
	TargetAgentID   string            `json:"target_agent_id"`
	TargetPath      string            `json:"target_path"`
	RestoreVolumes  bool              `json:"restore_volumes"`
	RestoreImages   bool              `json:"restore_images"`
	StartContainers bool              `json:"start_containers"`
	PathMappings    map[string]string `json:"path_mappings,omitempty"`
}

// DiscoverDockerStacksRequest is the request to discover Docker stacks on an agent.
type DiscoverDockerStacksRequest struct {
	AgentID     string   `json:"agent_id"`
	SearchPaths []string `json:"search_paths"` // Paths to search for docker-compose files
}

// DiscoveredDockerStack represents a discovered Docker Compose stack.
type DiscoveredDockerStack struct {
	Name         string `json:"name"`
	ComposePath  string `json:"compose_path"`
	ServiceCount int    `json:"service_count"`
	IsRunning    bool   `json:"is_running"`
	IsRegistered bool   `json:"is_registered"` // Already registered in Keldris
}

// DiscoverDockerStacksResponse contains discovered Docker stacks.
type DiscoverDockerStacksResponse struct {
	Stacks []DiscoveredDockerStack `json:"stacks"`
}
