// Package search provides global search functionality across all entities.
package search

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// ResultType represents the type of search result.
type ResultType string

const (
	ResultTypeAgent      ResultType = "agent"
	ResultTypeBackup     ResultType = "backup"
	ResultTypeSnapshot   ResultType = "snapshot"
	ResultTypeSchedule   ResultType = "schedule"
	ResultTypeRepository ResultType = "repository"
)

// Result represents a single search result.
type Result struct {
	Type        ResultType `json:"type"`
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	Metadata    any        `json:"metadata,omitempty"`
}

// Filter contains filters for global search queries.
type Filter struct {
	Query    string      `json:"q"`
	Types    []string    `json:"types,omitempty"`     // agent, backup, snapshot, schedule, repository
	Status   string      `json:"status,omitempty"`    // filter by status
	TagIDs   []uuid.UUID `json:"tag_ids,omitempty"`   // filter by tags
	DateFrom *time.Time  `json:"date_from,omitempty"` // filter by date range
	DateTo   *time.Time  `json:"date_to,omitempty"`   // filter by date range
	SizeMin  *int64      `json:"size_min,omitempty"`  // filter by size (for backups)
	SizeMax  *int64      `json:"size_max,omitempty"`  // filter by size (for backups)
	Limit    int         `json:"limit,omitempty"`     // max results per type
}

// GroupedResults contains search results grouped by type.
type GroupedResults struct {
	Agents       []Result `json:"agents"`
	Backups      []Result `json:"backups"`
	Snapshots    []Result `json:"snapshots"`
	Schedules    []Result `json:"schedules"`
	Repositories []Result `json:"repositories"`
	Total        int      `json:"total"`
}

// Suggestion represents an autocomplete suggestion.
type Suggestion struct {
	Text   string     `json:"text"`
	Type   ResultType `json:"type"`
	ID     string     `json:"id"`
	Detail string     `json:"detail,omitempty"`
}

// RecentSearch represents a recent search entry.
type RecentSearch struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	OrgID     uuid.UUID  `json:"org_id"`
	Query     string     `json:"query"`
	Types     []string   `json:"types,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// GlobalSearcher provides global search functionality.
type GlobalSearcher struct {
	pool   *pgxpool.Pool
	logger zerolog.Logger
}

// NewGlobalSearcher creates a new GlobalSearcher.
func NewGlobalSearcher(pool *pgxpool.Pool, logger zerolog.Logger) *GlobalSearcher {
	return &GlobalSearcher{
		pool:   pool,
		logger: logger.With().Str("component", "global_search").Logger(),
	}
}

// Search performs a global search across all entity types.
func (s *GlobalSearcher) Search(ctx context.Context, orgID uuid.UUID, filter Filter) (*GroupedResults, error) {
	results := &GroupedResults{
		Agents:       []Result{},
		Backups:      []Result{},
		Snapshots:    []Result{},
		Schedules:    []Result{},
		Repositories: []Result{},
	}

	if filter.Query == "" {
		return results, nil
	}

	// Set default limit
	if filter.Limit <= 0 {
		filter.Limit = 10
	}

	// Determine which types to search
	searchAll := len(filter.Types) == 0
	typeSet := make(map[string]bool)
	for _, t := range filter.Types {
		typeSet[t] = true
	}

	// Search agents
	if searchAll || typeSet["agent"] {
		agentResults, err := s.searchAgents(ctx, orgID, filter)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to search agents")
		} else {
			results.Agents = agentResults
			results.Total += len(agentResults)
		}
	}

	// Search backups
	if searchAll || typeSet["backup"] {
		backupResults, err := s.searchBackups(ctx, orgID, filter)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to search backups")
		} else {
			results.Backups = backupResults
			results.Total += len(backupResults)
		}
	}

	// Search snapshots
	if searchAll || typeSet["snapshot"] {
		snapshotResults, err := s.searchSnapshots(ctx, orgID, filter)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to search snapshots")
		} else {
			results.Snapshots = snapshotResults
			results.Total += len(snapshotResults)
		}
	}

	// Search schedules
	if searchAll || typeSet["schedule"] {
		scheduleResults, err := s.searchSchedules(ctx, orgID, filter)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to search schedules")
		} else {
			results.Schedules = scheduleResults
			results.Total += len(scheduleResults)
		}
	}

	// Search repositories
	if searchAll || typeSet["repository"] {
		repoResults, err := s.searchRepositories(ctx, orgID, filter)
		if err != nil {
			s.logger.Error().Err(err).Msg("failed to search repositories")
		} else {
			results.Repositories = repoResults
			results.Total += len(repoResults)
		}
	}

	return results, nil
}

// searchAgents searches agents by hostname, IP, and tags.
func (s *GlobalSearcher) searchAgents(ctx context.Context, orgID uuid.UUID, filter Filter) ([]Result, error) {
	query := "%" + filter.Query + "%"

	sqlQuery := `
		SELECT DISTINCT a.id, a.hostname, a.status, a.created_at,
		       COALESCE((a.os_info->>'hostname')::text, ''),
		       COALESCE((SELECT string_agg(t.name, ', ')
		         FROM agent_tags at
		         JOIN tags t ON at.tag_id = t.id
		         WHERE at.agent_id = a.id), '')
		FROM agents a
		LEFT JOIN agent_tags at ON a.id = at.agent_id
		LEFT JOIN tags t ON at.tag_id = t.id
		WHERE a.org_id = $1 AND (
			a.hostname ILIKE $2
			OR (a.os_info->>'hostname')::text ILIKE $2
			OR t.name ILIKE $2
			OR EXISTS (
				SELECT 1 FROM jsonb_each_text(a.metadata) m
				WHERE m.value ILIKE $2
			)
		)
	`
	args := []any{orgID, query}
	argNum := 3

	if filter.Status != "" {
		sqlQuery += fmt.Sprintf(" AND a.status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		sqlQuery += fmt.Sprintf(" AND at.tag_id = ANY($%d)", argNum)
		args = append(args, filter.TagIDs)
		argNum++
	}

	if filter.DateFrom != nil {
		sqlQuery += fmt.Sprintf(" AND a.created_at >= $%d", argNum)
		args = append(args, filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		sqlQuery += fmt.Sprintf(" AND a.created_at <= $%d", argNum)
		args = append(args, filter.DateTo)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY a.hostname LIMIT %d", filter.Limit)

	rows, err := s.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var id uuid.UUID
		var hostname, status, osHostname, tagNames string
		var createdAt time.Time
		if err := rows.Scan(&id, &hostname, &status, &createdAt, &osHostname, &tagNames); err != nil {
			return nil, fmt.Errorf("scan agent: %w", err)
		}

		var tags []string
		if tagNames != "" {
			tags = strings.Split(tagNames, ", ")
		}

		desc := ""
		if osHostname != "" && osHostname != hostname {
			desc = osHostname
		}

		results = append(results, Result{
			Type:        ResultTypeAgent,
			ID:          id.String(),
			Name:        hostname,
			Description: desc,
			Status:      status,
			Tags:        tags,
			CreatedAt:   createdAt,
		})
	}

	return results, rows.Err()
}

// searchBackups searches backups by date, status, and tags.
func (s *GlobalSearcher) searchBackups(ctx context.Context, orgID uuid.UUID, filter Filter) ([]Result, error) {
	query := "%" + filter.Query + "%"

	sqlQuery := `
		SELECT DISTINCT b.id, b.snapshot_id, b.status, b.size_bytes, b.started_at, a.hostname,
		       COALESCE((SELECT string_agg(t.name, ', ')
		         FROM backup_tags bt
		         JOIN tags t ON bt.tag_id = t.id
		         WHERE bt.backup_id = b.id), '')
		FROM backups b
		JOIN agents a ON b.agent_id = a.id
		LEFT JOIN backup_tags bt ON b.id = bt.backup_id
		LEFT JOIN tags t ON bt.tag_id = t.id
		WHERE a.org_id = $1 AND (
			b.snapshot_id ILIKE $2
			OR b.id::text ILIKE $2
			OR a.hostname ILIKE $2
			OR t.name ILIKE $2
		)
	`
	args := []any{orgID, query}
	argNum := 3

	if filter.Status != "" {
		sqlQuery += fmt.Sprintf(" AND b.status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		sqlQuery += fmt.Sprintf(" AND bt.tag_id = ANY($%d)", argNum)
		args = append(args, filter.TagIDs)
		argNum++
	}

	if filter.DateFrom != nil {
		sqlQuery += fmt.Sprintf(" AND b.started_at >= $%d", argNum)
		args = append(args, filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		sqlQuery += fmt.Sprintf(" AND b.started_at <= $%d", argNum)
		args = append(args, filter.DateTo)
		argNum++
	}

	if filter.SizeMin != nil {
		sqlQuery += fmt.Sprintf(" AND b.size_bytes >= $%d", argNum)
		args = append(args, filter.SizeMin)
		argNum++
	}

	if filter.SizeMax != nil {
		sqlQuery += fmt.Sprintf(" AND b.size_bytes <= $%d", argNum)
		args = append(args, filter.SizeMax)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY b.started_at DESC LIMIT %d", filter.Limit)

	rows, err := s.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query backups: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var id uuid.UUID
		var snapshotID, status, hostname, tagNames string
		var sizeBytes *int64
		var startedAt time.Time
		if err := rows.Scan(&id, &snapshotID, &status, &sizeBytes, &startedAt, &hostname, &tagNames); err != nil {
			return nil, fmt.Errorf("scan backup: %w", err)
		}

		var tags []string
		if tagNames != "" {
			tags = strings.Split(tagNames, ", ")
		}

		name := snapshotID
		if name == "" {
			name = id.String()[:8]
		}

		desc := hostname
		if sizeBytes != nil {
			desc = fmt.Sprintf("%s - %s", hostname, formatBytes(*sizeBytes))
		}

		results = append(results, Result{
			Type:        ResultTypeBackup,
			ID:          id.String(),
			Name:        name,
			Description: desc,
			Status:      status,
			Tags:        tags,
			CreatedAt:   startedAt,
		})
	}

	return results, rows.Err()
}

// searchSnapshots searches snapshots by ID and file contents.
func (s *GlobalSearcher) searchSnapshots(ctx context.Context, orgID uuid.UUID, filter Filter) ([]Result, error) {
	query := "%" + filter.Query + "%"

	sqlQuery := `
		SELECT DISTINCT b.id, b.snapshot_id, b.status, b.files_total, b.started_at, a.hostname
		FROM backups b
		JOIN agents a ON b.agent_id = a.id
		WHERE a.org_id = $1
		  AND b.snapshot_id IS NOT NULL
		  AND b.snapshot_id != ''
		  AND b.snapshot_id ILIKE $2
	`
	args := []any{orgID, query}
	argNum := 3

	if filter.Status != "" {
		sqlQuery += fmt.Sprintf(" AND b.status = $%d", argNum)
		args = append(args, filter.Status)
		argNum++
	}

	if filter.DateFrom != nil {
		sqlQuery += fmt.Sprintf(" AND b.started_at >= $%d", argNum)
		args = append(args, filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		sqlQuery += fmt.Sprintf(" AND b.started_at <= $%d", argNum)
		args = append(args, filter.DateTo)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY b.started_at DESC LIMIT %d", filter.Limit)

	rows, err := s.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query snapshots: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var id uuid.UUID
		var snapshotID, status, hostname string
		var filesTotal *int64
		var startedAt time.Time
		if err := rows.Scan(&id, &snapshotID, &status, &filesTotal, &startedAt, &hostname); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}

		desc := hostname
		if filesTotal != nil {
			desc = fmt.Sprintf("%s - %d files", hostname, *filesTotal)
		}

		results = append(results, Result{
			Type:        ResultTypeSnapshot,
			ID:          snapshotID,
			Name:        snapshotID,
			Description: desc,
			Status:      status,
			CreatedAt:   startedAt,
			Metadata:    map[string]string{"backup_id": id.String()},
		})
	}

	return results, rows.Err()
}

// searchSchedules searches schedules by name.
func (s *GlobalSearcher) searchSchedules(ctx context.Context, orgID uuid.UUID, filter Filter) ([]Result, error) {
	query := "%" + filter.Query + "%"

	sqlQuery := `
		SELECT DISTINCT sc.id, sc.name, sc.enabled, sc.cron_expression, sc.created_at, a.hostname
		FROM schedules sc
		JOIN agents a ON sc.agent_id = a.id
		WHERE a.org_id = $1 AND (
			sc.name ILIKE $2
			OR a.hostname ILIKE $2
			OR sc.cron_expression ILIKE $2
		)
	`
	args := []any{orgID, query}
	argNum := 3

	if filter.DateFrom != nil {
		sqlQuery += fmt.Sprintf(" AND sc.created_at >= $%d", argNum)
		args = append(args, filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		sqlQuery += fmt.Sprintf(" AND sc.created_at <= $%d", argNum)
		args = append(args, filter.DateTo)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY sc.name LIMIT %d", filter.Limit)

	rows, err := s.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query schedules: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var id uuid.UUID
		var name, cronExpr, hostname string
		var enabled bool
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &enabled, &cronExpr, &createdAt, &hostname); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}

		status := "disabled"
		if enabled {
			status = "enabled"
		}

		results = append(results, Result{
			Type:        ResultTypeSchedule,
			ID:          id.String(),
			Name:        name,
			Description: fmt.Sprintf("%s - %s", hostname, cronExpr),
			Status:      status,
			CreatedAt:   createdAt,
		})
	}

	return results, rows.Err()
}

// searchRepositories searches repositories by name.
func (s *GlobalSearcher) searchRepositories(ctx context.Context, orgID uuid.UUID, filter Filter) ([]Result, error) {
	query := "%" + filter.Query + "%"

	sqlQuery := `
		SELECT id, name, type, created_at
		FROM repositories
		WHERE org_id = $1 AND (
			name ILIKE $2
			OR type ILIKE $2
		)
	`
	args := []any{orgID, query}
	argNum := 3

	if filter.DateFrom != nil {
		sqlQuery += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		sqlQuery += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, filter.DateTo)
		argNum++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY name LIMIT %d", filter.Limit)

	rows, err := s.pool.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query repositories: %w", err)
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var id uuid.UUID
		var name, repoType string
		var createdAt time.Time
		if err := rows.Scan(&id, &name, &repoType, &createdAt); err != nil {
			return nil, fmt.Errorf("scan repository: %w", err)
		}

		results = append(results, Result{
			Type:        ResultTypeRepository,
			ID:          id.String(),
			Name:        name,
			Description: repoType,
			CreatedAt:   createdAt,
		})
	}

	return results, rows.Err()
}

// GetSuggestions returns autocomplete suggestions for a partial query.
func (s *GlobalSearcher) GetSuggestions(ctx context.Context, orgID uuid.UUID, prefix string, limit int) ([]Suggestion, error) {
	if prefix == "" || limit <= 0 {
		return []Suggestion{}, nil
	}

	if limit > 20 {
		limit = 20
	}

	query := prefix + "%"
	var suggestions []Suggestion

	// Get agent hostname suggestions
	agentRows, err := s.pool.Query(ctx, `
		SELECT id, hostname FROM agents
		WHERE org_id = $1 AND hostname ILIKE $2
		ORDER BY hostname LIMIT $3
	`, orgID, query, limit/4+1)
	if err == nil {
		defer agentRows.Close()
		for agentRows.Next() {
			var id uuid.UUID
			var hostname string
			if err := agentRows.Scan(&id, &hostname); err == nil {
				suggestions = append(suggestions, Suggestion{
					Text:   hostname,
					Type:   ResultTypeAgent,
					ID:     id.String(),
					Detail: "Agent",
				})
			}
		}
	}

	// Get schedule name suggestions
	scheduleRows, err := s.pool.Query(ctx, `
		SELECT s.id, s.name FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND s.name ILIKE $2
		ORDER BY s.name LIMIT $3
	`, orgID, query, limit/4+1)
	if err == nil {
		defer scheduleRows.Close()
		for scheduleRows.Next() {
			var id uuid.UUID
			var name string
			if err := scheduleRows.Scan(&id, &name); err == nil {
				suggestions = append(suggestions, Suggestion{
					Text:   name,
					Type:   ResultTypeSchedule,
					ID:     id.String(),
					Detail: "Schedule",
				})
			}
		}
	}

	// Get repository name suggestions
	repoRows, err := s.pool.Query(ctx, `
		SELECT id, name, type FROM repositories
		WHERE org_id = $1 AND name ILIKE $2
		ORDER BY name LIMIT $3
	`, orgID, query, limit/4+1)
	if err == nil {
		defer repoRows.Close()
		for repoRows.Next() {
			var id uuid.UUID
			var name, repoType string
			if err := repoRows.Scan(&id, &name, &repoType); err == nil {
				suggestions = append(suggestions, Suggestion{
					Text:   name,
					Type:   ResultTypeRepository,
					ID:     id.String(),
					Detail: "Repository (" + repoType + ")",
				})
			}
		}
	}

	// Limit total suggestions
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

// SaveRecentSearch saves a search query to the user's recent search history.
func (s *GlobalSearcher) SaveRecentSearch(ctx context.Context, userID, orgID uuid.UUID, query string, types []string) error {
	// Delete existing identical search
	_, err := s.pool.Exec(ctx, `
		DELETE FROM recent_searches
		WHERE user_id = $1 AND org_id = $2 AND query = $3
	`, userID, orgID, query)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to delete duplicate recent search")
	}

	// Insert new search
	_, err = s.pool.Exec(ctx, `
		INSERT INTO recent_searches (id, user_id, org_id, query, types, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), userID, orgID, query, types, time.Now())
	if err != nil {
		return fmt.Errorf("insert recent search: %w", err)
	}

	// Keep only last 20 searches per user/org
	_, err = s.pool.Exec(ctx, `
		DELETE FROM recent_searches
		WHERE id IN (
			SELECT id FROM recent_searches
			WHERE user_id = $1 AND org_id = $2
			ORDER BY created_at DESC
			OFFSET 20
		)
	`, userID, orgID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to prune recent searches")
	}

	return nil
}

// GetRecentSearches returns the user's recent search history.
func (s *GlobalSearcher) GetRecentSearches(ctx context.Context, userID, orgID uuid.UUID, limit int) ([]RecentSearch, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, org_id, query, types, created_at
		FROM recent_searches
		WHERE user_id = $1 AND org_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, userID, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent searches: %w", err)
	}
	defer rows.Close()

	var searches []RecentSearch
	for rows.Next() {
		var rs RecentSearch
		if err := rows.Scan(&rs.ID, &rs.UserID, &rs.OrgID, &rs.Query, &rs.Types, &rs.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent search: %w", err)
		}
		searches = append(searches, rs)
	}

	return searches, rows.Err()
}

// DeleteRecentSearch deletes a specific recent search.
func (s *GlobalSearcher) DeleteRecentSearch(ctx context.Context, userID, orgID, searchID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM recent_searches
		WHERE id = $1 AND user_id = $2 AND org_id = $3
	`, searchID, userID, orgID)
	if err != nil {
		return fmt.Errorf("delete recent search: %w", err)
	}
	return nil
}

// ClearRecentSearches clears all recent searches for a user.
func (s *GlobalSearcher) ClearRecentSearches(ctx context.Context, userID, orgID uuid.UUID) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM recent_searches
		WHERE user_id = $1 AND org_id = $2
	`, userID, orgID)
	if err != nil {
		return fmt.Errorf("clear recent searches: %w", err)
	}
	return nil
}

// formatBytes formats bytes to human readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
