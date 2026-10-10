package timer

import (
	"inis/app/model"
)

// StatsStruct 数据统计相关定时任务
type StatsStruct struct{}

var Stats *StatsStruct

// Run - 资源采样与告警判定
func (this *StatsStruct) Run() {

	// 按 stats.sample_interval（默认 60 秒）采集一次系统资源，并做告警阈值判定。
	//
	// 这里刻意**不判断有没有管理员在线**（与 socket 的实时推送相反）：
	// 阈值告警必须在没人打开后台时也照常判定，否则告警就失去意义。
	// 单次成本只有 3 个 syscall + 一次配置读取，不落库、不查各表 COUNT。
	_ = Timer.Every(uint64(model.StatsSampleInterval())).Seconds().Do(model.SampleStats)
}
