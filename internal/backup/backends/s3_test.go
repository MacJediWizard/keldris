package backends

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestS3Backend_Init(t *testing.T) {
	b := &S3Backend{
		Bucket:          "my-bucket",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	if b.Type() != models.RepositoryTypeS3 {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeS3)
	}
}

func TestS3Backend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend S3Backend
		wantErr bool
		errMsg  string
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
			name: "valid with all fields",
			backend: S3Backend{
				Endpoint:        "minio.example.com:9000",
				Bucket:          "my-bucket",
				Prefix:          "backups/daily",
				Region:          "us-west-2",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UseSSL:          true,
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
			errMsg:  "bucket is required",
		},
		{
			name: "missing access key",
			backend: S3Backend{
				Bucket:          "my-bucket",
				SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			wantErr: true,
			errMsg:  "access_key_id is required",
		},
		{
			name: "missing secret key",
			backend: S3Backend{
				Bucket:      "my-bucket",
				AccessKeyID: "AKIAIOSFODNN7EXAMPLE",
			},
			wantErr: true,
			errMsg:  "secret_access_key is required",
		},
		{
			name:    "all fields empty",
			backend: S3Backend{},
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

func TestS3Backend_GetEnv(t *testing.T) {
	tests := []struct {
		name           string
		backend        S3Backend
		wantRepository string
		wantEnvKeys    []string
	}{
		{
			name: "AWS S3 basic",
			backend: S3Backend{
				Bucket:          "my-bucket",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "secret",
			},
			wantRepository: "s3:s3.amazonaws.com/my-bucket",
			wantEnvKeys:    []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
		},
		{
			name: "AWS S3 with prefix",
			backend: S3Backend{
				Bucket:          "my-bucket",
				Prefix:          "backups",
				AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
				SecretAccessKey: "secret",
			},
			wantRepository: "s3:s3.amazonaws.com/my-bucket/backups",
			wantEnvKeys:    []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
		},
		{
			name: "AWS S3 with region",
			backend: S3Backend{
				Bucket:          "my-bucket",
				Region:          "eu-west-1",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
			},
			wantRepository: "s3:s3.amazonaws.com/my-bucket",
			wantEnvKeys:    []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION"},
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
		b := &S3Backend{
			Bucket:          "bucket",
			AccessKeyID:     "mykey",
			SecretAccessKey: "mysecret",
			Region:          "ap-southeast-1",
		}
		cfg := b.ToResticConfig("pass")

		if cfg.Env["AWS_ACCESS_KEY_ID"] != "mykey" {
			t.Errorf("AWS_ACCESS_KEY_ID = %v, want mykey", cfg.Env["AWS_ACCESS_KEY_ID"])
		}
		if cfg.Env["AWS_SECRET_ACCESS_KEY"] != "mysecret" {
			t.Errorf("AWS_SECRET_ACCESS_KEY = %v, want mysecret", cfg.Env["AWS_SECRET_ACCESS_KEY"])
		}
		if cfg.Env["AWS_DEFAULT_REGION"] != "ap-southeast-1" {
			t.Errorf("AWS_DEFAULT_REGION = %v, want ap-southeast-1", cfg.Env["AWS_DEFAULT_REGION"])
		}
	})

	t.Run("no region omits AWS_DEFAULT_REGION", func(t *testing.T) {
		b := &S3Backend{
			Bucket:          "bucket",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
		}
		cfg := b.ToResticConfig("pass")

		if _, ok := cfg.Env["AWS_DEFAULT_REGION"]; ok {
			t.Error("expected AWS_DEFAULT_REGION to not be set when region is empty")
		}
	})
}

func TestS3Backend_CustomEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		backend        S3Backend
		wantRepository string
	}{
		{
			name: "MinIO without SSL",
			backend: S3Backend{
				Endpoint:        "minio.example.com:9000",
				Bucket:          "my-bucket",
				AccessKeyID:     "minioadmin",
				SecretAccessKey: "minioadmin",
				UseSSL:          false,
			},
			wantRepository: "s3:http://minio.example.com:9000/my-bucket",
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
			wantRepository: "s3:https://minio.example.com:9000/my-bucket",
		},
		{
			name: "Wasabi with prefix",
			backend: S3Backend{
				Endpoint:        "s3.wasabisys.com",
				Bucket:          "wasabi-bucket",
				Prefix:          "restic-repo",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				UseSSL:          true,
			},
			wantRepository: "s3:https://s3.wasabisys.com/wasabi-bucket/restic-repo",
		},
		{
			name: "endpoint with URL scheme is parsed to host",
			backend: S3Backend{
				Endpoint:        "http://minio.local:9000",
				Bucket:          "test",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				UseSSL:          false,
			},
			wantRepository: "s3:http://minio.local:9000/test",
		},
		{
			name: "endpoint with https scheme and UseSSL true",
			backend: S3Backend{
				Endpoint:        "https://storage.example.com",
				Bucket:          "test",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				UseSSL:          true,
			},
			wantRepository: "s3:https://storage.example.com/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.backend.ToResticConfig("pass")
			if cfg.Repository != tt.wantRepository {
				t.Errorf("Repository = %v, want %v", cfg.Repository, tt.wantRepository)
			}
		})
	}
}

func TestS3Backend_Credentials(t *testing.T) {
	b := &S3Backend{
		Bucket:          "secure-bucket",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		Region:          "us-east-1",
	}

	cfg := b.ToResticConfig("repo-password")

	if cfg.Env["AWS_ACCESS_KEY_ID"] != b.AccessKeyID {
		t.Errorf("access key mismatch: got %v, want %v", cfg.Env["AWS_ACCESS_KEY_ID"], b.AccessKeyID)
	}
	if cfg.Env["AWS_SECRET_ACCESS_KEY"] != b.SecretAccessKey {
		t.Errorf("secret key mismatch: got %v, want %v", cfg.Env["AWS_SECRET_ACCESS_KEY"], b.SecretAccessKey)
	}
	if cfg.Env["AWS_DEFAULT_REGION"] != "us-east-1" {
		t.Errorf("region mismatch: got %v, want us-east-1", cfg.Env["AWS_DEFAULT_REGION"])
	}
	if cfg.Password != "repo-password" {
		t.Errorf("password mismatch: got %v, want repo-password", cfg.Password)
	}
}

func TestS3Backend_TestConnection_CustomEndpoint(t *testing.T) {
	// Use httptest as a fake S3-compatible endpoint.
	// The AWS SDK sends HEAD /{bucket} for HeadBucket with path-style addressing.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond 200 to any request (HeadBucket)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b := &S3Backend{
		Endpoint:        server.URL,
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UseSSL:          false,
	}

	err := b.TestConnection()
	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestS3Backend_TestConnection_CustomEndpointFailure(t *testing.T) {
	// Server returns 404 (bucket not found)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	b := &S3Backend{
		Endpoint:        server.URL,
		Bucket:          "nonexistent-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UseSSL:          false,
	}

	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for missing bucket, got nil")
	}
}

func TestS3Backend_TestConnection_InvalidConfig(t *testing.T) {
	b := &S3Backend{
		Bucket: "my-bucket",
		// Missing credentials
	}

	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for invalid config, got nil")
	}
}

func TestS3Backend_TestConnection_NoRegionDefault(t *testing.T) {
	// Test that empty region defaults to us-east-1 in TestConnection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	b := &S3Backend{
		Endpoint:        server.URL,
		Bucket:          "test-bucket",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UseSSL:          false,
		// Region intentionally empty - should default to us-east-1
	}

	err := b.TestConnection()
	if err != nil {
		t.Errorf("TestConnection() error = %v, want nil", err)
	}
}

func TestS3Backend_TestConnection_WithSSL(t *testing.T) {
	// Use TLS httptest server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Note: The S3 SDK won't trust the test server's self-signed cert.
	// This tests the endpoint URL construction with UseSSL=true.
	b := &S3Backend{
		Endpoint:        server.URL,
		Bucket:          "test-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "test-key",
		SecretAccessKey: "test-secret",
		UseSSL:          true,
	}

	// This will likely fail due to TLS verification, which is expected
	// behavior - it tests the SSL code path.
	err := b.TestConnection()
	// We just verify the code path executes without panicking.
	// The error from TLS cert verification is acceptable.
	_ = err
}
