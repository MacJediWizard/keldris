package models

import (
	"time"

	"github.com/google/uuid"
)

// SSOGroupMapping represents a mapping from an OIDC group to a Keldris org/role.
type SSOGroupMapping struct {
	ID            uuid.UUID `json:"id"`
	OrgID         uuid.UUID `json:"org_id"`
	OIDCGroupName string    `json:"oidc_group_name"`
	Role          OrgRole   `json:"role"`
	AutoCreateOrg bool      `json:"auto_create_org"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// NewSSOGroupMapping creates a new SSOGroupMapping.
func NewSSOGroupMapping(orgID uuid.UUID, oidcGroupName string, role OrgRole) *SSOGroupMapping {
	now := time.Now()
	return &SSOGroupMapping{
		ID:            uuid.New(),
		OrgID:         orgID,
		OIDCGroupName: oidcGroupName,
		Role:          role,
		AutoCreateOrg: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// UserSSOGroups represents a user's OIDC groups from their last login.
type UserSSOGroups struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	OIDCGroups []string  `json:"oidc_groups"`
	SyncedAt   time.Time `json:"synced_at"`
}

// NewUserSSOGroups creates a new UserSSOGroups record.
func NewUserSSOGroups(userID uuid.UUID, groups []string) *UserSSOGroups {
	return &UserSSOGroups{
		ID:         uuid.New(),
		UserID:     userID,
		OIDCGroups: groups,
		SyncedAt:   time.Now(),
	}
}

// GroupSyncResult represents the result of syncing a user's groups.
type GroupSyncResult struct {
	UserID            uuid.UUID         `json:"user_id"`
	GroupsReceived    []string          `json:"groups_received"`
	MembershipsAdded  []SSOGroupMapping `json:"memberships_added"`
	MembershipsKept   []SSOGroupMapping `json:"memberships_kept"`
	UnmappedGroups    []string          `json:"unmapped_groups"`
	OrgsAutoCreated   []uuid.UUID       `json:"orgs_auto_created,omitempty"`
}

// CreateSSOGroupMappingRequest is the request to create a new group mapping.
type CreateSSOGroupMappingRequest struct {
	OIDCGroupName string  `json:"oidc_group_name" binding:"required"`
	Role          string  `json:"role" binding:"required"`
	AutoCreateOrg bool    `json:"auto_create_org"`
}

// UpdateSSOGroupMappingRequest is the request to update a group mapping.
type UpdateSSOGroupMappingRequest struct {
	Role          *string `json:"role,omitempty"`
	AutoCreateOrg *bool   `json:"auto_create_org,omitempty"`
}
