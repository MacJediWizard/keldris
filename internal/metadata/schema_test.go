package metadata

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

// ---- EntityType tests ----

func TestEntityType_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		input EntityType
		want  bool
	}{
		{"agent", EntityTypeAgent, true},
		{"repository", EntityTypeRepository, true},
		{"schedule", EntityTypeSchedule, true},
		{"invalid_type", EntityType("invalid"), false},
		{"empty_string", EntityType(""), false},
		{"uppercase", EntityType("Agent"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.IsValid()
			if got != tt.want {
				t.Errorf("EntityType(%q).IsValid() = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidEntityTypes(t *testing.T) {
	types := ValidEntityTypes()
	if len(types) != 3 {
		t.Fatalf("ValidEntityTypes() returned %d types, want 3", len(types))
	}

	expected := map[EntityType]bool{
		EntityTypeAgent:      true,
		EntityTypeRepository: true,
		EntityTypeSchedule:   true,
	}
	for _, et := range types {
		if !expected[et] {
			t.Errorf("unexpected entity type: %s", et)
		}
	}
}

// ---- FieldType tests ----

func TestFieldType_IsValid(t *testing.T) {
	tests := []struct {
		name  string
		input FieldType
		want  bool
	}{
		{"text", FieldTypeText, true},
		{"number", FieldTypeNumber, true},
		{"date", FieldTypeDate, true},
		{"select", FieldTypeSelect, true},
		{"boolean", FieldTypeBoolean, true},
		{"invalid", FieldType("invalid"), false},
		{"empty", FieldType(""), false},
		{"integer", FieldType("integer"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.IsValid()
			if got != tt.want {
				t.Errorf("FieldType(%q).IsValid() = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidFieldTypes(t *testing.T) {
	types := ValidFieldTypes()
	if len(types) != 5 {
		t.Fatalf("ValidFieldTypes() returned %d types, want 5", len(types))
	}
}

// ---- Schema creation and validation tests ----

func TestNewSchema(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Location", "location", FieldTypeText)

	if schema.ID == uuid.Nil {
		t.Error("ID should not be nil UUID")
	}
	if schema.OrgID != orgID {
		t.Errorf("OrgID = %s, want %s", schema.OrgID, orgID)
	}
	if schema.EntityType != EntityTypeAgent {
		t.Errorf("EntityType = %q, want %q", schema.EntityType, EntityTypeAgent)
	}
	if schema.Name != "Location" {
		t.Errorf("Name = %q, want %q", schema.Name, "Location")
	}
	if schema.FieldKey != "location" {
		t.Errorf("FieldKey = %q, want %q", schema.FieldKey, "location")
	}
	if schema.FieldType != FieldTypeText {
		t.Errorf("FieldType = %q, want %q", schema.FieldType, FieldTypeText)
	}
	if schema.Required {
		t.Error("Required should default to false")
	}
	if schema.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if schema.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestSchema_Validate(t *testing.T) {
	orgID := uuid.New()

	tests := []struct {
		name    string
		schema  *Schema
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid_text_schema",
			schema:  NewSchema(orgID, EntityTypeAgent, "Location", "location", FieldTypeText),
			wantErr: false,
		},
		{
			name:    "valid_number_schema",
			schema:  NewSchema(orgID, EntityTypeRepository, "Max Size", "max-size", FieldTypeNumber),
			wantErr: false,
		},
		{
			name:    "valid_boolean_schema",
			schema:  NewSchema(orgID, EntityTypeSchedule, "Active", "active", FieldTypeBoolean),
			wantErr: false,
		},
		{
			name:    "valid_date_schema",
			schema:  NewSchema(orgID, EntityTypeAgent, "Expiry", "expiry-date", FieldTypeDate),
			wantErr: false,
		},
		{
			name: "valid_select_schema",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Env", "env", FieldTypeSelect)
				s.Options = []SelectOption{
					{Value: "prod", Label: "Production"},
					{Value: "dev", Label: "Development"},
				}
				return s
			}(),
			wantErr: false,
		},
		{
			name: "invalid_entity_type",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityType("invalid"), "Name", "name", FieldTypeText)
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid entity type",
		},
		{
			name: "invalid_field_type",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldType("invalid"))
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid field type",
		},
		{
			name: "empty_name",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "", "key", FieldTypeText)
				return s
			}(),
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "empty_field_key",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Name", "", FieldTypeText)
				return s
			}(),
			wantErr: true,
			errMsg:  "field_key is required",
		},
		{
			name: "field_key_starts_with_number",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Name", "1abc", FieldTypeText)
				return s
			}(),
			wantErr: true,
			errMsg:  "field_key must start with a letter",
		},
		{
			name: "field_key_with_uppercase",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Name", "MyKey", FieldTypeText)
				return s
			}(),
			wantErr: true,
			errMsg:  "field_key must start with a letter",
		},
		{
			name: "field_key_with_spaces",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Name", "my key", FieldTypeText)
				return s
			}(),
			wantErr: true,
			errMsg:  "field_key must start with a letter",
		},
		{
			name: "select_without_options",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Env", "env", FieldTypeSelect)
				return s
			}(),
			wantErr: true,
			errMsg:  "options are required for select type",
		},
		{
			name: "select_with_empty_option_value",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Env", "env", FieldTypeSelect)
				s.Options = []SelectOption{{Value: "", Label: "Empty"}}
				return s
			}(),
			wantErr: true,
			errMsg:  "option value cannot be empty",
		},
		{
			name: "select_with_duplicate_options",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Env", "env", FieldTypeSelect)
				s.Options = []SelectOption{
					{Value: "prod", Label: "Production"},
					{Value: "prod", Label: "Production Again"},
				}
				return s
			}(),
			wantErr: true,
			errMsg:  "duplicate option value: prod",
		},
		{
			name: "valid_default_value",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Active", "active", FieldTypeBoolean)
				s.DefaultValue = true
				return s
			}(),
			wantErr: false,
		},
		{
			name: "invalid_default_value_type_mismatch",
			schema: func() *Schema {
				s := NewSchema(orgID, EntityTypeAgent, "Active", "active", FieldTypeBoolean)
				s.DefaultValue = "not-a-bool"
				return s
			}(),
			wantErr: true,
			errMsg:  "invalid default value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" {
					if got := err.Error(); !contains(got, tt.errMsg) {
						t.Errorf("error = %q, want it to contain %q", got, tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---- ValidateValue tests ----

func TestSchema_ValidateValue_Text(t *testing.T) {
	orgID := uuid.New()
	minLen := 3
	maxLen := 10
	pattern := `^[a-z]+$`

	schema := NewSchema(orgID, EntityTypeAgent, "Tag", "tag", FieldTypeText)
	schema.Validation = &ValidationRules{
		MinLength: &minLen,
		MaxLength: &maxLen,
		Pattern:   &pattern,
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid_text", "hello", false},
		{"valid_min_boundary", "abc", false},
		{"valid_max_boundary", "abcdefghij", false},
		{"too_short", "ab", true},
		{"too_long", "abcdefghijk", true},
		{"pattern_fail_digits", "abc123", true},
		{"pattern_fail_uppercase", "Hello", true},
		{"wrong_type_int", 42, true},
		{"wrong_type_bool", true, true},
		{"nil_value_not_required", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue(%v) error = %v, wantErr = %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestSchema_ValidateValue_Text_Required(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)
	schema.Required = true

	err := schema.ValidateValue(nil)
	if err == nil {
		t.Fatal("expected error for nil value on required field")
	}
	if !contains(err.Error(), "required") {
		t.Errorf("error = %q, want it to contain 'required'", err.Error())
	}
}

func TestSchema_ValidateValue_Number(t *testing.T) {
	orgID := uuid.New()
	minVal := 0.0
	maxVal := 100.0

	schema := NewSchema(orgID, EntityTypeAgent, "Score", "score", FieldTypeNumber)
	schema.Validation = &ValidationRules{
		Min: &minVal,
		Max: &maxVal,
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid_int", 50, false},
		{"valid_float64", 99.5, false},
		{"valid_float32", float32(50.5), false},
		{"valid_int64", int64(75), false},
		{"valid_zero_boundary", 0.0, false},
		{"valid_max_boundary", 100.0, false},
		{"below_min", -1.0, true},
		{"above_max", 101.0, true},
		{"wrong_type_string", "fifty", true},
		{"wrong_type_bool", true, true},
		{"json_number", json.Number("42.5"), false},
		{"json_number_invalid", json.Number("not-a-number"), true},
		{"nil_not_required", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue(%v) error = %v, wantErr = %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestSchema_ValidateValue_Date(t *testing.T) {
	orgID := uuid.New()
	minDate := "2024-01-01"
	maxDate := "2024-12-31"

	schema := NewSchema(orgID, EntityTypeAgent, "Expiry", "expiry", FieldTypeDate)
	schema.Validation = &ValidationRules{
		MinDate: &minDate,
		MaxDate: &maxDate,
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid_date", "2024-06-15", false},
		{"valid_min_boundary", "2024-01-01", false},
		{"valid_max_boundary", "2024-12-31", false},
		{"before_min", "2023-12-31", true},
		{"after_max", "2025-01-01", true},
		{"invalid_format", "06/15/2024", true},
		{"invalid_date", "not-a-date", true},
		{"wrong_type_int", 20240615, true},
		{"nil_not_required", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue(%v) error = %v, wantErr = %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestSchema_ValidateValue_Select(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Env", "env", FieldTypeSelect)
	schema.Options = []SelectOption{
		{Value: "prod", Label: "Production"},
		{Value: "staging", Label: "Staging"},
		{Value: "dev", Label: "Development"},
	}

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"valid_option", "prod", false},
		{"valid_option_staging", "staging", false},
		{"valid_option_dev", "dev", false},
		{"invalid_option", "testing", true},
		{"case_sensitive", "Prod", true},
		{"wrong_type_int", 1, true},
		{"nil_not_required", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue(%v) error = %v, wantErr = %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestSchema_ValidateValue_Boolean(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Active", "active", FieldTypeBoolean)

	tests := []struct {
		name    string
		value   interface{}
		wantErr bool
	}{
		{"true", true, false},
		{"false", false, false},
		{"string_true", "true", true},
		{"int_one", 1, true},
		{"nil_not_required", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := schema.ValidateValue(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateValue(%v) error = %v, wantErr = %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestSchema_ValidateValue_NoValidation(t *testing.T) {
	orgID := uuid.New()

	// Text without validation rules
	schema := NewSchema(orgID, EntityTypeAgent, "Notes", "notes", FieldTypeText)
	if err := schema.ValidateValue("any string is fine"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Number without validation rules
	numSchema := NewSchema(orgID, EntityTypeAgent, "Count", "count", FieldTypeNumber)
	if err := numSchema.ValidateValue(999999.0); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---- ValidateMetadata tests ----

func TestValidateMetadata(t *testing.T) {
	orgID := uuid.New()

	nameSchema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)
	nameSchema.Required = true

	envSchema := NewSchema(orgID, EntityTypeAgent, "Environment", "env", FieldTypeSelect)
	envSchema.Options = []SelectOption{
		{Value: "prod", Label: "Production"},
		{Value: "dev", Label: "Development"},
	}

	activeSchema := NewSchema(orgID, EntityTypeAgent, "Active", "active", FieldTypeBoolean)

	schemas := []*Schema{nameSchema, envSchema, activeSchema}

	tests := []struct {
		name     string
		metadata map[string]interface{}
		wantErr  bool
		errMsg   string
	}{
		{
			name: "all_valid",
			metadata: map[string]interface{}{
				"name":   "my-agent",
				"env":    "prod",
				"active": true,
			},
			wantErr: false,
		},
		{
			name: "missing_required_field",
			metadata: map[string]interface{}{
				"env":    "prod",
				"active": true,
			},
			wantErr: true,
			errMsg:  "required field name is missing",
		},
		{
			name: "invalid_select_value",
			metadata: map[string]interface{}{
				"name": "my-agent",
				"env":  "staging",
			},
			wantErr: true,
			errMsg:  "invalid value",
		},
		{
			name: "invalid_type_for_boolean",
			metadata: map[string]interface{}{
				"name":   "my-agent",
				"active": "yes",
			},
			wantErr: true,
			errMsg:  "must be a boolean",
		},
		{
			name: "unknown_fields_allowed",
			metadata: map[string]interface{}{
				"name":    "my-agent",
				"unknown": "some-value",
			},
			wantErr: false,
		},
		{
			name: "only_required_fields",
			metadata: map[string]interface{}{
				"name": "my-agent",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMetadata(schemas, tt.metadata)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// ---- ApplyDefaults tests ----

func TestApplyDefaults(t *testing.T) {
	orgID := uuid.New()

	envSchema := NewSchema(orgID, EntityTypeAgent, "Env", "env", FieldTypeText)
	envSchema.DefaultValue = "dev"

	activeSchema := NewSchema(orgID, EntityTypeAgent, "Active", "active", FieldTypeBoolean)
	activeSchema.DefaultValue = true

	noDefaultSchema := NewSchema(orgID, EntityTypeAgent, "Notes", "notes", FieldTypeText)

	schemas := []*Schema{envSchema, activeSchema, noDefaultSchema}

	t.Run("nil_metadata", func(t *testing.T) {
		result := ApplyDefaults(schemas, nil)
		if result == nil {
			t.Fatal("result should not be nil")
		}
		if result["env"] != "dev" {
			t.Errorf("env = %v, want %q", result["env"], "dev")
		}
		if result["active"] != true {
			t.Errorf("active = %v, want true", result["active"])
		}
		if _, exists := result["notes"]; exists {
			t.Error("notes should not have a default")
		}
	})

	t.Run("existing_values_not_overwritten", func(t *testing.T) {
		metadata := map[string]interface{}{
			"env": "prod",
		}
		result := ApplyDefaults(schemas, metadata)
		if result["env"] != "prod" {
			t.Errorf("env = %v, want %q (should not be overwritten)", result["env"], "prod")
		}
		if result["active"] != true {
			t.Errorf("active = %v, want true (should be applied)", result["active"])
		}
	})

	t.Run("empty_metadata", func(t *testing.T) {
		metadata := map[string]interface{}{}
		result := ApplyDefaults(schemas, metadata)
		if result["env"] != "dev" {
			t.Errorf("env = %v, want %q", result["env"], "dev")
		}
		if result["active"] != true {
			t.Errorf("active = %v, want true", result["active"])
		}
	})
}

// ---- JSON serialization tests ----

func TestSchema_OptionsJSON(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Env", "env", FieldTypeSelect)
	schema.Options = []SelectOption{
		{Value: "prod", Label: "Production"},
		{Value: "dev", Label: "Development"},
	}

	data, err := schema.OptionsJSON()
	if err != nil {
		t.Fatalf("OptionsJSON: %v", err)
	}
	if data == nil {
		t.Fatal("OptionsJSON returned nil")
	}

	// Round-trip test
	schema2 := NewSchema(orgID, EntityTypeAgent, "Env2", "env2", FieldTypeSelect)
	if err := schema2.SetOptionsJSON(data); err != nil {
		t.Fatalf("SetOptionsJSON: %v", err)
	}

	if len(schema2.Options) != 2 {
		t.Fatalf("Options count = %d, want 2", len(schema2.Options))
	}
	if schema2.Options[0].Value != "prod" {
		t.Errorf("Options[0].Value = %q, want %q", schema2.Options[0].Value, "prod")
	}
	if schema2.Options[1].Label != "Development" {
		t.Errorf("Options[1].Label = %q, want %q", schema2.Options[1].Label, "Development")
	}
}

func TestSchema_OptionsJSON_Nil(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)

	data, err := schema.OptionsJSON()
	if err != nil {
		t.Fatalf("OptionsJSON: %v", err)
	}
	if data != nil {
		t.Errorf("OptionsJSON = %v, want nil for nil options", data)
	}
}

func TestSchema_SetOptionsJSON_Empty(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)

	if err := schema.SetOptionsJSON(nil); err != nil {
		t.Fatalf("SetOptionsJSON(nil): %v", err)
	}
	if err := schema.SetOptionsJSON([]byte{}); err != nil {
		t.Fatalf("SetOptionsJSON(empty): %v", err)
	}
}

func TestSchema_ValidationJSON(t *testing.T) {
	orgID := uuid.New()
	minLen := 3
	maxLen := 100
	schema := NewSchema(orgID, EntityTypeAgent, "Tag", "tag", FieldTypeText)
	schema.Validation = &ValidationRules{
		MinLength: &minLen,
		MaxLength: &maxLen,
	}

	data, err := schema.ValidationJSON()
	if err != nil {
		t.Fatalf("ValidationJSON: %v", err)
	}

	// Round-trip
	schema2 := NewSchema(orgID, EntityTypeAgent, "Tag2", "tag2", FieldTypeText)
	if err := schema2.SetValidationJSON(data); err != nil {
		t.Fatalf("SetValidationJSON: %v", err)
	}

	if schema2.Validation == nil {
		t.Fatal("Validation is nil after SetValidationJSON")
	}
	if *schema2.Validation.MinLength != 3 {
		t.Errorf("MinLength = %d, want 3", *schema2.Validation.MinLength)
	}
	if *schema2.Validation.MaxLength != 100 {
		t.Errorf("MaxLength = %d, want 100", *schema2.Validation.MaxLength)
	}
}

func TestSchema_ValidationJSON_Nil(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)

	data, err := schema.ValidationJSON()
	if err != nil {
		t.Fatalf("ValidationJSON: %v", err)
	}
	if data != nil {
		t.Errorf("ValidationJSON = %v, want nil for nil validation", data)
	}
}

func TestSchema_SetValidationJSON_Empty(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)

	if err := schema.SetValidationJSON(nil); err != nil {
		t.Fatalf("SetValidationJSON(nil): %v", err)
	}
	if err := schema.SetValidationJSON([]byte{}); err != nil {
		t.Fatalf("SetValidationJSON(empty): %v", err)
	}
}

func TestSchema_DefaultValueJSON(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Active", "active", FieldTypeBoolean)
	schema.DefaultValue = true

	data, err := schema.DefaultValueJSON()
	if err != nil {
		t.Fatalf("DefaultValueJSON: %v", err)
	}

	schema2 := NewSchema(orgID, EntityTypeAgent, "Active2", "active2", FieldTypeBoolean)
	if err := schema2.SetDefaultValueJSON(data); err != nil {
		t.Fatalf("SetDefaultValueJSON: %v", err)
	}

	if schema2.DefaultValue != true {
		t.Errorf("DefaultValue = %v, want true", schema2.DefaultValue)
	}
}

func TestSchema_DefaultValueJSON_Nil(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)

	data, err := schema.DefaultValueJSON()
	if err != nil {
		t.Fatalf("DefaultValueJSON: %v", err)
	}
	if data != nil {
		t.Errorf("DefaultValueJSON = %v, want nil", data)
	}
}

func TestSchema_SetDefaultValueJSON_Empty(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Name", "name", FieldTypeText)

	if err := schema.SetDefaultValueJSON(nil); err != nil {
		t.Fatalf("SetDefaultValueJSON(nil): %v", err)
	}
	if err := schema.SetDefaultValueJSON([]byte{}); err != nil {
		t.Fatalf("SetDefaultValueJSON(empty): %v", err)
	}
}

// ---- Schema JSON round-trip ----

func TestSchema_JSONRoundTrip(t *testing.T) {
	orgID := uuid.New()
	schema := NewSchema(orgID, EntityTypeAgent, "Environment", "environment", FieldTypeSelect)
	schema.Description = "Deployment environment"
	schema.Required = true
	schema.DisplayOrder = 5
	schema.Options = []SelectOption{
		{Value: "prod", Label: "Production"},
		{Value: "dev", Label: "Development"},
	}

	data, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Schema
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != schema.ID {
		t.Errorf("ID = %s, want %s", decoded.ID, schema.ID)
	}
	if decoded.Name != "Environment" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Environment")
	}
	if decoded.FieldKey != "environment" {
		t.Errorf("FieldKey = %q, want %q", decoded.FieldKey, "environment")
	}
	if decoded.EntityType != EntityTypeAgent {
		t.Errorf("EntityType = %q, want %q", decoded.EntityType, EntityTypeAgent)
	}
	if decoded.FieldType != FieldTypeSelect {
		t.Errorf("FieldType = %q, want %q", decoded.FieldType, FieldTypeSelect)
	}
	if decoded.Description != "Deployment environment" {
		t.Errorf("Description = %q, want %q", decoded.Description, "Deployment environment")
	}
	if !decoded.Required {
		t.Error("Required = false, want true")
	}
	if decoded.DisplayOrder != 5 {
		t.Errorf("DisplayOrder = %d, want 5", decoded.DisplayOrder)
	}
	if len(decoded.Options) != 2 {
		t.Fatalf("Options count = %d, want 2", len(decoded.Options))
	}
}

// ---- FieldKey regex edge cases ----

func TestFieldKeyRegex(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		valid bool
	}{
		{"single_char", "a", true},
		{"two_chars", "ab", true},
		{"with_numbers", "abc123", true},
		{"with_underscores", "my_field", true},
		{"with_dashes", "my-field", true},
		{"mixed", "my-field_123", true},
		{"starts_with_number", "1field", false},
		{"starts_with_dash", "-field", false},
		{"starts_with_underscore", "_field", false},
		{"uppercase", "MyField", false},
		{"has_spaces", "my field", false},
		{"has_dots", "my.field", false},
		{"empty", "", false},
		{"ends_with_dash", "field-", true},       // regex allows trailing dash (via {0,98} group)
		{"ends_with_underscore", "field_", true}, // regex allows trailing underscore (via {0,98} group)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fieldKeyRegex.MatchString(tt.key)
			if got != tt.valid {
				t.Errorf("fieldKeyRegex.MatchString(%q) = %v, want %v", tt.key, got, tt.valid)
			}
		})
	}
}

// ---- Info functions tests ----

func TestGetFieldTypeInfo(t *testing.T) {
	infos := GetFieldTypeInfo()
	if len(infos) != 5 {
		t.Fatalf("GetFieldTypeInfo() returned %d items, want 5", len(infos))
	}

	types := make(map[FieldType]bool)
	for _, info := range infos {
		if info.Label == "" {
			t.Errorf("empty label for type %s", info.Type)
		}
		if info.Description == "" {
			t.Errorf("empty description for type %s", info.Type)
		}
		types[info.Type] = true
	}

	for _, ft := range ValidFieldTypes() {
		if !types[ft] {
			t.Errorf("missing info for field type %s", ft)
		}
	}
}

func TestGetEntityTypeInfo(t *testing.T) {
	infos := GetEntityTypeInfo()
	if len(infos) != 3 {
		t.Fatalf("GetEntityTypeInfo() returned %d items, want 3", len(infos))
	}

	types := make(map[EntityType]bool)
	for _, info := range infos {
		if info.Label == "" {
			t.Errorf("empty label for type %s", info.Type)
		}
		if info.Description == "" {
			t.Errorf("empty description for type %s", info.Type)
		}
		types[info.Type] = true
	}

	for _, et := range ValidEntityTypes() {
		if !types[et] {
			t.Errorf("missing info for entity type %s", et)
		}
	}
}

// ---- Validation rules edge cases ----

func TestValidationRules_DateBoundary(t *testing.T) {
	orgID := uuid.New()
	minDate := "2024-06-15"
	maxDate := "2024-06-15"

	schema := NewSchema(orgID, EntityTypeAgent, "Exact Date", "exact-date", FieldTypeDate)
	schema.Validation = &ValidationRules{
		MinDate: &minDate,
		MaxDate: &maxDate,
	}

	// Exact match should pass
	if err := schema.ValidateValue("2024-06-15"); err != nil {
		t.Errorf("exact boundary should pass: %v", err)
	}

	// One day before should fail
	if err := schema.ValidateValue("2024-06-14"); err == nil {
		t.Error("day before min should fail")
	}

	// One day after should fail
	if err := schema.ValidateValue("2024-06-16"); err == nil {
		t.Error("day after max should fail")
	}
}

func TestValidationRules_NumberBoundary(t *testing.T) {
	orgID := uuid.New()
	minVal := 10.0
	maxVal := 10.0

	schema := NewSchema(orgID, EntityTypeAgent, "Exact", "exact", FieldTypeNumber)
	schema.Validation = &ValidationRules{
		Min: &minVal,
		Max: &maxVal,
	}

	if err := schema.ValidateValue(10.0); err != nil {
		t.Errorf("exact boundary should pass: %v", err)
	}
	if err := schema.ValidateValue(9.9); err == nil {
		t.Error("below min should fail")
	}
	if err := schema.ValidateValue(10.1); err == nil {
		t.Error("above max should fail")
	}
}

// ---- MetadataValue test ----

func TestMetadataValue_JSON(t *testing.T) {
	mv := MetadataValue{
		Key:   "env",
		Value: "production",
	}

	data, err := json.Marshal(mv)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded MetadataValue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.Key != "env" {
		t.Errorf("Key = %q, want %q", decoded.Key, "env")
	}
	if decoded.Value != "production" {
		t.Errorf("Value = %v, want %q", decoded.Value, "production")
	}
}

// ---- helper ----

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
