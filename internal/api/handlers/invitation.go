package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/Conceptual-Machines/magda-api/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type InvitationHandler struct {
	db           *gorm.DB
	emailService *services.EmailService
}

func NewInvitationHandler(db *gorm.DB, emailService *services.EmailService) *InvitationHandler {
	return &InvitationHandler{
		db:           db,
		emailService: emailService,
	}
}

type CreateInvitationRequest struct {
	Note           string `json:"note"`
	MaxUses        int    `json:"max_uses"`         // Default 1 if not specified
	ExpiresInHours int    `json:"expires_in_hours"` // Optional: hours until expiration
}

type SendInvitationRequest struct {
	Email          string `json:"email" binding:"required,email"`
	Note           string `json:"note"`
	MaxUses        int    `json:"max_uses"`         // Default 1 if not specified
	ExpiresInHours int    `json:"expires_in_hours"` // Optional: hours until expiration
}

type ResendInvitationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type InvitationResponse struct {
	models.InvitationCode
	CreatedByEmail string `json:"created_by_email,omitempty"`
	UsedByEmail    string `json:"used_by_email,omitempty"`
}

// CreateInvitation generates a new invitation code
func (h *InvitationHandler) CreateInvitation(c *gin.Context) {
	var req CreateInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the current user from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Generate invitation code
	code, err := models.GenerateInvitationCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invitation code"})
		return
	}

	// Set defaults
	maxUses := req.MaxUses
	if maxUses <= 0 {
		maxUses = 1
	}

	invitation := models.InvitationCode{
		Code:        code,
		CreatedByID: userID.(uint),
		Note:        req.Note,
		MaxUses:     maxUses,
	}

	// Set expiration if specified
	if req.ExpiresInHours > 0 {
		expiresAt := time.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		invitation.ExpiresAt = &expiresAt
	}

	if err := h.db.Create(&invitation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invitation"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"invitation": invitation})
}

// ListInvitations returns all invitation codes
func (h *InvitationHandler) ListInvitations(c *gin.Context) {
	var invitations []models.InvitationCode

	// Query with creator and user relationships
	query := h.db.Preload("CreatedBy").Preload("UsedBy")

	// Optional filter by status
	status := c.Query("status")
	switch status {
	case "unused":
		query = query.Where("current_uses < max_uses")
	case "used":
		query = query.Where("current_uses >= max_uses")
	case "expired":
		query = query.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now())
	}

	if err := query.Order("created_at DESC").Find(&invitations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch invitations"})
		return
	}

	// Build response with email information
	var responses []InvitationResponse
	for _, inv := range invitations {
		resp := InvitationResponse{
			InvitationCode: inv,
		}
		if inv.CreatedBy != nil {
			resp.CreatedByEmail = inv.CreatedBy.Email
		}
		if inv.UsedBy != nil {
			resp.UsedByEmail = inv.UsedBy.Email
		}
		responses = append(responses, resp)
	}

	c.JSON(http.StatusOK, gin.H{"invitations": responses})
}

// DeleteInvitation removes an invitation code
func (h *InvitationHandler) DeleteInvitation(c *gin.Context) {
	invitationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}

	if err := h.db.Delete(&models.InvitationCode{}, invitationID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete invitation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation deleted successfully"})
}

// GetInvitationStats returns statistics about invitations
func (h *InvitationHandler) GetInvitationStats(c *gin.Context) {
	var total, used, unused, expired int64

	h.db.Model(&models.InvitationCode{}).Count(&total)
	h.db.Model(&models.InvitationCode{}).Where("current_uses >= max_uses").Count(&used)
	h.db.Model(&models.InvitationCode{}).Where("current_uses < max_uses").Count(&unused)
	h.db.Model(&models.InvitationCode{}).Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Count(&expired)

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"used":    used,
		"unused":  unused,
		"expired": expired,
	})
}

// SendInvitation creates an invitation code and sends it via email
func (h *InvitationHandler) SendInvitation(c *gin.Context) {
	var req SendInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get the current user from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Generate invitation code
	code, err := models.GenerateInvitationCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invitation code"})
		return
	}

	// Set defaults
	maxUses := req.MaxUses
	if maxUses <= 0 {
		maxUses = 1
	}

	note := req.Note
	if note == "" {
		note = req.Email
	}

	invitation := models.InvitationCode{
		Code:        code,
		CreatedByID: userID.(uint),
		Note:        note,
		MaxUses:     maxUses,
	}

	// Set expiration if specified
	if req.ExpiresInHours > 0 {
		expiresAt := time.Now().Add(time.Duration(req.ExpiresInHours) * time.Hour)
		invitation.ExpiresAt = &expiresAt
	}

	if err := h.db.Create(&invitation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invitation"})
		return
	}

	// Send invitation email
	if err := h.emailService.SendInvitationEmail(req.Email, code, note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invitation created but failed to send email"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Invitation created and sent successfully",
		"invitation": invitation,
	})
}

// ResendInvitation resends an existing invitation code via email
func (h *InvitationHandler) ResendInvitation(c *gin.Context) {
	invitationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}

	var req ResendInvitationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get invitation
	var invitation models.InvitationCode
	if err := h.db.First(&invitation, invitationID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		return
	}

	// Send invitation email
	if err := h.emailService.SendInvitationEmail(req.Email, invitation.Code, invitation.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation resent successfully"})
}
