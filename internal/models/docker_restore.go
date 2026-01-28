package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DockerRestoreStatus represents the current status of a Docker restore operation.
type DockerRestoreStatus string

const (
	// DockerRestoreStatusPending indicates the restore is queued.
	DockerRestoreStatusPending DockerRestoreStatus = "pending"
	// DockerRestoreStatusPreparing indicates the restore is preparing.
	DockerRestoreStatusPreparing DockerRestoreStatus = "preparing"
	// DockerRestoreStatusRestoringVolumes indicates volumes are being restored.
	DockerRestoreStatusRestoringVolumes DockerRestoreStatus = "restoring_volumes"
	// DockerRestoreStatusCreatingContainer indicates the container is being created.
	DockerRestoreStatusCreatingContainer DockerRestoreStatus = "creating_container"
	// DockerRestoreStatusStarting indicates the container is starting.
	DockerRestoreStatusStarting DockerRestoreStatus = "starting"
	// DockerRestoreStatusVerifying indicates the container start is being verified.
	DockerRestoreStatusVerifying DockerRestoreStatus = "verifying"
	// DockerRestoreStatusCompleted indicates the restore completed successfully.
	DockerRestoreStatusCompleted DockerRestoreStatus = "completed"
	// DockerRestoreStatusFailed indicates the restore failed.
	DockerRestoreStatusFailed DockerRestoreStatus = "failed"
	// DockerRestoreStatusCanceled indicates the restore was canceled.
	DockerRestoreStatusCanceled DockerRestoreStatus = "canceled"
)

// DockerRestoreTargetType represents the type of Docker host target.
type DockerRestoreTargetType string

const (
	// DockerTargetLocal restores to the local Docker host.
	DockerTargetLocal DockerRestoreTargetType = "local"
	// DockerTargetRemote restores to a remote Docker host.
	DockerTargetRemote DockerRestoreTargetType = "remote"
)

// DockerRestoreTarget represents the target Docker host for restore.
type DockerRestoreTarget struct {
	Type      DockerRestoreTargetType `json:"type"`
	Host      string                  `json:"host,omitempty"`       // For remote: docker host URL
	CertPath  string                  `json:"cert_path,omitempty"`  // For remote: TLS cert path
	TLSVerify bool                    `json:"tls_verify,omitempty"`
}

// DockerRestoreProgress represents the progress of a Docker restore operation.
type DockerRestoreProgress struct {
	Status         string `json:"status"`
	CurrentStep    string `json:"current_step"`
	TotalSteps     int    `json:"total_steps"`
	CompletedSteps int    `json:"completed_steps"`
	TotalBytes     int64  `json:"total_bytes"`
	RestoredBytes  int64  `json:"restored_bytes"`
	CurrentVolume  string `json:"current_volume,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

// PercentComplete returns the restore completion percentage.
func (p *DockerRestoreProgress) PercentComplete() float64 {
	if p.TotalSteps == 0 {
		return 0
	}
	return float64(p.CompletedSteps) / float64(p.TotalSteps) * 100
}

// DockerRestoreContainerInfo represents a backed up container's information for restore.
type DockerRestoreContainerInfo struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	Volumes   []string `json:"volumes,omitempty"`
	Ports     []string `json:"ports,omitempty"`
	Networks  []string `json:"networks,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// DockerRestoreVolumeInfo represents a backed up volume's information for restore.
type DockerRestoreVolumeInfo struct {
	Name      string `json:"name"`
	Driver    string `json:"driver"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}

// DockerRestoreConflict represents a conflict detected during restore planning.
type DockerRestoreConflict struct {
	Type        string `json:"type"` // "container", "volume", "network"
	Name        string `json:"name"`
	ExistingID  string `json:"existing_id,omitempty"`
	Description string `json:"description"`
}

// DockerRestore represents a Docker restore job execution record.
type DockerRestore struct {
	ID                uuid.UUID             `json:"id"`
	AgentID           uuid.UUID             `json:"agent_id"`
	RepositoryID      uuid.UUID             `json:"repository_id"`
	SnapshotID        string                `json:"snapshot_id"`
	ContainerName     string                `json:"container_name,omitempty"`
	VolumeName        string                `json:"volume_name,omitempty"`
	NewContainerName  string                `json:"new_container_name,omitempty"`
	NewVolumeName     string                `json:"new_volume_name,omitempty"`
	Target            *DockerRestoreTarget  `json:"target,omitempty"`
	OverwriteExisting bool                  `json:"overwrite_existing"`
	StartAfterRestore bool                  `json:"start_after_restore"`
	VerifyStart       bool                  `json:"verify_start"`
	Status            DockerRestoreStatus   `json:"status"`
	Progress          *DockerRestoreProgress `json:"progress,omitempty"`
	RestoredContainerID string              `json:"restored_container_id,omitempty"`
	RestoredVolumes   []string              `json:"restored_volumes,omitempty"`
	StartVerified     bool                  `json:"start_verified"`
	Warnings          []string              `json:"warnings,omitempty"`
	StartedAt         *time.Time            `json:"started_at,omitempty"`
	CompletedAt       *time.Time            `json:"completed_at,omitempty"`
	ErrorMessage      string                `json:"error_message,omitempty"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	OrgID             uuid.UUID             `json:"org_id"`
}

// NewDockerRestore creates a new DockerRestore job.
func NewDockerRestore(orgID, agentID, repositoryID uuid.UUID, snapshotID string) *DockerRestore {
	now := time.Now()
	return &DockerRestore{
		ID:           uuid.New(),
		OrgID:        orgID,
		AgentID:      agentID,
		RepositoryID: repositoryID,
		SnapshotID:   snapshotID,
		Status:       DockerRestoreStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// TargetJSON returns the target as JSON for database storage.
func (r *DockerRestore) TargetJSON() ([]byte, error) {
	if r.Target == nil {
		return nil, nil
	}
	return json.Marshal(r.Target)
}

// SetTargetFromJSON sets the target from JSON data.
func (r *DockerRestore) SetTargetFromJSON(data []byte) error {
	if len(data) == 0 {
		r.Target = nil
		return nil
	}
	var target DockerRestoreTarget
	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}
	r.Target = &target
	return nil
}

// ProgressJSON returns the progress as JSON for database storage.
func (r *DockerRestore) ProgressJSON() ([]byte, error) {
	if r.Progress == nil {
		return nil, nil
	}
	return json.Marshal(r.Progress)
}

// SetProgressFromJSON sets the progress from JSON data.
func (r *DockerRestore) SetProgressFromJSON(data []byte) error {
	if len(data) == 0 {
		r.Progress = nil
		return nil
	}
	var progress DockerRestoreProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return err
	}
	r.Progress = &progress
	return nil
}

// RestoredVolumesJSON returns the restored volumes as JSON for database storage.
func (r *DockerRestore) RestoredVolumesJSON() ([]byte, error) {
	if len(r.RestoredVolumes) == 0 {
		return nil, nil
	}
	return json.Marshal(r.RestoredVolumes)
}

// SetRestoredVolumesFromJSON sets the restored volumes from JSON data.
func (r *DockerRestore) SetRestoredVolumesFromJSON(data []byte) error {
	if len(data) == 0 {
		r.RestoredVolumes = nil
		return nil
	}
	return json.Unmarshal(data, &r.RestoredVolumes)
}

// WarningsJSON returns the warnings as JSON for database storage.
func (r *DockerRestore) WarningsJSON() ([]byte, error) {
	if len(r.Warnings) == 0 {
		return nil, nil
	}
	return json.Marshal(r.Warnings)
}

// SetWarningsFromJSON sets the warnings from JSON data.
func (r *DockerRestore) SetWarningsFromJSON(data []byte) error {
	if len(data) == 0 {
		r.Warnings = nil
		return nil
	}
	return json.Unmarshal(data, &r.Warnings)
}

// Start marks the restore as preparing.
func (r *DockerRestore) Start() {
	now := time.Now()
	r.StartedAt = &now
	r.Status = DockerRestoreStatusPreparing
	r.UpdatedAt = now
}

// UpdateStatus updates the restore status.
func (r *DockerRestore) UpdateStatus(status DockerRestoreStatus) {
	r.Status = status
	r.UpdatedAt = time.Now()
}

// UpdateProgress updates the restore progress.
func (r *DockerRestore) UpdateProgress(progress *DockerRestoreProgress) {
	r.Progress = progress
	r.UpdatedAt = time.Now()
}

// Complete marks the restore as completed successfully.
func (r *DockerRestore) Complete(containerID string, volumes []string, startVerified bool, warnings []string) {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = DockerRestoreStatusCompleted
	r.RestoredContainerID = containerID
	r.RestoredVolumes = volumes
	r.StartVerified = startVerified
	r.Warnings = warnings
	r.UpdatedAt = now
}

// Fail marks the restore as failed with the given error message.
func (r *DockerRestore) Fail(errMsg string) {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = DockerRestoreStatusFailed
	r.ErrorMessage = errMsg
	r.UpdatedAt = now
}

// Cancel marks the restore as canceled.
func (r *DockerRestore) Cancel() {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = DockerRestoreStatusCanceled
	r.UpdatedAt = now
}

// Duration returns the duration of the restore, or zero if not started/completed.
func (r *DockerRestore) Duration() time.Duration {
	if r.StartedAt == nil || r.CompletedAt == nil {
		return 0
	}
	return r.CompletedAt.Sub(*r.StartedAt)
}

// IsComplete returns true if the restore has finished.
func (r *DockerRestore) IsComplete() bool {
	return r.Status == DockerRestoreStatusCompleted ||
		r.Status == DockerRestoreStatusFailed ||
		r.Status == DockerRestoreStatusCanceled
}

// IsContainerRestore returns true if this restore includes a container.
func (r *DockerRestore) IsContainerRestore() bool {
	return r.ContainerName != ""
}

// IsVolumeOnlyRestore returns true if this restore is for volumes only.
func (r *DockerRestore) IsVolumeOnlyRestore() bool {
	return r.ContainerName == "" && r.VolumeName != ""
}

// DockerRestorePlan represents a preview of what will be restored.
type DockerRestorePlan struct {
	Container      *DockerRestoreContainerInfo `json:"container,omitempty"`
	Volumes        []DockerRestoreVolumeInfo   `json:"volumes,omitempty"`
	TotalSizeBytes int64                       `json:"total_size_bytes"`
	Conflicts      []DockerRestoreConflict     `json:"conflicts,omitempty"`
	Dependencies   []string                    `json:"dependencies,omitempty"`
}
