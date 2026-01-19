package models

import (
	"time"

	"github.com/google/uuid"
)

// RepositoryType defines the type of storage backend.
type RepositoryType string

const (
	// RepositoryTypeLocal is a local filesystem repository.
	RepositoryTypeLocal RepositoryType = "local"
	// RepositoryTypeS3 is an S3-compatible storage repository.
	RepositoryTypeS3 RepositoryType = "s3"
	// RepositoryTypeB2 is a Backblaze B2 storage repository.
	RepositoryTypeB2 RepositoryType = "b2"
	// RepositoryTypeSFTP is an SFTP storage repository.
	RepositoryTypeSFTP RepositoryType = "sftp"
	// RepositoryTypeRest is a Restic REST server repository.
	RepositoryTypeRest RepositoryType = "rest"
	// RepositoryTypeDropbox is a Dropbox storage repository (via rclone).
	RepositoryTypeDropbox RepositoryType = "dropbox"
)

// Repository represents a backup storage destination.
type Repository struct {
	ID              uuid.UUID      `json:"id"`
	OrgID           uuid.UUID      `json:"org_id"`
	Name            string         `json:"name"`
	Type            RepositoryType `json:"type"`
	ConfigEncrypted []byte         `json:"-"` // Encrypted, never expose in JSON
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// NewRepository creates a new Repository with the given details.
func NewRepository(orgID uuid.UUID, name string, repoType RepositoryType, configEncrypted []byte) *Repository {
	now := time.Now()
	return &Repository{
		ID:              uuid.New(),
		OrgID:           orgID,
		Name:            name,
		Type:            repoType,
		ConfigEncrypted: configEncrypted,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// ValidTypes returns all valid repository types.
func ValidRepositoryTypes() []RepositoryType {
	return []RepositoryType{
		RepositoryTypeLocal,
		RepositoryTypeS3,
		RepositoryTypeB2,
		RepositoryTypeSFTP,
		RepositoryTypeRest,
		RepositoryTypeDropbox,
	}
}

// IsValidType checks if the repository type is valid.
func (r *Repository) IsValidType() bool {
	for _, t := range ValidRepositoryTypes() {
		if r.Type == t {
			return true
		}
	}
	return false
}
