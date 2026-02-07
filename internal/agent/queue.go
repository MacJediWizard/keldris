// Package agent provides agent-side functionality for the Keldris backup agent.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// QueuedBackupStatus represents the status of a queued backup.
type QueuedBackupStatus string

const (
	// QueuedBackupStatusPending indicates the backup is waiting to be synced.
	QueuedBackupStatusPending QueuedBackupStatus = "pending"
	// QueuedBackupStatusSyncing indicates the backup is being synced to server.
	QueuedBackupStatusSyncing QueuedBackupStatus = "syncing"
	// QueuedBackupStatusSynced indicates the backup was successfully synced.
	QueuedBackupStatusSynced QueuedBackupStatus = "synced"
	// QueuedBackupStatusFailed indicates the sync failed.
	QueuedBackupStatusFailed QueuedBackupStatus = "failed"
)

// QueuedBackup represents a backup that was queued while offline.
type QueuedBackup struct {
	ID           uuid.UUID          `json:"id"`
	ScheduleID   uuid.UUID          `json:"schedule_id"`
	ScheduleName string             `json:"schedule_name"`
	ScheduledAt  time.Time          `json:"scheduled_at"`
	QueuedAt     time.Time          `json:"queued_at"`
	Status       QueuedBackupStatus `json:"status"`
	RetryCount   int                `json:"retry_count"`
	LastError    string             `json:"last_error,omitempty"`
	SyncedAt     *time.Time         `json:"synced_at,omitempty"`
	BackupResult *BackupResult      `json:"backup_result,omitempty"`
}

// BackupResult contains the result of a completed backup.
type BackupResult struct {
	Success       bool      `json:"success"`
	StartedAt     time.Time `json:"started_at"`
	CompletedAt   time.Time `json:"completed_at"`
	BytesAdded    int64     `json:"bytes_added,omitempty"`
	FilesNew      int       `json:"files_new,omitempty"`
	FilesChanged  int       `json:"files_changed,omitempty"`
	SnapshotID    string    `json:"snapshot_id,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	RepositoryID  uuid.UUID `json:"repository_id,omitempty"`
}

// QueueStatus represents the current state of the backup queue.
type QueueStatus struct {
	TotalQueued     int        `json:"total_queued"`
	PendingCount    int        `json:"pending_count"`
	SyncedCount     int        `json:"synced_count"`
	FailedCount     int        `json:"failed_count"`
	OldestQueuedAt  *time.Time `json:"oldest_queued_at,omitempty"`
	LastSyncAttempt *time.Time `json:"last_sync_attempt,omitempty"`
	LastSuccessSync *time.Time `json:"last_success_sync,omitempty"`
	ServerReachable bool       `json:"server_reachable"`
	MaxQueueSize    int        `json:"max_queue_size"`
}

// QueueStore defines the interface for queue persistence operations.
type QueueStore interface {
	// CreateQueuedBackup stores a new queued backup.
	CreateQueuedBackup(ctx context.Context, backup *QueuedBackup) error
	// GetQueuedBackup retrieves a queued backup by ID.
	GetQueuedBackup(ctx context.Context, id uuid.UUID) (*QueuedBackup, error)
	// UpdateQueuedBackup updates an existing queued backup.
	UpdateQueuedBackup(ctx context.Context, backup *QueuedBackup) error
	// DeleteQueuedBackup removes a queued backup.
	DeleteQueuedBackup(ctx context.Context, id uuid.UUID) error
	// ListPendingBackups returns all pending backups ordered by scheduled time.
	ListPendingBackups(ctx context.Context) ([]*QueuedBackup, error)
	// ListAllBackups returns all backups in the queue.
	ListAllBackups(ctx context.Context) ([]*QueuedBackup, error)
	// GetQueueCount returns the total number of entries in the queue.
	GetQueueCount(ctx context.Context) (int, error)
	// GetQueueStatus returns aggregate queue statistics.
	GetQueueStatus(ctx context.Context) (*QueueStatus, error)
	// PruneOldEntries removes synced/failed entries older than the given duration.
	PruneOldEntries(ctx context.Context, olderThan time.Duration) (int, error)
	// Close closes the store connection.
	Close() error
}

// ServerClient defines the interface for server communication.
type ServerClient interface {
	// CheckHealth checks if the server is reachable.
	CheckHealth(ctx context.Context) error
	// ReportQueuedBackups sends queued backup results to the server.
	ReportQueuedBackups(ctx context.Context, backups []*QueuedBackup) error
	// NotifyReconnection alerts the server about reconnection with queued items.
	NotifyReconnection(ctx context.Context, queuedCount int) error
}

// QueueConfig holds configuration for the backup queue.
type QueueConfig struct {
	MaxQueueSize      int           `yaml:"max_queue_size"`
	SyncInterval      time.Duration `yaml:"sync_interval"`
	HealthCheckPeriod time.Duration `yaml:"health_check_period"`
	RetryBackoff      time.Duration `yaml:"retry_backoff"`
	MaxRetries        int           `yaml:"max_retries"`
	PruneAge          time.Duration `yaml:"prune_age"`
}

// DefaultQueueConfig returns sensible default configuration.
func DefaultQueueConfig() QueueConfig {
	return QueueConfig{
		MaxQueueSize:      100,
		SyncInterval:      30 * time.Second,
		HealthCheckPeriod: 10 * time.Second,
		RetryBackoff:      5 * time.Second,
		MaxRetries:        5,
		PruneAge:          7 * 24 * time.Hour, // 7 days
	}
}

// Queue manages offline backup queuing and synchronization.
type Queue struct {
	store  QueueStore
	client ServerClient
	config QueueConfig
	logger zerolog.Logger

	mu              sync.RWMutex
	serverReachable bool
	lastHealthCheck time.Time
	lastSyncAttempt time.Time
	lastSuccessSync time.Time

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewQueue creates a new backup queue manager.
func NewQueue(store QueueStore, client ServerClient, config QueueConfig, logger zerolog.Logger) *Queue {
	return &Queue{
		store:  store,
		client: client,
		config: config,
		logger: logger.With().Str("component", "backup_queue").Logger(),
		stopCh: make(chan struct{}),
	}
}

// Start begins the background queue processing and health monitoring.
func (q *Queue) Start(ctx context.Context) error {
	// Initial health check
	if err := q.checkServerHealth(ctx); err != nil {
		q.logger.Warn().Err(err).Msg("initial server health check failed, starting in offline mode")
	}

	// Start background workers
	q.wg.Add(2)
	go q.healthCheckLoop()
	go q.syncLoop()

	q.logger.Info().
		Int("max_queue_size", q.config.MaxQueueSize).
		Dur("sync_interval", q.config.SyncInterval).
		Msg("backup queue started")

	return nil
}

// Stop gracefully stops the queue processing.
func (q *Queue) Stop() {
	close(q.stopCh)
	q.wg.Wait()
	q.logger.Info().Msg("backup queue stopped")
}

// QueueBackup adds a backup to the local queue when the server is unreachable.
func (q *Queue) QueueBackup(ctx context.Context, scheduleID uuid.UUID, scheduleName string, scheduledAt time.Time) (*QueuedBackup, error) {
	// Check queue size limit
	count, err := q.store.GetQueueCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("get queue count: %w", err)
	}

	if count >= q.config.MaxQueueSize {
		return nil, ErrQueueFull
	}

	backup := &QueuedBackup{
		ID:           uuid.New(),
		ScheduleID:   scheduleID,
		ScheduleName: scheduleName,
		ScheduledAt:  scheduledAt,
		QueuedAt:     time.Now(),
		Status:       QueuedBackupStatusPending,
		RetryCount:   0,
	}

	if err := q.store.CreateQueuedBackup(ctx, backup); err != nil {
		return nil, fmt.Errorf("create queued backup: %w", err)
	}

	q.logger.Info().
		Str("backup_id", backup.ID.String()).
		Str("schedule_id", scheduleID.String()).
		Str("schedule_name", scheduleName).
		Time("scheduled_at", scheduledAt).
		Msg("backup queued for offline execution")

	return backup, nil
}

// RecordBackupResult records the result of a backup that was executed while offline.
func (q *Queue) RecordBackupResult(ctx context.Context, backupID uuid.UUID, result *BackupResult) error {
	backup, err := q.store.GetQueuedBackup(ctx, backupID)
	if err != nil {
		return fmt.Errorf("get queued backup: %w", err)
	}

	backup.BackupResult = result

	if err := q.store.UpdateQueuedBackup(ctx, backup); err != nil {
		return fmt.Errorf("update queued backup: %w", err)
	}

	q.logger.Info().
		Str("backup_id", backupID.String()).
		Bool("success", result.Success).
		Msg("backup result recorded")

	return nil
}

// IsServerReachable returns true if the server is currently reachable.
func (q *Queue) IsServerReachable() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.serverReachable
}

// GetStatus returns the current queue status.
func (q *Queue) GetStatus(ctx context.Context) (*QueueStatus, error) {
	status, err := q.store.GetQueueStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get queue status: %w", err)
	}

	q.mu.RLock()
	status.ServerReachable = q.serverReachable
	if !q.lastSyncAttempt.IsZero() {
		status.LastSyncAttempt = &q.lastSyncAttempt
	}
	if !q.lastSuccessSync.IsZero() {
		status.LastSuccessSync = &q.lastSuccessSync
	}
	q.mu.RUnlock()

	status.MaxQueueSize = q.config.MaxQueueSize

	return status, nil
}

// SyncNow triggers an immediate sync attempt.
func (q *Queue) SyncNow(ctx context.Context) error {
	return q.syncPendingBackups(ctx)
}

// checkServerHealth checks if the server is reachable.
func (q *Queue) checkServerHealth(ctx context.Context) error {
	err := q.client.CheckHealth(ctx)

	q.mu.Lock()
	wasReachable := q.serverReachable
	q.serverReachable = err == nil
	q.lastHealthCheck = time.Now()
	q.mu.Unlock()

	if err != nil {
		q.logger.Debug().Err(err).Msg("server health check failed")
		return err
	}

	// If we just reconnected, check for queued backups
	if !wasReachable && q.serverReachable {
		q.handleReconnection(ctx)
	}

	return nil
}

// handleReconnection handles the case where the server becomes reachable after being offline.
func (q *Queue) handleReconnection(ctx context.Context) {
	q.logger.Info().Msg("server connection restored")

	// Get count of pending backups
	count, err := q.store.GetQueueCount(ctx)
	if err != nil {
		q.logger.Warn().Err(err).Msg("failed to get queue count on reconnection")
		return
	}

	if count == 0 {
		return
	}

	q.logger.Info().
		Int("queued_count", count).
		Msg("reconnected with queued backups, notifying server")

	// Notify server about queued backups
	if err := q.client.NotifyReconnection(ctx, count); err != nil {
		q.logger.Warn().Err(err).Msg("failed to notify server of reconnection")
	}

	// Trigger immediate sync
	go func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		if err := q.syncPendingBackups(syncCtx); err != nil {
			q.logger.Warn().Err(err).Msg("sync after reconnection failed")
		}
	}()
}

// syncPendingBackups attempts to sync all pending backups to the server.
func (q *Queue) syncPendingBackups(ctx context.Context) error {
	if !q.IsServerReachable() {
		return ErrServerUnreachable
	}

	q.mu.Lock()
	q.lastSyncAttempt = time.Now()
	q.mu.Unlock()

	backups, err := q.store.ListPendingBackups(ctx)
	if err != nil {
		return fmt.Errorf("list pending backups: %w", err)
	}

	if len(backups) == 0 {
		return nil
	}

	q.logger.Info().
		Int("backup_count", len(backups)).
		Msg("syncing queued backups to server")

	// Mark all as syncing
	for _, backup := range backups {
		backup.Status = QueuedBackupStatusSyncing
		if err := q.store.UpdateQueuedBackup(ctx, backup); err != nil {
			q.logger.Warn().Err(err).Str("backup_id", backup.ID.String()).Msg("failed to mark backup as syncing")
		}
	}

	// Report to server
	if err := q.client.ReportQueuedBackups(ctx, backups); err != nil {
		// Mark all as failed and increment retry count
		for _, backup := range backups {
			backup.Status = QueuedBackupStatusFailed
			backup.RetryCount++
			backup.LastError = err.Error()

			// If max retries reached, leave as failed; otherwise, reset to pending
			if backup.RetryCount < q.config.MaxRetries {
				backup.Status = QueuedBackupStatusPending
			}

			if updateErr := q.store.UpdateQueuedBackup(ctx, backup); updateErr != nil {
				q.logger.Warn().Err(updateErr).Str("backup_id", backup.ID.String()).Msg("failed to update backup after sync failure")
			}
		}
		return fmt.Errorf("report queued backups: %w", err)
	}

	// Mark all as synced
	now := time.Now()
	for _, backup := range backups {
		backup.Status = QueuedBackupStatusSynced
		backup.SyncedAt = &now
		if err := q.store.UpdateQueuedBackup(ctx, backup); err != nil {
			q.logger.Warn().Err(err).Str("backup_id", backup.ID.String()).Msg("failed to mark backup as synced")
		}
	}

	q.mu.Lock()
	q.lastSuccessSync = now
	q.mu.Unlock()

	q.logger.Info().
		Int("backup_count", len(backups)).
		Msg("queued backups synced successfully")

	return nil
}

// healthCheckLoop periodically checks server health.
func (q *Queue) healthCheckLoop() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_ = q.checkServerHealth(ctx)
			cancel()
		}
	}
}

// syncLoop periodically attempts to sync pending backups.
func (q *Queue) syncLoop() {
	defer q.wg.Done()

	ticker := time.NewTicker(q.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-q.stopCh:
			return
		case <-ticker.C:
			if q.IsServerReachable() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				if err := q.syncPendingBackups(ctx); err != nil {
					q.logger.Debug().Err(err).Msg("periodic sync failed")
				}
				cancel()

				// Prune old entries
				pruneCtx, pruneCancel := context.WithTimeout(context.Background(), 30*time.Second)
				pruned, err := q.store.PruneOldEntries(pruneCtx, q.config.PruneAge)
				if err != nil {
					q.logger.Warn().Err(err).Msg("failed to prune old queue entries")
				} else if pruned > 0 {
					q.logger.Debug().Int("pruned_count", pruned).Msg("pruned old queue entries")
				}
				pruneCancel()
			}
		}
	}
}

// Errors
var (
	// ErrQueueFull is returned when the queue has reached its maximum size.
	ErrQueueFull = errors.New("backup queue is full")
	// ErrServerUnreachable is returned when the server cannot be contacted.
	ErrServerUnreachable = errors.New("server is unreachable")
	// ErrBackupNotFound is returned when a queued backup cannot be found.
	ErrBackupNotFound = errors.New("queued backup not found")
)

// HTTPServerClient implements ServerClient using HTTP.
type HTTPServerClient struct {
	serverURL  string
	apiKey     string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewHTTPServerClient creates a new HTTP-based server client.
func NewHTTPServerClient(serverURL, apiKey string, logger zerolog.Logger) *HTTPServerClient {
	return &HTTPServerClient{
		serverURL: serverURL,
		apiKey:    apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.With().Str("component", "server_client").Logger(),
	}
}

// CheckHealth checks if the server is reachable.
func (c *HTTPServerClient) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// queuedBackupRequest represents a backup result for the API.
type queuedBackupRequest struct {
	ID           string     `json:"id"`
	ScheduleID   string     `json:"schedule_id"`
	ScheduleName string     `json:"schedule_name"`
	ScheduledAt  time.Time  `json:"scheduled_at"`
	QueuedAt     time.Time  `json:"queued_at"`
	Success      bool       `json:"success"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	BytesAdded   int64      `json:"bytes_added,omitempty"`
	FilesNew     int        `json:"files_new,omitempty"`
	FilesChanged int        `json:"files_changed,omitempty"`
	SnapshotID   string     `json:"snapshot_id,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	RepositoryID string     `json:"repository_id,omitempty"`
}

// ReportQueuedBackups sends queued backup results to the server.
func (c *HTTPServerClient) ReportQueuedBackups(ctx context.Context, backups []*QueuedBackup) error {
	if len(backups) == 0 {
		return nil
	}

	// Convert to API format
	requests := make([]queuedBackupRequest, 0, len(backups))
	for _, b := range backups {
		req := queuedBackupRequest{
			ID:           b.ID.String(),
			ScheduleID:   b.ScheduleID.String(),
			ScheduleName: b.ScheduleName,
			ScheduledAt:  b.ScheduledAt,
			QueuedAt:     b.QueuedAt,
		}

		if b.BackupResult != nil {
			req.Success = b.BackupResult.Success
			req.StartedAt = &b.BackupResult.StartedAt
			req.CompletedAt = &b.BackupResult.CompletedAt
			req.BytesAdded = b.BackupResult.BytesAdded
			req.FilesNew = b.BackupResult.FilesNew
			req.FilesChanged = b.BackupResult.FilesChanged
			req.SnapshotID = b.BackupResult.SnapshotID
			req.ErrorMessage = b.BackupResult.ErrorMessage
			if b.BackupResult.RepositoryID != uuid.Nil {
				req.RepositoryID = b.BackupResult.RepositoryID.String()
			}
		}

		requests = append(requests, req)
	}

	body := map[string]any{
		"backups": requests,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+"/api/v1/agent/queued-backups", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	c.logger.Info().
		Int("backup_count", len(backups)).
		Msg("reported queued backups to server")

	return nil
}

// NotifyReconnection alerts the server about reconnection with queued items.
func (c *HTTPServerClient) NotifyReconnection(ctx context.Context, queuedCount int) error {
	body := map[string]int{
		"queued_count": queuedCount,
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.serverURL+"/api/v1/agent/reconnect", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	c.logger.Info().
		Int("queued_count", queuedCount).
		Msg("notified server of reconnection with queued backups")

	return nil
}
