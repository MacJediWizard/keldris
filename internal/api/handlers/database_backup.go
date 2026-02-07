package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/maintenance"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DatabaseBackupStore defines the interface for database backup persistence.
type DatabaseBackupStore interface {
	GetDatabaseBackupByID(ctx context.Context, id uuid.UUID) (*models.DatabaseBackup, error)
	ListDatabaseBackups(ctx context.Context, limit, offset int) ([]*models.DatabaseBackup, int, error)
	GetLatestDatabaseBackup(ctx context.Context) (*models.DatabaseBackup, error)
	GetDatabaseBackupSummary(ctx context.Context) (*models.DatabaseBackupSummary, error)
	MarkDatabaseBackupVerified(ctx context.Context, id uuid.UUID) error
}

// DatabaseBackupHandler handles database backup-related HTTP endpoints.
type DatabaseBackupHandler struct {
	store         DatabaseBackupStore
	sessions      *auth.SessionStore
	backupService *maintenance.DatabaseBackupService
	logger        zerolog.Logger
}

// NewDatabaseBackupHandler creates a new DatabaseBackupHandler.
func NewDatabaseBackupHandler(
	store DatabaseBackupStore,
	sessions *auth.SessionStore,
	backupService *maintenance.DatabaseBackupService,
	logger zerolog.Logger,
) *DatabaseBackupHandler {
	return &DatabaseBackupHandler{
		store:         store,
		sessions:      sessions,
		backupService: backupService,
		logger:        logger.With().Str("component", "database_backup_handler").Logger(),
	}
}

// RegisterRoutes registers database backup routes on the given router group.
// These routes require superuser privileges.
func (h *DatabaseBackupHandler) RegisterRoutes(r *gin.RouterGroup) {
	dbBackup := r.Group("/superuser/database-backups")
	dbBackup.Use(middleware.SuperuserMiddleware(h.sessions, h.logger))
	{
		dbBackup.GET("", h.ListBackups)
		dbBackup.GET("/status", h.GetStatus)
		dbBackup.GET("/summary", h.GetSummary)
		dbBackup.POST("/trigger", h.TriggerBackup)
		dbBackup.GET("/:id", h.GetBackup)
		dbBackup.POST("/:id/verify", h.VerifyBackup)
		dbBackup.GET("/:id/restore-instructions", h.GetRestoreInstructions)
	}
}

// ListBackups returns a paginated list of database backups.
// GET /api/v1/superuser/database-backups
func (h *DatabaseBackupHandler) ListBackups(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	// Parse pagination
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := c.Query("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	backups, total, err := h.store.ListDatabaseBackups(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list database backups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backups": backups,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

// GetStatus returns the current database backup service status.
// GET /api/v1/superuser/database-backups/status
func (h *DatabaseBackupHandler) GetStatus(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "database backup service not configured",
			"enabled": false,
		})
		return
	}

	status, err := h.backupService.GetStatus(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get backup service status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": status})
}

// GetSummary returns a summary of database backups.
// GET /api/v1/superuser/database-backups/summary
func (h *DatabaseBackupHandler) GetSummary(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	summary, err := h.store.GetDatabaseBackupSummary(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get backup summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get summary"})
		return
	}

	// Add service status
	if h.backupService != nil {
		healthy, _ := h.backupService.IsHealthy(c.Request.Context())
		summary.BackupServiceUp = healthy
	}

	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// GetBackup returns a specific database backup by ID.
// GET /api/v1/superuser/database-backups/:id
func (h *DatabaseBackupHandler) GetBackup(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDatabaseBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backup": backup})
}

// TriggerBackup triggers an immediate database backup.
// POST /api/v1/superuser/database-backups/trigger
func (h *DatabaseBackupHandler) TriggerBackup(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database backup service not configured"})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("user_email", user.Email).
		Msg("manual database backup triggered")

	// Trigger backup asynchronously to avoid timeout
	go func() {
		backup, err := h.backupService.TriggerBackup(context.Background())
		if err != nil {
			h.logger.Error().
				Err(err).
				Str("user_id", user.ID.String()).
				Msg("manual database backup failed")
		} else {
			h.logger.Info().
				Str("backup_id", backup.ID.String()).
				Str("user_id", user.ID.String()).
				Msg("manual database backup completed")
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "database backup started",
		"status":  "Check /api/v1/superuser/database-backups/status for progress",
	})
}

// VerifyBackup verifies a database backup's integrity.
// POST /api/v1/superuser/database-backups/:id/verify
func (h *DatabaseBackupHandler) VerifyBackup(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database backup service not configured"})
		return
	}

	h.logger.Info().
		Str("user_id", user.ID.String()).
		Str("backup_id", backupID.String()).
		Msg("verifying database backup")

	if err := h.backupService.VerifyBackup(c.Request.Context(), backupID); err != nil {
		h.logger.Error().
			Err(err).
			Str("backup_id", backupID.String()).
			Msg("backup verification failed")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "backup verification failed",
			"details":  err.Error(),
			"verified": false,
		})
		return
	}

	// Mark as verified in database
	if err := h.store.MarkDatabaseBackupVerified(c.Request.Context(), backupID); err != nil {
		h.logger.Error().Err(err).Msg("failed to mark backup as verified")
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "backup verification passed",
		"verified": true,
	})
}

// GetRestoreInstructions returns restore instructions for a backup.
// GET /api/v1/superuser/database-backups/:id/restore-instructions
func (h *DatabaseBackupHandler) GetRestoreInstructions(c *gin.Context) {
	user := middleware.RequireSuperuser(c)
	if user == nil {
		return
	}

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	// Verify backup exists
	_, err = h.store.GetDatabaseBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	if h.backupService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database backup service not configured"})
		return
	}

	instructions := h.backupService.RestoreInstructions(backupID)

	c.JSON(http.StatusOK, gin.H{
		"backup_id":    backupID,
		"instructions": instructions,
	})
}
