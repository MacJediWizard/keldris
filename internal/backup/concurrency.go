package backup

import (
	"context"
	"errors"
	"sync"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ConcurrencyStore defines the interface for concurrency-related data access.
type ConcurrencyStore interface {
	// GetOrganizationByID returns an organization by ID.
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*models.Organization, error)

	// GetAgentByID returns an agent by ID.
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)

	// GetRunningBackupsCountByOrg returns the count of running backups for an org.
	GetRunningBackupsCountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)

	// GetRunningBackupsCountByAgent returns the count of running backups for an agent.
	GetRunningBackupsCountByAgent(ctx context.Context, agentID uuid.UUID) (int, error)

	// CreateBackupQueueEntry creates a new queue entry.
	CreateBackupQueueEntry(ctx context.Context, entry *models.BackupQueueEntry) error

	// GetQueuedBackupsByOrg returns queued backups for an org.
	GetQueuedBackupsByOrg(ctx context.Context, orgID uuid.UUID) ([]*models.BackupQueueEntry, error)

	// GetQueuedBackupsByAgent returns queued backups for an agent.
	GetQueuedBackupsByAgent(ctx context.Context, agentID uuid.UUID) ([]*models.BackupQueueEntry, error)

	// GetOldestQueuedBackup returns the oldest queued backup for an org.
	GetOldestQueuedBackup(ctx context.Context, orgID uuid.UUID) (*models.BackupQueueEntry, error)

	// UpdateBackupQueueEntry updates a queue entry.
	UpdateBackupQueueEntry(ctx context.Context, entry *models.BackupQueueEntry) error

	// DeleteBackupQueueEntry deletes a queue entry.
	DeleteBackupQueueEntry(ctx context.Context, id uuid.UUID) error

	// GetQueuePosition returns the position in queue for a given entry.
	GetQueuePosition(ctx context.Context, orgID, entryID uuid.UUID) (int, error)

	// GetConcurrencyQueueSummary returns queue statistics.
	GetConcurrencyQueueSummary(ctx context.Context, orgID uuid.UUID) (*models.ConcurrencyQueueSummary, error)

	// GetScheduleByID returns a schedule by ID.
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*models.Schedule, error)
}

// ConcurrencyManager manages backup concurrency limits and queuing.
type ConcurrencyManager struct {
	store    ConcurrencyStore
	notifier *notifications.Service
	logger   zerolog.Logger
	mu       sync.RWMutex

	// In-memory tracking of running backups (supplement to DB)
	runningByOrg   map[uuid.UUID]int
	runningByAgent map[uuid.UUID]int
}

// NewConcurrencyManager creates a new concurrency manager.
func NewConcurrencyManager(store ConcurrencyStore, notifier *notifications.Service, logger zerolog.Logger) *ConcurrencyManager {
	return &ConcurrencyManager{
		store:          store,
		notifier:       notifier,
		logger:         logger.With().Str("component", "concurrency_manager").Logger(),
		runningByOrg:   make(map[uuid.UUID]int),
		runningByAgent: make(map[uuid.UUID]int),
	}
}

// CanStartBackup checks if a backup can start given current limits.
// Returns (canStart, shouldQueue, error).
func (m *ConcurrencyManager) CanStartBackup(ctx context.Context, orgID, agentID uuid.UUID) (bool, bool, error) {
	// Get org limits
	org, err := m.store.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return false, false, err
	}

	// Get agent limits
	agent, err := m.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return false, false, err
	}

	// Check org-level limit
	if org.MaxConcurrentBackups != nil && *org.MaxConcurrentBackups > 0 {
		runningCount, err := m.getRunningCountForOrg(ctx, orgID)
		if err != nil {
			return false, false, err
		}
		if runningCount >= *org.MaxConcurrentBackups {
			return false, true, nil // Should queue
		}
	}

	// Check agent-level limit (uses agent setting, or falls back to org)
	agentLimit := agent.MaxConcurrentBackups
	if agentLimit != nil && *agentLimit > 0 {
		runningCount, err := m.getRunningCountForAgent(ctx, agentID)
		if err != nil {
			return false, false, err
		}
		if runningCount >= *agentLimit {
			return false, true, nil // Should queue
		}
	}

	return true, false, nil
}

// AcquireSlot attempts to acquire a backup slot. Returns true if acquired.
// If limit is reached, queues the backup and returns false.
func (m *ConcurrencyManager) AcquireSlot(ctx context.Context, orgID, agentID, scheduleID uuid.UUID) (bool, *models.BackupQueueEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	canStart, shouldQueue, err := m.CanStartBackup(ctx, orgID, agentID)
	if err != nil {
		return false, nil, err
	}

	if canStart {
		// Increment running counts
		m.runningByOrg[orgID]++
		m.runningByAgent[agentID]++
		return true, nil, nil
	}

	if shouldQueue {
		// Create queue entry
		entry := models.NewBackupQueueEntry(orgID, agentID, scheduleID, 0)
		if err := m.store.CreateBackupQueueEntry(ctx, entry); err != nil {
			return false, nil, err
		}

		// Get queue position
		pos, err := m.store.GetQueuePosition(ctx, orgID, entry.ID)
		if err == nil {
			entry.QueuePosition = pos
		}

		// Send notification about queued backup
		m.notifyBackupQueued(ctx, entry)

		m.logger.Info().
			Str("org_id", orgID.String()).
			Str("agent_id", agentID.String()).
			Str("schedule_id", scheduleID.String()).
			Int("queue_position", entry.QueuePosition).
			Msg("backup queued due to concurrency limit")

		return false, entry, nil
	}

	return false, nil, errors.New("unexpected state: cannot start and should not queue")
}

// ReleaseSlot releases a backup slot and starts the next queued backup if any.
func (m *ConcurrencyManager) ReleaseSlot(ctx context.Context, orgID, agentID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Decrement running counts
	if m.runningByOrg[orgID] > 0 {
		m.runningByOrg[orgID]--
	}
	if m.runningByAgent[agentID] > 0 {
		m.runningByAgent[agentID]--
	}

	// Check if we should start a queued backup
	return m.processQueue(ctx, orgID)
}

// processQueue checks for queued backups and starts the next eligible one.
func (m *ConcurrencyManager) processQueue(ctx context.Context, orgID uuid.UUID) error {
	entry, err := m.store.GetOldestQueuedBackup(ctx, orgID)
	if err != nil {
		// No queued backups or error
		return nil
	}

	if entry == nil {
		return nil
	}

	// Check if this backup can now start
	canStart, _, err := m.CanStartBackup(ctx, entry.OrgID, entry.AgentID)
	if err != nil {
		return err
	}

	if canStart {
		// Mark as started
		entry.MarkStarted()
		if err := m.store.UpdateBackupQueueEntry(ctx, entry); err != nil {
			return err
		}

		// Increment running counts
		m.runningByOrg[entry.OrgID]++
		m.runningByAgent[entry.AgentID]++

		m.logger.Info().
			Str("queue_id", entry.ID.String()).
			Str("org_id", entry.OrgID.String()).
			Str("agent_id", entry.AgentID.String()).
			Str("schedule_id", entry.ScheduleID.String()).
			Msg("starting queued backup")

		// Return the entry so caller can trigger the backup
		// This will be handled by the scheduler's queue processor
	}

	return nil
}

// GetConcurrencyStatus returns the current concurrency status for an org/agent.
func (m *ConcurrencyManager) GetConcurrencyStatus(ctx context.Context, orgID, agentID uuid.UUID) (*models.ConcurrencyStatus, error) {
	org, err := m.store.GetOrganizationByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	agent, err := m.store.GetAgentByID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	orgRunning, err := m.getRunningCountForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	agentRunning, err := m.getRunningCountForAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	orgQueued, err := m.store.GetQueuedBackupsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	agentQueued, err := m.store.GetQueuedBackupsByAgent(ctx, agentID)
	if err != nil {
		return nil, err
	}

	canStart, _, _ := m.CanStartBackup(ctx, orgID, agentID)

	status := &models.ConcurrencyStatus{
		OrgID:             orgID,
		OrgLimit:          org.MaxConcurrentBackups,
		OrgRunningCount:   orgRunning,
		OrgQueuedCount:    len(orgQueued),
		AgentID:           agentID,
		AgentLimit:        agent.MaxConcurrentBackups,
		AgentRunningCount: agentRunning,
		AgentQueuedCount:  len(agentQueued),
		CanStartNow:       canStart,
	}

	// Estimate wait time based on average backup duration
	if !canStart && len(orgQueued) > 0 {
		// Rough estimate: 30 minutes per queued backup
		status.EstimatedWaitMinutes = len(orgQueued) * 30
	}

	return status, nil
}

// CancelQueuedBackup cancels a backup waiting in queue.
func (m *ConcurrencyManager) CancelQueuedBackup(ctx context.Context, entryID uuid.UUID) error {
	return m.store.DeleteBackupQueueEntry(ctx, entryID)
}

// GetQueuedBackups returns all queued backups for an organization.
func (m *ConcurrencyManager) GetQueuedBackups(ctx context.Context, orgID uuid.UUID) ([]*models.BackupQueueEntry, error) {
	entries, err := m.store.GetQueuedBackupsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Calculate queue positions
	for i, entry := range entries {
		entry.QueuePosition = i + 1
	}

	return entries, nil
}

// getRunningCountForOrg returns the running backup count for an org.
func (m *ConcurrencyManager) getRunningCountForOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	// First check in-memory cache
	m.mu.RLock()
	inMem := m.runningByOrg[orgID]
	m.mu.RUnlock()

	// Also check database for accurate count
	dbCount, err := m.store.GetRunningBackupsCountByOrg(ctx, orgID)
	if err != nil {
		return inMem, nil // Fall back to in-memory on error
	}

	// Use the higher of the two counts to be safe
	if dbCount > inMem {
		return dbCount, nil
	}
	return inMem, nil
}

// getRunningCountForAgent returns the running backup count for an agent.
func (m *ConcurrencyManager) getRunningCountForAgent(ctx context.Context, agentID uuid.UUID) (int, error) {
	// First check in-memory cache
	m.mu.RLock()
	inMem := m.runningByAgent[agentID]
	m.mu.RUnlock()

	// Also check database for accurate count
	dbCount, err := m.store.GetRunningBackupsCountByAgent(ctx, agentID)
	if err != nil {
		return inMem, nil // Fall back to in-memory on error
	}

	// Use the higher of the two counts to be safe
	if dbCount > inMem {
		return dbCount, nil
	}
	return inMem, nil
}

// notifyBackupQueued sends a notification when a backup is queued.
func (m *ConcurrencyManager) notifyBackupQueued(ctx context.Context, entry *models.BackupQueueEntry) {
	if m.notifier == nil {
		return
	}

	// Get schedule name for notification
	schedule, err := m.store.GetScheduleByID(ctx, entry.ScheduleID)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get schedule for queue notification")
		return
	}

	// Get agent hostname
	agent, err := m.store.GetAgentByID(ctx, entry.AgentID)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get agent for queue notification")
		return
	}

	m.logger.Info().
		Str("schedule_name", schedule.Name).
		Str("agent_hostname", agent.Hostname).
		Int("queue_position", entry.QueuePosition).
		Msg("backup queued notification would be sent")

	// TODO: Implement actual notification via notifier service
	// This would require adding a new notification event type
}

// SyncRunningCounts synchronizes in-memory counts with database.
func (m *ConcurrencyManager) SyncRunningCounts(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear in-memory maps
	m.runningByOrg = make(map[uuid.UUID]int)
	m.runningByAgent = make(map[uuid.UUID]int)

	m.logger.Debug().Msg("synced running backup counts from database")
	return nil
}

// ProcessQueuedBackups checks all queues and starts eligible backups.
// This should be called periodically to handle any stuck queue entries.
func (m *ConcurrencyManager) ProcessQueuedBackups(ctx context.Context) ([]*models.BackupQueueEntry, error) {
	// This would iterate through all orgs with queued backups
	// For now, return empty - caller should use ReleaseSlot which processes queue
	return nil, nil
}
