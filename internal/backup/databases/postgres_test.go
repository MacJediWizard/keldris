package databases

import (
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/rs/zerolog"
)

// testLogger returns a no-op logger for testing.
func testLogger() zerolog.Logger {
	return zerolog.Nop()
}

func TestPostgresBackup_BuildConnectionString(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.PostgresBackupConfig
		contains []string
	}{
		{
			name: "basic connection",
			config: &models.PostgresBackupConfig{
				Host:     "db.example.com",
				Port:     5432,
				Username: "postgres",
				Database: "mydb",
			},
			contains: []string{
				"postgresql://",
				"postgres@",
				"db.example.com",
				":5432/",
				"mydb",
			},
		},
		{
			name: "no database defaults to postgres",
			config: &models.PostgresBackupConfig{
				Host:     "localhost",
				Port:     5432,
				Username: "admin",
			},
			contains: []string{
				"postgresql://admin@localhost:5432/postgres",
			},
		},
		{
			name: "custom port",
			config: &models.PostgresBackupConfig{
				Host:     "localhost",
				Port:     5433,
				Username: "user",
				Database: "testdb",
			},
			contains: []string{
				":5433/",
				"testdb",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := NewPostgresBackup(tt.config, testLogger())
			connStr := backup.buildConnectionString()

			for _, expected := range tt.contains {
				if !strings.Contains(connStr, expected) {
					t.Errorf("connection string %q should contain %q", connStr, expected)
				}
			}
		})
	}
}

func TestPostgresBackup_BuildPgDumpArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.PostgresBackupConfig
		database string
		outFile  string
		contains []string
		excludes []string
	}{
		{
			name: "basic single database custom format",
			config: &models.PostgresBackupConfig{
				Host:         "db.example.com",
				Port:         5432,
				Username:     "backup_user",
				OutputFormat: models.PostgresFormatCustom,
			},
			database: "production",
			outFile:  "/backups/production.dump",
			contains: []string{
				"-h", "db.example.com",
				"-p", "5432",
				"-U", "backup_user",
				"-d", "production",
				"-F", "c",
				"-f", "/backups/production.dump",
			},
		},
		{
			name: "plain format",
			config: &models.PostgresBackupConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				OutputFormat: models.PostgresFormatPlain,
			},
			database: "testdb",
			outFile:  "/backups/testdb.sql",
			contains: []string{
				"-F", "p",
			},
		},
		{
			name: "tar format",
			config: &models.PostgresBackupConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				OutputFormat: models.PostgresFormatTar,
			},
			database: "testdb",
			outFile:  "/backups/testdb.tar",
			contains: []string{
				"-F", "t",
			},
		},
		{
			name: "directory format",
			config: &models.PostgresBackupConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				OutputFormat: models.PostgresFormatDirectory,
			},
			database: "testdb",
			outFile:  "/backups/testdb",
			contains: []string{
				"-F", "d",
			},
		},
		{
			name: "with compression",
			config: &models.PostgresBackupConfig{
				Host:             "localhost",
				Port:             5432,
				Username:         "postgres",
				OutputFormat:     models.PostgresFormatCustom,
				CompressionLevel: 9,
			},
			database: "testdb",
			outFile:  "/backups/testdb.dump",
			contains: []string{
				"-Z", "9",
			},
		},
		{
			name: "compression ignored for plain format",
			config: &models.PostgresBackupConfig{
				Host:             "localhost",
				Port:             5432,
				Username:         "postgres",
				OutputFormat:     models.PostgresFormatPlain,
				CompressionLevel: 6,
			},
			database: "testdb",
			outFile:  "/backups/testdb.sql",
			excludes: []string{
				"-Z",
			},
		},
		{
			name: "schema only",
			config: &models.PostgresBackupConfig{
				Host:              "localhost",
				Port:              5432,
				Username:          "postgres",
				OutputFormat:      models.PostgresFormatCustom,
				IncludeSchemaOnly: true,
			},
			database: "testdb",
			outFile:  "/backups/testdb.dump",
			contains: []string{
				"-s",
			},
			excludes: []string{
				"-a",
			},
		},
		{
			name: "data only",
			config: &models.PostgresBackupConfig{
				Host:            "localhost",
				Port:            5432,
				Username:        "postgres",
				OutputFormat:    models.PostgresFormatCustom,
				IncludeDataOnly: true,
			},
			database: "testdb",
			outFile:  "/backups/testdb.dump",
			contains: []string{
				"-a",
			},
			excludes: []string{
				"-s",
			},
		},
		{
			name: "no owner and no privileges",
			config: &models.PostgresBackupConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				OutputFormat: models.PostgresFormatCustom,
				NoOwner:      true,
				NoPrivileges: true,
			},
			database: "testdb",
			outFile:  "/backups/testdb.dump",
			contains: []string{
				"-O",
				"-x",
			},
		},
		{
			name: "exclude tables",
			config: &models.PostgresBackupConfig{
				Host:          "localhost",
				Port:          5432,
				Username:      "postgres",
				OutputFormat:  models.PostgresFormatCustom,
				ExcludeTables: []string{"logs", "temp_data"},
			},
			database: "testdb",
			outFile:  "/backups/testdb.dump",
			contains: []string{
				"-T", "logs",
				"-T", "temp_data",
			},
		},
		{
			name: "include tables",
			config: &models.PostgresBackupConfig{
				Host:          "localhost",
				Port:          5432,
				Username:      "postgres",
				OutputFormat:  models.PostgresFormatCustom,
				IncludeTables: []string{"users", "orders"},
			},
			database: "testdb",
			outFile:  "/backups/testdb.dump",
			contains: []string{
				"-t", "users",
				"-t", "orders",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := NewPostgresBackup(tt.config, testLogger())
			args := backup.buildPgDumpArgs(tt.database, tt.outFile)
			joined := strings.Join(args, " ")

			for _, expected := range tt.contains {
				if !strings.Contains(joined, expected) {
					t.Errorf("args %v should contain %q", args, expected)
				}
			}
			for _, excluded := range tt.excludes {
				if strings.Contains(joined, excluded) {
					t.Errorf("args %v should NOT contain %q", args, excluded)
				}
			}
		})
	}
}

func TestPostgresBackup_BuildPgDumpAllArgs(t *testing.T) {
	tests := []struct {
		name     string
		config   *models.PostgresBackupConfig
		contains []string
		excludes []string
	}{
		{
			name: "basic",
			config: &models.PostgresBackupConfig{
				Host:     "db.example.com",
				Port:     5432,
				Username: "postgres",
			},
			contains: []string{
				"-h", "db.example.com",
				"-p", "5432",
				"-U", "postgres",
			},
		},
		{
			name: "schema only",
			config: &models.PostgresBackupConfig{
				Host:              "localhost",
				Port:              5432,
				Username:          "postgres",
				IncludeSchemaOnly: true,
			},
			contains: []string{
				"-s",
			},
		},
		{
			name: "data only",
			config: &models.PostgresBackupConfig{
				Host:            "localhost",
				Port:            5432,
				Username:        "postgres",
				IncludeDataOnly: true,
			},
			contains: []string{
				"-a",
			},
		},
		{
			name: "no owner and no privileges",
			config: &models.PostgresBackupConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				NoOwner:      true,
				NoPrivileges: true,
			},
			contains: []string{
				"-O",
				"-x",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := NewPostgresBackup(tt.config, testLogger())
			args := backup.buildPgDumpAllArgs()
			joined := strings.Join(args, " ")

			for _, expected := range tt.contains {
				if !strings.Contains(joined, expected) {
					t.Errorf("args %v should contain %q", args, expected)
				}
			}
			for _, excluded := range tt.excludes {
				if strings.Contains(joined, excluded) {
					t.Errorf("args %v should NOT contain %q", args, excluded)
				}
			}
		})
	}
}

func TestPostgresBackup_BuildEnvironment(t *testing.T) {
	t.Run("with password", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
		}, testLogger())
		backup.DecryptedPassword = "supersecret"

		env := backup.buildEnvironment()
		found := false
		for _, e := range env {
			if e == "PGPASSWORD=supersecret" {
				found = true
			}
		}
		if !found {
			t.Error("expected PGPASSWORD in environment")
		}
	})

	t.Run("without password", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
		}, testLogger())

		env := backup.buildEnvironment()
		for _, e := range env {
			if strings.HasPrefix(e, "PGPASSWORD=") {
				t.Error("PGPASSWORD should NOT be in environment when no password set")
			}
		}
	})

	t.Run("with SSL mode", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
			SSLMode:  "require",
		}, testLogger())

		env := backup.buildEnvironment()
		found := false
		for _, e := range env {
			if e == "PGSSLMODE=require" {
				found = true
			}
		}
		if !found {
			t.Error("expected PGSSLMODE=require in environment")
		}
	})

	t.Run("without SSL mode", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
		}, testLogger())

		env := backup.buildEnvironment()
		for _, e := range env {
			if strings.HasPrefix(e, "PGSSLMODE=") {
				t.Error("PGSSLMODE should NOT be set when SSLMode is empty")
			}
		}
	})
}

func TestPostgresBackup_PasswordNotInConnectionString(t *testing.T) {
	// Verify that the password is never embedded in the connection string.
	// It should only be passed via PGPASSWORD env var.
	backup := NewPostgresBackup(&models.PostgresBackupConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Database: "testdb",
	}, testLogger())
	backup.DecryptedPassword = "secretpassword"

	connStr := backup.buildConnectionString()
	if strings.Contains(connStr, "secretpassword") {
		t.Error("password should NOT appear in the connection string")
	}

	// Password should be in the environment instead
	env := backup.buildEnvironment()
	found := false
	for _, e := range env {
		if strings.Contains(e, "secretpassword") {
			found = true
		}
	}
	if !found {
		t.Error("password should be in PGPASSWORD env var")
	}
}

func TestPostgresBackup_PasswordNotInDumpArgs(t *testing.T) {
	backup := NewPostgresBackup(&models.PostgresBackupConfig{
		Host:         "localhost",
		Port:         5432,
		Username:     "postgres",
		OutputFormat: models.PostgresFormatCustom,
	}, testLogger())
	backup.DecryptedPassword = "secretpassword"

	args := backup.buildPgDumpArgs("testdb", "/tmp/backup.dump")
	for _, arg := range args {
		if strings.Contains(arg, "secretpassword") {
			t.Errorf("password leaked in pg_dump args: %q", arg)
		}
	}

	allArgs := backup.buildPgDumpAllArgs()
	for _, arg := range allArgs {
		if strings.Contains(arg, "secretpassword") {
			t.Errorf("password leaked in pg_dumpall args: %q", arg)
		}
	}
}

func TestPostgresBackup_GetFileExtension(t *testing.T) {
	tests := []struct {
		format   models.PostgresOutputFormat
		expected string
	}{
		{models.PostgresFormatCustom, ".dump"},
		{models.PostgresFormatDirectory, ""},
		{models.PostgresFormatTar, ".tar"},
		{models.PostgresFormatPlain, ".sql"},
		{"", ".dump"}, // empty format defaults to custom via NewPostgresBackup
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			backup := NewPostgresBackup(&models.PostgresBackupConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				OutputFormat: tt.format,
			}, testLogger())

			ext := backup.getFileExtension()
			if ext != tt.expected {
				t.Errorf("expected extension %q for format %q, got %q", tt.expected, tt.format, ext)
			}
		})
	}
}

func TestNewPostgresBackup_Defaults(t *testing.T) {
	t.Run("nil config gets defaults", func(t *testing.T) {
		backup := NewPostgresBackup(nil, testLogger())
		if backup.Config.Port != defaultPostgresPort {
			t.Errorf("expected port %d, got %d", defaultPostgresPort, backup.Config.Port)
		}
		if backup.Config.OutputFormat != models.PostgresFormatCustom {
			t.Errorf("expected format %q, got %q", models.PostgresFormatCustom, backup.Config.OutputFormat)
		}
	})

	t.Run("zero port gets default", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:     "localhost",
			Username: "postgres",
		}, testLogger())
		if backup.Config.Port != defaultPostgresPort {
			t.Errorf("expected port %d, got %d", defaultPostgresPort, backup.Config.Port)
		}
	})

	t.Run("empty format gets custom", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:     "localhost",
			Port:     5432,
			Username: "postgres",
		}, testLogger())
		if backup.Config.OutputFormat != models.PostgresFormatCustom {
			t.Errorf("expected format %q, got %q", models.PostgresFormatCustom, backup.Config.OutputFormat)
		}
	})

	t.Run("non-zero values preserved", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:         "custom-host",
			Port:         5433,
			Username:     "admin",
			OutputFormat: models.PostgresFormatPlain,
		}, testLogger())
		if backup.Config.Port != 5433 {
			t.Errorf("expected port 5433, got %d", backup.Config.Port)
		}
		if backup.Config.OutputFormat != models.PostgresFormatPlain {
			t.Errorf("expected format %q, got %q", models.PostgresFormatPlain, backup.Config.OutputFormat)
		}
	})
}

func TestPostgresBackup_GetRestoreInstructions(t *testing.T) {
	tests := []struct {
		name       string
		format     models.PostgresOutputFormat
		backupPath string
		expectCmd  string
	}{
		{
			name:       "custom format",
			format:     models.PostgresFormatCustom,
			backupPath: "/backups/mydb.dump",
			expectCmd:  "pg_restore",
		},
		{
			name:       "directory format",
			format:     models.PostgresFormatDirectory,
			backupPath: "/backups/mydb_dir",
			expectCmd:  "pg_restore",
		},
		{
			name:       "tar format",
			format:     models.PostgresFormatTar,
			backupPath: "/backups/mydb.tar",
			expectCmd:  "pg_restore",
		},
		{
			name:       "plain format",
			format:     models.PostgresFormatPlain,
			backupPath: "/backups/mydb.sql",
			expectCmd:  "psql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := NewPostgresBackup(&models.PostgresBackupConfig{
				Host:         "localhost",
				Port:         5432,
				Username:     "postgres",
				OutputFormat: tt.format,
			}, testLogger())

			instructions := backup.GetRestoreInstructions(tt.backupPath)

			if instructions.Format != string(tt.format) {
				t.Errorf("expected format %q, got %q", tt.format, instructions.Format)
			}
			if len(instructions.Instructions) == 0 {
				t.Error("expected non-empty instructions")
			}
			if len(instructions.Commands) == 0 {
				t.Error("expected non-empty commands")
			}

			// Verify the expected restore command is in the commands
			foundCmd := false
			for _, cmd := range instructions.Commands {
				if strings.Contains(cmd, tt.expectCmd) {
					foundCmd = true
				}
			}
			if !foundCmd {
				t.Errorf("expected restore command %q in instructions, got %v", tt.expectCmd, instructions.Commands)
			}

			// Verify backup path appears in commands
			foundPath := false
			for _, cmd := range instructions.Commands {
				if strings.Contains(cmd, tt.backupPath) {
					foundPath = true
				}
			}
			if !foundPath {
				t.Errorf("expected backup path %q in restore commands", tt.backupPath)
			}
		})
	}
}

func TestPostgresBackup_Validate(t *testing.T) {
	t.Run("nil config", func(t *testing.T) {
		backup := &PostgresBackup{Config: nil, logger: testLogger()}
		err := backup.Validate()
		if err == nil {
			t.Error("expected error for nil config")
		}
	})

	t.Run("missing host", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Username: "postgres",
		}, testLogger())
		backup.Config.Host = "" // override
		err := backup.Validate()
		if err == nil {
			t.Error("expected error for missing host")
		}
		if !strings.Contains(err.Error(), "host") {
			t.Errorf("error should mention host: %v", err)
		}
	})

	t.Run("missing username", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host: "localhost",
		}, testLogger())
		backup.Config.Username = "" // override
		err := backup.Validate()
		if err == nil {
			t.Error("expected error for missing username")
		}
		if !strings.Contains(err.Error(), "username") {
			t.Errorf("error should mention username: %v", err)
		}
	})

	t.Run("both schema and data only", func(t *testing.T) {
		backup := NewPostgresBackup(&models.PostgresBackupConfig{
			Host:              "localhost",
			Username:          "postgres",
			IncludeSchemaOnly: true,
			IncludeDataOnly:   true,
		}, testLogger())
		err := backup.Validate()
		if err == nil {
			t.Error("expected error when both schema-only and data-only are set")
		}
		if !strings.Contains(err.Error(), "schema-only") || !strings.Contains(err.Error(), "data-only") {
			t.Errorf("error should mention both options: %v", err)
		}
	})
}

func TestDefaultPostgresConfig(t *testing.T) {
	cfg := models.DefaultPostgresConfig()

	if cfg.Host != "localhost" {
		t.Errorf("expected host localhost, got %q", cfg.Host)
	}
	if cfg.Port != 5432 {
		t.Errorf("expected port 5432, got %d", cfg.Port)
	}
	if cfg.Username != "postgres" {
		t.Errorf("expected username postgres, got %q", cfg.Username)
	}
	if cfg.OutputFormat != models.PostgresFormatCustom {
		t.Errorf("expected format %q, got %q", models.PostgresFormatCustom, cfg.OutputFormat)
	}
	if cfg.CompressionLevel != 6 {
		t.Errorf("expected compression level 6, got %d", cfg.CompressionLevel)
	}
	if cfg.SSLMode != "prefer" {
		t.Errorf("expected SSL mode prefer, got %q", cfg.SSLMode)
	}
}
