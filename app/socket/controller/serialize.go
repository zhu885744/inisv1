package controller

import (
	"encoding/json"
	"fmt"
	"inis/config"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/cast"
)

var socket = websocket.Upgrader{
	// 读取存储空间大小
	ReadBufferSize: 1024,
	// 写入存储空间大小
	WriteBufferSize: 1024,
	// 允许跨域
	CheckOrigin: func(request *http.Request) bool {
		return true
	},
}

// 客户端是 websocket 连接和集线器之间的中间人。
type client struct {
	hub *hub
	// 客户端信息
	info *info
	// 出站消息的缓冲通道。
	send chan []byte
	// websocket 连接。
	conn *websocket.Conn
}

// 客户端信息结构体
type info struct {
	ID      string `json:"id"`
	To      string `json:"to"`
	Type    string `json:"type"`
	Content any    `json:"data"`
	MsgId   string `json:"msg_id,omitempty"`
}

type ackInfo struct {
	ID    string `json:"id"`
	MsgId string `json:"msg_id"`
	Type  string `json:"type"`
}

type pendingMessage struct {
	message []byte
	client  *client
	attempt int
	sentAt  time.Time
}

type clientState struct {
	clientId    string
	lastSeen    time.Time
	offlineMsgs []*offlineMessage
}

type offlineMessage struct {
	message []byte
	sentAt  time.Time
}

type rateLimit struct {
	timestamp time.Time
	count     int
}

type ipConnection struct {
	clientIds []string
	lastSeen  time.Time
}

type stats struct {
	totalConnections   int
	currentConnections int
	totalMessages      int64
	broadcastMessages  int64
	singleMessages     int64
	ackSuccess         int64
	ackFailed          int64
	offlineMessages    int64
	reconnectCount     int64
	rateLimitHits      int64
	ipLimitHits        int64
	startTime          time.Time
	messageLatency     []time.Duration
	maxLatency         time.Duration
	minLatency         time.Duration
	avgLatency         time.Duration
}

type securityConfig struct {
	blacklistedIPs    map[string]bool
	allowedOrigins    []string
	enableOriginCheck bool
	maxMessageSize    int
}

type chatSession struct {
	roomId       string
	participants []string
	lastMessage  time.Time
}

type messageStatus struct {
	msgId     string
	readBy    map[string]bool
	delivered map[string]bool
	sentAt    time.Time
}

type privateMessage struct {
	from    string
	to      string
	content map[string]any
	msgId   string
	sentAt  time.Time
	status  string
}

// Hub 维护活跃客户端的集合，并将消息广播
type hub struct {
	clients              map[string]*client
	notice               chan []byte
	connect              chan *client
	close                chan *client
	status               chan map[string]any
	pendingMessages      map[string]*pendingMessage
	ackTimeout           time.Duration
	maxRetries           int
	clientStates         map[string]*clientState
	offlineMsgTTL        time.Duration
	maxOfflineMsgs       int
	reconnectTimeout     time.Duration
	ipConnections        map[string]*ipConnection
	rateLimits           map[string]*rateLimit
	maxConnectionsPerIP  int
	maxMessagesPerMinute int
	stats                *stats
	security             *securityConfig
	chatSessions         map[string]*chatSession
	messageStatuses      map[string]*messageStatus
}

var Hub = func() *hub {
	return &hub{
		notice:               make(chan []byte),
		connect:              make(chan *client),
		close:                make(chan *client),
		status:               make(chan map[string]any),
		clients:              make(map[string]*client),
		pendingMessages:      make(map[string]*pendingMessage),
		clientStates:         make(map[string]*clientState),
		ipConnections:        make(map[string]*ipConnection),
		rateLimits:           make(map[string]*rateLimit),
		chatSessions:         make(map[string]*chatSession),
		messageStatuses:      make(map[string]*messageStatus),
		ackTimeout:           10 * time.Second,
		maxRetries:           3,
		offlineMsgTTL:        5 * time.Minute,
		maxOfflineMsgs:       100,
		reconnectTimeout:     30 * time.Second,
		maxConnectionsPerIP:  10,
		maxMessagesPerMinute: 100,
		stats: &stats{
			startTime:  time.Now(),
			minLatency: time.Hour,
		},
		security: &securityConfig{
			blacklistedIPs:    make(map[string]bool),
			allowedOrigins:    []string{},
			enableOriginCheck: false,
			maxMessageSize:    1024 * 1024,
		},
	}
}()

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	// 增加到120秒，给客户端更多响应时间
	pongWait = 120 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024
)

var (
	line  = []byte{'\n'}
	space = []byte{' '}
)

func getSocketConfig(key string, defaultValue any) any {
	return config.AppToml.Get("socket."+key, defaultValue)
}

func socketLog(format string, args ...any) {
	debug := cast.ToBool(getSocketConfig("debug", false))
	if debug {
		fmt.Printf("[socket] "+format+"\n", args...)
	}
}

func (hub *hub) checkIPLimit(ip string) bool {
	if conn, ok := hub.ipConnections[ip]; ok {
		if len(conn.clientIds) >= hub.maxConnectionsPerIP {
			socketLog("IP连接数超限: %s, 当前连接数: %d", ip, len(conn.clientIds))
			hub.stats.ipLimitHits++
			return false
		}
	}
	return true
}

func (hub *hub) addIPConnection(ip string, clientId string) {
	if conn, ok := hub.ipConnections[ip]; ok {
		conn.clientIds = append(conn.clientIds, clientId)
		conn.lastSeen = time.Now()
	} else {
		hub.ipConnections[ip] = &ipConnection{
			clientIds: []string{clientId},
			lastSeen:  time.Now(),
		}
	}
}

func (hub *hub) removeIPConnection(ip string, clientId string) {
	if conn, ok := hub.ipConnections[ip]; ok {
		for i, id := range conn.clientIds {
			if id == clientId {
				conn.clientIds = append(conn.clientIds[:i], conn.clientIds[i+1:]...)
				break
			}
		}
		if len(conn.clientIds) == 0 {
			delete(hub.ipConnections, ip)
		}
	}
}

func (hub *hub) checkRateLimit(clientId string) bool {
	now := time.Now()
	if rl, ok := hub.rateLimits[clientId]; ok {
		if now.Sub(rl.timestamp) < time.Minute {
			if rl.count >= hub.maxMessagesPerMinute {
				socketLog("消息频率超限: %s, 每分钟消息数: %d", clientId, rl.count)
				hub.stats.rateLimitHits++
				return false
			}
			rl.count++
		} else {
			rl.timestamp = now
			rl.count = 1
		}
	} else {
		hub.rateLimits[clientId] = &rateLimit{
			timestamp: now,
			count:     1,
		}
	}
	return true
}

func (hub *hub) recordConnection() {
	hub.stats.totalConnections++
	hub.stats.currentConnections = len(hub.clients)
}

func (hub *hub) recordDisconnection() {
	hub.stats.currentConnections = len(hub.clients)
}

func (hub *hub) recordMessage(msgType string) {
	hub.stats.totalMessages++
	if msgType == "broadcast" {
		hub.stats.broadcastMessages++
	} else if msgType == "single" {
		hub.stats.singleMessages++
	}
}

func (hub *hub) recordAck(success bool) {
	if success {
		hub.stats.ackSuccess++
	} else {
		hub.stats.ackFailed++
	}
}

func (hub *hub) recordOfflineMessage() {
	hub.stats.offlineMessages++
}

func (hub *hub) recordReconnect() {
	hub.stats.reconnectCount++
}

func (hub *hub) recordLatency(latency time.Duration) {
	hub.stats.messageLatency = append(hub.stats.messageLatency, latency)
	if latency > hub.stats.maxLatency {
		hub.stats.maxLatency = latency
	}
	if latency < hub.stats.minLatency {
		hub.stats.minLatency = latency
	}
	if len(hub.stats.messageLatency) > 0 {
		total := time.Duration(0)
		for _, l := range hub.stats.messageLatency {
			total += l
		}
		hub.stats.avgLatency = total / time.Duration(len(hub.stats.messageLatency))
	}
}

func (hub *hub) recordIPLimitHit() {
	hub.stats.ipLimitHits++
}

func (hub *hub) GetStats() map[string]any {
	uptime := time.Since(hub.stats.startTime)
	return map[string]any{
		"uptime":              uptime.String(),
		"total_connections":   hub.stats.totalConnections,
		"current_connections": hub.stats.currentConnections,
		"total_messages":      hub.stats.totalMessages,
		"broadcast_messages":  hub.stats.broadcastMessages,
		"single_messages":     hub.stats.singleMessages,
		"ack_success":         hub.stats.ackSuccess,
		"ack_failed":          hub.stats.ackFailed,
		"offline_messages":    hub.stats.offlineMessages,
		"reconnect_count":     hub.stats.reconnectCount,
		"rate_limit_hits":     hub.stats.rateLimitHits,
		"ip_limit_hits":       hub.stats.ipLimitHits,
		"max_latency":         hub.stats.maxLatency.String(),
		"min_latency":         hub.stats.minLatency.String(),
		"avg_latency":         hub.stats.avgLatency.String(),
	}
}

func (hub *hub) AddToBlacklist(ip string) {
	hub.security.blacklistedIPs[ip] = true
	socketLog("IP已加入黑名单: %s", ip)
}

func (hub *hub) RemoveFromBlacklist(ip string) {
	delete(hub.security.blacklistedIPs, ip)
	socketLog("IP已从黑名单移除: %s", ip)
}

func (hub *hub) IsBlacklisted(ip string) bool {
	return hub.security.blacklistedIPs[ip]
}

func (hub *hub) AddAllowedOrigin(origin string) {
	for _, o := range hub.security.allowedOrigins {
		if o == origin {
			return
		}
	}
	hub.security.allowedOrigins = append(hub.security.allowedOrigins, origin)
	socketLog("已添加允许的Origin: %s", origin)
}

func (hub *hub) RemoveAllowedOrigin(origin string) {
	for i, o := range hub.security.allowedOrigins {
		if o == origin {
			hub.security.allowedOrigins = append(hub.security.allowedOrigins[:i], hub.security.allowedOrigins[i+1:]...)
			socketLog("已移除允许的Origin: %s", origin)
			return
		}
	}
}

func (hub *hub) IsOriginAllowed(origin string) bool {
	if !hub.security.enableOriginCheck {
		return true
	}
	for _, o := range hub.security.allowedOrigins {
		if o == origin || o == "*" {
			return true
		}
	}
	return false
}

func (hub *hub) ValidateMessage(message []byte) (bool, string) {
	if len(message) > hub.security.maxMessageSize {
		return false, "消息超过最大长度限制"
	}

	if !json.Valid(message) {
		return false, "无效的JSON格式"
	}

	var content map[string]any
	if err := json.Unmarshal(message, &content); err != nil {
		return false, "JSON解析失败"
	}

	if _, ok := content["type"]; !ok {
		return false, "缺少必需字段: type"
	}

	msgType := cast.ToString(content["type"])
	allowedTypes := []string{"broadcast", "single", "private", "ack", "status", "read", "ping", "pong"}
	found := false
	for _, t := range allowedTypes {
		if t == msgType {
			found = true
			break
		}
	}
	if !found {
		return false, "不允许的消息类型: " + msgType
	}

	if msgType == "single" || msgType == "private" {
		if _, ok := content["to"]; !ok {
			return false, "私聊消息缺少必需字段: to"
		}
	}

	return true, ""
}

func (hub *hub) GetOrCreateChatSession(user1, user2 string) string {
	if user1 > user2 {
		user1, user2 = user2, user1
	}
	roomId := user1 + "_" + user2

	if _, ok := hub.chatSessions[roomId]; !ok {
		hub.chatSessions[roomId] = &chatSession{
			roomId:       roomId,
			participants: []string{user1, user2},
			lastMessage:  time.Now(),
		}
	}

	return roomId
}

func (hub *hub) SendPrivateMessage(from, to string, content map[string]any) (string, error) {
	msgId := guid()

	if client, ok := hub.clients[to]; ok {
		msg := &privateMessage{
			from:    from,
			to:      to,
			content: content,
			msgId:   msgId,
			sentAt:  time.Now(),
			status:  "sent",
		}

		data, _ := json.Marshal(map[string]any{
			"type":    "private",
			"from":    from,
			"to":      to,
			"msg_id":  msgId,
			"content": content,
			"sent_at": msg.sentAt.Unix(),
		})

		hub.messageStatuses[msgId] = &messageStatus{
			msgId:     msgId,
			readBy:    map[string]bool{from: false, to: false},
			delivered: map[string]bool{from: true, to: false},
			sentAt:    time.Now(),
		}

		select {
		case client.send <- data:
			hub.messageStatuses[msgId].delivered[to] = true
			return msgId, nil
		default:
			return msgId, fmt.Errorf("发送队列满")
		}
	} else {
		hub.storeOfflineMessage(to, content["data"].([]byte))
		return msgId, nil
	}
}

func (hub *hub) MarkMessageRead(msgId, userId string) {
	if status, ok := hub.messageStatuses[msgId]; ok {
		status.readBy[userId] = true
	}
}

func (hub *hub) GetUnreadCount(userId string) int {
	count := 0
	for _, status := range hub.messageStatuses {
		if !status.readBy[userId] {
			for _, participant := range hub.chatSessions {
				for _, p := range participant.participants {
					if p == userId {
						count++
						break
					}
				}
			}
		}
	}
	return count
}

func (hub *hub) GetChatHistory(user1, user2 string, limit int) []*privateMessage {
	var history []*privateMessage
	return history
}
