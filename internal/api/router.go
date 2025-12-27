package api

import (
	"github.com/Conceptual-Machines/magda-api/internal/api/handlers"
	"github.com/Conceptual-Machines/magda-api/internal/api/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/gin-gonic/gin"
)

func SetupRouter(cfg *config.Config, version string) *gin.Engine {
	router := gin.New()

	// Recovery middleware (must be first)
	router.Use(middleware.RecoverWithSentry())

	// Sentry middleware for error tracking
	router.Use(middleware.SentryMiddleware())

	// Request tracking and structured logging
	router.Use(middleware.RequestTracking())

	// CORS middleware
	router.Use(middleware.CORS())

	// Serve static files (logo, etc.)
	router.Static("/static", "./static")

	// Health check (no auth required)
	router.GET("/health", handlers.HealthCheck)

	// MCP status endpoint (no auth required)
	router.GET("/mcp/status", handlers.MCPStatus)

	// Metrics endpoint
	metricsHandler := handlers.NewMetricsHandler(version)
	router.GET("/api/metrics", metricsHandler.GetMetrics)

	// Initialize handlers
	magdaHandler := handlers.NewMagdaHandler(cfg)
	jsfxHandler := handlers.NewJSFXHandler(cfg)
	drummerHandler := handlers.NewDrummerHandler(cfg)
	mixHandler := handlers.NewMixHandler(cfg)
	generationHandler := handlers.NewGenerationHandler(cfg)

	// API routes v1 with conditional auth based on AUTH_MODE
	v1 := router.Group("/api/v1")
	v1.Use(getAuthMiddleware(cfg))
	{
		// AIDEAS endpoints - Music generation using arranger agent
		v1.POST("/aideas/generations", generationHandler.Generate)

		// MAGDA endpoints - DAW control using magda-agents
		v1.POST("/chat", magdaHandler.Chat)
		v1.POST("/chat/stream", magdaHandler.ChatStream) // Streaming endpoint
		v1.POST("/dsl/stream", magdaHandler.DSLStream)   // DSL streaming endpoint
		v1.POST("/dsl", magdaHandler.TestDSL)            // DSL parser endpoint

		// MAGDA Plugin endpoints
		v1.POST("/plugins/process", magdaHandler.ProcessPlugins)

		// MAGDA Mix Analysis endpoint
		v1.POST("/mix/analyze", mixHandler.MixAnalyze)
		v1.POST("/mix/analyze/stream", mixHandler.MixAnalyzeStream)

		// JSFX agent endpoint - AI-assisted JSFX effect generation
		v1.POST("/jsfx/generate", jsfxHandler.Generate)
		v1.POST("/jsfx/generate/stream", jsfxHandler.GenerateStream)

		// Drummer agent endpoint
		v1.POST("/drummer/generate", drummerHandler.Generate)
	}

	return router
}

// getAuthMiddleware returns the appropriate auth middleware based on AUTH_MODE
func getAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	switch cfg.AuthMode {
	case "gateway":
		// Trust X-User-* headers from magda-cloud gateway
		return middleware.GatewayAuth()
	case "none":
		// No auth - for self-hosted or local development
		return middleware.NoAuth()
	default:
		// Default to no auth for backward compatibility
		return middleware.NoAuth()
	}
}
