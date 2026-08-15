package timer

import (
	"github.com/jasonlvhit/gocron"
)

var Timer *gocron.Scheduler

func init() {
	Timer = gocron.NewScheduler()
}

func Run() {

	Log.Run()
	Device.Run()
	Ban.Run()
	Notification.Run()

	go func() {
		<- Timer.Start()
	}()
}