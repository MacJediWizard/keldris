package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

type stubSuperuserStore struct {
	users       map[uuid.UUID]*models.User
	superusers  []*models.User
	orgs        []*models.Organization
	allUsers    []*models.User
	setting     *models.SystemSetting
	settings    []*models.SystemSetting
	getUserErr  error
	setSuperErr error
}

func (s *stubSuperuserStore) GetUserByID(_ context.Context, id uuid.UUID) (*models.User, error) {
	if s.getUserErr != nil {
		return nil, s.getUserErr
	}
	if u, ok := s.users[id]; ok {
		return u, nil
	}
	return &models.User{ID: id}, nil
}

func (s *stubSuperuserStore) GetAllOrganizations(_ context.Context) ([]*models.Organization, error) {
	return s.orgs, nil
}

func (s *stubSuperuserStore) GetAllUsers(_ context.Context) ([]*models.User, error) {
	return s.allUsers, nil
}

func (s *stubSuperuserStore) SetUserSuperuser(_ context.Context, _ uuid.UUID, _ bool) error {
	return s.setSuperErr
}

func (s *stubSuperuserStore) GetSuperusers(_ context.Context) ([]*models.User, error) {
	return s.superusers, nil
}

func (s *stubSuperuserStore) GetUserByEmail(_ context.Context, _ string) (*models.User, error) {
	return nil, nil
}

func (s *stubSuperuserStore) CreateSuperuserAuditLog(_ context.Context, _ *models.SuperuserAuditLog) error {
	return nil
}

func (s *stubSuperuserStore) GetSuperuserAuditLogs(_ context.Context, _, _ int) ([]*models.SuperuserAuditLogWithUser, int, error) {
	return nil, 0, nil
}

func (s *stubSuperuserStore) GetSystemSetting(_ context.Context, _ string) (*models.SystemSetting, error) {
	return s.setting, nil
}

func (s *stubSuperuserStore) GetSystemSettings(_ context.Context) ([]*models.SystemSetting, error) {
	return s.settings, nil
}

func (s *stubSuperuserStore) UpdateSystemSetting(_ context.Context, _ string, _ interface{}, _ uuid.UUID) error {
	return nil
}

func TestIsSuperuser_True(t *testing.T) {
	id := uuid.New()
	store := &stubSuperuserStore{users: map[uuid.UUID]*models.User{id: {ID: id, IsSuperuser: true}}}
	s := NewSuperuser(store)

	ok, err := s.IsSuperuser(context.Background(), id)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !ok {
		t.Error("expected true")
	}
}

func TestIsSuperuser_False(t *testing.T) {
	id := uuid.New()
	store := &stubSuperuserStore{users: map[uuid.UUID]*models.User{id: {ID: id, IsSuperuser: false}}}
	s := NewSuperuser(store)

	ok, _ := s.IsSuperuser(context.Background(), id)
	if ok {
		t.Error("expected false")
	}
}

func TestIsSuperuser_StoreError(t *testing.T) {
	store := &stubSuperuserStore{getUserErr: errors.New("db down")}
	s := NewSuperuser(store)

	_, err := s.IsSuperuser(context.Background(), uuid.New())
	if err == nil {
		t.Error("expected error")
	}
}

func TestRequireSuperuser_NonSuperuserError(t *testing.T) {
	id := uuid.New()
	store := &stubSuperuserStore{users: map[uuid.UUID]*models.User{id: {ID: id, IsSuperuser: false}}}
	s := NewSuperuser(store)

	if err := s.RequireSuperuser(context.Background(), id); err != ErrNotSuperuser {
		t.Errorf("expected ErrNotSuperuser, got %v", err)
	}
}

func TestGrantSuperuser_RequiresGranterSuperuser(t *testing.T) {
	granter := uuid.New()
	target := uuid.New()
	store := &stubSuperuserStore{users: map[uuid.UUID]*models.User{granter: {ID: granter, IsSuperuser: false}}}
	s := NewSuperuser(store)

	if err := s.GrantSuperuser(context.Background(), granter, target); err == nil {
		t.Error("expected error when granter not superuser")
	}
}

func TestGrantSuperuser_Success(t *testing.T) {
	granter := uuid.New()
	target := uuid.New()
	store := &stubSuperuserStore{users: map[uuid.UUID]*models.User{granter: {ID: granter, IsSuperuser: true}}}
	s := NewSuperuser(store)

	if err := s.GrantSuperuser(context.Background(), granter, target); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestRevokeSuperuser_CannotRevokeSelf(t *testing.T) {
	id := uuid.New()
	store := &stubSuperuserStore{users: map[uuid.UUID]*models.User{id: {ID: id, IsSuperuser: true}}}
	s := NewSuperuser(store)

	if err := s.RevokeSuperuser(context.Background(), id, id); err != ErrCannotRevokeSelf {
		t.Errorf("expected ErrCannotRevokeSelf, got %v", err)
	}
}

func TestRevokeSuperuser_LastSuperuser(t *testing.T) {
	revoker := uuid.New()
	target := uuid.New()
	store := &stubSuperuserStore{
		users:      map[uuid.UUID]*models.User{revoker: {ID: revoker, IsSuperuser: true}},
		superusers: []*models.User{{ID: target}},
	}
	s := NewSuperuser(store)

	if err := s.RevokeSuperuser(context.Background(), revoker, target); err != ErrLastSuperuser {
		t.Errorf("expected ErrLastSuperuser, got %v", err)
	}
}

func TestRevokeSuperuser_Success(t *testing.T) {
	revoker := uuid.New()
	target := uuid.New()
	store := &stubSuperuserStore{
		users:      map[uuid.UUID]*models.User{revoker: {ID: revoker, IsSuperuser: true}},
		superusers: []*models.User{{ID: revoker}, {ID: target}},
	}
	s := NewSuperuser(store)

	if err := s.RevokeSuperuser(context.Background(), revoker, target); err != nil {
		t.Errorf("expected success, got %v", err)
	}
}

func TestGetAllOrganizations_RequiresSuperuser(t *testing.T) {
	id := uuid.New()
	store := &stubSuperuserStore{users: map[uuid.UUID]*models.User{id: {ID: id, IsSuperuser: false}}}
	s := NewSuperuser(store)

	_, err := s.GetAllOrganizations(context.Background(), id)
	if err == nil {
		t.Error("expected error for non-superuser")
	}
}

func TestGetAllUsers_PassesForSuperuser(t *testing.T) {
	id := uuid.New()
	store := &stubSuperuserStore{
		users:    map[uuid.UUID]*models.User{id: {ID: id, IsSuperuser: true}},
		allUsers: []*models.User{{ID: uuid.New()}},
	}
	s := NewSuperuser(store)

	users, err := s.GetAllUsers(context.Background(), id)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("expected 1 user, got %d", len(users))
	}
}
