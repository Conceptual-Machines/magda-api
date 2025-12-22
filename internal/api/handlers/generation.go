package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	magdaarranger "github.com/Conceptual-Machines/magda-agents-go/agents/arranger"
	magdaconfig "github.com/Conceptual-Machines/magda-agents-go/config"
	"github.com/Conceptual-Machines/magda-api/internal/api/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/openai/openai-go/responses"
)

const (
	defaultModel         = "gpt-5-mini"
	defaultReasoningMode = "medium"
	outputFormatDSL      = "dsl"
)

type GenerationHandler struct {
	genService *magdaarranger.GenerationService
	cfg        *config.Config
}

func NewGenerationHandler(cfg *config.Config) *GenerationHandler {
	// Convert config to magda-agents config
	magdaCfg := &magdaconfig.Config{
		OpenAIAPIKey: cfg.OpenAIAPIKey,
		GeminiAPIKey: cfg.GeminiAPIKey,
		MCPServerURL: cfg.MCPServerURL,
	}
	baseService := magdaarranger.NewGenerationService(magdaCfg)

	return &GenerationHandler{
		genService: baseService,
		cfg:        cfg,
	}
}

type GenerateRequest struct {
	Model string `json:"model"` // Model to use (e.g., gpt-5-mini, gpt-4o)
	// Optional: provider override (openai, gemini) - defaults to provider based on model
	Provider     string                   `json:"provider"`
	InputArray   []map[string]interface{} `json:"input_array" binding:"required"`
	Stream       bool                     `json:"stream"`        // Enable streaming
	OutputFormat string                   `json:"output_format"` // Output format: "dsl" (default, faster) or "json_schema" (structured JSON)

	// Generation parameters
	KeepContext   bool   `json:"keep_context"`   // Keep context between requests
	ReasoningMode string `json:"reasoning_mode"` // Reasoning mode (minimal, low, medium, high)
}

func (h *GenerationHandler) Generate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from gateway headers (required for this endpoint)
	userIDStr, exists := middleware.GetUserIDFromGateway(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	// For logging purposes
	_ = userIDStr

	// Use requested model or default to gpt-5-mini
	// Allow gpt-5-mini and gpt-5-nano
	model := req.Model
	if model == "" {
		model = defaultModel
	}

	// Validate model - support OpenAI GPT-5 and Google Gemini models
	allowedModels := map[string]bool{
		// OpenAI GPT-5 models
		"gpt-5-mini": true,
		"gpt-5-nano": true,
		// Google Gemini 2.5 models (latest)
		"gemini-2.5-flash": true,
		"gemini-2.5-pro":   true,
	}
	if !allowedModels[model] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid model. Allowed: gpt-5-mini, gpt-5-nano, gemini-2.5-flash, gemini-2.5-pro",
		})
		return
	}

	// Validate reasoning mode (allow empty, will default to medium)
	if req.ReasoningMode != "" {
		allowedReasoningModes := map[string]bool{
			"minimal": true,
			"low":     true,
			"medium":  true,
			"high":    true,
		}
		if !allowedReasoningModes[req.ReasoningMode] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reasoning_mode. Allowed: minimal, low, medium, high"})
			return
		}
	}

	// TODO: Credits service not yet implemented
	// Get current credits (for response, but don't block)
	// credits, err := h.creditsService.GetUserCredits(userID)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check credits"})
	// 	return
	// }

	// Soft warning if credits are low (but allow request to proceed)
	// if credits.Credits < lowCreditThreshold {
	// 	c.Header("X-Credits-Low", "true")
	// 	c.Header("X-Credits-Balance", fmt.Sprintf("%d", credits.Credits))
	// }

	// Route based on streaming preference
	if req.Stream {
		h.generateStream(c, req, model)
		return
	}

	h.generateOneShot(c, req, model)
}

// generateOneShot handles non-streaming one-shot generation
func (h *GenerationHandler) generateOneShot(c *gin.Context, req GenerateRequest, model string) {
	startTime := time.Now()

	// Use reasoning mode from request, default to "medium" for GPT-5
	reasoningMode := req.ReasoningMode
	if reasoningMode == "" {
		reasoningMode = defaultReasoningMode
	}

	// Note: Using magda-agents-go GenerationService which uses OpenAI provider from config
	// Provider selection is not currently supported

	// Create a service with the selected provider
	magdaCfg := &magdaconfig.Config{
		OpenAIAPIKey: h.cfg.OpenAIAPIKey,
		GeminiAPIKey: h.cfg.GeminiAPIKey,
		MCPServerURL: h.cfg.MCPServerURL,
	}
	genService := magdaarranger.NewGenerationService(magdaCfg)

	// Note: magda-agents-go GenerationService doesn't support outputFormat parameter
	// It always uses JSON Schema output format
	result, err := genService.Generate(c.Request.Context(), model, req.InputArray, reasoningMode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	duration := time.Since(startTime)

	// Extract token usage for logging/tracking (Sentry/metrics should be in API layer, not agents)
	totalTokens := h.extractTotalTokens(result.Usage)
	inputTokens := h.extractInputTokens(result.Usage)
	outputTokens := h.extractOutputTokens(result.Usage)
	reasoningTokens := h.extractReasoningTokens(result.Usage)

	// TODO: Log usage/metrics here (Sentry, database, etc.)
	log.Printf("📊 Token usage - Total: %d, Input: %d, Output: %d, Reasoning: %d, Duration: %v",
		totalTokens, inputTokens, outputTokens, reasoningTokens, duration)
	// Deduct credits (may go negative up to -50)
	// deductErr := h.creditsService.DeductCredits(userID, creditsCharged)
	// creditLimitExceeded := deductErr != nil

	// Log usage regardless of credit deduction result
	// usageLog := &models.UsageLog{
	// 	UserID:          userID,
	// 	Model:           model,
	// 	TotalTokens:     totalTokens,
	// 	InputTokens:     h.extractInputTokens(result.Usage),
	// 	OutputTokens:    h.extractOutputTokens(result.Usage),
	// 	ReasoningTokens: h.extractReasoningTokens(result.Usage),
	// 	CreditsCharged:  creditsCharged,
	// 	MCPUsed:         result.MCPUsed,
	// 	MCPCalls:        result.MCPCalls,
	// 	MCPTools:        strings.Join(result.MCPTools, ","),
	// 	DurationMS:      int(duration.Milliseconds()),
	// 	RequestID:       c.GetString("request_id"),
	// }
	// if err := h.creditsService.LogUsage(usageLog); err != nil {
	// 	fmt.Printf("Failed to log usage: %v\n", err)
	// }

	// Add request ID to response
	response := gin.H{
		"request_id":    c.GetString("request_id"),
		"output_parsed": result.OutputParsed,
		"usage":         result.Usage,
		// "credits_charged":   creditsCharged,
		// "credits_remaining": creditsRemaining,
	}

	// Add optional fields if present
	if result.MCPUsed {
		response["mcpUsed"] = result.MCPUsed
		response["mcpCalls"] = result.MCPCalls
		response["mcpTools"] = result.MCPTools
	}

	c.JSON(http.StatusOK, response)
}

// Helper functions to extract token counts from usage
func (h *GenerationHandler) extractTotalTokens(usage any) int {
	// Try ResponseUsage struct first (streaming response)
	if u, ok := usage.(responses.ResponseUsage); ok {
		return int(u.TotalTokens)
	}
	// Fallback to map (non-streaming response)
	if usageMap, ok := usage.(map[string]any); ok {
		if total, ok := usageMap["total_tokens"].(float64); ok {
			return int(total)
		}
	}
	return 0
}

func (h *GenerationHandler) extractInputTokens(usage any) int {
	// Try ResponseUsage struct first (streaming response)
	if u, ok := usage.(responses.ResponseUsage); ok {
		return int(u.InputTokens)
	}
	// Fallback to map (non-streaming response)
	if usageMap, ok := usage.(map[string]any); ok {
		if input, ok := usageMap["input_tokens"].(float64); ok {
			return int(input)
		}
	}
	return 0
}

func (h *GenerationHandler) extractOutputTokens(usage any) int {
	// Try ResponseUsage struct first (streaming response)
	if u, ok := usage.(responses.ResponseUsage); ok {
		return int(u.OutputTokens)
	}
	// Fallback to map (non-streaming response)
	if usageMap, ok := usage.(map[string]any); ok {
		if output, ok := usageMap["output_tokens"].(float64); ok {
			return int(output)
		}
	}
	return 0
}

func (h *GenerationHandler) extractReasoningTokens(usage any) int {
	// Try ResponseUsage struct first (streaming response)
	if u, ok := usage.(responses.ResponseUsage); ok {
		return int(u.OutputTokensDetails.ReasoningTokens)
	}
	// Fallback to map (non-streaming response)
	if usageMap, ok := usage.(map[string]any); ok {
		if details, ok := usageMap["output_tokens_details"].(map[string]any); ok {
			if reasoning, ok := details["reasoning_tokens"].(float64); ok {
				return int(reasoning)
			}
		}
		// Legacy format (flat reasoning_tokens)
		if reasoning, ok := usageMap["reasoning_tokens"].(float64); ok {
			return int(reasoning)
		}
	}
	return 0
}

func (h *GenerationHandler) generateStream(c *gin.Context, req GenerateRequest, model string) {
	startTime := time.Now()

	// Use reasoning mode from request, default to "medium"
	reasoningMode := req.ReasoningMode
	if reasoningMode == "" {
		reasoningMode = defaultReasoningMode
	}

	// Auth is handled by middleware before reaching this point

	// Create a service (uses default OpenAI provider from config)
	magdaCfg := &magdaconfig.Config{
		OpenAIAPIKey: h.cfg.OpenAIAPIKey,
		GeminiAPIKey: h.cfg.GeminiAPIKey,
		MCPServerURL: h.cfg.MCPServerURL,
	}
	genService := magdaarranger.NewGenerationService(magdaCfg)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Note: magda-agents-go GenerationService always uses JSON Schema output format
	// outputFormat parameter is ignored

	// Stream events and capture result
	// Note: magda-agents-go GenerationService doesn't support outputFormat parameter
	result, err := genService.GenerateStream(
		c.Request.Context(), model, req.InputArray, reasoningMode,
		func(event magdaarranger.StreamEvent) error {
			eventJSON, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				return marshalErr
			}
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
			c.Writer.Flush()
			return nil
		})

	if err != nil {
		errorEvent := magdaarranger.StreamEvent{
			Type:    "error",
			Message: err.Error(),
		}
		eventJSON, _ := json.Marshal(errorEvent)
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
		c.Writer.Flush()
		return
	}

	duration := time.Since(startTime)

	// Extract token usage for logging/tracking (Sentry/metrics should be in API layer, not agents)
	totalTokens := h.extractTotalTokens(result.Usage)
	inputTokens := h.extractInputTokens(result.Usage)
	outputTokens := h.extractOutputTokens(result.Usage)
	reasoningTokens := h.extractReasoningTokens(result.Usage)

	// TODO: Log usage/metrics here (Sentry, database, etc.)
	log.Printf("📊 Token usage (streaming) - Total: %d, Input: %d, Output: %d, Reasoning: %d, Duration: %v",
		totalTokens, inputTokens, outputTokens, reasoningTokens, duration)

	// Send final result event with complete output_parsed data
	finalEvent := magdaarranger.StreamEvent{
		Type:    "result",
		Message: "Generation complete",
		Data: map[string]interface{}{
			"output_parsed": map[string]interface{}{
				"choices": result.OutputParsed.Choices,
			},
			"mcp_used": result.MCPUsed,
		},
	}
	finalJSON, _ := json.Marshal(finalEvent)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", finalJSON)
	c.Writer.Flush()

	// Send done event
	doneEvent := magdaarranger.StreamEvent{
		Type:    "done",
		Message: "Stream complete",
		Data: map[string]interface{}{
			"request_id": c.GetString("request_id"),
		},
	}
	eventJSON, _ := json.Marshal(doneEvent)
	_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", eventJSON)
	c.Writer.Flush()
}
