package export

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// ImporterStore defines the interface for data access needed by the importer.
type ImporterStore interface {
	// Agent operations
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	CreateAgent(ctx context.Context, agent *models.Agent) error

	// Schedule operations
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) error
	UpdateSchedule(ctx context.Context, schedule *models.Schedule) error
	SetScheduleRepositories(ctx context.Context, scheduleID uuid.UUID, repos []models.ScheduleRepository) error

	// Repository operations
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
}

// Importer handles configuration imports.
type Importer struct {
	store  ImporterStore
	logger zerolog.Logger
}

// NewImporter creates a new Importer.
func NewImporter(store ImporterStore, logger zerolog.Logger) *Importer {
	return &Importer{
		store:  store,
		logger: logger.With().Str("component", "config_importer").Logger(),
	}
}

// ParseConfig parses exported configuration data.
func (i *Importer) ParseConfig(data []byte, format Format) (any, ConfigType, error) {
	// Try to determine the config type from the metadata
	var rawConfig map[string]any

	var err error
	switch format {
	case FormatYAML:
		err = yaml.Unmarshal(data, &rawConfig)
	case FormatJSON:
		err = json.Unmarshal(data, &rawConfig)
	default:
		// Try JSON first, then YAML
		if err = json.Unmarshal(data, &rawConfig); err != nil {
			err = yaml.Unmarshal(data, &rawConfig)
		}
	}

	if err != nil {
		return nil, "", fmt.Errorf("failed to parse config: %w", err)
	}

	// Extract metadata to determine config type
	metadata, ok := rawConfig["metadata"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("config missing metadata")
	}

	configType, ok := metadata["type"].(string)
	if !ok {
		return nil, "", fmt.Errorf("config metadata missing type")
	}

	// Parse into the appropriate type
	switch ConfigType(configType) {
	case ConfigTypeAgent:
		var config AgentConfig
		if err := i.unmarshal(data, &config, format); err != nil {
			return nil, "", fmt.Errorf("failed to parse agent config: %w", err)
		}
		return &config, ConfigTypeAgent, nil

	case ConfigTypeSchedule:
		var config ScheduleConfig
		if err := i.unmarshal(data, &config, format); err != nil {
			return nil, "", fmt.Errorf("failed to parse schedule config: %w", err)
		}
		return &config, ConfigTypeSchedule, nil

	case ConfigTypeRepository:
		var config RepositoryConfig
		if err := i.unmarshal(data, &config, format); err != nil {
			return nil, "", fmt.Errorf("failed to parse repository config: %w", err)
		}
		return &config, ConfigTypeRepository, nil

	case ConfigTypeBundle:
		var config BundleConfig
		if err := i.unmarshal(data, &config, format); err != nil {
			return nil, "", fmt.Errorf("failed to parse bundle config: %w", err)
		}
		return &config, ConfigTypeBundle, nil

	default:
		return nil, "", fmt.Errorf("unknown config type: %s", configType)
	}
}

// ValidateImport validates an import request and returns any conflicts or issues.
func (i *Importer) ValidateImport(ctx context.Context, orgID uuid.UUID, config any, configType ConfigType) (*ValidationResult, error) {
	result := &ValidationResult{Valid: true}

	switch configType {
	case ConfigTypeAgent:
		agentConfig, ok := config.(*AgentConfig)
		if !ok {
			return nil, fmt.Errorf("invalid agent config type")
		}
		i.validateAgentConfig(ctx, orgID, agentConfig, result)

	case ConfigTypeSchedule:
		scheduleConfig, ok := config.(*ScheduleConfig)
		if !ok {
			return nil, fmt.Errorf("invalid schedule config type")
		}
		i.validateScheduleConfig(ctx, orgID, scheduleConfig, result)

	case ConfigTypeRepository:
		repoConfig, ok := config.(*RepositoryConfig)
		if !ok {
			return nil, fmt.Errorf("invalid repository config type")
		}
		i.validateRepositoryConfig(ctx, orgID, repoConfig, result)

	case ConfigTypeBundle:
		bundleConfig, ok := config.(*BundleConfig)
		if !ok {
			return nil, fmt.Errorf("invalid bundle config type")
		}
		i.validateBundleConfig(ctx, orgID, bundleConfig, result)
	}

	return result, nil
}

// Import imports a configuration into the organization.
func (i *Importer) Import(ctx context.Context, orgID uuid.UUID, config any, configType ConfigType, req ImportRequest) (*ImportResult, error) {
	result := &ImportResult{Success: true}

	switch configType {
	case ConfigTypeAgent:
		agentConfig, ok := config.(*AgentConfig)
		if !ok {
			return nil, fmt.Errorf("invalid agent config type")
		}
		if err := i.importAgent(ctx, orgID, agentConfig, req, result); err != nil {
			return nil, err
		}

	case ConfigTypeSchedule:
		scheduleConfig, ok := config.(*ScheduleConfig)
		if !ok {
			return nil, fmt.Errorf("invalid schedule config type")
		}
		if err := i.importSchedule(ctx, orgID, scheduleConfig, req, result); err != nil {
			return nil, err
		}

	case ConfigTypeBundle:
		bundleConfig, ok := config.(*BundleConfig)
		if !ok {
			return nil, fmt.Errorf("invalid bundle config type")
		}
		if err := i.importBundle(ctx, orgID, bundleConfig, req, result); err != nil {
			return nil, err
		}

	case ConfigTypeRepository:
		// Repositories cannot be fully imported without secrets
		result.Warnings = append(result.Warnings, "Repository configurations cannot be imported directly - secrets must be re-entered manually")
		repoConfig, ok := config.(*RepositoryConfig)
		if !ok {
			return nil, fmt.Errorf("invalid repository config type")
		}
		result.Skipped = append(result.Skipped, SkippedItem{
			Type:   ConfigTypeRepository,
			Name:   repoConfig.Name,
			Reason: "Repository configs require manual secret entry - use as template only",
		})
	}

	if len(result.Errors) > 0 {
		result.Success = false
		result.Message = fmt.Sprintf("Import completed with %d error(s)", len(result.Errors))
	} else {
		result.Message = "Import completed successfully"
	}

	return result, nil
}

// validateAgentConfig validates an agent configuration.
func (i *Importer) validateAgentConfig(ctx context.Context, orgID uuid.UUID, config *AgentConfig, result *ValidationResult) {
	if config.Hostname == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "hostname",
			Message: "hostname is required",
		})
		return
	}

	// Check for existing agent with same hostname
	agents, err := i.store.GetAgentsByOrgID(ctx, orgID)
	if err != nil {
		i.logger.Warn().Err(err).Msg("failed to check for existing agents")
		return
	}

	for _, agent := range agents {
		if agent.Hostname == config.Hostname {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:         ConfigTypeAgent,
				Name:         config.Hostname,
				ExistingID:   agent.ID.String(),
				ExistingName: agent.Hostname,
				Message:      "An agent with this hostname already exists",
			})
			break
		}
	}
}

// validateScheduleConfig validates a schedule configuration.
func (i *Importer) validateScheduleConfig(ctx context.Context, orgID uuid.UUID, config *ScheduleConfig, result *ValidationResult) {
	if config.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: "name is required",
		})
	}

	if config.CronExpression == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "cron_expression",
			Message: "cron_expression is required",
		})
	}

	if len(config.Paths) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "paths",
			Message: "at least one path is required",
		})
	}

	// Check if referenced repositories exist
	repos, err := i.store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		i.logger.Warn().Err(err).Msg("failed to check for existing repositories")
		return
	}

	repoMap := make(map[string]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.Name] = repo
	}

	for _, repoRef := range config.Repositories {
		if _, exists := repoMap[repoRef.RepositoryName]; !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"Repository '%s' not found - will need to be mapped during import",
				repoRef.RepositoryName,
			))
		}
	}

	result.Suggestions = append(result.Suggestions, "Ensure a target agent is specified for schedule import")
}

// validateRepositoryConfig validates a repository configuration.
func (i *Importer) validateRepositoryConfig(ctx context.Context, orgID uuid.UUID, config *RepositoryConfig, result *ValidationResult) {
	if config.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: "name is required",
		})
	}

	// Check for existing repository with same name
	repos, err := i.store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		i.logger.Warn().Err(err).Msg("failed to check for existing repositories")
		return
	}

	for _, repo := range repos {
		if repo.Name == config.Name {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:         ConfigTypeRepository,
				Name:         config.Name,
				ExistingID:   repo.ID.String(),
				ExistingName: repo.Name,
				Message:      "A repository with this name already exists",
			})
			break
		}
	}

	result.Warnings = append(result.Warnings, "Repository configuration does not include secrets - these must be provided separately")
}

// validateBundleConfig validates a bundle configuration.
func (i *Importer) validateBundleConfig(ctx context.Context, orgID uuid.UUID, config *BundleConfig, result *ValidationResult) {
	for _, agentConfig := range config.Agents {
		i.validateAgentConfig(ctx, orgID, &agentConfig, result)
	}

	for _, scheduleConfig := range config.Schedules {
		i.validateScheduleConfig(ctx, orgID, &scheduleConfig, result)
	}

	for _, repoConfig := range config.Repositories {
		i.validateRepositoryConfig(ctx, orgID, &repoConfig, result)
	}
}

// importAgent imports an agent configuration.
func (i *Importer) importAgent(ctx context.Context, orgID uuid.UUID, config *AgentConfig, req ImportRequest, result *ImportResult) error {
	// Check for existing agent
	agents, err := i.store.GetAgentsByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to check for existing agents: %w", err)
	}

	var existingAgent *models.Agent
	for _, agent := range agents {
		if agent.Hostname == config.Hostname {
			existingAgent = agent
			break
		}
	}

	if existingAgent != nil {
		switch req.ConflictResolution {
		case ConflictResolutionSkip:
			result.Skipped = append(result.Skipped, SkippedItem{
				Type:   ConfigTypeAgent,
				Name:   config.Hostname,
				Reason: "Agent with same hostname already exists",
			})
			return nil

		case ConflictResolutionFail:
			result.Errors = append(result.Errors, ImportError{
				Type:    ConfigTypeAgent,
				Name:    config.Hostname,
				Message: "Agent with same hostname already exists",
			})
			return nil

		case ConflictResolutionRename:
			config.Hostname = fmt.Sprintf("%s-imported-%d", config.Hostname, time.Now().Unix())
		}
	}

	// Create new agent
	// Note: Agent creation requires an API key hash which is generated during registration
	// For imported agents, we mark them as pending for registration
	agent := &models.Agent{
		ID:        uuid.New(),
		OrgID:     orgID,
		Hostname:  config.Hostname,
		OSInfo:    config.OSInfo,
		Status:    models.AgentStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Convert network mounts if present
	if config.NetworkMounts != nil {
		agent.NetworkMounts = make([]models.NetworkMount, len(config.NetworkMounts))
		for j, mount := range config.NetworkMounts {
			agent.NetworkMounts[j] = models.NetworkMount{
				Path:   mount.Path,
				Type:   models.MountType(mount.MountType),
				Remote: mount.Remote,
			}
		}
	}

	if err := i.store.CreateAgent(ctx, agent); err != nil {
		result.Errors = append(result.Errors, ImportError{
			Type:    ConfigTypeAgent,
			Name:    config.Hostname,
			Message: fmt.Sprintf("Failed to create agent: %s", err.Error()),
		})
		return nil
	}

	result.Imported.AgentCount++
	result.Imported.AgentIDs = append(result.Imported.AgentIDs, agent.ID.String())

	i.logger.Info().
		Str("agent_id", agent.ID.String()).
		Str("hostname", config.Hostname).
		Msg("imported agent configuration")

	return nil
}

// importSchedule imports a schedule configuration.
func (i *Importer) importSchedule(ctx context.Context, orgID uuid.UUID, config *ScheduleConfig, req ImportRequest, result *ImportResult) error {
	// Determine target agent
	var targetAgentID uuid.UUID
	if req.TargetAgentID != "" {
		parsed, err := uuid.Parse(req.TargetAgentID)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    ConfigTypeSchedule,
				Name:    config.Name,
				Message: "Invalid target agent ID",
			})
			return nil
		}

		// Verify agent belongs to org
		agent, err := i.store.GetAgentByID(ctx, parsed)
		if err != nil || agent.OrgID != orgID {
			result.Errors = append(result.Errors, ImportError{
				Type:    ConfigTypeSchedule,
				Name:    config.Name,
				Message: "Target agent not found or does not belong to organization",
			})
			return nil
		}
		targetAgentID = parsed
	} else {
		result.Errors = append(result.Errors, ImportError{
			Type:    ConfigTypeSchedule,
			Name:    config.Name,
			Message: "Target agent ID is required for schedule import",
		})
		return nil
	}

	// Check for existing schedule with same name on this agent
	existingSchedules, err := i.store.GetSchedulesByAgentID(ctx, targetAgentID)
	if err != nil {
		i.logger.Warn().Err(err).Msg("failed to check for existing schedules")
	}

	for _, existing := range existingSchedules {
		if existing.Name == config.Name {
			switch req.ConflictResolution {
			case ConflictResolutionSkip:
				result.Skipped = append(result.Skipped, SkippedItem{
					Type:   ConfigTypeSchedule,
					Name:   config.Name,
					Reason: "Schedule with same name already exists on this agent",
				})
				return nil

			case ConflictResolutionFail:
				result.Errors = append(result.Errors, ImportError{
					Type:    ConfigTypeSchedule,
					Name:    config.Name,
					Message: "Schedule with same name already exists on this agent",
				})
				return nil

			case ConflictResolutionReplace:
				// Delete existing and continue with import
				// Note: We'll update the existing instead
				config.Name = existing.Name // Keep the same name
				existing.CronExpression = config.CronExpression
				existing.Paths = config.Paths
				existing.Excludes = config.Excludes
				existing.RetentionPolicy = config.RetentionPolicy
				existing.BandwidthLimitKB = config.BandwidthLimitKB
				existing.BackupWindow = config.BackupWindow
				existing.ExcludedHours = config.ExcludedHours
				existing.CompressionLevel = config.CompressionLevel
				existing.OnMountUnavailable = models.MountBehavior(config.OnMountUnavailable)
				existing.Enabled = config.Enabled
				existing.UpdatedAt = time.Now()

				if err := i.store.UpdateSchedule(ctx, existing); err != nil {
					result.Errors = append(result.Errors, ImportError{
						Type:    ConfigTypeSchedule,
						Name:    config.Name,
						Message: fmt.Sprintf("Failed to update schedule: %s", err.Error()),
					})
					return nil
				}

				result.Imported.ScheduleCount++
				result.Imported.ScheduleIDs = append(result.Imported.ScheduleIDs, existing.ID.String())
				return nil

			case ConflictResolutionRename:
				config.Name = fmt.Sprintf("%s-imported-%d", config.Name, time.Now().Unix())
			}
			break
		}
	}

	// Map repositories
	repos, err := i.store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	repoMap := make(map[string]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.Name] = repo
	}

	var scheduleRepos []models.ScheduleRepository
	for _, repoRef := range config.Repositories {
		// Check if there's a manual mapping
		var repoID uuid.UUID
		if mappedID, ok := req.RepositoryMappings[repoRef.RepositoryName]; ok {
			parsed, err := uuid.Parse(mappedID)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf(
					"Invalid repository mapping for '%s'",
					repoRef.RepositoryName,
				))
				continue
			}
			repoID = parsed
		} else if repo, exists := repoMap[repoRef.RepositoryName]; exists {
			repoID = repo.ID
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"Repository '%s' not found - skipping this repository association",
				repoRef.RepositoryName,
			))
			continue
		}

		scheduleRepos = append(scheduleRepos, models.ScheduleRepository{
			ID:           uuid.New(),
			RepositoryID: repoID,
			Priority:     repoRef.Priority,
			Enabled:      repoRef.Enabled,
			CreatedAt:    time.Now(),
		})
	}

	if len(scheduleRepos) == 0 && len(config.Repositories) > 0 {
		result.Errors = append(result.Errors, ImportError{
			Type:    ConfigTypeSchedule,
			Name:    config.Name,
			Message: "No valid repositories could be mapped for this schedule",
		})
		return nil
	}

	// Create schedule
	schedule := models.NewSchedule(targetAgentID, config.Name, config.CronExpression, config.Paths)
	schedule.Excludes = config.Excludes
	schedule.RetentionPolicy = config.RetentionPolicy
	schedule.BandwidthLimitKB = config.BandwidthLimitKB
	schedule.BackupWindow = config.BackupWindow
	schedule.ExcludedHours = config.ExcludedHours
	schedule.CompressionLevel = config.CompressionLevel
	schedule.OnMountUnavailable = models.MountBehavior(config.OnMountUnavailable)
	schedule.Enabled = config.Enabled
	schedule.Repositories = scheduleRepos

	if err := i.store.CreateSchedule(ctx, schedule); err != nil {
		result.Errors = append(result.Errors, ImportError{
			Type:    ConfigTypeSchedule,
			Name:    config.Name,
			Message: fmt.Sprintf("Failed to create schedule: %s", err.Error()),
		})
		return nil
	}

	result.Imported.ScheduleCount++
	result.Imported.ScheduleIDs = append(result.Imported.ScheduleIDs, schedule.ID.String())

	i.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("name", config.Name).
		Str("agent_id", targetAgentID.String()).
		Msg("imported schedule configuration")

	return nil
}

// importBundle imports a bundle of configurations.
func (i *Importer) importBundle(ctx context.Context, orgID uuid.UUID, config *BundleConfig, req ImportRequest, result *ImportResult) error {
	// Track created agent IDs for schedule assignment
	agentHostnameToID := make(map[string]uuid.UUID)

	// First, get existing agents for hostname mapping
	existingAgents, _ := i.store.GetAgentsByOrgID(ctx, orgID)
	for _, agent := range existingAgents {
		agentHostnameToID[agent.Hostname] = agent.ID
	}

	// Import agents first
	for _, agentConfig := range config.Agents {
		if err := i.importAgent(ctx, orgID, &agentConfig, req, result); err != nil {
			return err
		}
		// Update mapping if new agent was created
		for _, id := range result.Imported.AgentIDs {
			parsed, _ := uuid.Parse(id)
			agent, err := i.store.GetAgentByID(ctx, parsed)
			if err == nil {
				agentHostnameToID[agent.Hostname] = agent.ID
			}
		}
	}

	// Repository configs are just templates - log them but don't import
	for _, repoConfig := range config.Repositories {
		result.Skipped = append(result.Skipped, SkippedItem{
			Type:   ConfigTypeRepository,
			Name:   repoConfig.Name,
			Reason: "Repository configs require manual secret entry - use as template only",
		})
	}

	// Import schedules - if no target agent specified, try to match by hostname
	for _, scheduleConfig := range config.Schedules {
		scheduleReq := req
		if scheduleReq.TargetAgentID == "" {
			// Try to find an agent from the bundle
			if len(config.Agents) == 1 {
				// If there's only one agent in the bundle, use it
				if agentID, exists := agentHostnameToID[config.Agents[0].Hostname]; exists {
					scheduleReq.TargetAgentID = agentID.String()
				}
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf(
					"No target agent specified for schedule '%s' - please provide target_agent_id",
					scheduleConfig.Name,
				))
				result.Skipped = append(result.Skipped, SkippedItem{
					Type:   ConfigTypeSchedule,
					Name:   scheduleConfig.Name,
					Reason: "No target agent specified",
				})
				continue
			}
		}

		if err := i.importSchedule(ctx, orgID, &scheduleConfig, scheduleReq, result); err != nil {
			return err
		}
	}

	return nil
}

// unmarshal deserializes data based on format.
func (i *Importer) unmarshal(data []byte, v any, format Format) error {
	switch format {
	case FormatYAML:
		return yaml.Unmarshal(data, v)
	case FormatJSON:
		return json.Unmarshal(data, v)
	default:
		// Try JSON first, then YAML
		if err := json.Unmarshal(data, v); err != nil {
			return yaml.Unmarshal(data, v)
		}
		return nil
	}
}
