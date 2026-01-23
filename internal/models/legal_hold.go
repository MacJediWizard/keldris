package models

import (
	"time"

	"github.com/google/uuid"
)

// LegalHold represents a legal hold placed on a snapshot to prevent deletion.
type LegalHold struct {
	ID         uuid.UUID `json:"id"`
	OrgID      uuid.UUID `json:"org_id"`
	SnapshotID string    `json:"snapshot_id"`
	Reason     string    `json:"reason"`
	PlacedBy   uuid.UUID `json:"placed_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewLegalHold creates a new LegalHold.
func NewLegalHold(orgID uuid.UUID, snapshotID string, reason string, placedBy uuid.UUID) *LegalHold {
	now := time.Now()
	return &LegalHold{
		ID:         uuid.New(),
		OrgID:      orgID,
		SnapshotID: snapshotID,
		Reason:     reason,
		PlacedBy:   placedBy,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateLegalHoldRequest represents the request body for creating a legal hold.
type CreateLegalHoldRequest struct {
	Reason string `json:"reason" binding:"required"`
}
