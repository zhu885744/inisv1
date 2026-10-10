package model

import (
	"fmt"
	"math"
	"sync"
	"time"

	"inis/app/facade"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

// ============================== 资源采样与告警阈值 ==============================
//
// 用途：后台「数据统计」页配置的 CPU / 内存 / 磁盘 告警阈值。
//
// 采样由 app/timer/stats.go 按 stats.sample_interval（默认 60 秒）触发，
// **不判断有没有管理员在线** —— 阈值判定必须 24 小时持续进行，
// 否则就成了「没打开后台就收不到告警」，那这个功能没意义。
//
// 采样结果只在内存里过一遍（不落库）：页面上的实时曲线走 socket 推送
// （见 app/socket/controller/status.go），历史数据没有保留。
//
// 成本：3 个 syscall + 一次配置读取，没有 COUNT 查询。
type StatsSample struct {
	Cpu  float64
	Mem  float64
	Disk float64
	At   time.Time
}

// CollectStatsSample 采集一份系统资源样本
//
// 注意 cpu.Percent(0, false) 返回的是「距上一次调用」的平均使用率，
// 语义由调用间隔决定（默认 60 秒 = 最近一分钟的平均值）；
// 进程内首次调用必然为 0，从第二轮采样起就正常了，这是预期行为。
func CollectStatsSample() StatsSample {
	memInfo, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	diskInfo, _ := disk.Usage("/")

	sample := StatsSample{At: time.Now()}

	if memInfo != nil {
		sample.Mem = round2(memInfo.UsedPercent)
	}
	if diskInfo != nil {
		sample.Disk = round2(diskInfo.UsedPercent)
	}
	if len(cpuPercent) > 0 {
		sample.Cpu = round2(cpuPercent[0])
	}

	return sample
}

// SampleStats 采集一次并做阈值判定（定时任务的入口）
func SampleStats() {
	// 采样异常绝不能把定时任务线程带崩
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "资源采样失败")
		}
	}()

	checkStatsAlert(CollectStatsSample())
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// statsConfig 读 [stats] 段配置；老库没有这一段时用默认值，不需要改配置
func statsConfig(key string, defaultValue any) any {
	return facade.AppToml.Get("stats."+key, defaultValue)
}

// StatsSampleInterval 采样间隔（秒，默认 60；最小 10 秒兜底 —— 采样太密没有意义）
func StatsSampleInterval() int {
	seconds := cast.ToInt(statsConfig("sample_interval", 60))
	if seconds < 10 {
		seconds = 10
	}
	return seconds
}

// ---------- 告警阈值 ----------

// StatsAlertConfigKey 告警阈值在 config 表的 key（值存在 json 字段）
const StatsAlertConfigKey = "SYSTEM_ALERT_THRESHOLD"

// StatsAlertDefaultConfig 默认阈值（默认关闭，避免老库突然开始发通知）
func StatsAlertDefaultConfig() map[string]any {
	return map[string]any{
		"enabled":  0,    // 0 关闭 / 1 开启
		"cpu":      90,   // CPU 使用率阈值(%)
		"mem":      90,   // 内存使用率阈值(%)
		"disk":     90,   // 磁盘使用率阈值(%)
		"cooldown": 1800, // 同一类告警的最小通知间隔(秒)
	}
}

// StatsAlertMergeConfig 用默认值补齐缺失字段（老库、只存了部分字段都能用）
func StatsAlertMergeConfig(config map[string]any) map[string]any {
	result := StatsAlertDefaultConfig()
	for key := range result {
		if value, ok := config[key]; ok && !utils.Is.Empty(value) {
			result[key] = cast.ToInt(value)
		}
	}
	return result
}

// StatsAlertConfig 读取告警阈值（带缓存，缓存名与 config 控制器写入时清理的键一致）
func StatsAlertConfig() map[string]any {
	cacheName := "config[" + StatsAlertConfigKey + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache.Has(cacheName) {
		return StatsAlertMergeConfig(cast.ToStringMap(facade.Cache.Get(cacheName)))
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", StatsAlertConfigKey).Find()
	config := cast.ToStringMap(cast.ToStringMap(item)["json"])

	if cacheState {
		go facade.Cache.Set(cacheName, config)
	}

	return StatsAlertMergeConfig(config)
}

// 告警冷却记在进程内存里（不依赖缓存服务，缓存没开也要能防抖）
var (
	statsAlertMu     sync.Mutex
	statsAlertLastAt = map[string]int64{}
)

// checkStatsAlert 检查一份采样是否越过阈值，命中则通知管理员
//
// 两个防抖手段缺一不可：
//   - 冷却期（默认 30 分钟）内同一类告警不重复通知；
//   - 站内信 + 邮件双通道，站内信无开关（落库即投递），邮件受「系统设置 → 邮件通知」的 stats.alert 场景开关控制。
func checkStatsAlert(sample StatsSample) {
	defer func() {
		if err := recover(); err != nil {
			facade.Log.Error(map[string]any{"error": err}, "统计阈值告警检查失败")
		}
	}()

	config := StatsAlertConfig()
	if cast.ToInt(config["enabled"]) != 1 {
		return
	}

	cooldown := cast.ToInt64(config["cooldown"])
	if cooldown <= 0 {
		cooldown = 1800
	}

	now := time.Now().Unix()
	checks := []struct {
		key       string
		label     string
		value     float64
		threshold float64
	}{
		{"cpu", "CPU 使用率", sample.Cpu, cast.ToFloat64(config["cpu"])},
		{"mem", "内存使用率", sample.Mem, cast.ToFloat64(config["mem"])},
		{"disk", "磁盘使用率", sample.Disk, cast.ToFloat64(config["disk"])},
	}

	for _, item := range checks {
		if item.threshold <= 0 || item.value < item.threshold {
			continue
		}

		statsAlertMu.Lock()
		last := statsAlertLastAt[item.key]
		if last > 0 && now-last < cooldown {
			statsAlertMu.Unlock()
			continue
		}
		statsAlertLastAt[item.key] = now
		statsAlertMu.Unlock()

		title := "资源告警：" + item.label
		content := fmt.Sprintf("%s 达到 %.2f%%，超过阈值 %.2f%%（采样时间 %s）",
			item.label, item.value, item.threshold, sample.At.Format("2006-01-02 15:04:05"))

		facade.Log.Warn(map[string]any{
			"metric":    item.key,
			"value":     item.value,
			"threshold": item.threshold,
		}, title)

		// 站内信：无开关，落库即投递，保证管理员不会漏看
		NotifyAdmins(0, NotificationTypeSystem, title, content, "", 0)

		// 邮件：走队列发送（非阻塞），场景关闭时内部会静默跳过。
		// 站点行用 MailNotifySiteLine()：取不到站点地址时返回空串（会被 sendMailNotify 跳过），
		// 避免正文里出现「站点：」这样只有标签的悬空行。
		go MailNotifyAdmin("stats.alert", "[资源告警] "+title,
			content,
			MailNotifySiteLine(),
		)
	}
}
