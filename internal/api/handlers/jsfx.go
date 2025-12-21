package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/Conceptual-Machines/magda-agents-go/agents/jsfx"
	agentconfig "github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Log truncation limits
const (
	logMessageMaxLen  = 200
	logResponseMaxLen = 500
)

// Response messages
const (
	msgJSFXSuccess      = "JSFX generated successfully"
	msgJSFXCompileError = "JSFX generated but has compile errors. Please review and fix."
)

// JSFXHandler handles JSFX generation requests
type JSFXHandler struct {
	agent *jsfx.JSFXAgent
	db    *gorm.DB
}

// NewJSFXHandler creates a new JSFX handler
func NewJSFXHandler(cfg *config.Config, db *gorm.DB) *JSFXHandler {
	// Create agent config from API config
	agentCfg := &agentconfig.Config{
		OpenAIAPIKey: cfg.OpenAIAPIKey,
	}

	return &JSFXHandler{
		agent: jsfx.NewJSFXAgent(agentCfg),
		db:    db,
	}
}

// JSFXGenerateRequest is the request body for JSFX generation
type JSFXGenerateRequest struct {
	Message  string              `json:"message"`  // User's request (e.g., "Create a compressor")
	Code     string              `json:"code"`     // Current JSFX code in editor (context)
	Filename string              `json:"filename"` // Current filename
	History  []map[string]string `json:"history"`  // Chat history (optional)
}

// JSFXGenerateResponse is the response for JSFX generation
type JSFXGenerateResponse struct {
	JSFXCode     string `json:"jsfx_code"`               // Generated JSFX code (direct from LLM)
	CompileError string `json:"compile_error,omitempty"` // EEL2 compile error if validation enabled
	Message      string `json:"message"`                 // Message to user
}

// Generate handles JSFX generation requests
func (h *JSFXHandler) Generate(c *gin.Context) {
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ JSFX Generate: PANIC recovered: %v", r)
			log.Printf("   Stack trace:\n%s", string(debug.Stack()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      fmt.Sprintf("Internal server error: %v", r),
				"request_id": c.GetString("request_id"),
			})
		}
	}()

	var req JSFXGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ JSFX Generate: JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log request
	log.Printf("📨 JSFX Generate: Received request")
	log.Printf("   Message: %s", truncateStr(req.Message, logMessageMaxLen))
	log.Printf("   Code length: %d bytes", len(req.Code))
	log.Printf("   Filename: %s", req.Filename)

	// Get user from context
	userID, _ := middleware.GetCurrentUserID(c)
	if userID > 0 {
		log.Printf("   User ID: %d", userID)
	}

	// Build input messages for the agent
	inputArray := make([]map[string]any, 0)

	// Add code context if provided
	if req.Code != "" {
		inputArray = append(inputArray, map[string]any{
			"role":    "user",
			"content": fmt.Sprintf("Current JSFX code in %s:\n```\n%s\n```", req.Filename, req.Code),
		})
	}

	// Add chat history if provided
	for _, msg := range req.History {
		role := msg["role"]
		content := msg["content"]
		if role != "" && content != "" {
			inputArray = append(inputArray, map[string]any{
				"role":    role,
				"content": content,
			})
		}
	}

	// Add the current message
	inputArray = append(inputArray, map[string]any{
		"role":    "user",
		"content": req.Message,
	})

	// Call JSFX agent
	ctx := c.Request.Context()
	result, err := h.agent.Generate(ctx, "gpt-5.2", inputArray)
	if err != nil {
		log.Printf("❌ JSFX Generate: Agent error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      fmt.Sprintf("JSFX generation failed: %v", err),
			"request_id": c.GetString("request_id"),
		})
		return
	}

	// Build response
	response := JSFXGenerateResponse{
		JSFXCode:     result.JSFXCode,
		CompileError: result.CompileError,
	}

	// Set appropriate message based on result
	if result.CompileError != "" {
		response.Message = msgJSFXCompileError
		log.Printf("⚠️ JSFX Generate: Compile error: %s", result.CompileError)
	} else {
		response.Message = msgJSFXSuccess
		log.Printf("✅ JSFX Generate: Success")
	}

	// Log response
	responseJSON, _ := json.Marshal(response)
	log.Printf("   JSFX length: %d bytes", len(result.JSFXCode))
	if result.CompileError != "" {
		log.Printf("   Compile error: %s", result.CompileError)
	}
	log.Printf("   Response preview: %s", truncateStr(string(responseJSON), logResponseMaxLen))

	// Use PureJSON to avoid HTML-escaping < and > in JSFX code
	c.PureJSON(http.StatusOK, response)
}

// truncateStr truncates a string to maxLen characters
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// GenerateStream handles JSFX generation requests with TRUE SSE streaming
// Characters are streamed to the client in real-time as they arrive from the LLM
func (h *JSFXHandler) GenerateStream(c *gin.Context) {
	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ JSFX Stream: PANIC recovered: %v", r)
			log.Printf("   Stack trace:\n%s", string(debug.Stack()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":      fmt.Sprintf("Internal server error: %v", r),
				"request_id": c.GetString("request_id"),
			})
		}
	}()

	var req JSFXGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ JSFX Stream: JSON binding error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Log request
	log.Printf("📨 JSFX Stream: Received request (TRUE STREAMING)")
	log.Printf("   Message: %s", truncateStr(req.Message, logMessageMaxLen))
	log.Printf("   Code length: %d bytes", len(req.Code))
	log.Printf("   Filename: %s", req.Filename)

	// Set up SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Writer.Flush()

	// Utility to send SSE events
	sendEvent := func(event map[string]any) error {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			log.Printf("❌ JSFX Stream: Failed to marshal event: %v", err)
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON); err != nil {
			log.Printf("❌ JSFX Stream: Failed to write SSE event: %v", err)
			return err
		}
		c.Writer.Flush()
		return nil
	}

	// Send initial start event
	_ = sendEvent(map[string]any{
		"type":    "start",
		"message": "Generating JSFX...",
	})

	// Get user from context
	userID, _ := middleware.GetCurrentUserID(c)
	if userID > 0 {
		log.Printf("   User ID: %d", userID)
	}

	// Build input messages for the agent
	inputArray := make([]map[string]any, 0)

	// Add code context if provided
	if req.Code != "" {
		inputArray = append(inputArray, map[string]any{
			"role":    "user",
			"content": fmt.Sprintf("Current JSFX code in %s:\n```\n%s\n```", req.Filename, req.Code),
		})
	}

	// Add chat history if provided
	for _, msg := range req.History {
		role := msg["role"]
		content := msg["content"]
		if role != "" && content != "" {
			inputArray = append(inputArray, map[string]any{
				"role":    role,
				"content": content,
			})
		}
	}

	// Add the current message
	inputArray = append(inputArray, map[string]any{
		"role":    "user",
		"content": req.Message,
	})

	ctx := c.Request.Context()
	var streamedOutput strings.Builder
	chunkCount := 0

	// TRUE STREAMING: callback is called for each chunk as it arrives from the LLM
	streamCallback := func(chunk string) error {
		chunkCount++
		streamedOutput.WriteString(chunk)

		// Send chunk event to client in real-time
		// Client accumulates chunks to build the full code
		return sendEvent(map[string]any{
			"type":  "chunk",
			"chunk": chunk,
		})
	}

	// Call JSFX agent with TRUE streaming - chunks arrive in real-time from OpenAI
	log.Printf("🚀 JSFX Stream: Starting true streaming generation...")
	result, err := h.agent.GenerateStream(ctx, "gpt-5.2", inputArray, streamCallback)
	if err != nil {
		log.Printf("❌ JSFX Stream: Agent error: %v", err)
		_ = sendEvent(map[string]any{
			"type":    "error",
			"message": fmt.Sprintf("JSFX generation failed: %v", err),
		})
		return
	}

	log.Printf("✅ JSFX Stream: Completed with %d chunks streamed", chunkCount)

	// Build final code (use result or streamed output)
	finalCode := result.JSFXCode
	if finalCode == "" {
		finalCode = streamedOutput.String()
	}

	// Generate description for the code (separate fast call)
	var description string
	if finalCode != "" {
		log.Printf("📝 JSFX Stream: Generating description...")
		desc, descErr := h.agent.DescribeJSFX(ctx, "gpt-5.2", finalCode)
		if descErr != nil {
			log.Printf("⚠️ JSFX Stream: Description generation failed: %v", descErr)
		} else {
			description = desc
			log.Printf("✅ JSFX Stream: Description generated (%d chars)", len(description))
		}
	}

	// Prepare response message
	message := msgJSFXSuccess
	if result.CompileError != "" {
		message = msgJSFXCompileError
	}

	// Send final done event with complete code and description
	_ = sendEvent(map[string]any{
		"type":          "done",
		"jsfx_code":     finalCode,
		"description":   description,
		"compile_error": result.CompileError,
		"message":       message,
		"usage":         result.Usage,
	})
}
