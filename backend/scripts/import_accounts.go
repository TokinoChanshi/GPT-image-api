package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type Account struct {
	gorm.Model
	Email        string `gorm:"uniqueIndex;not null"`
	AccessToken  string `gorm:"type:text"`
	RefreshToken string `gorm:"type:text"`
	AccountType  string
	AccountID    string
	SessionID    string
	Proxy        string
	Status       string
	HasIMG2      bool `gorm:"default:false"`
	UsageLimit   int
	UsageCount   int
	NextResetAt  time.Time
}

type RefreshRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func main() {
	db, err := gorm.Open(sqlite.Open("evo_image_api.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	db.AutoMigrate(&Account{})

	tokenDir := `C:\Users\12562\创业网站项目？\代理\浏览器？\openai-pool-orchestrator-v5.1\data\tokens`
	proxy := "http://127.0.0.1:20002"
	clientID := "app_EMoamEEZ73f0CkXaXp7hrann"

	files, _ := os.ReadDir(tokenDir)
	var jsonFiles []string
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			jsonFiles = append(jsonFiles, filepath.Join(tokenDir, f.Name()))
		}
	}

	fmt.Printf("Updating %d potential accounts with session data...\n", len(jsonFiles))

	client := &http.Client{
		Transport: &http.Transport{
			Proxy:           http.ProxyURL(parseProxy(proxy)),
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 30 * time.Second,
	}

	var wg sync.WaitGroup
	jobs := make(chan string, len(jsonFiles))
	workerCount := 30

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				processFile(db, client, clientID, path)
			}
		}()
	}

	for _, path := range jsonFiles {
		jobs <- path
	}
	close(jobs)
	wg.Wait()

	fmt.Println("\nUpdate completed.")
}

func parseProxy(p string) *url.URL {
	u, _ := url.Parse(p)
	return u
}

func processFile(db *gorm.DB, client *http.Client, clientID, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	var tf map[string]interface{}
	json.Unmarshal(data, &tf)

	email, _ := tf["email"].(string)
	accID, _ := tf["account_id"].(string)
	idToken, _ := tf["id_token"].(string)

	if email == "" {
		return
	}

	// 提取 Session ID 从 JWT
	sessionID := extractSID(idToken)

	// 仅更新 SessionID 和 AccountID
	db.Model(&Account{}).Where("email = ?", email).Updates(map[string]interface{}{
		"account_id": accID,
		"session_id": sessionID,
	})
	
	fmt.Printf("[UPDATED] %s (SID: %s)\n", email, sessionID)
}

func extractSID(token string) string {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var data map[string]interface{}
	json.Unmarshal(payload, &data)
	
	if sid, ok := data["sid"].(string); ok {
		return sid
	}
	// Try access_token format
	if auth, ok := data["https://api.openai.com/auth"].(map[string]interface{}); ok {
		if sess, ok := auth["session_id"].(string); ok {
			return sess
		}
	}
	return ""
}

func doRefresh(client *http.Client, clientID, refreshToken string) (*RefreshResponse, error) {
	reqBody := RefreshRequest{
		GrantType:    "refresh_token",
		ClientID:     clientID,
		RefreshToken: refreshToken,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "https://auth0.openai.com/oauth/token", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var res RefreshResponse
	json.Unmarshal(body, &res)
	return &res, nil
}
