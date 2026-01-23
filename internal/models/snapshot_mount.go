package models

import (
	"time"

	"github.com/google/uuid"
)

// SnapshotMountStatus represents the current status of a snapshot mount.
type SnapshotMountStatus string

const (
	// SnapshotMountStatusPending indicates the mount is queued.
	SnapshotMountStatusPending SnapshotMountStatus = "pending"
	// SnapshotMountStatusMounting indicates the mount is in progress.
	SnapshotMountStatusMounting SnapshotMountStatus = "mounting"
	// SnapshotMountStatusMounted indicates the snapshot is mounted and accessible.
	SnapshotMountStatusMounted SnapshotMountStatus = "mounted"
	// SnapshotMountStatusUnmounting indicates the unmount is in progress.
	SnapshotMountStatusUnmounting SnapshotMountStatus = "unmounting"
	// SnapshotMountStatusUnmounted indicates the snapshot has been unmounted.
	SnapshotMountStatusUnmounted SnapshotMountStatus = "unmounted"
	// SnapshotMountStatusFailed indicates the mount/unmount operation failed.
	SnapshotMountStatusFailed SnapshotMountStatus = "failed"
)

// DefaultMountTimeout is the default duration before a mount expires.
const DefaultMountTimeout = 30 * time.Minute

// SnapshotMount represents a FUSE-mounted snapshot for browsing.
type SnapshotMount struct {
	ID           uuid.UUID   `json:"id"`
	OrgID        uuid.UUID   `json:"org_id"`
	AgentID      uuid.UUID   `json:"agent_id"`
	RepositoryID uuid.UUID   `json:"repository_id"`
	SnapshotID   string      `json:"snapshot_id"`
	MountPath    string      `json:"mount_path"`
	Status       SnapshotMountStatus `json:"status"`
	MountedAt    *time.Time  `json:"mounted_at,omitempty"`
	ExpiresAt    *time.Time  `json:"expires_at,omitempty"`
	UnmountedAt  *time.Time  `json:"unmounted_at,omitempty"`
	ErrorMessage string      `json:"error_message,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// NewSnapshotMount creates a new SnapshotMount with default values.
func NewSnapshotMount(orgID, agentID, repositoryID uuid.UUID, snapshotID, mountPath string) *SnapshotMount {
	now := time.Now()
	return &SnapshotMount{
		ID:           uuid.New(),
		OrgID:        orgID,
		AgentID:      agentID,
		RepositoryID: repositoryID,
		SnapshotID:   snapshotID,
		MountPath:    mountPath,
		Status:       SnapshotMountStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// StartMounting marks the mount as in progress.
func (m *SnapshotMount) StartMounting() {
	m.Status = SnapshotMountStatusMounting
	m.UpdatedAt = time.Now()
}

// MarkMounted marks the mount as successfully mounted with an expiration time.
func (m *SnapshotMount) MarkMounted(timeout time.Duration) {
	now := time.Now()
	expires := now.Add(timeout)
	m.MountedAt = &now
	m.ExpiresAt = &expires
	m.Status = SnapshotMountStatusMounted
	m.UpdatedAt = now
}

// StartUnmounting marks the mount as unmounting.
func (m *SnapshotMount) StartUnmounting() {
	m.Status = SnapshotMountStatusUnmounting
	m.UpdatedAt = time.Now()
}

// MarkUnmounted marks the mount as successfully unmounted.
func (m *SnapshotMount) MarkUnmounted() {
	now := time.Now()
	m.UnmountedAt = &now
	m.Status = SnapshotMountStatusUnmounted
	m.UpdatedAt = now
}

// Fail marks the mount as failed with the given error message.
func (m *SnapshotMount) Fail(errMsg string) {
	m.Status = SnapshotMountStatusFailed
	m.ErrorMessage = errMsg
	m.UpdatedAt = time.Now()
}

// ExtendExpiry extends the mount expiration time.
func (m *SnapshotMount) ExtendExpiry(duration time.Duration) {
	if m.ExpiresAt != nil {
		newExpiry := m.ExpiresAt.Add(duration)
		m.ExpiresAt = &newExpiry
		m.UpdatedAt = time.Now()
	}
}

// IsActive returns true if the mount is currently active.
func (m *SnapshotMount) IsActive() bool {
	return m.Status == SnapshotMountStatusPending ||
		m.Status == SnapshotMountStatusMounting ||
		m.Status == SnapshotMountStatusMounted
}

// IsExpired returns true if the mount has expired.
func (m *SnapshotMount) IsExpired() bool {
	if m.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*m.ExpiresAt)
}

// CreateSnapshotMountRequest is the request body for creating a mount.
type CreateSnapshotMountRequest struct {
	TimeoutMinutes int `json:"timeout_minutes,omitempty"`
}
