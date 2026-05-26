package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type mockTelemetryStore struct {
	settings *telemetry.Settings
	counts   *telemetry.TelemetryCounts
	features *telemetry.TelemetryFeatures
	err      error
}

func (m *mockTelemetryStore) GetTelemetrySettings(_ context.Context) (*telemetry.Settings, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.settings != nil {
		return m.settings, nil
	}
	return &telemetry.Settings{Enabled: false}, nil
}

func (m *mockTelemetryStore) UpdateTelemetrySettings(_ context.Context, s *telemetry.Settings) error {
	if m.err != nil {
		return m.err
	}
	m.settings = s
	return nil
}

func (m *mockTelemetryStore) EnableTelemetry(_ context.Context, _ string) error {
	return m.err
}

func (m *mockTelemetryStore) DisableTelemetry(_ context.Context) error {
	return m.err
}

func (m *mockTelemetryStore) UpdateTelemetryLastSent(_ context.Context, _ time.Time, _ *telemetry.TelemetryData) error {
	return m.err
}

func (m *mockTelemetryStore) CollectTelemetryData(_ context.Context) (*telemetry.TelemetryCounts, *telemetry.TelemetryFeatures, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	if m.counts == nil {
		m.counts = &telemetry.TelemetryCounts{}
	}
	if m.features == nil {
		m.features = &telemetry.TelemetryFeatures{}
	}
	return m.counts, m.features, nil
}

func setupTelemetryTestRouter(store TelemetryStore, user *auth.SessionUser) *gin.Engine {
	r := SetupTestRouter(user)
	handler := NewTelemetryHandler(store, nil, zerolog.Nop())
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	return r
}

func TestTelemetryGetStatus(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)

	t.Run("admin sees status", func(t *testing.T) {
		store := &mockTelemetryStore{}
		r := setupTelemetryTestRouter(store, user)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/telemetry"))
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
		}
	})

	t.Run("non-admin forbidden", func(t *testing.T) {
		viewer := testUser(orgID)
		viewer.CurrentOrgRole = "viewer"
		store := &mockTelemetryStore{}
		r := setupTelemetryTestRouter(store, viewer)

		resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/telemetry"))
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", resp.Code)
		}
	})
}

func TestTelemetryGetPrivacyExplanation(t *testing.T) {
	user := testUser(uuid.New())
	store := &mockTelemetryStore{}
	r := setupTelemetryTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/telemetry/privacy"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestTelemetryPreview(t *testing.T) {
	orgID := uuid.New()
	user := testUser(orgID)
	store := &mockTelemetryStore{}
	r := setupTelemetryTestRouter(store, user)

	resp := DoRequest(r, AuthenticatedRequest("GET", "/api/v1/telemetry/preview"))
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}
