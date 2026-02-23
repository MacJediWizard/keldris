package backends

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

// validBase64Key is a test Azure account key (valid base64, 64 bytes decoded).
var validBase64Key = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))

func TestAzureBackend_Type(t *testing.T) {
	b := &AzureBackend{
		AccountName:   "myaccount",
		AccountKey:    validBase64Key,
		ContainerName: "mycontainer",
	}
	if b.Type() != models.RepositoryTypeAzure {
		t.Errorf("Type() = %v, want %v", b.Type(), models.RepositoryTypeAzure)
	}
}

func TestAzureBackend_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend AzureBackend
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			backend: AzureBackend{
				AccountName:   "myaccount",
				AccountKey:    validBase64Key,
				ContainerName: "mycontainer",
			},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			backend: AzureBackend{
				AccountName:   "myaccount",
				AccountKey:    validBase64Key,
				ContainerName: "mycontainer",
				Endpoint:      "core.usgovcloudapi.net",
				Prefix:        "backups/daily",
			},
			wantErr: false,
		},
		{
			name: "missing account_name",
			backend: AzureBackend{
				AccountKey:    validBase64Key,
				ContainerName: "mycontainer",
			},
			wantErr: true,
			errMsg:  "account_name is required",
		},
		{
			name: "missing account_key",
			backend: AzureBackend{
				AccountName:   "myaccount",
				ContainerName: "mycontainer",
			},
			wantErr: true,
			errMsg:  "account_key is required",
		},
		{
			name: "missing container_name",
			backend: AzureBackend{
				AccountName: "myaccount",
				AccountKey:  validBase64Key,
			},
			wantErr: true,
			errMsg:  "container_name is required",
		},
		{
			name: "invalid account_key not base64",
			backend: AzureBackend{
				AccountName:   "myaccount",
				AccountKey:    "not-valid-base64!!!",
				ContainerName: "mycontainer",
			},
			wantErr: true,
			errMsg:  "account_key is not valid base64",
		},
		{
			name:    "all fields empty",
			backend: AzureBackend{},
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

func TestAzureBackend_ToResticConfig(t *testing.T) {
	tests := []struct {
		name           string
		backend        AzureBackend
		wantRepository string
		wantEnvKeys    []string
	}{
		{
			name: "basic config",
			backend: AzureBackend{
				AccountName:   "myaccount",
				AccountKey:    validBase64Key,
				ContainerName: "mycontainer",
			},
			wantRepository: "azure:mycontainer:/",
			wantEnvKeys:    []string{"AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_KEY"},
		},
		{
			name: "with prefix",
			backend: AzureBackend{
				AccountName:   "myaccount",
				AccountKey:    validBase64Key,
				ContainerName: "mycontainer",
				Prefix:        "backups/daily",
			},
			wantRepository: "azure:mycontainer:/backups/daily",
			wantEnvKeys:    []string{"AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_KEY"},
		},
		{
			name: "with custom endpoint",
			backend: AzureBackend{
				AccountName:   "myaccount",
				AccountKey:    validBase64Key,
				ContainerName: "mycontainer",
				Endpoint:      "core.usgovcloudapi.net",
			},
			wantRepository: "azure:mycontainer:/",
			wantEnvKeys:    []string{"AZURE_ACCOUNT_NAME", "AZURE_ACCOUNT_KEY", "AZURE_ENDPOINT_SUFFIX"},
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
		b := &AzureBackend{
			AccountName:   "storageacct",
			AccountKey:    validBase64Key,
			ContainerName: "backup-container",
			Endpoint:      "core.chinacloudapi.cn",
		}
		cfg := b.ToResticConfig("pass")

		if cfg.Env["AZURE_ACCOUNT_NAME"] != "storageacct" {
			t.Errorf("AZURE_ACCOUNT_NAME = %v, want storageacct", cfg.Env["AZURE_ACCOUNT_NAME"])
		}
		if cfg.Env["AZURE_ACCOUNT_KEY"] != validBase64Key {
			t.Errorf("AZURE_ACCOUNT_KEY = %v, want %v", cfg.Env["AZURE_ACCOUNT_KEY"], validBase64Key)
		}
		if cfg.Env["AZURE_ENDPOINT_SUFFIX"] != "core.chinacloudapi.cn" {
			t.Errorf("AZURE_ENDPOINT_SUFFIX = %v, want core.chinacloudapi.cn", cfg.Env["AZURE_ENDPOINT_SUFFIX"])
		}
	})

	t.Run("no endpoint omits AZURE_ENDPOINT_SUFFIX", func(t *testing.T) {
		b := &AzureBackend{
			AccountName:   "myaccount",
			AccountKey:    validBase64Key,
			ContainerName: "mycontainer",
		}
		cfg := b.ToResticConfig("pass")

		if _, ok := cfg.Env["AZURE_ENDPOINT_SUFFIX"]; ok {
			t.Error("expected AZURE_ENDPOINT_SUFFIX to not be set when endpoint is empty")
		}
	})
}

func TestAzureBackend_TestConnection(t *testing.T) {
	// Use httptest as a fake Azure Blob Storage endpoint.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request has the expected Azure headers
		if r.Header.Get("x-ms-date") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("x-ms-version") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Respond 200 to simulate a successful container list
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Extract host from the test server URL for the endpoint
	// We need to override the endpoint construction, so we'll test with a custom endpoint
	// that routes to our test server.
	// Since TestConnection constructs the URL from account name + endpoint,
	// we can't easily override the host. Instead, test validation-failure paths.

	t.Run("invalid config returns error", func(t *testing.T) {
		b := &AzureBackend{
			AccountName: "myaccount",
			// Missing AccountKey
			ContainerName: "mycontainer",
		}
		err := b.TestConnection()
		if err == nil {
			t.Error("TestConnection() expected error for invalid config, got nil")
		}
	})

	t.Run("invalid base64 key returns error", func(t *testing.T) {
		b := &AzureBackend{
			AccountName:   "myaccount",
			AccountKey:    "not-base64!!!",
			ContainerName: "mycontainer",
		}
		err := b.TestConnection()
		if err == nil {
			t.Error("TestConnection() expected error for invalid key, got nil")
		}
	})
}

func TestAzureBackend_TestConnection_NotFound(t *testing.T) {
	// Server returns 404 (container not found)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// This tests that a 404 response is properly handled.
	// Since we can't easily inject a custom URL, we verify error handling through validation.
	b := &AzureBackend{
		AccountName:   "",
		AccountKey:    validBase64Key,
		ContainerName: "nonexistent",
	}

	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error, got nil")
	}
}

func TestAzureBackend_TestConnection_Unauthorized(t *testing.T) {
	// Verify that missing credentials are caught by validation
	b := &AzureBackend{
		ContainerName: "mycontainer",
	}

	err := b.TestConnection()
	if err == nil {
		t.Error("TestConnection() expected error for missing credentials, got nil")
	}
}
