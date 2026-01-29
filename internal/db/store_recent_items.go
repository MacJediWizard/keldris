package db

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// MaxRecentItemsPerUser is the maximum number of recent items to store per user per type.
const MaxRecentItemsPerUser = 20

// CreateOrUpdateRecentItem creates a new recent item or updates the viewed_at timestamp if it exists.
// It also enforces the limit of MaxRecentItemsPerUser items per type.
func (db *DB) CreateOrUpdateRecentItem(ctx context.Context, item *models.RecentItem) error {
	// Use upsert to create or update the viewed_at timestamp
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO recent_items (
			id, org_id, user_id, item_type, item_id, item_name, page_path, viewed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (org_id, user_id, item_type, item_id)
		DO UPDATE SET
			item_name = EXCLUDED.item_name,
			page_path = EXCLUDED.page_path,
			viewed_at = EXCLUDED.viewed_at
	`, item.ID, item.OrgID, item.UserID, string(item.ItemType), item.ItemID, item.ItemName,
		item.PagePath, item.ViewedAt, item.CreatedAt)
	if err != nil {
		return fmt.Errorf("create/update recent item: %w", err)
	}

	// Cleanup old items beyond the limit for this type
	_, err = db.Pool.Exec(ctx, `
		DELETE FROM recent_items
		WHERE id IN (
			SELECT id FROM recent_items
			WHERE org_id = $1 AND user_id = $2 AND item_type = $3
			ORDER BY viewed_at DESC
			OFFSET $4
		)
	`, item.OrgID, item.UserID, string(item.ItemType), MaxRecentItemsPerUser)
	if err != nil {
		return fmt.Errorf("cleanup old recent items: %w", err)
	}

	return nil
}

// GetRecentItemsByUser returns recent items for a user, ordered by viewed_at descending.
func (db *DB) GetRecentItemsByUser(ctx context.Context, orgID, userID uuid.UUID, limit int) ([]*models.RecentItem, error) {
	if limit <= 0 {
		limit = MaxRecentItemsPerUser
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, user_id, item_type, item_id, item_name, page_path, viewed_at, created_at
		FROM recent_items
		WHERE org_id = $1 AND user_id = $2
		ORDER BY viewed_at DESC
		LIMIT $3
	`, orgID, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent items: %w", err)
	}
	defer rows.Close()

	return scanRecentItems(rows)
}

// GetRecentItemsByUserAndType returns recent items for a user filtered by type.
func (db *DB) GetRecentItemsByUserAndType(ctx context.Context, orgID, userID uuid.UUID, itemType models.RecentItemType, limit int) ([]*models.RecentItem, error) {
	if limit <= 0 {
		limit = MaxRecentItemsPerUser
	}

	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, user_id, item_type, item_id, item_name, page_path, viewed_at, created_at
		FROM recent_items
		WHERE org_id = $1 AND user_id = $2 AND item_type = $3
		ORDER BY viewed_at DESC
		LIMIT $4
	`, orgID, userID, string(itemType), limit)
	if err != nil {
		return nil, fmt.Errorf("list recent items by type: %w", err)
	}
	defer rows.Close()

	return scanRecentItems(rows)
}

// DeleteRecentItem deletes a specific recent item.
func (db *DB) DeleteRecentItem(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM recent_items WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete recent item: %w", err)
	}
	return nil
}

// DeleteRecentItemsForUser deletes all recent items for a user.
func (db *DB) DeleteRecentItemsForUser(ctx context.Context, orgID, userID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM recent_items WHERE org_id = $1 AND user_id = $2`, orgID, userID)
	if err != nil {
		return fmt.Errorf("delete all recent items for user: %w", err)
	}
	return nil
}

// GetRecentItemByID returns a recent item by ID.
func (db *DB) GetRecentItemByID(ctx context.Context, id uuid.UUID) (*models.RecentItem, error) {
	var item models.RecentItem
	var itemTypeStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, user_id, item_type, item_id, item_name, page_path, viewed_at, created_at
		FROM recent_items
		WHERE id = $1
	`, id).Scan(
		&item.ID, &item.OrgID, &item.UserID, &itemTypeStr, &item.ItemID, &item.ItemName,
		&item.PagePath, &item.ViewedAt, &item.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get recent item: %w", err)
	}
	item.ItemType = models.RecentItemType(itemTypeStr)
	return &item, nil
}

// scanRecentItems is a helper to scan multiple recent items.
func scanRecentItems(rows pgx.Rows) ([]*models.RecentItem, error) {
	var items []*models.RecentItem
	for rows.Next() {
		var item models.RecentItem
		var itemTypeStr string
		err := rows.Scan(
			&item.ID, &item.OrgID, &item.UserID, &itemTypeStr, &item.ItemID, &item.ItemName,
			&item.PagePath, &item.ViewedAt, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan recent item: %w", err)
		}
		item.ItemType = models.RecentItemType(itemTypeStr)
		items = append(items, &item)
	}
	return items, rows.Err()
}
