package main

import (
	"bytes"
	"encoding/json"
	"evo-image-api/core"
	"evo-image-api/models"
	"evo-image-api/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func main() {
	db, err := gorm.Open(sqlite.Open("evo_image_api.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	var acc models.Account
	// 使用确认命中的精品号
	db.Where("email = ?", "eleanora485751@ujj.cloudvxz.com").First(&acc)

	fmt.Printf("🎯 Inspecting account: %s\n", acc.Email)
	client, _ := core.NewOpenAIClient(&acc)

	// 1. 检查 /sentinel/chat-requirements
	reqToken := utils.NewPOWConfig("").RequirementsToken()
	body, _ := json.Marshal(map[string]string{"p": reqToken})
	req, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/sentinel/chat-requirements", bytes.NewBuffer(body))
	
	req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	if acc.AccountID != "" { req.Header.Set("oai-device-id", acc.AccountID) }
	if acc.SessionID != "" { req.Header.Set("Cookie", "authsession="+acc.SessionID) }

	resp, err := client.Client.Do(req)
	if err != nil { log.Fatal(err) }
	
	reqsBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("\n--- [SENTINEL REQUIREMENTS] ---\n%s\n", string(reqsBody))

	// 2. 发起生图并抓取 SSE 原始数据
	fmt.Println("\n🚀 Sending Generation Request (Dumping SSE)...")
	
	reqBody := map[string]interface{}{
		"action": "next",
		"messages": []map[string]interface{}{{
			"id":      utils.GenerateUUID(),
			"author":  map[string]string{"role": "user"},
			"content": map[string]interface{}{"content_type": "text", "parts": []string{"A futuristic city with dragons, cinematic view"}},
			"metadata": map[string]interface{}{"system_hints": []string{"picture_v2"}},
		}},
		"parent_message_id":     utils.GenerateUUID(),
		"model":                 "auto",
		"client_prepare_state":  "sent",
		"system_hints":          []string{"picture_v2"},
		"history_and_training_disabled": false,
	}
	
	jsonData, _ := json.Marshal(reqBody)
	reqSSE, _ := http.NewRequest("POST", "https://chatgpt.com/backend-api/f/conversation", bytes.NewBuffer(jsonData))
	
	reqSSE.Header.Set("Authorization", "Bearer "+acc.AccessToken)
	reqSSE.Header.Set("Content-Type", "application/json")
	reqSSE.Header.Set("Accept", "text/event-stream")
	reqSSE.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	
	// 从 requirements 中提取 token
	var reqsData map[string]interface{}
	json.Unmarshal(reqsBody, &reqsData)
	token, _ := reqsData["token"].(string)
	reqSSE.Header.Set("openai-sentinel-chat-requirements-token", token)

	respSSE, err := client.Client.Do(reqSSE)
	if err != nil { log.Fatal(err) }
	defer respSSE.Body.Close()

	// 保存原始 SSE 到文件
	f, _ := os.Create("raw_sse_dump.txt")
	defer f.Close()
	io.Copy(f, respSSE.Body)
	
	fmt.Println("✅ SSE stream dumped to raw_sse_dump.txt")
}
