// Package docker provides Docker container backup configuration via labels.
package docker

import (
	"strconv"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Label constants for Docker container backup configuration.
const (
	// LabelPrefix is the prefix for all Keldris backup labels.
	LabelPrefix = "keldris.backup"

	// LabelEnabled enables backup for the container (keldris.backup=true).
	LabelEnabled = "keldris.backup"

	// LabelSchedule sets the backup schedule (keldris.backup.schedule=daily).
	LabelSchedule = "keldris.backup.schedule"

	// LabelCron sets a custom cron expression (keldris.backup.cron=0 2 * * *).
	LabelCron = "keldris.backup.cron"

	// LabelExclude sets paths to exclude (keldris.backup.exclude=/tmp,/var/cache).
	LabelExclude = "keldris.backup.exclude"

	// LabelPreHook sets the pre-backup command (keldris.backup.pre-hook=pg_dump).
	LabelPreHook = "keldris.backup.pre-hook"

	// LabelPostHook sets the post-backup command (keldris.backup.post-hook=cleanup.sh).
	LabelPostHook = "keldris.backup.post-hook"

	// LabelStopOnBackup stops the container during backup (keldris.backup.stop=true).
	LabelStopOnBackup = "keldris.backup.stop"

	// LabelBackupVolumes enables volume backup (keldris.backup.volumes=true).
	LabelBackupVolumes = "keldris.backup.volumes"

	// LabelBackupBindMounts enables bind mount backup (keldris.backup.bind-mounts=true).
	LabelBackupBindMounts = "keldris.backup.bind-mounts"

	// LabelRetentionKeepLast sets retention keep-last count (keldris.backup.retention.keep-last=5).
	LabelRetentionKeepLast = "keldris.backup.retention.keep-last"

	// LabelRetentionKeepDaily sets retention keep-daily count (keldris.backup.retention.keep-daily=7).
	LabelRetentionKeepDaily = "keldris.backup.retention.keep-daily"

	// LabelRetentionKeepWeekly sets retention keep-weekly count (keldris.backup.retention.keep-weekly=4).
	LabelRetentionKeepWeekly = "keldris.backup.retention.keep-weekly"

	// LabelRetentionKeepMonthly sets retention keep-monthly count (keldris.backup.retention.keep-monthly=6).
	LabelRetentionKeepMonthly = "keldris.backup.retention.keep-monthly"

	// LabelPriority sets the backup priority (keldris.backup.priority=high).
	LabelPriority = "keldris.backup.priority"
)

// LabelParser parses Docker container labels into backup configuration.
type LabelParser struct{}

// NewLabelParser creates a new LabelParser.
func NewLabelParser() *LabelParser {
	return &LabelParser{}
}

// ContainerInfo represents minimal Docker container information for parsing.
type ContainerInfo struct {
	ID          string
	Name        string
	Image       string
	Labels      map[string]string
	Mounts      []MountInfo
	CreatedAt   time.Time
	Status      string
}

// MountInfo represents a container mount point.
type MountInfo struct {
	Type        string // "volume" or "bind"
	Source      string
	Destination string
	ReadOnly    bool
}

// ParseConfig parses Docker labels into a container backup configuration.
func (p *LabelParser) ParseConfig(agentID uuid.UUID, container ContainerInfo) *models.DockerContainerConfig {
	// Check if backup is enabled for this container
	if !p.isBackupEnabled(container.Labels) {
		return nil
	}

	config := models.NewDockerContainerConfig(
		agentID,
		container.ID,
		container.Name,
		container.Image,
	)

	// Parse schedule
	config.Schedule = p.parseSchedule(container.Labels)

	// Parse custom cron expression
	if cron := p.getString(container.Labels, LabelCron); cron != "" {
		config.Schedule = models.DockerBackupScheduleCustom
		config.CronExpression = cron
	}

	// Parse excludes (comma-separated)
	if excludes := p.getString(container.Labels, LabelExclude); excludes != "" {
		config.Excludes = p.parseCSV(excludes)
	}

	// Parse hooks
	config.PreHook = p.getString(container.Labels, LabelPreHook)
	config.PostHook = p.getString(container.Labels, LabelPostHook)

	// Parse boolean flags
	config.StopOnBackup = p.getBool(container.Labels, LabelStopOnBackup, false)
	config.BackupVolumes = p.getBool(container.Labels, LabelBackupVolumes, true)
	config.BackupBindMounts = p.getBool(container.Labels, LabelBackupBindMounts, false)

	// Store original labels
	config.Labels = p.filterKeldrisLabels(container.Labels)

	return config
}

// ParseRetentionPolicy parses retention policy from labels.
func (p *LabelParser) ParseRetentionPolicy(labels map[string]string) *models.RetentionPolicy {
	policy := &models.RetentionPolicy{}
	hasPolicy := false

	if val := p.getInt(labels, LabelRetentionKeepLast); val > 0 {
		policy.KeepLast = val
		hasPolicy = true
	}
	if val := p.getInt(labels, LabelRetentionKeepDaily); val > 0 {
		policy.KeepDaily = val
		hasPolicy = true
	}
	if val := p.getInt(labels, LabelRetentionKeepWeekly); val > 0 {
		policy.KeepWeekly = val
		hasPolicy = true
	}
	if val := p.getInt(labels, LabelRetentionKeepMonthly); val > 0 {
		policy.KeepMonthly = val
		hasPolicy = true
	}

	if !hasPolicy {
		return models.DefaultRetentionPolicy()
	}

	return policy
}

// isBackupEnabled checks if backup is enabled for the container.
func (p *LabelParser) isBackupEnabled(labels map[string]string) bool {
	val, ok := labels[LabelEnabled]
	if !ok {
		return false
	}
	return p.parseBool(val)
}

// parseSchedule parses the backup schedule from labels.
func (p *LabelParser) parseSchedule(labels map[string]string) models.DockerBackupSchedule {
	val, ok := labels[LabelSchedule]
	if !ok {
		return models.DockerBackupScheduleDaily
	}

	switch strings.ToLower(strings.TrimSpace(val)) {
	case "hourly":
		return models.DockerBackupScheduleHourly
	case "daily":
		return models.DockerBackupScheduleDaily
	case "weekly":
		return models.DockerBackupScheduleWeekly
	case "monthly":
		return models.DockerBackupScheduleMonthly
	default:
		return models.DockerBackupScheduleDaily
	}
}

// getString retrieves a string label value.
func (p *LabelParser) getString(labels map[string]string, key string) string {
	return strings.TrimSpace(labels[key])
}

// getBool retrieves a boolean label value with a default.
func (p *LabelParser) getBool(labels map[string]string, key string, defaultVal bool) bool {
	val, ok := labels[key]
	if !ok {
		return defaultVal
	}
	return p.parseBool(val)
}

// getInt retrieves an integer label value.
func (p *LabelParser) getInt(labels map[string]string, key string) int {
	val, ok := labels[key]
	if !ok {
		return 0
	}
	i, err := strconv.Atoi(strings.TrimSpace(val))
	if err != nil {
		return 0
	}
	return i
}

// parseBool parses a boolean string value.
func (p *LabelParser) parseBool(val string) bool {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "true", "yes", "1", "on", "enabled":
		return true
	default:
		return false
	}
}

// parseCSV parses a comma-separated value string into a slice.
func (p *LabelParser) parseCSV(val string) []string {
	if val == "" {
		return nil
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// filterKeldrisLabels filters and returns only Keldris backup labels.
func (p *LabelParser) filterKeldrisLabels(labels map[string]string) map[string]string {
	filtered := make(map[string]string)
	for key, val := range labels {
		if strings.HasPrefix(key, LabelPrefix) {
			filtered[key] = val
		}
	}
	return filtered
}

// HasBackupLabel checks if a container has any Keldris backup label.
func (p *LabelParser) HasBackupLabel(labels map[string]string) bool {
	for key := range labels {
		if strings.HasPrefix(key, LabelPrefix) {
			return true
		}
	}
	return false
}

// ValidateLabels validates the backup labels and returns any errors.
func (p *LabelParser) ValidateLabels(labels map[string]string) []string {
	var errors []string

	// Validate schedule
	if schedule, ok := labels[LabelSchedule]; ok {
		switch strings.ToLower(strings.TrimSpace(schedule)) {
		case "hourly", "daily", "weekly", "monthly", "custom":
			// Valid
		default:
			errors = append(errors, "invalid schedule value: "+schedule+". Use hourly, daily, weekly, monthly, or custom")
		}
	}

	// Validate cron if schedule is custom
	if schedule := labels[LabelSchedule]; strings.ToLower(schedule) == "custom" {
		if _, ok := labels[LabelCron]; !ok {
			errors = append(errors, "custom schedule requires "+LabelCron+" label")
		}
	}

	// Validate boolean values
	boolLabels := []string{LabelEnabled, LabelStopOnBackup, LabelBackupVolumes, LabelBackupBindMounts}
	validBoolValues := map[string]bool{
		"true": true, "false": true, "yes": true, "no": true,
		"1": true, "0": true, "on": true, "off": true, "enabled": true, "disabled": true,
	}
	for _, label := range boolLabels {
		if val, ok := labels[label]; ok {
			if !validBoolValues[strings.ToLower(strings.TrimSpace(val))] {
				errors = append(errors, "invalid boolean value for "+label+": "+val)
			}
		}
	}

	// Validate integer values
	intLabels := []string{LabelRetentionKeepLast, LabelRetentionKeepDaily, LabelRetentionKeepWeekly, LabelRetentionKeepMonthly}
	for _, label := range intLabels {
		if val, ok := labels[label]; ok {
			if _, err := strconv.Atoi(strings.TrimSpace(val)); err != nil {
				errors = append(errors, "invalid integer value for "+label+": "+val)
			}
		}
	}

	return errors
}

// GenerateLabelDocs generates documentation for all supported labels.
func (p *LabelParser) GenerateLabelDocs() *models.DockerLabelDocs {
	return &models.DockerLabelDocs{
		Labels: []models.DockerLabelDoc{
			{
				Label:       LabelEnabled,
				Description: "Enable backup for this container. When set to true, the container will be included in automatic backup discovery.",
				Type:        "boolean",
				Default:     "false",
				Examples:    []string{"true", "false", "yes", "no"},
				Required:    true,
			},
			{
				Label:       LabelSchedule,
				Description: "Backup schedule frequency. Determines how often the container is backed up.",
				Type:        "string",
				Default:     "daily",
				Examples:    []string{"hourly", "daily", "weekly", "monthly", "custom"},
				Required:    false,
			},
			{
				Label:       LabelCron,
				Description: "Custom cron expression for backup scheduling. Required when schedule is set to 'custom'. Uses standard 5-field cron format (minute hour day-of-month month day-of-week).",
				Type:        "string",
				Default:     "",
				Examples:    []string{"0 2 * * *", "0 */4 * * *", "0 0 * * 0"},
				Required:    false,
			},
			{
				Label:       LabelExclude,
				Description: "Comma-separated list of paths to exclude from backup. Supports glob patterns.",
				Type:        "string",
				Default:     "",
				Examples:    []string{"/tmp", "/var/cache,/tmp,*.log", "/data/temp/**"},
				Required:    false,
			},
			{
				Label:       LabelPreHook,
				Description: "Command to execute inside the container before starting the backup. Useful for database dumps, flushing caches, etc.",
				Type:        "string",
				Default:     "",
				Examples:    []string{"pg_dump -U postgres mydb > /backup/dump.sql", "redis-cli BGSAVE", "mysqldump -u root mydb > /backup/dump.sql"},
				Required:    false,
			},
			{
				Label:       LabelPostHook,
				Description: "Command to execute inside the container after the backup completes. Useful for cleanup operations.",
				Type:        "string",
				Default:     "",
				Examples:    []string{"rm /backup/dump.sql", "redis-cli BGREWRITEAOF"},
				Required:    false,
			},
			{
				Label:       LabelStopOnBackup,
				Description: "Stop the container during backup to ensure data consistency. The container will be automatically restarted after backup.",
				Type:        "boolean",
				Default:     "false",
				Examples:    []string{"true", "false"},
				Required:    false,
			},
			{
				Label:       LabelBackupVolumes,
				Description: "Include Docker named volumes in the backup.",
				Type:        "boolean",
				Default:     "true",
				Examples:    []string{"true", "false"},
				Required:    false,
			},
			{
				Label:       LabelBackupBindMounts,
				Description: "Include bind mounts in the backup. Be cautious as this may include host system files.",
				Type:        "boolean",
				Default:     "false",
				Examples:    []string{"true", "false"},
				Required:    false,
			},
			{
				Label:       LabelRetentionKeepLast,
				Description: "Number of most recent snapshots to keep regardless of age.",
				Type:        "integer",
				Default:     "5",
				Examples:    []string{"3", "5", "10"},
				Required:    false,
			},
			{
				Label:       LabelRetentionKeepDaily,
				Description: "Number of daily snapshots to keep.",
				Type:        "integer",
				Default:     "7",
				Examples:    []string{"7", "14", "30"},
				Required:    false,
			},
			{
				Label:       LabelRetentionKeepWeekly,
				Description: "Number of weekly snapshots to keep.",
				Type:        "integer",
				Default:     "4",
				Examples:    []string{"4", "8", "12"},
				Required:    false,
			},
			{
				Label:       LabelRetentionKeepMonthly,
				Description: "Number of monthly snapshots to keep.",
				Type:        "integer",
				Default:     "6",
				Examples:    []string{"3", "6", "12"},
				Required:    false,
			},
			{
				Label:       LabelPriority,
				Description: "Backup priority for this container. Higher priority containers are backed up first.",
				Type:        "string",
				Default:     "medium",
				Examples:    []string{"high", "medium", "low"},
				Required:    false,
			},
		},
		GeneratedAt: time.Now(),
	}
}

// GenerateDockerComposeExample generates an example Docker Compose snippet.
func (p *LabelParser) GenerateDockerComposeExample() string {
	return `# Example Docker Compose configuration with Keldris backup labels
version: '3.8'

services:
  postgres:
    image: postgres:15
    labels:
      - "keldris.backup=true"
      - "keldris.backup.schedule=daily"
      - "keldris.backup.pre-hook=pg_dump -U postgres -d mydb -f /var/lib/postgresql/data/backup.sql"
      - "keldris.backup.exclude=/var/lib/postgresql/data/pg_wal"
      - "keldris.backup.volumes=true"
      - "keldris.backup.retention.keep-last=5"
      - "keldris.backup.retention.keep-daily=7"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7
    labels:
      - "keldris.backup=true"
      - "keldris.backup.schedule=hourly"
      - "keldris.backup.pre-hook=redis-cli BGSAVE"
      - "keldris.backup.volumes=true"
    volumes:
      - redis_data:/data

  web:
    image: nginx:latest
    labels:
      - "keldris.backup=true"
      - "keldris.backup.schedule=weekly"
      - "keldris.backup.exclude=/tmp,/var/cache"
      - "keldris.backup.bind-mounts=true"
    volumes:
      - ./html:/usr/share/nginx/html:ro

volumes:
  postgres_data:
  redis_data:
`
}

// GenerateDockerRunExample generates an example docker run command.
func (p *LabelParser) GenerateDockerRunExample() string {
	return `# Example docker run command with Keldris backup labels
docker run -d \
  --name myapp-postgres \
  --label "keldris.backup=true" \
  --label "keldris.backup.schedule=daily" \
  --label "keldris.backup.pre-hook=pg_dump -U postgres -d mydb -f /backup/dump.sql" \
  --label "keldris.backup.exclude=/tmp" \
  --label "keldris.backup.volumes=true" \
  --label "keldris.backup.retention.keep-last=5" \
  -v postgres_data:/var/lib/postgresql/data \
  -v backup_data:/backup \
  postgres:15
`
}
