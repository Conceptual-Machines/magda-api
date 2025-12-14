package handlers

import (
	"fmt"
	"net/http"

	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
	"gorm.io/gorm"
)

type OAuthHandler struct {
	db  *gorm.DB
	cfg *config.Config
}

func NewOAuthHandler(db *gorm.DB, cfg *config.Config) *OAuthHandler {
	// Initialize gothic session store
	store := sessions.NewCookieStore([]byte(cfg.JWTSecret))
	store.Options.HttpOnly = true
	store.Options.Secure = cfg.Environment == "production" // Use secure cookies in production
	gothic.Store = store

	// Configure OAuth providers
	goth.UseProviders(
		google.New(
			cfg.GoogleClientID,
			cfg.GoogleClientSecret,
			cfg.BaseURL+"/api/auth/google/callback",
			"email", "profile",
		),
		github.New(
			cfg.GitHubClientID,
			cfg.GitHubClientSecret,
			cfg.BaseURL+"/api/auth/github/callback",
			"user:email",
		),
	)

	return &OAuthHandler{db: db, cfg: cfg}
}

// BeginAuth redirects user to OAuth provider login
func (h *OAuthHandler) BeginAuth(c *gin.Context) {
	provider := c.Param("provider")

	// Validate provider
	if provider != providerGoogle && provider != providerGitHub {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
		return
	}

	// Set provider in query param for gothic
	q := c.Request.URL.Query()
	q.Add("provider", provider)
	c.Request.URL.RawQuery = q.Encode()

	// Get auth URL
	gothic.BeginAuthHandler(c.Writer, c.Request)
}

// Callback handles OAuth provider callback
func (h *OAuthHandler) Callback(c *gin.Context) {
	provider := c.Param("provider")

	// Validate provider
	if provider != providerGoogle && provider != providerGitHub {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported provider"})
		return
	}

	// Set provider in query param for gothic
	q := c.Request.URL.Query()
	q.Add("provider", provider)
	c.Request.URL.RawQuery = q.Encode()

	// Complete OAuth flow
	gothUser, err := gothic.CompleteUserAuth(c.Writer, c.Request)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OAuth authentication failed"})
		return
	}

	// Find or create user
	user, isNew, err := h.findOrCreateOAuthUser(&gothUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate tokens
	authHandler := &AuthHandler{db: h.db, cfg: h.cfg}
	accessToken, err := authHandler.generateAccessToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := authHandler.generateRefreshToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Set HTTP-only cookies for web users (domain with leading dot works across all subdomains)
	// Use secure flag if HTTPS (required for cookie clearing to work properly)
	isHTTPS := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == forwardedProtoHTTPS
	c.SetCookie("access_token", accessToken, int(accessTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)
	c.SetCookie("refresh_token", refreshToken, int(refreshTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)

	// Redirect to web callback page with tokens (for plugin users to copy)
	redirectURL := fmt.Sprintf("/auth/callback?access_token=%s&refresh_token=%s&is_new=%v",
		accessToken, refreshToken, isNew)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// findOrCreateOAuthUser finds an existing OAuth user or creates a new one
func (h *OAuthHandler) findOrCreateOAuthUser(gothUser *goth.User) (*models.User, bool, error) {
	var oauthProvider models.OAuthProvider

	// Try to find existing OAuth provider record
	err := h.db.Where("provider = ? AND provider_user_id = ?",
		gothUser.Provider, gothUser.UserID).
		Preload("User").
		First(&oauthProvider).Error

	if err == nil {
		// User exists
		return &oauthProvider.User, false, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	// User doesn't exist, create new user
	return h.createOAuthUser(gothUser)
}

// createOAuthUser creates a new user from OAuth data
func (h *OAuthHandler) createOAuthUser(gothUser *goth.User) (*models.User, bool, error) {
	// Start transaction
	tx := h.db.Begin()

	// Check if email already exists (user signed up with email/password)
	var existingUser models.User
	emailExists := tx.Where("email = ?", gothUser.Email).First(&existingUser).Error == nil

	if emailExists {
		// Link OAuth to existing user
		oauthProvider := models.OAuthProvider{
			UserID:         existingUser.ID,
			Provider:       gothUser.Provider,
			ProviderUserID: gothUser.UserID,
		}

		if err := tx.Create(&oauthProvider).Error; err != nil {
			tx.Rollback()
			return nil, false, err
		}

		tx.Commit()
		return &existingUser, false, nil
	}

	// Create new user
	user := models.User{
		Email:    gothUser.Email,
		Name:     gothUser.Name,
		IsActive: true,
		// No password needed for OAuth users
		Password: "", // Will be hashed to empty string
	}

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		return nil, false, err
	}

	// Create OAuth provider record
	oauthProvider := models.OAuthProvider{
		UserID:         user.ID,
		Provider:       gothUser.Provider,
		ProviderUserID: gothUser.UserID,
	}

	if err := tx.Create(&oauthProvider).Error; err != nil {
		tx.Rollback()
		return nil, false, err
	}

	// Create initial credits based on user role
	credits := models.UserCredits{
		UserID:  user.ID,
		Credits: models.GetInitialCreditsForRole(user.Role),
	}

	if err := tx.Create(&credits).Error; err != nil {
		tx.Rollback()
		return nil, false, err
	}

	tx.Commit()
	return &user, true, nil
}
