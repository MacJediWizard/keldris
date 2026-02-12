package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	libredis "github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

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

	middleware := mgin.NewMiddleware(instance)
	return middleware, nil
}
