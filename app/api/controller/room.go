package controller

import (
	"fmt"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"
	socketctrl "inis/app/socket/controller"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

/**
 * Room - 聊天室（公共大厅 id=1 + 用户自建群聊）
 *
 * 接口（除 auto-clear 外都支持 room_id，缺省为公共大厅）：
 *   GET  room/list        群列表（scope=mine 我的 / discover 发现）
 *   GET  room/info        房间资料 + 我的状态（角色 / 是否禁言）
 *   GET  room/messages    历史消息（before_id 翻页；after_seq 断线补拉）
 *   GET  room/members     成员列表
 *   GET  room/requests    待审的入群申请（群主 / 管理员）
 *   GET  room/mutes       禁言名单（群主 / 管理员）
 *   GET  room/auto-clear  自动清空配置（超级管理员，仅大厅）
 *   POST room/create      创建群聊（有等级 / 注册天数门槛）
 *   POST room/join        入群（自由加入 / 提交申请）
 *   POST room/leave       退群
 *   POST room/dissolve    解散（群主 / 超级管理员）
 *   POST room/announce    发布群公告（群主）
 *   POST room/member-role 设置 / 撤销群管理员（群主）
 *   POST room/member-remove 移出成员（群主 / 管理员）
 *   POST room/request-handle 审批入群申请（群主 / 管理员）
 *   POST room/read        上报已读水位
 *   POST room/send        发言（可带 quote_id 引用）
 *   POST room/revoke      撤回（本人 2 分钟内 / 群管理）
 *   POST room/delete      删除（只对自己生效）
 *   POST room/mute        禁言（群主 / 管理员；大厅仅超级管理员）
 *   POST room/unmute      解除禁言（同上）
 *   PUT  room/update      修改名称 / 简介 / 头像 / 入群方式
 *
 * 权限：规则见 app/model/auth-rules.go 的 room 段（全部 type=login）。
 * 具体到「能不能管这个群」由 model.RoomCanManage / RoomCanModerate 判断。
 *
 * 实时性：落库后按房间扇出 —— 公共大厅全员广播，自建群只推给成员
 * （见 pushRoomEvent），否则 A 群的消息会串到 B 群，既乱又泄露。
 */

type Room struct {
	base
}

// roomMuteMaxMinutes - 单次禁言上限（一年）：避免手滑把「分钟」当「秒」填出几十年
const roomMuteMaxMinutes = 365 * 24 * 60

func (this *Room) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	// 房间资料与消息都是实时数据，禁止缓存（否则管理员改名后别人仍看到旧名字）
	ctx.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	ctx.Header("Pragma", "no-cache")

	allow := map[string]any{
		"list":       this.list,
		"info":       this.info,
		"messages":   this.messages,
		"members":    this.members,
		"requests":   this.requests,
		"invite":     this.invite,
		"invites":    this.inviteList,
		"mutes":      this.mutes,
		"auto-clear": this.autoClear,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Room) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"create":        this.create,
		"join":          this.join,
		"leave":         this.leave,
		"dissolve":      this.dissolve,
		"announce":      this.announce,
		"member-role":   this.memberRole,
		"member-add":    this.memberAdd,
		"member-remove": this.memberRemove,
		"invite-create": this.inviteCreate,
		"join-by-code":  this.joinByCode,
		"request-handle": this.requestHandle,
		"read":          this.read,
		"send":          this.send,
		"revoke":        this.revoke,
		"delete":        this.remove,
		"mute":          this.mute,
		"unmute":        this.unmute,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Room) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"update":     this.update,
		"auto-clear": this.saveAutoClear,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IDEL - 聊天室不走 REST DELETE：
// 单条消息的删除是「按人隐藏」（POST room/delete）；
// 解散群聊是 POST room/dissolve（需要走权限与成员清理）。
func (this *Room) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "聊天室消息请使用 room/delete 单条删除！"), 405)
}

func (this *Room) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// roomIdOf - 从参数里取 room_id，缺省为公共大厅
func (this *Room) roomIdOf(ctx *gin.Context) int {

	id := cast.ToInt(this.param(ctx, "room_id", 0))
	if id <= 0 {
		id = model.PublicRoomId
	}

	return id
}

// pushRoomEvent - 按房间扇出实时事件
//
// 大厅：所有在线连接都要收到（全员隐式成员）；
// 自建群：只推给当前成员，每条消息带 to → socket 的 dispatch 走单播。
// 离线成员不缓存：进群拉历史即可（与私信「DB 兜底」是同一思路）。
func pushRoomEvent(roomId int, payload map[string]any) {

	if roomId <= 0 || len(payload) == 0 {
		return
	}

	if roomId == model.PublicRoomId {
		socketctrl.BroadcastRoom(payload)
		return
	}

	for _, targetUid := range model.RoomMemberUids(roomId) {
		item := make(map[string]any, len(payload)+1)
		for key, value := range payload {
			item[key] = value
		}
		item["to"] = fmt.Sprintf("user_%d", targetUid)
		socketctrl.PushPrivate(targetUid, item)
	}
}

// pushRoomManagers - 只推给群主 / 管理员（入群申请、被踢通知等）
func pushRoomManagers(roomId int, payload map[string]any) {

	if roomId <= 0 || roomId == model.PublicRoomId {
		return
	}

	for _, uid := range model.RoomMemberUids(roomId) {
		if model.RoomRole(roomId, uid) < model.ChatRoomRoleAdmin {
			continue
		}
		item := make(map[string]any, len(payload)+1)
		for key, value := range payload {
			item[key] = value
		}
		item["to"] = fmt.Sprintf("user_%d", uid)
		socketctrl.PushPrivate(uid, item)
	}
}

// list - 群列表
//
// scope=mine（默认）：我加入的群；scope=discover：可发现并加入的群。
// 公共大厅不在列表里，前端用固定入口展示。
func (this *Room) list(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	scope := strings.TrimSpace(cast.ToString(this.param(ctx, "scope", "mine")))
	keyword := strings.TrimSpace(cast.ToString(this.param(ctx, "keyword", "")))

	// 后台：全量房间（含官方大厅），只有超级管理员能看
	if scope == "all" {
		if !model.IsRootAdmin(uid) {
			this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可查看全部聊天室！"), 403)
			return
		}

		page := cast.ToInt(this.param(ctx, "page", 1))
		size := cast.ToInt(this.param(ctx, "size", 20))
		list, total := model.RoomListAll(keyword, page, size)

		this.json(ctx, gin.H{"data": list, "total": total}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	list := model.RoomList(uid, scope, keyword)

	this.json(ctx, gin.H{
		"data":        list,
		"can_create":  roomCanCreateInfo(uid),
		"public_room": (&model.ChatRooms{}).Info(model.PublicRoomId),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// roomCanCreateInfo - 建群入口的状态（能建 / 为什么不能建）
func roomCanCreateInfo(uid int) map[string]any {

	ok, reason := model.RoomCanCreate(uid)

	return map[string]any{
		"allowed":      ok,
		"reason":       reason,
		"max_owned":    model.ChatRoomCreateMaxOwned,
		"owned":        model.RoomCountByOwner(uid),
		"min_days":     model.ChatRoomCreateMinDays,
		"min_exp":      model.ChatRoomCreateMinExp,
		"max_members":  model.ChatRoomMaxMembers,
	}
}

// create - 创建群聊
func (this *Room) create(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	room, err := model.RoomCreate(uid, this.params(ctx))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	this.json(ctx, room, facade.Lang(ctx, "创建成功！"), 200)
}

// info - 房间资料 + 我的状态
func (this *Room) info(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	room, err := model.RoomDetail(roomId, uid)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 404)
		return
	}

	muted, until := (&model.ChatRoomMutes{}).Active(roomId, uid)

	this.json(ctx, gin.H{
		"room":        room,
		"is_admin":    model.RoomCanManage(roomId, uid),
		"is_moderator": model.RoomCanModerate(roomId, uid),
		"muted":       muted,
		"muted_until": until,
		"uid":         uid,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// messages - 历史消息
//
// 两种用法：
//   - before_id：往前翻页（进页面时传 0 取最新一页）；
//   - after_seq：断线重连补拉，只取 id 更大的那些。
func (this *Room) messages(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	// 自建群：不是成员不能看历史（大厅跳过；超级管理员可进任意群审消息）
	if roomId != model.PublicRoomId && !model.RoomIsMember(roomId, uid) && !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "你还不是该聊天室的成员！"), 403)
		return
	}

	beforeId := cast.ToInt(this.param(ctx, "before_id", 0))
	afterSeq := cast.ToInt64(this.param(ctx, "after_seq", 0))
	limit := cast.ToInt(this.param(ctx, "limit", 30))
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	// 补拉模式：前端带着「本地已有的最大 seq」回来
	if afterSeq > 0 {
		list, lastSeq, hasMore := (&model.ChatRoomMessages{}).Sync(roomId, uid, afterSeq, limit)
		this.json(ctx, gin.H{
			"data":     this.attachUsers(list),
			"last_seq": lastSeq,
			"has_more": hasMore,
		}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	list, hasMore := (&model.ChatRoomMessages{}).History(roomId, uid, beforeId, limit)
	this.json(ctx, gin.H{
		"data":     this.attachUsers(list),
		"has_more": hasMore,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// send - 发言
func (this *Room) send(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录后再发言！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	params := this.params(ctx)

	// @提及：客户端从在线成员里选中的人。这里只做「去重 + 限量」，
	// 昵称是否真的出现在正文里由 model 层核对（不信任客户端）
	mentionUids := make([]int, 0, model.ChatRoomMentionMax)
	for _, id := range cast.ToIntSlice(params["mention_uids"]) {
		if id > 0 {
			mentionUids = append(mentionUids, id)
		}
		if len(mentionUids) >= model.ChatRoomMentionMax {
			break
		}
	}

	message, err := (&model.ChatRoomMessages{}).Send(
		roomId,
		uid,
		strings.TrimSpace(cast.ToString(params["msg_type"])),
		cast.ToString(params["content"]),
		strings.TrimSpace(cast.ToString(params["client_msg_id"])),
		cast.ToInt(params["quote_id"]),
		mentionUids...,
	)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	message = this.attachUsers([]map[string]any{message})[0]

	// 按房间扇出：在线成员立刻上屏；离线成员靠进群拉历史兜底
	pushRoomEvent(roomId, map[string]any{
		"type": "room",
		"data": message,
	})

	this.json(ctx, message, facade.Lang(ctx, "发送成功！"), 200)
}

// revoke - 撤回（本人 2 分钟内，或群管理 / 超级管理员）
func (this *Room) revoke(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	msgId := cast.ToInt(this.param(ctx, "msg_id", 0))
	if msgId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "msg_id"), 400)
		return
	}

	message, err := (&model.ChatRoomMessages{}).Revoke(roomId, msgId, uid, model.RoomCanModerate(roomId, uid))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 403)
		return
	}

	message = this.attachUsers([]map[string]any{message})[0]

	pushRoomEvent(roomId, map[string]any{
		"type": "room_revoke",
		"data": message,
	})

	this.json(ctx, message, facade.Lang(ctx, "已撤回！"), 200)
}

// remove - 删除一条消息（只对自己生效）
//
// 与撤回不同：不删表、不改内容、不广播，只给当前用户写一条隐藏记录。
// 同账号的其它在线设备通过 room_delete 私推同步移除（type 仍是 room* 前缀，
// 前端 store 会把它转给聊天室页处理）。
func (this *Room) remove(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	msgId := cast.ToInt(this.param(ctx, "msg_id", 0))
	if msgId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "msg_id"), 400)
		return
	}

	if err := (&model.ChatRoomMessages{}).Delete(roomId, uid, msgId); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	socketctrl.PushPrivate(uid, map[string]any{
		"type": "room_delete",
		"to":   fmt.Sprintf("user_%d", uid),
		"from": fmt.Sprintf("user_%d", uid),
		"data": map[string]any{"id": msgId, "room_id": roomId},
	})

	this.json(ctx, gin.H{"msg_id": msgId}, facade.Lang(ctx, "已删除！"), 200)
}

// mute - 禁言（群主 / 管理员；大厅仅超级管理员）
//
// minutes=0 表示永久；管理员本人不能被禁言，避免「手滑把自己关起来」。
func (this *Room) mute(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	// 大厅的禁言是全局处置，保持原口径：只有超级管理员能动
	if roomId == model.PublicRoomId && !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可执行该操作！"), 403)
		return
	}
	if roomId != model.PublicRoomId && !model.RoomCanModerate(roomId, uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅群主或管理员可执行该操作！"), 403)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}
	if targetUid == uid {
		this.json(ctx, nil, facade.Lang(ctx, "不能禁言自己！"), 400)
		return
	}
	if model.IsRootAdmin(targetUid) {
		this.json(ctx, nil, facade.Lang(ctx, "不能禁言其他管理员！"), 400)
		return
	}
	// 群主不能被群管理员禁言
	if roomId != model.PublicRoomId && model.RoomRole(roomId, targetUid) == model.ChatRoomRoleOwner {
		this.json(ctx, nil, facade.Lang(ctx, "不能禁言群主！"), 400)
		return
	}

	minutes := cast.ToInt(this.param(ctx, "minutes", 0))
	if minutes < 0 {
		minutes = 0
	}
	if minutes > roomMuteMaxMinutes {
		minutes = roomMuteMaxMinutes
	}

	until := int64(model.ChatRoomMuteForever)
	if minutes > 0 {
		until = time.Now().Unix() + int64(minutes)*60
	}

	reason := strings.TrimSpace(cast.ToString(this.param(ctx, "reason", "")))

	if err := (&model.ChatRoomMutes{}).Mute(roomId, targetUid, until, reason, uid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 广播禁言事件：被禁言的人若正开着聊天室，输入框会立刻变成「已禁言」
	pushRoomEvent(roomId, map[string]any{
		"type": "room_mute",
		"data": map[string]any{
			"room_id": roomId,
			"uid":     targetUid,
			"until":   until,
			"reason":  reason,
		},
	})

	this.json(ctx, gin.H{
		"uid":    targetUid,
		"until":  until,
		"reason": reason,
	}, facade.Lang(ctx, "已禁言！"), 200)
}

// unmute - 解除禁言
func (this *Room) unmute(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	if roomId == model.PublicRoomId && !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可执行该操作！"), 403)
		return
	}
	if roomId != model.PublicRoomId && !model.RoomCanModerate(roomId, uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅群主或管理员可执行该操作！"), 403)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	if err := (&model.ChatRoomMutes{}).Unmute(roomId, targetUid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	pushRoomEvent(roomId, map[string]any{
		"type": "room_unmute",
		"data": map[string]any{"room_id": roomId, "uid": targetUid},
	})

	this.json(ctx, gin.H{"uid": targetUid}, facade.Lang(ctx, "已解除禁言！"), 200)
}

// mutes - 禁言名单（群主 / 管理员）
func (this *Room) mutes(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	if roomId == model.PublicRoomId && !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可查看禁言名单！"), 403)
		return
	}
	if roomId != model.PublicRoomId && !model.RoomCanModerate(roomId, uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅群主或管理员可查看禁言名单！"), 403)
		return
	}

	list := (&model.ChatRoomMutes{}).List(roomId)

	ids := make([]int, 0, len(list))
	for _, item := range list {
		ids = append(ids, cast.ToInt(item["uid"]))
	}
	users := userBriefsByIds(ids)

	for _, item := range list {
		item["user"] = users[cast.ToInt(item["uid"])]
	}

	this.json(ctx, gin.H{"data": list}, facade.Lang(ctx, "查询成功！"), 200)
}

// members - 成员列表（所有成员都能看）
func (this *Room) members(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	if roomId == model.PublicRoomId {
		// 大厅不维护成员表：返回空数组，前端用 socket 的在线成员面板
		this.json(ctx, gin.H{"data": []any{}, "public_room": true}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	if !model.RoomIsMember(roomId, uid) && !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "你还不是该聊天室的成员！"), 403)
		return
	}

	list := model.RoomMembers(roomId, 200)

	ids := make([]int, 0, len(list))
	for _, item := range list {
		ids = append(ids, cast.ToInt(item["uid"]))
	}
	users := userBriefsByIds(ids)
	for _, item := range list {
		item["user"] = users[cast.ToInt(item["uid"])]
	}

	this.json(ctx, gin.H{"data": list}, facade.Lang(ctx, "查询成功！"), 200)
}

// requests - 待审的入群申请（群主 / 管理员）
func (this *Room) requests(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	if roomId == model.PublicRoomId {
		this.json(ctx, nil, facade.Lang(ctx, "公共聊天室不需要审批！"), 400)
		return
	}
	if !model.RoomCanModerate(roomId, uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅群主或管理员可查看申请！"), 403)
		return
	}

	list := model.RoomRequests(roomId)

	ids := make([]int, 0, len(list))
	for _, item := range list {
		ids = append(ids, cast.ToInt(item["uid"]))
	}
	users := userBriefsByIds(ids)
	for _, item := range list {
		item["user"] = users[cast.ToInt(item["uid"])]
	}

	this.json(ctx, gin.H{"data": list}, facade.Lang(ctx, "查询成功！"), 200)
}

// join - 入群（自由加入 / 提交申请）
func (this *Room) join(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	message := strings.TrimSpace(cast.ToString(this.param(ctx, "message", "")))

	result, err := model.RoomJoin(roomId, uid, message)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	if joined := cast.ToBool(result["joined"]); joined && roomId != model.PublicRoomId {
		user := this.user(ctx)
		pushRoomEvent(roomId, map[string]any{
			"type": "room_member_join",
			"data": map[string]any{
				"room_id":   roomId,
				"uid":       uid,
				"nickname":  user.Nickname,
				"join_time": time.Now().Unix(),
			},
		})
	}

	if cast.ToBool(result["pending"]) {
		pushRoomManagers(roomId, map[string]any{
			"type": "room_join_request",
			"data": map[string]any{
				"room_id": roomId,
				"uid":     uid,
				"message": message,
			},
		})
	}

	tip := "已加入聊天室！"
	if cast.ToBool(result["pending"]) {
		tip = "申请已提交，等待群主审批！"
	}

	this.json(ctx, result, facade.Lang(ctx, tip), 200)
}

// leave - 退群
func (this *Room) leave(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	if err := model.RoomLeave(roomId, uid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 先推给「还在群里的人」，再把自己从推送范围里摘掉的效果由前端处理
	pushRoomEvent(roomId, map[string]any{
		"type": "room_member_leave",
		"data": map[string]any{"room_id": roomId, "uid": uid},
	})

	this.json(ctx, gin.H{"room_id": roomId}, facade.Lang(ctx, "已退出聊天室！"), 200)
}

// dissolve - 解散群聊（群主 / 超级管理员）
func (this *Room) dissolve(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	if roomId == model.PublicRoomId {
		this.json(ctx, nil, facade.Lang(ctx, "公共聊天室不能解散！"), 400)
		return
	}

	// 解散后成员表会被清空，先取一份 uid 列表用于通知
	members := model.RoomMemberUids(roomId)

	if err := model.RoomDissolve(roomId, uid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 403)
		return
	}

	payload := map[string]any{
		"type": "room_dissolve",
		"data": map[string]any{"room_id": roomId, "operator_uid": uid},
	}
	for _, targetUid := range members {
		item := make(map[string]any, len(payload)+1)
		for key, value := range payload {
			item[key] = value
		}
		item["to"] = fmt.Sprintf("user_%d", targetUid)
		socketctrl.PushPrivate(targetUid, item)
	}

	this.json(ctx, gin.H{"room_id": roomId}, facade.Lang(ctx, "聊天室已解散！"), 200)
}

// announce - 发布群公告（群主 / 超级管理员；大厅由后台发布）
func (this *Room) announce(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	if roomId == model.PublicRoomId && !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可发布公共公告！"), 403)
		return
	}
	if roomId != model.PublicRoomId && !model.RoomCanManage(roomId, uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅群主可发布公告！"), 403)
		return
	}

	notice := cast.ToString(this.param(ctx, "notice", ""))

	room, err := (&model.ChatRooms{}).SetNotice(roomId, notice)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	pushRoomEvent(roomId, map[string]any{
		"type": "room_announce",
		"data": room,
	})

	this.json(ctx, room, facade.Lang(ctx, "公告已发布！"), 200)
}

// memberRole - 设置 / 撤销群管理员（群主）
func (this *Room) memberRole(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	targetUid := cast.ToInt(this.param(ctx, "uid", 0))
	role := cast.ToInt(this.param(ctx, "role", model.ChatRoomRoleMember))

	if err := model.RoomSetRole(roomId, uid, targetUid, role); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	pushRoomEvent(roomId, map[string]any{
		"type": "room_member_role",
		"data": map[string]any{"room_id": roomId, "uid": targetUid, "role": role},
	})

	this.json(ctx, gin.H{"uid": targetUid, "role": role}, facade.Lang(ctx, "设置成功！"), 200)
}

// invite - 邀请落地页：这个链接对应哪个群、我是否已在群里
//
// 参数用 code 而不是 room_id：拿到链接的人只知道自己手里的那串码，
// 服务端据此反查房间，避免「猜房间 id 就能看群资料」。
func (this *Room) invite(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	code := strings.TrimSpace(cast.ToString(this.param(ctx, "code", "")))

	info, err := model.RoomInviteInfo(code, uid)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 404)
		return
	}

	this.json(ctx, info, facade.Lang(ctx, "查询成功！"), 200)
}

// invites - 已生成的邀请列表（群主 / 管理员）
func (this *Room) inviteList(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	list, err := model.RoomInviteList(this.roomIdOf(ctx), uid)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 403)
		return
	}

	this.json(ctx, gin.H{"data": list}, facade.Lang(ctx, "查询成功！"), 200)
}

// inviteCreate - 生成邀请链接 / 二维码（群主 / 管理员）
//
// days：有效期天数（0 = 永久）；max_uses：可用次数（0 = 不限）。
// 返回的 code 由前端拼成 <站点>/join/<code>，二维码画的就是这个地址。
func (this *Room) inviteCreate(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	// 默认永久（0 = 不过期）：群邀请会被贴到主页、群公告、二维码海报上，
	// 带时限等于过几天变成死链。需要限时的调用方显式传 days 即可。
	days := cast.ToInt(this.param(ctx, "days", 0))
	maxUses := cast.ToInt(this.param(ctx, "max_uses", 0))

	invite, err := model.RoomCreateInvite(roomId, uid, days, maxUses)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	this.json(ctx, invite, facade.Lang(ctx, "邀请已生成！"), 200)
}

// joinByCode - 凭邀请码入群（不受入群方式限制，仍受人数上限约束）
func (this *Room) joinByCode(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	code := strings.TrimSpace(cast.ToString(this.param(ctx, "code", "")))

	result, err := model.RoomJoinByCode(code, uid)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	roomId := cast.ToInt(result["room_id"])

	// 已经在群里（重复点链接）不再广播，避免成员列表无意义地刷
	if !cast.ToBool(result["again"]) {
		user := this.user(ctx)
		pushRoomEvent(roomId, map[string]any{
			"type": "room_member_join",
			"data": map[string]any{
				"room_id":   roomId,
				"uid":       uid,
				"nickname":  user.Nickname,
				"join_time": time.Now().Unix(),
			},
		})
	}

	this.json(ctx, result, facade.Lang(ctx, "已加入聊天室！"), 200)
}

// memberAdd - 邀请成员（群主 / 管理员；「仅邀请」的群只能这样进人）
func (this *Room) memberAdd(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	targetUid := cast.ToInt(this.param(ctx, "uid", 0))

	if err := model.RoomAddMember(roomId, uid, targetUid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 成员列表实时刷新（被邀请的人若正开着这个群，也能从成员面板看到自己）
	pushRoomEvent(roomId, map[string]any{
		"type": "room_member_join",
		"data": map[string]any{"room_id": roomId, "uid": targetUid, "operator_uid": uid},
	})

	this.json(ctx, gin.H{"uid": targetUid}, facade.Lang(ctx, "已邀请加入聊天室！"), 200)
}

// memberRemove - 移出成员（群主 / 管理员）
func (this *Room) memberRemove(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	targetUid := cast.ToInt(this.param(ctx, "uid", 0))

	if err := model.RoomKick(roomId, uid, targetUid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	pushRoomEvent(roomId, map[string]any{
		"type": "room_member_remove",
		"data": map[string]any{"room_id": roomId, "uid": targetUid, "operator_uid": uid},
	})

	this.json(ctx, gin.H{"uid": targetUid}, facade.Lang(ctx, "已移出聊天室！"), 200)
}

// requestHandle - 审批入群申请
func (this *Room) requestHandle(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	requestId := cast.ToInt(this.param(ctx, "request_id", 0))
	approve := cast.ToBool(this.param(ctx, "approve", false))

	if err := model.RoomHandleRequest(roomId, uid, requestId, approve); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	if approve {
		pushRoomEvent(roomId, map[string]any{
			"type": "room_member_join",
			"data": map[string]any{"room_id": roomId, "uid": 0, "request_id": requestId},
		})
	}

	this.json(ctx, gin.H{"request_id": requestId, "approve": approve}, facade.Lang(ctx, "处理成功！"), 200)
}

// read - 上报已读水位（进群 / 拉到最新消息后调用）
func (this *Room) read(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)
	lastId := cast.ToInt(this.param(ctx, "last_id", 0))

	model.RoomMarkRead(roomId, uid, lastId)

	this.json(ctx, gin.H{"room_id": roomId, "last_id": lastId}, facade.Lang(ctx, "已更新！"), 200)
}

// update - 修改房间资料
//
// 大厅：超级管理员；自建群：群主（可改名称 / 简介 / 头像 / 入群方式）。
func (this *Room) update(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	roomId := this.roomIdOf(ctx)

	if roomId == model.PublicRoomId && !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可修改聊天室！"), 403)
		return
	}
	if roomId != model.PublicRoomId && !model.RoomCanManage(roomId, uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅群主可修改聊天室资料！"), 403)
		return
	}

	room, err := (&model.ChatRooms{}).Update(this.params(ctx), roomId)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	pushRoomEvent(roomId, map[string]any{
		"type": "room_update",
		"data": room,
	})

	this.json(ctx, room, facade.Lang(ctx, "修改成功！"), 200)
}

// autoClear - 自动清空配置（超级管理员，仅公共大厅）
//
// 自建群不参与自动清空：群主自己可以随时解散，没必要再来一套「按时清空」。
func (this *Room) autoClear(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	if !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可查看自动清空配置！"), 403)
		return
	}

	this.json(ctx, model.RoomAutoClearInfo(), facade.Lang(ctx, "查询成功！"), 200)
}

// saveAutoClear - 保存自动清空间隔（超级管理员）
//
// days 只接受下拉里提供的值（见 model.RoomAutoClearOptions），
// 传 0 或其它值一律拒绝，避免把「每隔 0 天清一次」这种配置写进库。
func (this *Room) saveAutoClear(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	if !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可修改自动清空配置！"), 403)
		return
	}

	days := cast.ToInt(this.param(ctx, "days", 0))

	info, err := model.SaveRoomAutoClear(days)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	this.json(ctx, info, facade.Lang(ctx, "保存成功！"), 200)
}

// attachUsers - 给消息批量补上发送者信息（昵称 / 头像），避免前端 N+1 查询
func (this *Room) attachUsers(list []map[string]any) []map[string]any {

	if len(list) == 0 {
		return list
	}

	ids := make([]int, 0, len(list))
	for _, item := range list {
		ids = append(ids, cast.ToInt(item["uid"]))
	}

	users := userBriefsByIds(ids)

	for _, item := range list {
		item["user"] = users[cast.ToInt(item["uid"])]
	}

	return list
}
