package backends

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/MacJediWizard/keldris/internal/models"
)

// LocalBackend represents a local filesystem storage backend.
type LocalBackend struct {
	Path string `json:"path"`
}

// Type returns the repository type.
func (b *LocalBackend) Type() models.RepositoryType {
	return models.RepositoryTypeLocal
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *LocalBackend) ToResticConfig(password string) ResticConfig {
	return ResticConfig{
		Repository: b.Path,
		Password:   password,
	}
}

// Validate checks if the configuration is valid.
func (b *LocalBackend) Validate() error {
	if b.Path == "" {
		return errors.New("local backend: path is required")
	}
	if !filepath.IsAbs(b.Path) {
		return errors.New("local backend: path must be absolute")
	}
	return nil
}

// TestConnection tests the local backend connection by checking if the path
// exists and is accessible.
func (b *LocalBackend) TestConnection() error {
	if err := b.Validate(); err != nil {
		return err
	}

	// Check if parent directory exists and is writable
	parentDir := filepath.Dir(b.Path)
	info, err := os.Stat(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.New("local backend: parent directory does not exist")
		}
		return err
	}

	if !info.IsDir() {
		return errors.New("local backend: parent path is not a directory")
	}

	// Check if the path itself exists
	_, err = os.Stat(b.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
