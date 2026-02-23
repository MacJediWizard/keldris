// Package backends provides storage backend implementations for Restic backups.
package backends

import (
	"encoding/json"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
)

// ResticConfig holds configuration for a Restic backup operation.
type ResticConfig struct {
	Repository string
	Password   string
	Env        map[string]string
}

// Backend defines the interface for backup storage backends.
type Backend interface {
	// Type returns the repository type.
	Type() models.RepositoryType

	// ToResticConfig converts the backend configuration to a ResticConfig.
	ToResticConfig(password string) ResticConfig

	// Validate checks if the configuration is valid.
	Validate() error

	// TestConnection tests the backend connection.
	TestConnection() error
}

// ParseBackend parses a backend configuration from JSON based on the repository type.
func ParseBackend(repoType models.RepositoryType, configJSON []byte) (Backend, error) {
	switch repoType {
	case models.RepositoryTypeLocal:
		var b LocalBackend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse local backend config: %w", err)
		}
		return &b, nil

	case models.RepositoryTypeS3:
		var b S3Backend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse s3 backend config: %w", err)
		}
		return &b, nil

	case models.RepositoryTypeB2:
		var b B2Backend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse b2 backend config: %w", err)
		}
		return &b, nil

	case models.RepositoryTypeSFTP:
		var b SFTPBackend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse sftp backend config: %w", err)
		}
		return &b, nil

	case models.RepositoryTypeRest:
		var b RestBackend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse rest backend config: %w", err)
		}
		return &b, nil

	case models.RepositoryTypeDropbox:
		var b DropboxBackend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse dropbox backend config: %w", err)
		}
		return &b, nil

	case models.RepositoryTypeAzure:
		var b AzureBackend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse azure backend config: %w", err)
		}
		return &b, nil

	case models.RepositoryTypeGCS:
		var b GCSBackend
		if err := json.Unmarshal(configJSON, &b); err != nil {
			return nil, fmt.Errorf("parse gcs backend config: %w", err)
		}
		return &b, nil

	default:
		return nil, fmt.Errorf("unsupported repository type: %s", repoType)
	}
}

// BackendConfig converts a Backend to its JSON representation.
func BackendConfig(backend Backend) ([]byte, error) {
	return json.Marshal(backend)
}
