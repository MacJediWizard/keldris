// Package diagnostics provides self-test functionality for the Keldris agent.
package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/config"
)

// CheckStatus represents the status of a diagnostic check.
type CheckStatus string

const (
	// StatusPass indicates the check passed.
	StatusPass CheckStatus = "pass"
	// StatusFail indicates the check failed.
	StatusFail CheckStatus = "fail"
	// StatusWarn indicates the check passed with warnings.
	StatusWarn CheckStatus = "warn"
	// StatusSkip indicates the check was skipped.
	StatusSkip CheckStatus = "skip"
)

// CheckResult represents the result of a single diagnostic check.
type CheckResult struct {
	Name    string      `json:"name"`
	Status  CheckStatus `json:"status"`
	Message string      `json:"message,omitempty"`
	Details any         `json:"details,omitempty"`
}

// DiagnosticsResult contains the complete diagnostics output.
type DiagnosticsResult struct {
	Timestamp    time.Time     `json:"timestamp"`
	AgentVersion string        `json:"agent_version"`
	Hostname     string        `json:"hostname"`
	OS           string        `json:"os"`
	Arch         string        `json:"arch"`
	Checks       []CheckResult `json:"checks"`
	Summary      Summary       `json:"summary"`
}

// Summary provides a quick overview of the diagnostics results.
type Summary struct {
	Total   int  `json:"total"`
	Passed  int  `json:"passed"`
	Failed  int  `json:"failed"`
	Warned  int  `json:"warned"`
	Skipped int  `json:"skipped"`
	AllPass bool `json:"all_pass"`
}

// DiskSpaceDetails contains disk space information.
type DiskSpaceDetails struct {
	Path       string `json:"path"`
	TotalBytes int64  `json:"total_bytes"`
	FreeBytes  int64  `json:"free_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
	UsedPct    float64 `json:"used_percent"`
}

// ResticDetails contains restic binary information.
type ResticDetails struct {
	Path    string `json:"path,omitempty"`
	Version string `json:"version,omitempty"`
}

// ServerDetails contains server connectivity information.
type ServerDetails struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code,omitempty"`
	Latency    string `json:"latency,omitempty"`
}

// Runner runs diagnostic checks.
type Runner struct {
	cfg     *config.AgentConfig
	version string
}

// NewRunner creates a new diagnostics runner.
func NewRunner(cfg *config.AgentConfig, version string) *Runner {
	return &Runner{
		cfg:     cfg,
		version: version,
	}
}

// Run executes all diagnostic checks and returns the result.
func (r *Runner) Run(ctx context.Context) *DiagnosticsResult {
	hostname, _ := os.Hostname()

	result := &DiagnosticsResult{
		Timestamp:    time.Now().UTC(),
		AgentVersion: r.version,
		Hostname:     hostname,
		OS:           runtime.GOOS,
		Arch:         runtime.GOARCH,
		Checks:       make([]CheckResult, 0),
	}

	// Run all checks
	result.Checks = append(result.Checks, r.checkServerConnectivity(ctx))
	result.Checks = append(result.Checks, r.checkAPIKey(ctx))
	result.Checks = append(result.Checks, r.checkResticBinary(ctx))
	result.Checks = append(result.Checks, r.checkDiskSpace(ctx))
	result.Checks = append(result.Checks, r.checkConfigPermissions(ctx))

	// Calculate summary
	for _, check := range result.Checks {
		result.Summary.Total++
		switch check.Status {
		case StatusPass:
			result.Summary.Passed++
		case StatusFail:
			result.Summary.Failed++
		case StatusWarn:
			result.Summary.Warned++
		case StatusSkip:
			result.Summary.Skipped++
		}
	}
	result.Summary.AllPass = result.Summary.Failed == 0

	return result
}

// checkServerConnectivity tests connection to the Keldris server.
func (r *Runner) checkServerConnectivity(ctx context.Context) CheckResult {
	check := CheckResult{
		Name: "server_connectivity",
	}

	if r.cfg == nil || r.cfg.ServerURL == "" {
		check.Status = StatusSkip
		check.Message = "Server URL not configured"
		return check
	}

	details := ServerDetails{
		URL: r.cfg.ServerURL,
	}

	client := &http.Client{Timeout: 10 * time.Second}
	healthURL := r.cfg.ServerURL + "/health"

	start := time.Now()
	resp, err := client.Get(healthURL)
	latency := time.Since(start)

	details.Latency = latency.String()

	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to connect to server: %v", err)
		check.Details = details
		return check
	}
	defer resp.Body.Close()

	details.StatusCode = resp.StatusCode

	if resp.StatusCode != http.StatusOK {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Server returned HTTP %d", resp.StatusCode)
		check.Details = details
		return check
	}

	check.Status = StatusPass
	check.Message = fmt.Sprintf("Connected to server (latency: %s)", latency.Round(time.Millisecond))
	check.Details = details
	return check
}

// checkAPIKey verifies the API key is valid by making an authenticated request.
func (r *Runner) checkAPIKey(ctx context.Context) CheckResult {
	check := CheckResult{
		Name: "api_key",
	}

	if r.cfg == nil || r.cfg.APIKey == "" {
		check.Status = StatusSkip
		check.Message = "API key not configured"
		return check
	}

	if r.cfg.ServerURL == "" {
		check.Status = StatusSkip
		check.Message = "Server URL not configured"
		return check
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Use the agent health endpoint to verify API key
	req, err := http.NewRequestWithContext(ctx, "GET", r.cfg.ServerURL+"/api/v1/agent/commands", nil)
	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to create request: %v", err)
		return check
	}

	req.Header.Set("X-API-Key", r.cfg.APIKey)

	resp, err := client.Do(req)
	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to verify API key: %v", err)
		return check
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		check.Status = StatusPass
		check.Message = "API key is valid"
	case http.StatusUnauthorized, http.StatusForbidden:
		check.Status = StatusFail
		check.Message = "API key is invalid or expired"
	default:
		check.Status = StatusWarn
		check.Message = fmt.Sprintf("Unexpected response (HTTP %d)", resp.StatusCode)
	}

	return check
}

// checkResticBinary verifies the restic binary is available and working.
func (r *Runner) checkResticBinary(ctx context.Context) CheckResult {
	check := CheckResult{
		Name: "restic_binary",
	}

	details := ResticDetails{}

	// Find restic binary
	resticPath, err := exec.LookPath("restic")
	if err != nil {
		check.Status = StatusFail
		check.Message = "Restic binary not found in PATH"
		return check
	}

	details.Path = resticPath

	// Get restic version
	cmd := exec.CommandContext(ctx, resticPath, "version")
	output, err := cmd.Output()
	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to run restic: %v", err)
		check.Details = details
		return check
	}

	versionStr := strings.TrimSpace(string(output))
	details.Version = versionStr

	check.Status = StatusPass
	check.Message = fmt.Sprintf("Restic available at %s", resticPath)
	check.Details = details
	return check
}

// checkDiskSpace verifies adequate disk space is available.
func (r *Runner) checkDiskSpace(ctx context.Context) CheckResult {
	check := CheckResult{
		Name: "disk_space",
	}

	// Check the home directory's disk
	homeDir, err := os.UserHomeDir()
	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to get home directory: %v", err)
		return check
	}

	details, err := getDiskSpace(homeDir)
	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to check disk space: %v", err)
		return check
	}

	// Warning if > 90% used, fail if > 95% used
	if details.UsedPct >= 95 {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Critically low disk space: %.1f%% used", details.UsedPct)
		check.Details = details
		return check
	}

	if details.UsedPct >= 90 {
		check.Status = StatusWarn
		check.Message = fmt.Sprintf("Low disk space warning: %.1f%% used", details.UsedPct)
		check.Details = details
		return check
	}

	check.Status = StatusPass
	check.Message = fmt.Sprintf("Disk space OK: %.1f%% used, %s free", details.UsedPct, formatBytes(details.FreeBytes))
	check.Details = details
	return check
}

// checkConfigPermissions verifies file permissions on config directory.
func (r *Runner) checkConfigPermissions(ctx context.Context) CheckResult {
	check := CheckResult{
		Name: "config_permissions",
	}

	configDir, err := config.DefaultConfigDir()
	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to get config directory: %v", err)
		return check
	}

	// Check if config directory exists
	info, err := os.Stat(configDir)
	if os.IsNotExist(err) {
		// Directory doesn't exist - check if we can create it
		testDir := filepath.Join(configDir, ".test")
		if err := os.MkdirAll(testDir, 0700); err != nil {
			check.Status = StatusFail
			check.Message = fmt.Sprintf("Cannot create config directory: %v", err)
			return check
		}
		os.RemoveAll(testDir)

		check.Status = StatusPass
		check.Message = "Config directory can be created"
		return check
	}

	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to check config directory: %v", err)
		return check
	}

	if !info.IsDir() {
		check.Status = StatusFail
		check.Message = "Config path exists but is not a directory"
		return check
	}

	// Check permissions
	mode := info.Mode().Perm()
	details := map[string]any{
		"path":        configDir,
		"permissions": fmt.Sprintf("%04o", mode),
	}

	// On Unix, config directory should be readable/writable only by owner
	if runtime.GOOS != "windows" {
		if mode&0077 != 0 {
			check.Status = StatusWarn
			check.Message = fmt.Sprintf("Config directory has loose permissions (%04o), recommend 0700", mode)
			check.Details = details
			return check
		}
	}

	// Check if we can write to the directory
	testFile := filepath.Join(configDir, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Cannot write to config directory: %v", err)
		check.Details = details
		return check
	}
	os.Remove(testFile)

	check.Status = StatusPass
	check.Message = "Config directory permissions OK"
	check.Details = details
	return check
}

// getDiskSpace is defined in platform-specific files:
// diskspace_unix.go (linux, darwin) and diskspace_windows.go

// formatBytes formats bytes as a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ToJSON returns the diagnostics result as JSON.
func (r *DiagnosticsResult) ToJSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// ToMap returns the diagnostics result as a map for command result reporting.
func (r *DiagnosticsResult) ToMap() map[string]any {
	return map[string]any{
		"timestamp":     r.Timestamp,
		"agent_version": r.AgentVersion,
		"hostname":      r.Hostname,
		"os":            r.OS,
		"arch":          r.Arch,
		"checks":        r.Checks,
		"summary":       r.Summary,
	}
}
