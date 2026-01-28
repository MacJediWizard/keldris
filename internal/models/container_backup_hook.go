package models

import (
	"time"

	"github.com/google/uuid"
)

// ContainerHookType represents when the hook should run relative to the container backup.
type ContainerHookType string

const (
	// ContainerHookTypePreBackup runs before the container backup starts.
	ContainerHookTypePreBackup ContainerHookType = "pre_backup"
	// ContainerHookTypePostBackup runs after the container backup completes (regardless of success).
	ContainerHookTypePostBackup ContainerHookType = "post_backup"
)

// ValidContainerHookTypes returns all valid hook types.
func ValidContainerHookTypes() []ContainerHookType {
	return []ContainerHookType{
		ContainerHookTypePreBackup,
		ContainerHookTypePostBackup,
	}
}

// IsValid checks if the hook type is valid.
func (t ContainerHookType) IsValid() bool {
	for _, valid := range ValidContainerHookTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

// ContainerHookTemplate represents a pre-defined hook template for common applications.
type ContainerHookTemplate string

const (
	// ContainerHookTemplateNone indicates a custom script with no template.
	ContainerHookTemplateNone ContainerHookTemplate = "none"
	// ContainerHookTemplatePostgres is for PostgreSQL database dumps.
	ContainerHookTemplatePostgres ContainerHookTemplate = "postgres"
	// ContainerHookTemplateMySQL is for MySQL/MariaDB database dumps.
	ContainerHookTemplateMySQL ContainerHookTemplate = "mysql"
	// ContainerHookTemplateMongoDB is for MongoDB database dumps.
	ContainerHookTemplateMongoDB ContainerHookTemplate = "mongodb"
	// ContainerHookTemplateRedis is for Redis persistence operations.
	ContainerHookTemplateRedis ContainerHookTemplate = "redis"
	// ContainerHookTemplateElasticsearch is for Elasticsearch snapshot operations.
	ContainerHookTemplateElasticsearch ContainerHookTemplate = "elasticsearch"
)

// ValidContainerHookTemplates returns all valid hook templates.
func ValidContainerHookTemplates() []ContainerHookTemplate {
	return []ContainerHookTemplate{
		ContainerHookTemplateNone,
		ContainerHookTemplatePostgres,
		ContainerHookTemplateMySQL,
		ContainerHookTemplateMongoDB,
		ContainerHookTemplateRedis,
		ContainerHookTemplateElasticsearch,
	}
}

// IsValid checks if the template is valid.
func (t ContainerHookTemplate) IsValid() bool {
	for _, valid := range ValidContainerHookTemplates() {
		if t == valid {
			return true
		}
	}
	return false
}

// ContainerBackupHook represents a hook to run inside a container before or after backup.
type ContainerBackupHook struct {
	ID              uuid.UUID             `json:"id"`
	ScheduleID      uuid.UUID             `json:"schedule_id"`
	ContainerName   string                `json:"container_name"`
	Type            ContainerHookType     `json:"type"`
	Template        ContainerHookTemplate `json:"template"`
	Command         string                `json:"command"`
	WorkingDir      string                `json:"working_dir,omitempty"`
	User            string                `json:"user,omitempty"`
	TimeoutSeconds  int                   `json:"timeout_seconds"`
	FailOnError     bool                  `json:"fail_on_error"`
	Enabled         bool                  `json:"enabled"`
	Description     string                `json:"description,omitempty"`
	TemplateVars    map[string]string     `json:"template_vars,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// NewContainerBackupHook creates a new ContainerBackupHook with the given details.
func NewContainerBackupHook(scheduleID uuid.UUID, containerName string, hookType ContainerHookType, command string) *ContainerBackupHook {
	now := time.Now()
	return &ContainerBackupHook{
		ID:             uuid.New(),
		ScheduleID:     scheduleID,
		ContainerName:  containerName,
		Type:           hookType,
		Template:       ContainerHookTemplateNone,
		Command:        command,
		TimeoutSeconds: 300, // Default 5 minutes
		FailOnError:    false,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// NewContainerBackupHookFromTemplate creates a new hook from a pre-defined template.
func NewContainerBackupHookFromTemplate(scheduleID uuid.UUID, containerName string, hookType ContainerHookType, template ContainerHookTemplate, templateVars map[string]string) *ContainerBackupHook {
	hook := NewContainerBackupHook(scheduleID, containerName, hookType, "")
	hook.Template = template
	hook.TemplateVars = templateVars
	return hook
}

// IsPreBackup returns true if this is a pre-backup hook.
func (h *ContainerBackupHook) IsPreBackup() bool {
	return h.Type == ContainerHookTypePreBackup
}

// IsPostBackup returns true if this is a post-backup hook.
func (h *ContainerBackupHook) IsPostBackup() bool {
	return h.Type == ContainerHookTypePostBackup
}

// ContainerHookExecution represents the result of running a container hook.
type ContainerHookExecution struct {
	HookID      uuid.UUID     `json:"hook_id"`
	BackupID    uuid.UUID     `json:"backup_id"`
	Container   string        `json:"container"`
	Type        ContainerHookType `json:"type"`
	Command     string        `json:"command"`
	Output      string        `json:"output"`
	ExitCode    int           `json:"exit_code"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	StartedAt   time.Time     `json:"started_at"`
	CompletedAt time.Time     `json:"completed_at"`
}

// CreateContainerBackupHookRequest is the request body for creating a container backup hook.
type CreateContainerBackupHookRequest struct {
	ContainerName  string                `json:"container_name" binding:"required"`
	Type           ContainerHookType     `json:"type" binding:"required"`
	Template       ContainerHookTemplate `json:"template,omitempty"`
	Command        string                `json:"command,omitempty"`
	WorkingDir     string                `json:"working_dir,omitempty"`
	User           string                `json:"user,omitempty"`
	TimeoutSeconds *int                  `json:"timeout_seconds,omitempty"`
	FailOnError    *bool                 `json:"fail_on_error,omitempty"`
	Enabled        *bool                 `json:"enabled,omitempty"`
	Description    string                `json:"description,omitempty"`
	TemplateVars   map[string]string     `json:"template_vars,omitempty"`
}

// UpdateContainerBackupHookRequest is the request body for updating a container backup hook.
type UpdateContainerBackupHookRequest struct {
	ContainerName  *string               `json:"container_name,omitempty"`
	Command        *string               `json:"command,omitempty"`
	WorkingDir     *string               `json:"working_dir,omitempty"`
	User           *string               `json:"user,omitempty"`
	TimeoutSeconds *int                  `json:"timeout_seconds,omitempty"`
	FailOnError    *bool                 `json:"fail_on_error,omitempty"`
	Enabled        *bool                 `json:"enabled,omitempty"`
	Description    *string               `json:"description,omitempty"`
	TemplateVars   map[string]string     `json:"template_vars,omitempty"`
}
