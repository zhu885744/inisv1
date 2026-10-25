package timer

import (
	"fmt"
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

// runSafely - 启动维护任务的兜底执行
//
// 这些任务跑在独立 goroutine 里，panic 不会被 gin 的 Recovery 捕获：
// 一旦 DB / Cache 尚未就绪（如安装向导刚写完配置、任务抢在初始化完成前执行），
// 整个进程会直接退出（现象：打印完「服务已启动」立刻 SIGSEGV）。
// 维护任务本身幂等，失败下次启动会重试，不能让服务跟着一起挂掉。
func runSafely(name string, fn func()) {
	defer func() {
		if err := recover(); err != nil {
			if facade.Log != nil {
				facade.Log.Error(map[string]any{
					"task":  name,
					"error": err,
				}, "启动维护任务发生panic，已跳过")
			} else {
				fmt.Printf("启动维护任务 %s 发生panic：%v\n", name, err)
			}
		}
	}()

	fn()
}

func start() {

	// 启动时执行一次的维护任务：
	// 0. 补齐系统内置权限分组（超级管理员 / 普通用户，见 model.EnsureAuthGroups）
	// 1. 补齐缺失的权限规则（新增接口在旧库里没有规则会被中间件判为「需权限点」）
	// 2. 纠正历史脏数据（规则类型 root → default）
	// 3. 清理已下线接口残留的权限规则（见 model.PruneAuthRules）
	// 第 1~3 步由 EnsureAuthRules 内部串起来执行。
	// 多实例同时重启时会抢锁：补齐本身幂等，但没必要 N 个实例同时写权限表。
	go runSafely("auth-rules", func() {
		if !tryLock("auth-rules", 10*time.Minute) {
			return
		}
		model.EnsureAuthGroups()
		model.EnsureAuthRules()
	})

	// 启动时补扫一次勋章：新装 / 刚上线勋章功能的老站，存量用户（注册满 N 天、
	// 已有 N 篇文章）不会自己触发事件，只能靠这次扫描（以及之后的每小时扫描）补发。
	go runSafely("badge-scan", func() {
		if !tryLock("badge-scan", 55*time.Minute) {
			return
		}
		model.BadgeScan()
	})

	Log.Run()
	Device.Run()
	Ban.Run()
	Notification.Run()
	Decoration.Run()
	Badge.Run()
	Stats.Run()
	QpsWarnClean.Run()

	go func() {
		<-Timer.Start()
	}()
}
