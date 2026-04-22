package models

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Password string `gorm:"not null"`
	Email    string `gorm:"uniqueIndex"`
	Balance  int    `gorm:"default:0"` // Credits
	Role     string `gorm:"default:user"` // admin, user
	APIKeys  []APIKey
}

type APIKey struct {
	gorm.Model
	UserID uint
	Key    string `gorm:"uniqueIndex;not null"`
	Status bool   `gorm:"default:true"`
}
