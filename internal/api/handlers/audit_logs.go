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
	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/license"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AuditLogStore defines the interface for audit log persistence operations.
type AuditLogStore interface {
	GetAuditLogsByOrgID(ctx context.Context, orgID uuid.UUID, filter db.AuditLogFilter) ([]*models.AuditLog, error)
	GetAuditLogByID(ctx context.Context, id uuid.UUID) (*models.AuditLog, error)
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
	CountAuditLogsByOrgID(ctx context.Context, orgID uuid.UUID, filter db.AuditLogFilter) (int64, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// AuditLogsHandler handles audit log HTTP endpoints.
type AuditLogsHandler struct {
	store   AuditLogStore
	checker *license.FeatureChecker
	logger  zerolog.Logger
}

// NewAuditLogsHandler creates a new AuditLogsHandler.
func NewAuditLogsHandler(store AuditLogStore, checker *license.FeatureChecker, logger zerolog.Logger) *AuditLogsHandler {
	return &AuditLogsHandler{
		store:   store,
		checker: checker,
		logger:  logger.With().Str("component", "audit_logs_handler").Logger(),
	}
}

// RegisterRoutes registers audit log routes on the given router group.
func (h *AuditLogsHandler) RegisterRoutes(r *gin.RouterGroup) {
	auditLogs := r.Group("/audit-logs")
	{
		auditLogs.GET("", h.List)
		auditLogs.GET("/:id", h.Get)
		auditLogs.GET("/export/csv", h.ExportCSV)
		auditLogs.GET("/export/json", h.ExportJSON)
	}
}

// AuditLogListResponse is the response for listing audit logs.
type AuditLogListResponse struct {
	AuditLogs  []*models.AuditLog `json:"audit_logs"`
	TotalCount int64              `json:"total_count"`
	Limit      int                `json:"limit"`
	Offset     int                `json:"offset"`
}

// List returns all audit logs for the authenticated user's organization.
// GET /api/v1/audit-logs
// Query params: action, resource_type, result, start_date, end_date, search, limit, offset
func (h *AuditLogsHandler) List(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureAuditLogs) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Get user's org ID
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Parse filter params
	filter := h.parseFilterParams(c)

	// Get audit logs
	logs, err := h.store.GetAuditLogsByOrgID(c.Request.Context(), dbUser.OrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list audit logs"})
		return
	}

	// Get total count for pagination
	totalCount, err := h.store.CountAuditLogsByOrgID(c.Request.Context(), dbUser.OrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to count audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count audit logs"})
		return
	}

	c.JSON(http.StatusOK, AuditLogListResponse{
		AuditLogs:  logs,
		TotalCount: totalCount,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	})
}

// Get returns a specific audit log by ID.
// GET /api/v1/audit-logs/:id
func (h *AuditLogsHandler) Get(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid audit log ID"})
		return
	}

	log, err := h.store.GetAuditLogByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error().Err(err).Str("audit_log_id", id.String()).Msg("failed to get audit log")
		c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
		return
	}

	// Verify user has access to this audit log's org
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if log.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "audit log not found"})
		return
	}

	c.JSON(http.StatusOK, log)
}

// ExportCSV exports audit logs as CSV.
// GET /api/v1/audit-logs/export/csv
func (h *AuditLogsHandler) ExportCSV(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureAuditLogs) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Get user's org ID
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Parse filter params (no pagination for export)
	filter := h.parseFilterParams(c)
	filter.Limit = 0
	filter.Offset = 0

	logs, err := h.store.GetAuditLogsByOrgID(c.Request.Context(), dbUser.OrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to export audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export audit logs"})
		return
	}

	// Set headers for CSV download
	filename := fmt.Sprintf("audit_logs_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{
		"ID", "Org ID", "User ID", "Agent ID", "Action", "Resource Type",
		"Resource ID", "Result", "IP Address", "User Agent", "Details", "Created At",
	}); err != nil {
		h.logger.Error().Err(err).Msg("failed to write CSV header")
		return
	}

	// Write rows
	for _, log := range logs {
		row := []string{
			log.ID.String(),
			log.OrgID.String(),
			uuidPtrToString(log.UserID),
			uuidPtrToString(log.AgentID),
			string(log.Action),
			log.ResourceType,
			uuidPtrToString(log.ResourceID),
			string(log.Result),
			log.IPAddress,
			log.UserAgent,
			log.Details,
			log.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			h.logger.Error().Err(err).Msg("failed to write CSV row")
			return
		}
	}

	h.logger.Info().
		Str("org_id", dbUser.OrgID.String()).
		Int("count", len(logs)).
		Msg("audit logs exported to CSV")
}

// ExportJSON exports audit logs as JSON.
// GET /api/v1/audit-logs/export/json
func (h *AuditLogsHandler) ExportJSON(c *gin.Context) {
	if !middleware.RequireFeature(c, h.checker, license.FeatureAuditLogs) {
		return
	}

	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	// Get user's org ID
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Parse filter params (no pagination for export)
	filter := h.parseFilterParams(c)
	filter.Limit = 0
	filter.Offset = 0

	logs, err := h.store.GetAuditLogsByOrgID(c.Request.Context(), dbUser.OrgID, filter)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to export audit logs")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export audit logs"})
		return
	}

	// Set headers for JSON download
	filename := fmt.Sprintf("audit_logs_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	encoder := json.NewEncoder(c.Writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(gin.H{"audit_logs": logs}); err != nil {
		h.logger.Error().Err(err).Msg("failed to write JSON")
		return
	}

	h.logger.Info().
		Str("org_id", dbUser.OrgID.String()).
		Int("count", len(logs)).
		Msg("audit logs exported to JSON")
}

// parseFilterParams extracts filter parameters from the query string.
func (h *AuditLogsHandler) parseFilterParams(c *gin.Context) db.AuditLogFilter {
	filter := db.AuditLogFilter{
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		Result:       c.Query("result"),
		Search:       c.Query("search"),
	}

	// Parse date filters
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			filter.StartDate = &t
		} else if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = &t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			filter.EndDate = &t
		} else if t, err := time.Parse("2006-01-02", endDate); err == nil {
			// Set to end of day
			endOfDay := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &endOfDay
		}
	}

	// Parse pagination
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 50 // Default limit
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	return filter
}

// uuidPtrToString converts a UUID pointer to a string, returning empty string if nil.
func uuidPtrToString(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}
