package models

import (
	"encoding/json"
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
	ID              uuid.UUID              `json:"id"`
	OrgID           uuid.UUID              `json:"org_id"`
	Name            string                 `json:"name"`
	Type            RepositoryType         `json:"type"`
	ConfigEncrypted []byte                 `json:"-"` // Encrypted, never expose in JSON
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
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

// SetMetadata sets the metadata from JSON bytes.
func (r *Repository) SetMetadata(data []byte) error {
	if len(data) == 0 {
		r.Metadata = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(data, &r.Metadata)
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (r *Repository) MetadataJSON() ([]byte, error) {
	if r.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(r.Metadata)
}
