package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/reports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ReportStore defines the interface for report persistence operations.
type ReportStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetReportSchedulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.ReportSchedule, error)
	GetReportScheduleByID(ctx context.Context, id uuid.UUID) (*models.ReportSchedule, error)
	CreateReportSchedule(ctx context.Context, schedule *models.ReportSchedule) error
	UpdateReportSchedule(ctx context.Context, schedule *models.ReportSchedule) error
	DeleteReportSchedule(ctx context.Context, id uuid.UUID) error
	GetReportHistoryByOrgID(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.ReportHistory, error)
	GetReportHistoryByID(ctx context.Context, id uuid.UUID) (*models.ReportHistory, error)
}

// ReportsHandler handles report-related HTTP endpoints.
type ReportsHandler struct {
	store     ReportStore
	scheduler *reports.Scheduler
	logger    zerolog.Logger
}

// NewReportsHandler creates a new ReportsHandler.
func NewReportsHandler(store ReportStore, scheduler *reports.Scheduler, logger zerolog.Logger) *ReportsHandler {
	return &ReportsHandler{
		store:     store,
		scheduler: scheduler,
		logger:    logger.With().Str("component", "reports_handler").Logger(),
	}
}

// RegisterRoutes registers report routes on the given router group.
func (h *ReportsHandler) RegisterRoutes(r *gin.RouterGroup) {
	rpts := r.Group("/reports")
	{
		// Schedules
		rpts.GET("/schedules", h.ListSchedules)
		rpts.POST("/schedules", h.CreateSchedule)
		rpts.GET("/schedules/:id", h.GetSchedule)
		rpts.PUT("/schedules/:id", h.UpdateSchedule)
		rpts.DELETE("/schedules/:id", h.DeleteSchedule)

		// Actions
		rpts.POST("/schedules/:id/send", h.SendReport)
		rpts.POST("/preview", h.PreviewReport)

		// History
		rpts.GET("/history", h.ListHistory)
		rpts.GET("/history/:id", h.GetHistoryEntry)
	}
}

// ListSchedules returns all report schedules for the organization.
// GET /api/v1/reports/schedules
func (h *ReportsHandler) ListSchedules(c *gin.Context) {
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

	schedules, err := h.store.GetReportSchedulesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list report schedules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list report schedules"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedules": schedules})
}

// GetSchedule returns a specific report schedule.
// GET /api/v1/reports/schedules/:id
func (h *ReportsHandler) GetSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetReportScheduleByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to get report schedule")
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if schedule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// CreateSchedule creates a new report schedule.
// POST /api/v1/reports/schedules
func (h *ReportsHandler) CreateSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.CreateReportScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate frequency
	if !isValidFrequency(req.Frequency) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frequency, must be daily, weekly, or monthly"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	schedule := models.NewReportSchedule(dbUser.OrgID, req.Name, models.ReportFrequency(req.Frequency), req.Recipients)

	if req.Timezone != "" {
		schedule.Timezone = req.Timezone
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	if req.ChannelID != nil {
		channelID, err := uuid.Parse(*req.ChannelID)
		if err == nil {
			schedule.ChannelID = &channelID
		}
	}

	if err := h.store.CreateReportSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Msg("failed to create report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create report schedule"})
		return
	}

	c.JSON(http.StatusCreated, schedule)
}

// UpdateSchedule updates an existing report schedule.
// PUT /api/v1/reports/schedules/:id
func (h *ReportsHandler) UpdateSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var req models.UpdateReportScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schedule, err := h.store.GetReportScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if schedule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		schedule.Name = *req.Name
	}
	if req.Frequency != nil {
		if !isValidFrequency(*req.Frequency) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frequency"})
			return
		}
		schedule.Frequency = models.ReportFrequency(*req.Frequency)
	}
	if req.Recipients != nil {
		schedule.Recipients = req.Recipients
	}
	if req.Timezone != nil {
		schedule.Timezone = *req.Timezone
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}
	if req.ChannelID != nil {
		if *req.ChannelID == "" {
			schedule.ChannelID = nil
		} else {
			channelID, err := uuid.Parse(*req.ChannelID)
			if err == nil {
				schedule.ChannelID = &channelID
			}
		}
	}

	if err := h.store.UpdateReportSchedule(c.Request.Context(), schedule); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to update report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update report schedule"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// DeleteSchedule deletes a report schedule.
// DELETE /api/v1/reports/schedules/:id
func (h *ReportsHandler) DeleteSchedule(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	schedule, err := h.store.GetReportScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if schedule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	if err := h.store.DeleteReportSchedule(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("schedule_id", id.String()).Msg("failed to delete report schedule")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete report schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "report schedule deleted"})
}

// SendReport manually triggers sending a report.
// POST /api/v1/reports/schedules/:id/send
func (h *ReportsHandler) SendReport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule ID"})
		return
	}

	var req models.SendReportRequest
	_ = c.ShouldBindJSON(&req) // Optional body

	schedule, err := h.store.GetReportScheduleByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if schedule.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "report schedule not found"})
		return
	}

	// Override recipients if provided
	if len(req.Recipients) > 0 {
		schedule.Recipients = req.Recipients
	}

	// Generate and send report
	periodStart, periodEnd := reports.CalculatePeriod(schedule.Frequency, schedule.Timezone)
	data, _, _, err := h.scheduler.GeneratePreview(c.Request.Context(), schedule.OrgID, schedule.Frequency, schedule.Timezone)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate report"})
		return
	}

	err = h.scheduler.SendReport(c.Request.Context(), schedule, data, periodStart, periodEnd, req.Preview)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to send report")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Preview {
		c.JSON(http.StatusOK, gin.H{
			"message": "report preview generated",
			"data":    data,
			"period": gin.H{
				"start": periodStart,
				"end":   periodEnd,
			},
		})
	} else {
		c.JSON(http.StatusOK, gin.H{"message": "report sent successfully"})
	}
}

// PreviewReport generates a report preview without saving.
// POST /api/v1/reports/preview
func (h *ReportsHandler) PreviewReport(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.PreviewReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !isValidFrequency(req.Frequency) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid frequency"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	timezone := req.Timezone
	if timezone == "" {
		timezone = "UTC"
	}

	data, periodStart, periodEnd, err := h.scheduler.GeneratePreview(
		c.Request.Context(),
		dbUser.OrgID,
		models.ReportFrequency(req.Frequency),
		timezone,
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to generate preview")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate preview"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"period": gin.H{
			"start": periodStart,
			"end":   periodEnd,
		},
	})
}

// ListHistory returns report history for the organization.
// GET /api/v1/reports/history
func (h *ReportsHandler) ListHistory(c *gin.Context) {
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

	history, err := h.store.GetReportHistoryByOrgID(c.Request.Context(), dbUser.OrgID, 100)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list report history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list report history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"history": history})
}

// GetHistoryEntry returns a specific report history entry.
// GET /api/v1/reports/history/:id
func (h *ReportsHandler) GetHistoryEntry(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid history ID"})
		return
	}

	entry, err := h.store.GetReportHistoryByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report history not found"})
		return
	}

	// Verify org ownership
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if entry.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "report history not found"})
		return
	}

	c.JSON(http.StatusOK, entry)
}

func isValidFrequency(f string) bool {
	switch models.ReportFrequency(f) {
	case models.ReportFrequencyDaily, models.ReportFrequencyWeekly, models.ReportFrequencyMonthly:
		return true
	default:
		return false
	}
}
