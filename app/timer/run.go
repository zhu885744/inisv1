package timer

import (
	"inis/app/facade"
	"inis/app/model"

	"github.com/jasonlvhit/gocron"
)

var Timer *gocron.Scheduler

func init() {
	Timer = gocron.NewScheduler()
}

func Run() {

	// 尚未完成安装（没有数据库配置）时不启动任何定时任务：
	// 这些任务都要读库，而 facade.DB 此时为空，会直接空指针 panic，
	// 表现为「服务刚打印完『服务已启动』就整体崩掉」。
	if !facade.Installed() {
		facade.Log.Info(map[string]any{}, "尚未完成安装，已跳过定时任务")
		return
	}

	// 启动时执行一次的维护任务：
	// 1. 补齐缺失的权限规则（新增接口在旧库里没有规则会被中间件判为「需权限点」）
	// 2. 纠正历史脏数据（规则类型 root → default）
	// 3. 清理已下线接口残留的权限规则（见 model.PruneAuthRules）
	// 以上三步都由 EnsureAuthRules 内部串起来执行
	go model.EnsureAuthRules()

	Log.Run()
	Device.Run()
	Ban.Run()
	Notification.Run()
	Decoration.Run()
	Stats.Run()

	go func() {
		<- Timer.Start()
	}()
}