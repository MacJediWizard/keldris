package models

import (
	"time"

	"github.com/google/uuid"
)

// KomodoIntegrationStatus represents the status of a Komodo integration
type KomodoIntegrationStatus string

const (
	KomodoStatusActive       KomodoIntegrationStatus = "active"
	KomodoStatusDisconnected KomodoIntegrationStatus = "disconnected"
	KomodoStatusError        KomodoIntegrationStatus = "error"
)

// KomodoIntegration represents a connection to a Komodo instance
type KomodoIntegration struct {
	ID              uuid.UUID               `json:"id"`
	OrgID           uuid.UUID               `json:"org_id"`
	Name            string                  `json:"name"`
	URL             string                  `json:"url"`
	ConfigEncrypted []byte                  `json:"-"`
	Status          KomodoIntegrationStatus `json:"status"`
	LastSyncAt      *time.Time              `json:"last_sync_at,omitempty"`
	LastError       string                  `json:"last_error,omitempty"`
	Enabled         bool                    `json:"enabled"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

// NewKomodoIntegration creates a new Komodo integration
func NewKomodoIntegration(orgID uuid.UUID, name, url string, configEncrypted []byte) *KomodoIntegration {
	now := time.Now()
	return &KomodoIntegration{
		ID:              uuid.New(),
		OrgID:           orgID,
		Name:            name,
		URL:             url,
		ConfigEncrypted: configEncrypted,
		Status:          KomodoStatusDisconnected,
		Enabled:         true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// MarkConnected marks the integration as connected
func (k *KomodoIntegration) MarkConnected() {
	now := time.Now()
	k.Status = KomodoStatusActive
	k.LastSyncAt = &now
	k.LastError = ""
	k.UpdatedAt = now
}

// MarkError marks the integration as having an error
func (k *KomodoIntegration) MarkError(errMsg string) {
	k.Status = KomodoStatusError
	k.LastError = errMsg
	k.UpdatedAt = time.Now()
}

// KomodoIntegrationConfig represents the encrypted configuration for a Komodo integration
type KomodoIntegrationConfig struct {
	APIKey   string `json:"api_key"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// KomodoContainerStatus represents the status of a Komodo container
type KomodoContainerStatus string

const (
	KomodoContainerRunning    KomodoContainerStatus = "running"
	KomodoContainerStopped    KomodoContainerStatus = "stopped"
	KomodoContainerRestarting KomodoContainerStatus = "restarting"
	KomodoContainerUnknown    KomodoContainerStatus = "unknown"
)

// KomodoContainer represents a container discovered from Komodo
type KomodoContainer struct {
	ID              uuid.UUID             `json:"id"`
	OrgID           uuid.UUID             `json:"org_id"`
	IntegrationID   uuid.UUID             `json:"integration_id"`
	KomodoID        string                `json:"komodo_id"`
	Name            string                `json:"name"`
	Image           string                `json:"image,omitempty"`
	StackName       string                `json:"stack_name,omitempty"`
	StackID         string                `json:"stack_id,omitempty"`
	Status          KomodoContainerStatus `json:"status"`
	AgentID         *uuid.UUID            `json:"agent_id,omitempty"`
	Volumes         []string              `json:"volumes,omitempty"`
	Labels          map[string]string     `json:"labels,omitempty"`
	BackupEnabled   bool                  `json:"backup_enabled"`
	LastDiscoveredAt time.Time            `json:"last_discovered_at"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

// NewKomodoContainer creates a new Komodo container record
func NewKomodoContainer(orgID, integrationID uuid.UUID, komodoID, name string) *KomodoContainer {
	now := time.Now()
	return &KomodoContainer{
		ID:               uuid.New(),
		OrgID:            orgID,
		IntegrationID:    integrationID,
		KomodoID:         komodoID,
		Name:             name,
		Status:           KomodoContainerUnknown,
		BackupEnabled:    false,
		LastDiscoveredAt: now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// KomodoStack represents a stack discovered from Komodo
type KomodoStack struct {
	ID              uuid.UUID `json:"id"`
	OrgID           uuid.UUID `json:"org_id"`
	IntegrationID   uuid.UUID `json:"integration_id"`
	KomodoID        string    `json:"komodo_id"`
	Name            string    `json:"name"`
	ServerID        string    `json:"server_id,omitempty"`
	ServerName      string    `json:"server_name,omitempty"`
	ContainerCount  int       `json:"container_count"`
	RunningCount    int       `json:"running_count"`
	LastDiscoveredAt time.Time `json:"last_discovered_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewKomodoStack creates a new Komodo stack record
func NewKomodoStack(orgID, integrationID uuid.UUID, komodoID, name string) *KomodoStack {
	now := time.Now()
	return &KomodoStack{
		ID:               uuid.New(),
		OrgID:            orgID,
		IntegrationID:    integrationID,
		KomodoID:         komodoID,
		Name:             name,
		LastDiscoveredAt: now,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// KomodoWebhookEventType represents types of webhook events from Komodo
type KomodoWebhookEventType string

const (
	KomodoWebhookContainerStart   KomodoWebhookEventType = "container.start"
	KomodoWebhookContainerStop    KomodoWebhookEventType = "container.stop"
	KomodoWebhookContainerRestart KomodoWebhookEventType = "container.restart"
	KomodoWebhookStackDeploy      KomodoWebhookEventType = "stack.deploy"
	KomodoWebhookStackUpdate      KomodoWebhookEventType = "stack.update"
	KomodoWebhookBackupTrigger    KomodoWebhookEventType = "backup.trigger"
	KomodoWebhookUnknown          KomodoWebhookEventType = "unknown"
)

// KomodoWebhookEventStatus represents the processing status of a webhook event
type KomodoWebhookEventStatus string

const (
	KomodoWebhookReceived   KomodoWebhookEventStatus = "received"
	KomodoWebhookProcessing KomodoWebhookEventStatus = "processing"
	KomodoWebhookProcessed  KomodoWebhookEventStatus = "processed"
	KomodoWebhookFailed     KomodoWebhookEventStatus = "failed"
)

// KomodoWebhookEvent represents a webhook event received from Komodo
type KomodoWebhookEvent struct {
	ID            uuid.UUID                `json:"id"`
	OrgID         uuid.UUID                `json:"org_id"`
	IntegrationID uuid.UUID                `json:"integration_id"`
	EventType     KomodoWebhookEventType   `json:"event_type"`
	Payload       []byte                   `json:"payload,omitempty"`
	Status        KomodoWebhookEventStatus `json:"status"`
	ErrorMessage  string                   `json:"error_message,omitempty"`
	ProcessedAt   *time.Time               `json:"processed_at,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
}

// NewKomodoWebhookEvent creates a new Komodo webhook event record
func NewKomodoWebhookEvent(orgID, integrationID uuid.UUID, eventType KomodoWebhookEventType, payload []byte) *KomodoWebhookEvent {
	return &KomodoWebhookEvent{
		ID:            uuid.New(),
		OrgID:         orgID,
		IntegrationID: integrationID,
		EventType:     eventType,
		Payload:       payload,
		Status:        KomodoWebhookReceived,
		CreatedAt:     time.Now(),
	}
}

// MarkProcessed marks the webhook event as processed
func (e *KomodoWebhookEvent) MarkProcessed() {
	now := time.Now()
	e.Status = KomodoWebhookProcessed
	e.ProcessedAt = &now
}

// MarkFailed marks the webhook event as failed
func (e *KomodoWebhookEvent) MarkFailed(errMsg string) {
	e.Status = KomodoWebhookFailed
	e.ErrorMessage = errMsg
}

// CreateKomodoIntegrationRequest represents a request to create a Komodo integration
type CreateKomodoIntegrationRequest struct {
	Name   string                  `json:"name" binding:"required,min=1,max=255"`
	URL    string                  `json:"url" binding:"required,url"`
	Config KomodoIntegrationConfig `json:"config" binding:"required"`
}

// UpdateKomodoIntegrationRequest represents a request to update a Komodo integration
type UpdateKomodoIntegrationRequest struct {
	Name    *string                  `json:"name,omitempty"`
	URL     *string                  `json:"url,omitempty"`
	Config  *KomodoIntegrationConfig `json:"config,omitempty"`
	Enabled *bool                    `json:"enabled,omitempty"`
}

// UpdateKomodoContainerRequest represents a request to update a Komodo container
type UpdateKomodoContainerRequest struct {
	AgentID       *uuid.UUID `json:"agent_id,omitempty"`
	BackupEnabled *bool      `json:"backup_enabled,omitempty"`
}

// KomodoDiscoveryResult represents the result of a discovery operation
type KomodoDiscoveryResult struct {
	Stacks          []*KomodoStack     `json:"stacks"`
	Containers      []*KomodoContainer `json:"containers"`
	NewStacks       int                `json:"new_stacks"`
	UpdatedStacks   int                `json:"updated_stacks"`
	NewContainers   int                `json:"new_containers"`
	UpdatedContainers int             `json:"updated_containers"`
	DiscoveredAt    time.Time          `json:"discovered_at"`
}

// KomodoSyncStatus represents the overall sync status for reporting to Komodo
type KomodoSyncStatus struct {
	IntegrationID     uuid.UUID  `json:"integration_id"`
	LastBackupAt      *time.Time `json:"last_backup_at,omitempty"`
	LastBackupStatus  string     `json:"last_backup_status,omitempty"`
	TotalBackups      int        `json:"total_backups"`
	SuccessfulBackups int        `json:"successful_backups"`
	FailedBackups     int        `json:"failed_backups"`
	TotalSizeBytes    int64      `json:"total_size_bytes"`
	NextScheduledAt   *time.Time `json:"next_scheduled_at,omitempty"`
	AgentCount        int        `json:"agent_count"`
	ContainerCount    int        `json:"container_count"`
}
