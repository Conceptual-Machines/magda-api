package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"

	magdaorchestrator "github.com/Conceptual-Machines/magda-agents-go/agents/coordination"
	magdadaw "github.com/Conceptual-Machines/magda-agents-go/agents/daw"
	magdaplugin "github.com/Conceptual-Machines/magda-agents-go/agents/plugin"
	magdaconfig "github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-api/internal/api/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/observability"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	// maxRequestPreviewLength is the maximum length for request body preview in logs
	maxRequestPreviewLength = 500
)

type MagdaHandler struct {
	orchestrator  *magdaorchestrator.Orchestrator
	pluginService *magdaplugin.PluginAgent
	db            *gorm.DB
	cfg           *config.Config
}

// Plugin types from magda-agents
type PluginInfo = magdaplugin.PluginInfo
type PluginAlias = magdaplugin.PluginAlias
type Preferences = magdaplugin.Preferences

func NewMagdaHandler(cfg *config.Config, db *gorm.DB) *MagdaHandler {
	// Convert aideas-api config to magda-agents config
	magdaCfg := &magdaconfig.Config{
		OpenAIAPIKey: cfg.OpenAIAPIKey,
		GeminiAPIKey: cfg.GeminiAPIKey,
		MCPServerURL: cfg.MCPServerURL,
	}

	return &MagdaHandler{
		orchestrator:  magdaorchestrator.NewOrchestrator(magdaCfg),
		pluginService: magdaplugin.NewPluginAgent(magdaCfg),
		db:            db,
		cfg:           cfg,
	}
}

type MagdaChatRequest struct {
	Question string                 `json:"question" binding:"required"`
	State    map[string]interface{} `json:"state"` // REAPER state snapshot
}

func (h *MagdaHandler) Chat(c *gin.Context) {
	// Add panic recovery with detailed logging
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ MAGDA Chat: PANIC recovered: %v", r)
			log.Printf("   Stack trace:\n%s", string(debug.Stack()))
			log.Printf("   Request ID: %s", c.GetString("request_id"))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      fmt.Sprintf("Internal server error: %v", r),
				"request_id": c.GetString("request_id"),
			})
		}
	}()

	var req MagdaChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ MAGDA Chat: JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log incoming request
	log.Printf("📨 MAGDA Chat: Received request")
	log.Printf("   Question length: %d", len(req.Question))
	if len(req.Question) > 0 {
		previewLen := 200
		if len(req.Question) < previewLen {
			previewLen = len(req.Question)
		}
		log.Printf("   Question preview: %s", req.Question[:previewLen])
	}
	if req.State != nil {
		log.Printf("   State keys: %d", len(req.State))
		// Log state size estimate
		stateJSON, _ := json.Marshal(req.State)
		log.Printf("   State JSON size: %d bytes", len(stateJSON))
	} else {
		log.Printf("   State: nil")
	}

	// Get user from context (optional - MAGDA might be public for now)
	userID, _ := middleware.GetCurrentUserID(c)
	if userID > 0 {
		log.Printf("   User ID: %d", userID)
	}

	// Start Langfuse trace for observability
	lfClient := observability.GetClient()
	log.Printf("🔍 Langfuse: Client enabled: %v", lfClient.IsEnabled())
	trace := lfClient.StartTrace(c.Request.Context(), "magda-chat", map[string]interface{}{
		"question": req.Question,
		"user_id":  userID,
	})
	log.Printf("🔍 Langfuse: Trace created, will finish on defer")
	defer func() {
		log.Printf("🔍 Langfuse: Finishing trace...")
		trace.Finish()
		log.Printf("🔍 Langfuse: Trace finished")
	}()

	// Generate actions from question and state using orchestrator
	log.Printf("🚀 MAGDA Chat: Calling Orchestrator.GenerateActions")
	gen := trace.Generation("orchestrator", map[string]interface{}{
		"question": req.Question,
	})
	log.Printf("🔍 Langfuse: Generation span created")
	gen.Input(req.Question)

	result, err := h.orchestrator.GenerateActions(c.Request.Context(), req.Question, req.State)
	if err != nil {
		log.Printf("❌ MAGDA Chat: GenerateActions error: %v", err)
		log.Printf("   Error type: %T", err)
		log.Printf("   Stack trace:\n%s", string(debug.Stack()))
		gen.SetLevel("ERROR")
		gen.Output(err.Error())
		gen.Finish()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Log result to Langfuse
	log.Printf("🔍 Langfuse: Setting generation output (%d actions)", len(result.Actions))
	gen.Output(result.Actions)
	gen.Metadata(map[string]interface{}{
		"actions_count": len(result.Actions),
	})
	log.Printf("🔍 Langfuse: Finishing generation span...")
	gen.Finish()
	log.Printf("🔍 Langfuse: Generation span finished")

	// Log result
	log.Printf("✅ MAGDA Chat: GenerateActions succeeded")
	log.Printf("   Actions count: %d", len(result.Actions))
	if len(result.Actions) > 0 {
		actionsJSON, _ := json.Marshal(result.Actions)
		previewLen := 500
		if len(actionsJSON) < previewLen {
			previewLen = len(actionsJSON)
		}
		log.Printf("   Actions preview: %s", string(actionsJSON[:previewLen]))
	}
	if result.Usage != nil {
		log.Printf("   Usage: %+v", result.Usage)
	}

	// Build response
	response := gin.H{
		"request_id": c.GetString("request_id"),
		"actions":    result.Actions,
		"usage":      result.Usage,
	}

	// Log response before sending
	responseJSON, _ := json.Marshal(response)
	log.Printf("📤 MAGDA Chat: Sending response (%d bytes)", len(responseJSON))
	previewLen := 500
	if len(responseJSON) < previewLen {
		previewLen = len(responseJSON)
	}
	log.Printf("   Response preview: %s", string(responseJSON[:previewLen]))

	// Return actions in the format MAGDA expects
	c.JSON(http.StatusOK, response)
}

// ChatStream handles streaming MAGDA chat requests (experimental - no structured output)
func (h *MagdaHandler) ChatStream(c *gin.Context) {
	var req MagdaChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ MAGDA ChatStream: JSON binding error: %v", err)
		log.Printf("   Request method: %s, Path: %s", c.Request.Method, c.Request.URL.Path)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log request details
	log.Printf("📨 MAGDA ChatStream: Question length=%d, State keys=%d", len(req.Question), len(req.State))
	if len(req.Question) > 0 {
		previewLen := 200
		if len(req.Question) < previewLen {
			previewLen = len(req.Question)
		}
		log.Printf("   Question: %s", req.Question[:previewLen])
	}
	if len(req.State) > 0 {
		log.Printf("   State has %d keys", len(req.State))
	}

	// Get user from context
	_, _ = middleware.GetCurrentUserID(c)

	// Set headers for SSE
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering
	c.Header("X-Request-ID", c.GetString("request_id"))

	// Flush headers
	c.Writer.Flush()

	// Stream callback - sends each action as it's parsed
	// Wrap action in an object with "type": "action" for the extension to parse
	actionCallback := func(action map[string]interface{}) error {
		// Wrap action in an event object
		event := gin.H{
			"type":    "action",
			"action":  action,
			"message": "Action received",
		}
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Printf("❌ MAGDA ChatStream: Failed to marshal action event: %v", err)
			return err
		}

		log.Printf("📤 MAGDA ChatStream: Sending action event: %s", string(eventJSON))

		// Write SSE event
		_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
		if err != nil {
			log.Printf("❌ MAGDA ChatStream: Failed to write SSE event: %v", err)
			return err
		}

		// Flush immediately
		c.Writer.Flush()
		return nil
	}

	// Call streaming orchestrator - coordinates DAW + Arranger agents
	// Emits actions progressively: create_track, create_clip immediately,
	// then add_midi once arranger notes are ready
	log.Printf("🚀 MAGDA ChatStream: Calling Orchestrator.GenerateActionsStream")
	result, err := h.orchestrator.GenerateActionsStream(c.Request.Context(), req.Question, req.State, actionCallback)
	if err != nil {
		log.Printf("❌ MAGDA ChatStream: GenerateActionsStream error: %v", err)
		// Send error event
		errorEvent := gin.H{
			"type":    "error",
			"message": err.Error(),
		}
		eventJSON, _ := json.Marshal(errorEvent)
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
		c.Writer.Flush()
		return
	}

	log.Printf("✅ MAGDA ChatStream: Completed successfully, %d actions generated", len(result.Actions))

	// Send final completion event
	finalEvent := gin.H{
		"type":       "done",
		"request_id": c.GetString("request_id"),
		"actions":    result.Actions,
		"usage":      result.Usage,
	}
	eventJSON, _ := json.Marshal(finalEvent)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
	c.Writer.Flush()
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// DSLStream handles streaming MAGDA requests with explicit DSL mode
// POST /api/v1/magda/dsl/stream
// This endpoint explicitly uses DSL/CFG mode for generation
func (h *MagdaHandler) DSLStream(c *gin.Context) {
	var req MagdaChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ MAGDA DSLStream: JSON binding error: %v", err)
		// Read the request body to log it, then replace it for subsequent handlers
		bodyBytes, _ := io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body for potential re-reading
		log.Printf("   Request body preview: %s", truncateString(string(bodyBytes), maxRequestPreviewLength))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("📨 MAGDA DSLStream: Question length=%d, State keys=%d", len(req.Question), len(req.State))

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	// Stream callback - called for each action as it's generated
	actionCount := 0
	streamCallback := func(action map[string]interface{}) error {
		actionCount++
		event := map[string]interface{}{
			"type":    "action",
			"action":  action,
			"message": "Action received",
		}
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Printf("❌ MAGDA DSLStream: Failed to marshal action event: %v", err)
			return err
		}
		log.Printf("📤 MAGDA DSLStream: Sending action event: %s", string(eventJSON))
		_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
		if err != nil {
			log.Printf("❌ MAGDA DSLStream: Failed to write SSE event: %v", err)
			return err
		}
		c.Writer.Flush()
		return nil
	}

	// Call streaming orchestrator - coordinates DAW + Arranger agents
	log.Printf("🚀 MAGDA DSLStream: Calling Orchestrator.GenerateActionsStream")
	result, err := h.orchestrator.GenerateActionsStream(c.Request.Context(), req.Question, req.State, streamCallback)
	if err != nil {
		// If we already sent actions via the callback, don't send an error
		// (DSL mode may report "no output" error even when actions were successfully parsed)
		if actionCount > 0 {
			log.Printf("⚠️  MAGDA DSLStream: GenerateActionsStream reported error but %d actions were already sent: %v", actionCount, err)
			// Continue to send final "done" event
		} else {
			log.Printf("❌ MAGDA DSLStream: GenerateActionsStream error: %v", err)
			errorEvent := map[string]interface{}{
				"type":    "error",
				"message": err.Error(),
			}
			eventJSON, _ := json.Marshal(errorEvent)
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
			c.Writer.Flush()
			return
		}
	}

	log.Printf("✅ MAGDA DSLStream: Completed successfully, %d actions generated", len(result.Actions))

	// Send final "done" event with all actions
	finalEvent := map[string]interface{}{
		"type":    "done",
		"actions": result.Actions,
		"usage":   result.Usage,
	}
	eventJSON, _ := json.Marshal(finalEvent)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
	c.Writer.Flush()
}

// TestDSL is a test endpoint for parsing DSL code directly
// POST /api/v1/magda/dsl
// Body: {"dsl": "track(instrument=\"Serum\").newClip(bar=3, length_bars=4)"}
func (h *MagdaHandler) TestDSL(c *gin.Context) {
	var req struct {
		DSL string `json:"dsl" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("🧪 Testing DSL parser with: %s", req.DSL)

	// Parse DSL directly
	parser := magdadaw.NewDSLParser()
	actions, err := parser.ParseDSL(req.DSL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"dsl":     req.DSL,
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"dsl":     req.DSL,
		"actions": actions,
		"count":   len(actions),
	})
}

// ProcessPlugins generates aliases for plugins
// POST /api/v1/magda/plugins/process
// Note: Plugins are already deduplicated by the REAPER extension before sending
func (h *MagdaHandler) ProcessPlugins(c *gin.Context) {
	var req struct {
		Plugins []PluginInfo `json:"plugins" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("📦 ProcessPlugins: Received %d plugins (already deduplicated by extension)", len(req.Plugins))

	// Plugins are already deduplicated by the REAPER extension
	// Just generate aliases for the provided plugins
	aliases, err := h.pluginService.GenerateAliases(c.Request.Context(), req.Plugins)
	if err != nil {
		log.Printf("❌ ProcessPlugins: Alias generation error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	log.Printf("✅ ProcessPlugins: Generated %d aliases", len(aliases))

	c.JSON(http.StatusOK, gin.H{
		"plugins":       req.Plugins,
		"aliases":       aliases,
		"plugins_count": len(req.Plugins),
		"aliases_count": len(aliases),
	})
}
