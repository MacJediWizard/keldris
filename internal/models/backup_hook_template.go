package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// BackupHookTemplateVisibility represents who can view and use a template.
type BackupHookTemplateVisibility string

const (
	// BackupHookTemplateVisibilityBuiltIn means this is a system-provided template.
	BackupHookTemplateVisibilityBuiltIn BackupHookTemplateVisibility = "built_in"
	// BackupHookTemplateVisibilityPrivate means only the creating user can see the template.
	BackupHookTemplateVisibilityPrivate BackupHookTemplateVisibility = "private"
	// BackupHookTemplateVisibilityOrg means all members of the organization can see the template.
	BackupHookTemplateVisibilityOrg BackupHookTemplateVisibility = "organization"
)

// ValidBackupHookTemplateVisibilities returns all valid visibility options.
func ValidBackupHookTemplateVisibilities() []BackupHookTemplateVisibility {
	return []BackupHookTemplateVisibility{
		BackupHookTemplateVisibilityBuiltIn,
		BackupHookTemplateVisibilityPrivate,
		BackupHookTemplateVisibilityOrg,
	}
}

// BackupHookTemplateVariable represents a customizable variable in a template.
type BackupHookTemplateVariable struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Default     string `json:"default" yaml:"default"`
	Required    bool   `json:"required" yaml:"required"`
	Sensitive   bool   `json:"sensitive,omitempty" yaml:"sensitive,omitempty"`
}

// BackupHookTemplateScript represents a script in a template.
type BackupHookTemplateScript struct {
	Script         string `json:"script" yaml:"script"`
	TimeoutSeconds int    `json:"timeout_seconds" yaml:"timeout_seconds"`
	FailOnError    bool   `json:"fail_on_error" yaml:"fail_on_error"`
}

// BackupHookTemplateScripts contains all scripts for a template.
type BackupHookTemplateScripts struct {
	PreBackup   *BackupHookTemplateScript `json:"pre_backup,omitempty" yaml:"pre_backup,omitempty"`
	PostSuccess *BackupHookTemplateScript `json:"post_success,omitempty" yaml:"post_success,omitempty"`
	PostFailure *BackupHookTemplateScript `json:"post_failure,omitempty" yaml:"post_failure,omitempty"`
	PostAlways  *BackupHookTemplateScript `json:"post_always,omitempty" yaml:"post_always,omitempty"`
}

// BackupHookTemplate represents a pre-built or custom backup hook template.
type BackupHookTemplate struct {
	ID          uuid.UUID                    `json:"id"`
	OrgID       uuid.UUID                    `json:"org_id,omitempty"`      // Nil for built-in templates
	CreatedByID uuid.UUID                    `json:"created_by_id,omitempty"` // Nil for built-in templates
	Name        string                       `json:"name"`
	Description string                       `json:"description,omitempty"`
	ServiceType string                       `json:"service_type"`
	Icon        string                       `json:"icon,omitempty"`
	Tags        []string                     `json:"tags,omitempty"`
	Variables   []BackupHookTemplateVariable `json:"variables,omitempty"`
	Scripts     BackupHookTemplateScripts    `json:"scripts"`
	Visibility  BackupHookTemplateVisibility `json:"visibility"`
	UsageCount  int                          `json:"usage_count"`
	CreatedAt   time.Time                    `json:"created_at"`
	UpdatedAt   time.Time                    `json:"updated_at"`
}

// NewBackupHookTemplate creates a new custom BackupHookTemplate.
func NewBackupHookTemplate(orgID, createdByID uuid.UUID, name, serviceType string) *BackupHookTemplate {
	now := time.Now()
	return &BackupHookTemplate{
		ID:          uuid.New(),
		OrgID:       orgID,
		CreatedByID: createdByID,
		Name:        name,
		ServiceType: serviceType,
		Visibility:  BackupHookTemplateVisibilityPrivate,
		UsageCount:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// IsBuiltIn returns true if this is a built-in system template.
func (t *BackupHookTemplate) IsBuiltIn() bool {
	return t.Visibility == BackupHookTemplateVisibilityBuiltIn
}

// SetTags sets the tags from JSON bytes.
func (t *BackupHookTemplate) SetTags(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &t.Tags)
}

// TagsJSON returns the tags as JSON bytes for database storage.
func (t *BackupHookTemplate) TagsJSON() ([]byte, error) {
	if t.Tags == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(t.Tags)
}

// SetVariables sets the variables from JSON bytes.
func (t *BackupHookTemplate) SetVariables(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &t.Variables)
}

// VariablesJSON returns the variables as JSON bytes for database storage.
func (t *BackupHookTemplate) VariablesJSON() ([]byte, error) {
	if t.Variables == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(t.Variables)
}

// SetScripts sets the scripts from JSON bytes.
func (t *BackupHookTemplate) SetScripts(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &t.Scripts)
}

// ScriptsJSON returns the scripts as JSON bytes for database storage.
func (t *BackupHookTemplate) ScriptsJSON() ([]byte, error) {
	return json.Marshal(t.Scripts)
}

// IsValidVisibility checks if the visibility is valid.
func (t *BackupHookTemplate) IsValidVisibility() bool {
	for _, v := range ValidBackupHookTemplateVisibilities() {
		if t.Visibility == v {
			return true
		}
	}
	return false
}

// CreateBackupHookTemplateRequest is the request body for creating a custom template.
type CreateBackupHookTemplateRequest struct {
	Name        string                       `json:"name" binding:"required,min=1,max=255"`
	Description string                       `json:"description,omitempty"`
	ServiceType string                       `json:"service_type" binding:"required,min=1,max=100"`
	Icon        string                       `json:"icon,omitempty"`
	Tags        []string                     `json:"tags,omitempty"`
	Variables   []BackupHookTemplateVariable `json:"variables,omitempty"`
	Scripts     BackupHookTemplateScripts    `json:"scripts" binding:"required"`
	Visibility  BackupHookTemplateVisibility `json:"visibility,omitempty"`
}

// UpdateBackupHookTemplateRequest is the request body for updating a custom template.
type UpdateBackupHookTemplateRequest struct {
	Name        *string                       `json:"name,omitempty"`
	Description *string                       `json:"description,omitempty"`
	ServiceType *string                       `json:"service_type,omitempty"`
	Icon        *string                       `json:"icon,omitempty"`
	Tags        []string                      `json:"tags,omitempty"`
	Variables   []BackupHookTemplateVariable  `json:"variables,omitempty"`
	Scripts     *BackupHookTemplateScripts    `json:"scripts,omitempty"`
	Visibility  *BackupHookTemplateVisibility `json:"visibility,omitempty"`
}

// ApplyBackupHookTemplateRequest is the request body for applying a template to a schedule.
type ApplyBackupHookTemplateRequest struct {
	ScheduleID     uuid.UUID         `json:"schedule_id" binding:"required"`
	VariableValues map[string]string `json:"variable_values,omitempty"`
}

// ApplyBackupHookTemplateResponse is the response when applying a template.
type ApplyBackupHookTemplateResponse struct {
	Scripts []*BackupScript `json:"scripts"`
	Message string          `json:"message"`
}

// BackupHookTemplatesResponse wraps a list of templates for API responses.
type BackupHookTemplatesResponse struct {
	Templates []*BackupHookTemplate `json:"templates"`
}
