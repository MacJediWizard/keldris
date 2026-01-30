// Package hooks provides backup hook template management functionality.
package hooks

import (
	"embed"
	"fmt"
	"strings"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

//go:embed templates/*.yaml
var templatesFS embed.FS

// BuiltInTemplateYAML represents the YAML structure of a built-in template file.
type BuiltInTemplateYAML struct {
	Name        string                                `yaml:"name"`
	Description string                                `yaml:"description"`
	ServiceType string                                `yaml:"service_type"`
	Icon        string                                `yaml:"icon"`
	Tags        []string                              `yaml:"tags"`
	Variables   []models.BackupHookTemplateVariable   `yaml:"variables"`
	Scripts     models.BackupHookTemplateScripts      `yaml:"scripts"`
}

// LoadBuiltInTemplates loads all built-in templates from embedded YAML files.
func LoadBuiltInTemplates() ([]*models.BackupHookTemplate, error) {
	entries, err := templatesFS.ReadDir("templates")
	if err != nil {
		return nil, fmt.Errorf("read templates directory: %w", err)
	}

	var templates []*models.BackupHookTemplate

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		data, err := templatesFS.ReadFile("templates/" + entry.Name())
		if err != nil {
			return nil, fmt.Errorf("read template file %s: %w", entry.Name(), err)
		}

		var yamlTemplate BuiltInTemplateYAML
		if err := yaml.Unmarshal(data, &yamlTemplate); err != nil {
			return nil, fmt.Errorf("parse template file %s: %w", entry.Name(), err)
		}

		template := &models.BackupHookTemplate{
			ID:          generateBuiltInTemplateID(yamlTemplate.ServiceType),
			Name:        yamlTemplate.Name,
			Description: yamlTemplate.Description,
			ServiceType: yamlTemplate.ServiceType,
			Icon:        yamlTemplate.Icon,
			Tags:        yamlTemplate.Tags,
			Variables:   yamlTemplate.Variables,
			Scripts:     yamlTemplate.Scripts,
			Visibility:  models.BackupHookTemplateVisibilityBuiltIn,
			UsageCount:  0,
		}

		templates = append(templates, template)
	}

	return templates, nil
}

// LoadBuiltInTemplate loads a specific built-in template by service type.
func LoadBuiltInTemplate(serviceType string) (*models.BackupHookTemplate, error) {
	templates, err := LoadBuiltInTemplates()
	if err != nil {
		return nil, err
	}

	for _, t := range templates {
		if t.ServiceType == serviceType {
			return t, nil
		}
	}

	return nil, fmt.Errorf("built-in template not found: %s", serviceType)
}

// GetBuiltInTemplateByID finds a built-in template by its ID.
func GetBuiltInTemplateByID(id uuid.UUID) (*models.BackupHookTemplate, error) {
	templates, err := LoadBuiltInTemplates()
	if err != nil {
		return nil, err
	}

	for _, t := range templates {
		if t.ID == id {
			return t, nil
		}
	}

	return nil, fmt.Errorf("built-in template not found: %s", id)
}

// generateBuiltInTemplateID generates a deterministic UUID for a built-in template.
// This ensures the same template always has the same ID across restarts.
func generateBuiltInTemplateID(serviceType string) uuid.UUID {
	// Use a namespace UUID for built-in templates
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace
	return uuid.NewSHA1(namespace, []byte("keldris-backup-hook-template-"+serviceType))
}

// RenderScript renders a script template with the provided variable values.
func RenderScript(script string, variables []models.BackupHookTemplateVariable, values map[string]string) string {
	result := script

	for _, v := range variables {
		value := v.Default
		if val, ok := values[v.Name]; ok {
			value = val
		}
		// Replace shell-style variables
		result = strings.ReplaceAll(result, "${"+v.Name+"}", value)
		result = strings.ReplaceAll(result, "${"+v.Name+":-"+v.Default+"}", value)
	}

	return result
}
