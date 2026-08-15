package timer

import (
	"fmt"
	"inis/app/model"
	"time"

	"inis/app/facade"

	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
)

type BanStruct struct{}

var Ban *BanStruct

func (this *BanStruct) Run() {
	// 每分钟执行一次自动解封检查
	_ = Timer.Every(1).Minute().Do(autoUnban)
}

// autoUnban 自动解封到期用户
func autoUnban() {
	// 数据库未初始化（未安装）时跳过
	if facade.DB == nil {
		return
	}

	now := time.Now().Unix()

	// 查找所有到期但未解封的封禁记录（expires_at > 0 且 <= 当前时间，且状态为生效中）
	var records []model.UserBanRecords
	_, err := facade.DB.Model(&records).Where("expires_at", ">", 0).Where("expires_at", "<=", now).Where("status", model.BanStatusActive).Select()
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "自动解封查询失败")
		return
	}

	if utils.Is.Empty(records) {
		return
	}

	for _, record := range records {
		// 恢复用户状态（若封禁时同时冻结了用户，则一并恢复为正常）
		userUpdate := map[string]any{
			"current_ban_id": 0,
			"restrictions":   0,
		}
		if record.FreezeUser == 1 {
			userUpdate["status"] = model.UserStatusNormal
		}

		// 事务：更新封禁记录状态 + 恢复用户状态
		err = facade.DB.Drive().Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&model.UserBanRecords{}).Where("id", record.Id).Updates(map[string]any{
				"status":     model.BanStatusExpired,
				"unban_time": now,
			}).Error; err != nil {
				return err
			}
			return tx.Model(&model.Users{}).Where("id", record.Uid).Updates(userUpdate).Error
		})
		if err != nil {
			facade.Log.Error(map[string]any{"record_id": record.Id, "error": err}, "自动解封失败")
			continue
		}

		// 清除用户缓存
		facade.Cache.Del(fmt.Sprintf("user[%v]", record.Uid))

		// 审计日志
		facade.Log.Info(map[string]any{
			"uid":       record.Uid,
			"record_id": record.Id,
			"duration":  record.Duration,
		}, "封禁到期自动解封")
	}
}
