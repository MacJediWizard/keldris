package apps

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

const (
	// Default Pi-hole paths
	defaultPiholeConfigDir   = "/etc/pihole"
	defaultDNSMasqConfigDir  = "/etc/dnsmasq.d"
	defaultGravityDB         = "/etc/pihole/gravity.db"
	defaultFTLDB             = "/etc/pihole/pihole-FTL.db"
	defaultPiholeSetupVars   = "/etc/pihole/setupVars.conf"
	defaultPiholeBinary      = "/usr/local/bin/pihole"
	defaultPiholeFTLBinary   = "/usr/bin/pihole-FTL"
	teleporterTimeout        = 5 * time.Minute
)

// PiholeBackup implements AppBackup for Pi-hole DNS sinkhole.
type PiholeBackup struct {
	// ConfigDir is the Pi-hole configuration directory (default: /etc/pihole).
	ConfigDir string `json:"config_dir,omitempty"`

	// DNSMasqDir is the dnsmasq.d configuration directory (default: /etc/dnsmasq.d).
	DNSMasqDir string `json:"dnsmasq_dir,omitempty"`

	// PiholeBinary is the path to the pihole CLI (default: /usr/local/bin/pihole).
	PiholeBinary string `json:"pihole_binary,omitempty"`

	// UseTeleporter uses pihole -a -t for backup instead of manual file copy.
	UseTeleporter bool `json:"use_teleporter"`

	// IncludeQueryLogs includes pihole-FTL.db (query logs) in backup.
	IncludeQueryLogs bool `json:"include_query_logs"`

	logger zerolog.Logger
}

// NewPiholeBackup creates a new PiholeBackup with default settings.
func NewPiholeBackup(logger zerolog.Logger) *PiholeBackup {
	return &PiholeBackup{
		ConfigDir:        defaultPiholeConfigDir,
		DNSMasqDir:       defaultDNSMasqConfigDir,
		PiholeBinary:     defaultPiholeBinary,
		UseTeleporter:    true,
		IncludeQueryLogs: true,
		logger:           logger.With().Str("app", "pihole").Logger(),
	}
}

// Type returns the application type.
func (p *PiholeBackup) Type() AppType {
	return AppTypePihole
}

// Detect checks if Pi-hole is installed and returns version info.
func (p *PiholeBackup) Detect(ctx context.Context) (*AppInfo, error) {
	info := &AppInfo{
		Type:      AppTypePihole,
		Installed: false,
	}

	// Check for pihole binary
	binary := p.PiholeBinary
	if binary == "" {
		binary = defaultPiholeBinary
	}

	// Try to find pihole in PATH if not absolute
	if !filepath.IsAbs(binary) {
		path, err := exec.LookPath(binary)
		if err != nil {
			// Try default location
			if _, err := os.Stat(defaultPiholeBinary); err == nil {
				binary = defaultPiholeBinary
			} else {
				return info, nil // Not installed
			}
		} else {
			binary = path
		}
	}

	// Check if binary exists
	if _, err := os.Stat(binary); err != nil {
		return info, nil
	}

	info.Path = binary
	info.Installed = true

	// Get version using pihole -v
	version, err := p.getVersion(ctx, binary)
	if err != nil {
		p.logger.Warn().Err(err).Msg("failed to get Pi-hole version")
	} else {
		info.Version = version
	}

	// Set config directory
	configDir := p.ConfigDir
	if configDir == "" {
		configDir = defaultPiholeConfigDir
	}
	if _, err := os.Stat(configDir); err == nil {
		info.ConfigDir = configDir
	}

	return info, nil
}

// getVersion retrieves the Pi-hole version.
func (p *PiholeBackup) getVersion(ctx context.Context, binary string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "-v", "-p")
	output, err := cmd.Output()
	if err != nil {
		// Try alternative version command
		cmd = exec.CommandContext(ctx, binary, "version")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("get version: %w", err)
		}
	}

	// Parse version from output
	// Output format: "Pi-hole version is v5.14.2 (Latest: v5.14.2)"
	// or just "v5.14.2"
	version := strings.TrimSpace(string(output))
	if strings.Contains(version, "version is") {
		parts := strings.Split(version, "version is")
		if len(parts) >= 2 {
			version = strings.TrimSpace(parts[1])
			// Remove "(Latest: ...)" if present
			if idx := strings.Index(version, "("); idx != -1 {
				version = strings.TrimSpace(version[:idx])
			}
		}
	}

	return version, nil
}

// Backup creates a backup of Pi-hole configuration and data.
func (p *PiholeBackup) Backup(ctx context.Context, outputDir string) (*BackupResult, error) {
	result := &BackupResult{
		Success:     false,
		BackupFiles: make([]string, 0),
	}

	// Validate Pi-hole installation
	if err := p.Validate(ctx); err != nil {
		result.ErrorMessage = err.Error()
		return result, err
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.ErrorMessage = fmt.Sprintf("create output directory: %v", err)
		return result, fmt.Errorf("create output directory: %w", err)
	}

	if p.UseTeleporter {
		return p.backupWithTeleporter(ctx, outputDir, result)
	}
	return p.backupManual(ctx, outputDir, result)
}

// backupWithTeleporter uses pihole -a -t to create a backup.
func (p *PiholeBackup) backupWithTeleporter(ctx context.Context, outputDir string, result *BackupResult) (*BackupResult, error) {
	p.logger.Info().Msg("creating Pi-hole backup using teleporter")

	ctx, cancel := context.WithTimeout(ctx, teleporterTimeout)
	defer cancel()

	// Generate backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(outputDir, fmt.Sprintf("pihole-teleporter_%s.tar.gz", timestamp))

	// Run pihole -a -t to create teleporter backup
	binary := p.PiholeBinary
	if binary == "" {
		binary = defaultPiholeBinary
	}

	cmd := exec.CommandContext(ctx, binary, "-a", "-t", backupFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("teleporter backup failed: %v: %s", err, string(output))
		return result, fmt.Errorf("teleporter backup: %w", err)
	}

	// Verify backup was created
	info, err := os.Stat(backupFile)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("backup file not created: %v", err)
		return result, fmt.Errorf("backup file not created: %w", err)
	}

	result.Success = true
	result.BackupPath = backupFile
	result.BackupFiles = append(result.BackupFiles, backupFile)
	result.SizeBytes = info.Size()

	p.logger.Info().
		Str("backup_file", backupFile).
		Int64("size_bytes", info.Size()).
		Msg("Pi-hole teleporter backup completed")

	// Additionally backup FTL database if requested
	if p.IncludeQueryLogs {
		ftlBackup, err := p.backupFTLDB(ctx, outputDir)
		if err != nil {
			p.logger.Warn().Err(err).Msg("failed to backup FTL database, continuing")
		} else if ftlBackup != "" {
			result.BackupFiles = append(result.BackupFiles, ftlBackup)
			if info, err := os.Stat(ftlBackup); err == nil {
				result.SizeBytes += info.Size()
			}
		}
	}

	return result, nil
}

// backupManual creates a manual backup by copying files.
func (p *PiholeBackup) backupManual(ctx context.Context, outputDir string, result *BackupResult) (*BackupResult, error) {
	p.logger.Info().Msg("creating Pi-hole backup manually")

	var totalSize int64

	// Backup gravity.db (blocklists)
	gravityDB := filepath.Join(p.ConfigDir, "gravity.db")
	if p.ConfigDir == "" {
		gravityDB = defaultGravityDB
	}
	if _, err := os.Stat(gravityDB); err == nil {
		dest := filepath.Join(outputDir, "gravity.db")
		if err := copyFile(gravityDB, dest); err != nil {
			result.ErrorMessage = fmt.Sprintf("backup gravity.db: %v", err)
			return result, fmt.Errorf("backup gravity.db: %w", err)
		}
		result.BackupFiles = append(result.BackupFiles, dest)
		if info, err := os.Stat(dest); err == nil {
			totalSize += info.Size()
		}
	}

	// Backup FTL database (query logs) if requested
	if p.IncludeQueryLogs {
		ftlBackup, err := p.backupFTLDB(ctx, outputDir)
		if err != nil {
			p.logger.Warn().Err(err).Msg("failed to backup FTL database")
		} else if ftlBackup != "" {
			result.BackupFiles = append(result.BackupFiles, ftlBackup)
			if info, err := os.Stat(ftlBackup); err == nil {
				totalSize += info.Size()
			}
		}
	}

	// Backup /etc/pihole/ configs
	configDir := p.ConfigDir
	if configDir == "" {
		configDir = defaultPiholeConfigDir
	}
	configFiles := []string{
		"setupVars.conf",
		"pihole-FTL.conf",
		"custom.list",
		"adlists.list",
		"regex.list",
		"whitelist.txt",
		"blacklist.txt",
		"local.list",
		"dhcp.leases",
	}
	configBackupDir := filepath.Join(outputDir, "pihole")
	if err := os.MkdirAll(configBackupDir, 0755); err != nil {
		result.ErrorMessage = fmt.Sprintf("create config backup dir: %v", err)
		return result, fmt.Errorf("create config backup dir: %w", err)
	}

	for _, file := range configFiles {
		src := filepath.Join(configDir, file)
		if _, err := os.Stat(src); err == nil {
			dest := filepath.Join(configBackupDir, file)
			if err := copyFile(src, dest); err != nil {
				p.logger.Warn().Err(err).Str("file", file).Msg("failed to backup config file")
				continue
			}
			result.BackupFiles = append(result.BackupFiles, dest)
			if info, err := os.Stat(dest); err == nil {
				totalSize += info.Size()
			}
		}
	}

	// Backup /etc/dnsmasq.d/ custom configs
	dnsmasqDir := p.DNSMasqDir
	if dnsmasqDir == "" {
		dnsmasqDir = defaultDNSMasqConfigDir
	}
	if entries, err := os.ReadDir(dnsmasqDir); err == nil {
		dnsmasqBackupDir := filepath.Join(outputDir, "dnsmasq.d")
		if err := os.MkdirAll(dnsmasqBackupDir, 0755); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				src := filepath.Join(dnsmasqDir, entry.Name())
				dest := filepath.Join(dnsmasqBackupDir, entry.Name())
				if err := copyFile(src, dest); err != nil {
					p.logger.Warn().Err(err).Str("file", entry.Name()).Msg("failed to backup dnsmasq config")
					continue
				}
				result.BackupFiles = append(result.BackupFiles, dest)
				if info, err := os.Stat(dest); err == nil {
					totalSize += info.Size()
				}
			}
		}
	}

	result.Success = true
	result.BackupPath = outputDir
	result.SizeBytes = totalSize

	p.logger.Info().
		Str("backup_dir", outputDir).
		Int("file_count", len(result.BackupFiles)).
		Int64("total_size", totalSize).
		Msg("Pi-hole manual backup completed")

	return result, nil
}

// backupFTLDB creates a backup of the FTL database.
func (p *PiholeBackup) backupFTLDB(ctx context.Context, outputDir string) (string, error) {
	ftlDB := filepath.Join(p.ConfigDir, "pihole-FTL.db")
	if p.ConfigDir == "" {
		ftlDB = defaultFTLDB
	}

	if _, err := os.Stat(ftlDB); err != nil {
		return "", nil // FTL DB doesn't exist, not an error
	}

	// Copy the database (it may be locked, so we use sqlite3 backup command if available)
	dest := filepath.Join(outputDir, "pihole-FTL.db")

	// Try using sqlite3 .backup command for safe copy
	sqlite3, err := exec.LookPath("sqlite3")
	if err == nil {
		cmd := exec.CommandContext(ctx, sqlite3, ftlDB, fmt.Sprintf(".backup '%s'", dest))
		if err := cmd.Run(); err == nil {
			p.logger.Debug().Str("file", dest).Msg("FTL database backed up using sqlite3")
			return dest, nil
		}
	}

	// Fall back to regular copy
	if err := copyFile(ftlDB, dest); err != nil {
		return "", fmt.Errorf("copy FTL database: %w", err)
	}

	return dest, nil
}

// Restore restores Pi-hole from a backup.
func (p *PiholeBackup) Restore(ctx context.Context, backupPath string) (*RestoreResult, error) {
	result := &RestoreResult{
		Success:       false,
		RestoredFiles: make([]string, 0),
		RestartNeeded: true,
	}

	// Check if backup exists
	info, err := os.Stat(backupPath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("backup not found: %v", err)
		return result, fmt.Errorf("backup not found: %w", err)
	}

	// If it's a teleporter archive (.tar.gz), use pihole -a -t to restore
	if !info.IsDir() && (strings.HasSuffix(backupPath, ".tar.gz") || strings.HasSuffix(backupPath, ".tgz")) {
		return p.restoreWithTeleporter(ctx, backupPath, result)
	}

	return p.restoreManual(ctx, backupPath, result)
}

// restoreWithTeleporter uses pihole teleporter to restore from archive.
func (p *PiholeBackup) restoreWithTeleporter(ctx context.Context, backupPath string, result *RestoreResult) (*RestoreResult, error) {
	p.logger.Info().Str("backup", backupPath).Msg("restoring Pi-hole using teleporter")

	ctx, cancel := context.WithTimeout(ctx, teleporterTimeout)
	defer cancel()

	binary := p.PiholeBinary
	if binary == "" {
		binary = defaultPiholeBinary
	}

	// Restore using teleporter
	// pihole -a -r <file> imports the teleporter backup
	cmd := exec.CommandContext(ctx, binary, "-a", "-r", backupPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("teleporter restore failed: %v: %s", err, string(output))
		return result, fmt.Errorf("teleporter restore: %w", err)
	}

	result.Success = true
	result.RestartNeeded = true
	result.ServiceRestart = true
	result.RestoredFiles = append(result.RestoredFiles, backupPath)

	p.logger.Info().Str("backup", backupPath).Msg("Pi-hole teleporter restore completed")

	// Restart Pi-hole services
	if err := p.restartServices(ctx); err != nil {
		p.logger.Warn().Err(err).Msg("failed to restart Pi-hole services")
	}

	return result, nil
}

// restoreManual restores Pi-hole from a manual backup directory.
func (p *PiholeBackup) restoreManual(ctx context.Context, backupPath string, result *RestoreResult) (*RestoreResult, error) {
	p.logger.Info().Str("backup", backupPath).Msg("restoring Pi-hole manually")

	configDir := p.ConfigDir
	if configDir == "" {
		configDir = defaultPiholeConfigDir
	}

	dnsmasqDir := p.DNSMasqDir
	if dnsmasqDir == "" {
		dnsmasqDir = defaultDNSMasqConfigDir
	}

	// Restore gravity.db
	gravityBackup := filepath.Join(backupPath, "gravity.db")
	if _, err := os.Stat(gravityBackup); err == nil {
		dest := filepath.Join(configDir, "gravity.db")
		if err := copyFile(gravityBackup, dest); err != nil {
			p.logger.Warn().Err(err).Msg("failed to restore gravity.db")
		} else {
			result.RestoredFiles = append(result.RestoredFiles, dest)
		}
	}

	// Restore pihole-FTL.db
	ftlBackup := filepath.Join(backupPath, "pihole-FTL.db")
	if _, err := os.Stat(ftlBackup); err == nil {
		dest := filepath.Join(configDir, "pihole-FTL.db")
		if err := copyFile(ftlBackup, dest); err != nil {
			p.logger.Warn().Err(err).Msg("failed to restore pihole-FTL.db")
		} else {
			result.RestoredFiles = append(result.RestoredFiles, dest)
		}
	}

	// Restore /etc/pihole/ configs
	piholeBackupDir := filepath.Join(backupPath, "pihole")
	if entries, err := os.ReadDir(piholeBackupDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := filepath.Join(piholeBackupDir, entry.Name())
			dest := filepath.Join(configDir, entry.Name())
			if err := copyFile(src, dest); err != nil {
				p.logger.Warn().Err(err).Str("file", entry.Name()).Msg("failed to restore config file")
				continue
			}
			result.RestoredFiles = append(result.RestoredFiles, dest)
		}
	}

	// Restore /etc/dnsmasq.d/ configs
	dnsmasqBackupDir := filepath.Join(backupPath, "dnsmasq.d")
	if entries, err := os.ReadDir(dnsmasqBackupDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			src := filepath.Join(dnsmasqBackupDir, entry.Name())
			dest := filepath.Join(dnsmasqDir, entry.Name())
			if err := copyFile(src, dest); err != nil {
				p.logger.Warn().Err(err).Str("file", entry.Name()).Msg("failed to restore dnsmasq config")
				continue
			}
			result.RestoredFiles = append(result.RestoredFiles, dest)
		}
	}

	result.Success = true
	result.RestartNeeded = true
	result.ServiceRestart = true

	p.logger.Info().
		Int("files_restored", len(result.RestoredFiles)).
		Msg("Pi-hole manual restore completed")

	// Restart services
	if err := p.restartServices(ctx); err != nil {
		p.logger.Warn().Err(err).Msg("failed to restart Pi-hole services")
	}

	// Update gravity
	if err := p.updateGravity(ctx); err != nil {
		p.logger.Warn().Err(err).Msg("failed to update gravity")
	}

	return result, nil
}

// restartServices restarts Pi-hole services after restore.
func (p *PiholeBackup) restartServices(ctx context.Context) error {
	binary := p.PiholeBinary
	if binary == "" {
		binary = defaultPiholeBinary
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "restartdns")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart dns: %w: %s", err, string(output))
	}

	p.logger.Info().Msg("Pi-hole DNS restarted")
	return nil
}

// updateGravity runs pihole -g to update gravity database.
func (p *PiholeBackup) updateGravity(ctx context.Context) error {
	binary := p.PiholeBinary
	if binary == "" {
		binary = defaultPiholeBinary
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, "-g")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update gravity: %w: %s", err, string(output))
	}

	p.logger.Info().Msg("Pi-hole gravity updated")
	return nil
}

// GetBackupPaths returns the default paths that should be backed up.
func (p *PiholeBackup) GetBackupPaths() []string {
	configDir := p.ConfigDir
	if configDir == "" {
		configDir = defaultPiholeConfigDir
	}
	dnsmasqDir := p.DNSMasqDir
	if dnsmasqDir == "" {
		dnsmasqDir = defaultDNSMasqConfigDir
	}

	paths := []string{
		configDir,
		dnsmasqDir,
	}

	return paths
}

// Validate checks if Pi-hole is installed and accessible.
func (p *PiholeBackup) Validate(ctx context.Context) error {
	info, err := p.Detect(ctx)
	if err != nil {
		return fmt.Errorf("detect Pi-hole: %w", err)
	}
	if !info.Installed {
		return errors.New("pi-hole is not installed")
	}

	// Check config directory
	configDir := p.ConfigDir
	if configDir == "" {
		configDir = defaultPiholeConfigDir
	}
	if _, err := os.Stat(configDir); err != nil {
		return fmt.Errorf("pi-hole config directory not accessible: %w", err)
	}

	return nil
}

// GetStatistics returns Pi-hole statistics from the API.
func (p *PiholeBackup) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Read setupVars.conf to get web password hash and other info
	configDir := p.ConfigDir
	if configDir == "" {
		configDir = defaultPiholeConfigDir
	}
	setupVars := filepath.Join(configDir, "setupVars.conf")
	if file, err := os.Open(setupVars); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PIHOLE_INTERFACE=") {
				stats["interface"] = strings.TrimPrefix(line, "PIHOLE_INTERFACE=")
			} else if strings.HasPrefix(line, "IPV4_ADDRESS=") {
				stats["ipv4_address"] = strings.TrimPrefix(line, "IPV4_ADDRESS=")
			} else if strings.HasPrefix(line, "BLOCKING_ENABLED=") {
				stats["blocking_enabled"] = strings.TrimPrefix(line, "BLOCKING_ENABLED=") == "true"
			}
		}
	}

	// Get info from pihole CLI
	binary := p.PiholeBinary
	if binary == "" {
		binary = defaultPiholeBinary
	}

	// Get status
	cmd := exec.CommandContext(ctx, binary, "status")
	if output, err := cmd.Output(); err == nil {
		stats["status"] = strings.TrimSpace(string(output))
	}

	// Get version info
	if version, err := p.getVersion(ctx, binary); err == nil {
		stats["version"] = version
	}

	return stats, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Get source file info for permissions
	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	// Copy contents
	buf := make([]byte, 32*1024)
	for {
		n, err := sourceFile.Read(buf)
		if n > 0 {
			if _, writeErr := destFile.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}
	}

	// Set permissions
	return os.Chmod(dst, info.Mode())
}
