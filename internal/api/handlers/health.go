package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// HealthCheck returns the health status of the API
func HealthCheck(c *gin.Context) {
	mcpURL := os.Getenv("MCP_SERVER_URL")
	mcpStatus := "disabled"

	if mcpURL != "" && strings.TrimSpace(mcpURL) != "" {
		mcpStatus = "enabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"mcp_server": gin.H{
			"status": mcpStatus,
			"url":    mcpURL,
		},
	})
}
