package docker

import (
	"strings"

	"github.com/MacJediWizard/keldris/internal/models"
)

// TemplateInfo contains information about a hook template.
type TemplateInfo struct {
	Name           string `json:"name"`
	Template       models.ContainerHookTemplate `json:"template"`
	Description    string `json:"description"`
	PreBackupCmd   string `json:"pre_backup_cmd"`
	PostBackupCmd  string `json:"post_backup_cmd"`
	RequiredVars   []string `json:"required_vars"`
	OptionalVars   []string `json:"optional_vars"`
	DefaultVars    map[string]string `json:"default_vars"`
}

// Templates contains the pre-defined hook templates.
var Templates = map[models.ContainerHookTemplate]TemplateInfo{
	models.ContainerHookTemplatePostgres: {
		Name:        "PostgreSQL",
		Template:    models.ContainerHookTemplatePostgres,
		Description: "Dumps PostgreSQL database before backup and cleans up after",
		PreBackupCmd: `pg_dump -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DB:-postgres} -F c -f /tmp/backup_dump.sql && echo "Database dump created successfully"`,
		PostBackupCmd: `rm -f /tmp/backup_dump.sql && echo "Cleaned up dump file"`,
		RequiredVars: []string{},
		OptionalVars: []string{"POSTGRES_USER", "POSTGRES_DB", "POSTGRES_HOST"},
		DefaultVars: map[string]string{
			"POSTGRES_USER": "postgres",
			"POSTGRES_DB":   "postgres",
		},
	},
	models.ContainerHookTemplateMySQL: {
		Name:        "MySQL/MariaDB",
		Template:    models.ContainerHookTemplateMySQL,
		Description: "Dumps MySQL/MariaDB database before backup and cleans up after",
		PreBackupCmd: `mysqldump -u${MYSQL_USER:-root} -p${MYSQL_PASSWORD} ${MYSQL_DATABASE:-mysql} > /tmp/backup_dump.sql && echo "Database dump created successfully"`,
		PostBackupCmd: `rm -f /tmp/backup_dump.sql && echo "Cleaned up dump file"`,
		RequiredVars: []string{"MYSQL_PASSWORD"},
		OptionalVars: []string{"MYSQL_USER", "MYSQL_DATABASE"},
		DefaultVars: map[string]string{
			"MYSQL_USER":     "root",
			"MYSQL_DATABASE": "mysql",
		},
	},
	models.ContainerHookTemplateMongoDB: {
		Name:        "MongoDB",
		Template:    models.ContainerHookTemplateMongoDB,
		Description: "Dumps MongoDB database before backup and cleans up after",
		PreBackupCmd: `mongodump --uri="${MONGO_URI:-mongodb://localhost:27017}" --out=/tmp/mongo_backup && echo "Database dump created successfully"`,
		PostBackupCmd: `rm -rf /tmp/mongo_backup && echo "Cleaned up dump directory"`,
		RequiredVars: []string{},
		OptionalVars: []string{"MONGO_URI", "MONGO_DATABASE"},
		DefaultVars: map[string]string{
			"MONGO_URI": "mongodb://localhost:27017",
		},
	},
	models.ContainerHookTemplateRedis: {
		Name:        "Redis",
		Template:    models.ContainerHookTemplateRedis,
		Description: "Triggers Redis BGSAVE before backup to ensure data persistence",
		PreBackupCmd: `redis-cli ${REDIS_AUTH:+-a $REDIS_AUTH} BGSAVE && sleep 2 && echo "Redis BGSAVE triggered successfully"`,
		PostBackupCmd: `echo "Redis backup hook completed"`,
		RequiredVars: []string{},
		OptionalVars: []string{"REDIS_AUTH"},
		DefaultVars:  map[string]string{},
	},
	models.ContainerHookTemplateElasticsearch: {
		Name:        "Elasticsearch",
		Template:    models.ContainerHookTemplateElasticsearch,
		Description: "Flushes Elasticsearch indices before backup for data consistency",
		PreBackupCmd: `curl -X POST "localhost:9200/_flush/synced" -H 'Content-Type: application/json' && echo "Elasticsearch indices flushed successfully"`,
		PostBackupCmd: `echo "Elasticsearch backup hook completed"`,
		RequiredVars: []string{},
		OptionalVars: []string{"ES_HOST", "ES_PORT", "ES_USER", "ES_PASSWORD"},
		DefaultVars: map[string]string{
			"ES_HOST": "localhost",
			"ES_PORT": "9200",
		},
	},
}

// GetTemplateCommand returns the command for a template and hook type.
func GetTemplateCommand(template models.ContainerHookTemplate, hookType models.ContainerHookType, vars map[string]string) string {
	info, ok := Templates[template]
	if !ok {
		return ""
	}

	var cmd string
	switch hookType {
	case models.ContainerHookTypePreBackup:
		cmd = info.PreBackupCmd
	case models.ContainerHookTypePostBackup:
		cmd = info.PostBackupCmd
	default:
		return ""
	}

	// Apply default vars first
	for key, value := range info.DefaultVars {
		placeholder := "${" + key + "}"
		if !strings.Contains(cmd, placeholder) {
			// Try with default syntax
			placeholder = "${" + key + ":-"
			if idx := strings.Index(cmd, placeholder); idx != -1 {
				// Find the closing brace
				end := strings.Index(cmd[idx:], "}")
				if end != -1 {
					// Check if custom var is provided
					if customVal, ok := vars[key]; ok && customVal != "" {
						// Replace entire ${VAR:-default} with custom value
						fullPlaceholder := cmd[idx : idx+end+1]
						cmd = strings.Replace(cmd, fullPlaceholder, customVal, -1)
					}
				}
			}
		} else {
			// Simple replacement
			if customVal, ok := vars[key]; ok && customVal != "" {
				cmd = strings.ReplaceAll(cmd, placeholder, customVal)
			} else {
				cmd = strings.ReplaceAll(cmd, placeholder, value)
			}
		}
	}

	// Apply custom vars
	for key, value := range vars {
		cmd = strings.ReplaceAll(cmd, "${"+key+"}", value)
		cmd = strings.ReplaceAll(cmd, "$"+key, value)
	}

	return cmd
}

// GetTemplateInfo returns information about a template.
func GetTemplateInfo(template models.ContainerHookTemplate) (TemplateInfo, bool) {
	info, ok := Templates[template]
	return info, ok
}

// ListTemplates returns all available templates.
func ListTemplates() []TemplateInfo {
	var templates []TemplateInfo
	for _, info := range Templates {
		templates = append(templates, info)
	}
	return templates
}

// ValidateTemplateVars checks if all required variables are provided.
func ValidateTemplateVars(template models.ContainerHookTemplate, vars map[string]string) error {
	info, ok := Templates[template]
	if !ok {
		return nil // No template means no required vars
	}

	for _, required := range info.RequiredVars {
		if val, ok := vars[required]; !ok || val == "" {
			return &MissingVariableError{Variable: required, Template: string(template)}
		}
	}

	return nil
}

// MissingVariableError is returned when a required template variable is missing.
type MissingVariableError struct {
	Variable string
	Template string
}

func (e *MissingVariableError) Error() string {
	return "missing required variable '" + e.Variable + "' for template '" + e.Template + "'"
}
