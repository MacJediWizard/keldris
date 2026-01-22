package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExcludePattern represents a set of exclude patterns for backups.
type ExcludePattern struct {
	ID          uuid.UUID  `json:"id"`
	OrgID       *uuid.UUID `json:"org_id,omitempty"` // nil for built-in patterns
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Patterns    []string   `json:"patterns"`
	Category    string     `json:"category"`
	IsBuiltin   bool       `json:"is_builtin"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewExcludePattern creates a new custom exclude pattern for an organization.
func NewExcludePattern(orgID uuid.UUID, name, description, category string, patterns []string) *ExcludePattern {
	now := time.Now()
	return &ExcludePattern{
		ID:          uuid.New(),
		OrgID:       &orgID,
		Name:        name,
		Description: description,
		Patterns:    patterns,
		Category:    category,
		IsBuiltin:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewBuiltinExcludePattern creates a new built-in exclude pattern (no org_id).
func NewBuiltinExcludePattern(name, description, category string, patterns []string) *ExcludePattern {
	now := time.Now()
	return &ExcludePattern{
		ID:          uuid.New(),
		OrgID:       nil,
		Name:        name,
		Description: description,
		Patterns:    patterns,
		Category:    category,
		IsBuiltin:   true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// SetPatterns sets the patterns from JSON bytes.
func (ep *ExcludePattern) SetPatterns(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &ep.Patterns)
}

// PatternsJSON returns the patterns as JSON bytes for database storage.
func (ep *ExcludePattern) PatternsJSON() ([]byte, error) {
	if ep.Patterns == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(ep.Patterns)
}

// CreateExcludePatternRequest represents the request to create a new exclude pattern.
type CreateExcludePatternRequest struct {
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Patterns    []string `json:"patterns" binding:"required"`
	Category    string   `json:"category" binding:"required"`
}

// UpdateExcludePatternRequest represents the request to update an exclude pattern.
type UpdateExcludePatternRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Patterns    []string `json:"patterns,omitempty"`
	Category    *string  `json:"category,omitempty"`
}
