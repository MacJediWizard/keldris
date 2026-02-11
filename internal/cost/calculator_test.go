package cost

import (
	"math"
	"testing"

	"github.com/MacJediWizard/keldris/internal/models"
)

func TestNewCalculator(t *testing.T) {
	calc := NewCalculator()
	if calc == nil {
		t.Fatal("expected non-nil calculator")
	}
	if calc.pricing == nil {
		t.Fatal("expected non-nil pricing map")
	}

	// Should have default pricing for all known types.
	for _, rt := range models.ValidRepositoryTypes() {
		p := calc.GetPricing(rt)
		if p.ProviderName == "" {
			t.Errorf("expected provider name for %s", rt)
		}
	}
}

func TestNewCalculatorWithPricing(t *testing.T) {
	custom := map[models.RepositoryType]StoragePricing{
		models.RepositoryTypeS3: {
			StoragePerGBMonth: 0.05,
			ProviderName:      "Custom S3",
		},
	}
	calc := NewCalculatorWithPricing(custom)
	if calc == nil {
		t.Fatal("expected non-nil calculator")
	}

	p := calc.GetPricing(models.RepositoryTypeS3)
	if p.StoragePerGBMonth != 0.05 {
		t.Errorf("expected 0.05, got %f", p.StoragePerGBMonth)
	}
	if p.ProviderName != "Custom S3" {
		t.Errorf("expected 'Custom S3', got %q", p.ProviderName)
	}

	// Types not in the custom map should return zero-value.
	p = calc.GetPricing(models.RepositoryTypeB2)
	if p.ProviderName != "" {
		t.Errorf("expected empty provider name for missing type, got %q", p.ProviderName)
	}
}

func TestSetPricing(t *testing.T) {
	// Use a separate pricing map to avoid mutating the shared DefaultPricing.
	custom := map[models.RepositoryType]StoragePricing{
		models.RepositoryTypeS3: {
			StoragePerGBMonth: 0.023,
			ProviderName:      "AWS S3",
		},
	}
	calc := NewCalculatorWithPricing(custom)

	wasabi := WasabiPricing
	calc.SetPricing(models.RepositoryTypeS3, wasabi)

	p := calc.GetPricing(models.RepositoryTypeS3)
	if p.StoragePerGBMonth != wasabi.StoragePerGBMonth {
		t.Errorf("expected %f, got %f", wasabi.StoragePerGBMonth, p.StoragePerGBMonth)
	}
	if p.ProviderName != "Wasabi" {
		t.Errorf("expected 'Wasabi', got %q", p.ProviderName)
	}
}

func TestGetPricing(t *testing.T) {
	calc := NewCalculator()

	t.Run("known type returns correct pricing", func(t *testing.T) {
		p := calc.GetPricing(models.RepositoryTypeS3)
		if p.StoragePerGBMonth != 0.023 {
			t.Errorf("expected 0.023, got %f", p.StoragePerGBMonth)
		}
		if p.EgressPerGB != 0.09 {
			t.Errorf("expected 0.09, got %f", p.EgressPerGB)
		}
		if p.ProviderName != "AWS S3" {
			t.Errorf("expected 'AWS S3', got %q", p.ProviderName)
		}
	})

	t.Run("unknown type returns zero-value", func(t *testing.T) {
		p := calc.GetPricing(models.RepositoryType("unknown"))
		if p.StoragePerGBMonth != 0 {
			t.Errorf("expected 0, got %f", p.StoragePerGBMonth)
		}
		if p.ProviderName != "" {
			t.Errorf("expected empty string, got %q", p.ProviderName)
		}
	})

	t.Run("all default types have pricing", func(t *testing.T) {
		types := map[models.RepositoryType]string{
			models.RepositoryTypeS3:      "AWS S3",
			models.RepositoryTypeB2:      "Backblaze B2",
			models.RepositoryTypeLocal:   "Local",
			models.RepositoryTypeSFTP:    "SFTP",
			models.RepositoryTypeRest:    "REST",
			models.RepositoryTypeDropbox: "Dropbox",
		}
		for rt, name := range types {
			p := calc.GetPricing(rt)
			if p.ProviderName != name {
				t.Errorf("type %s: expected %q, got %q", rt, name, p.ProviderName)
			}
		}
	})
}

func TestEstimateRepositoryCost(t *testing.T) {
	calc := NewCalculator()

	t.Run("S3 cost calculation", func(t *testing.T) {
		// 100 GB in bytes
		sizeBytes := int64(100 * 1024 * 1024 * 1024)
		est := calc.EstimateRepositoryCost("repo-1", "my-repo", models.RepositoryTypeS3, sizeBytes)

		if est.RepositoryID != "repo-1" {
			t.Errorf("expected 'repo-1', got %q", est.RepositoryID)
		}
		if est.RepositoryName != "my-repo" {
			t.Errorf("expected 'my-repo', got %q", est.RepositoryName)
		}
		if est.RepositoryType != "s3" {
			t.Errorf("expected 's3', got %q", est.RepositoryType)
		}
		if est.StorageSizeBytes != sizeBytes {
			t.Errorf("expected %d, got %d", sizeBytes, est.StorageSizeBytes)
		}

		expectedGB := 100.0
		if math.Abs(est.StorageSizeGB-expectedGB) > 0.001 {
			t.Errorf("expected ~%.2f GB, got %.2f GB", expectedGB, est.StorageSizeGB)
		}

		expectedMonthly := expectedGB * 0.023
		if math.Abs(est.MonthlyCost-expectedMonthly) > 0.001 {
			t.Errorf("expected monthly cost ~%.4f, got %.4f", expectedMonthly, est.MonthlyCost)
		}

		expectedYearly := expectedMonthly * 12
		if math.Abs(est.YearlyCost-expectedYearly) > 0.01 {
			t.Errorf("expected yearly cost ~%.4f, got %.4f", expectedYearly, est.YearlyCost)
		}

		if est.CostPerGB != 0.023 {
			t.Errorf("expected cost per GB 0.023, got %f", est.CostPerGB)
		}
		if est.Pricing.ProviderName != "AWS S3" {
			t.Errorf("expected 'AWS S3', got %q", est.Pricing.ProviderName)
		}
		if est.EstimatedAt.IsZero() {
			t.Error("expected non-zero estimated_at")
		}
	})

	t.Run("B2 cost calculation", func(t *testing.T) {
		sizeBytes := int64(500 * 1024 * 1024 * 1024) // 500 GB
		est := calc.EstimateRepositoryCost("repo-2", "b2-repo", models.RepositoryTypeB2, sizeBytes)

		expectedMonthly := 500.0 * 0.006
		if math.Abs(est.MonthlyCost-expectedMonthly) > 0.001 {
			t.Errorf("expected monthly cost ~%.4f, got %.4f", expectedMonthly, est.MonthlyCost)
		}
	})

	t.Run("local storage zero cost", func(t *testing.T) {
		sizeBytes := int64(1024 * 1024 * 1024 * 1024) // 1 TB
		est := calc.EstimateRepositoryCost("repo-3", "local-repo", models.RepositoryTypeLocal, sizeBytes)

		if est.MonthlyCost != 0 {
			t.Errorf("expected zero monthly cost for local, got %f", est.MonthlyCost)
		}
		if est.YearlyCost != 0 {
			t.Errorf("expected zero yearly cost for local, got %f", est.YearlyCost)
		}
	})

	t.Run("zero size", func(t *testing.T) {
		est := calc.EstimateRepositoryCost("repo-4", "empty-repo", models.RepositoryTypeS3, 0)

		if est.StorageSizeGB != 0 {
			t.Errorf("expected 0 GB, got %f", est.StorageSizeGB)
		}
		if est.MonthlyCost != 0 {
			t.Errorf("expected 0 monthly cost, got %f", est.MonthlyCost)
		}
	})

	t.Run("unknown type zero cost", func(t *testing.T) {
		est := calc.EstimateRepositoryCost("repo-5", "unknown-repo", models.RepositoryType("unknown"), 1024*1024*1024)

		if est.MonthlyCost != 0 {
			t.Errorf("expected zero cost for unknown type, got %f", est.MonthlyCost)
		}
	})

	t.Run("custom pricing via Wasabi", func(t *testing.T) {
		custom := map[models.RepositoryType]StoragePricing{
			models.RepositoryTypeS3: DefaultPricing[models.RepositoryTypeS3],
		}
		calc2 := NewCalculatorWithPricing(custom)
		calc2.SetPricing(models.RepositoryTypeS3, WasabiPricing)

		sizeBytes := int64(100 * 1024 * 1024 * 1024) // 100 GB
		est := calc2.EstimateRepositoryCost("repo-6", "wasabi-repo", models.RepositoryTypeS3, sizeBytes)

		expectedMonthly := 100.0 * 0.0069
		if math.Abs(est.MonthlyCost-expectedMonthly) > 0.001 {
			t.Errorf("expected monthly cost ~%.4f, got %.4f", expectedMonthly, est.MonthlyCost)
		}
		if est.Pricing.ProviderName != "Wasabi" {
			t.Errorf("expected 'Wasabi', got %q", est.Pricing.ProviderName)
		}
	})
}

func TestCalculateForecast(t *testing.T) {
	calc := NewCalculator()

	t.Run("basic forecast", func(t *testing.T) {
		forecasts := calc.CalculateForecast(100.0, 0.10, 0.023)

		if len(forecasts) != 3 {
			t.Fatalf("expected 3 forecasts, got %d", len(forecasts))
		}

		// Verify periods.
		expectedPeriods := []string{"3 months", "6 months", "12 months"}
		expectedMonths := []int{3, 6, 12}
		for i, f := range forecasts {
			if f.Period != expectedPeriods[i] {
				t.Errorf("forecast %d: expected period %q, got %q", i, expectedPeriods[i], f.Period)
			}
			if f.Months != expectedMonths[i] {
				t.Errorf("forecast %d: expected months %d, got %d", i, expectedMonths[i], f.Months)
			}
			if f.GrowthRate != 0.10 {
				t.Errorf("forecast %d: expected growth rate 0.10, got %f", i, f.GrowthRate)
			}
		}

		// 3-month forecast: 100 * (1.1)^3 = 133.1
		expected3m := 100.0 * math.Pow(1.10, 3)
		if math.Abs(forecasts[0].ProjectedSizeGB-expected3m) > 0.01 {
			t.Errorf("3-month size: expected ~%.2f, got %.2f", expected3m, forecasts[0].ProjectedSizeGB)
		}
		expectedCost3m := expected3m * 0.023
		if math.Abs(forecasts[0].ProjectedCost-expectedCost3m) > 0.001 {
			t.Errorf("3-month cost: expected ~%.4f, got %.4f", expectedCost3m, forecasts[0].ProjectedCost)
		}

		// 12-month forecast: 100 * (1.1)^12 = ~313.84
		expected12m := 100.0 * math.Pow(1.10, 12)
		if math.Abs(forecasts[2].ProjectedSizeGB-expected12m) > 0.01 {
			t.Errorf("12-month size: expected ~%.2f, got %.2f", expected12m, forecasts[2].ProjectedSizeGB)
		}
	})

	t.Run("zero growth rate", func(t *testing.T) {
		forecasts := calc.CalculateForecast(50.0, 0.0, 0.006)

		for _, f := range forecasts {
			if math.Abs(f.ProjectedSizeGB-50.0) > 0.001 {
				t.Errorf("expected projected size 50.0 with zero growth, got %f", f.ProjectedSizeGB)
			}
			expectedCost := 50.0 * 0.006
			if math.Abs(f.ProjectedCost-expectedCost) > 0.001 {
				t.Errorf("expected projected cost ~%.4f, got %.4f", expectedCost, f.ProjectedCost)
			}
		}
	})

	t.Run("zero size", func(t *testing.T) {
		forecasts := calc.CalculateForecast(0.0, 0.10, 0.023)

		for _, f := range forecasts {
			if f.ProjectedSizeGB != 0 {
				t.Errorf("expected 0 projected size, got %f", f.ProjectedSizeGB)
			}
			if f.ProjectedCost != 0 {
				t.Errorf("expected 0 projected cost, got %f", f.ProjectedCost)
			}
		}
	})

	t.Run("zero cost per GB", func(t *testing.T) {
		forecasts := calc.CalculateForecast(100.0, 0.10, 0.0)

		for _, f := range forecasts {
			if f.ProjectedCost != 0 {
				t.Errorf("expected 0 projected cost with zero cost/GB, got %f", f.ProjectedCost)
			}
			// Size should still grow.
			if f.ProjectedSizeGB <= 100.0 {
				t.Errorf("expected projected size > 100 with growth, got %f", f.ProjectedSizeGB)
			}
		}
	})

	t.Run("high growth rate", func(t *testing.T) {
		forecasts := calc.CalculateForecast(10.0, 1.0, 0.023)

		// 3-month: 10 * 2^3 = 80
		expected3m := 10.0 * math.Pow(2.0, 3)
		if math.Abs(forecasts[0].ProjectedSizeGB-expected3m) > 0.01 {
			t.Errorf("3-month: expected ~%.2f, got %.2f", expected3m, forecasts[0].ProjectedSizeGB)
		}
	})
}

func TestCalculateGrowthRate(t *testing.T) {
	calc := NewCalculator()

	t.Run("normal growth", func(t *testing.T) {
		sizes := []int64{1000, 1200, 1500, 2000}
		rate := calc.CalculateGrowthRate(sizes, 30)

		// Growth = (2000/1000 - 1) / 30 * 30 = 1.0
		if math.Abs(rate-1.0) > 0.001 {
			t.Errorf("expected growth rate ~1.0, got %f", rate)
		}
	})

	t.Run("insufficient data single element", func(t *testing.T) {
		rate := calc.CalculateGrowthRate([]int64{1000}, 30)
		if rate != 0 {
			t.Errorf("expected 0 for single element, got %f", rate)
		}
	})

	t.Run("insufficient data empty", func(t *testing.T) {
		rate := calc.CalculateGrowthRate([]int64{}, 30)
		if rate != 0 {
			t.Errorf("expected 0 for empty slice, got %f", rate)
		}
	})

	t.Run("nil slice", func(t *testing.T) {
		rate := calc.CalculateGrowthRate(nil, 30)
		if rate != 0 {
			t.Errorf("expected 0 for nil slice, got %f", rate)
		}
	})

	t.Run("no growth same values", func(t *testing.T) {
		sizes := []int64{1000, 1000}
		rate := calc.CalculateGrowthRate(sizes, 30)
		if rate != 0 {
			t.Errorf("expected 0 for no growth, got %f", rate)
		}
	})

	t.Run("negative growth", func(t *testing.T) {
		sizes := []int64{2000, 1000}
		rate := calc.CalculateGrowthRate(sizes, 30)
		if rate != 0 {
			t.Errorf("expected 0 for negative growth, got %f", rate)
		}
	})

	t.Run("oldest size zero", func(t *testing.T) {
		sizes := []int64{0, 1000}
		rate := calc.CalculateGrowthRate(sizes, 30)
		if rate != 0 {
			t.Errorf("expected 0 for zero oldest size, got %f", rate)
		}
	})

	t.Run("oldest size negative", func(t *testing.T) {
		sizes := []int64{-100, 1000}
		rate := calc.CalculateGrowthRate(sizes, 30)
		if rate != 0 {
			t.Errorf("expected 0 for negative oldest size, got %f", rate)
		}
	})

	t.Run("zero days", func(t *testing.T) {
		sizes := []int64{1000, 2000}
		rate := calc.CalculateGrowthRate(sizes, 0)

		// dailyGrowth = 1.0; not divided by days; monthlyGrowth = 1.0 * 30 = 30.0 -> capped at 5.0
		if rate != 5.0 {
			t.Errorf("expected capped rate 5.0 for zero days, got %f", rate)
		}
	})

	t.Run("growth rate capped at 5", func(t *testing.T) {
		sizes := []int64{100, 100000}
		rate := calc.CalculateGrowthRate(sizes, 1)

		if rate != 5.0 {
			t.Errorf("expected capped rate 5.0, got %f", rate)
		}
	})

	t.Run("moderate growth over 90 days", func(t *testing.T) {
		sizes := []int64{1000, 1100, 1200, 1300}
		rate := calc.CalculateGrowthRate(sizes, 90)

		// Growth = (1300/1000 - 1) / 90 * 30 = 0.3 / 90 * 30 = 0.1
		expectedRate := 0.1
		if math.Abs(rate-expectedRate) > 0.001 {
			t.Errorf("expected growth rate ~%.4f, got %f", expectedRate, rate)
		}
	})
}

func TestCheckCostAlert(t *testing.T) {
	calc := NewCalculator()

	t.Run("disabled alert", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          false,
			MonthlyThreshold: 10.0,
			NotifyOnExceed:   true,
		}
		exceeded, reason := calc.CheckCostAlert(alert, 100.0, nil)
		if exceeded {
			t.Error("expected disabled alert to not trigger")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("current cost exceeds threshold", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   true,
		}
		exceeded, reason := calc.CheckCostAlert(alert, 75.0, nil)
		if !exceeded {
			t.Error("expected alert to trigger when cost exceeds threshold")
		}
		if reason != "current monthly cost exceeds threshold" {
			t.Errorf("unexpected reason: %q", reason)
		}
	})

	t.Run("current cost equals threshold", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   true,
		}
		exceeded, reason := calc.CheckCostAlert(alert, 50.0, nil)
		if !exceeded {
			t.Error("expected alert to trigger when cost equals threshold")
		}
		if reason != "current monthly cost exceeds threshold" {
			t.Errorf("unexpected reason: %q", reason)
		}
	})

	t.Run("current cost below threshold", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   true,
		}
		exceeded, reason := calc.CheckCostAlert(alert, 25.0, nil)
		if exceeded {
			t.Error("expected alert to not trigger when cost is below threshold")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("exceed notification disabled", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   false,
		}
		exceeded, reason := calc.CheckCostAlert(alert, 100.0, nil)
		if exceeded {
			t.Error("expected alert to not trigger when NotifyOnExceed is false")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("forecast exceeds threshold", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 100.0,
			NotifyOnExceed:   false,
			NotifyOnForecast: true,
			ForecastMonths:   6,
		}
		forecasts := []CostForecast{
			{Period: "3 months", Months: 3, ProjectedCost: 50.0},
			{Period: "6 months", Months: 6, ProjectedCost: 150.0},
			{Period: "12 months", Months: 12, ProjectedCost: 300.0},
		}
		exceeded, reason := calc.CheckCostAlert(alert, 25.0, forecasts)
		if !exceeded {
			t.Error("expected alert to trigger on forecasted cost")
		}
		if reason != "forecasted cost exceeds threshold" {
			t.Errorf("unexpected reason: %q", reason)
		}
	})

	t.Run("forecast below threshold", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 200.0,
			NotifyOnExceed:   false,
			NotifyOnForecast: true,
			ForecastMonths:   6,
		}
		forecasts := []CostForecast{
			{Period: "3 months", Months: 3, ProjectedCost: 50.0},
			{Period: "6 months", Months: 6, ProjectedCost: 100.0},
			{Period: "12 months", Months: 12, ProjectedCost: 150.0},
		}
		exceeded, reason := calc.CheckCostAlert(alert, 25.0, forecasts)
		if exceeded {
			t.Error("expected alert to not trigger when forecast is below threshold")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("forecast notification disabled", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 10.0,
			NotifyOnExceed:   false,
			NotifyOnForecast: false,
			ForecastMonths:   6,
		}
		forecasts := []CostForecast{
			{Period: "6 months", Months: 6, ProjectedCost: 999.0},
		}
		exceeded, reason := calc.CheckCostAlert(alert, 5.0, forecasts)
		if exceeded {
			t.Error("expected alert to not trigger when forecast notifications are disabled")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("forecast months mismatch", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   false,
			NotifyOnForecast: true,
			ForecastMonths:   12,
		}
		forecasts := []CostForecast{
			{Period: "3 months", Months: 3, ProjectedCost: 200.0},
			{Period: "6 months", Months: 6, ProjectedCost: 200.0},
		}
		exceeded, reason := calc.CheckCostAlert(alert, 10.0, forecasts)
		if exceeded {
			t.Error("expected no trigger when forecast months don't match")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("both exceed and forecast with exceed triggering first", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   true,
			NotifyOnForecast: true,
			ForecastMonths:   6,
		}
		forecasts := []CostForecast{
			{Period: "6 months", Months: 6, ProjectedCost: 200.0},
		}
		exceeded, reason := calc.CheckCostAlert(alert, 75.0, forecasts)
		if !exceeded {
			t.Error("expected alert to trigger")
		}
		// Exceed check happens first.
		if reason != "current monthly cost exceeds threshold" {
			t.Errorf("unexpected reason: %q", reason)
		}
	})

	t.Run("empty forecasts with forecast enabled", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   false,
			NotifyOnForecast: true,
			ForecastMonths:   6,
		}
		exceeded, reason := calc.CheckCostAlert(alert, 10.0, []CostForecast{})
		if exceeded {
			t.Error("expected no trigger with empty forecasts")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})

	t.Run("nil forecasts with forecast enabled", func(t *testing.T) {
		alert := CostAlert{
			Enabled:          true,
			MonthlyThreshold: 50.0,
			NotifyOnExceed:   false,
			NotifyOnForecast: true,
			ForecastMonths:   6,
		}
		exceeded, reason := calc.CheckCostAlert(alert, 10.0, nil)
		if exceeded {
			t.Error("expected no trigger with nil forecasts")
		}
		if reason != "" {
			t.Errorf("expected empty reason, got %q", reason)
		}
	})
}

func TestDefaultPricingValues(t *testing.T) {
	t.Run("S3 pricing", func(t *testing.T) {
		p := DefaultPricing[models.RepositoryTypeS3]
		if p.StoragePerGBMonth != 0.023 {
			t.Errorf("expected 0.023, got %f", p.StoragePerGBMonth)
		}
		if p.EgressPerGB != 0.09 {
			t.Errorf("expected 0.09, got %f", p.EgressPerGB)
		}
		if p.OperationsPerK != 0.005 {
			t.Errorf("expected 0.005, got %f", p.OperationsPerK)
		}
	})

	t.Run("B2 pricing", func(t *testing.T) {
		p := DefaultPricing[models.RepositoryTypeB2]
		if p.StoragePerGBMonth != 0.006 {
			t.Errorf("expected 0.006, got %f", p.StoragePerGBMonth)
		}
		if p.EgressPerGB != 0.01 {
			t.Errorf("expected 0.01, got %f", p.EgressPerGB)
		}
	})

	t.Run("free backends have zero pricing", func(t *testing.T) {
		freeTypes := []models.RepositoryType{
			models.RepositoryTypeLocal,
			models.RepositoryTypeSFTP,
			models.RepositoryTypeRest,
			models.RepositoryTypeDropbox,
		}
		for _, rt := range freeTypes {
			p := DefaultPricing[rt]
			if p.StoragePerGBMonth != 0 {
				t.Errorf("%s: expected 0 storage cost, got %f", rt, p.StoragePerGBMonth)
			}
			if p.EgressPerGB != 0 {
				t.Errorf("%s: expected 0 egress cost, got %f", rt, p.EgressPerGB)
			}
			if p.OperationsPerK != 0 {
				t.Errorf("%s: expected 0 operations cost, got %f", rt, p.OperationsPerK)
			}
		}
	})
}

func TestWasabiPricingValues(t *testing.T) {
	if WasabiPricing.StoragePerGBMonth != 0.0069 {
		t.Errorf("expected 0.0069, got %f", WasabiPricing.StoragePerGBMonth)
	}
	if WasabiPricing.EgressPerGB != 0.0 {
		t.Errorf("expected 0 egress, got %f", WasabiPricing.EgressPerGB)
	}
	if WasabiPricing.OperationsPerK != 0.0 {
		t.Errorf("expected 0 operations, got %f", WasabiPricing.OperationsPerK)
	}
	if WasabiPricing.ProviderName != "Wasabi" {
		t.Errorf("expected 'Wasabi', got %q", WasabiPricing.ProviderName)
	}
}

func TestEstimateRepositoryCostDifferentBackends(t *testing.T) {
	calc := NewCalculator()

	types := []struct {
		repoType models.RepositoryType
		hasCloudCost bool
	}{
		{models.RepositoryTypeS3, true},
		{models.RepositoryTypeB2, true},
		{models.RepositoryTypeLocal, false},
		{models.RepositoryTypeSFTP, false},
		{models.RepositoryTypeRest, false},
		{models.RepositoryTypeDropbox, false},
	}

	sizeBytes := int64(10 * 1024 * 1024 * 1024) // 10 GB

	for _, tc := range types {
		t.Run(string(tc.repoType), func(t *testing.T) {
			est := calc.EstimateRepositoryCost("id", "name", tc.repoType, sizeBytes)

			if tc.hasCloudCost {
				if est.MonthlyCost <= 0 {
					t.Errorf("expected positive cost for %s, got %f", tc.repoType, est.MonthlyCost)
				}
			} else {
				if est.MonthlyCost != 0 {
					t.Errorf("expected zero cost for %s, got %f", tc.repoType, est.MonthlyCost)
				}
			}

			if est.RepositoryType != string(tc.repoType) {
				t.Errorf("expected type %q, got %q", tc.repoType, est.RepositoryType)
			}
		})
	}
}
