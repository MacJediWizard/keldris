package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type stubIPFilterStore struct {
	settings    *models.IPAllowlistSettings
	allowlist   []*models.IPAllowlist
	user        *models.User
	settingsErr error
	allowErr    error
}

func (s *stubIPFilterStore) GetOrCreateIPAllowlistSettings(_ context.Context, _ uuid.UUID) (*models.IPAllowlistSettings, error) {
	if s.settingsErr != nil {
		return nil, s.settingsErr
	}
	if s.settings != nil {
		return s.settings, nil
	}
	return &models.IPAllowlistSettings{Enabled: false}, nil
}

func (s *stubIPFilterStore) ListEnabledIPAllowlistsByOrg(_ context.Context, _ uuid.UUID, _ models.IPAllowlistType) ([]*models.IPAllowlist, error) {
	return s.allowlist, s.allowErr
}

func (s *stubIPFilterStore) CreateIPBlockedAttempt(_ context.Context, _ *models.IPBlockedAttempt) error {
	return nil
}

func (s *stubIPFilterStore) GetUserByID(_ context.Context, _ uuid.UUID) (*models.User, error) {
	return s.user, nil
}

func TestNewIPFilterCache(t *testing.T) {
	cache := NewIPFilterCache(5 * time.Second)
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}
}

func TestNewIPFilter(t *testing.T) {
	filter := NewIPFilter(&stubIPFilterStore{}, zerolog.Nop())
	if filter == nil {
		t.Fatal("expected non-nil filter")
	}
}

func TestCheckIP_DisabledAllowsAll(t *testing.T) {
	store := &stubIPFilterStore{settings: &models.IPAllowlistSettings{Enabled: false}}
	filter := NewIPFilter(store, zerolog.Nop())

	allowed, _ := filter.CheckIP(context.Background(), uuid.New(), "1.2.3.4", models.IPAllowlistTypeUI, false)
	if !allowed {
		t.Fatal("expected allowed when filtering disabled")
	}
}

func TestCheckIP_StoreErrorFailsOpen(t *testing.T) {
	store := &stubIPFilterStore{settingsErr: errors.New("db down")}
	filter := NewIPFilter(store, zerolog.Nop())

	allowed, _ := filter.CheckIP(context.Background(), uuid.New(), "1.2.3.4", models.IPAllowlistTypeUI, false)
	if !allowed {
		t.Fatal("expected allowed on store error (fail-open)")
	}
}

func TestCheckIP_UINotEnforced(t *testing.T) {
	store := &stubIPFilterStore{settings: &models.IPAllowlistSettings{Enabled: true, EnforceForUI: false}}
	filter := NewIPFilter(store, zerolog.Nop())

	allowed, _ := filter.CheckIP(context.Background(), uuid.New(), "1.2.3.4", models.IPAllowlistTypeUI, false)
	if !allowed {
		t.Fatal("expected allowed when UI not enforced")
	}
}

func TestCheckIP_AdminBypass(t *testing.T) {
	store := &stubIPFilterStore{
		settings: &models.IPAllowlistSettings{
			Enabled:          true,
			EnforceForUI:     true,
			AllowAdminBypass: true,
		},
	}
	filter := NewIPFilter(store, zerolog.Nop())

	allowed, _ := filter.CheckIP(context.Background(), uuid.New(), "1.2.3.4", models.IPAllowlistTypeUI, true)
	if !allowed {
		t.Fatal("expected admin bypass to allow")
	}
}

func TestCheckIP_NoEntriesDenied(t *testing.T) {
	store := &stubIPFilterStore{
		settings:  &models.IPAllowlistSettings{Enabled: true, EnforceForUI: true},
		allowlist: []*models.IPAllowlist{},
	}
	filter := NewIPFilter(store, zerolog.Nop())

	allowed, reason := filter.CheckIP(context.Background(), uuid.New(), "1.2.3.4", models.IPAllowlistTypeUI, false)
	if allowed {
		t.Fatal("expected denied when no entries configured")
	}
	if reason == "" {
		t.Fatal("expected reason to be set")
	}
}

func TestCheckIP_AllowedByCIDR(t *testing.T) {
	store := &stubIPFilterStore{
		settings: &models.IPAllowlistSettings{Enabled: true, EnforceForUI: true},
		allowlist: []*models.IPAllowlist{
			{CIDR: "10.0.0.0/8", Type: models.IPAllowlistTypeUI},
		},
	}
	filter := NewIPFilter(store, zerolog.Nop())

	allowed, _ := filter.CheckIP(context.Background(), uuid.New(), "10.1.2.3", models.IPAllowlistTypeUI, false)
	if !allowed {
		t.Fatal("expected 10.1.2.3 allowed by 10.0.0.0/8")
	}
}

func TestCheckIP_DeniedNotInCIDR(t *testing.T) {
	store := &stubIPFilterStore{
		settings: &models.IPAllowlistSettings{Enabled: true, EnforceForUI: true},
		allowlist: []*models.IPAllowlist{
			{CIDR: "10.0.0.0/8", Type: models.IPAllowlistTypeUI},
		},
	}
	filter := NewIPFilter(store, zerolog.Nop())

	allowed, reason := filter.CheckIP(context.Background(), uuid.New(), "192.168.1.1", models.IPAllowlistTypeUI, false)
	if allowed {
		t.Fatal("expected 192.168.1.1 denied")
	}
	if reason == "" {
		t.Fatal("expected reason")
	}
}

func TestInvalidateCache(t *testing.T) {
	filter := NewIPFilter(&stubIPFilterStore{}, zerolog.Nop())
	// Should not panic
	filter.InvalidateCache(uuid.New())
}
