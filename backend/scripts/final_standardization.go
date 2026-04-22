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

	// 1. 获取所有之前判定为 active 的账号
	var accounts []models.Account
	db.Where("status = ?", "active").Find(&accounts)

	fmt.Printf("🔍 Performing Final Standardization for %d active accounts...\n", len(accounts))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10)
	mu := sync.Mutex{}

	finalActive := 0
	finalIMG2 := 0

	for i, acc := range accounts {
		wg.Add(1)
		go func(index int, a models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			client, _ := core.NewOpenAIClient(&a)
			
			// 使用最轻量的直出测试来最终确权
			urls, err := client.GenerateImage("A small gold star")
			
			mu.Lock()
			defer mu.Unlock()

			if err == nil && len(urls) > 0 {
				finalActive++
				// 标记为最高等级的 IMG2 资源
				db.Model(&a).Updates(map[string]interface{}{
					"status":       "active",
					"has_img2":     true,
					"account_type": "Plus-Elite", // 确认为全量推送后的顶配号
					"usage_count":  0,            // 重置计数
				})
				finalIMG2++
				fmt.Printf("   💎 [VERIFIED] %s is PERFECT.\n", a.Email)
			} else {
				db.Model(&a).Update("status", "deactivated")
				fmt.Printf("   ⚠️ [DROPPED] %s failed final check.\n", a.Email)
			}
		}(i, acc)
	}

	wg.Wait()

	// 2. 清理数据库：删除所有已封禁或失效的账号，确保开源版数据纯净
	db.Where("status != ?", "active").Delete(&models.Account{})

	fmt.Printf("\n--- 🏁 Data Standardization Completed ---\n")
	fmt.Printf("Total Elite Accounts: %d\n", finalActive)
	fmt.Printf("Database cleaned. All 'Banned' or 'Legacy' garbage removed.\n")
	
	// 3. 清理文件系统
	cleanup()
}

func cleanup() {
	files, _ := filepath.Glob("logs/*.log")
	for _, f := range files { os.Remove(f) }
	
	outputFiles, _ := filepath.Glob("output/mass_test/*.webp")
	for _, f := range outputFiles { os.Remove(f) }
	
	fmt.Println("Logs and temporary test images cleared.")
}
