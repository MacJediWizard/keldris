package db

import (
	"context"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
)

// CreateProxmoxConnection creates a new Proxmox connection record.
func (db *DB) CreateProxmoxConnection(ctx context.Context, conn *models.ProxmoxConnection) error {
	query := `
		INSERT INTO proxmox_connections (
			id, org_id, name, host, port, node, username,
			token_id, token_secret_encrypted, verify_ssl, enabled,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := db.Pool.Exec(ctx, query,
		conn.ID,
		conn.OrgID,
		conn.Name,
		conn.Host,
		conn.Port,
		conn.Node,
		conn.Username,
		conn.TokenID,
		conn.TokenSecretEncrypted,
		conn.VerifySSL,
		conn.Enabled,
		conn.CreatedAt,
		conn.UpdatedAt,
	)

	return err
}

// GetProxmoxConnectionByID retrieves a Proxmox connection by its ID.
func (db *DB) GetProxmoxConnectionByID(ctx context.Context, id uuid.UUID) (*models.ProxmoxConnection, error) {
	query := `
		SELECT id, org_id, name, host, port, node, username,
			token_id, token_secret_encrypted, verify_ssl, enabled,
			last_connected_at, created_at, updated_at
		FROM proxmox_connections
		WHERE id = $1
	`

	conn := &models.ProxmoxConnection{}
	err := db.Pool.QueryRow(ctx, query, id).Scan(
		&conn.ID,
		&conn.OrgID,
		&conn.Name,
		&conn.Host,
		&conn.Port,
		&conn.Node,
		&conn.Username,
		&conn.TokenID,
		&conn.TokenSecretEncrypted,
		&conn.VerifySSL,
		&conn.Enabled,
		&conn.LastConnectedAt,
		&conn.CreatedAt,
		&conn.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// GetProxmoxConnectionsByOrgID retrieves all Proxmox connections for an organization.
func (db *DB) GetProxmoxConnectionsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ProxmoxConnection, error) {
	query := `
		SELECT id, org_id, name, host, port, node, username,
			token_id, token_secret_encrypted, verify_ssl, enabled,
			last_connected_at, created_at, updated_at
		FROM proxmox_connections
		WHERE org_id = $1
		ORDER BY name ASC
	`

	rows, err := db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []*models.ProxmoxConnection
	for rows.Next() {
		conn := &models.ProxmoxConnection{}
		err := rows.Scan(
			&conn.ID,
			&conn.OrgID,
			&conn.Name,
			&conn.Host,
			&conn.Port,
			&conn.Node,
			&conn.Username,
			&conn.TokenID,
			&conn.TokenSecretEncrypted,
			&conn.VerifySSL,
			&conn.Enabled,
			&conn.LastConnectedAt,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}

	return connections, nil
}

// UpdateProxmoxConnection updates an existing Proxmox connection.
func (db *DB) UpdateProxmoxConnection(ctx context.Context, conn *models.ProxmoxConnection) error {
	conn.UpdatedAt = time.Now()

	query := `
		UPDATE proxmox_connections SET
			name = $2,
			host = $3,
			port = $4,
			node = $5,
			username = $6,
			token_id = $7,
			token_secret_encrypted = $8,
			verify_ssl = $9,
			enabled = $10,
			last_connected_at = $11,
			updated_at = $12
		WHERE id = $1
	`

	_, err := db.Pool.Exec(ctx, query,
		conn.ID,
		conn.Name,
		conn.Host,
		conn.Port,
		conn.Node,
		conn.Username,
		conn.TokenID,
		conn.TokenSecretEncrypted,
		conn.VerifySSL,
		conn.Enabled,
		conn.LastConnectedAt,
		conn.UpdatedAt,
	)

	return err
}

// DeleteProxmoxConnection removes a Proxmox connection.
func (db *DB) DeleteProxmoxConnection(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM proxmox_connections WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, id)
	return err
}

// GetEnabledProxmoxConnectionsByOrgID retrieves all enabled Proxmox connections for an organization.
func (db *DB) GetEnabledProxmoxConnectionsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ProxmoxConnection, error) {
	query := `
		SELECT id, org_id, name, host, port, node, username,
			token_id, token_secret_encrypted, verify_ssl, enabled,
			last_connected_at, created_at, updated_at
		FROM proxmox_connections
		WHERE org_id = $1 AND enabled = true
		ORDER BY name ASC
	`

	rows, err := db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var connections []*models.ProxmoxConnection
	for rows.Next() {
		conn := &models.ProxmoxConnection{}
		err := rows.Scan(
			&conn.ID,
			&conn.OrgID,
			&conn.Name,
			&conn.Host,
			&conn.Port,
			&conn.Node,
			&conn.Username,
			&conn.TokenID,
			&conn.TokenSecretEncrypted,
			&conn.VerifySSL,
			&conn.Enabled,
			&conn.LastConnectedAt,
			&conn.CreatedAt,
			&conn.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		connections = append(connections, conn)
	}

	return connections, nil
}
