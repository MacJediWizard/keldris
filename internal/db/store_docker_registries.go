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

// Docker Registry methods

// GetDockerRegistriesByOrgID returns all Docker registries for an organization.
func (db *DB) GetDockerRegistriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerRegistry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata,
		       created_by, created_at, updated_at
		FROM docker_registries
		WHERE org_id = $1
		ORDER BY name
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list docker registries: %w", err)
	}
	defer rows.Close()

	var registries []*models.DockerRegistry
	for rows.Next() {
		r, err := scanDockerRegistry(rows)
		if err != nil {
			return nil, err
		}
		registries = append(registries, r)
	}

	return registries, nil
}

// GetDockerRegistryByID returns a Docker registry by ID.
func (db *DB) GetDockerRegistryByID(ctx context.Context, id uuid.UUID) (*models.DockerRegistry, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata,
		       created_by, created_at, updated_at
		FROM docker_registries
		WHERE id = $1
	`, id)

	return scanDockerRegistryRow(row)
}

// GetDefaultDockerRegistry returns the default Docker registry for an organization.
func (db *DB) GetDefaultDockerRegistry(ctx context.Context, orgID uuid.UUID) (*models.DockerRegistry, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata,
		       created_by, created_at, updated_at
		FROM docker_registries
		WHERE org_id = $1 AND is_default = true
	`, orgID)

	return scanDockerRegistryRow(row)
}

// CreateDockerRegistry creates a new Docker registry.
func (db *DB) CreateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error {
	metadataJSON, err := registry.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO docker_registries (
			id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
			health_status, last_health_check, last_health_error,
			credentials_rotated_at, credentials_expires_at, metadata,
			created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`,
		registry.ID, registry.OrgID, registry.Name, string(registry.Type), registry.URL,
		registry.CredentialsEncrypted, registry.IsDefault, registry.Enabled,
		string(registry.HealthStatus), registry.LastHealthCheck, registry.LastHealthError,
		registry.CredentialsRotatedAt, registry.CredentialsExpiresAt, metadataJSON,
		registry.CreatedBy, registry.CreatedAt, registry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create docker registry: %w", err)
	}

	return nil
}

// UpdateDockerRegistry updates an existing Docker registry.
func (db *DB) UpdateDockerRegistry(ctx context.Context, registry *models.DockerRegistry) error {
	registry.UpdatedAt = time.Now()

	metadataJSON, err := registry.MetadataJSON()
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE docker_registries SET
			name = $2, url = $3, enabled = $4, metadata = $5, updated_at = $6
		WHERE id = $1
	`,
		registry.ID, registry.Name, registry.URL, registry.Enabled, metadataJSON, registry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update docker registry: %w", err)
	}

	return nil
}

// DeleteDockerRegistry deletes a Docker registry by ID.
func (db *DB) DeleteDockerRegistry(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM docker_registries WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete docker registry: %w", err)
	}
	return nil
}

// UpdateDockerRegistryHealth updates the health status of a Docker registry.
func (db *DB) UpdateDockerRegistryHealth(ctx context.Context, id uuid.UUID, status models.DockerRegistryHealthStatus, errorMsg *string) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE docker_registries
		SET health_status = $2, last_health_check = $3, last_health_error = $4, updated_at = $5
		WHERE id = $1
	`, id, string(status), now, errorMsg, now)
	if err != nil {
		return fmt.Errorf("update docker registry health: %w", err)
	}
	return nil
}

// UpdateDockerRegistryCredentials updates the encrypted credentials and optional expiry for a Docker registry.
func (db *DB) UpdateDockerRegistryCredentials(ctx context.Context, id uuid.UUID, credentialsEncrypted []byte, expiresAt *time.Time) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE docker_registries
		SET credentials_encrypted = $2, credentials_rotated_at = $3, credentials_expires_at = $4, updated_at = $5
		WHERE id = $1
	`, id, credentialsEncrypted, now, expiresAt, now)
	if err != nil {
		return fmt.Errorf("update docker registry credentials: %w", err)
	}
	return nil
}

// SetDefaultDockerRegistry sets a registry as the default for an organization,
// clearing any existing default within a transaction.
func (db *DB) SetDefaultDockerRegistry(ctx context.Context, orgID uuid.UUID, registryID uuid.UUID) error {
	now := time.Now()
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		// Clear existing default for the org
		_, err := tx.Exec(ctx, `
			UPDATE docker_registries
			SET is_default = false, updated_at = $2
			WHERE org_id = $1 AND is_default = true
		`, orgID, now)
		if err != nil {
			return fmt.Errorf("clear default docker registry: %w", err)
		}

		// Set the new default
		_, err = tx.Exec(ctx, `
			UPDATE docker_registries
			SET is_default = true, updated_at = $2
			WHERE id = $1 AND org_id = $3
		`, registryID, now, orgID)
		if err != nil {
			return fmt.Errorf("set default docker registry: %w", err)
		}

		return nil
	})
}

// GetDockerRegistriesWithExpiringCredentials returns registries with credentials expiring before the given time.
func (db *DB) GetDockerRegistriesWithExpiringCredentials(ctx context.Context, orgID uuid.UUID, before time.Time) ([]*models.DockerRegistry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, name, type, url, credentials_encrypted, is_default, enabled,
		       health_status, last_health_check, last_health_error,
		       credentials_rotated_at, credentials_expires_at, metadata,
		       created_by, created_at, updated_at
		FROM docker_registries
		WHERE org_id = $1 AND credentials_expires_at IS NOT NULL AND credentials_expires_at <= $2
		ORDER BY credentials_expires_at ASC
	`, orgID, before)
	if err != nil {
		return nil, fmt.Errorf("list docker registries with expiring credentials: %w", err)
	}
	defer rows.Close()

	var registries []*models.DockerRegistry
	for rows.Next() {
		r, err := scanDockerRegistry(rows)
		if err != nil {
			return nil, err
		}
		registries = append(registries, r)
	}

	return registries, nil
}

// CreateDockerRegistryAuditLog creates a new audit log entry for a Docker registry action.
func (db *DB) CreateDockerRegistryAuditLog(ctx context.Context, orgID, registryID uuid.UUID, userID *uuid.UUID, action string, details map[string]interface{}, ipAddress, userAgent string) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("marshal audit log details: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO docker_registry_audit_log (
			id, org_id, registry_id, user_id, action, details, ip_address, user_agent, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`,
		uuid.New(), orgID, registryID, userID, action, detailsJSON, ipAddress, userAgent, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("create docker registry audit log: %w", err)
	}

	return nil
}

// scanDockerRegistry scans a Docker registry from a row set.
func scanDockerRegistry(rows pgx.Rows) (*models.DockerRegistry, error) {
	var r models.DockerRegistry
	var typeStr string
	var healthStatusStr string
	var metadataBytes []byte

	err := rows.Scan(
		&r.ID, &r.OrgID, &r.Name, &typeStr, &r.URL,
		&r.CredentialsEncrypted, &r.IsDefault, &r.Enabled,
		&healthStatusStr, &r.LastHealthCheck, &r.LastHealthError,
		&r.CredentialsRotatedAt, &r.CredentialsExpiresAt, &metadataBytes,
		&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan docker registry: %w", err)
	}

	r.Type = models.DockerRegistryType(typeStr)
	r.HealthStatus = models.DockerRegistryHealthStatus(healthStatusStr)

	if err := r.SetMetadata(metadataBytes); err != nil {
		return nil, fmt.Errorf("parse docker registry metadata: %w", err)
	}

	return &r, nil
}

// scanDockerRegistryRow scans a Docker registry from a single row.
func scanDockerRegistryRow(row pgx.Row) (*models.DockerRegistry, error) {
	var r models.DockerRegistry
	var typeStr string
	var healthStatusStr string
	var metadataBytes []byte

	err := row.Scan(
		&r.ID, &r.OrgID, &r.Name, &typeStr, &r.URL,
		&r.CredentialsEncrypted, &r.IsDefault, &r.Enabled,
		&healthStatusStr, &r.LastHealthCheck, &r.LastHealthError,
		&r.CredentialsRotatedAt, &r.CredentialsExpiresAt, &metadataBytes,
		&r.CreatedBy, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan docker registry: %w", err)
	}

	r.Type = models.DockerRegistryType(typeStr)
	r.HealthStatus = models.DockerRegistryHealthStatus(healthStatusStr)

	if err := r.SetMetadata(metadataBytes); err != nil {
		return nil, fmt.Errorf("parse docker registry metadata: %w", err)
	}

	return &r, nil
}
