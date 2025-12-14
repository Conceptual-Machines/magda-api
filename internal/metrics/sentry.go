package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

const (
	// HTTP status code threshold for considering a request successful
	successStatusCodeThreshold = http.StatusBadRequest
)

// SentryMetrics handles custom metrics for Sentry
type SentryMetrics struct {
	enabled bool
}

// NewSentryMetrics creates a new Sentry metrics client
func NewSentryMetrics() *SentryMetrics {
	return &SentryMetrics{
		enabled: true, // Always enabled if Sentry is configured
	}
}

// RecordAPIRequest records API request metrics
func (m *SentryMetrics) RecordAPIRequest(ctx context.Context, endpoint string, statusCode int, duration time.Duration) {
	if !m.enabled {
		return
	}

	// Create a span for API request tracking using the request context
	span := sentry.StartSpan(ctx, "api.request")
	defer span.Finish()

	// Set span tags
	span.SetTag("endpoint", endpoint)
	span.SetTag("status_code", fmt.Sprintf("%d", statusCode))
	span.SetTag("success", fmt.Sprintf("%t", statusCode < successStatusCodeThreshold))

	// Set span data
	span.SetData("duration_ms", duration.Milliseconds())
	span.SetData("endpoint", endpoint)
	span.SetData("status_code", statusCode)

	// Set span status based on response
	if statusCode < successStatusCodeThreshold {
		span.Status = sentry.SpanStatusOK
	} else {
		span.Status = sentry.SpanStatusInternalError
	}

	// Set span description
	span.Description = fmt.Sprintf("API Request: %s", endpoint)
}

// RecordTokenUsage records OpenAI token usage metrics
func (m *SentryMetrics) RecordTokenUsage(ctx context.Context, model string, totalTokens, inputTokens, outputTokens, reasoningTokens int) {
	if !m.enabled {
		return
	}

	fmt.Printf("DEBUG: RecordTokenUsage called with - Total: %d, Input: %d, Output: %d, Reasoning: %d\n",
		totalTokens, inputTokens, outputTokens, reasoningTokens)

	// Try adding data directly to the transaction span instead of creating a child span
	if transaction := sentry.TransactionFromContext(ctx); transaction != nil {
		fmt.Printf("DEBUG: Found transaction, adding data to transaction span\n")
		transaction.SetTag("openai.model", model)
		transaction.SetTag("openai.total_tokens", fmt.Sprintf("%d", totalTokens))
		transaction.SetTag("openai.input_tokens", fmt.Sprintf("%d", inputTokens))
		transaction.SetTag("openai.output_tokens", fmt.Sprintf("%d", outputTokens))
		transaction.SetTag("openai.reasoning_tokens", fmt.Sprintf("%d", reasoningTokens))
		transaction.SetData("openai.total_tokens", totalTokens)
		transaction.SetData("openai.input_tokens", inputTokens)
		transaction.SetData("openai.output_tokens", outputTokens)
		transaction.SetData("openai.reasoning_tokens", reasoningTokens)
	} else {
		fmt.Printf("DEBUG: No transaction found in context\n")
	}

	// Also create a child span for detailed tracking
	span := sentry.StartSpan(ctx, "openai.token_usage")
	defer span.Finish()

	// Set span tags and data
	span.SetTag("model", model)
	span.SetTag("total_tokens", fmt.Sprintf("%d", totalTokens))
	span.SetTag("input_tokens", fmt.Sprintf("%d", inputTokens))
	span.SetTag("output_tokens", fmt.Sprintf("%d", outputTokens))
	span.SetTag("reasoning_tokens", fmt.Sprintf("%d", reasoningTokens))

	// Set data fields
	span.SetData("total_tokens", totalTokens)
	span.SetData("input_tokens", inputTokens)
	span.SetData("output_tokens", outputTokens)
	span.SetData("reasoning_tokens", reasoningTokens)

	span.Status = sentry.SpanStatusOK
	span.Description = fmt.Sprintf("Token Usage: %s", model)
}

// RecordMCPUsage records MCP server usage metrics
func (m *SentryMetrics) RecordMCPUsage(used bool, callCount int) {
	if !m.enabled {
		return
	}

	// Create a span for MCP usage tracking
	ctx := context.Background()
	span := sentry.StartSpan(ctx, "mcp.usage")
	defer span.Finish()

	// Set span tags
	span.SetTag("mcp_used", fmt.Sprintf("%t", used))

	// Set span data
	span.SetData("used", used)
	span.SetData("call_count", callCount)

	span.Status = sentry.SpanStatusOK
	span.Description = fmt.Sprintf("MCP Usage: %t", used)
}

// RecordGenerationDuration records generation request duration
func (m *SentryMetrics) RecordGenerationDuration(ctx context.Context, duration time.Duration, success bool) {
	if !m.enabled {
		return
	}

	// Create a span for generation tracking using the request context
	span := sentry.StartSpan(ctx, "generation.request")
	defer span.Finish()

	// Set span tags
	span.SetTag("success", fmt.Sprintf("%t", success))

	// Set span data
	span.SetData("duration_ms", duration.Milliseconds())
	span.SetData("success", success)

	// Set span status
	if success {
		span.Status = sentry.SpanStatusOK
	} else {
		span.Status = sentry.SpanStatusInternalError
	}

	span.Description = fmt.Sprintf("Generation Request: %t", success)
}

// RecordCustomMetric sends a custom metric with arbitrary data
func (m *SentryMetrics) RecordCustomMetric(metricName string, data map[string]interface{}) {
	if !m.enabled {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetTag("metric_type", "custom")
		scope.SetTag("metric_name", metricName)

		scope.SetContext("custom_metric", data)

		sentry.CaptureMessage("Custom Metric: " + metricName)
	})
}

// RecordPerformanceMetric records performance data
func (m *SentryMetrics) RecordPerformanceMetric(operation string, duration time.Duration, metadata map[string]interface{}) {
	if !m.enabled {
		return
	}

	// Use Sentry's performance monitoring
	ctx := context.Background()
	span := sentry.StartSpan(ctx, operation)
	span.Description = operation
	span.SetData("duration_ms", duration.Milliseconds())

	// Add metadata
	for key, value := range metadata {
		span.SetData(key, value)
	}

	span.Finish()
}
