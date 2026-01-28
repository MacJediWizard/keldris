package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DockerStore defines the database operations needed by the Docker monitor.
type DockerStore interface {
	GetAllAgents(ctx context.Context) ([]*models.Agent, error)
	GetAgentDockerHealth(ctx context.Context, agentID uuid.UUID) (*models.AgentDockerHealth, error)
	UpdateAgentDockerHealth(ctx context.Context, health *models.AgentDockerHealth) error
	CreateAgentDockerHealth(ctx context.Context, health *models.AgentDockerHealth) error
	GetAgentDockerHealthByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentDockerHealth, error)
	GetOrgIDByAgentID(ctx context.Context, agentID uuid.UUID) (uuid.UUID, error)
	CreateContainerRestartEvent(ctx context.Context, event *models.ContainerRestartEvent) error
	GetContainerRestartEvents(ctx context.Context, agentID uuid.UUID, containerID string, since time.Time) ([]*models.ContainerRestartEvent, error)
	GetRecentContainerRestartEvents(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.ContainerRestartEvent, error)
	GetDockerHealthSummary(ctx context.Context, orgID uuid.UUID) (*models.DockerHealthSummary, error)
}

// DockerAlertService defines the interface for Docker-specific alert operations.
type DockerAlertService interface {
	CreateAlert(ctx context.Context, alert *models.Alert) error
	ResolveAlertsByResource(ctx context.Context, resourceType models.ResourceType, resourceID uuid.UUID) error
	HasActiveAlert(ctx context.Context, orgID uuid.UUID, resourceType models.ResourceType, resourceID uuid.UUID, alertType models.AlertType) (bool, error)
}

// DockerMonitorConfig holds the configuration for Docker monitoring.
type DockerMonitorConfig struct {
	// CheckInterval is how often to run Docker health checks.
	CheckInterval time.Duration
	// RestartThreshold is the number of restarts before alerting.
	RestartThreshold int
	// RestartWindowMinutes is the time window for counting restarts.
	RestartWindowMinutes int
	// VolumeUsageThresholdPercent is the volume usage percent to alert on.
	VolumeUsageThresholdPercent float64
	// ContainerMemoryThresholdPercent is the container memory usage to alert on.
	ContainerMemoryThresholdPercent float64
	// ContainerCPUThresholdPercent is the container CPU usage to alert on.
	ContainerCPUThresholdPercent float64
}

// DefaultDockerMonitorConfig returns a DockerMonitorConfig with sensible defaults.
func DefaultDockerMonitorConfig() DockerMonitorConfig {
	return DockerMonitorConfig{
		CheckInterval:                   1 * time.Minute,
		RestartThreshold:                5,
		RestartWindowMinutes:            60,
		VolumeUsageThresholdPercent:     85.0,
		ContainerMemoryThresholdPercent: 90.0,
		ContainerCPUThresholdPercent:    90.0,
	}
}

// DockerMonitor monitors Docker container health across all agents.
type DockerMonitor struct {
	store        DockerStore
	alertService DockerAlertService
	config       DockerMonitorConfig
	logger       zerolog.Logger

	// Track previous restart counts to detect new restarts
	prevRestartCounts map[string]int
	mu                sync.RWMutex

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewDockerMonitor creates a new DockerMonitor instance.
func NewDockerMonitor(store DockerStore, alertService DockerAlertService, config DockerMonitorConfig, logger zerolog.Logger) *DockerMonitor {
	return &DockerMonitor{
		store:             store,
		alertService:      alertService,
		config:            config,
		logger:            logger.With().Str("component", "docker_monitor").Logger(),
		prevRestartCounts: make(map[string]int),
		stopCh:            make(chan struct{}),
	}
}

// NewDockerMonitorWithDB creates a DockerMonitor using the database directly.
func NewDockerMonitorWithDB(database *db.DB, alertService DockerAlertService, config DockerMonitorConfig, logger zerolog.Logger) *DockerMonitor {
	return NewDockerMonitor(database, alertService, config, logger)
}

// Start begins the Docker monitoring loop.
func (m *DockerMonitor) Start(ctx context.Context) {
	m.wg.Add(1)
	go m.run(ctx)
	m.logger.Info().
		Dur("check_interval", m.config.CheckInterval).
		Int("restart_threshold", m.config.RestartThreshold).
		Float64("volume_threshold", m.config.VolumeUsageThresholdPercent).
		Msg("docker monitor started")
}

// Stop gracefully stops the Docker monitoring loop.
func (m *DockerMonitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
	m.logger.Info().Msg("docker monitor stopped")
}

// run is the main Docker monitoring loop.
func (m *DockerMonitor) run(ctx context.Context) {
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

// runChecks executes all Docker monitoring checks.
func (m *DockerMonitor) runChecks(ctx context.Context) {
	m.logger.Debug().Msg("running docker health checks")

	agents, err := m.store.GetAllAgents(ctx)
	if err != nil {
		m.logger.Error().Err(err).Msg("failed to get agents for docker health check")
		return
	}

	for _, agent := range agents {
		if err := m.checkAgentDockerHealth(ctx, agent); err != nil {
			m.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check agent docker health")
		}
	}
}

// checkAgentDockerHealth checks Docker health for a single agent.
func (m *DockerMonitor) checkAgentDockerHealth(ctx context.Context, agent *models.Agent) error {
	dockerHealth, err := m.store.GetAgentDockerHealth(ctx, agent.ID)
	if err != nil {
		// No Docker health data yet - agent might not have Docker or hasn't reported yet
		return nil
	}

	if dockerHealth.DockerHealth == nil {
		return nil
	}

	health := dockerHealth.DockerHealth

	// Check if Docker daemon is available
	if err := m.checkDockerDaemonAvailability(ctx, agent, health); err != nil {
		m.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check docker daemon availability")
	}

	// Check container health
	if err := m.checkContainerHealth(ctx, agent, health); err != nil {
		m.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check container health")
	}

	// Check for restart loops
	if err := m.checkContainerRestartLoops(ctx, agent, health); err != nil {
		m.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check container restart loops")
	}

	// Check container resource usage
	if err := m.checkContainerResourceUsage(ctx, agent, health); err != nil {
		m.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check container resource usage")
	}

	// Check volume space
	if err := m.checkVolumeSpace(ctx, agent, health); err != nil {
		m.logger.Error().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check volume space")
	}

	return nil
}

// checkDockerDaemonAvailability checks if Docker daemon is responding.
func (m *DockerMonitor) checkDockerDaemonAvailability(ctx context.Context, agent *models.Agent, health *models.DockerHealth) error {
	// Create a synthetic resource ID for Docker daemon alerts
	resourceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("docker-daemon-%s", agent.ID.String())))

	if !health.Available {
		// Docker daemon is not available
		hasAlert, err := m.alertService.HasActiveAlert(ctx, agent.OrgID, models.ResourceTypeAgent, resourceID, models.AlertTypeDockerDaemonUnavailable)
		if err != nil {
			m.logger.Warn().Err(err).Str("agent_id", agent.ID.String()).Msg("failed to check for existing docker daemon alert")
		}

		if !hasAlert {
			alert := models.NewAlert(
				agent.OrgID,
				models.AlertTypeDockerDaemonUnavailable,
				models.AlertSeverityWarning,
				fmt.Sprintf("Docker daemon unavailable on %s", agent.Hostname),
				fmt.Sprintf("Docker daemon is not responding on agent %s", agent.Hostname),
			)
			alert.SetResource(models.ResourceTypeAgent, agent.ID)
			alert.Metadata = map[string]any{
				"hostname":     agent.Hostname,
				"agent_id":     agent.ID.String(),
				"last_checked": health.LastChecked,
			}

			if err := m.alertService.CreateAlert(ctx, alert); err != nil {
				return fmt.Errorf("create docker daemon alert: %w", err)
			}

			m.logger.Info().
				Str("agent_id", agent.ID.String()).
				Str("hostname", agent.Hostname).
				Msg("docker daemon unavailable alert created")
		}
	} else {
		// Docker daemon is available - resolve any active alerts
		if err := m.alertService.ResolveAlertsByResource(ctx, models.ResourceTypeAgent, resourceID); err != nil {
			return fmt.Errorf("resolve docker daemon alerts: %w", err)
		}
	}

	return nil
}

// checkContainerHealth checks for unhealthy containers.
func (m *DockerMonitor) checkContainerHealth(ctx context.Context, agent *models.Agent, health *models.DockerHealth) error {
	for _, container := range health.Containers {
		// Create a synthetic resource ID for container alerts
		resourceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("container-%s-%s", agent.ID.String(), container.ContainerID)))

		if container.HealthStatus == models.ContainerHealthStatusUnhealthy {
			hasAlert, err := m.alertService.HasActiveAlert(ctx, agent.OrgID, models.ResourceTypeContainer, resourceID, models.AlertTypeContainerUnhealthy)
			if err != nil {
				m.logger.Warn().Err(err).Str("container", container.Name).Msg("failed to check for existing container health alert")
			}

			if !hasAlert {
				alert := models.NewAlert(
					agent.OrgID,
					models.AlertTypeContainerUnhealthy,
					models.AlertSeverityWarning,
					fmt.Sprintf("Container %s unhealthy on %s", container.Name, agent.Hostname),
					fmt.Sprintf("Container %s health check is failing on agent %s", container.Name, agent.Hostname),
				)
				alert.SetResource(models.ResourceTypeContainer, resourceID)
				alert.Metadata = map[string]any{
					"hostname":       agent.Hostname,
					"agent_id":       agent.ID.String(),
					"container_id":   container.ContainerID,
					"container_name": container.Name,
					"image":          container.Image,
					"health_status":  string(container.HealthStatus),
				}

				if err := m.alertService.CreateAlert(ctx, alert); err != nil {
					m.logger.Error().Err(err).Str("container", container.Name).Msg("failed to create container health alert")
				} else {
					m.logger.Info().
						Str("agent_id", agent.ID.String()).
						Str("container", container.Name).
						Msg("container unhealthy alert created")
				}
			}
		} else if container.HealthStatus == models.ContainerHealthStatusHealthy {
			// Resolve any active unhealthy alerts for this container
			if err := m.alertService.ResolveAlertsByResource(ctx, models.ResourceTypeContainer, resourceID); err != nil {
				m.logger.Warn().Err(err).Str("container", container.Name).Msg("failed to resolve container health alerts")
			}
		}
	}

	return nil
}

// checkContainerRestartLoops checks for containers in restart loops.
func (m *DockerMonitor) checkContainerRestartLoops(ctx context.Context, agent *models.Agent, health *models.DockerHealth) error {
	windowStart := time.Now().Add(-time.Duration(m.config.RestartWindowMinutes) * time.Minute)

	for _, container := range health.Containers {
		key := fmt.Sprintf("%s-%s", agent.ID.String(), container.ContainerID)

		m.mu.RLock()
		prevCount, exists := m.prevRestartCounts[key]
		m.mu.RUnlock()

		// Detect new restart
		if exists && container.RestartCount > prevCount {
			// Record restart event
			event := models.NewContainerRestartEvent(
				agent.OrgID,
				agent.ID,
				container.ContainerID,
				container.Name,
				container.RestartCount,
			)
			if container.ExitCode != nil {
				event.ExitCode = container.ExitCode
			}

			if err := m.store.CreateContainerRestartEvent(ctx, event); err != nil {
				m.logger.Warn().Err(err).Str("container", container.Name).Msg("failed to record restart event")
			}
		}

		// Update tracked restart count
		m.mu.Lock()
		m.prevRestartCounts[key] = container.RestartCount
		m.mu.Unlock()

		// Check restart count in window
		events, err := m.store.GetContainerRestartEvents(ctx, agent.ID, container.ContainerID, windowStart)
		if err != nil {
			m.logger.Warn().Err(err).Str("container", container.Name).Msg("failed to get restart events")
			continue
		}

		restartCount := len(events)
		if restartCount >= m.config.RestartThreshold || container.State == models.ContainerStateRestarting {
			resourceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("container-restart-%s-%s", agent.ID.String(), container.ContainerID)))

			hasAlert, err := m.alertService.HasActiveAlert(ctx, agent.OrgID, models.ResourceTypeContainer, resourceID, models.AlertTypeContainerRestartLoop)
			if err != nil {
				m.logger.Warn().Err(err).Str("container", container.Name).Msg("failed to check for existing restart loop alert")
			}

			if !hasAlert {
				alert := models.NewAlert(
					agent.OrgID,
					models.AlertTypeContainerRestartLoop,
					models.AlertSeverityCritical,
					fmt.Sprintf("Container %s in restart loop on %s", container.Name, agent.Hostname),
					fmt.Sprintf("Container %s has restarted %d times in the last %d minutes on agent %s",
						container.Name, restartCount, m.config.RestartWindowMinutes, agent.Hostname),
				)
				alert.SetResource(models.ResourceTypeContainer, resourceID)
				alert.Metadata = map[string]any{
					"hostname":       agent.Hostname,
					"agent_id":       agent.ID.String(),
					"container_id":   container.ContainerID,
					"container_name": container.Name,
					"image":          container.Image,
					"restart_count":  restartCount,
					"window_minutes": m.config.RestartWindowMinutes,
					"threshold":      m.config.RestartThreshold,
				}

				if err := m.alertService.CreateAlert(ctx, alert); err != nil {
					m.logger.Error().Err(err).Str("container", container.Name).Msg("failed to create restart loop alert")
				} else {
					m.logger.Info().
						Str("agent_id", agent.ID.String()).
						Str("container", container.Name).
						Int("restart_count", restartCount).
						Msg("container restart loop alert created")
				}
			}
		}
	}

	return nil
}

// checkContainerResourceUsage checks for containers using excessive resources.
func (m *DockerMonitor) checkContainerResourceUsage(ctx context.Context, agent *models.Agent, health *models.DockerHealth) error {
	for _, container := range health.Containers {
		if container.State != models.ContainerStateRunning {
			continue
		}

		resourceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("container-resource-%s-%s", agent.ID.String(), container.ContainerID)))

		// Check memory usage
		if container.MemoryPercent >= m.config.ContainerMemoryThresholdPercent {
			hasAlert, err := m.alertService.HasActiveAlert(ctx, agent.OrgID, models.ResourceTypeContainer, resourceID, models.AlertTypeContainerHighResource)
			if err != nil {
				m.logger.Warn().Err(err).Str("container", container.Name).Msg("failed to check for existing resource alert")
			}

			if !hasAlert {
				alert := models.NewAlert(
					agent.OrgID,
					models.AlertTypeContainerHighResource,
					models.AlertSeverityWarning,
					fmt.Sprintf("High memory usage: %s on %s", container.Name, agent.Hostname),
					fmt.Sprintf("Container %s is using %.1f%% memory on agent %s",
						container.Name, container.MemoryPercent, agent.Hostname),
				)
				alert.SetResource(models.ResourceTypeContainer, resourceID)
				alert.Metadata = map[string]any{
					"hostname":        agent.Hostname,
					"agent_id":        agent.ID.String(),
					"container_id":    container.ContainerID,
					"container_name":  container.Name,
					"image":           container.Image,
					"memory_percent":  container.MemoryPercent,
					"memory_usage":    container.MemoryUsage,
					"memory_limit":    container.MemoryLimit,
					"resource_type":   "memory",
					"threshold":       m.config.ContainerMemoryThresholdPercent,
				}

				if err := m.alertService.CreateAlert(ctx, alert); err != nil {
					m.logger.Error().Err(err).Str("container", container.Name).Msg("failed to create memory usage alert")
				}
			}
		}

		// Check CPU usage
		if container.CPUPercent >= m.config.ContainerCPUThresholdPercent {
			cpuResourceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("container-cpu-%s-%s", agent.ID.String(), container.ContainerID)))

			hasAlert, err := m.alertService.HasActiveAlert(ctx, agent.OrgID, models.ResourceTypeContainer, cpuResourceID, models.AlertTypeContainerHighResource)
			if err != nil {
				m.logger.Warn().Err(err).Str("container", container.Name).Msg("failed to check for existing CPU alert")
			}

			if !hasAlert {
				alert := models.NewAlert(
					agent.OrgID,
					models.AlertTypeContainerHighResource,
					models.AlertSeverityWarning,
					fmt.Sprintf("High CPU usage: %s on %s", container.Name, agent.Hostname),
					fmt.Sprintf("Container %s is using %.1f%% CPU on agent %s",
						container.Name, container.CPUPercent, agent.Hostname),
				)
				alert.SetResource(models.ResourceTypeContainer, cpuResourceID)
				alert.Metadata = map[string]any{
					"hostname":       agent.Hostname,
					"agent_id":       agent.ID.String(),
					"container_id":   container.ContainerID,
					"container_name": container.Name,
					"image":          container.Image,
					"cpu_percent":    container.CPUPercent,
					"resource_type":  "cpu",
					"threshold":      m.config.ContainerCPUThresholdPercent,
				}

				if err := m.alertService.CreateAlert(ctx, alert); err != nil {
					m.logger.Error().Err(err).Str("container", container.Name).Msg("failed to create CPU usage alert")
				}
			}
		}
	}

	return nil
}

// checkVolumeSpace checks for volumes running low on space.
func (m *DockerMonitor) checkVolumeSpace(ctx context.Context, agent *models.Agent, health *models.DockerHealth) error {
	for _, volume := range health.Volumes {
		resourceID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("volume-%s-%s", agent.ID.String(), volume.Name)))

		if volume.UsagePercent >= m.config.VolumeUsageThresholdPercent {
			hasAlert, err := m.alertService.HasActiveAlert(ctx, agent.OrgID, models.ResourceTypeVolume, resourceID, models.AlertTypeVolumeSpaceLow)
			if err != nil {
				m.logger.Warn().Err(err).Str("volume", volume.Name).Msg("failed to check for existing volume space alert")
			}

			// Determine severity based on usage
			var severity models.AlertSeverity
			switch {
			case volume.UsagePercent >= 95:
				severity = models.AlertSeverityCritical
			case volume.UsagePercent >= 90:
				severity = models.AlertSeverityWarning
			default:
				severity = models.AlertSeverityInfo
			}

			if !hasAlert {
				alert := models.NewAlert(
					agent.OrgID,
					models.AlertTypeVolumeSpaceLow,
					severity,
					fmt.Sprintf("Volume %s low on space on %s", volume.Name, agent.Hostname),
					fmt.Sprintf("Volume %s is at %.1f%% capacity (%s of %s used) on agent %s",
						volume.Name, volume.UsagePercent,
						formatBytes(volume.UsedBytes), formatBytes(volume.TotalBytes),
						agent.Hostname),
				)
				alert.SetResource(models.ResourceTypeVolume, resourceID)
				alert.Metadata = map[string]any{
					"hostname":      agent.Hostname,
					"agent_id":      agent.ID.String(),
					"volume_name":   volume.Name,
					"driver":        volume.Driver,
					"mountpoint":    volume.Mountpoint,
					"used_bytes":    volume.UsedBytes,
					"total_bytes":   volume.TotalBytes,
					"usage_percent": volume.UsagePercent,
					"threshold":     m.config.VolumeUsageThresholdPercent,
				}

				if err := m.alertService.CreateAlert(ctx, alert); err != nil {
					m.logger.Error().Err(err).Str("volume", volume.Name).Msg("failed to create volume space alert")
				} else {
					m.logger.Info().
						Str("agent_id", agent.ID.String()).
						Str("volume", volume.Name).
						Float64("usage_percent", volume.UsagePercent).
						Msg("volume low space alert created")
				}
			}
		} else {
			// Volume is below threshold - resolve any active alerts
			if err := m.alertService.ResolveAlertsByResource(ctx, models.ResourceTypeVolume, resourceID); err != nil {
				m.logger.Warn().Err(err).Str("volume", volume.Name).Msg("failed to resolve volume space alerts")
			}
		}
	}

	return nil
}

// UpdateAgentDockerHealth updates Docker health data for an agent.
// This is called when an agent reports its Docker health via the API.
func (m *DockerMonitor) UpdateAgentDockerHealth(ctx context.Context, agentID uuid.UUID, health *models.DockerHealth) error {
	orgID, err := m.store.GetOrgIDByAgentID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("get org ID: %w", err)
	}

	health.CalculateCounts()
	health.LastChecked = time.Now()

	existing, err := m.store.GetAgentDockerHealth(ctx, agentID)
	if err != nil {
		// Create new record
		agentHealth := models.NewAgentDockerHealth(orgID, agentID)
		agentHealth.DockerHealth = health
		if err := m.store.CreateAgentDockerHealth(ctx, agentHealth); err != nil {
			return fmt.Errorf("create docker health: %w", err)
		}
		m.logger.Debug().Str("agent_id", agentID.String()).Msg("docker health record created")
	} else {
		// Update existing
		existing.DockerHealth = health
		existing.UpdatedAt = time.Now()
		if err := m.store.UpdateAgentDockerHealth(ctx, existing); err != nil {
			return fmt.Errorf("update docker health: %w", err)
		}
		m.logger.Debug().Str("agent_id", agentID.String()).Msg("docker health record updated")
	}

	return nil
}

// GetAgentDockerHealth retrieves Docker health for an agent.
func (m *DockerMonitor) GetAgentDockerHealth(ctx context.Context, agentID uuid.UUID) (*models.DockerHealth, error) {
	agentHealth, err := m.store.GetAgentDockerHealth(ctx, agentID)
	if err != nil {
		return nil, err
	}
	return agentHealth.DockerHealth, nil
}

// GetDockerHealthSummary returns a fleet-wide Docker health summary.
func (m *DockerMonitor) GetDockerHealthSummary(ctx context.Context, orgID uuid.UUID) (*models.DockerHealthSummary, error) {
	return m.store.GetDockerHealthSummary(ctx, orgID)
}

// GetDockerDashboardWidget returns data for the Docker dashboard widget.
func (m *DockerMonitor) GetDockerDashboardWidget(ctx context.Context, orgID uuid.UUID) (*models.DockerDashboardWidget, error) {
	summary, err := m.store.GetDockerHealthSummary(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get docker health summary: %w", err)
	}

	restarts, err := m.store.GetRecentContainerRestartEvents(ctx, orgID, 10)
	if err != nil {
		m.logger.Warn().Err(err).Msg("failed to get recent restart events")
		restarts = [](*models.ContainerRestartEvent){}
	}

	widget := &models.DockerDashboardWidget{
		Summary:        summary,
		RecentRestarts: make([]models.ContainerRestartEvent, 0, len(restarts)),
	}

	for _, r := range restarts {
		widget.RecentRestarts = append(widget.RecentRestarts, *r)
	}

	return widget, nil
}
