package timer

import (
	"fmt"
	"inis/app/model"
	"time"

	"inis/app/facade"

	"github.com/unti-io/go-utils/utils"
)

type BanStruct struct{}

var Ban *BanStruct

func (this *BanStruct) Run() {
	// 每分钟执行一次自动解封检查
	_ = Timer.Every(1).Minute().Do(autoUnban)
}

// autoUnban 自动解封到期用户
func autoUnban() {
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
		// 更新封禁记录状态为已解封
		_, err := facade.DB.Model(&model.UserBanRecords{}).Where("id", record.Id).Update(map[string]any{
			"status":     model.BanStatusExpired,
			"unban_time": now,
		})
		if err != nil {
			facade.Log.Error(map[string]any{"record_id": record.Id, "error": err}, "自动解封更新记录失败")
			continue
		}

		// 恢复用户状态
		_, err = facade.DB.Model(&model.Users{}).Where("id", record.Uid).Update(map[string]any{
			"current_ban_id": 0,
			"restrictions":   0,
		})
		if err != nil {
			facade.Log.Error(map[string]any{"uid": record.Uid, "error": err}, "自动解封更新用户失败")
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
