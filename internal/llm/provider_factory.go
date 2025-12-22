package llm

import (
	"context"
	"fmt"
	"strings"
)

// ProviderFactory creates providers based on model name
type ProviderFactory struct {
	openaiAPIKey string
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory(openaiAPIKey string) *ProviderFactory {
	return &ProviderFactory{
		openaiAPIKey: openaiAPIKey,
	}
}

// GetProvider returns the appropriate provider for the given model
func (f *ProviderFactory) GetProvider(ctx context.Context, model string) (Provider, error) {
	return f.getProviderByModel(ctx, model)
}

// getProviderByModel infers provider from model name
func (f *ProviderFactory) getProviderByModel(_ context.Context, model string) (Provider, error) {
	modelLower := strings.ToLower(model)

	// GPT models use OpenAI
	if strings.HasPrefix(modelLower, "gpt-") {
		if f.openaiAPIKey == "" {
			return nil, fmt.Errorf("openai API key not configured")
		}
		return NewOpenAIProvider(f.openaiAPIKey), nil
	}

	// Default to OpenAI for unknown models
	if f.openaiAPIKey == "" {
		return nil, fmt.Errorf("openai API key not configured (default provider)")
	}
	return NewOpenAIProvider(f.openaiAPIKey), nil
}
