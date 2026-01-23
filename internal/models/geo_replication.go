package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// GeoReplicationStatus represents the current status of geo-replication.
type GeoReplicationStatus string

const (
	GeoReplicationStatusPending  GeoReplicationStatus = "pending"
	GeoReplicationStatusSyncing  GeoReplicationStatus = "syncing"
	GeoReplicationStatusSynced   GeoReplicationStatus = "synced"
	GeoReplicationStatusFailed   GeoReplicationStatus = "failed"
	GeoReplicationStatusDisabled GeoReplicationStatus = "disabled"
)

// GeoReplicationConfig represents the geo-replication configuration for a repository.
type GeoReplicationConfig struct {
	ID                   uuid.UUID  `json:"id"`
	OrgID                uuid.UUID  `json:"org_id"`
	SourceRepositoryID   uuid.UUID  `json:"source_repository_id"`
	TargetRepositoryID   uuid.UUID  `json:"target_repository_id"`
	SourceRegion         string     `json:"source_region"`
	TargetRegion         string     `json:"target_region"`
	Enabled              bool       `json:"enabled"`
	Status               string     `json:"status"`
	LastSnapshotID       string     `json:"last_snapshot_id,omitempty"`
	LastSyncAt           *time.Time `json:"last_sync_at,omitempty"`
	LastError            string     `json:"last_error,omitempty"`
	MaxLagSnapshots      int        `json:"max_lag_snapshots"`
	MaxLagDurationHours  int        `json:"max_lag_duration_hours"`
	AlertOnLag           bool       `json:"alert_on_lag"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`

	// Non-persisted fields for runtime use
	SourceRepository string            `json:"-"`
	SourcePassword   string            `json:"-"`
	SourceEnv        map[string]string `json:"-"`
	TargetRepository string            `json:"-"`
	TargetPassword   string            `json:"-"`
	TargetEnv        map[string]string `json:"-"`
}

// NewGeoReplicationConfig creates a new GeoReplicationConfig.
func NewGeoReplicationConfig(
	orgID, sourceRepoID, targetRepoID uuid.UUID,
	sourceRegion, targetRegion string,
) *GeoReplicationConfig {
	now := time.Now()
	return &GeoReplicationConfig{
		ID:                  uuid.New(),
		OrgID:               orgID,
		SourceRepositoryID:  sourceRepoID,
		TargetRepositoryID:  targetRepoID,
		SourceRegion:        sourceRegion,
		TargetRegion:        targetRegion,
		Enabled:             true,
		Status:              string(GeoReplicationStatusPending),
		MaxLagSnapshots:     5,
		MaxLagDurationHours: 24,
		AlertOnLag:          true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

// SetEnabled enables or disables geo-replication.
func (g *GeoReplicationConfig) SetEnabled(enabled bool) {
	g.Enabled = enabled
	g.UpdatedAt = time.Now()
	if !enabled {
		g.Status = string(GeoReplicationStatusDisabled)
	} else if g.Status == string(GeoReplicationStatusDisabled) {
		g.Status = string(GeoReplicationStatusPending)
	}
}

// RecordSync records a successful sync.
func (g *GeoReplicationConfig) RecordSync(snapshotID string) {
	now := time.Now()
	g.LastSnapshotID = snapshotID
	g.LastSyncAt = &now
	g.Status = string(GeoReplicationStatusSynced)
	g.LastError = ""
	g.UpdatedAt = now
}

// RecordError records a replication error.
func (g *GeoReplicationConfig) RecordError(err string) {
	g.Status = string(GeoReplicationStatusFailed)
	g.LastError = err
	g.UpdatedAt = time.Now()
}

// ReplicationEvent records a single replication operation.
type ReplicationEvent struct {
	ID           uuid.UUID     `json:"id"`
	ConfigID     uuid.UUID     `json:"config_id"`
	SnapshotID   string        `json:"snapshot_id"`
	Status       string        `json:"status"`
	StartedAt    time.Time     `json:"started_at"`
	CompletedAt  *time.Time    `json:"completed_at,omitempty"`
	Duration     time.Duration `json:"duration"`
	BytesCopied  int64         `json:"bytes_copied,omitempty"`
	ErrorMessage string        `json:"error_message,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
}

// NewReplicationEvent creates a new ReplicationEvent.
func NewReplicationEvent(configID uuid.UUID, snapshotID string) *ReplicationEvent {
	now := time.Now()
	return &ReplicationEvent{
		ID:         uuid.New(),
		ConfigID:   configID,
		SnapshotID: snapshotID,
		Status:     string(GeoReplicationStatusSyncing),
		StartedAt:  now,
		CreatedAt:  now,
	}
}

// Complete marks the event as completed successfully.
func (e *ReplicationEvent) Complete(bytesCopied int64) {
	now := time.Now()
	e.CompletedAt = &now
	e.Status = string(GeoReplicationStatusSynced)
	e.Duration = now.Sub(e.StartedAt)
	e.BytesCopied = bytesCopied
}

// Fail marks the event as failed.
func (e *ReplicationEvent) Fail(err string) {
	now := time.Now()
	e.CompletedAt = &now
	e.Status = string(GeoReplicationStatusFailed)
	e.Duration = now.Sub(e.StartedAt)
	e.ErrorMessage = err
}

// GeoRegion represents a geographic region for display purposes.
type GeoRegion struct {
	Code        string  `json:"code"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// GeoReplicationResponse is the API response for geo-replication config.
type GeoReplicationResponse struct {
	ID                  uuid.UUID  `json:"id"`
	SourceRepositoryID  uuid.UUID  `json:"source_repository_id"`
	TargetRepositoryID  uuid.UUID  `json:"target_repository_id"`
	SourceRegion        GeoRegion  `json:"source_region"`
	TargetRegion        GeoRegion  `json:"target_region"`
	Enabled             bool       `json:"enabled"`
	Status              string     `json:"status"`
	LastSnapshotID      string     `json:"last_snapshot_id,omitempty"`
	LastSyncAt          *time.Time `json:"last_sync_at,omitempty"`
	LastError           string     `json:"last_error,omitempty"`
	MaxLagSnapshots     int        `json:"max_lag_snapshots"`
	MaxLagDurationHours int        `json:"max_lag_duration_hours"`
	AlertOnLag          bool       `json:"alert_on_lag"`
	ReplicationLag      *ReplicationLagResponse `json:"replication_lag,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// ReplicationLagResponse provides lag information in the API response.
type ReplicationLagResponse struct {
	SnapshotsBehind int    `json:"snapshots_behind"`
	TimeBehindHours int    `json:"time_behind_hours"`
	IsHealthy       bool   `json:"is_healthy"`
	LastSyncAt      string `json:"last_sync_at,omitempty"`
}

// GeoReplicationCreateRequest is the request body for creating geo-replication.
type GeoReplicationCreateRequest struct {
	SourceRepositoryID  uuid.UUID `json:"source_repository_id" binding:"required"`
	TargetRepositoryID  uuid.UUID `json:"target_repository_id" binding:"required"`
	SourceRegion        string    `json:"source_region" binding:"required"`
	TargetRegion        string    `json:"target_region" binding:"required"`
	MaxLagSnapshots     *int      `json:"max_lag_snapshots,omitempty"`
	MaxLagDurationHours *int      `json:"max_lag_duration_hours,omitempty"`
	AlertOnLag          *bool     `json:"alert_on_lag,omitempty"`
}

// GeoReplicationUpdateRequest is the request body for updating geo-replication.
type GeoReplicationUpdateRequest struct {
	Enabled             *bool  `json:"enabled,omitempty"`
	MaxLagSnapshots     *int   `json:"max_lag_snapshots,omitempty"`
	MaxLagDurationHours *int   `json:"max_lag_duration_hours,omitempty"`
	AlertOnLag          *bool  `json:"alert_on_lag,omitempty"`
}

// SetMetadata sets the metadata from JSON bytes (for compatibility with other models).
func (g *GeoReplicationConfig) SetEnvFromJSON(sourceEnv, targetEnv []byte) error {
	if len(sourceEnv) > 0 {
		if err := json.Unmarshal(sourceEnv, &g.SourceEnv); err != nil {
			return err
		}
	}
	if len(targetEnv) > 0 {
		if err := json.Unmarshal(targetEnv, &g.TargetEnv); err != nil {
			return err
		}
	}
	return nil
}
