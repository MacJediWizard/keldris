package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TemplateType represents the type of configuration template.
type TemplateType string

const (
	// TemplateTypeSchedule represents a schedule template.
	TemplateTypeSchedule TemplateType = "schedule"
	// TemplateTypeAgent represents an agent configuration template.
	TemplateTypeAgent TemplateType = "agent"
	// TemplateTypeRepository represents a repository configuration template.
	TemplateTypeRepository TemplateType = "repository"
	// TemplateTypeBundle represents a bundle of multiple configurations.
	TemplateTypeBundle TemplateType = "bundle"
)

// TemplateVisibility represents who can view and use a template.
type TemplateVisibility string

const (
	// TemplateVisibilityPrivate means only the creating user can see the template.
	TemplateVisibilityPrivate TemplateVisibility = "private"
	// TemplateVisibilityOrg means all members of the organization can see the template.
	TemplateVisibilityOrg TemplateVisibility = "organization"
	// TemplateVisibilityPublic means all users can see the template (system-wide).
	TemplateVisibilityPublic TemplateVisibility = "public"
)

// ConfigTemplate represents a saved configuration template.
type ConfigTemplate struct {
	ID          uuid.UUID          `json:"id"`
	OrgID       uuid.UUID          `json:"org_id"`
	CreatedByID uuid.UUID          `json:"created_by_id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Type        TemplateType       `json:"type"`
	Visibility  TemplateVisibility `json:"visibility"`
	Tags        []string           `json:"tags,omitempty"`
	// Config contains the exported configuration data (JSON).
	Config    []byte    `json:"-"` // Raw bytes stored in DB
	UsageCount int       `json:"usage_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewConfigTemplate creates a new ConfigTemplate.
func NewConfigTemplate(orgID, createdByID uuid.UUID, name string, templateType TemplateType, config []byte) *ConfigTemplate {
	now := time.Now()
	return &ConfigTemplate{
		ID:          uuid.New(),
		OrgID:       orgID,
		CreatedByID: createdByID,
		Name:        name,
		Type:        templateType,
		Visibility:  TemplateVisibilityOrg,
		Config:      config,
		UsageCount:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetTags sets the tags from JSON bytes.
func (t *ConfigTemplate) SetTags(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &t.Tags)
}

// TagsJSON returns the tags as JSON bytes for database storage.
func (t *ConfigTemplate) TagsJSON() ([]byte, error) {
	if t.Tags == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(t.Tags)
}

// GetConfigMap returns the config as a map for JSON serialization.
func (t *ConfigTemplate) GetConfigMap() (map[string]any, error) {
	if len(t.Config) == 0 {
		return nil, nil
	}
	var config map[string]any
	if err := json.Unmarshal(t.Config, &config); err != nil {
		return nil, err
	}
	return config, nil
}

// ConfigTemplateWithConfig represents a template with its config exposed for API responses.
type ConfigTemplateWithConfig struct {
	ID          uuid.UUID          `json:"id"`
	OrgID       uuid.UUID          `json:"org_id"`
	CreatedByID uuid.UUID          `json:"created_by_id"`
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	Type        TemplateType       `json:"type"`
	Visibility  TemplateVisibility `json:"visibility"`
	Tags        []string           `json:"tags,omitempty"`
	Config      map[string]any     `json:"config"`
	UsageCount  int                `json:"usage_count"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// ToConfigTemplateWithConfig converts a ConfigTemplate to ConfigTemplateWithConfig.
func (t *ConfigTemplate) ToConfigTemplateWithConfig() (*ConfigTemplateWithConfig, error) {
	config, err := t.GetConfigMap()
	if err != nil {
		return nil, err
	}
	return &ConfigTemplateWithConfig{
		ID:          t.ID,
		OrgID:       t.OrgID,
		CreatedByID: t.CreatedByID,
		Name:        t.Name,
		Description: t.Description,
		Type:        t.Type,
		Visibility:  t.Visibility,
		Tags:        t.Tags,
		Config:      config,
		UsageCount:  t.UsageCount,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}, nil
}

// ValidTemplateTypes returns all valid template types.
func ValidTemplateTypes() []TemplateType {
	return []TemplateType{
		TemplateTypeSchedule,
		TemplateTypeAgent,
		TemplateTypeRepository,
		TemplateTypeBundle,
	}
}

// ValidTemplateVisibilities returns all valid visibility options.
func ValidTemplateVisibilities() []TemplateVisibility {
	return []TemplateVisibility{
		TemplateVisibilityPrivate,
		TemplateVisibilityOrg,
		TemplateVisibilityPublic,
	}
}

// IsValidType checks if the template type is valid.
func (t *ConfigTemplate) IsValidType() bool {
	for _, vt := range ValidTemplateTypes() {
		if t.Type == vt {
			return true
		}
	}
	return false
}

// IsValidVisibility checks if the visibility is valid.
func (t *ConfigTemplate) IsValidVisibility() bool {
	for _, v := range ValidTemplateVisibilities() {
		if t.Visibility == v {
			return true
		}
	}
	return false
}
