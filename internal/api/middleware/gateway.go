package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GatewayAuth trusts user info from gateway headers (X-User-ID, X-User-Email, X-User-Role)
// This is used when the Go API runs behind the Express gateway (magda-cloud)
// which handles JWT validation and billing checks.
//
// When AUTH_MODE=gateway, the API trusts these headers unconditionally.
// This should ONLY be used in the hosted environment with proper network isolation.
func GatewayAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for gateway headers
		userIDStr := c.GetHeader("X-User-ID")
		userEmail := c.GetHeader("X-User-Email")
		userRole := c.GetHeader("X-User-Role")

		if userIDStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Authentication required",
				"message": "Missing X-User-ID header from gateway",
			})
			c.Abort()
			return
		}

		// Parse user ID (could be numeric or string depending on gateway)
		var userID uint
		if id, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			userID = uint(id)
		}

		// Set context values
		c.Set("user_id", userID)
		c.Set("user_id_str", userIDStr) // Keep string version for compatibility
		c.Set("user_email", userEmail)
		c.Set("user_role", userRole)

		// Also set API key info if present
		if apiKeyID := c.GetHeader("X-API-Key-ID"); apiKeyID != "" {
			c.Set("api_key_id", apiKeyID)
			c.Set("api_key_scopes", c.GetHeader("X-API-Key-Scopes"))
		}

		c.Next()
	}
}

// OptionalGatewayAuth is like GatewayAuth but doesn't fail if headers are missing
// Useful for endpoints that support both authenticated and anonymous access
func OptionalGatewayAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-User-ID")

		if userIDStr != "" {
			// Parse user ID
			if id, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
				c.Set("user_id", uint(id))
			}
			c.Set("user_id_str", userIDStr)
			c.Set("user_email", c.GetHeader("X-User-Email"))
			c.Set("user_role", c.GetHeader("X-User-Role"))
		}

		c.Next()
	}
}

// GetUserIDFromGateway retrieves the user ID from gateway headers
// Returns the string ID and a boolean indicating if it was found
func GetUserIDFromGateway(c *gin.Context) (string, bool) {
	userIDStr, exists := c.Get("user_id_str")
	if !exists {
		return "", false
	}
	id, ok := userIDStr.(string)
	return id, ok
}

// GetUserEmailFromGateway retrieves the user email from gateway headers
func GetUserEmailFromGateway(c *gin.Context) (string, bool) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", false
	}
	e, ok := email.(string)
	return e, ok
}

// GetUserRoleFromGateway retrieves the user role from gateway headers
func GetUserRoleFromGateway(c *gin.Context) (string, bool) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", false
	}
	r, ok := role.(string)
	return r, ok
}
