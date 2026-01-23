// Package backup provides geo-replication functionality for backup repositories.
package backup

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// Common errors for geo-replication operations.
var (
	ErrRegionNotFound       = errors.New("region not found")
	ErrRegionPairNotFound   = errors.New("region pair not found")
	ErrReplicationDisabled  = errors.New("geo-replication is disabled for this repository")
	ErrReplicationInProgress = errors.New("replication already in progress")
)

// Region represents a geographic region for backup storage.
type Region struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

// RegionPair defines a primary-secondary region relationship for geo-replication.
type RegionPair struct {
	Primary   Region `json:"primary"`
	Secondary Region `json:"secondary"`
}

// Predefined regions for geo-replication.
var (
	RegionUSEast1 = Region{
		Code:        "us-east-1",
		Name:        "US East (N. Virginia)",
		DisplayName: "N. Virginia",
		Latitude:    37.4316,
		Longitude:   -78.6569,
	}
	RegionUSWest2 = Region{
		Code:        "us-west-2",
		Name:        "US West (Oregon)",
		DisplayName: "Oregon",
		Latitude:    45.5231,
		Longitude:   -122.6765,
	}
	RegionEUWest1 = Region{
		Code:        "eu-west-1",
		Name:        "EU (Ireland)",
		DisplayName: "Ireland",
		Latitude:    53.3498,
		Longitude:   -6.2603,
	}
	RegionEUCentral1 = Region{
		Code:        "eu-central-1",
		Name:        "EU (Frankfurt)",
		DisplayName: "Frankfurt",
		Latitude:    50.1109,
		Longitude:   8.6821,
	}
	RegionAPSoutheast1 = Region{
		Code:        "ap-southeast-1",
		Name:        "Asia Pacific (Singapore)",
		DisplayName: "Singapore",
		Latitude:    1.3521,
		Longitude:   103.8198,
	}
	RegionAPNortheast1 = Region{
		Code:        "ap-northeast-1",
		Name:        "Asia Pacific (Tokyo)",
		DisplayName: "Tokyo",
		Latitude:    35.6762,
		Longitude:   139.6503,
	}
)

// AllRegions returns all available regions.
func AllRegions() []Region {
	return []Region{
		RegionUSEast1,
		RegionUSWest2,
		RegionEUWest1,
		RegionEUCentral1,
		RegionAPSoutheast1,
		RegionAPNortheast1,
	}
}

// GetRegionByCode returns a region by its code.
func GetRegionByCode(code string) (Region, error) {
	for _, r := range AllRegions() {
		if r.Code == code {
			return r, nil
		}
	}
	return Region{}, ErrRegionNotFound
}

// DefaultRegionPairs returns the default region pairs for geo-replication.
// These pairs are designed for disaster recovery with geographic separation.
func DefaultRegionPairs() []RegionPair {
	return []RegionPair{
		{Primary: RegionUSEast1, Secondary: RegionUSWest2},
		{Primary: RegionUSWest2, Secondary: RegionUSEast1},
		{Primary: RegionEUWest1, Secondary: RegionEUCentral1},
		{Primary: RegionEUCentral1, Secondary: RegionEUWest1},
		{Primary: RegionAPSoutheast1, Secondary: RegionAPNortheast1},
		{Primary: RegionAPNortheast1, Secondary: RegionAPSoutheast1},
	}
}

// GetSecondaryRegion returns the default secondary region for a given primary region.
func GetSecondaryRegion(primaryCode string) (Region, error) {
	for _, pair := range DefaultRegionPairs() {
		if pair.Primary.Code == primaryCode {
			return pair.Secondary, nil
		}
	}
	return Region{}, ErrRegionPairNotFound
}

// ReplicationStatus represents the current status of a geo-replication operation.
type ReplicationStatus string

const (
	ReplicationStatusPending  ReplicationStatus = "pending"
	ReplicationStatusSyncing  ReplicationStatus = "syncing"
	ReplicationStatusSynced   ReplicationStatus = "synced"
	ReplicationStatusFailed   ReplicationStatus = "failed"
	ReplicationStatusDisabled ReplicationStatus = "disabled"
)

// ReplicationLag represents the replication delay metrics.
type ReplicationLag struct {
	SnapshotsBehind int           `json:"snapshots_behind"`
	TimeBehind      time.Duration `json:"time_behind"`
	LastSyncAt      *time.Time    `json:"last_sync_at,omitempty"`
	OldestPending   *time.Time    `json:"oldest_pending,omitempty"`
}

// IsHealthy returns true if the replication lag is within acceptable limits.
func (l *ReplicationLag) IsHealthy(maxSnapshots int, maxDuration time.Duration) bool {
	if l.SnapshotsBehind > maxSnapshots {
		return false
	}
	if l.TimeBehind > maxDuration {
		return false
	}
	return true
}

// GeoReplicationStore defines the interface for geo-replication persistence.
type GeoReplicationStore interface {
	CreateGeoReplicationConfig(ctx context.Context, config *models.GeoReplicationConfig) error
	GetGeoReplicationConfig(ctx context.Context, id uuid.UUID) (*models.GeoReplicationConfig, error)
	GetGeoReplicationConfigByRepository(ctx context.Context, repositoryID uuid.UUID) (*models.GeoReplicationConfig, error)
	UpdateGeoReplicationConfig(ctx context.Context, config *models.GeoReplicationConfig) error
	DeleteGeoReplicationConfig(ctx context.Context, id uuid.UUID) error
	ListGeoReplicationConfigsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.GeoReplicationConfig, error)
	ListPendingReplications(ctx context.Context) ([]*models.GeoReplicationConfig, error)
	RecordReplicationEvent(ctx context.Context, event *models.ReplicationEvent) error
	GetReplicationLag(ctx context.Context, configID uuid.UUID) (*ReplicationLag, error)
}

// GeoReplicator handles automatic geo-replication of backups.
type GeoReplicator struct {
	restic *Restic
	store  GeoReplicationStore
	logger zerolog.Logger

	// Track in-progress replications
	mu           sync.Mutex
	inProgress   map[uuid.UUID]bool
	stopChan     chan struct{}
	wg           sync.WaitGroup
}

// NewGeoReplicator creates a new GeoReplicator.
func NewGeoReplicator(restic *Restic, store GeoReplicationStore, logger zerolog.Logger) *GeoReplicator {
	return &GeoReplicator{
		restic:     restic,
		store:      store,
		logger:     logger.With().Str("component", "geo_replicator").Logger(),
		inProgress: make(map[uuid.UUID]bool),
		stopChan:   make(chan struct{}),
	}
}

// Start begins the background replication processor.
func (g *GeoReplicator) Start(ctx context.Context, checkInterval time.Duration) {
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		g.logger.Info().Dur("interval", checkInterval).Msg("starting geo-replication processor")

		for {
			select {
			case <-ctx.Done():
				g.logger.Info().Msg("geo-replication processor stopped (context canceled)")
				return
			case <-g.stopChan:
				g.logger.Info().Msg("geo-replication processor stopped")
				return
			case <-ticker.C:
				g.processPendingReplications(ctx)
			}
		}
	}()
}

// Stop gracefully stops the replication processor.
func (g *GeoReplicator) Stop() {
	close(g.stopChan)
	g.wg.Wait()
}

// processPendingReplications checks for and processes any pending replications.
func (g *GeoReplicator) processPendingReplications(ctx context.Context) {
	configs, err := g.store.ListPendingReplications(ctx)
	if err != nil {
		g.logger.Error().Err(err).Msg("failed to list pending replications")
		return
	}

	for _, config := range configs {
		if !config.Enabled {
			continue
		}

		// Check if already in progress
		g.mu.Lock()
		if g.inProgress[config.ID] {
			g.mu.Unlock()
			continue
		}
		g.inProgress[config.ID] = true
		g.mu.Unlock()

		// Process in goroutine
		g.wg.Add(1)
		go func(cfg *models.GeoReplicationConfig) {
			defer g.wg.Done()
			defer func() {
				g.mu.Lock()
				delete(g.inProgress, cfg.ID)
				g.mu.Unlock()
			}()

			if err := g.replicateLatestSnapshot(ctx, cfg); err != nil {
				g.logger.Error().
					Err(err).
					Str("config_id", cfg.ID.String()).
					Str("source_repo", cfg.SourceRepositoryID.String()).
					Msg("replication failed")
			}
		}(config)
	}
}

// ReplicateSnapshot copies a specific snapshot from source to target repository.
func (g *GeoReplicator) ReplicateSnapshot(ctx context.Context, config *models.GeoReplicationConfig, snapshotID string) error {
	if !config.Enabled {
		return ErrReplicationDisabled
	}

	// Mark as syncing
	config.Status = string(ReplicationStatusSyncing)
	if err := g.store.UpdateGeoReplicationConfig(ctx, config); err != nil {
		return fmt.Errorf("update config status: %w", err)
	}

	startTime := time.Now()

	// Build source and target configs
	sourceCfg := ResticConfig{
		Repository: config.SourceRepository,
		Password:   config.SourcePassword,
		Env:        config.SourceEnv,
	}

	targetCfg := ResticConfig{
		Repository: config.TargetRepository,
		Password:   config.TargetPassword,
		Env:        config.TargetEnv,
	}

	// Perform the copy
	err := g.restic.Copy(ctx, sourceCfg, targetCfg, snapshotID)

	// Record the event
	event := &models.ReplicationEvent{
		ID:        uuid.New(),
		ConfigID:  config.ID,
		SnapshotID: snapshotID,
		StartedAt: startTime,
		Duration:  time.Since(startTime),
		CreatedAt: time.Now(),
	}

	if err != nil {
		event.Status = string(ReplicationStatusFailed)
		event.ErrorMessage = err.Error()
		config.Status = string(ReplicationStatusFailed)
		config.LastError = err.Error()
	} else {
		event.Status = string(ReplicationStatusSynced)
		now := time.Now()
		config.Status = string(ReplicationStatusSynced)
		config.LastSyncAt = &now
		config.LastSnapshotID = snapshotID
		config.LastError = ""
	}

	if recordErr := g.store.RecordReplicationEvent(ctx, event); recordErr != nil {
		g.logger.Error().Err(recordErr).Msg("failed to record replication event")
	}

	if updateErr := g.store.UpdateGeoReplicationConfig(ctx, config); updateErr != nil {
		g.logger.Error().Err(updateErr).Msg("failed to update config after replication")
	}

	if err != nil {
		return fmt.Errorf("copy snapshot: %w", err)
	}

	g.logger.Info().
		Str("config_id", config.ID.String()).
		Str("snapshot_id", snapshotID).
		Dur("duration", event.Duration).
		Msg("snapshot replicated successfully")

	return nil
}

// replicateLatestSnapshot replicates the latest unreplicated snapshot.
func (g *GeoReplicator) replicateLatestSnapshot(ctx context.Context, config *models.GeoReplicationConfig) error {
	// Get snapshots from source
	sourceCfg := ResticConfig{
		Repository: config.SourceRepository,
		Password:   config.SourcePassword,
		Env:        config.SourceEnv,
	}

	snapshots, err := g.restic.Snapshots(ctx, sourceCfg)
	if err != nil {
		return fmt.Errorf("list source snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return nil // No snapshots to replicate
	}

	// Find the latest snapshot that hasn't been replicated
	var latestUnreplicated *Snapshot
	for i := len(snapshots) - 1; i >= 0; i-- {
		snap := snapshots[i]
		if config.LastSnapshotID == "" || snap.ID != config.LastSnapshotID {
			// Check if this is newer than the last replicated
			if latestUnreplicated == nil || snap.Time.After(latestUnreplicated.Time) {
				latestUnreplicated = &snap
			}
		}
	}

	if latestUnreplicated == nil {
		return nil // All snapshots already replicated
	}

	// If the last replicated snapshot is the same as the latest, skip
	if config.LastSnapshotID == latestUnreplicated.ID {
		return nil
	}

	return g.ReplicateSnapshot(ctx, config, latestUnreplicated.ID)
}

// TriggerReplication manually triggers replication for a repository.
func (g *GeoReplicator) TriggerReplication(ctx context.Context, repositoryID uuid.UUID) error {
	config, err := g.store.GetGeoReplicationConfigByRepository(ctx, repositoryID)
	if err != nil {
		return fmt.Errorf("get replication config: %w", err)
	}

	if !config.Enabled {
		return ErrReplicationDisabled
	}

	// Check if already in progress
	g.mu.Lock()
	if g.inProgress[config.ID] {
		g.mu.Unlock()
		return ErrReplicationInProgress
	}
	g.inProgress[config.ID] = true
	g.mu.Unlock()

	defer func() {
		g.mu.Lock()
		delete(g.inProgress, config.ID)
		g.mu.Unlock()
	}()

	return g.replicateLatestSnapshot(ctx, config)
}

// GetReplicationStatus returns the current replication status for a repository.
func (g *GeoReplicator) GetReplicationStatus(ctx context.Context, repositoryID uuid.UUID) (*models.GeoReplicationConfig, *ReplicationLag, error) {
	config, err := g.store.GetGeoReplicationConfigByRepository(ctx, repositoryID)
	if err != nil {
		return nil, nil, err
	}

	lag, err := g.store.GetReplicationLag(ctx, config.ID)
	if err != nil {
		return config, nil, err
	}

	return config, lag, nil
}

// CheckReplicationHealth checks if replication is within acceptable limits and creates alerts if not.
func (g *GeoReplicator) CheckReplicationHealth(ctx context.Context, config *models.GeoReplicationConfig, maxSnapshots int, maxDuration time.Duration) (bool, *ReplicationLag, error) {
	lag, err := g.store.GetReplicationLag(ctx, config.ID)
	if err != nil {
		return false, nil, err
	}

	healthy := lag.IsHealthy(maxSnapshots, maxDuration)
	return healthy, lag, nil
}

// ReplicationSummary provides a summary of replication status across all configs.
type ReplicationSummary struct {
	TotalConfigs    int                       `json:"total_configs"`
	EnabledConfigs  int                       `json:"enabled_configs"`
	SyncedCount     int                       `json:"synced_count"`
	SyncingCount    int                       `json:"syncing_count"`
	PendingCount    int                       `json:"pending_count"`
	FailedCount     int                       `json:"failed_count"`
	Configs         []*models.GeoReplicationConfig `json:"configs,omitempty"`
}

// GetReplicationSummary returns a summary of all replication configs for an org.
func (g *GeoReplicator) GetReplicationSummary(ctx context.Context, orgID uuid.UUID) (*ReplicationSummary, error) {
	configs, err := g.store.ListGeoReplicationConfigsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	summary := &ReplicationSummary{
		TotalConfigs: len(configs),
		Configs:      configs,
	}

	for _, cfg := range configs {
		if cfg.Enabled {
			summary.EnabledConfigs++
		}
		switch ReplicationStatus(cfg.Status) {
		case ReplicationStatusSynced:
			summary.SyncedCount++
		case ReplicationStatusSyncing:
			summary.SyncingCount++
		case ReplicationStatusPending:
			summary.PendingCount++
		case ReplicationStatusFailed:
			summary.FailedCount++
		}
	}

	return summary, nil
}
