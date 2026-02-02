package portal

import (
	"net/http"

	"github.com/MacJediWizard/keldris/internal/portal/portalctx"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// AuthMiddleware validates the portal session and injects the customer into context.
func AuthMiddleware(store portalctx.Store, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie
		token, err := c.Cookie(portalctx.SessionCookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// Validate session
		tokenHash := portalctx.HashSessionToken(token)
		session, err := store.GetSessionByTokenHash(c.Request.Context(), tokenHash)
		if err != nil || session == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			return
		}

		// Check if session is expired
		if session.IsExpired() {
			// Clean up expired session
			_ = store.DeleteSession(c.Request.Context(), session.ID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			return
		}

		// Get customer
		customer, err := store.GetCustomerByID(c.Request.Context(), session.CustomerID)
		if err != nil || customer == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "customer not found"})
			return
		}

		// Check if customer is active
		if !customer.IsActive() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account is not active"})
			return
		}

		// Check if customer is locked
		if customer.IsLocked() {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account is locked"})
			return
		}

		// Set customer in context
		sessionUser := &portalctx.SessionUser{
			ID:      customer.ID,
			Email:   customer.Email,
			Name:    customer.Name,
			Company: customer.Company,
		}
		c.Set(string(portalctx.CustomerContextKey), sessionUser)

		c.Next()
	}
}
