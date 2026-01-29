package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Test Restore Settings methods

// CreateTestRestoreSettings creates new test restore settings for a repository.
func (db *DB) CreateTestRestoreSettings(ctx context.Context, settings *models.TestRestoreSettings) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO test_restore_settings (
			id, repository_id, enabled, frequency, cron_expression,
			sample_percentage, last_run_at, last_run_status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, settings.ID, settings.RepositoryID, settings.Enabled, string(settings.Frequency),
		settings.CronExpression, settings.SamplePercentage, settings.LastRunAt,
		settings.LastRunStatus, settings.CreatedAt, settings.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create test restore settings: %w", err)
	}
	return nil
}

// UpdateTestRestoreSettings updates existing test restore settings.
func (db *DB) UpdateTestRestoreSettings(ctx context.Context, settings *models.TestRestoreSettings) error {
	settings.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE test_restore_settings SET
			enabled = $2,
			frequency = $3,
			cron_expression = $4,
			sample_percentage = $5,
			last_run_at = $6,
			last_run_status = $7,
			updated_at = $8
		WHERE id = $1
	`, settings.ID, settings.Enabled, string(settings.Frequency), settings.CronExpression,
		settings.SamplePercentage, settings.LastRunAt, settings.LastRunStatus, settings.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update test restore settings: %w", err)
	}
	return nil
}

// GetTestRestoreSettingsByID returns test restore settings by ID.
func (db *DB) GetTestRestoreSettingsByID(ctx context.Context, id uuid.UUID) (*models.TestRestoreSettings, error) {
	var settings models.TestRestoreSettings
	var frequencyStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, enabled, frequency, cron_expression,
		       sample_percentage, last_run_at, last_run_status, created_at, updated_at
		FROM test_restore_settings
		WHERE id = $1
	`, id).Scan(
		&settings.ID, &settings.RepositoryID, &settings.Enabled, &frequencyStr,
		&settings.CronExpression, &settings.SamplePercentage, &settings.LastRunAt,
		&settings.LastRunStatus, &settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get test restore settings: %w", err)
	}
	settings.Frequency = models.TestRestoreFrequency(frequencyStr)
	return &settings, nil
}

// GetTestRestoreSettingsByRepoID returns test restore settings for a repository.
func (db *DB) GetTestRestoreSettingsByRepoID(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreSettings, error) {
	var settings models.TestRestoreSettings
	var frequencyStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, enabled, frequency, cron_expression,
		       sample_percentage, last_run_at, last_run_status, created_at, updated_at
		FROM test_restore_settings
		WHERE repository_id = $1
	`, repoID).Scan(
		&settings.ID, &settings.RepositoryID, &settings.Enabled, &frequencyStr,
		&settings.CronExpression, &settings.SamplePercentage, &settings.LastRunAt,
		&settings.LastRunStatus, &settings.CreatedAt, &settings.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get test restore settings by repo: %w", err)
	}
	settings.Frequency = models.TestRestoreFrequency(frequencyStr)
	return &settings, nil
}

// GetEnabledTestRestoreSettings returns all enabled test restore settings.
func (db *DB) GetEnabledTestRestoreSettings(ctx context.Context) ([]*models.TestRestoreSettings, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, enabled, frequency, cron_expression,
		       sample_percentage, last_run_at, last_run_status, created_at, updated_at
		FROM test_restore_settings
		WHERE enabled = true
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("list enabled test restore settings: %w", err)
	}
	defer rows.Close()

	return scanTestRestoreSettings(rows)
}

// DeleteTestRestoreSettings deletes test restore settings by ID.
func (db *DB) DeleteTestRestoreSettings(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM test_restore_settings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete test restore settings: %w", err)
	}
	return nil
}

// scanTestRestoreSettings is a helper to scan multiple test restore settings.
func scanTestRestoreSettings(rows pgx.Rows) ([]*models.TestRestoreSettings, error) {
	var settings []*models.TestRestoreSettings
	for rows.Next() {
		var s models.TestRestoreSettings
		var frequencyStr string
		err := rows.Scan(
			&s.ID, &s.RepositoryID, &s.Enabled, &frequencyStr,
			&s.CronExpression, &s.SamplePercentage, &s.LastRunAt,
			&s.LastRunStatus, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan test restore settings: %w", err)
		}
		s.Frequency = models.TestRestoreFrequency(frequencyStr)
		settings = append(settings, &s)
	}
	return settings, nil
}

// Test Restore Result methods

// CreateTestRestoreResult creates a new test restore result record.
func (db *DB) CreateTestRestoreResult(ctx context.Context, result *models.TestRestoreResult) error {
	detailsBytes, err := result.DetailsJSON()
	if err != nil {
		return fmt.Errorf("marshal test restore details: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO test_restore_results (
			id, repository_id, snapshot_id, sample_percentage, started_at,
			completed_at, status, duration_ms, files_restored, files_verified,
			bytes_restored, error_message, details, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, result.ID, result.RepositoryID, result.SnapshotID, result.SamplePercentage,
		result.StartedAt, result.CompletedAt, string(result.Status), result.DurationMs,
		result.FilesRestored, result.FilesVerified, result.BytesRestored,
		result.ErrorMessage, detailsBytes, result.CreatedAt)
	if err != nil {
		return fmt.Errorf("create test restore result: %w", err)
	}
	return nil
}

// UpdateTestRestoreResult updates an existing test restore result record.
func (db *DB) UpdateTestRestoreResult(ctx context.Context, result *models.TestRestoreResult) error {
	detailsBytes, err := result.DetailsJSON()
	if err != nil {
		return fmt.Errorf("marshal test restore details: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE test_restore_results SET
			snapshot_id = $2,
			completed_at = $3,
			status = $4,
			duration_ms = $5,
			files_restored = $6,
			files_verified = $7,
			bytes_restored = $8,
			error_message = $9,
			details = $10
		WHERE id = $1
	`, result.ID, result.SnapshotID, result.CompletedAt, string(result.Status),
		result.DurationMs, result.FilesRestored, result.FilesVerified,
		result.BytesRestored, result.ErrorMessage, detailsBytes)
	if err != nil {
		return fmt.Errorf("update test restore result: %w", err)
	}
	return nil
}

// GetTestRestoreResultByID returns a test restore result by ID.
func (db *DB) GetTestRestoreResultByID(ctx context.Context, id uuid.UUID) (*models.TestRestoreResult, error) {
	var result models.TestRestoreResult
	var statusStr string
	var detailsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, snapshot_id, sample_percentage, started_at,
		       completed_at, status, duration_ms, files_restored, files_verified,
		       bytes_restored, error_message, details, created_at
		FROM test_restore_results
		WHERE id = $1
	`, id).Scan(
		&result.ID, &result.RepositoryID, &result.SnapshotID, &result.SamplePercentage,
		&result.StartedAt, &result.CompletedAt, &statusStr, &result.DurationMs,
		&result.FilesRestored, &result.FilesVerified, &result.BytesRestored,
		&result.ErrorMessage, &detailsBytes, &result.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get test restore result: %w", err)
	}
	result.Status = models.TestRestoreResultStatus(statusStr)
	if err := result.SetDetails(detailsBytes); err != nil {
		db.logger.Warn().Err(err).Str("result_id", result.ID.String()).Msg("failed to parse test restore details")
	}
	return &result, nil
}

// GetTestRestoreResultsByRepoID returns test restore results for a repository.
func (db *DB) GetTestRestoreResultsByRepoID(ctx context.Context, repoID uuid.UUID, limit int) ([]*models.TestRestoreResult, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, repository_id, snapshot_id, sample_percentage, started_at,
		       completed_at, status, duration_ms, files_restored, files_verified,
		       bytes_restored, error_message, details, created_at
		FROM test_restore_results
		WHERE repository_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, repoID, limit)
	if err != nil {
		return nil, fmt.Errorf("list test restore results: %w", err)
	}
	defer rows.Close()

	return db.scanTestRestoreResults(rows)
}

// GetLatestTestRestoreResultByRepoID returns the most recent test restore result for a repository.
func (db *DB) GetLatestTestRestoreResultByRepoID(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreResult, error) {
	var result models.TestRestoreResult
	var statusStr string
	var detailsBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, snapshot_id, sample_percentage, started_at,
		       completed_at, status, duration_ms, files_restored, files_verified,
		       bytes_restored, error_message, details, created_at
		FROM test_restore_results
		WHERE repository_id = $1
		ORDER BY started_at DESC
		LIMIT 1
	`, repoID).Scan(
		&result.ID, &result.RepositoryID, &result.SnapshotID, &result.SamplePercentage,
		&result.StartedAt, &result.CompletedAt, &statusStr, &result.DurationMs,
		&result.FilesRestored, &result.FilesVerified, &result.BytesRestored,
		&result.ErrorMessage, &detailsBytes, &result.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest test restore result: %w", err)
	}
	result.Status = models.TestRestoreResultStatus(statusStr)
	if err := result.SetDetails(detailsBytes); err != nil {
		db.logger.Warn().Err(err).Str("result_id", result.ID.String()).Msg("failed to parse test restore details")
	}
	return &result, nil
}

// GetConsecutiveFailedTestRestores returns the count of consecutive failed test restores.
func (db *DB) GetConsecutiveFailedTestRestores(ctx context.Context, repoID uuid.UUID) (int, error) {
	// Count consecutive failures from the most recent result backwards
	rows, err := db.Pool.Query(ctx, `
		SELECT status
		FROM test_restore_results
		WHERE repository_id = $1 AND status IN ('passed', 'failed')
		ORDER BY started_at DESC
		LIMIT 10
	`, repoID)
	if err != nil {
		return 0, fmt.Errorf("get consecutive failed test restores: %w", err)
	}
	defer rows.Close()

	consecutiveFails := 0
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return 0, fmt.Errorf("scan test restore status: %w", err)
		}
		if status == string(models.TestRestoreStatusFailed) {
			consecutiveFails++
		} else {
			break
		}
	}

	return consecutiveFails, nil
}

// scanTestRestoreResults is a helper to scan multiple test restore results.
func (db *DB) scanTestRestoreResults(rows pgx.Rows) ([]*models.TestRestoreResult, error) {
	var results []*models.TestRestoreResult
	for rows.Next() {
		var result models.TestRestoreResult
		var statusStr string
		var detailsBytes []byte
		err := rows.Scan(
			&result.ID, &result.RepositoryID, &result.SnapshotID, &result.SamplePercentage,
			&result.StartedAt, &result.CompletedAt, &statusStr, &result.DurationMs,
			&result.FilesRestored, &result.FilesVerified, &result.BytesRestored,
			&result.ErrorMessage, &detailsBytes, &result.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan test restore result: %w", err)
		}
		result.Status = models.TestRestoreResultStatus(statusStr)
		if err := result.SetDetails(detailsBytes); err != nil {
			db.logger.Warn().Err(err).Str("result_id", result.ID.String()).Msg("failed to parse test restore details")
		}
		results = append(results, &result)
	}
	return results, nil
}

// GetTestRestoreSummaryByOrgID returns test restore summary statistics for an organization.
func (db *DB) GetTestRestoreSummaryByOrgID(ctx context.Context, orgID uuid.UUID) (*TestRestoreSummary, error) {
	summary := &TestRestoreSummary{}

	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(DISTINCT trs.repository_id) as repositories_with_testing,
			COUNT(CASE WHEN trr.status = 'passed' THEN 1 END) as total_passed,
			COUNT(CASE WHEN trr.status = 'failed' THEN 1 END) as total_failed,
			COALESCE(MAX(trr.started_at), NOW()) as last_test_at
		FROM test_restore_settings trs
		INNER JOIN repositories r ON r.id = trs.repository_id
		LEFT JOIN test_restore_results trr ON trr.repository_id = trs.repository_id
		WHERE r.org_id = $1 AND trs.enabled = true
	`, orgID).Scan(
		&summary.RepositoriesWithTesting,
		&summary.TotalPassed,
		&summary.TotalFailed,
		&summary.LastTestAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get test restore summary: %w", err)
	}

	// Get repositories needing attention (failed last test)
	rows, err := db.Pool.Query(ctx, `
		WITH latest_results AS (
			SELECT DISTINCT ON (repository_id)
				repository_id, status
			FROM test_restore_results
			ORDER BY repository_id, started_at DESC
		)
		SELECT COUNT(*)
		FROM latest_results lr
		INNER JOIN repositories r ON r.id = lr.repository_id
		WHERE r.org_id = $1 AND lr.status = 'failed'
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get repositories needing attention: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		if err := rows.Scan(&summary.RepositoriesNeedingAttention); err != nil {
			return nil, fmt.Errorf("scan repositories needing attention: %w", err)
		}
	}

	return summary, nil
}

// TestRestoreSummary contains summary statistics for test restores.
type TestRestoreSummary struct {
	RepositoriesWithTesting       int       `json:"repositories_with_testing"`
	RepositoriesNeedingAttention  int       `json:"repositories_needing_attention"`
	TotalPassed                   int       `json:"total_passed"`
	TotalFailed                   int       `json:"total_failed"`
	LastTestAt                    time.Time `json:"last_test_at"`
}
