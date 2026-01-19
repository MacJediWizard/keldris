package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

// NewRateLimiter creates a Gin middleware for rate limiting.
// requests is the number of requests allowed per period.
// period is a duration string (e.g., "1m", "1h", "24h").
func NewRateLimiter(requests int64, period string) (gin.HandlerFunc, error) {
	duration, err := time.ParseDuration(period)
	if err != nil {
		return nil, fmt.Errorf("invalid rate limit period %q: %w", period, err)
	}

	rate := limiter.Rate{
		Period: duration,
		Limit:  requests,
	}

	store := memory.NewStore()
	instance := limiter.New(store, rate)

	middleware := mgin.NewMiddleware(instance)
	return middleware, nil
}
