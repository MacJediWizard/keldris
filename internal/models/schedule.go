package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// RetentionPolicy defines how long backups are retained.
type RetentionPolicy struct {
	KeepLast    int `json:"keep_last,omitempty"`
	KeepHourly  int `json:"keep_hourly,omitempty"`
	KeepDaily   int `json:"keep_daily,omitempty"`
	KeepWeekly  int `json:"keep_weekly,omitempty"`
	KeepMonthly int `json:"keep_monthly,omitempty"`
	KeepYearly  int `json:"keep_yearly,omitempty"`
}

// BackupWindow represents a time window during which backups are allowed.
type BackupWindow struct {
	Start string `json:"start,omitempty"` // HH:MM format (e.g., "02:00")
	End   string `json:"end,omitempty"`   // HH:MM format (e.g., "06:00")
}

// SchedulePriority represents the priority level for a backup schedule.
type SchedulePriority int

const (
	// PriorityHigh indicates high priority (1) - runs first, can preempt lower priority.
	PriorityHigh SchedulePriority = 1
	// PriorityMedium indicates medium priority (2) - default priority level.
	PriorityMedium SchedulePriority = 2
	// PriorityLow indicates low priority (3) - runs last, can be preempted.
	PriorityLow SchedulePriority = 3
)

// BackupType represents the type of backup to perform.
type BackupType string

const (
	// BackupTypeFile is a standard file/directory backup.
	BackupTypeFile BackupType = "file"
	// BackupTypeFiles is an alias for file-based backup using Restic.
	BackupTypeFiles BackupType = "files"
	// BackupTypeDocker backs up Docker volumes and container configs.
	BackupTypeDocker BackupType = "docker"
	// BackupTypePihole is a Pi-hole specific backup using teleporter.
	BackupTypePihole BackupType = "pihole"
	// BackupTypeMySQL is a MySQL/MariaDB database backup using mysqldump.
	BackupTypeMySQL BackupType = "mysql"
	// BackupTypePostgres is a PostgreSQL database backup using pg_dump.
	BackupTypePostgres BackupType = "postgres"
	// BackupTypeProxmox backs up Proxmox VMs and containers via vzdump.
	BackupTypeProxmox BackupType = "proxmox"
)

// ValidBackupTypes returns all valid backup types.
func ValidBackupTypes() []BackupType {
	return []BackupType{
		BackupTypeFile,
		BackupTypeFiles,
		BackupTypeDocker,
		BackupTypePihole,
		BackupTypeMySQL,
		BackupTypePostgres,
		BackupTypeProxmox,
	}
}

// IsValidBackupType checks if the backup type is valid.
func IsValidBackupType(t BackupType) bool {
	for _, valid := range ValidBackupTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// DockerBackupOptions contains Docker-specific backup configuration.
type DockerBackupOptions struct {
	// VolumeIDs specifies which Docker volumes to backup. Empty means all volumes.
	VolumeIDs []string `json:"volume_ids,omitempty"`
	// ContainerIDs specifies which container configs to backup. Empty means all containers.
	ContainerIDs []string `json:"container_ids,omitempty"`
	// PauseContainers pauses running containers during volume backup for consistency.
	PauseContainers bool `json:"pause_containers"`
	// IncludeContainerConfigs backs up container configurations as JSON.
	IncludeContainerConfigs bool `json:"include_container_configs"`
}

// PiholeBackupConfig contains Pi-hole specific backup configuration.
type PiholeBackupConfig struct {
	// UseTeleporter uses pihole -a -t for backup (recommended).
	UseTeleporter bool `json:"use_teleporter"`
	// IncludeQueryLogs includes pihole-FTL.db (query logs) in backup.
	IncludeQueryLogs bool `json:"include_query_logs"`
	// ConfigDir overrides the default /etc/pihole directory.
	ConfigDir string `json:"config_dir,omitempty"`
	// DNSMasqDir overrides the default /etc/dnsmasq.d directory.
	DNSMasqDir string `json:"dnsmasq_dir,omitempty"`
}

// MySQLBackupConfig contains MySQL/MariaDB specific backup configuration.
type MySQLBackupConfig struct {
	// DatabaseConnectionID is the ID of the database connection to use.
	DatabaseConnectionID *uuid.UUID `json:"database_connection_id,omitempty"`
	// Database is a specific database to backup. Empty means all databases.
	Database string `json:"database,omitempty"`
	// Databases is a list of specific databases to backup.
	Databases []string `json:"databases,omitempty"`
	// ExcludeDatabases are databases to exclude from "all databases" backup.
	ExcludeDatabases []string `json:"exclude_databases,omitempty"`
	// Compress enables gzip compression of the backup output.
	Compress bool `json:"compress"`
	// ExtraArgs are additional arguments to pass to mysqldump.
	ExtraArgs []string `json:"extra_args,omitempty"`
}

// PostgresOutputFormat represents the output format for pg_dump.
type PostgresOutputFormat string

const (
	// PostgresFormatPlain outputs plain SQL text (default).
	PostgresFormatPlain PostgresOutputFormat = "plain"
	// PostgresFormatCustom outputs custom archive format (recommended for restore flexibility).
	PostgresFormatCustom PostgresOutputFormat = "custom"
	// PostgresFormatDirectory outputs directory format (parallel restore).
	PostgresFormatDirectory PostgresOutputFormat = "directory"
	// PostgresFormatTar outputs tar archive format.
	PostgresFormatTar PostgresOutputFormat = "tar"
)

// PostgresBackupConfig contains PostgreSQL specific backup configuration.
type PostgresBackupConfig struct {
	// Host is the PostgreSQL server hostname or IP address.
	Host string `json:"host"`
	// Port is the PostgreSQL server port (default: 5432).
	Port int `json:"port,omitempty"`
	// Username is the database user for authentication.
	Username string `json:"username"`
	// PasswordEncrypted is the encrypted database password (never exposed in JSON responses).
	PasswordEncrypted string `json:"-"`
	// Database is the specific database to backup. Empty means all databases (pg_dumpall).
	Database string `json:"database,omitempty"`
	// Databases is a list of specific databases to backup (used when multiple but not all).
	Databases []string `json:"databases,omitempty"`
	// OutputFormat specifies the pg_dump output format.
	OutputFormat PostgresOutputFormat `json:"output_format,omitempty"`
	// CompressionLevel is the gzip compression level (0-9, 0=no compression).
	CompressionLevel int `json:"compression_level,omitempty"`
	// IncludeSchemaOnly backs up only schema definitions, not data.
	IncludeSchemaOnly bool `json:"include_schema_only,omitempty"`
	// IncludeDataOnly backs up only data, not schema definitions.
	IncludeDataOnly bool `json:"include_data_only,omitempty"`
	// ExcludeTables is a list of table patterns to exclude from backup.
	ExcludeTables []string `json:"exclude_tables,omitempty"`
	// IncludeTables is a list of specific tables to include (if set, only these tables are backed up).
	IncludeTables []string `json:"include_tables,omitempty"`
	// NoOwner omits owner information from the dump.
	NoOwner bool `json:"no_owner,omitempty"`
	// NoPrivileges omits privilege information from the dump.
	NoPrivileges bool `json:"no_privileges,omitempty"`
	// SSLMode specifies the SSL connection mode (disable, allow, prefer, require, verify-ca, verify-full).
	SSLMode string `json:"ssl_mode,omitempty"`
	// PgDumpPath overrides the default pg_dump binary path.
	PgDumpPath string `json:"pg_dump_path,omitempty"`
}

// ProxmoxBackupOptions contains Proxmox-specific backup configuration.
type ProxmoxBackupOptions struct {
	// ConnectionID is the ID of the Proxmox connection to use.
	ConnectionID string `json:"connection_id,omitempty"`
	// VMIDs specifies which VMs to backup (empty means all).
	VMIDs []int `json:"vm_ids,omitempty"`
	// ContainerIDs specifies which LXC containers to backup (empty means all).
	ContainerIDs []int `json:"container_ids,omitempty"`
	// Mode is the backup mode: snapshot, suspend, or stop.
	Mode string `json:"mode"`
	// Compress is the compression algorithm: 0 (none), gzip, lzo, or zstd.
	Compress string `json:"compress"`
	// Storage is the Proxmox storage for temporary backup files.
	Storage string `json:"storage,omitempty"`
	// MaxWait is the maximum wait time in minutes for backup task completion.
	MaxWait int `json:"max_wait,omitempty"`
	// IncludeRAM includes RAM state in VM backups (requires snapshot mode).
	IncludeRAM bool `json:"include_ram"`
	// RemoveAfter removes the backup from Proxmox after storing in Restic.
	RemoveAfter bool `json:"remove_after"`
}

// Schedule represents a backup schedule configuration.
// A schedule can be assigned to either an individual agent (via AgentID)
// or to an agent group (via AgentGroupID). When AgentGroupID is set,
// the schedule applies to all agents in that group.
type Schedule struct {
	ID                 uuid.UUID            `json:"id"`
	AgentID            uuid.UUID            `json:"agent_id"`
	AgentGroupID       *uuid.UUID           `json:"agent_group_id,omitempty"` // If set, applies to all agents in the group
	PolicyID           *uuid.UUID           `json:"policy_id,omitempty"`      // Policy this schedule was created from
	Name               string               `json:"name"`
	CronExpression     string               `json:"cron_expression"`
	Paths              []string             `json:"paths"`
	Excludes           []string             `json:"excludes,omitempty"`
	RetentionPolicy    *RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB   *int                 `json:"bandwidth_limit_kb,omitempty"`   // Upload limit in KB/s
	BackupWindow       *BackupWindow        `json:"backup_window,omitempty"`        // Allowed backup time window
	ExcludedHours      []int                `json:"excluded_hours,omitempty"`       // Hours (0-23) when backups should not run
	CompressionLevel   *string              `json:"compression_level,omitempty"`    // Compression level: off, auto, max
	OnMountUnavailable   MountBehavior        `json:"on_mount_unavailable,omitempty"` // Behavior when network mount unavailable
	DockerVolumes        []string             `json:"docker_volumes,omitempty"`        // Docker volume names to back up
	DockerPauseContainers bool                `json:"docker_pause_containers,omitempty"` // Pause containers during volume backup
	MaxFileSizeMB           *int                 `json:"max_file_size_mb,omitempty"`     // Max file size in MB (0 = disabled)
	ClassificationLevel     string               `json:"classification_level,omitempty"` // Data classification level: public, internal, confidential, restricted
	ClassificationDataTypes []string             `json:"classification_data_types,omitempty"` // Data types: pii, phi, pci, proprietary, general
	Priority                SchedulePriority     `json:"priority"`                       // Backup priority: 1=high, 2=medium, 3=low
	Preemptible             bool                 `json:"preemptible"`                    // Can be preempted by higher priority backups
	Enabled                 bool                 `json:"enabled"`
	Repositories         []ScheduleRepository `json:"repositories,omitempty"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
	BackupType              BackupType             `json:"backup_type"`              // Type of backup: file, docker
	ID                      uuid.UUID              `json:"id"`
	AgentID                 uuid.UUID              `json:"agent_id"`
	AgentGroupID            *uuid.UUID             `json:"agent_group_id,omitempty"` // If set, applies to all agents in the group
	PolicyID                *uuid.UUID             `json:"policy_id,omitempty"`      // Policy this schedule was created from
	Name                    string                 `json:"name"`
	BackupType              BackupType             `json:"backup_type"`              // Type of backup: file, docker, pihole, mysql
	CronExpression          string                 `json:"cron_expression"`
	Paths                   []string               `json:"paths"`
	Excludes                []string               `json:"excludes,omitempty"`
	RetentionPolicy         *RetentionPolicy       `json:"retention_policy,omitempty"`
	BandwidthLimitKB        *int                   `json:"bandwidth_limit_kb,omitempty"`   // Upload limit in KB/s
	BackupWindow            *BackupWindow          `json:"backup_window,omitempty"`        // Allowed backup time window
	ExcludedHours           []int                  `json:"excluded_hours,omitempty"`       // Hours (0-23) when backups should not run
	CompressionLevel        *string                `json:"compression_level,omitempty"`    // Compression level: off, auto, max
	MaxFileSizeMB           *int                   `json:"max_file_size_mb,omitempty"`     // Max file size in MB (0 = disabled)
	OnMountUnavailable      MountBehavior          `json:"on_mount_unavailable,omitempty"` // Behavior when network mount unavailable
	ClassificationLevel     string                 `json:"classification_level,omitempty"` // Data classification level: public, internal, confidential, restricted
	ClassificationDataTypes []string               `json:"classification_data_types,omitempty"` // Data types: pii, phi, pci, proprietary, general
	Priority                SchedulePriority       `json:"priority"`                       // Backup priority: 1=high, 2=medium, 3=low
	Preemptible             bool                   `json:"preemptible"`                    // Can be preempted by higher priority backups
	DockerOptions           *DockerBackupOptions   `json:"docker_options,omitempty"`       // Docker-specific backup options
	PiholeConfig            *PiholeBackupConfig    `json:"pihole_config,omitempty"`        // Pi-hole specific backup configuration
	MySQLConfig             *MySQLBackupConfig     `json:"mysql_config,omitempty"`         // MySQL/MariaDB specific backup configuration
	PostgresConfig          *PostgresBackupConfig  `json:"postgres_config,omitempty"`      // PostgreSQL specific backup configuration
	ProxmoxOptions          *ProxmoxBackupOptions  `json:"proxmox_options,omitempty"`      // Proxmox-specific backup options
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
	ID               uuid.UUID            `json:"id"`
	AgentID          uuid.UUID            `json:"agent_id"`
	AgentGroupID     *uuid.UUID           `json:"agent_group_id,omitempty"` // If set, applies to all agents in the group
	PolicyID         *uuid.UUID           `json:"policy_id,omitempty"`      // Policy this schedule was created from
	PolicyID         *uuid.UUID           `json:"policy_id,omitempty"` // Policy this schedule was created from
	ID               uuid.UUID            `json:"id"`
	AgentID          uuid.UUID            `json:"agent_id"`
	Name             string               `json:"name"`
	CronExpression   string               `json:"cron_expression"`
	Paths            []string             `json:"paths"`
	Excludes         []string             `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int                 `json:"bandwidth_limit_kb,omitempty"` // Upload limit in KB/s
	BackupWindow     *BackupWindow        `json:"backup_window,omitempty"`      // Allowed backup time window
	ExcludedHours    []int                `json:"excluded_hours,omitempty"`     // Hours (0-23) when backups should not run
	CompressionLevel *string              `json:"compression_level,omitempty"`  // Compression level: off, auto, max
	Enabled          bool                 `json:"enabled"`
	Repositories     []ScheduleRepository `json:"repositories,omitempty"`
	CreatedAt        time.Time            `json:"created_at"`
	UpdatedAt        time.Time            `json:"updated_at"`
	ID               uuid.UUID        `json:"id"`
	AgentID          uuid.UUID        `json:"agent_id"`
	RepositoryID     uuid.UUID        `json:"repository_id"`
	Name             string           `json:"name"`
	CronExpression   string           `json:"cron_expression"`
	Paths            []string         `json:"paths"`
	Excludes         []string         `json:"excludes,omitempty"`
	RetentionPolicy  *RetentionPolicy `json:"retention_policy,omitempty"`
	BandwidthLimitKB *int             `json:"bandwidth_limit_kb,omitempty"` // Upload limit in KB/s
	BackupWindow     *BackupWindow    `json:"backup_window,omitempty"`      // Allowed backup time window
	ExcludedHours    []int            `json:"excluded_hours,omitempty"`     // Hours (0-23) when backups should not run
	Enabled          bool             `json:"enabled"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	ID                      uuid.UUID            `json:"id"`
	AgentID                 uuid.UUID            `json:"agent_id"`
	AgentGroupID            *uuid.UUID           `json:"agent_group_id,omitempty"` // If set, applies to all agents in the group
	PolicyID                *uuid.UUID           `json:"policy_id,omitempty"`      // Policy this schedule was created from
	Name                    string               `json:"name"`
	CronExpression          string               `json:"cron_expression"`
	Paths                   []string             `json:"paths"`
	Excludes                []string             `json:"excludes,omitempty"`
	RetentionPolicy         *RetentionPolicy     `json:"retention_policy,omitempty"`
	BandwidthLimitKB        *int                 `json:"bandwidth_limit_kb,omitempty"`   // Upload limit in KB/s
	BackupWindow            *BackupWindow        `json:"backup_window,omitempty"`        // Allowed backup time window
	ExcludedHours           []int                `json:"excluded_hours,omitempty"`       // Hours (0-23) when backups should not run
	CompressionLevel        *string              `json:"compression_level,omitempty"`    // Compression level: off, auto, max
	MaxFileSizeMB           *int                 `json:"max_file_size_mb,omitempty"`     // Max file size in MB (0 = disabled)
	OnMountUnavailable      MountBehavior        `json:"on_mount_unavailable,omitempty"` // Behavior when network mount unavailable
	ClassificationLevel     string               `json:"classification_level,omitempty"` // Data classification level: public, internal, confidential, restricted
	ClassificationDataTypes []string             `json:"classification_data_types,omitempty"` // Data types: pii, phi, pci, proprietary, general
	Enabled                 bool                 `json:"enabled"`
	Repositories            []ScheduleRepository `json:"repositories,omitempty"`
	CreatedAt               time.Time            `json:"created_at"`
	UpdatedAt               time.Time            `json:"updated_at"`
}

// NewSchedule creates a new Schedule with the given details.
func NewSchedule(agentID uuid.UUID, name, cronExpr string, paths []string) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		BackupType:         BackupTypeFile, // Default to file backup
		CronExpression:     cronExpr,
		Paths:              paths,
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium, // Default to medium priority
		Preemptible:        false,          // Not preemptible by default
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// NewDockerSchedule creates a new Schedule for Docker backup.
func NewDockerSchedule(agentID uuid.UUID, name, cronExpr string, opts *DockerBackupOptions) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		BackupType:         BackupTypeDocker,
		CronExpression:     cronExpr,
		Paths:              []string{}, // Docker backups don't use paths
		DockerOptions:      opts,
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium,
		Preemptible:        false,
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// NewPiholeSchedule creates a new Schedule for Pi-hole backups.
func NewPiholeSchedule(agentID uuid.UUID, name, cronExpr string) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		CronExpression:     cronExpr,
		BackupType:         BackupTypePihole,
		Paths:              []string{"/etc/pihole", "/etc/dnsmasq.d"},
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium,
		Preemptible:        false,
		PiholeConfig: &PiholeBackupConfig{
			UseTeleporter:    true,
			IncludeQueryLogs: true,
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewPostgresSchedule creates a new Schedule for PostgreSQL backups.
func NewPostgresSchedule(agentID uuid.UUID, name, cronExpr string, config *PostgresBackupConfig) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		CronExpression:     cronExpr,
		BackupType:         BackupTypePostgres,
		Paths:              []string{}, // PostgreSQL backups don't use filesystem paths
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium,
		Preemptible:        false,
		PostgresConfig:     config,
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// NewProxmoxSchedule creates a new Schedule for Proxmox VM/container backups.
func NewProxmoxSchedule(agentID uuid.UUID, name, cronExpr string, opts *ProxmoxBackupOptions) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		CronExpression:     cronExpr,
		BackupType:         BackupTypeProxmox,
		Paths:              []string{}, // Proxmox backups don't use paths
		ProxmoxOptions:     opts,
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium,
		Preemptible:        false,
		Enabled:            true,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// IsDockerBackup returns true if this is a Docker backup schedule.
func (s *Schedule) IsDockerBackup() bool {
	return s.BackupType == BackupTypeDocker
}

// IsPiholeBackup returns true if this is a Pi-hole backup schedule.
func (s *Schedule) IsPiholeBackup() bool {
	return s.BackupType == BackupTypePihole
}

// IsPostgresBackup returns true if this is a PostgreSQL backup schedule.
func (s *Schedule) IsPostgresBackup() bool {
	return s.BackupType == BackupTypePostgres
}

// IsProxmoxBackup returns true if this is a Proxmox backup schedule.
func (s *Schedule) IsProxmoxBackup() bool {
	return s.BackupType == BackupTypeProxmox
}

// SetDockerOptions sets the Docker backup options from JSON bytes.
func (s *Schedule) SetDockerOptions(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var opts DockerBackupOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return err
	}
	s.DockerOptions = &opts
	return nil
}

// DockerOptionsJSON returns the Docker options as JSON bytes for database storage.
func (s *Schedule) DockerOptionsJSON() ([]byte, error) {
	if s.DockerOptions == nil {
		return nil, nil
	}
	return json.Marshal(s.DockerOptions)
}

// PriorityLabel returns a human-readable label for the priority.
func (s *Schedule) PriorityLabel() string {
	switch s.Priority {
	case PriorityHigh:
		return "high"
	case PriorityMedium:
		return "medium"
	case PriorityLow:
		return "low"
	default:
		return "medium"
	}
}

// IsHigherPriorityThan returns true if this schedule has higher priority than other.
func (s *Schedule) IsHigherPriorityThan(other *Schedule) bool {
	return s.Priority < other.Priority
}

	now := time.Now()
	return &Schedule{
		ID:             uuid.New(),
		AgentID:        agentID,
		Name:           name,
		CronExpression: cronExpr,
		Paths:          paths,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// GetPrimaryRepository returns the primary repository (priority 0), or nil if none.
func (s *Schedule) GetPrimaryRepository() *ScheduleRepository {
	for i := range s.Repositories {
		if s.Repositories[i].Priority == 0 && s.Repositories[i].Enabled {
			return &s.Repositories[i]
		}
	}
	return nil
}

// GetEnabledRepositories returns all enabled repositories sorted by priority.
func (s *Schedule) GetEnabledRepositories() []ScheduleRepository {
	var enabled []ScheduleRepository
	for _, r := range s.Repositories {
		if r.Enabled {
			enabled = append(enabled, r)
		}
	}
	// Sort by priority (already sorted in DB query, but ensure)
	for i := 0; i < len(enabled)-1; i++ {
		for j := i + 1; j < len(enabled); j++ {
			if enabled[i].Priority > enabled[j].Priority {
				enabled[i], enabled[j] = enabled[j], enabled[i]
			}
		}
	}
	return enabled
}

// SetPaths sets the paths from JSON bytes.
func (s *Schedule) SetPaths(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Paths)
}

// PathsJSON returns the paths as JSON bytes for database storage.
func (s *Schedule) PathsJSON() ([]byte, error) {
	if s.Paths == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(s.Paths)
}

// SetExcludes sets the excludes from JSON bytes.
func (s *Schedule) SetExcludes(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Excludes)
}

// ExcludesJSON returns the excludes as JSON bytes for database storage.
func (s *Schedule) ExcludesJSON() ([]byte, error) {
	if s.Excludes == nil {
		return nil, nil
	}
	return json.Marshal(s.Excludes)
}

// SetDockerVolumes sets the Docker volumes from JSON bytes.
func (s *Schedule) SetDockerVolumes(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.DockerVolumes)
}

// DockerVolumesJSON returns the Docker volumes as JSON bytes for database storage.
func (s *Schedule) DockerVolumesJSON() ([]byte, error) {
	if s.DockerVolumes == nil {
		return nil, nil
	}
	return json.Marshal(s.DockerVolumes)
}

// SetRetentionPolicy sets the retention policy from JSON bytes.
func (s *Schedule) SetRetentionPolicy(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var policy RetentionPolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return err
	}
	s.RetentionPolicy = &policy
	return nil
}

// RetentionPolicyJSON returns the retention policy as JSON bytes for database storage.
func (s *Schedule) RetentionPolicyJSON() ([]byte, error) {
	if s.RetentionPolicy == nil {
		return nil, nil
	}
	return json.Marshal(s.RetentionPolicy)
}

// DefaultRetentionPolicy returns a sensible default retention policy.
func DefaultRetentionPolicy() *RetentionPolicy {
	return &RetentionPolicy{
		KeepLast:    5,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 6,
	}
}

// SetBackupWindow sets the backup window from JSON bytes.
func (s *Schedule) SetBackupWindow(startTime, endTime *string) {
	if startTime == nil && endTime == nil {
		s.BackupWindow = nil
		return
	}
	s.BackupWindow = &BackupWindow{}
	if startTime != nil {
		s.BackupWindow.Start = *startTime
	}
	if endTime != nil {
		s.BackupWindow.End = *endTime
	}
}

// SetExcludedHours sets the excluded hours from JSON bytes.
func (s *Schedule) SetExcludedHours(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.ExcludedHours)
}

// ExcludedHoursJSON returns the excluded hours as JSON bytes for database storage.
func (s *Schedule) ExcludedHoursJSON() ([]byte, error) {
	if s.ExcludedHours == nil {
		return nil, nil
	}
	return json.Marshal(s.ExcludedHours)
}

// IsWithinBackupWindow checks if the given time is within the allowed backup window.
// Returns true if no window is set (always allowed) or if the time is within the window.
func (s *Schedule) IsWithinBackupWindow(t time.Time) bool {
	if s.BackupWindow == nil || (s.BackupWindow.Start == "" && s.BackupWindow.End == "") {
		return true
	}

	currentTime := t.Format("15:04")
	start := s.BackupWindow.Start
	end := s.BackupWindow.End

	// Handle window that doesn't cross midnight (e.g., 02:00 to 06:00)
	if start <= end {
		return currentTime >= start && currentTime < end
	}

	// Handle window that crosses midnight (e.g., 22:00 to 06:00)
	return currentTime >= start || currentTime < end
}

// IsHourExcluded checks if the given hour is in the excluded hours list.
func (s *Schedule) IsHourExcluded(hour int) bool {
	for _, h := range s.ExcludedHours {
		if h == hour {
			return true
		}
	}
	return false
}

// CanRunAt checks if a backup can run at the given time based on window and excluded hours.
func (s *Schedule) CanRunAt(t time.Time) bool {
	if !s.IsWithinBackupWindow(t) {
		return false
	}
	if s.IsHourExcluded(t.Hour()) {
		return false
	}
	return true
}

// NextAllowedTime finds the next time when a backup can run, starting from the given time.
// Returns the input time if it's already allowed.
func (s *Schedule) NextAllowedTime(t time.Time) time.Time {
	// Check up to 24 hours ahead
	for i := 0; i < 24*60; i++ {
		checkTime := t.Add(time.Duration(i) * time.Minute)
		if s.CanRunAt(checkTime) {
			return checkTime
		}
	}
	// Fallback: return original time if no valid window found
	return t
}

// SetClassificationDataTypes sets the classification data types from JSON bytes.
func (s *Schedule) SetClassificationDataTypes(data []byte) error {
	if len(data) == 0 {
		s.ClassificationDataTypes = []string{"general"}
		return nil
	}
	return json.Unmarshal(data, &s.ClassificationDataTypes)
}

// ClassificationDataTypesJSON returns the classification data types as JSON bytes.
func (s *Schedule) ClassificationDataTypesJSON() ([]byte, error) {
	if len(s.ClassificationDataTypes) == 0 {
		return []byte(`["general"]`), nil
	}
	return json.Marshal(s.ClassificationDataTypes)
}

// SetMetadata sets the metadata from JSON bytes.
func (s *Schedule) SetMetadata(data []byte) error {
	if len(data) == 0 {
		s.Metadata = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(data, &s.Metadata)
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (s *Schedule) MetadataJSON() ([]byte, error) {
	if s.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(s.Metadata)
}

// SetPiholeConfig sets the Pi-hole config from JSON bytes.
func (s *Schedule) SetPiholeConfig(data []byte) error {
	if len(data) == 0 {
		s.PiholeConfig = nil
		return nil
	}
	var config PiholeBackupConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	s.PiholeConfig = &config
	return nil
}

// PiholeConfigJSON returns the Pi-hole config as JSON bytes for database storage.
func (s *Schedule) PiholeConfigJSON() ([]byte, error) {
	if s.PiholeConfig == nil {
		return nil, nil
	}
	return json.Marshal(s.PiholeConfig)
}

// DefaultPiholeConfig returns a sensible default Pi-hole backup configuration.
func DefaultPiholeConfig() *PiholeBackupConfig {
	return &PiholeBackupConfig{
		UseTeleporter:    true,
		IncludeQueryLogs: true,
		ConfigDir:        "/etc/pihole",
		DNSMasqDir:       "/etc/dnsmasq.d",
	}
}

// SetMySQLConfig sets the MySQL config from JSON bytes.
func (s *Schedule) SetMySQLConfig(data []byte) error {
	if len(data) == 0 {
		s.MySQLConfig = nil
		return nil
	}
	var config MySQLBackupConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	s.MySQLConfig = &config
	return nil
}

// MySQLConfigJSON returns the MySQL config as JSON bytes for database storage.
func (s *Schedule) MySQLConfigJSON() ([]byte, error) {
	if s.MySQLConfig == nil {
		return nil, nil
	}
	return json.Marshal(s.MySQLConfig)
}

// DefaultMySQLConfig returns a sensible default MySQL backup configuration.
func DefaultMySQLConfig() *MySQLBackupConfig {
	return &MySQLBackupConfig{
		Compress:         true,
		ExcludeDatabases: []string{"information_schema", "performance_schema", "sys"},
	}
}

// IsMySQLBackup returns true if this is a MySQL backup schedule.
func (s *Schedule) IsMySQLBackup() bool {
	return s.BackupType == BackupTypeMySQL
}

// NewMySQLSchedule creates a new Schedule for MySQL backups.
func NewMySQLSchedule(agentID uuid.UUID, name, cronExpr string, databaseConnectionID *uuid.UUID) *Schedule {
	now := time.Now()
	return &Schedule{
		ID:                 uuid.New(),
		AgentID:            agentID,
		Name:               name,
		CronExpression:     cronExpr,
		BackupType:         BackupTypeMySQL,
		Paths:              []string{},
		OnMountUnavailable: MountBehaviorFail,
		Priority:           PriorityMedium,
		Preemptible:        false,
		MySQLConfig: &MySQLBackupConfig{
			DatabaseConnectionID: databaseConnectionID,
			Compress:             true,
			ExcludeDatabases:     []string{"information_schema", "performance_schema", "sys"},
		},
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// SetPostgresConfig sets the PostgreSQL config from JSON bytes.
func (s *Schedule) SetPostgresConfig(data []byte) error {
	if len(data) == 0 {
		s.PostgresConfig = nil
		return nil
	}
	var config PostgresBackupConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	s.PostgresConfig = &config
	return nil
}

// PostgresConfigJSON returns the PostgreSQL config as JSON bytes for database storage.
func (s *Schedule) PostgresConfigJSON() ([]byte, error) {
	if s.PostgresConfig == nil {
		return nil, nil
	}
	return json.Marshal(s.PostgresConfig)
}

// DefaultPostgresConfig returns a sensible default PostgreSQL backup configuration.
func DefaultPostgresConfig() *PostgresBackupConfig {
	return &PostgresBackupConfig{
		Host:             "localhost",
		Port:             5432,
		Username:         "postgres",
		OutputFormat:     PostgresFormatCustom,
		CompressionLevel: 6,
		SSLMode:          "prefer",
	}
}

// SetProxmoxOptions sets the Proxmox options from JSON bytes.
func (s *Schedule) SetProxmoxOptions(data []byte) error {
	if len(data) == 0 {
		s.ProxmoxOptions = nil
		return nil
	}
	var opts ProxmoxBackupOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return err
	}
	s.ProxmoxOptions = &opts
	return nil
}

// ProxmoxOptionsJSON returns the Proxmox options as JSON bytes for database storage.
func (s *Schedule) ProxmoxOptionsJSON() ([]byte, error) {
	if s.ProxmoxOptions == nil {
		return nil, nil
	}
	return json.Marshal(s.ProxmoxOptions)
}

// DefaultProxmoxOptions returns a sensible default Proxmox backup configuration.
func DefaultProxmoxOptions() *ProxmoxBackupOptions {
	return &ProxmoxBackupOptions{
		Mode:        "snapshot",
		Compress:    "zstd",
		MaxWait:     60, // 60 minutes
		IncludeRAM:  false,
		RemoveAfter: true,
	}
}

// BackupQueueStatus represents the status of a backup queue item.
type BackupQueueStatus string

const (
	// QueueStatusPending indicates the backup is waiting to run.
	QueueStatusPending BackupQueueStatus = "pending"
	// QueueStatusRunning indicates the backup is currently running.
	QueueStatusRunning BackupQueueStatus = "running"
	// QueueStatusCompleted indicates the backup completed successfully.
	QueueStatusCompleted BackupQueueStatus = "completed"
	// QueueStatusFailed indicates the backup failed.
	QueueStatusFailed BackupQueueStatus = "failed"
	// QueueStatusPreempted indicates the backup was preempted by a higher priority backup.
	QueueStatusPreempted BackupQueueStatus = "preempted"
	// QueueStatusCanceled indicates the backup was canceled.
	QueueStatusCanceled BackupQueueStatus = "canceled"
)

// BackupQueueItem represents a backup in the priority queue.
type BackupQueueItem struct {
	ID          uuid.UUID         `json:"id"`
	ScheduleID  uuid.UUID         `json:"schedule_id"`
	AgentID     uuid.UUID         `json:"agent_id"`
	Priority    SchedulePriority  `json:"priority"`
	Status      BackupQueueStatus `json:"status"`
	QueuedAt    time.Time         `json:"queued_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	PreemptedBy *uuid.UUID        `json:"preempted_by,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// NewBackupQueueItem creates a new backup queue item.
func NewBackupQueueItem(scheduleID, agentID uuid.UUID, priority SchedulePriority) *BackupQueueItem {
	now := time.Now()
	return &BackupQueueItem{
		ID:         uuid.New(),
		ScheduleID: scheduleID,
		AgentID:    agentID,
		Priority:   priority,
		Status:     QueueStatusPending,
		QueuedAt:   now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// BackupQueueSummary provides a summary of the backup queue for display.
type BackupQueueSummary struct {
	TotalPending   int `json:"total_pending"`
	TotalRunning   int `json:"total_running"`
	HighPriority   int `json:"high_priority"`
	MediumPriority int `json:"medium_priority"`
	LowPriority    int `json:"low_priority"`
}
