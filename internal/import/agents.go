// Package agentimport provides CSV import functionality for bulk agent registration.
package agentimport

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Column indices for CSV parsing
const (
	DefaultHostnameCol = 0
	DefaultGroupCol    = 1
	DefaultTagsCol     = 2
	DefaultConfigCol   = 3
)

// AgentImportEntry represents a single agent entry from the CSV import.
type AgentImportEntry struct {
	Hostname   string            `json:"hostname"`
	GroupName  string            `json:"group_name,omitempty"`
	Tags       []string          `json:"tags,omitempty"`
	Config     map[string]string `json:"config,omitempty"`
	RowNumber  int               `json:"row_number"`
	IsValid    bool              `json:"is_valid"`
	Errors     []string          `json:"errors,omitempty"`
}

// AgentImportResult represents the result of importing a single agent.
type AgentImportResult struct {
	AgentID           uuid.UUID `json:"agent_id"`
	Hostname          string    `json:"hostname"`
	GroupName         string    `json:"group_name,omitempty"`
	RegistrationToken string    `json:"registration_token"`
	RegistrationCode  string    `json:"registration_code"`
	ExpiresAt         time.Time `json:"expires_at"`
	RowNumber         int       `json:"row_number"`
}

// ColumnMapping defines the mapping of CSV columns to agent fields.
type ColumnMapping struct {
	Hostname int `json:"hostname"`
	Group    int `json:"group"`
	Tags     int `json:"tags"`
	Config   int `json:"config"`
}

// DefaultColumnMapping returns the default column mapping.
func DefaultColumnMapping() ColumnMapping {
	return ColumnMapping{
		Hostname: DefaultHostnameCol,
		Group:    DefaultGroupCol,
		Tags:     DefaultTagsCol,
		Config:   DefaultConfigCol,
	}
}

// ParseOptions configures the CSV parsing behavior.
type ParseOptions struct {
	HasHeader     bool          `json:"has_header"`
	ColumnMapping ColumnMapping `json:"column_mapping"`
	Delimiter     rune          `json:"delimiter"`
}

// DefaultParseOptions returns default CSV parsing options.
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		HasHeader:     true,
		ColumnMapping: DefaultColumnMapping(),
		Delimiter:     ',',
	}
}

// Parser handles CSV parsing for agent imports.
type Parser struct {
	options ParseOptions
}

// NewParser creates a new CSV parser with the given options.
func NewParser(options ParseOptions) *Parser {
	return &Parser{options: options}
}

// Parse reads and parses a CSV file into agent import entries.
func (p *Parser) Parse(reader io.Reader) ([]AgentImportEntry, error) {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = p.options.Delimiter
	csvReader.FieldsPerRecord = -1 // Allow variable number of fields
	csvReader.TrimLeadingSpace = true

	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, errors.New("CSV file is empty")
	}

	startRow := 0
	if p.options.HasHeader {
		startRow = 1
	}

	if len(records) <= startRow {
		return nil, errors.New("CSV file has no data rows")
	}

	entries := make([]AgentImportEntry, 0, len(records)-startRow)
	for i := startRow; i < len(records); i++ {
		entry := p.parseRow(records[i], i+1) // 1-indexed row numbers
		entries = append(entries, entry)
	}

	return entries, nil
}

// parseRow parses a single CSV row into an AgentImportEntry.
func (p *Parser) parseRow(row []string, rowNumber int) AgentImportEntry {
	entry := AgentImportEntry{
		RowNumber: rowNumber,
		IsValid:   true,
		Errors:    []string{},
	}

	// Parse hostname (required)
	if p.options.ColumnMapping.Hostname < len(row) {
		entry.Hostname = strings.TrimSpace(row[p.options.ColumnMapping.Hostname])
	}

	// Parse group name (optional)
	if p.options.ColumnMapping.Group >= 0 && p.options.ColumnMapping.Group < len(row) {
		entry.GroupName = strings.TrimSpace(row[p.options.ColumnMapping.Group])
	}

	// Parse tags (optional, semicolon-separated)
	if p.options.ColumnMapping.Tags >= 0 && p.options.ColumnMapping.Tags < len(row) {
		tagsStr := strings.TrimSpace(row[p.options.ColumnMapping.Tags])
		if tagsStr != "" {
			tags := strings.Split(tagsStr, ";")
			for _, tag := range tags {
				trimmed := strings.TrimSpace(tag)
				if trimmed != "" {
					entry.Tags = append(entry.Tags, trimmed)
				}
			}
		}
	}

	// Parse config (optional, key=value pairs separated by semicolon)
	if p.options.ColumnMapping.Config >= 0 && p.options.ColumnMapping.Config < len(row) {
		configStr := strings.TrimSpace(row[p.options.ColumnMapping.Config])
		if configStr != "" {
			entry.Config = parseKeyValuePairs(configStr)
		}
	}

	return entry
}

// parseKeyValuePairs parses a semicolon-separated list of key=value pairs.
func parseKeyValuePairs(s string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(s, ";")
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				result[key] = value
			}
		}
	}
	return result
}

// Validator validates agent import entries.
type Validator struct {
	existingHostnames map[string]bool
	hostnameRegex     *regexp.Regexp
}

// NewValidator creates a new validator with a list of existing hostnames.
func NewValidator(existingHostnames []string) *Validator {
	hostnames := make(map[string]bool)
	for _, h := range existingHostnames {
		hostnames[strings.ToLower(h)] = true
	}
	return &Validator{
		existingHostnames: hostnames,
		hostnameRegex:     regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-_.]{0,253}[a-zA-Z0-9])?$`),
	}
}

// Validate validates all entries and returns the validated entries.
func (v *Validator) Validate(entries []AgentImportEntry) []AgentImportEntry {
	seenHostnames := make(map[string]int) // hostname -> first occurrence row

	for i := range entries {
		v.validateEntry(&entries[i], seenHostnames)
	}

	return entries
}

// validateEntry validates a single entry.
func (v *Validator) validateEntry(entry *AgentImportEntry, seenHostnames map[string]int) {
	// Validate hostname is present
	if entry.Hostname == "" {
		entry.IsValid = false
		entry.Errors = append(entry.Errors, "hostname is required")
		return
	}

	// Validate hostname format
	if len(entry.Hostname) > 255 {
		entry.IsValid = false
		entry.Errors = append(entry.Errors, "hostname exceeds 255 characters")
	}

	if !v.hostnameRegex.MatchString(entry.Hostname) {
		entry.IsValid = false
		entry.Errors = append(entry.Errors, "hostname contains invalid characters")
	}

	// Check for duplicates within the CSV
	lowercaseHostname := strings.ToLower(entry.Hostname)
	if firstRow, exists := seenHostnames[lowercaseHostname]; exists {
		entry.IsValid = false
		entry.Errors = append(entry.Errors, fmt.Sprintf("duplicate hostname (first occurrence in row %d)", firstRow))
	} else {
		seenHostnames[lowercaseHostname] = entry.RowNumber
	}

	// Check if hostname already exists in the system
	if v.existingHostnames[lowercaseHostname] {
		entry.IsValid = false
		entry.Errors = append(entry.Errors, "hostname already exists in the system")
	}

	// Validate group name if present
	if entry.GroupName != "" && len(entry.GroupName) > 100 {
		entry.IsValid = false
		entry.Errors = append(entry.Errors, "group name exceeds 100 characters")
	}

	// Validate tags
	for _, tag := range entry.Tags {
		if len(tag) > 50 {
			entry.IsValid = false
			entry.Errors = append(entry.Errors, fmt.Sprintf("tag '%s' exceeds 50 characters", tag))
		}
	}
}

// GenerateRegistrationToken generates a unique registration token for an agent.
func GenerateRegistrationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "kld_reg_" + hex.EncodeToString(bytes), nil
}

// GenerateRegistrationCode generates a short 6-character registration code.
func GenerateRegistrationCode() (string, error) {
	// Use alphanumeric characters that are easy to distinguish
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	code := make([]byte, 6)
	for i := range bytes {
		code[i] = chars[int(bytes[i])%len(chars)]
	}
	return string(code), nil
}

// HashToken creates a SHA-256 hash of a token for secure storage.
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// RegistrationScriptTemplate contains the template for generating agent registration scripts.
const RegistrationScriptTemplate = `#!/bin/bash
# Keldris Agent Registration Script
# Generated for: {{.Hostname}}
# Expires: {{.ExpiresAt}}

set -e

REGISTRATION_CODE="{{.RegistrationCode}}"
SERVER_URL="{{.ServerURL}}"
ORG_ID="{{.OrgID}}"

echo "Registering agent: {{.Hostname}}"

# Download and install the agent
curl -fsSL "${SERVER_URL}/install.sh" | bash

# Register the agent
keldris-agent register \
  --server "${SERVER_URL}" \
  --org-id "${ORG_ID}" \
  --code "${REGISTRATION_CODE}" \
  --hostname "{{.Hostname}}"

echo "Agent registration complete!"
`

// ScriptTemplateData contains data for generating registration scripts.
type ScriptTemplateData struct {
	Hostname         string
	RegistrationCode string
	ServerURL        string
	OrgID            string
	ExpiresAt        string
}

// ImportSummary provides a summary of the import operation.
type ImportSummary struct {
	TotalRows      int                  `json:"total_rows"`
	ValidRows      int                  `json:"valid_rows"`
	InvalidRows    int                  `json:"invalid_rows"`
	Entries        []AgentImportEntry   `json:"entries"`
	ImportedAgents []AgentImportResult  `json:"imported_agents,omitempty"`
	GroupsCreated  []string             `json:"groups_created,omitempty"`
	Errors         []ImportError        `json:"errors,omitempty"`
}

// ImportError represents an error during import.
type ImportError struct {
	RowNumber int    `json:"row_number"`
	Hostname  string `json:"hostname,omitempty"`
	Message   string `json:"message"`
}

// NewImportSummary creates a new ImportSummary from validated entries.
func NewImportSummary(entries []AgentImportEntry) ImportSummary {
	summary := ImportSummary{
		TotalRows: len(entries),
		Entries:   entries,
	}

	for _, entry := range entries {
		if entry.IsValid {
			summary.ValidRows++
		} else {
			summary.InvalidRows++
		}
	}

	return summary
}

// GetValidEntries returns only the valid entries from the summary.
func (s *ImportSummary) GetValidEntries() []AgentImportEntry {
	valid := make([]AgentImportEntry, 0, s.ValidRows)
	for _, entry := range s.Entries {
		if entry.IsValid {
			valid = append(valid, entry)
		}
	}
	return valid
}

// CSVTemplateHeader returns the header row for the CSV template.
func CSVTemplateHeader() []string {
	return []string{"hostname", "group", "tags", "config"}
}

// CSVTemplateExample returns example data rows for the CSV template.
func CSVTemplateExample() [][]string {
	return [][]string{
		{"server-01.example.com", "production", "linux;critical", "region=us-east;env=prod"},
		{"server-02.example.com", "staging", "linux;web", "region=us-west;env=staging"},
		{"workstation-01", "development", "windows;developer", "team=engineering"},
	}
}
