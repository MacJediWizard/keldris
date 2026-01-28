package models

import (
	"time"

	"github.com/google/uuid"
)

// FavoriteEntityType represents the type of entity that can be favorited.
type FavoriteEntityType string

const (
	FavoriteEntityTypeAgent      FavoriteEntityType = "agent"
	FavoriteEntityTypeSchedule   FavoriteEntityType = "schedule"
	FavoriteEntityTypeRepository FavoriteEntityType = "repository"
)

// Favorite represents a user's favorited item.
type Favorite struct {
	ID         uuid.UUID          `json:"id"`
	UserID     uuid.UUID          `json:"user_id"`
	OrgID      uuid.UUID          `json:"org_id"`
	EntityType FavoriteEntityType `json:"entity_type"`
	EntityID   uuid.UUID          `json:"entity_id"`
	CreatedAt  time.Time          `json:"created_at"`
}

// NewFavorite creates a new Favorite with the given parameters.
func NewFavorite(userID, orgID, entityID uuid.UUID, entityType FavoriteEntityType) *Favorite {
	return &Favorite{
		ID:         uuid.New(),
		UserID:     userID,
		OrgID:      orgID,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  time.Now(),
	}
}

// CreateFavoriteRequest represents a request to create a new favorite.
type CreateFavoriteRequest struct {
	EntityType FavoriteEntityType `json:"entity_type" binding:"required"`
	EntityID   string             `json:"entity_id" binding:"required,uuid"`
}

// FavoriteWithEntity represents a favorite with its associated entity details.
type FavoriteWithEntity struct {
	Favorite
	EntityName string `json:"entity_name,omitempty"`
}
