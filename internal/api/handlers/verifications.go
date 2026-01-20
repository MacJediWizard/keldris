package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// VerificationStore defines the interface for verification persistence operations.
type VerificationStore interface {
	GetVerificationsByRepoID(ctx context.Context, repoID uuid.UUID) ([]*models.Verification, error)
	GetVerificationByID(ctx context.Context, id uuid.UUID) (*models.Verification, error)
	GetLatestVerificationByRepoID(ctx context.Context, repoID uuid.UUID) (*models.Verification, error)
	GetConsecutiveFailedVerifications(ctx context.Context, repoID uuid.UUID) (int, error)
	GetVerificationSchedulesByRepoID(ctx context.Context, repoID uuid.UUID) ([]*models.VerificationSchedule, error)
	GetVerificationScheduleByID(ctx context.Context, id uuid.UUID) (*models.VerificationSchedule, error)
	CreateVerificationSchedule(ctx context.Context, vs *models.VerificationSchedule) error
	UpdateVerificationSchedule(ctx context.Context, vs *models.VerificationSchedule) error
	DeleteVerificationSchedule(ctx context.Context, id uuid.UUID) error
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// VerificationTrigger allows manually triggering verifications.
type VerificationTrigger interface {
	TriggerVerification(ctx context.Context, repoID uuid.UUID, verType models.VerificationType) (*models.Verification, error)
	GetRepositoryVerificationStatus(ctx context.Context, repoID uuid.UUID) (*models.RepositoryVerificationStatus, error)
}

// VerificationsHandler handles verification-related HTTP endpoints.
type VerificationsHandler struct {
	store   VerificationStore
	trigger VerificationTrigger
	logger  zerolog.Logger
}

// NewVerificationsHandler creates a new VerificationsHandler.
func NewVerificationsHandler(store VerificationStore, trigger VerificationTrigger, logger zerolog.Logger) *VerificationsHandler {
	return &VerificationsHandler{
		store:   store,
		trigger: trigger,
		logger:  logger.With().Str("component", "verifications_handler").Logger(),
	}
}

// RegisterRoutes registers verification routes on the given router group.
func (h *VerificationsHandler) RegisterRoutes(r *gin.RouterGroup) {
	verifications := r.Group("/verifications")
	{
		verifications.GET("", h.List)
		verifications.GET("/:id", h.Get)
	}

	// Repository verification endpoints
	repos := r.Group("/repositories")
	{
		repos.GET("/:id/verifications", h.ListByRepository)
		repos.GET("/:id/verification-status", h.GetStatus)
		repos.POST("/:id/verifications", h.TriggerVerification)
		repos.GET("/:id/verification-schedules", h.ListSchedules)
		repos.POST("/:id/verification-schedules", h.CreateSchedule)
	}

	// Verification schedule management
	schedules := r.Group("/verification-schedules")
	{
		schedules.GET("/:id", h.GetSchedule)
		schedules.PUT("/:id", h.UpdateSchedule)
		schedules.DELETE("/:id", h.DeleteSchedule)
	}
}

// VerificationResponse is the API response for a verification.
type VerificationResponse struct {
	ID           string                       `json:"id"`
	RepositoryID string                       `json:"repository_id"`
	Type         string                       `json:"type"`
	SnapshotID   string                       `json:"snapshot_id,omitempty"`
	StartedAt    string                       `json:"started_at"`
	CompletedAt  string                       `json:"completed_at,omitempty"`
	Status       string                       `json:"status"`
	DurationMs   *int64                       `json:"duration_ms,omitempty"`
	ErrorMessage string                       `json:"error_message,omitempty"`
	Details      *models.VerificationDetails  `json:"details,omitempty"`
	CreatedAt    string                       `json:"created_at"`
}

func toVerificationResponse(v *models.Verification) VerificationResponse {
	resp := VerificationResponse{
		ID:           v.ID.String(),
		RepositoryID: v.RepositoryID.String(),
		Type:         string(v.Type),
		SnapshotID:   v.SnapshotID,
		StartedAt:    v.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:       string(v.Status),
		DurationMs:   v.DurationMs,
		ErrorMessage: v.ErrorMessage,
		Details:      v.Details,
		CreatedAt:    v.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if v.CompletedAt != nil {
		resp.CompletedAt = v.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}

// VerificationScheduleResponse is the API response for a verification schedule.
type VerificationScheduleResponse struct {
	ID             string `json:"id"`
	RepositoryID   string `json:"repository_id"`
	Type           string `json:"type"`
	CronExpression string `json:"cron_expression"`
	Enabled        bool   `json:"enabled"`
	ReadDataSubset string `json:"read_data_subset,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func toVerificationScheduleResponse(vs *models.VerificationSchedule) VerificationScheduleResponse {
	return VerificationScheduleResponse{
		ID:             vs.ID.String(),
		RepositoryID:   vs.RepositoryID.String(),
		Type:           string(vs.Type),
		CronExpression: vs.CronExpression,
		Enabled:        vs.Enabled,
		ReadDataSubset: vs.ReadDataSubset,
		CreatedAt:      vs.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:      vs.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// VerificationStatusResponse is the API response for repository verification status.
type VerificationStatusResponse struct {
	RepositoryID     string                `json:"repository_id"`
	LastVerification *VerificationResponse `json:"last_verification,omitempty"`
	NextScheduledAt  string                `json:"next_scheduled_at,omitempty"`
	ConsecutiveFails int                   `json:"consecutive_fails"`
}

// TriggerVerificationRequest is the request body for triggering a verification.
type TriggerVerificationRequest struct {
	Type string `json:"type" binding:"required,oneof=check check_read_data test_restore"`
}

// CreateVerificationScheduleRequest is the request body for creating a verification schedule.
type CreateVerificationScheduleRequest struct {
	Type           string `json:"type" binding:"required,oneof=check check_read_data test_restore"`
	CronExpression string `json:"cron_expression" binding:"required"`
	Enabled        *bool  `json:"enabled,omitempty"`
	ReadDataSubset string `json:"read_data_subset,omitempty"`
}

// UpdateVerificationScheduleRequest is the request body for updating a verification schedule.
type UpdateVerificationScheduleRequest struct {
	CronExpression string `json:"cron_expression,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
	ReadDataSubset string `json:"read_data_subset,omitempty"`
}

// List returns all verifications accessible to the user.
// GET /api/v1/verifications
func (h *VerificationsHandler) List(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// For now, just return an empty list or require repository_id filter
	c.JSON(http.StatusOK, gin.H{"verifications": []VerificationResponse{}})
}

// Get returns a specific verification by ID.
// GET /api/v1/verifications/:id
func (h *VerificationsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid verification ID"})
		return
	}

	verification, err := h.store.GetVerificationByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("verification_id", id.String()).Msg("failed to get verification")
		c.JSON(http.StatusNotFound, gin.H{"error": "verification not found"})
		return
	}

	// Verify access via repository
	if err := h.verifyRepoAccess(c, user.ID, verification.RepositoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "verification not found"})
		return
	}

	c.JSON(http.StatusOK, toVerificationResponse(verification))
}

// ListByRepository returns all verifications for a repository.
// GET /api/v1/repositories/:id/verifications
func (h *VerificationsHandler) ListByRepository(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	verifications, err := h.store.GetVerificationsByRepoID(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to list verifications")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list verifications"})
		return
	}

	responses := make([]VerificationResponse, len(verifications))
	for i, v := range verifications {
		responses[i] = toVerificationResponse(v)
	}

	c.JSON(http.StatusOK, gin.H{"verifications": responses})
}

// GetStatus returns the verification status for a repository.
// GET /api/v1/repositories/:id/verification-status
func (h *VerificationsHandler) GetStatus(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	status, err := h.trigger.GetRepositoryVerificationStatus(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to get verification status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get verification status"})
		return
	}

	resp := VerificationStatusResponse{
		RepositoryID:     status.RepositoryID.String(),
		ConsecutiveFails: status.ConsecutiveFails,
	}

	if status.LastVerification != nil {
		lastResp := toVerificationResponse(status.LastVerification)
		resp.LastVerification = &lastResp
	}

	if status.NextScheduledAt != nil {
		resp.NextScheduledAt = status.NextScheduledAt.Format("2006-01-02T15:04:05Z07:00")
	}

	c.JSON(http.StatusOK, resp)
}

// TriggerVerification manually triggers a verification for a repository.
// POST /api/v1/repositories/:id/verifications
func (h *VerificationsHandler) TriggerVerification(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req TriggerVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	verification, err := h.trigger.TriggerVerification(c.Request.Context(), repoID, models.VerificationType(req.Type))
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Str("type", req.Type).Msg("failed to trigger verification")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger verification"})
		return
	}

	h.logger.Info().
		Str("repo_id", repoID.String()).
		Str("verification_id", verification.ID.String()).
		Str("type", req.Type).
		Msg("verification triggered")

	c.JSON(http.StatusAccepted, toVerificationResponse(verification))
}

// ListSchedules returns verification schedules for a repository.
// GET /api/v1/repositories/:id/verification-schedules
func (h *VerificationsHandler) ListSchedules(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	schedules, err := h.store.GetVerificationSchedulesByRepoID(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to list verification schedules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list verification schedules"})
		return
	}

	responses := make([]VerificationScheduleResponse, len(schedules))
	for i, vs := range schedules {
		responses[i] = toVerificationScheduleResponse(vs)
	}

	c.JSON(http.StatusOK, gin.H{"schedules": responses})
}

// CreateSchedule creates a new verification schedule.
// POST /api/v1/repositories/:id/verification-schedules
func (h *VerificationsHandler) CreateSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req CreateVerificationScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	schedule := models.NewVerificationSchedule(repoID, models.VerificationType(req.Type), req.CronExpression)
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	schedule.ReadDataSubset = req.ReadDataSubset

	if err := h.store.CreateVerificationSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to create verification schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create verification schedule"})
		return
	}

	h.logger.Info().
		Str("repo_id", repoID.String()).
		Str("schedule_id", schedule.ID.String()).
		Str("type", req.Type).
		Msg("verification schedule created")

	c.JSON(http.StatusCreated, toVerificationScheduleResponse(schedule))
}

// GetSchedule returns a verification schedule by ID.
// GET /api/v1/verification-schedules/:id
func (h *VerificationsHandler) GetSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetVerificationScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, schedule.RepositoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	c.JSON(http.StatusOK, toVerificationScheduleResponse(schedule))
}

// UpdateSchedule updates a verification schedule.
// PUT /api/v1/verification-schedules/:id
func (h *VerificationsHandler) UpdateSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var req UpdateVerificationScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	schedule, err := h.store.GetVerificationScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, schedule.RepositoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	// Update fields
	if req.CronExpression != "" {
		schedule.CronExpression = req.CronExpression
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	if req.ReadDataSubset != "" {
		schedule.ReadDataSubset = req.ReadDataSubset
	}

	if err := h.store.UpdateVerificationSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update verification schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update verification schedule"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Msg("verification schedule updated")
	c.JSON(http.StatusOK, toVerificationScheduleResponse(schedule))
}

// DeleteSchedule deletes a verification schedule.
// DELETE /api/v1/verification-schedules/:id
func (h *VerificationsHandler) DeleteSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetVerificationScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, schedule.RepositoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	if err := h.store.DeleteVerificationSchedule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to delete verification schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete verification schedule"})
		return
	}

	h.logger.Info().Str("schedule_id", id.String()).Msg("verification schedule deleted")
	c.JSON(http.StatusOK, gin.H{"message": "verification schedule deleted"})
}

// verifyRepoAccess checks if the user has access to the repository.
func (h *VerificationsHandler) verifyRepoAccess(c *gin.Context, userID, repoID uuid.UUID) error {
	dbUser, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil {
		return err
	}

	if repo.OrgID != dbUser.OrgID {
		return err
	}

	return nil
}
