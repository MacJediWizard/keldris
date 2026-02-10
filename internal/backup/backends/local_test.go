package backends

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestLocalBackend_Init(t *testing.T) {
	b := &LocalBackend{Path: "/var/backups/restic"}
	if b.Type() != models.RepositoryTypeLocal {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeLocal)
	}
}

func TestLocalBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend LocalBackend
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid absolute path",
			backend: LocalBackend{Path: "/var/backups/restic"},
			wantErr: false,
		},
		{
			name:    "empty path",
			backend: LocalBackend{Path: ""},
			wantErr: true,
			errMsg:  "path is required",
		},
		{
			name:    "relative path",
			backend: LocalBackend{Path: "backups/restic"},
			wantErr: true,
			errMsg:  "path must be absolute",
		},
		{
			name:    "dot relative path",
			backend: LocalBackend{Path: "./backups"},
			wantErr: true,
			errMsg:  "path must be absolute",
		},
		{
			name:    "root path",
			backend: LocalBackend{Path: "/"},
			wantErr: false,
		},
		{
			name:    "deeply nested path",
			backend: LocalBackend{Path: "/a/b/c/d/e/f"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if got := err.Error(); !contains(got, tt.errMsg) {
					t.Errorf("Validate() error = %q, want to contain %q", got, tt.errMsg)
				}
			}
		})
	}
}

func TestLocalBackend_GetEnv(t *testing.T) {
	t.Run("basic config", func(t *testing.T) {
		b := &LocalBackend{Path: "/var/backups/restic"}
		cfg := b.ToResticConfig("mypassword")

		if cfg.Repository != "/var/backups/restic" {
			t.Errorf("Repository = %v, want /var/backups/restic", cfg.Repository)
		}
		if cfg.Password != "mypassword" {
			t.Errorf("Password = %v, want mypassword", cfg.Password)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		b := &LocalBackend{Path: "/backups"}
		cfg := b.ToResticConfig("")

		if cfg.Password != "" {
			t.Errorf("Password = %v, want empty string", cfg.Password)
		}
	})

	t.Run("no env vars set", func(t *testing.T) {
		b := &LocalBackend{Path: "/backups"}
		cfg := b.ToResticConfig("pass")

		if cfg.Env != nil {
			t.Errorf("Env = %v, want nil (local backend has no env vars)", cfg.Env)
		}
	})
}

func TestLocalBackend_PathExists(t *testing.T) {
	t.Run("parent directory exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		backupPath := filepath.Join(tmpDir, "backups")

		b := &LocalBackend{Path: backupPath}
		err := b.TestConnection()
		if err != nil {
			t.Errorf("TestConnection() error = %v, want nil", err)
		}
	})

	t.Run("path already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		backupPath := filepath.Join(tmpDir, "backups")
		if err := os.Mkdir(backupPath, 0o755); err != nil {
			t.Fatalf("failed to create test dir: %v", err)
		}

		b := &LocalBackend{Path: backupPath}
		err := b.TestConnection()
		if err != nil {
			t.Errorf("TestConnection() error = %v, want nil", err)
		}
	})

	t.Run("parent does not exist", func(t *testing.T) {
		b := &LocalBackend{Path: "/nonexistent-parent-dir-xyz/backups"}
		err := b.TestConnection()
		if err == nil {
			t.Error("TestConnection() expected error for nonexistent parent, got nil")
		}
	})

	t.Run("invalid config fails validation first", func(t *testing.T) {
		b := &LocalBackend{Path: ""}
		err := b.TestConnection()
		if err == nil {
			t.Error("TestConnection() expected error for empty path, got nil")
		}
	})

	t.Run("parent is a file not directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "afile")
		if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		b := &LocalBackend{Path: filepath.Join(filePath, "backups")}
		err := b.TestConnection()
		if err == nil {
			t.Error("TestConnection() expected error when parent is a file, got nil")
		}
	})
}

// contains checks if s contains substr.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
