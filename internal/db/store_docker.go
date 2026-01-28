package db

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Docker Health methods

// GetAgentDockerHealth returns Docker health for an agent.
func (db *DB) GetAgentDockerHealth(ctx context.Context, agentID uuid.UUID) (*models.AgentDockerHealth, error) {
	var health models.AgentDockerHealth
	var dockerHealthJSON []byte
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, agent_id, docker_health, created_at, updated_at
		FROM agent_docker_health
		WHERE agent_id = $1
	`, agentID).Scan(
		&health.ID, &health.OrgID, &health.AgentID, &dockerHealthJSON, &health.CreatedAt, &health.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("docker health not found for agent %s", agentID)
		}
		return nil, fmt.Errorf("get agent docker health: %w", err)
	}

	if err := health.SetDockerHealth(dockerHealthJSON); err != nil {
		return nil, fmt.Errorf("parse docker health: %w", err)
	}

	return &health, nil
}

// CreateAgentDockerHealth creates a new agent Docker health record.
func (db *DB) CreateAgentDockerHealth(ctx context.Context, health *models.AgentDockerHealth) error {
	dockerHealthJSON, err := health.DockerHealthJSON()
	if err != nil {
		return fmt.Errorf("marshal docker health: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO agent_docker_health (id, org_id, agent_id, docker_health, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, health.ID, health.OrgID, health.AgentID, dockerHealthJSON, health.CreatedAt, health.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent docker health: %w", err)
	}
	return nil
}

// UpdateAgentDockerHealth updates an existing agent Docker health record.
func (db *DB) UpdateAgentDockerHealth(ctx context.Context, health *models.AgentDockerHealth) error {
	dockerHealthJSON, err := health.DockerHealthJSON()
	if err != nil {
		return fmt.Errorf("marshal docker health: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE agent_docker_health
		SET docker_health = $2, updated_at = $3
		WHERE id = $1
	`, health.ID, dockerHealthJSON, health.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update agent docker health: %w", err)
	}
	return nil
}

// GetAgentDockerHealthByOrgID returns Docker health for all agents in an organization.
func (db *DB) GetAgentDockerHealthByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AgentDockerHealth, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, docker_health, created_at, updated_at
		FROM agent_docker_health
		WHERE org_id = $1
		ORDER BY updated_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list agent docker health: %w", err)
	}
	defer rows.Close()

	var results []*models.AgentDockerHealth
	for rows.Next() {
		var health models.AgentDockerHealth
		var dockerHealthJSON []byte
		if err := rows.Scan(
			&health.ID, &health.OrgID, &health.AgentID, &dockerHealthJSON, &health.CreatedAt, &health.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan agent docker health: %w", err)
		}

		if err := health.SetDockerHealth(dockerHealthJSON); err != nil {
			return nil, fmt.Errorf("parse docker health: %w", err)
		}

		results = append(results, &health)
	}

	return results, nil
}

// Container Restart Event methods

// CreateContainerRestartEvent creates a new container restart event.
func (db *DB) CreateContainerRestartEvent(ctx context.Context, event *models.ContainerRestartEvent) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO container_restart_events (
			id, org_id, agent_id, container_id, container_name,
			restart_count, exit_code, reason, occurred_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, event.ID, event.OrgID, event.AgentID, event.ContainerID, event.ContainerName,
		event.RestartCount, event.ExitCode, event.Reason, event.OccurredAt, event.CreatedAt)
	if err != nil {
		return fmt.Errorf("create container restart event: %w", err)
	}
	return nil
}

// GetContainerRestartEvents returns restart events for a container since a given time.
func (db *DB) GetContainerRestartEvents(ctx context.Context, agentID uuid.UUID, containerID string, since time.Time) ([]*models.ContainerRestartEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, container_id, container_name,
		       restart_count, exit_code, reason, occurred_at, created_at
		FROM container_restart_events
		WHERE agent_id = $1 AND container_id = $2 AND occurred_at >= $3
		ORDER BY occurred_at DESC
	`, agentID, containerID, since)
	if err != nil {
		return nil, fmt.Errorf("get container restart events: %w", err)
	}
	defer rows.Close()

	return scanContainerRestartEvents(rows)
}

// GetRecentContainerRestartEvents returns recent restart events across all agents.
func (db *DB) GetRecentContainerRestartEvents(ctx context.Context, orgID uuid.UUID, limit int) ([]*models.ContainerRestartEvent, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, agent_id, container_id, container_name,
		       restart_count, exit_code, reason, occurred_at, created_at
		FROM container_restart_events
		WHERE org_id = $1
		ORDER BY occurred_at DESC
		LIMIT $2
	`, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent restart events: %w", err)
	}
	defer rows.Close()

	return scanContainerRestartEvents(rows)
}

// scanContainerRestartEvents scans rows into ContainerRestartEvent slice.
func scanContainerRestartEvents(rows pgx.Rows) ([]*models.ContainerRestartEvent, error) {
	var events []*models.ContainerRestartEvent
	for rows.Next() {
		var event models.ContainerRestartEvent
		if err := rows.Scan(
			&event.ID, &event.OrgID, &event.AgentID, &event.ContainerID, &event.ContainerName,
			&event.RestartCount, &event.ExitCode, &event.Reason, &event.OccurredAt, &event.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan container restart event: %w", err)
		}
		events = append(events, &event)
	}
	return events, nil
}

// GetDockerHealthSummary returns a fleet-wide Docker health summary.
func (db *DB) GetDockerHealthSummary(ctx context.Context, orgID uuid.UUID) (*models.DockerHealthSummary, error) {
	summary := &models.DockerHealthSummary{}

	// Get all Docker health records for the org
	healthRecords, err := db.GetAgentDockerHealthByOrgID(ctx, orgID)
	if err != nil {
		return summary, nil // Return empty summary if no records
	}

	for _, record := range healthRecords {
		if record.DockerHealth == nil {
			continue
		}

		health := record.DockerHealth
		summary.TotalAgentsWithDocker++

		if health.Available {
			if health.UnhealthyCount > 0 || health.RestartingCount > 0 {
				summary.AgentsDockerUnhealthy++
			} else {
				summary.AgentsDockerHealthy++
			}
		} else {
			summary.AgentsDockerUnavailable++
		}

		summary.TotalContainers += health.ContainerCount
		summary.RunningContainers += health.RunningCount
		summary.StoppedContainers += health.StoppedCount
		summary.UnhealthyContainers += health.UnhealthyCount
		summary.RestartingContainers += health.RestartingCount

		// Count high restart containers (threshold: 5)
		highRestartContainers := health.GetHighRestartContainers(5)
		summary.HighRestartContainers += len(highRestartContainers)

		// Count low space volumes (threshold: 85%)
		lowSpaceVolumes := health.GetLowSpaceVolumes(85.0)
		summary.LowSpaceVolumes += len(lowSpaceVolumes)
	}

	return summary, nil
}
