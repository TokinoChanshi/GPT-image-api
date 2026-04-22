package main

import (
	"evo-image-api/models"
	"fmt"
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("evo_image_api.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	db.AutoMigrate(&models.User{}, &models.APIKey{}, &models.Account{})

	// 创建管理员
	admin := models.User{
		Username: "admin",
		Password: "password123", // 生产环境应加密
		Email:    "admin@evo.com",
		Balance:  99999,
		Role:     "admin",
	}
	db.Where(models.User{Username: "admin"}).FirstOrCreate(&admin)

	// 创建初始 API Key
	apiKey := models.APIKey{
		UserID: admin.ID,
		Key:    "sk-evo-test-key-001",
		Status: true,
	}
	db.Where(models.APIKey{Key: "sk-evo-test-key-001"}).FirstOrCreate(&apiKey)

	fmt.Println("Initial setup completed.")
	fmt.Println("Admin Username: admin")
	fmt.Println("Admin Password: password123")
	fmt.Println("Test API Key: sk-evo-test-key-001")
}
