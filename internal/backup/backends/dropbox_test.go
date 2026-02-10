package backends

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestDropboxBackend_Init(t *testing.T) {
	b := &DropboxBackend{
		RemoteName: "myremote",
		Path:       "/backups",
	}
	if b.Type() != models.RepositoryTypeDropbox {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeDropbox)
	}
}

func TestDropboxBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend DropboxBackend
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with remote name only",
			backend: DropboxBackend{
				RemoteName: "myremote",
			},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			backend: DropboxBackend{
				RemoteName: "myremote",
				Path:       "/backups/daily",
				Token:      "sl.test-token",
				AppKey:     "app-key-123",
				AppSecret:  "app-secret-456",
			},
			wantErr: false,
		},
		{
			name:    "missing remote name",
			backend: DropboxBackend{},
			wantErr: true,
			errMsg:  "remote_name is required",
		},
		{
			name: "missing remote name with other fields",
			backend: DropboxBackend{
				Path:  "/backups",
				Token: "token",
			},
			wantErr: true,
			errMsg:  "remote_name is required",
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

func TestDropboxBackend_ToResticConfig(t *testing.T) {
	t.Run("basic config", func(t *testing.T) {
		b := &DropboxBackend{
			RemoteName: "myremote",
			Path:       "/backups",
		}

		cfg := b.ToResticConfig("pass")

		if cfg.Repository != "rclone:myremote:/backups" {
			t.Errorf("Repository = %v, want rclone:myremote:/backups", cfg.Repository)
		}
		if cfg.Password != "pass" {
			t.Errorf("Password = %v, want pass", cfg.Password)
		}
	})

	t.Run("empty path defaults to root", func(t *testing.T) {
		b := &DropboxBackend{
			RemoteName: "myremote",
		}

		cfg := b.ToResticConfig("pass")

		if cfg.Repository != "rclone:myremote:/" {
			t.Errorf("Repository = %v, want rclone:myremote:/", cfg.Repository)
		}
	})

	t.Run("path without leading slash gets prefixed", func(t *testing.T) {
		b := &DropboxBackend{
			RemoteName: "myremote",
			Path:       "backups",
		}

		cfg := b.ToResticConfig("pass")

		if cfg.Repository != "rclone:myremote:/backups" {
			t.Errorf("Repository = %v, want rclone:myremote:/backups", cfg.Repository)
		}
	})

	t.Run("token sets rclone env vars", func(t *testing.T) {
		b := &DropboxBackend{
			RemoteName: "myremote",
			Path:       "/backups",
			Token:      "sl.test-token",
		}

		cfg := b.ToResticConfig("pass")

		typeKey := "RCLONE_CONFIG_MYREMOTE_TYPE"
		tokenKey := "RCLONE_CONFIG_MYREMOTE_TOKEN"

		if cfg.Env[typeKey] != "dropbox" {
			t.Errorf("Env[%s] = %v, want dropbox", typeKey, cfg.Env[typeKey])
		}
		if cfg.Env[tokenKey] != "sl.test-token" {
			t.Errorf("Env[%s] = %v, want sl.test-token", tokenKey, cfg.Env[tokenKey])
		}
	})

	t.Run("app credentials set rclone env vars", func(t *testing.T) {
		b := &DropboxBackend{
			RemoteName: "myremote",
			Path:       "/backups",
			AppKey:     "app-key-123",
			AppSecret:  "app-secret-456",
		}

		cfg := b.ToResticConfig("pass")

		clientIDKey := "RCLONE_CONFIG_MYREMOTE_CLIENT_ID"
		clientSecretKey := "RCLONE_CONFIG_MYREMOTE_CLIENT_SECRET"

		if cfg.Env[clientIDKey] != "app-key-123" {
			t.Errorf("Env[%s] = %v, want app-key-123", clientIDKey, cfg.Env[clientIDKey])
		}
		if cfg.Env[clientSecretKey] != "app-secret-456" {
			t.Errorf("Env[%s] = %v, want app-secret-456", clientSecretKey, cfg.Env[clientSecretKey])
		}
	})

	t.Run("env keys use uppercase remote name", func(t *testing.T) {
		b := &DropboxBackend{
			RemoteName: "myDropbox",
			Token:      "token",
		}

		cfg := b.ToResticConfig("pass")

		upperName := strings.ToUpper("myDropbox")
		typeKey := "RCLONE_CONFIG_" + upperName + "_TYPE"
		if _, ok := cfg.Env[typeKey]; !ok {
			t.Errorf("expected env key %s to be set", typeKey)
		}
	})

	t.Run("no token or credentials produces empty env", func(t *testing.T) {
		b := &DropboxBackend{
			RemoteName: "myremote",
			Path:       "/backups",
		}

		cfg := b.ToResticConfig("pass")

		if len(cfg.Env) != 0 {
			t.Errorf("Env should be empty when no token/credentials, got %v", cfg.Env)
		}
	})
}

func TestDropboxBackend_TestConnection_InvalidConfig(t *testing.T) {
	b := &DropboxBackend{} // Missing remote_name
	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for invalid config, got nil")
	}
}

func TestDropboxBackend_TestConnection_RcloneNotInstalled(t *testing.T) {
	// Only run this test if rclone is not installed
	if _, err := exec.LookPath("rclone"); err == nil {
		t.Skip("rclone is installed, skipping rclone-not-installed test")
	}

	b := &DropboxBackend{
		RemoteName: "myremote",
		Path:       "/backups",
	}

	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error when rclone not installed, got nil")
	}
	if !contains(err.Error(), "rclone is not installed") {
		t.Errorf("TestConnection() error = %q, want to contain 'rclone is not installed'", err.Error())
	}
}

func TestDropboxBackend_TestConnection_RcloneFails(t *testing.T) {
	// Only run this test if rclone IS installed
	if _, err := exec.LookPath("rclone"); err != nil {
		t.Skip("rclone is not installed, skipping rclone-failure test")
	}

	// Use a nonexistent remote that will cause rclone to fail
	b := &DropboxBackend{
		RemoteName: "nonexistent_remote_xyz",
		Path:       "/backups",
		Token:      "invalid-token",
	}

	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for invalid remote, got nil")
	}
	if !contains(err.Error(), "connection test failed") {
		t.Errorf("TestConnection() error = %q, want to contain 'connection test failed'", err.Error())
	}
}
