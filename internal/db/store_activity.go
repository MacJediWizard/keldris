package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// CreateActivityEvent creates a new activity event.
func (db *DB) CreateActivityEvent(ctx context.Context, event *models.ActivityEvent) error {
	metadataBytes, err := event.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO activity_events (id, org_id, type, category, title, description,
		                             user_id, user_name, agent_id, agent_name,
		                             resource_type, resource_id, resource_name,
		                             metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, event.ID, event.OrgID, string(event.Type), string(event.Category),
		event.Title, event.Description, event.UserID, event.UserName,
		event.AgentID, event.AgentName, event.ResourceType, event.ResourceID,
		event.ResourceName, metadataBytes, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("create activity event: %w", err)
	}
	return nil
}

// GetActivityEvents returns activity events for an organization with optional filtering.
func (db *DB) GetActivityEvents(ctx context.Context, orgID uuid.UUID, filter models.ActivityEventFilter) ([]*models.ActivityEvent, error) {
	query := `
		SELECT id, org_id, type, category, title, description,
		       user_id, user_name, agent_id, agent_name,
		       resource_type, resource_id, resource_name,
		       metadata, created_at
		FROM activity_events
		WHERE org_id = $1
	`
	args := []any{orgID}
	argIdx := 2

	// Apply filters
	if filter.Category != nil {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, string(*filter.Category))
		argIdx++
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, string(*filter.Type))
		argIdx++
	}

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *filter.UserID)
		argIdx++
	}

	if filter.AgentID != nil {
		query += fmt.Sprintf(" AND agent_id = $%d", argIdx)
		args = append(args, *filter.AgentID)
		argIdx++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filter.EndTime)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	// Apply pagination
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)
	argIdx++

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, filter.Offset)
	}

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list activity events: %w", err)
	}
	defer rows.Close()

	return db.scanActivityEvents(rows)
}

// GetActivityEventCount returns the count of activity events matching the filter.
func (db *DB) GetActivityEventCount(ctx context.Context, orgID uuid.UUID, filter models.ActivityEventFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM activity_events
		WHERE org_id = $1
	`
	args := []any{orgID}
	argIdx := 2

	// Apply filters (same as GetActivityEvents but without pagination)
	if filter.Category != nil {
		query += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, string(*filter.Category))
		argIdx++
	}

	if filter.Type != nil {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, string(*filter.Type))
		argIdx++
	}

	if filter.UserID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, *filter.UserID)
		argIdx++
	}

	if filter.AgentID != nil {
		query += fmt.Sprintf(" AND agent_id = $%d", argIdx)
		args = append(args, *filter.AgentID)
		argIdx++
	}

	if filter.StartTime != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *filter.StartTime)
		argIdx++
	}

	if filter.EndTime != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *filter.EndTime)
	}

	var count int
	err := db.Pool.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count activity events: %w", err)
	}
	return count, nil
}

// GetRecentActivityEvents returns the most recent activity events for an organization.
func (db *DB) GetRecentActivityEvents(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.ActivityEvent, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, type, category, title, description,
		       user_id, user_name, agent_id, agent_name,
		       resource_type, resource_id, resource_name,
		       metadata, created_at
		FROM activity_events
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent activity events: %w", err)
	}
	defer rows.Close()

	return db.scanActivityEvents(rows)
}

// GetActivityEventsByCategory returns activity events filtered by category.
func (db *DB) GetActivityEventsByCategory(ctx context.Context, orgID uuid.UUID, category models.ActivityEventCategory, limit int) ([]*models.ActivityEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, type, category, title, description,
		       user_id, user_name, agent_id, agent_name,
		       resource_type, resource_id, resource_name,
		       metadata, created_at
		FROM activity_events
		WHERE org_id = $1 AND category = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, orgID, string(category), limit)
	if err != nil {
		return nil, fmt.Errorf("get activity events by category: %w", err)
	}
	defer rows.Close()

	return db.scanActivityEvents(rows)
}

// DeleteOldActivityEvents deletes activity events older than the specified number of days.
func (db *DB) DeleteOldActivityEvents(ctx context.Context, orgID uuid.UUID, daysOld int) (int64, error) {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM activity_events
		WHERE org_id = $1 AND created_at < NOW() - INTERVAL '1 day' * $2
	`, orgID, daysOld)
	if err != nil {
		return 0, fmt.Errorf("delete old activity events: %w", err)
	}
	return result.RowsAffected(), nil
}

// scanActivityEvents scans rows into ActivityEvent slice.
func (db *DB) scanActivityEvents(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*models.ActivityEvent, error) {
	var events []*models.ActivityEvent

	for rows.Next() {
		var e models.ActivityEvent
		var typeStr, categoryStr string
		var metadataBytes []byte

		err := rows.Scan(
			&e.ID, &e.OrgID, &typeStr, &categoryStr, &e.Title, &e.Description,
			&e.UserID, &e.UserName, &e.AgentID, &e.AgentName,
			&e.ResourceType, &e.ResourceID, &e.ResourceName,
			&metadataBytes, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan activity event: %w", err)
		}

		e.Type = models.ActivityEventType(typeStr)
		e.Category = models.ActivityEventCategory(categoryStr)

		if err := e.ParseMetadata(metadataBytes); err != nil {
			db.logger.Warn().Err(err).Str("event_id", e.ID.String()).Msg("failed to parse activity event metadata")
		}

		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity events: %w", err)
	}

	return events, nil
}

// GetActivityCategories returns all distinct categories with their event counts.
func (db *DB) GetActivityCategories(ctx context.Context, orgID uuid.UUID) (map[string]int, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT category, COUNT(*)
		FROM activity_events
		WHERE org_id = $1
		GROUP BY category
		ORDER BY COUNT(*) DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get activity categories: %w", err)
	}
	defer rows.Close()

	categories := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		categories[category] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}

	return categories, nil
}

// SearchActivityEvents searches activity events by title or description.
func (db *DB) SearchActivityEvents(ctx context.Context, orgID uuid.UUID, query string, limit int) ([]*models.ActivityEvent, error) {
	if limit <= 0 {
		limit = 50
	}

	// Sanitize the search query
	searchQuery := "%" + strings.ToLower(query) + "%"

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, type, category, title, description,
		       user_id, user_name, agent_id, agent_name,
		       resource_type, resource_id, resource_name,
		       metadata, created_at
		FROM activity_events
		WHERE org_id = $1 AND (
			LOWER(title) LIKE $2 OR
			LOWER(description) LIKE $2 OR
			LOWER(COALESCE(user_name, '')) LIKE $2 OR
			LOWER(COALESCE(agent_name, '')) LIKE $2 OR
			LOWER(COALESCE(resource_name, '')) LIKE $2
		)
		ORDER BY created_at DESC
		LIMIT $3
	`, orgID, searchQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search activity events: %w", err)
	}
	defer rows.Close()

	return db.scanActivityEvents(rows)
}
