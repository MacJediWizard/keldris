package models

import (
	"time"

	"github.com/google/uuid"
)

// S3ObjectLockMode defines the S3 Object Lock retention mode.
type S3ObjectLockMode string

const (
	// S3ObjectLockModeGovernance allows users with special permissions to bypass the lock.
	S3ObjectLockModeGovernance S3ObjectLockMode = "GOVERNANCE"
	// S3ObjectLockModeCompliance prevents any user from deleting the object.
	S3ObjectLockModeCompliance S3ObjectLockMode = "COMPLIANCE"
)

// SnapshotImmutability represents an immutability lock on a snapshot.
type SnapshotImmutability struct {
	ID                  uuid.UUID         `json:"id"`
	OrgID               uuid.UUID         `json:"org_id"`
	RepositoryID        uuid.UUID         `json:"repository_id"`
	SnapshotID          string            `json:"snapshot_id"`
	ShortID             string            `json:"short_id"`
	LockedAt            time.Time         `json:"locked_at"`
	LockedUntil         time.Time         `json:"locked_until"`
	LockedBy            *uuid.UUID        `json:"locked_by,omitempty"`
	Reason              string            `json:"reason,omitempty"`
	S3ObjectLockEnabled bool              `json:"s3_object_lock_enabled"`
	S3ObjectLockMode    *S3ObjectLockMode `json:"s3_object_lock_mode,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// NewSnapshotImmutability creates a new SnapshotImmutability.
func NewSnapshotImmutability(
	orgID uuid.UUID,
	repositoryID uuid.UUID,
	snapshotID string,
	shortID string,
	lockedUntil time.Time,
	lockedBy *uuid.UUID,
	reason string,
) *SnapshotImmutability {
	now := time.Now()
	return &SnapshotImmutability{
		ID:           uuid.New(),
		OrgID:        orgID,
		RepositoryID: repositoryID,
		SnapshotID:   snapshotID,
		ShortID:      shortID,
		LockedAt:     now,
		LockedUntil:  lockedUntil,
		LockedBy:     lockedBy,
		Reason:       reason,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// IsLocked returns true if the snapshot is currently locked (not expired).
func (s *SnapshotImmutability) IsLocked() bool {
	return time.Now().Before(s.LockedUntil)
}

// RemainingDays returns the number of days remaining in the immutability period.
func (s *SnapshotImmutability) RemainingDays() int {
	if !s.IsLocked() {
		return 0
	}
	remaining := s.LockedUntil.Sub(time.Now())
	return int(remaining.Hours() / 24)
}

// SetS3ObjectLock configures S3 Object Lock settings.
func (s *SnapshotImmutability) SetS3ObjectLock(mode S3ObjectLockMode) {
	s.S3ObjectLockEnabled = true
	s.S3ObjectLockMode = &mode
	s.UpdatedAt = time.Now()
}

// CreateSnapshotImmutabilityRequest is the request to create an immutability lock.
type CreateSnapshotImmutabilityRequest struct {
	SnapshotID   string `json:"snapshot_id" binding:"required"`
	Days         int    `json:"days" binding:"required,min=1,max=36500"`
	Reason       string `json:"reason" binding:"max=500"`
	EnableS3Lock bool   `json:"enable_s3_lock"`
}

// UpdateSnapshotImmutabilityRequest is the request to extend an immutability lock.
type UpdateSnapshotImmutabilityRequest struct {
	Days   int    `json:"days" binding:"required,min=1,max=36500"`
	Reason string `json:"reason" binding:"max=500"`
}

// RepositoryImmutabilitySettings holds immutability configuration for a repository.
type RepositoryImmutabilitySettings struct {
	Enabled     bool `json:"enabled"`
	DefaultDays *int `json:"default_days,omitempty"`
}
