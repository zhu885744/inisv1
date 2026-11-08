package timer

import (
	"time"

	"inis/app/model"
	socketctrl "inis/app/socket/controller"
)

// RoomStruct - 公共聊天室的维护任务
type RoomStruct struct{}

var Room *RoomStruct

// Run - 公共聊天室自动清空
//
// 调度频率固定为每小时，是否真正清空由配置决定：
//
//	now - last_time >= days * 86400
//
// 为什么不按「管理员选的间隔」直接注册 cron：间隔是可变的，改配置后还要重建
// 调度器；而「每小时检查一次」既能在配置变化后立刻生效，也能在服务重启后
// 依据持久化的 last_time 自动补上错过的清理。
func (this *RoomStruct) Run() {

	_ = Timer.Every(1).Hour().Do(func() {
		// 多实例抢锁：清空是全表删除，绝不能让 N 个实例同时执行
		if !tryLock("room-auto-clear", 55*time.Minute) {
			return
		}
		clearRoomIfDue()
	})

	// 启动时补一次：停机期间到期的清理，起来后马上执行
	go runSafely("room-auto-clear", func() {
		if !tryLock("room-auto-clear-boot", 10*time.Minute) {
			return
		}
		clearRoomIfDue()
	})
}

// clearRoomIfDue - 到期则清空，并通知在线用户把本地时间线一起清掉
func clearRoomIfDue() {

	cleared := model.RoomAutoClearTick()
	if cleared <= 0 {
		return
	}

	// 在线页面收到后清空消息列表；离线用户下次进房间拉历史时自然是空的。
	// 类型必须以 room 开头：socket 的 dispatch 只广播 room* 前缀的事件。
	socketctrl.BroadcastRoom(map[string]any{
		"type": "room_clear",
		"data": map[string]any{
			"count": cleared,
			"time":  time.Now().Unix(),
		},
	})
}
