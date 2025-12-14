package handlers

import (
	"net/http"

	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const bootstrapSecret = "TEMP_ADMIN_BOOTSTRAP_2025" // Change this after first use

type BootstrapHandler struct {
	db *gorm.DB
}

func NewBootstrapHandler(db *gorm.DB) *BootstrapHandler {
	return &BootstrapHandler{db: db}
}

type SetAdminRoleRequest struct {
	Email  string `json:"email" binding:"required,email"`
	Secret string `json:"secret" binding:"required"`
}

// SetAdminRole is a one-time endpoint to set a user's role to admin
// Protected by a secret token from environment
func (h *BootstrapHandler) SetAdminRole(c *gin.Context) {
	var req SetAdminRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Simple protection - require a secret
	if req.Secret != bootstrapSecret {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid secret"})
		return
	}

	// Find user
	var user models.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Update role
	user.Role = "admin"
	user.IsActive = true
	user.EmailVerified = true

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User promoted to admin",
		"user": gin.H{
			"id":             user.ID,
			"email":          user.Email,
			"role":           user.Role,
			"is_active":      user.IsActive,
			"email_verified": user.EmailVerified,
		},
	})
}

// CleanupUsers deletes all users except admin@musicalaideas.com
// TEMPORARY endpoint - remove after use
func (h *BootstrapHandler) CleanupUsers(c *gin.Context) {
	secret := c.Query("secret")

	// Simple protection - require a secret
	if secret != bootstrapSecret {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid secret"})
		return
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get users to delete
	var usersToDelete []models.User
	if err := tx.Where("email != ?", "admin@musicalaideas.com").Find(&usersToDelete).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find users"})
		return
	}

	userIDs := make([]uint, len(usersToDelete))
	for i, user := range usersToDelete {
		userIDs[i] = user.ID
	}

	if len(userIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No users to delete", "deleted_count": 0})
		return
	}

	// Delete related data
	tx.Where("user_id IN ?", userIDs).Delete(&models.EmailVerificationToken{})
	tx.Where("user_id IN ?", userIDs).Delete(&models.UserCredits{})
	tx.Where("user_id IN ?", userIDs).Delete(&models.UsageLog{})
	tx.Model(&models.InvitationCode{}).Where("created_by_id IN ?", userIDs).Update("created_by_id", nil)
	tx.Model(&models.InvitationCode{}).Where("used_by_id IN ?", userIDs).Update("used_by_id", nil)
	// Note: CompositionPlan model has been removed as part of reverting to one-shot generation

	// Delete users
	if err := tx.Where("id IN ?", userIDs).Delete(&models.User{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete users"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Get remaining users
	var remainingUsers []models.User
	h.db.Select("id, email, role, is_active").Find(&remainingUsers)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Users deleted successfully",
		"deleted_count":   len(userIDs),
		"remaining_users": remainingUsers,
	})
}
