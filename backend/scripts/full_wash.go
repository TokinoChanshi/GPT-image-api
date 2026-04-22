package main

import (
	"evo-image-api/core"
	"evo-image-api/models"
	"fmt"
	"log"
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
	// 获取所有非 banned 的账号（刚导入的都是 active/new）
	db.Find(&accounts)

	total := len(accounts)
	fmt.Printf("🚀 Starting GLOBAL SALVAGE MISSION for %d accounts...\n", total)
	fmt.Println("This will verify REAL generation capability for every single account.")

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20) // 并发控制
	mu := sync.Mutex{}
	
	aliveCount := 0
	bannedCount := 0

	for i := range accounts {
		wg.Add(1)
		go func(acc *models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			client, _ := core.NewOpenAIClient(acc)
			
			// 核心：必须成功直出一张图才算活号
			urls, err := client.GenerateImage("a dot")
			
			mu.Lock()
			defer mu.Unlock()

			if err == nil && len(urls) > 0 {
				aliveCount++
				db.Model(acc).Update("status", "active")
				fmt.Printf("   💎 [LIVE] %s\n", acc.Email)
			} else {
				bannedCount++
				db.Model(acc).Update("status", "banned")
				// fmt.Printf("   ❌ [DEAD] %s\n", acc.Email)
			}
		}(&accounts[i])

		if i > 0 && i%50 == 0 {
			fmt.Printf("--- Progress: %d/%d (Success: %d, Failed: %d) ---\n", i, total, aliveCount, bannedCount)
		}
	}

	wg.Wait()
	fmt.Printf("\n🏁 GLOBAL SALVAGE COMPLETED.\n")
	fmt.Printf("Total Found Usable: %d\n", aliveCount)
	fmt.Println("All usable accounts are now marked as 'active' in the database.")
}
