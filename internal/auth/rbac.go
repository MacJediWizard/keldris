// Package auth provides authentication and authorization for Keldris.
package auth

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Permission defines an action that can be performed.
type Permission string

const (
	// Organization permissions
	PermOrgRead   Permission = "org:read"
	PermOrgUpdate Permission = "org:update"
	PermOrgDelete Permission = "org:delete"

	// Member management permissions
	PermMemberRead   Permission = "member:read"
	PermMemberInvite Permission = "member:invite"
	PermMemberUpdate Permission = "member:update"
	PermMemberRemove Permission = "member:remove"

	// Agent permissions
	PermAgentRead   Permission = "agent:read"
	PermAgentCreate Permission = "agent:create"
	PermAgentUpdate Permission = "agent:update"
	PermAgentDelete Permission = "agent:delete"

	// Repository permissions
	PermRepoRead   Permission = "repo:read"
	PermRepoCreate Permission = "repo:create"
	PermRepoUpdate Permission = "repo:update"
	PermRepoDelete Permission = "repo:delete"

	// Schedule permissions
	PermScheduleRead   Permission = "schedule:read"
	PermScheduleCreate Permission = "schedule:create"
	PermScheduleUpdate Permission = "schedule:update"
	PermScheduleDelete Permission = "schedule:delete"
	PermScheduleRun    Permission = "schedule:run"

	// Backup permissions
	PermBackupRead   Permission = "backup:read"
	PermBackupCreate Permission = "backup:create"
)

// rolePermissions maps roles to their allowed permissions.
var rolePermissions = map[models.OrgRole][]Permission{
	models.OrgRoleOwner: {
		// Organization
		PermOrgRead, PermOrgUpdate, PermOrgDelete,
		// Members
		PermMemberRead, PermMemberInvite, PermMemberUpdate, PermMemberRemove,
		// Agents
		PermAgentRead, PermAgentCreate, PermAgentUpdate, PermAgentDelete,
		// Repositories
		PermRepoRead, PermRepoCreate, PermRepoUpdate, PermRepoDelete,
		// Schedules
		PermScheduleRead, PermScheduleCreate, PermScheduleUpdate, PermScheduleDelete, PermScheduleRun,
		// Backups
		PermBackupRead, PermBackupCreate,
	},
	models.OrgRoleAdmin: {
		// Organization
		PermOrgRead, PermOrgUpdate,
		// Members (cannot remove owner)
		PermMemberRead, PermMemberInvite, PermMemberUpdate, PermMemberRemove,
		// Agents
		PermAgentRead, PermAgentCreate, PermAgentUpdate, PermAgentDelete,
		// Repositories
		PermRepoRead, PermRepoCreate, PermRepoUpdate, PermRepoDelete,
		// Schedules
		PermScheduleRead, PermScheduleCreate, PermScheduleUpdate, PermScheduleDelete, PermScheduleRun,
		// Backups
		PermBackupRead, PermBackupCreate,
	},
	models.OrgRoleMember: {
		// Organization
		PermOrgRead,
		// Members
		PermMemberRead,
		// Agents
		PermAgentRead, PermAgentCreate, PermAgentUpdate, PermAgentDelete,
		// Repositories
		PermRepoRead, PermRepoCreate, PermRepoUpdate, PermRepoDelete,
		// Schedules
		PermScheduleRead, PermScheduleCreate, PermScheduleUpdate, PermScheduleDelete, PermScheduleRun,
		// Backups
		PermBackupRead, PermBackupCreate,
	},
	models.OrgRoleReadonly: {
		// Organization
		PermOrgRead,
		// Members
		PermMemberRead,
		// Agents
		PermAgentRead,
		// Repositories
		PermRepoRead,
		// Schedules
		PermScheduleRead,
		// Backups
		PermBackupRead,
	},
}

// MembershipStore defines the interface for fetching membership data.
type MembershipStore interface {
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)
	GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OrgMembership, error)
}

// RBAC provides role-based access control functionality.
type RBAC struct {
	store MembershipStore
}

// NewRBAC creates a new RBAC instance.
func NewRBAC(store MembershipStore) *RBAC {
	return &RBAC{store: store}
}

// HasPermission checks if the user has the given permission in the organization.
func (r *RBAC) HasPermission(ctx context.Context, userID, orgID uuid.UUID, perm Permission) (bool, error) {
	membership, err := r.store.GetMembershipByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return false, fmt.Errorf("get membership: %w", err)
	}
	if membership == nil {
		return false, nil
	}

	return HasRolePermission(membership.Role, perm), nil
}

// HasRolePermission checks if a role has the given permission.
func HasRolePermission(role models.OrgRole, perm Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// RequirePermission checks if the user has permission and returns an error if not.
func (r *RBAC) RequirePermission(ctx context.Context, userID, orgID uuid.UUID, perm Permission) error {
	has, err := r.HasPermission(ctx, userID, orgID, perm)
	if err != nil {
		return err
	}
	if !has {
		return ErrPermissionDenied
	}
	return nil
}

// GetUserRole returns the user's role in the organization.
func (r *RBAC) GetUserRole(ctx context.Context, userID, orgID uuid.UUID) (models.OrgRole, error) {
	membership, err := r.store.GetMembershipByUserAndOrg(ctx, userID, orgID)
	if err != nil {
		return "", fmt.Errorf("get membership: %w", err)
	}
	if membership == nil {
		return "", ErrNotMember
	}
	return membership.Role, nil
}

// CanManageMember checks if a user can manage (update/remove) another member.
// Owners can manage anyone. Admins can manage members and readonly, but not other admins or owners.
func (r *RBAC) CanManageMember(ctx context.Context, actorID, targetID, orgID uuid.UUID) (bool, error) {
	if actorID == targetID {
		// Users can't manage themselves (except leaving the org)
		return false, nil
	}

	actorMembership, err := r.store.GetMembershipByUserAndOrg(ctx, actorID, orgID)
	if err != nil {
		return false, fmt.Errorf("get actor membership: %w", err)
	}
	if actorMembership == nil {
		return false, nil
	}

	targetMembership, err := r.store.GetMembershipByUserAndOrg(ctx, targetID, orgID)
	if err != nil {
		return false, fmt.Errorf("get target membership: %w", err)
	}
	if targetMembership == nil {
		return false, nil
	}

	// Owner can manage anyone
	if actorMembership.Role == models.OrgRoleOwner {
		return true, nil
	}

	// Admin can manage member and readonly, but not owner or other admins
	if actorMembership.Role == models.OrgRoleAdmin {
		if targetMembership.Role == models.OrgRoleMember || targetMembership.Role == models.OrgRoleReadonly {
			return true, nil
		}
		return false, nil
	}

	// Members and readonly cannot manage others
	return false, nil
}

// CanAssignRole checks if a user can assign a specific role to another user.
func (r *RBAC) CanAssignRole(ctx context.Context, actorID, orgID uuid.UUID, targetRole models.OrgRole) (bool, error) {
	actorMembership, err := r.store.GetMembershipByUserAndOrg(ctx, actorID, orgID)
	if err != nil {
		return false, fmt.Errorf("get actor membership: %w", err)
	}
	if actorMembership == nil {
		return false, nil
	}

	// Only owner can assign owner or admin role
	if targetRole == models.OrgRoleOwner || targetRole == models.OrgRoleAdmin {
		return actorMembership.Role == models.OrgRoleOwner, nil
	}

	// Admin can assign member or readonly
	if actorMembership.Role == models.OrgRoleOwner || actorMembership.Role == models.OrgRoleAdmin {
		return true, nil
	}

	return false, nil
}

// ErrPermissionDenied is returned when a user lacks required permissions.
var ErrPermissionDenied = fmt.Errorf("permission denied")

// ErrNotMember is returned when a user is not a member of the organization.
var ErrNotMember = fmt.Errorf("not a member of this organization")
