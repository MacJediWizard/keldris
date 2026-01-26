package models

import (
	"time"

	"github.com/google/uuid"
)

// RateLimitConfig represents rate limiting settings for an endpoint.
type RateLimitConfig struct {
	ID                uuid.UUID  `json:"id"`
	OrgID             uuid.UUID  `json:"org_id"`
	Endpoint          string     `json:"endpoint"`
	RequestsPerPeriod int        `json:"requests_per_period"`
	PeriodSeconds     int        `json:"period_seconds"`
	Enabled           bool       `json:"enabled"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// NewRateLimitConfig creates a new RateLimitConfig with defaults.
func NewRateLimitConfig(orgID uuid.UUID, endpoint string) *RateLimitConfig {
	now := time.Now()
	return &RateLimitConfig{
		ID:                uuid.New(),
		OrgID:             orgID,
		Endpoint:          endpoint,
		RequestsPerPeriod: 100,
		PeriodSeconds:     60,
		Enabled:           true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
}

// BlockedRequest represents a rate-limited request.
type BlockedRequest struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     *uuid.UUID `json:"org_id,omitempty"`
	IPAddress string     `json:"ip_address"`
	Endpoint  string     `json:"endpoint"`
	UserAgent string     `json:"user_agent,omitempty"`
	BlockedAt time.Time  `json:"blocked_at"`
	Reason    string     `json:"reason"`
}

// IPBan represents a temporary or permanent IP ban.
type IPBan struct {
	ID        uuid.UUID  `json:"id"`
	OrgID     *uuid.UUID `json:"org_id,omitempty"`
	IPAddress string     `json:"ip_address"`
	Reason    string     `json:"reason"`
	BanCount  int        `json:"ban_count"`
	BannedBy  *uuid.UUID `json:"banned_by,omitempty"`
	BannedAt  time.Time  `json:"banned_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// IsActive returns true if the ban is currently active.
func (b *IPBan) IsActive(now time.Time) bool {
	if b.ExpiresAt == nil {
		return true // Permanent ban
	}
	return now.Before(*b.ExpiresAt)
}

// RateLimitStats represents rate limiting statistics.
type RateLimitStats struct {
	BlockedToday     int              `json:"blocked_today"`
	TopBlockedIPs    []IPBlockCount   `json:"top_blocked_ips"`
	TopBlockedRoutes []RouteBlockCount `json:"top_blocked_endpoints"`
}

// IPBlockCount represents block count for an IP address.
type IPBlockCount struct {
	IPAddress string `json:"ip_address"`
	Count     int    `json:"count"`
}

// RouteBlockCount represents block count for an endpoint.
type RouteBlockCount struct {
	Endpoint string `json:"endpoint"`
	Count    int    `json:"count"`
}

// CreateRateLimitConfigRequest is the request body for creating a rate limit config.
type CreateRateLimitConfigRequest struct {
	Endpoint          string `json:"endpoint" binding:"required,min=1,max=255"`
	RequestsPerPeriod int    `json:"requests_per_period" binding:"required,min=1"`
	PeriodSeconds     int    `json:"period_seconds" binding:"required,min=1"`
	Enabled           *bool  `json:"enabled,omitempty"`
}

// UpdateRateLimitConfigRequest is the request body for updating a rate limit config.
type UpdateRateLimitConfigRequest struct {
	RequestsPerPeriod *int  `json:"requests_per_period,omitempty"`
	PeriodSeconds     *int  `json:"period_seconds,omitempty"`
	Enabled           *bool `json:"enabled,omitempty"`
}

// CreateIPBanRequest is the request body for creating an IP ban.
type CreateIPBanRequest struct {
	IPAddress      string `json:"ip_address" binding:"required"`
	Reason         string `json:"reason" binding:"required,min=1,max=500"`
	DurationMinutes *int   `json:"duration_minutes,omitempty"` // nil = permanent
}

// RateLimitConfigsResponse is the response for listing rate limit configs.
type RateLimitConfigsResponse struct {
	Configs []RateLimitConfig `json:"configs"`
}

// IPBansResponse is the response for listing IP bans.
type IPBansResponse struct {
	Bans []IPBan `json:"bans"`
}

// RateLimitStatsResponse is the response for rate limit statistics.
type RateLimitStatsResponse struct {
	Stats RateLimitStats `json:"stats"`
}
