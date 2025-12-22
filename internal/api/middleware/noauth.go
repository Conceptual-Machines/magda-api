package middleware

import (
	"github.com/gin-gonic/gin"
)

// NoAuth is a pass-through middleware for when AUTH_MODE=none or gateway.
// It allows all requests without authentication.
func NoAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set a dummy user ID for logging purposes
		c.Set("user_id", uint(0))
		c.Set("user_id_str", "anonymous")
		c.Next()
	}
}
