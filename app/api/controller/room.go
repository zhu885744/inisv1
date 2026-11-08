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
 * Room - 公共聊天室（id=1，全站唯一）
 *
 * 接口：
 *   GET  room/info      房间资料 + 我的状态（是否管理员 / 是否被禁言）
 *   GET  room/messages  历史消息（before_id 翻页；after_seq 断线补拉）
 *   POST room/send      发言（可带 quote_id 引用某条消息）
 *   POST room/revoke    撤回（本人 2 分钟内 / 管理员不限）
 *   POST room/delete    删除（只对自己生效，别人无感）
 *   POST room/mute      禁言（管理员）
 *   POST room/unmute    解除禁言（管理员）
 *   PUT  room/update    修改名称 / 简介 / 头像（管理员）
 *   GET  room/mutes     禁言名单（管理员）
 *   GET  room/auto-clear  自动清空配置（管理员）
 *   PUT  room/auto-clear  保存自动清空间隔（管理员，单位：天）
 *
 * 权限：
 *   - 发言 / 撤回：登录即可（撤回再看是不是本人或管理员）；
 *   - 禁言 / 解禁 / 改资料：超级管理员（model.IsRootAdmin，root 权限组且拥有 all）。
 *   规则见 app/model/auth-rules.go 的 room 段，未登记的接口会被中间件直接拦掉。
 *
 * 实时性：落库后通过 socket 全员广播（socketctrl.BroadcastRoom），
 * 在线用户立刻看到；离线用户进页面拉历史即可，和私信一条思路。
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
		"info":       this.info,
		"messages":   this.messages,
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
		"send":   this.send,
		"revoke": this.revoke,
		"delete": this.remove,
		"mute":   this.mute,
		"unmute": this.unmute,
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
// 单条消息的删除是「按人隐藏」（POST room/delete），不会删掉别人的时间线；
// 也不支持整段删除（定时任务会自动清空公共记录）。
func (this *Room) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "聊天室消息请使用 room/delete 单条删除！"), 405)
}

func (this *Room) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// info - 房间资料 + 我的状态
func (this *Room) info(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	room := (&model.ChatRooms{}).Info()
	if cast.ToInt(room["id"]) <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "聊天室不存在！"), 404)
		return
	}

	muted, until := (&model.ChatRoomMutes{}).Active(model.PublicRoomId, uid)

	this.json(ctx, gin.H{
		"room":        room,
		"is_admin":    model.IsRootAdmin(uid),
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

	beforeId := cast.ToInt(this.param(ctx, "before_id", 0))
	afterSeq := cast.ToInt64(this.param(ctx, "after_seq", 0))
	limit := cast.ToInt(this.param(ctx, "limit", 30))
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	// 补拉模式：前端带着「本地已有的最大 seq」回来
	if afterSeq > 0 {
		list, lastSeq, hasMore := (&model.ChatRoomMessages{}).Sync(uid, afterSeq, limit)
		this.json(ctx, gin.H{
			"data":     this.attachUsers(list),
			"last_seq": lastSeq,
			"has_more": hasMore,
		}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	list, hasMore := (&model.ChatRoomMessages{}).History(uid, beforeId, limit)
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

	params := this.params(ctx)

	message, err := (&model.ChatRoomMessages{}).Send(
		uid,
		strings.TrimSpace(cast.ToString(params["msg_type"])),
		cast.ToString(params["content"]),
		strings.TrimSpace(cast.ToString(params["client_msg_id"])),
		cast.ToInt(params["quote_id"]),
	)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	message = this.attachUsers([]map[string]any{message})[0]

	// 全员广播：在线用户立刻上屏；离线用户靠历史记录兜底
	socketctrl.BroadcastRoom(map[string]any{
		"type": "room",
		"data": message,
	})

	this.json(ctx, message, facade.Lang(ctx, "发送成功！"), 200)
}

// revoke - 撤回（本人 2 分钟内，或超级管理员）
func (this *Room) revoke(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	msgId := cast.ToInt(this.param(ctx, "msg_id", 0))
	if msgId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "msg_id"), 400)
		return
	}

	message, err := (&model.ChatRoomMessages{}).Revoke(msgId, uid, model.IsRootAdmin(uid))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 403)
		return
	}

	message = this.attachUsers([]map[string]any{message})[0]

	socketctrl.BroadcastRoom(map[string]any{
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

	msgId := cast.ToInt(this.param(ctx, "msg_id", 0))
	if msgId <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "msg_id"), 400)
		return
	}

	if err := (&model.ChatRoomMessages{}).Delete(uid, msgId); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	socketctrl.PushPrivate(uid, map[string]any{
		"type": "room_delete",
		"to":   fmt.Sprintf("user_%d", uid),
		"from": fmt.Sprintf("user_%d", uid),
		"data": map[string]any{"id": msgId},
	})

	this.json(ctx, gin.H{"msg_id": msgId}, facade.Lang(ctx, "已删除！"), 200)
}

// mute - 禁言（超级管理员）
//
// minutes=0 表示永久；管理员本人不能被禁言，避免「手滑把自己关起来」。
func (this *Room) mute(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	if !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可执行该操作！"), 403)
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

	if err := (&model.ChatRoomMutes{}).Mute(model.PublicRoomId, targetUid, until, reason, uid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 广播禁言事件：被禁言的人若正开着聊天室，输入框会立刻变成「已禁言」
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
	}, facade.Lang(ctx, "已禁言！"), 200)
}

// unmute - 解除禁言（超级管理员）
func (this *Room) unmute(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	if !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可执行该操作！"), 403)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "uid"), 400)
		return
	}

	if err := (&model.ChatRoomMutes{}).Unmute(model.PublicRoomId, targetUid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	socketctrl.BroadcastRoom(map[string]any{
		"type": "room_unmute",
		"data": map[string]any{"uid": targetUid},
	})

	this.json(ctx, gin.H{"uid": targetUid}, facade.Lang(ctx, "已解除禁言！"), 200)
}

// mutes - 禁言名单（超级管理员）
func (this *Room) mutes(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	if !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可执行该操作！"), 403)
		return
	}

	list := (&model.ChatRoomMutes{}).List(model.PublicRoomId)

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

// update - 修改房间资料（超级管理员）
func (this *Room) update(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}
	if !model.IsRootAdmin(uid) {
		this.json(ctx, nil, facade.Lang(ctx, "无权限：仅超级管理员可修改聊天室！"), 403)
		return
	}

	room, err := (&model.ChatRooms{}).Update(this.params(ctx))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 广播资料变更：开着的页面立刻换掉标题 / 头像
	socketctrl.BroadcastRoom(map[string]any{
		"type": "room_update",
		"data": room,
	})

	this.json(ctx, room, facade.Lang(ctx, "修改成功！"), 200)
}

// autoClear - 自动清空配置（超级管理员）
//
// 真正的清空由定时任务执行（app/timer/room.go → model.RoomAutoClearTick），
// 这里只负责读写「间隔天数 / 上次清空时间」，不触发删除。
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
