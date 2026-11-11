package controller

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"inis/app/facade"
	"inis/app/model"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

func init() {
	go Hub.run()
}

type Index struct {
	// 继承
	base
}

// Read - GET请求本体
func (this Index) Read(ctx *gin.Context) {
	// 转小写
	method := strings.ToLower(ctx.Param("method"))
	allow := map[string]any{
		//"connect": this.connect,
	}
	_, err := this.call(allow, method, ctx)

	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"data": nil,
			"msg":  "方法调用错误：" + err.Error(),
			"code": 500,
		})
		return
	}
}

// Connect - socket 连接
func (this Index) Connect(ctx *gin.Context) {

	id, exists := ctx.Get("client_id")
	if !exists {
		id = guid()
	}

	if ctx.GetHeader("Upgrade") != "websocket" {
		socketLog("非WebSocket连接请求: %s", ctx.Request.RemoteAddr)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "非WebSocket连接请求",
		})
		return
	}

	clientIP := ctx.ClientIP()

	if Hub.IsBlacklisted(clientIP) {
		socketLog("IP在黑名单中，拒绝连接: %s", clientIP)
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "您的IP已被封禁",
		})
		return
	}

	origin := ctx.GetHeader("Origin")
	if !Hub.IsOriginAllowed(origin, ctx.Request.Host) {
		socketLog("Origin不允许，拒绝连接: %s", origin)
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "不允许的来源",
		})
		return
	}

	if !Hub.checkIPLimit(clientIP) {
		socketLog("IP连接数超限，拒绝连接: %s", clientIP)
		ctx.JSON(http.StatusTooManyRequests, gin.H{
			"error": "连接数超限，请稍后再试",
		})
		return
	}

	conn, err := socket.Upgrade(ctx.Writer, ctx.Request, map[string][]string{
		"X-Client-Id":   {cast.ToString(id)},
		"X-Client-info": {"Welcome to inis pro socket service！"},
	})
	if err != nil {
		socketLog("WebSocket升级错误: %v", err)
		return
	}
	client := &client{
		hub:  Hub,
		conn: conn,
		// 256 的出站缓冲：正常聊天足够，慢消费者宁可丢帧也不阻塞 hub
		send:   make(chan []byte, 256),
		done:   make(chan struct{}),
		pongCh: make(chan struct{}, 1),
		info: &info{
			ID:      cast.ToString(id),
			Type:    "connect",
			Content: "连接成功",
		},
		isAdmin: cast.ToBool(ctx.GetBool("is_admin")),
		ip:      clientIP,
	}
	client.lastActive.Store(time.Now().Unix())
	// 登录用户：记录 uid 与用户名（访客无用户名，前端显示「访客」）
	// uid 是私聊投递的唯一依据：握手带 Cookie 时中间件已解析出用户，直接落一份
	if user, ok := ctx.Get("user"); ok {
		if userMap, ok := user.(map[string]any); ok {
			client.username = cast.ToString(userMap["nickname"])
			if uid := cast.ToInt(userMap["id"]); uid > 0 {
				client.uid.Store(int64(uid))
			}
		}
	}

	Hub.addIPConnection(clientIP, cast.ToString(id))
	client.hub.connect <- client

	go client.write()
	go client.read()
}

func (this *client) read() {
	socketLog("开始读取消息: %s", this.info.ID)

	defer func() {
		this.hub.close <- this
		this.conn.Close()
	}()
	this.conn.SetReadLimit(maxMessageSize)
	this.conn.SetReadDeadline(time.Now().Add(pongWait))
	this.conn.SetPongHandler(func(string) error {
		this.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		wsMsgType, msg, err := this.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				socketLog("连接异常关闭: %s, 错误: %v", this.info.ID, err)
			} else {
				socketLog("连接正常关闭: %s, 原因: %v", this.info.ID, err)
			}
			break
		}

		// 更新最后活跃时间
		this.lastActive.Store(time.Now().Unix())

		if wsMsgType == websocket.PingMessage {
			// 协议层 pong 必须由写协程发：gorilla/websocket 同一时刻只允许一个写者，
			// 在读协程里直接 WriteMessage 会与写协程并发写，轻则丢帧重则 panic
			socketLog("收到ping消息，转交写协程回pong: %s", this.info.ID)
			this.signalPong()
			continue
		}

		if wsMsgType == websocket.PongMessage {
			socketLog("收到pong消息: %s", this.info.ID)
			continue
		}

		if wsMsgType == websocket.CloseMessage {
			socketLog("收到关闭消息: %s", this.info.ID)
			break
		}

		if json.Valid(msg) {
			if valid, errMsg := this.hub.ValidateMessage(msg); !valid {
				socketLog("消息验证失败: %s, 错误: %s", this.info.ID, errMsg)
				continue
			}

			item := Json(msg)
			msgType := cast.ToString(item["type"])

			if msgType == "ack" {
				// 批量送达确认：{ type:'ack', data:{ target_uid, seq } }
				// 只写数据库（见 model.ChatStates.Ack），不碰 hub 里的任何 map ——
				// 多个读协程并发写 map 会直接 fatal。
				this.applyChatWatermark(item, false)
				continue
			}

			if msgType == "read" {
				// 已读上报：{ type:'read', data:{ target_uid, seq } }
				// 与 ACK 同理，只推进 last_read_seq；给对方的 chat_read 回执仍走 HTTP。
				this.applyChatWatermark(item, true)
				continue
			}

			if msgType == "ping" {
				socketLog("收到JSON ping消息，回复pong: %s", this.info.ID)
				pongMsg, _ := json.Marshal(map[string]any{"type": "pong"})
				this.push(pongMsg)
				continue
			}

			if msgType == "auth" {
				// 连接后鉴权，升级身份（跨域场景下 Cookie 无法携带，前端主动发 token）
				this.handleAuth(cast.ToString(item["token"]))
				continue
			}

			if msgType == "kick" {
				// 踢下线：转交 hub goroutine 处理（只有它才能安全读取 isAdmin 与操作 clients）
				this.hub.kick <- &kickRequest{from: this, to: cast.ToString(item["to"])}
				continue
			}

			if !this.hub.checkRateLimit(this.info.ID) {
				socketLog("消息频率超限，丢弃消息: %s", this.info.ID)
				continue
			}

			// 「正在输入」：纯瞬时事件，不落库、不走离线缓存。
			// 放在通用转发之前处理，是因为它既可能带 to（私聊），
			// 也可能不带（聊天室全员广播），通用分支对这两种形态都有额外校验。
			if msgType == "typing" {
				this.relayTyping(item)
				continue
			}

			// 转发类消息的身份要求（防止匿名的跨连接投递）：
			//   - 全站广播只给管理员，否则任意游客都能向所有在线连接下发任意内容；
			//   - 定向消息要求已登录，否则游客可以往 user_<uid>（可枚举）投递伪造的私信。
			// 客户端 ID 由 socket 中间件生成，只有 user_<uid> / guest_<uuid> 两种形态；
			// 跨域握手认不出用户时连接是 guest_*，完成 {type:'auth'} 后 uid 才有值 ——
			// 所以身份判断看 uid，不要看 info.ID 前缀。
			if utils.Is.Empty(item["to"]) {
				if !this.isAdmin {
					socketLog("非管理员尝试广播，已丢弃: %s", this.info.ID)
					continue
				}
			} else {
				target := cast.ToString(item["to"])
				legal := len(target) <= 80 &&
					(strings.HasPrefix(target, "user_") || strings.HasPrefix(target, "guest_"))
				if !legal {
					socketLog("非法的定向消息，已丢弃: %s -> %s", this.info.ID, target)
					continue
				}
				// 私聊：必须已登录（guest 或还没完成 auth 升级的连接一律拒绝）
				if msgType == "private" && this.uid.Load() <= 0 {
					socketLog("未登录尝试发私信，已丢弃: %s", this.info.ID)
					this.reply(map[string]any{"type": "error", "content": "请先登录后再发送私信"})
					continue
				}
				// 其它定向消息（single）：沿用老口径，只要求不是访客
				if msgType != "private" && strings.HasPrefix(this.info.ID, "guest_") {
					socketLog("非法的定向消息，已丢弃: %s -> %s", this.info.ID, target)
					continue
				}
			}

			info := &info{
				ID: this.info.ID,
			}
			if empty := utils.Is.Empty(item["to"]); empty {
				info.Type = "broadcast"
			} else {
				info.To = cast.ToString(item["to"])
				if msgType == "private" {
					info.Type = "private"
				} else {
					info.Type = "single"
				}
			}

			// 私聊：先落库，再用「规范化后的消息」替换 payload 里的 data
			// （与 HTTP 发送共用 model.ChatMessages.Send，两边得到的历史记录完全一致）
			if msgType == "private" {
				message, err := this.persistPrivate(cast.ToString(item["to"]), item)
				if err != nil {
					socketLog("私聊落库失败: %s -> %s, 错误: %v", this.info.ID, item["to"], err)
					this.reply(map[string]any{"type": "error", "content": err.Error()})
					continue
				}
				item["data"] = message
				item["from_uid"] = cast.ToInt(message["from_uid"])
			}

			delete(item, "to")
			info.Content = item

			msg, _ = json.Marshal(info)
			this.hub.notice <- msg
		} else {
			socketLog("无效的JSON数据: %s, 内容: %s", this.info.ID, string(msg))
		}
	}
}

// handleAuth 处理客户端身份认证消息，鉴权成功后升级连接身份
// 用于前后端分离（跨域）场景：WebSocket 握手无法携带 Cookie，前端连接后主动发送 token
func (this *client) handleAuth(token string) {
	if token == "" {
		return
	}

	// 校验凭证：先查服务端会话，未命中再兼容旧 JWT（与 HTTP 中间件同一口径）
	identity, err := facade.AuthenticateToken(token)
	if err != nil {
		socketLog("auth token 校验失败: %s, 错误: %v", this.info.ID, err)
		return
	}

	uid := identity.Uid
	if uid <= 0 {
		return
	}

	// 查询用户
	user, _ := facade.DB.Model(&model.Users{}).Find(uid)
	if utils.Is.Empty(user) {
		return
	}
	userMap := cast.ToStringMap(user)

	// 校验密码哈希（与 Jwt 中间件 validatePasswordHash 逻辑一致）：改密后旧会话立即失效
	if utils.Hash.Sum32(userMap["password"]) != identity.Hash {
		socketLog("auth token 密码哈希不匹配: %s", this.info.ID)
		return
	}

	// 账号状态校验（冻结 / 封禁登录）：与 HTTP 侧 Auth 中间件同一口径。
	// 否则被封禁的账号只要连上 WebSocket、不再发 HTTP 请求，就能继续收发私信。
	if accountBlocked(userMap) {
		socketLog("auth 账号已被冻结或限制登录: %s", this.info.ID)
		this.reply(map[string]any{"type": "error", "content": "账号状态异常，请联系管理员"})
		return
	}

	// 提交身份升级请求给 hub.run（统一在 hub goroutine 内更新，避免并发读写）
	this.hub.upgrade <- &clientUpgrade{
		client:   this,
		uid:      uid,
		username: cast.ToString(userMap["nickname"]),
		isAdmin:  model.IsRootAdmin(uid),
	}
}

// accountBlocked 账号是否被冻结 / 封禁登录
//
// 与 HTTP 侧 middleware.validateUserStatus 同一口径：WebSocket 也必须拦，
// 否则封禁只对 HTTP 生效。
func accountBlocked(user map[string]any) bool {
	if cast.ToInt(user["status"]) == model.UserStatusFrozen {
		return true
	}

	// 封禁限制位里没有「限制登录」就直接放行
	if cast.ToInt(user["restrictions"])&model.BanTypeLogin == 0 {
		return false
	}

	banId := cast.ToInt(user["current_ban_id"])
	if banId <= 0 {
		return false
	}

	banRecord, _ := facade.DB.Model(&model.UserBanRecords{}).Find(banId)
	if utils.Is.Empty(banRecord) {
		return false
	}

	// 申诉中 / 申诉驳回期间封禁继续生效（只有真正解封才放行）
	return model.BanStatusRestricted(cast.ToInt(cast.ToStringMap(banRecord)["status"]))
}

// write - 唯一的写协程
//
// 所有出站帧（消息 / ping / pong / close）都在这里写，读协程只负责入站，
// 满足 gorilla/websocket「同一时刻只能有一个写者」的约束。
func (this *client) write() {
	socketLog("开始写入消息: %s", this.info.ID)

	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		this.conn.Close()
	}()

	for {
		select {
		case <-this.done:
			// 连接已被 hub 摘除：尽力发一个关闭帧，然后退出
			this.conn.SetWriteDeadline(time.Now().Add(writeWait))
			_ = this.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return

		case msg := <-this.send:
			this.conn.SetWriteDeadline(time.Now().Add(writeWait))

			next, err := this.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				socketLog("创建消息写入器失败: %s, 错误: %v", this.info.ID, err)
				return
			}
			_, _ = next.Write(msg)

			// 顺手把队列里已经在等的消息拼进同一个帧，减少系统调用；
			// 用 select+default 而不是 len(chan)，避免和投递方抢同一个队列时卡死
		drain:
			for {
				select {
				case extra := <-this.send:
					_, _ = next.Write(line)
					_, _ = next.Write(extra)
				default:
					break drain
				}
			}

			if err := next.Close(); err != nil {
				socketLog("关闭消息写入器失败: %s, 错误: %v", this.info.ID, err)
				return
			}

		case <-this.pongCh:
			this.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := this.conn.WriteMessage(websocket.PongMessage, []byte{}); err != nil {
				socketLog("发送Pong失败: %s, 错误: %v", this.info.ID, err)
				return
			}

		case <-ticker.C:
			this.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := this.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				socketLog("发送Ping失败: %s, 错误: %v", this.info.ID, err)
				return
			}
		}
	}
}

func (hub *hub) run() {
	for {
		select {
		case client := <-hub.connect:
			socketLog("客户端连接: %s", client.info.ID)
			hub.clients[client.info.ID] = client
			// 多端在线：同一个 uid 的所有连接都进集合，私聊按集合扇出
			hub.trackUser(client)
			if client.isAdmin {
				hub.adminOnline.Add(1)
			}

			if state, ok := hub.clientStates[client.info.ID]; ok {
				socketLog("检测到重连，恢复客户端状态: %s", client.info.ID)
				hub.sendOfflineMessages(client, state)
				state.lastSeen = time.Now()
				hub.recordReconnect()
			} else {
				hub.clientStates[client.info.ID] = &clientState{
					clientId: client.info.ID,
					lastSeen: time.Now(),
				}
			}

			hub.recordConnection()
			client.push([]byte(`{"type":"connect","content":"连接成功","id":"` + client.info.ID + `"}`))
			hub.markStatusChanged()
			socketLog("当前在线人数: %d", len(hub.clients))
		case client := <-hub.close:
			socketLog("客户端断开连接: %s", client.info.ID)
			hub.dropClient(client)
		case message := <-hub.notice:
			// 本实例产生的消息：本地分发 + 转发给其它实例
			hub.dispatch(message, true)
			// 转发是网络操作，放协程里做，避免阻塞 hub goroutine
			go hub.publish(message)
		case message := <-hub.cluster:
			// 来自其它实例的消息：只做本地分发。
			// 这里**不能再 publish**，否则 A 转发给 B、B 又转发出去，形成回环。
			hub.dispatch(message, false)
		case targetId := <-hub.clusterKick:
			hub.kickRemote(targetId)
		case status := <-hub.status:
			// 该通道当前无写入方（系统状态统一走 notice 通道），
			// 保留此分支但仅推送给管理员，防止未来误用导致敏感信息泄露给非管理员
			statusMsg, _ := json.Marshal(map[string]any{
				"type":       "status",
				"content":    status,
				"admin_only": true,
			})
			hub.broadcastToAdmin(statusMsg)
		case req := <-hub.kick:
			hub.handleKick(req)
		case up := <-hub.upgrade:
			// 身份升级：更新用户名与管理员标记，并重新广播在线状态
			// 注意：仅更新显示身份与权限，client_id（info.ID）保持不变，
			// 以避免与 read goroutine 并发读写 info.ID 产生数据竞争。
			prevAdmin := up.client.isAdmin
			up.client.username = up.username
			up.client.isAdmin = up.isAdmin
			// 认领身份：之后这条连接就可以收发私聊了（info.ID 仍是握手时的 guest_xxx）。
			// 注意这里可能已经有一个「握手带 Cookie」时的 uid 记录，trackUser 是集合，重复加入无害。
			if up.uid > 0 {
				up.client.uid.Store(int64(up.uid))
				hub.trackUser(up.client)
			}
			if prevAdmin != up.isAdmin {
				if up.isAdmin {
					hub.adminOnline.Add(1)
				} else {
					hub.adminOnline.Add(-1)
				}
			}
			socketLog("客户端身份升级: %s (uid=%d, 管理员=%v)", up.client.info.ID, up.uid, up.isAdmin)
			hub.markStatusChanged()
		case <-hub.statusTicker.C:
			// 合并推送：连接 / 断开 / 身份升级只标脏，由这里按固定节奏统一广播一次，
			// 避免每次连接都把全量在线列表推给所有人（O(N²) 流量）
			if hub.statusDirty {
				hub.statusDirty = false
				hub.broadcastStatus()
			}
		case <-hub.refresh:
			// 其它 goroutine 请求刷新在线状态（见 Online.BroadcastOnlineStatus）
			hub.markStatusChanged()
		}
	}
}

// markStatusChanged - 在线列表发生变更：刷新只读快照并标记待推送
//
// 只在 hub goroutine 内调用。快照供其它 goroutine 安全读取（见 clientIDs），
// 推送由 statusTicker 合并（见 hub.run），避免每次连接都全量广播。
func (hub *hub) markStatusChanged() {
	ids := make([]string, 0, len(hub.clients))
	for id := range hub.clients {
		ids = append(ids, id)
	}
	hub.clientSnap.Store(&ids)
	hub.statusDirty = true
}

// clientIDs 在线客户端 ID 快照（任意 goroutine 可安全调用）
func (hub *hub) clientIDs() []string {
	if snap := hub.clientSnap.Load(); snap != nil {
		return *snap
	}
	return nil
}

// broadcastStatus 立即广播一次在线状态
//
// 只允许在 hub goroutine 内调用（会遍历 clients）。
func (hub *hub) broadcastStatus() {
	// 收集在线用户信息（用户名/访客 + 脱敏 IP + 活跃时间 + uid）
	//
	// 带上 uid 是为了让聊天室能展示在线成员并按人 @：
	// 跨域场景下连接 ID 是 guest_xxx（身份是后来 auth 升级的），
	// 只靠 id 前缀区分不出「这是哪位登录用户」。
	onlineUsers := make([]map[string]any, 0, len(hub.clients))
	for _, client := range hub.clients {
		name := client.username
		if name == "" {
			name = "访客"
		}
		onlineUsers = append(onlineUsers, map[string]any{
			"id":          client.info.ID,
			"uid":         int(client.uid.Load()),
			"name":        name,
			"ip":          maskIP(client.ip),
			"last_active": client.lastActive.Load(),
		})
	}

	// 构建状态消息
	statusMsg, _ := json.Marshal(map[string]any{
		"type": "status",
		"content": map[string]any{
			"online_users": onlineUsers,
			"online_count": len(onlineUsers),
		},
	})

	// 广播状态消息
	hub.broadcast(statusMsg)
}

// maskIP 对 IP 地址脱敏：IPv4 保留前两段，IPv6 保留前两组
func maskIP(ip string) string {
	if ip == "" {
		return ""
	}
	if strings.Contains(ip, ":") {
		// IPv6
		parts := strings.Split(ip, ":")
		if len(parts) >= 2 {
			return parts[0] + ":" + parts[1] + ":****"
		}
		return ip
	}
	// IPv4
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*.*"
	}
	return ip
}

// trackUser - 把连接登记到 uid 集合（多端在线）
func (hub *hub) trackUser(client *client) {
	if client == nil {
		return
	}
	uid := int(client.uid.Load())
	if uid <= 0 {
		return
	}
	item, _ := hub.userClients.LoadOrStore(uid, newConnSet())
	if set, ok := item.(*connSet); ok {
		set.add(client)
	}
}

// untrackUser - 从 uid 集合里摘掉连接；集合空了就删掉这个 key
func (hub *hub) untrackUser(client *client) {
	if client == nil {
		return
	}
	uid := int(client.uid.Load())
	if uid <= 0 {
		return
	}
	item, ok := hub.userClients.Load(uid)
	if !ok {
		return
	}
	set, ok := item.(*connSet)
	if !ok {
		return
	}
	if set.remove(client) == 0 {
		// CompareAndDelete：并发重连可能刚建了一个新的同 uid 集合，不能误删
		hub.userClients.CompareAndDelete(uid, set)
	}
}

// dropClient - 从 hub 摘除一条连接（幂等）
//
// 顺序很重要：先 CAS 标记 + 关闭 done，再清索引。
// 关闭 done 之后，所有 push 都会立刻返回 false，不会再有消息被写进这条连接的队列。
func (hub *hub) dropClient(client *client) {
	if client == nil {
		return
	}
	if !client.closed.CompareAndSwap(false, true) {
		return
	}
	close(client.done)

	_, registered := hub.clients[client.info.ID]
	if registered {
		delete(hub.clients, client.info.ID)
	}

	hub.untrackUser(client)
	if client.isAdmin {
		hub.adminOnline.Add(-1)
	}

	if !registered {
		return
	}

	hub.markStatusChanged()
	hub.recordDisconnection()

	if state, ok := hub.clientStates[client.info.ID]; ok {
		state.lastSeen = time.Now()
		go hub.cleanupClientStateAfterTimeout(client.info.ID)
	}

	// 按连接自身的 IP 精确摘除。
	// 原实现遍历整个 ipConnections（O(全部在线 IP)）去找这条连接，
	// 每次断开都要把全表的 clientIds 扫一遍；client.ip 在握手时就记下了，直接用它。
	if client.ip != "" {
		hub.removeIPConnection(client.ip, client.info.ID)
	}

	socketLog("当前在线人数: %d", len(hub.clients))
}

// broadcast 广播消息
func (hub *hub) broadcast(message []byte) {
	for _, client := range hub.snapshotClients() {
		if !client.push(message) {
			hub.dropClient(client)
		}
	}
}

// broadcastToAdmin 仅广播给管理员客户端（完整版状态数据）
func (hub *hub) broadcastToAdmin(message []byte) {
	for _, client := range hub.snapshotClients() {
		if !client.isAdmin {
			continue
		}
		if !client.push(message) {
			hub.dropClient(client)
		}
	}
}

// snapshotClients - 正在遍历时可能会摘除连接，先拷一份避免边遍历边删
func (hub *hub) snapshotClients() []*client {
	list := make([]*client, 0, len(hub.clients))
	for _, client := range hub.clients {
		list = append(list, client)
	}
	return list
}

// deliverUser - 把消息推给某个 uid 的所有在线连接（多端），返回成功投递的连接数
//
// 队列满的连接直接摘除：宁可让一台慢设备重连后补拉，也不要让它拖慢其余设备。
func (hub *hub) deliverUser(uid int, message []byte) int {
	if uid <= 0 {
		return 0
	}

	item, ok := hub.userClients.Load(uid)
	if !ok {
		return 0
	}
	set, ok := item.(*connSet)
	if !ok {
		return 0
	}

	delivered := 0
	for _, client := range set.snapshot() {
		if client.push(message) {
			delivered++
			continue
		}
		hub.dropClient(client)
	}

	return delivered
}

// handleKick 处理管理员踢下线请求（仅管理员可发起）
//
// 被踢的客户端会先收到 {"type":"kicked"}（说明原因，避免前端误以为网络抖动而立刻重连），
// 随后连接被服务端关闭；关闭后其 read() 会走统一的下线清理（hub.close），在线列表随之刷新。
func (hub *hub) handleKick(req *kickRequest) {
	if req == nil || req.from == nil {
		return
	}

	// 权限判定放在 hub goroutine 里做：client.isAdmin 只由本 goroutine 写入（见 upgrade 分支）
	if !req.from.isAdmin {
		socketLog("非管理员尝试踢下线，已忽略: %s", req.from.info.ID)
		hub.trySend(req.from, `{"type":"error","content":"无权限：仅管理员可执行该操作"}`)
		return
	}

	if utils.Is.Empty(req.to) {
		return
	}

	target, ok := hub.clients[req.to]
	if !ok || target == nil {
		// 目标不在本实例：多实例部署时客户端可能连在别的节点，转发出去让持有连接的实例处理
		if clusterEnabled() {
			socketLog("踢下线目标不在本实例，转发: %s", req.to)
			// 发布是网络操作，不能在 hub goroutine 里同步做
			go clusterPublish("kick", []byte(req.to))
			hub.trySend(req.from, `{"type":"kicked","content":"已发送断开指令（目标可能连接在其它节点）"}`)
			return
		}
		socketLog("踢下线目标已离线: %s", req.to)
		hub.trySend(req.from, `{"type":"kicked","content":"目标客户端已离线"}`)
		return
	}

	socketLog("管理员 %s 踢下线客户端: %s", req.from.info.ID, target.info.ID)

	hub.trySend(target, `{"type":"kicked","content":"您已被管理员断开连接"}`)
	hub.trySend(req.from, `{"type":"kicked","content":"已断开该连接"}`)

	// 稍等片刻再关闭，给上面那条消息留出写出的时间。
	// gorilla/websocket 的 Close 允许与其它方法并发调用，所以从新 goroutine 关闭是安全的。
	go func(client *client) {
		time.Sleep(200 * time.Millisecond)
		_ = client.conn.Close()
	}(target)
}

// trySend 非阻塞投递（队列满时丢弃，绝不阻塞 hub goroutine）
func (hub *hub) trySend(client *client, message string) {
	if client == nil {
		return
	}
	if !client.push([]byte(message)) {
		socketLog("发送队列满，丢弃消息: %s", client.info.ID)
	}
}

// dispatch 按消息类型分发到本地连接
//
// allowOffline：目标不在线时是否缓存离线消息。
//   - 本实例产生的消息（true）：保持一致的老行为，离线缓存等重连补发；
//   - 其它实例转发来的消息（false）：不在本实例就跳过 —— 否则 N 个实例各存一份，
//     目标重连时会收到 N 条重复消息。
func (hub *hub) dispatch(message []byte, allowOffline bool) {
	content := Json(message)
	msgType := cast.ToString(content["type"])

	switch {
	case msgType == "status":
		// 系统状态：仅推送给管理员（非管理员不推送）
		hub.recordMessage("broadcast")
		hub.broadcastToAdmin(message)
	case utils.Is.Empty(msgType) || msgType == "broadcast":
		// 客户端广播：发给所有在线客户端（含管理员）
		hub.recordMessage("broadcast")
		hub.broadcast(message)
	case msgType == "single" || msgType == "notification":
		hub.recordMessage("single")
		hub.singlecast(message, allowOffline)
	case msgType == "private":
		hub.recordMessage("single")
		hub.privatecast(message, allowOffline)
	case msgType == "room_delete" || msgType == "chat_delete":
		// 「按人删除」：只同步给操作者自己（服务端发给 user_{uid}）。
		// room_delete 不能落到下面的 room* 广播分支 —— 否则所有人都把这条消息删了；
		// chat_delete 是私信侧的同类事件，同样只发给自己。
		hub.recordMessage("single")
		hub.singlecast(message, allowOffline)
	case strings.HasPrefix(msgType, "room"):
		// 公共聊天室：消息与各类事件（撤回 / 禁言 / 解禁 / 改资料）都是全员可见，
		// 直接广播给所有在线连接；离线用户进房间时拉历史即可。
		// room_typing（正在输入）也走这里，前端按 uid 过滤掉自己。
		hub.recordMessage("broadcast")
		hub.broadcast(message)
	case msgType == "chat_typing" || msgType == "chat_mention":
		// 私聊的「正在输入」与被 @ 提醒：都是一次性事件，
		// 只投给目标用户当前在线的设备，不做离线缓存（缓存了重连才收到也没意义）
		hub.recordMessage("single")
		if targetUid := chatTargetUid(cast.ToString(content["to"])); targetUid > 0 {
			hub.deliverUser(targetUid, message)
		}
	}
}

// publish 把本实例的消息转发给其它实例
//
// status 不转发：它是「本机」的 CPU / 内存 / 表计数快照，
// 转发出去会让管理员看到随机某个实例的资源数据。
func (hub *hub) publish(message []byte) {
	if !clusterEnabled() {
		return
	}

	msgType := cast.ToString(Json(message)["type"])
	if msgType == "status" {
		return
	}

	clusterPublish("msg", message)
}

// kickRemote 处理其它实例转发来的踢下线请求（目标连在本实例上时才会命中）
func (hub *hub) kickRemote(targetId string) {
	target, ok := hub.clients[targetId]
	if !ok || target == nil {
		return
	}

	socketLog("执行跨实例踢下线: %s", targetId)
	hub.trySend(target, `{"type":"kicked","content":"您已被管理员断开连接"}`)

	// 稍等片刻再关闭，给上面那条消息留出写出的时间（理由同 handleKick）
	go func(client *client) {
		time.Sleep(200 * time.Millisecond)
		_ = client.conn.Close()
	}(target)
}

func (hub *hub) singlecast(message []byte, allowOffline bool) {
	content := Json(message)
	to := content["to"]
	socketLog("单播消息: %v", content)
	if empty := utils.Is.Empty(to); empty {
		to = content["id"]
		socketLog("未指定接收者，发送给自己: %s", to)
	}

	targetId := cast.ToString(to)
	if client, ok := hub.clients[targetId]; ok {
		if client.push(message) {
			socketLog("消息发送成功: %s -> %s", content["id"], to)
		} else {
			socketLog("发送队列满或连接已关闭，摘除连接: %s", client.info.ID)
			hub.dropClient(client)
			if allowOffline {
				hub.storeOfflineMessage(targetId, message)
			}
		}
	} else if allowOffline {
		socketLog("目标客户端离线，缓存消息: %s", targetId)
		hub.storeOfflineMessage(targetId, message)
	}
}

func (hub *hub) sendOfflineMessages(client *client, state *clientState) {
	now := time.Now()
	validMsgs := []*offlineMessage{}

	for _, msg := range state.offlineMsgs {
		if now.Sub(msg.sentAt) < hub.offlineMsgTTL {
			// 队列满就留在列表里等下次重连，别丢
			if !client.push(msg.message) {
				validMsgs = append(validMsgs, msg)
				continue
			}
			socketLog("发送离线消息: %s", client.info.ID)
		}
	}

	state.offlineMsgs = validMsgs
}

func (hub *hub) storeOfflineMessage(clientId string, message []byte) {
	if state, ok := hub.clientStates[clientId]; ok {
		if len(state.offlineMsgs) >= hub.maxOfflineMsgs {
			state.offlineMsgs = state.offlineMsgs[1:]
		}
		state.offlineMsgs = append(state.offlineMsgs, &offlineMessage{
			message: message,
			sentAt:  time.Now(),
		})
	} else {
		hub.clientStates[clientId] = &clientState{
			clientId: clientId,
			lastSeen: time.Now(),
			offlineMsgs: []*offlineMessage{
				{message: message, sentAt: time.Now()},
			},
		}
	}
}

func (hub *hub) cleanupClientStateAfterTimeout(clientId string) {
	time.Sleep(hub.reconnectTimeout)

	if _, ok := hub.clients[clientId]; !ok {
		if state, ok := hub.clientStates[clientId]; ok {
			if time.Since(state.lastSeen) >= hub.reconnectTimeout {
				socketLog("清理超时客户端状态: %s", clientId)
				delete(hub.clientStates, clientId)
			}
		}
	}
}

func (hub *hub) privatecast(message []byte, allowOffline bool) {
	content := Json(message)
	from := cast.ToString(content["id"])
	to := cast.ToString(content["to"])

	socketLog("私聊消息: %s -> %s", from, to)

	hub.GetOrCreateChatSession(from, to)

	msgId := cast.ToString(content["msg_id"])
	if msgId == "" {
		msgId = guid()
		content["msg_id"] = msgId
		message, _ = json.Marshal(content)
	}

	// 旧版逐条已读回执用的状态表（新链路已改走 HTTP chat_read，这里保留统计用途）
	hub.messageStatuses[msgId] = &messageStatus{
		msgId:     msgId,
		readBy:    map[string]bool{from: false, to: false},
		delivered: map[string]bool{from: true, to: false},
		sentAt:    time.Now(),
	}

	targetUid := chatTargetUid(to)
	if targetUid > 0 {
		// 多端在线：推给接收者的每一台设备
		delivered := hub.deliverUser(targetUid, message)
		if delivered > 0 {
			hub.messageStatuses[msgId].delivered[to] = true
		}

		// 发送者的其它设备：多端同步。发送端当前设备已经用接口返回值上屏，
		// 前端按消息 id 去重，这里重复推一次不会出现两条气泡。
		if fromUid := chatFromUid(content, from); fromUid > 0 && fromUid != targetUid {
			hub.deliverUser(fromUid, message)
		}

		if delivered == 0 && allowOffline {
			// 内存离线缓存只是「重连即刻补发」的加速层，真正的兜底是 DB + after_seq 补拉。
			// 跨实例转发来的消息（allowOffline=false）不缓存，避免多实例各存一份。
			socketLog("目标用户全部离线，缓存私聊消息: %s", to)
			hub.storeOfflineMessage(to, message)
		}
		return
	}

	// 非 uid 形态（老客户端直接发 guest_xxx）：按客户端 ID 单播
	if client := hub.resolveClient(to); client != nil {
		if client.push(message) {
			hub.messageStatuses[msgId].delivered[to] = true
			socketLog("私聊消息发送成功: %s -> %s", from, to)
		} else {
			hub.dropClient(client)
			if allowOffline {
				hub.storeOfflineMessage(to, message)
			}
		}
		return
	}

	if allowOffline {
		socketLog("目标客户端离线，缓存私聊消息: %s", to)
		hub.storeOfflineMessage(to, message)
	}
}

// chatFromUid - 解析私聊消息的发送者 uid
//
// 优先取落库后写入的 data.from_uid（HTTP 与 socket 发送都会补上），
// 兜底解析 id 字段的 user_<uid> 形式。
func chatFromUid(content map[string]any, from string) int {
	if data, ok := content["data"].(map[string]any); ok {
		if uid := cast.ToInt(data["from_uid"]); uid > 0 {
			return uid
		}
	}
	return chatTargetUid(from)
}
