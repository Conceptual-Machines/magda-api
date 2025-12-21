package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"

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
	DSL        string `json:"dsl"`                   // Raw DSL from LLM
	JSFXCode   string `json:"jsfx_code"`             // Generated JSFX code
	ParseError string `json:"parse_error,omitempty"` // Parser error for human-in-the-loop
	Message    string `json:"message"`               // Message to user
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
		DSL:        result.DSL,
		JSFXCode:   result.JSFXCode,
		ParseError: result.ParseError,
	}

	// Set appropriate message based on result
	if result.ParseError != "" {
		response.Message = "DSL generated but parsing failed. Please review and provide feedback."
		log.Printf("⚠️ JSFX Generate: Partial success (parse error)")
	} else {
		response.Message = "JSFX generated successfully"
		log.Printf("✅ JSFX Generate: Success")
	}

	// Log response
	responseJSON, _ := json.Marshal(response)
	log.Printf("   DSL length: %d bytes", len(result.DSL))
	log.Printf("   JSFX length: %d bytes", len(result.JSFXCode))
	if result.ParseError != "" {
		log.Printf("   Parse error: %s", result.ParseError)
	}
	log.Printf("   Response preview: %s", truncateStr(string(responseJSON), logResponseMaxLen))

	c.JSON(http.StatusOK, response)
}

// truncateStr truncates a string to maxLen characters
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
