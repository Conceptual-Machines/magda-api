package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/Conceptual-Machines/magda-api/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const (
	forwardedProtoHTTPS = "https"
)

type AuthHandler struct {
	db           *gorm.DB
	cfg          *config.Config
	emailService *services.EmailService
}

func NewAuthHandler(db *gorm.DB, cfg *config.Config, emailService *services.EmailService) *AuthHandler {
	return &AuthHandler{
		db:           db,
		cfg:          cfg,
		emailService: emailService,
	}
}

type RegisterRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	Name           string `json:"name"`
	InvitationCode string `json:"invitation_code"` // Optional for whitelisted emails
}

type AcceptInvitationRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Name           string `json:"name" binding:"required"`
	Password       string `json:"password" binding:"required,min=8"`
	InvitationCode string `json:"invitation_code" binding:"required"`
}

type BetaRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	User         models.User `json:"user"`
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresIn    int64       `json:"expires_in"` // seconds
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// JWT Claims
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

const (
	accessTokenDuration  = 1 * time.Hour
	refreshTokenDuration = 7 * 24 * time.Hour
)

// Whitelist of emails that don't need invitation codes
var emailWhitelist = []string{
	"romagnoli.luca@gmail.com",
	"admin@musicalaideas.com",
}

func isEmailWhitelisted(email string) bool {
	for _, whitelisted := range emailWhitelist {
		if email == whitelisted {
			return true
		}
	}
	return false
}

// validateInvitationCode checks if the invitation code is valid (returns nil if email is whitelisted)
func (h *AuthHandler) validateInvitationCode(email, code string) (*models.InvitationCode, error) {
	if isEmailWhitelisted(email) {
		return nil, nil
	}

	if code == "" {
		return nil, fmt.Errorf("invitation code required")
	}

	invitation := &models.InvitationCode{}
	if err := h.db.Where("code = ?", code).First(invitation).Error; err != nil {
		return nil, fmt.Errorf("invalid invitation code")
	}

	if !invitation.IsValid() {
		return nil, fmt.Errorf("invitation code has expired or been fully used")
	}

	return invitation, nil
}

// createUserWithCredits creates a user and their initial credits in a transaction
func (h *AuthHandler) createUserWithCredits(user *models.User, invitation *models.InvitationCode) error {
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create user: %w", err)
	}

	credits := models.UserCredits{
		UserID:  user.ID,
		Credits: models.GetInitialCreditsForRole(user.Role),
	}

	if err := tx.Create(&credits).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create user credits: %w", err)
	}

	if invitation != nil {
		if err := invitation.MarkAsUsed(user.ID, tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to process invitation code: %w", err)
		}
	}

	return tx.Commit().Error
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Validate invitation code
	invitation, validationErr := h.validateInvitationCode(req.Email, req.InvitationCode)
	if validationErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
		return
	}

	// Create new user
	user := models.User{
		Email: req.Email,
		Name:  req.Name,
	}

	if hashErr := user.HashPassword(req.Password); hashErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user with credits in transaction
	if createErr := h.createUserWithCredits(&user, invitation); createErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": createErr.Error()})
		return
	}

	// Generate and send verification email
	token, err := h.emailService.GenerateVerificationToken(user.ID)
	if err != nil {
		// Log error but don't fail registration
		fmt.Printf("Failed to generate verification token: %v\n", err)
	} else {
		if sendErr := h.emailService.SendVerificationEmail(&user, token); sendErr != nil {
			// Log error but don't fail registration
			fmt.Printf("Failed to send verification email: %v\n", sendErr)
		}
	}

	// Generate tokens
	accessToken, err := h.generateAccessToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.generateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Set HTTP-only cookies for web users (domain with leading dot works across all subdomains)
	// Use secure flag if HTTPS (required for cookie clearing to work properly)
	isHTTPS := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == forwardedProtoHTTPS
	c.SetCookie("access_token", accessToken, int(accessTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)
	c.SetCookie("refresh_token", refreshToken, int(refreshTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)

	c.JSON(http.StatusCreated, gin.H{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int64(accessTokenDuration.Seconds()),
		"message":       "Registration successful! Please check your email to verify your account.",
	})
}

// RegisterBeta creates a new beta user account with unlimited credits
func (h *AuthHandler) RegisterBeta(c *gin.Context) {
	var req BetaRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Create new user with beta role (auto-verified)
	now := time.Now()
	user := models.User{
		Email:         req.Email,
		Name:          req.Name,
		Role:          models.RoleBeta, // Auto-assign beta role
		IsActive:      true,
		EmailVerified: true, // Beta users are auto-verified
		VerifiedAt:    &now,
	}

	if err := user.HashPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Start transaction
	tx := h.db.Begin()

	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Create initial credits for beta user (won't be deducted anyway)
	credits := models.UserCredits{
		UserID:  user.ID,
		Credits: models.BetaInitialCredits,
	}

	if err := tx.Create(&credits).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user credits"})
		return
	}

	tx.Commit()

	// Generate tokens
	accessToken, err := h.generateAccessToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.generateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Set HTTP-only cookies for web users (domain with leading dot works across all subdomains)
	// Use secure flag if HTTPS (required for cookie clearing to work properly)
	isHTTPS := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == forwardedProtoHTTPS
	c.SetCookie("access_token", accessToken, int(accessTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)
	c.SetCookie("refresh_token", refreshToken, int(refreshTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)

	c.JSON(http.StatusCreated, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"role":       user.Role,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
		},
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int64(accessTokenDuration.Seconds()),
		"message":       "🎉 Welcome to AIDEAS Beta! You have unlimited generations.",
	})
}

// Login authenticates a user and returns tokens
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is disabled"})
		return
	}

	// Verify password
	if !user.CheckPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate tokens
	accessToken, err := h.generateAccessToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.generateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Set HTTP-only cookies for web users (domain with leading dot works across all subdomains)
	// Use secure flag if HTTPS (required for cookie clearing to work properly)
	isHTTPS := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == forwardedProtoHTTPS
	c.SetCookie("access_token", accessToken, int(accessTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)
	c.SetCookie("refresh_token", refreshToken, int(refreshTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)

	c.JSON(http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTokenDuration.Seconds()),
	})
}

// Refresh generates new tokens using a refresh token
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse and validate refresh token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(h.cfg.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired refresh token"})
		return
	}

	// Find user
	var user models.User
	if dbErr := h.db.First(&user, claims.UserID).Error; dbErr != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Check if user is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is disabled"})
		return
	}

	// Generate new tokens
	accessToken, err := h.generateAccessToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	newRefreshToken, err := h.generateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(accessTokenDuration.Seconds()),
	})
}

// Logout clears authentication cookies
func (h *AuthHandler) Logout(c *gin.Context) {
	// Clear cookies with BOTH secure flags to handle transition period
	// (Old cookies were set with secure=false, new ones with secure=true)

	// Clear with secure=false (for old cookies)
	c.SetCookie("access_token", "", -1, "/", ".musicalaideas.com", false, true)
	c.SetCookie("refresh_token", "", -1, "/", ".musicalaideas.com", false, true)

	// Clear with secure=true (for new cookies on HTTPS)
	c.SetCookie("access_token", "", -1, "/", ".musicalaideas.com", true, true)
	c.SetCookie("refresh_token", "", -1, "/", ".musicalaideas.com", true, true)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// generateAccessToken creates a new access token
func (h *AuthHandler) generateAccessToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "aideas-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}

// generateRefreshToken creates a new refresh token
func (h *AuthHandler) generateRefreshToken(user *models.User) (string, error) {
	claims := &Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "aideas-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}

// VerifyEmail verifies a user's email address
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification token is required"})
		return
	}

	if err := h.emailService.VerifyEmail(token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email verified successfully! You can now use all features.",
	})
}

// ResendVerification resends the verification email
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	if err := h.emailService.ResendVerificationEmail(email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Verification email sent! Please check your inbox.",
	})
}

// AcceptInvitation creates an account using an invitation link
func (h *AuthHandler) AcceptInvitation(c *gin.Context) {
	var req AcceptInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	var existingUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Validate invitation code (required for accept-invitation flow)
	invitation := &models.InvitationCode{}
	if err := h.db.Where("code = ?", req.InvitationCode).First(invitation).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation code"})
		return
	}

	if !invitation.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation code has expired or been fully used"})
		return
	}

	// Create new user with beta role (auto-verified since they used invitation)
	user := models.User{
		Email:         req.Email,
		Name:          req.Name,
		Role:          models.RoleBeta,
		IsActive:      true,
		EmailVerified: true,
	}
	now := time.Now()
	user.VerifiedAt = &now

	if err := user.HashPassword(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user with credits in transaction
	if err := h.createUserWithCredits(&user, invitation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Generate tokens
	accessToken, err := h.generateAccessToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate access token"})
		return
	}

	refreshToken, err := h.generateRefreshToken(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	// Set HTTP-only cookies
	isHTTPS := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == forwardedProtoHTTPS
	c.SetCookie("access_token", accessToken, int(accessTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)
	c.SetCookie("refresh_token", refreshToken, int(refreshTokenDuration.Seconds()), "/", ".musicalaideas.com", isHTTPS, true)

	c.JSON(http.StatusCreated, gin.H{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"expires_in":    int64(accessTokenDuration.Seconds()),
		"message":       "Account created successfully! Welcome to Musical AIDEAS.",
	})
}
