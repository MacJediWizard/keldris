package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DRTestStore defines the interface for DR test persistence operations.
type DRTestStore interface {
	GetDRTestsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DRTest, error)
	GetDRTestsByRunbookID(ctx context.Context, runbookID uuid.UUID) ([]*models.DRTest, error)
	GetDRTestByID(ctx context.Context, id uuid.UUID) (*models.DRTest, error)
	CreateDRTest(ctx context.Context, test *models.DRTest) error
	UpdateDRTest(ctx context.Context, test *models.DRTest) error
	GetDRRunbookByID(ctx context.Context, id uuid.UUID) (*models.DRRunbook, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// DRTestRunner defines the interface for running DR tests.
type DRTestRunner interface {
	TriggerDRTest(ctx context.Context, runbookID uuid.UUID) error
}

// DRTestsHandler handles DR test-related HTTP endpoints.
type DRTestsHandler struct {
	store   DRTestStore
	runner  DRTestRunner
	checker *license.FeatureChecker
	logger  zerolog.Logger
}

// NewDRTestsHandler creates a new DRTestsHandler.
func NewDRTestsHandler(store DRTestStore, runner DRTestRunner, checker *license.FeatureChecker, logger zerolog.Logger) *DRTestsHandler {
	return &DRTestsHandler{
		store:   store,
		runner:  runner,
		checker: checker,
		logger:  logger.With().Str("component", "dr_tests_handler").Logger(),
	}
}

// RegisterRoutes registers DR test routes on the given router group.
func (h *DRTestsHandler) RegisterRoutes(r *gin.RouterGroup) {
	drTests := r.Group("/dr-tests")
	{
		drTests.GET("", h.List)
		drTests.GET("/:id", h.Get)
		drTests.POST("", h.Run)
		drTests.POST("/:id/cancel", h.Cancel)
	}
}

// RunDRTestRequest is the request body for running a DR test.
type RunDRTestRequest struct {
	RunbookID uuid.UUID `json:"runbook_id" binding:"required"`
	Notes     string    `json:"notes,omitempty"`
}

// CancelDRTestRequest is the request body for canceling a DR test.
type CancelDRTestRequest struct {
	Notes string `json:"notes,omitempty"`
}

// List returns all DR tests for the authenticated user's organization.
// GET /api/v1/dr-tests
// Optional query params: runbook_id, status
func (h *DRTestsHandler) List(c *gin.Context) {
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

	// Check for runbook_id filter
	runbookIDParam := c.Query("runbook_id")
	if runbookIDParam != "" {
		runbookID, err := uuid.Parse(runbookIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid runbook_id"})
			return
		}

		// Verify runbook belongs to user's org
		runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), runbookID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
			return
		}
		if runbook.OrgID != dbUser.OrgID {
			c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
			return
		}

		tests, err := h.store.GetDRTestsByRunbookID(c.Request.Context(), runbookID)
		if err != nil {
			h.logger.Error().Err(err).Str("runbook_id", runbookID.String()).Msg("failed to list DR tests")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tests"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"tests": tests})
		return
	}

	// Get all tests for the org
	tests, err := h.store.GetDRTestsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list DR tests")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tests"})
		return
	}

	// Filter by status if provided
	statusFilter := c.Query("status")
	if statusFilter != "" {
		var filtered []*models.DRTest
		for _, t := range tests {
			if string(t.Status) == statusFilter {
				filtered = append(filtered, t)
			}
		}
		tests = filtered
	}

	c.JSON(http.StatusOK, gin.H{"tests": tests})
}

// Get returns a specific DR test by ID.
// GET /api/v1/dr-tests/:id
func (h *DRTestsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test ID"})
		return
	}

	test, err := h.store.GetDRTestByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("test_id", id.String()).Msg("failed to get test")
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	if err := h.verifyTestAccess(c, user.ID, test); err != nil {
		return
	}

	c.JSON(http.StatusOK, test)
}

// Run triggers a new DR test.
// POST /api/v1/dr-tests
func (h *DRTestsHandler) Run(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureDRTests) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req RunDRTestRequest
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

	// Verify runbook belongs to user's org
	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), req.RunbookID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}
	if runbook.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "runbook not found"})
		return
	}

	// Create test record
	test := models.NewDRTest(req.RunbookID)
	test.Notes = req.Notes

	// Associate with schedule if runbook has one
	if runbook.ScheduleID != nil {
		test.SetSchedule(*runbook.ScheduleID)
	}

	if err := h.store.CreateDRTest(c.Request.Context(), test); err != nil {
		h.logger.Error().Err(err).Str("runbook_id", req.RunbookID.String()).Msg("failed to create DR test")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create test"})
		return
	}

	// Trigger the test execution if runner is available
	if h.runner != nil {
		go func() {
			if err := h.runner.TriggerDRTest(context.Background(), req.RunbookID); err != nil {
				h.logger.Error().Err(err).Str("test_id", test.ID.String()).Msg("failed to trigger DR test")
			}
		}()
	}

	h.logger.Info().
		Str("test_id", test.ID.String()).
		Str("runbook_id", req.RunbookID.String()).
		Msg("DR test triggered")

	c.JSON(http.StatusCreated, test)
}

// Cancel cancels a running DR test.
// POST /api/v1/dr-tests/:id/cancel
func (h *DRTestsHandler) Cancel(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid test ID"})
		return
	}

	var req CancelDRTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body
		req = CancelDRTestRequest{}
	}

	test, err := h.store.GetDRTestByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return
	}

	if err := h.verifyTestAccess(c, user.ID, test); err != nil {
		return
	}

	// Can only cancel scheduled or running tests
	if test.Status != models.DRTestStatusScheduled && test.Status != models.DRTestStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "test cannot be canceled in current status"})
		return
	}

	test.Cancel(req.Notes)
	if err := h.store.UpdateDRTest(c.Request.Context(), test); err != nil {
		h.logger.Error().Err(err).Str("test_id", id.String()).Msg("failed to cancel test")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel test"})
		return
	}

	h.logger.Info().Str("test_id", id.String()).Msg("DR test canceled")
	c.JSON(http.StatusOK, test)
}

// verifyTestAccess checks if the user has access to the test.
func (h *DRTestsHandler) verifyTestAccess(c *gin.Context, userID uuid.UUID, test *models.DRTest) error {
	dbUser, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", userID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return err
	}

	runbook, err := h.store.GetDRRunbookByID(c.Request.Context(), test.RunbookID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return err
	}

	if runbook.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "test not found"})
		return err
	}

	return nil
}
