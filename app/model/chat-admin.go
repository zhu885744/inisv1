package model

import (
	"errors"
	"strings"
	"time"

	"inis/app/facade"
)

/**
 * 私信管理（后台）
 *
 * 与前台 chat 的区别：
 *   - 前台只能看「与我有关」的会话，后台需要看「任意两人」的会话；
 *   - 后台不受「用户按人删除」的影响（chat_message_deletes 只是用户侧视图），
 *     也不会把用户的已读状态改掉；
 *   - 后台撤回不限 2 分钟、不限发送者，但仍保留「撤回」语义（清空内容 + 双方看到已撤回），
 *     不做物理删除 —— 聊天记录本身也是互相的凭证，删掉会让双方对不上。
 *
 * 权限边界不在这里做：接口在 auth_rules 里登记为独立权限点（chat-admin），
 * 由中间件按权限组校验，只有被授予该权限点的账号才能调用。
 */

// AdminList - 全部私信会话（按最后消息时间倒序）
//
//	uid     只看某个用户参与的会话（命中 uid / target_uid 任一即可）
//	keyword 按最后一条消息摘要模糊匹配
func (this *ChatConversations) AdminList(page, limit, uid int, keyword string) ([]ChatConversations, int64) {

	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	where := "1 = 1"
	args := make([]any, 0, 4)

	if uid > 0 {
		where += " AND (uid = ? OR target_uid = ?)"
		args = append(args, uid, uid)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		where += " AND last_text LIKE ?"
		args = append(args, "%"+keyword+"%")
	}

	db := facade.DB.Drive()

	var count int64
	db.Model(&ChatConversations{}).Where(where, args...).Count(&count)

	var rows []ChatConversations
	_ = db.Where(where, args...).
		Order("last_time desc").
		Limit(limit).
		Offset((page - 1) * limit).
		Find(&rows).Error

	return rows, count
}

// AdminConversation - 单个会话（后台右侧面板的标题用）
func (this *ChatConversations) AdminConversation(id int) map[string]any {

	if id <= 0 {
		return map[string]any{}
	}

	var row ChatConversations
	if err := facade.DB.Drive().Where("id = ?", id).First(&row).Error; err != nil || row.Id <= 0 {
		return map[string]any{}
	}

	return map[string]any{
		"id":          row.Id,
		"uid":         row.Uid,
		"target_uid":  row.TargetUid,
		"last_msg_id": row.LastMsgId,
		"last_text":   row.LastText,
		"last_uid":    row.LastUid,
		"last_time":   row.LastTime,
		"create_time": row.CreateTime,
	}
}

// AdminHistory - 某会话的消息（倒序取一页、正序返回）
//
// 与前台 History 的差别：这里不做 chatHiddenQuery 过滤 ——
// 用户自己「删除」只是把他的视图藏起来，消息仍在库里，后台要能看到完整记录。
func (this *ChatMessages) AdminHistory(conversationId, beforeId, limit int) (list []map[string]any, hasMore bool) {

	if conversationId <= 0 {
		return []map[string]any{}, false
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	db := facade.DB.Drive().Model(&ChatMessages{}).Where("conversation_id = ?", conversationId)
	if beforeId > 0 {
		db = db.Where("id < ?", beforeId)
	}

	var rows []ChatMessages
	if err := db.Order("id desc").Limit(limit + 1).Find(&rows).Error; err != nil {
		return []map[string]any{}, false
	}

	hasMore = len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	list = make([]map[string]any, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		list = append(list, chatMessageMap(&rows[i]))
	}

	return list, hasMore
}

// AdminRevoke - 后台撤回（不限时间、不限发送者）
//
// 只清内容并置撤回标记：双方界面看到「撤回了一条消息」，会话摘要同步回退；
// 不物理删除，避免同一段对话在两人那里对不上。
func (this *ChatMessages) AdminRevoke(msgId int) (map[string]any, error) {

	if msgId <= 0 {
		return nil, errors.New("参数不完整！")
	}

	var message ChatMessages
	if err := facade.DB.Drive().Where("id = ?", msgId).First(&message).Error; err != nil || message.Id <= 0 {
		return nil, errors.New("消息不存在！")
	}
	if message.IsRevoked == 1 {
		return nil, errors.New("消息已经撤回了！")
	}

	now := time.Now().Unix()
	if err := facade.DB.Drive().Model(&ChatMessages{}).Where("id = ?", message.Id).Updates(map[string]any{
		"is_revoked":  1,
		"content":     "",
		"revoke_time": now,
	}).Error; err != nil {
		return nil, errors.New("撤回失败，请稍后重试！")
	}

	message.IsRevoked = 1
	message.Content = ""
	message.RevokeTime = now

	// 撤回的若是最后一条，会话摘要要退回到上一条还在的消息
	(&ChatMessages{}).refreshConversationSummary(message.ConversationId)

	return chatMessageMap(&message), nil
}
