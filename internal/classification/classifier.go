// Package classification provides data classification for backup paths.
// This enables compliance tracking and sensitive data management.
package classification

import (
	"path/filepath"
	"strings"
)

// Level represents the sensitivity level of data.
type Level string

const (
	// LevelPublic indicates non-sensitive, publicly shareable data.
	LevelPublic Level = "public"
	// LevelInternal indicates internal business data with limited access.
	LevelInternal Level = "internal"
	// LevelConfidential indicates sensitive data requiring protection.
	LevelConfidential Level = "confidential"
	// LevelRestricted indicates highly sensitive data with strict access controls.
	LevelRestricted Level = "restricted"
)

// DataType represents the type of sensitive data.
type DataType string

const (
	// DataTypePII indicates Personally Identifiable Information.
	DataTypePII DataType = "pii"
	// DataTypePHI indicates Protected Health Information (HIPAA).
	DataTypePHI DataType = "phi"
	// DataTypePCI indicates Payment Card Industry data (PCI-DSS).
	DataTypePCI DataType = "pci"
	// DataTypeProprietary indicates proprietary business data.
	DataTypeProprietary DataType = "proprietary"
	// DataTypeGeneral indicates general unclassified data.
	DataTypeGeneral DataType = "general"
)

// Classification represents the classification of a path or dataset.
type Classification struct {
	Level     Level      `json:"level"`
	DataTypes []DataType `json:"data_types"`
}

// AllLevels returns all classification levels in order of sensitivity.
func AllLevels() []Level {
	return []Level{LevelPublic, LevelInternal, LevelConfidential, LevelRestricted}
}

// AllDataTypes returns all data types.
func AllDataTypes() []DataType {
	return []DataType{DataTypePII, DataTypePHI, DataTypePCI, DataTypeProprietary, DataTypeGeneral}
}

// LevelPriority returns the priority/sensitivity of a level (higher = more sensitive).
func LevelPriority(l Level) int {
	switch l {
	case LevelPublic:
		return 1
	case LevelInternal:
		return 2
	case LevelConfidential:
		return 3
	case LevelRestricted:
		return 4
	default:
		return 0
	}
}

// MaxLevel returns the highest sensitivity level from the given levels.
func MaxLevel(levels ...Level) Level {
	max := LevelPublic
	maxPriority := LevelPriority(max)
	for _, l := range levels {
		p := LevelPriority(l)
		if p > maxPriority {
			max = l
			maxPriority = p
		}
	}
	return max
}

// PathRule defines a classification rule based on path patterns.
type PathRule struct {
	// Pattern is a glob pattern to match paths (e.g., "/home/*/medical/*").
	Pattern string `json:"pattern"`
	// Level is the classification level for matching paths.
	Level Level `json:"level"`
	// DataTypes are the data types for matching paths.
	DataTypes []DataType `json:"data_types"`
	// Description explains what this rule matches.
	Description string `json:"description"`
}

// Matches checks if the rule pattern matches the given path.
func (r *PathRule) Matches(path string) bool {
	// Normalize path separators
	pattern := filepath.Clean(r.Pattern)
	path = filepath.Clean(path)

	// Use filepath.Match for glob matching
	matched, err := filepath.Match(pattern, path)
	if err == nil && matched {
		return true
	}

	// Try matching path components
	patternParts := strings.Split(pattern, string(filepath.Separator))
	pathParts := strings.Split(path, string(filepath.Separator))

	return matchParts(patternParts, pathParts)
}

// matchParts performs recursive glob matching on path components.
func matchParts(pattern, path []string) bool {
	for len(pattern) > 0 && len(path) > 0 {
		if pattern[0] == "**" {
			// ** matches zero or more directories
			if len(pattern) == 1 {
				return true
			}
			// Try matching rest of pattern at each position
			for i := 0; i <= len(path); i++ {
				if matchParts(pattern[1:], path[i:]) {
					return true
				}
			}
			return false
		}

		matched, err := filepath.Match(pattern[0], path[0])
		if err != nil || !matched {
			return false
		}
		pattern = pattern[1:]
		path = path[1:]
	}

	// Check for trailing ** which matches empty
	if len(pattern) == 1 && pattern[0] == "**" {
		return true
	}

	return len(pattern) == 0 && len(path) == 0
}

// Classifier applies classification rules to paths.
type Classifier struct {
	rules []PathRule
}

// NewClassifier creates a new Classifier with the given rules.
func NewClassifier(rules []PathRule) *Classifier {
	return &Classifier{rules: rules}
}

// Classify determines the classification for a path based on configured rules.
// Returns nil if no rules match (defaults to public/general).
func (c *Classifier) Classify(path string) *Classification {
	var result *Classification

	for _, rule := range c.rules {
		if rule.Matches(path) {
			if result == nil {
				result = &Classification{
					Level:     rule.Level,
					DataTypes: make([]DataType, 0),
				}
			} else {
				// Use the highest classification level
				result.Level = MaxLevel(result.Level, rule.Level)
			}
			// Merge data types
			result.DataTypes = mergeDataTypes(result.DataTypes, rule.DataTypes)
		}
	}

	return result
}

// ClassifyPaths determines the aggregate classification for multiple paths.
func (c *Classifier) ClassifyPaths(paths []string) *Classification {
	result := &Classification{
		Level:     LevelPublic,
		DataTypes: []DataType{},
	}

	for _, path := range paths {
		classification := c.Classify(path)
		if classification != nil {
			result.Level = MaxLevel(result.Level, classification.Level)
			result.DataTypes = mergeDataTypes(result.DataTypes, classification.DataTypes)
		}
	}

	if len(result.DataTypes) == 0 {
		result.DataTypes = []DataType{DataTypeGeneral}
	}

	return result
}

// mergeDataTypes combines two slices of data types, removing duplicates.
func mergeDataTypes(a, b []DataType) []DataType {
	seen := make(map[DataType]bool)
	for _, dt := range a {
		seen[dt] = true
	}
	for _, dt := range b {
		seen[dt] = true
	}

	result := make([]DataType, 0, len(seen))
	for dt := range seen {
		result = append(result, dt)
	}
	return result
}

// DefaultRules returns commonly used classification rules.
func DefaultRules() []PathRule {
	return []PathRule{
		// Healthcare / PHI
		{
			Pattern:     "**/medical/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePHI},
			Description: "Medical records and health information",
		},
		{
			Pattern:     "**/health/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePHI},
			Description: "Health-related data",
		},
		{
			Pattern:     "**/hipaa/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePHI},
			Description: "HIPAA regulated data",
		},
		{
			Pattern:     "**/patient*/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePHI},
			Description: "Patient records",
		},

		// Payment / PCI
		{
			Pattern:     "**/payment/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePCI},
			Description: "Payment processing data",
		},
		{
			Pattern:     "**/credit*/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePCI},
			Description: "Credit card data",
		},
		{
			Pattern:     "**/cardholder/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePCI},
			Description: "Cardholder data",
		},
		{
			Pattern:     "**/pci/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePCI},
			Description: "PCI-DSS regulated data",
		},

		// PII
		{
			Pattern:     "**/personal/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypePII},
			Description: "Personal information",
		},
		{
			Pattern:     "**/customers/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypePII},
			Description: "Customer data",
		},
		{
			Pattern:     "**/users/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypePII},
			Description: "User data",
		},
		{
			Pattern:     "**/employees/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypePII},
			Description: "Employee records",
		},
		{
			Pattern:     "**/hr/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypePII},
			Description: "Human resources data",
		},
		{
			Pattern:     "**/ssn*",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypePII},
			Description: "Social security numbers",
		},

		// Proprietary / Business
		{
			Pattern:     "**/confidential/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypeProprietary},
			Description: "Confidential business data",
		},
		{
			Pattern:     "**/secrets/**",
			Level:       LevelRestricted,
			DataTypes:   []DataType{DataTypeProprietary},
			Description: "Secret data",
		},
		{
			Pattern:     "**/proprietary/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypeProprietary},
			Description: "Proprietary business data",
		},
		{
			Pattern:     "**/financial/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypeProprietary},
			Description: "Financial data",
		},
		{
			Pattern:     "**/contracts/**",
			Level:       LevelConfidential,
			DataTypes:   []DataType{DataTypeProprietary},
			Description: "Contract documents",
		},

		// Internal
		{
			Pattern:     "**/internal/**",
			Level:       LevelInternal,
			DataTypes:   []DataType{DataTypeProprietary},
			Description: "Internal business data",
		},
		{
			Pattern:     "**/docs/**",
			Level:       LevelInternal,
			DataTypes:   []DataType{DataTypeGeneral},
			Description: "Documentation",
		},

		// Public
		{
			Pattern:     "**/public/**",
			Level:       LevelPublic,
			DataTypes:   []DataType{DataTypeGeneral},
			Description: "Publicly accessible data",
		},
		{
			Pattern:     "**/www/**",
			Level:       LevelPublic,
			DataTypes:   []DataType{DataTypeGeneral},
			Description: "Web server public files",
		},
	}
}

// ValidateLevel checks if a level string is valid.
func ValidateLevel(level string) bool {
	switch Level(level) {
	case LevelPublic, LevelInternal, LevelConfidential, LevelRestricted:
		return true
	default:
		return false
	}
}

// ValidateDataType checks if a data type string is valid.
func ValidateDataType(dataType string) bool {
	switch DataType(dataType) {
	case DataTypePII, DataTypePHI, DataTypePCI, DataTypeProprietary, DataTypeGeneral:
		return true
	default:
		return false
	}
}

// LevelDisplayName returns a human-readable name for a level.
func LevelDisplayName(l Level) string {
	switch l {
	case LevelPublic:
		return "Public"
	case LevelInternal:
		return "Internal"
	case LevelConfidential:
		return "Confidential"
	case LevelRestricted:
		return "Restricted"
	default:
		return string(l)
	}
}

// DataTypeDisplayName returns a human-readable name for a data type.
func DataTypeDisplayName(dt DataType) string {
	switch dt {
	case DataTypePII:
		return "PII (Personal Info)"
	case DataTypePHI:
		return "PHI (Health Info)"
	case DataTypePCI:
		return "PCI (Payment Data)"
	case DataTypeProprietary:
		return "Proprietary"
	case DataTypeGeneral:
		return "General"
	default:
		return string(dt)
	}
}
