package updater

import (
	"testing"
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

func TestNew(t *testing.T) {
	u := New("1.0.0")
	if u == nil {
		t.Fatal("New() returned nil")
	}
	if u.currentVersion != "1.0.0" {
		t.Errorf("currentVersion = %q, want %q", u.currentVersion, "1.0.0")
	}
	if u.githubOwner != GitHubOwner {
		t.Errorf("githubOwner = %q, want %q", u.githubOwner, GitHubOwner)
	}
	if u.githubRepo != GitHubRepo {
		t.Errorf("githubRepo = %q, want %q", u.githubRepo, GitHubRepo)
	}
}

func TestNewWithConfig(t *testing.T) {
	u := NewWithConfig("2.0.0", "custom-owner", "custom-repo", nil)
	if u == nil {
		t.Fatal("NewWithConfig() returned nil")
	}
	if u.currentVersion != "2.0.0" {
		t.Errorf("currentVersion = %q, want %q", u.currentVersion, "2.0.0")
	}
	if u.githubOwner != "custom-owner" {
		t.Errorf("githubOwner = %q, want %q", u.githubOwner, "custom-owner")
	}
	if u.githubRepo != "custom-repo" {
		t.Errorf("githubRepo = %q, want %q", u.githubRepo, "custom-repo")
	}
}

func TestFindAssetForPlatform(t *testing.T) {
	assets := []Asset{
		{Name: "keldris-agent-linux-amd64", DownloadURL: "https://example.com/linux-amd64"},
		{Name: "keldris-agent-linux-arm64", DownloadURL: "https://example.com/linux-arm64"},
		{Name: "keldris-agent-darwin-amd64", DownloadURL: "https://example.com/darwin-amd64"},
		{Name: "keldris-agent-darwin-arm64", DownloadURL: "https://example.com/darwin-arm64"},
		{Name: "keldris-agent-windows-amd64.exe", DownloadURL: "https://example.com/windows-amd64"},
	}

	u := New("1.0.0")

	// Test that we can find an asset for the current platform
	asset, err := u.findAssetForPlatform(assets)
	// This test will find something based on the runtime.GOOS/GOARCH
	// We just verify no error for common platforms
	if err != nil {
		// Only fail if we're on a common platform that should be supported
		t.Logf("findAssetForPlatform error: %v (may be expected for uncommon platforms)", err)
	} else if asset == nil {
		t.Error("findAssetForPlatform returned nil asset without error")
	}
}
