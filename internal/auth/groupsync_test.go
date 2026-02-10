package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockGroupSyncStore implements GroupSyncStore for testing.
type mockGroupSyncStore struct {
	groupMappings map[string]*models.SSOGroupMapping // key: oidc_group_name
	memberships   map[string]*models.OrgMembership   // key: "userID:orgID"
	userGroups    map[uuid.UUID][]string
	organizations map[uuid.UUID]*models.Organization

	upsertGroupsErr    error
	getMappingsErr     error
	getMembershipsErr  error
	createMembershipErr error
	updateRoleErr      error

	createdMemberships []*models.OrgMembership
	updatedRoles       []roleUpdate
}

type roleUpdate struct {
	membershipID uuid.UUID
	role         models.OrgRole
}

func newMockGroupSyncStore() *mockGroupSyncStore {
	return &mockGroupSyncStore{
		groupMappings: make(map[string]*models.SSOGroupMapping),
		memberships:   make(map[string]*models.OrgMembership),
		userGroups:    make(map[uuid.UUID][]string),
		organizations: make(map[uuid.UUID]*models.Organization),
	}
}

func (m *mockGroupSyncStore) addGroupMapping(groupName string, orgID uuid.UUID, role models.OrgRole) {
	m.groupMappings[groupName] = &models.SSOGroupMapping{
		ID:            uuid.New(),
		OrgID:         orgID,
		OIDCGroupName: groupName,
		Role:          role,
	}
}

func (m *mockGroupSyncStore) addMembership(userID, orgID uuid.UUID, role models.OrgRole) *models.OrgMembership {
	key := userID.String() + ":" + orgID.String()
	membership := &models.OrgMembership{
		ID:     uuid.New(),
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
	}
	m.memberships[key] = membership
	return membership
}

func (m *mockGroupSyncStore) GetSSOGroupMappingsByGroupNames(_ context.Context, groupNames []string) ([]*models.SSOGroupMapping, error) {
	if m.getMappingsErr != nil {
		return nil, m.getMappingsErr
	}
	var result []*models.SSOGroupMapping
	for _, name := range groupNames {
		if mapping, ok := m.groupMappings[name]; ok {
			result = append(result, mapping)
		}
	}
	return result, nil
}

func (m *mockGroupSyncStore) GetSSOGroupMappingsByOrgID(_ context.Context, orgID uuid.UUID) ([]*models.SSOGroupMapping, error) {
	var result []*models.SSOGroupMapping
	for _, mapping := range m.groupMappings {
		if mapping.OrgID == orgID {
			result = append(result, mapping)
		}
	}
	return result, nil
}

func (m *mockGroupSyncStore) GetUserSSOGroups(_ context.Context, userID uuid.UUID) (*models.UserSSOGroups, error) {
	groups, ok := m.userGroups[userID]
	if !ok {
		return nil, nil
	}
	return &models.UserSSOGroups{
		ID:         uuid.New(),
		UserID:     userID,
		OIDCGroups: groups,
	}, nil
}

func (m *mockGroupSyncStore) UpsertUserSSOGroups(_ context.Context, userID uuid.UUID, groups []string) error {
	if m.upsertGroupsErr != nil {
		return m.upsertGroupsErr
	}
	m.userGroups[userID] = groups
	return nil
}

func (m *mockGroupSyncStore) GetMembershipsByUserID(_ context.Context, userID uuid.UUID) ([]*models.OrgMembership, error) {
	if m.getMembershipsErr != nil {
		return nil, m.getMembershipsErr
	}
	var result []*models.OrgMembership
	for _, membership := range m.memberships {
		if membership.UserID == userID {
			result = append(result, membership)
		}
	}
	return result, nil
}

func (m *mockGroupSyncStore) GetMembershipByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	key := userID.String() + ":" + orgID.String()
	membership, ok := m.memberships[key]
	if !ok {
		return nil, nil
	}
	return membership, nil
}

func (m *mockGroupSyncStore) CreateMembership(_ context.Context, membership *models.OrgMembership) error {
	if m.createMembershipErr != nil {
		return m.createMembershipErr
	}
	m.createdMemberships = append(m.createdMemberships, membership)
	key := membership.UserID.String() + ":" + membership.OrgID.String()
	m.memberships[key] = membership
	return nil
}

func (m *mockGroupSyncStore) UpdateMembershipRole(_ context.Context, membershipID uuid.UUID, role models.OrgRole) error {
	if m.updateRoleErr != nil {
		return m.updateRoleErr
	}
	m.updatedRoles = append(m.updatedRoles, roleUpdate{membershipID: membershipID, role: role})
	return nil
}

func (m *mockGroupSyncStore) GetOrganizationByID(_ context.Context, id uuid.UUID) (*models.Organization, error) {
	org, ok := m.organizations[id]
	if !ok {
		return nil, nil
	}
	return org, nil
}

func (m *mockGroupSyncStore) GetOrganizationSSOSettings(_ context.Context, _ uuid.UUID) (defaultRole *string, autoCreateOrgs bool, err error) {
	return nil, false, nil
}

func TestNewGroupSync(t *testing.T) {
	store := newMockGroupSyncStore()
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	if gs == nil {
		t.Fatal("expected non-nil GroupSync")
	}
}

func TestExtractStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []string
	}{
		{
			name:     "string slice",
			input:    []string{"group1", "group2"},
			expected: []string{"group1", "group2"},
		},
		{
			name:     "interface slice",
			input:    []interface{}{"group1", "group2", "group3"},
			expected: []string{"group1", "group2", "group3"},
		},
		{
			name:     "single string",
			input:    "single-group",
			expected: []string{"single-group"},
		},
		{
			name:     "interface slice with non-strings",
			input:    []interface{}{"group1", 42, "group2"},
			expected: []string{"group1", "group2"},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "integer input",
			input:    42,
			expected: nil,
		},
		{
			name:     "empty interface slice",
			input:    []interface{}{},
			expected: nil,
		},
		{
			name:     "empty string slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStringSlice(tt.input)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d items, got %d", len(tt.expected), len(result))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("item %d: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestGroupSync_SyncUserGroups_NewMembership(t *testing.T) {
	store := newMockGroupSyncStore()
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	store.addGroupMapping("engineering", orgID, models.OrgRoleMember)

	result, err := gs.SyncUserGroups(ctx, userID, []string{"engineering"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.MembershipsAdded) != 1 {
		t.Fatalf("expected 1 membership added, got %d", len(result.MembershipsAdded))
	}
	if result.MembershipsAdded[0].OIDCGroupName != "engineering" {
		t.Errorf("expected group name 'engineering', got %q", result.MembershipsAdded[0].OIDCGroupName)
	}
	if result.MembershipsAdded[0].Role != models.OrgRoleMember {
		t.Errorf("expected role member, got %s", result.MembershipsAdded[0].Role)
	}
	if len(store.createdMemberships) != 1 {
		t.Errorf("expected 1 membership created in store, got %d", len(store.createdMemberships))
	}
}

func TestGroupSync_SyncUserGroups_ExistingMembership_RoleUpdate(t *testing.T) {
	store := newMockGroupSyncStore()
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// User has member role, but SSO mapping says admin
	store.addMembership(userID, orgID, models.OrgRoleMember)
	store.addGroupMapping("admins", orgID, models.OrgRoleAdmin)

	result, err := gs.SyncUserGroups(ctx, userID, []string{"admins"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.MembershipsKept) != 1 {
		t.Fatalf("expected 1 membership kept, got %d", len(result.MembershipsKept))
	}
	if len(store.updatedRoles) != 1 {
		t.Fatalf("expected 1 role update, got %d", len(store.updatedRoles))
	}
	if store.updatedRoles[0].role != models.OrgRoleAdmin {
		t.Errorf("expected role updated to admin, got %s", store.updatedRoles[0].role)
	}
}

func TestGroupSync_SyncUserGroups_ExistingMembership_SameRole(t *testing.T) {
	store := newMockGroupSyncStore()
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	// User already has the role that the mapping specifies
	store.addMembership(userID, orgID, models.OrgRoleMember)
	store.addGroupMapping("developers", orgID, models.OrgRoleMember)

	result, err := gs.SyncUserGroups(ctx, userID, []string{"developers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.MembershipsKept) != 1 {
		t.Fatalf("expected 1 membership kept, got %d", len(result.MembershipsKept))
	}
	// No role update should have happened
	if len(store.updatedRoles) != 0 {
		t.Errorf("expected 0 role updates, got %d", len(store.updatedRoles))
	}
}

func TestGroupSync_SyncUserGroups_NoGroups(t *testing.T) {
	store := newMockGroupSyncStore()
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()

	result, err := gs.SyncUserGroups(ctx, userID, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.GroupsReceived) != 0 {
		t.Errorf("expected 0 groups received, got %d", len(result.GroupsReceived))
	}
	if len(result.MembershipsAdded) != 0 {
		t.Errorf("expected 0 memberships added, got %d", len(result.MembershipsAdded))
	}
}

func TestGroupSync_SyncUserGroups_UnmappedGroups(t *testing.T) {
	store := newMockGroupSyncStore()
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	store.addGroupMapping("engineering", orgID, models.OrgRoleMember)

	result, err := gs.SyncUserGroups(ctx, userID, []string{"engineering", "marketing", "sales"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.UnmappedGroups) != 2 {
		t.Fatalf("expected 2 unmapped groups, got %d", len(result.UnmappedGroups))
	}

	unmapped := make(map[string]bool)
	for _, g := range result.UnmappedGroups {
		unmapped[g] = true
	}
	if !unmapped["marketing"] {
		t.Error("expected 'marketing' in unmapped groups")
	}
	if !unmapped["sales"] {
		t.Error("expected 'sales' in unmapped groups")
	}
}

func TestGroupSync_SyncUserGroups_MixedScenario(t *testing.T) {
	store := newMockGroupSyncStore()
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	org1ID := uuid.New()
	org2ID := uuid.New()

	// User already has membership in org1 as member
	store.addMembership(userID, org1ID, models.OrgRoleMember)

	// Mappings: org1 -> admin (role upgrade), org2 -> member (new membership)
	store.addGroupMapping("team-leads", org1ID, models.OrgRoleAdmin)
	store.addGroupMapping("org2-dev", org2ID, models.OrgRoleMember)

	result, err := gs.SyncUserGroups(ctx, userID, []string{"team-leads", "org2-dev", "unknown-group"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.GroupsReceived) != 3 {
		t.Errorf("expected 3 groups received, got %d", len(result.GroupsReceived))
	}
	if len(result.MembershipsAdded) != 1 {
		t.Errorf("expected 1 membership added, got %d", len(result.MembershipsAdded))
	}
	if len(result.MembershipsKept) != 1 {
		t.Errorf("expected 1 membership kept, got %d", len(result.MembershipsKept))
	}
	if len(result.UnmappedGroups) != 1 {
		t.Errorf("expected 1 unmapped group, got %d", len(result.UnmappedGroups))
	}
	if len(store.updatedRoles) != 1 {
		t.Errorf("expected 1 role update, got %d", len(store.updatedRoles))
	}
}

func TestGroupSync_SyncUserGroups_UpsertGroupsError(t *testing.T) {
	store := newMockGroupSyncStore()
	store.upsertGroupsErr = fmt.Errorf("database error")
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()

	// Should not fail entirely - just logs the error
	result, err := gs.SyncUserGroups(ctx, userID, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestGroupSync_SyncUserGroups_GetMappingsError(t *testing.T) {
	store := newMockGroupSyncStore()
	store.getMappingsErr = fmt.Errorf("database error")
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()

	_, err := gs.SyncUserGroups(ctx, userID, []string{"some-group"})
	if err == nil {
		t.Error("expected error from get mappings failure")
	}
}

func TestGroupSync_SyncUserGroups_GetMembershipsError(t *testing.T) {
	store := newMockGroupSyncStore()
	store.getMembershipsErr = fmt.Errorf("database error")
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	store.addGroupMapping("group1", orgID, models.OrgRoleMember)

	_, err := gs.SyncUserGroups(ctx, userID, []string{"group1"})
	if err == nil {
		t.Error("expected error from get memberships failure")
	}
}

func TestGroupSync_SyncUserGroups_CreateMembershipError(t *testing.T) {
	store := newMockGroupSyncStore()
	store.createMembershipErr = fmt.Errorf("database error")
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	store.addGroupMapping("group1", orgID, models.OrgRoleMember)

	// Should not fail entirely - continues processing
	result, err := gs.SyncUserGroups(ctx, userID, []string{"group1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No membership added due to error
	if len(result.MembershipsAdded) != 0 {
		t.Errorf("expected 0 memberships added, got %d", len(result.MembershipsAdded))
	}
}

func TestGroupSync_SyncUserGroups_UpdateRoleError(t *testing.T) {
	store := newMockGroupSyncStore()
	store.updateRoleErr = fmt.Errorf("database error")
	logger := zerolog.Nop()
	gs := NewGroupSync(store, logger)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()

	store.addMembership(userID, orgID, models.OrgRoleMember)
	store.addGroupMapping("admins", orgID, models.OrgRoleAdmin)

	// Should not fail entirely - continues processing
	result, err := gs.SyncUserGroups(ctx, userID, []string{"admins"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Membership still counted as kept even though update failed
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}
