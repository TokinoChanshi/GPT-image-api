package api

import (
	"encoding/json"
	"evo-image-api/database"
	"evo-image-api/models"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetStats(c *gin.Context) {
	var total, active, banned, expired, img2Active int64
	database.DB.Model(&models.Account{}).Count(&total)
	database.DB.Model(&models.Account{}).Where("status = ?", "active").Count(&active)
	database.DB.Model(&models.Account{}).Where("status = ?", "banned").Count(&banned)
	database.DB.Model(&models.Account{}).Where("status = ?", "expired").Count(&expired)
	database.DB.Model(&models.Account{}).Where("status = ? AND has_img2 = ?", "active", true).Count(&img2Active)

	c.JSON(http.StatusOK, gin.H{
		"total":   total,
		"active":  active,
		"banned":  banned + expired, // Total offline nodes
		"img2":    img2Active,     // Only count active IMG2 nodes
	})
}

func GetAccounts(c *gin.Context) {
	var accounts []models.Account
	// Prioritize Active accounts, then by ID descending
	database.DB.Order("status asc, has_img2 desc, id desc").Limit(100).Find(&accounts)
	c.JSON(http.StatusOK, accounts)
}

func ImportAccounts(c *gin.Context) {
	file, _ := c.FormFile("file")
	if file == nil {
		c.JSON(400, gin.H{"error": "No file uploaded"})
		return
	}

	f, _ := file.Open()
	defer f.Close()
	content, _ := io.ReadAll(f)

	var raw interface{}
	if err := json.Unmarshal(content, &raw); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON"})
		return
	}

	imported := 0
	process := func(item map[string]interface{}) {
		email, _ := item["email"].(string)
		if email == "" { return }

		acc := models.Account{
			Email:        email,
			AccessToken:  getString(item, "access_token", "accessToken", "at"),
			RefreshToken: getString(item, "refresh_token", "refreshToken", "rt"),
			SessionID:    getString(item, "session_id", "session_token", "sid"),
			AccountID:    getString(item, "account_id", "aid"),
			Status:       "active",
		}
		
		database.DB.Where(models.Account{Email: email}).Assign(acc).FirstOrCreate(&acc)
		imported++
	}

	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok { process(m) }
		}
	case map[string]interface{}:
		process(v)
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("Imported %d accounts", imported)})
}

func ExportAccounts(c *gin.Context) {
	var accounts []models.Account
	database.DB.Where("status = ?", "active").Find(&accounts)
	c.Header("Content-Disposition", "attachment; filename=evo_accounts_export.json")
	c.JSON(200, accounts)
}

func getString(m map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" { return v }
	}
	return ""
}
