package backup

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// NetworkDrives provides network mount detection and validation.
type NetworkDrives struct {
	logger       zerolog.Logger
	staleTimeout time.Duration
}

// NewNetworkDrives creates a new NetworkDrives instance.
func NewNetworkDrives(logger zerolog.Logger) *NetworkDrives {
	return &NetworkDrives{
		logger:       logger.With().Str("component", "network_drives").Logger(),
		staleTimeout: 5 * time.Second,
	}
}

// DetectMounts detects all network mounts on the system.
func (nd *NetworkDrives) DetectMounts(ctx context.Context) ([]models.NetworkMount, error) {
	switch runtime.GOOS {
	case "linux":
		return nd.detectLinuxMounts(ctx)
	case "darwin":
		return nd.detectDarwinMounts(ctx)
	case "windows":
		return nd.detectWindowsMounts(ctx)
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// detectLinuxMounts parses /proc/mounts for NFS/SMB/CIFS mounts.
func (nd *NetworkDrives) detectLinuxMounts(ctx context.Context) ([]models.NetworkMount, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, fmt.Errorf("open /proc/mounts: %w", err)
	}
	defer file.Close()

	var mounts []models.NetworkMount
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}

		remote := fields[0]
		mountPath := fields[1]
		fsType := strings.ToLower(fields[2])

		var mountType models.MountType
		switch {
		case fsType == "nfs" || fsType == "nfs4":
			mountType = models.MountTypeNFS
		case fsType == "cifs":
			mountType = models.MountTypeCIFS
		case fsType == "smbfs":
			mountType = models.MountTypeSMB
		default:
			continue
		}

		status := nd.checkMountStatus(ctx, mountPath)

		mounts = append(mounts, models.NetworkMount{
			Path:        mountPath,
			Type:        mountType,
			Remote:      remote,
			Status:      status,
			LastChecked: time.Now(),
		})
	}

	return mounts, scanner.Err()
}

// detectDarwinMounts uses mount command on macOS.
func (nd *NetworkDrives) detectDarwinMounts(ctx context.Context) ([]models.NetworkMount, error) {
	cmd := exec.CommandContext(ctx, "mount")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run mount: %w", err)
	}

	var mounts []models.NetworkMount
	for _, line := range strings.Split(string(output), "\n") {
		// Format: remote on /path (type, options)
		parts := strings.SplitN(line, " on ", 2)
		if len(parts) != 2 {
			continue
		}

		remote := parts[0]
		rest := parts[1]

		pathEnd := strings.Index(rest, " (")
		if pathEnd == -1 {
			continue
		}

		mountPath := rest[:pathEnd]
		typeInfo := rest[pathEnd+2:]

		var mountType models.MountType
		switch {
		case strings.HasPrefix(typeInfo, "nfs"):
			mountType = models.MountTypeNFS
		case strings.HasPrefix(typeInfo, "smbfs"):
			mountType = models.MountTypeSMB
		case strings.HasPrefix(typeInfo, "cifs"):
			mountType = models.MountTypeCIFS
		default:
			continue
		}

		status := nd.checkMountStatus(ctx, mountPath)

		mounts = append(mounts, models.NetworkMount{
			Path:        mountPath,
			Type:        mountType,
			Remote:      remote,
			Status:      status,
			LastChecked: time.Now(),
		})
	}

	return mounts, nil
}

// detectWindowsMounts detects network drives on Windows.
func (nd *NetworkDrives) detectWindowsMounts(ctx context.Context) ([]models.NetworkMount, error) {
	cmd := exec.CommandContext(ctx, "net", "use")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("run net use: %w", err)
	}

	var mounts []models.NetworkMount
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		// Skip header lines and empty lines
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "\\\\") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		// Find the local drive letter and remote path
		var local, remote string
		for i, field := range fields {
			if strings.HasPrefix(field, "\\\\") {
				remote = field
				if i > 0 && len(fields[i-1]) <= 3 && strings.Contains(fields[i-1], ":") {
					local = fields[i-1]
				}
				break
			}
		}

		if remote == "" {
			continue
		}

		// Use remote as path if no local drive letter
		path := local
		if path == "" {
			path = remote
		}

		status := nd.checkMountStatus(ctx, path)

		mounts = append(mounts, models.NetworkMount{
			Path:        path,
			Type:        models.MountTypeSMB,
			Remote:      remote,
			Status:      status,
			LastChecked: time.Now(),
		})
	}

	return mounts, nil
}

// checkMountStatus checks if a mount is accessible (not stale).
func (nd *NetworkDrives) checkMountStatus(ctx context.Context, path string) models.MountStatus {
	// Create a timeout context for the check
	checkCtx, cancel := context.WithTimeout(ctx, nd.staleTimeout)
	defer cancel()

	done := make(chan models.MountStatus, 1)

	go func() {
		// Try to stat the mount point
		_, err := os.Stat(path)
		if err != nil {
			if isStaleNFSError(err) {
				done <- models.MountStatusStale
				return
			}
			done <- models.MountStatusDisconnected
			return
		}
		done <- models.MountStatusConnected
	}()

	select {
	case status := <-done:
		return status
	case <-checkCtx.Done():
		return models.MountStatusStale
	}
}

// isStaleNFSError checks if an error indicates a stale NFS handle.
func isStaleNFSError(err error) bool {
	if pathErr, ok := err.(*os.PathError); ok {
		if errno, ok := pathErr.Err.(syscall.Errno); ok {
			// ESTALE = 116 on Linux
			return errno == 116
		}
	}
	return false
}

// ValidateMountForBackup checks if a path is on a network mount and if it's available.
// Returns (isValid, mount, error). If the path is not on a network mount, mount will be nil.
func (nd *NetworkDrives) ValidateMountForBackup(ctx context.Context, path string, mounts []models.NetworkMount) (bool, *models.NetworkMount, error) {
	// Check if path is under any known mount
	for i := range mounts {
		mount := &mounts[i]
		if strings.HasPrefix(path, mount.Path) || strings.HasPrefix(path, mount.Path+string(os.PathSeparator)) {
			// Refresh status
			mount.Status = nd.checkMountStatus(ctx, mount.Path)
			mount.LastChecked = time.Now()

			if mount.Status != models.MountStatusConnected {
				return false, mount, fmt.Errorf("mount %s is %s", mount.Path, mount.Status)
			}
			return true, mount, nil
		}
	}

	// Path is not on a network mount (local path)
	return true, nil, nil
}

// GetNetworkPathsFromSchedule returns paths from a schedule that are on network mounts.
func (nd *NetworkDrives) GetNetworkPathsFromSchedule(paths []string, mounts []models.NetworkMount) []NetworkPathInfo {
	var networkPaths []NetworkPathInfo

	for _, path := range paths {
		for i := range mounts {
			if strings.HasPrefix(path, mounts[i].Path) || strings.HasPrefix(path, mounts[i].Path+string(os.PathSeparator)) {
				networkPaths = append(networkPaths, NetworkPathInfo{
					Path:  path,
					Mount: &mounts[i],
				})
				break
			}
		}
	}

	return networkPaths
}

// NetworkPathInfo contains information about a path on a network mount.
type NetworkPathInfo struct {
	Path  string
	Mount *models.NetworkMount
}

// RefreshMountStatuses updates the status of all mounts.
func (nd *NetworkDrives) RefreshMountStatuses(ctx context.Context, mounts []models.NetworkMount) []models.NetworkMount {
	now := time.Now()
	for i := range mounts {
		mounts[i].Status = nd.checkMountStatus(ctx, mounts[i].Path)
		mounts[i].LastChecked = now
	}
	return mounts
}
