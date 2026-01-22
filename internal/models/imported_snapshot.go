package models

import (
	"time"

	"github.com/google/uuid"
)

// ImportedSnapshot represents a snapshot that was imported from an existing Restic repository.
type ImportedSnapshot struct {
	ID               uuid.UUID  `json:"id"`
	RepositoryID     uuid.UUID  `json:"repository_id"`
	AgentID          *uuid.UUID `json:"agent_id,omitempty"`
	ResticSnapshotID string     `json:"restic_snapshot_id"`
	ShortID          string     `json:"short_id"`
	Hostname         string     `json:"hostname"`
	Username         string     `json:"username,omitempty"`
	SnapshotTime     time.Time  `json:"snapshot_time"`
	Paths            []string   `json:"paths"`
	Tags             []string   `json:"tags,omitempty"`
	ImportedAt       time.Time  `json:"imported_at"`
	CreatedAt        time.Time  `json:"created_at"`
}

// NewImportedSnapshot creates a new ImportedSnapshot.
func NewImportedSnapshot(
	repositoryID uuid.UUID,
	agentID *uuid.UUID,
	resticSnapshotID string,
	shortID string,
	hostname string,
	username string,
	snapshotTime time.Time,
	paths []string,
	tags []string,
) *ImportedSnapshot {
	now := time.Now()
	return &ImportedSnapshot{
		ID:               uuid.New(),
		RepositoryID:     repositoryID,
		AgentID:          agentID,
		ResticSnapshotID: resticSnapshotID,
		ShortID:          shortID,
		Hostname:         hostname,
		Username:         username,
		SnapshotTime:     snapshotTime,
		Paths:            paths,
		Tags:             tags,
		ImportedAt:       now,
		CreatedAt:        now,
	}
}
