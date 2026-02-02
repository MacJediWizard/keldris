package portal

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ContextKey is a type for context keys.
type ContextKey string

const (
	// CustomerContextKey is the context key for the authenticated customer.
	CustomerContextKey ContextKey = "portal_customer"
)

// AuthMiddleware validates the portal session and injects the customer into context.
func AuthMiddleware(store Store, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie
		token, err := c.Cookie(SessionCookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		// Validate session
		tokenHash := HashSessionToken(token)
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
		sessionUser := &SessionUser{
			ID:      customer.ID,
			Email:   customer.Email,
			Name:    customer.Name,
			Company: customer.Company,
		}
		c.Set(string(CustomerContextKey), sessionUser)

		c.Next()
	}
}

// GetCustomer returns the authenticated customer from context.
func GetCustomer(c *gin.Context) *SessionUser {
	if val, exists := c.Get(string(CustomerContextKey)); exists {
		if customer, ok := val.(*SessionUser); ok {
			return customer
		}
	}
	return nil
}

// RequireCustomer returns the authenticated customer or aborts with 401.
func RequireCustomer(c *gin.Context) *SessionUser {
	customer := GetCustomer(c)
	if customer == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return nil
	}
	return customer
}
