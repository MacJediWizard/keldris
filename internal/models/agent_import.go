package models

import (
	"time"

	"github.com/google/uuid"
)

// AgentImportJob represents a bulk agent import job.
type AgentImportJob struct {
	ID             uuid.UUID               `json:"id"`
	OrgID          uuid.UUID               `json:"org_id"`
	CreatedBy      uuid.UUID               `json:"created_by"`
	Status         AgentImportStatus       `json:"status"`
	TotalAgents    int                     `json:"total_agents"`
	ImportedCount  int                     `json:"imported_count"`
	FailedCount    int                     `json:"failed_count"`
	Results        []AgentImportJobResult  `json:"results,omitempty"`
	ErrorMessage   string                  `json:"error_message,omitempty"`
	CreatedAt      time.Time               `json:"created_at"`
	CompletedAt    *time.Time              `json:"completed_at,omitempty"`
}

// AgentImportStatus represents the status of an import job.
type AgentImportStatus string

const (
	// AgentImportStatusPending indicates the import job is pending.
	AgentImportStatusPending AgentImportStatus = "pending"
	// AgentImportStatusProcessing indicates the import job is being processed.
	AgentImportStatusProcessing AgentImportStatus = "processing"
	// AgentImportStatusCompleted indicates the import job completed successfully.
	AgentImportStatusCompleted AgentImportStatus = "completed"
	// AgentImportStatusFailed indicates the import job failed.
	AgentImportStatusFailed AgentImportStatus = "failed"
	// AgentImportStatusPartial indicates the import job completed with some failures.
	AgentImportStatusPartial AgentImportStatus = "partial"
)

// AgentImportJobResult represents the result for a single agent in an import job.
type AgentImportJobResult struct {
	RowNumber        int        `json:"row_number"`
	Hostname         string     `json:"hostname"`
	AgentID          *uuid.UUID `json:"agent_id,omitempty"`
	GroupID          *uuid.UUID `json:"group_id,omitempty"`
	GroupName        string     `json:"group_name,omitempty"`
	RegistrationCode string     `json:"registration_code,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	Success          bool       `json:"success"`
	ErrorMessage     string     `json:"error_message,omitempty"`
}

// NewAgentImportJob creates a new AgentImportJob.
func NewAgentImportJob(orgID, createdBy uuid.UUID, totalAgents int) *AgentImportJob {
	return &AgentImportJob{
		ID:          uuid.New(),
		OrgID:       orgID,
		CreatedBy:   createdBy,
		Status:      AgentImportStatusPending,
		TotalAgents: totalAgents,
		Results:     []AgentImportJobResult{},
		CreatedAt:   time.Now(),
	}
}

// AgentImportToken represents a registration token for bulk agent imports.
type AgentImportToken struct {
	ID            uuid.UUID  `json:"id"`
	OrgID         uuid.UUID  `json:"org_id"`
	ImportJobID   uuid.UUID  `json:"import_job_id"`
	Hostname      string     `json:"hostname"`
	GroupID       *uuid.UUID `json:"group_id,omitempty"`
	TokenHash     string     `json:"-"` // Never expose in JSON
	Code          string     `json:"code"`
	Tags          []string   `json:"tags,omitempty"`
	Config        string     `json:"config,omitempty"` // JSON-encoded config
	ExpiresAt     time.Time  `json:"expires_at"`
	UsedAt        *time.Time `json:"used_at,omitempty"`
	UsedByAgentID *uuid.UUID `json:"used_by_agent_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

// NewAgentImportToken creates a new AgentImportToken.
func NewAgentImportToken(orgID, importJobID uuid.UUID, hostname, tokenHash, code string, groupID *uuid.UUID, tags []string, config string, expiresAt time.Time) *AgentImportToken {
	return &AgentImportToken{
		ID:          uuid.New(),
		OrgID:       orgID,
		ImportJobID: importJobID,
		Hostname:    hostname,
		GroupID:     groupID,
		TokenHash:   tokenHash,
		Code:        code,
		Tags:        tags,
		Config:      config,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
	}
}

// IsExpired returns true if the token has expired.
func (t *AgentImportToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsed returns true if the token has been used.
func (t *AgentImportToken) IsUsed() bool {
	return t.UsedAt != nil
}

// IsValid returns true if the token is neither expired nor used.
func (t *AgentImportToken) IsValid() bool {
	return !t.IsExpired() && !t.IsUsed()
}

// MarkUsed marks the token as used by the given agent.
func (t *AgentImportToken) MarkUsed(agentID uuid.UUID) {
	now := time.Now()
	t.UsedAt = &now
	t.UsedByAgentID = &agentID
}

// AgentImportPreviewRequest is the request for previewing a CSV import.
type AgentImportPreviewRequest struct {
	HasHeader     bool `json:"has_header"`
	HostnameCol   int  `json:"hostname_col"`
	GroupCol      int  `json:"group_col"`
	TagsCol       int  `json:"tags_col"`
	ConfigCol     int  `json:"config_col"`
}

// AgentImportPreviewEntry represents a single entry in the import preview.
type AgentImportPreviewEntry struct {
	RowNumber  int               `json:"row_number"`
	Hostname   string            `json:"hostname"`
	GroupName  string            `json:"group_name,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Config     map[string]string `json:"config,omitempty"`
	IsValid    bool              `json:"is_valid"`
	Errors     []string          `json:"errors,omitempty"`
}

// AgentImportPreviewResponse is the response for a CSV import preview.
type AgentImportPreviewResponse struct {
	TotalRows      int                       `json:"total_rows"`
	ValidRows      int                       `json:"valid_rows"`
	InvalidRows    int                       `json:"invalid_rows"`
	Entries        []AgentImportPreviewEntry `json:"entries"`
	DetectedGroups []string                  `json:"detected_groups"`
	DetectedTags   []string                  `json:"detected_tags"`
}

// AgentImportRequest is the request for importing agents from CSV.
type AgentImportRequest struct {
	HasHeader          bool   `json:"has_header"`
	HostnameCol        int    `json:"hostname_col"`
	GroupCol           int    `json:"group_col"`
	TagsCol            int    `json:"tags_col"`
	ConfigCol          int    `json:"config_col"`
	CreateMissingGroups bool  `json:"create_missing_groups"`
	TokenExpiryHours   int    `json:"token_expiry_hours"` // Default 24 hours
}

// AgentImportResponse is the response for an agent import operation.
type AgentImportResponse struct {
	JobID          uuid.UUID                `json:"job_id"`
	TotalAgents    int                      `json:"total_agents"`
	ImportedCount  int                      `json:"imported_count"`
	FailedCount    int                      `json:"failed_count"`
	Results        []AgentImportJobResult   `json:"results"`
	GroupsCreated  []string                 `json:"groups_created,omitempty"`
}

// AgentImportTokensExportRequest is the request for exporting registration tokens.
type AgentImportTokensExportRequest struct {
	JobID uuid.UUID `json:"job_id" binding:"required"`
}

// AgentImportTokenExportEntry represents a token entry for CSV export.
type AgentImportTokenExportEntry struct {
	Hostname         string    `json:"hostname"`
	GroupName        string    `json:"group_name,omitempty"`
	RegistrationCode string    `json:"registration_code"`
	ExpiresAt        time.Time `json:"expires_at"`
	RegistrationURL  string    `json:"registration_url"`
}

// AgentImportTokensExportResponse is the response for exporting registration tokens.
type AgentImportTokensExportResponse struct {
	Tokens []AgentImportTokenExportEntry `json:"tokens"`
}

// AgentImportTemplateResponse is the response for downloading the CSV template.
type AgentImportTemplateResponse struct {
	Headers  []string   `json:"headers"`
	Examples [][]string `json:"examples"`
}

// AgentRegistrationScriptRequest is the request for generating a registration script.
type AgentRegistrationScriptRequest struct {
	Hostname         string `json:"hostname" binding:"required"`
	RegistrationCode string `json:"registration_code" binding:"required"`
}

// AgentRegistrationScriptResponse is the response containing the registration script.
type AgentRegistrationScriptResponse struct {
	Script           string    `json:"script"`
	Hostname         string    `json:"hostname"`
	RegistrationCode string    `json:"registration_code"`
	ExpiresAt        time.Time `json:"expires_at"`
}
