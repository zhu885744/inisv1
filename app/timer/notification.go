package timer

import (
	"inis/app/facade"
	"inis/app/model"
	"time"

	"github.com/spf13/cast"
)

type NotificationStruct struct{}

var Notification *NotificationStruct

func (this *NotificationStruct) Run() {
	// 每天凌晨 04:00:00 执行一次通知过期清理。
	// 多实例下抢锁执行：清理是幂等的，但没必要让每个实例都扫一遍全表。
	_ = Timer.Every(1).Day().At("04:00:00").Do(func() {
		if !tryLock("notification-clean", 10*time.Minute) {
			return
		}
		cleanExpiredNotifications()
	})
}

// cleanExpiredNotifications 清理过期的已读通知与广播通知
func cleanExpiredNotifications() {
	// 数据库未初始化（未安装）时跳过
	if facade.DB == nil {
		return
	}

	// 保留天数，从配置读取，默认 30 天
	days := cast.ToInt(facade.AppToml.Get("notification.retention_days", 30))
	if days <= 0 {
		days = 30
	}

	cleaned, err := (&model.Notification{}).CleanExpired(days)
	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "days": days}, "通知过期清理失败")
		return
	}

	facade.Log.Info(map[string]any{"cleaned": cleaned, "days": days}, "通知过期清理完成")
}
