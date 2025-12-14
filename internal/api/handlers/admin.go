package handlers

import (
	"net/http"
	"strconv"

	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	activeStatusTrue  = "true"
	maxUsageLogsLimit = 50
)

type AdminHandler struct {
	db *gorm.DB
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// ListUsers returns all users with their credits and usage stats
func (h *AdminHandler) ListUsers(c *gin.Context) {
	var users []models.User

	// Get query parameters for filtering
	role := c.Query("role")
	isActive := c.Query("is_active")

	query := h.db.Model(&models.User{})

	if role != "" {
		query = query.Where("role = ?", role)
	}

	if isActive != "" {
		query = query.Where("is_active = ?", isActive == activeStatusTrue)
	}

	if err := query.Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Attach credits to each user
	type UserWithCredits struct {
		models.User
		Credits int `json:"credits"`
	}

	var usersWithCredits []UserWithCredits
	for _, user := range users {
		var credits models.UserCredits
		h.db.Where("user_id = ?", user.ID).First(&credits)

		usersWithCredits = append(usersWithCredits, UserWithCredits{
			User:    user,
			Credits: credits.Credits,
		})
	}

	c.JSON(http.StatusOK, gin.H{"users": usersWithCredits})
}

// UpdateUserRole updates a user's role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required,oneof=admin beta user"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.Role = req.Role
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully", "user": user})
}

// ToggleUserActive toggles a user's active status
func (h *AdminHandler) ToggleUserActive(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsActive = !user.IsActive
	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User status updated", "user": user})
}

// UpdateUserCredits updates a user's credit balance
func (h *AdminHandler) UpdateUserCredits(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Credits int    `json:"credits" binding:"required,min=0"`
		Action  string `json:"action" binding:"required,oneof=set add subtract"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var credits models.UserCredits
	if err := h.db.Where("user_id = ?", userID).First(&credits).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create credits record if it doesn't exist
			credits = models.UserCredits{
				UserID:  uint(userID),
				Credits: 0,
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch credits"})
			return
		}
	}

	switch req.Action {
	case "set":
		credits.Credits = req.Credits
	case "add":
		credits.Credits += req.Credits
	case "subtract":
		credits.Credits -= req.Credits
		if credits.Credits < 0 {
			credits.Credits = 0
		}
	}

	if err := h.db.Save(&credits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update credits"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Credits updated successfully", "credits": credits})
}

// GetUserDetails returns detailed info about a specific user
func (h *AdminHandler) GetUserDetails(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var credits models.UserCredits
	h.db.Where("user_id = ?", userID).First(&credits)

	var usageLogs []models.UsageLog
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(maxUsageLogsLimit).Find(&usageLogs)

	c.JSON(http.StatusOK, gin.H{
		"user":       user,
		"credits":    credits.Credits,
		"usage_logs": usageLogs,
	})
}

// DeleteUser permanently deletes a user and all associated data
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Start a transaction to ensure all related data is deleted
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Check if user exists
	var user models.User
	if err := tx.First(&user, userID).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	// Delete related data
	// Delete user credits
	if err := tx.Where("user_id = ?", userID).Delete(&models.UserCredits{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user credits"})
		return
	}

	// Delete usage logs
	if err := tx.Where("user_id = ?", userID).Delete(&models.UsageLog{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete usage logs"})
		return
	}

	// Delete email verification tokens
	if err := tx.Where("user_id = ?", userID).Delete(&models.EmailVerificationToken{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete email verification tokens"})
		return
	}

	// Update invitations created by this user (set creator to null)
	if err := tx.Model(&models.InvitationCode{}).Where("created_by_id = ?", userID).Update("created_by_id", nil).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update created invitations"})
		return
	}

	// Update invitations used by this user (set used_by to null)
	if err := tx.Model(&models.InvitationCode{}).Where("used_by_id = ?", userID).Update("used_by_id", nil).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update used invitations"})
		return
	}

	// Delete composition plans and generated sequences
	// Note: These models have been removed as part of reverting to one-shot generation

	// Finally, delete the user
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deletion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
