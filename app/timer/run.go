package timer

import (
	"inis/app/facade"
	"inis/app/model"
	"sync"
	"time"

	"github.com/jasonlvhit/gocron"
)

var Timer *gocron.Scheduler

// runOnce 保证定时任务只注册一次
//
// app.toml 变更时 main 会 shutdown + InitApp + run 重新拉起服务，
// 而 Timer 是包级全局的：不做保护的话每保存一次配置就会把所有任务再注册一遍，
// 同一条解封 / 告警逻辑会被执行 2 次、3 次……，goroutine 与内存也跟着涨。
var runOnce sync.Once

func init() {
	Timer = gocron.NewScheduler()
}

func Run() {

	// 尚未完成安装（没有数据库配置）时不启动任何定时任务：
	// 这些任务都要读库，而 facade.DB 此时为空，会直接空指针 panic，
	// 表现为「服务刚打印完『服务已启动』就整体崩掉」。
	//
	// 注意这里**不消耗 runOnce**：安装完成后同一进程内再次 Run() 仍能正常注册任务。
	if !facade.Installed() {
		facade.Log.Info(map[string]any{}, "尚未完成安装，已跳过定时任务")
		return
	}

	runOnce.Do(start)
}

func start() {

	// 启动时执行一次的维护任务：
	// 1. 补齐缺失的权限规则（新增接口在旧库里没有规则会被中间件判为「需权限点」）
	// 2. 纠正历史脏数据（规则类型 root → default）
	// 3. 清理已下线接口残留的权限规则（见 model.PruneAuthRules）
	// 以上三步都由 EnsureAuthRules 内部串起来执行。
	// 多实例同时重启时会抢锁：规则补齐本身幂等，但没必要 N 个实例同时写规则表。
	go func() {
		if !tryLock("auth-rules", 10*time.Minute) {
			return
		}
		model.EnsureAuthRules()
	}()

	Log.Run()
	Device.Run()
	Ban.Run()
	Notification.Run()
	Decoration.Run()
	Stats.Run()
	QpsWarnClean.Run()

	go func() {
		<-Timer.Start()
	}()
}
