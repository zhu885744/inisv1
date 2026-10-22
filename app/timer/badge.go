package timer

import (
	"inis/app/model"
	"time"
)

// BadgeStruct - 勋章自动发放的定时兜底
type BadgeStruct struct{}

var Badge *BadgeStruct

// Run - 每小时扫描一次「满足条件但还没发」的用户
//
// 为什么需要定时任务：
//   - 注册满 N 天这类条件没有业务触发点，只能靠时间轮询；
//   - 数量类条件（发布 N 篇）平时在发布动作后由 model.BadgeAutoGrant 立即判定，
//     但「先有数据、后建勋章」的历史用户不会自己触发，扫一遍补齐。
//
// 多实例下抢锁：发放本身是幂等的，但每条都要写库，没必要 N 个实例同时扫。
func (this *BadgeStruct) Run() {
	_ = Timer.Every(1).Hour().Do(func() {
		if !tryLock("badge-scan", 55*time.Minute) {
			return
		}
		model.BadgeScan()
	})

	// 限定勋章到期提醒：每小时看一眼「N 天内要过期」的记录，
	// UserBadge.ExpireNotified 保证同一条只提醒一次
	_ = Timer.Every(1).Hour().Do(func() {
		if !tryLock("badge-expire-warn", 55*time.Minute) {
			return
		}
		model.BadgeExpireWarn()
	})
}
