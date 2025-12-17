package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db *gorm.DB
}

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) HealthCheck(c *gin.Context) {
	mcpURL := os.Getenv("MCP_SERVER_URL")
	mcpStatus := "disabled"

	if mcpURL != "" && strings.TrimSpace(mcpURL) != "" {
		mcpStatus = "enabled"
	}

	// Check database connectivity
	dbStatus := "healthy"
	sqlDB, err := h.db.DB()
	if err != nil {
		dbStatus = "error: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"database": gin.H{
				"status": dbStatus,
			},
			"mcp_server": gin.H{
				"status": mcpStatus,
				"url":    mcpURL,
			},
		})
		return
	}

	if err := sqlDB.Ping(); err != nil {
		dbStatus = "error: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"database": gin.H{
				"status": dbStatus,
			},
			"mcp_server": gin.H{
				"status": mcpStatus,
				"url":    mcpURL,
			},
		})
		return
	}

	// Verify API can process requests by checking if we can query users table
	var userCount int64
	if err := h.db.Model(&models.User{}).Count(&userCount).Error; err != nil {
		dbStatus = "error: cannot query database - " + err.Error()
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "unhealthy",
			"database": gin.H{
				"status": dbStatus,
			},
			"mcp_server": gin.H{
				"status": mcpStatus,
				"url":    mcpURL,
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"database": gin.H{
			"status": dbStatus,
		},
		"mcp_server": gin.H{
			"status": mcpStatus,
			"url":    mcpURL,
		},
	})
}

// Legacy function for backward compatibility
func HealthCheck(c *gin.Context) {
	// This will fail if db is not available, but maintains backward compatibility
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"mcp_server": gin.H{
			"status": "disabled",
			"url":    "",
		},
		"note": "Health check without database verification - use NewHealthHandler for full check",
	})
}
