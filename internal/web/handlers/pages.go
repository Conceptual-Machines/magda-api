package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/Conceptual-Machines/magda-api/internal/services"
	"github.com/Conceptual-Machines/magda-api/internal/web/templates"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	maxRecentUsageLogs = 10
)

type WebHandler struct {
	db             *gorm.DB
	creditsService *services.CreditsService
}

func NewWebHandler(db *gorm.DB) *WebHandler {
	return &WebHandler{
		db:             db,
		creditsService: services.NewCreditsService(db),
	}
}

// Home renders different pages based on the domain and login status
func (h *WebHandler) Home(c *gin.Context) {
	// Check if user is logged in - if so, redirect to dashboard
	if user, exists := middleware.GetCurrentUser(c); exists && user.ID > 0 {
		c.Redirect(http.StatusTemporaryRedirect, "/dashboard")
		return
	}

	host := c.Request.Host

	// Show beta signup on beta subdomain
	if host == "beta.musicalaideas.com" || host == "beta.musicalaideas.com:8080" {
		component := templates.BetaSignup()
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
		}
		return
	}

	// Show coming soon on main domain
	component := templates.ComingSoon()
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// Register renders the registration page
func (h *WebHandler) Register(c *gin.Context) {
	component := templates.Register()
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// BetaSignup renders the beta signup page
func (h *WebHandler) BetaSignup(c *gin.Context) {
	component := templates.BetaSignup()
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// Login renders the login page
func (h *WebHandler) Login(c *gin.Context) {
	component := templates.Login()
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// Dashboard renders the user dashboard
func (h *WebHandler) Dashboard(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	// Get user credits
	credits, err := h.creditsService.GetUserCredits(user.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load credits")
		return
	}

	// Get usage stats (all time - don't filter by date)
	stats, err := h.creditsService.GetUserUsageStats(user.ID, time.Time{}, time.Time{})
	if err != nil {
		// Log the error but use empty stats instead of failing
		log.Printf("Dashboard stats error for user %d: %v", user.ID, err)
		c.Header("X-Stats-Error", err.Error())
		stats = &services.UsageStats{}
	} else {
		log.Printf("Dashboard stats for user %d: requests=%d, credits=%d, tokens=%d",
			user.ID, stats.TotalRequests, stats.TotalCreditsUsed, stats.TotalTokensUsed)
	}

	data := templates.DashboardData{
		Email:            user.Email,
		Role:             user.Role,
		Credits:          credits.Credits,
		TotalRequests:    stats.TotalRequests,
		TotalTokens:      stats.TotalTokensUsed,
		TotalCreditsUsed: stats.TotalCreditsUsed,
		AvgDurationMS:    stats.AvgDurationMS,
	}

	component := templates.Dashboard(data)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

const (
	roleAdmin        = "admin"
	activeStatusTrue = "true"
)

// AdminPanel renders the admin dashboard
func (h *WebHandler) AdminPanel(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	if user.Role != roleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Get all users with their credits
	var users []models.User
	query := h.db.Model(&models.User{})

	// Apply filters if provided
	if role := c.Query("role"); role != "" {
		query = query.Where("role = ?", role)
	}
	if isActive := c.Query("is_active"); isActive != "" {
		query = query.Where("is_active = ?", isActive == activeStatusTrue)
	}

	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to load users")
		return
	}

	// Build user data with credits
	var userData []templates.UserData
	for _, u := range users {
		var credits models.UserCredits
		h.db.Where("user_id = ?", u.ID).First(&credits)

		userData = append(userData, templates.UserData{
			ID:            u.ID,
			Email:         u.Email,
			Name:          u.Name,
			Role:          u.Role,
			IsActive:      u.IsActive,
			EmailVerified: u.EmailVerified,
			Credits:       credits.Credits,
			CreatedAt:     u.CreatedAt.Format("2006-01-02"),
		})
	}

	data := templates.AdminPanelData{
		Email: user.Email,
		Users: userData,
	}

	component := templates.AdminPanel(data)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// InvitationsPanel renders the invitations management page
func (h *WebHandler) InvitationsPanel(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return
	}

	if user.Role != roleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
		return
	}

	// Get all invitations
	var invitations []models.InvitationCode
	if err := h.db.Preload("CreatedBy").Preload("UsedBy").Order("created_at DESC").Find(&invitations).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to load invitations")
		return
	}

	// Build invitation data
	var invitationData []templates.InvitationData
	for _, inv := range invitations {
		data := templates.InvitationData{
			ID:          inv.ID,
			Code:        inv.Code,
			Note:        inv.Note,
			MaxUses:     inv.MaxUses,
			CurrentUses: inv.CurrentUses,
			CreatedAt:   inv.CreatedAt.Format("2006-01-02 15:04"),
			IsValid:     inv.IsValid(),
		}

		if inv.ExpiresAt != nil {
			data.ExpiresAt = inv.ExpiresAt.Format("2006-01-02 15:04")
		}

		if inv.CreatedBy != nil {
			data.CreatedByEmail = inv.CreatedBy.Email
		}

		if inv.UsedBy != nil {
			data.UsedByEmail = inv.UsedBy.Email
		}

		invitationData = append(invitationData, data)
	}

	// Get stats
	var total, used, unused, expired int64
	h.db.Model(&models.InvitationCode{}).Count(&total)
	h.db.Model(&models.InvitationCode{}).Where("current_uses >= max_uses").Count(&used)
	h.db.Model(&models.InvitationCode{}).Where("current_uses < max_uses").Count(&unused)
	h.db.Model(&models.InvitationCode{}).Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Count(&expired)

	panelData := templates.InvitationsPanelData{
		Email:       user.Email,
		Invitations: invitationData,
	}
	panelData.Stats.Total = int(total)
	panelData.Stats.Used = int(used)
	panelData.Stats.Unused = int(unused)
	panelData.Stats.Expired = int(expired)

	component := templates.InvitationsPanel(panelData)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// AcceptInvitationPage renders the accept invitation page
func (h *WebHandler) AcceptInvitationPage(c *gin.Context) {
	email := c.Query("email")
	code := c.Query("code")
	errorMsg := c.Query("error")

	if email == "" || code == "" {
		c.String(http.StatusBadRequest, "Missing email or invitation code")
		return
	}

	// Validate invitation code exists
	var invitation models.InvitationCode
	if err := h.db.Where("code = ?", code).First(&invitation).Error; err != nil {
		c.String(http.StatusNotFound, "Invalid invitation code")
		return
	}

	if !invitation.IsValid() {
		c.String(http.StatusGone, "This invitation code has expired or been fully used")
		return
	}

	component := templates.AcceptInvitation(email, code, errorMsg)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// OAuthCallback renders the OAuth success page
func (h *WebHandler) OAuthCallback(c *gin.Context) {
	accessToken := c.Query("access_token")
	refreshToken := c.Query("refresh_token")
	isNew := c.Query("is_new") == "true"

	if accessToken == "" || refreshToken == "" {
		c.String(http.StatusBadRequest, "Missing tokens")
		return
	}

	component := templates.OAuthCallback(accessToken, refreshToken, isNew)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// UsageHistoryTable returns HTMX-compatible usage history table
func (h *WebHandler) UsageHistoryTable(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.String(http.StatusUnauthorized, "Please log in")
		return
	}

	// Get usage logs from database - only fields displayed to user
	var logs []struct {
		CreatedAt      time.Time
		CreditsCharged int
		DurationMS     int
	}

	if err := h.db.Table("usage_logs").
		Select("created_at, credits_charged, duration_ms").
		Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Limit(maxRecentUsageLogs).
		Find(&logs).Error; err != nil {
		c.String(http.StatusInternalServerError, "Failed to load usage history")
		return
	}

	// Convert to template format
	var items []templates.UsageHistoryItem
	for _, log := range logs {
		items = append(items, templates.UsageHistoryItem{
			CreatedAt:      log.CreatedAt,
			CreditsCharged: log.CreditsCharged,
			DurationMS:     log.DurationMS,
		})
	}

	// Hide credits column for unlimited users (admin/beta)
	showCredits := user.Role != "admin" && user.Role != "beta"

	tableData := templates.UsageTableData{
		Logs:        items,
		ShowCredits: showCredits,
	}

	component := templates.UsageTable(tableData)
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}

// VerifyEmailPage handles email verification and renders the result page
func (h *WebHandler) VerifyEmailPage(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		component := templates.VerifyEmailError("Verification token is missing")
		if err := component.Render(c.Request.Context(), c.Writer); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
		}
		return
	}

	// Call the API endpoint to verify
	// For now, we'll just render success - the actual verification happens via API
	// TODO: Call internal verification logic
	component := templates.VerifyEmailSuccess()
	if err := component.Render(c.Request.Context(), c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to render template"})
	}
}
