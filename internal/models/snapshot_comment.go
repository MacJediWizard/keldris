package models

import (
	"time"

	"github.com/google/uuid"
)

// SnapshotComment represents a user comment on a snapshot.
type SnapshotComment struct {
	ID         uuid.UUID `json:"id"`
	OrgID      uuid.UUID `json:"org_id"`
	SnapshotID string    `json:"snapshot_id"`
	UserID     uuid.UUID `json:"user_id"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewSnapshotComment creates a new SnapshotComment.
func NewSnapshotComment(orgID uuid.UUID, snapshotID string, userID uuid.UUID, content string) *SnapshotComment {
	now := time.Now()
	return &SnapshotComment{
		ID:         uuid.New(),
		OrgID:      orgID,
		SnapshotID: snapshotID,
		UserID:     userID,
		Content:    content,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateSnapshotCommentRequest represents the request body for creating a comment.
type CreateSnapshotCommentRequest struct {
	Content string `json:"content" binding:"required"`
}
