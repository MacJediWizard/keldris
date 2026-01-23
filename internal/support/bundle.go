// Package support provides diagnostic bundle generation for support requests.
package support

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// BundleInfo contains metadata about the generated bundle.
type BundleInfo struct {
	Filename    string    `json:"filename"`
	Size        int64     `json:"size"`
	GeneratedAt time.Time `json:"generated_at"`
}

// SystemInfo contains information about the system.
type SystemInfo struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	GoVersion    string `json:"go_version"`
	NumCPU       int    `json:"num_cpu"`
	Hostname     string `json:"hostname"`
	OSVersion    string `json:"os_version,omitempty"`
	KernelInfo   string `json:"kernel_info,omitempty"`
	UptimeInfo   string `json:"uptime_info,omitempty"`
	MemoryInfo   string `json:"memory_info,omitempty"`
}

// AgentInfo contains information about the agent.
type AgentInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	AgentID   string `json:"agent_id,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	ServerURL string `json:"server_url,omitempty"`
}

// ServerInfo contains information about the server.
type ServerInfo struct {
	Version     string `json:"version"`
	Commit      string `json:"commit"`
	BuildDate   string `json:"build_date"`
	DatabaseURL string `json:"database_url,omitempty"`
}

// ConfigInfo contains sanitized configuration.
type ConfigInfo struct {
	ServerURL       string `json:"server_url,omitempty"`
	AgentID         string `json:"agent_id,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	AutoCheckUpdate bool   `json:"auto_check_update"`
}

// ErrorInfo contains information about recent errors.
type ErrorInfo struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Component string    `json:"component,omitempty"`
}

// BundleOptions configures bundle generation.
type BundleOptions struct {
	// IncludeLogs includes sanitized log files.
	IncludeLogs bool
	// IncludeConfig includes sanitized configuration.
	IncludeConfig bool
	// IncludeSystemInfo includes system information.
	IncludeSystemInfo bool
	// IncludeAgentInfo includes agent information.
	IncludeAgentInfo bool
	// IncludeServerInfo includes server information (server only).
	IncludeServerInfo bool
	// LogDir is the directory containing log files.
	LogDir string
	// MaxLogLines is the maximum number of log lines to include per file.
	MaxLogLines int
	// MaxLogFiles is the maximum number of log files to include.
	MaxLogFiles int
}

// DefaultBundleOptions returns default options for bundle generation.
func DefaultBundleOptions() BundleOptions {
	return BundleOptions{
		IncludeLogs:       true,
		IncludeConfig:     true,
		IncludeSystemInfo: true,
		IncludeAgentInfo:  true,
		IncludeServerInfo: false,
		LogDir:            "",
		MaxLogLines:       10000,
		MaxLogFiles:       5,
	}
}

// Generator generates support bundles.
type Generator struct {
	logger  zerolog.Logger
	options BundleOptions
}

// NewGenerator creates a new bundle generator.
func NewGenerator(logger zerolog.Logger, options BundleOptions) *Generator {
	return &Generator{
		logger:  logger.With().Str("component", "support_bundle").Logger(),
		options: options,
	}
}

// Generate creates a support bundle and returns the zip data.
func (g *Generator) Generate(ctx context.Context, info BundleData) ([]byte, *BundleInfo, error) {
	g.logger.Info().Msg("generating support bundle")

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	timestamp := time.Now().UTC()

	// Add README with submission instructions
	if err := g.addReadme(zipWriter, timestamp); err != nil {
		return nil, nil, fmt.Errorf("add readme: %w", err)
	}

	// Add system info
	if g.options.IncludeSystemInfo {
		if err := g.addSystemInfo(ctx, zipWriter); err != nil {
			g.logger.Warn().Err(err).Msg("failed to add system info")
		}
	}

	// Add agent info
	if g.options.IncludeAgentInfo && info.AgentInfo != nil {
		if err := g.addAgentInfo(zipWriter, info.AgentInfo); err != nil {
			g.logger.Warn().Err(err).Msg("failed to add agent info")
		}
	}

	// Add server info
	if g.options.IncludeServerInfo && info.ServerInfo != nil {
		if err := g.addServerInfo(zipWriter, info.ServerInfo); err != nil {
			g.logger.Warn().Err(err).Msg("failed to add server info")
		}
	}

	// Add config
	if g.options.IncludeConfig && info.Config != nil {
		if err := g.addConfig(zipWriter, info.Config); err != nil {
			g.logger.Warn().Err(err).Msg("failed to add config")
		}
	}

	// Add logs
	if g.options.IncludeLogs && g.options.LogDir != "" {
		if err := g.addLogs(ctx, zipWriter); err != nil {
			g.logger.Warn().Err(err).Msg("failed to add logs")
		}
	}

	// Add recent errors
	if len(info.RecentErrors) > 0 {
		if err := g.addRecentErrors(zipWriter, info.RecentErrors); err != nil {
			g.logger.Warn().Err(err).Msg("failed to add recent errors")
		}
	}

	// Add custom data
	for name, data := range info.CustomData {
		if err := g.addJSONFile(zipWriter, name, data); err != nil {
			g.logger.Warn().Err(err).Str("name", name).Msg("failed to add custom data")
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, nil, fmt.Errorf("close zip: %w", err)
	}

	data := buf.Bytes()
	bundleInfo := &BundleInfo{
		Filename:    fmt.Sprintf("keldris-support-bundle-%s.zip", timestamp.Format("20060102-150405")),
		Size:        int64(len(data)),
		GeneratedAt: timestamp,
	}

	g.logger.Info().
		Str("filename", bundleInfo.Filename).
		Int64("size", bundleInfo.Size).
		Msg("support bundle generated")

	return data, bundleInfo, nil
}

// BundleData contains all data to include in the bundle.
type BundleData struct {
	AgentInfo    *AgentInfo
	ServerInfo   *ServerInfo
	Config       *ConfigInfo
	RecentErrors []ErrorInfo
	CustomData   map[string]any
}

// addReadme adds the README file with submission instructions.
func (g *Generator) addReadme(zw *zip.Writer, timestamp time.Time) error {
	content := fmt.Sprintf(`Keldris Support Bundle
======================

Generated: %s

This bundle contains diagnostic information to help troubleshoot issues
with your Keldris installation.

Contents:
- system_info.json    System information (OS, architecture, etc.)
- agent_info.json     Agent version and configuration (if applicable)
- server_info.json    Server version and configuration (if applicable)
- config.json         Sanitized configuration (secrets removed)
- logs/               Recent log files (sanitized)
- errors.json         Recent error messages

Submitting This Bundle
----------------------

1. Email: Send this file to support@keldris.io with a description of your issue.

2. GitHub: Open an issue at https://github.com/MacJediWizard/keldris/issues
   and attach this file.

Note: This bundle has been automatically sanitized to remove sensitive
information like API keys, passwords, and connection strings. However,
please review the contents before submitting if you have concerns about
sensitive data.

If you find any sensitive information that should have been sanitized,
please let us know so we can improve our sanitization process.
`, timestamp.Format(time.RFC3339))

	return g.addFile(zw, "README.txt", []byte(content))
}

// addSystemInfo adds system information to the bundle.
func (g *Generator) addSystemInfo(ctx context.Context, zw *zip.Writer) error {
	hostname, _ := os.Hostname()

	info := SystemInfo{
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		GoVersion: runtime.Version(),
		NumCPU:    runtime.NumCPU(),
		Hostname:  hostname,
	}

	// Get OS-specific information
	switch runtime.GOOS {
	case "linux":
		info.OSVersion = g.getLinuxVersion()
		info.KernelInfo = g.getCommandOutput(ctx, "uname", "-r")
		info.MemoryInfo = g.getCommandOutput(ctx, "free", "-h")
	case "darwin":
		info.OSVersion = g.getCommandOutput(ctx, "sw_vers", "-productVersion")
		info.KernelInfo = g.getCommandOutput(ctx, "uname", "-r")
	case "windows":
		info.OSVersion = g.getCommandOutput(ctx, "cmd", "/c", "ver")
	}

	info.UptimeInfo = g.getUptimeInfo(ctx)

	return g.addJSONFile(zw, "system_info.json", info)
}

// addAgentInfo adds agent information to the bundle.
func (g *Generator) addAgentInfo(zw *zip.Writer, info *AgentInfo) error {
	// Sanitize server URL
	sanitized := *info
	sanitized.ServerURL = sanitizeURL(info.ServerURL)
	return g.addJSONFile(zw, "agent_info.json", sanitized)
}

// addServerInfo adds server information to the bundle.
func (g *Generator) addServerInfo(zw *zip.Writer, info *ServerInfo) error {
	// Sanitize database URL
	sanitized := *info
	sanitized.DatabaseURL = sanitizeConnectionString(info.DatabaseURL)
	return g.addJSONFile(zw, "server_info.json", sanitized)
}

// addConfig adds sanitized configuration to the bundle.
func (g *Generator) addConfig(zw *zip.Writer, config *ConfigInfo) error {
	// Config is already expected to be sanitized
	sanitized := *config
	sanitized.ServerURL = sanitizeURL(config.ServerURL)
	return g.addJSONFile(zw, "config.json", sanitized)
}

// addLogs adds sanitized log files to the bundle.
func (g *Generator) addLogs(ctx context.Context, zw *zip.Writer) error {
	if g.options.LogDir == "" {
		return nil
	}

	entries, err := os.ReadDir(g.options.LogDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read log dir: %w", err)
	}

	// Filter and sort log files
	var logFiles []os.DirEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".json") {
			logFiles = append(logFiles, entry)
		}
	}

	// Limit number of log files
	if len(logFiles) > g.options.MaxLogFiles {
		logFiles = logFiles[len(logFiles)-g.options.MaxLogFiles:]
	}

	for _, entry := range logFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		logPath := filepath.Join(g.options.LogDir, entry.Name())
		content, err := g.readAndSanitizeLog(logPath)
		if err != nil {
			g.logger.Warn().Err(err).Str("file", entry.Name()).Msg("failed to read log file")
			continue
		}

		if err := g.addFile(zw, "logs/"+entry.Name(), content); err != nil {
			g.logger.Warn().Err(err).Str("file", entry.Name()).Msg("failed to add log file")
		}
	}

	return nil
}

// readAndSanitizeLog reads a log file and sanitizes sensitive information.
func (g *Generator) readAndSanitizeLog(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, 10*1024*1024)) // 10MB max
	if err != nil {
		return nil, err
	}

	content := string(data)

	// Limit number of lines
	lines := strings.Split(content, "\n")
	if len(lines) > g.options.MaxLogLines {
		lines = lines[len(lines)-g.options.MaxLogLines:]
		lines = append([]string{"[... truncated ...]"}, lines...)
	}
	content = strings.Join(lines, "\n")

	// Sanitize the content
	content = SanitizeLogContent(content)

	return []byte(content), nil
}

// addRecentErrors adds recent errors to the bundle.
func (g *Generator) addRecentErrors(zw *zip.Writer, errors []ErrorInfo) error {
	// Sanitize error messages
	sanitized := make([]ErrorInfo, len(errors))
	for i, e := range errors {
		sanitized[i] = e
		sanitized[i].Message = SanitizeLogContent(e.Message)
	}
	return g.addJSONFile(zw, "errors.json", sanitized)
}

// addJSONFile adds a JSON file to the zip.
func (g *Generator) addJSONFile(zw *zip.Writer, name string, data any) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	return g.addFile(zw, name, content)
}

// addFile adds a file to the zip.
func (g *Generator) addFile(zw *zip.Writer, name string, content []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("create zip entry: %w", err)
	}
	if _, err := w.Write(content); err != nil {
		return fmt.Errorf("write zip entry: %w", err)
	}
	return nil
}

// getLinuxVersion gets the Linux distribution version.
func (g *Generator) getLinuxVersion() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		}
	}
	return ""
}

// getCommandOutput runs a command and returns its output.
func (g *Generator) getCommandOutput(ctx context.Context, name string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getUptimeInfo gets system uptime information.
func (g *Generator) getUptimeInfo(ctx context.Context) string {
	switch runtime.GOOS {
	case "linux", "darwin":
		return g.getCommandOutput(ctx, "uptime")
	default:
		return ""
	}
}

// Sensitive data patterns for sanitization.
var sensitivePatterns = []*regexp.Regexp{
	// API keys (common formats)
	regexp.MustCompile(`(?i)(api[_-]?key|apikey)["\s:=]+["']?([a-zA-Z0-9_-]{16,})["']?`),
	regexp.MustCompile(`(?i)(kld_[a-zA-Z0-9]{20,})`),

	// Passwords
	regexp.MustCompile(`(?i)(password|passwd|pwd|secret)["\s:=]+["']?([^\s"',}]{4,})["']?`),

	// Tokens
	regexp.MustCompile(`(?i)(token|bearer|auth)["\s:=]+["']?([a-zA-Z0-9_.-]{20,})["']?`),

	// Connection strings (PostgreSQL, MySQL, etc.)
	regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^@]+@`),

	// AWS credentials
	regexp.MustCompile(`(?i)(AKIA[0-9A-Z]{16})`),
	regexp.MustCompile(`(?i)(aws[_-]?secret[_-]?access[_-]?key)["\s:=]+["']?([a-zA-Z0-9/+=]{40})["']?`),

	// Generic secrets in key=value format
	regexp.MustCompile(`(?i)(secret|credential|private[_-]?key)["\s:=]+["']?([^\s"',}]{8,})["']?`),

	// Base64 encoded data that might be credentials
	regexp.MustCompile(`(?i)(basic|bearer)\s+[A-Za-z0-9+/=]{20,}`),
}

// SanitizeLogContent sanitizes sensitive information from log content.
func SanitizeLogContent(content string) string {
	for _, pattern := range sensitivePatterns {
		content = pattern.ReplaceAllStringFunc(content, func(match string) string {
			// Preserve the key part but replace the value
			idx := strings.IndexAny(match, ":=")
			if idx > 0 {
				return match[:idx+1] + " [REDACTED]"
			}
			return "[REDACTED]"
		})
	}
	return content
}

// sanitizeURL removes credentials from URLs.
func sanitizeURL(urlStr string) string {
	if urlStr == "" {
		return ""
	}
	// Remove any credentials in URL
	re := regexp.MustCompile(`(https?://)[^@]+@`)
	return re.ReplaceAllString(urlStr, "${1}[REDACTED]@")
}

// sanitizeConnectionString sanitizes database connection strings.
func sanitizeConnectionString(connStr string) string {
	if connStr == "" {
		return ""
	}

	// Handle URL format: postgres://user:password@host/db
	if strings.Contains(connStr, "://") {
		re := regexp.MustCompile(`(://[^:]+:)[^@]+(@)`)
		connStr = re.ReplaceAllString(connStr, "${1}[REDACTED]${2}")
	}

	// Handle key=value format: host=x user=y password=z
	re := regexp.MustCompile(`(?i)(password|passwd|pwd|secret)=[^\s]+`)
	connStr = re.ReplaceAllString(connStr, "${1}=[REDACTED]")

	return connStr
}
