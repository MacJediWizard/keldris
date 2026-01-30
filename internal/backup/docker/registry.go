// Package docker provides Docker registry management functionality.
package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	// ErrRegistryNotFound indicates the registry was not found.
	ErrRegistryNotFound = errors.New("docker registry not found")
	// ErrRegistryDisabled indicates the registry is disabled.
	ErrRegistryDisabled = errors.New("docker registry is disabled")
	// ErrInvalidCredentials indicates the credentials are invalid.
	ErrInvalidCredentials = errors.New("invalid registry credentials")
	// ErrHealthCheckFailed indicates the health check failed.
	ErrHealthCheckFailed = errors.New("registry health check failed")
	// ErrLoginFailed indicates the login operation failed.
	ErrLoginFailed = errors.New("docker login failed")
)

// RegistryStore defines the interface for registry persistence operations.
type RegistryStore interface {
	GetDockerRegistriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRegistry, error)
	GetDockerRegistryByID(ctx context.Context, id uuid.UUID) (*models.DockerRegistry, error)
	GetDefaultDockerRegistry(ctx context.Context, orgID uuid.UUID) (*models.DockerRegistry, error)
	CreateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error
	UpdateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error
	DeleteDockerRegistry(ctx context.Context, id uuid.UUID) error
	UpdateDockerRegistryHealth(ctx context.Context, id uuid.UUID, status models.DockerRegistryHealthStatus, errorMsg *string) error
	UpdateDockerRegistryCredentials(ctx context.Context, id uuid.UUID, credentialsEncrypted []byte, expiresAt *time.Time) error
}

// RegistryManager manages Docker registry operations.
type RegistryManager struct {
	store      RegistryStore
	keyManager *crypto.KeyManager
	logger     zerolog.Logger
	httpClient *http.Client

	// Cache for logged-in registries (registry ID -> login time)
	loginCache   map[uuid.UUID]time.Time
	loginCacheMu sync.RWMutex
}

// NewRegistryManager creates a new RegistryManager.
func NewRegistryManager(store RegistryStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *RegistryManager {
	return &RegistryManager{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "docker_registry_manager").Logger(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		loginCache: make(map[uuid.UUID]time.Time),
	}
}

// CreateRegistry creates a new Docker registry with encrypted credentials.
func (rm *RegistryManager) CreateRegistry(ctx context.Context, orgID uuid.UUID, name string, registryType models.DockerRegistryType, url string, credentials *models.DockerRegistryCredentials, createdBy *uuid.UUID) (*models.DockerRegistry, error) {
	// Validate registry type
	registry := &models.DockerRegistry{Type: registryType}
	if !registry.IsValidType() {
		return nil, fmt.Errorf("invalid registry type: %s", registryType)
	}

	// Set default URL if not provided
	if url == "" {
		url = models.GetDefaultURL(registryType)
		if url == "" {
			return nil, errors.New("url is required for this registry type")
		}
	}

	// Encrypt credentials
	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return nil, fmt.Errorf("marshal credentials: %w", err)
	}

	encryptedCreds, err := rm.keyManager.Encrypt(credentialsJSON)
	if err != nil {
		return nil, fmt.Errorf("encrypt credentials: %w", err)
	}

	// Create registry
	newRegistry := models.NewDockerRegistry(orgID, name, registryType, url, encryptedCreds)
	newRegistry.CreatedBy = createdBy
	newRegistry.CredentialsRotatedAt = &newRegistry.CreatedAt

	if err := rm.store.CreateDockerRegistry(ctx, newRegistry); err != nil {
		return nil, fmt.Errorf("create registry: %w", err)
	}

	rm.logger.Info().
		Str("registry_id", newRegistry.ID.String()).
		Str("name", name).
		Str("type", string(registryType)).
		Msg("docker registry created")

	return newRegistry, nil
}

// GetRegistry retrieves a Docker registry by ID and decrypts its credentials.
func (rm *RegistryManager) GetRegistry(ctx context.Context, id uuid.UUID) (*models.DockerRegistry, *models.DockerRegistryCredentials, error) {
	registry, err := rm.store.GetDockerRegistryByID(ctx, id)
	if err != nil {
		return nil, nil, fmt.Errorf("get registry: %w", err)
	}

	credentials, err := rm.decryptCredentials(registry.CredentialsEncrypted)
	if err != nil {
		return nil, nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	return registry, credentials, nil
}

// GetRegistries retrieves all Docker registries for an organization.
func (rm *RegistryManager) GetRegistries(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRegistry, error) {
	return rm.store.GetDockerRegistriesByOrgID(ctx, orgID)
}

// UpdateCredentials updates the credentials for a registry (rotation support).
func (rm *RegistryManager) UpdateCredentials(ctx context.Context, id uuid.UUID, credentials *models.DockerRegistryCredentials, expiresAt *time.Time) error {
	// Encrypt new credentials
	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	encryptedCreds, err := rm.keyManager.Encrypt(credentialsJSON)
	if err != nil {
		return fmt.Errorf("encrypt credentials: %w", err)
	}

	if err := rm.store.UpdateDockerRegistryCredentials(ctx, id, encryptedCreds, expiresAt); err != nil {
		return fmt.Errorf("update credentials: %w", err)
	}

	// Invalidate login cache for this registry
	rm.loginCacheMu.Lock()
	delete(rm.loginCache, id)
	rm.loginCacheMu.Unlock()

	rm.logger.Info().
		Str("registry_id", id.String()).
		Msg("docker registry credentials rotated")

	return nil
}

// DeleteRegistry deletes a Docker registry.
func (rm *RegistryManager) DeleteRegistry(ctx context.Context, id uuid.UUID) error {
	// Invalidate login cache
	rm.loginCacheMu.Lock()
	delete(rm.loginCache, id)
	rm.loginCacheMu.Unlock()

	return rm.store.DeleteDockerRegistry(ctx, id)
}

// Login performs Docker login to a registry.
func (rm *RegistryManager) Login(ctx context.Context, id uuid.UUID) (*models.DockerLoginResult, error) {
	registry, credentials, err := rm.GetRegistry(ctx, id)
	if err != nil {
		return nil, err
	}

	if !registry.Enabled {
		return nil, ErrRegistryDisabled
	}

	result := &models.DockerLoginResult{
		RegistryID:  registry.ID,
		RegistryURL: registry.URL,
		LoggedInAt:  time.Now(),
	}

	// Build docker login command based on registry type
	if err := rm.performLogin(ctx, registry, credentials); err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	result.Success = true

	// Update login cache
	rm.loginCacheMu.Lock()
	rm.loginCache[id] = time.Now()
	rm.loginCacheMu.Unlock()

	rm.logger.Info().
		Str("registry_id", id.String()).
		Str("registry_url", registry.URL).
		Msg("docker login successful")

	return result, nil
}

// LoginAll logs into all enabled registries for an organization.
func (rm *RegistryManager) LoginAll(ctx context.Context, orgID uuid.UUID) ([]*models.DockerLoginResult, error) {
	registries, err := rm.store.GetDockerRegistriesByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get registries: %w", err)
	}

	var results []*models.DockerLoginResult
	for _, registry := range registries {
		if !registry.Enabled {
			continue
		}

		result, err := rm.Login(ctx, registry.ID)
		if err != nil {
			rm.logger.Warn().
				Err(err).
				Str("registry_id", registry.ID.String()).
				Str("registry_name", registry.Name).
				Msg("failed to login to registry")
		}
		results = append(results, result)
	}

	return results, nil
}

// EnsureLoggedIn ensures the registry is logged in, performing login if necessary.
// This is used for auto-login before image pull/push operations.
func (rm *RegistryManager) EnsureLoggedIn(ctx context.Context, id uuid.UUID) error {
	// Check if already logged in recently (within last hour)
	rm.loginCacheMu.RLock()
	loginTime, exists := rm.loginCache[id]
	rm.loginCacheMu.RUnlock()

	if exists && time.Since(loginTime) < time.Hour {
		return nil
	}

	_, err := rm.Login(ctx, id)
	return err
}

// EnsureAllLoggedIn ensures all registries for an organization are logged in.
func (rm *RegistryManager) EnsureAllLoggedIn(ctx context.Context, orgID uuid.UUID) error {
	registries, err := rm.store.GetDockerRegistriesByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get registries: %w", err)
	}

	for _, registry := range registries {
		if !registry.Enabled {
			continue
		}

		if err := rm.EnsureLoggedIn(ctx, registry.ID); err != nil {
			rm.logger.Warn().
				Err(err).
				Str("registry_id", registry.ID.String()).
				Msg("failed to ensure login for registry")
			// Continue with other registries
		}
	}

	return nil
}

// HealthCheck performs a health check on a registry.
func (rm *RegistryManager) HealthCheck(ctx context.Context, id uuid.UUID) (*models.DockerRegistryHealthCheck, error) {
	registry, err := rm.store.GetDockerRegistryByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get registry: %w", err)
	}

	result := &models.DockerRegistryHealthCheck{
		RegistryID: registry.ID,
		CheckedAt:  time.Now(),
	}

	start := time.Now()
	healthy, errorMsg := rm.checkRegistryHealth(ctx, registry)
	result.ResponseTime = time.Since(start).Milliseconds()

	if healthy {
		result.Status = models.DockerRegistryHealthHealthy
	} else {
		result.Status = models.DockerRegistryHealthUnhealthy
		result.ErrorMessage = errorMsg
	}

	// Update health status in database
	var errPtr *string
	if errorMsg != "" {
		errPtr = &errorMsg
	}
	if err := rm.store.UpdateDockerRegistryHealth(ctx, id, result.Status, errPtr); err != nil {
		rm.logger.Warn().Err(err).Str("registry_id", id.String()).Msg("failed to update health status")
	}

	return result, nil
}

// HealthCheckAll performs health checks on all registries for an organization.
func (rm *RegistryManager) HealthCheckAll(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRegistryHealthCheck, error) {
	registries, err := rm.store.GetDockerRegistriesByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get registries: %w", err)
	}

	var results []*models.DockerRegistryHealthCheck
	for _, registry := range registries {
		result, err := rm.HealthCheck(ctx, registry.ID)
		if err != nil {
			rm.logger.Warn().
				Err(err).
				Str("registry_id", registry.ID.String()).
				Msg("failed to health check registry")
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// GetExpiredCredentials returns registries with expired or expiring credentials.
func (rm *RegistryManager) GetExpiredCredentials(ctx context.Context, orgID uuid.UUID, warningDays int) ([]*models.DockerRegistry, error) {
	registries, err := rm.store.GetDockerRegistriesByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get registries: %w", err)
	}

	warningThreshold := time.Now().Add(time.Duration(warningDays) * 24 * time.Hour)
	var expiring []*models.DockerRegistry

	for _, registry := range registries {
		if registry.CredentialsExpiresAt != nil && registry.CredentialsExpiresAt.Before(warningThreshold) {
			expiring = append(expiring, registry)
		}
	}

	return expiring, nil
}

// decryptCredentials decrypts the registry credentials.
func (rm *RegistryManager) decryptCredentials(encrypted []byte) (*models.DockerRegistryCredentials, error) {
	if len(encrypted) == 0 {
		return nil, ErrInvalidCredentials
	}

	decrypted, err := rm.keyManager.Decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	var credentials models.DockerRegistryCredentials
	if err := json.Unmarshal(decrypted, &credentials); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}

	return &credentials, nil
}

// performLogin executes the Docker login command.
func (rm *RegistryManager) performLogin(ctx context.Context, registry *models.DockerRegistry, credentials *models.DockerRegistryCredentials) error {
	switch registry.Type {
	case models.DockerRegistryTypeECR:
		return rm.loginECR(ctx, registry, credentials)
	case models.DockerRegistryTypeGCR:
		return rm.loginGCR(ctx, registry, credentials)
	case models.DockerRegistryTypeACR:
		return rm.loginACR(ctx, registry, credentials)
	default:
		return rm.loginStandard(ctx, registry, credentials)
	}
}

// loginStandard performs a standard Docker login (DockerHub, GHCR, private).
func (rm *RegistryManager) loginStandard(ctx context.Context, registry *models.DockerRegistry, credentials *models.DockerRegistryCredentials) error {
	password := credentials.Password
	if password == "" {
		password = credentials.AccessToken
	}

	if credentials.Username == "" || password == "" {
		return ErrInvalidCredentials
	}

	// Use docker login command with password from stdin for security
	cmd := exec.CommandContext(ctx, "docker", "login", "--username", credentials.Username, "--password-stdin", registry.URL)
	cmd.Stdin = strings.NewReader(password)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrLoginFailed, string(output))
	}

	return nil
}

// loginECR performs AWS ECR login.
func (rm *RegistryManager) loginECR(ctx context.Context, registry *models.DockerRegistry, credentials *models.DockerRegistryCredentials) error {
	if credentials.AWSAccessKeyID == "" || credentials.AWSSecretAccessKey == "" {
		return ErrInvalidCredentials
	}

	// Set AWS credentials in environment
	env := os.Environ()
	env = append(env, "AWS_ACCESS_KEY_ID="+credentials.AWSAccessKeyID)
	env = append(env, "AWS_SECRET_ACCESS_KEY="+credentials.AWSSecretAccessKey)
	if credentials.AWSRegion != "" {
		env = append(env, "AWS_DEFAULT_REGION="+credentials.AWSRegion)
	}

	// Get ECR login password
	cmd := exec.CommandContext(ctx, "aws", "ecr", "get-login-password")
	cmd.Env = env
	password, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("get ECR password: %w", err)
	}

	// Login with the ECR password
	loginCmd := exec.CommandContext(ctx, "docker", "login", "--username", "AWS", "--password-stdin", registry.URL)
	loginCmd.Stdin = strings.NewReader(string(password))
	output, err := loginCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", ErrLoginFailed, string(output))
	}

	return nil
}

// loginGCR performs Google Container Registry login.
func (rm *RegistryManager) loginGCR(ctx context.Context, registry *models.DockerRegistry, credentials *models.DockerRegistryCredentials) error {
	if credentials.GCRKeyJSON == "" {
		return ErrInvalidCredentials
	}

	// Create a temporary file for the service account key
	tmpFile, err := os.CreateTemp("", "gcr-key-*.json")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(credentials.GCRKeyJSON); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write key file: %w", err)
	}
	tmpFile.Close()

	// Activate service account
	cmd := exec.CommandContext(ctx, "gcloud", "auth", "activate-service-account", "--key-file", tmpFile.Name())
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("activate service account: %s", string(output))
	}

	// Configure Docker with gcloud
	configCmd := exec.CommandContext(ctx, "gcloud", "auth", "configure-docker", "--quiet")
	if output, err := configCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("configure docker: %s", string(output))
	}

	return nil
}

// loginACR performs Azure Container Registry login.
func (rm *RegistryManager) loginACR(ctx context.Context, registry *models.DockerRegistry, credentials *models.DockerRegistryCredentials) error {
	if credentials.AzureClientID == "" || credentials.AzureClientSecret == "" || credentials.AzureTenantID == "" {
		return ErrInvalidCredentials
	}

	// Login to Azure
	cmd := exec.CommandContext(ctx, "az", "login", "--service-principal",
		"-u", credentials.AzureClientID,
		"-p", credentials.AzureClientSecret,
		"--tenant", credentials.AzureTenantID)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("az login: %s", string(output))
	}

	// Get ACR login
	// Extract registry name from URL (e.g., myregistry.azurecr.io -> myregistry)
	registryName := strings.Split(registry.URL, ".")[0]
	registryName = strings.TrimPrefix(registryName, "https://")
	registryName = strings.TrimPrefix(registryName, "http://")

	acrCmd := exec.CommandContext(ctx, "az", "acr", "login", "--name", registryName)
	if output, err := acrCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("acr login: %s", string(output))
	}

	return nil
}

// checkRegistryHealth checks if a registry is accessible.
func (rm *RegistryManager) checkRegistryHealth(ctx context.Context, registry *models.DockerRegistry) (bool, string) {
	// For most registries, we can check the v2 API endpoint
	url := registry.URL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}
	url += "v2/"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Sprintf("create request: %v", err)
	}

	resp, err := rm.httpClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("request failed: %v", err)
	}
	defer resp.Body.Close()

	// 401 is acceptable - it means the registry is up but requires auth
	// 200 means public access is allowed
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		return true, ""
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return false, fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, string(body))
}

// GetDockerConfigAuth returns the Docker config.json auth entry for a registry.
// This can be used to generate Docker configs for CI/CD systems.
func (rm *RegistryManager) GetDockerConfigAuth(ctx context.Context, id uuid.UUID) (map[string]interface{}, error) {
	registry, credentials, err := rm.GetRegistry(ctx, id)
	if err != nil {
		return nil, err
	}

	password := credentials.Password
	if password == "" {
		password = credentials.AccessToken
	}

	auth := base64.StdEncoding.EncodeToString([]byte(credentials.Username + ":" + password))

	return map[string]interface{}{
		"auths": map[string]interface{}{
			registry.URL: map[string]interface{}{
				"auth": auth,
			},
		},
	}, nil
}

// WriteDockerConfig writes a Docker config.json file with the registry credentials.
// This is useful for container operations that need authentication.
func (rm *RegistryManager) WriteDockerConfig(ctx context.Context, orgID uuid.UUID, configPath string) error {
	registries, err := rm.store.GetDockerRegistriesByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("get registries: %w", err)
	}

	auths := make(map[string]interface{})

	for _, registry := range registries {
		if !registry.Enabled {
			continue
		}

		credentials, err := rm.decryptCredentials(registry.CredentialsEncrypted)
		if err != nil {
			rm.logger.Warn().
				Err(err).
				Str("registry_id", registry.ID.String()).
				Msg("failed to decrypt credentials")
			continue
		}

		password := credentials.Password
		if password == "" {
			password = credentials.AccessToken
		}

		if credentials.Username != "" && password != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(credentials.Username + ":" + password))
			auths[registry.URL] = map[string]interface{}{
				"auth": auth,
			}
		}
	}

	config := map[string]interface{}{
		"auths": auths,
	}

	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, configJSON, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
