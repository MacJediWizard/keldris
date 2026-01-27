package models

import (
	"time"

	"github.com/google/uuid"
)

// RecentItemType represents the type of item that was recently viewed.
type RecentItemType string

const (
	RecentItemTypeAgent      RecentItemType = "agent"
	RecentItemTypeRepository RecentItemType = "repository"
	RecentItemTypeSchedule   RecentItemType = "schedule"
	RecentItemTypeBackup     RecentItemType = "backup"
	RecentItemTypePolicy     RecentItemType = "policy"
	RecentItemTypeSnapshot   RecentItemType = "snapshot"
)

// RecentItem represents a recently viewed item for quick access.
type RecentItem struct {
	ID        uuid.UUID      `json:"id"`
	OrgID     uuid.UUID      `json:"org_id"`
	UserID    uuid.UUID      `json:"user_id"`
	ItemType  RecentItemType `json:"item_type"`
	ItemID    uuid.UUID      `json:"item_id"`
	ItemName  string         `json:"item_name"`
	PagePath  string         `json:"page_path"`
	ViewedAt  time.Time      `json:"viewed_at"`
	CreatedAt time.Time      `json:"created_at"`
}

// NewRecentItem creates a new RecentItem entry.
func NewRecentItem(orgID, userID uuid.UUID, itemType RecentItemType, itemID uuid.UUID, itemName, pagePath string) *RecentItem {
	now := time.Now()
	return &RecentItem{
		ID:        uuid.New(),
		OrgID:     orgID,
		UserID:    userID,
		ItemType:  itemType,
		ItemID:    itemID,
		ItemName:  itemName,
		PagePath:  pagePath,
		ViewedAt:  now,
		CreatedAt: now,
	}
}

// ValidItemTypes returns all valid recent item types.
func ValidItemTypes() []RecentItemType {
	return []RecentItemType{
		RecentItemTypeAgent,
		RecentItemTypeRepository,
		RecentItemTypeSchedule,
		RecentItemTypeBackup,
		RecentItemTypePolicy,
		RecentItemTypeSnapshot,
	}
}

// IsValidItemType checks if the given type is a valid recent item type.
func IsValidItemType(t string) bool {
	switch RecentItemType(t) {
	case RecentItemTypeAgent, RecentItemTypeRepository, RecentItemTypeSchedule,
		RecentItemTypeBackup, RecentItemTypePolicy, RecentItemTypeSnapshot:
		return true
	}
	return false
}
