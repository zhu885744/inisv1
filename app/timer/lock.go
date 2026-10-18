package timer

import (
	"inis/app/facade"
	"time"
)

// tryLock 抢占定时任务的分布式锁
//
// 背景：定时任务由每个实例本地的 gocron 各自调度，多实例部署时同一时刻
// 会有 N 个实例同时执行同一个任务。对「改库 + 发通知」这类全局维护任务
// （自动解封、装扮过期扫描、通知清理）会直接导致重复通知 / 重复处理。
//
// 实现：用缓存的 SET NX 抢锁，只有抢到的实例才执行。
//   - redis 驱动：真正的跨实例互斥；
//   - file / ram 驱动：退化为单实例语义（与改造前行为一致，不会更差）。
//
// ttl 必须略小于任务间隔：抢到锁的实例即使崩溃，锁也会自动过期，
// 不会出现「锁一直不释放、任务再也不执行」。
//
// 注意：不要给「采集本机指标」类的任务加锁（stats / device / 日志重建），
// 那些任务本来就需要每个实例各跑一份。
func tryLock(name string, ttl time.Duration) bool {
	if facade.Cache == nil {
		// 缓存未初始化：不做限制，保持原有行为
		return true
	}

	if ttl <= 0 {
		ttl = time.Minute
	}

	return facade.Cache.SetNX("timer:lock:"+name, time.Now().Unix(), ttl)
}
