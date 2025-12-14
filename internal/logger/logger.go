package logger

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// Fields represents structured log fields
type Fields map[string]interface{}

// WithContext extracts request context for logging
func WithContext(c *gin.Context) Fields {
	fields := Fields{
		"request_id": c.GetString("request_id"),
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
	}

	if userID, exists := c.Get("user_id"); exists {
		fields["user_id"] = userID
	}

	return fields
}

// Info logs an informational message with structured fields
func Info(msg string, fields Fields) {
	log.Printf("[INFO] %s %v", msg, formatFields(fields))

	// Send to Sentry as breadcrumb
	if hub := sentry.CurrentHub(); hub.Client() != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Type:     "info",
			Category: "log",
			Message:  msg,
			Data:     convertFieldsToMap(fields),
			Level:    sentry.LevelInfo,
		})
	}
}

// Error logs an error message with structured fields and sends to Sentry
func Error(msg string, err error, fields Fields) {
	log.Printf("[ERROR] %s: %v %v", msg, err, formatFields(fields))

	// Send to Sentry
	if hub := sentry.CurrentHub(); hub.Client() != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			// Add structured fields as context
			for key, value := range fields {
				scope.SetContext(key, map[string]interface{}{
					"value": value,
				})
			}

			// Set tags for better filtering in Sentry
			if requestID, ok := fields["request_id"].(string); ok {
				scope.SetTag("request_id", requestID)
			}
			if model, ok := fields["model"].(string); ok {
				scope.SetTag("model", model)
			}

			hub.CaptureException(err)
		})
	}
}

// Warn logs a warning message with structured fields
func Warn(msg string, fields Fields) {
	log.Printf("[WARN] %s %v", msg, formatFields(fields))

	// Send to Sentry as breadcrumb
	if hub := sentry.CurrentHub(); hub.Client() != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Type:     "warning",
			Category: "log",
			Message:  msg,
			Data:     convertFieldsToMap(fields),
			Level:    sentry.LevelWarning,
		})
	}
}

// Debug logs a debug message with structured fields
func Debug(msg string, fields Fields) {
	log.Printf("[DEBUG] %s %v", msg, formatFields(fields))

	// Send to Sentry as breadcrumb (only in development/debug mode)
	if hub := sentry.CurrentHub(); hub.Client() != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Type:     "debug",
			Category: "log",
			Message:  msg,
			Data:     convertFieldsToMap(fields),
			Level:    sentry.LevelDebug,
		})
	}
}

// LogAPIRequest logs API request metrics
func LogAPIRequest(c *gin.Context, duration time.Duration, statusCode int, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}

	fields["duration_ms"] = duration.Milliseconds()
	fields["status_code"] = statusCode
	fields["request_id"] = c.GetString("request_id")
	fields["method"] = c.Request.Method
	fields["path"] = c.Request.URL.Path
	fields["client_ip"] = c.ClientIP()

	Info("API request completed", fields)

	// Add breadcrumb to Sentry
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Type:     "http",
		Category: "api",
		Message:  "API request",
		Data:     convertFieldsToMap(fields),
		Level:    sentry.LevelInfo,
	})
}

// LogGenerationRequest logs generation request metrics
func LogGenerationRequest(ctx context.Context, model string, duration time.Duration, tokenUsage map[string]interface{}, fields Fields) {
	if fields == nil {
		fields = Fields{}
	}

	fields["model"] = model
	fields["duration_ms"] = duration.Milliseconds()
	fields["total_tokens"] = tokenUsage["total_tokens"]
	fields["input_tokens"] = tokenUsage["input_tokens"]
	fields["output_tokens"] = tokenUsage["output_tokens"]

	Info("Generation request completed", fields)

	// Track performance in Sentry
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		span := sentry.StartSpan(ctx, "openai.generate")
		span.Description = model
		span.SetData("tokens", tokenUsage)
		span.Finish()
	}
}

// formatFields converts Fields to a readable string
func formatFields(fields Fields) string {
	if len(fields) == 0 {
		return ""
	}
	// Simple formatting - could use JSON for production
	result := "{"
	first := true
	for k, v := range fields {
		if !first {
			result += ", "
		}
		result += k + "="
		switch val := v.(type) {
		case string:
			result += val
		case int, int64, float64:
			result += formatValue(val)
		default:
			result += formatValue(v)
		}
		first = false
	}
	result += "}"
	return result
}

// LogToSentry sends a log message directly to Sentry as an event
func LogToSentry(level sentry.Level, msg string, fields Fields) {
	if hub := sentry.CurrentHub(); hub.Client() != nil {
		hub.WithScope(func(scope *sentry.Scope) {
			// Set the log level
			scope.SetLevel(level)

			// Add structured fields as context
			for key, value := range fields {
				scope.SetContext(key, map[string]interface{}{
					"value": value,
				})
			}

			// Set tags for better filtering
			if requestID, ok := fields["request_id"].(string); ok {
				scope.SetTag("request_id", requestID)
			}
			if model, ok := fields["model"].(string); ok {
				scope.SetTag("model", model)
			}

			// Send as message event
			hub.CaptureMessage(msg)
		})
	}
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case float64:
		return fmt.Sprintf("%.2f", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func convertFieldsToMap(fields Fields) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range fields {
		result[k] = v
	}
	return result
}
