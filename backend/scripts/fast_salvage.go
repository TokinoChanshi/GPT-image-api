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
	// 获取所有被标记为 banned 的账号，进行快速生存排查
	db.Where("status = ?", "banned").Find(&accounts)

	total := len(accounts)
	fmt.Printf("🚀 FAST SALVAGE START: checking %d accounts...\n", total)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20) // 降低并发确保稳健
	mu := sync.Mutex{}
	
	alive := 0

	for i := range accounts {
		wg.Add(1)
		go func(acc *models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			client, _ := core.NewOpenAIClient(acc)
			
			// 只看模型列表能不能拉到，这是最快的存活判定
			ok, err := client.CheckCapability()
			
			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				alive++
				db.Model(acc).Updates(map[string]interface{}{
					"status":   "active",
					"has_img2": ok, // 顺便打标
				})
				fmt.Printf("   💎 [FOUND] %s\n", acc.Email)
			}
		}(&accounts[i])

		if i > 0 && i%500 == 0 {
			fmt.Printf("--- Processed %d/%d (Found: %d) ---\n", i, total, alive)
		}
	}

	wg.Wait()
	fmt.Printf("\n🏁 FAST SALVAGE COMPLETE. Total Found: %d\n", alive)
}
