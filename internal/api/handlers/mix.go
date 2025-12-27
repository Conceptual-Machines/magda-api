package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

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

// MixAnalyzeStream handles mix analysis requests with TRUE SSE streaming
// POST /api/v1/mix/analyze/stream
func (h *MixHandler) MixAnalyzeStream(c *gin.Context) {
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Mix Stream: PANIC recovered: %v", r)
			log.Printf("   Stack trace:\n%s", string(debug.Stack()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      fmt.Sprintf("Internal server error: %v", r),
				"request_id": c.GetString("request_id"),
			})
		}
	}()

	var req mixagent.AnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Mix Stream: JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("🎛️ MixAnalyzeStream: Received request (TRUE STREAMING), mode=%s", req.Mode)
	if req.Context != nil {
		log.Printf("   Track type: %s", req.Context.TrackType)
		log.Printf("   Track name: %s", req.Context.TrackName)
	}

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	// Utility to send SSE events
	sendEvent := func(event map[string]any) error {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Printf("❌ Mix Stream: Failed to marshal event: %v", err)
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON); err != nil {
			log.Printf("❌ Mix Stream: Failed to write SSE event: %v", err)
			return err
		}
		c.Writer.Flush()
		return nil
	}

	// Send initial start event
	_ = sendEvent(map[string]any{
		"type":    "start",
		"message": "Analyzing mix...",
	})

	ctx := c.Request.Context()
	chunkCount := 0

	// TRUE STREAMING: callback is called for each chunk as it arrives from the LLM
	streamCallback := func(chunk string) error {
		chunkCount++
		// Send chunk event to client in real-time
		return sendEvent(map[string]any{
			"type":  "chunk",
			"chunk": chunk,
		})
	}

	// Call mix analysis agent with TRUE streaming
	log.Printf("🚀 Mix Stream: Starting true streaming analysis...")
	result, err := h.agent.AnalyzeStream(ctx, &req, streamCallback)
	if err != nil {
		log.Printf("❌ Mix Stream: Analysis error: %v", err)
		_ = sendEvent(map[string]any{
			"type":    "error",
			"message": fmt.Sprintf("Mix analysis failed: %v", err),
		})
		return
	}

	log.Printf("✅ Mix Stream: Completed with %d chunks streamed", chunkCount)

	// Send complete event with final result
	_ = sendEvent(map[string]any{
		"type":    "complete",
		"message": "Analysis complete",
		"result":  result,
	})
}
