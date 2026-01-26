package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/MacJediWizard/keldris/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// IPFilterStore defines the interface for IP allowlist persistence operations.
type IPFilterStore interface {
	GetOrCreateIPAllowlistSettings(ctx context.Context, orgID uuid.UUID) (*models.IPAllowlistSettings, error)
	ListEnabledIPAllowlistsByOrg(ctx context.Context, orgID uuid.UUID, allowlistType models.IPAllowlistType) ([]*models.IPAllowlist, error)
	CreateIPBlockedAttempt(ctx context.Context, b *models.IPBlockedAttempt) error
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

// IPFilterCache caches IP allowlist data for performance.
type IPFilterCache struct {
	mu        sync.RWMutex
	settings  map[uuid.UUID]*cachedSettings
	allowlist map[uuid.UUID]map[models.IPAllowlistType]*cachedAllowlist
	ttl       time.Duration
}

type cachedSettings struct {
	settings  *models.IPAllowlistSettings
	expiresAt time.Time
}

type cachedAllowlist struct {
	entries   []*models.IPAllowlist
	expiresAt time.Time
}

// NewIPFilterCache creates a new IP filter cache.
func NewIPFilterCache(ttl time.Duration) *IPFilterCache {
	return &IPFilterCache{
		settings:  make(map[uuid.UUID]*cachedSettings),
		allowlist: make(map[uuid.UUID]map[models.IPAllowlistType]*cachedAllowlist),
		ttl:       ttl,
	}
}

// GetSettings returns cached settings or fetches from store.
func (c *IPFilterCache) GetSettings(ctx context.Context, store IPFilterStore, orgID uuid.UUID) (*models.IPAllowlistSettings, error) {
	c.mu.RLock()
	if cached, ok := c.settings[orgID]; ok && time.Now().Before(cached.expiresAt) {
		c.mu.RUnlock()
		return cached.settings, nil
	}
	c.mu.RUnlock()

	settings, err := store.GetOrCreateIPAllowlistSettings(ctx, orgID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.settings[orgID] = &cachedSettings{
		settings:  settings,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return settings, nil
}

// GetAllowlist returns cached allowlist entries or fetches from store.
func (c *IPFilterCache) GetAllowlist(ctx context.Context, store IPFilterStore, orgID uuid.UUID, allowlistType models.IPAllowlistType) ([]*models.IPAllowlist, error) {
	c.mu.RLock()
	if orgCache, ok := c.allowlist[orgID]; ok {
		if cached, ok := orgCache[allowlistType]; ok && time.Now().Before(cached.expiresAt) {
			c.mu.RUnlock()
			return cached.entries, nil
		}
	}
	c.mu.RUnlock()

	entries, err := store.ListEnabledIPAllowlistsByOrg(ctx, orgID, allowlistType)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	if c.allowlist[orgID] == nil {
		c.allowlist[orgID] = make(map[models.IPAllowlistType]*cachedAllowlist)
	}
	c.allowlist[orgID][allowlistType] = &cachedAllowlist{
		entries:   entries,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return entries, nil
}

// Invalidate clears the cache for an organization.
func (c *IPFilterCache) Invalidate(orgID uuid.UUID) {
	c.mu.Lock()
	delete(c.settings, orgID)
	delete(c.allowlist, orgID)
	c.mu.Unlock()
}

// IPFilter provides IP filtering functionality.
type IPFilter struct {
	store  IPFilterStore
	cache  *IPFilterCache
	logger zerolog.Logger
}

// NewIPFilter creates a new IP filter.
func NewIPFilter(store IPFilterStore, logger zerolog.Logger) *IPFilter {
	return &IPFilter{
		store:  store,
		cache:  NewIPFilterCache(30 * time.Second),
		logger: logger.With().Str("component", "ip_filter").Logger(),
	}
}

// InvalidateCache clears the cache for an organization.
func (f *IPFilter) InvalidateCache(orgID uuid.UUID) {
	f.cache.Invalidate(orgID)
}

// CheckIP checks if an IP address is allowed for the given organization and access type.
func (f *IPFilter) CheckIP(ctx context.Context, orgID uuid.UUID, ipAddress string, accessType models.IPAllowlistType, isAdmin bool) (allowed bool, reason string) {
	// Get settings
	settings, err := f.cache.GetSettings(ctx, f.store, orgID)
	if err != nil {
		f.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to get IP allowlist settings")
		// Fail open - allow if we can't get settings
		return true, ""
	}

	// If IP filtering is not enabled, allow all
	if !settings.Enabled {
		return true, ""
	}

	// Check if the access type should be enforced
	if accessType == models.IPAllowlistTypeUI && !settings.EnforceForUI {
		return true, ""
	}
	if accessType == models.IPAllowlistTypeAgent && !settings.EnforceForAgent {
		return true, ""
	}

	// Check admin bypass
	if isAdmin && settings.AllowAdminBypass {
		f.logger.Debug().
			Str("org_id", orgID.String()).
			Str("ip", ipAddress).
			Msg("admin bypass allowed")
		return true, ""
	}

	// Get allowlist entries
	entries, err := f.cache.GetAllowlist(ctx, f.store, orgID, accessType)
	if err != nil {
		f.logger.Error().Err(err).Str("org_id", orgID.String()).Msg("failed to get IP allowlist entries")
		// Fail open - allow if we can't get entries
		return true, ""
	}

	// If no allowlist entries exist, deny all (except for admin bypass which was handled above)
	if len(entries) == 0 {
		return false, "no IP allowlist entries configured"
	}

	// Check if IP is in any allowlist entry
	for _, entry := range entries {
		if entry.ContainsIP(ipAddress) {
			f.logger.Debug().
				Str("org_id", orgID.String()).
				Str("ip", ipAddress).
				Str("matched_cidr", entry.CIDR).
				Msg("IP allowed by allowlist")
			return true, ""
		}
	}

	return false, "IP address not in allowlist"
}

// LogBlockedAttempt logs a blocked access attempt.
func (f *IPFilter) LogBlockedAttempt(ctx context.Context, orgID uuid.UUID, ipAddress, requestType, path, reason string, userID, agentID *uuid.UUID) {
	attempt := models.NewIPBlockedAttempt(orgID, ipAddress, requestType, path, reason)
	if userID != nil {
		attempt.WithUser(*userID)
	}
	if agentID != nil {
		attempt.WithAgent(*agentID)
	}

	// Log asynchronously to not block the request
	go func(ctx context.Context, b *models.IPBlockedAttempt) {
		if err := f.store.CreateIPBlockedAttempt(context.Background(), b); err != nil {
			f.logger.Error().Err(err).
				Str("ip", b.IPAddress).
				Str("org_id", b.OrgID.String()).
				Msg("failed to log blocked attempt")
		}
	}(ctx, attempt)

	f.logger.Warn().
		Str("org_id", orgID.String()).
		Str("ip", ipAddress).
		Str("request_type", requestType).
		Str("path", path).
		Str("reason", reason).
		Msg("IP access blocked")
}

// IPFilterMiddleware returns a Gin middleware that checks IP allowlists for UI access.
func IPFilterMiddleware(filter *IPFilter, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "ip_filter_middleware").Logger()

	return func(c *gin.Context) {
		// Get user from context (set by AuthMiddleware)
		user := GetUser(c)
		if user == nil {
			// No authenticated user, skip IP filtering
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()
		orgID := user.CurrentOrgID

		if orgID == uuid.Nil {
			// No organization selected, skip IP filtering
			c.Next()
			return
		}

		// Check if user is admin
		isAdmin := user.CurrentOrgRole == string(models.OrgRoleAdmin) || user.CurrentOrgRole == string(models.OrgRoleOwner)

		// Check IP
		allowed, reason := filter.CheckIP(c.Request.Context(), orgID, clientIP, models.IPAllowlistTypeUI, isAdmin)
		if !allowed {
			log.Warn().
				Str("user_id", user.ID.String()).
				Str("org_id", orgID.String()).
				Str("ip", clientIP).
				Str("path", c.Request.URL.Path).
				Str("reason", reason).
				Msg("IP access denied for UI request")

			// Log blocked attempt
			filter.LogBlockedAttempt(c.Request.Context(), orgID, clientIP, "ui", c.Request.URL.Path, reason, &user.ID, nil)

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "access denied",
				"message": "Your IP address is not allowed to access this resource",
				"code":    "IP_NOT_ALLOWED",
			})
			return
		}

		c.Next()
	}
}

// IPFilterAgentMiddleware returns a Gin middleware that checks IP allowlists for agent access.
func IPFilterAgentMiddleware(filter *IPFilter, logger zerolog.Logger) gin.HandlerFunc {
	log := logger.With().Str("component", "ip_filter_agent_middleware").Logger()

	return func(c *gin.Context) {
		// Get agent from context (set by APIKeyMiddleware)
		agent := GetAgent(c)
		if agent == nil {
			// No authenticated agent, skip IP filtering
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()
		orgID := agent.OrgID

		// Agents don't have admin bypass
		allowed, reason := filter.CheckIP(c.Request.Context(), orgID, clientIP, models.IPAllowlistTypeAgent, false)
		if !allowed {
			log.Warn().
				Str("agent_id", agent.ID.String()).
				Str("org_id", orgID.String()).
				Str("ip", clientIP).
				Str("path", c.Request.URL.Path).
				Str("reason", reason).
				Msg("IP access denied for agent request")

			// Log blocked attempt
			filter.LogBlockedAttempt(c.Request.Context(), orgID, clientIP, "agent", c.Request.URL.Path, reason, nil, &agent.ID)

			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "access denied",
				"message": "Agent IP address is not allowed to access this resource",
				"code":    "IP_NOT_ALLOWED",
			})
			return
		}

		c.Next()
	}
}
