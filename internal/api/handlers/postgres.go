package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/databases"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// PostgresStore defines the interface for PostgreSQL-related persistence operations.
type PostgresStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
}

// PostgresHandler handles PostgreSQL backup-related HTTP endpoints.
type PostgresHandler struct {
	store      PostgresStore
	keyManager *crypto.KeyManager
	logger     zerolog.Logger
}

// NewPostgresHandler creates a new PostgresHandler.
func NewPostgresHandler(store PostgresStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *PostgresHandler {
	return &PostgresHandler{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "postgres_handler").Logger(),
	}
}

// RegisterRoutes registers PostgreSQL routes on the given router group.
func (h *PostgresHandler) RegisterRoutes(r *gin.RouterGroup) {
	postgres := r.Group("/postgres")
	{
		postgres.POST("/test-connection", h.TestConnection)
		postgres.GET("/restore-instructions", h.GetRestoreInstructions)
	}
}

// PostgresTestConnectionRequest is the request body for testing a PostgreSQL connection.
type PostgresTestConnectionRequest struct {
	Host     string `json:"host" binding:"required"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Database string `json:"database,omitempty"`
	SSLMode  string `json:"ssl_mode,omitempty"`
}

// PostgresTestConnectionResponse is the response for a PostgreSQL connection test.
type PostgresTestConnectionResponse struct {
	Connected bool     `json:"connected"`
	Host      string   `json:"host"`
	Port      int      `json:"port"`
	Version   string   `json:"version,omitempty"`
	Databases []string `json:"databases,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// TestConnection tests a PostgreSQL connection with the provided credentials.
//
//	@Summary		Test PostgreSQL connection
//	@Description	Tests connectivity to a PostgreSQL server with the provided credentials
//	@Tags			PostgreSQL
//	@Accept			json
//	@Produce		json
//	@Param			request	body		PostgresTestConnectionRequest	true	"Connection details"
//	@Success		200		{object}	PostgresTestConnectionResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/postgres/test-connection [post]
func (h *PostgresHandler) TestConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req PostgresTestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Set default port
	port := req.Port
	if port == 0 {
		port = 5432
	}

	// Create PostgreSQL backup config for testing
	config := &models.PostgresBackupConfig{
		Host:     req.Host,
		Port:     port,
		Username: req.Username,
		Database: req.Database,
		SSLMode:  req.SSLMode,
	}

	// Create backup instance and set password
	pgBackup := databases.NewPostgresBackup(config, h.logger)
	pgBackup.DecryptedPassword = req.Password

	// Test connection
	info, err := pgBackup.TestConnection(c.Request.Context())

	response := PostgresTestConnectionResponse{
		Host: req.Host,
		Port: port,
	}

	if err != nil {
		response.Connected = false
		response.Error = err.Error()
		h.logger.Warn().
			Err(err).
			Str("host", req.Host).
			Int("port", port).
			Str("username", req.Username).
			Msg("PostgreSQL connection test failed")
	} else {
		response.Connected = info.Connected
		response.Version = info.Version
		response.Databases = info.Databases
		h.logger.Info().
			Str("host", req.Host).
			Int("port", port).
			Str("version", info.Version).
			Int("databases", len(info.Databases)).
			Msg("PostgreSQL connection test successful")
	}

	c.JSON(http.StatusOK, response)
}

// RestoreInstructionsRequest is the request for getting restore instructions.
type RestoreInstructionsRequest struct {
	Format     string `form:"format" binding:"omitempty,oneof=plain custom directory tar"`
	BackupPath string `form:"backup_path,omitempty"`
}

// RestoreInstructionsResponse is the response containing restore instructions.
type RestoreInstructionsResponse struct {
	Format       string   `json:"format"`
	Instructions []string `json:"instructions"`
	Commands     []string `json:"commands"`
	Notes        []string `json:"notes,omitempty"`
}

// GetRestoreInstructions returns instructions for restoring a PostgreSQL backup.
//
//	@Summary		Get PostgreSQL restore instructions
//	@Description	Returns step-by-step instructions for restoring a PostgreSQL backup based on format
//	@Tags			PostgreSQL
//	@Accept			json
//	@Produce		json
//	@Param			format		query		string	false	"Backup format (plain, custom, directory, tar)"
//	@Param			backup_path	query		string	false	"Path to the backup file"
//	@Success		200			{object}	RestoreInstructionsResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/postgres/restore-instructions [get]
func (h *PostgresHandler) GetRestoreInstructions(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	format := c.DefaultQuery("format", "custom")
	backupPath := c.DefaultQuery("backup_path", "<backup_file>")

	// Create a config with the specified format
	config := &models.PostgresBackupConfig{
		OutputFormat: models.PostgresOutputFormat(format),
	}

	pgBackup := databases.NewPostgresBackup(config, h.logger)
	instructions := pgBackup.GetRestoreInstructions(backupPath)

	c.JSON(http.StatusOK, RestoreInstructionsResponse{
		Format:       instructions.Format,
		Instructions: instructions.Instructions,
		Commands:     instructions.Commands,
		Notes:        instructions.Notes,
	})
}

// EncryptPasswordRequest is the request for encrypting a PostgreSQL password.
type EncryptPasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

// EncryptPasswordResponse is the response containing the encrypted password.
type EncryptPasswordResponse struct {
	EncryptedPassword string `json:"encrypted_password"`
}

// EncryptPassword encrypts a PostgreSQL password for secure storage.
// This endpoint is used when creating or updating a PostgreSQL backup schedule.
//
//	@Summary		Encrypt PostgreSQL password
//	@Description	Encrypts a PostgreSQL password for secure storage in backup schedules
//	@Tags			PostgreSQL
//	@Accept			json
//	@Produce		json
//	@Param			request	body		EncryptPasswordRequest	true	"Password to encrypt"
//	@Success		200		{object}	EncryptPasswordResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/postgres/encrypt-password [post]
func (h *PostgresHandler) EncryptPassword(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if h.keyManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption not configured"})
		return
	}

	var req EncryptPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	encrypted, err := h.keyManager.EncryptString(req.Password)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt password")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt password"})
		return
	}

	c.JSON(http.StatusOK, EncryptPasswordResponse{
		EncryptedPassword: encrypted,
	})
}
