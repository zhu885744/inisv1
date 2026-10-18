package timer

import (
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/spf13/cast"
)

// QpsWarnCleanStruct - QPS 告警记录清理
type QpsWarnCleanStruct struct{}

var QpsWarnClean *QpsWarnCleanStruct

// Run - 每小时清理一次过期的 QPS 告警记录
//
// 为什么需要：每次超限都会往 qps_warn 写一条，只增不减，
// 表会随站点访问量无限增长；而这张表是限流封禁的事后排查依据，
// 保留最近 7 天足够定位问题，更早的没有保留价值。
func (this *QpsWarnCleanStruct) Run() {
	_ = Timer.Every(1).Hour().Do(func() {
		if !tryLock("qps-warn-clean", 50*time.Minute) {
			return
		}
		cleanQpsWarn()
	})
}

// cleanQpsWarn - 真删 7 天前的告警记录
func cleanQpsWarn() {
	if facade.DB == nil {
		return
	}

	cutoff := time.Now().Add(-7 * 24 * time.Hour).Unix()

	list, _ := facade.DB.Model(&model.QpsWarn{}).Where("create_time", "<", cutoff).Column("id")
	ids := cast.ToIntSlice(list)
	if len(ids) == 0 {
		return
	}

	// 分批删除：一次删太多会长时间占用连接，也容易把慢查询拖出来
	const batch = 500
	for start := 0; start < len(ids); start += batch {
		end := start + batch
		if end > len(ids) {
			end = len(ids)
		}

		// Force：模型带软删除，这里要真正清掉，否则表只增不减
		if _, err := facade.DB.Model(&model.QpsWarn{}).Force().Delete(ids[start:end]); err != nil {
			facade.Log.Error(map[string]any{
				"error": err.Error(),
				"from":  start,
				"to":    end,
			}, "清理过期 QPS 告警记录失败")
			return
		}
	}

	facade.Log.Info(map[string]any{
		"count":  len(ids),
		"cutoff": cutoff,
	}, "清理过期 QPS 告警记录")
}
