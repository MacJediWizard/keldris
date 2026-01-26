package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// SavedFilter represents a saved dashboard filter configuration.
type SavedFilter struct {
	ID         uuid.UUID       `json:"id"`
	UserID     uuid.UUID       `json:"user_id"`
	OrgID      uuid.UUID       `json:"org_id"`
	Name       string          `json:"name"`
	EntityType string          `json:"entity_type"`
	Filters    json.RawMessage `json:"filters"`
	Shared     bool            `json:"shared"`
	IsDefault  bool            `json:"is_default"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

// NewSavedFilter creates a new SavedFilter with the given parameters.
func NewSavedFilter(userID, orgID uuid.UUID, name, entityType string, filters json.RawMessage) *SavedFilter {
	now := time.Now()
	if filters == nil {
		filters = json.RawMessage("{}")
	}
	return &SavedFilter{
		ID:         uuid.New(),
		UserID:     userID,
		OrgID:      orgID,
		Name:       name,
		EntityType: entityType,
		Filters:    filters,
		Shared:     false,
		IsDefault:  false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// CreateSavedFilterRequest represents a request to create a new saved filter.
type CreateSavedFilterRequest struct {
	Name       string          `json:"name" binding:"required,min=1,max=100"`
	EntityType string          `json:"entity_type" binding:"required,min=1,max=50"`
	Filters    json.RawMessage `json:"filters" binding:"required"`
	Shared     bool            `json:"shared"`
	IsDefault  bool            `json:"is_default"`
}

// UpdateSavedFilterRequest represents a request to update a saved filter.
type UpdateSavedFilterRequest struct {
	Name      *string          `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Filters   *json.RawMessage `json:"filters,omitempty"`
	Shared    *bool            `json:"shared,omitempty"`
	IsDefault *bool            `json:"is_default,omitempty"`
}
