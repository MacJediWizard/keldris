package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/docker"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DockerLogStore defines the interface for docker log persistence operations.
type DockerLogStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	CreateDockerLogBackup(ctx context.Context, backup *models.DockerLogBackup) error
	UpdateDockerLogBackup(ctx context.Context, backup *models.DockerLogBackup) error
	GetDockerLogBackupByID(ctx context.Context, id uuid.UUID) (*models.DockerLogBackup, error)
	GetDockerLogBackupsByAgentID(ctx context.Context, agentID uuid.UUID) ([]*models.DockerLogBackup, error)
	GetDockerLogBackupsByContainer(ctx context.Context, agentID uuid.UUID, containerID string) ([]*models.DockerLogBackup, error)
	GetDockerLogBackupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DockerLogBackup, error)
	DeleteDockerLogBackup(ctx context.Context, id uuid.UUID) error
	DeleteOldDockerLogBackups(ctx context.Context, agentID uuid.UUID, olderThan time.Time) (int64, error)
	GetDockerLogSettingsByAgentID(ctx context.Context, agentID uuid.UUID) (*models.DockerLogSettings, error)
	GetOrCreateDockerLogSettings(ctx context.Context, agentID uuid.UUID) (*models.DockerLogSettings, error)
	CreateDockerLogSettings(ctx context.Context, settings *models.DockerLogSettings) error
	UpdateDockerLogSettings(ctx context.Context, settings *models.DockerLogSettings) error
}

// DockerLogsHandler handles Docker log-related HTTP endpoints.
type DockerLogsHandler struct {
	store         DockerLogStore
	backupService *docker.LogBackupService
	logger        zerolog.Logger
}

// NewDockerLogsHandler creates a new DockerLogsHandler.
func NewDockerLogsHandler(store DockerLogStore, backupService *docker.LogBackupService, logger zerolog.Logger) *DockerLogsHandler {
	return &DockerLogsHandler{
		store:         store,
		backupService: backupService,
		logger:        logger.With().Str("component", "docker_logs_handler").Logger(),
	}
}

// RegisterRoutes registers Docker logs routes on the given router group.
func (h *DockerLogsHandler) RegisterRoutes(r *gin.RouterGroup) {
	dockerLogs := r.Group("/docker-logs")
	{
		// Backup management
		dockerLogs.GET("", h.ListBackups)
		dockerLogs.GET("/:id", h.GetBackup)
		dockerLogs.GET("/:id/view", h.ViewLogs)
		dockerLogs.GET("/:id/download", h.DownloadLogs)
		dockerLogs.DELETE("/:id", h.DeleteBackup)

		// Settings management
		dockerLogs.GET("/settings/:agent_id", h.GetSettings)
		dockerLogs.PUT("/settings/:agent_id", h.UpdateSettings)

		// Container-specific queries
		dockerLogs.GET("/agent/:agent_id", h.ListByAgent)
		dockerLogs.GET("/agent/:agent_id/container/:container_id", h.ListByContainer)

		// Storage statistics
		dockerLogs.GET("/stats/:agent_id", h.GetStorageStats)

		// Retention management
		dockerLogs.POST("/retention/:agent_id", h.ApplyRetention)
	}
}

// ListBackupsResponse is the response for listing docker log backups.
type ListBackupsResponse struct {
	Backups    []*models.DockerLogBackup `json:"backups"`
	TotalCount int                       `json:"total_count"`
}

// ListBackups returns all docker log backups for the organization.
//
//	@Summary		List Docker log backups
//	@Description	Returns all Docker log backups for the current organization
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			status	query		string	false	"Filter by status"
//	@Success		200		{object}	ListBackupsResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs [get]
func (h *DockerLogsHandler) ListBackups(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	backups, err := h.store.GetDockerLogBackupsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list docker log backups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}

	// Filter by status if specified
	status := c.Query("status")
	if status != "" {
		var filtered []*models.DockerLogBackup
		for _, b := range backups {
			if string(b.Status) == status {
				filtered = append(filtered, b)
			}
		}
		backups = filtered
	}

	c.JSON(http.StatusOK, ListBackupsResponse{
		Backups:    backups,
		TotalCount: len(backups),
	})
}

// GetBackup returns a specific docker log backup by ID.
//
//	@Summary		Get Docker log backup
//	@Description	Returns details of a specific Docker log backup
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Backup ID"
//	@Success		200	{object}	models.DockerLogBackup
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/{id} [get]
func (h *DockerLogsHandler) GetBackup(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDockerLogBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Verify the backup belongs to an agent in the user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// ViewLogs returns the contents of a backed up log file.
//
//	@Summary		View backed up Docker logs
//	@Description	Returns the contents of a backed up Docker log file
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string	true	"Backup ID"
//	@Param			offset	query		int		false	"Line offset"
//	@Param			limit	query		int		false	"Number of lines to return"
//	@Success		200		{object}	models.DockerLogViewResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/{id}/view [get]
func (h *DockerLogsHandler) ViewLogs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDockerLogBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Verify access
	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Parse pagination parameters
	offset, _ := strconv.ParseInt(c.DefaultQuery("offset", "0"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "1000"), 10, 64)

	if limit > 10000 {
		limit = 10000 // Cap at 10000 lines
	}

	response, err := h.backupService.RestoreContainerLogs(c.Request.Context(), backup, offset, limit)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", backupID.String()).Msg("failed to view logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read backup file"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DownloadLogs downloads a backed up log file.
//
//	@Summary		Download backed up Docker logs
//	@Description	Downloads a backed up Docker log file
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			id		path		string	true	"Backup ID"
//	@Param			format	query		string	false	"Output format (json, csv, raw)"
//	@Success		200		{file}		file
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/{id}/download [get]
func (h *DockerLogsHandler) DownloadLogs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDockerLogBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Verify access
	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	format := c.DefaultQuery("format", "json")

	// Get all logs
	response, err := h.backupService.RestoreContainerLogs(c.Request.Context(), backup, 0, 0)
	if err != nil {
		h.logger.Error().Err(err).Str("backup_id", backupID.String()).Msg("failed to read logs for download")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read backup file"})
		return
	}

	filename := fmt.Sprintf("%s_%s", backup.ContainerName, backup.StartTime.Format("2006-01-02_15-04-05"))

	switch format {
	case "csv":
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write header
		writer.Write([]string{"Line", "Timestamp", "Stream", "Message"})

		for _, entry := range response.Entries {
			writer.Write([]string{
				strconv.FormatInt(entry.LineNum, 10),
				entry.Timestamp.Format(time.RFC3339),
				entry.Stream,
				entry.Message,
			})
		}

	case "raw":
		c.Header("Content-Type", "text/plain")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.log", filename))

		for _, entry := range response.Entries {
			fmt.Fprintf(c.Writer, "%s %s %s\n", entry.Timestamp.Format(time.RFC3339), entry.Stream, entry.Message)
		}

	default: // json
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))

		encoder := json.NewEncoder(c.Writer)
		encoder.SetIndent("", "  ")
		encoder.Encode(response)
	}
}

// DeleteBackup deletes a docker log backup.
//
//	@Summary		Delete Docker log backup
//	@Description	Deletes a Docker log backup and its associated file
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Backup ID"
//	@Success		200	{object}	map[string]string
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/{id} [delete]
func (h *DockerLogsHandler) DeleteBackup(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	backupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup ID"})
		return
	}

	backup, err := h.store.GetDockerLogBackupByID(c.Request.Context(), backupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Verify access
	agent, err := h.store.GetAgentByID(c.Request.Context(), backup.AgentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}

	// Delete the file
	if err := h.backupService.DeleteBackup(c.Request.Context(), backup); err != nil {
		h.logger.Warn().Err(err).Str("backup_id", backupID.String()).Msg("failed to delete backup file")
	}

	// Delete the database record
	if err := h.store.DeleteDockerLogBackup(c.Request.Context(), backupID); err != nil {
		h.logger.Error().Err(err).Str("backup_id", backupID.String()).Msg("failed to delete backup record")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete backup"})
		return
	}

	h.logger.Info().Str("backup_id", backupID.String()).Msg("docker log backup deleted")
	c.JSON(http.StatusOK, gin.H{"message": "backup deleted successfully"})
}

// GetSettings returns docker log settings for an agent.
//
//	@Summary		Get Docker log settings
//	@Description	Returns Docker log backup settings for an agent
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	path		string	true	"Agent ID"
//	@Success		200			{object}	models.DockerLogSettings
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/settings/{agent_id} [get]
func (h *DockerLogsHandler) GetSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	settings, err := h.store.GetOrCreateDockerLogSettings(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get docker log settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// DockerLogSettingsRequest is the request body for updating docker log settings.
type DockerLogSettingsRequest struct {
	Enabled           *bool                            `json:"enabled,omitempty"`
	CronExpression    string                           `json:"cron_expression,omitempty"`
	RetentionPolicy   *models.DockerLogRetentionPolicy `json:"retention_policy,omitempty"`
	IncludeContainers []string                         `json:"include_containers,omitempty"`
	ExcludeContainers []string                         `json:"exclude_containers,omitempty"`
	IncludeLabels     map[string]string                `json:"include_labels,omitempty"`
	ExcludeLabels     map[string]string                `json:"exclude_labels,omitempty"`
	Timestamps        *bool                            `json:"timestamps,omitempty"`
	Tail              *int                             `json:"tail,omitempty"`
	Since             string                           `json:"since,omitempty"`
	Until             string                           `json:"until,omitempty"`
}

// UpdateSettings updates docker log settings for an agent.
//
//	@Summary		Update Docker log settings
//	@Description	Updates Docker log backup settings for an agent
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	path		string					true	"Agent ID"
//	@Param			request		body		DockerLogSettingsRequest	true	"Settings to update"
//	@Success		200			{object}	models.DockerLogSettings
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/settings/{agent_id} [put]
func (h *DockerLogsHandler) UpdateSettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	var req DockerLogSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	settings, err := h.store.GetOrCreateDockerLogSettings(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get docker log settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	// Apply updates
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}
	if req.CronExpression != "" {
		settings.CronExpression = req.CronExpression
	}
	if req.RetentionPolicy != nil {
		settings.RetentionPolicy = *req.RetentionPolicy
	}
	if req.IncludeContainers != nil {
		settings.IncludeContainers = req.IncludeContainers
	}
	if req.ExcludeContainers != nil {
		settings.ExcludeContainers = req.ExcludeContainers
	}
	if req.IncludeLabels != nil {
		settings.IncludeLabels = req.IncludeLabels
	}
	if req.ExcludeLabels != nil {
		settings.ExcludeLabels = req.ExcludeLabels
	}
	if req.Timestamps != nil {
		settings.Timestamps = *req.Timestamps
	}
	if req.Tail != nil {
		settings.Tail = *req.Tail
	}
	settings.Since = req.Since
	settings.Until = req.Until

	if err := h.store.UpdateDockerLogSettings(c.Request.Context(), settings); err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to update docker log settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	h.logger.Info().Str("agent_id", agentID.String()).Bool("enabled", settings.Enabled).Msg("docker log settings updated")
	c.JSON(http.StatusOK, settings)
}

// ListByAgent returns all docker log backups for a specific agent.
//
//	@Summary		List Docker log backups by agent
//	@Description	Returns all Docker log backups for a specific agent
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	path		string	true	"Agent ID"
//	@Success		200			{object}	ListBackupsResponse
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/agent/{agent_id} [get]
func (h *DockerLogsHandler) ListByAgent(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	backups, err := h.store.GetDockerLogBackupsByAgentID(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to list docker log backups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}

	c.JSON(http.StatusOK, ListBackupsResponse{
		Backups:    backups,
		TotalCount: len(backups),
	})
}

// ListByContainer returns all docker log backups for a specific container.
//
//	@Summary		List Docker log backups by container
//	@Description	Returns all Docker log backups for a specific container
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			agent_id		path		string	true	"Agent ID"
//	@Param			container_id	path		string	true	"Container ID"
//	@Success		200				{object}	ListBackupsResponse
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		404				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/agent/{agent_id}/container/{container_id} [get]
func (h *DockerLogsHandler) ListByContainer(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	containerID := c.Param("container_id")
	if containerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "container ID is required"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	backups, err := h.store.GetDockerLogBackupsByContainer(c.Request.Context(), agentID, containerID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Str("container_id", containerID).Msg("failed to list docker log backups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list backups"})
		return
	}

	c.JSON(http.StatusOK, ListBackupsResponse{
		Backups:    backups,
		TotalCount: len(backups),
	})
}

// GetStorageStats returns storage statistics for an agent's docker log backups.
//
//	@Summary		Get Docker log storage statistics
//	@Description	Returns storage usage statistics for an agent's Docker log backups
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			agent_id	path		string	true	"Agent ID"
//	@Success		200			{object}	docker.StorageStats
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Failure		404			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/stats/{agent_id} [get]
func (h *DockerLogsHandler) GetStorageStats(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	stats, err := h.backupService.GetStorageStats(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get storage stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get storage stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// ApplyRetentionResponse is the response for applying retention policy.
type ApplyRetentionResponse struct {
	RemovedCount int   `json:"removed_count"`
	RemovedBytes int64 `json:"removed_bytes"`
}

// ApplyRetention applies the retention policy for an agent's docker log backups.
//
//	@Summary		Apply Docker log retention policy
//	@Description	Applies the retention policy to remove old Docker log backups
//	@Tags			DockerLogs
//	@Accept			json
//	@Produce		json
//	@Param			agent_id		path		string	true	"Agent ID"
//	@Param			container_id	query		string	false	"Optional container ID to limit scope"
//	@Success		200				{object}	ApplyRetentionResponse
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		404				{object}	map[string]string
//	@Failure		500				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/docker-logs/retention/{agent_id} [post]
func (h *DockerLogsHandler) ApplyRetention(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agentID, err := uuid.Parse(c.Param("agent_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	// Verify agent belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil || agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Get settings for retention policy
	settings, err := h.store.GetOrCreateDockerLogSettings(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get docker log settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	containerID := c.Query("container_id")

	var totalRemoved int
	var totalBytes int64

	if containerID != "" {
		removed, bytes, err := h.backupService.ApplyRetentionPolicy(c.Request.Context(), agentID, containerID, settings.RetentionPolicy)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to apply retention policy")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply retention policy"})
			return
		}
		totalRemoved = removed
		totalBytes = bytes
	} else {
		// Apply to all containers - would need to iterate
		// For now, just clean up old DB records
		cutoffTime := time.Now().AddDate(0, 0, -settings.RetentionPolicy.MaxAgeDays)
		removed, err := h.store.DeleteOldDockerLogBackups(c.Request.Context(), agentID, cutoffTime)
		if err != nil {
			h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to delete old backups")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to apply retention policy"})
			return
		}
		totalRemoved = int(removed)
	}

	h.logger.Info().
		Str("agent_id", agentID.String()).
		Int("removed_count", totalRemoved).
		Int64("removed_bytes", totalBytes).
		Msg("retention policy applied")

	c.JSON(http.StatusOK, ApplyRetentionResponse{
		RemovedCount: totalRemoved,
		RemovedBytes: totalBytes,
	})
}
