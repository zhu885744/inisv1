package controller

import (
	"encoding/json"
)

// Online - 在线状态控制器
type Online struct {
	base
}

// GetOnlineUsers - 获取在线用户列表
func (this Online) GetOnlineUsers() []string {
	onlineUsers := []string{}
	for clientId := range Hub.clients {
		onlineUsers = append(onlineUsers, clientId)
	}
	return onlineUsers
}

// GetOnlineCount - 获取在线用户数量
func (this Online) GetOnlineCount() int {
	return len(Hub.clients)
}

// IsUserOnline - 检查用户是否在线
func (this Online) IsUserOnline(clientId string) bool {
	_, ok := Hub.clients[clientId]
	return ok
}

// BroadcastOnlineStatus - 广播在线状态
func (this Online) BroadcastOnlineStatus() {
	Hub.broadcastStatus()
}

// PushOnlineStatus - 推送在线状态给指定客户端
func (this Online) PushOnlineStatus(clientId string) {
	onlineUsers := []string{}
	for id := range Hub.clients {
		onlineUsers = append(onlineUsers, id)
	}

	statusMsg, _ := json.Marshal(map[string]any{
		"type": "status",
		"content": map[string]any{
			"online_users": onlineUsers,
			"online_count": len(onlineUsers),
		},
	})

	if client, ok := Hub.clients[clientId]; ok {
		client.send <- statusMsg
	}
}
