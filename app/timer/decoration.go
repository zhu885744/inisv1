package timer

import (
	"fmt"
	"inis/app/facade"
	"inis/app/model"
	"time"

	"github.com/spf13/cast"
)

// DecorationStruct - 装扮有效期定时任务
type DecorationStruct struct{}

var Decoration *DecorationStruct

// decorationScanLimit - 单轮最多处理多少条，避免一次性拉爆消息与数据库
const decorationScanLimit = 200

func (this *DecorationStruct) Run() {
	// 每小时检查一次装扮到期情况
	_ = Timer.Every(1).Hour().Do(decorationExpireScan)
}

// decorationExpireScan - 装扮有效期扫描
//
// 处理两件事：
//  1. 已过期：状态置为「已过期」，若正在佩戴则先自动卸下（否则头像框 / 头衔会继续显示
//     一个已经失效的装扮），并发站内通知；
//  2. 即将过期（DecorationExpireWarnDays 天内）：站内提醒一次 ——
//     ExpireNotified 标记防重复（续期时会被重置，所以新一轮有效期仍会提醒）。
func decorationExpireScan() {
	// 数据库未初始化（未安装）时跳过
	if facade.DB == nil {
		return
	}

	now := time.Now().Unix()

	expiredDecorationBatch(now)
	warnDecorationBatch(now)
}

// expiredDecorationBatch - 处理已过期的装扮
func expiredDecorationBatch(now int64) {
	var records []model.UserDecoration
	if _, err := facade.DB.Model(&records).
		Where("expire_time", ">", 0).
		Where("expire_time", "<=", now).
		Where("status", "!=", model.UserDecorationStatusExpired).
		Limit(decorationScanLimit).
		Select(); err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "装扮过期扫描失败")
		return
	}

	if len(records) == 0 {
		return
	}

	names := decorationNameMap(records)

	for _, record := range records {
		name := decorationNameOf(names, record.DecorationId)

		// 佩戴中的过期装扮先卸下：卸下会同时清理 users 上的渲染字段
		if model.IsWearingDecoration(record.Uid, record.DecorationId) {
			if err := model.UnwearDecoration(record.Uid, record.Type); err != nil {
				facade.Log.Warn(map[string]any{
					"uid":           record.Uid,
					"decoration_id": record.DecorationId,
					"error":         err.Error(),
				}, "过期装扮自动卸下失败")
			}
		}

		if err := facade.DB.Drive().Model(&model.UserDecoration{}).
			Where("id", record.Id).
			UpdateColumn("status", model.UserDecorationStatusExpired).Error; err != nil {
			facade.Log.Error(map[string]any{
				"uid":           record.Uid,
				"decoration_id": record.DecorationId,
				"error":         err.Error(),
			}, "装扮过期状态更新失败")
			continue
		}

		model.SendDecorationNotify(record.Uid, "装扮已过期："+name,
			"已于 "+model.DecorationExpireText(record.ExpireTime)+
				"到期并自动卸下 · 如需继续使用可到装扮商城重新获取",
			record.DecorationId)
	}
}

// warnDecorationBatch - 即将过期提醒（提前 DecorationExpireWarnDays 天，只提醒一次）
func warnDecorationBatch(now int64) {
	deadline := now + model.DecorationExpireWarnDays*86400

	var records []model.UserDecoration
	if _, err := facade.DB.Model(&records).
		Where("expire_time", ">", now).
		Where("expire_time", "<=", deadline).
		Where("expire_notified", 0).
		Where("status", "!=", model.UserDecorationStatusExpired).
		Limit(decorationScanLimit).
		Select(); err != nil {
		facade.Log.Error(map[string]any{"error": err.Error()}, "装扮即将过期扫描失败")
		return
	}

	if len(records) == 0 {
		return
	}

	names := decorationNameMap(records)

	for _, record := range records {
		name := decorationNameOf(names, record.DecorationId)

		// 剩余天数向上取整：还剩几小时也算「1 天」
		days := int((record.ExpireTime - now + 86399) / 86400)

		model.SendDecorationNotify(record.Uid, "装扮即将过期："+name,
			fmt.Sprintf("剩余 %d 天（%s）· 到期后会自动卸下，可到装扮商城重新获取",
				days, model.DecorationExpireText(record.ExpireTime)),
			record.DecorationId)

		// 标记已提醒：避免每小时重复打扰；续期时该标记会被重置
		if err := facade.DB.Drive().Model(&model.UserDecoration{}).
			Where("id", record.Id).
			UpdateColumn("expire_notified", 1).Error; err != nil {
			facade.Log.Warn(map[string]any{
				"uid":           record.Uid,
				"decoration_id": record.DecorationId,
				"error":         err.Error(),
			}, "装扮即将过期提醒标记写入失败")
		}
	}
}

// decorationNameMap - 批量取装扮名称（一次查询，避免逐条查库）
func decorationNameMap(records []model.UserDecoration) map[int]string {
	result := map[int]string{}

	ids := make([]int, 0, len(records))
	seen := map[int]bool{}
	for _, record := range records {
		if record.DecorationId > 0 && !seen[record.DecorationId] {
			seen[record.DecorationId] = true
			ids = append(ids, record.DecorationId)
		}
	}
	if len(ids) == 0 {
		return result
	}

	var list []model.Decoration
	rows, err := facade.DB.Model(&list).Where("id", "in", ids).Select()
	if err != nil {
		facade.Log.Warn(map[string]any{"error": err.Error()}, "装扮名称批量查询失败")
		return result
	}

	for _, row := range rows {
		result[cast.ToInt(row["id"])] = cast.ToString(row["name"])
	}

	return result
}

// decorationNameOf - 取装扮名（装扮被彻底删除时给个兜底文案）
func decorationNameOf(names map[int]string, id int) string {
	if name, ok := names[id]; ok && name != "" {
		return name
	}
	return fmt.Sprintf("装扮 #%d", id)
}
