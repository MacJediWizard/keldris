package models

import (
	"time"

	"github.com/google/uuid"
)

// Tag represents a label that can be applied to backups and snapshots.
type Tag struct {
	ID        uuid.UUID `json:"id"`
	OrgID     uuid.UUID `json:"org_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BackupTag represents an association between a backup and a tag.
type BackupTag struct {
	ID        uuid.UUID `json:"id"`
	BackupID  uuid.UUID `json:"backup_id"`
	TagID     uuid.UUID `json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

// SnapshotTag represents an association between a snapshot and a tag.
type SnapshotTag struct {
	ID         uuid.UUID `json:"id"`
	SnapshotID string    `json:"snapshot_id"`
	TagID      uuid.UUID `json:"tag_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewTag creates a new Tag with the given name and color.
func NewTag(orgID uuid.UUID, name, color string) *Tag {
	now := time.Now()
	if color == "" {
		color = "#6366f1" // Default indigo color
	}
	return &Tag{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Color:     color,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CreateTagRequest represents a request to create a new tag.
type CreateTagRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=100"`
	Color string `json:"color" binding:"omitempty,len=7"`
}

// UpdateTagRequest represents a request to update a tag.
type UpdateTagRequest struct {
	Name  *string `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Color *string `json:"color,omitempty" binding:"omitempty,len=7"`
}

// AssignTagsRequest represents a request to assign tags to a backup or snapshot.
type AssignTagsRequest struct {
	TagIDs []uuid.UUID `json:"tag_ids" binding:"required"`
}
