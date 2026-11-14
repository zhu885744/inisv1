package controller

import (
	"fmt"
	"math"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"
	socketctrl "inis/app/socket/controller"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

/**
 * ChatAdmin - 聊天管理（后台）
 *
 * 为什么独立成一个模块：前台 chat / room 的接口都围绕「我」做权限收敛
 * （只能看自己的会话、只能撤回自己的消息），后台需要看任意两人的会话、
 * 并能对违规用户直接处置，两者的鉴权口径完全不同。
 *
 * 接口：
 *   GET  chat-admin/conversations  全部私信会话（可按 uid / 最后一条摘要筛选，分页）
 *   GET  chat-admin/messages       某会话的消息内容（游标分页，含双方资料）
 *   GET  chat-admin/mutes          生效中的禁言名单（私信 + 聊天室）
 *   POST chat-admin/revoke         撤回一条消息（scene=dm / chat，推送给双方或全房广播）
 *   POST chat-admin/mute           禁言（私信 + 聊天室一次写两条，复用 chat_room_mutes）
 *   POST chat-admin/unmute         解除禁言（可按作用域，也可一次解除两处）
 *
 * 权限：以上接口都在 auth_rules 里登记为 type=default（独立权限点），
 * 只有被授予对应权限的权限组才能调用。
 * 禁言不另建「封禁」体系：私信用 room_id = ChatMuteDM(0) 落在同一张禁言表里，
 * 后台只有一个「禁言名单」，处置与解除都在这里完成。
 */
type ChatAdmin struct {
	base
}

// 私信管理单页上限（后台翻页用，与前台 chat 的 50 条区分开）
const (
	chatAdminDefaultLimit = 20
	chatAdminMaxLimit     = 100
	// chatAdminBanMaxMinutes - 单次封禁上限（一年），避免误填把账号封成十年
	chatAdminBanMaxMinutes = 365 * 24 * 60
)

func (this *ChatAdmin) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	// 会话摘要与消息内容都是实时数据，禁止缓存（否则撤回后仍能看到旧内容）
	ctx.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	ctx.Header("Pragma", "no-cache")

	allow := map[string]any{
		"conversations": this.conversations,
		"messages":      this.messages,
		"mutes":         this.mutes,
	}
	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *ChatAdmin) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"revoke": this.revoke,
		"mute":   this.mute,
		"unmute": this.unmute,
	}
	if err := this.call(allow, method, ctx); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *ChatAdmin) IPUT(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "私信管理暂不支持该操作！"), 405)
}

func (this *ChatAdmin) IDEL(ctx *gin.Context) {
	// 不做物理删除：聊天记录是双方的凭证，管理动作统一走「撤回」
	this.json(ctx, nil, facade.Lang(ctx, "私信消息请使用撤回，不做物理删除！"), 405)
}

func (this *ChatAdmin) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// conversations - 全部私信会话
func (this *ChatAdmin) conversations(ctx *gin.Context) {

	page := cast.ToInt(this.param(ctx, "page", 1))
	if page <= 0 {
		page = 1
	}
	limit := cast.ToInt(this.param(ctx, "limit", chatAdminDefaultLimit))
	if limit <= 0 || limit > chatAdminMaxLimit {
		limit = chatAdminDefaultLimit
	}

	// uid：只看某个用户参与的会话；keyword：按最后一条消息摘要模糊匹配
	uid := cast.ToInt(this.param(ctx, "uid", 0))
	if uid < 0 {
		uid = 0
	}
	keyword := strings.TrimSpace(cast.ToString(this.param(ctx, "keyword", "")))

	rows, count := (&model.ChatConversations{}).AdminList(page, limit, uid, keyword)

	// 批量补双方资料（每页最多 100 行 → 最多 200 个 uid，一次 IN 查询）
	ids := make([]int, 0, len(rows)*2)
	for _, row := range rows {
		ids = append(ids, row.Uid, row.TargetUid)
	}
	users := userBriefsByIds(ids)

	list := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		list = append(list, map[string]any{
			"id":          row.Id,
			"uid":         row.Uid,
			"target_uid":  row.TargetUid,
			"last_msg_id": row.LastMsgId,
			"last_text":   row.LastText,
			"last_uid":    row.LastUid,
			"last_time":   row.LastTime,
			"user":        users[row.Uid],
			"target":      users[row.TargetUid],
		})
	}

	this.json(ctx, gin.H{
		"data":  list,
		"count": count,
		"page":  math.Ceil(float64(count) / float64(limit)),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// messages - 某会话的消息内容
func (this *ChatAdmin) messages(ctx *gin.Context) {

	conversationId := cast.ToInt(this.param(ctx, "conversation_id", 0))
	if conversationId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "conversation_id"), 400)
		return
	}

	beforeId := cast.ToInt(this.param(ctx, "before_id", 0))
	limit := cast.ToInt(this.param(ctx, "limit", chatAdminDefaultLimit))
	if limit <= 0 || limit > chatAdminMaxLimit {
		limit = chatAdminDefaultLimit
	}

	list, hasMore := (&model.ChatMessages{}).AdminHistory(conversationId, beforeId, limit)

	// 会话与双方资料一起带回去：右侧面板的标题、气泡头像都靠它
	conversation := (&model.ChatConversations{}).AdminConversation(conversationId)
	users := userBriefsByIds([]int{
		cast.ToInt(conversation["uid"]),
		cast.ToInt(conversation["target_uid"]),
	})
	if len(conversation) > 0 {
		conversation["user"] = users[cast.ToInt(conversation["uid"])]
		conversation["target"] = users[cast.ToInt(conversation["target_uid"])]
	}

	this.json(ctx, gin.H{
		"data":         list,
		"has_more":     hasMore,
		"conversation": conversation,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// revoke - 撤回一条消息（后台，不限时间 / 不限发送者）
//
// scene=dm（默认）撤回私信，scene=chat 撤回聊天室消息；
// 聊天室撤回走 room_revoke 广播，与前台撤回同一套前端处理，所有人立刻看到「已撤回」。
// 这里不调用 room/revoke：那个接口写死了「仅超级管理员」，而聊天管理的授权是权限点，
// 被授予「撤回私信消息」的运营账号不该因为不是 root 就点不动。
func (this *ChatAdmin) revoke(ctx *gin.Context) {

	msgId := cast.ToInt(this.param(ctx, "msg_id", 0))
	if msgId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "msg_id"), 400)
		return
	}

	if scene := strings.TrimSpace(cast.ToString(this.param(ctx, "scene", model.ChatMuteScopeDM))); scene == model.SceneChat {
		// 多群：后台从「聊天室消息」列表进来时会带上 room_id；
		// 不带则按公共大厅处理，保持旧行为。
		roomId := cast.ToInt(this.param(ctx, "room_id", 0))
		if roomId <= 0 {
			roomId = model.PublicRoomId
		}

		message, err := (&model.ChatRoomMessages{}).Revoke(roomId, msgId, this.user(ctx).Id, true)
		if err != nil {
			this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
			return
		}
		// 按房间扇出：自建群的撤回只该被该群成员看到
		pushRoomEvent(roomId, map[string]any{
			"type": "room_revoke",
			"data": message,
		})
		this.json(ctx, message, facade.Lang(ctx, "已撤回！"), 200)
		return
	}

	message, err := (&model.ChatMessages{}).AdminRevoke(msgId)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 推给双方：正开着私信窗口的用户就地变成「撤回了一条消息」，不用刷新
	for _, key := range []string{"from_uid", "to_uid"} {
		targetUid := cast.ToInt(message[key])
		if targetUid <= 0 {
			continue
		}
		socketctrl.PushPrivate(targetUid, map[string]any{
			"type": "chat_revoke",
			"to":   fmt.Sprintf("user_%d", targetUid),
			"from": "admin",
			"data": message,
		})
	}

	this.json(ctx, message, facade.Lang(ctx, "已撤回！"), 200)
}

// mute - 禁言用户（私信 + 聊天室同时生效）
//
// 复用现有禁言体系（chat_room_mutes）：room_id = ChatMuteDM(0) 表示私信作用域，
// 聊天室仍用 PublicRoomId。一次处置写两条记录，两处一起失效、一起去「禁言名单」解除，
// 不另建「封禁」概念。
func (this *ChatAdmin) mute(ctx *gin.Context) {

	operatorUid := this.user(ctx).Id
	if operatorUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	targetUid := cast.ToInt(params["uid"])
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}
	if targetUid == operatorUid {
		this.json(ctx, nil, facade.Lang(ctx, "不能禁言自己！"), 400)
		return
	}
	// 管理员之间不互相禁言：避免有权限的账号把别的管理员踢出聊天室
	if model.IsRootAdmin(targetUid) {
		this.json(ctx, nil, facade.Lang(ctx, "不能禁言其他管理员！"), 400)
		return
	}

	minutes := cast.ToInt(params["minutes"])
	if minutes < 0 {
		minutes = 0
	}
	if minutes > chatAdminBanMaxMinutes {
		minutes = chatAdminBanMaxMinutes
	}

	until := int64(model.ChatRoomMuteForever)
	if minutes > 0 {
		until = time.Now().Unix() + int64(minutes)*60
	}

	reason := strings.TrimSpace(cast.ToString(params["reason"]))

	// 私信与聊天室一起禁：审核页只给一个「禁言」动作，避免运营在两处分别点一次
	for _, roomId := range []int{model.ChatMuteDM, model.PublicRoomId} {
		if err := (&model.ChatRoomMutes{}).Mute(roomId, targetUid, until, reason, operatorUid); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
			return
		}
	}

	// 广播：被禁言的人若正开着聊天室，输入框立刻变成「已禁言」
	socketctrl.BroadcastRoom(map[string]any{
		"type": "room_mute",
		"data": map[string]any{
			"uid":    targetUid,
			"until":  until,
			"reason": reason,
		},
	})

	this.json(ctx, gin.H{
		"uid":    targetUid,
		"until":  until,
		"reason": reason,
		"scopes": []string{model.ChatMuteScopeDM, model.ChatMuteScopeRoom},
	}, facade.Lang(ctx, "已禁言（私信 + 聊天室）！"), 200)
}

// mutes - 生效中的禁言名单（私信 + 聊天室）
//
// 后台「禁言名单」用：与前台 room/mutes 不同，这里不要求超级管理员，
// 走 chat-admin 的权限点；一次禁言会同时产生 dm / room 两条记录，这里都列出来。
func (this *ChatAdmin) mutes(ctx *gin.Context) {

	scope := strings.TrimSpace(cast.ToString(this.param(ctx, "scope", "")))
	if scope != "" && scope != model.ChatMuteScopeDM && scope != model.ChatMuteScopeRoom {
		this.json(ctx, nil, facade.Lang(ctx, "禁言范围不支持！"), 400)
		return
	}

	rows := (&model.ChatRoomMutes{}).ListAll()

	ids := make([]int, 0, len(rows)*2)
	for _, row := range rows {
		ids = append(ids, cast.ToInt(row["uid"]), cast.ToInt(row["operator_uid"]))
	}
	users := userBriefsByIds(ids)

	list := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		// 范围筛选放在这里做：ListAll 只查一次，避免同一份数据查两遍
		if scope != "" && cast.ToString(row["scope"]) != scope {
			continue
		}
		uid := cast.ToInt(row["uid"])
		row["scope_name"] = model.ChatMuteScopeName(cast.ToString(row["scope"]))
		row["user"] = users[uid]
		row["operator"] = users[cast.ToInt(row["operator_uid"])]
		list = append(list, row)
	}

	this.json(ctx, gin.H{"data": list, "count": len(list)}, facade.Lang(ctx, "查询成功！"), 200)
}

// unmute - 解除禁言
//
// scope = dm / room 解除单一作用域；不传 scope 则两个一起解除（与「一次禁言两处」对应）。
func (this *ChatAdmin) unmute(ctx *gin.Context) {

	params := this.params(ctx)

	uid := cast.ToInt(params["uid"])
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	scope := strings.TrimSpace(cast.ToString(params["scope"]))
	if scope != "" && scope != model.ChatMuteScopeDM && scope != model.ChatMuteScopeRoom {
		this.json(ctx, nil, facade.Lang(ctx, "禁言范围不支持！"), 400)
		return
	}

	targets := []int{model.ChatMuteDM, model.PublicRoomId}
	if scope == model.ChatMuteScopeDM {
		targets = []int{model.ChatMuteDM}
	}
	if scope == model.ChatMuteScopeRoom {
		targets = []int{model.PublicRoomId}
	}

	for _, roomId := range targets {
		if err := (&model.ChatRoomMutes{}).Unmute(roomId, uid); err != nil {
			this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
			return
		}
	}

	// 解除聊天室禁言要广播，否则正开着聊天室的用户输入框不会恢复
	if scope != model.ChatMuteScopeDM {
		socketctrl.BroadcastRoom(map[string]any{
			"type": "room_unmute",
			"data": map[string]any{"uid": uid},
		})
	}

	this.json(ctx, gin.H{
		"uid":    uid,
		"scopes": targets,
	}, facade.Lang(ctx, "已解除禁言！"), 200)
}
