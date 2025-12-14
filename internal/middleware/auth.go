package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const (
	bearerPrefix = "Bearer"
)

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// JWTAuth middleware validates JWT tokens and attaches user to context
func JWTAuth(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try to get token from Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// Extract token from "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == bearerPrefix {
				tokenString = parts[1]
			}
		}

		// If no header, try cookie (for web users)
		if tokenString == "" {
			tokenString, _ = c.Cookie("access_token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}

		// Parse and validate token
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// Verify signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Load user from database
		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		// Check if user is active
		if !user.IsActive {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is disabled"})
			c.Abort()
			return
		}

		// Check if email is verified (skip for admin and beta users)
		if !user.EmailVerified && user.Role != "admin" && user.Role != "beta" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":          "Email not verified",
				"message":        "Please verify your email to use the API",
				"email_verified": false,
			})
			c.Abort()
			return
		}

		// Attach user to context
		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// OptionalJWTAuth is like JWTAuth but doesn't abort if token is missing (useful for optional auth)
func OptionalJWTAuth(db *gorm.DB, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Try Authorization header first
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenString = parts[1]
			}
		}

		// Try cookie if no header
		if tokenString == "" {
			tokenString, _ = c.Cookie("access_token")
		}

		// Try refresh_token cookie if still no token
		if tokenString == "" {
			tokenString, _ = c.Cookie("refresh_token")
		}

		if tokenString == "" {
			c.Next()
			return
		}
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.Next()
			return
		}

		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.Next()
			return
		}

		if !user.IsActive {
			c.Next()
			return
		}

		c.Set("user", user)
		c.Set("user_id", user.ID)

		c.Next()
	}
}

// GetCurrentUser retrieves the user from context
func GetCurrentUser(c *gin.Context) (*models.User, bool) {
	userVal, exists := c.Get("user")
	if !exists {
		return nil, false
	}
	user, ok := userVal.(models.User)
	return &user, ok
}

// GetCurrentUserID retrieves the user ID from context
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := userIDVal.(uint)
	return userID, ok
}
