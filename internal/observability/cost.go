package observability

import (
	"strconv"

	"github.com/openai/openai-go/responses"
)

// Pricing constants per 1K tokens (in USD)
const (
	priceGPT51Input      = 0.001   // $0.001 per 1K input tokens
	priceGPT51Output     = 0.003   // $0.003 per 1K output tokens
	priceGPT51MiniInput  = 0.0005  // $0.0005 per 1K input tokens
	priceGPT51MiniOutput = 0.0015  // $0.0015 per 1K output tokens
	priceGPT4oInput      = 0.005   // $0.005 per 1K input tokens
	priceGPT4oOutput     = 0.015   // $0.015 per 1K output tokens
	priceGPT4oMiniInput  = 0.00015 // $0.00015 per 1K input tokens
	priceGPT4oMiniOutput = 0.0006  // $0.0006 per 1K output tokens
	priceGPT35Input      = 0.0005  // $0.0005 per 1K input tokens
	priceGPT35Output     = 0.0015  // $0.0015 per 1K output tokens
	priceGeminiInput     = 0.0005  // $0.0005 per 1K input tokens
	priceGeminiOutput    = 0.0015  // $0.0015 per 1K output tokens
	priceFree            = 0.0     // Free tier
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
		InputPricePer1K:  priceGPT51Input,
		OutputPricePer1K: priceGPT51Output,
	},
	"gpt-5.1-mini": {
		InputPricePer1K:  priceGPT51MiniInput,
		OutputPricePer1K: priceGPT51MiniOutput,
	},
	// GPT-4 models
	"gpt-4o": {
		InputPricePer1K:  priceGPT4oInput,
		OutputPricePer1K: priceGPT4oOutput,
	},
	"gpt-4o-mini": {
		InputPricePer1K:  priceGPT4oMiniInput,
		OutputPricePer1K: priceGPT4oMiniOutput,
	},
	// GPT-3.5 models
	"gpt-3.5-turbo": {
		InputPricePer1K:  priceGPT35Input,
		OutputPricePer1K: priceGPT35Output,
	},
	// Gemini models (example pricing - update with actual rates)
	"gemini-2.0-flash-exp": {
		InputPricePer1K:  priceFree,
		OutputPricePer1K: priceFree,
	},
	"gemini-pro": {
		InputPricePer1K:  priceGeminiInput,
		OutputPricePer1K: priceGeminiOutput,
	},
}

// CalculateOpenAICost calculates the cost in USD for an OpenAI API call
func CalculateOpenAICost(model string, usage responses.ResponseUsage) float64 {
	pricing, exists := PricingTable[model]
	if !exists {
		// Default to GPT-4 pricing if model not found
		pricing = PricingTable["gpt-4o"]
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

// CalculateGeminiCost calculates the cost in USD for a Gemini API call
func CalculateGeminiCost(model string, inputTokens, outputTokens int64) float64 {
	pricing, exists := PricingTable[model]
	if !exists {
		// Default to gemini-pro pricing if model not found
		pricing = PricingTable["gemini-pro"]
	}

	inputCost := (float64(inputTokens) / 1000.0) * pricing.InputPricePer1K
	outputCost := (float64(outputTokens) / 1000.0) * pricing.OutputPricePer1K

	return inputCost + outputCost
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
