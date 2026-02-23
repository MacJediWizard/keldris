package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// =============================================================================
// Methods recovered from feature branches after union merge.
// =============================================================================

// AcceptInvitation marks an invitation as accepted.
func (db *DB) AcceptInvitation(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_invitations
		SET accepted_at = $2
		WHERE id = $1
	`, id, now)
	if err != nil {
		return fmt.Errorf("accept invitation: %w", err)
	}
	return nil
}

// AddAgentToGroup adds an agent to a group.
func (db *DB) AddAgentToGroup(ctx context.Context, groupID, agentID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO agent_group_members (id, group_id, agent_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (group_id, agent_id) DO NOTHING
	`, uuid.New(), groupID, agentID, time.Now())
	if err != nil {
		return fmt.Errorf("add agent to group: %w", err)
	}
	return nil
}

// CancelAgentCommand cancels a pending or running command.
func (db *DB) CancelAgentCommand(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	result, err := db.Pool.Exec(ctx, `
		UPDATE agent_commands
		SET status = 'canceled', completed_at = $2, updated_at = $2
		WHERE id = $1 AND status IN ('pending', 'acknowledged', 'running')
	`, id, now)
	if err != nil {
		return fmt.Errorf("cancel agent command: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("command not found or already completed")
	}
	return nil
}

// CountAgentsByOrgID returns the total number of agents for an organization.
func (db *DB) CountAgentsByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM agents WHERE org_id = $1
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count agents by org: %w", err)
	}
	return count, nil
}

// CreateAgentLogs inserts multiple log entries for an agent.
func (db *DB) CreateAgentLogs(ctx context.Context, logs []*models.AgentLog) error {
	if len(logs) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	for _, log := range logs {
		metadataBytes, err := log.MetadataJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		batch.Queue(`
			INSERT INTO agent_logs (id, agent_id, org_id, level, message, component, metadata, timestamp, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, log.ID, log.AgentID, log.OrgID, string(log.Level), log.Message, log.Component, metadataBytes, log.Timestamp, log.CreatedAt)
	}

	results := db.Pool.SendBatch(ctx, batch)
	defer results.Close()

	for range logs {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("create agent log: %w", err)
		}
	}

	return nil
}

// CreateAnnouncement creates a new announcement.
func (db *DB) CreateAnnouncement(ctx context.Context, a *models.Announcement) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO announcements (id, org_id, title, message, type, dismissible, starts_at, ends_at,
		            active, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, a.ID, a.OrgID, a.Title, a.Message, a.Type, a.Dismissible, a.StartsAt, a.EndsAt,
		a.Active, a.CreatedBy, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create announcement: %w", err)
	}
	return nil
}

// CreateConfigTemplate creates a new config template.
func (db *DB) CreateConfigTemplate(ctx context.Context, template *models.ConfigTemplate) error {
	tagsJSON, err := template.TagsJSON()
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO config_templates (
			id, org_id, created_by_id, name, description, type, visibility,
			tags, config, usage_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		template.ID, template.OrgID, template.CreatedByID, template.Name,
		template.Description, template.Type, template.Visibility,
		tagsJSON, template.Config, template.UsageCount, template.CreatedAt, template.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create config template: %w", err)
	}
	return nil
}

// CreateCostAlert creates a new cost alert.
func (db *DB) CreateCostAlert(ctx context.Context, a *models.CostAlert) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO cost_alerts (id, org_id, name, monthly_threshold, enabled,
		             notify_on_exceed, notify_on_forecast, forecast_months,
		             created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, a.ID, a.OrgID, a.Name, a.MonthlyThreshold, a.Enabled,
		a.NotifyOnExceed, a.NotifyOnForecast, a.ForecastMonths,
		a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create cost alert: %w", err)
	}
	return nil
}

// CreateDefaultTierConfigs creates default tier configurations for an organization.
func (db *DB) CreateDefaultTierConfigs(ctx context.Context, orgID uuid.UUID) error {
	configs := models.DefaultTierConfigs(orgID)
	for _, config := range configs {
		if err := db.CreateStorageTierConfig(ctx, config); err != nil {
			return err
		}
	}
	return nil
}

// CreateDockerBackup creates a new Docker backup job.
func (db *DB) CreateDockerBackup(ctx context.Context, orgID uuid.UUID, req *models.DockerBackupParams) (*models.DockerBackupResult, error) {
	containerIDs, err := json.Marshal(req.ContainerIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal container ids: %w", err)
	}
	volumeNames, err := json.Marshal(req.VolumeNames)
	if err != nil {
		return nil, fmt.Errorf("marshal volume names: %w", err)
	}

	var result models.DockerBackupResult
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO docker_backups (org_id, agent_id, repository_id, container_ids, volume_names, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'queued', NOW())
		RETURNING id, status, created_at
	`, orgID, req.AgentID, req.RepositoryID, containerIDs, volumeNames).Scan(&result.ID, &result.Status, &result.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create docker backup: %w", err)
	}
	return &result, nil
}

// CreateDRRunbook creates a new DR runbook.
func (db *DB) CreateDRRunbook(ctx context.Context, runbook *models.DRRunbook) error {
	stepsBytes, err := runbook.StepsJSON()
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}

	contactsBytes, err := runbook.ContactsJSON()
	if err != nil {
		return fmt.Errorf("marshal contacts: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO dr_runbooks (id, org_id, schedule_id, name, description, steps, contacts,
		                         credentials_location, recovery_time_objective_minutes,
		                         recovery_point_objective_minutes, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, runbook.ID, runbook.OrgID, runbook.ScheduleID, runbook.Name, runbook.Description,
		stepsBytes, contactsBytes, runbook.CredentialsLocation,
		runbook.RecoveryTimeObjectiveMins, runbook.RecoveryPointObjectiveMins,
		string(runbook.Status), runbook.CreatedAt, runbook.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create DR runbook: %w", err)
	}
	return nil
}

// CreateDRTest creates a new DR test record.
func (db *DB) CreateDRTest(ctx context.Context, test *models.DRTest) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO dr_tests (id, runbook_id, schedule_id, agent_id, snapshot_id, status,
		                      started_at, completed_at, restore_size_bytes, restore_duration_seconds,
		                      verification_passed, notes, error_message, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, test.ID, test.RunbookID, test.ScheduleID, test.AgentID, test.SnapshotID,
		string(test.Status), test.StartedAt, test.CompletedAt, test.RestoreSizeBytes,
		test.RestoreDurationSeconds, test.VerificationPassed, test.Notes,
		test.ErrorMessage, test.CreatedAt)
	if err != nil {
		return fmt.Errorf("create DR test: %w", err)
	}
	return nil
}

// CreateExcludePattern creates a new exclude pattern.
func (db *DB) CreateExcludePattern(ctx context.Context, ep *models.ExcludePattern) error {
	patternsBytes, err := ep.PatternsJSON()
	if err != nil {
		return fmt.Errorf("marshal patterns: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO exclude_patterns (id, org_id, name, description, patterns, category, is_builtin, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, ep.ID, ep.OrgID, ep.Name, ep.Description, patternsBytes, ep.Category, ep.IsBuiltin, ep.CreatedAt, ep.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create exclude pattern: %w", err)
	}
	return nil
}

// CreateFavorite creates a new favorite.
func (db *DB) CreateFavorite(ctx context.Context, f *models.Favorite) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO user_favorites (id, user_id, org_id, entity_type, entity_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, f.ID, f.UserID, f.OrgID, f.EntityType, f.EntityID, f.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert favorite: %w", err)
	}
	return nil
}

// CreateGeoReplicationConfig creates a new geo-replication configuration.
func (db *DB) CreateGeoReplicationConfig(ctx context.Context, config *models.GeoReplicationConfig) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO geo_replication_configs (
			id, org_id, source_repository_id, target_repository_id,
			source_region, target_region, enabled, status,
			last_snapshot_id, last_sync_at, last_error,
			max_lag_snapshots, max_lag_duration_hours, alert_on_lag,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, config.ID, config.OrgID, config.SourceRepositoryID, config.TargetRepositoryID,
		config.SourceRegion, config.TargetRegion, config.Enabled, config.Status,
		config.LastSnapshotID, config.LastSyncAt, config.LastError,
		config.MaxLagSnapshots, config.MaxLagDurationHours, config.AlertOnLag,
		config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create geo-replication config: %w", err)
	}
	return nil
}

// CreateIPAllowlist creates a new IP allowlist entry.
func (db *DB) CreateIPAllowlist(ctx context.Context, a *models.IPAllowlist) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO ip_allowlists (id, org_id, cidr, description, type, enabled, created_by, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, a.ID, a.OrgID, a.CIDR, a.Description, a.Type, a.Enabled, a.CreatedBy, a.UpdatedBy, a.CreatedAt, a.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create ip allowlist: %w", err)
	}
	return nil
}

// CreateIPBan creates a new IP ban.
func (db *DB) CreateIPBan(ctx context.Context, b *models.IPBan) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO ip_bans (id, org_id, ip_address, reason, banned_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, b.ID, b.OrgID, b.IPAddress, b.Reason, b.BannedBy, b.ExpiresAt, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("create ip ban: %w", err)
	}
	return nil
}

// CreateIPBlockedAttempt records a blocked access attempt.
func (db *DB) CreateIPBlockedAttempt(ctx context.Context, b *models.IPBlockedAttempt) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO ip_blocked_attempts (id, org_id, ip_address, request_type, path, user_id, agent_id, reason, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, b.ID, b.OrgID, b.IPAddress, b.RequestType, b.Path, b.UserID, b.AgentID, b.Reason, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("create ip blocked attempt: %w", err)
	}
	return nil
}

// CreateImportedSnapshot creates a new imported snapshot record.
func (db *DB) CreateImportedSnapshot(ctx context.Context, snap *models.ImportedSnapshot) error {
	pathsJSON, err := json.Marshal(snap.Paths)
	if err != nil {
		return fmt.Errorf("marshal paths: %w", err)
	}

	tagsJSON, err := json.Marshal(snap.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO imported_snapshots (
			id, repository_id, agent_id, restic_snapshot_id, short_id,
			hostname, username, snapshot_time, paths, tags, imported_at, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, snap.ID, snap.RepositoryID, snap.AgentID, snap.ResticSnapshotID, snap.ShortID,
		snap.Hostname, snap.Username, snap.SnapshotTime, pathsJSON, tagsJSON,
		snap.ImportedAt, snap.CreatedAt)
	if err != nil {
		return fmt.Errorf("create imported snapshot: %w", err)
	}
	return nil
}

// CreateInvitation creates a new organization invitation.
func (db *DB) CreateInvitation(ctx context.Context, inv *models.OrgInvitation) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO org_invitations (id, org_id, email, role, token, invited_by, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, inv.ID, inv.OrgID, inv.Email, string(inv.Role), inv.Token, inv.InvitedBy, inv.ExpiresAt, inv.CreatedAt)
	if err != nil {
		return fmt.Errorf("create invitation: %w", err)
	}
	return nil
}

// CreateLegalHold creates a new legal hold on a snapshot.
func (db *DB) CreateLegalHold(ctx context.Context, hold *models.LegalHold) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO legal_holds (id, org_id, snapshot_id, reason, placed_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, hold.ID, hold.OrgID, hold.SnapshotID, hold.Reason, hold.PlacedBy, hold.CreatedAt, hold.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create legal hold: %w", err)
	}
	return nil
}

// CreateLifecycleDeletionEvent creates a deletion event for audit logging.
func (db *DB) CreateLifecycleDeletionEvent(ctx context.Context, event *models.LifecycleDeletionEvent) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO lifecycle_deletion_events (
			id, org_id, policy_id, snapshot_id, repository_id, reason, size_bytes, deleted_by, deleted_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, event.ID, event.OrgID, event.PolicyID, event.SnapshotID, event.RepositoryID,
		event.Reason, event.SizeBytes, event.DeletedBy, event.DeletedAt)
	if err != nil {
		return fmt.Errorf("create lifecycle deletion event: %w", err)
	}
	return nil
}

// CreateMaintenanceWindow creates a new maintenance window.
func (db *DB) CreateMaintenanceWindow(ctx context.Context, m *models.MaintenanceWindow) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO maintenance_windows (id, org_id, title, message, starts_at, ends_at,
		            notify_before_minutes, notification_sent, read_only, countdown_start_minutes,
		            emergency_override, overridden_by, overridden_at, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, m.ID, m.OrgID, m.Title, m.Message, m.StartsAt, m.EndsAt,
		m.NotifyBeforeMinutes, m.NotificationSent, m.ReadOnly, m.CountdownStartMinutes,
		m.EmergencyOverride, m.OverriddenBy, m.OverriddenAt, m.CreatedBy, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create maintenance window: %w", err)
	}
	return nil
}

// CreateMembership creates a new organization membership.
func (db *DB) CreateMembership(ctx context.Context, m *models.OrgMembership) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO org_memberships (id, user_id, org_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, m.ID, m.UserID, m.OrgID, string(m.Role), m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create membership: %w", err)
	}
	return nil
}

// CreateNotificationChannel creates a new notification channel.
func (db *DB) CreateNotificationChannel(ctx context.Context, channel *models.NotificationChannel) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO notification_channels (id, org_id, name, type, config_encrypted, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, channel.ID, channel.OrgID, channel.Name, string(channel.Type), channel.ConfigEncrypted,
		channel.Enabled, channel.CreatedAt, channel.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create notification channel: %w", err)
	}
	return nil
}

// CreateRansomwareSettings creates new ransomware settings for a schedule.
func (db *DB) CreateRansomwareSettings(ctx context.Context, settings *models.RansomwareSettings) error {
	extensionsBytes, err := settings.ExtensionsJSON()
	if err != nil {
		return fmt.Errorf("marshal extensions: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO ransomware_settings (id, schedule_id, enabled, change_threshold_percent,
		                                  extensions_to_detect, entropy_detection_enabled,
		                                  entropy_threshold, auto_pause_on_alert,
		                                  created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, settings.ID, settings.ScheduleID, settings.Enabled, settings.ChangeThresholdPercent,
		extensionsBytes, settings.EntropyDetectionEnabled, settings.EntropyThreshold,
		settings.AutoPauseOnAlert, settings.CreatedAt, settings.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create ransomware settings: %w", err)
	}
	return nil
}

// CreateRegistrationCode creates a new agent registration code.
func (db *DB) CreateRegistrationCode(ctx context.Context, code *models.RegistrationCode) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO agent_registration_codes (id, org_id, created_by, code, hostname, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, code.ID, code.OrgID, code.CreatedBy, code.Code, code.Hostname, code.ExpiresAt, code.CreatedAt)
	if err != nil {
		return fmt.Errorf("create registration code: %w", err)
	}
	return nil
}

// CreateReportSchedule creates a new report schedule.
func (db *DB) CreateReportSchedule(ctx context.Context, schedule *models.ReportSchedule) error {
	recipientsBytes, err := toJSONBytes(schedule.Recipients)
	if err != nil {
		return fmt.Errorf("marshal recipients: %w", err)
	}
	if recipientsBytes == nil {
		recipientsBytes = []byte("[]")
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO report_schedules (id, org_id, name, frequency, recipients, channel_id,
		                               timezone, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, schedule.ID, schedule.OrgID, schedule.Name, string(schedule.Frequency),
		recipientsBytes, schedule.ChannelID, schedule.Timezone, schedule.Enabled,
		schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create report schedule: %w", err)
	}
	return nil
}

// CreateRepositoryKey creates a new repository key.
func (db *DB) CreateRepositoryKey(ctx context.Context, rk *models.RepositoryKey) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO repository_keys (id, repository_id, encrypted_key, escrow_enabled, escrow_encrypted_key, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, rk.ID, rk.RepositoryID, rk.EncryptedKey, rk.EscrowEnabled,
		rk.EscrowEncryptedKey, rk.CreatedAt, rk.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create repository key: %w", err)
	}
	return nil
}

// CreateSavedFilter creates a new saved filter.
func (db *DB) CreateSavedFilter(ctx context.Context, f *models.SavedFilter) error {
	return db.ExecTx(ctx, func(tx pgx.Tx) error {
		// If setting as default, clear existing default for this user/entity type
		if f.IsDefault {
			_, err := tx.Exec(ctx, `
				UPDATE saved_filters
				SET is_default = FALSE, updated_at = NOW()
				WHERE user_id = $1 AND org_id = $2 AND entity_type = $3 AND is_default = TRUE
			`, f.UserID, f.OrgID, f.EntityType)
			if err != nil {
				return fmt.Errorf("clear existing default: %w", err)
			}
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO saved_filters (id, user_id, org_id, name, entity_type, filters, shared, is_default, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`, f.ID, f.UserID, f.OrgID, f.Name, f.EntityType, f.Filters, f.Shared, f.IsDefault, f.CreatedAt, f.UpdatedAt)
		if err != nil {
			return fmt.Errorf("create saved filter: %w", err)
		}
		return nil
	})
}

// CreateSLABreach creates a new breach record.
func (db *DB) CreateSLABreach(ctx context.Context, b *models.SLABreach) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_breaches (id, org_id, sla_id, agent_id, repository_id, breach_type, expected_value, actual_value,
		            breach_start, breach_end, duration_minutes, acknowledged, acknowledged_by, acknowledged_at,
		            resolved, resolved_at, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, b.ID, b.OrgID, b.SLAID, b.AgentID, b.RepositoryID, b.BreachType, b.ExpectedValue, b.ActualValue,
		b.BreachStart, b.BreachEnd, b.DurationMinutes, b.Acknowledged, b.AcknowledgedBy, b.AcknowledgedAt,
		b.Resolved, b.ResolvedAt, b.Description, b.CreatedAt)
	if err != nil {
		return fmt.Errorf("create sla breach: %w", err)
	}
	return nil
}

// CreateSnapshotComment creates a new snapshot comment.
func (db *DB) CreateSnapshotComment(ctx context.Context, comment *models.SnapshotComment) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO snapshot_comments (id, org_id, snapshot_id, user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, comment.ID, comment.OrgID, comment.SnapshotID, comment.UserID, comment.Content, comment.CreatedAt, comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create snapshot comment: %w", err)
	}
	return nil
}

// UpdateSnapshotComment updates a snapshot comment's content.
func (db *DB) UpdateSnapshotComment(ctx context.Context, comment *models.SnapshotComment) error {
	comment.UpdatedAt = time.Now()
	tag, err := db.Pool.Exec(ctx, `
		UPDATE snapshot_comments SET content = $2, updated_at = $3
		WHERE id = $1
	`, comment.ID, comment.Content, comment.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update snapshot comment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("snapshot comment not found")
	}
	return nil
}

// CreateSSOGroupMapping creates a new SSO group mapping.
func (db *DB) CreateSSOGroupMapping(ctx context.Context, m *models.SSOGroupMapping) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sso_group_mappings (id, org_id, oidc_group_name, role, auto_create_org, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, m.ID, m.OrgID, m.OIDCGroupName, string(m.Role), m.AutoCreateOrg, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create SSO group mapping: %w", err)
	}
	return nil
}

// CreateStorageTierConfig creates a new tier configuration.
func (db *DB) CreateStorageTierConfig(ctx context.Context, config *models.StorageTierConfig) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO storage_tier_configs (id, org_id, tier_type, name, description, cost_per_gb_month,
		            retrieval_cost, retrieval_time, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, config.ID, config.OrgID, string(config.TierType), config.Name, config.Description,
		config.CostPerGBMonth, config.RetrievalCost, config.RetrievalTime, config.Enabled,
		config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tier config: %w", err)
	}
	return nil
}

// CreateTag creates a new tag.
func (db *DB) CreateTag(ctx context.Context, tag *models.Tag) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO tags (id, org_id, name, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tag.ID, tag.OrgID, tag.Name, tag.Color, tag.CreatedAt, tag.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tag: %w", err)
	}
	return nil
}

// CreateVerificationSchedule creates a new verification schedule.
func (db *DB) CreateVerificationSchedule(ctx context.Context, vs *models.VerificationSchedule) error {
	var readDataSubset *string
	if vs.ReadDataSubset != "" {
		readDataSubset = &vs.ReadDataSubset
	}

	_, err := db.Pool.Exec(ctx, `
		INSERT INTO verification_schedules (id, repository_id, type, cron_expression, enabled, read_data_subset, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, vs.ID, vs.RepositoryID, string(vs.Type), vs.CronExpression,
		vs.Enabled, readDataSubset, vs.CreatedAt, vs.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create verification schedule: %w", err)
	}
	return nil
}

// GetActiveRansomwareAlertCountByOrgID returns the count of active ransomware alerts.
func (db *DB) GetActiveRansomwareAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ransomware_alerts
		WHERE org_id = $1 AND status IN ('active', 'investigating')
	`, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count active ransomware alerts: %w", err)
	}
	return count, nil
}

// GetAgentLogs returns logs for an agent with optional filtering.
func (db *DB) GetAgentLogs(ctx context.Context, agentID uuid.UUID, filter *models.AgentLogFilter) ([]*models.AgentLog, int, error) {
	if filter == nil {
		filter = &models.AgentLogFilter{}
	}
	if filter.Limit <= 0 {
		filter.Limit = 100
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000
	}

	// Build the query with optional filters
	var args []interface{}
	args = append(args, agentID)
	argIndex := 2

	whereClause := "WHERE agent_id = $1"

	if filter.Level != "" {
		whereClause += fmt.Sprintf(" AND level = $%d", argIndex)
		args = append(args, string(filter.Level))
		argIndex++
	}

	if filter.Component != "" {
		whereClause += fmt.Sprintf(" AND component = $%d", argIndex)
		args = append(args, filter.Component)
		argIndex++
	}

	if filter.Search != "" {
		whereClause += fmt.Sprintf(" AND to_tsvector('english', message) @@ plainto_tsquery('english', $%d)", argIndex)
		args = append(args, filter.Search)
		argIndex++
	}

	if !filter.Since.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp >= $%d", argIndex)
		args = append(args, filter.Since)
		argIndex++
	}

	if !filter.Until.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", argIndex)
		args = append(args, filter.Until)
		argIndex++
	}

	// Count total matching logs
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM agent_logs %s", whereClause)
	var totalCount int
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count agent logs: %w", err)
	}

	// Fetch logs with pagination
	args = append(args, filter.Limit, filter.Offset)
	query := fmt.Sprintf(`
		SELECT id, agent_id, org_id, level, message, component, metadata, timestamp, created_at
		FROM agent_logs
		%s
		ORDER BY timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("get agent logs: %w", err)
	}
	defer rows.Close()

	var logs []*models.AgentLog
	for rows.Next() {
		var log models.AgentLog
		var levelStr string
		var metadataBytes []byte
		err := rows.Scan(
			&log.ID, &log.AgentID, &log.OrgID, &levelStr,
			&log.Message, &log.Component, &metadataBytes,
			&log.Timestamp, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan agent log: %w", err)
		}
		log.Level = models.LogLevel(levelStr)
		if len(metadataBytes) > 0 {
			if err := log.SetMetadata(metadataBytes); err != nil {
				db.logger.Warn().Err(err).Str("log_id", log.ID.String()).Msg("failed to parse log metadata")
			}
		}
		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate agent logs: %w", err)
	}

	return logs, totalCount, nil
}

// GetAllBackups returns all backups across all organizations (for Prometheus metrics).
func (db *DB) GetAllBackups(ctx context.Context) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		ORDER BY started_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("get all backups: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetAllStorageGrowth returns aggregated storage growth for all repositories in an org.
func (db *DB) GetAllStorageGrowth(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthPoint, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DATE(s.collected_at) as date,
		       SUM(latest.raw_data_size) as raw_data_size,
		       SUM(latest.restore_size) as restore_size
		FROM (
			SELECT DISTINCT ON (repository_id, DATE(collected_at))
				repository_id, collected_at, raw_data_size, restore_size
			FROM storage_stats
			WHERE repository_id IN (SELECT id FROM repositories WHERE org_id = $1)
			  AND collected_at >= NOW() - INTERVAL '1 day' * $2
			ORDER BY repository_id, DATE(collected_at), collected_at DESC
		) as latest
		JOIN storage_stats s ON s.id = (
			SELECT id FROM storage_stats
			WHERE repository_id = latest.repository_id
			  AND DATE(collected_at) = DATE(latest.collected_at)
			ORDER BY collected_at DESC
			LIMIT 1
		)
		GROUP BY DATE(s.collected_at)
		ORDER BY date ASC
	`, orgID, days)
	if err != nil {
		return nil, fmt.Errorf("get all storage growth: %w", err)
	}
	defer rows.Close()

	var points []*models.StorageGrowthPoint
	for rows.Next() {
		var p models.StorageGrowthPoint
		err := rows.Scan(&p.Date, &p.RawDataSize, &p.RestoreSize)
		if err != nil {
			return nil, fmt.Errorf("scan all storage growth: %w", err)
		}
		points = append(points, &p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all storage growth: %w", err)
	}

	return points, nil
}

// GetBackupsByOrgIDAndDateRange returns backups for an org within a date range.
func (db *DB) GetBackupsByOrgIDAndDateRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT b.id, b.schedule_id, b.agent_id, b.repository_id, b.snapshot_id, b.started_at,
		       b.completed_at, b.status, b.size_bytes, b.files_new,
		       b.files_changed, b.error_message,
		       b.retention_applied, b.snapshots_removed, b.snapshots_kept, b.retention_error, b.created_at
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1 AND b.started_at >= $2 AND b.started_at <= $3
		ORDER BY b.started_at DESC
	`, orgID, start, end)
	if err != nil {
		return nil, fmt.Errorf("get backups by date range: %w", err)
	}
	defer rows.Close()
	return scanBackups(rows)
}

// GetMembershipByUserAndOrg returns a user's membership in an organization.
func (db *DB) GetMembershipByUserAndOrg(ctx context.Context, userID, orgID uuid.UUID) (*models.OrgMembership, error) {
	var m models.OrgMembership
	var roleStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, user_id, org_id, role, created_at, updated_at
		FROM org_memberships
		WHERE user_id = $1 AND org_id = $2
	`, userID, orgID).Scan(&m.ID, &m.UserID, &m.OrgID, &roleStr, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get membership: %w", err)
	}
	m.Role = models.OrgRole(roleStr)
	return &m, nil
}

// GetMembershipsByUserID returns all memberships for a user.
func (db *DB) GetMembershipsByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OrgMembership, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, user_id, org_id, role, created_at, updated_at
		FROM org_memberships
		WHERE user_id = $1
		ORDER BY created_at
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list memberships: %w", err)
	}
	defer rows.Close()

	var memberships []*models.OrgMembership
	for rows.Next() {
		var m models.OrgMembership
		var roleStr string
		if err := rows.Scan(&m.ID, &m.UserID, &m.OrgID, &roleStr, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan membership: %w", err)
		}
		m.Role = models.OrgRole(roleStr)
		memberships = append(memberships, &m)
	}
	return memberships, nil
}

// GetNotificationChannelByID returns a notification channel by ID.
func (db *DB) GetNotificationChannelByID(ctx context.Context, id uuid.UUID) (*models.NotificationChannel, error) {
	var c models.NotificationChannel
	var typeStr string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, name, type, config_encrypted, enabled, created_at, updated_at
		FROM notification_channels
		WHERE id = $1
	`, id).Scan(
		&c.ID, &c.OrgID, &c.Name, &typeStr, &c.ConfigEncrypted,
		&c.Enabled, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get notification channel: %w", err)
	}
	c.Type = models.NotificationChannelType(typeStr)
	return &c, nil
}

// GetOnboardingProgress returns the onboarding progress for an organization.
func (db *DB) GetOnboardingProgress(ctx context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error) {
	var p models.OnboardingProgress
	var currentStepStr string
	var completedStepsArr []string
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, current_step, completed_steps, skipped, completed_at, created_at, updated_at
		FROM onboarding_progress
		WHERE org_id = $1
	`, orgID).Scan(
		&p.ID, &p.OrgID, &currentStepStr, &completedStepsArr,
		&p.Skipped, &p.CompletedAt, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get onboarding progress: %w", err)
	}
	p.CurrentStep = models.OnboardingStep(currentStepStr)
	p.CompletedSteps = make([]models.OnboardingStep, len(completedStepsArr))
	for i, s := range completedStepsArr {
		p.CompletedSteps[i] = models.OnboardingStep(s)
	}
	return &p, nil
}

// GetRepositoryKeyByRepositoryID returns a repository key by repository ID.
func (db *DB) GetRepositoryKeyByRepositoryID(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryKey, error) {
	var rk models.RepositoryKey
	err := db.Pool.QueryRow(ctx, `
		SELECT id, repository_id, encrypted_key, escrow_enabled, escrow_encrypted_key, created_at, updated_at
		FROM repository_keys
		WHERE repository_id = $1
	`, repositoryID).Scan(
		&rk.ID, &rk.RepositoryID, &rk.EncryptedKey, &rk.EscrowEnabled,
		&rk.EscrowEncryptedKey, &rk.CreatedAt, &rk.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get repository key: %w", err)
	}
	return &rk, nil
}

// =============================================================================
// Second batch of recovered methods
// =============================================================================

// CountOrganizations returns the total number of organizations.
func (db *DB) CountOrganizations(ctx context.Context) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count organizations: %w", err)
	}
	return count, nil
}

// CreateAgentCommand creates a new agent command.
func (db *DB) CreateAgentCommand(ctx context.Context, cmd *models.AgentCommand) error {
	payloadBytes, err := cmd.PayloadJSON()
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO agent_commands (id, agent_id, org_id, type, status, payload, created_by,
		                            timeout_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, cmd.ID, cmd.AgentID, cmd.OrgID, string(cmd.Type), string(cmd.Status),
		payloadBytes, cmd.CreatedBy, cmd.TimeoutAt, cmd.CreatedAt, cmd.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent command: %w", err)
	}
	return nil
}

// CreateAgentGroup creates a new agent group.
func (db *DB) CreateAgentGroup(ctx context.Context, group *models.AgentGroup) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO agent_groups (id, org_id, name, description, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, group.ID, group.OrgID, group.Name, group.Description, group.Color,
		group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create agent group: %w", err)
	}
	return nil
}

// CreateAnnouncementDismissal records a user's dismissal of an announcement.
func (db *DB) CreateAnnouncementDismissal(ctx context.Context, d *models.AnnouncementDismissal) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO announcement_dismissals (id, org_id, announcement_id, user_id, dismissed_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (announcement_id, user_id) DO NOTHING
	`, d.ID, d.OrgID, d.AnnouncementID, d.UserID, d.DismissedAt)
	if err != nil {
		return fmt.Errorf("create announcement dismissal: %w", err)
	}
	return nil
}

// CreateCostEstimate creates a new cost estimate record.
func (db *DB) CreateCostEstimate(ctx context.Context, e *models.CostEstimateRecord) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO cost_estimates (id, org_id, repository_id, storage_size_bytes,
		             monthly_cost, yearly_cost, cost_per_gb, estimated_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, e.ID, e.OrgID, e.RepositoryID, e.StorageSizeBytes,
		e.MonthlyCost, e.YearlyCost, e.CostPerGB, e.EstimatedAt, e.CreatedAt)
	if err != nil {
		return fmt.Errorf("create cost estimate: %w", err)
	}
	return nil
}

// CreateDRTestSchedule creates a new DR test schedule.
func (db *DB) CreateDRTestSchedule(ctx context.Context, schedule *models.DRTestSchedule) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO dr_test_schedules (id, runbook_id, cron_expression, enabled, last_run_at, next_run_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, schedule.ID, schedule.RunbookID, schedule.CronExpression, schedule.Enabled,
		schedule.LastRunAt, schedule.NextRunAt, schedule.CreatedAt, schedule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create DR test schedule: %w", err)
	}
	return nil
}

// CreateImportedSnapshots creates multiple imported snapshot records in a batch.
func (db *DB) CreateImportedSnapshots(ctx context.Context, snapshots []*models.ImportedSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}

	for _, snap := range snapshots {
		if err := db.CreateImportedSnapshot(ctx, snap); err != nil {
			return err
		}
	}
	return nil
}

// CreateLifecyclePolicy creates a new lifecycle policy.
func (db *DB) CreateLifecyclePolicy(ctx context.Context, policy *models.LifecyclePolicy) error {
	rulesJSON, err := policy.RulesJSON()
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}

	repoIDsJSON, err := policy.RepositoryIDsJSON()
	if err != nil {
		return fmt.Errorf("marshal repository_ids: %w", err)
	}

	scheduleIDsJSON, err := policy.ScheduleIDsJSON()
	if err != nil {
		return fmt.Errorf("marshal schedule_ids: %w", err)
	}

	_, err = db.Pool.Exec(ctx, `
		INSERT INTO lifecycle_policies (
			id, org_id, name, description, status, rules, repository_ids, schedule_ids,
			deletion_count, bytes_reclaimed, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, policy.ID, policy.OrgID, policy.Name, policy.Description, policy.Status,
		rulesJSON, repoIDsJSON, scheduleIDsJSON, policy.DeletionCount, policy.BytesReclaimed,
		policy.CreatedBy, policy.CreatedAt, policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create lifecycle policy: %w", err)
	}
	return nil
}

// CreateNotificationPreference creates a new notification preference.
func (db *DB) CreateNotificationPreference(ctx context.Context, pref *models.NotificationPreference) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO notification_preferences (id, org_id, channel_id, event_type, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, pref.ID, pref.OrgID, pref.ChannelID, string(pref.EventType), pref.Enabled,
		pref.CreatedAt, pref.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create notification preference: %w", err)
	}
	return nil
}

// CreateOrganization creates a new organization.
func (db *DB) CreateOrganization(ctx context.Context, org *models.Organization) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`, org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

// CreateRateLimitConfig creates a new rate limit config.
func (db *DB) CreateRateLimitConfig(ctx context.Context, c *models.RateLimitConfig) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO rate_limit_configs (id, org_id, endpoint, requests_per_period, period_seconds, enabled, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, c.ID, c.OrgID, c.Endpoint, c.RequestsPerPeriod, c.PeriodSeconds, c.Enabled, c.CreatedBy, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create rate limit config: %w", err)
	}
	return nil
}

// CreateSLACompliance creates a new compliance record.
func (db *DB) CreateSLACompliance(ctx context.Context, c *models.SLACompliance) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO sla_compliance (id, org_id, sla_id, agent_id, repository_id, period_start, period_end,
		            rpo_compliant, rpo_actual_minutes, rpo_breaches, rto_compliant, rto_actual_minutes, rto_breaches,
		            uptime_compliant, uptime_actual_percentage, uptime_downtime_minutes, is_compliant, notes, calculated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, c.ID, c.OrgID, c.SLAID, c.AgentID, c.RepositoryID, c.PeriodStart, c.PeriodEnd,
		c.RPOCompliant, c.RPOActualMinutes, c.RPOBreaches, c.RTOCompliant, c.RTOActualMinutes, c.RTOBreaches,
		c.UptimeCompliant, c.UptimeActualPercentage, c.UptimeDowntimeMinutes, c.IsCompliant, c.Notes, c.CalculatedAt)
	if err != nil {
		return fmt.Errorf("create sla compliance: %w", err)
	}
	return nil
}

// CreateSnapshotMount creates a new snapshot mount record.
func (db *DB) CreateSnapshotMount(ctx context.Context, mount *models.SnapshotMount) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO snapshot_mounts (
			id, org_id, agent_id, repository_id, snapshot_id, mount_path,
			status, mounted_at, expires_at, unmounted_at, error_message,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, mount.ID, mount.OrgID, mount.AgentID, mount.RepositoryID, mount.SnapshotID,
		mount.MountPath, string(mount.Status), mount.MountedAt, mount.ExpiresAt,
		mount.UnmountedAt, mount.ErrorMessage, mount.CreatedAt, mount.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create snapshot mount: %w", err)
	}
	return nil
}

// CreateTierRule creates a new tier rule.
func (db *DB) CreateTierRule(ctx context.Context, rule *models.TierRule) error {
	_, err := db.Pool.Exec(ctx, `
		INSERT INTO tier_rules (id, org_id, repository_id, schedule_id, name, description,
		            from_tier, to_tier, age_threshold_days, min_copies, priority, enabled,
		            created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, rule.ID, rule.OrgID, rule.RepositoryID, rule.ScheduleID, rule.Name, rule.Description,
		string(rule.FromTier), string(rule.ToTier), rule.AgeThresholdDay, rule.MinCopies,
		rule.Priority, rule.Enabled, rule.CreatedAt, rule.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create tier rule: %w", err)
	}
	return nil
}

// DeleteConfigTemplate deletes a config template.
func (db *DB) DeleteConfigTemplate(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM config_templates WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete config template: %w", err)
	}
	return nil
}

// DeleteExcludePattern deletes an exclude pattern by ID.
func (db *DB) DeleteExcludePattern(ctx context.Context, id uuid.UUID) error {
	result, err := db.Pool.Exec(ctx, `
		DELETE FROM exclude_patterns
		WHERE id = $1 AND is_builtin = false
	`, id)
	if err != nil {
		return fmt.Errorf("delete exclude pattern: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("exclude pattern not found or is built-in")
	}
	return nil
}

// DeleteExpiredRegistrationCodes removes expired registration codes.
func (db *DB) DeleteExpiredRegistrationCodes(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM agent_registration_codes
		WHERE expires_at < NOW() AND used_at IS NULL
	`)
	if err != nil {
		return fmt.Errorf("delete expired registration codes: %w", err)
	}
	return nil
}

// DeleteFavorite deletes a favorite by user, entity type and entity ID.
func (db *DB) DeleteFavorite(ctx context.Context, userID uuid.UUID, entityType string, entityID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM user_favorites WHERE user_id = $1 AND entity_type = $2 AND entity_id = $3
	`, userID, entityType, entityID)
	if err != nil {
		return fmt.Errorf("delete favorite: %w", err)
	}
	return nil
}

// DeleteGeoReplicationConfig deletes a geo-replication configuration.
func (db *DB) DeleteGeoReplicationConfig(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM geo_replication_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete geo-replication config: %w", err)
	}
	return nil
}

// DeleteIPAllowlist deletes an IP allowlist entry.
func (db *DB) DeleteIPAllowlist(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM ip_allowlists WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ip allowlist: %w", err)
	}
	return nil
}

// DeleteLegalHold removes a legal hold by ID.
func (db *DB) DeleteLegalHold(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM legal_holds WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete legal hold: %w", err)
	}
	return nil
}

// DeleteMaintenanceWindow deletes a maintenance window.
func (db *DB) DeleteMaintenanceWindow(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `
		DELETE FROM maintenance_windows
		WHERE id = $1
	`, id)
	if err != nil {
		return fmt.Errorf("delete maintenance window: %w", err)
	}
	return nil
}

// DeleteRansomwareSettings deletes ransomware settings by ID.
func (db *DB) DeleteRansomwareSettings(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM ransomware_settings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete ransomware settings: %w", err)
	}
	return nil
}

// DeleteReportSchedule deletes a report schedule by ID.
func (db *DB) DeleteReportSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM report_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete report schedule: %w", err)
	}
	return nil
}

// DeleteSSOGroupMapping deletes an SSO group mapping.
func (db *DB) DeleteSSOGroupMapping(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM sso_group_mappings WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete SSO group mapping: %w", err)
	}
	return nil
}

// DeleteSavedFilter deletes a saved filter.
func (db *DB) DeleteSavedFilter(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM saved_filters WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete saved filter: %w", err)
	}
	return nil
}

// DeleteTag deletes a tag by ID.
func (db *DB) DeleteTag(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM tags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}

// DeleteVerificationSchedule deletes a verification schedule by ID.
func (db *DB) DeleteVerificationSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM verification_schedules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete verification schedule: %w", err)
	}
	return nil
}

// GetAgentCommandByID returns a command by ID.
func (db *DB) GetAgentCommandByID(ctx context.Context, id uuid.UUID) (*models.AgentCommand, error) {
	var cmd models.AgentCommand
	var typeStr, statusStr string
	var payloadBytes, resultBytes []byte

	err := db.Pool.QueryRow(ctx, `
		SELECT c.id, c.agent_id, c.org_id, c.type, c.status, c.payload, c.result,
		       c.created_by, c.acknowledged_at, c.started_at, c.completed_at,
		       c.timeout_at, c.created_at, c.updated_at,
		       COALESCE(u.name, '')
		FROM agent_commands c
		LEFT JOIN users u ON c.created_by = u.id
		WHERE c.id = $1
	`, id).Scan(
		&cmd.ID, &cmd.AgentID, &cmd.OrgID, &typeStr, &statusStr,
		&payloadBytes, &resultBytes, &cmd.CreatedBy,
		&cmd.AcknowledgedAt, &cmd.StartedAt, &cmd.CompletedAt,
		&cmd.TimeoutAt, &cmd.CreatedAt, &cmd.UpdatedAt,
		&cmd.CreatedByName,
	)
	if err != nil {
		return nil, fmt.Errorf("get agent command: %w", err)
	}

	cmd.Type = models.CommandType(typeStr)
	cmd.Status = models.CommandStatus(statusStr)
	if err := cmd.SetPayload(payloadBytes); err != nil {
		return nil, fmt.Errorf("parse payload: %w", err)
	}
	if err := cmd.SetResult(resultBytes); err != nil {
		return nil, fmt.Errorf("parse result: %w", err)
	}

	return &cmd, nil
}

// GetBackupDurationTrend returns backup duration trends over time.
func (db *DB) GetBackupDurationTrend(ctx context.Context, orgID uuid.UUID, days int) ([]*models.BackupDurationTrend, error) {
	since := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	rows, err := db.Pool.Query(ctx, `
		SELECT
			DATE(b.started_at) AS date,
			AVG(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)) * 1000)::BIGINT AS avg_duration_ms,
			MAX(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)) * 1000)::BIGINT AS max_duration_ms,
			MIN(EXTRACT(EPOCH FROM (b.completed_at - b.started_at)) * 1000)::BIGINT AS min_duration_ms,
			COUNT(*) AS backup_count
		FROM backups b
		JOIN schedules s ON b.schedule_id = s.id
		JOIN agents a ON s.agent_id = a.id
		WHERE a.org_id = $1
		  AND b.started_at >= $2
		  AND b.completed_at IS NOT NULL
		  AND b.status = 'completed'
		GROUP BY DATE(b.started_at)
		ORDER BY date ASC
	`, orgID, since)
	if err != nil {
		return nil, fmt.Errorf("get backup duration trend: %w", err)
	}
	defer rows.Close()

	var trends []*models.BackupDurationTrend
	for rows.Next() {
		var t models.BackupDurationTrend
		err := rows.Scan(&t.Date, &t.AvgDurationMs, &t.MaxDurationMs, &t.MinDurationMs, &t.BackupCount)
		if err != nil {
			return nil, fmt.Errorf("scan backup duration: %w", err)
		}
		trends = append(trends, &t)
	}

	return trends, nil
}

// GetBackupsByStatus returns all backups with the specified status (for Prometheus metrics).
func (db *DB) GetBackupsByStatus(ctx context.Context, status models.BackupStatus) ([]*models.Backup, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, schedule_id, agent_id, repository_id, snapshot_id, started_at, completed_at,
		       status, size_bytes, files_new, files_changed, error_message,
		       retention_applied, snapshots_removed, snapshots_kept, retention_error, created_at
		FROM backups
		WHERE status = $1
		ORDER BY started_at DESC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("get backups by status: %w", err)
	}
	defer rows.Close()

	return scanBackups(rows)
}

// GetDockerContainers returns Docker containers for the given agent.
func (db *DB) GetDockerContainers(ctx context.Context, orgID uuid.UUID, agentID uuid.UUID) ([]models.DockerContainer, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT container_id, name, image, status, state, created, ports
		FROM docker_containers
		WHERE org_id = $1 AND agent_id = $2
		ORDER BY name ASC
	`, orgID, agentID)
	if err != nil {
		return nil, fmt.Errorf("get docker containers: %w", err)
	}
	defer rows.Close()

	var containers []models.DockerContainer
	for rows.Next() {
		var c models.DockerContainer
		var portsJSON []byte
		if err := rows.Scan(&c.ID, &c.Name, &c.Image, &c.Status, &c.State, &c.Created, &portsJSON); err != nil {
			return nil, fmt.Errorf("scan docker container: %w", err)
		}
		if portsJSON != nil {
			if err := json.Unmarshal(portsJSON, &c.Ports); err != nil {
				return nil, fmt.Errorf("unmarshal ports: %w", err)
			}
		}
		containers = append(containers, c)
	}
	return containers, nil
}

// GetDRRunbookByID returns a DR runbook by ID.
func (db *DB) GetDRRunbookByID(ctx context.Context, id uuid.UUID) (*models.DRRunbook, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, schedule_id, name, description, steps, contacts,
		       credentials_location, recovery_time_objective_minutes,
		       recovery_point_objective_minutes, status, created_at, updated_at
		FROM dr_runbooks
		WHERE id = $1
	`, id)

	return scanDRRunbook(row)
}

// scanDRRunbook scans a DR runbook from a row.
func scanDRRunbook(row interface{ Scan(dest ...any) error }) (*models.DRRunbook, error) {
	var r models.DRRunbook
	var stepsBytes, contactsBytes []byte
	var statusStr string
	err := row.Scan(
		&r.ID, &r.OrgID, &r.ScheduleID, &r.Name, &r.Description,
		&stepsBytes, &contactsBytes, &r.CredentialsLocation,
		&r.RecoveryTimeObjectiveMins, &r.RecoveryPointObjectiveMins,
		&statusStr, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan DR runbook: %w", err)
	}

	r.Status = models.DRRunbookStatus(statusStr)
	if err := r.SetSteps(stepsBytes); err != nil {
		return nil, fmt.Errorf("parse steps: %w", err)
	}
	if err := r.SetContacts(contactsBytes); err != nil {
		return nil, fmt.Errorf("parse contacts: %w", err)
	}

	return &r, nil
}

// GetIPAllowlistSettings returns the IP allowlist settings for an organization.
func (db *DB) GetIPAllowlistSettings(ctx context.Context, orgID uuid.UUID) (*models.IPAllowlistSettings, error) {
	var s models.IPAllowlistSettings
	err := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, enabled, enforce_for_ui, enforce_for_agent, allow_admin_bypass, created_at, updated_at
		FROM ip_allowlist_settings
		WHERE org_id = $1
	`, orgID).Scan(
		&s.ID, &s.OrgID, &s.Enabled, &s.EnforceForUI, &s.EnforceForAgent, &s.AllowAdminBypass, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get ip allowlist settings: %w", err)
	}
	return &s, nil
}

// GetLatestStatsForAllRepos returns the latest storage stats for all repositories in an org.
func (db *DB) GetLatestStatsForAllRepos(ctx context.Context, orgID uuid.UUID) ([]*models.StorageStats, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT DISTINCT ON (s.repository_id)
			s.id, s.repository_id, s.total_size, s.total_file_count, s.raw_data_size,
			s.restore_size, s.dedup_ratio, s.space_saved, s.space_saved_pct,
			s.snapshot_count, s.collected_at, s.created_at
		FROM storage_stats s
		JOIN repositories r ON s.repository_id = r.id
		WHERE r.org_id = $1
		ORDER BY s.repository_id, s.collected_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("get latest stats for all repos: %w", err)
	}
	defer rows.Close()

	return scanStorageStats(rows)
}

// GetOrCreateIPAllowlistSettings returns existing settings or creates default settings.
func (db *DB) GetOrCreateIPAllowlistSettings(ctx context.Context, orgID uuid.UUID) (*models.IPAllowlistSettings, error) {
	settings, err := db.GetIPAllowlistSettings(ctx, orgID)
	if err == nil {
		return settings, nil
	}

	// Create default settings
	settings = models.NewIPAllowlistSettings(orgID)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO ip_allowlist_settings (id, org_id, enabled, enforce_for_ui, enforce_for_agent, allow_admin_bypass, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (org_id) DO NOTHING
	`, settings.ID, settings.OrgID, settings.Enabled, settings.EnforceForUI, settings.EnforceForAgent, settings.AllowAdminBypass, settings.CreatedAt, settings.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create ip allowlist settings: %w", err)
	}

	// Re-fetch to get the actual values (in case of conflict)
	return db.GetIPAllowlistSettings(ctx, orgID)
}

// GetOrCreateOnboardingProgress returns existing progress or creates new progress for an organization.
func (db *DB) GetOrCreateOnboardingProgress(ctx context.Context, orgID uuid.UUID) (*models.OnboardingProgress, error) {
	// Try to get existing progress
	p, err := db.GetOnboardingProgress(ctx, orgID)
	if err == nil {
		return p, nil
	}

	// Create new progress
	progress := models.NewOnboardingProgress(orgID)
	_, err = db.Pool.Exec(ctx, `
		INSERT INTO onboarding_progress (id, org_id, current_step, completed_steps, skipped, completed_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, progress.ID, progress.OrgID, string(progress.CurrentStep), []string{}, progress.Skipped, progress.CompletedAt, progress.CreatedAt, progress.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create onboarding progress: %w", err)
	}

	db.logger.Info().Str("org_id", orgID.String()).Msg("created onboarding progress")
	return progress, nil
}

// GetOrganizationSSOSettings returns an organization's SSO settings.
func (db *DB) GetOrganizationSSOSettings(ctx context.Context, orgID uuid.UUID) (defaultRole *string, autoCreateOrgs bool, err error) {
	err = db.Pool.QueryRow(ctx, `
		SELECT sso_default_role, sso_auto_create_orgs
		FROM organizations
		WHERE id = $1
	`, orgID).Scan(&defaultRole, &autoCreateOrgs)
	if err != nil {
		return nil, false, fmt.Errorf("get org SSO settings: %w", err)
	}
	return defaultRole, autoCreateOrgs, nil
}

// GetRepositoryKeysWithEscrowByOrgID returns all repository keys with escrow enabled for an organization.
func (db *DB) GetRepositoryKeysWithEscrowByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.RepositoryKey, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT rk.id, rk.repository_id, rk.encrypted_key, rk.escrow_enabled, rk.escrow_encrypted_key, rk.created_at, rk.updated_at
		FROM repository_keys rk
		INNER JOIN repositories r ON rk.repository_id = r.id
		WHERE r.org_id = $1 AND rk.escrow_enabled = true
		ORDER BY rk.created_at DESC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list repository keys with escrow: %w", err)
	}
	defer rows.Close()

	var keys []*models.RepositoryKey
	for rows.Next() {
		var rk models.RepositoryKey
		err := rows.Scan(
			&rk.ID, &rk.RepositoryID, &rk.EncryptedKey, &rk.EscrowEnabled,
			&rk.EscrowEncryptedKey, &rk.CreatedAt, &rk.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan repository key: %w", err)
		}
		keys = append(keys, &rk)
	}

	return keys, nil
}

// UpdateMembership updates an existing membership.
func (db *DB) UpdateMembership(ctx context.Context, m *models.OrgMembership) error {
	m.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, `
		UPDATE org_memberships
		SET role = $3, updated_at = $4
		WHERE user_id = $1 AND org_id = $2
	`, m.UserID, m.OrgID, string(m.Role), m.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update membership: %w", err)
	}
	return nil
}
