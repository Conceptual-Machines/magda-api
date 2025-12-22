package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	magdaconfig "github.com/Conceptual-Machines/magda-api/internal/agents/config"
	"github.com/Conceptual-Machines/magda-api/internal/agents/drummer"
	"github.com/Conceptual-Machines/magda-api/internal/api/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/gin-gonic/gin"
)

const (
	defaultDrummerModel = "gpt-5.1"
	drummerTimeoutSecs  = 120
)

type DrummerHandler struct {
	agent *drummer.DrummerAgent
	cfg   *config.Config
}

func NewDrummerHandler(cfg *config.Config) *DrummerHandler {
	// Convert config to magda-agents config
	magdaCfg := &magdaconfig.Config{
		OpenAIAPIKey: cfg.OpenAIAPIKey,
	}
	agent := drummer.NewDrummerAgent(magdaCfg)

	return &DrummerHandler{
		agent: agent,
		cfg:   cfg,
	}
}

type DrummerRequest struct {
	Model      string           `json:"model"`
	InputArray []map[string]any `json:"input_array" binding:"required"`
}

type DrummerResponse struct {
	DSL     string           `json:"dsl"`
	Actions []map[string]any `json:"actions"`
	Usage   any              `json:"usage,omitempty"`
}

func (h *DrummerHandler) Generate(c *gin.Context) {
	var req DrummerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from gateway headers (for logging - auth handled by gateway)
	userID, _ := middleware.GetUserIDFromGateway(c)
	log.Printf("🥁 Drummer request from user %s", userID)

	// Use requested model or default
	model := req.Model
	if model == "" {
		model = defaultDrummerModel
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Request.Context(), drummerTimeoutSecs*time.Second)
	defer cancel()

	// Call the drummer agent
	result, err := h.agent.Generate(ctx, model, req.InputArray)
	if err != nil {
		log.Printf("❌ Drummer generation failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return the result
	response := DrummerResponse{
		DSL:     result.DSL,
		Actions: result.Actions,
		Usage:   result.Usage,
	}

	c.JSON(http.StatusOK, response)
}
