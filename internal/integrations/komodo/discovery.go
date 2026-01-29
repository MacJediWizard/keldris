package komodo

import (
	"context"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DiscoveryService handles auto-discovery of Komodo stacks and containers
type DiscoveryService struct {
	client *Client
	logger zerolog.Logger
}

// NewDiscoveryService creates a new discovery service
func NewDiscoveryService(client *Client, logger zerolog.Logger) *DiscoveryService {
	return &DiscoveryService{
		client: client,
		logger: logger.With().Str("component", "komodo_discovery").Logger(),
	}
}

// DiscoverAll discovers all stacks and containers from Komodo
func (s *DiscoveryService) DiscoverAll(ctx context.Context, orgID, integrationID uuid.UUID) (*models.KomodoDiscoveryResult, error) {
	s.logger.Info().
		Str("integration_id", integrationID.String()).
		Msg("starting Komodo discovery")

	result := &models.KomodoDiscoveryResult{
		Stacks:       make([]*models.KomodoStack, 0),
		Containers:   make([]*models.KomodoContainer, 0),
		DiscoveredAt: time.Now(),
	}

	// Discover stacks
	stacks, err := s.discoverStacks(ctx, orgID, integrationID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to discover stacks")
		// Continue with containers even if stacks fail
	} else {
		result.Stacks = stacks
	}

	// Discover containers
	containers, err := s.discoverContainers(ctx, orgID, integrationID)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to discover containers")
	} else {
		result.Containers = containers
	}

	s.logger.Info().
		Int("stacks", len(result.Stacks)).
		Int("containers", len(result.Containers)).
		Msg("Komodo discovery completed")

	return result, nil
}

// discoverStacks fetches and converts stacks from Komodo API
func (s *DiscoveryService) discoverStacks(ctx context.Context, orgID, integrationID uuid.UUID) ([]*models.KomodoStack, error) {
	apiStacks, err := s.client.ListStacks(ctx)
	if err != nil {
		return nil, err
	}

	stacks := make([]*models.KomodoStack, 0, len(apiStacks))
	for _, apiStack := range apiStacks {
		stack := s.convertAPIStack(orgID, integrationID, &apiStack)
		stacks = append(stacks, stack)
	}

	return stacks, nil
}

// discoverContainers fetches and converts containers from Komodo API
func (s *DiscoveryService) discoverContainers(ctx context.Context, orgID, integrationID uuid.UUID) ([]*models.KomodoContainer, error) {
	apiContainers, err := s.client.ListContainers(ctx)
	if err != nil {
		return nil, err
	}

	containers := make([]*models.KomodoContainer, 0, len(apiContainers))
	for _, apiContainer := range apiContainers {
		container := s.convertAPIContainer(orgID, integrationID, &apiContainer)
		containers = append(containers, container)
	}

	return containers, nil
}

// convertAPIStack converts a Komodo API stack to a Keldris model
func (s *DiscoveryService) convertAPIStack(orgID, integrationID uuid.UUID, apiStack *APIStack) *models.KomodoStack {
	stack := models.NewKomodoStack(orgID, integrationID, apiStack.ID, apiStack.Name)
	stack.ServerID = apiStack.ServerID
	stack.ServerName = apiStack.ServerName
	stack.ContainerCount = apiStack.ContainerCount
	stack.RunningCount = apiStack.RunningCount
	return stack
}

// convertAPIContainer converts a Komodo API container to a Keldris model
func (s *DiscoveryService) convertAPIContainer(orgID, integrationID uuid.UUID, apiContainer *APIContainer) *models.KomodoContainer {
	container := models.NewKomodoContainer(orgID, integrationID, apiContainer.ID, apiContainer.Name)
	container.Image = apiContainer.Image
	container.StackID = apiContainer.StackID
	container.StackName = apiContainer.StackName
	container.Status = s.mapContainerStatus(apiContainer.State)
	container.Labels = apiContainer.Labels

	// Extract volume paths
	if len(apiContainer.Volumes) > 0 {
		volumes := make([]string, 0, len(apiContainer.Volumes))
		for _, v := range apiContainer.Volumes {
			volumes = append(volumes, v.Source+":"+v.Destination)
		}
		container.Volumes = volumes
	}

	return container
}

// mapContainerStatus maps Komodo container state to Keldris status
func (s *DiscoveryService) mapContainerStatus(state string) models.KomodoContainerStatus {
	switch state {
	case "running", "started", "up":
		return models.KomodoContainerRunning
	case "stopped", "exited", "dead":
		return models.KomodoContainerStopped
	case "restarting":
		return models.KomodoContainerRestarting
	default:
		return models.KomodoContainerUnknown
	}
}

// DiscoverStackContainers discovers containers for a specific stack
func (s *DiscoveryService) DiscoverStackContainers(ctx context.Context, orgID, integrationID uuid.UUID, stackID string) ([]*models.KomodoContainer, error) {
	apiContainers, err := s.client.ListContainersByStack(ctx, stackID)
	if err != nil {
		return nil, err
	}

	containers := make([]*models.KomodoContainer, 0, len(apiContainers))
	for _, apiContainer := range apiContainers {
		container := s.convertAPIContainer(orgID, integrationID, &apiContainer)
		containers = append(containers, container)
	}

	return containers, nil
}

// GetContainerVolumes returns the volume paths for a container that should be backed up
func (s *DiscoveryService) GetContainerVolumes(ctx context.Context, containerID string) ([]string, error) {
	container, err := s.client.GetContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	volumes := make([]string, 0, len(container.Volumes))
	for _, v := range container.Volumes {
		// Only include writable volumes from the host
		if v.RW && v.Source != "" {
			volumes = append(volumes, v.Source)
		}
	}

	return volumes, nil
}

// ShouldBackupContainer determines if a container should be backed up based on labels
func (s *DiscoveryService) ShouldBackupContainer(container *APIContainer) bool {
	if container.Labels == nil {
		return false
	}

	// Check for backup labels
	if val, ok := container.Labels["keldris.backup"]; ok {
		return val == "true" || val == "enabled"
	}
	if val, ok := container.Labels["backup.enabled"]; ok {
		return val == "true"
	}

	return false
}

// GetBackupPathsFromLabels extracts backup paths from container labels
func (s *DiscoveryService) GetBackupPathsFromLabels(container *APIContainer) []string {
	if container.Labels == nil {
		return nil
	}

	// Check for path labels
	if val, ok := container.Labels["keldris.backup.paths"]; ok && val != "" {
		return splitPaths(val)
	}
	if val, ok := container.Labels["backup.paths"]; ok && val != "" {
		return splitPaths(val)
	}

	return nil
}

// splitPaths splits a comma or semicolon separated path string
func splitPaths(s string) []string {
	paths := make([]string, 0)
	current := ""
	for _, c := range s {
		if c == ',' || c == ';' {
			if current != "" {
				paths = append(paths, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		paths = append(paths, current)
	}
	return paths
}
