package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RestoreStatus represents the current status of a restore operation.
type RestoreStatus string

const (
	// RestoreStatusPending indicates the restore is queued.
	RestoreStatusPending RestoreStatus = "pending"
	// RestoreStatusRunning indicates the restore is in progress.
	RestoreStatusRunning RestoreStatus = "running"
	// RestoreStatusCompleted indicates the restore completed successfully.
	RestoreStatusCompleted RestoreStatus = "completed"
	// RestoreStatusFailed indicates the restore failed.
	RestoreStatusFailed RestoreStatus = "failed"
	// RestoreStatusCanceled indicates the restore was canceled.
	RestoreStatusCanceled RestoreStatus = "canceled"
	// RestoreStatusUploading indicates the restore is uploading to cloud storage.
	RestoreStatusUploading RestoreStatus = "uploading"
	// RestoreStatusVerifying indicates the restore is verifying upload integrity.
	RestoreStatusVerifying RestoreStatus = "verifying"
)

// PathMapping represents a mapping from source path to target path for cross-agent restores.
type PathMapping struct {
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
}

// RestoreProgress tracks the progress of a restore operation.
type RestoreProgress struct {
	FilesRestored int64  `json:"files_restored"`
	BytesRestored int64  `json:"bytes_restored"`
	TotalFiles    *int64 `json:"total_files,omitempty"`
	TotalBytes    *int64 `json:"total_bytes,omitempty"`
	CurrentFile   string `json:"current_file,omitempty"`
}

// CloudRestoreTargetType represents the type of cloud storage target.
type CloudRestoreTargetType string

const (
	// CloudTargetS3 represents an S3-compatible storage target.
	CloudTargetS3 CloudRestoreTargetType = "s3"
	// CloudTargetB2 represents a Backblaze B2 storage target.
	CloudTargetB2 CloudRestoreTargetType = "b2"
	// CloudTargetRestic represents another Restic repository as a target.
	CloudTargetRestic CloudRestoreTargetType = "restic"
)

// CloudRestoreTarget represents the target cloud storage for a restore operation.
type CloudRestoreTarget struct {
	Type CloudRestoreTargetType `json:"type"`
	// S3/B2 configuration
	Bucket          string `json:"bucket,omitempty"`
	Prefix          string `json:"prefix,omitempty"`
	Region          string `json:"region,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	UseSSL          bool   `json:"use_ssl,omitempty"`
	// B2 specific
	AccountID      string `json:"account_id,omitempty"`
	ApplicationKey string `json:"application_key,omitempty"`
	// Restic repository configuration
	Repository         string `json:"repository,omitempty"`
	RepositoryPassword string `json:"repository_password,omitempty"`
}

// CloudRestoreProgress represents the progress of a cloud restore upload operation.
type CloudRestoreProgress struct {
	TotalFiles       int64  `json:"total_files"`
	TotalBytes       int64  `json:"total_bytes"`
	UploadedFiles    int64  `json:"uploaded_files"`
	UploadedBytes    int64  `json:"uploaded_bytes"`
	CurrentFile      string `json:"current_file,omitempty"`
	VerifiedChecksum bool   `json:"verified_checksum"`
}

// PercentComplete returns the upload completion percentage.
func (p *CloudRestoreProgress) PercentComplete() float64 {
	if p.TotalBytes == 0 {
		return 0
	}
	return float64(p.UploadedBytes) / float64(p.TotalBytes) * 100
}

// Restore represents a restore job execution record.
type Restore struct {
	ID           uuid.UUID     `json:"id"`
	AgentID      uuid.UUID     `json:"agent_id"`                       // Target agent (where restore executes)
	SourceAgentID *uuid.UUID   `json:"source_agent_id,omitempty"`      // Source agent for cross-agent restores
	RepositoryID uuid.UUID     `json:"repository_id"`
	SnapshotID   string        `json:"snapshot_id"`
	TargetPath   string        `json:"target_path"`
	IncludePaths []string      `json:"include_paths,omitempty"`
	ExcludePaths []string      `json:"exclude_paths,omitempty"`
	PathMappings []PathMapping `json:"path_mappings,omitempty"`        // Path remapping for cross-agent restores
	Status       RestoreStatus `json:"status"`
	Progress     *RestoreProgress `json:"progress,omitempty"`          // Real-time progress tracking
	StartedAt    *time.Time    `json:"started_at,omitempty"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	// Cloud restore fields
	CloudTarget         *CloudRestoreTarget   `json:"cloud_target,omitempty"`
	CloudProgress       *CloudRestoreProgress `json:"cloud_progress,omitempty"`
	CloudTargetLocation string                `json:"cloud_target_location,omitempty"`
	VerifyUpload        bool                  `json:"verify_upload,omitempty"`
}

// IsCloudRestore returns true if this is a cloud restore operation.
func (r *Restore) IsCloudRestore() bool {
	return r.CloudTarget != nil
}

// CloudTargetJSON returns the cloud target as JSON for database storage.
func (r *Restore) CloudTargetJSON() ([]byte, error) {
	if r.CloudTarget == nil {
		return nil, nil
	}
	return json.Marshal(r.CloudTarget)
}

// SetCloudTargetFromJSON sets the cloud target from JSON data.
func (r *Restore) SetCloudTargetFromJSON(data []byte) error {
	if len(data) == 0 {
		r.CloudTarget = nil
		return nil
	}
	var target CloudRestoreTarget
	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}
	r.CloudTarget = &target
	return nil
}

// CloudProgressJSON returns the cloud progress as JSON for database storage.
func (r *Restore) CloudProgressJSON() ([]byte, error) {
	if r.CloudProgress == nil {
		return nil, nil
	}
	return json.Marshal(r.CloudProgress)
}

// SetCloudProgressFromJSON sets the cloud progress from JSON data.
func (r *Restore) SetCloudProgressFromJSON(data []byte) error {
	if len(data) == 0 {
		r.CloudProgress = nil
		return nil
	}
	var progress CloudRestoreProgress
	if err := json.Unmarshal(data, &progress); err != nil {
		return err
	}
	r.CloudProgress = &progress
	return nil
}

// UpdateCloudProgress updates the cloud restore progress.
func (r *Restore) UpdateCloudProgress(progress *CloudRestoreProgress) {
	r.CloudProgress = progress
	r.UpdatedAt = time.Now()
}

// NewRestore creates a new Restore job.
func NewRestore(agentID, repositoryID uuid.UUID, snapshotID, targetPath string, includePaths, excludePaths []string) *Restore {
	now := time.Now()
	return &Restore{
		ID:           uuid.New(),
		AgentID:      agentID,
		RepositoryID: repositoryID,
		SnapshotID:   snapshotID,
		TargetPath:   targetPath,
		IncludePaths: includePaths,
		ExcludePaths: excludePaths,
		Status:       RestoreStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// NewCloudRestore creates a new Restore job with a cloud storage target.
func NewCloudRestore(agentID, repositoryID uuid.UUID, snapshotID string, includePaths, excludePaths []string, target *CloudRestoreTarget, verifyUpload bool) *Restore {
	now := time.Now()
	return &Restore{
		ID:           uuid.New(),
		AgentID:      agentID,
		RepositoryID: repositoryID,
		SnapshotID:   snapshotID,
		TargetPath:   "", // No local target path for cloud restore
		IncludePaths: includePaths,
		ExcludePaths: excludePaths,
		Status:       RestoreStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
		CloudTarget:  target,
		VerifyUpload: verifyUpload,
	}
}

// NewCrossRestore creates a new cross-agent Restore job.
func NewCrossRestore(sourceAgentID, targetAgentID, repositoryID uuid.UUID, snapshotID, targetPath string, includePaths, excludePaths []string, pathMappings []PathMapping) *Restore {
	now := time.Now()
	return &Restore{
		ID:            uuid.New(),
		AgentID:       targetAgentID,
		SourceAgentID: &sourceAgentID,
		RepositoryID:  repositoryID,
		SnapshotID:    snapshotID,
		TargetPath:    targetPath,
		IncludePaths:  includePaths,
		ExcludePaths:  excludePaths,
		PathMappings:  pathMappings,
		Status:        RestoreStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// Start marks the restore as running.
func (r *Restore) Start() {
	now := time.Now()
	r.StartedAt = &now
	r.Status = RestoreStatusRunning
	r.UpdatedAt = now
}

// StartUploading marks the restore as uploading to cloud storage.
func (r *Restore) StartUploading() {
	now := time.Now()
	r.Status = RestoreStatusUploading
	r.UpdatedAt = now
}

// StartVerifying marks the restore as verifying upload integrity.
func (r *Restore) StartVerifying() {
	now := time.Now()
	r.Status = RestoreStatusVerifying
	r.UpdatedAt = now
}

// Complete marks the restore as completed successfully.
func (r *Restore) Complete() {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = RestoreStatusCompleted
	r.UpdatedAt = now
}

// CompleteCloudRestore marks the cloud restore as completed with target location.
func (r *Restore) CompleteCloudRestore(targetLocation string, progress *CloudRestoreProgress) {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = RestoreStatusCompleted
	r.CloudTargetLocation = targetLocation
	r.CloudProgress = progress
	r.UpdatedAt = now
}

// Fail marks the restore as failed with the given error message.
func (r *Restore) Fail(errMsg string) {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = RestoreStatusFailed
	r.ErrorMessage = errMsg
	r.UpdatedAt = now
}

// Cancel marks the restore as canceled.
func (r *Restore) Cancel() {
	now := time.Now()
	r.CompletedAt = &now
	r.Status = RestoreStatusCanceled
	r.UpdatedAt = now
}

// Duration returns the duration of the restore, or zero if not started/completed.
func (r *Restore) Duration() time.Duration {
	if r.StartedAt == nil || r.CompletedAt == nil {
		return 0
	}
	return r.CompletedAt.Sub(*r.StartedAt)
}

// IsComplete returns true if the restore has finished.
func (r *Restore) IsComplete() bool {
	return r.Status == RestoreStatusCompleted ||
		r.Status == RestoreStatusFailed ||
		r.Status == RestoreStatusCanceled
}

// IsCrossAgentRestore returns true if this is a cross-agent restore.
func (r *Restore) IsCrossAgentRestore() bool {
	return r.SourceAgentID != nil && *r.SourceAgentID != r.AgentID
}

// GetSourceAgentID returns the source agent ID (same as AgentID for same-agent restores).
func (r *Restore) GetSourceAgentID() uuid.UUID {
	if r.SourceAgentID != nil {
		return *r.SourceAgentID
	}
	return r.AgentID
}

// UpdateProgress updates the restore progress.
func (r *Restore) UpdateProgress(filesRestored, bytesRestored int64, currentFile string) {
	if r.Progress == nil {
		r.Progress = &RestoreProgress{}
	}
	r.Progress.FilesRestored = filesRestored
	r.Progress.BytesRestored = bytesRestored
	r.Progress.CurrentFile = currentFile
	r.UpdatedAt = time.Now()
}

// SetTotalProgress sets the total files and bytes for progress tracking.
func (r *Restore) SetTotalProgress(totalFiles, totalBytes int64) {
	if r.Progress == nil {
		r.Progress = &RestoreProgress{}
	}
	r.Progress.TotalFiles = &totalFiles
	r.Progress.TotalBytes = &totalBytes
	r.UpdatedAt = time.Now()
}
