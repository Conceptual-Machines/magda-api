package observability

import (
	"strconv"

	"github.com/openai/openai-go/responses"
)

// Pricing constants
const (
	tokensPerKilo       = 1000.0
	costFormatPrecision = 6

	// GPT-5.1 pricing
	gpt51InputPrice  = 0.001
	gpt51OutputPrice = 0.003

	// GPT-5.1-mini pricing
	gpt51MiniInputPrice  = 0.0005
	gpt51MiniOutputPrice = 0.0015

	// GPT-4o pricing
	gpt4oInputPrice  = 0.005
	gpt4oOutputPrice = 0.015

	// GPT-4o-mini pricing
	gpt4oMiniInputPrice  = 0.00015
	gpt4oMiniOutputPrice = 0.0006
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
		InputPricePer1K:  gpt51InputPrice,
		OutputPricePer1K: gpt51OutputPrice,
	},
	"gpt-5.1-mini": {
		InputPricePer1K:  gpt51MiniInputPrice,
		OutputPricePer1K: gpt51MiniOutputPrice,
	},
	// GPT-4 models
	"gpt-4o": {
		InputPricePer1K:  gpt4oInputPrice,
		OutputPricePer1K: gpt4oOutputPrice,
	},
	"gpt-4o-mini": {
		InputPricePer1K:  gpt4oMiniInputPrice,
		OutputPricePer1K: gpt4oMiniOutputPrice,
	},
}

// CalculateOpenAICost calculates the cost in USD for an OpenAI API call
func CalculateOpenAICost(model string, usage responses.ResponseUsage) float64 {
	pricing, exists := PricingTable[model]
	if !exists {
		// Default to GPT-5.1 pricing if model not found
		pricing = PricingTable["gpt-5.1"]
	}

	inputCost := (float64(usage.InputTokens) / tokensPerKilo) * pricing.InputPricePer1K
	outputCost := (float64(usage.OutputTokens) / tokensPerKilo) * pricing.OutputPricePer1K

	// Add reasoning tokens if present
	reasoningCost := 0.0
	if usage.OutputTokensDetails.ReasoningTokens > 0 {
		// Reasoning tokens typically cost the same as input tokens
		reasoningCost = (float64(usage.OutputTokensDetails.ReasoningTokens) / tokensPerKilo) * pricing.InputPricePer1K
	}

	totalCost := inputCost + outputCost + reasoningCost
	return totalCost
}

// FormatCost formats a cost value as a USD string
func FormatCost(cost float64) string {
	// Format with specified precision for precision
	return "$" + formatFloat(cost, costFormatPrecision)
}

// formatFloat formats a float with specified precision using strconv
func formatFloat(f float64, precision int) string {
	return strconv.FormatFloat(f, 'f', precision, 64)
}
