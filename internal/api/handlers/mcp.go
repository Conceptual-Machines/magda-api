package handlers

import (
	"net/http"
	"net/url"
	"os"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

func MCPStatus(c *gin.Context) {
	mcpURL := os.Getenv("MCP_SERVER_URL")

	response := gin.H{
		"enabled": false,
		"url":     "",
		"label":   "",
		"status":  "disabled",
	}

	if mcpURL != "" && strings.TrimSpace(mcpURL) != "" {
		// Generate the same label logic as in the service
		mcpLabel := "mcp-server"
		if parsed, err := url.Parse(mcpURL); err == nil {
			host := strings.TrimSpace(parsed.Host)
			if host != "" {
				// Convert host to valid MCP label format (letters, digits, dashes, underscores only)
				mcpLabel = strings.ReplaceAll(host, ".", "-")
				mcpLabel = strings.ReplaceAll(mcpLabel, ":", "_")
				// Ensure it starts with a letter
				if len(mcpLabel) > 0 && !unicode.IsLetter(rune(mcpLabel[0])) {
					mcpLabel = "mcp-" + mcpLabel
				}
			}
		}

		response = gin.H{
			"enabled": true,
			"url":     mcpURL,
			"label":   mcpLabel,
			"status":  "enabled",
		}
	}

	c.JSON(http.StatusOK, response)
}
