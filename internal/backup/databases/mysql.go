// Package databases provides database-specific backup implementations.
package databases

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog"
)

const (
	// Default mysqldump timeout.
	defaultDumpTimeout = 30 * time.Minute

	// Default connection timeout.
	defaultConnectTimeout = 30 * time.Second
)

var (
	// ErrMySQLDumpNotFound is returned when mysqldump is not installed.
	ErrMySQLDumpNotFound = errors.New("mysqldump binary not found")

	// ErrConnectionFailed is returned when unable to connect to MySQL.
	ErrConnectionFailed = errors.New("failed to connect to MySQL server")

	// ErrBackupFailed is returned when the backup operation fails.
	ErrBackupFailed = errors.New("backup failed")
)

// MySQLConfig contains MySQL/MariaDB connection configuration.
type MySQLConfig struct {
	// Host is the MySQL server hostname or IP address.
	Host string `json:"host"`
	// Port is the MySQL server port (default: 3306).
	Port int `json:"port"`
	// Username for MySQL authentication.
	Username string `json:"username"`
	// Password for MySQL authentication.
	Password string `json:"password"`
	// Database is a specific database to backup. Empty means all databases.
	Database string `json:"database,omitempty"`
	// Databases is a list of specific databases to backup.
	Databases []string `json:"databases,omitempty"`
	// ExcludeDatabases are databases to exclude from "all databases" backup.
	ExcludeDatabases []string `json:"exclude_databases,omitempty"`
	// SSLMode controls SSL/TLS connection settings.
	SSLMode string `json:"ssl_mode,omitempty"`
	// Compress enables gzip compression of the backup output.
	Compress bool `json:"compress"`
	// DumpTimeout is the maximum time allowed for mysqldump to run.
	DumpTimeout time.Duration `json:"dump_timeout,omitempty"`
	// ConnectTimeout is the maximum time to wait for connection.
	ConnectTimeout time.Duration `json:"connect_timeout,omitempty"`
	// MySQLDumpPath overrides the default mysqldump binary path.
	MySQLDumpPath string `json:"mysqldump_path,omitempty"`
	// ExtraArgs are additional arguments to pass to mysqldump.
	ExtraArgs []string `json:"extra_args,omitempty"`
}

// DefaultMySQLConfig returns a MySQLConfig with sensible defaults.
func DefaultMySQLConfig() *MySQLConfig {
	return &MySQLConfig{
		Host:           "localhost",
		Port:           3306,
		Compress:       true,
		DumpTimeout:    defaultDumpTimeout,
		ConnectTimeout: defaultConnectTimeout,
	}
}

// DSN returns the MySQL Data Source Name for database/sql connection.
func (c *MySQLConfig) DSN() string {
	// Format: [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...]
	var dsn strings.Builder

	if c.Username != "" {
		dsn.WriteString(c.Username)
		if c.Password != "" {
			dsn.WriteString(":")
			dsn.WriteString(c.Password)
		}
		dsn.WriteString("@")
	}

	dsn.WriteString(fmt.Sprintf("tcp(%s:%d)/", c.Host, c.Port))

	if c.Database != "" {
		dsn.WriteString(c.Database)
	}

	// Add connection parameters
	params := []string{}

	timeout := c.ConnectTimeout
	if timeout == 0 {
		timeout = defaultConnectTimeout
	}
	params = append(params, fmt.Sprintf("timeout=%s", timeout.String()))

	if c.SSLMode != "" {
		params = append(params, fmt.Sprintf("tls=%s", c.SSLMode))
	}

	if len(params) > 0 {
		dsn.WriteString("?")
		dsn.WriteString(strings.Join(params, "&"))
	}

	return dsn.String()
}

// BackupResult contains the result of a MySQL backup operation.
type BackupResult struct {
	Success       bool     `json:"success"`
	BackupPath    string   `json:"backup_path,omitempty"`
	DatabaseNames []string `json:"database_names,omitempty"`
	SizeBytes     int64    `json:"size_bytes,omitempty"`
	Duration      time.Duration `json:"duration"`
	ErrorMessage  string   `json:"error_message,omitempty"`
	Compressed    bool     `json:"compressed"`
}

// ConnectionTestResult contains the result of a connection test.
type ConnectionTestResult struct {
	Success      bool          `json:"success"`
	Version      string        `json:"version,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorMessage string        `json:"error_message,omitempty"`
	Databases    []string      `json:"databases,omitempty"`
}

// MySQLBackup provides MySQL/MariaDB backup functionality using mysqldump.
type MySQLBackup struct {
	config *MySQLConfig
	logger zerolog.Logger
}

// NewMySQLBackup creates a new MySQLBackup with the given configuration.
func NewMySQLBackup(config *MySQLConfig, logger zerolog.Logger) *MySQLBackup {
	if config == nil {
		config = DefaultMySQLConfig()
	}
	if config.Port == 0 {
		config.Port = 3306
	}
	if config.DumpTimeout == 0 {
		config.DumpTimeout = defaultDumpTimeout
	}
	if config.ConnectTimeout == 0 {
		config.ConnectTimeout = defaultConnectTimeout
	}

	return &MySQLBackup{
		config: config,
		logger: logger.With().Str("component", "mysql_backup").Logger(),
	}
}

// TestConnection tests the MySQL connection and returns server info.
func (m *MySQLBackup) TestConnection(ctx context.Context) (*ConnectionTestResult, error) {
	m.logger.Debug().
		Str("host", m.config.Host).
		Int("port", m.config.Port).
		Str("username", m.config.Username).
		Msg("testing MySQL connection")

	start := time.Now()
	result := &ConnectionTestResult{
		Success: false,
	}

	// Connect to MySQL
	db, err := sql.Open("mysql", m.config.DSN())
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to open connection: %v", err)
		result.ResponseTime = time.Since(start)
		return result, fmt.Errorf("open connection: %w", err)
	}
	defer db.Close()

	// Set connection timeout
	ctx, cancel := context.WithTimeout(ctx, m.config.ConnectTimeout)
	defer cancel()

	// Ping to verify connection
	if err := db.PingContext(ctx); err != nil {
		result.ErrorMessage = fmt.Sprintf("connection failed: %v", err)
		result.ResponseTime = time.Since(start)
		return result, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	// Get MySQL version
	var version string
	if err := db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version); err != nil {
		m.logger.Warn().Err(err).Msg("failed to get MySQL version")
	} else {
		result.Version = version
	}

	// Get list of databases
	databases, err := m.listDatabases(ctx, db)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to list databases")
	} else {
		result.Databases = databases
	}

	result.Success = true
	result.ResponseTime = time.Since(start)

	m.logger.Info().
		Str("version", version).
		Int("databases", len(databases)).
		Dur("response_time", result.ResponseTime).
		Msg("MySQL connection test successful")

	return result, nil
}

// listDatabases returns a list of all databases on the server.
func (m *MySQLBackup) listDatabases(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var dbName string
		if err := rows.Scan(&dbName); err != nil {
			continue
		}
		databases = append(databases, dbName)
	}

	return databases, rows.Err()
}

// Backup performs a MySQL backup using mysqldump.
func (m *MySQLBackup) Backup(ctx context.Context, outputDir string) (*BackupResult, error) {
	m.logger.Info().
		Str("host", m.config.Host).
		Int("port", m.config.Port).
		Str("database", m.config.Database).
		Strs("databases", m.config.Databases).
		Bool("compress", m.config.Compress).
		Msg("starting MySQL backup")

	start := time.Now()
	result := &BackupResult{
		Success:    false,
		Compressed: m.config.Compress,
	}

	// Find mysqldump binary
	mysqldump, err := m.findMySQLDump()
	if err != nil {
		result.ErrorMessage = err.Error()
		result.Duration = time.Since(start)
		return result, err
	}

	// Create output directory if needed
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.ErrorMessage = fmt.Sprintf("create output directory: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("create output directory: %w", err)
	}

	// Determine which databases to backup
	var databases []string
	if m.config.Database != "" {
		databases = []string{m.config.Database}
	} else if len(m.config.Databases) > 0 {
		databases = m.config.Databases
	}

	// Build backup filename
	timestamp := time.Now().Format("20060102-150405")
	var filename string
	if len(databases) == 1 {
		filename = fmt.Sprintf("mysql_%s_%s.sql", databases[0], timestamp)
	} else {
		filename = fmt.Sprintf("mysql_all_%s.sql", timestamp)
	}
	if m.config.Compress {
		filename += ".gz"
	}
	backupPath := filepath.Join(outputDir, filename)

	// Build mysqldump command arguments
	args := m.buildDumpArgs(databases)

	m.logger.Debug().
		Str("mysqldump", mysqldump).
		Strs("args", m.sanitizeArgs(args)).
		Str("output", backupPath).
		Msg("executing mysqldump")

	// Set timeout context
	ctx, cancel := context.WithTimeout(ctx, m.config.DumpTimeout)
	defer cancel()

	// Execute mysqldump
	cmd := exec.CommandContext(ctx, mysqldump, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Create output file
	outFile, err := os.Create(backupPath)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("create output file: %v", err)
		result.Duration = time.Since(start)
		return result, fmt.Errorf("create output file: %w", err)
	}

	var writer io.WriteCloser = outFile
	if m.config.Compress {
		gzWriter := gzip.NewWriter(outFile)
		writer = gzWriter
		defer func() {
			gzWriter.Close()
			outFile.Close()
		}()
	} else {
		defer outFile.Close()
	}

	// Pipe mysqldump output to file (optionally through gzip)
	cmd.Stdout = writer

	if err := cmd.Run(); err != nil {
		// Clean up failed backup file
		os.Remove(backupPath)

		errMsg := stderr.String()
		if errMsg == "" {
			errMsg = err.Error()
		}
		result.ErrorMessage = fmt.Sprintf("mysqldump failed: %s", strings.TrimSpace(errMsg))
		result.Duration = time.Since(start)
		return result, fmt.Errorf("%w: %s", ErrBackupFailed, errMsg)
	}

	// Close writer to flush any buffered data
	if m.config.Compress {
		writer.Close()
	}

	// Get file size
	info, err := os.Stat(backupPath)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get backup file size")
	} else {
		result.SizeBytes = info.Size()
	}

	result.Success = true
	result.BackupPath = backupPath
	result.DatabaseNames = databases
	if len(databases) == 0 {
		result.DatabaseNames = []string{"all_databases"}
	}
	result.Duration = time.Since(start)

	m.logger.Info().
		Str("backup_path", backupPath).
		Int64("size_bytes", result.SizeBytes).
		Dur("duration", result.Duration).
		Bool("compressed", m.config.Compress).
		Msg("MySQL backup completed successfully")

	return result, nil
}

// buildDumpArgs constructs the mysqldump command arguments.
func (m *MySQLBackup) buildDumpArgs(databases []string) []string {
	args := []string{
		fmt.Sprintf("--host=%s", m.config.Host),
		fmt.Sprintf("--port=%d", m.config.Port),
		fmt.Sprintf("--user=%s", m.config.Username),
	}

	// Password via environment variable for security
	if m.config.Password != "" {
		args = append(args, fmt.Sprintf("--password=%s", m.config.Password))
	}

	// Add recommended options for consistent backups
	args = append(args,
		"--single-transaction",  // Consistent snapshot for InnoDB
		"--quick",               // Retrieve rows one at a time
		"--lock-tables=false",   // Don't lock tables (works with single-transaction)
		"--routines",            // Include stored procedures and functions
		"--triggers",            // Include triggers
		"--events",              // Include scheduled events
		"--set-gtid-purged=OFF", // Compatible with replication
	)

	// SSL/TLS configuration
	if m.config.SSLMode != "" {
		switch m.config.SSLMode {
		case "required", "verify-ca", "verify-identity":
			args = append(args, "--ssl-mode="+strings.ToUpper(m.config.SSLMode))
		}
	}

	// Add extra args if specified
	args = append(args, m.config.ExtraArgs...)

	// Specify databases
	if len(databases) == 0 {
		// Backup all databases
		args = append(args, "--all-databases")

		// Add ignore patterns for excluded databases
		for _, db := range m.config.ExcludeDatabases {
			args = append(args, fmt.Sprintf("--ignore-table=%s.*", db))
		}
	} else if len(databases) == 1 {
		// Single database
		args = append(args, databases[0])
	} else {
		// Multiple specific databases
		args = append(args, "--databases")
		args = append(args, databases...)
	}

	return args
}

// sanitizeArgs returns args with password masked for logging.
func (m *MySQLBackup) sanitizeArgs(args []string) []string {
	sanitized := make([]string, len(args))
	for i, arg := range args {
		if strings.HasPrefix(arg, "--password=") {
			sanitized[i] = "--password=***"
		} else {
			sanitized[i] = arg
		}
	}
	return sanitized
}

// findMySQLDump finds the mysqldump binary.
func (m *MySQLBackup) findMySQLDump() (string, error) {
	// Use configured path if specified
	if m.config.MySQLDumpPath != "" {
		if _, err := os.Stat(m.config.MySQLDumpPath); err == nil {
			return m.config.MySQLDumpPath, nil
		}
		return "", fmt.Errorf("mysqldump not found at %s", m.config.MySQLDumpPath)
	}

	// Try to find in PATH
	path, err := exec.LookPath("mysqldump")
	if err == nil {
		return path, nil
	}

	// Try common locations
	commonPaths := []string{
		"/usr/bin/mysqldump",
		"/usr/local/bin/mysqldump",
		"/usr/local/mysql/bin/mysqldump",
		"/opt/homebrew/bin/mysqldump",
		"/opt/mysql/bin/mysqldump",
		// MariaDB locations
		"/usr/bin/mariadb-dump",
		"/usr/local/bin/mariadb-dump",
	}

	for _, p := range commonPaths {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", ErrMySQLDumpNotFound
}

// GetRestoreInstructions returns instructions for restoring a MySQL backup.
func (m *MySQLBackup) GetRestoreInstructions(backupPath string) string {
	isCompressed := strings.HasSuffix(backupPath, ".gz")
	filename := filepath.Base(backupPath)

	var sb strings.Builder
	sb.WriteString("MySQL/MariaDB Restore Instructions\n")
	sb.WriteString("===================================\n\n")

	sb.WriteString(fmt.Sprintf("Backup File: %s\n\n", filename))

	sb.WriteString("IMPORTANT: Always test restore procedures on a non-production system first!\n\n")

	sb.WriteString("Option 1: Restore using mysql client\n")
	sb.WriteString("------------------------------------\n")
	if isCompressed {
		sb.WriteString(fmt.Sprintf("gunzip -c %s | mysql -h <host> -u <user> -p\n\n", filename))
	} else {
		sb.WriteString(fmt.Sprintf("mysql -h <host> -u <user> -p < %s\n\n", filename))
	}

	sb.WriteString("Option 2: Restore to a specific database\n")
	sb.WriteString("-----------------------------------------\n")
	if isCompressed {
		sb.WriteString(fmt.Sprintf("gunzip -c %s | mysql -h <host> -u <user> -p <database_name>\n\n", filename))
	} else {
		sb.WriteString(fmt.Sprintf("mysql -h <host> -u <user> -p <database_name> < %s\n\n", filename))
	}

	sb.WriteString("Option 3: View backup contents first\n")
	sb.WriteString("------------------------------------\n")
	if isCompressed {
		sb.WriteString(fmt.Sprintf("zcat %s | head -100\n", filename))
		sb.WriteString(fmt.Sprintf("zcat %s | grep 'CREATE DATABASE'\n\n", filename))
	} else {
		sb.WriteString(fmt.Sprintf("head -100 %s\n", filename))
		sb.WriteString(fmt.Sprintf("grep 'CREATE DATABASE' %s\n\n", filename))
	}

	sb.WriteString("Notes:\n")
	sb.WriteString("- Replace <host>, <user>, and <database_name> with your MySQL/MariaDB details\n")
	sb.WriteString("- You will be prompted for the password after running the command\n")
	sb.WriteString("- For full database restores, the user needs appropriate permissions\n")
	sb.WriteString("- Consider using --no-data for schema-only restore, or --no-create-info for data-only\n")

	return sb.String()
}

// ValidateMySQLDumpAvailable checks if mysqldump is available on the system.
func ValidateMySQLDumpAvailable() error {
	backup := &MySQLBackup{config: DefaultMySQLConfig()}
	_, err := backup.findMySQLDump()
	return err
}
