package controller

import (
	"encoding/json"
)

// Online - 在线状态控制器
//
// 注意：hub.clients 只允许 hub goroutine 读写，这里的每个方法都不能直接遍历它，
// 一律读 clientIDs()（hub 维护的只读快照），否则会与 hub goroutine 形成
// 数据竞争（并发读写 map 会直接 fatal）。
type Online struct {
	base
}

// GetOnlineUsers - 获取在线用户列表
func (this Online) GetOnlineUsers() []string {
	return Hub.clientIDs()
}

// GetOnlineCount - 获取在线用户数量
func (this Online) GetOnlineCount() int {
	return len(Hub.clientIDs())
}

// IsUserOnline - 检查用户是否在线
func (this Online) IsUserOnline(clientId string) bool {
	for _, id := range Hub.clientIDs() {
		if id == clientId {
			return true
		}
	}
	return false
}

// BroadcastOnlineStatus - 请求广播一次在线状态
//
// 不能直接调用 hub.broadcastStatus（它会遍历 clients，只能由 hub goroutine 执行），
// 这里投递一个刷新请求，由 hub goroutine 标脏、随定时任务合并推送。
func (this Online) BroadcastOnlineStatus() {
	select {
	case Hub.refresh <- struct{}{}:
	default:
	}
}

// PushOnlineStatus - 推送在线状态给指定客户端
func (this Online) PushOnlineStatus(clientId string) {
	ids := Hub.clientIDs()

	statusMsg, _ := json.Marshal(map[string]any{
		"type": "single",
		"to":   clientId,
		"data": map[string]any{
			"online_users": ids,
			"online_count": len(ids),
		},
	})

	// 走 hub 的投递通道，不直接读写 clients
	select {
	case Hub.notice <- statusMsg:
	default:
	}
}
