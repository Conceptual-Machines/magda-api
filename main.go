package main

import (
	"log"
	"os"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/api"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/database"
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

	// Initialize Sentry
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.Environment,
			Release:          "magda-api@" + releaseVersion,            // Use embedded release version
			EnableTracing:    true,                                     // Enable tracing for spans
			TracesSampleRate: 1.0,                                      // 100% sampling for now, adjust based on volume
			EnableLogs:       true,                                     // Enable Sentry Logs feature
			Debug:            cfg.Environment != environmentProduction, // Enable debug in non-prod
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
			// Flush on shutdown
			defer sentry.Flush(sentryFlushTimeout)
		}
	} else {
		log.Println("⚠️  Sentry not configured (SENTRY_DSN not set)")
	}

	// Initialize database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatal("Failed to connect to database:", err)
	}

	// Run migrations
	if err := database.Migrate(db); err != nil {
		sentry.CaptureException(err)
		log.Fatal("Failed to run migrations:", err)
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize router
	router := api.SetupRouter(db, cfg, GetVersion())

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Starting server on port %s", port)
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
