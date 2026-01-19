package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/MacJediWizard/keldris/internal/models"
)

// Backend defines the interface for backup storage backends.
type Backend interface {
	// Type returns the repository type.
	Type() models.RepositoryType

	// ToResticConfig converts the backend configuration to a ResticConfig.
	ToResticConfig(password string) ResticConfig

	// Validate checks if the configuration is valid.
	Validate() error
}

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

// S3Backend represents an S3-compatible storage backend.
type S3Backend struct {
	Endpoint        string `json:"endpoint,omitempty"`
	Bucket          string `json:"bucket"`
	Prefix          string `json:"prefix,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	UseSSL          bool   `json:"use_ssl"`
}

// Type returns the repository type.
func (b *S3Backend) Type() models.RepositoryType {
	return models.RepositoryTypeS3
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *S3Backend) ToResticConfig(password string) ResticConfig {
	// Build the repository URL
	var repository string
	if b.Endpoint != "" {
		// Custom endpoint (MinIO, Wasabi, etc.)
		scheme := "http"
		if b.UseSSL {
			scheme = "https"
		}
		endpoint := b.Endpoint
		// Parse and rebuild to ensure proper formatting
		if u, err := url.Parse(b.Endpoint); err == nil && u.Host != "" {
			endpoint = u.Host
		}
		repository = fmt.Sprintf("s3:%s://%s/%s", scheme, endpoint, b.Bucket)
	} else {
		// AWS S3
		repository = fmt.Sprintf("s3:s3.amazonaws.com/%s", b.Bucket)
	}

	if b.Prefix != "" {
		repository = repository + "/" + b.Prefix
	}

	env := map[string]string{
		"AWS_ACCESS_KEY_ID":     b.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": b.SecretAccessKey,
	}

	if b.Region != "" {
		env["AWS_DEFAULT_REGION"] = b.Region
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env:        env,
	}
}

// Validate checks if the configuration is valid.
func (b *S3Backend) Validate() error {
	if b.Bucket == "" {
		return errors.New("s3 backend: bucket is required")
	}
	if b.AccessKeyID == "" {
		return errors.New("s3 backend: access_key_id is required")
	}
	if b.SecretAccessKey == "" {
		return errors.New("s3 backend: secret_access_key is required")
	}
	return nil
}

// B2Backend represents a Backblaze B2 storage backend.
type B2Backend struct {
	Bucket         string `json:"bucket"`
	Prefix         string `json:"prefix,omitempty"`
	AccountID      string `json:"account_id"`
	ApplicationKey string `json:"application_key"`
}

// Type returns the repository type.
func (b *B2Backend) Type() models.RepositoryType {
	return models.RepositoryTypeB2
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *B2Backend) ToResticConfig(password string) ResticConfig {
	repository := fmt.Sprintf("b2:%s", b.Bucket)
	if b.Prefix != "" {
		repository = repository + ":" + b.Prefix
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env: map[string]string{
			"B2_ACCOUNT_ID":  b.AccountID,
			"B2_ACCOUNT_KEY": b.ApplicationKey,
		},
	}
}

// Validate checks if the configuration is valid.
func (b *B2Backend) Validate() error {
	if b.Bucket == "" {
		return errors.New("b2 backend: bucket is required")
	}
	if b.AccountID == "" {
		return errors.New("b2 backend: account_id is required")
	}
	if b.ApplicationKey == "" {
		return errors.New("b2 backend: application_key is required")
	}
	return nil
}

// SFTPBackend represents an SFTP storage backend.
type SFTPBackend struct {
	Host       string `json:"host"`
	Port       int    `json:"port,omitempty"`
	User       string `json:"user"`
	Path       string `json:"path"`
	PrivateKey string `json:"private_key,omitempty"`
}

// Type returns the repository type.
func (b *SFTPBackend) Type() models.RepositoryType {
	return models.RepositoryTypeSFTP
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *SFTPBackend) ToResticConfig(password string) ResticConfig {
	port := b.Port
	if port == 0 {
		port = 22
	}

	repository := fmt.Sprintf("sftp:%s@%s:%d%s", b.User, b.Host, port, b.Path)

	env := make(map[string]string)
	if b.PrivateKey != "" {
		// Restic uses SSH_AUTH_SOCK or standard SSH key locations
		// Private key can be passed via environment or SSH agent
		env["RESTIC_SFTP_ARGS"] = fmt.Sprintf("-i %s", b.PrivateKey)
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
		Env:        env,
	}
}

// Validate checks if the configuration is valid.
func (b *SFTPBackend) Validate() error {
	if b.Host == "" {
		return errors.New("sftp backend: host is required")
	}
	if b.User == "" {
		return errors.New("sftp backend: user is required")
	}
	if b.Path == "" {
		return errors.New("sftp backend: path is required")
	}
	if !filepath.IsAbs(b.Path) {
		return errors.New("sftp backend: path must be absolute")
	}
	return nil
}

// RestBackend represents a Restic REST server backend.
type RestBackend struct {
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// Type returns the repository type.
func (b *RestBackend) Type() models.RepositoryType {
	return models.RepositoryTypeRest
}

// ToResticConfig converts the backend to a ResticConfig.
func (b *RestBackend) ToResticConfig(password string) ResticConfig {
	repository := b.URL

	// Handle authentication in URL
	if b.Username != "" && b.Password != "" {
		if u, err := url.Parse(b.URL); err == nil {
			u.User = url.UserPassword(b.Username, b.Password)
			repository = u.String()
		}
	}

	// Ensure rest: prefix
	if !isRestURL(repository) {
		repository = "rest:" + repository
	}

	return ResticConfig{
		Repository: repository,
		Password:   password,
	}
}

// Validate checks if the configuration is valid.
func (b *RestBackend) Validate() error {
	if b.URL == "" {
		return errors.New("rest backend: url is required")
	}
	parsedURL := b.URL
	if isRestURL(parsedURL) {
		parsedURL = parsedURL[5:] // Remove "rest:" prefix
	}
	if _, err := url.Parse(parsedURL); err != nil {
		return fmt.Errorf("rest backend: invalid url: %w", err)
	}
	return nil
}

// isRestURL checks if the URL has the rest: prefix.
func isRestURL(s string) bool {
	return len(s) > 5 && s[:5] == "rest:"
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

	default:
		return nil, fmt.Errorf("unsupported repository type: %s", repoType)
	}
}

// BackendConfig converts a Backend to its JSON representation.
func BackendConfig(backend Backend) ([]byte, error) {
	return json.Marshal(backend)
}
