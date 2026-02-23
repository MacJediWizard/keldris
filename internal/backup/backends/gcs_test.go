package backends

import (
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestGCSBackend_Type(t *testing.T) {
	b := &GCSBackend{
		BucketName: "my-bucket",
		ProjectID:  "my-project",
	}
	if b.Type() != models.RepositoryTypeGCS {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeGCS)
	}
}

func TestGCSBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend GCSBackend
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with credentials_json",
			backend: GCSBackend{
				BucketName:      "my-bucket",
				ProjectID:       "my-project",
				CredentialsJSON: "eyJ0eXBlIjoic2VydmljZV9hY2NvdW50In0=", // base64 of {"type":"service_account"}
			},
			wantErr: false,
		},
		{
			name: "valid with credentials_file",
			backend: GCSBackend{
				BucketName:      "my-bucket",
				ProjectID:       "my-project",
				CredentialsFile: "/path/to/credentials.json",
			},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			backend: GCSBackend{
				BucketName:      "my-bucket",
				Prefix:          "backups/daily",
				ProjectID:       "my-project",
				CredentialsJSON: "eyJ0eXBlIjoic2VydmljZV9hY2NvdW50In0=",
			},
			wantErr: false,
		},
		{
			name: "missing bucket_name",
			backend: GCSBackend{
				ProjectID:       "my-project",
				CredentialsJSON: "eyJ0eXBlIjoic2VydmljZV9hY2NvdW50In0=",
			},
			wantErr: true,
			errMsg:  "bucket_name is required",
		},
		{
			name: "missing project_id",
			backend: GCSBackend{
				BucketName:      "my-bucket",
				CredentialsJSON: "eyJ0eXBlIjoic2VydmljZV9hY2NvdW50In0=",
			},
			wantErr: true,
			errMsg:  "project_id is required",
		},
		{
			name: "missing credentials",
			backend: GCSBackend{
				BucketName: "my-bucket",
				ProjectID:  "my-project",
			},
			wantErr: true,
			errMsg:  "either credentials_json or credentials_file is required",
		},
		{
			name:    "all fields empty",
			backend: GCSBackend{},
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

func TestGCSBackend_ToResticConfig(t *testing.T) {
	tests := []struct {
		name           string
		backend        GCSBackend
		wantRepository string
		wantEnvKeys    []string
	}{
		{
			name: "basic without prefix",
			backend: GCSBackend{
				BucketName:      "my-bucket",
				ProjectID:       "my-project",
				CredentialsFile: "/path/to/creds.json",
			},
			wantRepository: "gs:my-bucket:/",
			wantEnvKeys:    []string{"GOOGLE_PROJECT_ID", "GOOGLE_APPLICATION_CREDENTIALS", "RESTIC_PASSWORD"},
		},
		{
			name: "with prefix",
			backend: GCSBackend{
				BucketName:      "my-bucket",
				Prefix:          "backups/daily",
				ProjectID:       "my-project",
				CredentialsFile: "/path/to/creds.json",
			},
			wantRepository: "gs:my-bucket:/backups/daily",
			wantEnvKeys:    []string{"GOOGLE_PROJECT_ID", "GOOGLE_APPLICATION_CREDENTIALS", "RESTIC_PASSWORD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.backend.ToResticConfig("testpass")

			if cfg.Repository != tt.wantRepository {
				t.Errorf("Repository = %v, want %v", cfg.Repository, tt.wantRepository)
			}
			if cfg.Password != "testpass" {
				t.Errorf("Password = %v, want testpass", cfg.Password)
			}
			for _, key := range tt.wantEnvKeys {
				if _, ok := cfg.Env[key]; !ok {
					t.Errorf("expected env key %q to be set", key)
				}
			}
		})
	}

	t.Run("env values match backend fields", func(t *testing.T) {
		b := &GCSBackend{
			BucketName:      "bucket",
			ProjectID:       "my-project-123",
			CredentialsFile: "/tmp/gcs-creds.json",
		}
		cfg := b.ToResticConfig("pass")

		if cfg.Env["GOOGLE_PROJECT_ID"] != "my-project-123" {
			t.Errorf("GOOGLE_PROJECT_ID = %v, want my-project-123", cfg.Env["GOOGLE_PROJECT_ID"])
		}
		if cfg.Env["GOOGLE_APPLICATION_CREDENTIALS"] != "/tmp/gcs-creds.json" {
			t.Errorf("GOOGLE_APPLICATION_CREDENTIALS = %v, want /tmp/gcs-creds.json", cfg.Env["GOOGLE_APPLICATION_CREDENTIALS"])
		}
		if cfg.Env["RESTIC_PASSWORD"] != "pass" {
			t.Errorf("RESTIC_PASSWORD = %v, want pass", cfg.Env["RESTIC_PASSWORD"])
		}
	})

	t.Run("credentials_json creates temp file", func(t *testing.T) {
		b := &GCSBackend{
			BucketName:      "bucket",
			ProjectID:       "my-project",
			CredentialsJSON: "eyJ0eXBlIjoic2VydmljZV9hY2NvdW50In0=", // base64 of {"type":"service_account"}
		}
		cfg := b.ToResticConfig("pass")

		credPath, ok := cfg.Env["GOOGLE_APPLICATION_CREDENTIALS"]
		if !ok {
			t.Fatal("expected GOOGLE_APPLICATION_CREDENTIALS to be set")
		}
		if credPath == "" {
			t.Error("expected GOOGLE_APPLICATION_CREDENTIALS to be non-empty")
		}
	})
}

func TestGCSBackend_TestConnection(t *testing.T) {
	t.Run("valid config succeeds", func(t *testing.T) {
		b := &GCSBackend{
			BucketName:      "my-bucket",
			ProjectID:       "my-project",
			CredentialsFile: "/path/to/creds.json",
		}
		if err := b.TestConnection(); err != nil {
			t.Errorf("TestConnection() error = %v, want nil", err)
		}
	})

	t.Run("invalid config fails", func(t *testing.T) {
		b := &GCSBackend{
			BucketName: "my-bucket",
			// Missing ProjectID and credentials
		}
		if err := b.TestConnection(); err == nil {
			t.Error("TestConnection() expected error for invalid config, got nil")
		}
	})
}
