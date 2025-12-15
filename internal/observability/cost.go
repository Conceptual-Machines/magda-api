package observability

import (
	"strconv"

	"github.com/openai/openai-go/responses"
)

// ModelPricing contains pricing information per 1K tokens
type ModelPricing struct {
	InputPricePer1K  float64 // Price per 1K input tokens in USD
	OutputPricePer1K float64 // Price per 1K output tokens in USD
}

// PricingTable contains pricing for all models
var PricingTable = map[string]ModelPricing{
	// GPT-5.1 models (example pricing - update with actual rates)
	"gpt-5.1": {
		InputPricePer1K:  0.001, // $0.001 per 1K input tokens
		OutputPricePer1K: 0.003, // $0.003 per 1K output tokens
	},
	"gpt-5.1-mini": {
		InputPricePer1K:  0.0005, // $0.0005 per 1K input tokens
		OutputPricePer1K: 0.0015, // $0.0015 per 1K output tokens
	},
	// GPT-4 models
	"gpt-4o": {
		InputPricePer1K:  0.005, // $0.005 per 1K input tokens
		OutputPricePer1K: 0.015, // $0.015 per 1K output tokens
	},
	"gpt-4o-mini": {
		InputPricePer1K:  0.00015, // $0.00015 per 1K input tokens
		OutputPricePer1K: 0.0006,  // $0.0006 per 1K output tokens
	},
}

// CalculateOpenAICost calculates the cost in USD for an OpenAI API call
func CalculateOpenAICost(model string, usage responses.ResponseUsage) float64 {
	pricing, exists := PricingTable[model]
	if !exists {
		// Default to GPT-5.1 pricing if model not found
		pricing = PricingTable["gpt-5.1"]
	}

	inputCost := (float64(usage.InputTokens) / 1000.0) * pricing.InputPricePer1K
	outputCost := (float64(usage.OutputTokens) / 1000.0) * pricing.OutputPricePer1K

	// Add reasoning tokens if present
	reasoningCost := 0.0
	if usage.OutputTokensDetails.ReasoningTokens > 0 {
		// Reasoning tokens typically cost the same as input tokens
		reasoningCost = (float64(usage.OutputTokensDetails.ReasoningTokens) / 1000.0) * pricing.InputPricePer1K
	}

	totalCost := inputCost + outputCost + reasoningCost
	return totalCost
}

// FormatCost formats a cost value as a USD string
func FormatCost(cost float64) string {
	// Format with 6 decimal places for precision
	return "$" + formatFloat(cost, 6)
}

// formatFloat formats a float with specified precision using strconv
func formatFloat(f float64, precision int) string {
	return strconv.FormatFloat(f, 'f', precision, 64)
}
