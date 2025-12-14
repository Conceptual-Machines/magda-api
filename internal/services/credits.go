package services

import (
	"fmt"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/models"
	"gorm.io/gorm"
)

type CreditsService struct {
	db *gorm.DB
}

func NewCreditsService(db *gorm.DB) *CreditsService {
	return &CreditsService{db: db}
}

// GetUserCredits retrieves the current credit balance for a user
func (s *CreditsService) GetUserCredits(userID uint) (*models.UserCredits, error) {
	var credits models.UserCredits
	if err := s.db.Where("user_id = ?", userID).First(&credits).Error; err != nil {
		return nil, err
	}
	return &credits, nil
}

// HasSufficientCredits checks if a user has enough credits for a generation
func (s *CreditsService) HasSufficientCredits(userID uint, requiredCredits int) (bool, int, error) {
	credits, err := s.GetUserCredits(userID)
	if err != nil {
		return false, 0, err
	}
	return credits.Credits >= requiredCredits, credits.Credits, nil
}

// CalculateCredits calculates credit cost for a generation
// Flat rate: 1 credit per generation (regardless of token usage)
// Tokens are still logged for analytics and cost tracking
func (s *CreditsService) CalculateCredits(_ int) int {
	return 1 // Flat rate: 1 credit = 1 generation
}

// DeductCredits deducts credits from a user's balance
// If already negative, blocks the request (must top up first)
// If positive, allows going negative by one request (overdraft grace)
// Users with unlimited credits (e.g., admins) are not deducted
func (s *CreditsService) DeductCredits(userID uint, credits int) error {
	// Check if user has unlimited credits
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return err
	}

	if models.HasUnlimitedCredits(user.Role) {
		// Don't deduct for unlimited users (admins)
		return nil
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Lock the row to prevent race conditions
		var userCredits models.UserCredits
		if err := tx.Raw("SELECT * FROM user_credits WHERE user_id = ? FOR UPDATE", userID).
			Scan(&userCredits).Error; err != nil {
			return err
		}

		// If already in overdraft (negative), require top-up before next request
		if userCredits.Credits < 0 {
			return fmt.Errorf("account in overdraft (%d credits). Please purchase credits to continue",
				userCredits.Credits)
		}

		// Allow going negative (one-time overdraft grace)
		userCredits.Credits -= credits
		return tx.Save(&userCredits).Error
	})
}

// AddCredits adds credits to a user's balance (for purchases/rewards)
// If balance is negative, resets to 0 first (forgives overdraft), then adds credits
func (s *CreditsService) AddCredits(userID uint, credits int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var userCredits models.UserCredits
		if err := tx.Raw("SELECT * FROM user_credits WHERE user_id = ? FOR UPDATE", userID).
			Scan(&userCredits).Error; err != nil {
			return err
		}

		// If in overdraft, reset to 0 first (forgive the debt)
		if userCredits.Credits < 0 {
			userCredits.Credits = credits
		} else {
			userCredits.Credits += credits
		}
		return tx.Save(&userCredits).Error
	})
}

// LogUsage logs API usage and credit consumption
func (s *CreditsService) LogUsage(log *models.UsageLog) error {
	return s.db.Create(log).Error
}

// GetUserUsageStats retrieves usage statistics for a user
func (s *CreditsService) GetUserUsageStats(userID uint, from, to time.Time) (*UsageStats, error) {
	var stats UsageStats

	query := s.db.Model(&models.UsageLog{}).Where("user_id = ?", userID)

	if !from.IsZero() {
		query = query.Where("created_at >= ?", from)
	}
	if !to.IsZero() {
		query = query.Where("created_at <= ?", to)
	}

	// Get aggregated stats - only query fields that exist in database
	if err := query.Select(
		"COUNT(*) as total_requests",
		"COALESCE(SUM(credits_charged), 0) as total_credits_used",
		"COALESCE(AVG(duration_ms), 0) as avg_duration_ms",
	).Scan(&stats).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

type UsageStats struct {
	TotalRequests        int64            `json:"total_requests"`
	TotalTokensUsed      int64            `json:"total_tokens_used"`
	TotalInputTokens     int64            `json:"total_input_tokens"`
	TotalOutputTokens    int64            `json:"total_output_tokens"`
	TotalReasoningTokens int64            `json:"total_reasoning_tokens"`
	TotalCreditsUsed     int64            `json:"total_credits_used"`
	MCPRequests          int64            `json:"mcp_requests"`
	TotalMCPCalls        int64            `json:"total_mcp_calls"`
	AvgDurationMS        float64          `json:"avg_duration_ms"`
	ModelUsage           map[string]int64 `json:"model_usage"`
	MCPToolsUsage        map[string]int64 `json:"mcp_tools_usage"`
}
