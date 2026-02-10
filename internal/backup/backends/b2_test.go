package backends

import (
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestB2Backend_Init(t *testing.T) {
	b := &B2Backend{
		Bucket:         "my-bucket",
		AccountID:      "account123",
		ApplicationKey: "key456",
	}
	if b.Type() != models.RepositoryTypeB2 {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeB2)
	}
}

func TestB2Backend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend B2Backend
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid",
			backend: B2Backend{
				Bucket:         "my-bucket",
				AccountID:      "account123",
				ApplicationKey: "key456",
			},
			wantErr: false,
		},
		{
			name: "valid with prefix",
			backend: B2Backend{
				Bucket:         "my-bucket",
				Prefix:         "backups/daily",
				AccountID:      "account123",
				ApplicationKey: "key456",
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			backend: B2Backend{
				AccountID:      "account123",
				ApplicationKey: "key456",
			},
			wantErr: true,
			errMsg:  "bucket is required",
		},
		{
			name: "missing account id",
			backend: B2Backend{
				Bucket:         "my-bucket",
				ApplicationKey: "key456",
			},
			wantErr: true,
			errMsg:  "account_id is required",
		},
		{
			name: "missing application key",
			backend: B2Backend{
				Bucket:    "my-bucket",
				AccountID: "account123",
			},
			wantErr: true,
			errMsg:  "application_key is required",
		},
		{
			name:    "all fields empty",
			backend: B2Backend{},
			wantErr: true,
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

func TestB2Backend_GetEnv(t *testing.T) {
	t.Run("without prefix", func(t *testing.T) {
		b := &B2Backend{
			Bucket:         "my-bucket",
			AccountID:      "account123",
			ApplicationKey: "key456",
		}
		cfg := b.ToResticConfig("mypassword")

		if cfg.Repository != "b2:my-bucket" {
			t.Errorf("Repository = %v, want b2:my-bucket", cfg.Repository)
		}
		if cfg.Password != "mypassword" {
			t.Errorf("Password = %v, want mypassword", cfg.Password)
		}
		if cfg.Env["B2_ACCOUNT_ID"] != "account123" {
			t.Errorf("B2_ACCOUNT_ID = %v, want account123", cfg.Env["B2_ACCOUNT_ID"])
		}
		if cfg.Env["B2_ACCOUNT_KEY"] != "key456" {
			t.Errorf("B2_ACCOUNT_KEY = %v, want key456", cfg.Env["B2_ACCOUNT_KEY"])
		}
	})

	t.Run("with prefix", func(t *testing.T) {
		b := &B2Backend{
			Bucket:         "my-bucket",
			Prefix:         "backups",
			AccountID:      "account123",
			ApplicationKey: "key456",
		}
		cfg := b.ToResticConfig("pass")

		if cfg.Repository != "b2:my-bucket:backups" {
			t.Errorf("Repository = %v, want b2:my-bucket:backups", cfg.Repository)
		}
	})

	t.Run("empty password", func(t *testing.T) {
		b := &B2Backend{
			Bucket:         "bucket",
			AccountID:      "acct",
			ApplicationKey: "key",
		}
		cfg := b.ToResticConfig("")

		if cfg.Password != "" {
			t.Errorf("Password = %v, want empty", cfg.Password)
		}
	})
}

func TestB2Backend_TestConnection_InvalidConfig(t *testing.T) {
	b := &B2Backend{
		Bucket: "my-bucket",
		// Missing AccountID and ApplicationKey
	}

	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for invalid config, got nil")
	}
}

func TestB2Backend_TestConnection_NetworkError(t *testing.T) {
	// TestConnection with valid config will attempt to reach the B2 API.
	// In most environments this will either succeed (with invalid creds → 401)
	// or fail with a network error. Both paths exercise the TestConnection code.
	b := &B2Backend{
		Bucket:         "test-bucket",
		AccountID:      "fake-account-id",
		ApplicationKey: "fake-application-key",
	}

	// We just verify the function returns an error (invalid creds or network issue)
	// without panicking. This exercises the HTTP request, response handling, and
	// error wrapping code paths.
	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error with fake credentials, got nil")
	}
}
