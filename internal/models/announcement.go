package models

import (
	"time"

	"github.com/google/uuid"
)

// AnnouncementType represents the type/severity of an announcement.
type AnnouncementType string

const (
	AnnouncementTypeInfo     AnnouncementType = "info"
	AnnouncementTypeWarning  AnnouncementType = "warning"
	AnnouncementTypeCritical AnnouncementType = "critical"
)

// Announcement represents a system-wide announcement.
type Announcement struct {
	ID          uuid.UUID        `json:"id"`
	OrgID       uuid.UUID        `json:"org_id"`
	Title       string           `json:"title"`
	Message     string           `json:"message,omitempty"`
	Type        AnnouncementType `json:"type"`
	Dismissible bool             `json:"dismissible"`
	StartsAt    *time.Time       `json:"starts_at,omitempty"`
	EndsAt      *time.Time       `json:"ends_at,omitempty"`
	Active      bool             `json:"active"`
	CreatedBy   *uuid.UUID       `json:"created_by,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// NewAnnouncement creates a new Announcement with the given details.
func NewAnnouncement(orgID uuid.UUID, title string, announcementType AnnouncementType) *Announcement {
	now := time.Now()
	return &Announcement{
		ID:          uuid.New(),
		OrgID:       orgID,
		Title:       title,
		Type:        announcementType,
		Dismissible: true,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsScheduled returns true if the announcement has scheduling constraints.
func (a *Announcement) IsScheduled() bool {
	return a.StartsAt != nil || a.EndsAt != nil
}

// IsVisible returns true if the announcement should be shown at the given time.
func (a *Announcement) IsVisible(now time.Time) bool {
	if !a.Active {
		return false
	}

	// If no schedule, always visible when active
	if a.StartsAt == nil && a.EndsAt == nil {
		return true
	}

	// Check start time
	if a.StartsAt != nil && now.Before(*a.StartsAt) {
		return false
	}

	// Check end time
	if a.EndsAt != nil && now.After(*a.EndsAt) {
		return false
	}

	return true
}

// AnnouncementDismissal tracks when a user dismisses an announcement.
type AnnouncementDismissal struct {
	ID             uuid.UUID `json:"id"`
	OrgID          uuid.UUID `json:"org_id"`
	AnnouncementID uuid.UUID `json:"announcement_id"`
	UserID         uuid.UUID `json:"user_id"`
	DismissedAt    time.Time `json:"dismissed_at"`
}

// NewAnnouncementDismissal creates a new dismissal record.
func NewAnnouncementDismissal(orgID, announcementID, userID uuid.UUID) *AnnouncementDismissal {
	return &AnnouncementDismissal{
		ID:             uuid.New(),
		OrgID:          orgID,
		AnnouncementID: announcementID,
		UserID:         userID,
		DismissedAt:    time.Now(),
	}
}

// CreateAnnouncementRequest is the request body for creating an announcement.
type CreateAnnouncementRequest struct {
	Title       string           `json:"title" binding:"required,min=1,max=255"`
	Message     string           `json:"message,omitempty"`
	Type        AnnouncementType `json:"type" binding:"required,oneof=info warning critical"`
	Dismissible *bool            `json:"dismissible,omitempty"`
	StartsAt    *time.Time       `json:"starts_at,omitempty"`
	EndsAt      *time.Time       `json:"ends_at,omitempty"`
	Active      *bool            `json:"active,omitempty"`
}

// UpdateAnnouncementRequest is the request body for updating an announcement.
type UpdateAnnouncementRequest struct {
	Title       *string          `json:"title,omitempty"`
	Message     *string          `json:"message,omitempty"`
	Type        *AnnouncementType `json:"type,omitempty"`
	Dismissible *bool            `json:"dismissible,omitempty"`
	StartsAt    *time.Time       `json:"starts_at,omitempty"`
	EndsAt      *time.Time       `json:"ends_at,omitempty"`
	Active      *bool            `json:"active,omitempty"`
}

// AnnouncementsResponse is the response for listing announcements.
type AnnouncementsResponse struct {
	Announcements []Announcement `json:"announcements"`
}
