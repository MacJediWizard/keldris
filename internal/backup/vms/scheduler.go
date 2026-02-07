package vms

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// ProxmoxBackupService handles Proxmox VM/container backup operations.
type ProxmoxBackupService struct {
	logger zerolog.Logger
}

// NewProxmoxBackupService creates a new Proxmox backup service.
func NewProxmoxBackupService(logger zerolog.Logger) *ProxmoxBackupService {
	return &ProxmoxBackupService{
		logger: logger.With().Str("component", "proxmox_backup_service").Logger(),
	}
}

// BackupResult contains the results of a Proxmox backup operation.
type BackupResult struct {
	Success       bool
	BackupPaths   []string // Local paths to backup files (for Restic)
	TotalSize     int64
	VMsBackedUp   int
	ErrorMessage  string
	VMResults     []VMBackupResult
}

// VMBackupResult contains the result for a single VM backup.
type VMBackupResult struct {
	VMID      int
	Name      string
	Type      string
	Success   bool
	BackupPath string
	Size      int64
	Duration  time.Duration
	Error     string
}

// BackupVMs executes vzdump backups for the specified VMs and downloads the results.
func (s *ProxmoxBackupService) BackupVMs(
	ctx context.Context,
	client *ProxmoxClient,
	opts *models.ProxmoxBackupOptions,
	tempDir string,
) (*BackupResult, error) {
	result := &BackupResult{
		Success:     true,
		BackupPaths: []string{},
		VMResults:   []VMBackupResult{},
	}

	// Get list of VMs to backup
	vmsToBackup, err := s.getVMsToBackup(ctx, client, opts)
	if err != nil {
		return nil, fmt.Errorf("get VMs to backup: %w", err)
	}

	if len(vmsToBackup) == 0 {
		s.logger.Info().Msg("no VMs match backup criteria")
		return result, nil
	}

	s.logger.Info().Int("count", len(vmsToBackup)).Msg("starting Proxmox backup")

	// Determine max wait time
	maxWait := 60 * time.Minute
	if opts.MaxWait > 0 {
		maxWait = time.Duration(opts.MaxWait) * time.Minute
	}

	// Backup each VM
	for _, vm := range vmsToBackup {
		vmResult := s.backupSingleVM(ctx, client, vm, opts, tempDir, maxWait)
		result.VMResults = append(result.VMResults, vmResult)

		if vmResult.Success {
			result.VMsBackedUp++
			result.TotalSize += vmResult.Size
			if vmResult.BackupPath != "" {
				result.BackupPaths = append(result.BackupPaths, vmResult.BackupPath)
			}
		} else {
			result.Success = false
			if result.ErrorMessage == "" {
				result.ErrorMessage = vmResult.Error
			} else {
				result.ErrorMessage += "; " + vmResult.Error
			}
		}
	}

	return result, nil
}

// getVMsToBackup determines which VMs should be backed up based on options.
func (s *ProxmoxBackupService) getVMsToBackup(
	ctx context.Context,
	client *ProxmoxClient,
	opts *models.ProxmoxBackupOptions,
) ([]ProxmoxVM, error) {
	allVMs, err := client.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	// If no specific IDs are provided, backup all
	if len(opts.VMIDs) == 0 && len(opts.ContainerIDs) == 0 {
		return allVMs, nil
	}

	// Build lookup maps
	vmIDSet := make(map[int]bool)
	for _, id := range opts.VMIDs {
		vmIDSet[id] = true
	}
	containerIDSet := make(map[int]bool)
	for _, id := range opts.ContainerIDs {
		containerIDSet[id] = true
	}

	// Filter VMs
	var filtered []ProxmoxVM
	for _, vm := range allVMs {
		if vm.Type == "qemu" && vmIDSet[vm.VMID] {
			filtered = append(filtered, vm)
		} else if vm.Type == "lxc" && containerIDSet[vm.VMID] {
			filtered = append(filtered, vm)
		}
	}

	return filtered, nil
}

// backupSingleVM performs vzdump backup for a single VM or container.
func (s *ProxmoxBackupService) backupSingleVM(
	ctx context.Context,
	client *ProxmoxClient,
	vm ProxmoxVM,
	opts *models.ProxmoxBackupOptions,
	tempDir string,
	maxWait time.Duration,
) VMBackupResult {
	startTime := time.Now()
	result := VMBackupResult{
		VMID: vm.VMID,
		Name: vm.Name,
		Type: vm.Type,
	}

	s.logger.Info().
		Int("vmid", vm.VMID).
		Str("name", vm.Name).
		Str("type", vm.Type).
		Msg("starting VM backup")

	// Start vzdump backup
	vzdumpOpts := VzdumpOptions{
		VMID:       vm.VMID,
		Type:       vm.Type,
		Mode:       opts.Mode,
		Compress:   opts.Compress,
		Storage:    opts.Storage,
		IncludeRAM: opts.IncludeRAM,
	}

	job, err := client.StartBackup(ctx, vzdumpOpts)
	if err != nil {
		result.Error = fmt.Sprintf("start backup failed: %v", err)
		s.logger.Error().Err(err).Int("vmid", vm.VMID).Msg("failed to start vzdump")
		return result
	}

	s.logger.Info().
		Int("vmid", vm.VMID).
		Str("upid", job.UPID).
		Msg("vzdump task started")

	// Wait for task to complete
	completedJob, err := client.WaitForTask(ctx, job.UPID, maxWait)
	if err != nil {
		result.Error = fmt.Sprintf("wait for backup failed: %v", err)
		s.logger.Error().Err(err).Int("vmid", vm.VMID).Msg("error waiting for vzdump")
		return result
	}

	if completedJob.ExitCode != "OK" {
		result.Error = fmt.Sprintf("vzdump failed with exit code: %s", completedJob.ExitCode)
		s.logger.Error().
			Int("vmid", vm.VMID).
			Str("exit_code", completedJob.ExitCode).
			Msg("vzdump task failed")
		return result
	}

	s.logger.Info().
		Int("vmid", vm.VMID).
		Msg("vzdump completed successfully")

	// Find and download the backup file
	if opts.Storage != "" {
		backupPath, size, err := s.downloadLatestBackup(ctx, client, vm.VMID, opts.Storage, tempDir)
		if err != nil {
			result.Error = fmt.Sprintf("download backup failed: %v", err)
			s.logger.Error().Err(err).Int("vmid", vm.VMID).Msg("failed to download backup")
			return result
		}

		result.BackupPath = backupPath
		result.Size = size

		// Remove from Proxmox storage if configured
		if opts.RemoveAfter {
			// Note: We'd need the volid to delete - this is simplified
			s.logger.Info().Int("vmid", vm.VMID).Msg("backup removal after Restic storage configured")
		}
	}

	result.Success = true
	result.Duration = time.Since(startTime)

	s.logger.Info().
		Int("vmid", vm.VMID).
		Str("name", vm.Name).
		Int64("size", result.Size).
		Dur("duration", result.Duration).
		Msg("VM backup completed")

	return result
}

// downloadLatestBackup downloads the most recent backup for a VM.
func (s *ProxmoxBackupService) downloadLatestBackup(
	ctx context.Context,
	client *ProxmoxClient,
	vmid int,
	storage string,
	destDir string,
) (string, int64, error) {
	backups, err := client.ListBackupFiles(ctx, storage, vmid)
	if err != nil {
		return "", 0, fmt.Errorf("list backup files: %w", err)
	}

	if len(backups) == 0 {
		return "", 0, fmt.Errorf("no backup files found for VMID %d", vmid)
	}

	// Find most recent backup
	var latest *BackupFile
	for i := range backups {
		if latest == nil || backups[i].Ctime > latest.Ctime {
			latest = &backups[i]
		}
	}

	// Create VM-specific subdirectory
	vmDir := filepath.Join(destDir, fmt.Sprintf("vm-%d", vmid))
	if err := os.MkdirAll(vmDir, 0755); err != nil {
		return "", 0, fmt.Errorf("create VM directory: %w", err)
	}

	// Download the backup
	localPath, err := client.DownloadBackup(ctx, latest.Volid, vmDir)
	if err != nil {
		return "", 0, fmt.Errorf("download backup: %w", err)
	}

	return localPath, latest.Size, nil
}

// CleanupTempFiles removes temporary backup files.
func (s *ProxmoxBackupService) CleanupTempFiles(paths []string) {
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil {
			s.logger.Warn().Err(err).Str("path", path).Msg("failed to cleanup temp file")
		}
	}
}
