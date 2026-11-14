package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"inis/app/model"

	"github.com/spf13/cast"
)

/**
 * 私聊（私信）与 socket 的对接
 *
 * 设计：socket 只负责「实时送达」，消息本体统一由 model.ChatMessages.Send 落库。
 *   - 客户端发 {type:'private', to:'user_<uid>', data:{content, msg_type, client_msg_id}}
 *     → 服务端先落库，再把规范化后的消息（含 id / from_uid / create_time）推给对方；
 *   - HTTP POST /api/chat/send 落库后调用 PushPrivate 推给在线用户，离线则由历史记录兜底。
 *
 * 投递目标用 uid 索引（hub.userClients）而不是客户端 ID：
 * 跨域场景下 WebSocket 握手带不上 Cookie，连接先以 guest_<uuid> 落地，
 * 完成 {type:'auth'} 之后才认领 uid —— 此时 clients["user_<uid>"] 是查不到的。
 */

func init() {
	// 机审撤回消息后要就地推给双方。model 不能反过来 import 本包（会循环依赖），
	// 所以由这里把推送能力注入进去；未注入时撤回只改库，刷新后也能看到
	model.PushChatRevoke = PushPrivate
	// 聊天室被 @ 时给被提及者推一条提醒（同样走 dispatch 的 chat_mention 分支）
	model.PushChatMention = PushPrivate
}

// PushPrivate - 把一条私聊消息推给指定用户（供 API 控制器调用）
//
// 走到 hub.notice 只是为了复用「本地分发 + 跨实例转发」的既有链路；
// 加超时保护，避免 hub 正忙时把 HTTP 请求挂住。
func PushPrivate(targetUid int, payload map[string]any) {

	if targetUid <= 0 || len(payload) == 0 {
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	select {
	case Hub.notice <- data:
	case <-time.After(500 * time.Millisecond):
		socketLog("私聊推送超时: uid=%d", targetUid)
	}
}

// BroadcastRoom - 把聊天室消息 / 事件广播给所有在线连接
//
// 聊天室没有「接收人」：一条消息全员可见，所以走 notice → dispatch 的 broadcast 分支。
// payload 的 type 必须是 dispatch 里登记过的 room* 类型，否则会被静默丢弃。
func BroadcastRoom(payload map[string]any) {

	if len(payload) == 0 {
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	select {
	case Hub.notice <- data:
	case <-time.After(500 * time.Millisecond):
		socketLog("聊天室广播超时")
	}
}

// IsChatOnline - 该用户当前是否有 WebSocket 连接（聊天页的在线状态点）
//
// 多端在线：该 uid 下只要还有任意一条连接就算在线。
func IsChatOnline(uid int) bool {
	if uid <= 0 {
		return false
	}
	item, ok := Hub.userClients.Load(uid)
	if !ok || item == nil {
		return false
	}
	set, ok := item.(*connSet)
	if !ok {
		return false
	}
	return set.size() > 0
}

// resolveClient - 按「客户端 ID」或「user_<uid>」找到连接
//
// 只在 hub goroutine（dispatch → privatecast）里调用：hub.clients 由它独占读写。
func (hub *hub) resolveClient(to string) *client {

	if client, ok := hub.clients[to]; ok {
		return client
	}

	uid, err := strconv.Atoi(strings.TrimPrefix(to, "user_"))
	if err != nil || uid <= 0 {
		return nil
	}

	item, ok := hub.userClients.Load(uid)
	if !ok || item == nil {
		return nil
	}

	set, ok := item.(*connSet)
	if !ok {
		return nil
	}

	// 单播类消息（single / notification）只投递给其中一台设备即可
	list := set.snapshot()
	if len(list) == 0 {
		return nil
	}
	return list[0]
}

// reply - 只回给当前连接的提示消息（错误 / 限流等），队列满就丢弃
func (this *client) reply(payload map[string]any) {

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	if !this.push(data) {
		socketLog("发送队列满或连接已关闭，提示消息已丢弃: %s", this.info.ID)
	}
}

// chatTargetUid - 从 "user_12" 里解析出 12（也接受纯数字）
func chatTargetUid(to string) int {
	to = strings.TrimSpace(to)
	if to == "" {
		return 0
	}
	uid, err := strconv.Atoi(strings.TrimPrefix(to, "user_"))
	if err != nil || uid <= 0 {
		return 0
	}
	return uid
}

// applyChatWatermark - 处理客户端通过 socket 上报的 ACK / 已读水位
//
// 为什么放在读协程里做：这只是「写一行数据库」的事，不涉及 hub 的任何内存结构，
// 因此不需要像 kick 那样转交 hub goroutine；反过来，走 HTTP 每收到一批消息
// 就发一次请求，明显更贵，也会把 ACK 拖到网络往返之后。
//
// 老客户端发的是逐条的 { msg_id }（内存里的临时 id，早已废弃）：
// 解析不出 seq 就直接忽略，不影响聊天本身。
func (this *client) applyChatWatermark(item map[string]any, read bool) {

	uid := int(this.uid.Load())
	if uid <= 0 {
		return
	}

	data := cast.ToStringMap(item["data"])
	if len(data) == 0 {
		data = item
	}

	targetUid := chatTargetUid(cast.ToString(data["target_uid"]))
	seq := cast.ToInt64(data["seq"])
	if targetUid <= 0 || seq <= 0 || targetUid == uid {
		socketLog("忽略无效的聊天水位上报: %s, target=%d, seq=%d", this.info.ID, targetUid, seq)
		return
	}

	if read {
		(&model.ChatStates{}).Read(uid, targetUid, seq)
		return
	}

	(&model.ChatStates{}).Ack(uid, targetUid, seq)
}

// typingIntervalMs - 同一连接两次「正在输入」的最小间隔（毫秒）
//
// 前端已经做了 3 秒节流，这里再兜一道：输入状态丢了不影响任何数据，
// 但把 notice 通道打满会拖慢真正的私信与聊天室消息，所以宁可多丢几条。
const typingIntervalMs int64 = 1500

// relayTyping - 转发「正在输入」状态
//
// 两种形态：
//   - 带 to：私聊，只投递给对方的所有设备（多端在线时每台都要看到提示）；
//   - 带 room=true：聊天室，广播给所有在线连接，前端按 uid 过滤掉自己。
//
// 只发 uid 不带昵称：昵称由前端从在线成员快照里查，
// 避免服务端为了一个瞬时事件去读用户表（这里每秒可能有几十次调用）。
func (this *client) relayTyping(item map[string]any) {

	uid := int(this.uid.Load())
	if uid <= 0 {
		return
	}

	now := time.Now().UnixMilli()
	if last := this.lastTyping.Load(); last > 0 && now-last < typingIntervalMs {
		return
	}
	this.lastTyping.Store(now)

	if cast.ToBool(item["room"]) {
		// 多群：room_id 缺省按公共大厅处理（老客户端不带这个字段）
		roomId := cast.ToInt(item["room_id"])
		if roomId <= 0 {
			roomId = model.PublicRoomId
		}

		payload := map[string]any{
			"type": "room_typing",
			"data": map[string]any{"uid": uid, "room_id": roomId},
		}

		if roomId == model.PublicRoomId {
			BroadcastRoom(payload)
			return
		}

		// 自建群：只推给成员，别让别的群看到「有人在输入」
		for _, targetUid := range model.RoomMemberUids(roomId) {
			one := make(map[string]any, len(payload)+1)
			for key, value := range payload {
				one[key] = value
			}
			one["to"] = fmt.Sprintf("user_%d", targetUid)
			PushPrivate(targetUid, one)
		}
		return
	}

	targetUid := chatTargetUid(cast.ToString(item["to"]))
	if targetUid <= 0 || targetUid == uid {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"type": "chat_typing",
		"to":   fmt.Sprintf("user_%d", targetUid),
		"data": map[string]any{"from_uid": uid},
	})
	if err != nil {
		return
	}

	// 与 PushPrivate 同样的超时保护：hub 正忙时不能把这个读协程挂住
	select {
	case Hub.notice <- payload:
	case <-time.After(500 * time.Millisecond):
	}
}

// persistPrivate - 把 socket 收到的私聊消息落库，返回规范化后的消息结构
//
// 校验项与 HTTP 发送保持一致（拉黑 / 空内容 / 长度 / 类型都在 model 层做），
// 这里只负责把 socket 的 payload 拆成业务字段。
func (this *client) persistPrivate(to string, item map[string]any) (map[string]any, error) {

	sendUid := cast.ToInt(this.uid.Load())
	if sendUid <= 0 {
		return nil, errors.New("请先登录！")
	}

	targetUid := chatTargetUid(to)
	if targetUid <= 0 {
		return nil, errors.New("私聊对象不正确！")
	}

	// 拉黑是双向的：任一方向存在拉黑都不能发
	if !(&model.UserBlocks{}).CanInteract(sendUid, targetUid) {
		return nil, errors.New("你们之间存在拉黑关系，无法私聊！")
	}

	payload := cast.ToStringMap(item["data"])
	if len(payload) == 0 {
		// 兼容老客户端：直接把文本放在 content 字段上
		payload = cast.ToStringMap(item)
	}

	content := strings.TrimSpace(cast.ToString(payload["content"]))
	msgType := strings.TrimSpace(cast.ToString(payload["msg_type"]))
	clientMsgId := strings.TrimSpace(cast.ToString(payload["client_msg_id"]))

	// 拍一拍走 HTTP /chat/pat（有频率限制），socket 直发只允许文本 / 图片，
	// 避免绕过节流刷屏
	if msgType != "" && msgType != "text" && msgType != "image" {
		return nil, errors.New("不支持的消息类型！")
	}

	// 引用回复：客户端只传被引用消息的 id，内容快照由服务端回查后落库
	quoteId := cast.ToInt(payload["quote_id"])

	return (&model.ChatMessages{}).Send(sendUid, targetUid, msgType, content, clientMsgId, quoteId)
}
