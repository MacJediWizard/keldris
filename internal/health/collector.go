// Package health provides health checking and metrics collection for agents.
package health

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// Metrics contains system metrics collected from an agent.
type Metrics struct {
	CPUUsage          float64     `json:"cpu_usage"`
	MemoryUsage       float64     `json:"memory_usage"`
	DiskUsage         float64     `json:"disk_usage"`
	DiskFreeBytes     int64       `json:"disk_free_bytes"`
	DiskTotalBytes    int64       `json:"disk_total_bytes"`
	NetworkUp         bool        `json:"network_up"`
	UptimeSeconds     int64       `json:"uptime_seconds"`
	ResticVersion     string      `json:"restic_version,omitempty"`
	ResticAvailable   bool        `json:"restic_available"`
	PiholeInfo        *PiholeInfo `json:"pihole_info,omitempty"`
}

// PiholeInfo contains Pi-hole detection and version information.
type PiholeInfo struct {
	Installed       bool   `json:"installed"`
	Version         string `json:"version,omitempty"`
	FTLVersion      string `json:"ftl_version,omitempty"`
	WebVersion      string `json:"web_version,omitempty"`
	ConfigDir       string `json:"config_dir,omitempty"`
	BlockingEnabled bool   `json:"blocking_enabled"`
}

// Collector collects system metrics.
type Collector struct {
	startTime    time.Time
	serverURL    string
	resticBinary string
}

// NewCollector creates a new metrics collector.
func NewCollector(serverURL, resticBinary string) *Collector {
	return &Collector{
		startTime:    time.Now(),
		serverURL:    serverURL,
		resticBinary: resticBinary,
	}
}

// Collect gathers all system metrics.
func (c *Collector) Collect(ctx context.Context) (*Metrics, error) {
	m := &Metrics{
		UptimeSeconds: int64(time.Since(c.startTime).Seconds()),
	}

	// CPU usage (average over 1 second)
	cpuPercent, err := cpu.PercentWithContext(ctx, time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		m.CPUUsage = cpuPercent[0]
	}

	// Memory usage
	memStat, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		m.MemoryUsage = memStat.UsedPercent
	}

	// Disk usage - check the root filesystem or current drive
	diskPath := "/"
	if runtime.GOOS == "windows" {
		diskPath = "C:\\"
	}
	diskStat, err := disk.UsageWithContext(ctx, diskPath)
	if err == nil {
		m.DiskUsage = diskStat.UsedPercent
		m.DiskFreeBytes = int64(diskStat.Free)
		m.DiskTotalBytes = int64(diskStat.Total)
	}

	// Network connectivity - check if server is reachable
	m.NetworkUp = c.checkNetworkConnectivity(ctx)

	// Restic version
	m.ResticVersion, m.ResticAvailable = c.getResticVersion(ctx)

	// Pi-hole detection
	m.PiholeInfo = c.detectPihole(ctx)

	return m, nil
}

// checkNetworkConnectivity tests if the server URL is reachable.
func (c *Collector) checkNetworkConnectivity(ctx context.Context) bool {
	if c.serverURL == "" {
		return false
	}

	// Check network interfaces are up
	interfaces, err := net.InterfacesWithContext(ctx)
	if err != nil {
		return false
	}

	hasActiveInterface := false
	for _, iface := range interfaces {
		// Skip loopback
		if strings.Contains(strings.ToLower(iface.Name), "lo") ||
			strings.Contains(strings.ToLower(iface.Name), "loopback") {
			continue
		}
		// Check if interface has addresses
		if len(iface.Addrs) > 0 {
			hasActiveInterface = true
			break
		}
	}

	return hasActiveInterface
}

// getResticVersion gets the installed restic version.
func (c *Collector) getResticVersion(ctx context.Context) (string, bool) {
	binary := c.resticBinary
	if binary == "" {
		binary = "restic"
	}

	// Try to find restic in PATH if not absolute path
	if !strings.HasPrefix(binary, "/") && !strings.HasPrefix(binary, "C:") {
		path, err := exec.LookPath(binary)
		if err != nil {
			return "", false
		}
		binary = path
	}

	cmd := exec.CommandContext(ctx, binary, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", false
	}

	// Parse version from "restic 0.16.0 compiled with go1.21.0 on linux/amd64"
	version := strings.TrimSpace(string(output))
	parts := strings.Fields(version)
	if len(parts) >= 2 {
		return parts[1], true
	}

	return version, true
}

// GetOSInfo returns operating system information.
func GetOSInfo() map[string]string {
	hostname, _ := os.Hostname()
	return map[string]string{
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"hostname": hostname,
		"version":  getOSVersion(),
	}
}

// getOSVersion returns the OS version string.
func getOSVersion() string {
	switch runtime.GOOS {
	case "linux":
		// Try to read /etc/os-release
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				}
			}
		}
	case "darwin":
		cmd := exec.Command("sw_vers", "-productVersion")
		output, err := cmd.Output()
		if err == nil {
			return fmt.Sprintf("macOS %s", strings.TrimSpace(string(output)))
		}
	case "windows":
		cmd := exec.Command("cmd", "/c", "ver")
		output, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(output))
		}
	}
	return runtime.GOOS
}

// detectPihole checks if Pi-hole is installed and returns version info.
func (c *Collector) detectPihole(ctx context.Context) *PiholeInfo {
	info := &PiholeInfo{
		Installed: false,
	}

	// Check for pihole binary
	piholeBinary := "/usr/local/bin/pihole"
	if _, err := os.Stat(piholeBinary); err != nil {
		// Try to find in PATH
		path, err := exec.LookPath("pihole")
		if err != nil {
			return info
		}
		piholeBinary = path
	}

	info.Installed = true

	// Get Pi-hole version
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Get core version
	cmd := exec.CommandContext(ctx, piholeBinary, "-v", "-p")
	if output, err := cmd.Output(); err == nil {
		info.Version = parsePiholeVersion(string(output))
	}

	// Get FTL version
	cmd = exec.CommandContext(ctx, piholeBinary, "-v", "-f")
	if output, err := cmd.Output(); err == nil {
		info.FTLVersion = parsePiholeVersion(string(output))
	}

	// Get web version
	cmd = exec.CommandContext(ctx, piholeBinary, "-v", "-a")
	if output, err := cmd.Output(); err == nil {
		info.WebVersion = parsePiholeVersion(string(output))
	}

	// Check config directory
	configDir := "/etc/pihole"
	if _, err := os.Stat(configDir); err == nil {
		info.ConfigDir = configDir
	}

	// Check if blocking is enabled from setupVars.conf
	setupVars := "/etc/pihole/setupVars.conf"
	if data, err := os.ReadFile(setupVars); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "BLOCKING_ENABLED=") {
				info.BlockingEnabled = strings.TrimPrefix(line, "BLOCKING_ENABLED=") == "true"
				break
			}
		}
	}

	return info
}

// parsePiholeVersion extracts version string from pihole -v output.
func parsePiholeVersion(output string) string {
	output = strings.TrimSpace(output)
	// Output format: "Pi-hole version is v5.14.2 (Latest: v5.14.2)"
	// or "FTL version is v5.20 (Latest: v5.20)"
	// or just "v5.14.2"
	if strings.Contains(output, "version is") {
		parts := strings.Split(output, "version is")
		if len(parts) >= 2 {
			version := strings.TrimSpace(parts[1])
			// Remove "(Latest: ...)" if present
			if idx := strings.Index(version, "("); idx != -1 {
				version = strings.TrimSpace(version[:idx])
			}
			return version
		}
	}
	return output
}
