package diagnostics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/MacJediWizard/keldris/internal/config"
)

func TestRunner_Run(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		case "/api/v1/agent/commands":
			// Check API key
			if r.Header.Get("X-API-Key") == "test-api-key" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"commands":[]}`))
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	cfg := &config.AgentConfig{
		ServerURL: ts.URL,
		APIKey:    "test-api-key",
	}

	runner := NewRunner(cfg, "test-version")
	result := runner.Run(context.Background())

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	if result.AgentVersion != "test-version" {
		t.Errorf("expected agent version 'test-version', got %s", result.AgentVersion)
	}

	if result.Summary.Total == 0 {
		t.Error("expected at least one check to be run")
	}

	// Server connectivity check should pass
	serverCheck := findCheck(result.Checks, "server_connectivity")
	if serverCheck == nil {
		t.Error("expected server_connectivity check")
	} else if serverCheck.Status != StatusPass {
		t.Errorf("expected server_connectivity to pass, got %s: %s", serverCheck.Status, serverCheck.Message)
	}

	// API key check should pass
	apiKeyCheck := findCheck(result.Checks, "api_key")
	if apiKeyCheck == nil {
		t.Error("expected api_key check")
	} else if apiKeyCheck.Status != StatusPass {
		t.Errorf("expected api_key to pass, got %s: %s", apiKeyCheck.Status, apiKeyCheck.Message)
	}
}

func TestRunner_Run_NoConfig(t *testing.T) {
	cfg := &config.AgentConfig{}

	runner := NewRunner(cfg, "test-version")
	result := runner.Run(context.Background())

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	// Server connectivity check should be skipped
	serverCheck := findCheck(result.Checks, "server_connectivity")
	if serverCheck == nil {
		t.Error("expected server_connectivity check")
	} else if serverCheck.Status != StatusSkip {
		t.Errorf("expected server_connectivity to be skipped, got %s", serverCheck.Status)
	}

	// API key check should be skipped
	apiKeyCheck := findCheck(result.Checks, "api_key")
	if apiKeyCheck == nil {
		t.Error("expected api_key check")
	} else if apiKeyCheck.Status != StatusSkip {
		t.Errorf("expected api_key to be skipped, got %s", apiKeyCheck.Status)
	}
}

func TestRunner_Run_InvalidAPIKey(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/agent/commands":
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer ts.Close()

	cfg := &config.AgentConfig{
		ServerURL: ts.URL,
		APIKey:    "invalid-key",
	}

	runner := NewRunner(cfg, "test-version")
	result := runner.Run(context.Background())

	apiKeyCheck := findCheck(result.Checks, "api_key")
	if apiKeyCheck == nil {
		t.Error("expected api_key check")
	} else if apiKeyCheck.Status != StatusFail {
		t.Errorf("expected api_key to fail, got %s", apiKeyCheck.Status)
	}
}

func TestRunner_DiskSpace(t *testing.T) {
	cfg := &config.AgentConfig{}

	runner := NewRunner(cfg, "test-version")
	result := runner.Run(context.Background())

	diskCheck := findCheck(result.Checks, "disk_space")
	if diskCheck == nil {
		t.Error("expected disk_space check")
	} else {
		// Disk check should pass or warn (not fail unless disk is really full)
		if diskCheck.Status == StatusFail {
			t.Logf("disk_space check failed (disk may be full): %s", diskCheck.Message)
		}
	}
}

func TestRunner_ConfigPermissions(t *testing.T) {
	cfg := &config.AgentConfig{}

	runner := NewRunner(cfg, "test-version")
	result := runner.Run(context.Background())

	configCheck := findCheck(result.Checks, "config_permissions")
	if configCheck == nil {
		t.Error("expected config_permissions check")
	}
}

func TestDiagnosticsResult_ToJSON(t *testing.T) {
	result := &DiagnosticsResult{
		AgentVersion: "1.0.0",
		Hostname:     "test-host",
		OS:           "linux",
		Arch:         "amd64",
		Checks: []CheckResult{
			{
				Name:    "test_check",
				Status:  StatusPass,
				Message: "Test passed",
			},
		},
		Summary: Summary{
			Total:   1,
			Passed:  1,
			AllPass: true,
		},
	}

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("expected non-empty JSON output")
	}

	// Verify it contains expected fields
	json := string(data)
	if !contains(json, "agent_version") {
		t.Error("expected JSON to contain agent_version")
	}
	if !contains(json, "test_check") {
		t.Error("expected JSON to contain test_check")
	}
}

func TestDiagnosticsResult_ToMap(t *testing.T) {
	result := &DiagnosticsResult{
		AgentVersion: "1.0.0",
		Hostname:     "test-host",
	}

	m := result.ToMap()
	if m["agent_version"] != "1.0.0" {
		t.Errorf("expected agent_version '1.0.0', got %v", m["agent_version"])
	}
	if m["hostname"] != "test-host" {
		t.Errorf("expected hostname 'test-host', got %v", m["hostname"])
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestGetDiskSpace(t *testing.T) {
	// Test with home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("could not get home directory")
	}

	details, err := getDiskSpace(homeDir)
	if err != nil {
		t.Fatalf("getDiskSpace failed: %v", err)
	}

	if details.TotalBytes <= 0 {
		t.Error("expected positive total bytes")
	}
	if details.FreeBytes < 0 {
		t.Error("expected non-negative free bytes")
	}
	if details.UsedPct < 0 || details.UsedPct > 100 {
		t.Errorf("expected used percent between 0 and 100, got %.2f", details.UsedPct)
	}
}

// Helper functions

func findCheck(checks []CheckResult, name string) *CheckResult {
	for i := range checks {
		if checks[i].Name == name {
			return &checks[i]
		}
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
