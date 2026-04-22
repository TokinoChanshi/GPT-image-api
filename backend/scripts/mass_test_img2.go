package main

import (
	"evo-image-api/core"
	"evo-image-api/models"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
	// 测试所有活号 (82个)，重新找回那 38 个 IMG2
	db.Where("status = 'active'").Find(&accounts)

	fmt.Printf("🚀 Starting RAW DIRECT-OUTPUT Mass Test for %d active accounts...\n", len(accounts))

	// 使用用户要求的 raw 直出提示词，无任何改动
	prompt := "生成一张最终幻想 tifa lockhart 和 calvin klein 内衣的联动活动宣传图，人物占 80% 画面，躺在床上摆弄姿态，穿着联动款的产品，要贴合"

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 5 并发

	for i, acc := range accounts {
		wg.Add(1)
		go func(index int, a models.Account) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			logDir := "logs"
			outputDir := filepath.Join("output", "mass_test")
			
			logFile, _ := os.Create(filepath.Join(logDir, fmt.Sprintf("test_%d_%s.log", index+1, a.Email)))
			defer logFile.Close()

			fmt.Fprintf(logFile, "Testing Account: %s\n", a.Email)
			
			client, err := core.NewOpenAIClient(&a)
			if err != nil {
				fmt.Fprintf(logFile, "Client Init Failed: %v\n", err)
				return
			}
			client.Logger = logFile

			fmt.Printf("[%d/%d] Generating for %s...\n", index+1, len(accounts), a.Email)
			
			urls, err := client.GenerateImage(prompt)
			if err != nil {
				fmt.Fprintf(logFile, "\n❌ FAILED: %v\n", err)
				fmt.Printf("   ❌ %s FAILED\n", a.Email)
				return
			}

			// 通过日志逻辑判定是否是真正 IMG2
			// 由于 GenerateImage 内部已经做了辨证并打印到了 logFile
			// 我们只需要在这里简单下载
			
			fmt.Fprintf(logFile, "\n✨ SUCCESS! Found %d images.\n", len(urls))
			fmt.Printf("   ✅ %s SUCCESS! Images: %d\n", a.Email, len(urls))

			for j, u := range urls {
				suffix := "img1"
				if strings.Contains(u, "p=fs") { suffix = "img2" }
				filename := fmt.Sprintf("beijing_%d_%s_%d.webp", index+1, suffix, j+1)
				path := filepath.Join(outputDir, filename)
				err := client.DownloadImage(u, path)
				if err == nil {
					fmt.Fprintf(logFile, "   📥 Downloaded: %s\n", filename)
				}
			}
		}(i, acc)
	}

	wg.Wait()
	fmt.Println("\n🏁 Mass test completed. Check 'backend/logs' and 'backend/output/mass_test'.")
}
