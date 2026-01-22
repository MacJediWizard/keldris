// Package cost provides cloud storage cost estimation functionality.
package cost

import (
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
)

// DefaultPricing contains default pricing per GB per month for common cloud storage providers.
// Prices are in USD and represent storage costs only (not egress/transfer costs).
var DefaultPricing = map[models.RepositoryType]StoragePricing{
	models.RepositoryTypeS3: {
		StoragePerGBMonth:   0.023,  // S3 Standard
		EgressPerGB:         0.09,   // First 10TB/month
		OperationsPerK:      0.005,  // PUT/COPY/POST/LIST per 1000
		ProviderName:        "AWS S3",
		ProviderDescription: "Amazon S3 Standard storage",
	},
	models.RepositoryTypeB2: {
		StoragePerGBMonth:   0.006, // Backblaze B2
		EgressPerGB:         0.01,  // B2 egress
		OperationsPerK:      0.004, // Class B transactions per 10000
		ProviderName:        "Backblaze B2",
		ProviderDescription: "Backblaze B2 Cloud Storage",
	},
	models.RepositoryTypeLocal: {
		StoragePerGBMonth:   0.0, // Local storage has no cloud costs
		EgressPerGB:         0.0,
		OperationsPerK:      0.0,
		ProviderName:        "Local",
		ProviderDescription: "Local filesystem storage",
	},
	models.RepositoryTypeSFTP: {
		StoragePerGBMonth:   0.0, // SFTP costs depend on hosting
		EgressPerGB:         0.0,
		OperationsPerK:      0.0,
		ProviderName:        "SFTP",
		ProviderDescription: "SFTP server storage",
	},
	models.RepositoryTypeRest: {
		StoragePerGBMonth:   0.0, // REST server costs depend on hosting
		EgressPerGB:         0.0,
		OperationsPerK:      0.0,
		ProviderName:        "REST",
		ProviderDescription: "Restic REST server",
	},
	models.RepositoryTypeDropbox: {
		StoragePerGBMonth:   0.0, // Dropbox has subscription pricing
		EgressPerGB:         0.0,
		OperationsPerK:      0.0,
		ProviderName:        "Dropbox",
		ProviderDescription: "Dropbox cloud storage",
	},
}

// WasabiPricing represents Wasabi S3-compatible pricing (used via S3 backend).
var WasabiPricing = StoragePricing{
	StoragePerGBMonth:   0.0069, // Wasabi hot storage
	EgressPerGB:         0.0,    // No egress fees
	OperationsPerK:      0.0,    // No API fees
	ProviderName:        "Wasabi",
	ProviderDescription: "Wasabi Hot Cloud Storage",
}

// StoragePricing defines pricing structure for a storage provider.
type StoragePricing struct {
	StoragePerGBMonth   float64 `json:"storage_per_gb_month"`
	EgressPerGB         float64 `json:"egress_per_gb"`
	OperationsPerK      float64 `json:"operations_per_k"`
	ProviderName        string  `json:"provider_name"`
	ProviderDescription string  `json:"provider_description"`
}

// Calculator provides cost estimation functionality.
type Calculator struct {
	pricing map[models.RepositoryType]StoragePricing
}

// NewCalculator creates a new cost calculator with default pricing.
func NewCalculator() *Calculator {
	return &Calculator{
		pricing: DefaultPricing,
	}
}

// NewCalculatorWithPricing creates a new cost calculator with custom pricing.
func NewCalculatorWithPricing(pricing map[models.RepositoryType]StoragePricing) *Calculator {
	return &Calculator{
		pricing: pricing,
	}
}

// SetPricing updates pricing for a specific repository type.
func (c *Calculator) SetPricing(repoType models.RepositoryType, pricing StoragePricing) {
	c.pricing[repoType] = pricing
}

// GetPricing returns pricing for a specific repository type.
func (c *Calculator) GetPricing(repoType models.RepositoryType) StoragePricing {
	if p, ok := c.pricing[repoType]; ok {
		return p
	}
	return StoragePricing{}
}

// CostEstimate represents a cost estimation result.
type CostEstimate struct {
	RepositoryID        string          `json:"repository_id"`
	RepositoryName      string          `json:"repository_name"`
	RepositoryType      string          `json:"repository_type"`
	StorageSizeBytes    int64           `json:"storage_size_bytes"`
	StorageSizeGB       float64         `json:"storage_size_gb"`
	MonthlyCost         float64         `json:"monthly_cost"`
	YearlyCost          float64         `json:"yearly_cost"`
	CostPerGB           float64         `json:"cost_per_gb"`
	Pricing             StoragePricing  `json:"pricing"`
	EstimatedAt         time.Time       `json:"estimated_at"`
}

// CostForecast represents projected costs over time.
type CostForecast struct {
	Period          string  `json:"period"`
	Months          int     `json:"months"`
	ProjectedSizeGB float64 `json:"projected_size_gb"`
	ProjectedCost   float64 `json:"projected_cost"`
	GrowthRate      float64 `json:"growth_rate"`
}

// CostSummary represents aggregated cost information.
type CostSummary struct {
	TotalMonthlyCost     float64                        `json:"total_monthly_cost"`
	TotalYearlyCost      float64                        `json:"total_yearly_cost"`
	TotalStorageSizeGB   float64                        `json:"total_storage_size_gb"`
	RepositoryCount      int                            `json:"repository_count"`
	ByType               map[string]float64             `json:"by_type"`
	Repositories         []CostEstimate                 `json:"repositories"`
	Forecasts            []CostForecast                 `json:"forecasts"`
	EstimatedAt          time.Time                      `json:"estimated_at"`
}

// EstimateRepositoryCost calculates estimated cost for a single repository.
func (c *Calculator) EstimateRepositoryCost(repoID, repoName string, repoType models.RepositoryType, storageSizeBytes int64) CostEstimate {
	pricing := c.GetPricing(repoType)
	storageSizeGB := float64(storageSizeBytes) / (1024 * 1024 * 1024)
	monthlyCost := storageSizeGB * pricing.StoragePerGBMonth

	return CostEstimate{
		RepositoryID:     repoID,
		RepositoryName:   repoName,
		RepositoryType:   string(repoType),
		StorageSizeBytes: storageSizeBytes,
		StorageSizeGB:    storageSizeGB,
		MonthlyCost:      monthlyCost,
		YearlyCost:       monthlyCost * 12,
		CostPerGB:        pricing.StoragePerGBMonth,
		Pricing:          pricing,
		EstimatedAt:      time.Now(),
	}
}

// CalculateForecast projects future costs based on historical growth.
func (c *Calculator) CalculateForecast(currentSizeGB float64, monthlyGrowthRate float64, costPerGB float64) []CostForecast {
	forecasts := []CostForecast{
		{Period: "3 months", Months: 3},
		{Period: "6 months", Months: 6},
		{Period: "12 months", Months: 12},
	}

	for i := range forecasts {
		// Compound growth: size * (1 + rate)^months
		growthMultiplier := 1.0
		for j := 0; j < forecasts[i].Months; j++ {
			growthMultiplier *= (1 + monthlyGrowthRate)
		}
		projectedSize := currentSizeGB * growthMultiplier
		forecasts[i].ProjectedSizeGB = projectedSize
		forecasts[i].ProjectedCost = projectedSize * costPerGB
		forecasts[i].GrowthRate = monthlyGrowthRate
	}

	return forecasts
}

// CalculateGrowthRate calculates monthly growth rate from historical data points.
// Returns 0 if insufficient data or no growth.
func (c *Calculator) CalculateGrowthRate(historicalSizes []int64, days int) float64 {
	if len(historicalSizes) < 2 {
		return 0
	}

	// Calculate average daily growth rate
	oldestSize := historicalSizes[0]
	newestSize := historicalSizes[len(historicalSizes)-1]

	if oldestSize <= 0 || newestSize <= oldestSize {
		return 0
	}

	// Growth rate = (new/old)^(1/days) - 1
	dailyGrowth := float64(newestSize)/float64(oldestSize) - 1
	if days > 0 {
		dailyGrowth = dailyGrowth / float64(days)
	}

	// Convert to monthly (assuming 30 days per month)
	monthlyGrowth := dailyGrowth * 30

	// Cap at reasonable growth rate (500% per month max)
	if monthlyGrowth > 5.0 {
		monthlyGrowth = 5.0
	}

	return monthlyGrowth
}

// CostAlert represents a cost threshold alert configuration.
type CostAlert struct {
	ID                string    `json:"id"`
	OrgID             string    `json:"org_id"`
	Name              string    `json:"name"`
	MonthlyThreshold  float64   `json:"monthly_threshold"`
	Enabled           bool      `json:"enabled"`
	NotifyOnExceed    bool      `json:"notify_on_exceed"`
	NotifyOnForecast  bool      `json:"notify_on_forecast"`
	ForecastMonths    int       `json:"forecast_months"`
	LastTriggeredAt   *time.Time `json:"last_triggered_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CheckCostAlert checks if current or forecasted costs exceed alert threshold.
func (c *Calculator) CheckCostAlert(alert CostAlert, currentMonthlyCost float64, forecasts []CostForecast) (exceeded bool, reason string) {
	if !alert.Enabled {
		return false, ""
	}

	if alert.NotifyOnExceed && currentMonthlyCost >= alert.MonthlyThreshold {
		return true, "current monthly cost exceeds threshold"
	}

	if alert.NotifyOnForecast {
		for _, forecast := range forecasts {
			if forecast.Months == alert.ForecastMonths && forecast.ProjectedCost >= alert.MonthlyThreshold {
				return true, "forecasted cost exceeds threshold"
			}
		}
	}

	return false, ""
}
