package backup

import (
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestLocalBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend LocalBackend
		wantErr bool
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
		},
		{
			name:    "relative path",
			backend: LocalBackend{Path: "backups/restic"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLocalBackend_ToResticConfig(t *testing.T) {
	backend := LocalBackend{Path: "/var/backups/restic"}
	cfg := backend.ToResticConfig("mypassword")

	if cfg.Repository != "/var/backups/restic" {
		t.Errorf("Repository = %v, want /var/backups/restic", cfg.Repository)
	}
	if cfg.Password != "mypassword" {
		t.Errorf("Password = %v, want mypassword", cfg.Password)
	}
}

func TestLocalBackend_Type(t *testing.T) {
	backend := LocalBackend{}
	if backend.Type() != models.RepositoryTypeLocal {
		t.Errorf("Type() = %v, want %v", backend.Type(), models.RepositoryTypeLocal)
	}
}

func TestS3Backend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend S3Backend
		wantErr bool
	}{
		{
			name: "valid AWS S3",
			backend: S3Backend{
				Bucket:          "my-bucket",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			wantErr: false,
		},
		{
			name: "valid MinIO",
			backend: S3Backend{
				Endpoint:        "minio.example.com:9000",
				Bucket:          "my-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
			},
			wantErr: false,
		},
		{
			name: "missing bucket",
			backend: S3Backend{
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			wantErr: true,
		},
		{
			name: "missing access key",
			backend: S3Backend{
				Bucket:          "my-bucket",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			wantErr: true,
		},
		{
			name: "missing secret key",
			backend: S3Backend{
				Bucket:      "my-bucket",
				AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestS3Backend_ToResticConfig(t *testing.T) {
	tests := []struct {
		name       string
		backend    S3Backend
		wantPrefix string
	}{
		{
			name: "AWS S3",
			backend: S3Backend{
				Bucket:          "my-bucket",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "secret",
			},
			wantPrefix: "s3:s3.amazonaws.com/my-bucket",
		},
		{
			name: "AWS S3 with prefix",
			backend: S3Backend{
				Bucket:          "my-bucket",
				Prefix:          "backups",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "secret",
			},
			wantPrefix: "s3:s3.amazonaws.com/my-bucket/backups",
		},
		{
			name: "MinIO with SSL",
			backend: S3Backend{
				Endpoint:        "minio.example.com:9000",
				Bucket:          "my-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UseSSL:          true,
			},
			wantPrefix: "s3:https://minio.example.com:9000/my-bucket",
		},
		{
			name: "MinIO without SSL",
			backend: S3Backend{
				Endpoint:        "minio.example.com:9000",
				Bucket:          "my-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UseSSL:          false,
			},
			wantPrefix: "s3:http://minio.example.com:9000/my-bucket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.backend.ToResticConfig("mypassword")
			if cfg.Repository != tt.wantPrefix {
				t.Errorf("Repository = %v, want %v", cfg.Repository, tt.wantPrefix)
			}
			if cfg.Env["AWS_ACCESS_KEY_ID"] != tt.backend.AccessKeyID {
				t.Errorf("AWS_ACCESS_KEY_ID = %v, want %v", cfg.Env["AWS_ACCESS_KEY_ID"], tt.backend.AccessKeyID)
			}
			if cfg.Env["AWS_SECRET_ACCESS_KEY"] != tt.backend.SecretAccessKey {
				t.Errorf("AWS_SECRET_ACCESS_KEY = %v, want %v", cfg.Env["AWS_SECRET_ACCESS_KEY"], tt.backend.SecretAccessKey)
			}
		})
	}
}

func TestB2Backend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend B2Backend
		wantErr bool
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
			name: "missing bucket",
			backend: B2Backend{
				AccountID:      "account123",
				ApplicationKey: "key456",
			},
			wantErr: true,
		},
		{
			name: "missing account id",
			backend: B2Backend{
				Bucket:         "my-bucket",
				ApplicationKey: "key456",
			},
			wantErr: true,
		},
		{
			name: "missing application key",
			backend: B2Backend{
				Bucket:    "my-bucket",
				AccountID: "account123",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestB2Backend_ToResticConfig(t *testing.T) {
	backend := B2Backend{
		Bucket:         "my-bucket",
		Prefix:         "backups",
		AccountID:      "account123",
		ApplicationKey: "key456",
	}

	cfg := backend.ToResticConfig("mypassword")

	if cfg.Repository != "b2:my-bucket:backups" {
		t.Errorf("Repository = %v, want b2:my-bucket:backups", cfg.Repository)
	}
	if cfg.Env["B2_ACCOUNT_ID"] != "account123" {
		t.Errorf("B2_ACCOUNT_ID = %v, want account123", cfg.Env["B2_ACCOUNT_ID"])
	}
	if cfg.Env["B2_ACCOUNT_KEY"] != "key456" {
		t.Errorf("B2_ACCOUNT_KEY = %v, want key456", cfg.Env["B2_ACCOUNT_KEY"])
	}
}

func TestSFTPBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend SFTPBackend
		wantErr bool
	}{
		{
			name: "valid",
			backend: SFTPBackend{
				Host: "backup.example.com",
				User: "backupuser",
				Path: "/var/backups/restic",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			backend: SFTPBackend{
				User: "backupuser",
				Path: "/var/backups/restic",
			},
			wantErr: true,
		},
		{
			name: "missing user",
			backend: SFTPBackend{
				Host: "backup.example.com",
				Path: "/var/backups/restic",
			},
			wantErr: true,
		},
		{
			name: "missing path",
			backend: SFTPBackend{
				Host: "backup.example.com",
				User: "backupuser",
			},
			wantErr: true,
		},
		{
			name: "relative path",
			backend: SFTPBackend{
				Host: "backup.example.com",
				User: "backupuser",
				Path: "backups/restic",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSFTPBackend_ToResticConfig(t *testing.T) {
	tests := []struct {
		name       string
		backend    SFTPBackend
		wantPrefix string
	}{
		{
			name: "default port",
			backend: SFTPBackend{
				Host: "backup.example.com",
				User: "backupuser",
				Path: "/var/backups/restic",
			},
			wantPrefix: "sftp:backupuser@backup.example.com:22/var/backups/restic",
		},
		{
			name: "custom port",
			backend: SFTPBackend{
				Host: "backup.example.com",
				Port: 2222,
				User: "backupuser",
				Path: "/var/backups/restic",
			},
			wantPrefix: "sftp:backupuser@backup.example.com:2222/var/backups/restic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.backend.ToResticConfig("mypassword")
			if cfg.Repository != tt.wantPrefix {
				t.Errorf("Repository = %v, want %v", cfg.Repository, tt.wantPrefix)
			}
		})
	}
}

func TestRestBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend RestBackend
		wantErr bool
	}{
		{
			name:    "valid URL",
			backend: RestBackend{URL: "https://rest.example.com:8000/"},
			wantErr: false,
		},
		{
			name:    "valid URL with rest prefix",
			backend: RestBackend{URL: "rest:https://rest.example.com:8000/"},
			wantErr: false,
		},
		{
			name:    "missing URL",
			backend: RestBackend{URL: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.backend.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRestBackend_ToResticConfig(t *testing.T) {
	tests := []struct {
		name       string
		backend    RestBackend
		wantPrefix string
	}{
		{
			name:       "without auth",
			backend:    RestBackend{URL: "https://rest.example.com:8000/"},
			wantPrefix: "rest:https://rest.example.com:8000/",
		},
		{
			name: "with auth",
			backend: RestBackend{
				URL:      "https://rest.example.com:8000/",
				Username: "user",
				Password: "pass",
			},
			wantPrefix: "rest:https://user:pass@rest.example.com:8000/",
		},
		{
			name:       "already has rest prefix",
			backend:    RestBackend{URL: "rest:https://rest.example.com:8000/"},
			wantPrefix: "rest:https://rest.example.com:8000/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.backend.ToResticConfig("mypassword")
			if cfg.Repository != tt.wantPrefix {
				t.Errorf("Repository = %v, want %v", cfg.Repository, tt.wantPrefix)
			}
		})
	}
}

func TestParseBackend(t *testing.T) {
	tests := []struct {
		name     string
		repoType models.RepositoryType
		config   string
		wantType models.RepositoryType
		wantErr  bool
	}{
		{
			name:     "local backend",
			repoType: models.RepositoryTypeLocal,
			config:   `{"path": "/var/backups/restic"}`,
			wantType: models.RepositoryTypeLocal,
			wantErr:  false,
		},
		{
			name:     "s3 backend",
			repoType: models.RepositoryTypeS3,
			config:   `{"bucket": "my-bucket", "access_key_id": "key", "secret_access_key": "secret"}`,
			wantType: models.RepositoryTypeS3,
			wantErr:  false,
		},
		{
			name:     "b2 backend",
			repoType: models.RepositoryTypeB2,
			config:   `{"bucket": "my-bucket", "account_id": "acct", "application_key": "key"}`,
			wantType: models.RepositoryTypeB2,
			wantErr:  false,
		},
		{
			name:     "sftp backend",
			repoType: models.RepositoryTypeSFTP,
			config:   `{"host": "example.com", "user": "user", "path": "/backups"}`,
			wantType: models.RepositoryTypeSFTP,
			wantErr:  false,
		},
		{
			name:     "rest backend",
			repoType: models.RepositoryTypeRest,
			config:   `{"url": "https://rest.example.com:8000/"}`,
			wantType: models.RepositoryTypeRest,
			wantErr:  false,
		},
		{
			name:     "invalid json",
			repoType: models.RepositoryTypeLocal,
			config:   `{invalid}`,
			wantType: "",
			wantErr:  true,
		},
		{
			name:     "unknown type",
			repoType: models.RepositoryType("unknown"),
			config:   `{}`,
			wantType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := ParseBackend(tt.repoType, []byte(tt.config))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBackend() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && backend.Type() != tt.wantType {
				t.Errorf("ParseBackend() type = %v, want %v", backend.Type(), tt.wantType)
			}
		})
	}
}

func TestIsRestURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"rest:https://example.com", true},
		{"rest:http://example.com", true},
		{"https://example.com", false},
		{"http://example.com", false},
		{"rest", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := isRestURL(tt.url); got != tt.want {
				t.Errorf("isRestURL(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
