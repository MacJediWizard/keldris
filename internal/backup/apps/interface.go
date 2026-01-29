// Package apps provides application-specific backup implementations.
package apps

import (
	"context"
)

// AppType represents the type of application being backed up.
type AppType string

const (
	// AppTypePihole is the Pi-hole DNS sinkhole application.
	AppTypePihole AppType = "pihole"
)

// AppInfo contains information about a detected application.
type AppInfo struct {
	Type      AppType `json:"type"`
	Version   string  `json:"version,omitempty"`
	Installed bool    `json:"installed"`
	Path      string  `json:"path,omitempty"`
	ConfigDir string  `json:"config_dir,omitempty"`
}

// BackupResult contains the result of an application backup operation.
type BackupResult struct {
	Success      bool     `json:"success"`
	BackupPath   string   `json:"backup_path,omitempty"`
	BackupFiles  []string `json:"backup_files,omitempty"`
	SizeBytes    int64    `json:"size_bytes,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// RestoreResult contains the result of an application restore operation.
type RestoreResult struct {
	Success        bool     `json:"success"`
	RestoredFiles  []string `json:"restored_files,omitempty"`
	ErrorMessage   string   `json:"error_message,omitempty"`
	RestartNeeded  bool     `json:"restart_needed"`
	ServiceRestart bool     `json:"service_restart"`
}

// AppBackup defines the interface for application-specific backup operations.
type AppBackup interface {
	// Type returns the application type.
	Type() AppType

	// Detect checks if the application is installed and returns info.
	Detect(ctx context.Context) (*AppInfo, error)

	// Backup creates a backup of the application data.
	// outputDir is the directory where backup files should be placed.
	Backup(ctx context.Context, outputDir string) (*BackupResult, error)

	// Restore restores application data from a backup.
	// backupPath is the path to the backup file or directory.
	Restore(ctx context.Context, backupPath string) (*RestoreResult, error)

	// GetBackupPaths returns the default paths that should be backed up.
	GetBackupPaths() []string

	// Validate checks if the application is in a valid state for backup.
	Validate(ctx context.Context) error
}

// ValidAppTypes returns all valid application types.
func ValidAppTypes() []AppType {
	return []AppType{
		AppTypePihole,
	}
}

// IsValidAppType checks if the given type is a valid application type.
func IsValidAppType(t AppType) bool {
	for _, valid := range ValidAppTypes() {
		if t == valid {
			return true
		}
	}
	return false
}
