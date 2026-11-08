package controller

import (
	"fmt"
	"strings"

	"inis/app/facade"
	"inis/app/model"
	socketctrl "inis/app/socket/controller"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * Chat - 用户私聊（私信）
 *
 * 分工：
 *   - HTTP：会话列表 / 历史记录 / 发送兜底 / 未读数 / 已读；
 *   - WebSocket（复用 /socket，见 app/socket/controller）：实时送达。
 *     落库只有一条路径（model.ChatMessages.Send），socket 收到的私聊同样走它，
 *     所以两条通道发出的消息在历史记录、未读数上没有差异。
 *
 * 接口：
 *   GET  chat/conversations  我的会话列表（对方信息 + 最后一条摘要 + 未读数）
 *   GET  chat/messages       与某人的历史消息（游标分页，顺带标记已读）
 *   GET  chat/sync           断线重连补拉（after_seq 之后的消息 + 双方同步水位）
 *   GET  chat/unread         我的私信未读总数
 *   GET  chat/status         能否与某人私聊（拉黑 / 自己 / 用户不存在）
 *   POST chat/send           发送私信（写库 + 通过 socket 推给在线对方）
 *   POST chat/delete         删除一条消息（只对自己生效，对方记录不动）
 *   PUT  chat/ack            批量确认「已收到 seq 及之前的消息」（水位只增不减）
 *   PUT  chat/read           把某人的消息标记为已读
 *
 * 权限：全部 type=login（私聊是用户之间的基础能力，不设权限点）。
 * 发送门槛：拉黑与「仅互关好友」统一走 model.ChatPermission，
 * 不要在这里直接查关注 / 拉黑表（socket 发送也要过同一道校验）。
 */

type Chat struct {
	base
}

// 私信相关上限
const (
	// chatDefaultLimit - 会话列表 / 历史记录的默认条数
	chatDefaultLimit = 20
	// chatMaxLimit - 单页上限
	chatMaxLimit = 50
	// chatMaxLength - 单条消息最大字符数（与表字段 2000 对齐）
	chatMaxLength = 2000
)

func (this *Chat) IGET(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	// 私信数据全部是实时的（未读总数、会话列表、历史记录），禁止浏览器 / 代理缓存。
	// 少了这一行就会出现「消息已经标已读，角标却一直显示旧数字」——
	// 因为 GET /chat/unread 被浏览器直接复用了几分钟前的响应。
	ctx.Header("Cache-Control", "no-store, no-cache, must-revalidate")
	ctx.Header("Pragma", "no-cache")
	ctx.Header("Expires", "0")

	allow := map[string]any{
		"conversations": this.conversations,
		"messages":      this.messages,
		"sync":          this.sync,
		"unread":        this.unread,
		"status":        this.status,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Chat) IPOST(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"send":   this.send,
		"revoke": this.revoke,
		"delete": this.remove,
		"pat":    this.pat,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

func (this *Chat) IPUT(ctx *gin.Context) {
	method := strings.ToLower(ctx.Param("method"))

	allow := map[string]any{
		"ack":  this.ack,
		"read": this.read,
	}
	err := this.call(allow, method, ctx)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, "方法调用错误：%v", err.Error()), 405)
	}
}

// IDEL - 私信不走 REST 的 DELETE：
// 单条消息删除是「按人隐藏」（POST chat/delete），不是删掉双方的共同记录，
// 也不支持整段会话删除。
func (this *Chat) IDEL(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "私信记录暂不支持整段删除！"), 405)
}

func (this *Chat) INDEX(ctx *gin.Context) {
	this.json(ctx, nil, facade.Lang(ctx, "没什么用！"), 202)
}

// conversations - 我的会话列表
func (this *Chat) conversations(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	limit := cast.ToInt(this.param(ctx, "limit", chatDefaultLimit))
	if limit <= 0 || limit > chatMaxLimit {
		limit = chatDefaultLimit
	}

	list := (&model.ChatConversations{}).ListMine(uid, limit)

	// 批量取对方信息：会话最多 50 条，一次 IN 查询即可
	ids := make([]int, 0, len(list))
	for _, item := range list {
		ids = append(ids, cast.ToInt(item["other_uid"]))
	}

	users := this.userBriefs(ids)

	for _, item := range list {
		other := cast.ToInt(item["other_uid"])
		item["user"] = users[other]
		item["online"] = socketctrl.IsChatOnline(other)
	}

	this.json(ctx, gin.H{"data": list}, facade.Lang(ctx, "查询成功！"), 200)
}

// messages - 与某人的历史消息（倒序取一页、正序返回；同时把对方发来的消息标记已读）
func (this *Chat) messages(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "target_uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_uid"), 400)
		return
	}
	if targetUid == uid {
		this.json(ctx, nil, facade.Lang(ctx, "不能和自己私聊！"), 400)
		return
	}

	beforeId := cast.ToInt(this.param(ctx, "before_id", 0))
	limit := cast.ToInt(this.param(ctx, "limit", chatDefaultLimit))
	if limit <= 0 || limit > chatMaxLimit {
		limit = chatDefaultLimit
	}

	target, _ := facade.DB.Model(&model.Users{}).Find(targetUid)
	if utils.Is.Empty(target) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	list, hasMore := (&model.ChatMessages{}).History(uid, targetUid, beforeId, limit)

	// 进入会话即视为已读：只标记「对方发给我的」未读，自己发的保持原状；
	// 有实际标记时顺手给对方发已读回执（对方据此把气泡上的「未读」改成「已读」）
	if beforeId <= 0 {
		if rows, lastId := (&model.ChatMessages{}).MarkRead(uid, targetUid); rows > 0 && lastId > 0 {
			notifyChatRead(targetUid, uid, lastId)
		}
	}

	// 不满足发送条件（拉黑 / 未互关）时仍允许看历史（否则聊天记录会凭空消失），
	// 只是不能再发；原因一并返回，前端直接显示在输入区上方
	canSend, reason := true, ""
	if err := model.ChatPermission(uid, targetUid); err != nil {
		canSend, reason = false, facade.Lang(ctx, err.Error())
	}

	user := utils.Map.WithField(target, []string{"id", "nickname", "avatar", "description", "title"})
	model.SanitizeAuthor(user)

	this.json(ctx, gin.H{
		"data":         list,
		"has_more":     hasMore,
		"target":       user,
		"can_send":     canSend,
		"can_send_tip": reason,
		"online":       socketctrl.IsChatOnline(targetUid),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// sync - 断线重连补拉
//
// 客户端本地记着「这个会话我收到的最大 seq」，重连后带着 after_seq 回来，
// 服务端只返回 seq 更大的消息（按 id 升序），漏掉的实时推送就此补齐。
// 顺带把两个水位带回去，客户端可以直接对比，不必自己推。
func (this *Chat) sync(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "target_uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_uid"), 400)
		return
	}
	if targetUid == uid {
		this.json(ctx, nil, facade.Lang(ctx, "不能和自己私聊！"), 400)
		return
	}

	afterSeq := cast.ToInt64(this.param(ctx, "after_seq", 0))
	limit := cast.ToInt(this.param(ctx, "limit", chatMaxLimit))
	if limit <= 0 || limit > 100 {
		limit = chatMaxLimit
	}

	list, lastSeq, hasMore := (&model.ChatMessages{}).Sync(uid, targetUid, afterSeq, limit)
	lastAck, lastRead := (&model.ChatStates{}).State(uid, targetUid)

	this.json(ctx, gin.H{
		"data":          list,
		"last_seq":      lastSeq,
		"has_more":      hasMore,
		"last_ack_seq":  lastAck,
		"last_read_seq": lastRead,
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// unread - 我的私信未读总数
func (this *Chat) unread(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	count := (&model.ChatMessages{}).UnreadTotal(uid)

	this.json(ctx, gin.H{"count": count}, facade.Lang(ctx, "查询成功！"), 200)
}

// status - 能否与某人私聊（作者主页的「发私信」按钮据此决定是否可点）
func (this *Chat) status(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "target_uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_uid"), 400)
		return
	}

	if targetUid == uid {
		this.json(ctx, gin.H{"can_send": false, "reason": facade.Lang(ctx, "不能和自己私聊！")}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	target, _ := facade.DB.Model(&model.Users{}).Find(targetUid)
	if utils.Is.Empty(target) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	// 拉黑 / 未互关都在 ChatPermission 里判断，提示文案与发送接口保持一致
	if err := model.ChatPermission(uid, targetUid); err != nil {
		this.json(ctx, gin.H{
			"can_send": false,
			"reason":   facade.Lang(ctx, err.Error()),
			"online":   socketctrl.IsChatOnline(targetUid),
		}, facade.Lang(ctx, "查询成功！"), 200)
		return
	}

	this.json(ctx, gin.H{
		"can_send": true,
		"reason":   "",
		"online":   socketctrl.IsChatOnline(targetUid),
	}, facade.Lang(ctx, "查询成功！"), 200)
}

// send - 发送私信
//
// 与 socket 私聊共用 model.ChatMessages.Send：
//  1. 落库（带幂等键，重试不会产生重复消息）；
//  2. 通过 socket 推给在线的对方（离线不影响，对方下次进会话拉历史即可）。
func (this *Chat) send(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	targetUid := cast.ToInt(params["target_uid"])
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_uid"), 400)
		return
	}
	if targetUid == uid {
		this.json(ctx, nil, facade.Lang(ctx, "不能给自己发私信！"), 400)
		return
	}

	target, _ := facade.DB.Model(&model.Users{}).Find(targetUid)
	if utils.Is.Empty(target) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	// 发送门槛：拉黑（双向）与「仅互关好友」都在这里统一判断
	if err := model.ChatPermission(uid, targetUid); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 403)
		return
	}

	content := strings.TrimSpace(cast.ToString(params["content"]))
	if content == "" {
		this.json(ctx, nil, facade.Lang(ctx, "消息内容不能为空！"), 400)
		return
	}
	if len([]rune(content)) > chatMaxLength {
		this.json(ctx, nil, facade.Lang(ctx, "消息内容太长了！"), 400)
		return
	}

	msgType := strings.TrimSpace(cast.ToString(params["msg_type"]))
	if msgType == "" {
		msgType = "text"
	}
	// 拍一拍必须走 /pat（那里有频率限制），发送入口只接受 text / image，
	// 否则有人可以用 /send 绕过拍一拍的节流
	if !utils.InArray(msgType, []string{"text", "image"}) {
		this.json(ctx, nil, facade.Lang(ctx, "不支持的消息类型！"), 400)
		return
	}

	clientMsgId := strings.TrimSpace(cast.ToString(params["client_msg_id"]))
	// 引用回复：只接受被引用消息的 id，摘要由 model 层回查后落快照
	quoteId := cast.ToInt(params["quote_id"])

	message, err := (&model.ChatMessages{}).Send(uid, targetUid, msgType, content, clientMsgId, quoteId)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 实时推送：对方在线就立刻收到，离线时这条 push 会被丢弃，由历史记录兜底
	socketctrl.PushPrivate(targetUid, map[string]any{
		"type": "private",
		"to":   fmt.Sprintf("user_%d", targetUid),
		"from": fmt.Sprintf("user_%d", uid),
		"data": message,
	})

	this.json(ctx, message, facade.Lang(ctx, "发送成功！"), 200)
}

// ack - 批量确认已收到 seq 及之前的消息
//
// 客户端每收到一批消息只上报一次最大 seq，服务端把它记进
// chat_states.last_ack_seq（只增不减）。它只表示「送达」，不等于已读：
// 已读另走 PUT /chat/read，两者分开才能区分「收到但没看」。
func (this *Chat) ack(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "target_uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_uid"), 400)
		return
	}

	seq := cast.ToInt64(this.param(ctx, "seq", 0))
	if seq <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "seq"), 400)
		return
	}

	lastAck := (&model.ChatStates{}).Ack(uid, targetUid, seq)

	this.json(ctx, gin.H{"last_ack_seq": lastAck}, facade.Lang(ctx, "已确认！"), 200)
}

// read - 标记某人的消息为已读
func (this *Chat) read(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	targetUid := cast.ToInt(this.param(ctx, "target_uid", 0))
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_uid"), 400)
		return
	}

	rows, lastId := (&model.ChatMessages{}).MarkRead(uid, targetUid)

	// 客户端也可以直接上报读到哪一条（比如本地已经有完整历史、只是没点开）
	if seq := cast.ToInt64(this.param(ctx, "seq", 0)); seq > int64(lastId) {
		lastId = int(seq)
		(&model.ChatStates{}).Read(uid, targetUid, seq)
	}

	// 有实际标记才回执：对方据此把 id <= lastId 的自己消息标成「已读」
	if rows > 0 && lastId > 0 {
		notifyChatRead(targetUid, uid, lastId)
	}

	// 顺带把「标记完之后的未读总数」带回去：
	// 客户端拿它直接覆盖角标，不必再发一次 GET /chat/unread
	// （那一来多一次往返，二来并发/缓存都可能让角标停在旧值上）。
	this.json(ctx, gin.H{
		"updated":     rows,
		"last_msg_id": lastId,
		"unread":      (&model.ChatMessages{}).UnreadTotal(uid),
	}, facade.Lang(ctx, "已标记为已读！"), 200)
}

// revoke - 撤回自己发的消息（2 分钟内）
//
// 撤回只清内容、留标记，双方界面都显示「撤回了一条消息」；
// 对方通过 socket 的 chat_revoke 事件就地替换气泡，不用刷新。
func (this *Chat) revoke(ctx *gin.Context) {

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

	message, err := (&model.ChatMessages{}).Revoke(uid, msgId)
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 通知接收方（自己的界面用接口返回值就地更新）
	toUid := cast.ToInt(message["to_uid"])
	socketctrl.PushPrivate(toUid, map[string]any{
		"type": "chat_revoke",
		"to":   fmt.Sprintf("user_%d", toUid),
		"from": fmt.Sprintf("user_%d", uid),
		"data": message,
	})

	this.json(ctx, message, facade.Lang(ctx, "已撤回！"), 200)
}

// remove - 删除一条消息（只对自己生效）
//
// 语义与微信的「删除」一致：本人的历史 / 未读数里不再出现，
// 对方完全无感（不推送、不改内容）。多设备同步靠推送给自己：
// 其它在线设备收到 chat_delete 后把对应气泡就地移除。
func (this *Chat) remove(ctx *gin.Context) {

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

	if err := (&model.ChatMessages{}).Delete(uid, msgId); err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 推给自己：同一账号在别的设备上打开着这段会话时，也要把消息移除
	socketctrl.PushPrivate(uid, map[string]any{
		"type": "chat_delete",
		"to":   fmt.Sprintf("user_%d", uid),
		"from": fmt.Sprintf("user_%d", uid),
		"data": map[string]any{"id": msgId},
	})

	this.json(ctx, gin.H{"msg_id": msgId}, facade.Lang(ctx, "已删除！"), 200)
}

// pat - 拍一拍（往会话里写一条系统提示，并实时推给对方）
//
// suffix 是「的后脑勺」这类后缀，前端可自定义；不传就是「拍了拍你」。
func (this *Chat) pat(ctx *gin.Context) {

	uid := this.user(ctx).Id
	if uid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "请先登录！"), 401)
		return
	}

	params := this.params(ctx)

	targetUid := cast.ToInt(params["target_uid"])
	if targetUid <= 0 {
		this.json(ctx, nil, facade.Lang(ctx, "%s 不能为空！", "target_uid"), 400)
		return
	}
	if targetUid == uid {
		this.json(ctx, nil, facade.Lang(ctx, "不能拍自己！"), 400)
		return
	}

	target, _ := facade.DB.Model(&model.Users{}).Find(targetUid)
	if utils.Is.Empty(target) {
		this.json(ctx, nil, facade.Lang(ctx, "用户不存在！"), 404)
		return
	}

	if !(&model.UserBlocks{}).CanInteract(uid, targetUid) {
		this.json(ctx, nil, facade.Lang(ctx, "你们之间存在拉黑关系，无法私聊！"), 403)
		return
	}

	message, err := (&model.ChatMessages{}).Pat(uid, targetUid, cast.ToString(params["suffix"]))
	if err != nil {
		this.json(ctx, nil, facade.Lang(ctx, err.Error()), 400)
		return
	}

	// 拍一拍也是一条消息：对方直接在聊天窗里看到灰色提示
	socketctrl.PushPrivate(targetUid, map[string]any{
		"type": "private",
		"to":   fmt.Sprintf("user_%d", targetUid),
		"from": fmt.Sprintf("user_%d", uid),
		"data": message,
	})

	this.json(ctx, message, facade.Lang(ctx, "已拍一拍！"), 200)
}

// notifyChatRead - 已读回执
//
// 只发「读到哪一条」，不逐条回执：对方把 id <= lastMsgId 的自己消息标成已读即可。
func notifyChatRead(targetUid, byUid, lastMsgId int) {

	if targetUid <= 0 || byUid <= 0 || lastMsgId <= 0 {
		return
	}

	socketctrl.PushPrivate(targetUid, map[string]any{
		"type": "chat_read",
		"to":   fmt.Sprintf("user_%d", targetUid),
		"from": fmt.Sprintf("user_%d", byUid),
		"data": map[string]any{
			"by_uid":      byUid,
			"last_msg_id": lastMsgId,
		},
	})
}

// userBriefs - 批量取用户公开信息（会话列表用，避免 N+1）
func (this *Chat) userBriefs(ids []int) map[int]map[string]any {
	return userBriefsByIds(ids)
}

// userBriefsByIds - 批量取用户公开信息
//
// 包级函数供私信与聊天室共用：两条链路的「作者信息」口径必须一致，
// 否则同一个人在私信里是昵称、在聊天室里是「用户 #id」。
func userBriefsByIds(ids []int) map[int]map[string]any {

	result := make(map[int]map[string]any, len(ids))

	uids := make([]int, 0, len(ids))
	for _, id := range ids {
		if id > 0 {
			uids = append(uids, id)
		}
	}
	if len(uids) == 0 {
		return result
	}

	var users []model.Users
	if err := facade.DB.Drive().Where("id IN ?", uids).Find(&users).Error; err != nil {
		return result
	}

	for _, user := range users {
		item := map[string]any{
			"id":          user.Id,
			"nickname":    user.Nickname,
			"avatar":      user.Avatar,
			"description": user.Description,
			"title":       user.Title,
		}
		model.SanitizeAuthor(item)
		result[user.Id] = item
	}

	return result
}
