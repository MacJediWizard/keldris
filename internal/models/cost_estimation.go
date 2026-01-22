package models

import (
	"time"

	"github.com/google/uuid"
)

// StoragePricing represents custom pricing configuration for a repository type.
type StoragePricing struct {
	ID                  uuid.UUID `json:"id"`
	OrgID               uuid.UUID `json:"org_id"`
	RepositoryType      string    `json:"repository_type"`
	StoragePerGBMonth   float64   `json:"storage_per_gb_month"`
	EgressPerGB         float64   `json:"egress_per_gb"`
	OperationsPerK      float64   `json:"operations_per_k"`
	ProviderName        string    `json:"provider_name,omitempty"`
	ProviderDescription string    `json:"provider_description,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// NewStoragePricing creates a new StoragePricing record.
func NewStoragePricing(orgID uuid.UUID, repoType string) *StoragePricing {
	now := time.Now()
	return &StoragePricing{
		ID:             uuid.New(),
		OrgID:          orgID,
		RepositoryType: repoType,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

// CostEstimateRecord represents a stored cost estimation for a repository.
type CostEstimateRecord struct {
	ID               uuid.UUID `json:"id"`
	OrgID            uuid.UUID `json:"org_id"`
	RepositoryID     uuid.UUID `json:"repository_id"`
	StorageSizeBytes int64     `json:"storage_size_bytes"`
	MonthlyCost      float64   `json:"monthly_cost"`
	YearlyCost       float64   `json:"yearly_cost"`
	CostPerGB        float64   `json:"cost_per_gb"`
	EstimatedAt      time.Time `json:"estimated_at"`
	CreatedAt        time.Time `json:"created_at"`
}

// NewCostEstimateRecord creates a new CostEstimateRecord.
func NewCostEstimateRecord(orgID, repositoryID uuid.UUID) *CostEstimateRecord {
	now := time.Now()
	return &CostEstimateRecord{
		ID:           uuid.New(),
		OrgID:        orgID,
		RepositoryID: repositoryID,
		EstimatedAt:  now,
		CreatedAt:    now,
	}
}

// CostAlert represents a cost alert configuration.
type CostAlert struct {
	ID               uuid.UUID  `json:"id"`
	OrgID            uuid.UUID  `json:"org_id"`
	Name             string     `json:"name"`
	MonthlyThreshold float64    `json:"monthly_threshold"`
	Enabled          bool       `json:"enabled"`
	NotifyOnExceed   bool       `json:"notify_on_exceed"`
	NotifyOnForecast bool       `json:"notify_on_forecast"`
	ForecastMonths   int        `json:"forecast_months"`
	LastTriggeredAt  *time.Time `json:"last_triggered_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// NewCostAlert creates a new CostAlert.
func NewCostAlert(orgID uuid.UUID, name string, threshold float64) *CostAlert {
	now := time.Now()
	return &CostAlert{
		ID:               uuid.New(),
		OrgID:            orgID,
		Name:             name,
		MonthlyThreshold: threshold,
		Enabled:          true,
		NotifyOnExceed:   true,
		NotifyOnForecast: false,
		ForecastMonths:   3,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// CreateStoragePricingRequest represents a request to create custom pricing.
type CreateStoragePricingRequest struct {
	RepositoryType      string  `json:"repository_type" binding:"required"`
	StoragePerGBMonth   float64 `json:"storage_per_gb_month" binding:"gte=0"`
	EgressPerGB         float64 `json:"egress_per_gb" binding:"gte=0"`
	OperationsPerK      float64 `json:"operations_per_k" binding:"gte=0"`
	ProviderName        string  `json:"provider_name"`
	ProviderDescription string  `json:"provider_description"`
}

// UpdateStoragePricingRequest represents a request to update custom pricing.
type UpdateStoragePricingRequest struct {
	StoragePerGBMonth   *float64 `json:"storage_per_gb_month,omitempty"`
	EgressPerGB         *float64 `json:"egress_per_gb,omitempty"`
	OperationsPerK      *float64 `json:"operations_per_k,omitempty"`
	ProviderName        *string  `json:"provider_name,omitempty"`
	ProviderDescription *string  `json:"provider_description,omitempty"`
}

// CreateCostAlertRequest represents a request to create a cost alert.
type CreateCostAlertRequest struct {
	Name             string  `json:"name" binding:"required"`
	MonthlyThreshold float64 `json:"monthly_threshold" binding:"required,gt=0"`
	Enabled          *bool   `json:"enabled"`
	NotifyOnExceed   *bool   `json:"notify_on_exceed"`
	NotifyOnForecast *bool   `json:"notify_on_forecast"`
	ForecastMonths   *int    `json:"forecast_months"`
}

// UpdateCostAlertRequest represents a request to update a cost alert.
type UpdateCostAlertRequest struct {
	Name             *string  `json:"name,omitempty"`
	MonthlyThreshold *float64 `json:"monthly_threshold,omitempty"`
	Enabled          *bool    `json:"enabled,omitempty"`
	NotifyOnExceed   *bool    `json:"notify_on_exceed,omitempty"`
	NotifyOnForecast *bool    `json:"notify_on_forecast,omitempty"`
	ForecastMonths   *int     `json:"forecast_months,omitempty"`
}
