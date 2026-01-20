package monitoring

import (
	"context"
	"fmt"

	"github.com/MacJediWizard/keldris/internal/db"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

// AlertStore defines the database operations needed by the alert service.
type AlertStore interface {
	CreateAlert(ctx context.Context, alert *models.Alert) error
	UpdateAlert(ctx context.Context, alert *models.Alert) error
	GetAlertByID(ctx context.Context, id uuid.UUID) (*models.Alert, error)
	GetAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error)
	GetActiveAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error)
	GetActiveAlertCountByOrgID(ctx context.Context, orgID uuid.UUID) (int, error)
	GetAlertByResourceAndType(ctx context.Context, orgID uuid.UUID, resourceType models.ResourceType, resourceID uuid.UUID, alertType models.AlertType) (*models.Alert, error)
	ResolveAlertsByResource(ctx context.Context, resourceType models.ResourceType, resourceID uuid.UUID) error
	GetAlertRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AlertRule, error)
	GetEnabledAlertRulesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.AlertRule, error)
	GetAlertRuleByID(ctx context.Context, id uuid.UUID) (*models.AlertRule, error)
	CreateAlertRule(ctx context.Context, rule *models.AlertRule) error
	UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error
	DeleteAlertRule(ctx context.Context, id uuid.UUID) error
}

// NotificationSender defines the interface for sending alert notifications.
type NotificationSender interface {
	SendAlertNotification(ctx context.Context, alert *models.Alert) error
}

// AlertServiceImpl implements AlertService for creating and managing alerts.
type AlertServiceImpl struct {
	store        AlertStore
	notification NotificationSender
	logger       zerolog.Logger
}

// NewAlertService creates a new AlertServiceImpl instance.
func NewAlertService(store AlertStore, notification NotificationSender, logger zerolog.Logger) *AlertServiceImpl {
	return &AlertServiceImpl{
		store:        store,
		notification: notification,
		logger:       logger.With().Str("component", "alert_service").Logger(),
	}
}

// NewAlertServiceWithDB creates an AlertServiceImpl using the database directly.
func NewAlertServiceWithDB(database *db.DB, notification NotificationSender, logger zerolog.Logger) *AlertServiceImpl {
	return NewAlertService(database, notification, logger)
}

// CreateAlert creates a new alert and optionally sends a notification.
func (s *AlertServiceImpl) CreateAlert(ctx context.Context, alert *models.Alert) error {
	if err := s.store.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("store alert: %w", err)
	}

	s.logger.Info().
		Str("alert_id", alert.ID.String()).
		Str("type", string(alert.Type)).
		Str("severity", string(alert.Severity)).
		Str("title", alert.Title).
		Msg("alert created")

	// Send notification if configured
	if s.notification != nil {
		if err := s.notification.SendAlertNotification(ctx, alert); err != nil {
			// Log but don't fail the alert creation
			s.logger.Error().Err(err).Str("alert_id", alert.ID.String()).Msg("failed to send alert notification")
		}
	}

	return nil
}

// GetAlert retrieves an alert by ID.
func (s *AlertServiceImpl) GetAlert(ctx context.Context, id uuid.UUID) (*models.Alert, error) {
	return s.store.GetAlertByID(ctx, id)
}

// ListAlerts returns all alerts for an organization.
func (s *AlertServiceImpl) ListAlerts(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	return s.store.GetAlertsByOrgID(ctx, orgID)
}

// ListActiveAlerts returns active (non-resolved) alerts for an organization.
func (s *AlertServiceImpl) ListActiveAlerts(ctx context.Context, orgID uuid.UUID) ([]*models.Alert, error) {
	return s.store.GetActiveAlertsByOrgID(ctx, orgID)
}

// GetActiveAlertCount returns the count of active alerts for an organization.
func (s *AlertServiceImpl) GetActiveAlertCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	return s.store.GetActiveAlertCountByOrgID(ctx, orgID)
}

// AcknowledgeAlert marks an alert as acknowledged by a user.
func (s *AlertServiceImpl) AcknowledgeAlert(ctx context.Context, alertID uuid.UUID, userID uuid.UUID) error {
	alert, err := s.store.GetAlertByID(ctx, alertID)
	if err != nil {
		return fmt.Errorf("get alert: %w", err)
	}

	if alert.Status == models.AlertStatusResolved {
		return fmt.Errorf("cannot acknowledge a resolved alert")
	}

	alert.Acknowledge(userID)

	if err := s.store.UpdateAlert(ctx, alert); err != nil {
		return fmt.Errorf("update alert: %w", err)
	}

	s.logger.Info().
		Str("alert_id", alert.ID.String()).
		Str("acknowledged_by", userID.String()).
		Msg("alert acknowledged")

	return nil
}

// ResolveAlert marks an alert as resolved.
func (s *AlertServiceImpl) ResolveAlert(ctx context.Context, alertID uuid.UUID) error {
	alert, err := s.store.GetAlertByID(ctx, alertID)
	if err != nil {
		return fmt.Errorf("get alert: %w", err)
	}

	if alert.Status == models.AlertStatusResolved {
		return nil // Already resolved
	}

	alert.Resolve()

	if err := s.store.UpdateAlert(ctx, alert); err != nil {
		return fmt.Errorf("update alert: %w", err)
	}

	s.logger.Info().
		Str("alert_id", alert.ID.String()).
		Msg("alert resolved")

	return nil
}

// ResolveAlertsByResource resolves all active alerts for a specific resource.
func (s *AlertServiceImpl) ResolveAlertsByResource(ctx context.Context, resourceType models.ResourceType, resourceID uuid.UUID) error {
	if err := s.store.ResolveAlertsByResource(ctx, resourceType, resourceID); err != nil {
		return fmt.Errorf("resolve alerts: %w", err)
	}

	s.logger.Debug().
		Str("resource_type", string(resourceType)).
		Str("resource_id", resourceID.String()).
		Msg("alerts resolved for resource")

	return nil
}

// HasActiveAlert checks if there's an active alert for a specific resource and type.
func (s *AlertServiceImpl) HasActiveAlert(ctx context.Context, orgID uuid.UUID, resourceType models.ResourceType, resourceID uuid.UUID, alertType models.AlertType) (bool, error) {
	_, err := s.store.GetAlertByResourceAndType(ctx, orgID, resourceType, resourceID, alertType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check active alert: %w", err)
	}
	return true, nil
}

// Alert Rule management

// CreateAlertRule creates a new alert rule.
func (s *AlertServiceImpl) CreateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	if err := s.store.CreateAlertRule(ctx, rule); err != nil {
		return fmt.Errorf("create alert rule: %w", err)
	}

	s.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("name", rule.Name).
		Str("type", string(rule.Type)).
		Msg("alert rule created")

	return nil
}

// GetAlertRule retrieves an alert rule by ID.
func (s *AlertServiceImpl) GetAlertRule(ctx context.Context, id uuid.UUID) (*models.AlertRule, error) {
	return s.store.GetAlertRuleByID(ctx, id)
}

// ListAlertRules returns all alert rules for an organization.
func (s *AlertServiceImpl) ListAlertRules(ctx context.Context, orgID uuid.UUID) ([]*models.AlertRule, error) {
	return s.store.GetAlertRulesByOrgID(ctx, orgID)
}

// ListEnabledAlertRules returns enabled alert rules for an organization.
func (s *AlertServiceImpl) ListEnabledAlertRules(ctx context.Context, orgID uuid.UUID) ([]*models.AlertRule, error) {
	return s.store.GetEnabledAlertRulesByOrgID(ctx, orgID)
}

// UpdateAlertRule updates an existing alert rule.
func (s *AlertServiceImpl) UpdateAlertRule(ctx context.Context, rule *models.AlertRule) error {
	if err := s.store.UpdateAlertRule(ctx, rule); err != nil {
		return fmt.Errorf("update alert rule: %w", err)
	}

	s.logger.Info().
		Str("rule_id", rule.ID.String()).
		Str("name", rule.Name).
		Msg("alert rule updated")

	return nil
}

// DeleteAlertRule deletes an alert rule.
func (s *AlertServiceImpl) DeleteAlertRule(ctx context.Context, id uuid.UUID) error {
	if err := s.store.DeleteAlertRule(ctx, id); err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}

	s.logger.Info().
		Str("rule_id", id.String()).
		Msg("alert rule deleted")

	return nil
}

// NoOpNotificationSender is a notification sender that does nothing.
// Used when notifications are not configured.
type NoOpNotificationSender struct{}

// SendAlertNotification does nothing and returns nil.
func (n *NoOpNotificationSender) SendAlertNotification(_ context.Context, _ *models.Alert) error {
	return nil
}
