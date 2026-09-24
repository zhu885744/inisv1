package timer

import (
	"inis/app/model"

	"github.com/jasonlvhit/gocron"
)

var Timer *gocron.Scheduler

func init() {
	Timer = gocron.NewScheduler()
}

func Run() {

	// 启动时执行一次的维护任务：
	// 1. 补齐缺失的权限规则（新增接口在旧库里没有规则会被中间件判为「需权限点」）
	// 2. 纠正历史脏数据（规则类型 root → default）
	go model.EnsureAuthRules()

	Log.Run()
	Device.Run()
	Ban.Run()
	Notification.Run()

	go func() {
		<- Timer.Start()
	}()
}