package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DockerImageBackupStatus represents the status of a Docker image backup.
type DockerImageBackupStatus string

const (
	// DockerImageBackupStatusPending indicates the backup is pending.
	DockerImageBackupStatusPending DockerImageBackupStatus = "pending"
	// DockerImageBackupStatusRunning indicates the backup is in progress.
	DockerImageBackupStatusRunning DockerImageBackupStatus = "running"
	// DockerImageBackupStatusCompleted indicates the backup completed successfully.
	DockerImageBackupStatusCompleted DockerImageBackupStatus = "completed"
	// DockerImageBackupStatusFailed indicates the backup failed.
	DockerImageBackupStatusFailed DockerImageBackupStatus = "failed"
)

// DockerImageBackupMode defines how images should be backed up.
type DockerImageBackupMode string

const (
	// DockerImageBackupModeAll backs up all images on the system.
	DockerImageBackupModeAll DockerImageBackupMode = "all"
	// DockerImageBackupModeContainers backs up only images used by containers.
	DockerImageBackupModeContainers DockerImageBackupMode = "containers"
	// DockerImageBackupModeCustomOnly backs up only custom (non-public) images.
	DockerImageBackupModeCustomOnly DockerImageBackupMode = "custom_only"
	// DockerImageBackupModeSelected backs up only selected images.
	DockerImageBackupModeSelected DockerImageBackupMode = "selected"
)

// DockerImageBackupJob represents a Docker image backup job record.
type DockerImageBackupJob struct {
	ID                 uuid.UUID               `json:"id"`
	OrgID              uuid.UUID               `json:"org_id"`
	AgentID            uuid.UUID               `json:"agent_id"`
	ScheduleID         *uuid.UUID              `json:"schedule_id,omitempty"`
	Status             DockerImageBackupStatus `json:"status"`
	Mode               DockerImageBackupMode   `json:"mode"`
	BackupPath         string                  `json:"backup_path"`
	ImagesBackedUp     int                     `json:"images_backed_up"`
	ImagesSkipped      int                     `json:"images_skipped"`
	ImagesDeduplicated int                     `json:"images_deduplicated"`
	TotalSizeBytes     int64                   `json:"total_size_bytes"`
	ErrorMessage       string                  `json:"error_message,omitempty"`
	StartedAt          time.Time               `json:"started_at"`
	CompletedAt        *time.Time              `json:"completed_at,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
}

// NewDockerImageBackupJob creates a new Docker image backup record.
func NewDockerImageBackupJob(orgID, agentID uuid.UUID, mode DockerImageBackupMode) *DockerImageBackupJob {
	now := time.Now()
	return &DockerImageBackupJob{
		ID:        uuid.New(),
		OrgID:     orgID,
		AgentID:   agentID,
		Status:    DockerImageBackupStatusPending,
		Mode:      mode,
		StartedAt: now,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Start marks the backup as running.
func (b *DockerImageBackupJob) Start() {
	b.Status = DockerImageBackupStatusRunning
	b.StartedAt = time.Now()
	b.UpdatedAt = time.Now()
}

// Complete marks the backup as completed.
func (b *DockerImageBackupJob) Complete(backupPath string, backed, skipped, deduplicated int, sizeBytes int64) {
	now := time.Now()
	b.Status = DockerImageBackupStatusCompleted
	b.BackupPath = backupPath
	b.ImagesBackedUp = backed
	b.ImagesSkipped = skipped
	b.ImagesDeduplicated = deduplicated
	b.TotalSizeBytes = sizeBytes
	b.CompletedAt = &now
	b.UpdatedAt = now
}

// Fail marks the backup as failed.
func (b *DockerImageBackupJob) Fail(errMsg string) {
	now := time.Now()
	b.Status = DockerImageBackupStatusFailed
	b.ErrorMessage = errMsg
	b.CompletedAt = &now
	b.UpdatedAt = now
}

// Duration returns the duration of the backup.
func (b *DockerImageBackupJob) Duration() time.Duration {
	if b.CompletedAt == nil {
		return time.Since(b.StartedAt)
	}
	return b.CompletedAt.Sub(b.StartedAt)
}

// DockerImageVersion tracks a specific version of an image per container.
type DockerImageVersion struct {
	ID             uuid.UUID  `json:"id"`
	BackupID       uuid.UUID  `json:"backup_id"`
	ContainerID    string     `json:"container_id"`
	ContainerName  string     `json:"container_name"`
	ImageID        string     `json:"image_id"`
	ImageTag       string     `json:"image_tag"`
	ImageDigest    string     `json:"image_digest,omitempty"`
	SizeBytes      int64      `json:"size_bytes"`
	Checksum       string     `json:"checksum,omitempty"`
	BackupPath     string     `json:"backup_path"`
	IsDeduplicated bool       `json:"is_deduplicated"`
	DeduplicatedOf *string    `json:"deduplicated_of,omitempty"` // Original backup path if deduplicated
	CreatedAt      time.Time  `json:"created_at"`
}

// NewDockerImageVersion creates a new Docker image version record.
func NewDockerImageVersion(backupID uuid.UUID, containerID, containerName, imageID, imageTag string) *DockerImageVersion {
	return &DockerImageVersion{
		ID:            uuid.New(),
		BackupID:      backupID,
		ContainerID:   containerID,
		ContainerName: containerName,
		ImageID:       imageID,
		ImageTag:      imageTag,
		CreatedAt:     time.Now(),
	}
}

// MarkDeduplicated marks this version as deduplicated from another backup.
func (v *DockerImageVersion) MarkDeduplicated(originalPath string) {
	v.IsDeduplicated = true
	v.DeduplicatedOf = &originalPath
}

// DockerImageScheduleOptions contains Docker-specific scheduling options.
type DockerImageScheduleOptions struct {
	// BackupImages enables image backup for this schedule.
	BackupImages bool `json:"backup_images"`

	// ImageBackupMode determines which images to backup.
	ImageBackupMode DockerImageBackupMode `json:"image_backup_mode"`

	// ExcludePublicImages skips backing up images from public registries.
	ExcludePublicImages bool `json:"exclude_public_images"`

	// SelectedImages is a list of image names/IDs to backup (when mode is "selected").
	SelectedImages []string `json:"selected_images,omitempty"`

	// CustomRegistries are additional registries to consider as private.
	CustomRegistries []string `json:"custom_registries,omitempty"`

	// RestoreImagesFirst ensures images are restored before containers on restore.
	RestoreImagesFirst bool `json:"restore_images_first"`

	// ImageRetentionDays is how long to keep image backups.
	ImageRetentionDays int `json:"image_retention_days"`
}

// DefaultDockerImageScheduleOptions returns default options.
func DefaultDockerImageScheduleOptions() DockerImageScheduleOptions {
	return DockerImageScheduleOptions{
		BackupImages:        false,
		ImageBackupMode:     DockerImageBackupModeContainers,
		ExcludePublicImages: false,
		RestoreImagesFirst:  true,
		ImageRetentionDays:  30,
	}
}

// SetSelectedImages sets the selected images from JSON bytes.
func (o *DockerImageScheduleOptions) SetSelectedImages(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &o.SelectedImages)
}

// SelectedImagesJSON returns the selected images as JSON bytes.
func (o *DockerImageScheduleOptions) SelectedImagesJSON() ([]byte, error) {
	if o.SelectedImages == nil {
		return nil, nil
	}
	return json.Marshal(o.SelectedImages)
}

// SetCustomRegistries sets custom registries from JSON bytes.
func (o *DockerImageScheduleOptions) SetCustomRegistries(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &o.CustomRegistries)
}

// CustomRegistriesJSON returns custom registries as JSON bytes.
func (o *DockerImageScheduleOptions) CustomRegistriesJSON() ([]byte, error) {
	if o.CustomRegistries == nil {
		return nil, nil
	}
	return json.Marshal(o.CustomRegistries)
}

// DockerPrivateRegistry represents a private Docker registry to backup.
type DockerPrivateRegistry struct {
	ID           uuid.UUID `json:"id"`
	OrgID        uuid.UUID `json:"org_id"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Username     string    `json:"username,omitempty"`
	PasswordHash []byte    `json:"-"` // Encrypted, never expose
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewDockerPrivateRegistry creates a new private registry record.
func NewDockerPrivateRegistry(orgID uuid.UUID, name, url string) *DockerPrivateRegistry {
	now := time.Now()
	return &DockerPrivateRegistry{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		URL:       url,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// DockerRegistryBackup represents a backup of a private registry.
type DockerRegistryBackup struct {
	ID             uuid.UUID               `json:"id"`
	RegistryID     uuid.UUID               `json:"registry_id"`
	Status         DockerImageBackupStatus `json:"status"`
	BackupPath     string                  `json:"backup_path"`
	ImagesCount    int                     `json:"images_count"`
	TotalSizeBytes int64                   `json:"total_size_bytes"`
	ErrorMessage   string                  `json:"error_message,omitempty"`
	StartedAt      time.Time               `json:"started_at"`
	CompletedAt    *time.Time              `json:"completed_at,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
}

// NewDockerRegistryBackup creates a new registry backup record.
func NewDockerRegistryBackup(registryID uuid.UUID) *DockerRegistryBackup {
	now := time.Now()
	return &DockerRegistryBackup{
		ID:         uuid.New(),
		RegistryID: registryID,
		Status:     DockerImageBackupStatusPending,
		StartedAt:  now,
		CreatedAt:  now,
	}
}

// Complete marks the registry backup as completed.
func (b *DockerRegistryBackup) Complete(backupPath string, imagesCount int, sizeBytes int64) {
	now := time.Now()
	b.Status = DockerImageBackupStatusCompleted
	b.BackupPath = backupPath
	b.ImagesCount = imagesCount
	b.TotalSizeBytes = sizeBytes
	b.CompletedAt = &now
}

// Fail marks the registry backup as failed.
func (b *DockerRegistryBackup) Fail(errMsg string) {
	now := time.Now()
	b.Status = DockerImageBackupStatusFailed
	b.ErrorMessage = errMsg
	b.CompletedAt = &now
}

// DockerImageDeduplicationEntry tracks deduplicated images across backups.
type DockerImageDeduplicationEntry struct {
	ID               uuid.UUID `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	ImageID          string    `json:"image_id"`
	Checksum         string    `json:"checksum"`
	OriginalBackupID uuid.UUID `json:"original_backup_id"`
	OriginalPath     string    `json:"original_path"`
	SizeBytes        int64     `json:"size_bytes"`
	ReferenceCount   int       `json:"reference_count"`
	FirstSeenAt      time.Time `json:"first_seen_at"`
	LastSeenAt       time.Time `json:"last_seen_at"`
}

// NewDockerImageDeduplicationEntry creates a new deduplication entry.
func NewDockerImageDeduplicationEntry(orgID uuid.UUID, imageID, checksum string, backupID uuid.UUID, path string, size int64) *DockerImageDeduplicationEntry {
	now := time.Now()
	return &DockerImageDeduplicationEntry{
		ID:               uuid.New(),
		OrgID:            orgID,
		ImageID:          imageID,
		Checksum:         checksum,
		OriginalBackupID: backupID,
		OriginalPath:     path,
		SizeBytes:        size,
		ReferenceCount:   1,
		FirstSeenAt:      now,
		LastSeenAt:       now,
	}
}

// IncrementReference increases the reference count and updates last seen time.
func (e *DockerImageDeduplicationEntry) IncrementReference() {
	e.ReferenceCount++
	e.LastSeenAt = time.Now()
}

// DecrementReference decreases the reference count.
func (e *DockerImageDeduplicationEntry) DecrementReference() {
	if e.ReferenceCount > 0 {
		e.ReferenceCount--
	}
}

// DockerImageSchedule represents a schedule for Docker image backups.
type DockerImageSchedule struct {
	ID             uuid.UUID                  `json:"id"`
	OrgID          uuid.UUID                  `json:"org_id"`
	AgentID        uuid.UUID                  `json:"agent_id"`
	Name           string                     `json:"name"`
	CronExpression string                     `json:"cron_expression"`
	Options        DockerImageScheduleOptions `json:"options"`
	Enabled        bool                       `json:"enabled"`
	CreatedAt      time.Time                  `json:"created_at"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

// NewDockerImageSchedule creates a new Docker image schedule.
func NewDockerImageSchedule(orgID, agentID uuid.UUID, name, cronExpr string) *DockerImageSchedule {
	now := time.Now()
	return &DockerImageSchedule{
		ID:             uuid.New(),
		OrgID:          orgID,
		AgentID:        agentID,
		Name:           name,
		CronExpression: cronExpr,
		Options:        DefaultDockerImageScheduleOptions(),
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// SetOptions sets the schedule options from JSON bytes.
func (s *DockerImageSchedule) SetOptions(data []byte) error {
	if len(data) == 0 {
		s.Options = DefaultDockerImageScheduleOptions()
		return nil
	}
	return json.Unmarshal(data, &s.Options)
}

// OptionsJSON returns the options as JSON bytes.
func (s *DockerImageSchedule) OptionsJSON() ([]byte, error) {
	return json.Marshal(s.Options)
}
