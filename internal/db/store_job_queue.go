package db

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Job Queue Methods

// CreateJob creates a new job in the queue.
func (db *DB) CreateJob(ctx context.Context, job *models.Job) error {
	payloadBytes, err := job.PayloadJSON()
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO job_queue (
			id, org_id, job_type, priority, status, payload,
			retry_count, max_retries, next_retry_at, error_message, last_error_at,
			created_at, started_at, completed_at,
			agent_id, repository_id, schedule_id
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17
		)
	`, job.ID, job.OrgID, job.JobType, job.Priority, job.Status, payloadBytes,
		job.RetryCount, job.MaxRetries, job.NextRetryAt, job.ErrorMessage, job.LastErrorAt,
		job.CreatedAt, job.StartedAt, job.CompletedAt,
		job.AgentID, job.RepositoryID, job.ScheduleID)
	if err != nil {
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// GetJobByID returns a job by its ID.
func (db *DB) GetJobByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	var jobTypeStr, statusStr string
	var payloadBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, job_type, priority, status, payload,
		       retry_count, max_retries, next_retry_at, error_message, last_error_at,
		       created_at, started_at, completed_at,
		       agent_id, repository_id, schedule_id
		FROM job_queue
		WHERE id = $1
	`, id).Scan(
		&job.ID, &job.OrgID, &jobTypeStr, &job.Priority, &statusStr, &payloadBytes,
		&job.RetryCount, &job.MaxRetries, &job.NextRetryAt, &job.ErrorMessage, &job.LastErrorAt,
		&job.CreatedAt, &job.StartedAt, &job.CompletedAt,
		&job.AgentID, &job.RepositoryID, &job.ScheduleID,
	)
	if err != nil {
		return nil, fmt.Errorf("get job by ID: %w", err)
	}

	job.JobType = models.JobType(jobTypeStr)
	job.Status = models.JobStatus(statusStr)
	if err := job.SetPayload(payloadBytes); err != nil {
		db.logger.Warn().Err(err).Str("job_id", job.ID.String()).Msg("failed to parse job payload")
	}

	return &job, nil
}

// ListJobsByOrg returns all jobs for an organization with optional filters.
func (db *DB) ListJobsByOrg(ctx context.Context, orgID uuid.UUID, status *models.JobStatus, jobType *models.JobType, limit int) ([]*models.Job, error) {
	query := `
		SELECT id, org_id, job_type, priority, status, payload,
		       retry_count, max_retries, next_retry_at, error_message, last_error_at,
		       created_at, started_at, completed_at,
		       agent_id, repository_id, schedule_id
		FROM job_queue
		WHERE org_id = $1
	`
	args := []interface{}{orgID}
	argNum := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *status)
		argNum++
	}

	if jobType != nil {
		query += fmt.Sprintf(" AND job_type = $%d", argNum)
		args = append(args, *jobType)
		argNum++
	}

	query += " ORDER BY priority DESC, created_at ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs by org: %w", err)
	}
	defer rows.Close()

	return db.scanJobs(rows)
}

// ListPendingJobs returns jobs ready to be processed.
func (db *DB) ListPendingJobs(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.Job, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, job_type, priority, status, payload,
		       retry_count, max_retries, next_retry_at, error_message, last_error_at,
		       created_at, started_at, completed_at,
		       agent_id, repository_id, schedule_id
		FROM job_queue
		WHERE org_id = $1 AND status = 'pending'
		ORDER BY priority DESC, created_at ASC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending jobs: %w", err)
	}
	defer rows.Close()

	return db.scanJobs(rows)
}

// ListRunningJobs returns currently running jobs.
func (db *DB) ListRunningJobs(ctx context.Context, orgID uuid.UUID) ([]*models.Job, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, job_type, priority, status, payload,
		       retry_count, max_retries, next_retry_at, error_message, last_error_at,
		       created_at, started_at, completed_at,
		       agent_id, repository_id, schedule_id
		FROM job_queue
		WHERE org_id = $1 AND status = 'running'
		ORDER BY started_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list running jobs: %w", err)
	}
	defer rows.Close()

	return db.scanJobs(rows)
}

// ListFailedJobs returns failed jobs that may be retried.
func (db *DB) ListFailedJobs(ctx context.Context, orgID uuid.UUID) ([]*models.Job, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, job_type, priority, status, payload,
		       retry_count, max_retries, next_retry_at, error_message, last_error_at,
		       created_at, started_at, completed_at,
		       agent_id, repository_id, schedule_id
		FROM job_queue
		WHERE org_id = $1 AND status = 'failed'
		ORDER BY last_error_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list failed jobs: %w", err)
	}
	defer rows.Close()

	return db.scanJobs(rows)
}

// ListDeadLetterJobs returns jobs in the dead letter queue.
func (db *DB) ListDeadLetterJobs(ctx context.Context, orgID uuid.UUID) ([]*models.Job, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, job_type, priority, status, payload,
		       retry_count, max_retries, next_retry_at, error_message, last_error_at,
		       created_at, started_at, completed_at,
		       agent_id, repository_id, schedule_id
		FROM job_queue
		WHERE org_id = $1 AND status = 'dead_letter'
		ORDER BY completed_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list dead letter jobs: %w", err)
	}
	defer rows.Close()

	return db.scanJobs(rows)
}

// ListJobsReadyForRetry returns failed jobs ready to be retried.
func (db *DB) ListJobsReadyForRetry(ctx context.Context, limit int) ([]*models.Job, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, job_type, priority, status, payload,
		       retry_count, max_retries, next_retry_at, error_message, last_error_at,
		       created_at, started_at, completed_at,
		       agent_id, repository_id, schedule_id
		FROM job_queue
		WHERE status = 'failed'
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY priority DESC, next_retry_at ASC NULLS FIRST
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list jobs ready for retry: %w", err)
	}
	defer rows.Close()

	return db.scanJobs(rows)
}

// UpdateJob updates a job in the queue.
func (db *DB) UpdateJob(ctx context.Context, job *models.Job) error {
	payloadBytes, err := job.PayloadJSON()
	if err != nil {
		return fmt.Errorf("marshal job payload: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE job_queue
		SET status = $2, payload = $3,
		    retry_count = $4, max_retries = $5, next_retry_at = $6,
		    error_message = $7, last_error_at = $8,
		    started_at = $9, completed_at = $10
		WHERE id = $1
	`, job.ID, job.Status, payloadBytes,
		job.RetryCount, job.MaxRetries, job.NextRetryAt,
		job.ErrorMessage, job.LastErrorAt,
		job.StartedAt, job.CompletedAt)
	if err != nil {
		return fmt.Errorf("update job: %w", err)
	}
	return nil
}

// DeleteJob deletes a job from the queue.
func (db *DB) DeleteJob(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM job_queue WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete job: %w", err)
	}
	return nil
}

// GetJobQueueSummary returns queue statistics for an organization.
func (db *DB) GetJobQueueSummary(ctx context.Context, orgID uuid.UUID) (*models.JobQueueSummary, error) {
	summary := &models.JobQueueSummary{
		ByType: make(map[models.JobType]int),
	}

	// Get counts by status
	err := db.Pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'pending') as pending,
			COUNT(*) FILTER (WHERE status = 'running') as running,
			COUNT(*) FILTER (WHERE status = 'completed') as completed,
			COUNT(*) FILTER (WHERE status = 'failed') as failed,
			COUNT(*) FILTER (WHERE status = 'dead_letter') as dead_letter,
			MIN(created_at) FILTER (WHERE status = 'pending') as oldest_pending
		FROM job_queue
		WHERE org_id = $1
	`, orgID).Scan(
		&summary.TotalPending, &summary.TotalRunning, &summary.TotalCompleted,
		&summary.TotalFailed, &summary.TotalDeadLetter, &summary.OldestPending,
	)
	if err != nil {
		return nil, fmt.Errorf("get job queue summary: %w", err)
	}

	// Get counts by type for pending jobs
	rows, err := db.Pool.Query(ctx, `
		SELECT job_type, COUNT(*)
		FROM job_queue
		WHERE org_id = $1 AND status = 'pending'
		GROUP BY job_type
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get job queue summary by type: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var jobTypeStr string
		var count int
		if err := rows.Scan(&jobTypeStr, &count); err != nil {
			return nil, fmt.Errorf("scan job type count: %w", err)
		}
		summary.ByType[models.JobType(jobTypeStr)] = count
	}

	// Calculate average wait time from completed jobs in last 24 hours
	var avgWait *float64
	err = db.Pool.QueryRow(ctx, `
		SELECT AVG(EXTRACT(EPOCH FROM (started_at - created_at)) / 60)
		FROM job_queue
		WHERE org_id = $1
		  AND status = 'completed'
		  AND started_at IS NOT NULL
		  AND completed_at > NOW() - INTERVAL '24 hours'
	`, orgID).Scan(&avgWait)
	if err == nil && avgWait != nil {
		summary.AvgWaitMinutes = *avgWait
	}

	return summary, nil
}

// GetJobsWithDetails returns jobs with related entity details for display.
func (db *DB) GetJobsWithDetails(ctx context.Context, orgID uuid.UUID, status *models.JobStatus, limit int) ([]*models.JobWithDetails, error) {
	query := `
		SELECT j.id, j.org_id, j.job_type, j.priority, j.status, j.payload,
		       j.retry_count, j.max_retries, j.next_retry_at, j.error_message, j.last_error_at,
		       j.created_at, j.started_at, j.completed_at,
		       j.agent_id, j.repository_id, j.schedule_id,
		       COALESCE(a.hostname, '') as agent_hostname,
		       COALESCE(r.name, '') as repository_name,
		       COALESCE(s.name, '') as schedule_name
		FROM job_queue j
		LEFT JOIN agents a ON j.agent_id = a.id
		LEFT JOIN repositories r ON j.repository_id = r.id
		LEFT JOIN schedules s ON j.schedule_id = s.id
		WHERE j.org_id = $1
	`
	args := []interface{}{orgID}
	argNum := 2

	if status != nil {
		query += fmt.Sprintf(" AND j.status = $%d", argNum)
		args = append(args, *status)
		argNum++
	}

	query += " ORDER BY j.priority DESC, j.created_at ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argNum)
		args = append(args, limit)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get jobs with details: %w", err)
	}
	defer rows.Close()

	var jobs []*models.JobWithDetails
	position := 1
	for rows.Next() {
		var j models.JobWithDetails
		var jobTypeStr, statusStr string
		var payloadBytes []byte

		err := rows.Scan(
			&j.ID, &j.OrgID, &jobTypeStr, &j.Priority, &statusStr, &payloadBytes,
			&j.RetryCount, &j.MaxRetries, &j.NextRetryAt, &j.ErrorMessage, &j.LastErrorAt,
			&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
			&j.AgentID, &j.RepositoryID, &j.ScheduleID,
			&j.AgentHostname, &j.RepositoryName, &j.ScheduleName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan job with details: %w", err)
		}

		j.JobType = models.JobType(jobTypeStr)
		j.Status = models.JobStatus(statusStr)
		if err := j.SetPayload(payloadBytes); err != nil {
			db.logger.Warn().Err(err).Str("job_id", j.ID.String()).Msg("failed to parse job payload")
		}

		if j.Status == models.JobStatusPending {
			j.QueuePosition = position
			position++
		}

		jobs = append(jobs, &j)
	}

	return jobs, nil
}

// CleanupOldJobs removes completed and dead letter jobs older than the specified days.
func (db *DB) CleanupOldJobs(ctx context.Context, retentionDays int) (int64, error) {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM job_queue
		WHERE status IN ('completed', 'dead_letter')
		  AND completed_at < NOW() - INTERVAL '1 day' * $1
	`, retentionDays)
	if err != nil {
		return 0, fmt.Errorf("cleanup old jobs: %w", err)
	}
	return result.RowsAffected(), nil
}

// GetNextPendingJob atomically claims the next pending job for processing.
func (db *DB) GetNextPendingJob(ctx context.Context, orgID uuid.UUID) (*models.Job, error) {
	var job models.Job
	var jobTypeStr, statusStr string
	var payloadBytes []byte

	err := db.Pool.QueryRow(ctx, `
		UPDATE job_queue
		SET status = 'running', started_at = NOW()
		WHERE id = (
			SELECT id FROM job_queue
			WHERE org_id = $1 AND status = 'pending'
			ORDER BY priority DESC, created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, org_id, job_type, priority, status, payload,
		          retry_count, max_retries, next_retry_at, error_message, last_error_at,
		          created_at, started_at, completed_at,
		          agent_id, repository_id, schedule_id
	`, orgID).Scan(
		&job.ID, &job.OrgID, &jobTypeStr, &job.Priority, &statusStr, &payloadBytes,
		&job.RetryCount, &job.MaxRetries, &job.NextRetryAt, &job.ErrorMessage, &job.LastErrorAt,
		&job.CreatedAt, &job.StartedAt, &job.CompletedAt,
		&job.AgentID, &job.RepositoryID, &job.ScheduleID,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("get next pending job: %w", err)
	}

	job.JobType = models.JobType(jobTypeStr)
	job.Status = models.JobStatus(statusStr)
	if err := job.SetPayload(payloadBytes); err != nil {
		db.logger.Warn().Err(err).Str("job_id", job.ID.String()).Msg("failed to parse job payload")
	}

	return &job, nil
}

// Helper to scan job rows
func (db *DB) scanJobs(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]*models.Job, error) {
	var jobs []*models.Job
	for rows.Next() {
		var j models.Job
		var jobTypeStr, statusStr string
		var payloadBytes []byte

		err := rows.Scan(
			&j.ID, &j.OrgID, &jobTypeStr, &j.Priority, &statusStr, &payloadBytes,
			&j.RetryCount, &j.MaxRetries, &j.NextRetryAt, &j.ErrorMessage, &j.LastErrorAt,
			&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
			&j.AgentID, &j.RepositoryID, &j.ScheduleID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}

		j.JobType = models.JobType(jobTypeStr)
		j.Status = models.JobStatus(statusStr)
		if err := j.SetPayload(payloadBytes); err != nil {
			db.logger.Warn().Err(err).Str("job_id", j.ID.String()).Msg("failed to parse job payload")
		}

		jobs = append(jobs, &j)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}

	return jobs, nil
}
