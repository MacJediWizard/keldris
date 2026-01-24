// Package backup provides cross-agent restore functionality.
package backup

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ErrSourceAgentNotFound is returned when the source agent cannot be found.
var ErrSourceAgentNotFound = errors.New("source agent not found")

// ErrTargetAgentNotFound is returned when the target agent cannot be found.
var ErrTargetAgentNotFound = errors.New("target agent not found")

// ErrTargetAgentNoRepoAccess is returned when the target agent doesn't have access to the repository.
var ErrTargetAgentNoRepoAccess = errors.New("target agent does not have access to the repository")

// ErrUserNoAccessToSourceAgent is returned when the user doesn't have access to the source agent.
var ErrUserNoAccessToSourceAgent = errors.New("user does not have access to source agent")

// ErrUserNoAccessToTargetAgent is returned when the user doesn't have access to the target agent.
var ErrUserNoAccessToTargetAgent = errors.New("user does not have access to target agent")

// ErrInvalidPathMapping is returned when a path mapping is invalid.
var ErrInvalidPathMapping = errors.New("invalid path mapping")

// CrossRestoreRequest contains the parameters for a cross-agent restore operation.
type CrossRestoreRequest struct {
	SourceAgentID uuid.UUID              `json:"source_agent_id"`
	TargetAgentID uuid.UUID              `json:"target_agent_id"`
	RepositoryID  uuid.UUID              `json:"repository_id"`
	SnapshotID    string                 `json:"snapshot_id"`
	TargetPath    string                 `json:"target_path"`
	IncludePaths  []string               `json:"include_paths,omitempty"`
	ExcludePaths  []string               `json:"exclude_paths,omitempty"`
	PathMappings  []models.PathMapping   `json:"path_mappings,omitempty"`
}

// CrossRestoreJob represents a cross-agent restore job with tracking information.
type CrossRestoreJob struct {
	ID             uuid.UUID                `json:"id"`
	SourceAgentID  uuid.UUID                `json:"source_agent_id"`
	TargetAgentID  uuid.UUID                `json:"target_agent_id"`
	RepositoryID   uuid.UUID                `json:"repository_id"`
	SnapshotID     string                   `json:"snapshot_id"`
	TargetPath     string                   `json:"target_path"`
	IncludePaths   []string                 `json:"include_paths,omitempty"`
	ExcludePaths   []string                 `json:"exclude_paths,omitempty"`
	PathMappings   []models.PathMapping     `json:"path_mappings,omitempty"`
	Status         models.RestoreStatus     `json:"status"`
	Progress       *CrossRestoreProgress    `json:"progress,omitempty"`
	StartedAt      *time.Time               `json:"started_at,omitempty"`
	CompletedAt    *time.Time               `json:"completed_at,omitempty"`
	ErrorMessage   string                   `json:"error_message,omitempty"`
	CreatedAt      time.Time                `json:"created_at"`
	UpdatedAt      time.Time                `json:"updated_at"`
}

// CrossRestoreProgress tracks the progress of a cross-agent restore operation.
type CrossRestoreProgress struct {
	FilesRestored int64 `json:"files_restored"`
	BytesRestored int64 `json:"bytes_restored"`
	TotalFiles    int64 `json:"total_files,omitempty"`
	TotalBytes    int64 `json:"total_bytes,omitempty"`
	CurrentFile   string `json:"current_file,omitempty"`
}

// CrossRestoreStore defines the interface for cross-restore persistence operations.
type CrossRestoreStore interface {
	GetAgentByID(ctx context.Context, id uuid.UUID) (*models.Agent, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	CreateRestore(ctx context.Context, restore *models.Restore) error
	UpdateRestore(ctx context.Context, restore *models.Restore) error
	GetRestoreByID(ctx context.Context, id uuid.UUID) (*models.Restore, error)
	GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error)
}

// CrossRestoreService handles cross-agent restore operations.
type CrossRestoreService struct {
	store  CrossRestoreStore
	logger zerolog.Logger
}

// NewCrossRestoreService creates a new CrossRestoreService.
func NewCrossRestoreService(store CrossRestoreStore, logger zerolog.Logger) *CrossRestoreService {
	return &CrossRestoreService{
		store:  store,
		logger: logger.With().Str("component", "cross_restore").Logger(),
	}
}

// ValidateTargetAgentAccess validates that the target agent has access to the repository.
func (s *CrossRestoreService) ValidateTargetAgentAccess(ctx context.Context, targetAgentID, repositoryID uuid.UUID) error {
	targetAgent, err := s.store.GetAgentByID(ctx, targetAgentID)
	if err != nil {
		return ErrTargetAgentNotFound
	}

	repo, err := s.store.GetRepositoryByID(ctx, repositoryID)
	if err != nil {
		return fmt.Errorf("repository not found: %w", err)
	}

	// Verify target agent and repository are in the same organization
	if targetAgent.OrgID != repo.OrgID {
		return ErrTargetAgentNoRepoAccess
	}

	return nil
}

// ValidateUserAccess validates that the user has access to both source and target agents.
func (s *CrossRestoreService) ValidateUserAccess(ctx context.Context, userID, sourceAgentID, targetAgentID uuid.UUID) error {
	user, err := s.store.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	sourceAgent, err := s.store.GetAgentByID(ctx, sourceAgentID)
	if err != nil {
		return ErrSourceAgentNotFound
	}

	targetAgent, err := s.store.GetAgentByID(ctx, targetAgentID)
	if err != nil {
		return ErrTargetAgentNotFound
	}

	// Verify user has access to both agents (same organization)
	if sourceAgent.OrgID != user.OrgID {
		return ErrUserNoAccessToSourceAgent
	}

	if targetAgent.OrgID != user.OrgID {
		return ErrUserNoAccessToTargetAgent
	}

	return nil
}

// ValidatePathMappings validates that path mappings are well-formed.
func (s *CrossRestoreService) ValidatePathMappings(mappings []models.PathMapping) error {
	for _, mapping := range mappings {
		if mapping.SourcePath == "" || mapping.TargetPath == "" {
			return fmt.Errorf("%w: source and target paths must be non-empty", ErrInvalidPathMapping)
		}

		// Ensure paths are absolute
		if !filepath.IsAbs(mapping.SourcePath) || !filepath.IsAbs(mapping.TargetPath) {
			return fmt.Errorf("%w: paths must be absolute", ErrInvalidPathMapping)
		}

		// Basic sanitization - no path traversal
		if strings.Contains(mapping.SourcePath, "..") || strings.Contains(mapping.TargetPath, "..") {
			return fmt.Errorf("%w: paths cannot contain '..'", ErrInvalidPathMapping)
		}
	}

	return nil
}

// ApplyPathMappings transforms paths according to the provided mappings.
func (s *CrossRestoreService) ApplyPathMappings(originalPath string, mappings []models.PathMapping) string {
	if len(mappings) == 0 {
		return originalPath
	}

	for _, mapping := range mappings {
		// Check if the original path starts with the source path
		if strings.HasPrefix(originalPath, mapping.SourcePath) {
			// Replace the source prefix with the target prefix
			relativePath := strings.TrimPrefix(originalPath, mapping.SourcePath)
			return filepath.Join(mapping.TargetPath, relativePath)
		}
	}

	// No mapping matched, return original path
	return originalPath
}

// CreateCrossRestoreJob creates a new cross-agent restore job.
func (s *CrossRestoreService) CreateCrossRestoreJob(ctx context.Context, req CrossRestoreRequest) (*models.Restore, error) {
	s.logger.Info().
		Str("source_agent_id", req.SourceAgentID.String()).
		Str("target_agent_id", req.TargetAgentID.String()).
		Str("repository_id", req.RepositoryID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("target_path", req.TargetPath).
		Msg("creating cross-agent restore job")

	// Validate path mappings if provided
	if len(req.PathMappings) > 0 {
		if err := s.ValidatePathMappings(req.PathMappings); err != nil {
			return nil, err
		}
	}

	// Validate target agent has access to repository
	if err := s.ValidateTargetAgentAccess(ctx, req.TargetAgentID, req.RepositoryID); err != nil {
		return nil, err
	}

	// Verify snapshot exists
	_, err := s.store.GetBackupBySnapshotID(ctx, req.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	// Create the restore job - use target agent as the executing agent
	// but track the source agent for reference
	restore := models.NewCrossRestore(
		req.SourceAgentID,
		req.TargetAgentID,
		req.RepositoryID,
		req.SnapshotID,
		req.TargetPath,
		req.IncludePaths,
		req.ExcludePaths,
		req.PathMappings,
	)

	if err := s.store.CreateRestore(ctx, restore); err != nil {
		return nil, fmt.Errorf("create restore: %w", err)
	}

	s.logger.Info().
		Str("restore_id", restore.ID.String()).
		Str("source_agent_id", req.SourceAgentID.String()).
		Str("target_agent_id", req.TargetAgentID.String()).
		Msg("cross-agent restore job created")

	return restore, nil
}

// IsCrossAgentRestore returns true if the restore is a cross-agent restore.
func IsCrossAgentRestore(restore *models.Restore) bool {
	return restore.SourceAgentID != nil && *restore.SourceAgentID != restore.AgentID
}

// GetRestoreAgentIDs returns both source and target agent IDs for a restore.
// For non-cross-agent restores, both IDs will be the same.
func GetRestoreAgentIDs(restore *models.Restore) (sourceAgentID, targetAgentID uuid.UUID) {
	targetAgentID = restore.AgentID
	if restore.SourceAgentID != nil {
		sourceAgentID = *restore.SourceAgentID
	} else {
		sourceAgentID = restore.AgentID
	}
	return
}
