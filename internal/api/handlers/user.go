package handlers

import (
	"net/http"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/middleware"
	"github.com/Conceptual-Machines/magda-api/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	db             *gorm.DB
	creditsService *services.CreditsService
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{
		db:             db,
		creditsService: services.NewCreditsService(db),
	}
}

// GetProfile returns the current user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	user, exists := middleware.GetCurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get credits
	credits, err := h.creditsService.GetUserCredits(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get credits"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":         user.ID,
			"email":      user.Email,
			"name":       user.Name,
			"is_active":  user.IsActive,
			"created_at": user.CreatedAt,
		},
		"credits": credits.Credits,
	})
}

// GetCredits returns the current user's credit balance
func (h *UserHandler) GetCredits(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	credits, err := h.creditsService.GetUserCredits(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get credits"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"credits":    credits.Credits,
		"updated_at": credits.UpdatedAt,
	})
}

// GetUsageStats returns usage statistics for the current user
func (h *UserHandler) GetUsageStats(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse time range from query params
	var from, to time.Time

	if fromStr := c.Query("from"); fromStr != "" {
		parsed, err := time.Parse(time.RFC3339, fromStr)
		if err == nil {
			from = parsed
		}
	}

	if toStr := c.Query("to"); toStr != "" {
		parsed, err := time.Parse(time.RFC3339, toStr)
		if err == nil {
			to = parsed
		}
	}

	// Default to last 30 days if not specified
	if from.IsZero() {
		from = time.Now().AddDate(0, 0, -30)
	}
	if to.IsZero() {
		to = time.Now()
	}

	stats, err := h.creditsService.GetUserUsageStats(userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get usage stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
		"period": gin.H{
			"from": from.Format(time.RFC3339),
			"to":   to.Format(time.RFC3339),
		},
	})
}

// GetUsageHistory returns paginated usage history
func (h *UserHandler) GetUsageHistory(c *gin.Context) {
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Parse pagination params
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := time.Parse("1", pageStr); err == nil {
			page = int(p.Unix())
		}
	}

	pageSize := 20
	if sizeStr := c.Query("page_size"); sizeStr != "" {
		if s, err := time.Parse("1", sizeStr); err == nil {
			pageSize = int(s.Unix())
			if pageSize > maxHistoryPageSize {
				pageSize = maxHistoryPageSize
			}
		}
	}

	offset := (page - 1) * pageSize

	// Get usage logs
	var logs []map[string]interface{}
	query := h.db.Table("usage_logs").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset)

	if err := query.Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get usage history"})
		return
	}

	// Get total count
	var totalCount int64
	if err := h.db.Table("usage_logs").Where("user_id = ?", userID).Count(&totalCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count usage history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total_count": totalCount,
			"total_pages": (totalCount + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}
