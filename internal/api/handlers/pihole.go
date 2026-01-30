package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup/apps"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// PiholeStore defines the interface for Pi-hole persistence operations.
type PiholeStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
}

// PiholeHandler handles Pi-hole-related HTTP endpoints.
type PiholeHandler struct {
	store  PiholeStore
	logger zerolog.Logger
}

// NewPiholeHandler creates a new PiholeHandler.
func NewPiholeHandler(store PiholeStore, logger zerolog.Logger) *PiholeHandler {
	return &PiholeHandler{
		store:  store,
		logger: logger.With().Str("component", "pihole_handler").Logger(),
	}
}

// RegisterRoutes registers Pi-hole routes on the given router group.
func (h *PiholeHandler) RegisterRoutes(r *gin.RouterGroup) {
	pihole := r.Group("/pihole")
	{
		pihole.GET("/agents", h.ListAgentsWithPihole)
		pihole.GET("/agents/:id/status", h.GetPiholeStatus)
		pihole.POST("/agents/:id/backup", h.CreateBackup)
		pihole.POST("/agents/:id/restore", h.RestoreBackup)
	}
}

// PiholeAgentInfo contains Pi-hole information for an agent.
type PiholeAgentInfo struct {
	AgentID         uuid.UUID          `json:"agent_id"`
	Hostname        string             `json:"hostname"`
	PiholeInstalled bool               `json:"pihole_installed"`
	PiholeVersion   string             `json:"pihole_version,omitempty"`
	FTLVersion      string             `json:"ftl_version,omitempty"`
	WebVersion      string             `json:"web_version,omitempty"`
	BlockingEnabled bool               `json:"blocking_enabled"`
	ConfigDir       string             `json:"config_dir,omitempty"`
	AgentStatus     models.AgentStatus `json:"agent_status"`
}

// ListAgentsWithPihole returns all agents that have Pi-hole installed.
//
//	@Summary		List agents with Pi-hole
//	@Description	Returns all agents in the organization that have Pi-hole detected
//	@Tags			Pi-hole
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]PiholeAgentInfo
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/pihole/agents [get]
func (h *PiholeHandler) ListAgentsWithPihole(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	agents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", user.CurrentOrgID.String()).Msg("failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list agents"})
		return
	}

	var piholeAgents []PiholeAgentInfo
	for _, agent := range agents {
		if agent.HealthMetrics != nil && agent.HealthMetrics.PiholeInfo != nil && agent.HealthMetrics.PiholeInfo.Installed {
			info := PiholeAgentInfo{
				AgentID:         agent.ID,
				Hostname:        agent.Hostname,
				PiholeInstalled: true,
				PiholeVersion:   agent.HealthMetrics.PiholeInfo.Version,
				FTLVersion:      agent.HealthMetrics.PiholeInfo.FTLVersion,
				WebVersion:      agent.HealthMetrics.PiholeInfo.WebVersion,
				BlockingEnabled: agent.HealthMetrics.PiholeInfo.BlockingEnabled,
				ConfigDir:       agent.HealthMetrics.PiholeInfo.ConfigDir,
				AgentStatus:     agent.Status,
			}
			piholeAgents = append(piholeAgents, info)
		}
	}

	c.JSON(http.StatusOK, gin.H{"agents": piholeAgents})
}

// PiholeStatusResponse contains detailed Pi-hole status for an agent.
type PiholeStatusResponse struct {
	AgentID         uuid.UUID              `json:"agent_id"`
	Hostname        string                 `json:"hostname"`
	PiholeInstalled bool                   `json:"pihole_installed"`
	PiholeVersion   string                 `json:"pihole_version,omitempty"`
	FTLVersion      string                 `json:"ftl_version,omitempty"`
	WebVersion      string                 `json:"web_version,omitempty"`
	BlockingEnabled bool                   `json:"blocking_enabled"`
	ConfigDir       string                 `json:"config_dir,omitempty"`
	Statistics      map[string]interface{} `json:"statistics,omitempty"`
}

// GetPiholeStatus returns Pi-hole status for a specific agent.
//
//	@Summary		Get Pi-hole status
//	@Description	Returns detailed Pi-hole status for a specific agent
//	@Tags			Pi-hole
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Agent ID"
//	@Success		200	{object}	PiholeStatusResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/pihole/agents/{id}/status [get]
func (h *PiholeHandler) GetPiholeStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	agentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get agent")
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	response := PiholeStatusResponse{
		AgentID:         agent.ID,
		Hostname:        agent.Hostname,
		PiholeInstalled: false,
	}

	if agent.HealthMetrics != nil && agent.HealthMetrics.PiholeInfo != nil {
		response.PiholeInstalled = agent.HealthMetrics.PiholeInfo.Installed
		response.PiholeVersion = agent.HealthMetrics.PiholeInfo.Version
		response.FTLVersion = agent.HealthMetrics.PiholeInfo.FTLVersion
		response.WebVersion = agent.HealthMetrics.PiholeInfo.WebVersion
		response.BlockingEnabled = agent.HealthMetrics.PiholeInfo.BlockingEnabled
		response.ConfigDir = agent.HealthMetrics.PiholeInfo.ConfigDir
	}

	c.JSON(http.StatusOK, response)
}

// CreateBackupRequest is the request body for creating a Pi-hole backup.
type CreateBackupRequest struct {
	OutputDir        string `json:"output_dir" binding:"required"`
	UseTeleporter    bool   `json:"use_teleporter"`
	IncludeQueryLogs bool   `json:"include_query_logs"`
}

// CreateBackupResponse is the response for Pi-hole backup creation.
type CreateBackupResponse struct {
	Success     bool     `json:"success"`
	BackupPath  string   `json:"backup_path,omitempty"`
	BackupFiles []string `json:"backup_files,omitempty"`
	SizeBytes   int64    `json:"size_bytes,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// CreateBackup creates a Pi-hole backup on the specified agent.
//
//	@Summary		Create Pi-hole backup
//	@Description	Creates a backup of Pi-hole configuration and data on the specified agent
//	@Tags			Pi-hole
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Agent ID"
//	@Param			request	body		CreateBackupRequest	true	"Backup options"
//	@Success		200		{object}	CreateBackupResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/pihole/agents/{id}/backup [post]
func (h *PiholeHandler) CreateBackup(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	agentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get agent")
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Check if Pi-hole is installed
	if agent.HealthMetrics == nil || agent.HealthMetrics.PiholeInfo == nil || !agent.HealthMetrics.PiholeInfo.Installed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pi-hole is not installed on this agent"})
		return
	}

	// Create Pi-hole backup instance
	pihole := apps.NewPiholeBackup(h.logger)
	pihole.UseTeleporter = req.UseTeleporter
	pihole.IncludeQueryLogs = req.IncludeQueryLogs
	if agent.HealthMetrics.PiholeInfo.ConfigDir != "" {
		pihole.ConfigDir = agent.HealthMetrics.PiholeInfo.ConfigDir
	}

	// Execute backup
	result, err := pihole.Backup(c.Request.Context(), req.OutputDir)
	if err != nil {
		h.logger.Error().Err(err).
			Str("agent_id", agentID.String()).
			Str("output_dir", req.OutputDir).
			Msg("Pi-hole backup failed")
		c.JSON(http.StatusInternalServerError, CreateBackupResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, CreateBackupResponse{
		Success:     result.Success,
		BackupPath:  result.BackupPath,
		BackupFiles: result.BackupFiles,
		SizeBytes:   result.SizeBytes,
	})
}

// RestoreBackupRequest is the request body for restoring a Pi-hole backup.
type RestoreBackupRequest struct {
	BackupPath string `json:"backup_path" binding:"required"`
}

// RestoreBackupResponse is the response for Pi-hole backup restoration.
type RestoreBackupResponse struct {
	Success       bool     `json:"success"`
	RestoredFiles []string `json:"restored_files,omitempty"`
	RestartNeeded bool     `json:"restart_needed"`
	Error         string   `json:"error,omitempty"`
}

// RestoreBackup restores a Pi-hole backup on the specified agent.
//
//	@Summary		Restore Pi-hole backup
//	@Description	Restores Pi-hole configuration and data from a backup on the specified agent
//	@Tags			Pi-hole
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Agent ID"
//	@Param			request	body		RestoreBackupRequest	true	"Restore options"
//	@Success		200		{object}	RestoreBackupResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/pihole/agents/{id}/restore [post]
func (h *PiholeHandler) RestoreBackup(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	idParam := c.Param("id")
	agentID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid agent ID"})
		return
	}

	var req RestoreBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	agent, err := h.store.GetAgentByID(c.Request.Context(), agentID)
	if err != nil {
		h.logger.Error().Err(err).Str("agent_id", agentID.String()).Msg("failed to get agent")
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Verify agent belongs to current org
	if agent.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "agent not found"})
		return
	}

	// Check if Pi-hole is installed
	if agent.HealthMetrics == nil || agent.HealthMetrics.PiholeInfo == nil || !agent.HealthMetrics.PiholeInfo.Installed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pi-hole is not installed on this agent"})
		return
	}

	// Create Pi-hole backup instance
	pihole := apps.NewPiholeBackup(h.logger)
	if agent.HealthMetrics.PiholeInfo.ConfigDir != "" {
		pihole.ConfigDir = agent.HealthMetrics.PiholeInfo.ConfigDir
	}

	// Execute restore
	result, err := pihole.Restore(c.Request.Context(), req.BackupPath)
	if err != nil {
		h.logger.Error().Err(err).
			Str("agent_id", agentID.String()).
			Str("backup_path", req.BackupPath).
			Msg("Pi-hole restore failed")
		c.JSON(http.StatusInternalServerError, RestoreBackupResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, RestoreBackupResponse{
		Success:       result.Success,
		RestoredFiles: result.RestoredFiles,
		RestartNeeded: result.RestartNeeded,
	})
}
