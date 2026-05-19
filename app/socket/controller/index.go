package controller

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

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
	if !Hub.IsOriginAllowed(origin) {
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
		send: make(chan []byte, 256),
		info: &info{
			ID:      cast.ToString(id),
			Type:    "connect",
			Content: "连接成功",
		},
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

		if wsMsgType == websocket.PingMessage {
			socketLog("收到ping消息，回复pong: %s", this.info.ID)
			this.conn.WriteMessage(websocket.PongMessage, []byte{})
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
				msgId := cast.ToString(item["msg_id"])
				this.handleAck(msgId)
				continue
			}

			if msgType == "read" {
				msgId := cast.ToString(item["msg_id"])
				this.hub.MarkMessageRead(msgId, this.info.ID)
				socketLog("消息已读: %s, msgId: %s", this.info.ID, msgId)
				continue
			}

			if msgType == "ping" {
				socketLog("收到JSON ping消息，回复pong: %s", this.info.ID)
				pongMsg, _ := json.Marshal(map[string]any{"type": "pong"})
				this.send <- pongMsg
				continue
			}

			if !this.hub.checkRateLimit(this.info.ID) {
				socketLog("消息频率超限，丢弃消息: %s", this.info.ID)
				continue
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

			delete(item, "to")
			info.Content = item

			msg, _ = json.Marshal(info)
			this.hub.notice <- msg
		} else {
			socketLog("无效的JSON数据: %s, 内容: %s", this.info.ID, string(msg))
		}
	}
}

func (this *client) handleAck(msgId string) {
	if _, ok := this.hub.pendingMessages[msgId]; ok {
		socketLog("收到ACK确认: %s, 客户端: %s", msgId, this.info.ID)
		delete(this.hub.pendingMessages, msgId)
	}
}

func (this *client) sendWithAck(message []byte) {
	msgId := guid()

	var content map[string]any
	if err := json.Unmarshal(message, &content); err == nil {
		content["msg_id"] = msgId
		message, _ = json.Marshal(content)
	}

	this.hub.pendingMessages[msgId] = &pendingMessage{
		message: message,
		client:  this,
		attempt: 1,
		sentAt:  time.Now(),
	}

	select {
	case this.send <- message:
		socketLog("发送消息(带ACK): %s, msgId: %s", this.info.ID, msgId)
		go this.waitForAck(msgId)
	default:
		socketLog("发送队列满: %s", this.info.ID)
		delete(this.hub.pendingMessages, msgId)
	}
}

func (this *client) waitForAck(msgId string) {
	ticker := time.NewTicker(this.hub.ackTimeout)
	defer ticker.Stop()

	for range ticker.C {
		if pm, ok := this.hub.pendingMessages[msgId]; ok {
			if pm.attempt >= this.hub.maxRetries {
				socketLog("消息重试次数耗尽: %s, msgId: %s", this.info.ID, msgId)
				delete(this.hub.pendingMessages, msgId)
				return
			}

			pm.attempt++
			pm.sentAt = time.Now()
			socketLog("重传消息: %s, msgId: %s, 尝试次数: %d", this.info.ID, msgId, pm.attempt)

			select {
			case this.send <- pm.message:
			default:
				socketLog("重传队列满: %s", this.info.ID)
			}
		} else {
			return
		}
	}
}

func (this *client) write() {
	socketLog("开始写入消息: %s", this.info.ID)

	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		this.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-this.send:
			this.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				this.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			next, err := this.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				socketLog("创建消息写入器失败: %s, 错误: %v", this.info.ID, err)
				return
			}
			next.Write(msg)

			len := len(this.send)
			for i := 0; i < len; i++ {
				next.Write(line)
				next.Write(<-this.send)
			}

			if err := next.Close(); err != nil {
				socketLog("关闭消息写入器失败: %s, 错误: %v", this.info.ID, err)
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
			client.send <- []byte(`{"type":"connect","content":"连接成功","id":"` + client.info.ID + `"}`)
			hub.broadcastStatus()
			socketLog("当前在线人数: %d", len(hub.clients))
		case client := <-hub.close:
			socketLog("客户端断开连接: %s", client.info.ID)
			if _, ok := hub.clients[client.info.ID]; ok {
				delete(hub.clients, client.info.ID)
				close(client.send)
				hub.broadcastStatus()
				socketLog("当前在线人数: %d", len(hub.clients))

				hub.recordDisconnection()

				if state, ok := hub.clientStates[client.info.ID]; ok {
					state.lastSeen = time.Now()
					go hub.cleanupClientStateAfterTimeout(client.info.ID)
				}

				for ip, conn := range hub.ipConnections {
					for _, cid := range conn.clientIds {
						if cid == client.info.ID {
							hub.removeIPConnection(ip, client.info.ID)
							break
						}
					}
				}
			}
		case message := <-hub.notice:
			content := Json(message)
			msgType := cast.ToString(content["type"])
			if empty := utils.Is.Empty(msgType); empty || msgType == "broadcast" || msgType == "status" {
				hub.recordMessage("broadcast")
				hub.broadcast(message)
			} else if msgType == "single" {
				hub.recordMessage("single")
				hub.singlecast(message)
			} else if msgType == "private" {
				hub.recordMessage("single")
				hub.privatecast(message)
			}
		case status := <-hub.status:
			statusMsg, _ := json.Marshal(map[string]any{
				"type":    "status",
				"content": status,
			})
			hub.broadcast(statusMsg)
		}
	}
}

// 广播在线状态
func (hub *hub) broadcastStatus() {
	// 收集在线用户ID
	onlineUsers := []string{}
	for clientId := range hub.clients {
		onlineUsers = append(onlineUsers, clientId)
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

// 广播消息
func (hub *hub) broadcast(message []byte) {
	for _, client := range hub.clients {
		select {
		case client.send <- message:
		default:
			close(client.send)
			delete(hub.clients, client.info.ID)
		}
	}
}

func (hub *hub) singlecast(message []byte) {
	content := Json(message)
	to := content["to"]
	socketLog("单播消息: %v", content)
	if empty := utils.Is.Empty(to); empty {
		to = content["id"]
		socketLog("未指定接收者，发送给自己: %s", to)
	}

	targetId := cast.ToString(to)
	if client, ok := hub.clients[targetId]; ok {
		select {
		case client.send <- message:
			socketLog("消息发送成功: %s -> %s", content["id"], to)
		default:
			socketLog("发送队列满，断开连接: %s", client.info.ID)
			close(client.send)
			delete(hub.clients, client.info.ID)
		}
	} else {
		socketLog("目标客户端离线，缓存消息: %s", targetId)
		hub.storeOfflineMessage(targetId, message)
	}
}

func (hub *hub) sendOfflineMessages(client *client, state *clientState) {
	now := time.Now()
	validMsgs := []*offlineMessage{}

	for _, msg := range state.offlineMsgs {
		if now.Sub(msg.sentAt) < hub.offlineMsgTTL {
			validMsgs = append(validMsgs, msg)
			client.send <- msg.message
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

func (hub *hub) privatecast(message []byte) {
	content := Json(message)
	from := cast.ToString(content["id"])
	to := cast.ToString(content["to"])

	socketLog("私聊消息: %s -> %s", from, to)

	hub.GetOrCreateChatSession(from, to)

	if client, ok := hub.clients[to]; ok {
		msgId := cast.ToString(content["msg_id"])
		if msgId == "" {
			msgId = guid()
			content["msg_id"] = msgId
			message, _ = json.Marshal(content)
		}

		hub.messageStatuses[msgId] = &messageStatus{
			msgId:     msgId,
			readBy:    map[string]bool{from: false, to: false},
			delivered: map[string]bool{from: true, to: false},
			sentAt:    time.Now(),
		}

		select {
		case client.send <- message:
			hub.messageStatuses[msgId].delivered[to] = true
			socketLog("私聊消息发送成功: %s -> %s", from, to)
		default:
			socketLog("私聊发送队列满: %s", to)
			hub.storeOfflineMessage(to, message)
		}
	} else {
		socketLog("目标客户端离线，缓存私聊消息: %s", to)
		hub.storeOfflineMessage(to, message)
	}
}
