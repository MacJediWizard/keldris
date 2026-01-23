package export

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// ExporterStore defines the interface for data access needed by the exporter.
type ExporterStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
}

// Exporter handles configuration exports.
type Exporter struct {
	store  ExporterStore
	logger zerolog.Logger
}

// NewExporter creates a new Exporter.
func NewExporter(store ExporterStore, logger zerolog.Logger) *Exporter {
	return &Exporter{
		store:  store,
		logger: logger.With().Str("component", "config_exporter").Logger(),
	}
}

// ExportOptions contains options for exporting configurations.
type ExportOptions struct {
	Format      Format
	Description string
	ExportedBy  string
}

// DefaultExportOptions returns the default export options.
func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		Format: FormatJSON,
	}
}

// ExportAgent exports an agent configuration.
func (e *Exporter) ExportAgent(ctx context.Context, agentID uuid.UUID, opts ExportOptions) ([]byte, error) {
	agent, err := e.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	config := AgentConfig{
		Metadata: ExportMetadata{
			Version:     ExportVersion,
			Type:        ConfigTypeAgent,
			ExportedAt:  time.Now(),
			ExportedBy:  opts.ExportedBy,
			Description: opts.Description,
		},
		Hostname: agent.Hostname,
		OSInfo:   agent.OSInfo,
	}

	// Convert network mounts
	if agent.NetworkMounts != nil {
		config.NetworkMounts = make([]NetworkMount, len(agent.NetworkMounts))
		for i, mount := range agent.NetworkMounts {
			config.NetworkMounts[i] = NetworkMount{
				Path:      mount.Path,
				MountType: string(mount.Type),
				Remote:    mount.Remote,
			}
		}
	}

	e.logger.Info().
		Str("agent_id", agentID.String()).
		Str("hostname", agent.Hostname).
		Str("format", string(opts.Format)).
		Msg("exporting agent configuration")

	return e.marshal(config, opts.Format)
}

// ExportSchedule exports a schedule configuration.
func (e *Exporter) ExportSchedule(ctx context.Context, scheduleID uuid.UUID, opts ExportOptions) ([]byte, error) {
	schedule, err := e.store.GetScheduleByID(ctx, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	config := ScheduleConfig{
		Metadata: ExportMetadata{
			Version:     ExportVersion,
			Type:        ConfigTypeSchedule,
			ExportedAt:  time.Now(),
			ExportedBy:  opts.ExportedBy,
			Description: opts.Description,
		},
		Name:               schedule.Name,
		CronExpression:     schedule.CronExpression,
		Paths:              schedule.Paths,
		Excludes:           schedule.Excludes,
		RetentionPolicy:    schedule.RetentionPolicy,
		BandwidthLimitKB:   schedule.BandwidthLimitKB,
		BackupWindow:       schedule.BackupWindow,
		ExcludedHours:      schedule.ExcludedHours,
		CompressionLevel:   schedule.CompressionLevel,
		OnMountUnavailable: string(schedule.OnMountUnavailable),
		Enabled:            schedule.Enabled,
	}

	// Convert repository references (use names instead of IDs for portability)
	if schedule.Repositories != nil {
		config.Repositories = make([]ScheduleRepositoryRef, len(schedule.Repositories))
		for i, repo := range schedule.Repositories {
			// Get repository name
			repoData, err := e.store.GetRepositoryByID(ctx, repo.RepositoryID)
			if err != nil {
				e.logger.Warn().
					Err(err).
					Str("repository_id", repo.RepositoryID.String()).
					Msg("could not get repository name, using ID as placeholder")
				config.Repositories[i] = ScheduleRepositoryRef{
					RepositoryName: repo.RepositoryID.String(),
					Priority:       repo.Priority,
					Enabled:        repo.Enabled,
				}
			} else {
				config.Repositories[i] = ScheduleRepositoryRef{
					RepositoryName: repoData.Name,
					Priority:       repo.Priority,
					Enabled:        repo.Enabled,
				}
			}
		}
	}

	e.logger.Info().
		Str("schedule_id", scheduleID.String()).
		Str("name", schedule.Name).
		Str("format", string(opts.Format)).
		Msg("exporting schedule configuration")

	return e.marshal(config, opts.Format)
}

// ExportRepository exports a repository configuration (without secrets).
func (e *Exporter) ExportRepository(ctx context.Context, repositoryID uuid.UUID, opts ExportOptions) ([]byte, error) {
	repo, err := e.store.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}

	config := RepositoryConfig{
		Metadata: ExportMetadata{
			Version:     ExportVersion,
			Type:        ConfigTypeRepository,
			ExportedAt:  time.Now(),
			ExportedBy:  opts.ExportedBy,
			Description: opts.Description,
		},
		Name: repo.Name,
		Type: repo.Type,
		// Note: We intentionally do not export the encrypted config
		// as it contains sensitive data
		Config: make(map[string]any),
	}

	// Add only non-sensitive type information
	config.Config["type"] = string(repo.Type)

	e.logger.Info().
		Str("repository_id", repositoryID.String()).
		Str("name", repo.Name).
		Str("type", string(repo.Type)).
		Str("format", string(opts.Format)).
		Msg("exporting repository configuration (without secrets)")

	return e.marshal(config, opts.Format)
}

// ExportBundle exports multiple configurations as a bundle.
func (e *Exporter) ExportBundle(ctx context.Context, agentIDs, scheduleIDs, repositoryIDs []uuid.UUID, opts ExportOptions) ([]byte, error) {
	bundle := BundleConfig{
		Metadata: ExportMetadata{
			Version:     ExportVersion,
			Type:        ConfigTypeBundle,
			ExportedAt:  time.Now(),
			ExportedBy:  opts.ExportedBy,
			Description: opts.Description,
		},
	}

	// Export agents
	for _, agentID := range agentIDs {
		agent, err := e.store.GetAgentByID(ctx, agentID)
		if err != nil {
			e.logger.Warn().Err(err).Str("agent_id", agentID.String()).Msg("skipping agent in bundle export")
			continue
		}

		agentConfig := AgentConfig{
			Metadata: ExportMetadata{
				Version:    ExportVersion,
				Type:       ConfigTypeAgent,
				ExportedAt: time.Now(),
			},
			Hostname: agent.Hostname,
			OSInfo:   agent.OSInfo,
		}

		if agent.NetworkMounts != nil {
			agentConfig.NetworkMounts = make([]NetworkMount, len(agent.NetworkMounts))
			for i, mount := range agent.NetworkMounts {
				agentConfig.NetworkMounts[i] = NetworkMount{
					Path:      mount.Path,
					MountType: string(mount.Type),
					Remote:    mount.Remote,
				}
			}
		}

		bundle.Agents = append(bundle.Agents, agentConfig)
	}

	// Export schedules
	for _, scheduleID := range scheduleIDs {
		schedule, err := e.store.GetScheduleByID(ctx, scheduleID)
		if err != nil {
			e.logger.Warn().Err(err).Str("schedule_id", scheduleID.String()).Msg("skipping schedule in bundle export")
			continue
		}

		scheduleConfig := ScheduleConfig{
			Metadata: ExportMetadata{
				Version:    ExportVersion,
				Type:       ConfigTypeSchedule,
				ExportedAt: time.Now(),
			},
			Name:               schedule.Name,
			CronExpression:     schedule.CronExpression,
			Paths:              schedule.Paths,
			Excludes:           schedule.Excludes,
			RetentionPolicy:    schedule.RetentionPolicy,
			BandwidthLimitKB:   schedule.BandwidthLimitKB,
			BackupWindow:       schedule.BackupWindow,
			ExcludedHours:      schedule.ExcludedHours,
			CompressionLevel:   schedule.CompressionLevel,
			OnMountUnavailable: string(schedule.OnMountUnavailable),
			Enabled:            schedule.Enabled,
		}

		if schedule.Repositories != nil {
			scheduleConfig.Repositories = make([]ScheduleRepositoryRef, len(schedule.Repositories))
			for i, repo := range schedule.Repositories {
				repoData, err := e.store.GetRepositoryByID(ctx, repo.RepositoryID)
				if err != nil {
					scheduleConfig.Repositories[i] = ScheduleRepositoryRef{
						RepositoryName: repo.RepositoryID.String(),
						Priority:       repo.Priority,
						Enabled:        repo.Enabled,
					}
				} else {
					scheduleConfig.Repositories[i] = ScheduleRepositoryRef{
						RepositoryName: repoData.Name,
						Priority:       repo.Priority,
						Enabled:        repo.Enabled,
					}
				}
			}
		}

		bundle.Schedules = append(bundle.Schedules, scheduleConfig)
	}

	// Export repositories
	for _, repositoryID := range repositoryIDs {
		repo, err := e.store.GetRepositoryByID(ctx, repositoryID)
		if err != nil {
			e.logger.Warn().Err(err).Str("repository_id", repositoryID.String()).Msg("skipping repository in bundle export")
			continue
		}

		repoConfig := RepositoryConfig{
			Metadata: ExportMetadata{
				Version:    ExportVersion,
				Type:       ConfigTypeRepository,
				ExportedAt: time.Now(),
			},
			Name:   repo.Name,
			Type:   repo.Type,
			Config: map[string]any{"type": string(repo.Type)},
		}

		bundle.Repositories = append(bundle.Repositories, repoConfig)
	}

	e.logger.Info().
		Int("agent_count", len(bundle.Agents)).
		Int("schedule_count", len(bundle.Schedules)).
		Int("repository_count", len(bundle.Repositories)).
		Str("format", string(opts.Format)).
		Msg("exporting configuration bundle")

	return e.marshal(bundle, opts.Format)
}

// ExportAgentWithSchedules exports an agent along with all its schedules.
func (e *Exporter) ExportAgentWithSchedules(ctx context.Context, agentID uuid.UUID, opts ExportOptions) ([]byte, error) {
	agent, err := e.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	schedules, err := e.store.GetSchedulesByAgentID(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get schedules: %w", err)
	}

	// Build schedule IDs
	scheduleIDs := make([]uuid.UUID, len(schedules))
	for i, s := range schedules {
		scheduleIDs[i] = s.ID
	}

	// Collect repository IDs from schedules
	repoIDMap := make(map[uuid.UUID]bool)
	for _, s := range schedules {
		for _, r := range s.Repositories {
			repoIDMap[r.RepositoryID] = true
		}
	}

	repoIDs := make([]uuid.UUID, 0, len(repoIDMap))
	for id := range repoIDMap {
		repoIDs = append(repoIDs, id)
	}

	e.logger.Info().
		Str("agent_id", agentID.String()).
		Str("hostname", agent.Hostname).
		Int("schedule_count", len(scheduleIDs)).
		Int("repository_count", len(repoIDs)).
		Msg("exporting agent with schedules")

	return e.ExportBundle(ctx, []uuid.UUID{agentID}, scheduleIDs, repoIDs, opts)
}

// marshal converts the config to the specified format.
func (e *Exporter) marshal(v any, format Format) ([]byte, error) {
	switch format {
	case FormatYAML:
		return yaml.Marshal(v)
	case FormatJSON:
		return json.MarshalIndent(v, "", "  ")
	default:
		return json.MarshalIndent(v, "", "  ")
	}
}

// RedactSensitiveData removes sensitive fields from a configuration map.
func RedactSensitiveData(config map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range config {
		if isSensitiveField(k) {
			result[k] = "[REDACTED]"
		} else if nested, ok := v.(map[string]any); ok {
			result[k] = RedactSensitiveData(nested)
		} else {
			result[k] = v
		}
	}
	return result
}

// isSensitiveField checks if a field name indicates sensitive data.
func isSensitiveField(name string) bool {
	nameLower := strings.ToLower(name)
	for _, sensitive := range SensitiveFields {
		if strings.Contains(nameLower, sensitive) {
			return true
		}
	}
	return false
}
