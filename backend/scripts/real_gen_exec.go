package main

import (
	"evo-image-api/core"
	"evo-image-api/models"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("evo_image_api.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var accounts []models.Account
	// 选取我们刚刚确认存活的精华号
	db.Where("status = 'active' AND has_img2 = 1").Limit(5).Find(&accounts)
	if len(accounts) == 0 {
		fmt.Println("No active accounts found.")
		return
	}

	// 使用用户要求的直出提示词，不进行任何额外优化
	prompt := "生成一张最终幻想 tifa lockhart 和 calvin klein 内衣的联动活动宣传图，人物占 80% 画面，躺在床上摆弄姿态，穿着联动款的产品，要贴合"
	
	success := false
	for _, acc := range accounts {
		fmt.Printf("\n🎯 [DIRECT-TEST] Target Account: %s\n", acc.Email)
		fmt.Println("🚀 Executing RAW Prompt Generation (No optimization)...")

		client, err := core.NewOpenAIClient(&acc)
		if err != nil {
			fmt.Printf("   ❌ Client Init Failed: %v\n", err)
			continue
		}
		client.Logger = os.Stdout

		imageURLs, err := client.GenerateImage(prompt)
		if err != nil {
			fmt.Printf("\n❌ GENERATION FAILED for %s: %v\n", acc.Email, err)
			continue // 换下一个号测
		}

		fmt.Printf("\n✨ SUCCESS! Found %d images.\n", len(imageURLs))

		for i, url := range imageURLs {
			filename := fmt.Sprintf("tifa_ck_collab_%d.webp", i+1)
			outputPath := filepath.Join("output", filename)
			
			fmt.Printf("📥 Downloading Image %d to %s...\n", i+1, outputPath)
			err := client.DownloadImage(url, outputPath)
			if err != nil {
				fmt.Printf("   ❌ Download Failed: %v\n", err)
			} else {
				fmt.Printf("   ✅ Downloaded Successfully!\n")
				fmt.Printf("   🔗 Source: %s\n", url)
			}
		}
		success = true
		break
	}

	if !success {
		fmt.Println("\n🏁 All attempted accounts failed to generate the image.")
	} else {
		fmt.Println("\n🏁 Task completed successfully. Check the 'backend/output' folder.")
	}
}
