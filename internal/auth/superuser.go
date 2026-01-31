// Package auth provides authentication and authorization for Keldris.
package auth

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// SuperuserStore defines the interface for superuser-related data operations.
type SuperuserStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GetAllUsers(ctx context.Context) ([]*models.User, error)
	SetUserSuperuser(ctx context.Context, userID uuid.UUID, isSuperuser bool) error
	GetSuperusers(ctx context.Context) ([]*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateSuperuserAuditLog(ctx context.Context, log *models.SuperuserAuditLog) error
	GetSuperuserAuditLogs(ctx context.Context, limit, offset int) ([]*models.SuperuserAuditLogWithUser, int, error)
	GetSystemSetting(ctx context.Context, key string) (*models.SystemSetting, error)
	GetSystemSettings(ctx context.Context) ([]*models.SystemSetting, error)
	UpdateSystemSetting(ctx context.Context, key string, value interface{}, updatedBy uuid.UUID) error
}

// Superuser provides superuser authorization checks.
type Superuser struct {
	store SuperuserStore
}

// NewSuperuser creates a new Superuser instance.
func NewSuperuser(store SuperuserStore) *Superuser {
	return &Superuser{store: store}
}

// IsSuperuser checks if the user with the given ID is a superuser.
func (s *Superuser) IsSuperuser(ctx context.Context, userID uuid.UUID) (bool, error) {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("get user: %w", err)
	}
	return user.IsSuperuser, nil
}

// RequireSuperuser checks if the user is a superuser and returns an error if not.
func (s *Superuser) RequireSuperuser(ctx context.Context, userID uuid.UUID) error {
	isSuperuser, err := s.IsSuperuser(ctx, userID)
	if err != nil {
		return err
	}
	if !isSuperuser {
		return ErrNotSuperuser
	}
	return nil
}

// GrantSuperuser grants superuser privileges to a user.
func (s *Superuser) GrantSuperuser(ctx context.Context, grantedBy, targetUserID uuid.UUID) error {
	// Verify the granter is a superuser
	if err := s.RequireSuperuser(ctx, grantedBy); err != nil {
		return err
	}
	return s.store.SetUserSuperuser(ctx, targetUserID, true)
}

// RevokeSuperuser revokes superuser privileges from a user.
func (s *Superuser) RevokeSuperuser(ctx context.Context, revokedBy, targetUserID uuid.UUID) error {
	// Verify the revoker is a superuser
	if err := s.RequireSuperuser(ctx, revokedBy); err != nil {
		return err
	}

	// Prevent revoking own superuser status
	if revokedBy == targetUserID {
		return ErrCannotRevokeSelf
	}

	// Ensure at least one superuser remains
	superusers, err := s.store.GetSuperusers(ctx)
	if err != nil {
		return fmt.Errorf("get superusers: %w", err)
	}
	if len(superusers) <= 1 {
		return ErrLastSuperuser
	}

	return s.store.SetUserSuperuser(ctx, targetUserID, false)
}

// GetAllOrganizations returns all organizations (superuser only).
func (s *Superuser) GetAllOrganizations(ctx context.Context, userID uuid.UUID) ([]*models.Organization, error) {
	if err := s.RequireSuperuser(ctx, userID); err != nil {
		return nil, err
	}
	return s.store.GetAllOrganizations(ctx)
}

// GetAllUsers returns all users across all organizations (superuser only).
func (s *Superuser) GetAllUsers(ctx context.Context, userID uuid.UUID) ([]*models.User, error) {
	if err := s.RequireSuperuser(ctx, userID); err != nil {
		return nil, err
	}
	return s.store.GetAllUsers(ctx)
}

// GetSystemSetting returns a system setting by key (superuser only).
func (s *Superuser) GetSystemSetting(ctx context.Context, userID uuid.UUID, key string) (*models.SystemSetting, error) {
	if err := s.RequireSuperuser(ctx, userID); err != nil {
		return nil, err
	}
	return s.store.GetSystemSetting(ctx, key)
}

// GetSystemSettings returns all system settings (superuser only).
func (s *Superuser) GetSystemSettings(ctx context.Context, userID uuid.UUID) ([]*models.SystemSetting, error) {
	if err := s.RequireSuperuser(ctx, userID); err != nil {
		return nil, err
	}
	return s.store.GetSystemSettings(ctx)
}

// UpdateSystemSetting updates a system setting (superuser only).
func (s *Superuser) UpdateSystemSetting(ctx context.Context, userID uuid.UUID, key string, value interface{}) error {
	if err := s.RequireSuperuser(ctx, userID); err != nil {
		return err
	}
	return s.store.UpdateSystemSetting(ctx, key, value, userID)
}

// ErrNotSuperuser is returned when a user is not a superuser.
var ErrNotSuperuser = fmt.Errorf("superuser privileges required")

// ErrCannotRevokeSelf is returned when a superuser tries to revoke their own privileges.
var ErrCannotRevokeSelf = fmt.Errorf("cannot revoke own superuser privileges")

// ErrLastSuperuser is returned when attempting to remove the last superuser.
var ErrLastSuperuser = fmt.Errorf("cannot remove the last superuser")
