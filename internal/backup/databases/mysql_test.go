package databases

import (
	"strings"
	"testing"
	"time"
)

func TestMySQLConfig_DSN(t *testing.T) {
	tests := []struct {
		name     string
		config   MySQLConfig
		contains []string
		excludes []string
	}{
		{
			name: "basic connection",
			config: MySQLConfig{
				Host:           "db.example.com",
				Port:           3306,
				Username:       "admin",
				Password:       "secret",
				Database:       "mydb",
				ConnectTimeout: 30 * time.Second,
			},
			contains: []string{
				"admin:secret@",
				"tcp(db.example.com:3306)/",
				"mydb",
				"timeout=30s",
			},
		},
		{
			name: "no password",
			config: MySQLConfig{
				Host:           "localhost",
				Port:           3306,
				Username:       "root",
				ConnectTimeout: 30 * time.Second,
			},
			contains: []string{
				"root@",
				"tcp(localhost:3306)/",
			},
			excludes: []string{
				"root:@", // should not have trailing colon
			},
		},
		{
			name: "no username",
			config: MySQLConfig{
				Host:           "localhost",
				Port:           3306,
				ConnectTimeout: 30 * time.Second,
			},
			contains: []string{
				"tcp(localhost:3306)/",
			},
			excludes: []string{
				"@tcp", // no credentials
			},
		},
		{
			name: "no database",
			config: MySQLConfig{
				Host:           "localhost",
				Port:           3306,
				Username:       "root",
				ConnectTimeout: 30 * time.Second,
			},
			contains: []string{
				"tcp(localhost:3306)/",
			},
		},
		{
			name: "with SSL",
			config: MySQLConfig{
				Host:           "localhost",
				Port:           3306,
				Username:       "root",
				SSLMode:        "required",
				ConnectTimeout: 30 * time.Second,
			},
			contains: []string{
				"tls=required",
			},
		},
		{
			name: "custom port",
			config: MySQLConfig{
				Host:           "localhost",
				Port:           3307,
				Username:       "root",
				ConnectTimeout: 30 * time.Second,
			},
			contains: []string{
				"tcp(localhost:3307)/",
			},
		},
		{
			name: "default connect timeout when zero",
			config: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				// ConnectTimeout intentionally zero
			},
			contains: []string{
				"timeout=30s", // defaultConnectTimeout
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dsn := tt.config.DSN()

			for _, expected := range tt.contains {
				if !strings.Contains(dsn, expected) {
					t.Errorf("DSN %q should contain %q", dsn, expected)
				}
			}
			for _, excluded := range tt.excludes {
				if strings.Contains(dsn, excluded) {
					t.Errorf("DSN %q should NOT contain %q", dsn, excluded)
				}
			}
		})
	}
}

func TestMySQLConfig_BuildDumpArgs(t *testing.T) {
	tests := []struct {
		name      string
		config    MySQLConfig
		databases []string
		contains  []string
		excludes  []string
	}{
		{
			name: "single database",
			config: MySQLConfig{
				Host:     "db.example.com",
				Port:     3306,
				Username: "backup_user",
				Password: "backuppass",
			},
			databases: []string{"production"},
			contains: []string{
				"--host=db.example.com",
				"--port=3306",
				"--user=backup_user",
				"--password=backuppass",
				"--single-transaction",
				"--quick",
				"--lock-tables=false",
				"--routines",
				"--triggers",
				"--events",
				"--set-gtid-purged=OFF",
				"production",
			},
			excludes: []string{
				"--all-databases",
				"--databases",
			},
		},
		{
			name: "all databases",
			config: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
			},
			databases: nil,
			contains: []string{
				"--all-databases",
			},
			excludes: []string{
				"--databases",
			},
		},
		{
			name: "multiple databases",
			config: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
			},
			databases: []string{"db1", "db2", "db3"},
			contains: []string{
				"--databases",
				"db1",
				"db2",
				"db3",
			},
			excludes: []string{
				"--all-databases",
			},
		},
		{
			name: "all databases with exclusions",
			config: MySQLConfig{
				Host:             "localhost",
				Port:             3306,
				Username:         "root",
				ExcludeDatabases: []string{"test", "temp"},
			},
			databases: nil,
			contains: []string{
				"--all-databases",
				"--ignore-table=test.*",
				"--ignore-table=temp.*",
			},
		},
		{
			name: "no password in args",
			config: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
			},
			databases: []string{"mydb"},
			excludes: []string{
				"--password=",
			},
		},
		{
			name: "SSL mode required",
			config: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				SSLMode:  "required",
			},
			databases: []string{"mydb"},
			contains: []string{
				"--ssl-mode=REQUIRED",
			},
		},
		{
			name: "SSL mode verify-ca",
			config: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				SSLMode:  "verify-ca",
			},
			databases: []string{"mydb"},
			contains: []string{
				"--ssl-mode=VERIFY-CA",
			},
		},
		{
			name: "unsupported SSL mode ignored",
			config: MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				SSLMode:  "disabled",
			},
			databases: []string{"mydb"},
			excludes: []string{
				"--ssl-mode=",
			},
		},
		{
			name: "extra args",
			config: MySQLConfig{
				Host:      "localhost",
				Port:      3306,
				Username:  "root",
				ExtraArgs: []string{"--max-allowed-packet=512M", "--column-statistics=0"},
			},
			databases: []string{"mydb"},
			contains: []string{
				"--max-allowed-packet=512M",
				"--column-statistics=0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backup := NewMySQLBackup(&tt.config, testLogger())
			args := backup.buildDumpArgs(tt.databases)
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

func TestMySQLConfig_SanitizeArgs(t *testing.T) {
	backup := NewMySQLBackup(&MySQLConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
		Password: "topsecret",
	}, testLogger())

	args := []string{
		"--host=localhost",
		"--port=3306",
		"--user=root",
		"--password=topsecret",
		"--single-transaction",
	}

	sanitized := backup.sanitizeArgs(args)

	for _, arg := range sanitized {
		if strings.Contains(arg, "topsecret") {
			t.Errorf("sanitized args should not contain the password, found: %q", arg)
		}
	}

	// Verify the password arg is masked
	found := false
	for _, arg := range sanitized {
		if arg == "--password=***" {
			found = true
		}
	}
	if !found {
		t.Error("expected --password=*** in sanitized args")
	}

	// Non-password args should be unchanged
	for i, arg := range args {
		if !strings.HasPrefix(arg, "--password=") {
			if sanitized[i] != arg {
				t.Errorf("non-password arg %q was modified to %q", arg, sanitized[i])
			}
		}
	}
}

func TestMySQLConfig_SanitizeArgs_NoPassword(t *testing.T) {
	backup := NewMySQLBackup(&MySQLConfig{
		Host:     "localhost",
		Port:     3306,
		Username: "root",
	}, testLogger())

	args := []string{
		"--host=localhost",
		"--user=root",
	}

	sanitized := backup.sanitizeArgs(args)
	if len(sanitized) != len(args) {
		t.Fatalf("sanitized length %d != args length %d", len(sanitized), len(args))
	}
	for i, arg := range sanitized {
		if arg != args[i] {
			t.Errorf("arg %d changed from %q to %q", i, args[i], arg)
		}
	}
}

func TestNewMySQLBackup_Defaults(t *testing.T) {
	t.Run("nil config gets defaults", func(t *testing.T) {
		backup := NewMySQLBackup(nil, testLogger())
		if backup.config.Port != 3306 {
			t.Errorf("expected default port 3306, got %d", backup.config.Port)
		}
		if backup.config.DumpTimeout != defaultDumpTimeout {
			t.Errorf("expected default dump timeout %v, got %v", defaultDumpTimeout, backup.config.DumpTimeout)
		}
		if backup.config.ConnectTimeout != defaultConnectTimeout {
			t.Errorf("expected default connect timeout %v, got %v", defaultConnectTimeout, backup.config.ConnectTimeout)
		}
	})

	t.Run("zero port gets default", func(t *testing.T) {
		backup := NewMySQLBackup(&MySQLConfig{Host: "localhost"}, testLogger())
		if backup.config.Port != 3306 {
			t.Errorf("expected port 3306, got %d", backup.config.Port)
		}
	})

	t.Run("zero timeouts get defaults", func(t *testing.T) {
		backup := NewMySQLBackup(&MySQLConfig{
			Host: "localhost",
			Port: 3306,
		}, testLogger())
		if backup.config.DumpTimeout != defaultDumpTimeout {
			t.Errorf("expected dump timeout %v, got %v", defaultDumpTimeout, backup.config.DumpTimeout)
		}
		if backup.config.ConnectTimeout != defaultConnectTimeout {
			t.Errorf("expected connect timeout %v, got %v", defaultConnectTimeout, backup.config.ConnectTimeout)
		}
	})

	t.Run("non-zero values preserved", func(t *testing.T) {
		customTimeout := 5 * time.Minute
		backup := NewMySQLBackup(&MySQLConfig{
			Host:           "custom-host",
			Port:           3307,
			DumpTimeout:    customTimeout,
			ConnectTimeout: customTimeout,
		}, testLogger())
		if backup.config.Port != 3307 {
			t.Errorf("expected port 3307, got %d", backup.config.Port)
		}
		if backup.config.DumpTimeout != customTimeout {
			t.Errorf("expected dump timeout %v, got %v", customTimeout, backup.config.DumpTimeout)
		}
		if backup.config.ConnectTimeout != customTimeout {
			t.Errorf("expected connect timeout %v, got %v", customTimeout, backup.config.ConnectTimeout)
		}
	})
}

func TestDefaultMySQLConfig(t *testing.T) {
	cfg := DefaultMySQLConfig()

	if cfg.Host != "localhost" {
		t.Errorf("expected host localhost, got %q", cfg.Host)
	}
	if cfg.Port != 3306 {
		t.Errorf("expected port 3306, got %d", cfg.Port)
	}
	if !cfg.Compress {
		t.Error("expected compress to be true by default")
	}
	if cfg.DumpTimeout != defaultDumpTimeout {
		t.Errorf("expected dump timeout %v, got %v", defaultDumpTimeout, cfg.DumpTimeout)
	}
	if cfg.ConnectTimeout != defaultConnectTimeout {
		t.Errorf("expected connect timeout %v, got %v", defaultConnectTimeout, cfg.ConnectTimeout)
	}
}

func TestMySQLBackupFilenaming(t *testing.T) {
	// Test buildDumpArgs to verify the database names that end up in args
	// which drive the filename construction in Backup().

	tests := []struct {
		name          string
		database      string
		databases     []string
		expectSingle  bool
		expectAll     bool
		expectMulti   bool
	}{
		{
			name:         "single database",
			database:     "mydb",
			expectSingle: true,
		},
		{
			name:        "multiple databases",
			databases:   []string{"db1", "db2"},
			expectMulti: true,
		},
		{
			name:      "all databases",
			expectAll: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
			}

			var databases []string
			if tt.database != "" {
				databases = []string{tt.database}
			} else if len(tt.databases) > 0 {
				databases = tt.databases
			}

			backup := NewMySQLBackup(config, testLogger())
			args := backup.buildDumpArgs(databases)
			joined := strings.Join(args, " ")

			if tt.expectSingle {
				if !strings.Contains(joined, tt.database) {
					t.Errorf("expected single database %q in args", tt.database)
				}
				if strings.Contains(joined, "--all-databases") {
					t.Error("single database should not use --all-databases")
				}
			}
			if tt.expectAll {
				if !strings.Contains(joined, "--all-databases") {
					t.Error("expected --all-databases in args")
				}
			}
			if tt.expectMulti {
				if !strings.Contains(joined, "--databases") {
					t.Error("expected --databases flag in args")
				}
				for _, db := range tt.databases {
					if !strings.Contains(joined, db) {
						t.Errorf("expected database %q in args", db)
					}
				}
			}
		})
	}
}

func TestMySQLCredentialNotLeakedInSanitize(t *testing.T) {
	passwords := []string{
		"simple",
		"p@$$w0rd!",
		"contains spaces",
		"has=equals",
		"has--dashes",
		`has"quotes`,
	}

	for _, pw := range passwords {
		t.Run(pw, func(t *testing.T) {
			backup := NewMySQLBackup(&MySQLConfig{
				Host:     "localhost",
				Port:     3306,
				Username: "root",
				Password: pw,
			}, testLogger())

			args := backup.buildDumpArgs([]string{"testdb"})
			sanitized := backup.sanitizeArgs(args)

			for _, arg := range sanitized {
				if strings.Contains(arg, pw) {
					t.Errorf("password %q leaked in sanitized arg: %q", pw, arg)
				}
			}
		})
	}
}

func TestMySQLGetRestoreInstructions(t *testing.T) {
	backup := NewMySQLBackup(DefaultMySQLConfig(), testLogger())

	t.Run("compressed backup", func(t *testing.T) {
		instructions := backup.GetRestoreInstructions("/backups/mysql_mydb_20260308-120000.sql.gz")
		if !strings.Contains(instructions, "gunzip") {
			t.Error("compressed backup instructions should mention gunzip")
		}
		if !strings.Contains(instructions, "mysql_mydb_20260308-120000.sql.gz") {
			t.Error("instructions should include the backup filename")
		}
	})

	t.Run("uncompressed backup", func(t *testing.T) {
		instructions := backup.GetRestoreInstructions("/backups/mysql_mydb_20260308-120000.sql")
		if strings.Contains(instructions, "gunzip") {
			t.Error("uncompressed backup instructions should NOT mention gunzip")
		}
		if !strings.Contains(instructions, "mysql_mydb_20260308-120000.sql") {
			t.Error("instructions should include the backup filename")
		}
	})
}
