package migration

import (
	"bytes"
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

// ImporterStore defines the interface for data access needed by the importer.
type ImporterStore interface {
	// Organization operations
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error)
	CreateOrganization(ctx context.Context, org *models.Organization) error

	// User operations
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUsersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error

	// Agent operations
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	CreateAgent(ctx context.Context, agent *models.Agent) error

	// Repository operations
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	CreateRepository(ctx context.Context, repo *models.Repository) error

	// Schedule operations
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) error
	SetScheduleRepositories(ctx context.Context, scheduleID uuid.UUID, repos []models.ScheduleRepository) error

	// Policy operations
	GetPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Policy, error)
	CreatePolicy(ctx context.Context, policy *models.Policy) error

	// Agent group operations
	GetAgentGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentGroup, error)
	CreateAgentGroup(ctx context.Context, group *models.AgentGroup) error
	AddAgentToGroup(ctx context.Context, groupID, agentID uuid.UUID) error
}

// Importer handles full system imports for migration.
type Importer struct {
	store  ImporterStore
	logger zerolog.Logger
}

// NewImporter creates a new migration Importer.
func NewImporter(store ImporterStore, logger zerolog.Logger) *Importer {
	return &Importer{
		store:  store,
		logger: logger.With().Str("component", "migration_importer").Logger(),
	}
}

// encryptedHeader is the prefix for encrypted export files.
const encryptedHeader = "KELDRIS_ENCRYPTED_EXPORT_V1:"

// Parse parses a migration export from JSON data.
func (i *Importer) Parse(data []byte, decryptionKey []byte) (*MigrationExport, error) {
	// Check if data is encrypted
	if bytes.HasPrefix(data, []byte(encryptedHeader)) {
		if len(decryptionKey) == 0 {
			return nil, fmt.Errorf("export is encrypted but no decryption key provided")
		}

		// Decode base64
		encryptedData := data[len(encryptedHeader):]
		decoded, err := base64.StdEncoding.DecodeString(string(encryptedData))
		if err != nil {
			return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
		}

		// Decrypt
		km, err := crypto.NewKeyManager(decryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create key manager: %w", err)
		}

		data, err = km.Decrypt(decoded)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt export: %w", err)
		}
	}

	var export MigrationExport
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, fmt.Errorf("failed to parse export: %w", err)
	}

	return &export, nil
}

// Validate validates a migration export and returns any issues.
func (i *Importer) Validate(ctx context.Context, export *MigrationExport) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid: true,
		Summary: &ImportSummary{
			Organizations: len(export.Organizations),
			Users:         len(export.Users),
			Agents:        len(export.Agents),
			Repositories:  len(export.Repositories),
			Schedules:     len(export.Schedules),
			Policies:      len(export.Policies),
			AgentGroups:   len(export.AgentGroups),
			HasSecrets:    !export.Metadata.SecretsOmitted,
			Encrypted:     export.Metadata.Encrypted,
		},
	}

	// Validate version
	if export.Metadata.Version != MigrationVersion {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"Export version %s differs from current version %s",
			export.Metadata.Version, MigrationVersion,
		))
	}

	// Check for existing organizations
	for _, orgExport := range export.Organizations {
		existing, err := i.store.GetOrganizationBySlug(ctx, orgExport.Slug)
		if err == nil && existing != nil {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:         "organization",
				Name:         orgExport.Slug,
				ExistingID:   existing.ID.String(),
				ExistingName: existing.Name,
				Message:      "Organization with this slug already exists",
			})
		}
	}

	// Check for existing users
	for _, userExport := range export.Users {
		existing, err := i.store.GetUserByEmail(ctx, userExport.Email)
		if err == nil && existing != nil {
			result.Conflicts = append(result.Conflicts, Conflict{
				Type:         "user",
				Name:         userExport.Email,
				ExistingID:   existing.ID.String(),
				ExistingName: existing.Email,
				Message:      "User with this email already exists",
			})
		}
	}

	// Validate organization references
	orgSlugs := make(map[string]bool)
	for _, org := range export.Organizations {
		orgSlugs[org.Slug] = true
	}

	for _, agent := range export.Agents {
		if !orgSlugs[agent.OrgSlug] {
			result.Errors = append(result.Errors, ValidationError{
				Type:    "agent",
				Field:   "org_slug",
				Message: fmt.Sprintf("Agent %s references non-existent organization %s", agent.Hostname, agent.OrgSlug),
			})
			result.Valid = false
		}
	}

	// Add warnings for secrets
	if export.Metadata.SecretsOmitted {
		result.Warnings = append(result.Warnings,
			"Repository secrets were not included in export. You will need to reconfigure repository credentials after import.")
	}

	return result, nil
}

// Import imports a migration export into the system.
func (i *Importer) Import(ctx context.Context, export *MigrationExport, req ImportRequest) (*ImportResult, error) {
	result := &ImportResult{
		Success:    true,
		DryRun:     req.DryRun,
		IDMappings: NewIDMappings(),
	}

	// If dry run, just validate
	if req.DryRun {
		validation, err := i.Validate(ctx, export)
		if err != nil {
			return nil, err
		}

		result.Success = validation.Valid
		if !validation.Valid {
			result.Errors = convertValidationErrors(validation.Errors)
		}
		result.Warnings = validation.Warnings
		result.Message = "Dry run completed - no changes made"
		return result, nil
	}

	// Import organizations first
	orgIDMap := make(map[string]uuid.UUID)
	for _, orgExport := range export.Organizations {
		if req.TargetOrgSlug != "" && orgExport.Slug != req.TargetOrgSlug {
			continue
		}

		orgID, skipped, err := i.importOrganization(ctx, orgExport, req.ConflictResolution)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    "organization",
				Name:    orgExport.Slug,
				Message: err.Error(),
			})
			continue
		}

		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
		} else {
			result.Imported.Organizations++
			result.IDMappings.Organizations[orgExport.ID] = orgID
		}
		orgIDMap[orgExport.Slug] = orgID
	}

	// Import agent groups
	agentGroupIDMap := make(map[string]uuid.UUID)      // original ID -> new ID
	agentGroupNameMap := make(map[string]uuid.UUID)    // "orgSlug:groupName" -> new ID
	for _, agExport := range export.AgentGroups {
		orgID, ok := orgIDMap[agExport.OrgSlug]
		if !ok {
			continue
		}

		groupID, skipped, err := i.importAgentGroup(ctx, agExport, orgID, req.ConflictResolution)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    "agent_group",
				Name:    agExport.Name,
				Message: err.Error(),
			})
			continue
		}

		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
		} else {
			result.Imported.AgentGroups++
			result.IDMappings.AgentGroups[agExport.ID] = groupID
		}
		agentGroupIDMap[agExport.ID] = groupID
		agentGroupNameMap[agExport.OrgSlug+":"+agExport.Name] = groupID
	}

	// Import users
	for _, userExport := range export.Users {
		orgID, ok := orgIDMap[userExport.OrgSlug]
		if !ok {
			continue
		}

		userID, skipped, err := i.importUser(ctx, userExport, orgID, req.ConflictResolution)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    "user",
				Name:    userExport.Email,
				Message: err.Error(),
			})
			continue
		}

		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
		} else {
			result.Imported.Users++
			result.IDMappings.Users[userExport.ID] = userID
		}
	}

	// Import policies
	policyIDMap := make(map[string]uuid.UUID)
	for _, policyExport := range export.Policies {
		orgID, ok := orgIDMap[policyExport.OrgSlug]
		if !ok {
			continue
		}

		policyID, skipped, err := i.importPolicy(ctx, policyExport, orgID, req.ConflictResolution)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    "policy",
				Name:    policyExport.Name,
				Message: err.Error(),
			})
			continue
		}

		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
		} else {
			result.Imported.Policies++
			result.IDMappings.Policies[policyExport.ID] = policyID
		}
		policyIDMap[policyExport.Name] = policyID
	}

	// Import repositories
	repoIDMap := make(map[string]uuid.UUID)
	for _, repoExport := range export.Repositories {
		orgID, ok := orgIDMap[repoExport.OrgSlug]
		if !ok {
			continue
		}

		repoID, skipped, err := i.importRepository(ctx, repoExport, orgID, req.ConflictResolution, req.DecryptionKey)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    "repository",
				Name:    repoExport.Name,
				Message: err.Error(),
			})
			continue
		}

		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
		} else {
			result.Imported.Repositories++
			result.IDMappings.Repositories[repoExport.ID] = repoID

			if repoExport.EncryptedSecrets == "" {
				result.Warnings = append(result.Warnings, fmt.Sprintf(
					"Repository %s imported without secrets - credentials must be reconfigured",
					repoExport.Name,
				))
			}
		}
		repoIDMap[repoExport.Name] = repoID
	}

	// Import agents
	agentIDMap := make(map[string]uuid.UUID)
	for _, agentExport := range export.Agents {
		orgID, ok := orgIDMap[agentExport.OrgSlug]
		if !ok {
			continue
		}

		// Build map of group names to IDs for this agent's org
		groupNameToID := make(map[string]uuid.UUID)
		for _, groupName := range agentExport.GroupNames {
			key := agentExport.OrgSlug + ":" + groupName
			if groupID, ok := agentGroupNameMap[key]; ok {
				groupNameToID[groupName] = groupID
			}
		}

		agentID, skipped, err := i.importAgent(ctx, agentExport, orgID, groupNameToID, req.ConflictResolution)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    "agent",
				Name:    agentExport.Hostname,
				Message: err.Error(),
			})
			continue
		}

		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
		} else {
			result.Imported.Agents++
			result.IDMappings.Agents[agentExport.ID] = agentID
		}
		agentIDMap[agentExport.Hostname] = agentID
	}

	// Import schedules
	for _, scheduleExport := range export.Schedules {
		agentID, ok := agentIDMap[scheduleExport.AgentHostname]
		if !ok {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"Schedule %s skipped - agent %s not found",
				scheduleExport.Name, scheduleExport.AgentHostname,
			))
			continue
		}

		var policyID *uuid.UUID
		if scheduleExport.PolicyName != nil {
			if pid, ok := policyIDMap[*scheduleExport.PolicyName]; ok {
				policyID = &pid
			}
		}

		var groupID *uuid.UUID
		if scheduleExport.AgentGroupID != nil {
			if gid, ok := agentGroupIDMap[*scheduleExport.AgentGroupID]; ok {
				groupID = &gid
			}
		}

		scheduleID, skipped, err := i.importSchedule(ctx, scheduleExport, agentID, policyID, groupID, repoIDMap, req.ConflictResolution)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{
				Type:    "schedule",
				Name:    scheduleExport.Name,
				Message: err.Error(),
			})
			continue
		}

		if skipped != nil {
			result.Skipped = append(result.Skipped, *skipped)
		} else {
			result.Imported.Schedules++
			result.IDMappings.Schedules[scheduleExport.ID] = scheduleID
		}
	}

	// Set result message
	if len(result.Errors) > 0 {
		result.Success = false
		result.Message = fmt.Sprintf("Import completed with %d error(s)", len(result.Errors))
	} else {
		result.Message = "Import completed successfully"
	}

	i.logger.Info().
		Int("organizations", result.Imported.Organizations).
		Int("users", result.Imported.Users).
		Int("agents", result.Imported.Agents).
		Int("repositories", result.Imported.Repositories).
		Int("schedules", result.Imported.Schedules).
		Int("policies", result.Imported.Policies).
		Int("errors", len(result.Errors)).
		Msg("migration import complete")

	return result, nil
}

// importOrganization imports a single organization.
func (i *Importer) importOrganization(ctx context.Context, orgExport OrganizationExport, conflictRes ConflictResolution) (uuid.UUID, *SkippedItem, error) {
	existing, err := i.store.GetOrganizationBySlug(ctx, orgExport.Slug)
	if err == nil && existing != nil {
		switch conflictRes {
		case ConflictResolutionSkip:
			return existing.ID, &SkippedItem{
				Type:   "organization",
				Name:   orgExport.Slug,
				Reason: "Organization with this slug already exists",
			}, nil
		case ConflictResolutionFail:
			return uuid.Nil, nil, fmt.Errorf("organization with slug %s already exists", orgExport.Slug)
		case ConflictResolutionReplace:
			// Use existing org
			return existing.ID, nil, nil
		case ConflictResolutionRename:
			orgExport.Slug = fmt.Sprintf("%s-imported-%d", orgExport.Slug, time.Now().Unix())
			orgExport.Name = fmt.Sprintf("%s (Imported)", orgExport.Name)
		}
	}

	org := models.NewOrganization(orgExport.Name, orgExport.Slug)
	org.MaxConcurrentBackups = orgExport.MaxConcurrentBackups

	if err := i.store.CreateOrganization(ctx, org); err != nil {
		return uuid.Nil, nil, err
	}

	return org.ID, nil, nil
}

// importUser imports a single user.
func (i *Importer) importUser(ctx context.Context, userExport UserExport, orgID uuid.UUID, conflictRes ConflictResolution) (uuid.UUID, *SkippedItem, error) {
	existing, err := i.store.GetUserByEmail(ctx, userExport.Email)
	if err == nil && existing != nil {
		switch conflictRes {
		case ConflictResolutionSkip:
			return existing.ID, &SkippedItem{
				Type:   "user",
				Name:   userExport.Email,
				Reason: "User with this email already exists",
			}, nil
		case ConflictResolutionFail:
			return uuid.Nil, nil, fmt.Errorf("user with email %s already exists", userExport.Email)
		case ConflictResolutionReplace:
			return existing.ID, nil, nil
		case ConflictResolutionRename:
			// Can't rename emails
			return existing.ID, &SkippedItem{
				Type:   "user",
				Name:   userExport.Email,
				Reason: "User with this email already exists (cannot rename)",
			}, nil
		}
	}

	user := models.NewUser(orgID, "", userExport.Email, userExport.Name, models.UserRole(userExport.Role))
	user.Status = models.UserStatus(userExport.Status)
	user.IsSuperuser = userExport.IsSuperuser

	if err := i.store.CreateUser(ctx, user); err != nil {
		return uuid.Nil, nil, err
	}

	return user.ID, nil, nil
}

// importAgent imports a single agent.
func (i *Importer) importAgent(ctx context.Context, agentExport AgentExport, orgID uuid.UUID, groupNameToID map[string]uuid.UUID, conflictRes ConflictResolution) (uuid.UUID, *SkippedItem, error) {
	// Check for existing agent with same hostname in org
	agents, err := i.store.GetAgentsByOrgID(ctx, orgID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to check existing agents: %w", err)
	}

	for _, existing := range agents {
		if existing.Hostname == agentExport.Hostname {
			switch conflictRes {
			case ConflictResolutionSkip:
				return existing.ID, &SkippedItem{
					Type:   "agent",
					Name:   agentExport.Hostname,
					Reason: "Agent with this hostname already exists",
				}, nil
			case ConflictResolutionFail:
				return uuid.Nil, nil, fmt.Errorf("agent with hostname %s already exists", agentExport.Hostname)
			case ConflictResolutionReplace:
				return existing.ID, nil, nil
			case ConflictResolutionRename:
				agentExport.Hostname = fmt.Sprintf("%s-imported-%d", agentExport.Hostname, time.Now().Unix())
			}
		}
	}

	agent := &models.Agent{
		ID:        uuid.New(),
		OrgID:     orgID,
		Hostname:  agentExport.Hostname,
		OSInfo:    agentExport.OSInfo,
		DebugMode: agentExport.DebugMode,
		Status:    models.AgentStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Convert network mounts
	if len(agentExport.NetworkMounts) > 0 {
		agent.NetworkMounts = make([]models.NetworkMount, len(agentExport.NetworkMounts))
		for j, mount := range agentExport.NetworkMounts {
			agent.NetworkMounts[j] = models.NetworkMount{
				Path:   mount.Path,
				Type:   models.MountType(mount.MountType),
				Remote: mount.Remote,
			}
		}
	}

	if err := i.store.CreateAgent(ctx, agent); err != nil {
		return uuid.Nil, nil, err
	}

	// Add agent to groups
	for groupName, groupID := range groupNameToID {
		if err := i.store.AddAgentToGroup(ctx, groupID, agent.ID); err != nil {
			i.logger.Warn().Err(err).
				Str("agent_id", agent.ID.String()).
				Str("group_name", groupName).
				Msg("failed to add agent to group")
		}
	}

	return agent.ID, nil, nil
}

// importRepository imports a single repository.
func (i *Importer) importRepository(ctx context.Context, repoExport RepositoryExport, orgID uuid.UUID, conflictRes ConflictResolution, decryptionKey []byte) (uuid.UUID, *SkippedItem, error) {
	repos, err := i.store.GetRepositoriesByOrgID(ctx, orgID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to check existing repositories: %w", err)
	}

	for _, existing := range repos {
		if existing.Name == repoExport.Name {
			switch conflictRes {
			case ConflictResolutionSkip:
				return existing.ID, &SkippedItem{
					Type:   "repository",
					Name:   repoExport.Name,
					Reason: "Repository with this name already exists",
				}, nil
			case ConflictResolutionFail:
				return uuid.Nil, nil, fmt.Errorf("repository with name %s already exists", repoExport.Name)
			case ConflictResolutionReplace:
				return existing.ID, nil, nil
			case ConflictResolutionRename:
				repoExport.Name = fmt.Sprintf("%s-imported-%d", repoExport.Name, time.Now().Unix())
			}
		}
	}

	repo := &models.Repository{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      repoExport.Name,
		Type:      models.RepositoryType(repoExport.Type),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Decrypt and restore secrets if present
	if repoExport.EncryptedSecrets != "" && len(decryptionKey) > 0 {
		km, err := crypto.NewKeyManager(decryptionKey)
		if err == nil {
			decoded, err := base64.StdEncoding.DecodeString(repoExport.EncryptedSecrets)
			if err == nil {
				decrypted, err := km.Decrypt(decoded)
				if err == nil {
					repo.ConfigEncrypted = decrypted
				}
			}
		}
	}

	if err := i.store.CreateRepository(ctx, repo); err != nil {
		return uuid.Nil, nil, err
	}

	return repo.ID, nil, nil
}

// importPolicy imports a single policy.
func (i *Importer) importPolicy(ctx context.Context, policyExport PolicyExport, orgID uuid.UUID, conflictRes ConflictResolution) (uuid.UUID, *SkippedItem, error) {
	policies, err := i.store.GetPoliciesByOrgID(ctx, orgID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to check existing policies: %w", err)
	}

	for _, existing := range policies {
		if existing.Name == policyExport.Name {
			switch conflictRes {
			case ConflictResolutionSkip:
				return existing.ID, &SkippedItem{
					Type:   "policy",
					Name:   policyExport.Name,
					Reason: "Policy with this name already exists",
				}, nil
			case ConflictResolutionFail:
				return uuid.Nil, nil, fmt.Errorf("policy with name %s already exists", policyExport.Name)
			case ConflictResolutionReplace:
				return existing.ID, nil, nil
			case ConflictResolutionRename:
				policyExport.Name = fmt.Sprintf("%s-imported-%d", policyExport.Name, time.Now().Unix())
			}
		}
	}

	policy := models.NewPolicy(orgID, policyExport.Name)
	policy.Description = policyExport.Description
	policy.Paths = policyExport.Paths
	policy.Excludes = policyExport.Excludes
	policy.RetentionPolicy = policyExport.RetentionPolicy
	policy.BandwidthLimitKB = policyExport.BandwidthLimitKB
	policy.BackupWindow = policyExport.BackupWindow
	policy.ExcludedHours = policyExport.ExcludedHours
	policy.CronExpression = policyExport.CronExpression

	if err := i.store.CreatePolicy(ctx, policy); err != nil {
		return uuid.Nil, nil, err
	}

	return policy.ID, nil, nil
}

// importAgentGroup imports a single agent group.
func (i *Importer) importAgentGroup(ctx context.Context, groupExport AgentGroupExport, orgID uuid.UUID, conflictRes ConflictResolution) (uuid.UUID, *SkippedItem, error) {
	groups, err := i.store.GetAgentGroupsByOrgID(ctx, orgID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to check existing agent groups: %w", err)
	}

	for _, existing := range groups {
		if existing.Name == groupExport.Name {
			switch conflictRes {
			case ConflictResolutionSkip:
				return existing.ID, &SkippedItem{
					Type:   "agent_group",
					Name:   groupExport.Name,
					Reason: "Agent group with this name already exists",
				}, nil
			case ConflictResolutionFail:
				return uuid.Nil, nil, fmt.Errorf("agent group with name %s already exists", groupExport.Name)
			case ConflictResolutionReplace:
				return existing.ID, nil, nil
			case ConflictResolutionRename:
				groupExport.Name = fmt.Sprintf("%s-imported-%d", groupExport.Name, time.Now().Unix())
			}
		}
	}

	group := &models.AgentGroup{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      groupExport.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := i.store.CreateAgentGroup(ctx, group); err != nil {
		return uuid.Nil, nil, err
	}

	return group.ID, nil, nil
}

// importSchedule imports a single schedule.
func (i *Importer) importSchedule(
	ctx context.Context,
	scheduleExport ScheduleExport,
	agentID uuid.UUID,
	policyID *uuid.UUID,
	groupID *uuid.UUID,
	repoIDMap map[string]uuid.UUID,
	conflictRes ConflictResolution,
) (uuid.UUID, *SkippedItem, error) {
	schedules, err := i.store.GetSchedulesByAgentID(ctx, agentID)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("failed to check existing schedules: %w", err)
	}

	for _, existing := range schedules {
		if existing.Name == scheduleExport.Name {
			switch conflictRes {
			case ConflictResolutionSkip:
				return existing.ID, &SkippedItem{
					Type:   "schedule",
					Name:   scheduleExport.Name,
					Reason: "Schedule with this name already exists on this agent",
				}, nil
			case ConflictResolutionFail:
				return uuid.Nil, nil, fmt.Errorf("schedule with name %s already exists on agent", scheduleExport.Name)
			case ConflictResolutionReplace:
				return existing.ID, nil, nil
			case ConflictResolutionRename:
				scheduleExport.Name = fmt.Sprintf("%s-imported-%d", scheduleExport.Name, time.Now().Unix())
			}
		}
	}

	schedule := models.NewSchedule(agentID, scheduleExport.Name, scheduleExport.CronExpression, scheduleExport.Paths)
	schedule.AgentGroupID = groupID
	schedule.PolicyID = policyID
	schedule.BackupType = models.BackupType(scheduleExport.BackupType)
	schedule.Excludes = scheduleExport.Excludes
	schedule.RetentionPolicy = scheduleExport.RetentionPolicy
	schedule.BandwidthLimitKB = scheduleExport.BandwidthLimitKB
	schedule.BackupWindow = scheduleExport.BackupWindow
	schedule.ExcludedHours = scheduleExport.ExcludedHours
	schedule.CompressionLevel = scheduleExport.CompressionLevel
	schedule.OnMountUnavailable = models.MountBehavior(scheduleExport.OnMountUnavailable)
	schedule.Enabled = scheduleExport.Enabled

	// Map repository references
	for _, repoRef := range scheduleExport.Repositories {
		if repoID, ok := repoIDMap[repoRef.RepositoryName]; ok {
			schedule.Repositories = append(schedule.Repositories, models.ScheduleRepository{
				ID:           uuid.New(),
				RepositoryID: repoID,
				Priority:     repoRef.Priority,
				Enabled:      repoRef.Enabled,
				CreatedAt:    time.Now(),
			})
		}
	}

	if err := i.store.CreateSchedule(ctx, schedule); err != nil {
		return uuid.Nil, nil, err
	}

	// Set repository associations if any
	if len(schedule.Repositories) > 0 {
		if err := i.store.SetScheduleRepositories(ctx, schedule.ID, schedule.Repositories); err != nil {
			i.logger.Warn().Err(err).Str("schedule", schedule.Name).Msg("failed to set schedule repositories")
		}
	}

	return schedule.ID, nil, nil
}

// convertValidationErrors converts validation errors to import errors.
func convertValidationErrors(errors []ValidationError) []ImportError {
	result := make([]ImportError, len(errors))
	for i, e := range errors {
		result[i] = ImportError{
			Type:    e.Type,
			Name:    e.Field,
			Message: e.Message,
		}
	}
	return result
}

// GenerateEncryptionKey generates a new encryption key for exports.
func GenerateEncryptionKey() ([]byte, error) {
	return crypto.GenerateMasterKey()
}

// KeyToBase64 encodes an encryption key to base64 for display/storage.
func KeyToBase64(key []byte) string {
	return crypto.MasterKeyToBase64(key)
}

// KeyFromBase64 decodes an encryption key from base64.
func KeyFromBase64(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	return crypto.MasterKeyFromBase64(encoded)
}
