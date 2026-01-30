package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DiscoveryStore defines the interface for Docker container configuration storage.
type DiscoveryStore interface {
	// GetDockerContainersByAgentID returns all Docker containers for an agent.
	GetDockerContainersByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerContainerConfig, error)

	// GetDockerContainerByID returns a Docker container by ID.
	GetDockerContainerByID(ctx context.Context, id uuid.UUID) (*models.DockerContainerConfig, error)

	// GetDockerContainerByContainerID returns a Docker container by its Docker container ID.
	GetDockerContainerByContainerID(ctx context.Context, agentID uuid.UUID, containerID string) (*models.DockerContainerConfig, error)

	// CreateDockerContainer creates a new Docker container configuration.
	CreateDockerContainer(ctx context.Context, config *models.DockerContainerConfig) error

	// UpdateDockerContainer updates an existing Docker container configuration.
	UpdateDockerContainer(ctx context.Context, config *models.DockerContainerConfig) error

	// DeleteDockerContainer deletes a Docker container configuration.
	DeleteDockerContainer(ctx context.Context, id uuid.UUID) error

	// GetAgentByID returns an agent by ID.
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
}

// DockerClient defines the interface for Docker API operations.
// This interface allows for mocking in tests and using different Docker client implementations.
type DockerClient interface {
	// ListContainers returns all containers with their labels.
	ListContainers(ctx context.Context) ([]ContainerInfo, error)

	// InspectContainer returns detailed information about a container.
	InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error)

	// ExecCommand executes a command inside a container and returns the output.
	ExecCommand(ctx context.Context, containerID string, cmd []string) (string, error)

	// StopContainer stops a container.
	StopContainer(ctx context.Context, containerID string, timeout *time.Duration) error

	// StartContainer starts a container.
	StartContainer(ctx context.Context, containerID string) error
}

// DiscoveryConfig holds configuration for the discovery service.
type DiscoveryConfig struct {
	// RefreshInterval is how often to auto-discover containers.
	RefreshInterval time.Duration

	// AutoEnable automatically enables backup for newly discovered containers with labels.
	AutoEnable bool

	// RemoveStale removes configurations for containers that no longer exist.
	RemoveStale bool

	// StaleThreshold is how long a container must be missing before removal.
	StaleThreshold time.Duration
}

// DefaultDiscoveryConfig returns a DiscoveryConfig with sensible defaults.
func DefaultDiscoveryConfig() DiscoveryConfig {
	return DiscoveryConfig{
		RefreshInterval: 5 * time.Minute,
		AutoEnable:      true,
		RemoveStale:     false,
		StaleThreshold:  24 * time.Hour,
	}
}

// DiscoveryService manages Docker container discovery and backup configuration.
type DiscoveryService struct {
	store        DiscoveryStore
	parser       *LabelParser
	config       DiscoveryConfig
	logger       zerolog.Logger
	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
	dockerClient DockerClient // Optional, set via SetDockerClient
}

// NewDiscoveryService creates a new DiscoveryService.
func NewDiscoveryService(store DiscoveryStore, config DiscoveryConfig, logger zerolog.Logger) *DiscoveryService {
	return &DiscoveryService{
		store:  store,
		parser: NewLabelParser(),
		config: config,
		logger: logger.With().Str("component", "docker_discovery").Logger(),
		stopCh: make(chan struct{}),
	}
}

// SetDockerClient sets the Docker client for container operations.
// This allows the service to be used with different Docker client implementations.
func (s *DiscoveryService) SetDockerClient(client DockerClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dockerClient = client
}

// Start starts the discovery service background refresh loop.
func (s *DiscoveryService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("discovery service already running")
	}
	s.running = true
	s.mu.Unlock()

	s.logger.Info().
		Dur("refresh_interval", s.config.RefreshInterval).
		Bool("auto_enable", s.config.AutoEnable).
		Msg("starting docker discovery service")

	go s.refreshLoop(ctx)

	return nil
}

// Stop stops the discovery service.
func (s *DiscoveryService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	close(s.stopCh)
	s.logger.Info().Msg("docker discovery service stopped")
}

// refreshLoop periodically discovers containers.
func (s *DiscoveryService) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(s.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()

			if !running {
				return
			}

			// Note: Discovery is triggered per-agent by the agent service
			// This loop is here for future use with server-side Docker discovery
		}
	}
}

// DiscoverContainers discovers Docker containers on an agent and syncs configurations.
// This method is called when an agent reports its container information.
func (s *DiscoveryService) DiscoverContainers(ctx context.Context, agentID uuid.UUID, containers []ContainerInfo) (*models.DockerDiscoveryResult, error) {
	logger := s.logger.With().Str("agent_id", agentID.String()).Logger()
	logger.Info().Int("container_count", len(containers)).Msg("discovering containers")

	result := &models.DockerDiscoveryResult{
		Containers:   make([]*models.DockerContainerConfig, 0),
		DiscoveredAt: time.Now(),
	}

	// Get existing configurations for this agent
	existingConfigs, err := s.store.GetDockerContainersByAgentID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("get existing configs: %w", err)
	}

	// Build a map of existing configs by container ID
	existingByContainerID := make(map[string]*models.DockerContainerConfig)
	for _, config := range existingConfigs {
		existingByContainerID[config.ContainerID] = config
	}

	// Track which containers we've seen
	seenContainerIDs := make(map[string]bool)

	// Process each discovered container
	for _, container := range containers {
		seenContainerIDs[container.ID] = true

		// Parse labels into config
		config := s.parser.ParseConfig(agentID, container)
		if config == nil {
			// Container doesn't have backup labels, skip
			continue
		}

		result.TotalDiscovered++
		if config.Enabled {
			result.TotalEnabled++
		}

		// Check if this container already exists
		existing, exists := existingByContainerID[container.ID]
		if exists {
			// Update existing configuration (preserve overrides)
			s.updateExistingConfig(existing, config)
			existing.UpdatedAt = time.Now()

			if err := s.store.UpdateDockerContainer(ctx, existing); err != nil {
				logger.Error().Err(err).
					Str("container_id", container.ID).
					Msg("failed to update container config")
				continue
			}

			// Apply overrides for the response
			existing.ApplyOverrides()
			result.Containers = append(result.Containers, existing)
		} else {
			// Create new configuration
			if err := s.store.CreateDockerContainer(ctx, config); err != nil {
				logger.Error().Err(err).
					Str("container_id", container.ID).
					Msg("failed to create container config")
				continue
			}

			result.NewContainers++
			result.Containers = append(result.Containers, config)

			logger.Info().
				Str("container_id", container.ID).
				Str("container_name", container.Name).
				Str("schedule", string(config.Schedule)).
				Msg("new container discovered")
		}
	}

	// Handle stale containers if configured
	if s.config.RemoveStale {
		for containerID, config := range existingByContainerID {
			if !seenContainerIDs[containerID] {
				// Container no longer exists
				staleDuration := time.Since(config.UpdatedAt)
				if staleDuration > s.config.StaleThreshold {
					if err := s.store.DeleteDockerContainer(ctx, config.ID); err != nil {
						logger.Error().Err(err).
							Str("container_id", containerID).
							Msg("failed to delete stale container config")
						continue
					}

					result.RemovedContainers++
					logger.Info().
						Str("container_id", containerID).
						Dur("stale_duration", staleDuration).
						Msg("removed stale container config")
				}
			}
		}
	}

	logger.Info().
		Int("total_discovered", result.TotalDiscovered).
		Int("total_enabled", result.TotalEnabled).
		Int("new_containers", result.NewContainers).
		Int("removed_containers", result.RemovedContainers).
		Msg("container discovery completed")

	return result, nil
}

// updateExistingConfig updates an existing config with new label values while preserving overrides.
func (s *DiscoveryService) updateExistingConfig(existing, newConfig *models.DockerContainerConfig) {
	// Update fields that come from labels (not overrides)
	existing.ContainerName = newConfig.ContainerName
	existing.ImageName = newConfig.ImageName
	existing.Labels = newConfig.Labels

	// Only update non-override fields if no overrides exist
	if existing.Overrides == nil {
		existing.Schedule = newConfig.Schedule
		existing.CronExpression = newConfig.CronExpression
		existing.Excludes = newConfig.Excludes
		existing.PreHook = newConfig.PreHook
		existing.PostHook = newConfig.PostHook
		existing.StopOnBackup = newConfig.StopOnBackup
		existing.BackupVolumes = newConfig.BackupVolumes
		existing.BackupBindMounts = newConfig.BackupBindMounts
		existing.Enabled = newConfig.Enabled
	}
}

// GetContainerConfig returns the backup configuration for a container.
func (s *DiscoveryService) GetContainerConfig(ctx context.Context, id uuid.UUID) (*models.DockerContainerConfig, error) {
	config, err := s.store.GetDockerContainerByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Apply overrides
	config.ApplyOverrides()
	return config, nil
}

// GetContainersByAgent returns all backup configurations for an agent.
func (s *DiscoveryService) GetContainersByAgent(ctx context.Context, agentID uuid.UUID) ([]*models.DockerContainerConfig, error) {
	configs, err := s.store.GetDockerContainersByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Apply overrides to each config
	for _, config := range configs {
		config.ApplyOverrides()
	}

	return configs, nil
}

// UpdateContainerOverrides updates the UI overrides for a container configuration.
func (s *DiscoveryService) UpdateContainerOverrides(ctx context.Context, id uuid.UUID, overrides *models.ContainerOverrides) error {
	config, err := s.store.GetDockerContainerByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get container: %w", err)
	}

	config.Overrides = overrides
	config.UpdatedAt = time.Now()

	if err := s.store.UpdateDockerContainer(ctx, config); err != nil {
		return fmt.Errorf("update container: %w", err)
	}

	s.logger.Info().
		Str("container_id", config.ContainerID).
		Msg("container overrides updated")

	return nil
}

// DeleteContainerConfig deletes a container backup configuration.
func (s *DiscoveryService) DeleteContainerConfig(ctx context.Context, id uuid.UUID) error {
	config, err := s.store.GetDockerContainerByID(ctx, id)
	if err != nil {
		return fmt.Errorf("get container: %w", err)
	}

	if err := s.store.DeleteDockerContainer(ctx, id); err != nil {
		return fmt.Errorf("delete container: %w", err)
	}

	s.logger.Info().
		Str("container_id", config.ContainerID).
		Msg("container config deleted")

	return nil
}

// ValidateContainerLabels validates the labels for a container.
func (s *DiscoveryService) ValidateContainerLabels(labels map[string]string) []string {
	return s.parser.ValidateLabels(labels)
}

// GetLabelDocs returns documentation for Docker backup labels.
func (s *DiscoveryService) GetLabelDocs() *models.DockerLabelDocs {
	return s.parser.GenerateLabelDocs()
}

// GetDockerComposeExample returns an example Docker Compose configuration.
func (s *DiscoveryService) GetDockerComposeExample() string {
	return s.parser.GenerateDockerComposeExample()
}

// GetDockerRunExample returns an example docker run command.
func (s *DiscoveryService) GetDockerRunExample() string {
	return s.parser.GenerateDockerRunExample()
}

// GetEnabledContainersForBackup returns all enabled containers ready for backup on an agent.
func (s *DiscoveryService) GetEnabledContainersForBackup(ctx context.Context, agentID uuid.UUID) ([]*models.DockerContainerConfig, error) {
	configs, err := s.store.GetDockerContainersByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	var enabled []*models.DockerContainerConfig
	for _, config := range configs {
		// Apply overrides first
		config.ApplyOverrides()

		if config.Enabled {
			enabled = append(enabled, config)
		}
	}

	return enabled, nil
}

// RefreshContainer manually refreshes a single container's configuration from Docker.
// This is used when the Docker client is available on the server side.
func (s *DiscoveryService) RefreshContainer(ctx context.Context, agentID uuid.UUID, containerID string) (*models.DockerContainerConfig, error) {
	s.mu.RLock()
	client := s.dockerClient
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("docker client not available")
	}

	container, err := client.InspectContainer(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	// Parse labels into config
	config := s.parser.ParseConfig(agentID, *container)
	if config == nil {
		return nil, fmt.Errorf("container does not have backup labels")
	}

	// Check if this container already exists
	existing, err := s.store.GetDockerContainerByContainerID(ctx, agentID, containerID)
	if err == nil {
		// Update existing configuration
		s.updateExistingConfig(existing, config)
		existing.UpdatedAt = time.Now()

		if err := s.store.UpdateDockerContainer(ctx, existing); err != nil {
			return nil, fmt.Errorf("update container: %w", err)
		}

		existing.ApplyOverrides()
		return existing, nil
	}

	// Create new configuration
	if err := s.store.CreateDockerContainer(ctx, config); err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	return config, nil
}
