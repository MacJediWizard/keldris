package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DockerRegistryType defines the type of Docker registry.
type DockerRegistryType string

const (
	// DockerRegistryTypeDockerHub is Docker Hub registry.
	DockerRegistryTypeDockerHub DockerRegistryType = "dockerhub"
	// DockerRegistryTypeGCR is Google Container Registry.
	DockerRegistryTypeGCR DockerRegistryType = "gcr"
	// DockerRegistryTypeECR is Amazon Elastic Container Registry.
	DockerRegistryTypeECR DockerRegistryType = "ecr"
	// DockerRegistryTypeACR is Azure Container Registry.
	DockerRegistryTypeACR DockerRegistryType = "acr"
	// DockerRegistryTypeGHCR is GitHub Container Registry.
	DockerRegistryTypeGHCR DockerRegistryType = "ghcr"
	// DockerRegistryTypePrivate is a self-hosted private registry.
	DockerRegistryTypePrivate DockerRegistryType = "private"
)

// DockerRegistryHealthStatus defines the health status of a registry.
type DockerRegistryHealthStatus string

const (
	// DockerRegistryHealthHealthy indicates the registry is accessible.
	DockerRegistryHealthHealthy DockerRegistryHealthStatus = "healthy"
	// DockerRegistryHealthUnhealthy indicates the registry is not accessible.
	DockerRegistryHealthUnhealthy DockerRegistryHealthStatus = "unhealthy"
	// DockerRegistryHealthUnknown indicates the health is not yet checked.
	DockerRegistryHealthUnknown DockerRegistryHealthStatus = "unknown"
)

// DockerRegistry represents a private Docker registry configuration.
type DockerRegistry struct {
	ID                  uuid.UUID                  `json:"id"`
	OrgID               uuid.UUID                  `json:"org_id"`
	Name                string                     `json:"name"`
	Type                DockerRegistryType         `json:"type"`
	URL                 string                     `json:"url"`
	CredentialsEncrypted []byte                    `json:"-"` // Encrypted, never expose in JSON
	IsDefault           bool                       `json:"is_default"`
	Enabled             bool                       `json:"enabled"`
	HealthStatus        DockerRegistryHealthStatus `json:"health_status"`
	LastHealthCheck     *time.Time                 `json:"last_health_check,omitempty"`
	LastHealthError     *string                    `json:"last_health_error,omitempty"`
	CredentialsRotatedAt *time.Time                `json:"credentials_rotated_at,omitempty"`
	CredentialsExpiresAt *time.Time                `json:"credentials_expires_at,omitempty"`
	Metadata            map[string]interface{}     `json:"metadata,omitempty"`
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
	CreatedBy           *uuid.UUID                 `json:"created_by,omitempty"`
}

// DockerRegistryCredentials holds the credentials for a Docker registry.
// This is stored encrypted in the database.
type DockerRegistryCredentials struct {
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	// For AWS ECR
	AWSAccessKeyID     string `json:"aws_access_key_id,omitempty"`
	AWSSecretAccessKey string `json:"aws_secret_access_key,omitempty"`
	AWSRegion          string `json:"aws_region,omitempty"`
	// For Azure ACR
	AzureTenantID     string `json:"azure_tenant_id,omitempty"`
	AzureClientID     string `json:"azure_client_id,omitempty"`
	AzureClientSecret string `json:"azure_client_secret,omitempty"`
	// For GCR
	GCRKeyJSON string `json:"gcr_key_json,omitempty"`
}

// NewDockerRegistry creates a new DockerRegistry with the given details.
func NewDockerRegistry(orgID uuid.UUID, name string, registryType DockerRegistryType, url string, credentialsEncrypted []byte) *DockerRegistry {
	now := time.Now()
	return &DockerRegistry{
		ID:                   uuid.New(),
		OrgID:                orgID,
		Name:                 name,
		Type:                 registryType,
		URL:                  url,
		CredentialsEncrypted: credentialsEncrypted,
		IsDefault:            false,
		Enabled:              true,
		HealthStatus:         DockerRegistryHealthUnknown,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// ValidDockerRegistryTypes returns all valid Docker registry types.
func ValidDockerRegistryTypes() []DockerRegistryType {
	return []DockerRegistryType{
		DockerRegistryTypeDockerHub,
		DockerRegistryTypeGCR,
		DockerRegistryTypeECR,
		DockerRegistryTypeACR,
		DockerRegistryTypeGHCR,
		DockerRegistryTypePrivate,
	}
}

// IsValidType checks if the Docker registry type is valid.
func (r *DockerRegistry) IsValidType() bool {
	for _, t := range ValidDockerRegistryTypes() {
		if r.Type == t {
			return true
		}
	}
	return false
}

// SetMetadata sets the metadata from JSON bytes.
func (r *DockerRegistry) SetMetadata(data []byte) error {
	if len(data) == 0 {
		r.Metadata = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(data, &r.Metadata)
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (r *DockerRegistry) MetadataJSON() ([]byte, error) {
	if r.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(r.Metadata)
}

// GetDefaultURL returns the default URL for a given registry type.
func GetDefaultURL(registryType DockerRegistryType) string {
	switch registryType {
	case DockerRegistryTypeDockerHub:
		return "https://index.docker.io/v1/"
	case DockerRegistryTypeGCR:
		return "https://gcr.io"
	case DockerRegistryTypeGHCR:
		return "https://ghcr.io"
	default:
		return ""
	}
}

// DockerLoginResult contains the result of a Docker login operation.
type DockerLoginResult struct {
	Success      bool      `json:"success"`
	RegistryID   uuid.UUID `json:"registry_id"`
	RegistryURL  string    `json:"registry_url"`
	ErrorMessage string    `json:"error_message,omitempty"`
	LoggedInAt   time.Time `json:"logged_in_at"`
}

// DockerRegistryHealthCheck contains the result of a registry health check.
type DockerRegistryHealthCheck struct {
	RegistryID   uuid.UUID                  `json:"registry_id"`
	Status       DockerRegistryHealthStatus `json:"status"`
	ResponseTime int64                      `json:"response_time_ms"`
	ErrorMessage string                     `json:"error_message,omitempty"`
	CheckedAt    time.Time                  `json:"checked_at"`
}
