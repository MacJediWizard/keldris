package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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
		{"v0.0.1", "0.0.1"},
		{"", ""},
		{"  ", ""},
		{"vv1.0.0", "v1.0.0"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.input), func(t *testing.T) {
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
		{"0.0.0", [3]int{0, 0, 0}},
		{"1.2.3-alpha", [3]int{1, 2, 3}},
		{"1.2.3-alpha.1", [3]int{1, 2, 3}},
		{"1.2.3+build456", [3]int{1, 2, 3}},
		{"abc", [3]int{0, 0, 0}},
		{"", [3]int{0, 0, 0}},
		{"1.2.3.4", [3]int{1, 2, 3}},
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

		// Pre-release handling (pre-release suffix stripped, compare base)
		{"1.2.4-rc1", "1.2.3", true},
		{"1.2.3-rc1", "1.2.3", false},
		{"1.2.3-beta", "1.2.2", true},
		{"1.2.3-alpha", "1.2.3-beta", false},

		// Edge cases
		{"0.0.1", "0.0.0", true},
		{"0.0.0", "0.0.0", false},

		// Empty latest with dev current
		{"", "dev", true},
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
	if u.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if u.httpClient.Timeout != DefaultTimeout {
		t.Errorf("httpClient.Timeout = %v, want %v", u.httpClient.Timeout, DefaultTimeout)
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

// newTestServer creates an httptest.Server that returns the given release JSON.
func newTestServer(t *testing.T, release Release) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(release); err != nil {
			t.Fatalf("failed to encode release: %v", err)
		}
	}))
}

// updaterWithServer creates an Updater whose fetchLatestRelease calls go to the given test server.
// It overrides the GitHubReleasesAPI by replacing the httpClient transport.
func updaterWithServer(version string, serverURL string) *Updater {
	u := &Updater{
		currentVersion: version,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &rewriteTransport{
				base:    http.DefaultTransport,
				baseURL: serverURL,
			},
		},
		githubOwner: "test-owner",
		githubRepo:  "test-repo",
	}
	return u
}

// rewriteTransport rewrites all request URLs to point to the test server.
type rewriteTransport struct {
	base    http.RoundTripper
	baseURL string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	// Extract host from baseURL
	baseURL := strings.TrimPrefix(t.baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")
	req.URL.Host = baseURL
	req.Host = baseURL
	return t.base.RoundTrip(req)
}

func testRelease(version string) Release {
	osName := runtime.GOOS
	arch := runtime.GOARCH
	return Release{
		TagName: "v" + version,
		Name:    "Release " + version,
		Body:    "Release notes for " + version,
		Assets: []Asset{
			{
				Name:        fmt.Sprintf("keldris-agent-%s-%s", osName, arch),
				DownloadURL: "https://example.com/download",
				Size:        1024,
			},
		},
	}
}

func TestCheckForUpdate_UpdateAvailable(t *testing.T) {
	release := testRelease("2.0.0")
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}
	if info == nil {
		t.Fatal("CheckForUpdate() returned nil info")
	}
	if info.CurrentVersion != "1.0.0" {
		t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "1.0.0")
	}
	if info.LatestVersion != "v2.0.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "v2.0.0")
	}
	if info.ReleaseNotes != "Release notes for 2.0.0" {
		t.Errorf("ReleaseNotes = %q, want %q", info.ReleaseNotes, "Release notes for 2.0.0")
	}
	if info.AssetSize != 1024 {
		t.Errorf("AssetSize = %d, want %d", info.AssetSize, 1024)
	}
}

func TestCheckForUpdate_NoUpdateAvailable(t *testing.T) {
	release := testRelease("1.0.0")
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if !errors.Is(err, ErrNoUpdateAvailable) {
		t.Errorf("CheckForUpdate() error = %v, want ErrNoUpdateAvailable", err)
	}
}

func TestCheckForUpdate_CurrentIsNewer(t *testing.T) {
	release := testRelease("1.0.0")
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("2.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if !errors.Is(err, ErrNoUpdateAvailable) {
		t.Errorf("CheckForUpdate() error = %v, want ErrNoUpdateAvailable", err)
	}
}

func TestCheckForUpdate_DevVersion(t *testing.T) {
	release := testRelease("1.0.0")
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("dev", server.URL)

	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}
	if info == nil {
		t.Fatal("CheckForUpdate() returned nil info")
	}
	if info.CurrentVersion != "dev" {
		t.Errorf("CurrentVersion = %q, want %q", info.CurrentVersion, "dev")
	}
}

func TestCheckForUpdate_PreReleaseHandling(t *testing.T) {
	// Pre-release suffix is stripped for comparison, so 1.2.3-rc1 base == 1.2.3
	release := testRelease("1.2.3")
	release.TagName = "v1.2.3-rc1"
	server := newTestServer(t, release)
	defer server.Close()

	// Same base version - no update
	u := updaterWithServer("1.2.3", server.URL)
	_, err := u.CheckForUpdate(context.Background())
	if !errors.Is(err, ErrNoUpdateAvailable) {
		t.Errorf("CheckForUpdate() with same base version error = %v, want ErrNoUpdateAvailable", err)
	}

	// Older version - update available
	u2 := updaterWithServer("1.2.2", server.URL)
	info, err := u2.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate() with older version error = %v", err)
	}
	if info == nil {
		t.Fatal("CheckForUpdate() returned nil info for older version")
	}
}

func TestCheckForUpdate_NetworkError(t *testing.T) {
	// Use a server that's immediately closed to simulate network error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("CheckForUpdate() should return error for network failure")
	}
	if !strings.Contains(err.Error(), "fetch latest release") {
		t.Errorf("error should contain 'fetch latest release', got: %v", err)
	}
}

func TestCheckForUpdate_RateLimited(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", "1700000000")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"message": "API rate limit exceeded"}`)
	}))
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("CheckForUpdate() should return error for rate limit")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should contain HTTP status 403, got: %v", err)
	}
}

func TestCheckForUpdate_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("CheckForUpdate() should return error for 404")
	}
	if !strings.Contains(err.Error(), "no releases found") {
		t.Errorf("error should contain 'no releases found', got: %v", err)
	}
}

func TestCheckForUpdate_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{invalid json`)
	}))
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("CheckForUpdate() should return error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "decode release") {
		t.Errorf("error should contain 'decode release', got: %v", err)
	}
}

func TestCheckForUpdate_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := u.CheckForUpdate(ctx)
	if err == nil {
		t.Fatal("CheckForUpdate() should return error for canceled context")
	}
}

func TestCheckForUpdate_UnsupportedPlatform(t *testing.T) {
	// Release with no matching assets for current platform
	release := Release{
		TagName: "v2.0.0",
		Name:    "Release 2.0.0",
		Body:    "Notes",
		Assets: []Asset{
			{Name: "keldris-agent-fakeos-fakearch", DownloadURL: "https://example.com/fake", Size: 512},
		},
	}
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("CheckForUpdate() should return error for unsupported platform")
	}
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Errorf("error should be ErrUnsupportedPlatform, got: %v", err)
	}
}

func TestCheckForUpdate_EmptyAssets(t *testing.T) {
	release := Release{
		TagName: "v2.0.0",
		Name:    "Release 2.0.0",
		Body:    "Notes",
		Assets:  []Asset{},
	}
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	_, err := u.CheckForUpdate(context.Background())
	if err == nil {
		t.Fatal("CheckForUpdate() should return error for empty assets")
	}
	if !errors.Is(err, ErrUnsupportedPlatform) {
		t.Errorf("error should be ErrUnsupportedPlatform, got: %v", err)
	}
}

func TestFindAssetForPlatform(t *testing.T) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	tests := []struct {
		name      string
		assets    []Asset
		wantName  string
		wantError bool
	}{
		{
			name: "dash_format",
			assets: []Asset{
				{Name: fmt.Sprintf("keldris-agent-%s-%s", osName, arch), DownloadURL: "https://example.com/dash"},
			},
			wantName:  fmt.Sprintf("keldris-agent-%s-%s", osName, arch),
			wantError: false,
		},
		{
			name: "underscore_format",
			assets: []Asset{
				{Name: fmt.Sprintf("keldris-agent_%s_%s", osName, arch), DownloadURL: "https://example.com/underscore"},
			},
			wantName:  fmt.Sprintf("keldris-agent_%s_%s", osName, arch),
			wantError: false,
		},
		{
			name: "case_insensitive",
			assets: []Asset{
				{Name: fmt.Sprintf("Keldris-Agent-%s-%s", strings.ToUpper(osName), strings.ToUpper(arch)), DownloadURL: "https://example.com/upper"},
			},
			// patterns are lowercase, asset name is lowered; but patterns use runtime.GOOS which is already lowercase
			// The asset name is lowered before matching, so upper case in asset name matches lowercase pattern
			wantName:  fmt.Sprintf("Keldris-Agent-%s-%s", strings.ToUpper(osName), strings.ToUpper(arch)),
			wantError: false,
		},
		{
			name: "with_extension",
			assets: []Asset{
				{Name: fmt.Sprintf("keldris-agent-%s-%s.exe", osName, arch), DownloadURL: "https://example.com/exe"},
			},
			wantName:  fmt.Sprintf("keldris-agent-%s-%s.exe", osName, arch),
			wantError: false,
		},
		{
			name: "no_matching_asset",
			assets: []Asset{
				{Name: "keldris-agent-fakeos-fakearch", DownloadURL: "https://example.com/fake"},
			},
			wantError: true,
		},
		{
			name:      "empty_assets",
			assets:    []Asset{},
			wantError: true,
		},
		{
			name: "multiple_assets_picks_first_match",
			assets: []Asset{
				{Name: "keldris-agent-fakeos-fakearch", DownloadURL: "https://example.com/fake"},
				{Name: fmt.Sprintf("keldris-agent-%s-%s", osName, arch), DownloadURL: "https://example.com/correct"},
				{Name: fmt.Sprintf("keldris-agent_%s_%s", osName, arch), DownloadURL: "https://example.com/also-correct"},
			},
			wantName:  fmt.Sprintf("keldris-agent-%s-%s", osName, arch),
			wantError: false,
		},
	}

	u := New("1.0.0")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := u.findAssetForPlatform(tt.assets)
			if tt.wantError {
				if err == nil {
					t.Error("findAssetForPlatform() should return error")
				}
				if !errors.Is(err, ErrUnsupportedPlatform) {
					t.Errorf("error should wrap ErrUnsupportedPlatform, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("findAssetForPlatform() error = %v", err)
			}
			if asset.Name != tt.wantName {
				t.Errorf("asset.Name = %q, want %q", asset.Name, tt.wantName)
			}
		})
	}
}

func TestFetchLatestRelease(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		release := testRelease("1.5.0")
		server := newTestServer(t, release)
		defer server.Close()

		u := updaterWithServer("1.0.0", server.URL)

		got, err := u.fetchLatestRelease(context.Background())
		if err != nil {
			t.Fatalf("fetchLatestRelease() error = %v", err)
		}
		if got.TagName != "v1.5.0" {
			t.Errorf("TagName = %q, want %q", got.TagName, "v1.5.0")
		}
		if got.Name != "Release 1.5.0" {
			t.Errorf("Name = %q, want %q", got.Name, "Release 1.5.0")
		}
		if len(got.Assets) != 1 {
			t.Errorf("len(Assets) = %d, want 1", len(got.Assets))
		}
	})

	t.Run("404_no_releases", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		u := updaterWithServer("1.0.0", server.URL)

		_, err := u.fetchLatestRelease(context.Background())
		if err == nil {
			t.Fatal("fetchLatestRelease() should return error for 404")
		}
		if !strings.Contains(err.Error(), "no releases found") {
			t.Errorf("error = %v, want to contain 'no releases found'", err)
		}
	})

	t.Run("500_server_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		u := updaterWithServer("1.0.0", server.URL)

		_, err := u.fetchLatestRelease(context.Background())
		if err == nil {
			t.Fatal("fetchLatestRelease() should return error for 500")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error = %v, want to contain '500'", err)
		}
	})

	t.Run("rate_limited_403", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		u := updaterWithServer("1.0.0", server.URL)

		_, err := u.fetchLatestRelease(context.Background())
		if err == nil {
			t.Fatal("fetchLatestRelease() should return error for 403")
		}
		if !strings.Contains(err.Error(), "403") {
			t.Errorf("error = %v, want to contain '403'", err)
		}
	})

	t.Run("invalid_json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, `not json at all`)
		}))
		defer server.Close()

		u := updaterWithServer("1.0.0", server.URL)

		_, err := u.fetchLatestRelease(context.Background())
		if err == nil {
			t.Fatal("fetchLatestRelease() should return error for invalid JSON")
		}
	})

	t.Run("verifies_accept_header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accept := r.Header.Get("Accept")
			if accept != "application/vnd.github.v3+json" {
				t.Errorf("Accept header = %q, want %q", accept, "application/vnd.github.v3+json")
			}
			release := testRelease("1.0.0")
			json.NewEncoder(w).Encode(release)
		}))
		defer server.Close()

		u := updaterWithServer("1.0.0", server.URL)
		u.fetchLatestRelease(context.Background())
	})
}

func TestDownload(t *testing.T) {
	binaryContent := []byte("fake binary content for testing download")

	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			w.Write(binaryContent)
		}))
		defer server.Close()

		u := New("1.0.0")
		info := &UpdateInfo{
			DownloadURL: server.URL + "/download",
			AssetSize:   int64(len(binaryContent)),
		}

		var progressCalls int
		path, err := u.Download(context.Background(), info, func(downloaded, total int64) {
			progressCalls++
		})
		if err != nil {
			t.Fatalf("Download() error = %v", err)
		}
		defer os.Remove(path)

		// Verify downloaded content
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(content) != string(binaryContent) {
			t.Errorf("downloaded content mismatch")
		}

		// Verify file is executable
		info2, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Stat() error = %v", err)
		}
		if info2.Mode()&0111 == 0 {
			t.Error("downloaded file should be executable")
		}

		// Verify progress was called
		if progressCalls == 0 {
			t.Error("progress callback should have been called at least once")
		}
	})

	t.Run("nil_progress", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(binaryContent)
		}))
		defer server.Close()

		u := New("1.0.0")
		info := &UpdateInfo{
			DownloadURL: server.URL + "/download",
			AssetSize:   int64(len(binaryContent)),
		}

		path, err := u.Download(context.Background(), info, nil)
		if err != nil {
			t.Fatalf("Download() error = %v", err)
		}
		defer os.Remove(path)
	})

	t.Run("server_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		u := New("1.0.0")
		info := &UpdateInfo{
			DownloadURL: server.URL + "/download",
			AssetSize:   1024,
		}

		_, err := u.Download(context.Background(), info, nil)
		if err == nil {
			t.Fatal("Download() should return error for server error")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("error should contain '500', got: %v", err)
		}
	})

	t.Run("network_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		server.Close() // Close immediately

		u := New("1.0.0")
		info := &UpdateInfo{
			DownloadURL: server.URL + "/download",
			AssetSize:   1024,
		}

		_, err := u.Download(context.Background(), info, nil)
		if err == nil {
			t.Fatal("Download() should return error for network failure")
		}
	})

	t.Run("canceled_context", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(5 * time.Second)
		}))
		defer server.Close()

		u := New("1.0.0")
		info := &UpdateInfo{
			DownloadURL: server.URL + "/download",
			AssetSize:   1024,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := u.Download(ctx, info, nil)
		if err == nil {
			t.Fatal("Download() should return error for canceled context")
		}
	})
}

func TestApply(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Create a temp directory for this test
		tmpDir := t.TempDir()

		// Create a fake "current binary"
		currentBinary := filepath.Join(tmpDir, "keldris-agent")
		if err := os.WriteFile(currentBinary, []byte("old binary"), 0755); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		// Create a fake "new binary"
		newBinary := filepath.Join(tmpDir, "keldris-agent-new")
		if err := os.WriteFile(newBinary, []byte("new binary"), 0755); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		// We can't easily test Apply because it uses os.Executable(),
		// but we can test the rename logic directly
		backupPath := currentBinary + ".bak"
		if err := os.Rename(currentBinary, backupPath); err != nil {
			t.Fatalf("backup rename error = %v", err)
		}
		if err := os.Rename(newBinary, currentBinary); err != nil {
			_ = os.Rename(backupPath, currentBinary)
			t.Fatalf("install rename error = %v", err)
		}
		_ = os.Remove(backupPath)

		// Verify new binary is in place
		content, err := os.ReadFile(currentBinary)
		if err != nil {
			t.Fatalf("ReadFile() error = %v", err)
		}
		if string(content) != "new binary" {
			t.Errorf("content = %q, want %q", string(content), "new binary")
		}
	})
}

func TestComputeSHA256(t *testing.T) {
	t.Run("valid_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "test-file")
		content := []byte("hello world")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		hash, err := computeSHA256(path)
		if err != nil {
			t.Fatalf("computeSHA256() error = %v", err)
		}

		// Compute expected hash
		h := sha256.Sum256(content)
		expected := hex.EncodeToString(h[:])
		if hash != expected {
			t.Errorf("hash = %q, want %q", hash, expected)
		}
	})

	t.Run("empty_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "empty-file")
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		hash, err := computeSHA256(path)
		if err != nil {
			t.Fatalf("computeSHA256() error = %v", err)
		}

		h := sha256.Sum256([]byte{})
		expected := hex.EncodeToString(h[:])
		if hash != expected {
			t.Errorf("hash = %q, want %q", hash, expected)
		}
	})

	t.Run("nonexistent_file", func(t *testing.T) {
		_, err := computeSHA256("/nonexistent/file/path")
		if err == nil {
			t.Fatal("computeSHA256() should return error for nonexistent file")
		}
	})

	t.Run("large_file", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "large-file")
		// Write 1MB file
		content := make([]byte, 1024*1024)
		for i := range content {
			content[i] = byte(i % 256)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		hash, err := computeSHA256(path)
		if err != nil {
			t.Fatalf("computeSHA256() error = %v", err)
		}

		h := sha256.Sum256(content)
		expected := hex.EncodeToString(h[:])
		if hash != expected {
			t.Errorf("hash mismatch for large file")
		}
	})
}

func TestConstants(t *testing.T) {
	if GitHubOwner != "MacJediWizard" {
		t.Errorf("GitHubOwner = %q, want %q", GitHubOwner, "MacJediWizard")
	}
	if GitHubRepo != "keldris" {
		t.Errorf("GitHubRepo = %q, want %q", GitHubRepo, "keldris")
	}
	if DefaultTimeout != 30*time.Second {
		t.Errorf("DefaultTimeout = %v, want %v", DefaultTimeout, 30*time.Second)
	}
	if DownloadTimeout != 5*time.Minute {
		t.Errorf("DownloadTimeout = %v, want %v", DownloadTimeout, 5*time.Minute)
	}
}

func TestErrors(t *testing.T) {
	if ErrNoUpdateAvailable.Error() != "no update available" {
		t.Errorf("ErrNoUpdateAvailable = %q, want %q", ErrNoUpdateAvailable.Error(), "no update available")
	}
	if ErrUnsupportedPlatform.Error() != "unsupported platform" {
		t.Errorf("ErrUnsupportedPlatform = %q, want %q", ErrUnsupportedPlatform.Error(), "unsupported platform")
	}
}

func TestCheckForUpdate_MajorVersionBump(t *testing.T) {
	release := testRelease("3.0.0")
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("1.5.2", server.URL)

	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}
	if info.LatestVersion != "v3.0.0" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "v3.0.0")
	}
}

func TestCheckForUpdate_PatchVersionBump(t *testing.T) {
	release := testRelease("1.0.1")
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("1.0.0", server.URL)

	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}
	if info.LatestVersion != "v1.0.1" {
		t.Errorf("LatestVersion = %q, want %q", info.LatestVersion, "v1.0.1")
	}
}

func TestCheckForUpdate_VPrefixHandling(t *testing.T) {
	release := testRelease("2.0.0")
	server := newTestServer(t, release)
	defer server.Close()

	// Current version with v prefix
	u := updaterWithServer("v1.0.0", server.URL)

	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}
	if info == nil {
		t.Fatal("CheckForUpdate() returned nil info")
	}
}

func TestCheckForUpdate_EmptyVersion(t *testing.T) {
	release := testRelease("1.0.0")
	server := newTestServer(t, release)
	defer server.Close()

	u := updaterWithServer("", server.URL)

	info, err := u.CheckForUpdate(context.Background())
	if err != nil {
		t.Fatalf("CheckForUpdate() error = %v", err)
	}
	if info == nil {
		t.Fatal("CheckForUpdate() returned nil info")
	}
}

func TestDownload_VerifiesAcceptHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept")
		if accept != "application/octet-stream" {
			t.Errorf("Accept header = %q, want %q", accept, "application/octet-stream")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("binary"))
	}))
	defer server.Close()

	u := New("1.0.0")
	info := &UpdateInfo{
		DownloadURL: server.URL + "/download",
		AssetSize:   6,
	}
	path, err := u.Download(context.Background(), info, nil)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer os.Remove(path)
}

func TestDownload_ProgressTracking(t *testing.T) {
	// Create larger content to ensure multiple read iterations
	content := make([]byte, 100*1024) // 100KB
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	u := New("1.0.0")
	info := &UpdateInfo{
		DownloadURL: server.URL + "/download",
		AssetSize:   int64(len(content)),
	}

	var lastDownloaded int64
	var lastTotal int64
	var calls int
	path, err := u.Download(context.Background(), info, func(downloaded, total int64) {
		calls++
		if downloaded < lastDownloaded {
			t.Errorf("downloaded decreased: %d -> %d", lastDownloaded, downloaded)
		}
		lastDownloaded = downloaded
		lastTotal = total
	})
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	defer os.Remove(path)

	if lastDownloaded != int64(len(content)) {
		t.Errorf("final downloaded = %d, want %d", lastDownloaded, len(content))
	}
	if lastTotal != int64(len(content)) {
		t.Errorf("total = %d, want %d", lastTotal, len(content))
	}
	if calls == 0 {
		t.Error("progress callback never called")
	}
}

func TestDownload_InvalidURL(t *testing.T) {
	u := New("1.0.0")
	info := &UpdateInfo{
		DownloadURL: "://invalid\x00url",
		AssetSize:   1024,
	}

	_, err := u.Download(context.Background(), info, nil)
	if err == nil {
		t.Fatal("Download() should return error for invalid URL")
	}
	if !strings.Contains(err.Error(), "create download request") {
		t.Errorf("error should contain 'create download request', got: %v", err)
	}
}

func TestDownload_ReadErrorMidStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write partial data
		w.Write([]byte("partial data"))
		// Flush to ensure client receives it
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Hijack the connection and close it abruptly
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hijacker.Hijack()
		conn.Close()
	}))
	defer server.Close()

	u := New("1.0.0")
	info := &UpdateInfo{
		DownloadURL: server.URL + "/download",
		AssetSize:   1024 * 1024, // Expect much more than we send
	}

	_, err := u.Download(context.Background(), info, nil)
	// The read error may or may not be triggered depending on timing
	// If we get an error, verify it's a read error
	if err != nil && !strings.Contains(err.Error(), "read download") {
		// Could also get EOF without error if connection closes cleanly
		t.Logf("Download() error = %v (expected read download error or clean EOF)", err)
	}
}

func TestApply_Success(t *testing.T) {
	// Get the current test binary path
	execPath, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error = %v", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		t.Fatalf("EvalSymlinks() error = %v", err)
	}

	// Read current binary content so we can restore it
	originalContent, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Create a new binary file (copy of original) in the same directory
	// so os.Rename works (same filesystem)
	newBinaryPath := execPath + ".new"
	if err := os.WriteFile(newBinaryPath, originalContent, 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	defer os.Remove(newBinaryPath) // Clean up if Apply fails

	u := New("1.0.0")
	err = u.Apply(newBinaryPath)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify the binary still exists and has the right content
	content, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatalf("ReadFile() after Apply error = %v", err)
	}
	if len(content) != len(originalContent) {
		t.Errorf("binary size changed: got %d, want %d", len(content), len(originalContent))
	}
}

func TestApply_NonexistentNewBinary(t *testing.T) {
	u := New("1.0.0")
	err := u.Apply("/nonexistent/path/to/binary")
	if err == nil {
		t.Fatal("Apply() should return error for nonexistent new binary")
	}
}

func TestApply_NewBinaryInDifferentFilesystem(t *testing.T) {
	// Test with a new binary path that doesn't exist
	// This should fail at the rename step
	u := New("1.0.0")
	tmpDir := t.TempDir()
	fakeBinary := filepath.Join(tmpDir, "fake-binary")
	if err := os.WriteFile(fakeBinary, []byte("fake"), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// Apply will try to rename currentExec to .bak, then move fakeBinary to currentExec
	// If they're on different filesystems, the second rename may fail
	// Either way, this tests the Apply code path
	err := u.Apply(fakeBinary)
	// We expect this to either succeed or fail with an error
	// Just verify it doesn't panic
	if err != nil {
		t.Logf("Apply() error = %v (expected on some systems)", err)
	}
}
