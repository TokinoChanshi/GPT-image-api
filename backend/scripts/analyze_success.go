package main

import (
	"evo-image-api/core"
	"evo-image-api/models"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("evo_image_api.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var acc models.Account
	// 使用之前成功的账号
	db.Where("email = ?", "elisabet4806fb@1eg.moonairse.com").First(&acc)

	// 我们需要找回那个成功的 Conversation ID
	// 暂时没存，但我可以尝试获取最近的对话列表
	fmt.Printf("🎯 Analyzing Session for: %s\n", acc.Email)

	client, _ := core.NewOpenAIClient(&acc)
	
	convID := "69e6fc75-99d0-83ea-beee-9a9c035d1ca6"
	req, _ := http.NewRequest("GET", "https://chatgpt.com/backend-api/conversation/"+convID, nil)
	req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	
	resp, err := client.Client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Full Mapping: %s\n", string(body))
}
