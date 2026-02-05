package portalctx

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ContextKey is a type for context keys.
type ContextKey string

const (
	// CustomerContextKey is the context key for the authenticated customer.
	CustomerContextKey ContextKey = "portal_customer"
)

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
