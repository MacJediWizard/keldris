package telemetry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// mockCollector implements DataCollector for testing.
type mockCollector struct {
	counts   *TelemetryCounts
	features *TelemetryFeatures
	err      error
}

func (m *mockCollector) CollectTelemetryData(ctx context.Context) (*TelemetryCounts, *TelemetryFeatures, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.counts, m.features, nil
}

func TestDefaultSettings(t *testing.T) {
	settings := DefaultSettings()

	if settings.Enabled {
		t.Error("telemetry should be disabled by default")
	}

	if settings.InstallID == "" {
		t.Error("install_id should be generated")
	}

	if _, err := uuid.Parse(settings.InstallID); err != nil {
		t.Errorf("install_id should be valid UUID: %v", err)
	}

	if settings.Endpoint != DefaultEndpoint {
		t.Errorf("endpoint should be default: got %s, want %s", settings.Endpoint, DefaultEndpoint)
	}
}

func TestSettingsValidate(t *testing.T) {
	tests := []struct {
		name    string
		setting Settings
		wantErr bool
	}{
		{
			name:    "valid settings",
			setting: DefaultSettings(),
			wantErr: false,
		},
		{
			name: "empty install_id",
			setting: Settings{
				InstallID: "",
			},
			wantErr: true,
		},
		{
			name: "invalid install_id",
			setting: Settings{
				InstallID: "not-a-uuid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.setting.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTelemetryDataMarshal(t *testing.T) {
	data := TelemetryData{
		InstallID:   uuid.New().String(),
		CollectedAt: time.Now().UTC(),
		Version:     "1.0.0",
		Commit:      "abc123",
		Counts: TelemetryCounts{
			TotalAgents:       10,
			ActiveAgents:      8,
			TotalBackups:      100,
			SuccessfulBackups: 95,
		},
		Features: TelemetryFeatures{
			OIDCEnabled:  true,
			SMTPEnabled:  false,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal telemetry data: %v", err)
	}

	var unmarshaled TelemetryData
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("failed to unmarshal telemetry data: %v", err)
	}

	if unmarshaled.InstallID != data.InstallID {
		t.Errorf("install_id mismatch: got %s, want %s", unmarshaled.InstallID, data.InstallID)
	}
	if unmarshaled.Counts.TotalAgents != data.Counts.TotalAgents {
		t.Errorf("total_agents mismatch: got %d, want %d", unmarshaled.Counts.TotalAgents, data.Counts.TotalAgents)
	}
}

func TestServiceCollect(t *testing.T) {
	logger := zerolog.Nop()
	collector := &mockCollector{
		counts: &TelemetryCounts{
			TotalAgents:       5,
			ActiveAgents:      3,
			TotalBackups:      50,
			SuccessfulBackups: 48,
		},
		features: &TelemetryFeatures{
			OIDCEnabled: true,
		},
	}

	service := NewService(collector, "1.0.0", "abc123", "2024-01-01", logger)
	settings := DefaultSettings()
	settings.Enabled = true
	service.SetSettings(settings)

	data, err := service.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	if data.Counts.TotalAgents != 5 {
		t.Errorf("total_agents = %d, want 5", data.Counts.TotalAgents)
	}
	if data.Version != "1.0.0" {
		t.Errorf("version = %s, want 1.0.0", data.Version)
	}
	if !data.Features.OIDCEnabled {
		t.Error("oidc_enabled should be true")
	}
}

func TestServiceSend(t *testing.T) {
	var receivedData TelemetryData
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json, got %s", r.Header.Get("Content-Type"))
		}

		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&receivedData); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	collector := &mockCollector{
		counts:   &TelemetryCounts{TotalAgents: 10},
		features: &TelemetryFeatures{},
	}

	service := NewService(collector, "1.0.0", "abc123", "2024-01-01", logger)
	settings := DefaultSettings()
	settings.Enabled = true
	settings.Endpoint = server.URL
	service.SetSettings(settings)

	data := &TelemetryData{
		InstallID:   settings.InstallID,
		CollectedAt: time.Now().UTC(),
		Version:     "1.0.0",
		Counts:      TelemetryCounts{TotalAgents: 10},
		Features:    TelemetryFeatures{},
	}

	err := service.Send(context.Background(), data)
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	if receivedData.InstallID != data.InstallID {
		t.Errorf("received install_id = %s, want %s", receivedData.InstallID, data.InstallID)
	}
}

func TestServiceIsEnabled(t *testing.T) {
	logger := zerolog.Nop()
	collector := &mockCollector{}

	service := NewService(collector, "1.0.0", "", "", logger)

	if service.IsEnabled() {
		t.Error("service should be disabled by default")
	}

	settings := DefaultSettings()
	settings.Enabled = true
	service.SetSettings(settings)

	if !service.IsEnabled() {
		t.Error("service should be enabled after SetSettings")
	}
}

func TestGetPrivacyExplanation(t *testing.T) {
	explanation := GetPrivacyExplanation()

	if explanation == "" {
		t.Error("privacy explanation should not be empty")
	}

	// Check for key phrases
	mustContain := []string{
		"WHAT WE COLLECT",
		"WHAT WE DO NOT COLLECT",
		"opt-in",
		"Hostnames",
		"IP addresses",
	}

	for _, phrase := range mustContain {
		if !containsString(explanation, phrase) {
			t.Errorf("explanation should contain %q", phrase)
		}
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || containsString(s[1:], substr)))
}

func TestTelemetryCountsNoSensitiveData(t *testing.T) {
	// Verify that TelemetryCounts only contains safe, aggregate data
	counts := TelemetryCounts{
		TotalAgents:        100,
		ActiveAgents:       50,
		TotalBackups:       1000,
		SuccessfulBackups:  950,
		TotalRepositories:  10,
		TotalSchedules:     25,
		TotalOrganizations: 5,
		TotalUsers:         20,
	}

	data, err := json.Marshal(counts)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// Check that no sensitive field names appear
	sensitiveFields := []string{
		"hostname", "ip", "path", "email", "name", "url", "credential", "secret", "password",
	}

	for _, field := range sensitiveFields {
		if containsString(string(data), field) {
			t.Errorf("telemetry data should not contain field: %s", field)
		}
	}
}
