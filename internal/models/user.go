package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Email         string         `gorm:"uniqueIndex;not null" json:"email"`
	Password      string         `gorm:"not null" json:"-"`
	Name          string         `json:"name"`
	Role          string         `gorm:"default:'user';index" json:"role"` // "admin", "beta", "user"
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	EmailVerified bool           `gorm:"default:false" json:"email_verified"`
	VerifiedAt    *time.Time     `json:"verified_at,omitempty"`
}

// HashPassword hashes the user's password using bcrypt
func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// CheckPassword compares a password with the user's hashed password
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// UserCredits tracks user credit balance
type UserCredits struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	Credits   int            `gorm:"default:0;not null" json:"credits"`
}

// UsageLog tracks API usage and credit consumption
type UsageLog struct {
	ID              uint      `gorm:"primarykey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UserID          uint      `gorm:"not null;index" json:"user_id"`
	User            User      `gorm:"foreignKey:UserID" json:"-"`
	Model           string    `gorm:"not null" json:"model"`
	TotalTokens     int       `gorm:"not null" json:"total_tokens"`
	InputTokens     int       `gorm:"not null" json:"input_tokens"`
	OutputTokens    int       `gorm:"not null" json:"output_tokens"`
	ReasoningTokens int       `gorm:"default:0" json:"reasoning_tokens"`
	CreditsCharged  int       `gorm:"not null" json:"credits_charged"`
	MCPUsed         bool      `gorm:"default:false" json:"mcp_used"`
	MCPCalls        int       `gorm:"default:0" json:"mcp_calls"`
	MCPTools        string    `gorm:"type:text" json:"mcp_tools"` // Comma-separated list
	DurationMS      int       `gorm:"not null" json:"duration_ms"`
	RequestID       string    `gorm:"index" json:"request_id"`
}

// OAuthProvider tracks social login providers
type OAuthProvider struct {
	ID             uint           `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
	UserID         uint           `gorm:"not null;index" json:"user_id"`
	User           User           `gorm:"foreignKey:UserID" json:"-"`
	Provider       string         `gorm:"not null;index" json:"provider"` // "google", "github"
	ProviderUserID string         `gorm:"not null;uniqueIndex:idx_provider_user" json:"provider_user_id"`
}

// EmailVerificationToken stores tokens for email verification
type EmailVerificationToken struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	UserID    uint           `gorm:"not null;index" json:"user_id"`
	User      User           `gorm:"foreignKey:UserID" json:"-"`
	Token     string         `gorm:"uniqueIndex;not null" json:"token"`
	ExpiresAt time.Time      `gorm:"not null" json:"expires_at"`
	UsedAt    *time.Time     `json:"used_at,omitempty"`
}
