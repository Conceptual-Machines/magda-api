package handlers

import (
	"log"
	"net/http"

	magdaconfig "github.com/Conceptual-Machines/magda-api/internal/agents/core/config"
	mixagent "github.com/Conceptual-Machines/magda-api/internal/agents/shared/mix"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/gin-gonic/gin"
)

type MixHandler struct {
	agent *mixagent.MixAnalysisAgent
	cfg   *config.Config
}

func NewMixHandler(cfg *config.Config) *MixHandler {
	// Convert magda-api config to magda-agents config
	magdaCfg := &magdaconfig.Config{
		OpenAIAPIKey: cfg.OpenAIAPIKey,
		MCPServerURL: cfg.MCPServerURL,
	}

	return &MixHandler{
		agent: mixagent.NewMixAnalysisAgent(magdaCfg),
		cfg:   cfg,
	}
}

// MixAnalyze handles mix analysis requests
// POST /api/v1/mix/analyze
func (h *MixHandler) MixAnalyze(c *gin.Context) {
	var req mixagent.AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ MixAnalyze: JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("🎛️ MixAnalyze: Received request, mode=%s", req.Mode)
	if req.Context != nil {
		log.Printf("   Track type: %s", req.Context.TrackType)
		log.Printf("   Track name: %s", req.Context.TrackName)
	}
	if req.UserRequest != "" {
		log.Printf("   User request: %s", req.UserRequest)
	}

	// Call the mix analysis agent
	result, err := h.agent.Analyze(c.Request.Context(), &req)
	if err != nil {
		log.Printf("❌ MixAnalyze: Analysis error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("✅ MixAnalyze: Analysis complete, %d recommendations", len(result.Recommendations))

	// Return the analysis result
	c.JSON(http.StatusOK, result)
}
