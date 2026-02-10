package backends

import (
	"encoding/json"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestBackendFactory_Create(t *testing.T) {
	tests := []struct {
		name     string
		repoType models.RepositoryType
		config   string
		wantType models.RepositoryType
	}{
		{
			name:     "local backend",
			repoType: models.RepositoryTypeLocal,
			config:   `{"path": "/var/backups/restic"}`,
			wantType: models.RepositoryTypeLocal,
		},
		{
			name:     "s3 backend",
			repoType: models.RepositoryTypeS3,
			config:   `{"bucket": "my-bucket", "access_key_id": "key", "secret_access_key": "secret"}`,
			wantType: models.RepositoryTypeS3,
		},
		{
			name:     "b2 backend",
			repoType: models.RepositoryTypeB2,
			config:   `{"bucket": "my-bucket", "account_id": "acct", "application_key": "key"}`,
			wantType: models.RepositoryTypeB2,
		},
		{
			name:     "sftp backend",
			repoType: models.RepositoryTypeSFTP,
			config:   `{"host": "example.com", "user": "user", "path": "/backups"}`,
			wantType: models.RepositoryTypeSFTP,
		},
		{
			name:     "rest backend",
			repoType: models.RepositoryTypeRest,
			config:   `{"url": "https://rest.example.com:8000/"}`,
			wantType: models.RepositoryTypeRest,
		},
		{
			name:     "dropbox backend",
			repoType: models.RepositoryTypeDropbox,
			config:   `{"remote_name": "myremote", "path": "/backups"}`,
			wantType: models.RepositoryTypeDropbox,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := ParseBackend(tt.repoType, []byte(tt.config))
			if err != nil {
				t.Fatalf("ParseBackend() error = %v", err)
			}
			if backend.Type() != tt.wantType {
				t.Errorf("Type() = %v, want %v", backend.Type(), tt.wantType)
			}
		})
	}
}

func TestBackendFactory_UnknownType(t *testing.T) {
	tests := []struct {
		name     string
		repoType models.RepositoryType
		config   string
	}{
		{
			name:     "unknown type",
			repoType: models.RepositoryType("unknown"),
			config:   `{}`,
		},
		{
			name:     "empty type",
			repoType: models.RepositoryType(""),
			config:   `{}`,
		},
		{
			name:     "random string type",
			repoType: models.RepositoryType("ftp"),
			config:   `{"host": "ftp.example.com"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseBackend(tt.repoType, []byte(tt.config))
			if err == nil {
				t.Error("ParseBackend() expected error for unknown type, got nil")
			}
			if !contains(err.Error(), "unsupported repository type") {
				t.Errorf("ParseBackend() error = %q, want to contain 'unsupported repository type'", err.Error())
			}
		})
	}
}

func TestBackendFactory_InvalidJSON(t *testing.T) {
	types := []models.RepositoryType{
		models.RepositoryTypeLocal,
		models.RepositoryTypeS3,
		models.RepositoryTypeB2,
		models.RepositoryTypeSFTP,
		models.RepositoryTypeRest,
		models.RepositoryTypeDropbox,
	}

	for _, repoType := range types {
		t.Run(string(repoType), func(t *testing.T) {
			_, err := ParseBackend(repoType, []byte(`{invalid json}`))
			if err == nil {
				t.Errorf("ParseBackend(%s) expected error for invalid JSON, got nil", repoType)
			}
		})
	}
}

func TestBackendConfig_RoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		repoType models.RepositoryType
		backend  Backend
	}{
		{
			name:     "local",
			repoType: models.RepositoryTypeLocal,
			backend:  &LocalBackend{Path: "/var/backups"},
		},
		{
			name:     "s3",
			repoType: models.RepositoryTypeS3,
			backend: &S3Backend{
				Bucket:          "my-bucket",
				Prefix:          "backups",
				Region:          "us-east-1",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				UseSSL:          true,
			},
		},
		{
			name:     "b2",
			repoType: models.RepositoryTypeB2,
			backend: &B2Backend{
				Bucket:         "my-bucket",
				Prefix:         "restic",
				AccountID:      "acct",
				ApplicationKey: "appkey",
			},
		},
		{
			name:     "sftp",
			repoType: models.RepositoryTypeSFTP,
			backend: &SFTPBackend{
				Host:     "backup.example.com",
				Port:     2222,
				User:     "backupuser",
				Path:     "/var/backups",
				Password: "secret",
			},
		},
		{
			name:     "rest",
			repoType: models.RepositoryTypeRest,
			backend: &RestBackend{
				URL:      "https://rest.example.com:8000/",
				Username: "user",
				Password: "pass",
			},
		},
		{
			name:     "dropbox",
			repoType: models.RepositoryTypeDropbox,
			backend: &DropboxBackend{
				RemoteName: "myremote",
				Path:       "/backups",
				Token:      "token123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := BackendConfig(tt.backend)
			if err != nil {
				t.Fatalf("BackendConfig() error = %v", err)
			}

			// Verify it's valid JSON
			var raw map[string]interface{}
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("BackendConfig() produced invalid JSON: %v", err)
			}

			// Parse back
			parsed, err := ParseBackend(tt.repoType, data)
			if err != nil {
				t.Fatalf("ParseBackend() error = %v", err)
			}

			// Verify type matches
			if parsed.Type() != tt.backend.Type() {
				t.Errorf("round-trip Type() = %v, want %v", parsed.Type(), tt.backend.Type())
			}

			// Re-marshal and compare
			data2, err := BackendConfig(parsed)
			if err != nil {
				t.Fatalf("BackendConfig() round-trip error = %v", err)
			}

			if string(data) != string(data2) {
				t.Errorf("round-trip JSON mismatch:\n  got:  %s\n  want: %s", data2, data)
			}
		})
	}
}
