package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"evo-image-api/config"
	"evo-image-api/core"
	"evo-image-api/database"
	"evo-image-api/models"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type ImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

func GenerateImageHandler(c *gin.Context) {
	var req ImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, _ := c.Get("user")
	currentUser := user.(models.User)

	if currentUser.Balance <= 0 && currentUser.Role != "admin" {
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "Insufficient balance"})
		return
	}

	scheduler := core.NewScheduler()
	requireIMG2 := req.Model == "gpt-image-2" || req.Model == "dall-e-3-v2"
	acc, err := scheduler.GetAvailableAccount(requireIMG2)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "No available accounts"})
		return
	}
	defer scheduler.ReleaseAccount(acc.ID)

	client, _ := core.NewOpenAIClient(acc)
	urls, err := client.GenerateImage(req.Prompt)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	// 扣费
	database.DB.Model(&currentUser).Update("balance", currentUser.Balance-1)
	scheduler.UpdateAccountUsage(acc.ID)

	// 转换为 HMAC 代理 URL
	var proxiedData []map[string]string
	secret := config.AppConfig.JWTSecret
	for _, u := range urls {
		encodedURL := base64.StdEncoding.EncodeToString([]byte(u))
		// Generate HMAC
		h := hmac.New(sha256.New, []byte(secret))
		h.Write([]byte(encodedURL + fmt.Sprint(acc.ID)))
		sig := hex.EncodeToString(h.Sum(nil))

		proxyURL := fmt.Sprintf("http://%s/v1/p/img?url=%s&acc=%d&sig=%s", c.Request.Host, encodedURL, acc.ID, sig)
		proxiedData = append(proxiedData, map[string]string{"url": proxyURL})
	}

	c.JSON(http.StatusOK, gin.H{
		"created": time.Now().Unix(),
		"data":    proxiedData,
	})
}

func ProxyImageHandler(c *gin.Context) {
	encodedURL := c.Query("url")
	accIDStr := c.Query("acc")
	sig := c.Query("sig")

	// Verify HMAC
	secret := config.AppConfig.JWTSecret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(encodedURL + accIDStr))
	expectedSig := hex.EncodeToString(h.Sum(nil))

	if sig != expectedSig {
		c.String(403, "Invalid signature")
		return
	}

	decodedURL, err := base64.StdEncoding.DecodeString(encodedURL)
	if err != nil {
		c.String(400, "Invalid URL encoding")
		return
	}
	targetURL := string(decodedURL)

	var acc models.Account
	database.DB.First(&acc, accIDStr)
	if acc.ID == 0 {
		c.String(404, "Account not found")
		return
	}

	client, _ := core.NewOpenAIClient(&acc)
	req, _ := http.NewRequest("GET", targetURL, nil)
	if strings.Contains(targetURL, "oaiusercontent.com") {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	} else {
		// 回源地址需要 Auth
		req.Header.Set("Authorization", "Bearer "+acc.AccessToken)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		if acc.AccountID != "" { req.Header.Set("oai-device-id", acc.AccountID) }
		if acc.SessionID != "" { req.Header.Set("Cookie", "authsession="+acc.SessionID) }
	}

	resp, err := client.Client.Do(req)
	if err != nil {
		c.String(502, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		c.String(resp.StatusCode, "Upstream error")
		return
	}

	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Header("Cache-Control", "public, max-age=3600")
	io.Copy(c.Writer, resp.Body)
}
