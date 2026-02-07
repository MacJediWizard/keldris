package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/databases"
	"github.com/MacJediWizard/keldris/internal/crypto"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DatabaseConnectionStore defines the interface for database connection persistence operations.
type DatabaseConnectionStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetDatabaseConnectionsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DatabaseConnection, error)
	GetDatabaseConnectionsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DatabaseConnection, error)
	GetDatabaseConnectionByID(ctx context.Context, id uuid.UUID) (*models.DatabaseConnection, error)
	CreateDatabaseConnection(ctx context.Context, conn *models.DatabaseConnection) error
	UpdateDatabaseConnection(ctx context.Context, conn *models.DatabaseConnection) error
	UpdateDatabaseConnectionCredentials(ctx context.Context, id uuid.UUID, credentialsEncrypted []byte) error
	DeleteDatabaseConnection(ctx context.Context, id uuid.UUID) error
	UpdateDatabaseConnectionHealth(ctx context.Context, id uuid.UUID, status models.DatabaseConnectionHealthStatus, version *string, errorMsg *string) error
}

// DatabaseConnectionsHandler handles database connection-related HTTP endpoints.
type DatabaseConnectionsHandler struct {
	store      DatabaseConnectionStore
	keyManager *crypto.KeyManager
	logger     zerolog.Logger
}

// NewDatabaseConnectionsHandler creates a new DatabaseConnectionsHandler.
func NewDatabaseConnectionsHandler(store DatabaseConnectionStore, keyManager *crypto.KeyManager, logger zerolog.Logger) *DatabaseConnectionsHandler {
	return &DatabaseConnectionsHandler{
		store:      store,
		keyManager: keyManager,
		logger:     logger.With().Str("component", "database_connections_handler").Logger(),
	}
}

// RegisterRoutes registers database connection routes on the given router group.
func (h *DatabaseConnectionsHandler) RegisterRoutes(r *gin.RouterGroup) {
	connections := r.Group("/database-connections")
	{
		connections.GET("", h.ListConnections)
		connections.POST("", h.CreateConnection)
		connections.GET("/types", h.ListConnectionTypes)
		connections.GET("/restore-instructions", h.GetRestoreInstructions)
		connections.GET("/:id", h.GetConnection)
		connections.PUT("/:id", h.UpdateConnection)
		connections.DELETE("/:id", h.DeleteConnection)
		connections.POST("/:id/test", h.TestConnection)
		connections.POST("/:id/update-credentials", h.UpdateCredentials)
		connections.POST("/test-new", h.TestNewConnection)
	}
}

// ListConnections returns all database connections for the authenticated user's organization.
// GET /api/v1/database-connections
func (h *DatabaseConnectionsHandler) ListConnections(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	connections, err := h.store.GetDatabaseConnectionsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list database connections")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list database connections"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"connections": connections})
}

// GetConnection returns a specific database connection by ID.
// GET /api/v1/database-connections/:id
func (h *DatabaseConnectionsHandler) GetConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetDatabaseConnectionByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("connection_id", id.String()).Msg("failed to get database connection")
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if conn.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"connection": conn})
}

// CreateConnectionRequest is the request body for creating a database connection.
type CreateConnectionRequest struct {
	Name     string               `json:"name" binding:"required,min=1,max=255"`
	Type     models.DatabaseType  `json:"type" binding:"required"`
	Host     string               `json:"host" binding:"required"`
	Port     int                  `json:"port"`
	Username string               `json:"username" binding:"required"`
	Password string               `json:"password" binding:"required"`
	SSLMode  string               `json:"ssl_mode,omitempty"`
	AgentID  *uuid.UUID           `json:"agent_id,omitempty"`
}

// CreateConnection creates a new database connection.
// POST /api/v1/database-connections
func (h *DatabaseConnectionsHandler) CreateConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Validate database type
	validType := false
	for _, t := range models.ValidDatabaseTypes() {
		if req.Type == t {
			validType = true
			break
		}
	}
	if !validType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid database type"})
		return
	}

	// Set default port if not provided
	port := req.Port
	if port == 0 {
		port = models.DefaultPort(req.Type)
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Encrypt the password
	credentials := &models.DatabaseCredentials{Password: req.Password}
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process credentials"})
		return
	}

	credentialsEncrypted, err := h.keyManager.Encrypt(credJSON)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt credentials"})
		return
	}

	conn := models.NewDatabaseConnection(dbUser.OrgID, req.Name, req.Type, req.Host, port, req.Username, credentialsEncrypted)
	conn.SSLMode = req.SSLMode
	conn.AgentID = req.AgentID
	conn.CreatedBy = &user.ID

	if err := h.store.CreateDatabaseConnection(c.Request.Context(), conn); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create database connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create database connection: " + err.Error()})
		return
	}

	h.logger.Info().
		Str("connection_id", conn.ID.String()).
		Str("name", req.Name).
		Str("type", string(req.Type)).
		Str("host", req.Host).
		Msg("database connection created")

	c.JSON(http.StatusCreated, gin.H{"connection": conn})
}

// UpdateConnectionRequest is the request body for updating a database connection.
type UpdateConnectionRequest struct {
	Name     *string `json:"name,omitempty"`
	Host     *string `json:"host,omitempty"`
	Port     *int    `json:"port,omitempty"`
	Username *string `json:"username,omitempty"`
	SSLMode  *string `json:"ssl_mode,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

// UpdateConnection updates an existing database connection.
// PUT /api/v1/database-connections/:id
func (h *DatabaseConnectionsHandler) UpdateConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	var req UpdateConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	conn, err := h.store.GetDatabaseConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if conn.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		conn.Name = *req.Name
	}
	if req.Host != nil {
		conn.Host = *req.Host
	}
	if req.Port != nil {
		conn.Port = *req.Port
	}
	if req.Username != nil {
		conn.Username = *req.Username
	}
	if req.SSLMode != nil {
		conn.SSLMode = *req.SSLMode
	}
	if req.Enabled != nil {
		conn.Enabled = *req.Enabled
	}

	if err := h.store.UpdateDatabaseConnection(c.Request.Context(), conn); err != nil {
		h.logger.Error().Err(err).Str("connection_id", id.String()).Msg("failed to update database connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update database connection"})
		return
	}

	h.logger.Info().Str("connection_id", id.String()).Msg("database connection updated")
	c.JSON(http.StatusOK, gin.H{"connection": conn})
}

// DeleteConnection removes a database connection.
// DELETE /api/v1/database-connections/:id
func (h *DatabaseConnectionsHandler) DeleteConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetDatabaseConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if conn.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	if err := h.store.DeleteDatabaseConnection(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("connection_id", id.String()).Msg("failed to delete database connection")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete database connection"})
		return
	}

	h.logger.Info().Str("connection_id", id.String()).Msg("database connection deleted")
	c.JSON(http.StatusOK, gin.H{"message": "database connection deleted"})
}

// TestConnection tests an existing database connection and updates its health status.
// POST /api/v1/database-connections/:id/test
func (h *DatabaseConnectionsHandler) TestConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	conn, err := h.store.GetDatabaseConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if conn.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	// Decrypt credentials
	credJSON, err := h.keyManager.Decrypt(conn.CredentialsEncrypted)
	if err != nil {
		h.logger.Error().Err(err).Str("connection_id", id.String()).Msg("failed to decrypt credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to decrypt credentials"})
		return
	}

	var credentials models.DatabaseCredentials
	if err := json.Unmarshal(credJSON, &credentials); err != nil {
		h.logger.Error().Err(err).Msg("failed to unmarshal credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process credentials"})
		return
	}

	// Create MySQL config and test connection
	mysqlConfig := &databases.MySQLConfig{
		Host:     conn.Host,
		Port:     conn.Port,
		Username: conn.Username,
		Password: credentials.Password,
		SSLMode:  conn.SSLMode,
	}

	mysqlBackup := databases.NewMySQLBackup(mysqlConfig, h.logger)
	testResult, err := mysqlBackup.TestConnection(c.Request.Context())

	// Update health status
	var healthStatus models.DatabaseConnectionHealthStatus
	var healthError *string
	var version *string

	if err != nil || !testResult.Success {
		healthStatus = models.DatabaseConnectionHealthUnhealthy
		errMsg := "connection test failed"
		if testResult != nil && testResult.ErrorMessage != "" {
			errMsg = testResult.ErrorMessage
		} else if err != nil {
			errMsg = err.Error()
		}
		healthError = &errMsg
	} else {
		healthStatus = models.DatabaseConnectionHealthHealthy
		if testResult.Version != "" {
			version = &testResult.Version
		}
	}

	if updateErr := h.store.UpdateDatabaseConnectionHealth(c.Request.Context(), id, healthStatus, version, healthError); updateErr != nil {
		h.logger.Warn().Err(updateErr).Str("connection_id", id.String()).Msg("failed to update connection health")
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
			"result":  testResult,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": testResult.Success,
		"result":  testResult,
	})
}

// TestNewConnectionRequest is the request body for testing a new connection without saving.
type TestNewConnectionRequest struct {
	Type     models.DatabaseType `json:"type" binding:"required"`
	Host     string              `json:"host" binding:"required"`
	Port     int                 `json:"port"`
	Username string              `json:"username" binding:"required"`
	Password string              `json:"password" binding:"required"`
	SSLMode  string              `json:"ssl_mode,omitempty"`
}

// TestNewConnection tests a database connection without saving it.
// POST /api/v1/database-connections/test-new
func (h *DatabaseConnectionsHandler) TestNewConnection(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req TestNewConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Set default port if not provided
	port := req.Port
	if port == 0 {
		port = models.DefaultPort(req.Type)
	}

	// Create MySQL config and test connection
	mysqlConfig := &databases.MySQLConfig{
		Host:     req.Host,
		Port:     port,
		Username: req.Username,
		Password: req.Password,
		SSLMode:  req.SSLMode,
	}

	mysqlBackup := databases.NewMySQLBackup(mysqlConfig, h.logger)
	testResult, err := mysqlBackup.TestConnection(c.Request.Context())

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
			"result":  testResult,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": testResult.Success,
		"result":  testResult,
	})
}

// UpdateCredentialsRequest is the request body for updating connection credentials.
type UpdateCredentialsRequest struct {
	Password string `json:"password" binding:"required"`
}

// UpdateCredentials updates the password for a database connection.
// POST /api/v1/database-connections/:id/update-credentials
func (h *DatabaseConnectionsHandler) UpdateCredentials(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid connection ID"})
		return
	}

	var req UpdateCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	conn, err := h.store.GetDatabaseConnectionByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if conn.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "database connection not found"})
		return
	}

	// Encrypt the new password
	credentials := &models.DatabaseCredentials{Password: req.Password}
	credJSON, err := json.Marshal(credentials)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process credentials"})
		return
	}

	credentialsEncrypted, err := h.keyManager.Encrypt(credJSON)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to encrypt credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt credentials"})
		return
	}

	if err := h.store.UpdateDatabaseConnectionCredentials(c.Request.Context(), id, credentialsEncrypted); err != nil {
		h.logger.Error().Err(err).Str("connection_id", id.String()).Msg("failed to update credentials")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update credentials"})
		return
	}

	h.logger.Info().Str("connection_id", id.String()).Msg("database connection credentials updated")
	c.JSON(http.StatusOK, gin.H{"message": "credentials updated successfully"})
}

// ListConnectionTypes returns information about available database connection types.
// GET /api/v1/database-connections/types
func (h *DatabaseConnectionsHandler) ListConnectionTypes(c *gin.Context) {
	types := []map[string]interface{}{
		{
			"type":         models.DatabaseTypeMySQL,
			"name":         "MySQL",
			"description":  "MySQL database server",
			"default_port": 3306,
			"fields":       []string{"host", "port", "username", "password", "ssl_mode"},
		},
		{
			"type":         models.DatabaseTypeMariaDB,
			"name":         "MariaDB",
			"description":  "MariaDB database server",
			"default_port": 3306,
			"fields":       []string{"host", "port", "username", "password", "ssl_mode"},
		},
	}

	c.JSON(http.StatusOK, gin.H{"types": types})
}

// GetRestoreInstructions returns instructions for restoring MySQL/MariaDB backups.
// GET /api/v1/database-connections/restore-instructions
func (h *DatabaseConnectionsHandler) GetRestoreInstructions(c *gin.Context) {
	backupFile := c.Query("backup_file")
	if backupFile == "" {
		backupFile = "mysql_backup_YYYYMMDD-HHMMSS.sql.gz"
	}

	mysqlBackup := databases.NewMySQLBackup(nil, h.logger)
	instructions := mysqlBackup.GetRestoreInstructions(backupFile)

	c.JSON(http.StatusOK, gin.H{
		"instructions":     instructions,
		"backup_file":      backupFile,
		"restore_commands": getRestoreCommands(backupFile),
	})
}

// getRestoreCommands returns structured restore commands for the given backup file.
func getRestoreCommands(backupFile string) []map[string]interface{} {
	isCompressed := len(backupFile) > 3 && backupFile[len(backupFile)-3:] == ".gz"

	commands := []map[string]interface{}{}

	if isCompressed {
		commands = append(commands, map[string]interface{}{
			"name":        "Restore all databases (compressed)",
			"command":     "gunzip -c " + backupFile + " | mysql -h <host> -u <user> -p",
			"description": "Restores all databases from a compressed backup",
		})
		commands = append(commands, map[string]interface{}{
			"name":        "Restore to specific database (compressed)",
			"command":     "gunzip -c " + backupFile + " | mysql -h <host> -u <user> -p <database>",
			"description": "Restores to a specific database from a compressed backup",
		})
		commands = append(commands, map[string]interface{}{
			"name":        "Preview backup contents",
			"command":     "zcat " + backupFile + " | head -100",
			"description": "View the first 100 lines of the backup file",
		})
	} else {
		commands = append(commands, map[string]interface{}{
			"name":        "Restore all databases",
			"command":     "mysql -h <host> -u <user> -p < " + backupFile,
			"description": "Restores all databases from the backup",
		})
		commands = append(commands, map[string]interface{}{
			"name":        "Restore to specific database",
			"command":     "mysql -h <host> -u <user> -p <database> < " + backupFile,
			"description": "Restores to a specific database from the backup",
		})
		commands = append(commands, map[string]interface{}{
			"name":        "Preview backup contents",
			"command":     "head -100 " + backupFile,
			"description": "View the first 100 lines of the backup file",
		})
	}

	commands = append(commands, map[string]interface{}{
		"name":        "Schema only restore",
		"command":     "mysql -h <host> -u <user> -p --no-data < " + backupFile,
		"description": "Restore only the database schema without data",
	})
	commands = append(commands, map[string]interface{}{
		"name":        "Data only restore",
		"command":     "mysql -h <host> -u <user> -p --no-create-info < " + backupFile,
		"description": "Restore only the data without creating tables",
	})

	return commands
}
