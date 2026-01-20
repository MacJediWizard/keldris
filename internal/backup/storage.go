// Package backup provides Restic backup functionality and scheduling.
// Storage backends are in the backends subpackage.
package backup

import (
	"github.com/MacJediWizard/keldris/internal/backup/backends"
	"github.com/MacJediWizard/keldris/internal/models"
)

// Backend is an alias to backends.Backend for backwards compatibility.
type Backend = backends.Backend

// Type aliases for backwards compatibility.
type (
	LocalBackend   = backends.LocalBackend
	S3Backend      = backends.S3Backend
	B2Backend      = backends.B2Backend
	SFTPBackend    = backends.SFTPBackend
	RestBackend    = backends.RestBackend
	DropboxBackend = backends.DropboxBackend
)

// ParseBackend parses a backend configuration from JSON based on the repository type.
// This is a wrapper around backends.ParseBackend for backwards compatibility.
func ParseBackend(repoType models.RepositoryType, configJSON []byte) (Backend, error) {
	return backends.ParseBackend(repoType, configJSON)
}

// BackendConfig converts a Backend to its JSON representation.
// This is a wrapper around backends.BackendConfig for backwards compatibility.
func BackendConfig(backend Backend) ([]byte, error) {
	return backends.BackendConfig(backend)
}
