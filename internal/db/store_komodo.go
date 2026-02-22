package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// --- Komodo Integration CRUD ---

// GetKomodoIntegrationsByOrgID returns all Komodo integrations for an organization.
func (db *DB) GetKomodoIntegrationsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoIntegration, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, url, config_encrypted, status,
		       last_sync_at, last_error, enabled, created_at, updated_at
		FROM komodo_integrations
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list komodo integrations: %w", err)
	}
	defer rows.Close()

	var integrations []*models.KomodoIntegration
	for rows.Next() {
		i, err := scanKomodoIntegration(rows)
		if err != nil {
			return nil, err
		}
		integrations = append(integrations, i)
	}

	return integrations, nil
}

// GetKomodoIntegrationByID returns a Komodo integration by ID.
func (db *DB) GetKomodoIntegrationByID(ctx context.Context, id uuid.UUID) (*models.KomodoIntegration, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, url, config_encrypted, status,
		       last_sync_at, last_error, enabled, created_at, updated_at
		FROM komodo_integrations
		WHERE id = $1
	`, id)

	return scanKomodoIntegrationRow(row)
}

// CreateKomodoIntegration creates a new Komodo integration.
func (db *DB) CreateKomodoIntegration(ctx context.Context, integration *models.KomodoIntegration) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_integrations (
			id, org_id, name, url, config_encrypted, status,
			last_sync_at, last_error, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`,
		integration.ID, integration.OrgID, integration.Name, integration.URL,
		integration.ConfigEncrypted, string(integration.Status),
		integration.LastSyncAt, integration.LastError, integration.Enabled,
		integration.CreatedAt, integration.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create komodo integration: %w", err)
	}
	return nil
}

// UpdateKomodoIntegration updates an existing Komodo integration.
func (db *DB) UpdateKomodoIntegration(ctx context.Context, integration *models.KomodoIntegration) error {
	integration.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE komodo_integrations
		SET name = $2, url = $3, config_encrypted = $4, status = $5,
		    last_sync_at = $6, last_error = $7, enabled = $8, updated_at = $9
		WHERE id = $1
	`,
		integration.ID, integration.Name, integration.URL,
		integration.ConfigEncrypted, string(integration.Status),
		integration.LastSyncAt, integration.LastError, integration.Enabled,
		integration.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update komodo integration: %w", err)
	}
	return nil
}

// DeleteKomodoIntegration deletes a Komodo integration by ID.
func (db *DB) DeleteKomodoIntegration(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM komodo_integrations WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete komodo integration: %w", err)
	}
	return nil
}

// scanKomodoIntegration scans a Komodo integration from a multi-row result.
func scanKomodoIntegration(rows pgx.Rows) (*models.KomodoIntegration, error) {
	var i models.KomodoIntegration
	var statusStr string

	err := rows.Scan(
		&i.ID, &i.OrgID, &i.Name, &i.URL, &i.ConfigEncrypted, &statusStr,
		&i.LastSyncAt, &i.LastError, &i.Enabled, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan komodo integration: %w", err)
	}
	i.Status = models.KomodoIntegrationStatus(statusStr)
	return &i, nil
}

// scanKomodoIntegrationRow scans a Komodo integration from a single row.
func scanKomodoIntegrationRow(row pgx.Row) (*models.KomodoIntegration, error) {
	var i models.KomodoIntegration
	var statusStr string

	err := row.Scan(
		&i.ID, &i.OrgID, &i.Name, &i.URL, &i.ConfigEncrypted, &statusStr,
		&i.LastSyncAt, &i.LastError, &i.Enabled, &i.CreatedAt, &i.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan komodo integration: %w", err)
	}
	i.Status = models.KomodoIntegrationStatus(statusStr)
	return &i, nil
}

// --- Komodo Container CRUD ---

// GetKomodoContainersByOrgID returns all Komodo containers for an organization.
func (db *DB) GetKomodoContainersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoContainer, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled,
		       last_discovered_at, created_at, updated_at
		FROM komodo_containers
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list komodo containers by org: %w", err)
	}
	defer rows.Close()

	var containers []*models.KomodoContainer
	for rows.Next() {
		c, err := scanKomodoContainer(rows)
		if err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}

	return containers, nil
}

// GetKomodoContainersByIntegrationID returns all Komodo containers for an integration.
func (db *DB) GetKomodoContainersByIntegrationID(ctx context.Context, integrationID uuid.UUID) ([]*models.KomodoContainer, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled,
		       last_discovered_at, created_at, updated_at
		FROM komodo_containers
		WHERE integration_id = $1
		ORDER BY name
	`, integrationID)
	if err != nil {
		return nil, fmt.Errorf("list komodo containers by integration: %w", err)
	}
	defer rows.Close()

	var containers []*models.KomodoContainer
	for rows.Next() {
		c, err := scanKomodoContainer(rows)
		if err != nil {
			return nil, err
		}
		containers = append(containers, c)
	}

	return containers, nil
}

// GetKomodoContainerByID returns a Komodo container by ID.
func (db *DB) GetKomodoContainerByID(ctx context.Context, id uuid.UUID) (*models.KomodoContainer, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled,
		       last_discovered_at, created_at, updated_at
		FROM komodo_containers
		WHERE id = $1
	`, id)

	return scanKomodoContainerRow(row)
}

// GetKomodoContainerByKomodoID returns a Komodo container by its Komodo ID within an integration.
func (db *DB) GetKomodoContainerByKomodoID(ctx context.Context, integrationID uuid.UUID, komodoID string) (*models.KomodoContainer, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
		       status, agent_id, volumes, labels, backup_enabled,
		       last_discovered_at, created_at, updated_at
		FROM komodo_containers
		WHERE integration_id = $1 AND komodo_id = $2
	`, integrationID, komodoID)

	return scanKomodoContainerRow(row)
}

// CreateKomodoContainer creates a new Komodo container.
func (db *DB) CreateKomodoContainer(ctx context.Context, container *models.KomodoContainer) error {
	labelsJSON, err := json.Marshal(container.Labels)
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO komodo_containers (
			id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
			status, agent_id, volumes, labels, backup_enabled,
			last_discovered_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`,
		container.ID, container.OrgID, container.IntegrationID, container.KomodoID,
		container.Name, container.Image, container.StackName, container.StackID,
		string(container.Status), container.AgentID, container.Volumes, labelsJSON,
		container.BackupEnabled, container.LastDiscoveredAt, container.CreatedAt, container.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create komodo container: %w", err)
	}
	return nil
}

// UpdateKomodoContainer updates an existing Komodo container.
func (db *DB) UpdateKomodoContainer(ctx context.Context, container *models.KomodoContainer) error {
	container.UpdatedAt = time.Now()

	labelsJSON, err := json.Marshal(container.Labels)
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE komodo_containers
		SET name = $2, image = $3, stack_name = $4, stack_id = $5,
		    status = $6, agent_id = $7, volumes = $8, labels = $9,
		    backup_enabled = $10, last_discovered_at = $11, updated_at = $12
		WHERE id = $1
	`,
		container.ID, container.Name, container.Image, container.StackName, container.StackID,
		string(container.Status), container.AgentID, container.Volumes, labelsJSON,
		container.BackupEnabled, container.LastDiscoveredAt, container.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update komodo container: %w", err)
	}
	return nil
}

// DeleteKomodoContainer deletes a Komodo container by ID.
func (db *DB) DeleteKomodoContainer(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM komodo_containers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete komodo container: %w", err)
	}
	return nil
}

// UpsertKomodoContainer creates or updates a Komodo container keyed by (integration_id, komodo_id).
func (db *DB) UpsertKomodoContainer(ctx context.Context, container *models.KomodoContainer) error {
	container.UpdatedAt = time.Now()

	labelsJSON, err := json.Marshal(container.Labels)
	if err != nil {
		return fmt.Errorf("marshal labels: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO komodo_containers (
			id, org_id, integration_id, komodo_id, name, image, stack_name, stack_id,
			status, agent_id, volumes, labels, backup_enabled,
			last_discovered_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (integration_id, komodo_id)
		DO UPDATE SET
			name = EXCLUDED.name,
			image = EXCLUDED.image,
			stack_name = EXCLUDED.stack_name,
			stack_id = EXCLUDED.stack_id,
			status = EXCLUDED.status,
			volumes = EXCLUDED.volumes,
			labels = EXCLUDED.labels,
			last_discovered_at = EXCLUDED.last_discovered_at,
			updated_at = EXCLUDED.updated_at
	`,
		container.ID, container.OrgID, container.IntegrationID, container.KomodoID,
		container.Name, container.Image, container.StackName, container.StackID,
		string(container.Status), container.AgentID, container.Volumes, labelsJSON,
		container.BackupEnabled, container.LastDiscoveredAt, container.CreatedAt, container.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert komodo container: %w", err)
	}
	return nil
}

// scanKomodoContainer scans a Komodo container from a multi-row result.
func scanKomodoContainer(rows pgx.Rows) (*models.KomodoContainer, error) {
	var c models.KomodoContainer
	var statusStr string
	var labelsBytes []byte

	err := rows.Scan(
		&c.ID, &c.OrgID, &c.IntegrationID, &c.KomodoID, &c.Name, &c.Image,
		&c.StackName, &c.StackID, &statusStr, &c.AgentID, &c.Volumes, &labelsBytes,
		&c.BackupEnabled, &c.LastDiscoveredAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan komodo container: %w", err)
	}
	c.Status = models.KomodoContainerStatus(statusStr)

	if len(labelsBytes) > 0 {
		if err := json.Unmarshal(labelsBytes, &c.Labels); err != nil {
			return nil, fmt.Errorf("parse komodo container labels: %w", err)
		}
	}

	return &c, nil
}

// scanKomodoContainerRow scans a Komodo container from a single row.
func scanKomodoContainerRow(row pgx.Row) (*models.KomodoContainer, error) {
	var c models.KomodoContainer
	var statusStr string
	var labelsBytes []byte

	err := row.Scan(
		&c.ID, &c.OrgID, &c.IntegrationID, &c.KomodoID, &c.Name, &c.Image,
		&c.StackName, &c.StackID, &statusStr, &c.AgentID, &c.Volumes, &labelsBytes,
		&c.BackupEnabled, &c.LastDiscoveredAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan komodo container: %w", err)
	}
	c.Status = models.KomodoContainerStatus(statusStr)

	if len(labelsBytes) > 0 {
		if err := json.Unmarshal(labelsBytes, &c.Labels); err != nil {
			return nil, fmt.Errorf("parse komodo container labels: %w", err)
		}
	}

	return &c, nil
}

// --- Komodo Stack CRUD ---

// GetKomodoStacksByOrgID returns all Komodo stacks for an organization.
func (db *DB) GetKomodoStacksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.KomodoStack, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, server_id, server_name,
		       container_count, running_count, last_discovered_at, created_at, updated_at
		FROM komodo_stacks
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list komodo stacks by org: %w", err)
	}
	defer rows.Close()

	var stacks []*models.KomodoStack
	for rows.Next() {
		s, err := scanKomodoStack(rows)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, s)
	}

	return stacks, nil
}

// GetKomodoStacksByIntegrationID returns all Komodo stacks for an integration.
func (db *DB) GetKomodoStacksByIntegrationID(ctx context.Context, integrationID uuid.UUID) ([]*models.KomodoStack, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, server_id, server_name,
		       container_count, running_count, last_discovered_at, created_at, updated_at
		FROM komodo_stacks
		WHERE integration_id = $1
		ORDER BY name
	`, integrationID)
	if err != nil {
		return nil, fmt.Errorf("list komodo stacks by integration: %w", err)
	}
	defer rows.Close()

	var stacks []*models.KomodoStack
	for rows.Next() {
		s, err := scanKomodoStack(rows)
		if err != nil {
			return nil, err
		}
		stacks = append(stacks, s)
	}

	return stacks, nil
}

// GetKomodoStackByID returns a Komodo stack by ID.
func (db *DB) GetKomodoStackByID(ctx context.Context, id uuid.UUID) (*models.KomodoStack, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, integration_id, komodo_id, name, server_id, server_name,
		       container_count, running_count, last_discovered_at, created_at, updated_at
		FROM komodo_stacks
		WHERE id = $1
	`, id)

	return scanKomodoStackRow(row)
}

// UpsertKomodoStack creates or updates a Komodo stack keyed by (integration_id, komodo_id).
func (db *DB) UpsertKomodoStack(ctx context.Context, stack *models.KomodoStack) error {
	stack.UpdatedAt = time.Now()

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_stacks (
			id, org_id, integration_id, komodo_id, name, server_id, server_name,
			container_count, running_count, last_discovered_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (integration_id, komodo_id)
		DO UPDATE SET
			name = EXCLUDED.name,
			server_id = EXCLUDED.server_id,
			server_name = EXCLUDED.server_name,
			container_count = EXCLUDED.container_count,
			running_count = EXCLUDED.running_count,
			last_discovered_at = EXCLUDED.last_discovered_at,
			updated_at = EXCLUDED.updated_at
	`,
		stack.ID, stack.OrgID, stack.IntegrationID, stack.KomodoID,
		stack.Name, stack.ServerID, stack.ServerName,
		stack.ContainerCount, stack.RunningCount,
		stack.LastDiscoveredAt, stack.CreatedAt, stack.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert komodo stack: %w", err)
	}
	return nil
}

// scanKomodoStack scans a Komodo stack from a multi-row result.
func scanKomodoStack(rows pgx.Rows) (*models.KomodoStack, error) {
	var s models.KomodoStack

	err := rows.Scan(
		&s.ID, &s.OrgID, &s.IntegrationID, &s.KomodoID, &s.Name,
		&s.ServerID, &s.ServerName, &s.ContainerCount, &s.RunningCount,
		&s.LastDiscoveredAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan komodo stack: %w", err)
	}

	return &s, nil
}

// scanKomodoStackRow scans a Komodo stack from a single row.
func scanKomodoStackRow(row pgx.Row) (*models.KomodoStack, error) {
	var s models.KomodoStack

	err := row.Scan(
		&s.ID, &s.OrgID, &s.IntegrationID, &s.KomodoID, &s.Name,
		&s.ServerID, &s.ServerName, &s.ContainerCount, &s.RunningCount,
		&s.LastDiscoveredAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan komodo stack: %w", err)
	}

	return &s, nil
}

// --- Komodo Webhook Event methods ---

// CreateKomodoWebhookEvent creates a new Komodo webhook event.
func (db *DB) CreateKomodoWebhookEvent(ctx context.Context, event *models.KomodoWebhookEvent) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO komodo_webhook_events (
			id, org_id, integration_id, event_type, payload, status,
			error_message, processed_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		event.ID, event.OrgID, event.IntegrationID, string(event.EventType),
		event.Payload, string(event.Status), event.ErrorMessage,
		event.ProcessedAt, event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create komodo webhook event: %w", err)
	}
	return nil
}

// GetKomodoWebhookEventsByOrgID returns recent webhook events for an organization.
func (db *DB) GetKomodoWebhookEventsByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.KomodoWebhookEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, integration_id, event_type, payload, status,
		       error_message, processed_at, created_at
		FROM komodo_webhook_events
		WHERE org_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("list komodo webhook events: %w", err)
	}
	defer rows.Close()

	var events []*models.KomodoWebhookEvent
	for rows.Next() {
		var e models.KomodoWebhookEvent
		var eventTypeStr string
		var statusStr string

		err := rows.Scan(
			&e.ID, &e.OrgID, &e.IntegrationID, &eventTypeStr, &e.Payload,
			&statusStr, &e.ErrorMessage, &e.ProcessedAt, &e.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan komodo webhook event: %w", err)
		}
		e.EventType = models.KomodoWebhookEventType(eventTypeStr)
		e.Status = models.KomodoWebhookEventStatus(statusStr)
		events = append(events, &e)
	}

	return events, nil
}

// UpdateKomodoWebhookEvent updates an existing Komodo webhook event.
func (db *DB) UpdateKomodoWebhookEvent(ctx context.Context, event *models.KomodoWebhookEvent) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE komodo_webhook_events
		SET status = $2, error_message = $3, processed_at = $4
		WHERE id = $1
	`,
		event.ID, string(event.Status), event.ErrorMessage, event.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("update komodo webhook event: %w", err)
	}
	return nil
}
