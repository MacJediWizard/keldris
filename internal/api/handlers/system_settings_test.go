package handlers

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockSystemSettingsStore struct {
	all       *settings.SystemSettingsResponse
	smtp      *settings.SMTPSettings
	oidc      *settings.OIDCSettings
	storage   *settings.StorageDefaultSettings
	security  *settings.SecuritySettings
	audit     []*settings.SettingsAuditLog
	getErr    error
	updateErr error
	auditErr  error
}

func (m *mockSystemSettingsStore) GetAllSettings(_ context.Context, _ uuid.UUID) (*settings.SystemSettingsResponse, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.all != nil {
		return m.all, nil
	}
	return &settings.SystemSettingsResponse{
		SMTP:            settings.SMTPSettings{},
		OIDC:            settings.OIDCSettings{},
		StorageDefaults: settings.StorageDefaultSettings{},
		Security:        settings.SecuritySettings{},
	}, nil
}

func (m *mockSystemSettingsStore) GetSMTPSettings(_ context.Context, _ uuid.UUID) (*settings.SMTPSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.smtp != nil {
		return m.smtp, nil
	}
	return &settings.SMTPSettings{}, nil
}

func (m *mockSystemSettingsStore) UpdateSMTPSettings(_ context.Context, _ uuid.UUID, _ *settings.SMTPSettings) error {
	return m.updateErr
}

func (m *mockSystemSettingsStore) GetOIDCSettings(_ context.Context, _ uuid.UUID) (*settings.OIDCSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.oidc != nil {
		return m.oidc, nil
	}
	return &settings.OIDCSettings{}, nil
}

func (m *mockSystemSettingsStore) UpdateOIDCSettings(_ context.Context, _ uuid.UUID, _ *settings.OIDCSettings) error {
	return m.updateErr
}

func (m *mockSystemSettingsStore) GetStorageDefaultSettings(_ context.Context, _ uuid.UUID) (*settings.StorageDefaultSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.storage != nil {
		return m.storage, nil
	}
	return &settings.StorageDefaultSettings{}, nil
}

func (m *mockSystemSettingsStore) UpdateStorageDefaultSettings(_ context.Context, _ uuid.UUID, _ *settings.StorageDefaultSettings) error {
	return m.updateErr
}

func (m *mockSystemSettingsStore) GetSecuritySettings(_ context.Context, _ uuid.UUID) (*settings.SecuritySettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.security != nil {
		return m.security, nil
	}
	return &settings.SecuritySettings{}, nil
}

func (m *mockSystemSettingsStore) UpdateSecuritySettings(_ context.Context, _ uuid.UUID, _ *settings.SecuritySettings) error {
	return m.updateErr
}

func (m *mockSystemSettingsStore) CreateSettingsAuditLog(_ context.Context, _ *settings.SettingsAuditLog) error {
	return m.auditErr
}

func (m *mockSystemSettingsStore) GetSettingsAuditLogs(_ context.Context, _ uuid.UUID, _, _ int) ([]*settings.SettingsAuditLog, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.audit, nil
}

func (m *mockSystemSettingsStore) EnsureSystemSettingsExist(_ context.Context, _ uuid.UUID) error {
	return nil
}

// mockSysSettingsFeatureStore returns Enterprise tier so OIDC feature is allowed.
type mockSysSettingsFeatureStore struct{}

func (m *mockSysSettingsFeatureStore) GetOrgTier(_ context.Context, _ uuid.UUID) (license.Tier, error) {
	return license.TierEnterprise, nil
}

func (m *mockSysSettingsFeatureStore) SetOrgTier(_ context.Context, _ uuid.UUID, _ license.Tier) error {
	return nil
}

func setupSystemSettingsTestRouter(store SystemSettingsStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	checker := license.NewFeatureChecker(&mockSysSettingsFeatureStore{})
	handler := NewSystemSettingsHandler(store, checker, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestSystemSettingsGetAll(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns all settings", func(t *testing.T) {
		store := &mockSystemSettingsStore{}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("missing org returns 400", func(t *testing.T) {
		store := &mockSystemSettingsStore{}
		r := setupSystemSettingsTestRouter(store, testUserNoOrg())
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings"))
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.Code)
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSystemSettingsStore{getErr: errors.New("db down")}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestSystemSettingsGetSMTP(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns smtp settings", func(t *testing.T) {
		store := &mockSystemSettingsStore{smtp: &settings.SMTPSettings{Host: "smtp.test", Password: "secret"}}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/smtp"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSystemSettingsStore{getErr: errors.New("db down")}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/smtp"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestSystemSettingsGetOIDC(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns oidc settings", func(t *testing.T) {
		store := &mockSystemSettingsStore{oidc: &settings.OIDCSettings{Issuer: "https://oidc.test"}}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/oidc"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSystemSettingsStore{getErr: errors.New("db down")}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/oidc"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestSystemSettingsGetStorageDefaults(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns storage defaults", func(t *testing.T) {
		store := &mockSystemSettingsStore{}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/storage"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSystemSettingsStore{getErr: errors.New("db down")}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/storage"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestSystemSettingsGetSecurity(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns security settings", func(t *testing.T) {
		store := &mockSystemSettingsStore{}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/security"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSystemSettingsStore{getErr: errors.New("db down")}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/security"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}

func TestSystemSettingsGetAuditLog(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("returns audit log", func(t *testing.T) {
		store := &mockSystemSettingsStore{audit: []*settings.SettingsAuditLog{}}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/audit-log"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("store error returns 500", func(t *testing.T) {
		store := &mockSystemSettingsStore{getErr: errors.New("db down")}
		r := setupSystemSettingsTestRouter(store, user)
		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/system-settings/audit-log"))
		if resp.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500, got %d", resp.Code)
		}
	})
}
