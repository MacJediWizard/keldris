package models

import (
	"time"

	"github.com/google/uuid"
)

// ScheduleRepository represents the relationship between a schedule and a repository
// with priority ordering for multi-repository backup support.
type ScheduleRepository struct {
	ID           uuid.UUID `json:"id"`
	ScheduleID   uuid.UUID `json:"schedule_id"`
	RepositoryID uuid.UUID `json:"repository_id"`
	Priority     int       `json:"priority"` // 0 = primary, 1+ = secondary by order
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

// NewScheduleRepository creates a new ScheduleRepository association.
func NewScheduleRepository(scheduleID, repositoryID uuid.UUID, priority int) *ScheduleRepository {
	return &ScheduleRepository{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		RepositoryID: repositoryID,
		Priority:     priority,
		Enabled:      true,
		CreatedAt:    time.Now(),
	}
}

// IsPrimary returns true if this is the primary repository (priority 0).
func (sr *ScheduleRepository) IsPrimary() bool {
	return sr.Priority == 0
}
