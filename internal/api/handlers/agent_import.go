package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"text/template"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	agentimport "github.com/MacJediWizard/keldris/internal/import"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// AgentImportStore defines the interface for agent import persistence operations.
type AgentImportStore interface {
	GetAgentsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Agent, error)
	GetAgentGroupsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentGroup, error)
	CreateAgentGroup(ctx context.Context, group *models.AgentGroup) error
	CreateAgent(ctx context.Context, agent *models.Agent) error
	AddAgentToGroup(ctx context.Context, groupID, agentID uuid.UUID) error
	CreateRegistrationCode(ctx context.Context, code *models.RegistrationCode) error
	GetRegistrationCodeByCode(ctx context.Context, orgID uuid.UUID, code string) (*models.RegistrationCode, error)
	CreateAuditLog(ctx context.Context, log *models.AuditLog) error
}

// AgentImportHandler handles agent import endpoints.
type AgentImportHandler struct {
	store     AgentImportStore
	serverURL string
	logger    zerolog.Logger
}

// NewAgentImportHandler creates a new AgentImportHandler.
func NewAgentImportHandler(store AgentImportStore, serverURL string, logger zerolog.Logger) *AgentImportHandler {
	return &AgentImportHandler{
		store:     store,
		serverURL: serverURL,
		logger:    logger.With().Str("component", "agent_import_handler").Logger(),
	}
}

// RegisterRoutes registers agent import routes on the given router group.
func (h *AgentImportHandler) RegisterRoutes(r *gin.RouterGroup) {
	importRoutes := r.Group("/agents/import")
	{
		importRoutes.POST("/preview", h.Preview)
		importRoutes.POST("", h.Import)
		importRoutes.GET("/template", h.Template)
		importRoutes.POST("/script", h.GenerateScript)
		importRoutes.GET("/tokens/export", h.ExportTokens)
	}
}

// Preview parses and validates a CSV file without creating agents.
// POST /api/v1/agents/import/preview
func (h *AgentImportHandler) Preview(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Get the uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// Parse form values
	hasHeader := c.PostForm("has_header") != "false"
	hostnameCol := parseIntFormValue(c.PostForm("hostname_col"), 0)
	groupCol := parseIntFormValue(c.PostForm("group_col"), 1)
	tagsCol := parseIntFormValue(c.PostForm("tags_col"), 2)
	configCol := parseIntFormValue(c.PostForm("config_col"), 3)

	// Create parser options
	options := agentimport.ParseOptions{
		HasHeader: hasHeader,
		ColumnMapping: agentimport.ColumnMapping{
			Hostname: hostnameCol,
			Group:    groupCol,
			Tags:     tagsCol,
			Config:   configCol,
		},
		Delimiter: ',',
	}

	// Parse CSV
	parser := agentimport.NewParser(options)
	entries, err := parser.Parse(file)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to parse CSV")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse CSV: " + err.Error()})
		return
	}

	// Get existing hostnames for validation
	existingAgents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get existing agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate agents"})
		return
	}

	existingHostnames := make([]string, len(existingAgents))
	for i, agent := range existingAgents {
		existingHostnames[i] = agent.Hostname
	}

	// Validate entries
	validator := agentimport.NewValidator(existingHostnames)
	validatedEntries := validator.Validate(entries)

	// Build preview response
	response := h.buildPreviewResponse(validatedEntries)

	h.logger.Info().
		Int("total_rows", response.TotalRows).
		Int("valid_rows", response.ValidRows).
		Int("invalid_rows", response.InvalidRows).
		Msg("agent import preview completed")

	c.JSON(http.StatusOK, response)
}

// buildPreviewResponse builds a preview response from validated entries.
func (h *AgentImportHandler) buildPreviewResponse(entries []agentimport.AgentImportEntry) models.AgentImportPreviewResponse {
	response := models.AgentImportPreviewResponse{
		TotalRows: len(entries),
		Entries:   make([]models.AgentImportPreviewEntry, len(entries)),
	}

	groupsSet := make(map[string]bool)
	tagsSet := make(map[string]bool)

	for i, entry := range entries {
		response.Entries[i] = models.AgentImportPreviewEntry{
			RowNumber: entry.RowNumber,
			Hostname:  entry.Hostname,
			GroupName: entry.GroupName,
			Tags:      entry.Tags,
			Config:    entry.Config,
			IsValid:   entry.IsValid,
			Errors:    entry.Errors,
		}

		if entry.IsValid {
			response.ValidRows++
		} else {
			response.InvalidRows++
		}

		if entry.GroupName != "" {
			groupsSet[entry.GroupName] = true
		}
		for _, tag := range entry.Tags {
			tagsSet[tag] = true
		}
	}

	// Convert sets to sorted slices
	response.DetectedGroups = mapKeysToSortedSlice(groupsSet)
	response.DetectedTags = mapKeysToSortedSlice(tagsSet)

	return response
}

// Import processes a CSV file and creates agents with registration tokens.
// POST /api/v1/agents/import
func (h *AgentImportHandler) Import(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Get the uploaded file
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	defer file.Close()

	// Parse form values
	hasHeader := c.PostForm("has_header") != "false"
	hostnameCol := parseIntFormValue(c.PostForm("hostname_col"), 0)
	groupCol := parseIntFormValue(c.PostForm("group_col"), 1)
	tagsCol := parseIntFormValue(c.PostForm("tags_col"), 2)
	configCol := parseIntFormValue(c.PostForm("config_col"), 3)
	createMissingGroups := c.PostForm("create_missing_groups") == "true"
	tokenExpiryHours := parseIntFormValue(c.PostForm("token_expiry_hours"), 24)

	if tokenExpiryHours <= 0 {
		tokenExpiryHours = 24
	}
	if tokenExpiryHours > 168 { // Max 7 days
		tokenExpiryHours = 168
	}

	// Create parser options
	options := agentimport.ParseOptions{
		HasHeader: hasHeader,
		ColumnMapping: agentimport.ColumnMapping{
			Hostname: hostnameCol,
			Group:    groupCol,
			Tags:     tagsCol,
			Config:   configCol,
		},
		Delimiter: ',',
	}

	// Parse CSV
	parser := agentimport.NewParser(options)
	entries, err := parser.Parse(file)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to parse CSV")
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse CSV: " + err.Error()})
		return
	}

	// Get existing hostnames for validation
	existingAgents, err := h.store.GetAgentsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get existing agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to validate agents"})
		return
	}

	existingHostnames := make([]string, len(existingAgents))
	for i, agent := range existingAgents {
		existingHostnames[i] = agent.Hostname
	}

	// Validate entries
	validator := agentimport.NewValidator(existingHostnames)
	validatedEntries := validator.Validate(entries)

	// Get existing groups
	existingGroups, err := h.store.GetAgentGroupsByOrgID(c.Request.Context(), user.CurrentOrgID)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get existing groups")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get agent groups"})
		return
	}

	groupsByName := make(map[string]*models.AgentGroup)
	for _, g := range existingGroups {
		groupsByName[strings.ToLower(g.Name)] = g
	}

	// Process valid entries
	jobID := uuid.New()
	results := make([]models.AgentImportJobResult, 0, len(validatedEntries))
	groupsCreated := []string{}
	importedCount := 0
	failedCount := 0
	expiresAt := time.Now().Add(time.Duration(tokenExpiryHours) * time.Hour)

	for _, entry := range validatedEntries {
		result := models.AgentImportJobResult{
			RowNumber: entry.RowNumber,
			Hostname:  entry.Hostname,
			GroupName: entry.GroupName,
		}

		if !entry.IsValid {
			result.Success = false
			if len(entry.Errors) > 0 {
				result.ErrorMessage = strings.Join(entry.Errors, "; ")
			}
			failedCount++
			results = append(results, result)
			continue
		}

		// Get or create group if specified
		var groupID *uuid.UUID
		if entry.GroupName != "" {
			group, exists := groupsByName[strings.ToLower(entry.GroupName)]
			if !exists {
				if createMissingGroups {
					// Create the group
					newGroup := models.NewAgentGroup(user.CurrentOrgID, entry.GroupName, "", "")
					if err := h.store.CreateAgentGroup(c.Request.Context(), newGroup); err != nil {
						h.logger.Error().Err(err).Str("group", entry.GroupName).Msg("failed to create group")
						result.Success = false
						result.ErrorMessage = "failed to create group: " + err.Error()
						failedCount++
						results = append(results, result)
						continue
					}
					groupsByName[strings.ToLower(entry.GroupName)] = newGroup
					groupsCreated = append(groupsCreated, entry.GroupName)
					group = newGroup
				} else {
					result.Success = false
					result.ErrorMessage = fmt.Sprintf("group '%s' does not exist", entry.GroupName)
					failedCount++
					results = append(results, result)
					continue
				}
			}
			groupID = &group.ID
			result.GroupID = groupID
		}

		// Generate registration code
		code, err := agentimport.GenerateRegistrationCode()
		if err != nil {
			h.logger.Error().Err(err).Str("hostname", entry.Hostname).Msg("failed to generate registration code")
			result.Success = false
			result.ErrorMessage = "failed to generate registration code"
			failedCount++
			results = append(results, result)
			continue
		}

		// Create registration code record
		var hostname *string
		if entry.Hostname != "" {
			hostname = &entry.Hostname
		}

		regCode := models.NewRegistrationCode(user.CurrentOrgID, user.ID, code, hostname, expiresAt)
		if err := h.store.CreateRegistrationCode(c.Request.Context(), regCode); err != nil {
			h.logger.Error().Err(err).Str("hostname", entry.Hostname).Msg("failed to create registration code")
			result.Success = false
			result.ErrorMessage = "failed to create registration code"
			failedCount++
			results = append(results, result)
			continue
		}

		result.RegistrationCode = code
		result.ExpiresAt = &expiresAt
		result.Success = true
		importedCount++
		results = append(results, result)
	}

	// Log the audit event
	h.logAuditEvent(c, user.CurrentOrgID, user.ID, models.AuditActionCreate, "agent_import",
		&jobID, models.AuditResultSuccess, fmt.Sprintf("imported %d agents (%d failed)", importedCount, failedCount))

	h.logger.Info().
		Str("job_id", jobID.String()).
		Int("imported_count", importedCount).
		Int("failed_count", failedCount).
		Int("groups_created", len(groupsCreated)).
		Msg("agent import completed")

	c.JSON(http.StatusCreated, models.AgentImportResponse{
		JobID:         jobID,
		TotalAgents:   len(validatedEntries),
		ImportedCount: importedCount,
		FailedCount:   failedCount,
		Results:       results,
		GroupsCreated: groupsCreated,
	})
}

// Template returns the CSV template for agent imports.
// GET /api/v1/agents/import/template
func (h *AgentImportHandler) Template(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	format := c.Query("format")
	if format == "csv" {
		// Return as downloadable CSV file
		var buf bytes.Buffer
		writer := csv.NewWriter(&buf)

		// Write header
		if err := writer.Write(agentimport.CSVTemplateHeader()); err != nil {
			h.logger.Error().Err(err).Msg("failed to write CSV header")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate template"})
			return
		}

		// Write example rows
		for _, row := range agentimport.CSVTemplateExample() {
			if err := writer.Write(row); err != nil {
				h.logger.Error().Err(err).Msg("failed to write CSV row")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate template"})
				return
			}
		}

		writer.Flush()
		if err := writer.Error(); err != nil {
			h.logger.Error().Err(err).Msg("failed to flush CSV writer")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate template"})
			return
		}

		c.Header("Content-Disposition", "attachment; filename=agent_import_template.csv")
		c.Header("Content-Type", "text/csv")
		c.Data(http.StatusOK, "text/csv", buf.Bytes())
		return
	}

	// Return as JSON
	c.JSON(http.StatusOK, models.AgentImportTemplateResponse{
		Headers:  agentimport.CSVTemplateHeader(),
		Examples: agentimport.CSVTemplateExample(),
	})
}

// GenerateScript generates a registration script for a specific agent.
// POST /api/v1/agents/import/script
func (h *AgentImportHandler) GenerateScript(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	var req models.AgentRegistrationScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify the registration code exists and is valid
	regCode, err := h.store.GetRegistrationCodeByCode(c.Request.Context(), user.CurrentOrgID, req.RegistrationCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "registration code not found"})
		return
	}

	if !regCode.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "registration code is expired or already used"})
		return
	}

	// Generate script from template
	tmpl, err := template.New("script").Parse(agentimport.RegistrationScriptTemplate)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to parse script template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate script"})
		return
	}

	data := agentimport.ScriptTemplateData{
		Hostname:         req.Hostname,
		RegistrationCode: req.RegistrationCode,
		ServerURL:        h.serverURL,
		OrgID:            user.CurrentOrgID.String(),
		ExpiresAt:        regCode.ExpiresAt.Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		h.logger.Error().Err(err).Msg("failed to execute script template")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate script"})
		return
	}

	c.JSON(http.StatusOK, models.AgentRegistrationScriptResponse{
		Script:           buf.String(),
		Hostname:         req.Hostname,
		RegistrationCode: req.RegistrationCode,
		ExpiresAt:        regCode.ExpiresAt,
	})
}

// ExportTokens exports registration tokens as a CSV file.
// GET /api/v1/agents/import/tokens/export
func (h *AgentImportHandler) ExportTokens(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	if user.CurrentOrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no organization selected"})
		return
	}

	// Parse the results from query parameter (JSON-encoded)
	resultsJSON := c.Query("results")
	if resultsJSON == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "results parameter is required"})
		return
	}

	var results []models.AgentImportJobResult
	if err := json.Unmarshal([]byte(resultsJSON), &results); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid results format"})
		return
	}

	// Build CSV
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	header := []string{"hostname", "group", "registration_code", "expires_at", "registration_url"}
	if err := writer.Write(header); err != nil {
		h.logger.Error().Err(err).Msg("failed to write CSV header")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export tokens"})
		return
	}

	// Write token rows
	for _, result := range results {
		if !result.Success || result.RegistrationCode == "" {
			continue
		}

		expiresAt := ""
		if result.ExpiresAt != nil {
			expiresAt = result.ExpiresAt.Format(time.RFC3339)
		}

		registrationURL := fmt.Sprintf("%s/register?code=%s&org=%s", h.serverURL, result.RegistrationCode, user.CurrentOrgID.String())

		row := []string{
			result.Hostname,
			result.GroupName,
			result.RegistrationCode,
			expiresAt,
			registrationURL,
		}

		if err := writer.Write(row); err != nil {
			h.logger.Error().Err(err).Msg("failed to write CSV row")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export tokens"})
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		h.logger.Error().Err(err).Msg("failed to flush CSV writer")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to export tokens"})
		return
	}

	c.Header("Content-Disposition", "attachment; filename=agent_registration_tokens.csv")
	c.Header("Content-Type", "text/csv")
	c.Data(http.StatusOK, "text/csv", buf.Bytes())
}

// logAuditEvent logs an audit event for agent import actions.
func (h *AgentImportHandler) logAuditEvent(c *gin.Context, orgID, userID uuid.UUID, action models.AuditAction, resourceType string, resourceID *uuid.UUID, result models.AuditResult, details string) {
	auditLog := models.NewAuditLog(orgID, action, resourceType, result).
		WithRequestInfo(c.ClientIP(), c.Request.UserAgent()).
		WithDetails(details)

	if userID != uuid.Nil {
		auditLog.WithUser(userID)
	}

	if resourceID != nil {
		auditLog.WithResource(*resourceID)
	}

	go func() {
		if err := h.store.CreateAuditLog(context.Background(), auditLog); err != nil {
			h.logger.Error().Err(err).
				Str("action", string(action)).
				Str("resource_type", resourceType).
				Msg("failed to create audit log")
		}
	}()
}

// parseIntFormValue parses an int from a form value with a default.
func parseIntFormValue(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var result int
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		result = result*10 + int(ch-'0')
	}
	return result
}

// mapKeysToSortedSlice converts map keys to a sorted slice.
func mapKeysToSortedSlice(m map[string]bool) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}
