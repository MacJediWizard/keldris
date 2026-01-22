package maintenance

import (
	"context"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/MacJediWizard/keldris/internal/notifications"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// MaintenanceStore defines the interface for maintenance data access.
type MaintenanceStore interface {
	ListActiveMaintenanceWindows(ctx context.Context, orgID uuid.UUID, now time.Time) ([]*models.MaintenanceWindow, error)
	ListUpcomingMaintenanceWindows(ctx context.Context, orgID uuid.UUID, now time.Time, withinMinutes int) ([]*models.MaintenanceWindow, error)
	ListPendingMaintenanceNotifications(ctx context.Context) ([]*models.MaintenanceWindow, error)
	MarkMaintenanceNotificationSent(ctx context.Context, id uuid.UUID) error
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
}

// Service manages maintenance windows and their integration with backups.
type Service struct {
	store         MaintenanceStore
	notifier      *notifications.Service
	logger        zerolog.Logger
	mu            sync.RWMutex
	activeWindows map[uuid.UUID][]*models.MaintenanceWindow // org_id -> active windows
	lastRefresh   time.Time
}

// NewService creates a new maintenance service.
func NewService(store MaintenanceStore, notifier *notifications.Service, logger zerolog.Logger) *Service {
	return &Service{
		store:         store,
		notifier:      notifier,
		logger:        logger.With().Str("component", "maintenance").Logger(),
		activeWindows: make(map[uuid.UUID][]*models.MaintenanceWindow),
	}
}

// RefreshCache reloads active maintenance windows from the database for all organizations.
func (s *Service) RefreshCache(ctx context.Context) error {
	orgs, err := s.store.GetAllOrganizations(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	newCache := make(map[uuid.UUID][]*models.MaintenanceWindow)

	for _, org := range orgs {
		windows, err := s.store.ListActiveMaintenanceWindows(ctx, org.ID, now)
		if err != nil {
			s.logger.Error().Err(err).
				Str("org_id", org.ID.String()).
				Msg("failed to load active maintenance windows")
			continue
		}
		if len(windows) > 0 {
			newCache[org.ID] = windows
		}
	}

	s.mu.Lock()
	s.activeWindows = newCache
	s.lastRefresh = now
	s.mu.Unlock()

	s.logger.Debug().
		Int("org_count", len(newCache)).
		Msg("refreshed maintenance window cache")

	return nil
}

// IsMaintenanceActive checks if any maintenance window is currently active for an org.
func (s *Service) IsMaintenanceActive(orgID uuid.UUID) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	windows := s.activeWindows[orgID]
	for _, w := range windows {
		if w.IsActive(now) {
			return true
		}
	}
	return false
}

// GetActiveWindow returns the current active maintenance window for an org.
// If multiple windows are active, returns the one ending soonest.
func (s *Service) GetActiveWindow(orgID uuid.UUID) *models.MaintenanceWindow {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	windows := s.activeWindows[orgID]
	for _, w := range windows {
		if w.IsActive(now) {
			return w
		}
	}
	return nil
}

// GetActiveWindowFromDB returns the current active maintenance window from the database.
// This is used for API requests where fresh data is needed.
func (s *Service) GetActiveWindowFromDB(ctx context.Context, orgID uuid.UUID) (*models.MaintenanceWindow, error) {
	now := time.Now()
	windows, err := s.store.ListActiveMaintenanceWindows(ctx, orgID, now)
	if err != nil {
		return nil, err
	}
	if len(windows) > 0 {
		return windows[0], nil
	}
	return nil, nil
}

// GetUpcomingWindowFromDB returns the soonest upcoming maintenance window from the database.
func (s *Service) GetUpcomingWindowFromDB(ctx context.Context, orgID uuid.UUID, withinMinutes int) (*models.MaintenanceWindow, error) {
	now := time.Now()
	windows, err := s.store.ListUpcomingMaintenanceWindows(ctx, orgID, now, withinMinutes)
	if err != nil {
		return nil, err
	}
	if len(windows) > 0 {
		return windows[0], nil
	}
	return nil, nil
}

// CheckAndSendNotifications checks for upcoming maintenance and sends notifications.
// This should be called periodically (e.g., every refresh cycle).
func (s *Service) CheckAndSendNotifications(ctx context.Context) {
	windows, err := s.store.ListPendingMaintenanceNotifications(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to list pending maintenance notifications")
		return
	}

	for _, window := range windows {
		if s.notifier != nil {
			s.notifier.NotifyMaintenanceScheduled(ctx, window)
		}

		// Mark notification as sent
		if err := s.store.MarkMaintenanceNotificationSent(ctx, window.ID); err != nil {
			s.logger.Error().Err(err).
				Str("window_id", window.ID.String()).
				Msg("failed to mark maintenance notification as sent")
		} else {
			s.logger.Info().
				Str("window_id", window.ID.String()).
				Str("org_id", window.OrgID.String()).
				Str("title", window.Title).
				Time("starts_at", window.StartsAt).
				Msg("sent maintenance notification")
		}
	}
}

// LastRefresh returns the time of the last cache refresh.
func (s *Service) LastRefresh() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastRefresh
}
