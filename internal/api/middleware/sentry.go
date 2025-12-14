package middleware

import (
	"net/http"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/logger"
	"github.com/Conceptual-Machines/magda-api/internal/metrics"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	httpStatusBadRequest          = http.StatusBadRequest
	httpStatusInternalServerError = http.StatusInternalServerError
	sentryFlushTimeout            = 2 * time.Second
)

// Global metrics instance
var sentryMetrics = metrics.NewSentryMetrics()

// RequestTracking adds request ID and logging to all requests
func RequestTracking() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)

		// Add to response header
		c.Header("X-Request-ID", requestID)

		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Log request completion
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		fields := logger.Fields{
			"request_id":  requestID,
			"duration_ms": duration.Milliseconds(),
			"status_code": statusCode,
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"client_ip":   c.ClientIP(),
		}

		// Log based on status code
		if statusCode >= httpStatusInternalServerError {
			logger.Error("Request failed with server error", nil, fields)
		} else if statusCode >= httpStatusBadRequest {
			logger.Warn("Request failed with client error", fields)
		} else {
			logger.Info("Request completed", fields)
		}

		// Record API metrics in Sentry
		sentryMetrics.RecordAPIRequest(c.Request.Context(), c.Request.URL.Path, statusCode, duration)
	}
}

// SentryMiddleware returns the Sentry middleware with custom configuration
func SentryMiddleware() gin.HandlerFunc {
	return sentrygin.New(sentrygin.Options{
		Repanic:         true,
		WaitForDelivery: false,
		Timeout:         sentryFlushTimeout,
	})
}

// RecoverWithSentry recovers from panics and sends them to Sentry
func RecoverWithSentry() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Capture panic in Sentry
				if hub := sentrygin.GetHubFromContext(c); hub != nil {
					hub.WithScope(func(scope *sentry.Scope) {
						scope.SetRequest(c.Request)
						scope.SetContext("request", map[string]interface{}{
							"request_id": c.GetString("request_id"),
							"method":     c.Request.Method,
							"path":       c.Request.URL.Path,
							"client_ip":  c.ClientIP(),
						})

						if userID, exists := c.Get("user_id"); exists {
							scope.SetUser(sentry.User{
								ID: userID.(string),
							})
						}

						hub.RecoverWithContext(c.Request.Context(), err)
					})
				}

				// Log the panic
				logger.Error("Panic recovered", nil, logger.Fields{
					"request_id": c.GetString("request_id"),
					"error":      err,
					"path":       c.Request.URL.Path,
				})

				// Return 500
				c.JSON(httpStatusInternalServerError, gin.H{
					"error":      "Internal server error",
					"request_id": c.GetString("request_id"),
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}
