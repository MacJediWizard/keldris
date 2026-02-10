package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// mockMembershipStore implements MembershipStore for testing.
type mockMembershipStore struct {
	memberships map[string]*models.OrgMembership // key: "userID:orgID"
	err         error
}

func newMockMembershipStore() *mockMembershipStore {
	return &mockMembershipStore{
		memberships: make(map[string]*models.OrgMembership),
	}
}

func (m *mockMembershipStore) addMembership(userID, orgID uuid.UUID, role models.OrgRole) {
	key := userID.String() + ":" + orgID.String()
	m.memberships[key] = &models.OrgMembership{
		ID:     uuid.New(),
		UserID: userID,
		OrgID:  orgID,
		Role:   role,
	}
}

func (m *mockMembershipStore) GetMembershipByUserAndOrg(_ context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	if m.err != nil {
		return nil, m.err
	}
	key := userID.String() + ":" + orgID.String()
	membership, ok := m.memberships[key]
	if !ok {
		return nil, nil
	}
	return membership, nil
}

func (m *mockMembershipStore) GetMembershipsByUserID(_ context.Context, userID uuid.UUID) ([]*models.OrgMembership, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []*models.OrgMembership
	for _, membership := range m.memberships {
		if membership.UserID == userID {
			result = append(result, membership)
		}
	}
	return result, nil
}

func TestHasRolePermission_Owner(t *testing.T) {
	// Owner should have all permissions
	allPerms := []Permission{
		PermOrgRead, PermOrgUpdate, PermOrgDelete,
		PermMemberRead, PermMemberInvite, PermMemberUpdate, PermMemberRemove,
		PermAgentRead, PermAgentCreate, PermAgentUpdate, PermAgentDelete,
		PermRepoRead, PermRepoCreate, PermRepoUpdate, PermRepoDelete,
		PermScheduleRead, PermScheduleCreate, PermScheduleUpdate, PermScheduleDelete, PermScheduleRun,
		PermBackupRead, PermBackupCreate,
	}

	for _, perm := range allPerms {
		if !HasRolePermission(models.OrgRoleOwner, perm) {
			t.Errorf("owner should have permission %s", perm)
		}
	}
}

func TestHasRolePermission_Admin(t *testing.T) {
	// Admin should have most permissions but NOT org:delete
	allowed := []Permission{
		PermOrgRead, PermOrgUpdate,
		PermMemberRead, PermMemberInvite, PermMemberUpdate, PermMemberRemove,
		PermAgentRead, PermAgentCreate, PermAgentUpdate, PermAgentDelete,
		PermRepoRead, PermRepoCreate, PermRepoUpdate, PermRepoDelete,
		PermScheduleRead, PermScheduleCreate, PermScheduleUpdate, PermScheduleDelete, PermScheduleRun,
		PermBackupRead, PermBackupCreate,
	}
	denied := []Permission{PermOrgDelete}

	for _, perm := range allowed {
		if !HasRolePermission(models.OrgRoleAdmin, perm) {
			t.Errorf("admin should have permission %s", perm)
		}
	}
	for _, perm := range denied {
		if HasRolePermission(models.OrgRoleAdmin, perm) {
			t.Errorf("admin should NOT have permission %s", perm)
		}
	}
}

func TestHasRolePermission_Member(t *testing.T) {
	// Member can read org, read members, and manage agents/repos/schedules/backups
	allowed := []Permission{
		PermOrgRead,
		PermMemberRead,
		PermAgentRead, PermAgentCreate, PermAgentUpdate, PermAgentDelete,
		PermRepoRead, PermRepoCreate, PermRepoUpdate, PermRepoDelete,
		PermScheduleRead, PermScheduleCreate, PermScheduleUpdate, PermScheduleDelete, PermScheduleRun,
		PermBackupRead, PermBackupCreate,
	}
	denied := []Permission{
		PermOrgUpdate, PermOrgDelete,
		PermMemberInvite, PermMemberUpdate, PermMemberRemove,
	}

	for _, perm := range allowed {
		if !HasRolePermission(models.OrgRoleMember, perm) {
			t.Errorf("member should have permission %s", perm)
		}
	}
	for _, perm := range denied {
		if HasRolePermission(models.OrgRoleMember, perm) {
			t.Errorf("member should NOT have permission %s", perm)
		}
	}
}

func TestHasRolePermission_Readonly(t *testing.T) {
	// Readonly should only have read permissions
	allowed := []Permission{
		PermOrgRead,
		PermMemberRead,
		PermAgentRead,
		PermRepoRead,
		PermScheduleRead,
		PermBackupRead,
	}
	denied := []Permission{
		PermOrgUpdate, PermOrgDelete,
		PermMemberInvite, PermMemberUpdate, PermMemberRemove,
		PermAgentCreate, PermAgentUpdate, PermAgentDelete,
		PermRepoCreate, PermRepoUpdate, PermRepoDelete,
		PermScheduleCreate, PermScheduleUpdate, PermScheduleDelete, PermScheduleRun,
		PermBackupCreate,
	}

	for _, perm := range allowed {
		if !HasRolePermission(models.OrgRoleReadonly, perm) {
			t.Errorf("readonly should have permission %s", perm)
		}
	}
	for _, perm := range denied {
		if HasRolePermission(models.OrgRoleReadonly, perm) {
			t.Errorf("readonly should NOT have permission %s", perm)
		}
	}
}

func TestHasRolePermission_InvalidRole(t *testing.T) {
	if HasRolePermission(models.OrgRole("nonexistent"), PermOrgRead) {
		t.Error("invalid role should not have any permissions")
	}
}

func TestRBAC_HasPermission(t *testing.T) {
	store := newMockMembershipStore()
	rbac := NewRBAC(store)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	store.addMembership(userID, orgID, models.OrgRoleAdmin)

	t.Run("user has permission", func(t *testing.T) {
		has, err := rbac.HasPermission(ctx, userID, orgID, PermAgentCreate)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !has {
			t.Error("expected admin to have agent:create permission")
		}
	})

	t.Run("user lacks permission", func(t *testing.T) {
		has, err := rbac.HasPermission(ctx, userID, orgID, PermOrgDelete)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected admin to NOT have org:delete permission")
		}
	})

	t.Run("not a member", func(t *testing.T) {
		otherOrgID := uuid.New()
		has, err := rbac.HasPermission(ctx, userID, otherOrgID, PermOrgRead)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if has {
			t.Error("expected non-member to have no permissions")
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := newMockMembershipStore()
		errStore.err = fmt.Errorf("database connection failed")
		errRBAC := NewRBAC(errStore)

		_, err := errRBAC.HasPermission(ctx, userID, orgID, PermOrgRead)
		if err == nil {
			t.Error("expected error from store failure")
		}
	})
}

func TestRBAC_RequirePermission(t *testing.T) {
	store := newMockMembershipStore()
	rbac := NewRBAC(store)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	store.addMembership(userID, orgID, models.OrgRoleMember)

	t.Run("permission granted", func(t *testing.T) {
		err := rbac.RequirePermission(ctx, userID, orgID, PermAgentCreate)
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("permission denied", func(t *testing.T) {
		err := rbac.RequirePermission(ctx, userID, orgID, PermMemberInvite)
		if err != ErrPermissionDenied {
			t.Errorf("expected ErrPermissionDenied, got: %v", err)
		}
	})

	t.Run("not a member", func(t *testing.T) {
		otherOrgID := uuid.New()
		err := rbac.RequirePermission(ctx, userID, otherOrgID, PermOrgRead)
		if err != ErrPermissionDenied {
			t.Errorf("expected ErrPermissionDenied, got: %v", err)
		}
	})
}

func TestRBAC_GetUserRole(t *testing.T) {
	store := newMockMembershipStore()
	rbac := NewRBAC(store)
	ctx := context.Background()

	userID := uuid.New()
	orgID := uuid.New()
	store.addMembership(userID, orgID, models.OrgRoleOwner)

	t.Run("member found", func(t *testing.T) {
		role, err := rbac.GetUserRole(ctx, userID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if role != models.OrgRoleOwner {
			t.Errorf("expected role owner, got %s", role)
		}
	})

	t.Run("not a member", func(t *testing.T) {
		otherOrgID := uuid.New()
		_, err := rbac.GetUserRole(ctx, userID, otherOrgID)
		if err != ErrNotMember {
			t.Errorf("expected ErrNotMember, got: %v", err)
		}
	})

	t.Run("store error", func(t *testing.T) {
		errStore := newMockMembershipStore()
		errStore.err = fmt.Errorf("database error")
		errRBAC := NewRBAC(errStore)

		_, err := errRBAC.GetUserRole(ctx, userID, orgID)
		if err == nil {
			t.Error("expected error from store failure")
		}
	})
}

func TestRBAC_CanManageMember(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("owner can manage admin", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		ownerID := uuid.New()
		adminID := uuid.New()
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)

		can, err := rbac.CanManageMember(ctx, ownerID, adminID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("owner should be able to manage admin")
		}
	})

	t.Run("owner can manage member", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		ownerID := uuid.New()
		memberID := uuid.New()
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)
		store.addMembership(memberID, orgID, models.OrgRoleMember)

		can, err := rbac.CanManageMember(ctx, ownerID, memberID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("owner should be able to manage member")
		}
	})

	t.Run("admin can manage member", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		adminID := uuid.New()
		memberID := uuid.New()
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)
		store.addMembership(memberID, orgID, models.OrgRoleMember)

		can, err := rbac.CanManageMember(ctx, adminID, memberID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("admin should be able to manage member")
		}
	})

	t.Run("admin can manage readonly", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		adminID := uuid.New()
		readonlyID := uuid.New()
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)
		store.addMembership(readonlyID, orgID, models.OrgRoleReadonly)

		can, err := rbac.CanManageMember(ctx, adminID, readonlyID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("admin should be able to manage readonly")
		}
	})

	t.Run("admin cannot manage owner", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		adminID := uuid.New()
		ownerID := uuid.New()
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)

		can, err := rbac.CanManageMember(ctx, adminID, ownerID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("admin should NOT be able to manage owner")
		}
	})

	t.Run("admin cannot manage another admin", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		admin1 := uuid.New()
		admin2 := uuid.New()
		store.addMembership(admin1, orgID, models.OrgRoleAdmin)
		store.addMembership(admin2, orgID, models.OrgRoleAdmin)

		can, err := rbac.CanManageMember(ctx, admin1, admin2, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("admin should NOT be able to manage another admin")
		}
	})

	t.Run("member cannot manage anyone", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		memberID := uuid.New()
		readonlyID := uuid.New()
		store.addMembership(memberID, orgID, models.OrgRoleMember)
		store.addMembership(readonlyID, orgID, models.OrgRoleReadonly)

		can, err := rbac.CanManageMember(ctx, memberID, readonlyID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("member should NOT be able to manage anyone")
		}
	})

	t.Run("cannot manage self", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		ownerID := uuid.New()
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)

		can, err := rbac.CanManageMember(ctx, ownerID, ownerID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("user should NOT be able to manage self")
		}
	})

	t.Run("actor not a member", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		nonMemberID := uuid.New()
		memberID := uuid.New()
		store.addMembership(memberID, orgID, models.OrgRoleMember)

		can, err := rbac.CanManageMember(ctx, nonMemberID, memberID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("non-member should NOT be able to manage anyone")
		}
	})

	t.Run("target not a member", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		ownerID := uuid.New()
		nonMemberID := uuid.New()
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)

		can, err := rbac.CanManageMember(ctx, ownerID, nonMemberID, orgID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("should not be able to manage non-member target")
		}
	})
}

func TestRBAC_CanAssignRole(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()

	t.Run("owner can assign owner role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		ownerID := uuid.New()
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)

		can, err := rbac.CanAssignRole(ctx, ownerID, orgID, models.OrgRoleOwner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("owner should be able to assign owner role")
		}
	})

	t.Run("owner can assign admin role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		ownerID := uuid.New()
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)

		can, err := rbac.CanAssignRole(ctx, ownerID, orgID, models.OrgRoleAdmin)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("owner should be able to assign admin role")
		}
	})

	t.Run("owner can assign member role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		ownerID := uuid.New()
		store.addMembership(ownerID, orgID, models.OrgRoleOwner)

		can, err := rbac.CanAssignRole(ctx, ownerID, orgID, models.OrgRoleMember)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("owner should be able to assign member role")
		}
	})

	t.Run("admin cannot assign owner role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		adminID := uuid.New()
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)

		can, err := rbac.CanAssignRole(ctx, adminID, orgID, models.OrgRoleOwner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("admin should NOT be able to assign owner role")
		}
	})

	t.Run("admin cannot assign admin role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		adminID := uuid.New()
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)

		can, err := rbac.CanAssignRole(ctx, adminID, orgID, models.OrgRoleAdmin)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("admin should NOT be able to assign admin role")
		}
	})

	t.Run("admin can assign member role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		adminID := uuid.New()
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)

		can, err := rbac.CanAssignRole(ctx, adminID, orgID, models.OrgRoleMember)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("admin should be able to assign member role")
		}
	})

	t.Run("admin can assign readonly role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		adminID := uuid.New()
		store.addMembership(adminID, orgID, models.OrgRoleAdmin)

		can, err := rbac.CanAssignRole(ctx, adminID, orgID, models.OrgRoleReadonly)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !can {
			t.Error("admin should be able to assign readonly role")
		}
	})

	t.Run("member cannot assign any role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		memberID := uuid.New()
		store.addMembership(memberID, orgID, models.OrgRoleMember)

		for _, role := range models.ValidOrgRoles() {
			can, err := rbac.CanAssignRole(ctx, memberID, orgID, role)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if can {
				t.Errorf("member should NOT be able to assign %s role", role)
			}
		}
	})

	t.Run("non-member cannot assign any role", func(t *testing.T) {
		store := newMockMembershipStore()
		rbac := NewRBAC(store)
		nonMemberID := uuid.New()

		can, err := rbac.CanAssignRole(ctx, nonMemberID, orgID, models.OrgRoleMember)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if can {
			t.Error("non-member should NOT be able to assign any role")
		}
	})
}
