package updates

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "1.0.0"},
		{"1.0.0", "1.0.0"},
		{"  v2.1.3  ", "2.1.3"},
		{"dev", "dev"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeVersion(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input    string
		expected [3]int
	}{
		{"1.0.0", [3]int{1, 0, 0}},
		{"2.1.3", [3]int{2, 1, 3}},
		{"10.20.30", [3]int{10, 20, 30}},
		{"1.2.3-rc1", [3]int{1, 2, 3}},
		{"1.2.3-beta+build123", [3]int{1, 2, 3}},
		{"1.2", [3]int{1, 2, 0}},
		{"1", [3]int{1, 0, 0}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSemver(tt.input)
			if result != tt.expected {
				t.Errorf("parseSemver(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest   string
		current  string
		expected bool
	}{
		// Basic comparisons
		{"1.0.0", "0.9.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.1.0", "1.0.9", true},
		{"2.0.0", "1.9.9", true},

		// Same version
		{"1.0.0", "1.0.0", false},
		{"2.5.3", "2.5.3", false},

		// Current is newer
		{"1.0.0", "1.0.1", false},
		{"1.0.0", "2.0.0", false},

		// Dev versions
		{"1.0.0", "dev", true},
		{"dev", "1.0.0", false},
		{"dev", "", true},
		{"", "1.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.latest+"_vs_"+tt.current, func(t *testing.T) {
			result := isNewerVersion(tt.latest, tt.current)
			if result != tt.expected {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, result, tt.expected)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig("1.0.0")

	if cfg.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", cfg.CurrentVersion, "1.0.0")
	}
	if cfg.CheckInterval != DefaultCheckInterval {
		t.Errorf("CheckInterval = %v, want %v", cfg.CheckInterval, DefaultCheckInterval)
	}
	if cfg.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("HTTPTimeout = %v, want %v", cfg.HTTPTimeout, DefaultHTTPTimeout)
	}
	if !cfg.Enabled {
		t.Error("Enabled should be true by default")
	}
	if cfg.AirGapMode {
		t.Error("AirGapMode should be false by default")
	}
}

func TestNewChecker(t *testing.T) {
	logger := zerolog.Nop()
	cfg := DefaultConfig("1.0.0")

	checker := NewChecker(cfg, logger)

	if checker == nil {
		t.Fatal("NewChecker returned nil")
	}
	if checker.config.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", checker.config.CurrentVersion, "1.0.0")
	}
	if checker.config.GitHubOwner != GitHubOwner {
		t.Errorf("GitHubOwner = %q, want %q", checker.config.GitHubOwner, GitHubOwner)
	}
	if checker.config.GitHubRepo != GitHubRepo {
		t.Errorf("GitHubRepo = %q, want %q", checker.config.GitHubRepo, GitHubRepo)
	}
}

func TestChecker_CheckDisabled(t *testing.T) {
	logger := zerolog.Nop()
	cfg := DefaultConfig("1.0.0")
	cfg.Enabled = false

	checker := NewChecker(cfg, logger)

	_, err := checker.Check(context.Background())
	if err != ErrCheckDisabled {
		t.Errorf("Check() error = %v, want %v", err, ErrCheckDisabled)
	}
}

func TestChecker_AirGapMode(t *testing.T) {
	logger := zerolog.Nop()
	cfg := DefaultConfig("1.0.0")
	cfg.AirGapMode = true

	checker := NewChecker(cfg, logger)

	_, err := checker.Check(context.Background())
	if err != ErrAirGapMode {
		t.Errorf("Check() error = %v, want %v", err, ErrAirGapMode)
	}
}

func TestChecker_IsEnabled(t *testing.T) {
	logger := zerolog.Nop()

	tests := []struct {
		name       string
		enabled    bool
		airGapMode bool
		expected   bool
	}{
		{"enabled_no_airgap", true, false, true},
		{"enabled_with_airgap", true, true, false},
		{"disabled_no_airgap", false, false, false},
		{"disabled_with_airgap", false, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig("1.0.0")
			cfg.Enabled = tt.enabled
			cfg.AirGapMode = tt.airGapMode

			checker := NewChecker(cfg, logger)

			if got := checker.IsEnabled(); got != tt.expected {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestChecker_SetEnabled(t *testing.T) {
	logger := zerolog.Nop()
	cfg := DefaultConfig("1.0.0")
	checker := NewChecker(cfg, logger)

	checker.SetEnabled(false)
	if checker.IsEnabled() {
		t.Error("Expected checker to be disabled")
	}

	checker.SetEnabled(true)
	if !checker.IsEnabled() {
		t.Error("Expected checker to be enabled")
	}
}

func TestChecker_SetAirGapMode(t *testing.T) {
	logger := zerolog.Nop()
	cfg := DefaultConfig("1.0.0")
	checker := NewChecker(cfg, logger)

	checker.SetAirGapMode(true)
	if checker.IsEnabled() {
		t.Error("Expected checker to be disabled in air-gap mode")
	}

	checker.SetAirGapMode(false)
	if !checker.IsEnabled() {
		t.Error("Expected checker to be enabled after disabling air-gap mode")
	}
}

func TestChecker_Check_WithMockServer(t *testing.T) {
	release := Release{
		TagName:     "v1.1.0",
		Name:        "Version 1.1.0",
		Body:        "Release notes here",
		HTMLURL:     "https://github.com/MacJediWizard/keldris/releases/tag/v1.1.0",
		PublishedAt: "2024-01-15T10:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	cfg := DefaultConfig("1.0.0")
	checker := NewChecker(cfg, logger)

	// Override the HTTP client to use the test server
	checker.httpClient = server.Client()

	// We can't easily override the URL, so we'll test the methods that don't require network calls
	// For full integration tests, we'd need to make the URL configurable
}

func TestChecker_Caching(t *testing.T) {
	callCount := 0
	release := Release{
		TagName:     "v1.1.0",
		Name:        "Version 1.1.0",
		Body:        "Release notes",
		HTMLURL:     "https://example.com/release",
		PublishedAt: "2024-01-15T10:00:00Z",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()

	logger := zerolog.Nop()
	cfg := Config{
		CurrentVersion: "1.0.0",
		CheckInterval:  1 * time.Hour,
		HTTPTimeout:    5 * time.Second,
		Enabled:        true,
		AirGapMode:     false,
		GitHubOwner:    "test",
		GitHubRepo:     "test",
	}
	checker := NewChecker(cfg, logger)

	// Manually set cached info to simulate a previous check
	checker.mu.Lock()
	checker.cachedInfo = &UpdateInfo{
		UpdateAvailable: true,
		CurrentVersion:  "1.0.0",
		LatestVersion:   "v1.1.0",
		CheckedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	checker.lastCheckAt = time.Now()
	checker.mu.Unlock()

	// Check should return cached result
	info, err := checker.Check(context.Background())
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !info.UpdateAvailable {
		t.Error("Expected UpdateAvailable to be true from cache")
	}

	// Verify cached info is returned
	cached := checker.GetCachedInfo()
	if cached == nil {
		t.Fatal("GetCachedInfo() returned nil")
	}
	if cached.LatestVersion != "v1.1.0" {
		t.Errorf("Cached LatestVersion = %q, want %q", cached.LatestVersion, "v1.1.0")
	}
}

func TestChecker_GetCachedInfo_Empty(t *testing.T) {
	logger := zerolog.Nop()
	cfg := DefaultConfig("1.0.0")
	checker := NewChecker(cfg, logger)

	cached := checker.GetCachedInfo()
	if cached != nil {
		t.Errorf("GetCachedInfo() = %v, want nil", cached)
	}
}

func TestUpdateInfo_Fields(t *testing.T) {
	info := UpdateInfo{
		UpdateAvailable:        true,
		CurrentVersion:         "1.0.0",
		LatestVersion:          "v1.1.0",
		ReleaseNotes:           "New features",
		ReleaseURL:             "https://example.com/release",
		ChangelogURL:           "https://example.com/changelog",
		UpgradeInstructionsURL: "https://example.com/upgrade",
		PublishedAt:            "2024-01-15T10:00:00Z",
		CheckedAt:              "2024-01-16T10:00:00Z",
		NextCheckAt:            "2024-01-17T10:00:00Z",
	}

	// Verify JSON marshaling
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var unmarshaled UpdateInfo
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if unmarshaled.UpdateAvailable != info.UpdateAvailable {
		t.Errorf("UpdateAvailable = %v, want %v", unmarshaled.UpdateAvailable, info.UpdateAvailable)
	}
	if unmarshaled.LatestVersion != info.LatestVersion {
		t.Errorf("LatestVersion = %q, want %q", unmarshaled.LatestVersion, info.LatestVersion)
	}
}
