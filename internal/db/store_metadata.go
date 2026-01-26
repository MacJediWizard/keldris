package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/metadata"
	"github.com/google/uuid"
)

// GetMetadataSchemasByOrgAndEntity returns all metadata schemas for an organization and entity type.
func (db *DB) GetMetadataSchemasByOrgAndEntity(ctx context.Context, orgID uuid.UUID, entityType metadata.EntityType) ([]*metadata.Schema, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, entity_type, name, field_key, field_type, description,
		       required, default_value, options, validation, display_order,
		       created_at, updated_at
		FROM metadata_schemas
		WHERE org_id = $1 AND entity_type = $2
		ORDER BY display_order, name
	`, orgID, string(entityType))
	if err != nil {
		return nil, fmt.Errorf("list metadata schemas: %w", err)
	}
	defer rows.Close()

	var schemas []*metadata.Schema
	for rows.Next() {
		var s metadata.Schema
		var entityTypeStr, fieldTypeStr string
		var defaultValueBytes, optionsBytes, validationBytes []byte
		var description *string
		err := rows.Scan(
			&s.ID, &s.OrgID, &entityTypeStr, &s.Name, &s.FieldKey, &fieldTypeStr,
			&description, &s.Required, &defaultValueBytes, &optionsBytes, &validationBytes,
			&s.DisplayOrder, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan metadata schema: %w", err)
		}
		s.EntityType = metadata.EntityType(entityTypeStr)
		s.FieldType = metadata.FieldType(fieldTypeStr)
		if description != nil {
			s.Description = *description
		}
		if err := s.SetDefaultValueJSON(defaultValueBytes); err != nil {
			db.logger.Warn().Err(err).Str("schema_id", s.ID.String()).Msg("failed to parse default value")
		}
		if err := s.SetOptionsJSON(optionsBytes); err != nil {
			db.logger.Warn().Err(err).Str("schema_id", s.ID.String()).Msg("failed to parse options")
		}
		if err := s.SetValidationJSON(validationBytes); err != nil {
			db.logger.Warn().Err(err).Str("schema_id", s.ID.String()).Msg("failed to parse validation")
		}
		schemas = append(schemas, &s)
	}

	return schemas, nil
}

// GetMetadataSchemaByID returns a metadata schema by ID.
func (db *DB) GetMetadataSchemaByID(ctx context.Context, id uuid.UUID) (*metadata.Schema, error) {
	var s metadata.Schema
	var entityTypeStr, fieldTypeStr string
	var defaultValueBytes, optionsBytes, validationBytes []byte
	var description *string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, entity_type, name, field_key, field_type, description,
		       required, default_value, options, validation, display_order,
		       created_at, updated_at
		FROM metadata_schemas
		WHERE id = $1
	`, id).Scan(
		&s.ID, &s.OrgID, &entityTypeStr, &s.Name, &s.FieldKey, &fieldTypeStr,
		&description, &s.Required, &defaultValueBytes, &optionsBytes, &validationBytes,
		&s.DisplayOrder, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get metadata schema: %w", err)
	}
	s.EntityType = metadata.EntityType(entityTypeStr)
	s.FieldType = metadata.FieldType(fieldTypeStr)
	if description != nil {
		s.Description = *description
	}
	if err := s.SetDefaultValueJSON(defaultValueBytes); err != nil {
		db.logger.Warn().Err(err).Str("schema_id", s.ID.String()).Msg("failed to parse default value")
	}
	if err := s.SetOptionsJSON(optionsBytes); err != nil {
		db.logger.Warn().Err(err).Str("schema_id", s.ID.String()).Msg("failed to parse options")
	}
	if err := s.SetValidationJSON(validationBytes); err != nil {
		db.logger.Warn().Err(err).Str("schema_id", s.ID.String()).Msg("failed to parse validation")
	}
	return &s, nil
}

// CreateMetadataSchema creates a new metadata schema.
func (db *DB) CreateMetadataSchema(ctx context.Context, schema *metadata.Schema) error {
	defaultValueBytes, err := schema.DefaultValueJSON()
	if err != nil {
		return fmt.Errorf("marshal default value: %w", err)
	}
	optionsBytes, err := schema.OptionsJSON()
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}
	validationBytes, err := schema.ValidationJSON()
	if err != nil {
		return fmt.Errorf("marshal validation: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO metadata_schemas (
			id, org_id, entity_type, name, field_key, field_type, description,
			required, default_value, options, validation, display_order,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, schema.ID, schema.OrgID, string(schema.EntityType), schema.Name, schema.FieldKey,
		string(schema.FieldType), schema.Description, schema.Required, defaultValueBytes,
		optionsBytes, validationBytes, schema.DisplayOrder, schema.CreatedAt, schema.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create metadata schema: %w", err)
	}
	return nil
}

// UpdateMetadataSchema updates an existing metadata schema.
func (db *DB) UpdateMetadataSchema(ctx context.Context, schema *metadata.Schema) error {
	defaultValueBytes, err := schema.DefaultValueJSON()
	if err != nil {
		return fmt.Errorf("marshal default value: %w", err)
	}
	optionsBytes, err := schema.OptionsJSON()
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}
	validationBytes, err := schema.ValidationJSON()
	if err != nil {
		return fmt.Errorf("marshal validation: %w", err)
	}

	schema.UpdatedAt = time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE metadata_schemas
		SET name = $2, field_key = $3, field_type = $4, description = $5,
		    required = $6, default_value = $7, options = $8, validation = $9,
		    display_order = $10, updated_at = $11
		WHERE id = $1
	`, schema.ID, schema.Name, schema.FieldKey, string(schema.FieldType), schema.Description,
		schema.Required, defaultValueBytes, optionsBytes, validationBytes,
		schema.DisplayOrder, schema.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update metadata schema: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("metadata schema not found")
	}
	return nil
}

// DeleteMetadataSchema deletes a metadata schema by ID.
func (db *DB) DeleteMetadataSchema(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `DELETE FROM metadata_schemas WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete metadata schema: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("metadata schema not found")
	}
	return nil
}

// UpdateAgentMetadata updates an agent's metadata.
func (db *DB) UpdateAgentMetadata(ctx context.Context, agentID uuid.UUID, metadata map[string]interface{}) error {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	result, err := db.Pool.Exec(ctx, `
		UPDATE agents
		SET metadata = $2, updated_at = $3
		WHERE id = $1
	`, agentID, metadataBytes, time.Now())
	if err != nil {
		return fmt.Errorf("update agent metadata: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("agent not found")
	}
	return nil
}

// UpdateRepositoryMetadata updates a repository's metadata.
func (db *DB) UpdateRepositoryMetadata(ctx context.Context, repoID uuid.UUID, metadata map[string]interface{}) error {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	result, err := db.Pool.Exec(ctx, `
		UPDATE repositories
		SET metadata = $2, updated_at = $3
		WHERE id = $1
	`, repoID, metadataBytes, time.Now())
	if err != nil {
		return fmt.Errorf("update repository metadata: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("repository not found")
	}
	return nil
}

// UpdateScheduleMetadata updates a schedule's metadata.
func (db *DB) UpdateScheduleMetadata(ctx context.Context, scheduleID uuid.UUID, metadata map[string]interface{}) error {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	result, err := db.Pool.Exec(ctx, `
		UPDATE schedules
		SET metadata = $2, updated_at = $3
		WHERE id = $1
	`, scheduleID, metadataBytes, time.Now())
	if err != nil {
		return fmt.Errorf("update schedule metadata: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("schedule not found")
	}
	return nil
}

// SearchAgentsByMetadata searches for agents matching metadata criteria.
func (db *DB) SearchAgentsByMetadata(ctx context.Context, orgID uuid.UUID, key, value string) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id FROM agents
		WHERE org_id = $1 AND metadata @> $2::jsonb
	`, orgID, fmt.Sprintf(`{"%s": "%s"}`, key, value))
	if err != nil {
		return nil, fmt.Errorf("search agents by metadata: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan agent id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// SearchRepositoriesByMetadata searches for repositories matching metadata criteria.
func (db *DB) SearchRepositoriesByMetadata(ctx context.Context, orgID uuid.UUID, key, value string) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id FROM repositories
		WHERE org_id = $1 AND metadata @> $2::jsonb
	`, orgID, fmt.Sprintf(`{"%s": "%s"}`, key, value))
	if err != nil {
		return nil, fmt.Errorf("search repositories by metadata: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan repository id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// SearchSchedulesByMetadata searches for schedules matching metadata criteria.
func (db *DB) SearchSchedulesByMetadata(ctx context.Context, orgID uuid.UUID, key, value string) ([]uuid.UUID, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT s.id FROM schedules s
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND s.metadata @> $2::jsonb
	`, orgID, fmt.Sprintf(`{"%s": "%s"}`, key, value))
	if err != nil {
		return nil, fmt.Errorf("search schedules by metadata: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan schedule id: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
