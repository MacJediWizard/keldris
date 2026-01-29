// Package komodo provides integration with the Komodo container management platform.
package komodo

import "time"

// APIStack represents a stack returned from the Komodo API
type APIStack struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	ServerID      string            `json:"server_id,omitempty"`
	ServerName    string            `json:"server_name,omitempty"`
	State         string            `json:"state,omitempty"`
	Status        string            `json:"status,omitempty"`
	ContainerCount int              `json:"container_count,omitempty"`
	RunningCount  int               `json:"running_count,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
	CreatedAt     *time.Time        `json:"created_at,omitempty"`
	UpdatedAt     *time.Time        `json:"updated_at,omitempty"`
}

// APIContainer represents a container returned from the Komodo API
type APIContainer struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Image       string            `json:"image,omitempty"`
	ImageID     string            `json:"image_id,omitempty"`
	State       string            `json:"state,omitempty"`
	Status      string            `json:"status,omitempty"`
	StackID     string            `json:"stack_id,omitempty"`
	StackName   string            `json:"stack_name,omitempty"`
	ServerID    string            `json:"server_id,omitempty"`
	ServerName  string            `json:"server_name,omitempty"`
	Volumes     []APIVolume       `json:"volumes,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Ports       []APIPort         `json:"ports,omitempty"`
	Networks    []string          `json:"networks,omitempty"`
	CreatedAt   *time.Time        `json:"created_at,omitempty"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
}

// APIVolume represents a volume mount in a container
type APIVolume struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode,omitempty"`
	RW          bool   `json:"rw,omitempty"`
}

// APIPort represents a port mapping in a container
type APIPort struct {
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      int    `json:"host_port,omitempty"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"`
}

// APIServer represents a server/host in Komodo
type APIServer struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Hostname    string     `json:"hostname,omitempty"`
	Address     string     `json:"address,omitempty"`
	Status      string     `json:"status,omitempty"`
	AgentStatus string     `json:"agent_status,omitempty"`
	OS          string     `json:"os,omitempty"`
	Arch        string     `json:"arch,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// ListStacksResponse represents the response from listing stacks
type ListStacksResponse struct {
	Stacks     []APIStack `json:"stacks"`
	Total      int        `json:"total,omitempty"`
	Page       int        `json:"page,omitempty"`
	PerPage    int        `json:"per_page,omitempty"`
	TotalPages int        `json:"total_pages,omitempty"`
}

// ListContainersResponse represents the response from listing containers
type ListContainersResponse struct {
	Containers []APIContainer `json:"containers"`
	Total      int            `json:"total,omitempty"`
	Page       int            `json:"page,omitempty"`
	PerPage    int            `json:"per_page,omitempty"`
	TotalPages int            `json:"total_pages,omitempty"`
}

// ListServersResponse represents the response from listing servers
type ListServersResponse struct {
	Servers    []APIServer `json:"servers"`
	Total      int         `json:"total,omitempty"`
	Page       int         `json:"page,omitempty"`
	PerPage    int         `json:"per_page,omitempty"`
	TotalPages int         `json:"total_pages,omitempty"`
}

// WebhookPayload represents an incoming webhook payload from Komodo
type WebhookPayload struct {
	Event      string                 `json:"event"`
	Timestamp  time.Time              `json:"timestamp"`
	Source     string                 `json:"source,omitempty"`
	ServerID   string                 `json:"server_id,omitempty"`
	ServerName string                 `json:"server_name,omitempty"`
	StackID    string                 `json:"stack_id,omitempty"`
	StackName  string                 `json:"stack_name,omitempty"`
	ContainerID   string              `json:"container_id,omitempty"`
	ContainerName string              `json:"container_name,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// WebhookBackupTrigger represents a backup trigger request from Komodo
type WebhookBackupTrigger struct {
	ContainerID   string   `json:"container_id,omitempty"`
	ContainerName string   `json:"container_name,omitempty"`
	StackID       string   `json:"stack_id,omitempty"`
	StackName     string   `json:"stack_name,omitempty"`
	ServerID      string   `json:"server_id,omitempty"`
	Paths         []string `json:"paths,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Priority      int      `json:"priority,omitempty"`
}

// StatusUpdateRequest represents a status update to send to Komodo
type StatusUpdateRequest struct {
	ContainerID     string     `json:"container_id,omitempty"`
	StackID         string     `json:"stack_id,omitempty"`
	BackupStatus    string     `json:"backup_status"`
	LastBackupAt    *time.Time `json:"last_backup_at,omitempty"`
	LastBackupSize  int64      `json:"last_backup_size,omitempty"`
	NextScheduledAt *time.Time `json:"next_scheduled_at,omitempty"`
	SnapshotCount   int        `json:"snapshot_count,omitempty"`
	TotalSize       int64      `json:"total_size,omitempty"`
	ErrorMessage    string     `json:"error_message,omitempty"`
}

// StatusUpdateResponse represents the response from a status update
type StatusUpdateResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// APIError represents an error response from the Komodo API
type APIError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Error implements the error interface for APIError
func (e *APIError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// HealthResponse represents the health check response from Komodo
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
}

// AuthResponse represents the authentication response from Komodo
type AuthResponse struct {
	Token     string     `json:"token,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	User      *APIUser   `json:"user,omitempty"`
}

// APIUser represents a user in Komodo
type APIUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Role     string `json:"role,omitempty"`
}
