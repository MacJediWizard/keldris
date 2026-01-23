// Package backup provides backup functionality including FUSE mount support.
package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MountInfo tracks information about an active mount.
type MountInfo struct {
	ID           uuid.UUID
	SnapshotID   string
	MountPath    string
	StartTime    time.Time
	ExpiresAt    time.Time
	cmd          *exec.Cmd
	cancelFunc   context.CancelFunc
	unmountMutex sync.Mutex
}

// MountManager manages FUSE mounts for restic snapshots.
type MountManager struct {
	binary     string
	basePath   string
	mounts     map[uuid.UUID]*MountInfo
	mountMutex sync.RWMutex
	logger     zerolog.Logger
}

// NewMountManager creates a new MountManager.
func NewMountManager(basePath string, logger zerolog.Logger) *MountManager {
	return &MountManager{
		binary:   "restic",
		basePath: basePath,
		mounts:   make(map[uuid.UUID]*MountInfo),
		logger:   logger.With().Str("component", "mount_manager").Logger(),
	}
}

// NewMountManagerWithBinary creates a new MountManager with a custom restic binary.
func NewMountManagerWithBinary(binary, basePath string, logger zerolog.Logger) *MountManager {
	return &MountManager{
		binary:   binary,
		basePath: basePath,
		mounts:   make(map[uuid.UUID]*MountInfo),
		logger:   logger.With().Str("component", "mount_manager").Logger(),
	}
}

// Mount starts a restic mount process for the given snapshot.
func (m *MountManager) Mount(ctx context.Context, id uuid.UUID, cfg backends.ResticConfig, snapshotID string, timeout time.Duration) (*MountInfo, error) {
	// Create unique mount path
	mountPath := filepath.Join(m.basePath, id.String())

	// Create mount directory
	if err := os.MkdirAll(mountPath, 0755); err != nil {
		return nil, fmt.Errorf("create mount directory: %w", err)
	}

	m.logger.Info().
		Str("mount_id", id.String()).
		Str("snapshot_id", snapshotID).
		Str("mount_path", mountPath).
		Dur("timeout", timeout).
		Msg("starting mount")

	// Create cancellable context for the mount process
	mountCtx, cancelFunc := context.WithCancel(context.Background())

	// Build mount command
	args := []string{
		"mount",
		"--repo", cfg.Repository,
		"--snapshot", snapshotID,
		mountPath,
	}

	cmd := exec.CommandContext(mountCtx, m.binary, args...)

	// Set environment variables
	cmd.Env = append(os.Environ(), fmt.Sprintf("RESTIC_PASSWORD=%s", cfg.Password))
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Start the mount process
	if err := cmd.Start(); err != nil {
		cancelFunc()
		os.RemoveAll(mountPath)
		return nil, fmt.Errorf("start mount process: %w", err)
	}

	now := time.Now()
	info := &MountInfo{
		ID:         id,
		SnapshotID: snapshotID,
		MountPath:  mountPath,
		StartTime:  now,
		ExpiresAt:  now.Add(timeout),
		cmd:        cmd,
		cancelFunc: cancelFunc,
	}

	// Store mount info
	m.mountMutex.Lock()
	m.mounts[id] = info
	m.mountMutex.Unlock()

	// Start a goroutine to wait for the process and cleanup
	go m.waitForMount(info)

	// Start auto-unmount timer
	go m.scheduleAutoUnmount(info, timeout)

	m.logger.Info().
		Str("mount_id", id.String()).
		Str("mount_path", mountPath).
		Time("expires_at", info.ExpiresAt).
		Msg("mount started successfully")

	return info, nil
}

// waitForMount waits for the mount process to exit and cleans up.
func (m *MountManager) waitForMount(info *MountInfo) {
	err := info.cmd.Wait()

	m.mountMutex.Lock()
	delete(m.mounts, info.ID)
	m.mountMutex.Unlock()

	// Cleanup mount directory
	os.RemoveAll(info.MountPath)

	if err != nil {
		m.logger.Warn().
			Err(err).
			Str("mount_id", info.ID.String()).
			Msg("mount process exited with error")
	} else {
		m.logger.Info().
			Str("mount_id", info.ID.String()).
			Msg("mount process exited cleanly")
	}
}

// scheduleAutoUnmount schedules automatic unmount after the timeout.
func (m *MountManager) scheduleAutoUnmount(info *MountInfo, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	<-timer.C

	m.mountMutex.RLock()
	_, exists := m.mounts[info.ID]
	m.mountMutex.RUnlock()

	if exists {
		m.logger.Info().
			Str("mount_id", info.ID.String()).
			Msg("auto-unmounting expired mount")

		if err := m.Unmount(context.Background(), info.ID); err != nil {
			m.logger.Error().
				Err(err).
				Str("mount_id", info.ID.String()).
				Msg("failed to auto-unmount")
		}
	}
}

// Unmount stops the mount process and cleans up.
func (m *MountManager) Unmount(ctx context.Context, id uuid.UUID) error {
	m.mountMutex.RLock()
	info, exists := m.mounts[id]
	m.mountMutex.RUnlock()

	if !exists {
		return fmt.Errorf("mount not found: %s", id.String())
	}

	info.unmountMutex.Lock()
	defer info.unmountMutex.Unlock()

	m.logger.Info().
		Str("mount_id", id.String()).
		Str("mount_path", info.MountPath).
		Msg("unmounting snapshot")

	// Try graceful FUSE unmount first
	unmountCmd := exec.CommandContext(ctx, "fusermount", "-u", info.MountPath)
	if err := unmountCmd.Run(); err != nil {
		m.logger.Warn().
			Err(err).
			Str("mount_path", info.MountPath).
			Msg("fusermount failed, trying umount")

		// Fallback to umount
		umountCmd := exec.CommandContext(ctx, "umount", info.MountPath)
		if err := umountCmd.Run(); err != nil {
			m.logger.Warn().
				Err(err).
				Str("mount_path", info.MountPath).
				Msg("umount failed, force killing process")
		}
	}

	// Cancel the mount context to kill the process
	info.cancelFunc()

	m.logger.Info().
		Str("mount_id", id.String()).
		Msg("unmount completed")

	return nil
}

// ExtendMount extends the expiration time of an active mount.
func (m *MountManager) ExtendMount(id uuid.UUID, extension time.Duration) error {
	m.mountMutex.Lock()
	defer m.mountMutex.Unlock()

	info, exists := m.mounts[id]
	if !exists {
		return fmt.Errorf("mount not found: %s", id.String())
	}

	info.ExpiresAt = info.ExpiresAt.Add(extension)

	m.logger.Info().
		Str("mount_id", id.String()).
		Time("new_expires_at", info.ExpiresAt).
		Msg("mount expiration extended")

	return nil
}

// GetMount returns information about an active mount.
func (m *MountManager) GetMount(id uuid.UUID) (*MountInfo, bool) {
	m.mountMutex.RLock()
	defer m.mountMutex.RUnlock()

	info, exists := m.mounts[id]
	return info, exists
}

// ListMounts returns all active mounts.
func (m *MountManager) ListMounts() []*MountInfo {
	m.mountMutex.RLock()
	defer m.mountMutex.RUnlock()

	mounts := make([]*MountInfo, 0, len(m.mounts))
	for _, info := range m.mounts {
		mounts = append(mounts, info)
	}
	return mounts
}

// IsSnapshotMounted checks if a snapshot is currently mounted.
func (m *MountManager) IsSnapshotMounted(snapshotID string) bool {
	m.mountMutex.RLock()
	defer m.mountMutex.RUnlock()

	for _, info := range m.mounts {
		if info.SnapshotID == snapshotID {
			return true
		}
	}
	return false
}

// UnmountAll unmounts all active mounts.
func (m *MountManager) UnmountAll(ctx context.Context) error {
	m.mountMutex.RLock()
	ids := make([]uuid.UUID, 0, len(m.mounts))
	for id := range m.mounts {
		ids = append(ids, id)
	}
	m.mountMutex.RUnlock()

	var lastErr error
	for _, id := range ids {
		if err := m.Unmount(ctx, id); err != nil {
			m.logger.Error().
				Err(err).
				Str("mount_id", id.String()).
				Msg("failed to unmount during cleanup")
			lastErr = err
		}
	}

	return lastErr
}

// Cleanup removes any stale mount directories.
func (m *MountManager) Cleanup() error {
	entries, err := os.ReadDir(m.basePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read mount base directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if this is a known active mount
		id, err := uuid.Parse(entry.Name())
		if err != nil {
			continue
		}

		m.mountMutex.RLock()
		_, exists := m.mounts[id]
		m.mountMutex.RUnlock()

		if !exists {
			// Stale mount directory, clean it up
			dirPath := filepath.Join(m.basePath, entry.Name())
			m.logger.Info().
				Str("path", dirPath).
				Msg("cleaning up stale mount directory")

			os.RemoveAll(dirPath)
		}
	}

	return nil
}
