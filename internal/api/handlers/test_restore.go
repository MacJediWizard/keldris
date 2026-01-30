package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// TestRestoreStore defines the interface for test restore persistence operations.
type TestRestoreStore interface {
	GetTestRestoreSettingsByID(ctx context.Context, id uuid.UUID) (*models.TestRestoreSettings, error)
	GetTestRestoreSettingsByRepoID(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreSettings, error)
	CreateTestRestoreSettings(ctx context.Context, settings *models.TestRestoreSettings) error
	UpdateTestRestoreSettings(ctx context.Context, settings *models.TestRestoreSettings) error
	DeleteTestRestoreSettings(ctx context.Context, id uuid.UUID) error
	GetTestRestoreResultsByRepoID(ctx context.Context, repoID uuid.UUID, limit int) ([]*models.TestRestoreResult, error)
	GetTestRestoreResultByID(ctx context.Context, id uuid.UUID) (*models.TestRestoreResult, error)
	GetLatestTestRestoreResultByRepoID(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreResult, error)
	GetConsecutiveFailedTestRestores(ctx context.Context, repoID uuid.UUID) (int, error)
	GetTestRestoreSummaryByOrgID(ctx context.Context, orgID uuid.UUID) (*db.TestRestoreSummary, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// TestRestoreTrigger allows manually triggering test restores.
type TestRestoreTrigger interface {
	TriggerTestRestore(ctx context.Context, repoID uuid.UUID, samplePercentage int) (*models.TestRestoreResult, error)
	GetRepositoryTestRestoreStatus(ctx context.Context, repoID uuid.UUID) (*models.TestRestoreStatus, error)
}

// TestRestoreHandler handles test restore-related HTTP endpoints.
type TestRestoreHandler struct {
	store   TestRestoreStore
	trigger TestRestoreTrigger
	logger  zerolog.Logger
}

// NewTestRestoreHandler creates a new TestRestoreHandler.
func NewTestRestoreHandler(store TestRestoreStore, trigger TestRestoreTrigger, logger zerolog.Logger) *TestRestoreHandler {
	return &TestRestoreHandler{
		store:   store,
		trigger: trigger,
		logger:  logger.With().Str("component", "test_restore_handler").Logger(),
	}
}

// RegisterRoutes registers test restore routes on the given router group.
func (h *TestRestoreHandler) RegisterRoutes(r *gin.RouterGroup) {
	// Dashboard summary endpoint
	dashboard := r.Group("/dashboard")
	{
		dashboard.GET("/test-restore-summary", h.GetSummary)
	}

	// Repository test restore endpoints
	repos := r.Group("/repositories")
	{
		repos.GET("/:id/test-restore-status", h.GetStatus)
		repos.GET("/:id/test-restore-results", h.ListResults)
		repos.POST("/:id/test-restore", h.TriggerTestRestore)
		repos.GET("/:id/test-restore-settings", h.GetSettings)
		repos.POST("/:id/test-restore-settings", h.CreateSettings)
		repos.PUT("/:id/test-restore-settings", h.UpdateSettings)
		repos.DELETE("/:id/test-restore-settings", h.DeleteSettings)
	}

	// Test restore result endpoints
	results := r.Group("/test-restore-results")
	{
		results.GET("/:id", h.GetResult)
	}
}

// TestRestoreResultResponse is the API response for a test restore result.
type TestRestoreResultResponse struct {
	ID               string                      `json:"id"`
	RepositoryID     string                      `json:"repository_id"`
	SnapshotID       string                      `json:"snapshot_id,omitempty"`
	SamplePercentage int                         `json:"sample_percentage"`
	StartedAt        string                      `json:"started_at"`
	CompletedAt      string                      `json:"completed_at,omitempty"`
	Status           string                      `json:"status"`
	DurationMs       *int64                      `json:"duration_ms,omitempty"`
	FilesRestored    int                         `json:"files_restored"`
	FilesVerified    int                         `json:"files_verified"`
	BytesRestored    int64                       `json:"bytes_restored"`
	ErrorMessage     string                      `json:"error_message,omitempty"`
	Details          *models.TestRestoreDetails  `json:"details,omitempty"`
	CreatedAt        string                      `json:"created_at"`
}

func toTestRestoreResultResponse(r *models.TestRestoreResult) TestRestoreResultResponse {
	resp := TestRestoreResultResponse{
		ID:               r.ID.String(),
		RepositoryID:     r.RepositoryID.String(),
		SnapshotID:       r.SnapshotID,
		SamplePercentage: r.SamplePercentage,
		StartedAt:        r.StartedAt.Format("2006-01-02T15:04:05Z07:00"),
		Status:           string(r.Status),
		DurationMs:       r.DurationMs,
		FilesRestored:    r.FilesRestored,
		FilesVerified:    r.FilesVerified,
		BytesRestored:    r.BytesRestored,
		ErrorMessage:     r.ErrorMessage,
		Details:          r.Details,
		CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if r.CompletedAt != nil {
		resp.CompletedAt = r.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	return resp
}

// TestRestoreSettingsResponse is the API response for test restore settings.
type TestRestoreSettingsResponse struct {
	ID               string `json:"id"`
	RepositoryID     string `json:"repository_id"`
	Enabled          bool   `json:"enabled"`
	Frequency        string `json:"frequency"`
	CronExpression   string `json:"cron_expression"`
	SamplePercentage int    `json:"sample_percentage"`
	LastRunAt        string `json:"last_run_at,omitempty"`
	LastRunStatus    string `json:"last_run_status,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func toTestRestoreSettingsResponse(s *models.TestRestoreSettings) TestRestoreSettingsResponse {
	resp := TestRestoreSettingsResponse{
		ID:               s.ID.String(),
		RepositoryID:     s.RepositoryID.String(),
		Enabled:          s.Enabled,
		Frequency:        string(s.Frequency),
		CronExpression:   s.CronExpression,
		SamplePercentage: s.SamplePercentage,
		CreatedAt:        s.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:        s.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if s.LastRunAt != nil {
		resp.LastRunAt = s.LastRunAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if s.LastRunStatus != nil {
		resp.LastRunStatus = *s.LastRunStatus
	}
	return resp
}

// TestRestoreStatusResponse is the API response for repository test restore status.
type TestRestoreStatusResponse struct {
	RepositoryID     string                       `json:"repository_id"`
	Settings         *TestRestoreSettingsResponse `json:"settings,omitempty"`
	LastResult       *TestRestoreResultResponse   `json:"last_result,omitempty"`
	NextScheduledAt  string                       `json:"next_scheduled_at,omitempty"`
	ConsecutiveFails int                          `json:"consecutive_fails"`
}

// TriggerTestRestoreRequest is the request body for triggering a test restore.
type TriggerTestRestoreRequest struct {
	SamplePercentage int `json:"sample_percentage,omitempty"`
}

// CreateTestRestoreSettingsRequest is the request body for creating test restore settings.
type CreateTestRestoreSettingsRequest struct {
	Frequency        string `json:"frequency" binding:"required,oneof=weekly monthly custom"`
	CronExpression   string `json:"cron_expression,omitempty"`
	SamplePercentage int    `json:"sample_percentage" binding:"required,min=1,max=100"`
	Enabled          *bool  `json:"enabled,omitempty"`
}

// UpdateTestRestoreSettingsRequest is the request body for updating test restore settings.
type UpdateTestRestoreSettingsRequest struct {
	Frequency        string `json:"frequency,omitempty"`
	CronExpression   string `json:"cron_expression,omitempty"`
	SamplePercentage int    `json:"sample_percentage,omitempty"`
	Enabled          *bool  `json:"enabled,omitempty"`
}

// GetSummary returns test restore summary for the dashboard.
// GET /api/v1/dashboard/test-restore-summary
func (h *TestRestoreHandler) GetSummary(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}

	summary, err := h.store.GetTestRestoreSummaryByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get test restore summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get test restore summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"repositories_with_testing":      summary.RepositoriesWithTesting,
		"repositories_needing_attention": summary.RepositoriesNeedingAttention,
		"total_passed":                   summary.TotalPassed,
		"total_failed":                   summary.TotalFailed,
		"last_test_at":                   summary.LastTestAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetStatus returns the test restore status for a repository.
// GET /api/v1/repositories/:id/test-restore-status
func (h *TestRestoreHandler) GetStatus(c *gin.Context) {
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

	status, err := h.trigger.GetRepositoryTestRestoreStatus(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to get test restore status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get test restore status"})
		return
	}

	resp := TestRestoreStatusResponse{
		RepositoryID:     status.RepositoryID.String(),
		ConsecutiveFails: status.ConsecutiveFails,
	}

	if status.Settings != nil {
		settingsResp := toTestRestoreSettingsResponse(status.Settings)
		resp.Settings = &settingsResp
	}

	if status.LastResult != nil {
		resultResp := toTestRestoreResultResponse(status.LastResult)
		resp.LastResult = &resultResp
	}

	if status.NextScheduledAt != nil {
		resp.NextScheduledAt = status.NextScheduledAt.Format("2006-01-02T15:04:05Z07:00")
	}

	c.JSON(http.StatusOK, resp)
}

// ListResults returns test restore results for a repository.
// GET /api/v1/repositories/:id/test-restore-results
func (h *TestRestoreHandler) ListResults(c *gin.Context) {
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

	results, err := h.store.GetTestRestoreResultsByRepoID(c.Request.Context(), repoID, 50)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to list test restore results")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list test restore results"})
		return
	}

	responses := make([]TestRestoreResultResponse, len(results))
	for i, r := range results {
		responses[i] = toTestRestoreResultResponse(r)
	}

	c.JSON(http.StatusOK, gin.H{"results": responses})
}

// GetResult returns a specific test restore result by ID.
// GET /api/v1/test-restore-results/:id
func (h *TestRestoreHandler) GetResult(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid result ID"})
		return
	}

	result, err := h.store.GetTestRestoreResultByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "result not found"})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, result.RepositoryID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "result not found"})
		return
	}

	c.JSON(http.StatusOK, toTestRestoreResultResponse(result))
}

// TriggerTestRestore manually triggers a test restore for a repository.
// POST /api/v1/repositories/:id/test-restore
func (h *TestRestoreHandler) TriggerTestRestore(c *gin.Context) {
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

	var req TriggerTestRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body, use default
		req.SamplePercentage = 10
	}

	if req.SamplePercentage <= 0 || req.SamplePercentage > 100 {
		req.SamplePercentage = 10
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	result, err := h.trigger.TriggerTestRestore(c.Request.Context(), repoID, req.SamplePercentage)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to trigger test restore")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to trigger test restore"})
		return
	}

	h.logger.Info().
		Str("repo_id", repoID.String()).
		Str("result_id", result.ID.String()).
		Int("sample_percentage", req.SamplePercentage).
		Msg("test restore triggered")

	c.JSON(http.StatusAccepted, toTestRestoreResultResponse(result))
}

// GetSettings returns test restore settings for a repository.
// GET /api/v1/repositories/:id/test-restore-settings
func (h *TestRestoreHandler) GetSettings(c *gin.Context) {
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

	settings, err := h.store.GetTestRestoreSettingsByRepoID(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to get test restore settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get test restore settings"})
		return
	}

	if settings == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test restore settings not configured"})
		return
	}

	c.JSON(http.StatusOK, toTestRestoreSettingsResponse(settings))
}

// CreateSettings creates test restore settings for a repository.
// POST /api/v1/repositories/:id/test-restore-settings
func (h *TestRestoreHandler) CreateSettings(c *gin.Context) {
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

	var req CreateTestRestoreSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Check if settings already exist
	existing, err := h.store.GetTestRestoreSettingsByRepoID(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to check existing settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing settings"})
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "test restore settings already exist for this repository"})
		return
	}

	settings := models.NewTestRestoreSettings(repoID)
	settings.SetFrequency(models.TestRestoreFrequency(req.Frequency), req.CronExpression)
	settings.SamplePercentage = req.SamplePercentage
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}

	if err := h.store.CreateTestRestoreSettings(c.Request.Context(), settings); err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to create test restore settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create test restore settings"})
		return
	}

	h.logger.Info().
		Str("repo_id", repoID.String()).
		Str("settings_id", settings.ID.String()).
		Str("frequency", req.Frequency).
		Int("sample_percentage", req.SamplePercentage).
		Msg("test restore settings created")

	c.JSON(http.StatusCreated, toTestRestoreSettingsResponse(settings))
}

// UpdateSettings updates test restore settings for a repository.
// PUT /api/v1/repositories/:id/test-restore-settings
func (h *TestRestoreHandler) UpdateSettings(c *gin.Context) {
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

	var req UpdateTestRestoreSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	if err := h.verifyRepoAccess(c, user.ID, repoID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	settings, err := h.store.GetTestRestoreSettingsByRepoID(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to get test restore settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get test restore settings"})
		return
	}
	if settings == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test restore settings not configured"})
		return
	}

	// Update fields
	if req.Frequency != "" {
		settings.SetFrequency(models.TestRestoreFrequency(req.Frequency), req.CronExpression)
	} else if req.CronExpression != "" {
		settings.CronExpression = req.CronExpression
	}
	if req.SamplePercentage > 0 && req.SamplePercentage <= 100 {
		settings.SamplePercentage = req.SamplePercentage
	}
	if req.Enabled != nil {
		settings.Enabled = *req.Enabled
	}

	if err := h.store.UpdateTestRestoreSettings(c.Request.Context(), settings); err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to update test restore settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update test restore settings"})
		return
	}

	h.logger.Info().Str("repo_id", repoID.String()).Msg("test restore settings updated")
	c.JSON(http.StatusOK, toTestRestoreSettingsResponse(settings))
}

// DeleteSettings deletes test restore settings for a repository.
// DELETE /api/v1/repositories/:id/test-restore-settings
func (h *TestRestoreHandler) DeleteSettings(c *gin.Context) {
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

	settings, err := h.store.GetTestRestoreSettingsByRepoID(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to get test restore settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get test restore settings"})
		return
	}
	if settings == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test restore settings not configured"})
		return
	}

	if err := h.store.DeleteTestRestoreSettings(c.Request.Context(), settings.ID); err != nil {
		h.logger.Error().Err(err).Str("repo_id", repoID.String()).Msg("failed to delete test restore settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete test restore settings"})
		return
	}

	h.logger.Info().Str("repo_id", repoID.String()).Msg("test restore settings deleted")
	c.JSON(http.StatusOK, gin.H{"message": "test restore settings deleted"})
}

// verifyRepoAccess checks if the user has access to the repository.
func (h *TestRestoreHandler) verifyRepoAccess(c *gin.Context, userID, repoID uuid.UUID) error {
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
