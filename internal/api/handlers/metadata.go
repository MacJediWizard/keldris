package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/metadata"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MetadataStore defines the interface for metadata persistence operations.
type MetadataStore interface {
	GetMetadataSchemasByOrgAndEntity(ctx context.Context, orgID uuid.UUID, entityType metadata.EntityType) ([]*metadata.Schema, error)
	GetMetadataSchemaByID(ctx context.Context, id uuid.UUID) (*metadata.Schema, error)
	CreateMetadataSchema(ctx context.Context, schema *metadata.Schema) error
	UpdateMetadataSchema(ctx context.Context, schema *metadata.Schema) error
	DeleteMetadataSchema(ctx context.Context, id uuid.UUID) error
	UpdateAgentMetadata(ctx context.Context, agentID uuid.UUID, metadata map[string]interface{}) error
	UpdateRepositoryMetadata(ctx context.Context, repoID uuid.UUID, metadata map[string]interface{}) error
	UpdateScheduleMetadata(ctx context.Context, scheduleID uuid.UUID, metadata map[string]interface{}) error
	SearchAgentsByMetadata(ctx context.Context, orgID uuid.UUID, key, value string) ([]uuid.UUID, error)
	SearchRepositoriesByMetadata(ctx context.Context, orgID uuid.UUID, key, value string) ([]uuid.UUID, error)
	SearchSchedulesByMetadata(ctx context.Context, orgID uuid.UUID, key, value string) ([]uuid.UUID, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (interface{}, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (interface{}, error)
}

// MetadataHandler handles metadata-related HTTP endpoints.
type MetadataHandler struct {
	store  MetadataStore
	logger zerolog.Logger
}

// NewMetadataHandler creates a new MetadataHandler.
func NewMetadataHandler(store MetadataStore, logger zerolog.Logger) *MetadataHandler {
	return &MetadataHandler{
		store:  store,
		logger: logger.With().Str("component", "metadata_handler").Logger(),
	}
}

// RegisterRoutes registers metadata routes on the given router group.
func (h *MetadataHandler) RegisterRoutes(r *gin.RouterGroup) {
	schemas := r.Group("/metadata/schemas")
	{
		schemas.GET("", h.ListSchemas)
		schemas.POST("", h.CreateSchema)
		schemas.GET("/types", h.ListFieldTypes)
		schemas.GET("/entities", h.ListEntityTypes)
		schemas.GET("/:id", h.GetSchema)
		schemas.PUT("/:id", h.UpdateSchema)
		schemas.DELETE("/:id", h.DeleteSchema)
	}

	// Entity metadata endpoints
	r.PUT("/agents/:id/metadata", h.UpdateAgentMetadata)
	r.PUT("/repositories/:id/metadata", h.UpdateRepositoryMetadata)
	r.PUT("/schedules/:id/metadata", h.UpdateScheduleMetadata)

	// Search by metadata
	r.GET("/metadata/search", h.SearchByMetadata)
}

// CreateSchemaRequest is the request body for creating a metadata schema.
type CreateSchemaRequest struct {
	EntityType   string                    `json:"entity_type" binding:"required"`
	Name         string                    `json:"name" binding:"required"`
	FieldKey     string                    `json:"field_key" binding:"required"`
	FieldType    string                    `json:"field_type" binding:"required"`
	Description  string                    `json:"description,omitempty"`
	Required     bool                      `json:"required"`
	DefaultValue interface{}               `json:"default_value,omitempty"`
	Options      []metadata.SelectOption   `json:"options,omitempty"`
	Validation   *metadata.ValidationRules `json:"validation,omitempty"`
	DisplayOrder int                       `json:"display_order"`
}

// UpdateSchemaRequest is the request body for updating a metadata schema.
type UpdateSchemaRequest struct {
	Name         *string                   `json:"name,omitempty"`
	FieldKey     *string                   `json:"field_key,omitempty"`
	FieldType    *string                   `json:"field_type,omitempty"`
	Description  *string                   `json:"description,omitempty"`
	Required     *bool                     `json:"required,omitempty"`
	DefaultValue interface{}               `json:"default_value,omitempty"`
	Options      []metadata.SelectOption   `json:"options,omitempty"`
	Validation   *metadata.ValidationRules `json:"validation,omitempty"`
	DisplayOrder *int                      `json:"display_order,omitempty"`
}

// UpdateMetadataRequest is the request body for updating entity metadata.
type UpdateMetadataRequest struct {
	Metadata map[string]interface{} `json:"metadata" binding:"required"`
}

// SearchMetadataRequest is the request params for searching by metadata.
type SearchMetadataRequest struct {
	EntityType string `form:"entity_type" binding:"required"`
	Key        string `form:"key" binding:"required"`
	Value      string `form:"value" binding:"required"`
}

// ListSchemas returns all metadata schemas for the organization filtered by entity type.
// GET /api/v1/metadata/schemas?entity_type=agent
func (h *MetadataHandler) ListSchemas(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	entityTypeParam := c.Query("entity_type")
	if entityTypeParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type query parameter is required"})
		return
	}

	entityType := metadata.EntityType(entityTypeParam)
	if !entityType.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity_type"})
		return
	}

	schemas, err := h.store.GetMetadataSchemasByOrgAndEntity(c.Request.Context(), user.CurrentOrgID, entityType)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Str("entity_type", entityTypeParam).Msg("failed to list metadata schemas")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list metadata schemas"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schemas": schemas})
}

// GetSchema returns a specific metadata schema by ID.
// GET /api/v1/metadata/schemas/:id
func (h *MetadataHandler) GetSchema(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schema ID"})
		return
	}

	schema, err := h.store.GetMetadataSchemaByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("schema_id", id.String()).Msg("failed to get metadata schema")
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata schema not found"})
		return
	}

	// Verify schema belongs to user's org
	if schema.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata schema not found"})
		return
	}

	c.JSON(http.StatusOK, schema)
}

// CreateSchema creates a new metadata schema.
// POST /api/v1/metadata/schemas
func (h *MetadataHandler) CreateSchema(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req CreateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entityType := metadata.EntityType(req.EntityType)
	if !entityType.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity_type"})
		return
	}

	fieldType := metadata.FieldType(req.FieldType)
	if !fieldType.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid field_type"})
		return
	}

	schema := metadata.NewSchema(user.CurrentOrgID, entityType, req.Name, req.FieldKey, fieldType)
	schema.Description = req.Description
	schema.Required = req.Required
	schema.DefaultValue = req.DefaultValue
	schema.Options = req.Options
	schema.Validation = req.Validation
	schema.DisplayOrder = req.DisplayOrder

	// Validate the schema
	if err := schema.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.CreateMetadataSchema(c.Request.Context(), schema); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create metadata schema")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create metadata schema"})
		return
	}

	h.logger.Info().Str("schema_id", schema.ID.String()).Str("name", schema.Name).Msg("metadata schema created")
	c.JSON(http.StatusCreated, schema)
}

// UpdateSchema updates an existing metadata schema.
// PUT /api/v1/metadata/schemas/:id
func (h *MetadataHandler) UpdateSchema(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schema ID"})
		return
	}

	schema, err := h.store.GetMetadataSchemaByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata schema not found"})
		return
	}

	// Verify schema belongs to user's org
	if schema.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata schema not found"})
		return
	}

	var req UpdateSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		schema.Name = *req.Name
	}
	if req.FieldKey != nil {
		schema.FieldKey = *req.FieldKey
	}
	if req.FieldType != nil {
		fieldType := metadata.FieldType(*req.FieldType)
		if !fieldType.IsValid() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid field_type"})
			return
		}
		schema.FieldType = fieldType
	}
	if req.Description != nil {
		schema.Description = *req.Description
	}
	if req.Required != nil {
		schema.Required = *req.Required
	}
	if req.DefaultValue != nil {
		schema.DefaultValue = req.DefaultValue
	}
	if req.Options != nil {
		schema.Options = req.Options
	}
	if req.Validation != nil {
		schema.Validation = req.Validation
	}
	if req.DisplayOrder != nil {
		schema.DisplayOrder = *req.DisplayOrder
	}

	// Validate the updated schema
	if err := schema.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateMetadataSchema(c.Request.Context(), schema); err != nil {
		h.logger.Error().Err(err).Str("schema_id", id.String()).Msg("failed to update metadata schema")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update metadata schema"})
		return
	}

	h.logger.Info().Str("schema_id", schema.ID.String()).Msg("metadata schema updated")
	c.JSON(http.StatusOK, schema)
}

// DeleteSchema deletes a metadata schema.
// DELETE /api/v1/metadata/schemas/:id
func (h *MetadataHandler) DeleteSchema(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schema ID"})
		return
	}

	schema, err := h.store.GetMetadataSchemaByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata schema not found"})
		return
	}

	// Verify schema belongs to user's org
	if schema.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata schema not found"})
		return
	}

	if err := h.store.DeleteMetadataSchema(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("schema_id", id.String()).Msg("failed to delete metadata schema")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete metadata schema"})
		return
	}

	h.logger.Info().Str("schema_id", id.String()).Msg("metadata schema deleted")
	c.JSON(http.StatusOK, gin.H{"message": "metadata schema deleted"})
}

// ListFieldTypes returns all available field types.
// GET /api/v1/metadata/schemas/types
func (h *MetadataHandler) ListFieldTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"types": metadata.GetFieldTypeInfo()})
}

// ListEntityTypes returns all available entity types.
// GET /api/v1/metadata/schemas/entities
func (h *MetadataHandler) ListEntityTypes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"entities": metadata.GetEntityTypeInfo()})
}

// UpdateAgentMetadata updates metadata for an agent.
// PUT /api/v1/agents/:id/metadata
func (h *MetadataHandler) UpdateAgentMetadata(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent exists and belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), id)
	if err != nil || agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	var req UpdateMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate metadata against schema
	schemas, err := h.store.GetMetadataSchemasByOrgAndEntity(c.Request.Context(), user.CurrentOrgID, metadata.EntityTypeAgent)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get metadata schemas")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate metadata"})
		return
	}

	if err := metadata.ValidateMetadata(schemas, req.Metadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateAgentMetadata(c.Request.Context(), id, req.Metadata); err != nil {
		h.logger.Error().Err(err).Str("agent_id", id.String()).Msg("failed to update agent metadata")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update agent metadata"})
		return
	}

	h.logger.Info().Str("agent_id", id.String()).Msg("agent metadata updated")
	c.JSON(http.StatusOK, gin.H{"message": "agent metadata updated", "metadata": req.Metadata})
}

// UpdateRepositoryMetadata updates metadata for a repository.
// PUT /api/v1/repositories/:id/metadata
func (h *MetadataHandler) UpdateRepositoryMetadata(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	// Verify repository exists and belongs to user's org
	repo, err := h.store.GetRepositoryByID(c.Request.Context(), id)
	if err != nil || repo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	var req UpdateMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate metadata against schema
	schemas, err := h.store.GetMetadataSchemasByOrgAndEntity(c.Request.Context(), user.CurrentOrgID, metadata.EntityTypeRepository)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get metadata schemas")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate metadata"})
		return
	}

	if err := metadata.ValidateMetadata(schemas, req.Metadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateRepositoryMetadata(c.Request.Context(), id, req.Metadata); err != nil {
		h.logger.Error().Err(err).Str("repository_id", id.String()).Msg("failed to update repository metadata")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update repository metadata"})
		return
	}

	h.logger.Info().Str("repository_id", id.String()).Msg("repository metadata updated")
	c.JSON(http.StatusOK, gin.H{"message": "repository metadata updated", "metadata": req.Metadata})
}

// UpdateScheduleMetadata updates metadata for a schedule.
// PUT /api/v1/schedules/:id/metadata
func (h *MetadataHandler) UpdateScheduleMetadata(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	// Verify schedule exists and belongs to user's org
	schedule, err := h.store.GetScheduleByID(c.Request.Context(), id)
	if err != nil || schedule == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	var req UpdateMetadataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate metadata against schema
	schemas, err := h.store.GetMetadataSchemasByOrgAndEntity(c.Request.Context(), user.CurrentOrgID, metadata.EntityTypeSchedule)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get metadata schemas")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate metadata"})
		return
	}

	if err := metadata.ValidateMetadata(schemas, req.Metadata); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.store.UpdateScheduleMetadata(c.Request.Context(), id, req.Metadata); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update schedule metadata")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update schedule metadata"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Msg("schedule metadata updated")
	c.JSON(http.StatusOK, gin.H{"message": "schedule metadata updated", "metadata": req.Metadata})
}

// SearchByMetadata searches for entities by metadata key/value.
// GET /api/v1/metadata/search?entity_type=agent&key=department&value=engineering
func (h *MetadataHandler) SearchByMetadata(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req SearchMetadataRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entityType := metadata.EntityType(req.EntityType)
	if !entityType.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entity_type"})
		return
	}

	var ids []uuid.UUID
	var err error

	switch entityType {
	case metadata.EntityTypeAgent:
		ids, err = h.store.SearchAgentsByMetadata(c.Request.Context(), user.CurrentOrgID, req.Key, req.Value)
	case metadata.EntityTypeRepository:
		ids, err = h.store.SearchRepositoriesByMetadata(c.Request.Context(), user.CurrentOrgID, req.Key, req.Value)
	case metadata.EntityTypeSchedule:
		ids, err = h.store.SearchSchedulesByMetadata(c.Request.Context(), user.CurrentOrgID, req.Key, req.Value)
	}

	if err != nil {
		h.logger.Error().Err(err).Str("entity_type", req.EntityType).Str("key", req.Key).Msg("failed to search by metadata")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search by metadata"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ids": ids, "count": len(ids)})
}
