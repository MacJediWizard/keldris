// Package updater provides self-update functionality for the Keldris agent.
package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
	"github.com/MacJediWizard/keldris/internal/httpclient"
)

const (
	// GitHubOwner is the GitHub repository owner.
	GitHubOwner = "MacJediWizard"
	// GitHubRepo is the GitHub repository name.
	GitHubRepo = "keldris"
	// GitHubReleasesAPI is the GitHub releases API endpoint.
	GitHubReleasesAPI = "https://api.github.com/repos/%s/%s/releases/latest"

	// DefaultTimeout is the default HTTP timeout for update operations.
	DefaultTimeout = 30 * time.Second
	// DownloadTimeout is the timeout for downloading binaries.
	DownloadTimeout = 5 * time.Minute
)

// ErrNoUpdateAvailable is returned when the current version is up to date.
var ErrNoUpdateAvailable = errors.New("no update available")

// ErrUnsupportedPlatform is returned when the platform is not supported.
var ErrUnsupportedPlatform = errors.New("unsupported platform")

// ErrAirGapMode is returned when update checks are blocked by air-gap mode.
var ErrAirGapMode = errors.New("updates disabled in air-gap mode")

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Body    string  `json:"body"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset (binary file).
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// UpdateInfo contains information about an available update.
type UpdateInfo struct {
	CurrentVersion string
	LatestVersion  string
	ReleaseNotes   string
	DownloadURL    string
	AssetName      string
	AssetSize      int64
}

// Updater handles checking for and applying updates.
type Updater struct {
	currentVersion string
	httpClient     *http.Client
	githubOwner    string
	githubRepo     string
	airGapMode     bool
	proxyConfig    *config.ProxyConfig
}

// New creates a new Updater with the given current version.
func New(currentVersion string) *Updater {
	return NewWithProxy(currentVersion, nil)
}

// NewWithProxy creates an Updater with proxy configuration.
func NewWithProxy(currentVersion string, proxyConfig *config.ProxyConfig) *Updater {
	client, err := httpclient.New(httpclient.Options{
		Timeout:     DefaultTimeout,
		ProxyConfig: proxyConfig,
	})
	if err != nil {
		// Fall back to simple client if proxy config fails
		client = httpclient.NewSimple(DefaultTimeout)
	}

	return &Updater{
		currentVersion: currentVersion,
		httpClient:     client,
		githubOwner:    GitHubOwner,
		githubRepo:     GitHubRepo,
		proxyConfig:    proxyConfig,
	}
}

// NewWithConfig creates an Updater with custom GitHub configuration.
func NewWithConfig(currentVersion, owner, repo string, proxyConfig *config.ProxyConfig) *Updater {
	client, err := httpclient.New(httpclient.Options{
		Timeout:     DefaultTimeout,
		ProxyConfig: proxyConfig,
	})
	if err != nil {
		// Fall back to simple client if proxy config fails
		client = httpclient.NewSimple(DefaultTimeout)
	}

	return &Updater{
		currentVersion: currentVersion,
		httpClient:     client,
		githubOwner:    owner,
		githubRepo:     repo,
		proxyConfig:    proxyConfig,
	}
}

// SetAirGapMode enables or disables air-gap mode on the updater.
func (u *Updater) SetAirGapMode(enabled bool) {
	u.airGapMode = enabled
}

// CheckForUpdate checks if a new version is available.
func (u *Updater) CheckForUpdate(ctx context.Context) (*UpdateInfo, error) {
	if u.airGapMode {
		return nil, ErrAirGapMode
	}

	release, err := u.fetchLatestRelease(ctx)
	if err != nil {
		return nil, err
	}

	latestVersion := normalizeVersion(release.TagName)
	currentVersion := normalizeVersion(u.currentVersion)

	if !isNewerVersion(latestVersion, currentVersion) {
		return nil, ErrNoUpdateAvailable
	}

	asset, err := u.findAssetForPlatform(release.Assets)
	if err != nil {
		return nil, err
	}

	return &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  release.TagName,
		ReleaseNotes:   release.Body,
		DownloadURL:    asset.DownloadURL,
		AssetName:      asset.Name,
		AssetSize:      asset.Size,
	}, nil
}

// Download downloads the update binary to a temporary file.
func (u *Updater) Download(ctx context.Context, info *UpdateInfo, progress func(downloaded, total int64)) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.DownloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}

	// GitHub requires Accept header for binary downloads
	req.Header.Set("Accept", "application/octet-stream")

	client, err := httpclient.New(httpclient.Options{
		Timeout:     DownloadTimeout,
		ProxyConfig: u.proxyConfig,
	})
	if err != nil {
		return "", fmt.Errorf("create download client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Create temporary file
	tmpDir := os.TempDir()
	tmpFile, err := os.CreateTemp(tmpDir, "keldris-agent-update-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Download with progress tracking
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				os.Remove(tmpFile.Name())
				return "", fmt.Errorf("write temp file: %w", writeErr)
			}
			downloaded += int64(n)
			if progress != nil {
				progress(downloaded, info.AssetSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			os.Remove(tmpFile.Name())
			return "", fmt.Errorf("read download: %w", err)
		}
	}

	// Make executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("chmod: %w", err)
	}

	return tmpFile.Name(), nil
}

// Apply replaces the current binary with the new one and restarts.
func (u *Updater) Apply(newBinaryPath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	// Create backup
	backupPath := execPath + ".bak"
	if err := os.Rename(execPath, backupPath); err != nil {
		return fmt.Errorf("create backup: %w", err)
	}

	// Move new binary into place
	if err := os.Rename(newBinaryPath, execPath); err != nil {
		// Restore backup
		_ = os.Rename(backupPath, execPath)
		return fmt.Errorf("install new binary: %w", err)
	}

	// Remove backup
	_ = os.Remove(backupPath)

	return nil
}

// Restart restarts the agent process with the same arguments.
func (u *Updater) Restart() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}

	// Use exec syscall to replace current process
	return syscall.Exec(execPath, os.Args, os.Environ())
}

// fetchLatestRelease fetches the latest release from GitHub.
func (u *Updater) fetchLatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf(GitHubReleasesAPI, u.githubOwner, u.githubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no releases found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: HTTP %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	return &release, nil
}

// findAssetForPlatform finds the download URL for the current OS/arch.
func (u *Updater) findAssetForPlatform(assets []Asset) (*Asset, error) {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Build expected asset name patterns
	patterns := []string{
		fmt.Sprintf("keldris-agent-%s-%s", osName, arch),
		fmt.Sprintf("keldris-agent_%s_%s", osName, arch),
	}

	for _, asset := range assets {
		nameLower := strings.ToLower(asset.Name)
		for _, pattern := range patterns {
			if strings.Contains(nameLower, pattern) {
				return &asset, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: %s/%s", ErrUnsupportedPlatform, osName, arch)
}

// normalizeVersion removes 'v' prefix and returns clean semver string.
func normalizeVersion(version string) string {
	return strings.TrimPrefix(strings.TrimSpace(version), "v")
}

// isNewerVersion compares two semver strings.
// Returns true if latest is newer than current.
func isNewerVersion(latest, current string) bool {
	// Handle dev version
	if current == "dev" || current == "" {
		return true
	}
	if latest == "dev" || latest == "" {
		return false
	}

	latestParts := parseSemver(latest)
	currentParts := parseSemver(current)

	for i := 0; i < 3; i++ {
		if latestParts[i] > currentParts[i] {
			return true
		}
		if latestParts[i] < currentParts[i] {
			return false
		}
	}

	return false
}

// parseSemver parses a semver string into [major, minor, patch].
func parseSemver(version string) [3]int {
	var parts [3]int
	// Remove any pre-release suffix (e.g., -rc1, -beta)
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}

	segments := strings.Split(version, ".")
	for i := 0; i < 3 && i < len(segments); i++ {
		var val int
		fmt.Sscanf(segments[i], "%d", &val)
		parts[i] = val
	}

	return parts
}

// computeSHA256 computes the SHA256 hash of a file.
func computeSHA256(path string) (string, error) {
	return ComputeSHA256(path)
}

// ComputeSHA256 computes the SHA256 hash of a file.
func ComputeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
