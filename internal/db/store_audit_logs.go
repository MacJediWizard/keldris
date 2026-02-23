package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// AuditLogFilter defines filters for querying audit logs.
type AuditLogFilter struct {
	Action       string
	ResourceType string
	Result       string
	Search       string
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}

// CreateAuditLog inserts a new audit log entry.
func (db *DB) CreateAuditLog(ctx context.Context, log *models.AuditLog) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO audit_logs (id, org_id, user_id, agent_id, action, resource_type,
		                        resource_id, result, ip_address, user_agent, details, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, log.ID, log.OrgID, log.UserID, log.AgentID, string(log.Action), log.ResourceType,
		log.ResourceID, string(log.Result), log.IPAddress, log.UserAgent, log.Details, log.CreatedAt)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

// GetAuditLogByID returns a single audit log entry by ID.
func (db *DB) GetAuditLogByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error) {
	var log models.AuditLog
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, agent_id, action, resource_type,
		       resource_id, result, ip_address, user_agent, details, created_at
		FROM audit_logs
		WHERE id = $1
	`, id).Scan(&log.ID, &log.OrgID, &log.UserID, &log.AgentID, &log.Action, &log.ResourceType,
		&log.ResourceID, &log.Result, &log.IPAddress, &log.UserAgent, &log.Details, &log.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get audit log: %w", err)
	}
	return &log, nil
}

// GetAuditLogsByOrgID returns audit logs for an organization with optional filtering.
func (db *DB) GetAuditLogsByOrgID(ctx context.Context, orgID uuid.UUID, filter AuditLogFilter) ([]*models.AuditLog, error) {
	query := `
		SELECT id, org_id, user_id, agent_id, action, resource_type,
		       resource_id, result, ip_address, user_agent, details, created_at
		FROM audit_logs
		WHERE org_id = $1
	`
	args := []any{orgID}
	argIdx := 2

	query, args, argIdx = appendAuditLogFilters(query, args, argIdx, filter)

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, filter.Limit)
		argIdx++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		var log models.AuditLog
		if err := rows.Scan(&log.ID, &log.OrgID, &log.UserID, &log.AgentID, &log.Action, &log.ResourceType,
			&log.ResourceID, &log.Result, &log.IPAddress, &log.UserAgent, &log.Details, &log.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		logs = append(logs, &log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}

	return logs, nil
}

// CountAuditLogsByOrgID returns the count of audit logs for an organization with optional filtering.
func (db *DB) CountAuditLogsByOrgID(ctx context.Context, orgID uuid.UUID, filter AuditLogFilter) (int64, error) {
	query := `SELECT COUNT(*) FROM audit_logs WHERE org_id = $1`
	args := []any{orgID}
	argIdx := 2

	query, args, _ = appendAuditLogFilters(query, args, argIdx, filter)

	var count int64
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count audit logs: %w", err)
	}
	return count, nil
}

// appendAuditLogFilters appends WHERE clauses for the given filter to the query.
func appendAuditLogFilters(query string, args []any, argIdx int, filter AuditLogFilter) (string, []any, int) {
	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, filter.Action)
		argIdx++
	}

	if filter.ResourceType != "" {
		query += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, filter.ResourceType)
		argIdx++
	}

	if filter.Result != "" {
		query += fmt.Sprintf(" AND result = $%d", argIdx)
		args = append(args, filter.Result)
		argIdx++
	}

	if filter.Search != "" {
		query += fmt.Sprintf(" AND (details ILIKE $%d OR resource_type ILIKE $%d)", argIdx, argIdx)
		args = append(args, "%"+strings.ReplaceAll(filter.Search, "%", "\\%")+"%")
		argIdx++
	}

	if filter.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filter.EndDate)
		argIdx++
	}

	return query, args, argIdx
}
