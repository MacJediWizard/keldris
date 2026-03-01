package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	libredis "github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

// RateLimitConfig holds configuration for rate limiting.
type RateLimitConfig struct {
	// Requests is the number of requests allowed per period.
	Requests int64
	// Period is the duration for the rate limit window.
	Period time.Duration
}

// RateLimitStats holds statistics for rate limiting.
type RateLimitStats struct {
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
	Reset     int64 `json:"reset"` // Unix timestamp
}

// EndpointRateLimitConfig allows configuring different limits for different endpoints.
type EndpointRateLimitConfig struct {
	// Pattern is a path pattern (e.g., "/api/v1/backups")
	Pattern string
	// Requests is the number of requests allowed per period.
	Requests int64
	// Period is a duration string (e.g., "1m", "1h").
	Period string
}

// RateLimitManager manages rate limiters for different endpoints.
type RateLimitManager struct {
	defaultLimiter *limiter.Limiter
	defaultRate    limiter.Rate
	endpointLimiter map[string]*limiter.Limiter
	endpointRates   map[string]limiter.Rate
	store          limiter.Store
	mu             sync.RWMutex
	stats          map[string]*RateLimitClientStats
	statsMu        sync.RWMutex
}

// RateLimitClientStats holds per-client statistics.
type RateLimitClientStats struct {
	ClientIP      string    `json:"client_ip"`
	TotalRequests int64     `json:"total_requests"`
	RejectedCount int64     `json:"rejected_count"`
	LastRequest   time.Time `json:"last_request"`
}

// RateLimitDashboardStats holds statistics for the admin dashboard.
type RateLimitDashboardStats struct {
	DefaultLimit    int64                    `json:"default_limit"`
	DefaultPeriod   string                   `json:"default_period"`
	EndpointConfigs []EndpointRateLimitInfo  `json:"endpoint_configs"`
	ClientStats     []*RateLimitClientStats  `json:"client_stats"`
	TotalRequests   int64                    `json:"total_requests"`
	TotalRejected   int64                    `json:"total_rejected"`
}

// EndpointRateLimitInfo holds rate limit info for an endpoint.
type EndpointRateLimitInfo struct {
	Pattern  string `json:"pattern"`
	Limit    int64  `json:"limit"`
	Period   string `json:"period"`
}

var globalRateLimitManager *RateLimitManager

// NewRateLimiter creates a Gin middleware for rate limiting.
// requests is the number of requests allowed per period.
// period is a duration string (e.g., "1m", "1h", "24h").
// redisURL, when non-empty, enables a Redis-backed store for distributed rate limiting.
func NewRateLimiter(requests int64, period string, redisURL string) (gin.HandlerFunc, error) {
	duration, err := time.ParseDuration(period)
	if err != nil {
		return nil, fmt.Errorf("invalid rate limit period %q: %w", period, err)
	}

	rate := limiter.Rate{
		Period: duration,
		Limit:  requests,
	}

	var store limiter.Store
	if redisURL != "" {
		opts, err := libredis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("invalid redis URL: %w", err)
		}
		client := libredis.NewClient(opts)
		store, err = sredis.NewStore(client)
		if err != nil {
			return nil, fmt.Errorf("failed to create redis rate limit store: %w", err)
		}
	} else {
		store = memory.NewStore()
	}

	instance := limiter.New(store, rate)

	globalRateLimitManager = &RateLimitManager{
		defaultLimiter:  instance,
		defaultRate:     rate,
		endpointLimiter: make(map[string]*limiter.Limiter),
		endpointRates:   make(map[string]limiter.Rate),
		store:           store,
		stats:           make(map[string]*RateLimitClientStats),
	}

	return rateLimitMiddleware(globalRateLimitManager), nil
}

// NewRateLimiterWithEndpoints creates a rate limiter with per-endpoint configuration.
func NewRateLimiterWithEndpoints(defaultRequests int64, defaultPeriod string, endpointConfigs []EndpointRateLimitConfig) (gin.HandlerFunc, error) {
	duration, err := time.ParseDuration(defaultPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid default rate limit period %q: %w", defaultPeriod, err)
	}

	defaultRate := limiter.Rate{
		Period: duration,
		Limit:  defaultRequests,
	}

	store := memory.NewStore()
	defaultInstance := limiter.New(store, defaultRate)

	manager := &RateLimitManager{
		defaultLimiter:  defaultInstance,
		defaultRate:     defaultRate,
		endpointLimiter: make(map[string]*limiter.Limiter),
		endpointRates:   make(map[string]limiter.Rate),
		store:           store,
		stats:           make(map[string]*RateLimitClientStats),
	}

	for _, cfg := range endpointConfigs {
		epDuration, err := time.ParseDuration(cfg.Period)
		if err != nil {
			return nil, fmt.Errorf("invalid rate limit period %q for endpoint %q: %w", cfg.Period, cfg.Pattern, err)
		}
		epRate := limiter.Rate{
			Period: epDuration,
			Limit:  cfg.Requests,
		}
		manager.endpointLimiter[cfg.Pattern] = limiter.New(store, epRate)
		manager.endpointRates[cfg.Pattern] = epRate
	}

	globalRateLimitManager = manager
	return rateLimitMiddleware(manager), nil
}

// GetRateLimitManager returns the global rate limit manager for dashboard access.
func GetRateLimitManager() *RateLimitManager {
	return globalRateLimitManager
}

// GetDashboardStats returns statistics for the admin dashboard.
func (m *RateLimitManager) GetDashboardStats() RateLimitDashboardStats {
	m.statsMu.RLock()
	defer m.statsMu.RUnlock()

	stats := RateLimitDashboardStats{
		DefaultLimit:    m.defaultRate.Limit,
		DefaultPeriod:   m.defaultRate.Period.String(),
		EndpointConfigs: make([]EndpointRateLimitInfo, 0),
		ClientStats:     make([]*RateLimitClientStats, 0, len(m.stats)),
	}

	m.mu.RLock()
	for pattern, rate := range m.endpointRates {
		stats.EndpointConfigs = append(stats.EndpointConfigs, EndpointRateLimitInfo{
			Pattern: pattern,
			Limit:   rate.Limit,
			Period:  rate.Period.String(),
		})
	}
	m.mu.RUnlock()

	for _, clientStats := range m.stats {
		stats.ClientStats = append(stats.ClientStats, clientStats)
		stats.TotalRequests += clientStats.TotalRequests
		stats.TotalRejected += clientStats.RejectedCount
	}

	return stats
}

// rateLimitMiddleware creates the actual middleware handler.
func rateLimitMiddleware(manager *RateLimitManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for static assets and health checks.
		// These are cache-busted files or infrastructure probes that
		// should not consume the user's API rate limit budget.
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/assets/") ||
			strings.HasPrefix(path, "/api/v1/setup") ||
			path == "/health" ||
			path == "/favicon.ico" {
			c.Next()
			return
		}

		// Get client IP
		key := c.ClientIP()

		// Select the appropriate limiter
		lim := manager.defaultLimiter
		rate := manager.defaultRate

		manager.mu.RLock()
		for pattern, epLimiter := range manager.endpointLimiter {
			if matchPath(c.Request.URL.Path, pattern) {
				lim = epLimiter
				rate = manager.endpointRates[pattern]
				break
			}
		}
		manager.mu.RUnlock()

		// Get rate limit context
		ctx, err := lim.Get(c.Request.Context(), key)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "rate limiter error"})
			return
		}

		// Update statistics
		manager.statsMu.Lock()
		if _, exists := manager.stats[key]; !exists {
			manager.stats[key] = &RateLimitClientStats{
				ClientIP: key,
			}
		}
		manager.stats[key].TotalRequests++
		manager.stats[key].LastRequest = time.Now()
		manager.statsMu.Unlock()

		// Set rate limit headers
		// ctx.Reset is a Unix timestamp (int64)
		c.Header("X-RateLimit-Limit", strconv.FormatInt(rate.Limit, 10))
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(ctx.Remaining, 10))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(ctx.Reset, 10))

		// Check if rate limit exceeded
		if ctx.Reached {
			// Update rejected count
			manager.statsMu.Lock()
			manager.stats[key].RejectedCount++
			manager.statsMu.Unlock()

			// Calculate Retry-After in seconds
			resetTime := time.Unix(ctx.Reset, 0)
			retryAfter := time.Until(resetTime).Seconds()
			if retryAfter < 1 {
				retryAfter = 1
			}
			c.Header("Retry-After", strconv.FormatInt(int64(retryAfter), 10))

			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": int64(retryAfter),
				"rate_limit": gin.H{
					"limit":     rate.Limit,
					"remaining": ctx.Remaining,
					"reset":     ctx.Reset,
				},
			})
			return
		}

		// Store rate limit info in context for optional inclusion in responses
		c.Set("rate_limit", &RateLimitStats{
			Limit:     rate.Limit,
			Remaining: ctx.Remaining,
			Reset:     ctx.Reset,
		})

		c.Next()
	}
}

// matchPath checks if a request path matches a pattern.
// Supports simple prefix matching and exact matching.
func matchPath(path, pattern string) bool {
	// Exact match
	if path == pattern {
		return true
	}
	// Prefix match (pattern ends with /*)
	if len(pattern) > 2 && pattern[len(pattern)-2:] == "/*" {
		prefix := pattern[:len(pattern)-2]
		return len(path) >= len(prefix) && path[:len(prefix)] == prefix
	}
	// Simple prefix match
	return len(path) >= len(pattern) && path[:len(pattern)] == pattern
}

// GetRateLimitStats retrieves rate limit stats from the context if available.
func GetRateLimitStats(c *gin.Context) *RateLimitStats {
	if stats, exists := c.Get("rate_limit"); exists {
		if rlStats, ok := stats.(*RateLimitStats); ok {
			return rlStats
		}
	}
	return nil
}
