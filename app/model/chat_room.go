package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"inis/app/facade"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/soft_delete"
)

/**
 * 公共聊天室
 *
 * 与私信（chat.go）是两套结构，刻意不复用 chat_messages：
 *   - 私信是「点对点」，未读 / 已读 / 会话摘要都围绕两个人算；
 *   - 聊天室是「一对多」，没有接收者、没有逐人已读，只有一条全局时间线，
 *     多出来的只是「谁发的」「管理员撤回了谁」「谁被禁言了」。
 * 混在一张表里会让私信的未读统计到处加排除条件，得不偿失。
 *
 * 房间只有 id=1 一个（PublicRoomId）：全站公共，所有登录用户都能发言。
 * 消息的 seq 仍复用自增主键 id，前端按 id 去重 / 断线后 after_seq 补拉。
 */

const (
	// PublicRoomId - 公共聊天室的固定 ID：前后端都直接写死它，不需要「按名字找房间」
	PublicRoomId = 1

	// ChatRoomRevokeWindow - 本人撤回自己消息的时间窗（秒），与私信保持一致
	ChatRoomRevokeWindow = 120

	// ChatRoomMuteForever - 禁言到期的特殊值：0 表示永久
	ChatRoomMuteForever = 0

	// ChatMuteScopeRoom / ChatMuteScopeDM - 禁言作用域（后台名单展示用）
	ChatMuteScopeRoom = "room"
	ChatMuteScopeDM   = "dm"

	// ChatMuteDM - 「私信禁言」复用同一张禁言表：room_id = 0 表示私信作用域
	//
	// 为什么不另建一张「封禁表」：禁言本来就有到期时间 / 原因 / 操作人这套字段，
	// 私信只是换了个作用域。单独造一张表会出现两套解除口径、两个后台名单，
	// 处置时要分别记两笔。这里只把作用域折进 room_id（聊天室 = 1，私信 = 0）。
	ChatMuteDM = 0
)

// ChatRooms - 聊天室
type ChatRooms struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Name        string `gorm:"size:60; default:'公共聊天室'; comment:名称;" json:"name"`
	Description string `gorm:"size:255; default:''; comment:简介;" json:"description"`
	Avatar      string `gorm:"size:255; default:''; comment:头像;" json:"avatar"`
	// Status - 1 正常 / 0 关闭（关闭后只能看历史，不能发言）
	Status int `gorm:"type:tinyint(1); default:1; comment:状态;" json:"status"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// ChatRoomMessages - 聊天室消息（一条全局时间线）
type ChatRoomMessages struct {
	Id      int    `gorm:"type:int(32); comment:主键;" json:"id"`
	RoomId  int    `gorm:"type:int(32); index:idx_chat_room_messages_room; comment:房间ID;" json:"room_id"`
	Uid     int    `gorm:"type:int(32); index:idx_chat_room_messages_uid; comment:发送者;" json:"uid"`
	MsgType string `gorm:"size:16; default:'text'; comment:消息类型 text/image;" json:"msg_type"`
	Content string `gorm:"size:2000; comment:消息内容;" json:"content"`
	// IsRevoked - 撤回标记：内容清空但保留这一行，时间线不会凭空少一条
	IsRevoked  int   `gorm:"type:tinyint(1); default:0; comment:是否已撤回;" json:"is_revoked"`
	RevokeUid  int   `gorm:"type:int(32); default:0; comment:撤回操作人（本人或管理员）;" json:"revoke_uid"`
	RevokeTime int64 `gorm:"comment:撤回时间;" json:"revoke_time"`
	// ClientMsgId - 幂等键：连点 / 网络重试 / socket 重发都靠它去重
	ClientMsgId string `gorm:"size:64; default:''; comment:客户端消息ID（幂等）;" json:"client_msg_id"`
	// 引用回复：与私信同一套约定，存被引用消息的快照（撤回后引用内容依然可读）
	QuoteId   int    `gorm:"type:int(32); default:0; comment:引用的消息ID;" json:"quote_id"`
	QuoteUid  int    `gorm:"type:int(32); default:0; comment:被引用消息的发送者;" json:"quote_uid"`
	QuoteType string `gorm:"size:16; default:''; comment:被引用消息的类型;" json:"quote_type"`
	QuoteText string `gorm:"size:300; default:''; comment:被引用消息的摘要（快照）;" json:"quote_text"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// ChatRoomMessageDeletes - 聊天室消息的「按人删除」记录
//
// 公共时间线是所有人的共同记录，右键「删除」只应影响操作者自己：
// 这里记一条 (uid, msg_id)，该用户的历史 / 补拉会过滤掉它；
// 其它人（包括同账号的其它设备，靠 socket 推送同步）完全无感。
type ChatRoomMessageDeletes struct {
	Id         int   `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid        int   `gorm:"type:int(32); uniqueIndex:uk_chat_room_message_deletes,priority:1; comment:删除者;" json:"uid"`
	MsgId      int   `gorm:"type:int(32); uniqueIndex:uk_chat_room_message_deletes,priority:2; comment:消息ID;" json:"msg_id"`
	CreateTime int64 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
}

// ChatRoomMutes - 聊天室禁言记录
//
// 一次禁言一条记录，解除时软删除；Active 只认「未删除 + 未到期」的那条。
// 不加唯一索引：同一个人可以被反复禁言 / 解禁，历史记录留着便于排查。
type ChatRoomMutes struct {
	Id          int    `gorm:"type:int(32); comment:主键;" json:"id"`
	RoomId      int    `gorm:"type:int(32); index:idx_chat_room_mutes_room; comment:房间ID;" json:"room_id"`
	Uid         int    `gorm:"type:int(32); index:idx_chat_room_mutes_uid; comment:被禁言用户;" json:"uid"`
	Until       int64  `gorm:"comment:解禁时间戳（0=永久）;" json:"until"`
	Reason      string `gorm:"size:120; default:''; comment:禁言原因;" json:"reason"`
	OperatorUid int    `gorm:"type:int(32); default:0; comment:操作人;" json:"operator_uid"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// 迁移

func InitChatRooms() {
	if err := facade.DB.Drive().AutoMigrate(&ChatRooms{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatRooms表迁移失败")
	}
	// 建完表立刻保证 id=1 存在：前端写死了 /room/1，房间缺了整页就是空的
	(&ChatRooms{}).Ensure()
}

func InitChatRoomMessages() {
	if err := facade.DB.Drive().AutoMigrate(&ChatRoomMessages{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatRoomMessages表迁移失败")
	}
}

func InitChatRoomMutes() {
	if err := facade.DB.Drive().AutoMigrate(&ChatRoomMutes{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatRoomMutes表迁移失败")
	}
}

func InitChatRoomMessageDeletes() {
	if err := facade.DB.Drive().AutoMigrate(&ChatRoomMessageDeletes{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatRoomMessageDeletes表迁移失败")
	}
}

// chatRoomHiddenQuery - 我在聊天室里删除过的消息 id 子查询
func chatRoomHiddenQuery(uid int) *gorm.DB {
	return facade.DB.Drive().Model(&ChatRoomMessageDeletes{}).
		Select("msg_id").Where("uid = ?", uid)
}

// Ensure - 幂等创建公共聊天室（已有 id=1 时什么都不做）
func (this *ChatRooms) Ensure() {

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", PublicRoomId).First(&room).Error; err == nil && room.Id > 0 {
		return
	}

	seed := ChatRooms{
		Id:          PublicRoomId,
		Name:        "公共聊天室",
		Description: "全站用户都在这儿，随便聊点什么",
		Status:      1,
	}

	if err := facade.DB.Drive().Create(&seed).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err}, "公共聊天室初始化失败")
	}
}

// Info - 房间信息（公开字段，未登录也能看）
func (this *ChatRooms) Info() map[string]any {

	var room ChatRooms
	if err := facade.DB.Drive().Where("id = ?", PublicRoomId).First(&room).Error; err != nil || room.Id <= 0 {
		return map[string]any{}
	}

	return map[string]any{
		"id":          room.Id,
		"name":        room.Name,
		"description": room.Description,
		"avatar":      room.Avatar,
		"status":      room.Status,
		"create_time": room.CreateTime,
		"update_time": room.UpdateTime,
	}
}

// Update - 修改房间资料（仅超级管理员，权限判断在控制器里做）
func (this *ChatRooms) Update(params map[string]any) (map[string]any, error) {

	updates := map[string]any{}

	if value, ok := params["name"]; ok {
		name := strings.TrimSpace(cast.ToString(value))
		if name == "" {
			return nil, errors.New("聊天室名称不能为空！")
		}
		if len([]rune(name)) > 30 {
			return nil, errors.New("聊天室名称不能超过 30 个字！")
		}
		updates["name"] = name
	}

	if value, ok := params["description"]; ok {
		description := strings.TrimSpace(cast.ToString(value))
		if len([]rune(description)) > 200 {
			return nil, errors.New("聊天室简介不能超过 200 个字！")
		}
		updates["description"] = description
	}

	if value, ok := params["avatar"]; ok {
		avatar := strings.TrimSpace(cast.ToString(value))
		if avatar != "" && !isRoomMediaUrl(avatar) {
			return nil, errors.New("头像地址不合法！")
		}
		updates["avatar"] = avatar
	}

	if len(updates) == 0 {
		return nil, errors.New("没有需要修改的内容！")
	}

	updates["update_time"] = time.Now().Unix()

	if err := facade.DB.Drive().Model(&ChatRooms{}).Where("id = ?", PublicRoomId).Updates(updates).Error; err != nil {
		return nil, errors.New("修改失败，请稍后重试！")
	}

	return (&ChatRooms{}).Info(), nil
}

// Active - 该用户当前是否被禁言，以及解禁时间（0 = 永久）
//
// roomId 传 PublicRoomId 查聊天室禁言，传 ChatMuteDM（0）查私信禁言。
func (this *ChatRoomMutes) Active(roomId, uid int) (muted bool, until int64) {

	if roomId < 0 || uid <= 0 {
		return false, 0
	}

	var row ChatRoomMutes
	if err := facade.DB.Drive().
		Where("room_id = ? AND uid = ?", roomId, uid).
		Order("id desc").First(&row).Error; err != nil {
		return false, 0
	}

	// 到点自动失效：不依赖定时任务去改数据，读的时候判断一次最省事
	if row.Until > 0 && row.Until <= time.Now().Unix() {
		return false, 0
	}

	return true, row.Until
}

// Mute - 禁言（until=0 表示永久）
//
// 先把同一用户还没解除的旧记录物理删掉，保证「生效中的记录只有一条」，
// 免得多条记录叠在一起，解禁时还要猜该删哪一条。
func (this *ChatRoomMutes) Mute(roomId, uid int, until int64, reason string, operatorUid int) error {

	if roomId < 0 || uid <= 0 {
		return errors.New("参数错误！")
	}
	if uid == operatorUid {
		return errors.New("不能禁言自己！")
	}
	if until > 0 && until <= time.Now().Unix() {
		return errors.New("解禁时间必须晚于当前时间！")
	}

	db := facade.DB.Drive()
	db.Unscoped().Where("room_id = ? AND uid = ?", roomId, uid).Delete(&ChatRoomMutes{})

	row := ChatRoomMutes{
		RoomId:      roomId,
		Uid:         uid,
		Until:       until,
		Reason:      strings.TrimSpace(reason),
		OperatorUid: operatorUid,
	}

	if err := db.Create(&row).Error; err != nil {
		return errors.New("禁言失败，请稍后重试！")
	}

	return nil
}

// Unmute - 解除禁言
func (this *ChatRoomMutes) Unmute(roomId, uid int) error {

	if roomId < 0 || uid <= 0 {
		return errors.New("参数错误！")
	}

	if err := facade.DB.Drive().
		Where("room_id = ? AND uid = ?", roomId, uid).
		Delete(&ChatRoomMutes{}).Error; err != nil {
		return errors.New("解除禁言失败，请稍后重试！")
	}

	return nil
}

// List - 生效中的禁言列表（管理面板用）
func (this *ChatRoomMutes) List(roomId int) []map[string]any {

	result := make([]map[string]any, 0)
	if roomId <= 0 {
		return result
	}

	var rows []ChatRoomMutes
	if err := facade.DB.Drive().
		Where("room_id = ?", roomId).
		Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		return result
	}

	now := time.Now().Unix()

	for _, row := range rows {
		// 已到期的记录不再展示（数据留着，下次有人查再说，不做后台清理）
		if row.Until > 0 && row.Until <= now {
			continue
		}
		result = append(result, map[string]any{
			"id":           row.Id,
			"uid":          row.Uid,
			"until":        row.Until,
			"reason":       row.Reason,
			"operator_uid": row.OperatorUid,
			"create_time":  row.CreateTime,
		})
	}

	return result
}

// ListAll - 生效中的禁言（不区分作用域，后台「禁言名单」用）
//
// 每行带上 scope：room_id = ChatMuteDM(0) 的是私信禁言，其余是聊天室禁言。
func (this *ChatRoomMutes) ListAll() []map[string]any {

	result := make([]map[string]any, 0)

	var rows []ChatRoomMutes
	if err := facade.DB.Drive().
		Order("id desc").Limit(200).Find(&rows).Error; err != nil {
		return result
	}

	now := time.Now().Unix()

	for _, row := range rows {
		if row.Until > 0 && row.Until <= now {
			continue
		}

		scope := ChatMuteScopeRoom
		if row.RoomId == ChatMuteDM {
			scope = ChatMuteScopeDM
		}

		result = append(result, map[string]any{
			"id":           row.Id,
			"uid":          row.Uid,
			"room_id":      row.RoomId,
			"scope":        scope,
			"until":        row.Until,
			"reason":       row.Reason,
			"operator_uid": row.OperatorUid,
			"create_time":  row.CreateTime,
		})
	}

	return result
}

// ChatMuteScopeName - 禁言作用域的中文名（后台名单展示用）
func ChatMuteScopeName(scope string) string {
	if scope == ChatMuteScopeDM {
		return "私信"
	}
	return "聊天室"
}

// ChatRoomMessageItem - 消息对外结构
//
// seq 与 id 同值：前端靠它去重、按 after_seq 补拉（与私信同一套约定）。
func ChatRoomMessageItem(item *ChatRoomMessages) map[string]any {

	if item == nil {
		return map[string]any{}
	}

	return map[string]any{
		"id":            item.Id,
		"seq":           item.Id,
		"room_id":       item.RoomId,
		"uid":           item.Uid,
		"msg_type":      item.MsgType,
		"content":       item.Content,
		"is_revoked":    item.IsRevoked,
		"revoke_uid":    item.RevokeUid,
		"revoke_time":   item.RevokeTime,
		"client_msg_id": item.ClientMsgId,
		"quote":         chatRoomQuoteMap(item),
		"create_time":   item.CreateTime,
	}
}

// chatRoomQuoteMap - 引用消息的输出结构（没有引用时为 nil）
func chatRoomQuoteMap(item *ChatRoomMessages) any {
	if item.QuoteId <= 0 {
		return nil
	}
	return map[string]any{
		"id":       item.QuoteId,
		"uid":      item.QuoteUid,
		"msg_type": utils.Default(item.QuoteType, "text"),
		"content":  item.QuoteText,
	}
}

// Send - 发一条聊天室消息
//
// quoteId > 0 表示引用当前聊天室里的某条消息：服务端回查原消息并落快照
// （存在 / 未撤回），不信任客户端传来的引用内容。
func (this *ChatRoomMessages) Send(uid int, msgType, content, clientMsgId string, quoteId int) (map[string]any, error) {

	if uid <= 0 {
		return nil, errors.New("请先登录后再发言！")
	}

	room := (&ChatRooms{}).Info()
	if cast.ToInt(room["id"]) <= 0 {
		return nil, errors.New("聊天室不存在！")
	}
	if cast.ToInt(room["status"]) != 1 {
		return nil, errors.New("聊天室已关闭，暂时不能发言！")
	}

	if muted, until := (&ChatRoomMutes{}).Active(PublicRoomId, uid); muted {
		if until > 0 {
			return nil, fmt.Errorf("你已被禁言，%s 后自动解除！", time.Unix(until, 0).Format("01-02 15:04"))
		}
		return nil, errors.New("你已被禁言，暂时不能发言！")
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("消息内容不能为空！")
	}
	if len([]rune(content)) > 2000 {
		return nil, errors.New("消息内容太长了！")
	}

	if msgType == "" {
		msgType = "text"
	}
	if msgType != "text" && msgType != "image" {
		return nil, errors.New("不支持的消息类型！")
	}
	if msgType == "image" && !isRoomMediaUrl(content) {
		return nil, errors.New("图片地址不合法！")
	}

	// 敏感词：聊天室实时性高、扩散快，按 block / limit 拦截；
	// review / warn / log 放行 —— 放行的消息在入库后留一条「命中片段」作为证据，
	// 后台「聊天审核」据此判断违规，并可配合撤回 / 禁言处置。
	sensitiveHit := SensitiveDecision{}
	if msgType != "image" {
		sensitiveHit = (&Sensitive{}).Decide(SceneChat, uid, content)
		if sensitiveHit.Blocked() {
			return nil, errors.New(sensitiveHit.UserMessage())
		}
	}

	db := facade.DB.Drive()

	// 幂等：同一个 client_msg_id 重复提交（连点 / 重试）直接返回已存在的那条
	if clientMsgId != "" {
		var exist ChatRoomMessages
		if err := db.Where("room_id = ? AND uid = ? AND client_msg_id = ?", PublicRoomId, uid, clientMsgId).
			First(&exist).Error; err == nil && exist.Id > 0 {
			return ChatRoomMessageItem(&exist), nil
		}
	}

	// 引用：只接受同一聊天室里、未被撤回的消息
	var quote ChatRoomMessages
	if quoteId > 0 {
		if err := db.Where("room_id = ? AND id = ?", PublicRoomId, quoteId).First(&quote).Error; err != nil || quote.Id <= 0 {
			return nil, errors.New("引用的消息不存在！")
		}
		if quote.IsRevoked == 1 {
			return nil, errors.New("引用的消息已撤回！")
		}
	}

	row := ChatRoomMessages{
		RoomId:      PublicRoomId,
		Uid:         uid,
		MsgType:     msgType,
		Content:     content,
		ClientMsgId: clientMsgId,
	}

	if quote.Id > 0 {
		row.QuoteId = quote.Id
		row.QuoteUid = quote.Uid
		row.QuoteType = utils.Default(quote.MsgType, "text")
		row.QuoteText = chatQuoteText(quote.MsgType, quote.Content)
	}

	if err := db.Create(&row).Error; err != nil {
		return nil, errors.New("发送失败，请稍后重试！")
	}

	// 命中词库留证：只存命中片段，不存整条消息（后台「聊天审核」据此判断违规）
	if sensitiveHit.Action != "" && sensitiveHit.Action != ActionAllow {
		NoteChatModeration(SceneChat, uid, 0, content, sensitiveHit, row.Id)
	}

	return ChatRoomMessageItem(&row), nil
}

// History - 往前翻历史（beforeId=0 表示取最新一页，返回值按时间正序）
func (this *ChatRoomMessages) History(uid, beforeId, limit int) (list []map[string]any, hasMore bool) {

	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}

	db := facade.DB.Drive().Model(&ChatRoomMessages{}).
		Where("room_id = ?", PublicRoomId).
		// 我删除过的消息不再出现在我的历史里（删除只对自己生效）
		Where("id NOT IN (?)", chatRoomHiddenQuery(uid))
	if beforeId > 0 {
		db = db.Where("id < ?", beforeId)
	}

	var rows []ChatRoomMessages
	if err := db.Order("id desc").Limit(limit + 1).Find(&rows).Error; err != nil {
		return []map[string]any{}, false
	}

	hasMore = len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	list = make([]map[string]any, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		list = append(list, ChatRoomMessageItem(&rows[i]))
	}

	return list, hasMore
}

// Sync - 断线补拉：id > afterSeq 的消息（正序）
func (this *ChatRoomMessages) Sync(uid int, afterSeq int64, limit int) (list []map[string]any, lastSeq int64, hasMore bool) {

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if afterSeq < 0 {
		afterSeq = 0
	}

	var rows []ChatRoomMessages
	err := facade.DB.Drive().Model(&ChatRoomMessages{}).
		Where("room_id = ?", PublicRoomId).
		Where("id NOT IN (?)", chatRoomHiddenQuery(uid)).
		Where("id > ?", afterSeq).
		Order("id asc").Limit(limit + 1).Find(&rows).Error
	if err != nil {
		return []map[string]any{}, afterSeq, false
	}

	hasMore = len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	list = make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, ChatRoomMessageItem(&rows[i]))
	}

	lastSeq = afterSeq
	if len(rows) > 0 {
		lastSeq = int64(rows[len(rows)-1].Id)
	}

	return list, lastSeq, hasMore
}

// Revoke - 撤回一条消息
//
// 权限：消息作者本人（2 分钟内）或超级管理员（不限时间、不限作者）。
// 撤回只清内容并打标记，不删行 —— 时间线不能凭空少一条，别人会以为看错了。
func (this *ChatRoomMessages) Revoke(msgId, operatorUid int, isAdmin bool) (map[string]any, error) {

	if msgId <= 0 || operatorUid <= 0 {
		return nil, errors.New("参数错误！")
	}

	db := facade.DB.Drive()

	var row ChatRoomMessages
	if err := db.Where("room_id = ? AND id = ?", PublicRoomId, msgId).First(&row).Error; err != nil || row.Id <= 0 {
		return nil, errors.New("消息不存在或已被删除！")
	}

	if row.IsRevoked == 1 {
		return ChatRoomMessageItem(&row), nil
	}

	if !isAdmin {
		if row.Uid != operatorUid {
			return nil, errors.New("只能撤回自己的消息！")
		}
		if time.Now().Unix()-row.CreateTime > ChatRoomRevokeWindow {
			return nil, errors.New("超过 2 分钟的消息不能撤回！")
		}
	}

	now := time.Now().Unix()
	if err := db.Model(&ChatRoomMessages{}).Where("id = ?", row.Id).Updates(map[string]any{
		"is_revoked":  1,
		"revoke_uid":  operatorUid,
		"revoke_time": now,
		"content":     "",
	}).Error; err != nil {
		return nil, errors.New("撤回失败，请稍后重试！")
	}

	row.IsRevoked = 1
	row.RevokeUid = operatorUid
	row.RevokeTime = now
	row.Content = ""

	return ChatRoomMessageItem(&row), nil
}

// Delete - 删除一条消息（只对自己生效）
//
// 与 Revoke 的区别：
//   - 撤回：清空内容并广播给所有人，时间线上留一条「撤回」提示；
//   - 删除：只写一条 (uid, msg_id) 隐藏记录，别人完全看不到变化。
//
// 公共聊天室里所有登录用户都能看到每条消息，所以不校验作者：
// 任何人都可以把任意一条从「自己的」视图里删掉。
func (this *ChatRoomMessages) Delete(uid, msgId int) error {

	if uid <= 0 || msgId <= 0 {
		return errors.New("参数不完整！")
	}

	var row ChatRoomMessages
	if err := facade.DB.Drive().
		Where("room_id = ? AND id = ?", PublicRoomId, msgId).
		First(&row).Error; err != nil || row.Id <= 0 {
		return errors.New("消息不存在！")
	}

	record := ChatRoomMessageDeletes{Uid: uid, MsgId: msgId}
	if err := facade.DB.Drive().Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": uid, "msg_id": msgId}, "删除聊天室消息失败")
		return errors.New("删除失败，请稍后重试！")
	}

	return nil
}

// ---------- 自动清空 ----------

const (
	// RoomAutoClearConfigKey - 自动清空配置在 config 表的 key（值存在 json 字段）
	RoomAutoClearConfigKey = "ROOM_AUTO_CLEAR"
	// RoomAutoClearDefaultDays - 默认每 3 天（72 小时）清空一次
	RoomAutoClearDefaultDays = 3
	// roomAutoClearDaySeconds - 一天的秒数
	roomAutoClearDaySeconds int64 = 86400
)

// RoomAutoClearOptions - 后台可选的清空间隔（天）
//
// 182 天按「半年」展示、365 天按「一年」展示（文案在前端）。
// 默认值 3 天（72 小时）也在选项里：否则初始值与下拉选项对不上，
// 后台会显示成「未选择」。
func RoomAutoClearOptions() []int {
	return []int{1, 3, 7, 30, 182, 365}
}

// RoomAutoClearDefaultConfig - 默认配置：每 3 天清空一次，尚未清空过
//
// last_time 是「上次清空时间」（秒级时间戳）：
//   - 定时任务按 last_time + days 判断是否到期，重启 / 多实例都不会重置节奏；
//   - 为 0 表示还没开始计时（首次启动会把它填成当前时间，避免一上线就误删历史）。
func RoomAutoClearDefaultConfig() map[string]any {
	return map[string]any{
		"days":       RoomAutoClearDefaultDays,
		"last_time":  int64(0),
		"last_count": int64(0),
	}
}

// RoomAutoClearMergeConfig - 用默认值补齐并收敛非法值（老库 / 手工改库都能用）
func RoomAutoClearMergeConfig(config map[string]any) map[string]any {
	result := RoomAutoClearDefaultConfig()

	days := cast.ToInt(config["days"])
	if !roomAutoClearDayAllowed(days) {
		days = RoomAutoClearDefaultDays
	}
	result["days"] = days
	result["last_time"] = cast.ToInt64(config["last_time"])
	result["last_count"] = cast.ToInt64(config["last_count"])

	return result
}

// roomAutoClearDayAllowed - 只接受下拉里提供的间隔，避免配置被改成 0 天（每次检查都清空）
func roomAutoClearDayAllowed(days int) bool {
	for _, item := range RoomAutoClearOptions() {
		if item == days {
			return true
		}
	}
	return false
}

// RoomAutoClearConfig - 读取自动清空配置（带缓存，缓存键与 config 控制器的写入约定一致）
func RoomAutoClearConfig() map[string]any {
	cacheName := "config[" + RoomAutoClearConfigKey + "]"
	cacheState := cast.ToBool(facade.CacheToml.Get("open"))

	if cacheState && facade.Cache != nil && facade.Cache.Has(cacheName) {
		return RoomAutoClearMergeConfig(cast.ToStringMap(facade.Cache.Get(cacheName)))
	}

	item, _ := facade.DB.Model(&Config{}).Where("key", RoomAutoClearConfigKey).Find()
	config := cast.ToStringMap(cast.ToStringMap(item)["json"])

	if cacheState && facade.Cache != nil {
		go facade.Cache.Set(cacheName, config)
	}

	return RoomAutoClearMergeConfig(config)
}

// RoomAutoClearInfo - 后台展示结构：当前间隔 + 上次 / 下次清空时间
func RoomAutoClearInfo() map[string]any {
	config := RoomAutoClearConfig()

	days := cast.ToInt(config["days"])
	lastTime := cast.ToInt64(config["last_time"])

	nextTime := int64(0)
	if lastTime > 0 {
		nextTime = lastTime + int64(days)*roomAutoClearDaySeconds
	}

	return map[string]any{
		"days":       days,
		"last_time":  lastTime,
		"last_count": cast.ToInt64(config["last_count"]),
		"next_time":  nextTime,
		"options":    RoomAutoClearOptions(),
		"interval":   int64(days) * roomAutoClearDaySeconds,
	}
}

// SaveRoomAutoClear - 保存清空间隔（天），返回最新配置
//
// 保留 last_time：换间隔只影响「从上次清空起算」的下一轮，
// 不会因为管理员保存一次就无条件清空 / 重新计时。
func SaveRoomAutoClear(days int) (map[string]any, error) {

	if !roomAutoClearDayAllowed(days) {
		return nil, errors.New("不支持的清空间隔！")
	}

	config := RoomAutoClearConfig()
	config["days"] = days

	if err := saveRoomAutoClearConfig(config); err != nil {
		return nil, errors.New("保存失败，请稍后重试！")
	}

	return RoomAutoClearInfo(), nil
}

// saveRoomAutoClearConfig - 写回 config 表并清缓存（配置读取有缓存，不清理会读到旧间隔）
func saveRoomAutoClearConfig(config map[string]any) error {

	jsonValue := utils.Json.Encode(config)
	now := time.Now().Unix()

	exist, _ := facade.DB.Model(&Config{}).WithTrashed().
		Where("key", RoomAutoClearConfigKey).Exist()

	var err error
	if exist {
		_, err = facade.DB.Model(&Config{}).WithTrashed().
			Where("key", RoomAutoClearConfigKey).
			Update(map[string]any{"json": jsonValue, "update_time": now})
	} else {
		table := Config{
			Key:        RoomAutoClearConfigKey,
			Json:       jsonValue,
			Remark:     "公共聊天室自动清空（间隔天数 / 上次清空时间）",
			CreateTime: now,
			UpdateTime: now,
		}
		_, err = facade.DB.Model(&table).Create(&table)
	}

	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "key": RoomAutoClearConfigKey}, "保存聊天室自动清空配置失败")
		return err
	}

	if facade.Cache != nil {
		facade.Cache.Del("config[" + RoomAutoClearConfigKey + "]")
		facade.Cache.DelTags([]any{"[GET]", "config"})
	}

	return nil
}

// Clear - 物理删除公共聊天室的全部消息（自动清空任务调用）
//
// 用 DELETE 而不是 TRUNCATE：TRUNCATE 会把自增 id 重置为 1，
// 而前端把消息 id 当 seq 游标（after_seq / before_id）。id 归零后，
// 本地还留着旧消息的客户端会认为「新消息都比我的水位小」，补拉永远拿不到。
// DELETE 保留自增序列，新消息 id 始终大于历史 id，游标语义不会被破坏。
func (this *ChatRoomMessages) Clear() (int64, error) {

	if facade.DB == nil {
		return 0, errors.New("数据库尚未就绪！")
	}

	tx := facade.DB.Drive().Unscoped().
		Where("room_id = ?", PublicRoomId).
		Delete(&ChatRoomMessages{})
	if tx.Error != nil {
		return 0, tx.Error
	}

	// 消息都清空了，隐藏记录已无意义：一并清掉，避免长期累积一堆指向不存在消息的行
	facade.DB.Drive().Unscoped().
		Where("1 = 1").
		Delete(&ChatRoomMessageDeletes{})

	return tx.RowsAffected, nil
}

// RoomAutoClearTick - 定时任务的单次检查：到期就清空，返回本次删除的条数
//
// 间隔以「天」为单位，调度本身每小时跑一次；这样管理员改间隔后无需重启，
// 也不会像固定 cron 那样把配置变化漏掉。多实例通过 timer 的分布式锁串行执行。
func RoomAutoClearTick() int64 {

	if facade.DB == nil {
		return 0
	}

	config := RoomAutoClearConfig()
	days := cast.ToInt(config["days"])
	lastTime := cast.ToInt64(config["last_time"])
	now := time.Now().Unix()

	// 首次启用：从现在开始计时，避免一上线就把已有的聊天记录清掉
	if lastTime <= 0 {
		config["last_time"] = now
		_ = saveRoomAutoClearConfig(config)
		return 0
	}

	if now-lastTime < int64(days)*roomAutoClearDaySeconds {
		return 0
	}

	cleared, err := (&ChatRoomMessages{}).Clear()
	if err != nil {
		facade.Log.Error(map[string]any{"error": err}, "公共聊天室自动清空失败")
		return 0
	}

	config["last_time"] = now
	config["last_count"] = cleared
	if err := saveRoomAutoClearConfig(config); err != nil {
		// 配置没写回去也没关系：下一小时会再清一次（此刻表已空，代价很小），
		// 但必须记日志，否则「明明清空了，后台时间却没变」很难排查
		facade.Log.Error(map[string]any{"error": err}, "记录聊天室清空时间失败")
	}

	facade.Log.Info(map[string]any{
		"cleared": cleared,
		"days":    days,
		"at":      now,
	}, "公共聊天室已自动清空")

	return cleared
}

// isRoomMediaUrl - 图片 / 头像地址合法性：站内相对路径或 http(s) 绝对地址
func isRoomMediaUrl(url string) bool {

	url = strings.TrimSpace(url)
	if url == "" || len(url) > 500 {
		return false
	}

	lower := strings.ToLower(url)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(url, "/")
}
