package handlers

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/migration"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MigrationStore defines the interface for migration-related persistence operations.
type MigrationStore interface {
	// Organization operations
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)
	GetOrganizationBySlug(ctx context.Context, slug string) (*models.Organization, error)
	CreateOrganization(ctx context.Context, org *models.Organization) error

	// User operations
	GetAllUsers(ctx context.Context) ([]*models.User, error)
	GetUsersByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error

	// Agent operations
	GetAllAgents(ctx context.Context) ([]*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	CreateAgent(ctx context.Context, agent *models.Agent) error

	// Repository operations
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	CreateRepository(ctx context.Context, repo *models.Repository) error

	// Schedule operations
	GetSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Schedule, error)
	GetSchedulesByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.Schedule, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	CreateSchedule(ctx context.Context, schedule *models.Schedule) error
	SetScheduleRepositories(ctx context.Context, scheduleID uuid.UUID, repos []models.ScheduleRepository) error

	// Policy operations
	GetPoliciesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Policy, error)
	GetPolicyByID(ctx context.Context, id uuid.UUID) (*models.Policy, error)
	CreatePolicy(ctx context.Context, policy *models.Policy) error

	// Agent group operations
	GetAgentGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentGroup, error)
	GetAgentGroupByID(ctx context.Context, id uuid.UUID) (*models.AgentGroup, error)
	GetGroupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.AgentGroup, error)
	CreateAgentGroup(ctx context.Context, group *models.AgentGroup) error
	AddAgentToGroup(ctx context.Context, groupID, agentID uuid.UUID) error

	// System settings
	GetSystemSettings(ctx context.Context) ([]*models.SystemSetting, error)

	// Audit log
	CreateSuperuserAuditLog(ctx context.Context, log *models.SuperuserAuditLog) error
}

// MigrationHandler handles migration export/import HTTP endpoints.
type MigrationHandler struct {
	store    MigrationStore
	exporter *migration.Exporter
	importer *migration.Importer
	sessions *auth.SessionStore
	logger   zerolog.Logger
}

// NewMigrationHandler creates a new MigrationHandler.
func NewMigrationHandler(store MigrationStore, sessions *auth.SessionStore, logger zerolog.Logger) *MigrationHandler {
	return &MigrationHandler{
		store:    store,
		exporter: migration.NewExporter(store, logger),
		importer: migration.NewImporter(store, logger),
		sessions: sessions,
		logger:   logger.With().Str("component", "migration_handler").Logger(),
	}
}

// RegisterRoutes registers migration routes on the given router group.
// These routes require superuser privileges.
func (h *MigrationHandler) RegisterRoutes(r *gin.RouterGroup) {
	mig := r.Group("/migration")
	mig.Use(middleware.SuperuserMiddleware(h.sessions, h.logger))
	{
		// Export
		mig.POST("/export", h.Export)
		mig.POST("/export/generate-key", h.GenerateExportKey)

		// Import
		mig.POST("/import", h.Import)
		mig.POST("/import/validate", h.ValidateImport)
	}
}

// ExportRequest is the request body for exporting.
type ExportRequest struct {
	IncludeSecrets      bool   `json:"include_secrets"`
	IncludeSystemConfig bool   `json:"include_system_config"`
	EncryptionKey       string `json:"encryption_key,omitempty"`
	Description         string `json:"description,omitempty"`
}

// ExportResponse is the response for export operations.
type ExportResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message,omitempty"`
	Summary   *ExportSummary `json:"summary,omitempty"`
	ExportData string     `json:"export_data,omitempty"`
}

// ExportSummary provides a summary of the export.
type ExportSummary struct {
	Organizations int  `json:"organizations"`
	Users         int  `json:"users"`
	Agents        int  `json:"agents"`
	Repositories  int  `json:"repositories"`
	Schedules     int  `json:"schedules"`
	Policies      int  `json:"policies"`
	Encrypted     bool `json:"encrypted"`
	SecretsOmitted bool `json:"secrets_omitted"`
}

// Export exports the entire system configuration.
// POST /api/v1/migration/export
func (h *MigrationHandler) Export(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	var req ExportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body with defaults
		req = ExportRequest{}
	}

	// Parse encryption key if provided
	var encryptionKey []byte
	if req.EncryptionKey != "" {
		key, err := migration.KeyFromBase64(req.EncryptionKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encryption key format"})
			return
		}
		encryptionKey = key
	}

	opts := migration.ExportOptions{
		IncludeSecrets:      req.IncludeSecrets,
		IncludeSystemConfig: req.IncludeSystemConfig,
		EncryptionKey:       encryptionKey,
		Description:         req.Description,
		ExportedBy:          user.Email,
	}

	// Perform export
	data, err := h.exporter.ExportToJSON(c.Request.Context(), opts)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to export system configuration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export: " + err.Error()})
		return
	}

	// Log the export action
	h.logAction(c, user.ID, models.SuperuserActionExport, "migration", nil, nil)

	// Set headers for file download
	filename := "keldris-migration-" + time.Now().Format("2006-01-02-150405") + ".json"
	if len(encryptionKey) > 0 {
		filename = "keldris-migration-" + time.Now().Format("2006-01-02-150405") + ".encrypted"
	}

	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/octet-stream", data)
}

// GenerateExportKeyResponse is the response for generating an export key.
type GenerateExportKeyResponse struct {
	Key string `json:"key"`
}

// GenerateExportKey generates a new encryption key for exports.
// POST /api/v1/migration/export/generate-key
func (h *MigrationHandler) GenerateExportKey(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	key, err := migration.GenerateEncryptionKey()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate encryption key")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}

	c.JSON(http.StatusOK, GenerateExportKeyResponse{
		Key: migration.KeyToBase64(key),
	})
}

// ImportRequest is the request body for importing via JSON.
type ImportRequestBody struct {
	Data               string `json:"data"`
	DecryptionKey      string `json:"decryption_key,omitempty"`
	ConflictResolution string `json:"conflict_resolution,omitempty"`
	DryRun             bool   `json:"dry_run"`
	TargetOrgSlug      string `json:"target_org_slug,omitempty"`
}

// Import imports a migration export file.
// POST /api/v1/migration/import
func (h *MigrationHandler) Import(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	var data []byte
	var req ImportRequestBody

	// Try to read as multipart form first
	file, _, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read file"})
			return
		}
		// Get other form fields
		req.DecryptionKey = c.PostForm("decryption_key")
		req.ConflictResolution = c.PostForm("conflict_resolution")
		req.DryRun = c.PostForm("dry_run") == "true"
		req.TargetOrgSlug = c.PostForm("target_org_slug")
	} else {
		// Try JSON body
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
			return
		}
		data = []byte(req.Data)
	}

	if len(data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no import data provided"})
		return
	}

	// Parse decryption key if provided
	var decryptionKey []byte
	if req.DecryptionKey != "" {
		key, err := migration.KeyFromBase64(req.DecryptionKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decryption key format"})
			return
		}
		decryptionKey = key
	}

	// Parse the export
	export, err := h.importer.Parse(data, decryptionKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse export: " + err.Error()})
		return
	}

	// Set conflict resolution
	conflictRes := migration.ConflictResolutionSkip
	if req.ConflictResolution != "" {
		conflictRes = migration.ConflictResolution(req.ConflictResolution)
	}

	// Perform import
	importReq := migration.ImportRequest{
		Data:               data,
		DecryptionKey:      decryptionKey,
		ConflictResolution: conflictRes,
		DryRun:             req.DryRun,
		TargetOrgSlug:      req.TargetOrgSlug,
	}

	result, err := h.importer.Import(c.Request.Context(), export, importReq)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to import migration")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import: " + err.Error()})
		return
	}

	// Log the import action (only if not dry run)
	if !req.DryRun {
		h.logAction(c, user.ID, models.SuperuserActionImport, "migration", nil, nil)
	}

	if result.Success {
		c.JSON(http.StatusOK, result)
	} else {
		c.JSON(http.StatusUnprocessableEntity, result)
	}
}

// MigrationValidateRequest is the request body for validating a migration import.
type MigrationValidateRequest struct {
	Data          string `json:"data" binding:"required"`
	DecryptionKey string `json:"decryption_key,omitempty"`
}

// ValidateImport validates a migration export without importing.
// POST /api/v1/migration/import/validate
func (h *MigrationHandler) ValidateImport(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	var req MigrationValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Parse decryption key if provided
	var decryptionKey []byte
	if req.DecryptionKey != "" {
		key, err := migration.KeyFromBase64(req.DecryptionKey)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decryption key format"})
			return
		}
		decryptionKey = key
	}

	// Parse the export
	export, err := h.importer.Parse([]byte(req.Data), decryptionKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse export: " + err.Error()})
		return
	}

	// Validate
	result, err := h.importer.Validate(c.Request.Context(), export)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "validation failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// logAction logs a superuser action to the audit log.
func (h *MigrationHandler) logAction(c *gin.Context, userID uuid.UUID, action models.SuperuserAction, targetType string, targetID, targetOrgID *uuid.UUID) {
	log := models.NewSuperuserAuditLog(userID, action, targetType).
		WithRequestInfo(c.ClientIP(), c.Request.UserAgent())

	if targetID != nil {
		log.WithTargetID(*targetID)
	}
	if targetOrgID != nil {
		log.WithTargetOrgID(*targetOrgID)
	}

	if err := h.store.CreateSuperuserAuditLog(c.Request.Context(), log); err != nil {
		h.logger.Warn().Err(err).Msg("failed to create audit log")
	}
}
