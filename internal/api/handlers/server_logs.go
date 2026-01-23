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
	"github.com/MacJediWizard/keldris/internal/logs"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ServerLogStore defines the interface for fetching user and membership data.
type ServerLogStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error)
}

// ServerLogsHandler handles server log HTTP endpoints.
type ServerLogsHandler struct {
	store     ServerLogStore
	logBuffer *logs.LogBuffer
	logger    zerolog.Logger
}

// NewServerLogsHandler creates a new ServerLogsHandler.
func NewServerLogsHandler(store ServerLogStore, logBuffer *logs.LogBuffer, logger zerolog.Logger) *ServerLogsHandler {
	return &ServerLogsHandler{
		store:     store,
		logBuffer: logBuffer,
		logger:    logger.With().Str("component", "server_logs_handler").Logger(),
	}
}

// RegisterRoutes registers server log routes on the given router group.
func (h *ServerLogsHandler) RegisterRoutes(r *gin.RouterGroup) {
	adminLogs := r.Group("/admin/logs")
	{
		adminLogs.GET("", h.List)
		adminLogs.GET("/components", h.ListComponents)
		adminLogs.GET("/export/csv", h.ExportCSV)
		adminLogs.GET("/export/json", h.ExportJSON)
		adminLogs.DELETE("", h.Clear)
	}
}

// ServerLogListResponse is the response for listing server logs.
type ServerLogListResponse struct {
	Logs       []logs.LogEntry `json:"logs"`
	TotalCount int             `json:"total_count"`
	Limit      int             `json:"limit"`
	Offset     int             `json:"offset"`
}

// ComponentsResponse is the response for listing log components.
type ComponentsResponse struct {
	Components []string `json:"components"`
}

// isAdmin checks if the current user is an admin or owner.
func (h *ServerLogsHandler) isAdmin(c *gin.Context) bool {
	user := middleware.GetUser(c)
	if user == nil {
		return false
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		return false
	}

	membership, err := h.store.GetMembershipByUserAndOrg(c.Request.Context(), user.ID, dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get membership")
		return false
	}

	return membership.Role == models.OrgRoleOwner || membership.Role == models.OrgRoleAdmin
}

// requireAdmin ensures the user is an admin and aborts if not.
func (h *ServerLogsHandler) requireAdmin(c *gin.Context) bool {
	if !h.isAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin access required"})
		return false
	}
	return true
}

// List returns server logs with optional filtering.
// GET /api/v1/admin/logs
// Query params: level, component, search, start_time, end_time, limit, offset
func (h *ServerLogsHandler) List(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	filter := h.parseFilterParams(c)
	entries, totalCount := h.logBuffer.Get(filter)

	c.JSON(http.StatusOK, ServerLogListResponse{
		Logs:       entries,
		TotalCount: totalCount,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	})
}

// ListComponents returns unique component names in the log buffer.
// GET /api/v1/admin/logs/components
func (h *ServerLogsHandler) ListComponents(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	components := h.logBuffer.GetComponents()
	c.JSON(http.StatusOK, ComponentsResponse{Components: components})
}

// ExportCSV exports server logs as CSV.
// GET /api/v1/admin/logs/export/csv
func (h *ServerLogsHandler) ExportCSV(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	filter := h.parseFilterParams(c)
	filter.Limit = 0
	filter.Offset = 0

	entries, _ := h.logBuffer.Get(filter)

	filename := fmt.Sprintf("server_logs_%s.csv", time.Now().Format("2006-01-02_15-04-05"))
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	if err := writer.Write([]string{
		"Timestamp", "Level", "Component", "Message", "Fields",
	}); err != nil {
		h.logger.Error().Err(err).Msg("failed to write CSV header")
		return
	}

	for _, entry := range entries {
		fieldsJSON, _ := json.Marshal(entry.Fields)
		row := []string{
			entry.Timestamp.Format(time.RFC3339),
			string(entry.Level),
			entry.Component,
			entry.Message,
			string(fieldsJSON),
		}
		if err := writer.Write(row); err != nil {
			h.logger.Error().Err(err).Msg("failed to write CSV row")
			return
		}
	}

	h.logger.Info().
		Int("count", len(entries)).
		Msg("server logs exported to CSV")
}

// ExportJSON exports server logs as JSON.
// GET /api/v1/admin/logs/export/json
func (h *ServerLogsHandler) ExportJSON(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	filter := h.parseFilterParams(c)
	filter.Limit = 0
	filter.Offset = 0

	entries, _ := h.logBuffer.Get(filter)

	filename := fmt.Sprintf("server_logs_%s.json", time.Now().Format("2006-01-02_15-04-05"))
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	encoder := json.NewEncoder(c.Writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(gin.H{"logs": entries}); err != nil {
		h.logger.Error().Err(err).Msg("failed to write JSON")
		return
	}

	h.logger.Info().
		Int("count", len(entries)).
		Msg("server logs exported to JSON")
}

// Clear removes all entries from the log buffer.
// DELETE /api/v1/admin/logs
func (h *ServerLogsHandler) Clear(c *gin.Context) {
	if !h.requireAdmin(c) {
		return
	}

	h.logBuffer.Clear()
	h.logger.Info().Msg("server logs cleared by admin")

	c.JSON(http.StatusOK, gin.H{"message": "logs cleared successfully"})
}

// parseFilterParams extracts filter parameters from the query string.
func (h *ServerLogsHandler) parseFilterParams(c *gin.Context) logs.LogFilter {
	filter := logs.LogFilter{
		Level:     logs.LogLevel(c.Query("level")),
		Component: c.Query("component"),
		Search:    c.Query("search"),
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			filter.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			filter.EndTime = t
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if filter.Limit == 0 {
		filter.Limit = 100
	}

	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	return filter
}
