// Package databases provides database-specific backup implementations.
package databases

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

const (
	// Default PostgreSQL settings
	defaultPgDumpBinary    = "pg_dump"
	defaultPgDumpAllBinary = "pg_dumpall"
	defaultPsqlBinary      = "psql"
	defaultPostgresPort    = 5432
	defaultPgBackupTimeout = 30 * time.Minute
	defaultPgConnTimeout   = 30 * time.Second
)

// PostgresBackup implements database backup for PostgreSQL using pg_dump.
type PostgresBackup struct {
	// Config contains the PostgreSQL backup configuration.
	Config *models.PostgresBackupConfig

	// DecryptedPassword is the decrypted database password (set before backup).
	DecryptedPassword string

	logger zerolog.Logger
}

// DatabaseInfo contains information about a detected PostgreSQL server.
type DatabaseInfo struct {
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Version   string   `json:"version,omitempty"`
	Connected bool     `json:"connected"`
	Databases []string `json:"databases,omitempty"`
}

// PostgresBackupResult contains the result of a PostgreSQL backup operation.
type PostgresBackupResult struct {
	Success      bool     `json:"success"`
	BackupPath   string   `json:"backup_path,omitempty"`
	BackupFiles  []string `json:"backup_files,omitempty"`
	SizeBytes    int64    `json:"size_bytes,omitempty"`
	Duration     string   `json:"duration,omitempty"`
	DatabaseName string   `json:"database_name,omitempty"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// PostgresRestoreResult contains the result of a PostgreSQL restore operation.
type PostgresRestoreResult struct {
	Success       bool   `json:"success"`
	DatabaseName  string `json:"database_name,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	Duration      string `json:"duration,omitempty"`
}

// RestoreInstructions provides guidance on how to restore a PostgreSQL backup.
type RestoreInstructions struct {
	Format       string   `json:"format"`
	Instructions []string `json:"instructions"`
	Commands     []string `json:"commands"`
	Notes        []string `json:"notes,omitempty"`
}

// NewPostgresBackup creates a new PostgresBackup with the given configuration.
func NewPostgresBackup(config *models.PostgresBackupConfig, logger zerolog.Logger) *PostgresBackup {
	if config == nil {
		config = models.DefaultPostgresConfig()
	}
	if config.Port == 0 {
		config.Port = defaultPostgresPort
	}
	if config.OutputFormat == "" {
		config.OutputFormat = models.PostgresFormatCustom
	}

	return &PostgresBackup{
		Config: config,
		logger: logger.With().Str("component", "postgres_backup").Logger(),
	}
}

// TestConnection tests the PostgreSQL connection with the configured credentials.
func (p *PostgresBackup) TestConnection(ctx context.Context) (*DatabaseInfo, error) {
	info := &DatabaseInfo{
		Host:      p.Config.Host,
		Port:      p.Config.Port,
		Connected: false,
	}

	ctx, cancel := context.WithTimeout(ctx, defaultPgConnTimeout)
	defer cancel()

	// Build connection string for psql
	connStr := p.buildConnectionString()

	// Test connection using psql
	psqlBinary := p.findBinary(defaultPsqlBinary)
	cmd := exec.CommandContext(ctx, psqlBinary, connStr, "-c", "SELECT version();")
	cmd.Env = p.buildEnvironment()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return info, fmt.Errorf("connection failed: %w: %s", err, string(output))
	}

	info.Connected = true

	// Parse version from output
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "PostgreSQL") {
			info.Version = strings.TrimSpace(line)
			break
		}
	}

	// List databases
	databases, err := p.listDatabases(ctx)
	if err != nil {
		p.logger.Warn().Err(err).Msg("failed to list databases")
	} else {
		info.Databases = databases
	}

	p.logger.Info().
		Str("host", p.Config.Host).
		Int("port", p.Config.Port).
		Str("version", info.Version).
		Int("databases", len(info.Databases)).
		Msg("PostgreSQL connection test successful")

	return info, nil
}

// listDatabases returns a list of all databases on the server.
func (p *PostgresBackup) listDatabases(ctx context.Context) ([]string, error) {
	connStr := p.buildConnectionString()
	psqlBinary := p.findBinary(defaultPsqlBinary)

	cmd := exec.CommandContext(ctx, psqlBinary, connStr, "-t", "-c",
		"SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname;")
	cmd.Env = p.buildEnvironment()

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}

	var databases []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		db := strings.TrimSpace(line)
		if db != "" {
			databases = append(databases, db)
		}
	}

	return databases, nil
}

// Backup creates a backup of the PostgreSQL database(s).
func (p *PostgresBackup) Backup(ctx context.Context, outputDir string) (*PostgresBackupResult, error) {
	startTime := time.Now()
	result := &PostgresBackupResult{
		Success:     false,
		BackupFiles: make([]string, 0),
	}

	// Validate configuration
	if err := p.Validate(); err != nil {
		result.ErrorMessage = err.Error()
		return result, err
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.ErrorMessage = fmt.Sprintf("create output directory: %v", err)
		return result, fmt.Errorf("create output directory: %w", err)
	}

	// Determine which databases to backup
	if p.Config.Database != "" {
		// Single database backup
		return p.backupSingleDatabase(ctx, outputDir, p.Config.Database, result, startTime)
	} else if len(p.Config.Databases) > 0 {
		// Multiple specific databases
		return p.backupMultipleDatabases(ctx, outputDir, p.Config.Databases, result, startTime)
	} else {
		// All databases using pg_dumpall
		return p.backupAllDatabases(ctx, outputDir, result, startTime)
	}
}

// backupSingleDatabase backs up a single database using pg_dump.
func (p *PostgresBackup) backupSingleDatabase(ctx context.Context, outputDir, database string, result *PostgresBackupResult, startTime time.Time) (*PostgresBackupResult, error) {
	p.logger.Info().
		Str("host", p.Config.Host).
		Str("database", database).
		Msg("starting PostgreSQL backup")

	ctx, cancel := context.WithTimeout(ctx, defaultPgBackupTimeout)
	defer cancel()

	// Generate backup filename
	timestamp := time.Now().Format("20060102-150405")
	extension := p.getFileExtension()
	backupFile := filepath.Join(outputDir, fmt.Sprintf("postgres_%s_%s%s", database, timestamp, extension))

	// Build pg_dump command
	args := p.buildPgDumpArgs(database, backupFile)
	pgDumpBinary := p.findBinary(defaultPgDumpBinary)

	p.logger.Debug().
		Str("binary", pgDumpBinary).
		Strs("args", args).
		Msg("executing pg_dump")

	cmd := exec.CommandContext(ctx, pgDumpBinary, args...)
	cmd.Env = p.buildEnvironment()

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("pg_dump failed: %v: %s", err, string(output))
		return result, fmt.Errorf("pg_dump: %w", err)
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
	result.DatabaseName = database
	result.Duration = time.Since(startTime).String()

	p.logger.Info().
		Str("backup_file", backupFile).
		Int64("size_bytes", info.Size()).
		Str("duration", result.Duration).
		Msg("PostgreSQL backup completed")

	return result, nil
}

// backupMultipleDatabases backs up multiple specific databases.
func (p *PostgresBackup) backupMultipleDatabases(ctx context.Context, outputDir string, databases []string, result *PostgresBackupResult, startTime time.Time) (*PostgresBackupResult, error) {
	var totalSize int64
	var lastErr error

	for _, database := range databases {
		dbResult, err := p.backupSingleDatabase(ctx, outputDir, database, &PostgresBackupResult{}, startTime)
		if err != nil {
			p.logger.Error().Err(err).Str("database", database).Msg("failed to backup database")
			lastErr = err
			continue
		}

		result.BackupFiles = append(result.BackupFiles, dbResult.BackupFiles...)
		totalSize += dbResult.SizeBytes
	}

	if len(result.BackupFiles) == 0 {
		result.ErrorMessage = "no databases were backed up successfully"
		return result, lastErr
	}

	result.Success = true
	result.BackupPath = outputDir
	result.SizeBytes = totalSize
	result.Duration = time.Since(startTime).String()

	return result, nil
}

// backupAllDatabases backs up all databases using pg_dumpall.
func (p *PostgresBackup) backupAllDatabases(ctx context.Context, outputDir string, result *PostgresBackupResult, startTime time.Time) (*PostgresBackupResult, error) {
	p.logger.Info().
		Str("host", p.Config.Host).
		Msg("starting PostgreSQL backup of all databases")

	ctx, cancel := context.WithTimeout(ctx, defaultPgBackupTimeout)
	defer cancel()

	// Generate backup filename
	timestamp := time.Now().Format("20060102-150405")
	backupFile := filepath.Join(outputDir, fmt.Sprintf("postgres_all_%s.sql", timestamp))

	// Build pg_dumpall command
	args := p.buildPgDumpAllArgs()
	pgDumpAllBinary := p.findBinary(defaultPgDumpAllBinary)

	p.logger.Debug().
		Str("binary", pgDumpAllBinary).
		Strs("args", args).
		Msg("executing pg_dumpall")

	// Create output file
	outFile, err := os.Create(backupFile)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("create backup file: %v", err)
		return result, fmt.Errorf("create backup file: %w", err)
	}
	defer outFile.Close()

	cmd := exec.CommandContext(ctx, pgDumpAllBinary, args...)
	cmd.Env = p.buildEnvironment()
	cmd.Stdout = outFile

	stderrOutput, err := cmd.StderrPipe()
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("setup stderr: %v", err)
		return result, fmt.Errorf("setup stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		result.ErrorMessage = fmt.Sprintf("start pg_dumpall: %v", err)
		return result, fmt.Errorf("start pg_dumpall: %w", err)
	}

	// Read stderr for any errors
	stderrBytes := make([]byte, 4096)
	n, _ := stderrOutput.Read(stderrBytes)
	stderr := string(stderrBytes[:n])

	if err := cmd.Wait(); err != nil {
		result.ErrorMessage = fmt.Sprintf("pg_dumpall failed: %v: %s", err, stderr)
		return result, fmt.Errorf("pg_dumpall: %w", err)
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
	result.DatabaseName = "all"
	result.Duration = time.Since(startTime).String()

	p.logger.Info().
		Str("backup_file", backupFile).
		Int64("size_bytes", info.Size()).
		Str("duration", result.Duration).
		Msg("PostgreSQL backup of all databases completed")

	return result, nil
}

// buildPgDumpArgs builds the command line arguments for pg_dump.
func (p *PostgresBackup) buildPgDumpArgs(database, outputFile string) []string {
	args := []string{
		"-h", p.Config.Host,
		"-p", strconv.Itoa(p.Config.Port),
		"-U", p.Config.Username,
		"-d", database,
	}

	// Output format
	switch p.Config.OutputFormat {
	case models.PostgresFormatCustom:
		args = append(args, "-F", "c")
	case models.PostgresFormatDirectory:
		args = append(args, "-F", "d")
	case models.PostgresFormatTar:
		args = append(args, "-F", "t")
	case models.PostgresFormatPlain:
		args = append(args, "-F", "p")
	}

	// Compression (only for custom and directory formats)
	if p.Config.CompressionLevel > 0 && (p.Config.OutputFormat == models.PostgresFormatCustom ||
		p.Config.OutputFormat == models.PostgresFormatDirectory) {
		args = append(args, "-Z", strconv.Itoa(p.Config.CompressionLevel))
	}

	// Schema/data only options
	if p.Config.IncludeSchemaOnly {
		args = append(args, "-s")
	}
	if p.Config.IncludeDataOnly {
		args = append(args, "-a")
	}

	// Owner and privileges
	if p.Config.NoOwner {
		args = append(args, "-O")
	}
	if p.Config.NoPrivileges {
		args = append(args, "-x")
	}

	// Table filters
	for _, table := range p.Config.ExcludeTables {
		args = append(args, "-T", table)
	}
	for _, table := range p.Config.IncludeTables {
		args = append(args, "-t", table)
	}

	// Output file
	args = append(args, "-f", outputFile)

	return args
}

// buildPgDumpAllArgs builds the command line arguments for pg_dumpall.
func (p *PostgresBackup) buildPgDumpAllArgs() []string {
	args := []string{
		"-h", p.Config.Host,
		"-p", strconv.Itoa(p.Config.Port),
		"-U", p.Config.Username,
	}

	// Schema/data only options
	if p.Config.IncludeSchemaOnly {
		args = append(args, "-s")
	}
	if p.Config.IncludeDataOnly {
		args = append(args, "-a")
	}

	// Owner and privileges
	if p.Config.NoOwner {
		args = append(args, "-O")
	}
	if p.Config.NoPrivileges {
		args = append(args, "-x")
	}

	return args
}

// buildConnectionString builds a PostgreSQL connection string for psql.
func (p *PostgresBackup) buildConnectionString() string {
	database := p.Config.Database
	if database == "" {
		database = "postgres"
	}

	return fmt.Sprintf("postgresql://%s@%s:%d/%s",
		p.Config.Username,
		p.Config.Host,
		p.Config.Port,
		database,
	)
}

// buildEnvironment builds the environment variables for PostgreSQL commands.
func (p *PostgresBackup) buildEnvironment() []string {
	env := os.Environ()

	// Add password if available
	if p.DecryptedPassword != "" {
		env = append(env, "PGPASSWORD="+p.DecryptedPassword)
	}

	// Add SSL mode if specified
	if p.Config.SSLMode != "" {
		env = append(env, "PGSSLMODE="+p.Config.SSLMode)
	}

	return env
}

// getFileExtension returns the appropriate file extension for the output format.
func (p *PostgresBackup) getFileExtension() string {
	switch p.Config.OutputFormat {
	case models.PostgresFormatCustom:
		return ".dump"
	case models.PostgresFormatDirectory:
		return "" // Directory format doesn't use extension
	case models.PostgresFormatTar:
		return ".tar"
	default:
		return ".sql"
	}
}

// findBinary locates the PostgreSQL binary, checking config override first.
func (p *PostgresBackup) findBinary(defaultBinary string) string {
	if p.Config.PgDumpPath != "" && strings.Contains(defaultBinary, "pg_dump") {
		return p.Config.PgDumpPath
	}

	// Try to find in PATH
	path, err := exec.LookPath(defaultBinary)
	if err == nil {
		return path
	}

	// Common PostgreSQL installation paths
	commonPaths := []string{
		"/usr/bin/" + defaultBinary,
		"/usr/local/bin/" + defaultBinary,
		"/usr/local/pgsql/bin/" + defaultBinary,
		"/opt/homebrew/bin/" + defaultBinary,
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	return defaultBinary
}

// Validate checks if the PostgreSQL configuration is valid.
func (p *PostgresBackup) Validate() error {
	if p.Config == nil {
		return errors.New("postgres configuration is required")
	}
	if p.Config.Host == "" {
		return errors.New("postgres host is required")
	}
	if p.Config.Username == "" {
		return errors.New("postgres username is required")
	}
	if p.Config.IncludeSchemaOnly && p.Config.IncludeDataOnly {
		return errors.New("cannot specify both schema-only and data-only options")
	}

	// Verify pg_dump is available
	pgDumpBinary := p.findBinary(defaultPgDumpBinary)
	if _, err := exec.LookPath(pgDumpBinary); err != nil {
		return fmt.Errorf("pg_dump not found: %w", err)
	}

	return nil
}

// GetRestoreInstructions returns instructions for restoring a PostgreSQL backup.
func (p *PostgresBackup) GetRestoreInstructions(backupPath string) *RestoreInstructions {
	format := p.Config.OutputFormat
	if format == "" {
		format = models.PostgresFormatCustom
	}

	instructions := &RestoreInstructions{
		Format: string(format),
	}

	switch format {
	case models.PostgresFormatCustom:
		instructions.Instructions = []string{
			"To restore from a custom format backup, use pg_restore:",
			"1. Create the target database if it doesn't exist",
			"2. Use pg_restore to restore the backup",
		}
		instructions.Commands = []string{
			"createdb -h <host> -U <user> <database_name>",
			"pg_restore -h <host> -U <user> -d <database_name> -v " + backupPath,
		}
		instructions.Notes = []string{
			"Add --clean to drop existing objects before restoring",
			"Add --no-owner to skip ownership restoration",
			"Add --jobs=N for parallel restore (only with directory format)",
		}

	case models.PostgresFormatDirectory:
		instructions.Instructions = []string{
			"To restore from a directory format backup, use pg_restore with parallel option:",
			"1. Create the target database if it doesn't exist",
			"2. Use pg_restore with --jobs for faster parallel restore",
		}
		instructions.Commands = []string{
			"createdb -h <host> -U <user> <database_name>",
			"pg_restore -h <host> -U <user> -d <database_name> --jobs=4 -v " + backupPath,
		}

	case models.PostgresFormatTar:
		instructions.Instructions = []string{
			"To restore from a tar format backup, use pg_restore:",
			"1. Create the target database if it doesn't exist",
			"2. Use pg_restore to restore from the tar archive",
		}
		instructions.Commands = []string{
			"createdb -h <host> -U <user> <database_name>",
			"pg_restore -h <host> -U <user> -d <database_name> -v " + backupPath,
		}

	case models.PostgresFormatPlain:
		instructions.Instructions = []string{
			"To restore from a plain SQL backup, use psql:",
			"1. Create the target database if it doesn't exist",
			"2. Use psql to execute the SQL file",
		}
		instructions.Commands = []string{
			"createdb -h <host> -U <user> <database_name>",
			"psql -h <host> -U <user> -d <database_name> -f " + backupPath,
		}
		instructions.Notes = []string{
			"For compressed SQL files (.sql.gz), decompress first: gunzip -c backup.sql.gz | psql ...",
		}
	}

	return instructions
}
