// Package backup provides immutability management for backup snapshots.
package backup

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

var (
	// ErrSnapshotLocked is returned when attempting to delete a locked snapshot.
	ErrSnapshotLocked = errors.New("snapshot is locked and cannot be deleted")
	// ErrImmutabilityNotFound is returned when the lock doesn't exist.
	ErrImmutabilityNotFound = errors.New("immutability lock not found")
	// ErrCannotShortenLock is returned when attempting to shorten a lock period.
	ErrCannotShortenLock = errors.New("cannot shorten immutability period; can only extend")
)

// ImmutabilityStore defines the interface for immutability persistence operations.
type ImmutabilityStore interface {
	// CreateSnapshotImmutability creates a new immutability lock.
	CreateSnapshotImmutability(ctx context.Context, lock *models.SnapshotImmutability) error

	// GetSnapshotImmutability returns the immutability lock for a snapshot.
	GetSnapshotImmutability(ctx context.Context, repositoryID uuid.UUID, snapshotID string) (*models.SnapshotImmutability, error)

	// GetSnapshotImmutabilityByID returns an immutability lock by ID.
	GetSnapshotImmutabilityByID(ctx context.Context, id uuid.UUID) (*models.SnapshotImmutability, error)

	// UpdateSnapshotImmutability updates an existing immutability lock.
	UpdateSnapshotImmutability(ctx context.Context, lock *models.SnapshotImmutability) error

	// DeleteExpiredImmutabilityLocks removes expired locks.
	DeleteExpiredImmutabilityLocks(ctx context.Context) (int, error)

	// GetActiveImmutabilityLocksByRepositoryID returns all active locks for a repository.
	GetActiveImmutabilityLocksByRepositoryID(ctx context.Context, repositoryID uuid.UUID) ([]*models.SnapshotImmutability, error)

	// GetActiveImmutabilityLocksByOrgID returns all active locks for an organization.
	GetActiveImmutabilityLocksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SnapshotImmutability, error)

	// IsSnapshotLocked checks if a snapshot has an active lock.
	IsSnapshotLocked(ctx context.Context, repositoryID uuid.UUID, snapshotID string) (bool, error)

	// GetRepository returns a repository by ID.
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)
}

// ImmutabilityManager manages immutability locks for snapshots.
type ImmutabilityManager struct {
	store  ImmutabilityStore
	logger zerolog.Logger
}

// NewImmutabilityManager creates a new ImmutabilityManager.
func NewImmutabilityManager(store ImmutabilityStore, logger zerolog.Logger) *ImmutabilityManager {
	return &ImmutabilityManager{
		store:  store,
		logger: logger.With().Str("component", "immutability").Logger(),
	}
}

// LockSnapshot creates an immutability lock on a snapshot.
func (m *ImmutabilityManager) LockSnapshot(
	ctx context.Context,
	orgID uuid.UUID,
	repositoryID uuid.UUID,
	snapshotID string,
	shortID string,
	days int,
	lockedBy *uuid.UUID,
	reason string,
) (*models.SnapshotImmutability, error) {
	// Check if snapshot is already locked
	existing, err := m.store.GetSnapshotImmutability(ctx, repositoryID, snapshotID)
	if err == nil && existing != nil && existing.IsLocked() {
		return nil, fmt.Errorf("snapshot %s is already locked until %s", shortID, existing.LockedUntil.Format(time.RFC3339))
	}

	lockedUntil := time.Now().AddDate(0, 0, days)

	lock := models.NewSnapshotImmutability(
		orgID,
		repositoryID,
		snapshotID,
		shortID,
		lockedUntil,
		lockedBy,
		reason,
	)

	if err := m.store.CreateSnapshotImmutability(ctx, lock); err != nil {
		return nil, fmt.Errorf("create immutability lock: %w", err)
	}

	m.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("repository_id", repositoryID.String()).
		Int("days", days).
		Time("locked_until", lockedUntil).
		Msg("snapshot immutability lock created")

	return lock, nil
}

// ExtendLock extends an existing immutability lock.
// The new lock period must be longer than the current one.
func (m *ImmutabilityManager) ExtendLock(
	ctx context.Context,
	repositoryID uuid.UUID,
	snapshotID string,
	additionalDays int,
	reason string,
) (*models.SnapshotImmutability, error) {
	lock, err := m.store.GetSnapshotImmutability(ctx, repositoryID, snapshotID)
	if err != nil {
		return nil, ErrImmutabilityNotFound
	}

	if !lock.IsLocked() {
		return nil, fmt.Errorf("lock has already expired")
	}

	newLockedUntil := lock.LockedUntil.AddDate(0, 0, additionalDays)
	lock.LockedUntil = newLockedUntil
	lock.UpdatedAt = time.Now()
	if reason != "" {
		lock.Reason = reason
	}

	if err := m.store.UpdateSnapshotImmutability(ctx, lock); err != nil {
		return nil, fmt.Errorf("update immutability lock: %w", err)
	}

	m.logger.Info().
		Str("snapshot_id", snapshotID).
		Str("repository_id", repositoryID.String()).
		Int("additional_days", additionalDays).
		Time("new_locked_until", newLockedUntil).
		Msg("snapshot immutability lock extended")

	return lock, nil
}

// CheckDeleteAllowed checks if a snapshot can be deleted.
// Returns nil if deletion is allowed, or an error with details if not.
func (m *ImmutabilityManager) CheckDeleteAllowed(
	ctx context.Context,
	repositoryID uuid.UUID,
	snapshotID string,
) error {
	locked, err := m.store.IsSnapshotLocked(ctx, repositoryID, snapshotID)
	if err != nil {
		return fmt.Errorf("check snapshot lock: %w", err)
	}

	if locked {
		lock, err := m.store.GetSnapshotImmutability(ctx, repositoryID, snapshotID)
		if err != nil {
			return ErrSnapshotLocked
		}
		return fmt.Errorf("%w: locked until %s (%d days remaining)",
			ErrSnapshotLocked,
			lock.LockedUntil.Format("2006-01-02"),
			lock.RemainingDays())
	}

	return nil
}

// GetLockStatus returns the immutability status for a snapshot.
func (m *ImmutabilityManager) GetLockStatus(
	ctx context.Context,
	repositoryID uuid.UUID,
	snapshotID string,
) (*models.SnapshotImmutability, error) {
	return m.store.GetSnapshotImmutability(ctx, repositoryID, snapshotID)
}

// GetActiveLocksForRepository returns all active locks for a repository.
func (m *ImmutabilityManager) GetActiveLocksForRepository(
	ctx context.Context,
	repositoryID uuid.UUID,
) ([]*models.SnapshotImmutability, error) {
	return m.store.GetActiveImmutabilityLocksByRepositoryID(ctx, repositoryID)
}

// GetActiveLocksForOrg returns all active locks for an organization.
func (m *ImmutabilityManager) GetActiveLocksForOrg(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*models.SnapshotImmutability, error) {
	return m.store.GetActiveImmutabilityLocksByOrgID(ctx, orgID)
}

// CleanupExpiredLocks removes expired immutability locks.
func (m *ImmutabilityManager) CleanupExpiredLocks(ctx context.Context) (int, error) {
	count, err := m.store.DeleteExpiredImmutabilityLocks(ctx)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired locks: %w", err)
	}

	if count > 0 {
		m.logger.Info().
			Int("count", count).
			Msg("cleaned up expired immutability locks")
	}

	return count, nil
}

// ImmutabilityStatus represents the lock status of a snapshot for API responses.
type ImmutabilityStatus struct {
	IsLocked      bool       `json:"is_locked"`
	LockedUntil   *time.Time `json:"locked_until,omitempty"`
	RemainingDays int        `json:"remaining_days,omitempty"`
	Reason        string     `json:"reason,omitempty"`
	LockedAt      *time.Time `json:"locked_at,omitempty"`
}

// GetStatus returns a summary status for a snapshot.
func (m *ImmutabilityManager) GetStatus(
	ctx context.Context,
	repositoryID uuid.UUID,
	snapshotID string,
) (*ImmutabilityStatus, error) {
	lock, err := m.store.GetSnapshotImmutability(ctx, repositoryID, snapshotID)
	if err != nil {
		// No lock found - snapshot is not locked
		return &ImmutabilityStatus{
			IsLocked: false,
		}, nil
	}

	if !lock.IsLocked() {
		return &ImmutabilityStatus{
			IsLocked: false,
		}, nil
	}

	return &ImmutabilityStatus{
		IsLocked:      true,
		LockedUntil:   &lock.LockedUntil,
		RemainingDays: lock.RemainingDays(),
		Reason:        lock.Reason,
		LockedAt:      &lock.LockedAt,
	}, nil
}
