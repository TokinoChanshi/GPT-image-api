package core

import (
	"evo-image-api/database"
	"evo-image-api/models"
	"log"
	"time"
)

type Detector struct {
	Interval time.Duration
}

func NewDetector(interval time.Duration) *Detector {
	return &Detector{
		Interval: interval,
	}
}

// Start 启动后台探测任务
func (d *Detector) Start() {
	ticker := time.NewTicker(d.Interval)
	go func() {
		for range ticker.C {
			d.ScanAllAccounts()
		}
	}()
}

// ScanAllAccounts 扫描所有账号并更新 IMG2 状态
func (d *Detector) ScanAllAccounts() {
	var accounts []models.Account
	err := database.DB.Where("status = ?", "active").Find(&accounts).Error
	if err != nil {
		log.Printf("[Detector] Failed to fetch accounts: %v", err)
		return
	}

	for _, acc := range accounts {
		d.ScanAccount(&acc)
	}
}

// ScanAccount 扫描单个账号
func (d *Detector) ScanAccount(acc *models.Account) {
	client, err := NewOpenAIClient(acc)
	if err != nil {
		log.Printf("[Detector] Failed to init client for %s: %v", acc.Email, err)
		return
	}

	hasIMG2, err := client.CheckCapability()
	if err != nil {
		log.Printf("[Detector] Check failed for %s: %v", acc.Email, err)
		return
	}

	if acc.HasIMG2 != hasIMG2 {
		database.DB.Model(acc).Update("has_img2", hasIMG2)
		log.Printf("[Detector] Account %s IMG2 status updated to: %v", acc.Email, hasIMG2)
	}
}
