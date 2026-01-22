package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/dr"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DRRunbookStore defines the interface for DR runbook persistence operations.
type DRRunbookStore interface {
	GetDRRunbooksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DRRunbook, error)
	GetDRRunbookByID(ctx context.Context, id uuid.UUID) (*models.DRRunbook, error)
	CreateDRRunbook(ctx context.Context, runbook *models.DRRunbook) error
	UpdateDRRunbook(ctx context.Context, runbook *models.DRRunbook) error
	DeleteDRRunbook(ctx context.Context, id uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetLatestDRTestByRunbookID(ctx context.Context, runbookID uuid.UUID) (*models.DRTest, error)
	GetDRTestSchedulesByRunbookID(ctx context.Context, runbookID uuid.UUID) ([]*models.DRTestSchedule, error)
	CreateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error
	UpdateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error
	DeleteDRTestSchedule(ctx context.Context, id uuid.UUID) error
	GetDRStatus(ctx context.Context, orgID uuid.UUID) (*models.DRStatus, error)
}

// DRRunbooksHandler handles DR runbook-related HTTP endpoints.
type DRRunbooksHandler struct {
	store     DRRunbookStore
	generator *dr.RunbookGenerator
	logger    zerolog.Logger
}

// NewDRRunbooksHandler creates a new DRRunbooksHandler.
func NewDRRunbooksHandler(store DRRunbookStore, logger zerolog.Logger) *DRRunbooksHandler {
	return &DRRunbooksHandler{
		store:     store,
		generator: dr.NewRunbookGenerator(store),
		logger:    logger.With().Str("component", "dr_runbooks_handler").Logger(),
	}
}

// RegisterRoutes registers DR runbook routes on the given router group.
func (h *DRRunbooksHandler) RegisterRoutes(r *gin.RouterGroup) {
	drRunbooks := r.Group("/dr-runbooks")
	{
		drRunbooks.GET("", h.List)
		drRunbooks.POST("", h.Create)
		drRunbooks.GET("/status", h.GetStatus)
		drRunbooks.GET("/:id", h.Get)
		drRunbooks.PUT("/:id", h.Update)
		drRunbooks.DELETE("/:id", h.Delete)
		drRunbooks.POST("/:id/activate", h.Activate)
		drRunbooks.POST("/:id/archive", h.Archive)
		drRunbooks.GET("/:id/render", h.Render)
		drRunbooks.POST("/:id/generate", h.GenerateFromSchedule)
		drRunbooks.GET("/:id/test-schedules", h.ListTestSchedules)
		drRunbooks.POST("/:id/test-schedules", h.CreateTestSchedule)
	}
}

// CreateDRRunbookRequest is the request body for creating a DR runbook.
type CreateDRRunbookRequest struct {
	ScheduleID                 *uuid.UUID               `json:"schedule_id,omitempty"`
	Name                       string                   `json:"name" binding:"required,min=1,max=255"`
	Description                string                   `json:"description,omitempty"`
	Steps                      []models.DRRunbookStep   `json:"steps,omitempty"`
	Contacts                   []models.DRRunbookContact `json:"contacts,omitempty"`
	CredentialsLocation        string                   `json:"credentials_location,omitempty"`
	RecoveryTimeObjectiveMins  *int                     `json:"recovery_time_objective_minutes,omitempty"`
	RecoveryPointObjectiveMins *int                     `json:"recovery_point_objective_minutes,omitempty"`
}

// UpdateDRRunbookRequest is the request body for updating a DR runbook.
type UpdateDRRunbookRequest struct {
	Name                       string                   `json:"name,omitempty"`
	Description                string                   `json:"description,omitempty"`
	Steps                      []models.DRRunbookStep   `json:"steps,omitempty"`
	Contacts                   []models.DRRunbookContact `json:"contacts,omitempty"`
	CredentialsLocation        string                   `json:"credentials_location,omitempty"`
	RecoveryTimeObjectiveMins  *int                     `json:"recovery_time_objective_minutes,omitempty"`
	RecoveryPointObjectiveMins *int                     `json:"recovery_point_objective_minutes,omitempty"`
	ScheduleID                 *uuid.UUID               `json:"schedule_id,omitempty"`
}

// CreateTestScheduleRequest is the request body for creating a test schedule.
type CreateTestScheduleRequest struct {
	CronExpression string `json:"cron_expression" binding:"required"`
	Enabled        *bool  `json:"enabled,omitempty"`
}

// List returns all DR runbooks for the authenticated user's organization.
// GET /api/v1/dr-runbooks
func (h *DRRunbooksHandler) List(c *gin.Context) {
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

	runbooks, err := h.store.GetDRRunbooksByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list DR runbooks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list runbooks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"runbooks": runbooks})
}

// Get returns a specific DR runbook by ID.
// GET /api/v1/dr-runbooks/:id
func (h *DRRunbooksHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to get runbook")
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	c.JSON(http.StatusOK, runbook)
}

// Create creates a new DR runbook.
// POST /api/v1/dr-runbooks
func (h *DRRunbooksHandler) Create(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateDRRunbookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Verify schedule belongs to user's org if provided
	if req.ScheduleID != nil {
		schedule, err := h.store.GetScheduleByID(c.Request.Context(), *req.ScheduleID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "schedule not found"})
			return
		}
		agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
		if err != nil || agent.OrgID != dbUser.OrgID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "schedule not found"})
			return
		}
	}

	runbook := models.NewDRRunbook(dbUser.OrgID, req.Name)
	runbook.ScheduleID = req.ScheduleID
	runbook.Description = req.Description
	runbook.CredentialsLocation = req.CredentialsLocation
	runbook.RecoveryTimeObjectiveMins = req.RecoveryTimeObjectiveMins
	runbook.RecoveryPointObjectiveMins = req.RecoveryPointObjectiveMins

	if req.Steps != nil {
		runbook.Steps = req.Steps
	}
	if req.Contacts != nil {
		runbook.Contacts = req.Contacts
	}

	if err := h.store.CreateDRRunbook(c.Request.Context(), runbook); err != nil {
		h.logger.Error().Err(err).Str("name", req.Name).Msg("failed to create runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create runbook"})
		return
	}

	h.logger.Info().
		Str("runbook_id", runbook.ID.String()).
		Str("name", req.Name).
		Msg("DR runbook created")

	c.JSON(http.StatusCreated, runbook)
}

// Update updates an existing DR runbook.
// PUT /api/v1/dr-runbooks/:id
func (h *DRRunbooksHandler) Update(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	var req UpdateDRRunbookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	// Update fields
	if req.Name != "" {
		runbook.Name = req.Name
	}
	if req.Description != "" {
		runbook.Description = req.Description
	}
	if req.Steps != nil {
		runbook.Steps = req.Steps
	}
	if req.Contacts != nil {
		runbook.Contacts = req.Contacts
	}
	if req.CredentialsLocation != "" {
		runbook.CredentialsLocation = req.CredentialsLocation
	}
	if req.RecoveryTimeObjectiveMins != nil {
		runbook.RecoveryTimeObjectiveMins = req.RecoveryTimeObjectiveMins
	}
	if req.RecoveryPointObjectiveMins != nil {
		runbook.RecoveryPointObjectiveMins = req.RecoveryPointObjectiveMins
	}
	if req.ScheduleID != nil {
		runbook.ScheduleID = req.ScheduleID
	}

	if err := h.store.UpdateDRRunbook(c.Request.Context(), runbook); err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to update runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update runbook"})
		return
	}

	h.logger.Info().Str("runbook_id", id.String()).Msg("DR runbook updated")
	c.JSON(http.StatusOK, runbook)
}

// Delete removes a DR runbook.
// DELETE /api/v1/dr-runbooks/:id
func (h *DRRunbooksHandler) Delete(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	if err := h.store.DeleteDRRunbook(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to delete runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete runbook"})
		return
	}

	h.logger.Info().Str("runbook_id", id.String()).Msg("DR runbook deleted")
	c.JSON(http.StatusOK, gin.H{"message": "runbook deleted"})
}

// Activate sets the runbook status to active.
// POST /api/v1/dr-runbooks/:id/activate
func (h *DRRunbooksHandler) Activate(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	runbook.Activate()
	if err := h.store.UpdateDRRunbook(c.Request.Context(), runbook); err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to activate runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to activate runbook"})
		return
	}

	h.logger.Info().Str("runbook_id", id.String()).Msg("DR runbook activated")
	c.JSON(http.StatusOK, runbook)
}

// Archive sets the runbook status to archived.
// POST /api/v1/dr-runbooks/:id/archive
func (h *DRRunbooksHandler) Archive(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	runbook.Archive()
	if err := h.store.UpdateDRRunbook(c.Request.Context(), runbook); err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to archive runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to archive runbook"})
		return
	}

	h.logger.Info().Str("runbook_id", id.String()).Msg("DR runbook archived")
	c.JSON(http.StatusOK, runbook)
}

// Render returns the runbook as rendered markdown.
// GET /api/v1/dr-runbooks/:id/render
func (h *DRRunbooksHandler) Render(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	content, err := h.generator.RenderText(c.Request.Context(), runbook)
	if err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to render runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to render runbook"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": content, "format": "markdown"})
}

// GenerateFromSchedule generates a runbook from a schedule.
// POST /api/v1/dr-runbooks/:id/generate
func (h *DRRunbooksHandler) GenerateFromSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	scheduleID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	schedule, err := h.store.GetScheduleByID(c.Request.Context(), scheduleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Verify schedule belongs to user's org
	agent, err := h.store.GetAgentByID(c.Request.Context(), schedule.AgentID)
	if err != nil || agent.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	runbook, err := h.generator.GenerateForSchedule(c.Request.Context(), schedule, dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to generate runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate runbook"})
		return
	}

	if err := h.store.CreateDRRunbook(c.Request.Context(), runbook); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", scheduleID.String()).Msg("failed to save generated runbook")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save runbook"})
		return
	}

	h.logger.Info().
		Str("runbook_id", runbook.ID.String()).
		Str("schedule_id", scheduleID.String()).
		Msg("DR runbook generated from schedule")

	c.JSON(http.StatusCreated, runbook)
}

// GetStatus returns the overall DR status for the organization.
// GET /api/v1/dr-runbooks/status
func (h *DRRunbooksHandler) GetStatus(c *gin.Context) {
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

	status, err := h.store.GetDRStatus(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get DR status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get DR status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ListTestSchedules returns test schedules for a runbook.
// GET /api/v1/dr-runbooks/:id/test-schedules
func (h *DRRunbooksHandler) ListTestSchedules(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	schedules, err := h.store.GetDRTestSchedulesByRunbookID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to list test schedules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list test schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// CreateTestSchedule creates a new test schedule for a runbook.
// POST /api/v1/dr-runbooks/:id/test-schedules
func (h *DRRunbooksHandler) CreateTestSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook ID"})
		return
	}

	var req CreateTestScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	if err := h.verifyRunbookAccess(c, user.ID, runbook); err != nil {
		return
	}

	schedule := models.NewDRTestSchedule(id, req.CronExpression)
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if err := h.store.CreateDRTestSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("runbook_id", id.String()).Msg("failed to create test schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create test schedule"})
		return
	}

	h.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("runbook_id", id.String()).
		Msg("DR test schedule created")

	c.JSON(http.StatusCreated, schedule)
}

// verifyRunbookAccess checks if the user has access to the runbook.
func (h *DRRunbooksHandler) verifyRunbookAccess(c *gin.Context, userID uuid.UUID, runbook *models.DRRunbook) error {
	dbUser, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return err
	}

	if runbook.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return err
	}

	return nil
}
