package models

import (
	"time"

	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"`
	AccessToken  string `gorm:"type:text"`
	RefreshToken string `gorm:"type:text"`
	AccountType  string // Plus, Free
	AccountID    string
	SessionID    string
	Proxy        string // IP:Port:User:Pass
	Status       string // active, limited, expired
	HasIMG2      bool   `gorm:"default:false"`
	UsageLimit   int    // e.g., 40
	UsageCount   int    // current usage in window
	NextResetAt  time.Time
	DeviceID     string
}
