package migration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ExporterStore defines the interface for data access needed by the exporter.
type ExporterStore interface {
	// Organization operations
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)

	// User operations
	GetAllUsers(ctx context.Context) ([]*models.User, error)
	GetUsersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.User, error)

	// Agent operations
	GetAllAgents(ctx context.Context) ([]*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)

	// Repository operations
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)

	// Schedule operations
	GetSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)

	// Policy operations
	GetPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Policy, error)
	GetPolicyByID(ctx context.Context, id uuid.UUID) (*models.Policy, error)

	// Agent group operations
	GetAgentGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentGroup, error)
	GetAgentGroupByID(ctx context.Context, id uuid.UUID) (*models.AgentGroup, error)
	GetGroupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.AgentGroup, error)

	// System settings
	GetSystemSettings(ctx context.Context) ([]*models.SystemSetting, error)
}

// Exporter handles full system exports for migration.
type Exporter struct {
	store  ExporterStore
	logger zerolog.Logger
}

// NewExporter creates a new migration Exporter.
func NewExporter(store ExporterStore, logger zerolog.Logger) *Exporter {
	return &Exporter{
		store:  store,
		logger: logger.With().Str("component", "migration_exporter").Logger(),
	}
}

// Export exports the entire system configuration.
func (e *Exporter) Export(ctx context.Context, opts ExportOptions) (*MigrationExport, error) {
	export := &MigrationExport{
		Metadata: MigrationMetadata{
			Version:        MigrationVersion,
			ExportedAt:     time.Now(),
			ExportedBy:     opts.ExportedBy,
			Description:    opts.Description,
			Encrypted:      len(opts.EncryptionKey) > 0,
			SecretsOmitted: !opts.IncludeSecrets,
		},
	}

	// Get all organizations
	orgs, err := e.store.GetAllOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	// Create org slug lookup map
	orgSlugMap := make(map[uuid.UUID]string)
	for _, org := range orgs {
		orgSlugMap[org.ID] = org.Slug
		export.Organizations = append(export.Organizations, OrganizationExport{
			ID:                   org.ID.String(),
			Name:                 org.Name,
			Slug:                 org.Slug,
			MaxConcurrentBackups: org.MaxConcurrentBackups,
		})
	}

	e.logger.Info().Int("count", len(orgs)).Msg("exported organizations")

	// Export users
	users, err := e.store.GetAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	for _, user := range users {
		orgSlug := orgSlugMap[user.OrgID]
		export.Users = append(export.Users, UserExport{
			ID:          user.ID.String(),
			OrgSlug:     orgSlug,
			Email:       user.Email,
			Name:        user.Name,
			Role:        string(user.Role),
			Status:      string(user.Status),
			IsSuperuser: user.IsSuperuser,
		})
	}

	e.logger.Info().Int("count", len(users)).Msg("exported users")

	// Export agents, repositories, schedules, policies, and agent groups per org
	for _, org := range orgs {
		if err := e.exportOrganizationData(ctx, org, orgSlugMap, export, opts); err != nil {
			e.logger.Warn().Err(err).Str("org", org.Slug).Msg("failed to export organization data")
		}
	}

	// Export system config if requested
	if opts.IncludeSystemConfig {
		sysConfig, err := e.exportSystemConfig(ctx)
		if err != nil {
			e.logger.Warn().Err(err).Msg("failed to export system config")
		} else {
			export.SystemConfig = sysConfig
		}
	}

	// Add checksums
	export.Metadata.Checksums = &Checksums{
		Organizations: len(export.Organizations),
		Users:         len(export.Users),
		Agents:        len(export.Agents),
		Repositories:  len(export.Repositories),
		Schedules:     len(export.Schedules),
		Policies:      len(export.Policies),
	}

	e.logger.Info().
		Int("organizations", len(export.Organizations)).
		Int("users", len(export.Users)).
		Int("agents", len(export.Agents)).
		Int("repositories", len(export.Repositories)).
		Int("schedules", len(export.Schedules)).
		Int("policies", len(export.Policies)).
		Msg("migration export complete")

	return export, nil
}

// exportOrganizationData exports all data for a single organization.
func (e *Exporter) exportOrganizationData(
	ctx context.Context,
	org *models.Organization,
	orgSlugMap map[uuid.UUID]string,
	export *MigrationExport,
	opts ExportOptions,
) error {
	// Export agent groups first (needed for agent references)
	agentGroups, err := e.store.GetAgentGroupsByOrgID(ctx, org.ID)
	if err != nil {
		e.logger.Warn().Err(err).Str("org", org.Slug).Msg("failed to get agent groups")
	} else {
		for _, ag := range agentGroups {
			export.AgentGroups = append(export.AgentGroups, AgentGroupExport{
				ID:      ag.ID.String(),
				OrgSlug: org.Slug,
				Name:    ag.Name,
			})
		}
	}

	// Create agent group ID to name map
	agentGroupMap := make(map[uuid.UUID]string)
	for _, ag := range agentGroups {
		agentGroupMap[ag.ID] = ag.ID.String()
	}

	// Export agents
	agents, err := e.store.GetAgentsByOrgID(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("failed to get agents: %w", err)
	}

	agentMap := make(map[uuid.UUID]string) // agentID -> hostname
	for _, agent := range agents {
		agentMap[agent.ID] = agent.Hostname

		agentExport := AgentExport{
			ID:        agent.ID.String(),
			OrgSlug:   org.Slug,
			Hostname:  agent.Hostname,
			OSInfo:    agent.OSInfo,
			DebugMode: agent.DebugMode,
		}

		// Get groups this agent belongs to
		groups, err := e.store.GetGroupsByAgentID(ctx, agent.ID)
		if err != nil {
			e.logger.Warn().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to get agent groups")
		} else {
			for _, group := range groups {
				agentExport.GroupNames = append(agentExport.GroupNames, group.Name)
			}
		}

		if agent.NetworkMounts != nil {
			for _, mount := range agent.NetworkMounts {
				agentExport.NetworkMounts = append(agentExport.NetworkMounts, NetworkMountExport{
					Path:      mount.Path,
					MountType: string(mount.Type),
					Remote:    mount.Remote,
				})
			}
		}

		export.Agents = append(export.Agents, agentExport)
	}

	// Export repositories
	repos, err := e.store.GetRepositoriesByOrgID(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("failed to get repositories: %w", err)
	}

	repoMap := make(map[uuid.UUID]string) // repoID -> name
	for _, repo := range repos {
		repoMap[repo.ID] = repo.Name

		repoExport := RepositoryExport{
			ID:      repo.ID.String(),
			OrgSlug: org.Slug,
			Name:    repo.Name,
			Type:    string(repo.Type),
			Config:  make(map[string]any),
		}

		// Include non-sensitive type info
		repoExport.Config["type"] = string(repo.Type)

		// Include encrypted secrets if requested and encryption key provided
		if opts.IncludeSecrets && len(opts.EncryptionKey) > 0 && len(repo.ConfigEncrypted) > 0 {
			// Re-encrypt with the export encryption key
			km, err := crypto.NewKeyManager(opts.EncryptionKey)
			if err == nil {
				encrypted, err := km.Encrypt(repo.ConfigEncrypted)
				if err == nil {
					repoExport.EncryptedSecrets = base64.StdEncoding.EncodeToString(encrypted)
				}
			}
		}

		export.Repositories = append(export.Repositories, repoExport)
	}

	// Export policies
	policies, err := e.store.GetPoliciesByOrgID(ctx, org.ID)
	if err != nil {
		e.logger.Warn().Err(err).Str("org", org.Slug).Msg("failed to get policies")
	} else {
		for _, policy := range policies {
			export.Policies = append(export.Policies, PolicyExport{
				ID:               policy.ID.String(),
				OrgSlug:          org.Slug,
				Name:             policy.Name,
				Description:      policy.Description,
				Paths:            policy.Paths,
				Excludes:         policy.Excludes,
				RetentionPolicy:  policy.RetentionPolicy,
				BandwidthLimitKB: policy.BandwidthLimitKB,
				BackupWindow:     policy.BackupWindow,
				ExcludedHours:    policy.ExcludedHours,
				CronExpression:   policy.CronExpression,
			})
		}
	}

	// Create policy map for schedule references
	policyMap := make(map[uuid.UUID]string)
	for _, p := range policies {
		policyMap[p.ID] = p.Name
	}

	// Export schedules
	schedules, err := e.store.GetSchedulesByOrgID(ctx, org.ID)
	if err != nil {
		return fmt.Errorf("failed to get schedules: %w", err)
	}

	for _, schedule := range schedules {
		scheduleExport := ScheduleExport{
			ID:                 schedule.ID.String(),
			OrgSlug:            org.Slug,
			AgentHostname:      agentMap[schedule.AgentID],
			Name:               schedule.Name,
			BackupType:         string(schedule.BackupType),
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

		if schedule.AgentGroupID != nil {
			groupID := schedule.AgentGroupID.String()
			scheduleExport.AgentGroupID = &groupID
		}

		if schedule.PolicyID != nil {
			if policyName, ok := policyMap[*schedule.PolicyID]; ok {
				scheduleExport.PolicyName = &policyName
			}
		}

		// Convert repository references
		for _, repoRef := range schedule.Repositories {
			if repoName, ok := repoMap[repoRef.RepositoryID]; ok {
				scheduleExport.Repositories = append(scheduleExport.Repositories, ScheduleRepositoryRef{
					RepositoryName: repoName,
					Priority:       repoRef.Priority,
					Enabled:        repoRef.Enabled,
				})
			}
		}

		export.Schedules = append(export.Schedules, scheduleExport)
	}

	return nil
}

// exportSystemConfig exports system-wide configuration.
func (e *Exporter) exportSystemConfig(ctx context.Context) (*SystemConfigExport, error) {
	settings, err := e.store.GetSystemSettings(ctx)
	if err != nil {
		return nil, err
	}

	config := &SystemConfigExport{}

	for _, setting := range settings {
		var value map[string]any
		if err := json.Unmarshal(setting.Value, &value); err != nil {
			continue
		}

		// Redact sensitive values
		value = redactSensitiveData(value)

		switch {
		case strings.HasPrefix(setting.Key, "smtp_"):
			if config.SMTPSettings == nil {
				config.SMTPSettings = make(map[string]any)
			}
			config.SMTPSettings[setting.Key] = value
		case strings.HasPrefix(setting.Key, "oidc_"):
			if config.OIDCSettings == nil {
				config.OIDCSettings = make(map[string]any)
			}
			config.OIDCSettings[setting.Key] = value
		case strings.HasPrefix(setting.Key, "storage_"):
			if config.StorageDefaults == nil {
				config.StorageDefaults = make(map[string]any)
			}
			config.StorageDefaults[setting.Key] = value
		case strings.HasPrefix(setting.Key, "security_"):
			if config.SecuritySettings == nil {
				config.SecuritySettings = make(map[string]any)
			}
			config.SecuritySettings[setting.Key] = value
		}
	}

	return config, nil
}

// ExportToJSON exports the system and returns JSON bytes.
func (e *Exporter) ExportToJSON(ctx context.Context, opts ExportOptions) ([]byte, error) {
	export, err := e.Export(ctx, opts)
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal export: %w", err)
	}

	// Encrypt if key is provided
	if len(opts.EncryptionKey) > 0 {
		km, err := crypto.NewKeyManager(opts.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create key manager: %w", err)
		}

		encrypted, err := km.Encrypt(data)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt export: %w", err)
		}

		// Return base64-encoded encrypted data with header
		header := []byte("KELDRIS_ENCRYPTED_EXPORT_V1:")
		return append(header, []byte(base64.StdEncoding.EncodeToString(encrypted))...), nil
	}

	return data, nil
}

// redactSensitiveData removes sensitive fields from a configuration map.
func redactSensitiveData(config map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range config {
		if isSensitiveField(k) {
			result[k] = "[REDACTED]"
		} else if nested, ok := v.(map[string]any); ok {
			result[k] = redactSensitiveData(nested)
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
