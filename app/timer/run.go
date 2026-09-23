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

	// 启动时执行一次的维护任务：纠正权限规则历史脏数据（type=root → default）
	go model.NormalizeAuthRuleTypes()

	Log.Run()
	Device.Run()
	Ban.Run()
	Notification.Run()

	go func() {
		<- Timer.Start()
	}()
}