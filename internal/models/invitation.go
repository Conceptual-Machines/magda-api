package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
)

// InvitationCode represents an invitation code for beta signup
type InvitationCode struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	Code        string     `gorm:"uniqueIndex;not null" json:"code"`
	CreatedByID uint       `gorm:"index" json:"created_by_id"`        // Admin who created this
	CreatedBy   *User      `gorm:"foreignKey:CreatedByID" json:"-"`   // Relationship to admin
	UsedByID    *uint      `gorm:"index" json:"used_by_id,omitempty"` // User who used it
	UsedBy      *User      `gorm:"foreignKey:UsedByID" json:"-"`      // Relationship to user
	UsedAt      *time.Time `json:"used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	MaxUses     int        `gorm:"default:1" json:"max_uses"`       // How many times it can be used (default 1)
	CurrentUses int        `gorm:"default:0" json:"current_uses"`   // How many times it's been used
	Note        string     `gorm:"type:text" json:"note,omitempty"` // Optional note about who it's for
}

const invitationCodeBytes = 16

// GenerateInvitationCode creates a random invitation code
func GenerateInvitationCode() (string, error) {
	bytes := make([]byte, invitationCodeBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// IsValid checks if the invitation code is still valid
func (i *InvitationCode) IsValid() bool {
	// Check if already fully used
	if i.CurrentUses >= i.MaxUses {
		return false
	}

	// Check if expired
	if i.ExpiresAt != nil && time.Now().After(*i.ExpiresAt) {
		return false
	}

	return true
}

// MarkAsUsed marks the invitation code as used by a user
func (i *InvitationCode) MarkAsUsed(userID uint, db *gorm.DB) error {
	now := time.Now()
	i.CurrentUses++

	// Only set UsedBy for single-use codes
	if i.MaxUses == 1 {
		i.UsedByID = &userID
		i.UsedAt = &now
	}

	return db.Save(i).Error
}
