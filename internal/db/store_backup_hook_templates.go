package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Backup Hook Templates methods

// GetBackupHookTemplatesByOrgID returns all backup hook templates accessible to an organization.
// This includes org-specific templates and built-in templates.
func (db *DB) GetBackupHookTemplatesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.BackupHookTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, service_type, icon,
		       tags, variables, scripts, visibility, usage_count, created_at, updated_at
		FROM backup_hook_templates
		WHERE org_id = $1 OR visibility = 'built_in'
		ORDER BY visibility DESC, name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get backup hook templates: %w", err)
	}
	defer rows.Close()

	var templates []*models.BackupHookTemplate
	for rows.Next() {
		t, err := scanBackupHookTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup hook templates: %w", err)
	}
	return templates, nil
}

// GetBackupHookTemplateByID returns a backup hook template by ID.
func (db *DB) GetBackupHookTemplateByID(ctx context.Context, id uuid.UUID) (*models.BackupHookTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, service_type, icon,
		       tags, variables, scripts, visibility, usage_count, created_at, updated_at
		FROM backup_hook_templates
		WHERE id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("get backup hook template: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, fmt.Errorf("backup hook template not found")
	}
	return scanBackupHookTemplate(rows)
}

// GetBackupHookTemplatesByServiceType returns templates for a specific service type.
func (db *DB) GetBackupHookTemplatesByServiceType(ctx context.Context, orgID uuid.UUID, serviceType string) ([]*models.BackupHookTemplate, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, created_by_id, name, description, service_type, icon,
		       tags, variables, scripts, visibility, usage_count, created_at, updated_at
		FROM backup_hook_templates
		WHERE (org_id = $1 OR visibility = 'built_in') AND service_type = $2
		ORDER BY visibility DESC, name
	`, orgID, serviceType)
	if err != nil {
		return nil, fmt.Errorf("get backup hook templates by service type: %w", err)
	}
	defer rows.Close()

	var templates []*models.BackupHookTemplate
	for rows.Next() {
		t, err := scanBackupHookTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup hook templates: %w", err)
	}
	return templates, nil
}

// GetBackupHookTemplatesByVisibility returns templates with a specific visibility.
func (db *DB) GetBackupHookTemplatesByVisibility(ctx context.Context, orgID uuid.UUID, visibility models.BackupHookTemplateVisibility) ([]*models.BackupHookTemplate, error) {
	var rows interface {
		Next() bool
		Scan(dest ...any) error
		Close()
		Err() error
	}
	var err error

	if visibility == models.BackupHookTemplateVisibilityBuiltIn {
		rows, err = db.Pool.Query(ctx, `
			SELECT id, org_id, created_by_id, name, description, service_type, icon,
			       tags, variables, scripts, visibility, usage_count, created_at, updated_at
			FROM backup_hook_templates
			WHERE visibility = 'built_in'
			ORDER BY name
		`)
	} else {
		rows, err = db.Pool.Query(ctx, `
			SELECT id, org_id, created_by_id, name, description, service_type, icon,
			       tags, variables, scripts, visibility, usage_count, created_at, updated_at
			FROM backup_hook_templates
			WHERE org_id = $1 AND visibility = $2
			ORDER BY name
		`, orgID, string(visibility))
	}

	if err != nil {
		return nil, fmt.Errorf("get backup hook templates by visibility: %w", err)
	}
	defer rows.Close()

	var templates []*models.BackupHookTemplate
	for rows.Next() {
		t, err := scanBackupHookTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup hook templates: %w", err)
	}
	return templates, nil
}

// CreateBackupHookTemplate creates a new backup hook template.
func (db *DB) CreateBackupHookTemplate(ctx context.Context, template *models.BackupHookTemplate) error {
	tagsJSON, err := template.TagsJSON()
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	variablesJSON, err := template.VariablesJSON()
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	scriptsJSON, err := template.ScriptsJSON()
	if err != nil {
		return fmt.Errorf("marshal scripts: %w", err)
	}

	// Handle nil org_id and created_by_id for built-in templates
	var orgIDPtr, createdByIDPtr *uuid.UUID
	if template.OrgID != uuid.Nil {
		orgIDPtr = &template.OrgID
	}
	if template.CreatedByID != uuid.Nil {
		createdByIDPtr = &template.CreatedByID
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO backup_hook_templates (id, org_id, created_by_id, name, description,
		                                   service_type, icon, tags, variables, scripts,
		                                   visibility, usage_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, template.ID, orgIDPtr, createdByIDPtr, template.Name, template.Description,
		template.ServiceType, template.Icon, tagsJSON, variablesJSON, scriptsJSON,
		string(template.Visibility), template.UsageCount, template.CreatedAt, template.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create backup hook template: %w", err)
	}
	return nil
}

// UpdateBackupHookTemplate updates an existing backup hook template.
func (db *DB) UpdateBackupHookTemplate(ctx context.Context, template *models.BackupHookTemplate) error {
	template.UpdatedAt = time.Now()

	tagsJSON, err := template.TagsJSON()
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	variablesJSON, err := template.VariablesJSON()
	if err != nil {
		return fmt.Errorf("marshal variables: %w", err)
	}
	scriptsJSON, err := template.ScriptsJSON()
	if err != nil {
		return fmt.Errorf("marshal scripts: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE backup_hook_templates
		SET name = $2, description = $3, service_type = $4, icon = $5,
		    tags = $6, variables = $7, scripts = $8, visibility = $9, updated_at = $10
		WHERE id = $1
	`, template.ID, template.Name, template.Description, template.ServiceType, template.Icon,
		tagsJSON, variablesJSON, scriptsJSON, string(template.Visibility), template.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update backup hook template: %w", err)
	}
	return nil
}

// DeleteBackupHookTemplate deletes a backup hook template.
func (db *DB) DeleteBackupHookTemplate(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM backup_hook_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete backup hook template: %w", err)
	}
	return nil
}

// IncrementBackupHookTemplateUsage increments the usage count for a template.
func (db *DB) IncrementBackupHookTemplateUsage(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE backup_hook_templates
		SET usage_count = usage_count + 1
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("increment backup hook template usage: %w", err)
	}
	return nil
}

// scanBackupHookTemplate scans a row into a BackupHookTemplate.
func scanBackupHookTemplate(rows interface{ Scan(dest ...any) error }) (*models.BackupHookTemplate, error) {
	var t models.BackupHookTemplate
	var orgID, createdByID *uuid.UUID
	var description, icon *string
	var visibilityStr string
	var tagsBytes, variablesBytes, scriptsBytes []byte

	err := rows.Scan(
		&t.ID, &orgID, &createdByID, &t.Name, &description, &t.ServiceType, &icon,
		&tagsBytes, &variablesBytes, &scriptsBytes, &visibilityStr, &t.UsageCount,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan backup hook template: %w", err)
	}

	if orgID != nil {
		t.OrgID = *orgID
	}
	if createdByID != nil {
		t.CreatedByID = *createdByID
	}
	if description != nil {
		t.Description = *description
	}
	if icon != nil {
		t.Icon = *icon
	}
	t.Visibility = models.BackupHookTemplateVisibility(visibilityStr)

	if err := t.SetTags(tagsBytes); err != nil {
		return nil, fmt.Errorf("parse tags: %w", err)
	}
	if err := t.SetVariables(variablesBytes); err != nil {
		return nil, fmt.Errorf("parse variables: %w", err)
	}
	if err := t.SetScripts(scriptsBytes); err != nil {
		return nil, fmt.Errorf("parse scripts: %w", err)
	}

	return &t, nil
}
