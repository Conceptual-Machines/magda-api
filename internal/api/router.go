package api

import (
	"github.com/Conceptual-Machines/magda-api/internal/api/handlers"
	apimiddleware "github.com/Conceptual-Machines/magda-api/internal/api/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/services"
	webhandlers "github.com/Conceptual-Machines/magda-api/internal/web/handlers"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRouter(db *gorm.DB, cfg *config.Config, version string) *gin.Engine {
	router := gin.New()

	// Recovery middleware (must be first)
	router.Use(apimiddleware.RecoverWithSentry())

	// Sentry middleware for error tracking
	router.Use(apimiddleware.SentryMiddleware())

	// Request tracking and structured logging
	router.Use(apimiddleware.RequestTracking())

	// CORS middleware
	router.Use(apimiddleware.CORS())

	// Serve static files (logo, etc.)
	router.Static("/static", "./static")

	// Health check
	healthHandler := handlers.NewHealthHandler(db)
	router.GET("/health", healthHandler.HealthCheck)

	// MCP status endpoint
	router.GET("/mcp/status", handlers.MCPStatus)

	// Bootstrap endpoint (one-time admin setup)
	bootstrapHandler := handlers.NewBootstrapHandler(db)
	router.POST("/api/bootstrap/set-admin", bootstrapHandler.SetAdminRole)
	router.POST("/api/bootstrap/cleanup-users", bootstrapHandler.CleanupUsers) // TEMPORARY - remove after use

	// Metrics endpoint
	metricsHandler := handlers.NewMetricsHandler(version)
	router.GET("/api/metrics", metricsHandler.GetMetrics)

	// Web pages
	webHandler := webhandlers.NewWebHandler(db)
	router.GET("/", middleware.OptionalJWTAuth(db, cfg), webHandler.Home) // Coming soon/beta signup or redirect to dashboard if logged in
	router.GET("/login", webHandler.Login)                                // Login page
	router.GET("/beta", webHandler.BetaSignup)                            // Beta signup page
	router.GET("/signup/beta", webHandler.BetaSignup)                     // Alternative URL
	router.GET("/auth/callback", webHandler.OAuthCallback)
	router.GET("/auth/accept-invitation", webHandler.AcceptInvitationPage) // Accept invitation page
	router.GET("/verify-email", webHandler.VerifyEmailPage)                // Email verification page

	// Dashboard (protected)
	router.GET("/dashboard", middleware.OptionalJWTAuth(db, cfg), webHandler.Dashboard)

	// Admin Panel (admin only)
	router.GET("/admin", middleware.JWTAuth(db, cfg), middleware.AdminRequired(), webHandler.AdminPanel)
	router.GET("/admin/invitations", middleware.JWTAuth(db, cfg), middleware.AdminRequired(), webHandler.InvitationsPanel)

	// HTMX endpoints (protected)
	router.GET("/htmx/usage-history", middleware.OptionalJWTAuth(db, cfg), webHandler.UsageHistoryTable)

	// Initialize email service for all handlers
	emailService := services.NewEmailService(db, cfg)

	// Auth routes (public)
	auth := router.Group("/api/auth")
	{
		authHandler := handlers.NewAuthHandler(db, cfg, emailService)
		auth.POST("/register", authHandler.Register)
		auth.POST("/register/beta", authHandler.RegisterBeta)         // Beta signup with unlimited credits
		auth.POST("/accept-invitation", authHandler.AcceptInvitation) // Accept invitation and create account
		auth.POST("/login", authHandler.Login)
		auth.POST("/logout", authHandler.Logout) // Logout (clears cookies)
		auth.POST("/refresh", authHandler.Refresh)
		auth.GET("/verify-email", authHandler.VerifyEmail)                // Email verification
		auth.POST("/resend-verification", authHandler.ResendVerification) // Resend verification email

		// OAuth routes
		oauthHandler := handlers.NewOAuthHandler(db, cfg)
		auth.GET("/:provider", oauthHandler.BeginAuth)         // /api/auth/google or /api/auth/github
		auth.GET("/:provider/callback", oauthHandler.Callback) // OAuth callback
	}

	// Protected API routes v1 (require JWT)
	v1 := router.Group("/api/v1")
	v1.Use(middleware.JWTAuth(db, cfg))
	{
		// AIDEAS endpoints - Music generation using arranger agent
		aideasHandler := handlers.NewGenerationHandler(cfg, db)
		v1.POST("/aideas/generations", aideasHandler.Generate)

		// MAGDA endpoints - DAW control using magda-agents
		magdaHandler := handlers.NewMagdaHandler(cfg, db)
		v1.POST("/chat", magdaHandler.Chat)
		v1.POST("/chat/stream", magdaHandler.ChatStream) // Experimental streaming endpoint
		v1.POST("/dsl/stream", magdaHandler.DSLStream)   // DSL streaming endpoint (explicit DSL mode)
		v1.POST("/dsl", magdaHandler.TestDSL)            // DSL parser endpoint

		// MAGDA Plugin endpoints
		v1.POST("/plugins/process", magdaHandler.ProcessPlugins) // Deduplicate plugins and generate aliases

		// MAGDA Mix Analysis endpoint
		mixHandler := handlers.NewMixHandler(cfg, db)
		v1.POST("/mix/analyze", mixHandler.MixAnalyze) // Mix analysis endpoint

		// Drummer agent endpoint
		drummerHandler := handlers.NewDrummerHandler(cfg, db)
		v1.POST("/drummer/generate", drummerHandler.Generate)

		// User/dashboard endpoints
		userHandler := handlers.NewUserHandler(db)
		v1.GET("/me", userHandler.GetProfile)
		v1.GET("/credits", userHandler.GetCredits)
		v1.GET("/usage/stats", userHandler.GetUsageStats)
		v1.GET("/usage/history", userHandler.GetUsageHistory)
	}

	// Admin API routes (admin only)
	admin := router.Group("/api/admin")
	admin.Use(middleware.JWTAuth(db, cfg), middleware.AdminRequired())
	{
		adminHandler := handlers.NewAdminHandler(db)
		admin.GET("/users", adminHandler.ListUsers)
		admin.GET("/users/:id", adminHandler.GetUserDetails)
		admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
		admin.PUT("/users/:id/toggle-active", adminHandler.ToggleUserActive)
		admin.PUT("/users/:id/credits", adminHandler.UpdateUserCredits)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)

		// Invitation management
		invitationHandler := handlers.NewInvitationHandler(db, emailService)
		admin.POST("/invitations", invitationHandler.CreateInvitation)
		admin.POST("/invitations/send", invitationHandler.SendInvitation)
		admin.POST("/invitations/:id/resend", invitationHandler.ResendInvitation)
		admin.GET("/invitations", invitationHandler.ListInvitations)
		admin.GET("/invitations/stats", invitationHandler.GetInvitationStats)
		admin.DELETE("/invitations/:id", invitationHandler.DeleteInvitation)
	}

	return router
}
