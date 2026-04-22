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
	db.Where("status = ?", "active").Find(&accounts)

	fmt.Printf("Total active accounts to check: %d\n", len(accounts))

	var wg sync.WaitGroup
	jobs := make(chan *models.Account, len(accounts))
	workerCount := 20

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for acc := range jobs {
				checkAccount(db, acc)
			}
		}()
	}

	for i := range accounts {
		jobs <- &accounts[i]
	}
	close(jobs)
	wg.Wait()

	var img2Count int64
	db.Model(&models.Account{}).Where("has_img2 = ?", true).Count(&img2Count)
	fmt.Printf("\nCheck completed. Found %d accounts with IMG2 capability.\n", img2Count)
}

func checkAccount(db *gorm.DB, acc *models.Account) {
	client, _ := core.NewOpenAIClient(acc)
	hasIMG2, err := client.CheckCapability()
	if err != nil {
		fmt.Printf("[CHECK-FAIL] %s: %v\n", acc.Email, err)
		return
	}

	if hasIMG2 {
		db.Model(acc).Update("has_img2", true)
		fmt.Printf("[IMG2-HIT!] %s supports gpt-image-2\n", acc.Email)
	} else {
		// fmt.Printf("[NORMAL] %s only DALL-E 3\n", acc.Email)
	}
}
