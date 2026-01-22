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
	CPUUsage        float64 `json:"cpu_usage"`
	MemoryUsage     float64 `json:"memory_usage"`
	DiskUsage       float64 `json:"disk_usage"`
	DiskFreeBytes   int64   `json:"disk_free_bytes"`
	DiskTotalBytes  int64   `json:"disk_total_bytes"`
	NetworkUp       bool    `json:"network_up"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
	ResticVersion   string  `json:"restic_version,omitempty"`
	ResticAvailable bool    `json:"restic_available"`
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
