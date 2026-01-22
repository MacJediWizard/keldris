package auth

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"golang.org/x/oauth2"
)

// GroupSyncStore defines the interface for group sync persistence operations.
type GroupSyncStore interface {
	// Group mapping operations
	GetSSOGroupMappingsByGroupNames(ctx context.Context, groupNames []string) ([]*models.SSOGroupMapping, error)
	GetSSOGroupMappingsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SSOGroupMapping, error)

	// User SSO groups operations
	GetUserSSOGroups(ctx context.Context, userID uuid.UUID) (*models.UserSSOGroups, error)
	UpsertUserSSOGroups(ctx context.Context, userID uuid.UUID, groups []string) error

	// Membership operations
	GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OrgMembership, error)
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)
	CreateMembership(ctx context.Context, m *models.OrgMembership) error
	UpdateMembershipRole(ctx context.Context, membershipID uuid.UUID, role models.OrgRole) error

	// Organization operations
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID) (defaultRole *string, autoCreateOrgs bool, err error)
}

// GroupSync handles OIDC group synchronization to Keldris roles.
type GroupSync struct {
	store  GroupSyncStore
	logger zerolog.Logger
}

// NewGroupSync creates a new GroupSync instance.
func NewGroupSync(store GroupSyncStore, logger zerolog.Logger) *GroupSync {
	return &GroupSync{
		store:  store,
		logger: logger.With().Str("component", "group_sync").Logger(),
	}
}

// ExtractGroupsFromToken extracts groups from an OIDC token.
// Different OIDC providers use different claim names for groups.
func (gs *GroupSync) ExtractGroupsFromToken(ctx context.Context, oidc *OIDC, token *oauth2.Token) ([]string, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, fmt.Errorf("no id_token in token response")
	}

	idToken, err := oidc.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("verify ID token: %w", err)
	}

	// Try to extract groups from various possible claim names
	var claims map[string]interface{}
	if err := idToken.Claims(&claims); err != nil {
		return nil, fmt.Errorf("extract claims: %w", err)
	}

	// Common group claim names used by different OIDC providers
	groupClaimNames := []string{
		"groups",           // Common (Keycloak, Azure AD, Okta)
		"group",            // Alternative
		"cognito:groups",   // AWS Cognito
		"roles",            // Some providers use roles
		"memberOf",         // LDAP-style
	}

	for _, claimName := range groupClaimNames {
		if groupsClaim, ok := claims[claimName]; ok {
			groups := extractStringSlice(groupsClaim)
			if len(groups) > 0 {
				gs.logger.Debug().
					Str("claim_name", claimName).
					Strs("groups", groups).
					Msg("extracted groups from token")
				return groups, nil
			}
		}
	}

	gs.logger.Debug().Msg("no groups found in token claims")
	return []string{}, nil
}

// extractStringSlice converts various possible claim formats to a string slice.
func extractStringSlice(claim interface{}) []string {
	switch v := claim.(type) {
	case []string:
		return v
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case string:
		// Single group as a string
		return []string{v}
	default:
		return nil
	}
}

// SyncUserGroups syncs a user's OIDC groups to Keldris memberships.
// This should be called during login after extracting groups from the token.
func (gs *GroupSync) SyncUserGroups(ctx context.Context, userID uuid.UUID, groups []string) (*models.GroupSyncResult, error) {
	result := &models.GroupSyncResult{
		UserID:         userID,
		GroupsReceived: groups,
	}

	// Store the user's current SSO groups
	if err := gs.store.UpsertUserSSOGroups(ctx, userID, groups); err != nil {
		gs.logger.Error().Err(err).Str("user_id", userID.String()).Msg("failed to store user SSO groups")
		// Don't fail the entire sync, just log the error
	}

	if len(groups) == 0 {
		gs.logger.Debug().Str("user_id", userID.String()).Msg("no groups to sync")
		return result, nil
	}

	// Get mappings for the user's groups
	mappings, err := gs.store.GetSSOGroupMappingsByGroupNames(ctx, groups)
	if err != nil {
		return nil, fmt.Errorf("get group mappings: %w", err)
	}

	// Track which groups have mappings
	mappedGroups := make(map[string]bool)
	for _, m := range mappings {
		mappedGroups[m.OIDCGroupName] = true
	}

	// Find unmapped groups
	for _, g := range groups {
		if !mappedGroups[g] {
			result.UnmappedGroups = append(result.UnmappedGroups, g)
		}
	}

	// Get user's current memberships
	existingMemberships, err := gs.store.GetMembershipsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user memberships: %w", err)
	}

	// Create a map of existing memberships by org ID
	existingByOrg := make(map[uuid.UUID]*models.OrgMembership)
	for _, m := range existingMemberships {
		existingByOrg[m.OrgID] = m
	}

	// Process each mapping
	for _, mapping := range mappings {
		if existing, ok := existingByOrg[mapping.OrgID]; ok {
			// User already has membership in this org
			// Update role if different (SSO takes precedence)
			if existing.Role != mapping.Role {
				if err := gs.store.UpdateMembershipRole(ctx, existing.ID, mapping.Role); err != nil {
					gs.logger.Error().
						Err(err).
						Str("user_id", userID.String()).
						Str("org_id", mapping.OrgID.String()).
						Str("new_role", string(mapping.Role)).
						Msg("failed to update membership role")
					continue
				}
				gs.logger.Info().
					Str("user_id", userID.String()).
					Str("org_id", mapping.OrgID.String()).
					Str("old_role", string(existing.Role)).
					Str("new_role", string(mapping.Role)).
					Msg("updated membership role from SSO group")
			}
			result.MembershipsKept = append(result.MembershipsKept, *mapping)
		} else {
			// Create new membership
			newMembership := models.NewOrgMembership(userID, mapping.OrgID, mapping.Role)
			if err := gs.store.CreateMembership(ctx, newMembership); err != nil {
				gs.logger.Error().
					Err(err).
					Str("user_id", userID.String()).
					Str("org_id", mapping.OrgID.String()).
					Str("role", string(mapping.Role)).
					Msg("failed to create membership from SSO group")
				continue
			}
			result.MembershipsAdded = append(result.MembershipsAdded, *mapping)
			gs.logger.Info().
				Str("user_id", userID.String()).
				Str("org_id", mapping.OrgID.String()).
				Str("role", string(mapping.Role)).
				Str("oidc_group", mapping.OIDCGroupName).
				Msg("created membership from SSO group")
		}
	}

	gs.logger.Info().
		Str("user_id", userID.String()).
		Int("groups_received", len(groups)).
		Int("memberships_added", len(result.MembershipsAdded)).
		Int("memberships_kept", len(result.MembershipsKept)).
		Int("unmapped_groups", len(result.UnmappedGroups)).
		Msg("completed group sync")

	return result, nil
}
