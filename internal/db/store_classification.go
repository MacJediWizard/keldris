package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/classification"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Path Classification Rules

// GetPathClassificationRulesByOrgID returns all classification rules for an organization.
func (db *DB) GetPathClassificationRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.PathClassificationRule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, pattern, level, data_types, description, is_builtin, priority, enabled, created_at, updated_at
		FROM path_classification_rules
		WHERE org_id = $1
		ORDER BY priority DESC, created_at ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list path classification rules: %w", err)
	}
	defer rows.Close()

	var rules []*models.PathClassificationRule
	for rows.Next() {
		var r models.PathClassificationRule
		var levelStr string
		var dataTypesBytes []byte
		err := rows.Scan(
			&r.ID, &r.OrgID, &r.Pattern, &levelStr, &dataTypesBytes,
			&r.Description, &r.IsBuiltin, &r.Priority, &r.Enabled,
			&r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan path classification rule: %w", err)
		}
		r.Level = classification.Level(levelStr)
		if err := r.SetDataTypes(dataTypesBytes); err != nil {
			db.logger.Warn().Err(err).Str("rule_id", r.ID.String()).Msg("failed to parse data types")
		}
		rules = append(rules, &r)
	}

	return rules, nil
}

// GetPathClassificationRuleByID returns a classification rule by ID.
func (db *DB) GetPathClassificationRuleByID(ctx context.Context, id uuid.UUID) (*models.PathClassificationRule, error) {
	var r models.PathClassificationRule
	var levelStr string
	var dataTypesBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, pattern, level, data_types, description, is_builtin, priority, enabled, created_at, updated_at
		FROM path_classification_rules
		WHERE id = $1
	`, id).Scan(
		&r.ID, &r.OrgID, &r.Pattern, &levelStr, &dataTypesBytes,
		&r.Description, &r.IsBuiltin, &r.Priority, &r.Enabled,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get path classification rule: %w", err)
	}
	r.Level = classification.Level(levelStr)
	if err := r.SetDataTypes(dataTypesBytes); err != nil {
		db.logger.Warn().Err(err).Str("rule_id", r.ID.String()).Msg("failed to parse data types")
	}
	return &r, nil
}

// CreatePathClassificationRule creates a new path classification rule.
func (db *DB) CreatePathClassificationRule(ctx context.Context, rule *models.PathClassificationRule) error {
	dataTypesJSON, err := rule.DataTypesJSON()
	if err != nil {
		return fmt.Errorf("marshal data types: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO path_classification_rules (id, org_id, pattern, level, data_types, description, is_builtin, priority, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, rule.ID, rule.OrgID, rule.Pattern, string(rule.Level), dataTypesJSON,
		rule.Description, rule.IsBuiltin, rule.Priority, rule.Enabled,
		rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create path classification rule: %w", err)
	}
	return nil
}

// UpdatePathClassificationRule updates an existing path classification rule.
func (db *DB) UpdatePathClassificationRule(ctx context.Context, rule *models.PathClassificationRule) error {
	dataTypesJSON, err := rule.DataTypesJSON()
	if err != nil {
		return fmt.Errorf("marshal data types: %w", err)
	}

	rule.UpdatedAt = time.Now()
	_, err = db.Pool.Exec(ctx, `
		UPDATE path_classification_rules
		SET pattern = $2, level = $3, data_types = $4, description = $5, priority = $6, enabled = $7, updated_at = $8
		WHERE id = $1
	`, rule.ID, rule.Pattern, string(rule.Level), dataTypesJSON,
		rule.Description, rule.Priority, rule.Enabled, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update path classification rule: %w", err)
	}
	return nil
}

// DeletePathClassificationRule deletes a path classification rule.
func (db *DB) DeletePathClassificationRule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM path_classification_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete path classification rule: %w", err)
	}
	return nil
}

// Schedule Classifications

// GetScheduleClassification returns the classification for a schedule.
func (db *DB) GetScheduleClassification(ctx context.Context, scheduleID uuid.UUID) (*models.ScheduleClassification, error) {
	var c models.ScheduleClassification
	var levelStr string
	var dataTypesBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, schedule_id, level, data_types, auto_classified, classified_at, created_at, updated_at
		FROM schedule_classifications
		WHERE schedule_id = $1
	`, scheduleID).Scan(
		&c.ID, &c.ScheduleID, &levelStr, &dataTypesBytes,
		&c.AutoClassified, &c.ClassifiedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get schedule classification: %w", err)
	}
	c.Level = classification.Level(levelStr)
	if err := c.SetDataTypes(dataTypesBytes); err != nil {
		db.logger.Warn().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to parse data types")
	}
	return &c, nil
}

// SetScheduleClassification creates or updates a schedule's classification.
func (db *DB) SetScheduleClassification(ctx context.Context, c *models.ScheduleClassification) error {
	dataTypesJSON, err := c.DataTypesJSON()
	if err != nil {
		return fmt.Errorf("marshal data types: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO schedule_classifications (id, schedule_id, level, data_types, auto_classified, classified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (schedule_id)
		DO UPDATE SET level = $3, data_types = $4, auto_classified = $5, classified_at = $6, updated_at = $8
	`, c.ID, c.ScheduleID, string(c.Level), dataTypesJSON,
		c.AutoClassified, c.ClassifiedAt, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("set schedule classification: %w", err)
	}
	return nil
}

// UpdateScheduleClassificationLevel updates the classification level on the schedule table directly.
func (db *DB) UpdateScheduleClassificationLevel(ctx context.Context, scheduleID uuid.UUID, level string, dataTypes []string) error {
	dataTypesJSON := []byte(`["general"]`)
	if len(dataTypes) > 0 {
		var err error
		dataTypesJSON, err = jsonMarshalHelper(dataTypes)
		if err != nil {
			return fmt.Errorf("marshal data types: %w", err)
		}
	}

	_, err := db.Pool.Exec(ctx, `
		UPDATE schedules
		SET classification_level = $2, classification_data_types = $3, updated_at = NOW()
		WHERE id = $1
	`, scheduleID, level, dataTypesJSON)
	if err != nil {
		return fmt.Errorf("update schedule classification level: %w", err)
	}
	return nil
}

// GetSchedulesByClassificationLevel returns schedules with a specific classification level.
func (db *DB) GetSchedulesByClassificationLevel(ctx context.Context, orgID uuid.UUID, level string) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.cron_expression,
		       s.paths, s.excludes, s.retention_policy, s.bandwidth_limit_kbps,
		       s.backup_window_start, s.backup_window_end, s.excluded_hours,
		       s.compression_level, s.on_mount_unavailable, s.classification_level,
		       s.classification_data_types, s.enabled, s.created_at, s.updated_at
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND s.classification_level = $2
		ORDER BY s.name
	`, orgID, level)
	if err != nil {
		return nil, fmt.Errorf("list schedules by classification: %w", err)
	}
	defer rows.Close()

	return db.scanSchedules(rows)
}

// GetSchedulesByOrgID returns all schedules for an organization.
func (db *DB) GetSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id, s.agent_id, s.agent_group_id, s.policy_id, s.name, s.cron_expression,
		       s.paths, s.excludes, s.retention_policy, s.bandwidth_limit_kbps,
		       s.backup_window_start, s.backup_window_end, s.excluded_hours,
		       s.compression_level, s.on_mount_unavailable, s.classification_level,
		       s.classification_data_types, s.enabled, s.created_at, s.updated_at
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
		ORDER BY s.name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list schedules by org: %w", err)
	}
	defer rows.Close()

	return db.scanSchedules(rows)
}

// scanSchedules scans schedule rows into a slice of Schedule pointers.
func (db *DB) scanSchedules(rows interface{ Next() bool; Scan(dest ...interface{}) error }) ([]*models.Schedule, error) {
	var schedules []*models.Schedule
	for rows.Next() {
		var s models.Schedule
		var pathsBytes, excludesBytes, retentionBytes, excludedHoursBytes, classDataTypesBytes []byte
		var windowStart, windowEnd, compressionLevel, onMountUnavailable *string

		err := rows.Scan(
			&s.ID, &s.AgentID, &s.AgentGroupID, &s.PolicyID, &s.Name, &s.CronExpression,
			&pathsBytes, &excludesBytes, &retentionBytes, &s.BandwidthLimitKB,
			&windowStart, &windowEnd, &excludedHoursBytes,
			&compressionLevel, &onMountUnavailable, &s.ClassificationLevel,
			&classDataTypesBytes, &s.Enabled, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}

		if err := s.SetPaths(pathsBytes); err != nil {
			db.logger.Warn().Err(err).Str("schedule_id", s.ID.String()).Msg("failed to parse paths")
		}
		if err := s.SetExcludes(excludesBytes); err != nil {
			db.logger.Warn().Err(err).Str("schedule_id", s.ID.String()).Msg("failed to parse excludes")
		}
		if err := s.SetRetentionPolicy(retentionBytes); err != nil {
			db.logger.Warn().Err(err).Str("schedule_id", s.ID.String()).Msg("failed to parse retention policy")
		}
		if err := s.SetExcludedHours(excludedHoursBytes); err != nil {
			db.logger.Warn().Err(err).Str("schedule_id", s.ID.String()).Msg("failed to parse excluded hours")
		}
		if err := s.SetClassificationDataTypes(classDataTypesBytes); err != nil {
			db.logger.Warn().Err(err).Str("schedule_id", s.ID.String()).Msg("failed to parse classification data types")
		}

		s.SetBackupWindow(windowStart, windowEnd)
		s.CompressionLevel = compressionLevel
		if onMountUnavailable != nil {
			s.OnMountUnavailable = models.MountBehavior(*onMountUnavailable)
		}

		schedules = append(schedules, &s)
	}
	return schedules, nil
}

// Backup Classifications

// GetBackupClassification returns the classification for a backup.
func (db *DB) GetBackupClassification(ctx context.Context, backupID uuid.UUID) (*models.BackupClassification, error) {
	var c models.BackupClassification
	var levelStr string
	var dataTypesBytes, pathsBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, backup_id, schedule_id, level, data_types, paths_classified, created_at
		FROM backup_classifications
		WHERE backup_id = $1
	`, backupID).Scan(
		&c.ID, &c.BackupID, &c.ScheduleID, &levelStr, &dataTypesBytes, &pathsBytes, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get backup classification: %w", err)
	}
	c.Level = classification.Level(levelStr)
	if err := c.SetDataTypes(dataTypesBytes); err != nil {
		db.logger.Warn().Err(err).Str("backup_id", backupID.String()).Msg("failed to parse data types")
	}
	if err := c.SetPathsClassified(pathsBytes); err != nil {
		db.logger.Warn().Err(err).Str("backup_id", backupID.String()).Msg("failed to parse paths classified")
	}
	return &c, nil
}

// GetBackupsByClassificationLevel returns backups with a specific classification level.
func (db *DB) GetBackupsByClassificationLevel(ctx context.Context, orgID uuid.UUID, level string, limit int) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id,
		       b.started_at, b.completed_at, b.status, b.size_bytes,
		       b.files_new, b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error,
		       b.pre_script_output, b.pre_script_error, b.post_script_output, b.post_script_error,
		       b.classification_level, b.classification_data_types, b.created_at
		FROM backups b
		JOIN agents a ON b.agent_id = a.id
		WHERE a.org_id = $1 AND b.classification_level = $2
		ORDER BY b.started_at DESC
		LIMIT $3
	`, orgID, level, limit)
	if err != nil {
		return nil, fmt.Errorf("list backups by classification: %w", err)
	}
	defer rows.Close()

	var backups []*models.Backup
	for rows.Next() {
		var b models.Backup
		var statusStr string
		var classDataTypesBytes []byte

		err := rows.Scan(
			&b.ID, &b.ScheduleID, &b.AgentID, &b.RepositoryID, &b.SnapshotID,
			&b.StartedAt, &b.CompletedAt, &statusStr, &b.SizeBytes,
			&b.FilesNew, &b.FilesChanged, &b.ErrorMessage,
			&b.RetentionApplied, &b.SnapshotsRemoved, &b.SnapshotsKept, &b.RetentionError,
			&b.PreScriptOutput, &b.PreScriptError, &b.PostScriptOutput, &b.PostScriptError,
			&b.ClassificationLevel, &classDataTypesBytes, &b.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}
		b.Status = models.BackupStatus(statusStr)
		if err := b.SetClassificationDataTypes(classDataTypesBytes); err != nil {
			db.logger.Warn().Err(err).Str("backup_id", b.ID.String()).Msg("failed to parse classification data types")
		}
		backups = append(backups, &b)
	}
	return backups, nil
}

// Classification Summary

// GetClassificationSummary returns aggregated classification statistics for an organization.
func (db *DB) GetClassificationSummary(ctx context.Context, orgID uuid.UUID) (*models.ClassificationSummary, error) {
	summary := &models.ClassificationSummary{
		ByLevel:    make(map[string]int),
		ByDataType: make(map[string]int),
	}

	// Count schedules by classification level
	rows, err := db.Pool.Query(ctx, `
		SELECT COALESCE(s.classification_level, 'public') as level, COUNT(*) as count
		FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
		GROUP BY s.classification_level
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("count schedules by classification: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var level string
		var count int
		if err := rows.Scan(&level, &count); err != nil {
			return nil, fmt.Errorf("scan schedule count: %w", err)
		}
		summary.ByLevel[level] = count
		summary.TotalSchedules += count

		switch level {
		case "public":
			summary.PublicCount = count
		case "internal":
			summary.InternalCount = count
		case "confidential":
			summary.ConfidentialCount = count
		case "restricted":
			summary.RestrictedCount = count
		}
	}

	// Count backups
	err = db.Pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM backups b
		JOIN agents a ON b.agent_id = a.id
		WHERE a.org_id = $1
	`, orgID).Scan(&summary.TotalBackups)
	if err != nil {
		return nil, fmt.Errorf("count backups: %w", err)
	}

	return summary, nil
}

// jsonMarshalHelper marshals a value to JSON.
func jsonMarshalHelper(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
