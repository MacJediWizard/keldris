// Package backup provides automated test restore notification functionality.
package backup

import (
	"context"

	"github.com/MacJediWizard/keldris/internal/models"
)

// TestRestoreNotificationService defines the interface for sending test restore notifications.
type TestRestoreNotificationService interface {
	NotifyTestRestoreFailed(ctx context.Context, result *models.TestRestoreResult, repo *models.Repository, consecutiveFails int)
}

// TestRestoreNotifierAdapter adapts the notification service to the TestRestoreNotifier interface.
type TestRestoreNotifierAdapter struct {
	notificationService TestRestoreNotificationService
}

// NewTestRestoreNotifierAdapter creates a new TestRestoreNotifierAdapter.
func NewTestRestoreNotifierAdapter(svc TestRestoreNotificationService) *TestRestoreNotifierAdapter {
	return &TestRestoreNotifierAdapter{
		notificationService: svc,
	}
}

// NotifyTestRestoreFailed sends an alert about a failed test restore.
func (n *TestRestoreNotifierAdapter) NotifyTestRestoreFailed(ctx context.Context, result *models.TestRestoreResult, repo *models.Repository, consecutiveFails int) error {
	n.notificationService.NotifyTestRestoreFailed(ctx, result, repo, consecutiveFails)
	return nil
}
