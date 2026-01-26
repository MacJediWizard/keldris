package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// DowntimeStore defines the database operations needed by the downtime service.
type DowntimeStore interface {
	CreateDowntimeEvent(ctx context.Context, event *models.DowntimeEvent) error
	UpdateDowntimeEvent(ctx context.Context, event *models.DowntimeEvent) error
	GetDowntimeEventByID(ctx context.Context, id uuid.UUID) (*models.DowntimeEvent, error)
	GetDowntimeEventsByOrgID(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.DowntimeEvent, error)
	GetActiveDowntimeEventsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeEvent, error)
	GetDowntimeEventsByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID *uuid.UUID) ([]*models.DowntimeEvent, error)
	GetDowntimeEventsByTimeRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.DowntimeEvent, error)
	GetActiveDowntimeByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID uuid.UUID) (*models.DowntimeEvent, error)
	DeleteDowntimeEvent(ctx context.Context, id uuid.UUID) error

	// Uptime stats
	CreateOrUpdateUptimeStats(ctx context.Context, stats *models.UptimeStats) error
	GetUptimeStatsByOrgID(ctx context.Context, orgID uuid.UUID, periodStart, periodEnd time.Time) ([]*models.UptimeStats, error)
	GetUptimeStatsByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID *uuid.UUID, periodStart, periodEnd time.Time) (*models.UptimeStats, error)

	// Uptime badges
	CreateOrUpdateUptimeBadge(ctx context.Context, badge *models.UptimeBadge) error
	GetUptimeBadgesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.UptimeBadge, error)
	GetUptimeBadge(ctx context.Context, orgID uuid.UUID, componentType *models.ComponentType, componentID *uuid.UUID, badgeType models.BadgeType) (*models.UptimeBadge, error)

	// Downtime alerts
	CreateDowntimeAlert(ctx context.Context, alert *models.DowntimeAlert) error
	UpdateDowntimeAlert(ctx context.Context, alert *models.DowntimeAlert) error
	GetDowntimeAlertByID(ctx context.Context, id uuid.UUID) (*models.DowntimeAlert, error)
	GetDowntimeAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeAlert, error)
	GetEnabledDowntimeAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeAlert, error)
	DeleteDowntimeAlert(ctx context.Context, id uuid.UUID) error
}

// DowntimeServiceConfig holds the configuration for the downtime service.
type DowntimeServiceConfig struct {
	// BadgeUpdateInterval is how often to update uptime badges.
	BadgeUpdateInterval time.Duration
	// StatsAggregationInterval is how often to aggregate uptime stats.
	StatsAggregationInterval time.Duration
}

// DefaultDowntimeServiceConfig returns a DowntimeServiceConfig with sensible defaults.
func DefaultDowntimeServiceConfig() DowntimeServiceConfig {
	return DowntimeServiceConfig{
		BadgeUpdateInterval:      15 * time.Minute,
		StatsAggregationInterval: 1 * time.Hour,
	}
}

// DowntimeService manages downtime tracking and uptime calculations.
type DowntimeService struct {
	store  DowntimeStore
	config DowntimeServiceConfig
	logger zerolog.Logger
}

// NewDowntimeService creates a new DowntimeService instance.
func NewDowntimeService(store DowntimeStore, config DowntimeServiceConfig, logger zerolog.Logger) *DowntimeService {
	return &DowntimeService{
		store:  store,
		config: config,
		logger: logger.With().Str("component", "downtime_service").Logger(),
	}
}

// NewDowntimeServiceWithDB creates a DowntimeService using the database directly.
func NewDowntimeServiceWithDB(database *db.DB, config DowntimeServiceConfig, logger zerolog.Logger) *DowntimeService {
	return NewDowntimeService(database, config, logger)
}

// RecordDowntimeStart records the start of a downtime event.
func (s *DowntimeService) RecordDowntimeStart(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID *uuid.UUID, componentName string, severity models.DowntimeSeverity, cause string) (*models.DowntimeEvent, error) {
	// Check if there's already an active downtime for this component
	if componentID != nil {
		existing, err := s.store.GetActiveDowntimeByComponent(ctx, orgID, componentType, *componentID)
		if err == nil && existing != nil {
			// Already tracking downtime for this component
			s.logger.Debug().
				Str("component_type", string(componentType)).
				Str("component_id", componentID.String()).
				Msg("downtime already being tracked for component")
			return existing, nil
		}
	}

	event := models.NewDowntimeEvent(orgID, componentType, componentName, severity)
	if componentID != nil {
		event.SetComponent(*componentID)
	}
	if cause != "" {
		event.Cause = &cause
	}

	if err := s.store.CreateDowntimeEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("create downtime event: %w", err)
	}

	s.logger.Info().
		Str("event_id", event.ID.String()).
		Str("component_type", string(componentType)).
		Str("component_name", componentName).
		Str("severity", string(severity)).
		Msg("downtime event started")

	return event, nil
}

// RecordDowntimeEnd records the end of a downtime event.
func (s *DowntimeService) RecordDowntimeEnd(ctx context.Context, eventID uuid.UUID, resolvedBy *uuid.UUID, notes string) (*models.DowntimeEvent, error) {
	event, err := s.store.GetDowntimeEventByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("get downtime event: %w", err)
	}

	if !event.IsActive() {
		return event, nil // Already ended
	}

	event.End(resolvedBy)
	if notes != "" {
		event.Notes = &notes
	}

	if err := s.store.UpdateDowntimeEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("update downtime event: %w", err)
	}

	s.logger.Info().
		Str("event_id", event.ID.String()).
		Str("component_type", string(event.ComponentType)).
		Str("component_name", event.ComponentName).
		Int("duration_seconds", *event.DurationSeconds).
		Msg("downtime event ended")

	return event, nil
}

// ResolveDowntimeByComponent ends any active downtime for a specific component.
func (s *DowntimeService) ResolveDowntimeByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID uuid.UUID, resolvedBy *uuid.UUID) error {
	event, err := s.store.GetActiveDowntimeByComponent(ctx, orgID, componentType, componentID)
	if err != nil {
		return nil // No active downtime, that's fine
	}

	if event != nil {
		_, err = s.RecordDowntimeEnd(ctx, event.ID, resolvedBy, "Component recovered")
		if err != nil {
			return fmt.Errorf("end downtime event: %w", err)
		}
	}

	return nil
}

// GetDowntimeEvent retrieves a downtime event by ID.
func (s *DowntimeService) GetDowntimeEvent(ctx context.Context, id uuid.UUID) (*models.DowntimeEvent, error) {
	return s.store.GetDowntimeEventByID(ctx, id)
}

// ListDowntimeEvents returns downtime events for an organization.
func (s *DowntimeService) ListDowntimeEvents(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*models.DowntimeEvent, error) {
	return s.store.GetDowntimeEventsByOrgID(ctx, orgID, limit, offset)
}

// ListActiveDowntime returns active (ongoing) downtime events for an organization.
func (s *DowntimeService) ListActiveDowntime(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeEvent, error) {
	return s.store.GetActiveDowntimeEventsByOrgID(ctx, orgID)
}

// ListDowntimeByComponent returns downtime events for a specific component.
func (s *DowntimeService) ListDowntimeByComponent(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID *uuid.UUID) ([]*models.DowntimeEvent, error) {
	return s.store.GetDowntimeEventsByComponent(ctx, orgID, componentType, componentID)
}

// ListDowntimeByTimeRange returns downtime events within a time range.
func (s *DowntimeService) ListDowntimeByTimeRange(ctx context.Context, orgID uuid.UUID, start, end time.Time) ([]*models.DowntimeEvent, error) {
	return s.store.GetDowntimeEventsByTimeRange(ctx, orgID, start, end)
}

// UpdateDowntimeEvent updates an existing downtime event.
func (s *DowntimeService) UpdateDowntimeEvent(ctx context.Context, event *models.DowntimeEvent) error {
	event.UpdatedAt = time.Now()
	return s.store.UpdateDowntimeEvent(ctx, event)
}

// DeleteDowntimeEvent deletes a downtime event.
func (s *DowntimeService) DeleteDowntimeEvent(ctx context.Context, id uuid.UUID) error {
	return s.store.DeleteDowntimeEvent(ctx, id)
}

// CalculateUptime calculates uptime statistics for a period.
func (s *DowntimeService) CalculateUptime(ctx context.Context, orgID uuid.UUID, componentType models.ComponentType, componentID *uuid.UUID, componentName string, start, end time.Time) (*models.UptimeStats, error) {
	stats := models.NewUptimeStats(orgID, componentType, componentName, start, end)
	if componentID != nil {
		stats.ComponentID = componentID
	}

	// Get all downtime events in the period
	events, err := s.store.GetDowntimeEventsByTimeRange(ctx, orgID, start, end)
	if err != nil {
		return nil, fmt.Errorf("get downtime events: %w", err)
	}

	// Filter events for this component and calculate total downtime
	for _, event := range events {
		if event.ComponentType != componentType {
			continue
		}
		if componentID != nil && (event.ComponentID == nil || *event.ComponentID != *componentID) {
			continue
		}

		stats.IncidentCount++

		// Calculate overlap with the period
		eventStart := event.StartedAt
		if eventStart.Before(start) {
			eventStart = start
		}

		var eventEnd time.Time
		if event.EndedAt != nil {
			eventEnd = *event.EndedAt
		} else {
			eventEnd = time.Now()
		}
		if eventEnd.After(end) {
			eventEnd = end
		}

		if eventEnd.After(eventStart) {
			stats.DowntimeSeconds += int(eventEnd.Sub(eventStart).Seconds())
		}
	}

	stats.CalculateUptime()

	if err := s.store.CreateOrUpdateUptimeStats(ctx, stats); err != nil {
		return nil, fmt.Errorf("save uptime stats: %w", err)
	}

	return stats, nil
}

// GetUptimeSummary returns an uptime summary for the organization.
func (s *DowntimeService) GetUptimeSummary(ctx context.Context, orgID uuid.UUID) (*models.UptimeSummary, error) {
	now := time.Now()
	summary := &models.UptimeSummary{}

	// Get active incidents
	activeEvents, err := s.store.GetActiveDowntimeEventsByOrgID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get active downtime events: %w", err)
	}
	summary.ActiveIncidents = len(activeEvents)
	summary.RecentIncidents = activeEvents

	// Get badges
	badges, err := s.store.GetUptimeBadgesByOrgID(ctx, orgID)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to get uptime badges")
	} else {
		summary.Badges = badges
		for _, badge := range badges {
			if badge.ComponentType == nil {
				// Overall badge
				switch badge.BadgeType {
				case models.BadgeType7D:
					summary.OverallUptime7D = badge.UptimePercent
				case models.BadgeType30D:
					summary.OverallUptime30D = badge.UptimePercent
				case models.BadgeType90D:
					summary.OverallUptime90D = badge.UptimePercent
				}
			}
		}
	}

	// Get 30-day stats for component breakdown
	start30d := now.AddDate(0, 0, -30)
	stats30d, err := s.store.GetUptimeStatsByOrgID(ctx, orgID, start30d, now)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to get uptime stats")
	} else {
		for _, stat := range stats30d {
			status := "up"
			if stat.UptimePercent < 99.0 {
				status = "degraded"
			}

			// Check if currently down
			for _, event := range activeEvents {
				if event.ComponentType == stat.ComponentType &&
					(stat.ComponentID == nil || (event.ComponentID != nil && *event.ComponentID == *stat.ComponentID)) {
					status = "down"
					break
				}
			}

			component := models.ComponentUptime{
				ComponentType:    stat.ComponentType,
				ComponentID:      stat.ComponentID,
				ComponentName:    stat.ComponentName,
				Status:           status,
				UptimePercent30D: stat.UptimePercent,
				IncidentCount30D: stat.IncidentCount,
			}
			summary.ComponentBreakdown = append(summary.ComponentBreakdown, component)

			if status == "up" || status == "degraded" {
				summary.ComponentsUp++
			} else {
				summary.ComponentsDown++
			}
			summary.TotalComponents++
		}
	}

	return summary, nil
}

// UpdateUptimeBadges refreshes all uptime badges for an organization.
func (s *DowntimeService) UpdateUptimeBadges(ctx context.Context, orgID uuid.UUID) error {
	now := time.Now()
	periods := []struct {
		badgeType models.BadgeType
		days      int
	}{
		{models.BadgeType7D, 7},
		{models.BadgeType30D, 30},
		{models.BadgeType90D, 90},
		{models.BadgeType365D, 365},
	}

	for _, period := range periods {
		start := now.AddDate(0, 0, -period.days)

		// Get all downtime events in the period
		events, err := s.store.GetDowntimeEventsByTimeRange(ctx, orgID, start, now)
		if err != nil {
			return fmt.Errorf("get downtime events for %s: %w", period.badgeType, err)
		}

		// Calculate total downtime
		totalSeconds := period.days * 24 * 60 * 60
		downtimeSeconds := 0

		for _, event := range events {
			eventStart := event.StartedAt
			if eventStart.Before(start) {
				eventStart = start
			}

			var eventEnd time.Time
			if event.EndedAt != nil {
				eventEnd = *event.EndedAt
			} else {
				eventEnd = now
			}
			if eventEnd.After(now) {
				eventEnd = now
			}

			if eventEnd.After(eventStart) {
				downtimeSeconds += int(eventEnd.Sub(eventStart).Seconds())
			}
		}

		uptimePercent := float64(totalSeconds-downtimeSeconds) / float64(totalSeconds) * 100.0
		if uptimePercent < 0 {
			uptimePercent = 0
		}

		badge := models.NewUptimeBadge(orgID, period.badgeType, uptimePercent)
		if err := s.store.CreateOrUpdateUptimeBadge(ctx, badge); err != nil {
			return fmt.Errorf("update badge %s: %w", period.badgeType, err)
		}

		s.logger.Debug().
			Str("badge_type", string(period.badgeType)).
			Float64("uptime_percent", uptimePercent).
			Msg("uptime badge updated")
	}

	return nil
}

// GetMonthlyReport generates a monthly uptime report.
func (s *DowntimeService) GetMonthlyReport(ctx context.Context, orgID uuid.UUID, year int, month int) (*models.MonthlyUptimeReport, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0).Add(-time.Second)

	report := &models.MonthlyUptimeReport{
		OrgID:       orgID,
		Month:       fmt.Sprintf("%d-%02d", year, month),
		Year:        year,
		MonthNum:    month,
		GeneratedAt: time.Now(),
	}

	// Get all downtime events for the month
	events, err := s.store.GetDowntimeEventsByTimeRange(ctx, orgID, startOfMonth, endOfMonth)
	if err != nil {
		return nil, fmt.Errorf("get downtime events: %w", err)
	}

	report.IncidentCount = len(events)

	// Calculate total downtime
	totalSeconds := int(endOfMonth.Sub(startOfMonth).Seconds())
	downtimeSeconds := 0

	for _, event := range events {
		eventStart := event.StartedAt
		if eventStart.Before(startOfMonth) {
			eventStart = startOfMonth
		}

		var eventEnd time.Time
		if event.EndedAt != nil {
			eventEnd = *event.EndedAt
		} else {
			eventEnd = time.Now()
		}
		if eventEnd.After(endOfMonth) {
			eventEnd = endOfMonth
		}

		if eventEnd.After(eventStart) {
			downtimeSeconds += int(eventEnd.Sub(eventStart).Seconds())
		}
	}

	report.TotalDowntime = downtimeSeconds
	report.OverallUptime = float64(totalSeconds-downtimeSeconds) / float64(totalSeconds) * 100.0

	// Generate daily breakdown
	for day := startOfMonth; day.Before(endOfMonth); day = day.AddDate(0, 0, 1) {
		dayEnd := day.AddDate(0, 0, 1)
		if dayEnd.After(endOfMonth) {
			dayEnd = endOfMonth
		}

		dayDowntime := 0
		dayIncidents := 0

		for _, event := range events {
			eventStart := event.StartedAt
			if eventStart.Before(day) {
				eventStart = day
			}

			var eventEnd time.Time
			if event.EndedAt != nil {
				eventEnd = *event.EndedAt
			} else {
				eventEnd = time.Now()
			}
			if eventEnd.After(dayEnd) {
				eventEnd = dayEnd
			}

			if eventEnd.After(eventStart) && eventStart.Before(dayEnd) {
				dayDowntime += int(eventEnd.Sub(eventStart).Seconds())
				dayIncidents++
			}
		}

		daySeconds := int(dayEnd.Sub(day).Seconds())
		dayUptime := float64(daySeconds-dayDowntime) / float64(daySeconds) * 100.0

		report.DailyBreakdown = append(report.DailyBreakdown, models.DailyUptime{
			Date:            day.Format("2006-01-02"),
			UptimePercent:   dayUptime,
			DowntimeSeconds: dayDowntime,
			IncidentCount:   dayIncidents,
		})
	}

	return report, nil
}

// Downtime Alert Management

// CreateDowntimeAlert creates a new downtime alert.
func (s *DowntimeService) CreateDowntimeAlert(ctx context.Context, alert *models.DowntimeAlert) error {
	if err := s.store.CreateDowntimeAlert(ctx, alert); err != nil {
		return fmt.Errorf("create downtime alert: %w", err)
	}

	s.logger.Info().
		Str("alert_id", alert.ID.String()).
		Str("name", alert.Name).
		Float64("threshold", alert.UptimeThreshold).
		Msg("downtime alert created")

	return nil
}

// GetDowntimeAlert retrieves a downtime alert by ID.
func (s *DowntimeService) GetDowntimeAlert(ctx context.Context, id uuid.UUID) (*models.DowntimeAlert, error) {
	return s.store.GetDowntimeAlertByID(ctx, id)
}

// ListDowntimeAlerts returns all downtime alerts for an organization.
func (s *DowntimeService) ListDowntimeAlerts(ctx context.Context, orgID uuid.UUID) ([]*models.DowntimeAlert, error) {
	return s.store.GetDowntimeAlertsByOrgID(ctx, orgID)
}

// UpdateDowntimeAlert updates an existing downtime alert.
func (s *DowntimeService) UpdateDowntimeAlert(ctx context.Context, alert *models.DowntimeAlert) error {
	alert.UpdatedAt = time.Now()
	if err := s.store.UpdateDowntimeAlert(ctx, alert); err != nil {
		return fmt.Errorf("update downtime alert: %w", err)
	}

	s.logger.Info().
		Str("alert_id", alert.ID.String()).
		Str("name", alert.Name).
		Msg("downtime alert updated")

	return nil
}

// DeleteDowntimeAlert deletes a downtime alert.
func (s *DowntimeService) DeleteDowntimeAlert(ctx context.Context, id uuid.UUID) error {
	if err := s.store.DeleteDowntimeAlert(ctx, id); err != nil {
		return fmt.Errorf("delete downtime alert: %w", err)
	}

	s.logger.Info().
		Str("alert_id", id.String()).
		Msg("downtime alert deleted")

	return nil
}
