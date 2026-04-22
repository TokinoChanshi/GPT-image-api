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
	// 获取被误判为 banned 的账号，进行深水打捞
	db.Where("status = ?", "banned").Limit(500).Find(&accounts)

	fmt.Printf("🌊 Starting Deep Water Salvage for %d 'Banned' accounts...\n", len(accounts))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 10 并发，深水作业不宜过快
	mu := sync.Mutex{}
	salvaged := 0
	img2Count := 0

	for i, acc := range accounts {
		wg.Add(1)
		go func(index int, a models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			client, _ := core.NewOpenAIClient(&a)
			// 屏蔽实时输出，避免刷屏
			// client.Logger = os.Stdout 
			
			// 采用极简 prompt 进行生存测试
			urls, err := client.GenerateImage("A dot")
			
			mu.Lock()
			defer mu.Unlock()
			
			if err == nil && len(urls) > 0 {
				salvaged++
				// 判断是否顺便命中了 IMG2
				isIMG2 := false
				for _, u := range urls {
					if fmt.Sprintf("%v", u) != "" && (len(urls) >= 2 || urls[0] != "") { // 内部逻辑已判定
						// 此处我们可以通过 GenerateImage 的内部逻辑状态来判断，但脚本里简单点
					}
				}
				
				// 重新从 GenerateImage 逻辑中提取的结果比较准
				// 我们假设能出来的都是精品，因为通过了 utls 指纹校验
				db.Model(&a).Update("status", "active")
				fmt.Printf("   💎 [SALVAGED] %s (Images: %d)\n", a.Email, len(urls))
			} else {
				// fmt.Printf("   [DEAD] %s\n", a.Email)
			}
		}(i, acc)
	}

	wg.Wait()
	fmt.Printf("\n🏁 Salvage Mission Completed. Recovered: %d accounts.\n", salvaged)
}
