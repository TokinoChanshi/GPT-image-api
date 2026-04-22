package main

import (
	"evo-image-api/core"
	"evo-image-api/models"
	"fmt"
	"log"
	"strings"
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
	// 获取当前所有活号进行实战生图探测
	db.Where("status = ?", "active").Find(&accounts)

	fmt.Printf("🔍 Hunting for real IMG2 across %d active accounts...\n", len(accounts))

	var wg sync.WaitGroup
	mu := sync.Mutex{}
	img2Found := 0

	for _, acc := range accounts {
		wg.Add(1)
		go func(a models.Account) {
			defer wg.Done()
			
			client, _ := core.NewOpenAIClient(&a)
			fmt.Printf("🎯 Testing %s...\n", a.Email)
			
			// 发起一次极简生图请求
			urls, err := client.GenerateImage("A simple red circle")
			
			mu.Lock()
			defer mu.Unlock()
			
			if err != nil {
				fmt.Printf("   ❌ FAILED %s: %v\n", a.Email, err)
				return
			}

			// 通过返回的 URL 数量或内容判定
			isRealIMG2 := false
			if len(urls) >= 2 {
				isRealIMG2 = true
			}
			for _, u := range urls {
				if strings.Contains(u, "p=fs") { // file-service
					isRealIMG2 = true
				}
			}

			if isRealIMG2 {
				img2Found++
				db.Model(&a).Update("has_img2", true)
				fmt.Printf("🔥 [IMG2-CONFIRMED] %s is a real IMG2 account! (Images: %d)\n", a.Email, len(urls))
			} else {
				fmt.Printf("✅ [IMG1-ONLY] %s generated 1 image.\n", a.Email)
			}
		}(acc)
	}

	wg.Wait()
	fmt.Printf("\n--- Hunt Completed. Found %d IMG2 accounts. ---\n", img2Found)
}
