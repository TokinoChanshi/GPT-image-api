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
	if err != nil { log.Fatal(err) }

	var accounts []models.Account
	// 获取所有 active 的号进行真伪验证
	db.Where("status = ?", "active").Find(&accounts)

	fmt.Printf("🔍 DEEP AUDIT: Verifying %d 'Active' accounts for deactivation...\n", len(accounts))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 20)
	mu := sync.Mutex{}
	
	realAlive := 0
	deactivated := 0

	for i := range accounts {
		wg.Add(1)
		go func(acc *models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			client, _ := core.NewOpenAIClient(acc)
			
			// 尝试拉取模型列表
			hasIMG2, err := client.CheckCapability()
			
			mu.Lock()
			defer mu.Unlock()

			if err == nil {
				realAlive++
				fmt.Printf("   ✅ [REAL-LIVE] %s (IMG2: %v)\n", acc.Email, hasIMG2)
			} else {
				errStr := err.Error()
				if strings.Contains(errStr, "deactivated") || strings.Contains(errStr, "401") {
					deactivated++
					db.Model(acc).Update("status", "banned")
				}
			}
		}(&accounts[i])
	}

	wg.Wait()
	fmt.Printf("\n🏁 AUDIT COMPLETE. Real Alive: %d, Deactivated: %d\n", realAlive, deactivated)
}
