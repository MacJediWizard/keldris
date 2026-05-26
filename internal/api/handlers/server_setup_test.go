package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/settings"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockServerSetupStore struct {
	setup    *models.ServerSetup
	complete bool
	hasSuper bool
	hasOrg   bool
	license  *models.LicenseKey
	smtp     *settings.SMTPSettings
	oidc     *settings.OIDCSettings
	err      error
}

func (m *mockServerSetupStore) GetServerSetup(_ context.Context) (*models.ServerSetup, error) {
	if m.setup != nil {
		return m.setup, m.err
	}
	return &models.ServerSetup{SetupCompleted: m.complete}, m.err
}

func (m *mockServerSetupStore) IsSetupComplete(_ context.Context) (bool, error) {
	return m.complete, m.err
}

func (m *mockServerSetupStore) CompleteSetupStep(_ context.Context, _ models.ServerSetupStep) error {
	return m.err
}

func (m *mockServerSetupStore) FinalizeSetup(_ context.Context, _ *uuid.UUID) error {
	return m.err
}

func (m *mockServerSetupStore) HasAnySuperuser(_ context.Context) (bool, error) {
	return m.hasSuper, m.err
}

func (m *mockServerSetupStore) CreateSuperuserWithPassword(_ context.Context, _, _, _ string) (*models.User, *models.Organization, error) {
	return nil, nil, m.err
}

func (m *mockServerSetupStore) GetActiveLicense(_ context.Context) (*models.LicenseKey, error) {
	return m.license, m.err
}

func (m *mockServerSetupStore) ActivateLicense(_ context.Context, _ string, _ *uuid.UUID) (*models.LicenseKey, error) {
	return m.license, m.err
}

func (m *mockServerSetupStore) CreateTrialLicense(_ context.Context, _, _ string, _ *uuid.UUID) (*models.LicenseKey, error) {
	return m.license, m.err
}

func (m *mockServerSetupStore) HasAnyOrganization(_ context.Context) (bool, error) {
	return m.hasOrg, m.err
}

func (m *mockServerSetupStore) CreateFirstOrganization(_ context.Context, _ string, _ uuid.UUID) (*models.Organization, error) {
	return nil, m.err
}

func (m *mockServerSetupStore) GetSMTPSettings(_ context.Context, _ uuid.UUID) (*settings.SMTPSettings, error) {
	return m.smtp, m.err
}

func (m *mockServerSetupStore) UpdateSMTPSettings(_ context.Context, _ uuid.UUID, _ *settings.SMTPSettings) error {
	return m.err
}

func (m *mockServerSetupStore) GetOIDCSettings(_ context.Context, _ uuid.UUID) (*settings.OIDCSettings, error) {
	return m.oidc, m.err
}

func (m *mockServerSetupStore) UpdateOIDCSettings(_ context.Context, _ uuid.UUID, _ *settings.OIDCSettings) error {
	return m.err
}

func (m *mockServerSetupStore) EnsureSystemSettingsExist(_ context.Context, _ uuid.UUID) error {
	return m.err
}

func (m *mockServerSetupStore) CreateServerSetupAuditLog(_ context.Context, _ *models.ServerSetupAuditLog) error {
	return nil
}

type mockDBPinger struct{ err error }

func (m *mockDBPinger) Ping(_ context.Context) error { return m.err }

// setupServerSetupTestRouter bypasses SetupLockMiddleware (which requires a real *DB).
func setupServerSetupTestRouter(store ServerSetupStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewServerSetupHandler(store, &mockDBPinger{}, nil, zerolog.Nop())
	r.GET("/api/v1/setup/status", handler.GetStatus)
	return r
}

func TestServerSetupGetStatus(t *testing.T) {
	store := &mockServerSetupStore{}
	r := setupServerSetupTestRouter(store, testUser(uuid.New()))

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/setup/status"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
