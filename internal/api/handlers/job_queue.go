package handlers

import (
	"context"
	"net/http"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/auth"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// JobQueueStore defines the interface for job queue persistence operations.
type JobQueueStore interface {
	GetJobByID(ctx context.Context, id uuid.UUID) (*models.Job, error)
	GetJobsWithDetails(ctx context.Context, orgID uuid.UUID, status *models.JobStatus, limit int) ([]*models.JobWithDetails, error)
	GetJobQueueSummary(ctx context.Context, orgID uuid.UUID) (*models.JobQueueSummary, error)
	UpdateJob(ctx context.Context, job *models.Job) error
	DeleteJob(ctx context.Context, id uuid.UUID) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// JobQueueHandler handles job queue-related HTTP endpoints.
type JobQueueHandler struct {
	store  JobQueueStore
	rbac   *auth.RBAC
	logger zerolog.Logger
}

// NewJobQueueHandler creates a new JobQueueHandler.
func NewJobQueueHandler(store JobQueueStore, rbac *auth.RBAC, logger zerolog.Logger) *JobQueueHandler {
	return &JobQueueHandler{
		store:  store,
		rbac:   rbac,
		logger: logger.With().Str("component", "job_queue_handler").Logger(),
	}
}

// RegisterRoutes registers job queue routes on the given router group.
func (h *JobQueueHandler) RegisterRoutes(r *gin.RouterGroup) {
	jobs := r.Group("/job-queue")
	{
		jobs.GET("", h.ListJobs)
		jobs.GET("/summary", h.GetSummary)
		jobs.GET("/:id", h.GetJob)
		jobs.DELETE("/:id", h.CancelJob)
		jobs.POST("/:id/retry", h.RetryJob)
	}
}

// JobResponse is the API response for a job.
type JobResponse struct {
	ID             string                 `json:"id"`
	OrgID          string                 `json:"org_id"`
	JobType        string                 `json:"job_type"`
	Priority       int                    `json:"priority"`
	Status         string                 `json:"status"`
	Payload        models.JobPayload      `json:"payload"`
	RetryCount     int                    `json:"retry_count"`
	MaxRetries     int                    `json:"max_retries"`
	NextRetryAt    string                 `json:"next_retry_at,omitempty"`
	ErrorMessage   string                 `json:"error_message,omitempty"`
	LastErrorAt    string                 `json:"last_error_at,omitempty"`
	CreatedAt      string                 `json:"created_at"`
	StartedAt      string                 `json:"started_at,omitempty"`
	CompletedAt    string                 `json:"completed_at,omitempty"`
	AgentID        string                 `json:"agent_id,omitempty"`
	RepositoryID   string                 `json:"repository_id,omitempty"`
	ScheduleID     string                 `json:"schedule_id,omitempty"`
	AgentHostname  string                 `json:"agent_hostname,omitempty"`
	RepositoryName string                 `json:"repository_name,omitempty"`
	ScheduleName   string                 `json:"schedule_name,omitempty"`
	QueuePosition  int                    `json:"queue_position,omitempty"`
}

func toJobResponse(j *models.Job) JobResponse {
	resp := JobResponse{
		ID:           j.ID.String(),
		OrgID:        j.OrgID.String(),
		JobType:      string(j.JobType),
		Priority:     j.Priority,
		Status:       string(j.Status),
		Payload:      j.Payload,
		RetryCount:   j.RetryCount,
		MaxRetries:   j.MaxRetries,
		ErrorMessage: j.ErrorMessage,
		CreatedAt:    j.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if j.NextRetryAt != nil {
		resp.NextRetryAt = j.NextRetryAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if j.LastErrorAt != nil {
		resp.LastErrorAt = j.LastErrorAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if j.StartedAt != nil {
		resp.StartedAt = j.StartedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if j.CompletedAt != nil {
		resp.CompletedAt = j.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
	}
	if j.AgentID != nil {
		resp.AgentID = j.AgentID.String()
	}
	if j.RepositoryID != nil {
		resp.RepositoryID = j.RepositoryID.String()
	}
	if j.ScheduleID != nil {
		resp.ScheduleID = j.ScheduleID.String()
	}

	return resp
}

func toJobResponseWithDetails(j *models.JobWithDetails) JobResponse {
	resp := toJobResponse(&j.Job)
	resp.AgentHostname = j.AgentHostname
	resp.RepositoryName = j.RepositoryName
	resp.ScheduleName = j.ScheduleName
	resp.QueuePosition = j.QueuePosition
	return resp
}

// JobQueueSummaryResponse is the API response for queue summary.
type JobQueueSummaryResponse struct {
	TotalPending    int                    `json:"total_pending"`
	TotalRunning    int                    `json:"total_running"`
	TotalCompleted  int                    `json:"total_completed"`
	TotalFailed     int                    `json:"total_failed"`
	TotalDeadLetter int                    `json:"total_dead_letter"`
	ByType          map[string]int         `json:"by_type,omitempty"`
	AvgWaitMinutes  float64                `json:"avg_wait_minutes"`
	OldestPending   string                 `json:"oldest_pending,omitempty"`
}

func toJobQueueSummaryResponse(s *models.JobQueueSummary) JobQueueSummaryResponse {
	resp := JobQueueSummaryResponse{
		TotalPending:    s.TotalPending,
		TotalRunning:    s.TotalRunning,
		TotalCompleted:  s.TotalCompleted,
		TotalFailed:     s.TotalFailed,
		TotalDeadLetter: s.TotalDeadLetter,
		AvgWaitMinutes:  s.AvgWaitMinutes,
	}

	if s.OldestPending != nil {
		resp.OldestPending = s.OldestPending.Format("2006-01-02T15:04:05Z07:00")
	}

	if len(s.ByType) > 0 {
		resp.ByType = make(map[string]int)
		for k, v := range s.ByType {
			resp.ByType[string(k)] = v
		}
	}

	return resp
}

// ListJobs returns all jobs for the current organization.
// GET /api/v1/job-queue
func (h *JobQueueHandler) ListJobs(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	// Parse query parameters
	var status *models.JobStatus
	if statusParam := c.Query("status"); statusParam != "" {
		s := models.JobStatus(statusParam)
		status = &s
	}

	limit := 100
	// Note: limit parameter parsing can be added if needed

	jobs, err := h.store.GetJobsWithDetails(c.Request.Context(), user.CurrentOrgID, status, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to list jobs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list jobs"})
		return
	}

	response := make([]JobResponse, len(jobs))
	for i, j := range jobs {
		response[i] = toJobResponseWithDetails(j)
	}

	c.JSON(http.StatusOK, gin.H{"jobs": response})
}

// GetSummary returns queue statistics for the current organization.
// GET /api/v1/job-queue/summary
func (h *JobQueueHandler) GetSummary(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	summary, err := h.store.GetJobQueueSummary(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get job queue summary")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get job queue summary"})
		return
	}

	c.JSON(http.StatusOK, toJobQueueSummaryResponse(summary))
}

// GetJob returns a specific job by ID.
// GET /api/v1/job-queue/:id
func (h *JobQueueHandler) GetJob(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleRead); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	job, err := h.store.GetJobByID(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error().Err(err).Str("job_id", jobID.String()).Msg("failed to get job")
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Verify job belongs to user's organization
	if job.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, toJobResponse(job))
}

// CancelJob cancels a pending job.
// DELETE /api/v1/job-queue/:id
func (h *JobQueueHandler) CancelJob(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	job, err := h.store.GetJobByID(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error().Err(err).Str("job_id", jobID.String()).Msg("failed to get job")
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Verify job belongs to user's organization
	if job.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Can only cancel pending jobs
	if job.Status != models.JobStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "can only cancel pending jobs"})
		return
	}

	if !job.Cancel() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to cancel job"})
		return
	}

	if err := h.store.UpdateJob(c.Request.Context(), job); err != nil {
		h.logger.Error().Err(err).Str("job_id", jobID.String()).Msg("failed to update job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel job"})
		return
	}

	h.logger.Info().
		Str("job_id", jobID.String()).
		Str("user_id", user.ID.String()).
		Msg("job canceled")

	c.JSON(http.StatusOK, gin.H{"message": "job canceled"})
}

// RetryJob retries a failed or dead letter job.
// POST /api/v1/job-queue/:id/retry
func (h *JobQueueHandler) RetryJob(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job ID"})
		return
	}

	// Check permission
	if err := h.rbac.RequirePermission(c.Request.Context(), user.ID, user.CurrentOrgID, auth.PermScheduleUpdate); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "permission denied"})
		return
	}

	job, err := h.store.GetJobByID(c.Request.Context(), jobID)
	if err != nil {
		h.logger.Error().Err(err).Str("job_id", jobID.String()).Msg("failed to get job")
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Verify job belongs to user's organization
	if job.OrgID != user.CurrentOrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	// Can only retry failed or dead letter jobs
	if !job.CanRetry() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "can only retry failed or dead letter jobs"})
		return
	}

	if !job.Retry() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to retry job"})
		return
	}

	if err := h.store.UpdateJob(c.Request.Context(), job); err != nil {
		h.logger.Error().Err(err).Str("job_id", jobID.String()).Msg("failed to update job")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retry job"})
		return
	}

	h.logger.Info().
		Str("job_id", jobID.String()).
		Str("user_id", user.ID.String()).
		Msg("job queued for retry")

	c.JSON(http.StatusOK, gin.H{"message": "job queued for retry"})
}
