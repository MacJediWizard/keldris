// Package updates provides version checking functionality for the Keldris server.
package updates

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	// GitHubOwner is the GitHub repository owner.
	GitHubOwner = "MacJediWizard"
	// GitHubRepo is the GitHub repository name.
	GitHubRepo = "keldris"
	// GitHubReleasesAPI is the GitHub releases API endpoint.
	GitHubReleasesAPI = "https://api.github.com/repos/%s/%s/releases/latest"
	// ChangelogURL is the URL to the changelog.
	ChangelogURL = "https://github.com/MacJediWizard/keldris/blob/main/CHANGELOG.md"
	// UpgradeInstructionsURL is the URL to upgrade instructions.
	UpgradeInstructionsURL = "https://github.com/MacJediWizard/keldris/blob/main/docs/upgrade.md"

	// DefaultCheckInterval is the default interval between version checks (24 hours).
	DefaultCheckInterval = 24 * time.Hour
	// DefaultHTTPTimeout is the default HTTP timeout for API calls.
	DefaultHTTPTimeout = 30 * time.Second
)

// ErrAirGapMode is returned when update checking is disabled due to air-gap mode.
var ErrAirGapMode = errors.New("update checking disabled: air-gap mode enabled")

// ErrCheckDisabled is returned when update checking is disabled.
var ErrCheckDisabled = errors.New("update checking disabled")

// Release represents a GitHub release.
type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

// UpdateInfo contains information about an available update.
type UpdateInfo struct {
	UpdateAvailable        bool   `json:"update_available"`
	CurrentVersion         string `json:"current_version"`
	LatestVersion          string `json:"latest_version,omitempty"`
	ReleaseNotes           string `json:"release_notes,omitempty"`
	ReleaseURL             string `json:"release_url,omitempty"`
	ChangelogURL           string `json:"changelog_url,omitempty"`
	UpgradeInstructionsURL string `json:"upgrade_instructions_url,omitempty"`
	PublishedAt            string `json:"published_at,omitempty"`
	CheckedAt              string `json:"checked_at"`
	NextCheckAt            string `json:"next_check_at,omitempty"`
}

// Config holds configuration for the update checker.
type Config struct {
	// CurrentVersion is the current server version.
	CurrentVersion string
	// CheckInterval is the interval between version checks.
	CheckInterval time.Duration
	// HTTPTimeout is the timeout for HTTP requests.
	HTTPTimeout time.Duration
	// Enabled controls whether update checking is enabled.
	Enabled bool
	// AirGapMode disables all external network calls for update checking.
	AirGapMode bool
	// GitHubOwner is the GitHub repository owner (optional, defaults to MacJediWizard).
	GitHubOwner string
	// GitHubRepo is the GitHub repository name (optional, defaults to keldris).
	GitHubRepo string
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig(currentVersion string) Config {
	return Config{
		CurrentVersion: currentVersion,
		CheckInterval:  DefaultCheckInterval,
		HTTPTimeout:    DefaultHTTPTimeout,
		Enabled:        true,
		AirGapMode:     false,
		GitHubOwner:    GitHubOwner,
		GitHubRepo:     GitHubRepo,
	}
}

// Checker handles checking for Keldris version updates.
type Checker struct {
	config     Config
	httpClient *http.Client
	logger     zerolog.Logger

	mu          sync.RWMutex
	cachedInfo  *UpdateInfo
	lastCheckAt time.Time
}

// NewChecker creates a new update Checker.
func NewChecker(config Config, logger zerolog.Logger) *Checker {
	if config.GitHubOwner == "" {
		config.GitHubOwner = GitHubOwner
	}
	if config.GitHubRepo == "" {
		config.GitHubRepo = GitHubRepo
	}
	if config.CheckInterval == 0 {
		config.CheckInterval = DefaultCheckInterval
	}
	if config.HTTPTimeout == 0 {
		config.HTTPTimeout = DefaultHTTPTimeout
	}

	return &Checker{
		config: config,
		httpClient: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		logger: logger.With().Str("component", "update_checker").Logger(),
	}
}

// Check checks for available updates.
// Returns cached result if within the check interval.
func (c *Checker) Check(ctx context.Context) (*UpdateInfo, error) {
	if !c.config.Enabled {
		return nil, ErrCheckDisabled
	}
	if c.config.AirGapMode {
		return nil, ErrAirGapMode
	}

	c.mu.RLock()
	if c.cachedInfo != nil && time.Since(c.lastCheckAt) < c.config.CheckInterval {
		cached := *c.cachedInfo
		c.mu.RUnlock()
		return &cached, nil
	}
	c.mu.RUnlock()

	return c.forceCheck(ctx)
}

// ForceCheck performs an update check regardless of cache.
func (c *Checker) ForceCheck(ctx context.Context) (*UpdateInfo, error) {
	if !c.config.Enabled {
		return nil, ErrCheckDisabled
	}
	if c.config.AirGapMode {
		return nil, ErrAirGapMode
	}

	return c.forceCheck(ctx)
}

func (c *Checker) forceCheck(ctx context.Context) (*UpdateInfo, error) {
	c.logger.Debug().Msg("checking for updates")

	release, err := c.fetchLatestRelease(ctx)
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to fetch latest release")
		return nil, fmt.Errorf("fetch latest release: %w", err)
	}

	now := time.Now().UTC()
	info := &UpdateInfo{
		CurrentVersion:         c.config.CurrentVersion,
		LatestVersion:          release.TagName,
		ReleaseNotes:           release.Body,
		ReleaseURL:             release.HTMLURL,
		ChangelogURL:           ChangelogURL,
		UpgradeInstructionsURL: UpgradeInstructionsURL,
		PublishedAt:            release.PublishedAt,
		CheckedAt:              now.Format(time.RFC3339),
		NextCheckAt:            now.Add(c.config.CheckInterval).Format(time.RFC3339),
	}

	latestVersion := normalizeVersion(release.TagName)
	currentVersion := normalizeVersion(c.config.CurrentVersion)
	info.UpdateAvailable = isNewerVersion(latestVersion, currentVersion)

	if info.UpdateAvailable {
		c.logger.Info().
			Str("current_version", c.config.CurrentVersion).
			Str("latest_version", release.TagName).
			Msg("update available")
	} else {
		c.logger.Debug().
			Str("current_version", c.config.CurrentVersion).
			Str("latest_version", release.TagName).
			Msg("no update available")
	}

	c.mu.Lock()
	c.cachedInfo = info
	c.lastCheckAt = now
	c.mu.Unlock()

	return info, nil
}

// GetCachedInfo returns the cached update info without making a network call.
// Returns nil if no cached info is available.
func (c *Checker) GetCachedInfo() *UpdateInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cachedInfo == nil {
		return nil
	}
	cached := *c.cachedInfo
	return &cached
}

// IsEnabled returns whether update checking is enabled.
func (c *Checker) IsEnabled() bool {
	return c.config.Enabled && !c.config.AirGapMode
}

// SetEnabled enables or disables update checking.
func (c *Checker) SetEnabled(enabled bool) {
	c.config.Enabled = enabled
}

// SetAirGapMode enables or disables air-gap mode.
func (c *Checker) SetAirGapMode(enabled bool) {
	c.config.AirGapMode = enabled
}

func (c *Checker) fetchLatestRelease(ctx context.Context) (*Release, error) {
	url := fmt.Sprintf(GitHubReleasesAPI, c.config.GitHubOwner, c.config.GitHubRepo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("Keldris/%s", c.config.CurrentVersion))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
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
