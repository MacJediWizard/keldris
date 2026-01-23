// Package export provides configuration export and import functionality.
package export

import (
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
)

// ConfigType represents the type of configuration being exported.
type ConfigType string

const (
	// ConfigTypeAgent represents an agent configuration export.
	ConfigTypeAgent ConfigType = "agent"
	// ConfigTypeSchedule represents a schedule configuration export.
	ConfigTypeSchedule ConfigType = "schedule"
	// ConfigTypeRepository represents a repository configuration export (without secrets).
	ConfigTypeRepository ConfigType = "repository"
	// ConfigTypeBundle represents a bundle of multiple configurations.
	ConfigTypeBundle ConfigType = "bundle"
)

// Format represents the export format.
type Format string

const (
	// FormatJSON exports configurations as JSON.
	FormatJSON Format = "json"
	// FormatYAML exports configurations as YAML.
	FormatYAML Format = "yaml"
)

// ExportMetadata contains metadata about an exported configuration.
type ExportMetadata struct {
	Version     string     `json:"version" yaml:"version"`
	Type        ConfigType `json:"type" yaml:"type"`
	ExportedAt  time.Time  `json:"exported_at" yaml:"exported_at"`
	ExportedBy  string     `json:"exported_by,omitempty" yaml:"exported_by,omitempty"`
	Description string     `json:"description,omitempty" yaml:"description,omitempty"`
}

// AgentConfig represents an exportable agent configuration.
type AgentConfig struct {
	Metadata      ExportMetadata `json:"metadata" yaml:"metadata"`
	Hostname      string         `json:"hostname" yaml:"hostname"`
	OSInfo        *models.OSInfo `json:"os_info,omitempty" yaml:"os_info,omitempty"`
	NetworkMounts []NetworkMount `json:"network_mounts,omitempty" yaml:"network_mounts,omitempty"`
}

// NetworkMount represents an exportable network mount configuration.
type NetworkMount struct {
	Path      string `json:"path" yaml:"path"`
	MountType string `json:"mount_type" yaml:"mount_type"`
	Remote    string `json:"remote,omitempty" yaml:"remote,omitempty"`
}

// ScheduleConfig represents an exportable schedule configuration.
type ScheduleConfig struct {
	Metadata           ExportMetadata           `json:"metadata" yaml:"metadata"`
	Name               string                   `json:"name" yaml:"name"`
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
// Uses name instead of ID for portability.
type ScheduleRepositoryRef struct {
	RepositoryName string `json:"repository_name" yaml:"repository_name"`
	Priority       int    `json:"priority" yaml:"priority"`
	Enabled        bool   `json:"enabled" yaml:"enabled"`
}

// RepositoryConfig represents an exportable repository configuration.
// Secrets (passwords, API keys) are never exported.
type RepositoryConfig struct {
	Metadata ExportMetadata        `json:"metadata" yaml:"metadata"`
	Name     string                `json:"name" yaml:"name"`
	Type     models.RepositoryType `json:"type" yaml:"type"`
	// Config contains non-sensitive backend configuration.
	// Sensitive values like passwords and API keys are redacted.
	Config map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
}

// BundleConfig represents a bundle of multiple configurations for export.
type BundleConfig struct {
	Metadata     ExportMetadata     `json:"metadata" yaml:"metadata"`
	Agents       []AgentConfig      `json:"agents,omitempty" yaml:"agents,omitempty"`
	Schedules    []ScheduleConfig   `json:"schedules,omitempty" yaml:"schedules,omitempty"`
	Repositories []RepositoryConfig `json:"repositories,omitempty" yaml:"repositories,omitempty"`
}

// ImportRequest represents a request to import a configuration.
type ImportRequest struct {
	// Config is the exported configuration data (JSON or YAML).
	Config []byte `json:"config"`
	// Format specifies the format of the configuration data.
	Format Format `json:"format"`
	// TargetAgentID is the agent to apply schedule configurations to (optional).
	// If not specified, a new agent will be created or matched by hostname.
	TargetAgentID string `json:"target_agent_id,omitempty"`
	// RepositoryMappings maps exported repository names to existing repository IDs.
	// If not specified, repositories will be matched by name or created.
	RepositoryMappings map[string]string `json:"repository_mappings,omitempty"`
	// ConflictResolution specifies how to handle conflicts.
	ConflictResolution ConflictResolution `json:"conflict_resolution,omitempty"`
}

// ConflictResolution specifies how to handle import conflicts.
type ConflictResolution string

const (
	// ConflictResolutionSkip skips importing items that conflict with existing ones.
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
	Success    bool             `json:"success"`
	Message    string           `json:"message"`
	Imported   ImportedItems    `json:"imported"`
	Skipped    []SkippedItem    `json:"skipped,omitempty"`
	Errors     []ImportError    `json:"errors,omitempty"`
	Warnings   []string         `json:"warnings,omitempty"`
}

// ImportedItems contains counts and IDs of successfully imported items.
type ImportedItems struct {
	AgentCount      int      `json:"agent_count"`
	AgentIDs        []string `json:"agent_ids,omitempty"`
	ScheduleCount   int      `json:"schedule_count"`
	ScheduleIDs     []string `json:"schedule_ids,omitempty"`
	RepositoryCount int      `json:"repository_count"`
	RepositoryIDs   []string `json:"repository_ids,omitempty"`
}

// SkippedItem represents an item that was skipped during import.
type SkippedItem struct {
	Type   ConfigType `json:"type"`
	Name   string     `json:"name"`
	Reason string     `json:"reason"`
}

// ImportError represents an error that occurred during import.
type ImportError struct {
	Type    ConfigType `json:"type"`
	Name    string     `json:"name"`
	Message string     `json:"message"`
}

// ValidationResult contains the results of validating an import configuration.
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationError `json:"errors,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
	Conflicts   []Conflict        `json:"conflicts,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
}

// ValidationError represents a validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Conflict represents a conflict detected during import validation.
type Conflict struct {
	Type         ConfigType `json:"type"`
	Name         string     `json:"name"`
	ExistingID   string     `json:"existing_id"`
	ExistingName string     `json:"existing_name"`
	Message      string     `json:"message"`
}

// SensitiveFields contains the list of fields that should be redacted during export.
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

// ExportVersion is the current version of the export format.
const ExportVersion = "1.0"
