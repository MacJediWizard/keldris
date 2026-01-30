package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// GetContainerBackupHooksByScheduleID returns all container backup hooks for a schedule.
func (db *DB) GetContainerBackupHooksByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.ContainerBackupHook, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, container_name, type, template, command, working_dir, user_name,
		       timeout_seconds, fail_on_error, enabled, description, template_vars, created_at, updated_at
		FROM container_backup_hooks
		WHERE schedule_id = $1
		ORDER BY container_name, type
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list container backup hooks by schedule: %w", err)
	}
	defer rows.Close()

	return scanContainerBackupHooks(rows)
}

// GetContainerBackupHookByID returns a container backup hook by ID.
func (db *DB) GetContainerBackupHookByID(ctx context.Context, id uuid.UUID) (*models.ContainerBackupHook, error) {
	var h models.ContainerBackupHook
	var typeStr, templateStr string
	var workingDir, userName, description *string
	var templateVarsBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, container_name, type, template, command, working_dir, user_name,
		       timeout_seconds, fail_on_error, enabled, description, template_vars, created_at, updated_at
		FROM container_backup_hooks
		WHERE id = $1
	`, id).Scan(
		&h.ID, &h.ScheduleID, &h.ContainerName, &typeStr, &templateStr, &h.Command,
		&workingDir, &userName, &h.TimeoutSeconds, &h.FailOnError, &h.Enabled,
		&description, &templateVarsBytes, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get container backup hook: %w", err)
	}

	h.Type = models.ContainerHookType(typeStr)
	h.Template = models.ContainerHookTemplate(templateStr)
	if workingDir != nil {
		h.WorkingDir = *workingDir
	}
	if userName != nil {
		h.User = *userName
	}
	if description != nil {
		h.Description = *description
	}
	if len(templateVarsBytes) > 0 {
		if err := json.Unmarshal(templateVarsBytes, &h.TemplateVars); err != nil {
			return nil, fmt.Errorf("parse template vars: %w", err)
		}
	}

	return &h, nil
}

// GetEnabledContainerBackupHooksByScheduleID returns all enabled container backup hooks for a schedule.
func (db *DB) GetEnabledContainerBackupHooksByScheduleID(ctx context.Context, scheduleID uuid.UUID) ([]*models.ContainerBackupHook, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, container_name, type, template, command, working_dir, user_name,
		       timeout_seconds, fail_on_error, enabled, description, template_vars, created_at, updated_at
		FROM container_backup_hooks
		WHERE schedule_id = $1 AND enabled = true
		ORDER BY container_name, type
	`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("list enabled container backup hooks by schedule: %w", err)
	}
	defer rows.Close()

	return scanContainerBackupHooks(rows)
}

// CreateContainerBackupHook creates a new container backup hook.
func (db *DB) CreateContainerBackupHook(ctx context.Context, hook *models.ContainerBackupHook) error {
	templateVarsBytes, err := json.Marshal(hook.TemplateVars)
	if err != nil {
		return fmt.Errorf("marshal template vars: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO container_backup_hooks (id, schedule_id, container_name, type, template, command,
		            working_dir, user_name, timeout_seconds, fail_on_error, enabled, description,
		            template_vars, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, hook.ID, hook.ScheduleID, hook.ContainerName, string(hook.Type), string(hook.Template), hook.Command,
		nullableString(hook.WorkingDir), nullableString(hook.User), hook.TimeoutSeconds, hook.FailOnError,
		hook.Enabled, nullableString(hook.Description), templateVarsBytes, hook.CreatedAt, hook.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create container backup hook: %w", err)
	}
	return nil
}

// UpdateContainerBackupHook updates an existing container backup hook.
func (db *DB) UpdateContainerBackupHook(ctx context.Context, hook *models.ContainerBackupHook) error {
	hook.UpdatedAt = time.Now()

	templateVarsBytes, err := json.Marshal(hook.TemplateVars)
	if err != nil {
		return fmt.Errorf("marshal template vars: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE container_backup_hooks
		SET container_name = $2, command = $3, working_dir = $4, user_name = $5,
		    timeout_seconds = $6, fail_on_error = $7, enabled = $8, description = $9,
		    template_vars = $10, updated_at = $11
		WHERE id = $1
	`, hook.ID, hook.ContainerName, hook.Command, nullableString(hook.WorkingDir),
		nullableString(hook.User), hook.TimeoutSeconds, hook.FailOnError, hook.Enabled,
		nullableString(hook.Description), templateVarsBytes, hook.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update container backup hook: %w", err)
	}
	return nil
}

// DeleteContainerBackupHook deletes a container backup hook by ID.
func (db *DB) DeleteContainerBackupHook(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM container_backup_hooks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete container backup hook: %w", err)
	}
	return nil
}

// CreateContainerHookExecution creates a new container hook execution record.
func (db *DB) CreateContainerHookExecution(ctx context.Context, exec *models.ContainerHookExecution) error {
	durationMs := int(exec.Duration.Milliseconds())

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO container_hook_executions (hook_id, backup_id, container_name, type, command,
		            output, exit_code, error, duration_ms, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, exec.HookID, exec.BackupID, exec.Container, string(exec.Type), exec.Command,
		exec.Output, exec.ExitCode, nullableString(exec.Error), durationMs,
		exec.StartedAt, exec.CompletedAt)
	if err != nil {
		return fmt.Errorf("create container hook execution: %w", err)
	}
	return nil
}

// GetContainerHookExecutionsByBackupID returns all hook executions for a backup.
func (db *DB) GetContainerHookExecutionsByBackupID(ctx context.Context, backupID uuid.UUID) ([]*models.ContainerHookExecution, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT hook_id, backup_id, container_name, type, command, output, exit_code, error,
		       duration_ms, started_at, completed_at
		FROM container_hook_executions
		WHERE backup_id = $1
		ORDER BY started_at
	`, backupID)
	if err != nil {
		return nil, fmt.Errorf("list container hook executions by backup: %w", err)
	}
	defer rows.Close()

	var executions []*models.ContainerHookExecution
	for rows.Next() {
		var e models.ContainerHookExecution
		var typeStr string
		var errStr *string
		var durationMs int

		if err := rows.Scan(
			&e.HookID, &e.BackupID, &e.Container, &typeStr, &e.Command,
			&e.Output, &e.ExitCode, &errStr, &durationMs, &e.StartedAt, &e.CompletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan container hook execution: %w", err)
		}

		e.Type = models.ContainerHookType(typeStr)
		e.Duration = time.Duration(durationMs) * time.Millisecond
		if errStr != nil {
			e.Error = *errStr
		}
		executions = append(executions, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate container hook executions: %w", err)
	}

	return executions, nil
}

// scanContainerBackupHooks scans multiple container backup hook rows.
func scanContainerBackupHooks(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.ContainerBackupHook, error) {
	var hooks []*models.ContainerBackupHook
	for rows.Next() {
		var h models.ContainerBackupHook
		var typeStr, templateStr string
		var workingDir, userName, description *string
		var templateVarsBytes []byte

		err := rows.Scan(
			&h.ID, &h.ScheduleID, &h.ContainerName, &typeStr, &templateStr, &h.Command,
			&workingDir, &userName, &h.TimeoutSeconds, &h.FailOnError, &h.Enabled,
			&description, &templateVarsBytes, &h.CreatedAt, &h.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan container backup hook: %w", err)
		}

		h.Type = models.ContainerHookType(typeStr)
		h.Template = models.ContainerHookTemplate(templateStr)
		if workingDir != nil {
			h.WorkingDir = *workingDir
		}
		if userName != nil {
			h.User = *userName
		}
		if description != nil {
			h.Description = *description
		}
		if len(templateVarsBytes) > 0 {
			if err := json.Unmarshal(templateVarsBytes, &h.TemplateVars); err != nil {
				return nil, fmt.Errorf("parse template vars: %w", err)
			}
		}

		hooks = append(hooks, &h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate container backup hooks: %w", err)
	}

	return hooks, nil
}

// nullableString returns a pointer to a string if non-empty, nil otherwise.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
