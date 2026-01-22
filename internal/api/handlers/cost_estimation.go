package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/MacJediWizard/keldris/internal/api/middleware"
	"github.com/MacJediWizard/keldris/internal/cost"
	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// CostEstimationStore defines the interface for cost estimation persistence operations.
type CostEstimationStore interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetRepositoriesByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.Repository, error)
	GetRepositoryByID(ctx context.Context, id uuid.UUID) (*models.Repository, error)
	GetLatestStorageStats(ctx context.Context, repositoryID uuid.UUID) (*models.StorageStats, error)
	GetLatestStatsForAllRepos(ctx context.Context, orgID uuid.UUID) ([]*models.StorageStats, error)
	GetStorageGrowth(ctx context.Context, repositoryID uuid.UUID, days int) ([]*models.StorageGrowthPoint, error)
	GetAllStorageGrowth(ctx context.Context, orgID uuid.UUID, days int) ([]*models.StorageGrowthPoint, error)
	GetStoragePricingByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.StoragePricing, error)
	GetStoragePricingByType(ctx context.Context, orgID uuid.UUID, repoType string) (*models.StoragePricing, error)
	CreateStoragePricing(ctx context.Context, p *models.StoragePricing) error
	UpdateStoragePricing(ctx context.Context, p *models.StoragePricing) error
	DeleteStoragePricing(ctx context.Context, id uuid.UUID) error
	CreateCostEstimate(ctx context.Context, e *models.CostEstimateRecord) error
	GetLatestCostEstimates(ctx context.Context, orgID uuid.UUID) ([]*models.CostEstimateRecord, error)
	GetCostEstimateHistory(ctx context.Context, repoID uuid.UUID, days int) ([]*models.CostEstimateRecord, error)
	GetCostAlertsByOrgID(ctx context.Context, orgID uuid.UUID) ([]*models.CostAlert, error)
	GetCostAlertByID(ctx context.Context, id uuid.UUID) (*models.CostAlert, error)
	CreateCostAlert(ctx context.Context, a *models.CostAlert) error
	UpdateCostAlert(ctx context.Context, a *models.CostAlert) error
	DeleteCostAlert(ctx context.Context, id uuid.UUID) error
}

// CostEstimationHandler handles cost estimation related HTTP endpoints.
type CostEstimationHandler struct {
	store      CostEstimationStore
	calculator *cost.Calculator
	logger     zerolog.Logger
}

// NewCostEstimationHandler creates a new CostEstimationHandler.
func NewCostEstimationHandler(store CostEstimationStore, logger zerolog.Logger) *CostEstimationHandler {
	return &CostEstimationHandler{
		store:      store,
		calculator: cost.NewCalculator(),
		logger:     logger.With().Str("component", "cost_estimation_handler").Logger(),
	}
}

// RegisterRoutes registers cost estimation routes on the given router group.
func (h *CostEstimationHandler) RegisterRoutes(r *gin.RouterGroup) {
	costs := r.Group("/costs")
	{
		costs.GET("/summary", h.GetCostSummary)
		costs.GET("/repositories", h.ListRepositoryCosts)
		costs.GET("/repositories/:id", h.GetRepositoryCost)
		costs.GET("/forecast", h.GetCostForecast)
		costs.GET("/history", h.GetCostHistory)
	}

	pricing := r.Group("/pricing")
	{
		pricing.GET("", h.ListPricing)
		pricing.GET("/defaults", h.GetDefaultPricing)
		pricing.POST("", h.CreatePricing)
		pricing.PUT("/:id", h.UpdatePricing)
		pricing.DELETE("/:id", h.DeletePricing)
	}

	alerts := r.Group("/cost-alerts")
	{
		alerts.GET("", h.ListCostAlerts)
		alerts.GET("/:id", h.GetCostAlert)
		alerts.POST("", h.CreateCostAlert)
		alerts.PUT("/:id", h.UpdateCostAlert)
		alerts.DELETE("/:id", h.DeleteCostAlert)
	}
}

// GetCostSummary returns aggregated cost estimation for the organization.
// GET /api/v1/costs/summary
func (h *CostEstimationHandler) GetCostSummary(c *gin.Context) {
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

	// Get all repositories
	repos, err := h.store.GetRepositoriesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get repositories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve repositories"})
		return
	}

	// Get latest stats for all repos
	stats, err := h.store.GetLatestStatsForAllRepos(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get storage stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage stats"})
		return
	}

	// Get custom pricing
	customPricing, _ := h.store.GetStoragePricingByOrgID(c.Request.Context(), dbUser.OrgID)

	// Build custom pricing map
	pricingMap := make(map[string]*models.StoragePricing)
	for _, p := range customPricing {
		pricingMap[p.RepositoryType] = p
	}

	// Create repo map for names and types
	repoMap := make(map[uuid.UUID]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.ID] = repo
	}

	// Calculate costs
	var summary cost.CostSummary
	summary.ByType = make(map[string]float64)

	for _, stat := range stats {
		repo, ok := repoMap[stat.RepositoryID]
		if !ok {
			continue
		}

		// Use custom pricing if available, otherwise use defaults
		var costPerGB float64
		if custom, ok := pricingMap[string(repo.Type)]; ok {
			costPerGB = custom.StoragePerGBMonth
		} else {
			pricing := h.calculator.GetPricing(repo.Type)
			costPerGB = pricing.StoragePerGBMonth
		}

		estimate := h.calculator.EstimateRepositoryCost(
			repo.ID.String(),
			repo.Name,
			repo.Type,
			stat.RawDataSize,
		)
		estimate.CostPerGB = costPerGB
		estimate.MonthlyCost = estimate.StorageSizeGB * costPerGB
		estimate.YearlyCost = estimate.MonthlyCost * 12

		summary.TotalMonthlyCost += estimate.MonthlyCost
		summary.TotalYearlyCost += estimate.YearlyCost
		summary.TotalStorageSizeGB += estimate.StorageSizeGB
		summary.ByType[string(repo.Type)] += estimate.MonthlyCost
		summary.Repositories = append(summary.Repositories, estimate)
	}

	summary.RepositoryCount = len(summary.Repositories)

	// Calculate forecasts based on growth
	if len(stats) > 0 {
		growth, err := h.store.GetAllStorageGrowth(c.Request.Context(), dbUser.OrgID, 30)
		if err == nil && len(growth) > 1 {
			sizes := make([]int64, len(growth))
			for i, g := range growth {
				sizes[i] = g.RawDataSize
			}
			monthlyGrowthRate := h.calculator.CalculateGrowthRate(sizes, 30)

			// Get average cost per GB
			avgCostPerGB := 0.0
			if summary.TotalStorageSizeGB > 0 {
				avgCostPerGB = summary.TotalMonthlyCost / summary.TotalStorageSizeGB
			}

			summary.Forecasts = h.calculator.CalculateForecast(
				summary.TotalStorageSizeGB,
				monthlyGrowthRate,
				avgCostPerGB,
			)
		}
	}

	c.JSON(http.StatusOK, summary)
}

// ListRepositoryCosts returns cost estimates for all repositories.
// GET /api/v1/costs/repositories
func (h *CostEstimationHandler) ListRepositoryCosts(c *gin.Context) {
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

	repos, err := h.store.GetRepositoriesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get repositories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve repositories"})
		return
	}

	stats, err := h.store.GetLatestStatsForAllRepos(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get storage stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage stats"})
		return
	}

	// Get custom pricing
	customPricing, _ := h.store.GetStoragePricingByOrgID(c.Request.Context(), dbUser.OrgID)
	pricingMap := make(map[string]*models.StoragePricing)
	for _, p := range customPricing {
		pricingMap[p.RepositoryType] = p
	}

	repoMap := make(map[uuid.UUID]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.ID] = repo
	}

	statsMap := make(map[uuid.UUID]*models.StorageStats)
	for _, stat := range stats {
		statsMap[stat.RepositoryID] = stat
	}

	var estimates []cost.CostEstimate
	for _, repo := range repos {
		stat := statsMap[repo.ID]
		var storageSize int64
		if stat != nil {
			storageSize = stat.RawDataSize
		}

		var costPerGB float64
		if custom, ok := pricingMap[string(repo.Type)]; ok {
			costPerGB = custom.StoragePerGBMonth
		} else {
			pricing := h.calculator.GetPricing(repo.Type)
			costPerGB = pricing.StoragePerGBMonth
		}

		estimate := h.calculator.EstimateRepositoryCost(
			repo.ID.String(),
			repo.Name,
			repo.Type,
			storageSize,
		)
		estimate.CostPerGB = costPerGB
		estimate.MonthlyCost = estimate.StorageSizeGB * costPerGB
		estimate.YearlyCost = estimate.MonthlyCost * 12

		estimates = append(estimates, estimate)
	}

	c.JSON(http.StatusOK, gin.H{"repositories": estimates})
}

// GetRepositoryCost returns cost estimate for a specific repository.
// GET /api/v1/costs/repositories/:id
func (h *CostEstimationHandler) GetRepositoryCost(c *gin.Context) {
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

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify access"})
		return
	}

	repo, err := h.store.GetRepositoryByID(c.Request.Context(), repoID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	if repo.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	stat, _ := h.store.GetLatestStorageStats(c.Request.Context(), repoID)

	var storageSize int64
	if stat != nil {
		storageSize = stat.RawDataSize
	}

	// Check for custom pricing
	var costPerGB float64
	customPricing, err := h.store.GetStoragePricingByType(c.Request.Context(), dbUser.OrgID, string(repo.Type))
	if err == nil && customPricing != nil {
		costPerGB = customPricing.StoragePerGBMonth
	} else {
		pricing := h.calculator.GetPricing(repo.Type)
		costPerGB = pricing.StoragePerGBMonth
	}

	estimate := h.calculator.EstimateRepositoryCost(
		repo.ID.String(),
		repo.Name,
		repo.Type,
		storageSize,
	)
	estimate.CostPerGB = costPerGB
	estimate.MonthlyCost = estimate.StorageSizeGB * costPerGB
	estimate.YearlyCost = estimate.MonthlyCost * 12

	// Calculate repository-specific forecast
	growth, err := h.store.GetStorageGrowth(c.Request.Context(), repoID, 30)
	var forecasts []cost.CostForecast
	if err == nil && len(growth) > 1 {
		sizes := make([]int64, len(growth))
		for i, g := range growth {
			sizes[i] = g.RawDataSize
		}
		monthlyGrowthRate := h.calculator.CalculateGrowthRate(sizes, 30)
		forecasts = h.calculator.CalculateForecast(estimate.StorageSizeGB, monthlyGrowthRate, costPerGB)
	}

	c.JSON(http.StatusOK, gin.H{
		"estimate":  estimate,
		"forecasts": forecasts,
	})
}

// GetCostForecast returns cost forecasts for the organization.
// GET /api/v1/costs/forecast?months=3
func (h *CostEstimationHandler) GetCostForecast(c *gin.Context) {
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

	// Get storage growth history
	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	growth, err := h.store.GetAllStorageGrowth(c.Request.Context(), dbUser.OrgID, days)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get storage growth")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage growth"})
		return
	}

	if len(growth) < 2 {
		c.JSON(http.StatusOK, gin.H{
			"forecasts": []cost.CostForecast{},
			"message":   "insufficient data for forecasting",
		})
		return
	}

	// Get current costs
	stats, err := h.store.GetLatestStatsForAllRepos(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get storage stats")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve storage stats"})
		return
	}

	repos, err := h.store.GetRepositoriesByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get repositories")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve repositories"})
		return
	}

	// Get custom pricing
	customPricing, _ := h.store.GetStoragePricingByOrgID(c.Request.Context(), dbUser.OrgID)
	pricingMap := make(map[string]*models.StoragePricing)
	for _, p := range customPricing {
		pricingMap[p.RepositoryType] = p
	}

	repoMap := make(map[uuid.UUID]*models.Repository)
	for _, repo := range repos {
		repoMap[repo.ID] = repo
	}

	var totalStorageGB, totalMonthlyCost float64
	for _, stat := range stats {
		repo, ok := repoMap[stat.RepositoryID]
		if !ok {
			continue
		}

		var costPerGB float64
		if custom, ok := pricingMap[string(repo.Type)]; ok {
			costPerGB = custom.StoragePerGBMonth
		} else {
			pricing := h.calculator.GetPricing(repo.Type)
			costPerGB = pricing.StoragePerGBMonth
		}

		storageGB := float64(stat.RawDataSize) / (1024 * 1024 * 1024)
		totalStorageGB += storageGB
		totalMonthlyCost += storageGB * costPerGB
	}

	// Calculate growth rate
	sizes := make([]int64, len(growth))
	for i, g := range growth {
		sizes[i] = g.RawDataSize
	}
	monthlyGrowthRate := h.calculator.CalculateGrowthRate(sizes, days)

	// Get average cost per GB
	avgCostPerGB := 0.0
	if totalStorageGB > 0 {
		avgCostPerGB = totalMonthlyCost / totalStorageGB
	}

	forecasts := h.calculator.CalculateForecast(totalStorageGB, monthlyGrowthRate, avgCostPerGB)

	c.JSON(http.StatusOK, gin.H{
		"forecasts":           forecasts,
		"current_storage_gb":  totalStorageGB,
		"current_monthly_cost": totalMonthlyCost,
		"monthly_growth_rate": monthlyGrowthRate,
	})
}

// GetCostHistory returns historical cost data.
// GET /api/v1/costs/history?days=30
func (h *CostEstimationHandler) GetCostHistory(c *gin.Context) {
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

	days := 30
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	estimates, err := h.store.GetLatestCostEstimates(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get cost estimates")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve cost history"})
		return
	}

	// Also get growth data for chart
	growth, _ := h.store.GetAllStorageGrowth(c.Request.Context(), dbUser.OrgID, days)

	c.JSON(http.StatusOK, gin.H{
		"estimates":     estimates,
		"storage_growth": growth,
	})
}

// ListPricing returns all custom pricing configurations.
// GET /api/v1/pricing
func (h *CostEstimationHandler) ListPricing(c *gin.Context) {
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

	pricing, err := h.store.GetStoragePricingByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get pricing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve pricing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"pricing": pricing})
}

// GetDefaultPricing returns the default pricing configuration.
// GET /api/v1/pricing/defaults
func (h *CostEstimationHandler) GetDefaultPricing(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	defaults := make(map[string]cost.StoragePricing)
	for repoType, pricing := range cost.DefaultPricing {
		defaults[string(repoType)] = pricing
	}

	c.JSON(http.StatusOK, gin.H{"defaults": defaults})
}

// CreatePricing creates a new custom pricing configuration.
// POST /api/v1/pricing
func (h *CostEstimationHandler) CreatePricing(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.CreateStoragePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	pricing := models.NewStoragePricing(dbUser.OrgID, req.RepositoryType)
	pricing.StoragePerGBMonth = req.StoragePerGBMonth
	pricing.EgressPerGB = req.EgressPerGB
	pricing.OperationsPerK = req.OperationsPerK
	pricing.ProviderName = req.ProviderName
	pricing.ProviderDescription = req.ProviderDescription

	if err := h.store.CreateStoragePricing(c.Request.Context(), pricing); err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to create pricing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create pricing"})
		return
	}

	c.JSON(http.StatusCreated, pricing)
}

// UpdatePricing updates an existing pricing configuration.
// PUT /api/v1/pricing/:id
func (h *CostEstimationHandler) UpdatePricing(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pricing ID"})
		return
	}

	var req models.UpdateStoragePricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Get existing pricing by querying all and filtering
	allPricing, err := h.store.GetStoragePricingByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing not found"})
		return
	}

	var pricing *models.StoragePricing
	for _, p := range allPricing {
		if p.ID == id {
			pricing = p
			break
		}
	}

	if pricing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing not found"})
		return
	}

	// Apply updates
	if req.StoragePerGBMonth != nil {
		pricing.StoragePerGBMonth = *req.StoragePerGBMonth
	}
	if req.EgressPerGB != nil {
		pricing.EgressPerGB = *req.EgressPerGB
	}
	if req.OperationsPerK != nil {
		pricing.OperationsPerK = *req.OperationsPerK
	}
	if req.ProviderName != nil {
		pricing.ProviderName = *req.ProviderName
	}
	if req.ProviderDescription != nil {
		pricing.ProviderDescription = *req.ProviderDescription
	}

	if err := h.store.UpdateStoragePricing(c.Request.Context(), pricing); err != nil {
		h.logger.Error().Err(err).Str("pricing_id", id.String()).Msg("failed to update pricing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update pricing"})
		return
	}

	c.JSON(http.StatusOK, pricing)
}

// DeletePricing deletes a pricing configuration.
// DELETE /api/v1/pricing/:id
func (h *CostEstimationHandler) DeletePricing(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pricing ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	// Verify ownership
	allPricing, _ := h.store.GetStoragePricingByOrgID(c.Request.Context(), dbUser.OrgID)
	found := false
	for _, p := range allPricing {
		if p.ID == id {
			found = true
			break
		}
	}

	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "pricing not found"})
		return
	}

	if err := h.store.DeleteStoragePricing(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("pricing_id", id.String()).Msg("failed to delete pricing")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete pricing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "pricing deleted"})
}

// ListCostAlerts returns all cost alerts for the organization.
// GET /api/v1/cost-alerts
func (h *CostEstimationHandler) ListCostAlerts(c *gin.Context) {
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

	alerts, err := h.store.GetCostAlertsByOrgID(c.Request.Context(), dbUser.OrgID)
	if err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to get cost alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve cost alerts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// GetCostAlert returns a specific cost alert.
// GET /api/v1/cost-alerts/:id
func (h *CostEstimationHandler) GetCostAlert(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	alert, err := h.store.GetCostAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// CreateCostAlert creates a new cost alert.
// POST /api/v1/cost-alerts
func (h *CostEstimationHandler) CreateCostAlert(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	var req models.CreateCostAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	alert := models.NewCostAlert(dbUser.OrgID, req.Name, req.MonthlyThreshold)
	if req.Enabled != nil {
		alert.Enabled = *req.Enabled
	}
	if req.NotifyOnExceed != nil {
		alert.NotifyOnExceed = *req.NotifyOnExceed
	}
	if req.NotifyOnForecast != nil {
		alert.NotifyOnForecast = *req.NotifyOnForecast
	}
	if req.ForecastMonths != nil {
		alert.ForecastMonths = *req.ForecastMonths
	}

	if err := h.store.CreateCostAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("org_id", dbUser.OrgID.String()).Msg("failed to create cost alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create cost alert"})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

// UpdateCostAlert updates an existing cost alert.
// PUT /api/v1/cost-alerts/:id
func (h *CostEstimationHandler) UpdateCostAlert(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	var req models.UpdateCostAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	alert, err := h.store.GetCostAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	// Apply updates
	if req.Name != nil {
		alert.Name = *req.Name
	}
	if req.MonthlyThreshold != nil {
		alert.MonthlyThreshold = *req.MonthlyThreshold
	}
	if req.Enabled != nil {
		alert.Enabled = *req.Enabled
	}
	if req.NotifyOnExceed != nil {
		alert.NotifyOnExceed = *req.NotifyOnExceed
	}
	if req.NotifyOnForecast != nil {
		alert.NotifyOnForecast = *req.NotifyOnForecast
	}
	if req.ForecastMonths != nil {
		alert.ForecastMonths = *req.ForecastMonths
	}

	if err := h.store.UpdateCostAlert(c.Request.Context(), alert); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to update cost alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update cost alert"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

// DeleteCostAlert deletes a cost alert.
// DELETE /api/v1/cost-alerts/:id
func (h *CostEstimationHandler) DeleteCostAlert(c *gin.Context) {
	user := middleware.RequireUser(c)
	if user == nil {
		return
	}

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert ID"})
		return
	}

	dbUser, err := h.store.GetUserByID(c.Request.Context(), user.ID)
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", user.ID.String()).Msg("failed to get user")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve user"})
		return
	}

	alert, err := h.store.GetCostAlertByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	if alert.OrgID != dbUser.OrgID {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	if err := h.store.DeleteCostAlert(c.Request.Context(), id); err != nil {
		h.logger.Error().Err(err).Str("alert_id", id.String()).Msg("failed to delete cost alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete cost alert"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cost alert deleted"})
}
