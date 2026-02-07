package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// Database Connection methods

// GetDatabaseConnectionsByOrgID returns all database connections for an organization.
func (db *DB) GetDatabaseConnectionsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DatabaseConnection, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, name, type, host, port, username, credentials_encrypted,
		       ssl_mode, enabled, health_status, last_health_check, last_health_error,
		       version, metadata, created_by, created_at, updated_at
		FROM database_connections
		WHERE org_id = $1
		ORDER BY name ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query database connections: %w", err)
	}
	defer rows.Close()

	return scanDatabaseConnections(rows)
}

// GetDatabaseConnectionsByAgentID returns all database connections for a specific agent.
func (db *DB) GetDatabaseConnectionsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DatabaseConnection, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, name, type, host, port, username, credentials_encrypted,
		       ssl_mode, enabled, health_status, last_health_check, last_health_error,
		       version, metadata, created_by, created_at, updated_at
		FROM database_connections
		WHERE agent_id = $1 AND enabled = true
		ORDER BY name ASC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("query database connections by agent: %w", err)
	}
	defer rows.Close()

	return scanDatabaseConnections(rows)
}

// GetDatabaseConnectionByID returns a database connection by ID.
func (db *DB) GetDatabaseConnectionByID(ctx context.Context, id uuid.UUID) (*models.DatabaseConnection, error) {
	var c models.DatabaseConnection
	var typeStr, healthStatusStr string
	var metadataBytes []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, agent_id, name, type, host, port, username, credentials_encrypted,
		       ssl_mode, enabled, health_status, last_health_check, last_health_error,
		       version, metadata, created_by, created_at, updated_at
		FROM database_connections
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.OrgID, &c.AgentID, &c.Name, &typeStr, &c.Host, &c.Port, &c.Username, &c.CredentialsEncrypted,
		&c.SSLMode, &c.Enabled, &healthStatusStr, &c.LastHealthCheck, &c.LastHealthError,
		&c.Version, &metadataBytes, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get database connection: %w", err)
	}
	c.Type = models.DatabaseType(typeStr)
	c.HealthStatus = models.DatabaseConnectionHealthStatus(healthStatusStr)
	if err := c.SetMetadata(metadataBytes); err != nil {
		db.logger.Warn().Err(err).Str("connection_id", c.ID.String()).Msg("failed to parse connection metadata")
	}
	return &c, nil
}

// CreateDatabaseConnection creates a new database connection.
func (db *DB) CreateDatabaseConnection(ctx context.Context, conn *models.DatabaseConnection) error {
	metadataBytes, err := conn.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO database_connections (id, org_id, agent_id, name, type, host, port, username, credentials_encrypted,
		    ssl_mode, enabled, health_status, metadata, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, conn.ID, conn.OrgID, conn.AgentID, conn.Name, string(conn.Type), conn.Host, conn.Port, conn.Username, conn.CredentialsEncrypted,
		conn.SSLMode, conn.Enabled, string(conn.HealthStatus), metadataBytes, conn.CreatedBy, conn.CreatedAt, conn.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create database connection: %w", err)
	}
	return nil
}

// UpdateDatabaseConnection updates an existing database connection.
func (db *DB) UpdateDatabaseConnection(ctx context.Context, conn *models.DatabaseConnection) error {
	metadataBytes, err := conn.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE database_connections
		SET name = $2, type = $3, host = $4, port = $5, username = $6, ssl_mode = $7,
		    enabled = $8, metadata = $9, updated_at = $10
		WHERE id = $1
	`, conn.ID, conn.Name, string(conn.Type), conn.Host, conn.Port, conn.Username, conn.SSLMode,
		conn.Enabled, metadataBytes, time.Now())
	if err != nil {
		return fmt.Errorf("update database connection: %w", err)
	}
	return nil
}

// UpdateDatabaseConnectionCredentials updates the encrypted credentials of a connection.
func (db *DB) UpdateDatabaseConnectionCredentials(ctx context.Context, id uuid.UUID, credentialsEncrypted []byte) error {
	_, err := db.Pool.Exec(ctx, `
		UPDATE database_connections
		SET credentials_encrypted = $2, updated_at = $3
		WHERE id = $1
	`, id, credentialsEncrypted, time.Now())
	if err != nil {
		return fmt.Errorf("update database connection credentials: %w", err)
	}
	return nil
}

// DeleteDatabaseConnection deletes a database connection.
func (db *DB) DeleteDatabaseConnection(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM database_connections WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete database connection: %w", err)
	}
	return nil
}

// UpdateDatabaseConnectionHealth updates the health status of a database connection.
func (db *DB) UpdateDatabaseConnectionHealth(ctx context.Context, id uuid.UUID, status models.DatabaseConnectionHealthStatus, version *string, errorMsg *string) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE database_connections
		SET health_status = $2, last_health_check = $3, last_health_error = $4, version = $5, updated_at = $6
		WHERE id = $1
	`, id, string(status), now, errorMsg, version, now)
	if err != nil {
		return fmt.Errorf("update database connection health: %w", err)
	}
	return nil
}

// GetDatabaseConnectionsWithHealthCheck returns connections that need a health check.
func (db *DB) GetDatabaseConnectionsWithHealthCheck(ctx context.Context, orgID uuid.UUID, olderThan time.Duration) ([]*models.DatabaseConnection, error) {
	threshold := time.Now().Add(-olderThan)
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, name, type, host, port, username, credentials_encrypted,
		       ssl_mode, enabled, health_status, last_health_check, last_health_error,
		       version, metadata, created_by, created_at, updated_at
		FROM database_connections
		WHERE org_id = $1 AND enabled = true
		  AND (last_health_check IS NULL OR last_health_check < $2)
		ORDER BY last_health_check ASC NULLS FIRST
	`, orgID, threshold)
	if err != nil {
		return nil, fmt.Errorf("query database connections for health check: %w", err)
	}
	defer rows.Close()

	return scanDatabaseConnections(rows)
}

// CountDatabaseConnectionsByOrgID returns the count of database connections for an organization.
func (db *DB) CountDatabaseConnectionsByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM database_connections WHERE org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count database connections: %w", err)
	}
	return count, nil
}

// scanDatabaseConnections scans rows into DatabaseConnection objects.
func scanDatabaseConnections(rows interface{ Next() bool; Scan(dest ...interface{}) error; Err() error }) ([]*models.DatabaseConnection, error) {
	var connections []*models.DatabaseConnection
	for rows.Next() {
		var c models.DatabaseConnection
		var typeStr, healthStatusStr string
		var metadataBytes []byte
		err := rows.Scan(
			&c.ID, &c.OrgID, &c.AgentID, &c.Name, &typeStr, &c.Host, &c.Port, &c.Username, &c.CredentialsEncrypted,
			&c.SSLMode, &c.Enabled, &healthStatusStr, &c.LastHealthCheck, &c.LastHealthError,
			&c.Version, &metadataBytes, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan database connection: %w", err)
		}
		c.Type = models.DatabaseType(typeStr)
		c.HealthStatus = models.DatabaseConnectionHealthStatus(healthStatusStr)
		_ = c.SetMetadata(metadataBytes)
		connections = append(connections, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate database connections: %w", err)
	}
	return connections, nil
}
