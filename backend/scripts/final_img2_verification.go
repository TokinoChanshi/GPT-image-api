package main

import (
	"evo-image-api/core"
	"evo-image-api/models"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("evo_image_api.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var accounts []models.Account
	// 获取 38 个活跃号进行最终辩证测试
	db.Where("status = 'active'").Find(&accounts)

	fmt.Printf("🚀 Starting Final Dialectic Audit for %d active accounts...\n", len(accounts))

	prompt := "北京旅游地图攻略海报，电影级构图，极致高清细节，中国风插画风格。"

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for i, acc := range accounts {
		wg.Add(1)
		go func(index int, a models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			logPath := filepath.Join("logs", fmt.Sprintf("audit_%d_%s.log", index+1, a.Email))
			logFile, _ := os.Create(logPath)
			defer logFile.Close()

			fmt.Fprintf(logFile, "Account: %s\n", a.Email)
			
			client, _ := core.NewOpenAIClient(&a)
			client.Logger = logFile

			fmt.Printf("[%d/%d] Auditing %s...\n", index+1, len(accounts), a.Email)
			
			urls, err := client.GenerateImage(prompt)
			if err != nil {
				fmt.Printf("   ❌ %s FAILED: %v\n", a.Email, err)
				return
			}

			fmt.Printf("   ✅ %s COMPLETED. Images: %d\n", a.Email, len(urls))

			for j, u := range urls {
				filename := fmt.Sprintf("audit_beijing_%d_%d.webp", index+1, j+1)
				_ = client.DownloadImage(u, filepath.Join("output", "mass_test", filename))
			}
		}(i, acc)
	}

	wg.Wait()
	fmt.Println("\n🏁 Final Audit Completed. Analyze logs in 'backend/logs/audit_*.log'.")
}
