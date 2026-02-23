package backends

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/MacJediWizard/keldris/internal/models"
)

// GCSBackend represents a Google Cloud Storage backend.
type GCSBackend struct {
	BucketName      string `json:"bucket_name"`
	Prefix          string `json:"prefix,omitempty"`
	ProjectID       string `json:"project_id"`
	CredentialsJSON string `json:"credentials_json,omitempty"` // base64-encoded service account JSON
	CredentialsFile string `json:"credentials_file,omitempty"` // path to credentials file
}

// Type returns the repository type.
func (b *GCSBackend) Type() models.RepositoryType {
	return models.RepositoryTypeGCS
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *GCSBackend) ToResticConfig(password string) ResticConfig {
	// Build the repository URL: gs:bucket_name:/prefix
	repository := fmt.Sprintf("gs:%s:/", b.BucketName)
	if b.Prefix != "" {
		repository = fmt.Sprintf("gs:%s:/%s", b.BucketName, b.Prefix)
	}

	env := map[string]string{
		"GOOGLE_PROJECT_ID": b.ProjectID,
		"RESTIC_PASSWORD":   password,
	}

	// If base64-encoded credentials JSON is provided, decode it and write to a temp file
	if b.CredentialsJSON != "" {
		decoded, err := base64.StdEncoding.DecodeString(b.CredentialsJSON)
		if err == nil {
			tmpDir := os.TempDir()
			credFile := filepath.Join(tmpDir, "keldris-gcs-credentials.json")
			if writeErr := os.WriteFile(credFile, decoded, 0600); writeErr == nil {
				env["GOOGLE_APPLICATION_CREDENTIALS"] = credFile
			}
		}
	} else if b.CredentialsFile != "" {
		env["GOOGLE_APPLICATION_CREDENTIALS"] = b.CredentialsFile
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env:        env,
	}
}

// Validate checks if the configuration is valid.
func (b *GCSBackend) Validate() error {
	if b.BucketName == "" {
		return errors.New("gcs backend: bucket_name is required")
	}
	if b.ProjectID == "" {
		return errors.New("gcs backend: project_id is required")
	}
	if b.CredentialsJSON == "" && b.CredentialsFile == "" {
		return errors.New("gcs backend: either credentials_json or credentials_file is required")
	}
	return nil
}

// TestConnection validates the GCS backend configuration.
func (b *GCSBackend) TestConnection() error {
	return b.Validate()
}
