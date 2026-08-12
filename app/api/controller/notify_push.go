package controller

import (
	"encoding/json"
	"inis/app/model"
	socket "inis/app/socket/controller"
	"strings"

	"github.com/spf13/cast"
)

// PushNotification 通过WebSocket推送通知给指定用户
func PushNotification(uid int, notif *model.Notification) {
	if notif == nil || uid <= 0 {
		return
	}

	targetId := "user_" + cast.ToString(uid)

	data := map[string]any{
		"id":          notif.Id,
		"type":        notif.Type,
		"title":       notif.Title,
		"content":     notif.Content,
		"bind_id":     notif.BindId,
		"bind_type":   notif.BindType,
		"from_uid":    notif.FromUid,
		"create_time": notif.CreateTime,
	}

	msg, err := json.Marshal(map[string]any{
		"type":    "notification",
		"to":      targetId,
		"content": data,
	})

	if err != nil {
		return
	}

	// 通过Hub单播推送给目标用户
	socket.Hub.PushNotice(msg)
}

// truncateSafe 安全截断文本，保留 [emoji: URL] 格式的完整性
// 避免截断 emoji 标签导致显示异常
func truncateSafe(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}

	truncated := string(runes[:maxLen])

	// 查找截断结果中最后一个未闭合的 [emoji: 标签
	lastStart := strings.LastIndex(truncated, "[emoji:")
	if lastStart == -1 {
		return truncated + "..."
	}

	// 检查该标签是否已闭合（之后是否有 ]）
	if strings.Contains(truncated[lastStart:], "]") {
		return truncated + "..."
	}

	// 未闭合，在原始文本中找到对应的 ] 并扩展到完整标签
	closeIdx := strings.Index(text[lastStart:], "]")
	if closeIdx != -1 {
		return text[:lastStart+closeIdx+1] + "..."
	}

	return truncated + "..."
}
