package core

import (
	"evo-image-api/database"
	"evo-image-api/models"
	"fmt"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	accountLocks sync.Map // [uint]bool
)

type Scheduler struct {
	DB *gorm.DB
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		DB: database.DB,
	}
}

// GetAvailableAccount 获取一个可用的账号
func (s *Scheduler) GetAvailableAccount(requireIMG2 bool) (*models.Account, error) {
	var accounts []models.Account
	now := time.Now()
	query := s.DB.Where("status = ? AND (usage_count < usage_limit OR next_reset_at < ?)", "active", now)
	
	if requireIMG2 {
		query = query.Where("has_img2 = ?", true)
	}

	err := query.Order("usage_count asc").Find(&accounts).Error
	if err != nil {
		return nil, err
	}

	for _, acc := range accounts {
		// 使用 sync.Map 实现简单的本地排他锁
		if _, loaded := accountLocks.LoadOrStore(acc.ID, true); !loaded {
			return &acc, nil
		}
	}

	return nil, fmt.Errorf("no available accounts in pool")
}

// ReleaseAccount 释放账号锁
func (s *Scheduler) ReleaseAccount(accID uint) {
	accountLocks.Delete(accID)
}

// UpdateAccountUsage 更新账号使用情况
func (s *Scheduler) UpdateAccountUsage(accID uint) {
	s.DB.Model(&models.Account{}).Where("id = ?", accID).UpdateColumn("usage_count", gorm.Expr("usage_count + 1"))
}
