package models

import (
	"time"

	"github.com/google/uuid"
)

// ReplicationStatusType represents the status of a replication operation.
type ReplicationStatusType string

const (
	// ReplicationStatusPending indicates replication has not started.
	ReplicationStatusPending ReplicationStatusType = "pending"
	// ReplicationStatusSyncing indicates replication is in progress.
	ReplicationStatusSyncing ReplicationStatusType = "syncing"
	// ReplicationStatusSynced indicates replication completed successfully.
	ReplicationStatusSynced ReplicationStatusType = "synced"
	// ReplicationStatusFailed indicates replication failed.
	ReplicationStatusFailed ReplicationStatusType = "failed"
)

// ReplicationStatus tracks the sync status between two repositories for a schedule.
type ReplicationStatus struct {
	ID                 uuid.UUID             `json:"id"`
	ScheduleID         uuid.UUID             `json:"schedule_id"`
	SourceRepositoryID uuid.UUID             `json:"source_repository_id"`
	TargetRepositoryID uuid.UUID             `json:"target_repository_id"`
	LastSnapshotID     *string               `json:"last_snapshot_id,omitempty"`
	LastSyncAt         *time.Time            `json:"last_sync_at,omitempty"`
	Status             ReplicationStatusType `json:"status"`
	ErrorMessage       *string               `json:"error_message,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

// NewReplicationStatus creates a new ReplicationStatus record.
func NewReplicationStatus(scheduleID, sourceRepoID, targetRepoID uuid.UUID) *ReplicationStatus {
	now := time.Now()
	return &ReplicationStatus{
		ID:                 uuid.New(),
		ScheduleID:         scheduleID,
		SourceRepositoryID: sourceRepoID,
		TargetRepositoryID: targetRepoID,
		Status:             ReplicationStatusPending,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// MarkSyncing marks the replication as in progress.
func (rs *ReplicationStatus) MarkSyncing() {
	rs.Status = ReplicationStatusSyncing
	rs.UpdatedAt = time.Now()
}

// MarkSynced marks the replication as completed successfully.
func (rs *ReplicationStatus) MarkSynced(snapshotID string) {
	now := time.Now()
	rs.Status = ReplicationStatusSynced
	rs.LastSnapshotID = &snapshotID
	rs.LastSyncAt = &now
	rs.ErrorMessage = nil
	rs.UpdatedAt = now
}

// MarkFailed marks the replication as failed with an error message.
func (rs *ReplicationStatus) MarkFailed(errMsg string) {
	rs.Status = ReplicationStatusFailed
	rs.ErrorMessage = &errMsg
	rs.UpdatedAt = time.Now()
}

// IsSynced returns true if the replication is in a synced state.
func (rs *ReplicationStatus) IsSynced() bool {
	return rs.Status == ReplicationStatusSynced
}
