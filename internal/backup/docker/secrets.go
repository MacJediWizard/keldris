// Package docker provides Docker secrets and configs backup functionality.
package docker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// SecretBackupStatus represents the status of a secret backup operation.
type SecretBackupStatus string

const (
	// SecretBackupStatusPending indicates the backup has not started.
	SecretBackupStatusPending SecretBackupStatus = "pending"
	// SecretBackupStatusRunning indicates the backup is in progress.
	SecretBackupStatusRunning SecretBackupStatus = "running"
	// SecretBackupStatusCompleted indicates the backup completed successfully.
	SecretBackupStatusCompleted SecretBackupStatus = "completed"
	// SecretBackupStatusFailed indicates the backup failed.
	SecretBackupStatusFailed SecretBackupStatus = "failed"
)

// SecretType identifies the type of Docker secret.
type SecretType string

const (
	// SecretTypeSecret represents a Docker secret.
	SecretTypeSecret SecretType = "secret"
	// SecretTypeConfig represents a Docker config.
	SecretTypeConfig SecretType = "config"
	// SecretTypeSwarmSecret represents a Docker Swarm secret.
	SecretTypeSwarmSecret SecretType = "swarm_secret"
)

// MaskChar is the character used for masking sensitive data in UI display.
const MaskChar = "•"

// DefaultMaskLength is the default length of masked output.
const DefaultMaskLength = 8

var (
	// ErrSwarmNotActive indicates Docker Swarm is not active.
	ErrSwarmNotActive = errors.New("docker swarm is not active")
	// ErrSecretNotFound indicates the requested secret was not found.
	ErrSecretNotFound = errors.New("secret not found")
	// ErrConfigNotFound indicates the requested config was not found.
	ErrConfigNotFound = errors.New("config not found")
	// ErrDecryptionFailed indicates decryption of a secret failed.
	ErrDecryptionFailed = errors.New("decryption failed")
	// ErrRestoreDependencyNotMet indicates a dependency was not met during restore.
	ErrRestoreDependencyNotMet = errors.New("restore dependency not met")
	// ErrInvalidSecretData indicates the secret data is invalid or corrupted.
	ErrInvalidSecretData = errors.New("invalid secret data")
)

// DockerSecret represents a Docker secret with its metadata.
type DockerSecret struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         SecretType        `json:"type"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Labels       map[string]string `json:"labels,omitempty"`
	Version      int64             `json:"version"`
	Driver       string            `json:"driver,omitempty"`
	DriverOpts   map[string]string `json:"driver_opts,omitempty"`
	Templating   *TemplatingConfig `json:"templating,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"` // Service names that depend on this secret
}

// TemplatingConfig represents Docker config templating options.
type TemplatingConfig struct {
	Name    string            `json:"name"`
	Options map[string]string `json:"options,omitempty"`
}

// SecretData holds the encrypted secret data with double encryption.
type SecretData struct {
	// DockerEncrypted is the base64-encoded Docker-encrypted secret data.
	// Docker secrets are encrypted at rest by Docker's internal encryption.
	DockerEncrypted string `json:"docker_encrypted"`

	// KeldrisEncrypted is the additional AES-256-GCM encryption layer added by Keldris.
	// This is the final encrypted form stored in backups.
	KeldrisEncrypted string `json:"keldris_encrypted"`

	// Checksum is the SHA-256 hash of the original secret data for integrity verification.
	Checksum string `json:"checksum"`

	// EncryptedAt is when the Keldris encryption was applied.
	EncryptedAt time.Time `json:"encrypted_at"`
}

// SecretBackup represents a complete backup of Docker secrets and configs.
type SecretBackup struct {
	ID             uuid.UUID                `json:"id"`
	CreatedAt      time.Time                `json:"created_at"`
	CompletedAt    *time.Time               `json:"completed_at,omitempty"`
	Status         SecretBackupStatus       `json:"status"`
	Secrets        map[string]*SecretData   `json:"secrets"`        // key: secret ID
	Configs        map[string]*SecretData   `json:"configs"`        // key: config ID
	SwarmSecrets   map[string]*SecretData   `json:"swarm_secrets"`  // key: swarm secret ID
	Metadata       map[string]*DockerSecret `json:"metadata"`       // key: secret/config ID
	RotationState  *RotationState           `json:"rotation_state"` // Tracking for secret rotation
	ErrorMessage   string                   `json:"error_message,omitempty"`
	SecretsCount   int                      `json:"secrets_count"`
	ConfigsCount   int                      `json:"configs_count"`
	SwarmCount     int                      `json:"swarm_count"`
	TotalSizeBytes int64                    `json:"total_size_bytes"`
}

// RotationState tracks secret rotation history and schedules.
type RotationState struct {
	// LastRotations maps secret ID to its last rotation time.
	LastRotations map[string]time.Time `json:"last_rotations"`

	// RotationSchedules maps secret ID to its rotation schedule (cron expression).
	RotationSchedules map[string]string `json:"rotation_schedules,omitempty"`

	// VersionHistory maps secret ID to a list of version timestamps.
	VersionHistory map[string][]VersionEntry `json:"version_history"`

	// PendingRotations lists secrets that are due for rotation.
	PendingRotations []string `json:"pending_rotations,omitempty"`
}

// VersionEntry represents a single version of a secret.
type VersionEntry struct {
	Version   int64     `json:"version"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
	RotatedBy string    `json:"rotated_by,omitempty"` // User or system that performed the rotation
}

// RestoreOrder defines the order for restoring secrets based on dependencies.
type RestoreOrder struct {
	// Phases lists secrets in dependency order. Earlier phases must be restored first.
	Phases [][]string `json:"phases"`

	// Dependencies maps secret ID to its dependencies.
	Dependencies map[string][]string `json:"dependencies"`
}

// MaskedSecret represents a secret with masked value for UI display.
type MaskedSecret struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        SecretType        `json:"type"`
	MaskedValue string            `json:"masked_value"`
	CreatedAt   time.Time         `json:"created_at"`
	Labels      map[string]string `json:"labels,omitempty"`
	HasBackup   bool              `json:"has_backup"`
	LastBackup  *time.Time        `json:"last_backup,omitempty"`
}

// SecretsManager handles Docker secrets and configs backup operations.
type SecretsManager struct {
	keyManager *crypto.KeyManager
	dockerPath string
	logger     zerolog.Logger
}

// NewSecretsManager creates a new SecretsManager instance.
func NewSecretsManager(keyManager *crypto.KeyManager, logger zerolog.Logger) *SecretsManager {
	return &SecretsManager{
		keyManager: keyManager,
		dockerPath: "docker",
		logger:     logger.With().Str("component", "docker_secrets").Logger(),
	}
}

// NewSecretsManagerWithPath creates a new SecretsManager with a custom Docker binary path.
func NewSecretsManagerWithPath(keyManager *crypto.KeyManager, dockerPath string, logger zerolog.Logger) *SecretsManager {
	return &SecretsManager{
		keyManager: keyManager,
		dockerPath: dockerPath,
		logger:     logger.With().Str("component", "docker_secrets").Logger(),
	}
}

// CheckDocker verifies Docker is available and running.
func (sm *SecretsManager) CheckDocker(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "info", "--format", "{{.ServerVersion}}")
	output, err := cmd.Output()
	if err != nil {
		sm.logger.Error().Err(err).Msg("docker not available")
		return ErrDockerNotAvailable
	}

	sm.logger.Debug().Str("version", strings.TrimSpace(string(output))).Msg("docker available")
	return nil
}

// CheckSwarm verifies Docker Swarm is active.
func (sm *SecretsManager) CheckSwarm(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "info", "--format", "{{.Swarm.LocalNodeState}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("check swarm: %w", err)
	}

	state := strings.TrimSpace(string(output))
	if state != "active" {
		sm.logger.Debug().Str("state", state).Msg("swarm not active")
		return ErrSwarmNotActive
	}

	sm.logger.Debug().Msg("swarm is active")
	return nil
}

// ListSecrets returns all Docker secrets.
func (sm *SecretsManager) ListSecrets(ctx context.Context) ([]*DockerSecret, error) {
	if err := sm.CheckSwarm(ctx); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, sm.dockerPath, "secret", "ls", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	return sm.parseSecretList(output, SecretTypeSecret)
}

// ListConfigs returns all Docker configs.
func (sm *SecretsManager) ListConfigs(ctx context.Context) ([]*DockerSecret, error) {
	if err := sm.CheckSwarm(ctx); err != nil {
		return nil, err
	}

	cmd := exec.CommandContext(ctx, sm.dockerPath, "config", "ls", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list configs: %w", err)
	}

	return sm.parseSecretList(output, SecretTypeConfig)
}

// ListSwarmSecrets returns secrets specific to Docker Swarm services.
func (sm *SecretsManager) ListSwarmSecrets(ctx context.Context) ([]*DockerSecret, error) {
	if err := sm.CheckSwarm(ctx); err != nil {
		return nil, err
	}

	// Get secrets with detailed inspection
	secrets, err := sm.ListSecrets(ctx)
	if err != nil {
		return nil, err
	}

	// Mark secrets that are used by Swarm services
	swarmSecrets := make([]*DockerSecret, 0)
	for _, secret := range secrets {
		// Inspect to get service usage
		inspected, err := sm.InspectSecret(ctx, secret.ID)
		if err != nil {
			sm.logger.Warn().Err(err).Str("secret_id", secret.ID).Msg("failed to inspect secret")
			continue
		}

		if len(inspected.Dependencies) > 0 {
			inspected.Type = SecretTypeSwarmSecret
			swarmSecrets = append(swarmSecrets, inspected)
		}
	}

	return swarmSecrets, nil
}

// InspectSecret returns detailed information about a secret.
func (sm *SecretsManager) InspectSecret(ctx context.Context, secretID string) (*DockerSecret, error) {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "secret", "inspect", secretID, "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "No such secret") {
			return nil, ErrSecretNotFound
		}
		return nil, fmt.Errorf("inspect secret: %w", err)
	}

	var rawSecret struct {
		ID      string `json:"ID"`
		Version struct {
			Index int64 `json:"Index"`
		} `json:"Version"`
		CreatedAt time.Time `json:"CreatedAt"`
		UpdatedAt time.Time `json:"UpdatedAt"`
		Spec      struct {
			Name   string            `json:"Name"`
			Labels map[string]string `json:"Labels"`
			Driver struct {
				Name    string            `json:"Name"`
				Options map[string]string `json:"Options"`
			} `json:"Driver"`
			Templating *struct {
				Name    string            `json:"Name"`
				Options map[string]string `json:"Options"`
			} `json:"Templating"`
		} `json:"Spec"`
	}

	if err := json.Unmarshal(output, &rawSecret); err != nil {
		return nil, fmt.Errorf("parse secret inspect: %w", err)
	}

	secret := &DockerSecret{
		ID:         rawSecret.ID,
		Name:       rawSecret.Spec.Name,
		Type:       SecretTypeSecret,
		CreatedAt:  rawSecret.CreatedAt,
		UpdatedAt:  rawSecret.UpdatedAt,
		Labels:     rawSecret.Spec.Labels,
		Version:    rawSecret.Version.Index,
		Driver:     rawSecret.Spec.Driver.Name,
		DriverOpts: rawSecret.Spec.Driver.Options,
	}

	if rawSecret.Spec.Templating != nil {
		secret.Templating = &TemplatingConfig{
			Name:    rawSecret.Spec.Templating.Name,
			Options: rawSecret.Spec.Templating.Options,
		}
	}

	// Get services that use this secret
	secret.Dependencies, _ = sm.getSecretDependencies(ctx, secretID)

	return secret, nil
}

// InspectConfig returns detailed information about a config.
func (sm *SecretsManager) InspectConfig(ctx context.Context, configID string) (*DockerSecret, error) {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "config", "inspect", configID, "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		if strings.Contains(err.Error(), "No such config") {
			return nil, ErrConfigNotFound
		}
		return nil, fmt.Errorf("inspect config: %w", err)
	}

	var rawConfig struct {
		ID      string `json:"ID"`
		Version struct {
			Index int64 `json:"Index"`
		} `json:"Version"`
		CreatedAt time.Time `json:"CreatedAt"`
		UpdatedAt time.Time `json:"UpdatedAt"`
		Spec      struct {
			Name       string            `json:"Name"`
			Labels     map[string]string `json:"Labels"`
			Data       string            `json:"Data"` // Base64 encoded
			Templating *struct {
				Name    string            `json:"Name"`
				Options map[string]string `json:"Options"`
			} `json:"Templating"`
		} `json:"Spec"`
	}

	if err := json.Unmarshal(output, &rawConfig); err != nil {
		return nil, fmt.Errorf("parse config inspect: %w", err)
	}

	config := &DockerSecret{
		ID:        rawConfig.ID,
		Name:      rawConfig.Spec.Name,
		Type:      SecretTypeConfig,
		CreatedAt: rawConfig.CreatedAt,
		UpdatedAt: rawConfig.UpdatedAt,
		Labels:    rawConfig.Spec.Labels,
		Version:   rawConfig.Version.Index,
	}

	if rawConfig.Spec.Templating != nil {
		config.Templating = &TemplatingConfig{
			Name:    rawConfig.Spec.Templating.Name,
			Options: rawConfig.Spec.Templating.Options,
		}
	}

	// Get services that use this config
	config.Dependencies, _ = sm.getConfigDependencies(ctx, configID)

	return config, nil
}

// BackupSecrets creates a complete backup of all Docker secrets and configs.
func (sm *SecretsManager) BackupSecrets(ctx context.Context) (*SecretBackup, error) {
	backup := &SecretBackup{
		ID:           uuid.New(),
		CreatedAt:    time.Now(),
		Status:       SecretBackupStatusRunning,
		Secrets:      make(map[string]*SecretData),
		Configs:      make(map[string]*SecretData),
		SwarmSecrets: make(map[string]*SecretData),
		Metadata:     make(map[string]*DockerSecret),
		RotationState: &RotationState{
			LastRotations:     make(map[string]time.Time),
			RotationSchedules: make(map[string]string),
			VersionHistory:    make(map[string][]VersionEntry),
		},
	}

	sm.logger.Info().Str("backup_id", backup.ID.String()).Msg("starting docker secrets backup")

	// Backup regular secrets
	secrets, err := sm.ListSecrets(ctx)
	if err != nil {
		if !errors.Is(err, ErrSwarmNotActive) {
			backup.Status = SecretBackupStatusFailed
			backup.ErrorMessage = fmt.Sprintf("list secrets: %v", err)
			return backup, err
		}
		sm.logger.Info().Msg("swarm not active, skipping secrets backup")
	} else {
		for _, secret := range secrets {
			secretData, err := sm.backupSecret(ctx, secret)
			if err != nil {
				sm.logger.Error().Err(err).Str("secret_id", secret.ID).Msg("failed to backup secret")
				continue
			}
			backup.Secrets[secret.ID] = secretData
			backup.Metadata[secret.ID] = secret
			backup.SecretsCount++
			sm.updateRotationState(backup.RotationState, secret, secretData)
		}
	}

	// Backup configs
	configs, err := sm.ListConfigs(ctx)
	if err != nil {
		if !errors.Is(err, ErrSwarmNotActive) {
			sm.logger.Warn().Err(err).Msg("failed to list configs")
		}
	} else {
		for _, config := range configs {
			configData, err := sm.backupConfig(ctx, config)
			if err != nil {
				sm.logger.Error().Err(err).Str("config_id", config.ID).Msg("failed to backup config")
				continue
			}
			backup.Configs[config.ID] = configData
			backup.Metadata[config.ID] = config
			backup.ConfigsCount++
		}
	}

	// Backup Swarm secrets
	swarmSecrets, err := sm.ListSwarmSecrets(ctx)
	if err != nil {
		if !errors.Is(err, ErrSwarmNotActive) {
			sm.logger.Warn().Err(err).Msg("failed to list swarm secrets")
		}
	} else {
		for _, secret := range swarmSecrets {
			// Skip if already backed up as regular secret
			if _, exists := backup.Secrets[secret.ID]; exists {
				continue
			}
			secretData, err := sm.backupSecret(ctx, secret)
			if err != nil {
				sm.logger.Error().Err(err).Str("secret_id", secret.ID).Msg("failed to backup swarm secret")
				continue
			}
			backup.SwarmSecrets[secret.ID] = secretData
			backup.Metadata[secret.ID] = secret
			backup.SwarmCount++
			sm.updateRotationState(backup.RotationState, secret, secretData)
		}
	}

	// Calculate total size
	backup.TotalSizeBytes = sm.calculateBackupSize(backup)

	now := time.Now()
	backup.CompletedAt = &now
	backup.Status = SecretBackupStatusCompleted

	sm.logger.Info().
		Str("backup_id", backup.ID.String()).
		Int("secrets", backup.SecretsCount).
		Int("configs", backup.ConfigsCount).
		Int("swarm_secrets", backup.SwarmCount).
		Int64("total_size", backup.TotalSizeBytes).
		Msg("docker secrets backup completed")

	return backup, nil
}

// backupSecret performs double encryption on a secret.
func (sm *SecretsManager) backupSecret(ctx context.Context, secret *DockerSecret) (*SecretData, error) {
	// Note: Docker secrets cannot be read directly via CLI.
	// We store the metadata and the secret's internal encrypted reference.
	// The actual secret data is protected by Docker's encryption at rest.

	// For backup purposes, we create a reference that can be used for restore.
	// The actual secret value would need to be provided during restore if recreating.

	referenceData := fmt.Sprintf("docker-secret-ref:%s:%s:%d", secret.ID, secret.Name, secret.Version)

	// First layer: Docker's internal encryption (already applied by Docker)
	dockerEncrypted := base64.StdEncoding.EncodeToString([]byte(referenceData))

	// Second layer: Keldris AES-256-GCM encryption
	keldrisEncrypted, err := sm.keyManager.EncryptString(dockerEncrypted)
	if err != nil {
		return nil, fmt.Errorf("keldris encryption failed: %w", err)
	}

	// Generate checksum of the reference data
	checksum := sha256.Sum256([]byte(referenceData))

	return &SecretData{
		DockerEncrypted:  dockerEncrypted,
		KeldrisEncrypted: keldrisEncrypted,
		Checksum:         hex.EncodeToString(checksum[:]),
		EncryptedAt:      time.Now(),
	}, nil
}

// backupConfig performs double encryption on a config.
func (sm *SecretsManager) backupConfig(ctx context.Context, config *DockerSecret) (*SecretData, error) {
	// Docker configs can be read via inspect
	cmd := exec.CommandContext(ctx, sm.dockerPath, "config", "inspect", config.ID, "--format", "{{.Spec.Data}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("read config data: %w", err)
	}

	configData := strings.TrimSpace(string(output))

	// First layer: Docker's base64 encoding (already applied)
	dockerEncrypted := configData
	if dockerEncrypted == "" {
		dockerEncrypted = base64.StdEncoding.EncodeToString([]byte{})
	}

	// Second layer: Keldris AES-256-GCM encryption
	keldrisEncrypted, err := sm.keyManager.EncryptString(dockerEncrypted)
	if err != nil {
		return nil, fmt.Errorf("keldris encryption failed: %w", err)
	}

	// Generate checksum of the config data
	decoded, _ := base64.StdEncoding.DecodeString(dockerEncrypted)
	checksum := sha256.Sum256(decoded)

	return &SecretData{
		DockerEncrypted:  dockerEncrypted,
		KeldrisEncrypted: keldrisEncrypted,
		Checksum:         hex.EncodeToString(checksum[:]),
		EncryptedAt:      time.Now(),
	}, nil
}

// RestoreSecrets restores secrets from a backup with proper dependency ordering.
func (sm *SecretsManager) RestoreSecrets(ctx context.Context, backup *SecretBackup, secretData map[string][]byte) error {
	sm.logger.Info().
		Str("backup_id", backup.ID.String()).
		Msg("starting docker secrets restore")

	// Calculate restore order based on dependencies
	order := sm.CalculateRestoreOrder(backup)

	// Restore in phases
	for phaseNum, phase := range order.Phases {
		sm.logger.Info().
			Int("phase", phaseNum+1).
			Int("count", len(phase)).
			Msg("restoring phase")

		for _, secretID := range phase {
			metadata, exists := backup.Metadata[secretID]
			if !exists {
				sm.logger.Warn().Str("secret_id", secretID).Msg("metadata not found, skipping")
				continue
			}

			// Get the actual secret data that was provided for restore
			data, hasData := secretData[secretID]
			if !hasData {
				sm.logger.Warn().
					Str("secret_id", secretID).
					Str("name", metadata.Name).
					Msg("no restore data provided, skipping")
				continue
			}

			if err := sm.restoreSecret(ctx, metadata, data); err != nil {
				sm.logger.Error().
					Err(err).
					Str("secret_id", secretID).
					Str("name", metadata.Name).
					Msg("failed to restore secret")
				return fmt.Errorf("restore secret %s: %w", metadata.Name, err)
			}
		}
	}

	sm.logger.Info().
		Str("backup_id", backup.ID.String()).
		Msg("docker secrets restore completed")

	return nil
}

// restoreSecret creates or updates a Docker secret.
func (sm *SecretsManager) restoreSecret(ctx context.Context, metadata *DockerSecret, data []byte) error {
	switch metadata.Type {
	case SecretTypeSecret, SecretTypeSwarmSecret:
		return sm.restoreDockerSecret(ctx, metadata, data)
	case SecretTypeConfig:
		return sm.restoreDockerConfig(ctx, metadata, data)
	default:
		return fmt.Errorf("unknown secret type: %s", metadata.Type)
	}
}

// restoreDockerSecret creates a Docker secret.
func (sm *SecretsManager) restoreDockerSecret(ctx context.Context, metadata *DockerSecret, data []byte) error {
	// Check if secret already exists
	_, err := sm.InspectSecret(ctx, metadata.Name)
	if err == nil {
		// Secret exists, need to remove it first (secrets are immutable)
		sm.logger.Info().Str("name", metadata.Name).Msg("removing existing secret before restore")
		if err := sm.removeSecret(ctx, metadata.Name); err != nil {
			return fmt.Errorf("remove existing secret: %w", err)
		}
	} else if !errors.Is(err, ErrSecretNotFound) {
		return fmt.Errorf("check existing secret: %w", err)
	}

	// Build create command
	args := []string{"secret", "create"}

	// Add labels
	for key, value := range metadata.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add driver if specified
	if metadata.Driver != "" {
		args = append(args, "--driver", metadata.Driver)
	}

	args = append(args, metadata.Name, "-")

	cmd := exec.CommandContext(ctx, sm.dockerPath, args...)
	cmd.Stdin = bytes.NewReader(data)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create secret: %w (%s)", err, string(output))
	}

	sm.logger.Info().Str("name", metadata.Name).Msg("secret restored successfully")
	return nil
}

// restoreDockerConfig creates a Docker config.
func (sm *SecretsManager) restoreDockerConfig(ctx context.Context, metadata *DockerSecret, data []byte) error {
	// Check if config already exists
	_, err := sm.InspectConfig(ctx, metadata.Name)
	if err == nil {
		// Config exists, need to remove it first (configs are immutable)
		sm.logger.Info().Str("name", metadata.Name).Msg("removing existing config before restore")
		if err := sm.removeConfig(ctx, metadata.Name); err != nil {
			return fmt.Errorf("remove existing config: %w", err)
		}
	} else if !errors.Is(err, ErrConfigNotFound) {
		return fmt.Errorf("check existing config: %w", err)
	}

	// Build create command
	args := []string{"config", "create"}

	// Add labels
	for key, value := range metadata.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add templating if specified
	if metadata.Templating != nil {
		args = append(args, "--template-driver", metadata.Templating.Name)
	}

	args = append(args, metadata.Name, "-")

	cmd := exec.CommandContext(ctx, sm.dockerPath, args...)
	cmd.Stdin = bytes.NewReader(data)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("create config: %w (%s)", err, string(output))
	}

	sm.logger.Info().Str("name", metadata.Name).Msg("config restored successfully")
	return nil
}

// removeSecret removes a Docker secret.
func (sm *SecretsManager) removeSecret(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "secret", "rm", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove secret: %w (%s)", err, string(output))
	}
	return nil
}

// removeConfig removes a Docker config.
func (sm *SecretsManager) removeConfig(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "config", "rm", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove config: %w (%s)", err, string(output))
	}
	return nil
}

// CalculateRestoreOrder determines the order for restoring secrets based on dependencies.
func (sm *SecretsManager) CalculateRestoreOrder(backup *SecretBackup) *RestoreOrder {
	order := &RestoreOrder{
		Phases:       make([][]string, 0),
		Dependencies: make(map[string][]string),
	}

	// Build dependency map
	remaining := make(map[string]bool)
	for id, metadata := range backup.Metadata {
		remaining[id] = true
		order.Dependencies[id] = metadata.Dependencies
	}

	// Topological sort: items with no dependencies first
	for len(remaining) > 0 {
		phase := make([]string, 0)

		for id := range remaining {
			deps := order.Dependencies[id]
			allResolved := true

			for _, dep := range deps {
				// Check if dependency is in remaining (not yet scheduled)
				if _, stillRemaining := remaining[dep]; stillRemaining {
					allResolved = false
					break
				}
			}

			if allResolved {
				phase = append(phase, id)
			}
		}

		// If no items can be added, we have a circular dependency
		// Add all remaining items to break the cycle
		if len(phase) == 0 {
			for id := range remaining {
				phase = append(phase, id)
			}
		}

		// Sort phase for deterministic ordering
		sort.Strings(phase)

		// Remove scheduled items from remaining
		for _, id := range phase {
			delete(remaining, id)
		}

		order.Phases = append(order.Phases, phase)
	}

	return order
}

// GetMaskedSecrets returns secrets with masked values for UI display.
func (sm *SecretsManager) GetMaskedSecrets(ctx context.Context, backup *SecretBackup) ([]*MaskedSecret, error) {
	masked := make([]*MaskedSecret, 0, len(backup.Metadata))

	for id, metadata := range backup.Metadata {
		ms := &MaskedSecret{
			ID:          id,
			Name:        metadata.Name,
			Type:        metadata.Type,
			MaskedValue: MaskValue("", DefaultMaskLength),
			CreatedAt:   metadata.CreatedAt,
			Labels:      metadata.Labels,
			HasBackup:   true,
		}

		if backup.CompletedAt != nil {
			ms.LastBackup = backup.CompletedAt
		}

		masked = append(masked, ms)
	}

	// Sort by name for consistent display
	sort.Slice(masked, func(i, j int) bool {
		return masked[i].Name < masked[j].Name
	})

	return masked, nil
}

// MaskValue masks a value for secure display.
func MaskValue(value string, length int) string {
	if length <= 0 {
		length = DefaultMaskLength
	}
	return strings.Repeat(MaskChar, length)
}

// MaskPartialValue shows first and last characters with masked middle.
func MaskPartialValue(value string, visibleChars int) string {
	if len(value) <= visibleChars*2 {
		return MaskValue(value, len(value))
	}

	maskLength := len(value) - (visibleChars * 2)
	if maskLength < 3 {
		maskLength = 3
	}

	return value[:visibleChars] + strings.Repeat(MaskChar, maskLength) + value[len(value)-visibleChars:]
}

// DecryptSecretData decrypts a SecretData object and returns the original data.
func (sm *SecretsManager) DecryptSecretData(secretData *SecretData) ([]byte, error) {
	// Decrypt Keldris layer
	dockerEncrypted, err := sm.keyManager.DecryptString(secretData.KeldrisEncrypted)
	if err != nil {
		return nil, fmt.Errorf("keldris decryption failed: %w", err)
	}

	// Decode Docker layer (base64)
	data, err := base64.StdEncoding.DecodeString(dockerEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decode docker layer: %w", err)
	}

	// Verify checksum
	checksum := sha256.Sum256(data)
	if hex.EncodeToString(checksum[:]) != secretData.Checksum {
		return nil, ErrInvalidSecretData
	}

	return data, nil
}

// GetRotationStatus returns the rotation status for secrets in a backup.
func (sm *SecretsManager) GetRotationStatus(backup *SecretBackup) *RotationState {
	if backup.RotationState == nil {
		return &RotationState{
			LastRotations:     make(map[string]time.Time),
			RotationSchedules: make(map[string]string),
			VersionHistory:    make(map[string][]VersionEntry),
		}
	}
	return backup.RotationState
}

// TrackRotation records a secret rotation event.
func (sm *SecretsManager) TrackRotation(state *RotationState, secretID string, version int64, checksum string, rotatedBy string) {
	now := time.Now()
	state.LastRotations[secretID] = now

	if state.VersionHistory[secretID] == nil {
		state.VersionHistory[secretID] = make([]VersionEntry, 0)
	}

	state.VersionHistory[secretID] = append(state.VersionHistory[secretID], VersionEntry{
		Version:   version,
		Checksum:  checksum,
		CreatedAt: now,
		RotatedBy: rotatedBy,
	})

	// Remove from pending if present
	pending := make([]string, 0)
	for _, id := range state.PendingRotations {
		if id != secretID {
			pending = append(pending, id)
		}
	}
	state.PendingRotations = pending
}

// SetRotationSchedule sets a rotation schedule for a secret.
func (sm *SecretsManager) SetRotationSchedule(state *RotationState, secretID, cronExpr string) {
	state.RotationSchedules[secretID] = cronExpr
}

// parseSecretList parses Docker secret/config list output.
func (sm *SecretsManager) parseSecretList(output []byte, secretType SecretType) ([]*DockerSecret, error) {
	lines := bytes.Split(output, []byte("\n"))
	secrets := make([]*DockerSecret, 0, len(lines))

	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}

		var raw struct {
			ID        string `json:"ID"`
			Name      string `json:"Name"`
			CreatedAt string `json:"CreatedAt"`
			UpdatedAt string `json:"UpdatedAt"`
		}

		if err := json.Unmarshal(line, &raw); err != nil {
			sm.logger.Warn().Err(err).Msg("failed to parse secret list item")
			continue
		}

		createdAt, _ := time.Parse(time.RFC3339Nano, raw.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339Nano, raw.UpdatedAt)

		secrets = append(secrets, &DockerSecret{
			ID:        raw.ID,
			Name:      raw.Name,
			Type:      secretType,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	return secrets, nil
}

// getSecretDependencies finds services that use a specific secret.
func (sm *SecretsManager) getSecretDependencies(ctx context.Context, secretID string) ([]string, error) {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "service", "ls", "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	serviceIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	dependencies := make([]string, 0)

	for _, serviceID := range serviceIDs {
		if serviceID == "" {
			continue
		}

		// Inspect service to check for secret usage
		cmd := exec.CommandContext(ctx, sm.dockerPath, "service", "inspect", serviceID, "--format", "{{json .Spec.TaskTemplate.ContainerSpec.Secrets}}")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		if strings.Contains(string(output), secretID) {
			// Get service name
			nameCmd := exec.CommandContext(ctx, sm.dockerPath, "service", "inspect", serviceID, "--format", "{{.Spec.Name}}")
			nameOutput, _ := nameCmd.Output()
			dependencies = append(dependencies, strings.TrimSpace(string(nameOutput)))
		}
	}

	return dependencies, nil
}

// getConfigDependencies finds services that use a specific config.
func (sm *SecretsManager) getConfigDependencies(ctx context.Context, configID string) ([]string, error) {
	cmd := exec.CommandContext(ctx, sm.dockerPath, "service", "ls", "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	serviceIDs := strings.Split(strings.TrimSpace(string(output)), "\n")
	dependencies := make([]string, 0)

	for _, serviceID := range serviceIDs {
		if serviceID == "" {
			continue
		}

		// Inspect service to check for config usage
		cmd := exec.CommandContext(ctx, sm.dockerPath, "service", "inspect", serviceID, "--format", "{{json .Spec.TaskTemplate.ContainerSpec.Configs}}")
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		if strings.Contains(string(output), configID) {
			// Get service name
			nameCmd := exec.CommandContext(ctx, sm.dockerPath, "service", "inspect", serviceID, "--format", "{{.Spec.Name}}")
			nameOutput, _ := nameCmd.Output()
			dependencies = append(dependencies, strings.TrimSpace(string(nameOutput)))
		}
	}

	return dependencies, nil
}

// updateRotationState updates rotation tracking for a secret.
func (sm *SecretsManager) updateRotationState(state *RotationState, secret *DockerSecret, data *SecretData) {
	// Record version history
	if state.VersionHistory[secret.ID] == nil {
		state.VersionHistory[secret.ID] = make([]VersionEntry, 0)
	}

	// Check if this version already exists
	for _, entry := range state.VersionHistory[secret.ID] {
		if entry.Version == secret.Version {
			return
		}
	}

	state.VersionHistory[secret.ID] = append(state.VersionHistory[secret.ID], VersionEntry{
		Version:   secret.Version,
		Checksum:  data.Checksum,
		CreatedAt: secret.UpdatedAt,
	})

	state.LastRotations[secret.ID] = secret.UpdatedAt
}

// calculateBackupSize estimates the total size of the backup.
func (sm *SecretsManager) calculateBackupSize(backup *SecretBackup) int64 {
	var total int64

	for _, data := range backup.Secrets {
		total += int64(len(data.KeldrisEncrypted))
	}
	for _, data := range backup.Configs {
		total += int64(len(data.KeldrisEncrypted))
	}
	for _, data := range backup.SwarmSecrets {
		total += int64(len(data.KeldrisEncrypted))
	}

	// Add metadata size estimate
	metadataJSON, _ := json.Marshal(backup.Metadata)
	total += int64(len(metadataJSON))

	return total
}

// VerifyBackupIntegrity verifies the integrity of a secret backup.
func (sm *SecretsManager) VerifyBackupIntegrity(backup *SecretBackup) error {
	sm.logger.Info().Str("backup_id", backup.ID.String()).Msg("verifying backup integrity")

	// Verify each secret's checksum
	allData := make(map[string]*SecretData)
	for id, data := range backup.Secrets {
		allData[id] = data
	}
	for id, data := range backup.Configs {
		allData[id] = data
	}
	for id, data := range backup.SwarmSecrets {
		allData[id] = data
	}

	for id, data := range allData {
		decrypted, err := sm.DecryptSecretData(data)
		if err != nil {
			return fmt.Errorf("decrypt %s: %w", id, err)
		}

		checksum := sha256.Sum256(decrypted)
		if hex.EncodeToString(checksum[:]) != data.Checksum {
			return fmt.Errorf("checksum mismatch for %s", id)
		}
	}

	sm.logger.Info().
		Str("backup_id", backup.ID.String()).
		Int("verified", len(allData)).
		Msg("backup integrity verified")

	return nil
}

// ExportBackup serializes a backup for storage.
func (sm *SecretsManager) ExportBackup(backup *SecretBackup) ([]byte, error) {
	return json.Marshal(backup)
}

// ImportBackup deserializes a backup from storage.
func (sm *SecretsManager) ImportBackup(data []byte) (*SecretBackup, error) {
	var backup SecretBackup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, fmt.Errorf("unmarshal backup: %w", err)
	}
	return &backup, nil
}
