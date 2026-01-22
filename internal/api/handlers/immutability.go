package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/backup"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// ImmutabilityStore defines the interface for immutability persistence operations.
type ImmutabilityStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetRepository(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	CreateSnapshotImmutability(ctx context.Context, lock *models.SnapshotImmutability) error
	GetSnapshotImmutability(ctx context.Context, repositoryID uuid.UUID, snapshotID string) (*models.SnapshotImmutability, error)
	GetSnapshotImmutabilityByID(ctx context.Context, id uuid.UUID) (*models.SnapshotImmutability, error)
	UpdateSnapshotImmutability(ctx context.Context, lock *models.SnapshotImmutability) error
	DeleteExpiredImmutabilityLocks(ctx context.Context) (int, error)
	GetActiveImmutabilityLocksByRepositoryID(ctx context.Context, repositoryID uuid.UUID) ([]*models.SnapshotImmutability, error)
	GetActiveImmutabilityLocksByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.SnapshotImmutability, error)
	IsSnapshotLocked(ctx context.Context, repositoryID uuid.UUID, snapshotID string) (bool, error)
	GetRepositoryImmutabilitySettings(ctx context.Context, repositoryID uuid.UUID) (*models.RepositoryImmutabilitySettings, error)
	UpdateRepositoryImmutabilitySettings(ctx context.Context, repositoryID uuid.UUID, settings *models.RepositoryImmutabilitySettings) error
	GetBackupBySnapshotID(ctx context.Context, snapshotID string) (*models.Backup, error)
}

// ImmutabilityHandler handles immutability-related HTTP endpoints.
type ImmutabilityHandler struct {
	store   ImmutabilityStore
	manager *backup.ImmutabilityManager
	logger  zerolog.Logger
}

// NewImmutabilityHandler creates a new ImmutabilityHandler.
func NewImmutabilityHandler(store ImmutabilityStore, logger zerolog.Logger) *ImmutabilityHandler {
	manager := backup.NewImmutabilityManager(store, logger)
	return &ImmutabilityHandler{
		store:   store,
		manager: manager,
		logger:  logger.With().Str("component", "immutability_handler").Logger(),
	}
}

// RegisterRoutes registers immutability routes on the given router group.
func (h *ImmutabilityHandler) RegisterRoutes(r *gin.RouterGroup) {
	immutability := r.Group("/immutability")
	{
		immutability.GET("", h.ListLocks)
		immutability.POST("", h.CreateLock)
		immutability.GET("/:id", h.GetLock)
		immutability.PUT("/:id", h.ExtendLock)
	}

	// Add routes under snapshots for checking immutability status
	snapshots := r.Group("/snapshots")
	{
		snapshots.GET("/:id/immutability", h.GetSnapshotImmutability)
	}

	// Add routes under repositories for immutability settings
	repos := r.Group("/repositories")
	{
		repos.GET("/:id/immutability", h.GetRepositoryImmutabilitySettings)
		repos.PUT("/:id/immutability", h.UpdateRepositoryImmutabilitySettings)
		repos.GET("/:id/immutability/locks", h.ListRepositoryLocks)
	}
}

// ImmutabilityLockResponse represents an immutability lock in API responses.
type ImmutabilityLockResponse struct {
	ID                  string  `json:"id"`
	RepositoryID        string  `json:"repository_id"`
	SnapshotID          string  `json:"snapshot_id"`
	ShortID             string  `json:"short_id"`
	LockedAt            string  `json:"locked_at"`
	LockedUntil         string  `json:"locked_until"`
	LockedBy            *string `json:"locked_by,omitempty"`
	Reason              string  `json:"reason,omitempty"`
	RemainingDays       int     `json:"remaining_days"`
	S3ObjectLockEnabled bool    `json:"s3_object_lock_enabled"`
	CreatedAt           string  `json:"created_at"`
}

func toImmutabilityLockResponse(lock *models.SnapshotImmutability) ImmutabilityLockResponse {
	resp := ImmutabilityLockResponse{
		ID:                  lock.ID.String(),
		RepositoryID:        lock.RepositoryID.String(),
		SnapshotID:          lock.SnapshotID,
		ShortID:             lock.ShortID,
		LockedAt:            lock.LockedAt.Format(time.RFC3339),
		LockedUntil:         lock.LockedUntil.Format(time.RFC3339),
		Reason:              lock.Reason,
		RemainingDays:       lock.RemainingDays(),
		S3ObjectLockEnabled: lock.S3ObjectLockEnabled,
		CreatedAt:           lock.CreatedAt.Format(time.RFC3339),
	}
	if lock.LockedBy != nil {
		s := lock.LockedBy.String()
		resp.LockedBy = &s
	}
	return resp
}

// CreateLockRequest is the request body for creating an immutability lock.
type CreateLockRequest struct {
	RepositoryID string `json:"repository_id" binding:"required"`
	SnapshotID   string `json:"snapshot_id" binding:"required"`
	Days         int    `json:"days" binding:"required,min=1,max=36500"`
	Reason       string `json:"reason" binding:"max=500"`
	EnableS3Lock bool   `json:"enable_s3_lock"`
}

// ExtendLockRequest is the request body for extending an immutability lock.
type ExtendLockRequest struct {
	AdditionalDays int    `json:"additional_days" binding:"required,min=1,max=36500"`
	Reason         string `json:"reason" binding:"max=500"`
}

// ImmutabilitySettingsRequest is the request body for updating repository immutability settings.
type ImmutabilitySettingsRequest struct {
	Enabled     bool `json:"enabled"`
	DefaultDays *int `json:"default_days,omitempty"`
}

// CreateLock creates a new immutability lock on a snapshot.
//
//	@Summary		Create immutability lock
//	@Description	Creates an immutability lock on a snapshot to prevent deletion for a specified period
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Param			request	body		CreateLockRequest	true	"Lock details"
//	@Success		201		{object}	ImmutabilityLockResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		409		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/immutability [post]
func (h *ImmutabilityHandler) CreateLock(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req CreateLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	repoID, err := uuid.Parse(req.RepositoryID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
		return
	}

	// Verify user has access to the repository
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepository(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Only admins can create immutability locks
	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can create immutability locks"})
		return
	}

	// Compute short ID
	shortID := req.SnapshotID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	lock, err := h.manager.LockSnapshot(
		c.Request.Context(),
		dbUser.OrgID,
		repoID,
		req.SnapshotID,
		shortID,
		req.Days,
		&dbUser.ID,
		req.Reason,
	)
	if err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", req.SnapshotID).Msg("failed to create lock")
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().
		Str("lock_id", lock.ID.String()).
		Str("snapshot_id", req.SnapshotID).
		Str("repository_id", req.RepositoryID).
		Int("days", req.Days).
		Str("user_id", dbUser.ID.String()).
		Msg("immutability lock created")

	c.JSON(http.StatusCreated, toImmutabilityLockResponse(lock))
}

// GetLock returns a specific immutability lock by ID.
//
//	@Summary		Get immutability lock
//	@Description	Returns a specific immutability lock by ID
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Lock ID"
//	@Success		200	{object}	ImmutabilityLockResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/immutability/{id} [get]
func (h *ImmutabilityHandler) GetLock(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lock ID"})
		return
	}

	lock, err := h.store.GetSnapshotImmutabilityByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lock not found"})
		return
	}

	// Verify user access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if lock.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "lock not found"})
		return
	}

	c.JSON(http.StatusOK, toImmutabilityLockResponse(lock))
}

// ExtendLock extends an existing immutability lock.
//
//	@Summary		Extend immutability lock
//	@Description	Extends an existing immutability lock by adding additional days
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string				true	"Lock ID"
//	@Param			request	body		ExtendLockRequest	true	"Extension details"
//	@Success		200		{object}	ImmutabilityLockResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/immutability/{id} [put]
func (h *ImmutabilityHandler) ExtendLock(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lock ID"})
		return
	}

	var req ExtendLockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	lock, err := h.store.GetSnapshotImmutabilityByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lock not found"})
		return
	}

	// Verify user access and admin status
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	if lock.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "lock not found"})
		return
	}

	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can extend immutability locks"})
		return
	}

	extendedLock, err := h.manager.ExtendLock(
		c.Request.Context(),
		lock.RepositoryID,
		lock.SnapshotID,
		req.AdditionalDays,
		req.Reason,
	)
	if err != nil {
		h.logger.Error().Err(err).Str("lock_id", id.String()).Msg("failed to extend lock")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info().
		Str("lock_id", id.String()).
		Int("additional_days", req.AdditionalDays).
		Str("user_id", dbUser.ID.String()).
		Msg("immutability lock extended")

	c.JSON(http.StatusOK, toImmutabilityLockResponse(extendedLock))
}

// ListLocks returns all active immutability locks for the organization.
//
//	@Summary		List immutability locks
//	@Description	Returns all active immutability locks for the current organization
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string][]ImmutabilityLockResponse
//	@Failure		401	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/immutability [get]
func (h *ImmutabilityHandler) ListLocks(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	locks, err := h.store.GetActiveImmutabilityLocksByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to list locks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list locks"})
		return
	}

	responses := make([]ImmutabilityLockResponse, len(locks))
	for i, lock := range locks {
		responses[i] = toImmutabilityLockResponse(lock)
	}

	c.JSON(http.StatusOK, gin.H{"locks": responses})
}

// ListRepositoryLocks returns all active immutability locks for a repository.
//
//	@Summary		List repository immutability locks
//	@Description	Returns all active immutability locks for a specific repository
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Repository ID"
//	@Success		200	{object}	map[string][]ImmutabilityLockResponse
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id}/immutability/locks [get]
func (h *ImmutabilityHandler) ListRepositoryLocks(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepository(c.Request.Context(), repoID)
	if err != nil || repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	locks, err := h.store.GetActiveImmutabilityLocksByRepositoryID(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repository_id", repoID.String()).Msg("failed to list locks")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list locks"})
		return
	}

	responses := make([]ImmutabilityLockResponse, len(locks))
	for i, lock := range locks {
		responses[i] = toImmutabilityLockResponse(lock)
	}

	c.JSON(http.StatusOK, gin.H{"locks": responses})
}

// GetSnapshotImmutability returns the immutability status for a snapshot.
//
//	@Summary		Get snapshot immutability status
//	@Description	Returns the immutability status for a specific snapshot
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string	true	"Snapshot ID"
//	@Param			repository_id	query		string	true	"Repository ID"
//	@Success		200				{object}	backup.ImmutabilityStatus
//	@Failure		400				{object}	map[string]string
//	@Failure		401				{object}	map[string]string
//	@Failure		404				{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/snapshots/{id}/immutability [get]
func (h *ImmutabilityHandler) GetSnapshotImmutability(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	snapshotID := c.Param("id")
	if snapshotID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "snapshot ID required"})
		return
	}

	repoIDParam := c.Query("repository_id")
	if repoIDParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repository_id query parameter required"})
		return
	}

	repoID, err := uuid.Parse(repoIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository_id"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepository(c.Request.Context(), repoID)
	if err != nil || repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	status, err := h.manager.GetStatus(c.Request.Context(), repoID, snapshotID)
	if err != nil {
		h.logger.Error().Err(err).Str("snapshot_id", snapshotID).Msg("failed to get immutability status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get status"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetRepositoryImmutabilitySettings returns the immutability settings for a repository.
//
//	@Summary		Get repository immutability settings
//	@Description	Returns the immutability settings for a specific repository
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Repository ID"
//	@Success		200	{object}	models.RepositoryImmutabilitySettings
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id}/immutability [get]
func (h *ImmutabilityHandler) GetRepositoryImmutabilitySettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	// Verify access
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepository(c.Request.Context(), repoID)
	if err != nil || repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	settings, err := h.store.GetRepositoryImmutabilitySettings(c.Request.Context(), repoID)
	if err != nil {
		h.logger.Error().Err(err).Str("repository_id", repoID.String()).Msg("failed to get settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateRepositoryImmutabilitySettings updates the immutability settings for a repository.
//
//	@Summary		Update repository immutability settings
//	@Description	Updates the immutability settings for a specific repository
//	@Tags			Immutability
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Repository ID"
//	@Param			request	body		ImmutabilitySettingsRequest		true	"Settings"
//	@Success		200		{object}	models.RepositoryImmutabilitySettings
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		403		{object}	map[string]string
//	@Failure		404		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		SessionAuth
//	@Router			/repositories/{id}/immutability [put]
func (h *ImmutabilityHandler) UpdateRepositoryImmutabilitySettings(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	repoID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repository ID"})
		return
	}

	var req ImmutabilitySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Verify access and admin status
	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepository(c.Request.Context(), repoID)
	if err != nil || repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if !dbUser.IsAdmin() {
		c.JSON(http.StatusForbidden, gin.H{"error": "only administrators can update immutability settings"})
		return
	}

	settings := &models.RepositoryImmutabilitySettings{
		Enabled:     req.Enabled,
		DefaultDays: req.DefaultDays,
	}

	if err := h.store.UpdateRepositoryImmutabilitySettings(c.Request.Context(), repoID, settings); err != nil {
		h.logger.Error().Err(err).Str("repository_id", repoID.String()).Msg("failed to update settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	h.logger.Info().
		Str("repository_id", repoID.String()).
		Bool("enabled", req.Enabled).
		Str("user_id", dbUser.ID.String()).
		Msg("immutability settings updated")

	c.JSON(http.StatusOK, settings)
}

// CheckDeleteAllowed is a helper to be used by other handlers before deleting a snapshot.
func (h *ImmutabilityHandler) CheckDeleteAllowed(ctx context.Context, repositoryID uuid.UUID, snapshotID string) error {
	return h.manager.CheckDeleteAllowed(ctx, repositoryID, snapshotID)
}

// IsSnapshotLocked returns true if the snapshot is locked.
func (h *ImmutabilityHandler) IsSnapshotLocked(ctx context.Context, repositoryID uuid.UUID, snapshotID string) bool {
	err := h.manager.CheckDeleteAllowed(ctx, repositoryID, snapshotID)
	return errors.Is(err, backup.ErrSnapshotLocked)
}
