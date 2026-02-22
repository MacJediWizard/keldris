// Package migration provides full system export and import functionality for Keldris.
// This is designed for migrating an entire Keldris installation to a new server.
package migration

import (
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// MigrationVersion is the current version of the migration export format.
const MigrationVersion = "1.0"

// MigrationExport represents a complete system export for migration.
type MigrationExport struct {
	Metadata      MigrationMetadata      `json:"metadata" yaml:"metadata"`
	Organizations []OrganizationExport   `json:"organizations" yaml:"organizations"`
	Users         []UserExport           `json:"users" yaml:"users"`
	Agents        []AgentExport          `json:"agents" yaml:"agents"`
	Repositories  []RepositoryExport     `json:"repositories" yaml:"repositories"`
	Schedules     []ScheduleExport       `json:"schedules" yaml:"schedules"`
	Policies      []PolicyExport         `json:"policies" yaml:"policies"`
	AgentGroups   []AgentGroupExport     `json:"agent_groups,omitempty" yaml:"agent_groups,omitempty"`
	SystemConfig  *SystemConfigExport    `json:"system_config,omitempty" yaml:"system_config,omitempty"`
}

// MigrationMetadata contains metadata about the migration export.
type MigrationMetadata struct {
	Version       string    `json:"version" yaml:"version"`
	ExportedAt    time.Time `json:"exported_at" yaml:"exported_at"`
	ExportedBy    string    `json:"exported_by,omitempty" yaml:"exported_by,omitempty"`
	SourceServer  string    `json:"source_server,omitempty" yaml:"source_server,omitempty"`
	Description   string    `json:"description,omitempty" yaml:"description,omitempty"`
	Encrypted     bool      `json:"encrypted" yaml:"encrypted"`
	SecretsOmitted bool     `json:"secrets_omitted" yaml:"secrets_omitted"`
	Checksums     *Checksums `json:"checksums,omitempty" yaml:"checksums,omitempty"`
}

// Checksums contains integrity checksums for validation.
type Checksums struct {
	Organizations int `json:"organizations" yaml:"organizations"`
	Users         int `json:"users" yaml:"users"`
	Agents        int `json:"agents" yaml:"agents"`
	Repositories  int `json:"repositories" yaml:"repositories"`
	Schedules     int `json:"schedules" yaml:"schedules"`
	Policies      int `json:"policies" yaml:"policies"`
}

// OrganizationExport represents an exportable organization.
type OrganizationExport struct {
	ID                   string `json:"id" yaml:"id"`
	Name                 string `json:"name" yaml:"name"`
	Slug                 string `json:"slug" yaml:"slug"`
	MaxConcurrentBackups *int   `json:"max_concurrent_backups,omitempty" yaml:"max_concurrent_backups,omitempty"`
}

// UserExport represents an exportable user.
// Note: Passwords and sensitive auth data are never exported.
type UserExport struct {
	ID              string `json:"id" yaml:"id"`
	OrgSlug         string `json:"org_slug" yaml:"org_slug"`
	Email           string `json:"email" yaml:"email"`
	Name            string `json:"name,omitempty" yaml:"name,omitempty"`
	Role            string `json:"role" yaml:"role"`
	Status          string `json:"status" yaml:"status"`
	IsSuperuser     bool   `json:"is_superuser" yaml:"is_superuser"`
}

// AgentExport represents an exportable agent configuration.
type AgentExport struct {
	ID            string               `json:"id" yaml:"id"`
	OrgSlug       string               `json:"org_slug" yaml:"org_slug"`
	Hostname      string               `json:"hostname" yaml:"hostname"`
	OSInfo        *models.OSInfo       `json:"os_info,omitempty" yaml:"os_info,omitempty"`
	NetworkMounts []NetworkMountExport `json:"network_mounts,omitempty" yaml:"network_mounts,omitempty"`
	GroupNames    []string             `json:"group_names,omitempty" yaml:"group_names,omitempty"`
	DebugMode     bool                 `json:"debug_mode" yaml:"debug_mode"`
}

// NetworkMountExport represents an exportable network mount.
type NetworkMountExport struct {
	Path      string `json:"path" yaml:"path"`
	MountType string `json:"mount_type" yaml:"mount_type"`
	Remote    string `json:"remote,omitempty" yaml:"remote,omitempty"`
}

// RepositoryExport represents an exportable repository.
// Secrets are optionally included only when encrypted export is used.
type RepositoryExport struct {
	ID               string         `json:"id" yaml:"id"`
	OrgSlug          string         `json:"org_slug" yaml:"org_slug"`
	Name             string         `json:"name" yaml:"name"`
	Type             string         `json:"type" yaml:"type"`
	Config           map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
	EncryptedSecrets string         `json:"encrypted_secrets,omitempty" yaml:"encrypted_secrets,omitempty"`
}

// ScheduleExport represents an exportable schedule.
type ScheduleExport struct {
	ID                 string                   `json:"id" yaml:"id"`
	OrgSlug            string                   `json:"org_slug" yaml:"org_slug"`
	AgentHostname      string                   `json:"agent_hostname" yaml:"agent_hostname"`
	AgentGroupID       *string                  `json:"agent_group_id,omitempty" yaml:"agent_group_id,omitempty"`
	PolicyName         *string                  `json:"policy_name,omitempty" yaml:"policy_name,omitempty"`
	Name               string                   `json:"name" yaml:"name"`
	BackupType         string                   `json:"backup_type,omitempty" yaml:"backup_type,omitempty"`
	CronExpression     string                   `json:"cron_expression" yaml:"cron_expression"`
	Paths              []string                 `json:"paths" yaml:"paths"`
	Excludes           []string                 `json:"excludes,omitempty" yaml:"excludes,omitempty"`
	RetentionPolicy    *models.RetentionPolicy  `json:"retention_policy,omitempty" yaml:"retention_policy,omitempty"`
	BandwidthLimitKB   *int                     `json:"bandwidth_limit_kb,omitempty" yaml:"bandwidth_limit_kb,omitempty"`
	BackupWindow       *models.BackupWindow     `json:"backup_window,omitempty" yaml:"backup_window,omitempty"`
	ExcludedHours      []int                    `json:"excluded_hours,omitempty" yaml:"excluded_hours,omitempty"`
	CompressionLevel   *string                  `json:"compression_level,omitempty" yaml:"compression_level,omitempty"`
	OnMountUnavailable string                   `json:"on_mount_unavailable,omitempty" yaml:"on_mount_unavailable,omitempty"`
	Enabled            bool                     `json:"enabled" yaml:"enabled"`
	Repositories       []ScheduleRepositoryRef  `json:"repositories,omitempty" yaml:"repositories,omitempty"`
}

// ScheduleRepositoryRef is a reference to a repository in a schedule export.
type ScheduleRepositoryRef struct {
	RepositoryName string `json:"repository_name" yaml:"repository_name"`
	Priority       int    `json:"priority" yaml:"priority"`
	Enabled        bool   `json:"enabled" yaml:"enabled"`
}

// PolicyExport represents an exportable backup policy.
type PolicyExport struct {
	ID               string                  `json:"id" yaml:"id"`
	OrgSlug          string                  `json:"org_slug" yaml:"org_slug"`
	Name             string                  `json:"name" yaml:"name"`
	Description      string                  `json:"description,omitempty" yaml:"description,omitempty"`
	Paths            []string                `json:"paths,omitempty" yaml:"paths,omitempty"`
	Excludes         []string                `json:"excludes,omitempty" yaml:"excludes,omitempty"`
	RetentionPolicy  *models.RetentionPolicy `json:"retention_policy,omitempty" yaml:"retention_policy,omitempty"`
	BandwidthLimitKB *int                    `json:"bandwidth_limit_kb,omitempty" yaml:"bandwidth_limit_kb,omitempty"`
	BackupWindow     *models.BackupWindow    `json:"backup_window,omitempty" yaml:"backup_window,omitempty"`
	ExcludedHours    []int                   `json:"excluded_hours,omitempty" yaml:"excluded_hours,omitempty"`
	CronExpression   string                  `json:"cron_expression,omitempty" yaml:"cron_expression,omitempty"`
}

// AgentGroupExport represents an exportable agent group.
type AgentGroupExport struct {
	ID      string `json:"id" yaml:"id"`
	OrgSlug string `json:"org_slug" yaml:"org_slug"`
	Name    string `json:"name" yaml:"name"`
}

// SystemConfigExport represents system-wide configuration.
type SystemConfigExport struct {
	SMTPSettings     map[string]any `json:"smtp_settings,omitempty" yaml:"smtp_settings,omitempty"`
	OIDCSettings     map[string]any `json:"oidc_settings,omitempty" yaml:"oidc_settings,omitempty"`
	StorageDefaults  map[string]any `json:"storage_defaults,omitempty" yaml:"storage_defaults,omitempty"`
	SecuritySettings map[string]any `json:"security_settings,omitempty" yaml:"security_settings,omitempty"`
}

// ExportOptions configures the export operation.
type ExportOptions struct {
	IncludeSecrets   bool   // Include repository secrets (will be encrypted)
	IncludeSystemConfig bool // Include system-wide configuration
	EncryptionKey    []byte // Encryption key for the export file
	Description      string
	ExportedBy       string
}

// DefaultExportOptions returns sensible default export options.
func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		IncludeSecrets:     false,
		IncludeSystemConfig: false,
	}
}

// ImportRequest represents a request to import a migration export.
type ImportRequest struct {
	Data               []byte
	DecryptionKey      []byte // Key to decrypt encrypted exports
	ConflictResolution ConflictResolution
	DryRun             bool   // If true, validate only without importing
	TargetOrgSlug      string // Optional: import into specific org only
}

// ConflictResolution specifies how to handle import conflicts.
type ConflictResolution string

const (
	// ConflictResolutionSkip skips items that conflict with existing ones.
	ConflictResolutionSkip ConflictResolution = "skip"
	// ConflictResolutionReplace replaces existing items with imported ones.
	ConflictResolutionReplace ConflictResolution = "replace"
	// ConflictResolutionRename renames imported items to avoid conflicts.
	ConflictResolutionRename ConflictResolution = "rename"
	// ConflictResolutionFail fails the entire import if any conflicts exist.
	ConflictResolutionFail ConflictResolution = "fail"
)

// ImportResult contains the results of an import operation.
type ImportResult struct {
	Success         bool                `json:"success"`
	DryRun          bool                `json:"dry_run"`
	Message         string              `json:"message"`
	Imported        ImportedCounts      `json:"imported"`
	Skipped         []SkippedItem       `json:"skipped,omitempty"`
	Errors          []ImportError       `json:"errors,omitempty"`
	Warnings        []string            `json:"warnings,omitempty"`
	IDMappings      *IDMappings         `json:"id_mappings,omitempty"`
}

// ImportedCounts contains counts of successfully imported items.
type ImportedCounts struct {
	Organizations int `json:"organizations"`
	Users         int `json:"users"`
	Agents        int `json:"agents"`
	Repositories  int `json:"repositories"`
	Schedules     int `json:"schedules"`
	Policies      int `json:"policies"`
	AgentGroups   int `json:"agent_groups"`
}

// IDMappings tracks the mapping of original IDs to new IDs after import.
type IDMappings struct {
	Organizations map[string]uuid.UUID `json:"organizations"`
	Users         map[string]uuid.UUID `json:"users"`
	Agents        map[string]uuid.UUID `json:"agents"`
	Repositories  map[string]uuid.UUID `json:"repositories"`
	Schedules     map[string]uuid.UUID `json:"schedules"`
	Policies      map[string]uuid.UUID `json:"policies"`
	AgentGroups   map[string]uuid.UUID `json:"agent_groups"`
}

// NewIDMappings creates an initialized IDMappings.
func NewIDMappings() *IDMappings {
	return &IDMappings{
		Organizations: make(map[string]uuid.UUID),
		Users:         make(map[string]uuid.UUID),
		Agents:        make(map[string]uuid.UUID),
		Repositories:  make(map[string]uuid.UUID),
		Schedules:     make(map[string]uuid.UUID),
		Policies:      make(map[string]uuid.UUID),
		AgentGroups:   make(map[string]uuid.UUID),
	}
}

// SkippedItem represents an item that was skipped during import.
type SkippedItem struct {
	Type   string `json:"type"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ImportError represents an error that occurred during import.
type ImportError struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

// ValidationResult contains the results of validating an import.
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationError `json:"errors,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
	Conflicts   []Conflict        `json:"conflicts,omitempty"`
	Summary     *ImportSummary    `json:"summary,omitempty"`
}

// ValidationError represents a validation error.
type ValidationError struct {
	Type    string `json:"type"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Conflict represents a conflict detected during import validation.
type Conflict struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	ExistingID   string `json:"existing_id"`
	ExistingName string `json:"existing_name"`
	Message      string `json:"message"`
}

// ImportSummary provides a summary of what will be imported.
type ImportSummary struct {
	Organizations int `json:"organizations"`
	Users         int `json:"users"`
	Agents        int `json:"agents"`
	Repositories  int `json:"repositories"`
	Schedules     int `json:"schedules"`
	Policies      int `json:"policies"`
	AgentGroups   int `json:"agent_groups"`
	HasSecrets    bool `json:"has_secrets"`
	Encrypted     bool `json:"encrypted"`
}

// SensitiveFields contains the list of fields that should be redacted.
var SensitiveFields = []string{
	"password",
	"api_key",
	"secret",
	"access_key",
	"secret_key",
	"token",
	"key",
	"credential",
	"credentials",
}
