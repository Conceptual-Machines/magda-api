package services

import (
	"github.com/openai/openai-go/responses"
)

// LLMStage represents which stage of generation we're in
type LLMStage string

const (
	LLMStagePlanner   LLMStage = "planner"
	LLMStageGenerator LLMStage = "generator"
	LLMStageOneShot   LLMStage = "one_shot" // Single-prompt mode
)

// Reasoning effort constants
const (
	reasoningEffortMinimal = "minimal"
	reasoningEffortMedium  = "medium"
)

// LLMParameters contains the configuration for Responses API calls
// Using gpt-5-mini with Responses API
// Reference: https://cookbook.openai.com/examples/gpt-5/gpt-5_new_params_and_tools
type LLMParameters struct {
	Model           string                    // Always "gpt-5-mini"
	ReasoningEffort responses.ReasoningEffort // minimal, low, medium, high
	TextFormat      *map[string]any           // JSON schema for structured output
}

// GetLLMParameters returns the appropriate parameters for each stage
// Based on GPT-5 best practices: https://cookbook.openai.com/examples/gpt-5/gpt-5_new_params_and_tools
func GetLLMParameters(stage LLMStage, jsonSchema map[string]any) LLMParameters {
	switch stage {
	case LLMStagePlanner:
		// Planner: Creative planning with low reasoning (optimized for speed)
		return LLMParameters{
			Model:           "gpt-5-mini",
			ReasoningEffort: responses.ReasoningEffortLow,
		}

	case LLMStageGenerator:
		// Generator: Fast execution with MINIMAL reasoning (GPT-5 feature)
		// Minimal = fastest time-to-first-token for deterministic output
		return LLMParameters{
			Model:           "gpt-5-mini",
			ReasoningEffort: "minimal",   // GPT-5 minimal reasoning
			TextFormat:      &jsonSchema, // Strict JSON schema
		}

	case LLMStageOneShot:
		fallthrough
	default:
		// One-shot: Single-prompt approach (current generation.go)
		return LLMParameters{
			Model:           "gpt-5-mini",
			ReasoningEffort: responses.ReasoningEffortLow,
			TextFormat:      &jsonSchema,
		}
	}
}

// GetReasoningEffort returns the reasoning effort for Responses API
// Used by one-shot mode when user specifies reasoning preference
// GPT-5 supports: minimal (fastest), low, medium, high (most thorough)
func GetReasoningEffort(reasoningMode string) responses.ReasoningEffort {
	switch reasoningMode {
	case "high":
		return responses.ReasoningEffortHigh
	case reasoningEffortMedium:
		return responses.ReasoningEffortMedium
	case "low":
		return responses.ReasoningEffortLow
	case reasoningEffortMinimal:
		return reasoningEffortMinimal // GPT-5 minimal reasoning
	default:
		return responses.ReasoningEffortLow
	}
}
