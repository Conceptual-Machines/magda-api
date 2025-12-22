package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/api"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/observability"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const (
	sentryFlushTimeout    = 2 * time.Second
	environmentProduction = "production"
)

// releaseVersion is set via ldflags during build
var releaseVersion = "dev"

// GetVersion returns the current release version
func GetVersion() string {
	return releaseVersion
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize Sentry (optional)
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.Environment,
			Release:          "magda-api@" + releaseVersion,
			EnableTracing:    true,
			TracesSampleRate: 1.0,
			EnableLogs:       true,
			Debug:            cfg.Environment != environmentProduction,
			BeforeSend: func(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
				// Filter out sensitive data
				if event.Request != nil {
					event.Request.Headers = filterSensitiveHeaders(event.Request.Headers)
				}
				return event
			},
		}); err != nil {
			log.Printf("Failed to initialize Sentry: %v", err)
		} else {
			log.Printf("✅ Sentry initialized (environment: %s, release: %s)", cfg.Environment, releaseVersion)
			defer sentry.Flush(sentryFlushTimeout)
		}
	} else {
		log.Println("⚠️  Sentry not configured (SENTRY_DSN not set)")
	}

	// Initialize Langfuse for LLM observability (optional)
	if cfg.LangfuseEnabled && cfg.LangfuseSecretKey != "" {
		os.Setenv("LANGFUSE_PUBLIC_KEY", cfg.LangfusePublicKey)
		os.Setenv("LANGFUSE_SECRET_KEY", cfg.LangfuseSecretKey)
		if cfg.LangfuseHost != "" {
			os.Setenv("LANGFUSE_HOST", cfg.LangfuseHost)
		}
	}
	observability.InitializeLangfuse(context.Background(), cfg)

	// Log auth mode
	log.Printf("🔐 Auth mode: %s", cfg.AuthMode)

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router (no database needed)
	router := api.SetupRouter(cfg, GetVersion())

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Starting magda-api on port %s", port)
	if err := router.Run(":" + port); err != nil {
		sentry.CaptureException(err)
		log.Fatal("Failed to start server:", err)
	}
}

func filterSensitiveHeaders(headers map[string]string) map[string]string {
	filtered := make(map[string]string)
	sensitiveKeys := map[string]bool{
		"authorization": true,
		"cookie":        true,
		"x-api-key":     true,
	}

	for k, v := range headers {
		if sensitiveKeys[k] {
			filtered[k] = "[REDACTED]"
		} else {
			filtered[k] = v
		}
	}
	return filtered
}
