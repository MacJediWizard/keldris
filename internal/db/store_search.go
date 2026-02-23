package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SearchFilter defines filters for the global search function.
type SearchFilter struct {
	Query    string      `json:"q"`
	Types    []string    `json:"types,omitempty"`
	Status   string      `json:"status,omitempty"`
	TagIDs   []uuid.UUID `json:"tag_ids,omitempty"`
	DateFrom *time.Time  `json:"date_from,omitempty"`
	DateTo   *time.Time  `json:"date_to,omitempty"`
	SizeMin  *int64      `json:"size_min,omitempty"`
	SizeMax  *int64      `json:"size_max,omitempty"`
	Limit    int         `json:"limit,omitempty"`
}

// SearchResult represents a single search result from the global search.
type SearchResult struct {
	Type        string    `json:"type"`
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Search performs a global search across agents, repositories, schedules, and backups.
func (db *DB) Search(ctx context.Context, orgID uuid.UUID, filter SearchFilter) ([]SearchResult, error) {
	if filter.Query == "" {
		return []SearchResult{}, nil
	}

	var results []SearchResult
	searchPattern := "%" + strings.ReplaceAll(filter.Query, "%", "\\%") + "%"

	// Determine which types to search
	typesToSearch := map[string]bool{
		"agent":      true,
		"repository": true,
		"schedule":   true,
		"backup":     true,
	}
	if len(filter.Types) > 0 {
		typesToSearch = map[string]bool{}
		for _, t := range filter.Types {
			typesToSearch[t] = true
		}
	}

	limit := 10
	if filter.Limit > 0 {
		limit = filter.Limit
	}

	// Search agents
	if typesToSearch["agent"] {
		agentResults, err := db.searchAgents(ctx, orgID, searchPattern, filter, limit)
		if err != nil {
			return nil, fmt.Errorf("search agents: %w", err)
		}
		results = append(results, agentResults...)
	}

	// Search repositories
	if typesToSearch["repository"] {
		repoResults, err := db.searchRepositories(ctx, orgID, searchPattern, filter, limit)
		if err != nil {
			return nil, fmt.Errorf("search repositories: %w", err)
		}
		results = append(results, repoResults...)
	}

	// Search schedules
	if typesToSearch["schedule"] {
		schedResults, err := db.searchSchedules(ctx, orgID, searchPattern, filter, limit)
		if err != nil {
			return nil, fmt.Errorf("search schedules: %w", err)
		}
		results = append(results, schedResults...)
	}

	// Search backups
	if typesToSearch["backup"] {
		backupResults, err := db.searchBackups(ctx, orgID, searchPattern, filter, limit)
		if err != nil {
			return nil, fmt.Errorf("search backups: %w", err)
		}
		results = append(results, backupResults...)
	}

	return results, nil
}

func (db *DB) searchAgents(ctx context.Context, orgID uuid.UUID, pattern string, filter SearchFilter, limit int) ([]SearchResult, error) {
	query := `
		SELECT id, hostname, COALESCE(status, ''), created_at
		FROM agents
		WHERE org_id = $1 AND (hostname ILIKE $2 OR CAST(id AS TEXT) ILIKE $2)
	`
	args := []any{orgID, pattern}
	argIdx := 3

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Type = "agent"
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) searchRepositories(ctx context.Context, orgID uuid.UUID, pattern string, filter SearchFilter, limit int) ([]SearchResult, error) {
	query := `
		SELECT id, name, COALESCE(description, ''), created_at
		FROM repositories
		WHERE org_id = $1 AND (name ILIKE $2 OR COALESCE(description, '') ILIKE $2 OR CAST(id AS TEXT) ILIKE $2)
	`
	args := []any{orgID, pattern}
	argIdx := 3

	_ = filter // status filter not applicable for repositories

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Type = "repository"
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) searchSchedules(ctx context.Context, orgID uuid.UUID, pattern string, filter SearchFilter, limit int) ([]SearchResult, error) {
	query := `
		SELECT s.id, s.name, COALESCE(s.status, ''), s.created_at
		FROM schedules s
		JOIN agents a ON a.id = s.agent_id
		WHERE a.org_id = $1 AND (s.name ILIKE $2 OR CAST(s.id AS TEXT) ILIKE $2)
	`
	args := []any{orgID, pattern}
	argIdx := 3

	if filter.Status != "" {
		query += fmt.Sprintf(" AND s.status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY s.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		r.Type = "schedule"
		results = append(results, r)
	}
	return results, rows.Err()
}

func (db *DB) searchBackups(ctx context.Context, orgID uuid.UUID, pattern string, filter SearchFilter, limit int) ([]SearchResult, error) {
	query := `
		SELECT b.id, COALESCE(b.snapshot_id, ''), COALESCE(b.status, ''), b.created_at, b.size_bytes
		FROM backups b
		JOIN agents a ON a.id = b.agent_id
		WHERE a.org_id = $1 AND (COALESCE(b.snapshot_id, '') ILIKE $2 OR CAST(b.id AS TEXT) ILIKE $2)
	`
	args := []any{orgID, pattern}
	argIdx := 3

	if filter.Status != "" {
		query += fmt.Sprintf(" AND b.status = $%d", argIdx)
		args = append(args, filter.Status)
		argIdx++
	}

	if filter.DateFrom != nil {
		query += fmt.Sprintf(" AND b.created_at >= $%d", argIdx)
		args = append(args, *filter.DateFrom)
		argIdx++
	}

	if filter.DateTo != nil {
		query += fmt.Sprintf(" AND b.created_at <= $%d", argIdx)
		args = append(args, *filter.DateTo)
		argIdx++
	}

	if filter.SizeMin != nil {
		query += fmt.Sprintf(" AND b.size_bytes >= $%d", argIdx)
		args = append(args, *filter.SizeMin)
		argIdx++
	}

	if filter.SizeMax != nil {
		query += fmt.Sprintf(" AND b.size_bytes <= $%d", argIdx)
		args = append(args, *filter.SizeMax)
		argIdx++
	}

	if len(filter.TagIDs) > 0 {
		query += fmt.Sprintf(` AND b.id IN (SELECT backup_id FROM backup_tags WHERE tag_id = ANY($%d))`, argIdx)
		args = append(args, filter.TagIDs)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY b.created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var sizeBytes *int64
		if err := rows.Scan(&r.ID, &r.Name, &r.Status, &r.CreatedAt, &sizeBytes); err != nil {
			return nil, err
		}
		r.Type = "backup"
		results = append(results, r)
	}
	return results, rows.Err()
}
