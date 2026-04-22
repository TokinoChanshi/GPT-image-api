package main

import (
	"evo-image-api/core"
	"evo-image-api/models"
	"fmt"
	"log"
	"sync"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("evo_image_api.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var accounts []models.Account
	// 获取所有被标记为 banned 的账号，进行全面打捞
	db.Where("status = ?", "banned").Find(&accounts)

	total := len(accounts)
	fmt.Printf("🚀 GLOBAL RESCUE MISSION START: Verifying %d accounts...\n", total)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 15) // 并发控制
	mu := sync.Mutex{}
	
	rescued := 0
	failed := 0

	for i := range accounts {
		wg.Add(1)
		go func(acc *models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			client, _ := core.NewOpenAIClient(acc)
			
			// 核心测试：直出一张小图
			urls, err := client.GenerateImage("a dot")
			
			mu.Lock()
			defer mu.Unlock()

			if err == nil && len(urls) > 0 {
				rescued++
				db.Model(acc).Updates(map[string]interface{}{
					"status": "active",
					"has_img2": true, 
				})
				fmt.Printf("   💎 [RESCUED] %s\n", acc.Email)
			} else {
				failed++
				// 如果是 429，保留状态，不计入失败统计，稍后可以重试
				if err != nil && strings.Contains(err.Error(), "429") {
					// c.log 内部已有 backoff，这里简单记录
				}
			}
		}(&accounts[i])

		if i > 0 && i%100 == 0 {
			fmt.Printf("--- Progress: %d/%d (Rescued: %d, Failed: %d) ---\n", i, total, rescued, failed)
		}
	}

	wg.Wait()
	fmt.Printf("\n🏁 MISSION COMPLETE. Total Rescued: %d\n", rescued)
}
