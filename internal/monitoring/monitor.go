// Package monitoring provides agent heartbeat checking, backup SLA tracking,
// and storage usage monitoring for the Keldris backup system.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// Store defines the database operations needed by the monitor.
type Store interface {
	GetAllAgents(ctx context.Context) ([]*models.Agent, error)
	GetAllSchedules(ctx context.Context) ([]*models.Schedule, error)
	GetLatestBackupByScheduleID(ctx context.Context, scheduleID uuid.UUID) (*models.Backup, error)
	GetOrgIDByAgentID(ctx context.Context, agentID uuid.UUID) (uuid.UUID, error)
	GetOrgIDByScheduleID(ctx context.Context, scheduleID uuid.UUID) (uuid.UUID, error)
	UpdateAgent(ctx context.Context, agent *models.Agent) error
}

// AlertService defines the interface for creating and resolving alerts.
type AlertService interface {
	CreateAlert(ctx context.Context, alert *models.Alert) error
	ResolveAlertsByResource(ctx context.Context, resourceType models.ResourceType, resourceID uuid.UUID) error
	HasActiveAlert(ctx context.Context, orgID uuid.UUID, resourceType models.ResourceType, resourceID uuid.UUID, alertType models.AlertType) (bool, error)
}

// Config holds the configuration for the monitor.
type Config struct {
	// AgentOfflineThreshold is the duration after which an agent is considered offline.
	AgentOfflineThreshold time.Duration
	// BackupSLAMaxHours is the maximum hours since last successful backup before alerting.
	BackupSLAMaxHours int
	// CheckInterval is how often to run monitoring checks.
	CheckInterval time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		AgentOfflineThreshold: 5 * time.Minute,
		BackupSLAMaxHours:     24,
		CheckInterval:         1 * time.Minute,
	}
}

// Monitor runs periodic checks on agents and backups.
type Monitor struct {
	store        Store
	alertService AlertService
	config       Config
	logger       zerolog.Logger

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewMonitor creates a new Monitor instance.
func NewMonitor(store Store, alertService AlertService, config Config, logger zerolog.Logger) *Monitor {
	return &Monitor{
		store:        store,
		alertService: alertService,
		config:       config,
		logger:       logger.With().Str("component", "monitor").Logger(),
		stopCh:       make(chan struct{}),
	}
}

// Start begins the monitoring loop.
func (m *Monitor) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.run(ctx)
	m.logger.Info().
		Dur("check_interval", m.config.CheckInterval).
		Dur("agent_offline_threshold", m.config.AgentOfflineThreshold).
		Int("backup_sla_max_hours", m.config.BackupSLAMaxHours).
		Msg("monitor started")
}

// Stop gracefully stops the monitoring loop.
func (m *Monitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info().Msg("monitor stopped")
}

// run is the main monitoring loop.
func (m *Monitor) run(ctx context.Context) {
	defer m.wg.Done()

	// Run immediately on start
	m.runChecks(ctx)

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.runChecks(ctx)
		}
	}
}

// runChecks executes all monitoring checks.
func (m *Monitor) runChecks(ctx context.Context) {
	m.logger.Debug().Msg("running monitoring checks")

	// Run checks concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := m.checkAgentHeartbeats(ctx); err != nil {
			m.logger.Error().Err(err).Msg("agent heartbeat check failed")
		}
	}()

	go func() {
		defer wg.Done()
		if err := m.checkBackupSLA(ctx); err != nil {
			m.logger.Error().Err(err).Msg("backup SLA check failed")
		}
	}()

	wg.Wait()
}

// checkAgentHeartbeats checks all agents for offline status.
func (m *Monitor) checkAgentHeartbeats(ctx context.Context) error {
	agents, err := m.store.GetAllAgents(ctx)
	if err != nil {
		return fmt.Errorf("get all agents: %w", err)
	}

	for _, agent := range agents {
		if err := m.checkAgentHeartbeat(ctx, agent); err != nil {
			m.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check agent heartbeat")
		}
	}

	return nil
}

// checkAgentHeartbeat checks a single agent's heartbeat status.
func (m *Monitor) checkAgentHeartbeat(ctx context.Context, agent *models.Agent) error {
	isOnline := agent.IsOnline(m.config.AgentOfflineThreshold)

	if !isOnline && agent.Status == models.AgentStatusActive {
		// Agent has gone offline
		agent.Status = models.AgentStatusOffline
		if err := m.store.UpdateAgent(ctx, agent); err != nil {
			return fmt.Errorf("update agent status: %w", err)
		}

		// Check if alert already exists
		hasAlert, err := m.alertService.HasActiveAlert(ctx, agent.OrgID, models.ResourceTypeAgent, agent.ID, models.AlertTypeAgentOffline)
		if err != nil {
			m.logger.Warn().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check for existing alert")
		}

		if !hasAlert {
			alert := models.NewAlert(
				agent.OrgID,
				models.AlertTypeAgentOffline,
				models.AlertSeverityWarning,
				fmt.Sprintf("Agent %s is offline", agent.Hostname),
				fmt.Sprintf("Agent %s has not sent a heartbeat in %v", agent.Hostname, m.config.AgentOfflineThreshold),
			)
			alert.SetResource(models.ResourceTypeAgent, agent.ID)
			alert.Metadata = map[string]any{
				"hostname":  agent.Hostname,
				"last_seen": agent.LastSeen,
			}

			if err := m.alertService.CreateAlert(ctx, alert); err != nil {
				return fmt.Errorf("create offline alert: %w", err)
			}

			m.logger.Info().
				Str("agent_id", agent.ID.String()).
				Str("hostname", agent.Hostname).
				Msg("agent offline alert created")
		}
	} else if isOnline && agent.Status == models.AgentStatusOffline {
		// Agent has come back online
		agent.Status = models.AgentStatusActive
		if err := m.store.UpdateAgent(ctx, agent); err != nil {
			return fmt.Errorf("update agent status: %w", err)
		}

		// Resolve any active offline alerts
		if err := m.alertService.ResolveAlertsByResource(ctx, models.ResourceTypeAgent, agent.ID); err != nil {
			return fmt.Errorf("resolve offline alerts: %w", err)
		}

		m.logger.Info().
			Str("agent_id", agent.ID.String()).
			Str("hostname", agent.Hostname).
			Msg("agent back online, alerts resolved")
	}

	return nil
}

// checkBackupSLA checks all schedules for backup SLA violations.
func (m *Monitor) checkBackupSLA(ctx context.Context) error {
	schedules, err := m.store.GetAllSchedules(ctx)
	if err != nil {
		return fmt.Errorf("get all schedules: %w", err)
	}

	for _, schedule := range schedules {
		if err := m.checkScheduleBackupSLA(ctx, schedule); err != nil {
			m.logger.Error().Err(err).Str("schedule_id", schedule.ID.String()).Msg("failed to check schedule backup SLA")
		}
	}

	return nil
}

// checkScheduleBackupSLA checks a single schedule's backup SLA.
func (m *Monitor) checkScheduleBackupSLA(ctx context.Context, schedule *models.Schedule) error {
	orgID, err := m.store.GetOrgIDByScheduleID(ctx, schedule.ID)
	if err != nil {
		return fmt.Errorf("get org ID: %w", err)
	}

	backup, err := m.store.GetLatestBackupByScheduleID(ctx, schedule.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No backups yet - check if schedule is old enough to warrant an alert
			scheduleAge := time.Since(schedule.CreatedAt)
			if scheduleAge > time.Duration(m.config.BackupSLAMaxHours)*time.Hour {
				return m.createBackupSLAAlert(ctx, orgID, schedule, nil)
			}
			return nil
		}
		return fmt.Errorf("get latest backup: %w", err)
	}

	// Check if the last successful backup is within SLA
	if backup.Status != models.BackupStatusCompleted {
		// Last backup failed, find the last successful one
		// For now, treat this as a potential SLA violation
		hoursSinceBackup := time.Since(backup.StartedAt).Hours()
		if hoursSinceBackup > float64(m.config.BackupSLAMaxHours) {
			return m.createBackupSLAAlert(ctx, orgID, schedule, backup)
		}
	} else {
		// Last backup was successful
		hoursSinceBackup := time.Since(backup.StartedAt).Hours()
		if hoursSinceBackup > float64(m.config.BackupSLAMaxHours) {
			return m.createBackupSLAAlert(ctx, orgID, schedule, backup)
		} else {
			// Backup is within SLA, resolve any active alerts
			if err := m.alertService.ResolveAlertsByResource(ctx, models.ResourceTypeSchedule, schedule.ID); err != nil {
				return fmt.Errorf("resolve SLA alerts: %w", err)
			}
		}
	}

	return nil
}

// createBackupSLAAlert creates an alert for a backup SLA violation.
func (m *Monitor) createBackupSLAAlert(ctx context.Context, orgID uuid.UUID, schedule *models.Schedule, lastBackup *models.Backup) error {
	// Check if alert already exists
	hasAlert, err := m.alertService.HasActiveAlert(ctx, orgID, models.ResourceTypeSchedule, schedule.ID, models.AlertTypeBackupSLA)
	if err != nil {
		m.logger.Warn().Err(err).Str("schedule_id", schedule.ID.String()).Msg("failed to check for existing alert")
	}

	if hasAlert {
		return nil
	}

	var message string
	var metadata map[string]any

	if lastBackup != nil {
		hoursSince := int(time.Since(lastBackup.StartedAt).Hours())
		message = fmt.Sprintf("Schedule '%s' has not had a successful backup in %d hours (SLA: %d hours)",
			schedule.Name, hoursSince, m.config.BackupSLAMaxHours)
		metadata = map[string]any{
			"schedule_name":    schedule.Name,
			"last_backup_id":   lastBackup.ID.String(),
			"last_backup_time": lastBackup.StartedAt,
			"last_backup_status": string(lastBackup.Status),
			"sla_hours":        m.config.BackupSLAMaxHours,
		}
	} else {
		message = fmt.Sprintf("Schedule '%s' has never completed a backup (SLA: %d hours)",
			schedule.Name, m.config.BackupSLAMaxHours)
		metadata = map[string]any{
			"schedule_name": schedule.Name,
			"sla_hours":     m.config.BackupSLAMaxHours,
		}
	}

	alert := models.NewAlert(
		orgID,
		models.AlertTypeBackupSLA,
		models.AlertSeverityCritical,
		fmt.Sprintf("Backup SLA violation: %s", schedule.Name),
		message,
	)
	alert.SetResource(models.ResourceTypeSchedule, schedule.ID)
	alert.Metadata = metadata

	if err := m.alertService.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("create SLA alert: %w", err)
	}

	m.logger.Info().
		Str("schedule_id", schedule.ID.String()).
		Str("schedule_name", schedule.Name).
		Msg("backup SLA alert created")

	return nil
}

// StorageUsageResult contains the result of a storage usage check.
type StorageUsageResult struct {
	RepositoryID   uuid.UUID
	RepositoryName string
	UsedBytes      int64
	TotalBytes     int64
	UsagePercent   float64
}

// CheckStorageUsage checks storage usage for a repository.
// This is designed to be called by the backup service after each backup.
func (m *Monitor) CheckStorageUsage(ctx context.Context, orgID uuid.UUID, result StorageUsageResult, thresholdPercent int) error {
	if result.UsagePercent < float64(thresholdPercent) {
		// Usage is below threshold, resolve any active alerts
		if err := m.alertService.ResolveAlertsByResource(ctx, models.ResourceTypeRepository, result.RepositoryID); err != nil {
			return fmt.Errorf("resolve storage alerts: %w", err)
		}
		return nil
	}

	// Check if alert already exists
	hasAlert, err := m.alertService.HasActiveAlert(ctx, orgID, models.ResourceTypeRepository, result.RepositoryID, models.AlertTypeStorageUsage)
	if err != nil {
		m.logger.Warn().Err(err).Str("repository_id", result.RepositoryID.String()).Msg("failed to check for existing alert")
	}

	if hasAlert {
		return nil
	}

	// Determine severity based on usage
	var severity models.AlertSeverity
	switch {
	case result.UsagePercent >= 95:
		severity = models.AlertSeverityCritical
	case result.UsagePercent >= 85:
		severity = models.AlertSeverityWarning
	default:
		severity = models.AlertSeverityInfo
	}

	alert := models.NewAlert(
		orgID,
		models.AlertTypeStorageUsage,
		severity,
		fmt.Sprintf("High storage usage: %s", result.RepositoryName),
		fmt.Sprintf("Repository '%s' is at %.1f%% capacity (%s of %s used)",
			result.RepositoryName,
			result.UsagePercent,
			formatBytes(result.UsedBytes),
			formatBytes(result.TotalBytes)),
	)
	alert.SetResource(models.ResourceTypeRepository, result.RepositoryID)
	alert.Metadata = map[string]any{
		"repository_name": result.RepositoryName,
		"used_bytes":      result.UsedBytes,
		"total_bytes":     result.TotalBytes,
		"usage_percent":   result.UsagePercent,
	}

	if err := m.alertService.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("create storage alert: %w", err)
	}

	m.logger.Info().
		Str("repository_id", result.RepositoryID.String()).
		Str("repository_name", result.RepositoryName).
		Float64("usage_percent", result.UsagePercent).
		Msg("storage usage alert created")

	return nil
}

// formatBytes formats bytes into a human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// NewMonitorWithDB creates a Monitor using the database directly.
func NewMonitorWithDB(database *db.DB, alertService AlertService, config Config, logger zerolog.Logger) *Monitor {
	return NewMonitor(database, alertService, config, logger)
}
