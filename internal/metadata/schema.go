// Package metadata provides user-defined metadata schema definitions and validation.
package metadata

import (
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// EntityType represents the type of entity that can have metadata.
type EntityType string

const (
	// EntityTypeAgent represents an agent entity.
	EntityTypeAgent EntityType = "agent"
	// EntityTypeRepository represents a repository entity.
	EntityTypeRepository EntityType = "repository"
	// EntityTypeSchedule represents a schedule entity.
	EntityTypeSchedule EntityType = "schedule"
)

// ValidEntityTypes returns all valid entity types.
func ValidEntityTypes() []EntityType {
	return []EntityType{
		EntityTypeAgent,
		EntityTypeRepository,
		EntityTypeSchedule,
	}
}

// IsValid checks if the entity type is valid.
func (e EntityType) IsValid() bool {
	for _, t := range ValidEntityTypes() {
		if e == t {
			return true
		}
	}
	return false
}

// FieldType represents the type of a metadata field.
type FieldType string

const (
	// FieldTypeText represents a text field.
	FieldTypeText FieldType = "text"
	// FieldTypeNumber represents a numeric field.
	FieldTypeNumber FieldType = "number"
	// FieldTypeDate represents a date field.
	FieldTypeDate FieldType = "date"
	// FieldTypeSelect represents a single-select field.
	FieldTypeSelect FieldType = "select"
	// FieldTypeBoolean represents a boolean field.
	FieldTypeBoolean FieldType = "boolean"
)

// ValidFieldTypes returns all valid field types.
func ValidFieldTypes() []FieldType {
	return []FieldType{
		FieldTypeText,
		FieldTypeNumber,
		FieldTypeDate,
		FieldTypeSelect,
		FieldTypeBoolean,
	}
}

// IsValid checks if the field type is valid.
func (f FieldType) IsValid() bool {
	for _, t := range ValidFieldTypes() {
		if f == t {
			return true
		}
	}
	return false
}

// ValidationRules contains validation rules for metadata fields.
type ValidationRules struct {
	// MinLength is the minimum length for text fields.
	MinLength *int `json:"min_length,omitempty"`
	// MaxLength is the maximum length for text fields.
	MaxLength *int `json:"max_length,omitempty"`
	// Pattern is a regex pattern for text fields.
	Pattern *string `json:"pattern,omitempty"`
	// Min is the minimum value for number fields.
	Min *float64 `json:"min,omitempty"`
	// Max is the maximum value for number fields.
	Max *float64 `json:"max,omitempty"`
	// MinDate is the minimum date for date fields.
	MinDate *string `json:"min_date,omitempty"`
	// MaxDate is the maximum date for date fields.
	MaxDate *string `json:"max_date,omitempty"`
}

// SelectOption represents an option for select fields.
type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Schema represents a metadata field schema.
type Schema struct {
	ID           uuid.UUID        `json:"id"`
	OrgID        uuid.UUID        `json:"org_id"`
	EntityType   EntityType       `json:"entity_type"`
	Name         string           `json:"name"`
	FieldKey     string           `json:"field_key"`
	FieldType    FieldType        `json:"field_type"`
	Description  string           `json:"description,omitempty"`
	Required     bool             `json:"required"`
	DefaultValue interface{}      `json:"default_value,omitempty"`
	Options      []SelectOption   `json:"options,omitempty"` // For select type
	Validation   *ValidationRules `json:"validation,omitempty"`
	DisplayOrder int              `json:"display_order"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// NewSchema creates a new Schema with the given details.
func NewSchema(orgID uuid.UUID, entityType EntityType, name, fieldKey string, fieldType FieldType) *Schema {
	now := time.Now()
	return &Schema{
		ID:         uuid.New(),
		OrgID:      orgID,
		EntityType: entityType,
		Name:       name,
		FieldKey:   fieldKey,
		FieldType:  fieldType,
		Required:   false,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// fieldKeyRegex validates field keys (alphanumeric, underscore, dash).
var fieldKeyRegex = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,98}[a-z0-9]?$`)

// Validate validates the schema definition.
func (s *Schema) Validate() error {
	if !s.EntityType.IsValid() {
		return fmt.Errorf("invalid entity type: %s", s.EntityType)
	}

	if !s.FieldType.IsValid() {
		return fmt.Errorf("invalid field type: %s", s.FieldType)
	}

	if s.Name == "" {
		return fmt.Errorf("name is required")
	}

	if s.FieldKey == "" {
		return fmt.Errorf("field_key is required")
	}

	if !fieldKeyRegex.MatchString(s.FieldKey) {
		return fmt.Errorf("field_key must start with a letter, contain only lowercase letters, numbers, underscores, and dashes, and be 1-100 characters")
	}

	// Validate options for select type
	if s.FieldType == FieldTypeSelect {
		if len(s.Options) == 0 {
			return fmt.Errorf("options are required for select type")
		}
		seen := make(map[string]bool)
		for _, opt := range s.Options {
			if opt.Value == "" {
				return fmt.Errorf("option value cannot be empty")
			}
			if seen[opt.Value] {
				return fmt.Errorf("duplicate option value: %s", opt.Value)
			}
			seen[opt.Value] = true
		}
	}

	// Validate default value type matches field type
	if s.DefaultValue != nil {
		if err := s.ValidateValue(s.DefaultValue); err != nil {
			return fmt.Errorf("invalid default value: %w", err)
		}
	}

	return nil
}

// ValidateValue validates a value against this schema.
func (s *Schema) ValidateValue(value interface{}) error {
	if value == nil {
		if s.Required {
			return fmt.Errorf("field %s is required", s.FieldKey)
		}
		return nil
	}

	switch s.FieldType {
	case FieldTypeText:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %s must be a string", s.FieldKey)
		}
		if s.Validation != nil {
			if s.Validation.MinLength != nil && len(str) < *s.Validation.MinLength {
				return fmt.Errorf("field %s must be at least %d characters", s.FieldKey, *s.Validation.MinLength)
			}
			if s.Validation.MaxLength != nil && len(str) > *s.Validation.MaxLength {
				return fmt.Errorf("field %s must be at most %d characters", s.FieldKey, *s.Validation.MaxLength)
			}
			if s.Validation.Pattern != nil {
				re, err := regexp.Compile(*s.Validation.Pattern)
				if err != nil {
					return fmt.Errorf("invalid pattern in schema: %w", err)
				}
				if !re.MatchString(str) {
					return fmt.Errorf("field %s does not match pattern", s.FieldKey)
				}
			}
		}

	case FieldTypeNumber:
		var num float64
		switch v := value.(type) {
		case float64:
			num = v
		case float32:
			num = float64(v)
		case int:
			num = float64(v)
		case int64:
			num = float64(v)
		case json.Number:
			f, err := v.Float64()
			if err != nil {
				return fmt.Errorf("field %s must be a number", s.FieldKey)
			}
			num = f
		default:
			return fmt.Errorf("field %s must be a number", s.FieldKey)
		}
		if s.Validation != nil {
			if s.Validation.Min != nil && num < *s.Validation.Min {
				return fmt.Errorf("field %s must be at least %f", s.FieldKey, *s.Validation.Min)
			}
			if s.Validation.Max != nil && num > *s.Validation.Max {
				return fmt.Errorf("field %s must be at most %f", s.FieldKey, *s.Validation.Max)
			}
		}

	case FieldTypeDate:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %s must be a date string", s.FieldKey)
		}
		parsedDate, err := time.Parse("2006-01-02", str)
		if err != nil {
			return fmt.Errorf("field %s must be a valid date (YYYY-MM-DD)", s.FieldKey)
		}
		if s.Validation != nil {
			if s.Validation.MinDate != nil {
				minDate, err := time.Parse("2006-01-02", *s.Validation.MinDate)
				if err == nil && parsedDate.Before(minDate) {
					return fmt.Errorf("field %s must be on or after %s", s.FieldKey, *s.Validation.MinDate)
				}
			}
			if s.Validation.MaxDate != nil {
				maxDate, err := time.Parse("2006-01-02", *s.Validation.MaxDate)
				if err == nil && parsedDate.After(maxDate) {
					return fmt.Errorf("field %s must be on or before %s", s.FieldKey, *s.Validation.MaxDate)
				}
			}
		}

	case FieldTypeSelect:
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %s must be a string", s.FieldKey)
		}
		valid := false
		for _, opt := range s.Options {
			if opt.Value == str {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("field %s has invalid value: %s", s.FieldKey, str)
		}

	case FieldTypeBoolean:
		_, ok := value.(bool)
		if !ok {
			return fmt.Errorf("field %s must be a boolean", s.FieldKey)
		}
	}

	return nil
}

// SetOptionsJSON sets options from JSON bytes.
func (s *Schema) SetOptionsJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.Options)
}

// OptionsJSON returns options as JSON bytes.
func (s *Schema) OptionsJSON() ([]byte, error) {
	if s.Options == nil {
		return nil, nil
	}
	return json.Marshal(s.Options)
}

// SetValidationJSON sets validation rules from JSON bytes.
func (s *Schema) SetValidationJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var v ValidationRules
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	s.Validation = &v
	return nil
}

// ValidationJSON returns validation rules as JSON bytes.
func (s *Schema) ValidationJSON() ([]byte, error) {
	if s.Validation == nil {
		return nil, nil
	}
	return json.Marshal(s.Validation)
}

// SetDefaultValueJSON sets the default value from JSON bytes.
func (s *Schema) SetDefaultValueJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, &s.DefaultValue)
}

// DefaultValueJSON returns the default value as JSON bytes.
func (s *Schema) DefaultValueJSON() ([]byte, error) {
	if s.DefaultValue == nil {
		return nil, nil
	}
	return json.Marshal(s.DefaultValue)
}

// MetadataValue represents a key-value pair in entity metadata.
type MetadataValue struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// ValidateMetadata validates metadata values against the provided schemas.
func ValidateMetadata(schemas []*Schema, metadata map[string]interface{}) error {
	schemaMap := make(map[string]*Schema)
	for _, s := range schemas {
		schemaMap[s.FieldKey] = s
	}

	// Check required fields
	for _, s := range schemas {
		if s.Required {
			if _, ok := metadata[s.FieldKey]; !ok {
				return fmt.Errorf("required field %s is missing", s.FieldKey)
			}
		}
	}

	// Validate each provided value
	for key, value := range metadata {
		schema, ok := schemaMap[key]
		if !ok {
			// Unknown field - allow it but don't validate
			continue
		}
		if err := schema.ValidateValue(value); err != nil {
			return err
		}
	}

	return nil
}

// ApplyDefaults applies default values from schemas to metadata.
func ApplyDefaults(schemas []*Schema, metadata map[string]interface{}) map[string]interface{} {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	for _, s := range schemas {
		if s.DefaultValue != nil {
			if _, ok := metadata[s.FieldKey]; !ok {
				metadata[s.FieldKey] = s.DefaultValue
			}
		}
	}
	return metadata
}

// SchemaInfo provides information about a field type for the UI.
type SchemaInfo struct {
	Type        FieldType `json:"type"`
	Label       string    `json:"label"`
	Description string    `json:"description"`
}

// GetFieldTypeInfo returns information about all field types.
func GetFieldTypeInfo() []SchemaInfo {
	return []SchemaInfo{
		{Type: FieldTypeText, Label: "Text", Description: "Single or multi-line text input"},
		{Type: FieldTypeNumber, Label: "Number", Description: "Numeric value (integer or decimal)"},
		{Type: FieldTypeDate, Label: "Date", Description: "Date value (YYYY-MM-DD format)"},
		{Type: FieldTypeSelect, Label: "Select", Description: "Single selection from predefined options"},
		{Type: FieldTypeBoolean, Label: "Boolean", Description: "True/false toggle"},
	}
}

// EntityTypeInfo provides information about an entity type for the UI.
type EntityTypeInfo struct {
	Type        EntityType `json:"type"`
	Label       string     `json:"label"`
	Description string     `json:"description"`
}

// GetEntityTypeInfo returns information about all entity types.
func GetEntityTypeInfo() []EntityTypeInfo {
	return []EntityTypeInfo{
		{Type: EntityTypeAgent, Label: "Agent", Description: "Backup agents installed on hosts"},
		{Type: EntityTypeRepository, Label: "Repository", Description: "Backup storage destinations"},
		{Type: EntityTypeSchedule, Label: "Schedule", Description: "Backup schedule configurations"},
	}
}
