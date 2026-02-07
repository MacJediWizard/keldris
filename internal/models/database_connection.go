package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DatabaseType defines the type of database.
type DatabaseType string

const (
	// DatabaseTypeMySQL is MySQL database.
	DatabaseTypeMySQL DatabaseType = "mysql"
	// DatabaseTypeMariaDB is MariaDB database.
	DatabaseTypeMariaDB DatabaseType = "mariadb"
)

// DatabaseConnectionHealthStatus defines the health status of a database connection.
type DatabaseConnectionHealthStatus string

const (
	// DatabaseConnectionHealthHealthy indicates the connection is working.
	DatabaseConnectionHealthHealthy DatabaseConnectionHealthStatus = "healthy"
	// DatabaseConnectionHealthUnhealthy indicates the connection is not working.
	DatabaseConnectionHealthUnhealthy DatabaseConnectionHealthStatus = "unhealthy"
	// DatabaseConnectionHealthUnknown indicates the health is not yet checked.
	DatabaseConnectionHealthUnknown DatabaseConnectionHealthStatus = "unknown"
)

// DatabaseConnection represents a database connection configuration.
type DatabaseConnection struct {
	ID                   uuid.UUID                      `json:"id"`
	OrgID                uuid.UUID                      `json:"org_id"`
	AgentID              *uuid.UUID                     `json:"agent_id,omitempty"` // If set, connection is agent-specific
	Name                 string                         `json:"name"`
	Type                 DatabaseType                   `json:"type"`
	Host                 string                         `json:"host"`
	Port                 int                            `json:"port"`
	Username             string                         `json:"username"`
	CredentialsEncrypted []byte                         `json:"-"` // Encrypted password, never expose in JSON
	SSLMode              string                         `json:"ssl_mode,omitempty"`
	Enabled              bool                           `json:"enabled"`
	HealthStatus         DatabaseConnectionHealthStatus `json:"health_status"`
	LastHealthCheck      *time.Time                     `json:"last_health_check,omitempty"`
	LastHealthError      *string                        `json:"last_health_error,omitempty"`
	Version              *string                        `json:"version,omitempty"` // Detected database version
	Metadata             map[string]interface{}         `json:"metadata,omitempty"`
	CreatedAt            time.Time                      `json:"created_at"`
	UpdatedAt            time.Time                      `json:"updated_at"`
	CreatedBy            *uuid.UUID                     `json:"created_by,omitempty"`
}

// DatabaseCredentials holds the credentials for a database connection.
// This is stored encrypted in the database.
type DatabaseCredentials struct {
	Password string `json:"password"`
}

// NewDatabaseConnection creates a new DatabaseConnection with the given details.
func NewDatabaseConnection(orgID uuid.UUID, name string, dbType DatabaseType, host string, port int, username string, credentialsEncrypted []byte) *DatabaseConnection {
	now := time.Now()
	return &DatabaseConnection{
		ID:                   uuid.New(),
		OrgID:                orgID,
		Name:                 name,
		Type:                 dbType,
		Host:                 host,
		Port:                 port,
		Username:             username,
		CredentialsEncrypted: credentialsEncrypted,
		Enabled:              true,
		HealthStatus:         DatabaseConnectionHealthUnknown,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

// ValidDatabaseTypes returns all valid database types.
func ValidDatabaseTypes() []DatabaseType {
	return []DatabaseType{
		DatabaseTypeMySQL,
		DatabaseTypeMariaDB,
	}
}

// IsValidType checks if the database type is valid.
func (c *DatabaseConnection) IsValidType() bool {
	for _, t := range ValidDatabaseTypes() {
		if c.Type == t {
			return true
		}
	}
	return false
}

// DefaultPort returns the default port for a given database type.
func DefaultPort(dbType DatabaseType) int {
	switch dbType {
	case DatabaseTypeMySQL, DatabaseTypeMariaDB:
		return 3306
	default:
		return 3306
	}
}

// SetMetadata sets the metadata from JSON bytes.
func (c *DatabaseConnection) SetMetadata(data []byte) error {
	if len(data) == 0 {
		c.Metadata = make(map[string]interface{})
		return nil
	}
	return json.Unmarshal(data, &c.Metadata)
}

// MetadataJSON returns the metadata as JSON bytes for database storage.
func (c *DatabaseConnection) MetadataJSON() ([]byte, error) {
	if c.Metadata == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(c.Metadata)
}

// UpdateHealth updates the health status of the connection.
func (c *DatabaseConnection) UpdateHealth(status DatabaseConnectionHealthStatus, version *string, errorMsg *string) {
	now := time.Now()
	c.HealthStatus = status
	c.LastHealthCheck = &now
	c.LastHealthError = errorMsg
	if version != nil {
		c.Version = version
	}
	c.UpdatedAt = now
}

// MarkHealthy marks the connection as healthy with version info.
func (c *DatabaseConnection) MarkHealthy(version string) {
	c.UpdateHealth(DatabaseConnectionHealthHealthy, &version, nil)
}

// MarkUnhealthy marks the connection as unhealthy with an error message.
func (c *DatabaseConnection) MarkUnhealthy(errorMsg string) {
	c.UpdateHealth(DatabaseConnectionHealthUnhealthy, nil, &errorMsg)
}

// DatabaseConnectionTestResult contains the result of a connection test.
type DatabaseConnectionTestResult struct {
	Success      bool          `json:"success"`
	ConnectionID uuid.UUID     `json:"connection_id"`
	Version      string        `json:"version,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	Databases    []string      `json:"databases,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// DatabaseConnectionList represents a list response with pagination.
type DatabaseConnectionList struct {
	Connections []DatabaseConnection `json:"connections"`
	Total       int                  `json:"total"`
	Page        int                  `json:"page"`
	PageSize    int                  `json:"page_size"`
}
