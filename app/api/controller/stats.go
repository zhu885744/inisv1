package controller

import (
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// ============================== 数据统计（资源告警阈值） ==============================
//
// 页面上的实时数据不在这里：那部分由 WebSocket 主动推送（见 app/socket/controller/status.go 与 docs/socket.md）。
// 本控制器只负责 CPU / 内存 / 磁盘告警阈值的读取与保存（存 config 表 SYSTEM_ALERT_THRESHOLD 的 json 字段），
// 阈值判定发生在服务端的定时采样里（app/timer/stats.go → model.SampleStats）。
type Stats struct {
	base
}

// IGET - 只读查询
func (this *Stats) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"alert": this.alert,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IPOST - 保存告警阈值
func (this *Stats) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"alert": this.saveAlert,
	}

	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IPUT / IDEL - 无对应场景（历史数据由定时任务维护）
func (this *Stats) IPUT(ctx *gin.Context) { this.deny(ctx) }
func (this *Stats) IDEL(ctx *gin.Context) { this.deny(ctx) }

func (this *Stats) deny(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "统计数据不支持该操作！"), 405)
}

func (this *Stats) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// alert - 读取告警阈值
func (this *Stats) alert(ctx *gin.Context) {
	this.json(ctx, model.StatsAlertConfig(), facade.Lang(ctx, "查询成功！"), 200)
}

// saveAlert - 保存告警阈值（仅超级管理员）
//
// 值存在 config 表 SYSTEM_ALERT_THRESHOLD 的 json 字段，与「邮件通知」等配置共用一套存储，
// 因此写入后必须清掉 config[KEY] 缓存，否则要等缓存过期才生效（读侧见 model.StatsAlertConfig）。
func (this *Stats) saveAlert(ctx *gin.Context) {
	if !this.meta.root(ctx) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限！"), 403)
		return
	}

	params := this.params(ctx, map[string]any{})

	config := model.StatsAlertDefaultConfig()
	for key := range config {
		if value, ok := params[key]; ok && !utils.Is.Empty(value) {
			config[key] = cast.ToInt(value)
		}
	}

	// 收敛取值范围：开关 0/1；阈值 1~100；冷却至少 60 秒
	// （冷却填 0 会变成「每分钟都发一条告警」，对管理员是灾难）
	config["enabled"] = statsClamp(cast.ToInt(config["enabled"]), 0, 1)
	for _, key := range []string{"cpu", "mem", "disk"} {
		config[key] = statsClamp(cast.ToInt(config[key]), 1, 100)
	}
	config["cooldown"] = statsClamp(cast.ToInt(config["cooldown"]), 60, 86400)

	jsonValue := utils.Json.Encode(config)
	now := time.Now().Unix()

	exist, _ := facade.DB.Model(&model.Config{}).WithTrashed().Where("key", model.StatsAlertConfigKey).Exist()

	var err error
	if exist {
		if _, err = facade.DB.Model(&model.Config{}).WithTrashed().
			Where("key", model.StatsAlertConfigKey).
			Update(map[string]any{"json": jsonValue, "update_time": now}); err != nil {
			this.json(ctx, nil, err.Error(), 400)
			return
		}
	} else {
		table := model.Config{
			Key:        model.StatsAlertConfigKey,
			Json:       jsonValue,
			Remark:     "资源告警阈值（数据统计页）",
			CreateTime: now,
			UpdateTime: now,
		}
		_, err = facade.DB.Model(&table).Create(&table)
	}

	if err != nil {
		this.json(ctx, nil, err.Error(), 400)
		return
	}

	facade.Cache.Del("config[" + model.StatsAlertConfigKey + "]")
	facade.Cache.DelTags([]any{"[GET]", "config"})

	this.json(ctx, config, facade.Lang(ctx, "保存成功！"), 200)
}

// statsClamp 把数值收敛到 [min, max]
func statsClamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
