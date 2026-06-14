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
	// IP临时封禁相关
	ipBanEnabled   bool
	ipBanThreshold int
	ipBanDuration  time.Duration
	ipBanRecords   map[string]*ipBanRecord
}

// IP封禁记录
type ipBanRecord struct {
	hits       int
	bannedAt   time.Time
	isBanned   bool
	expireTime time.Time
}

var Hub = func() *hub {
	// 从 TOML 配置读取所有 socket 参数
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
		ipBanRecords:         make(map[string]*ipBanRecord),
		ackTimeout:           time.Duration(cast.ToInt(getSocketConfig("ack_timeout", 10))) * time.Second,
		maxRetries:           cast.ToInt(getSocketConfig("max_retries", 3)),
		offlineMsgTTL:        time.Duration(cast.ToInt(getSocketConfig("offline_msg_ttl", 300))) * time.Second,
		maxOfflineMsgs:       cast.ToInt(getSocketConfig("max_offline_msgs", 100)),
		reconnectTimeout:     time.Duration(cast.ToInt(getSocketConfig("reconnect_timeout", 30))) * time.Second,
		maxConnectionsPerIP:  cast.ToInt(getSocketConfig("max_connections_per_ip", 10)),
		maxMessagesPerMinute: cast.ToInt(getSocketConfig("max_messages_per_minute", 100)),
		ipBanEnabled:         cast.ToBool(getSocketConfig("enable_ip_ban", false)),
		ipBanThreshold:       cast.ToInt(getSocketConfig("ip_ban_threshold", 3)),
		ipBanDuration:        time.Duration(cast.ToInt(getSocketConfig("ip_ban_duration", 300))) * time.Second,
		stats: &stats{
			startTime:  time.Now(),
			minLatency: time.Hour,
		},
		security: &securityConfig{
			blacklistedIPs:    make(map[string]bool),
			allowedOrigins:    []string{},
			enableOriginCheck: false,
			maxMessageSize:    cast.ToInt(getSocketConfig("max_message_size", 1024*1024)),
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

// checkIPBan 检查IP是否被临时封禁
func (hub *hub) checkIPBan(ip string) bool {
	if !hub.ipBanEnabled {
		return false
	}
	if record, ok := hub.ipBanRecords[ip]; ok && record.isBanned {
		if time.Now().After(record.expireTime) {
			// 封禁到期，自动解封
			record.isBanned = false
			record.hits = 0
			socketLog("IP封禁到期自动解封: %s", ip)
			return false
		}
		socketLog("IP处于封禁中，拒绝连接: %s, 剩余时长: %.0f秒", ip, time.Until(record.expireTime).Seconds())
		return true
	}
	return false
}

// recordIPLimitHit 记录IP超限次数，达到阈值时临时封禁
func (hub *hub) recordIPLimitHit(ip string) {
	hub.stats.ipLimitHits++
	if !hub.ipBanEnabled {
		return
	}
	if record, ok := hub.ipBanRecords[ip]; ok {
		record.hits++
		if record.hits >= hub.ipBanThreshold && !record.isBanned {
			record.isBanned = true
			record.bannedAt = time.Now()
			record.expireTime = time.Now().Add(hub.ipBanDuration)
			socketLog("IP已达超限阈值，临时封禁: %s, 时长: %.0f秒", ip, hub.ipBanDuration.Seconds())
		}
	} else {
		hub.ipBanRecords[ip] = &ipBanRecord{
			hits: 1,
		}
	}
}

func (hub *hub) checkIPLimit(ip string) bool {
	// 先检查是否被封禁
	if hub.checkIPBan(ip) {
		return false
	}
	if conn, ok := hub.ipConnections[ip]; ok {
		if len(conn.clientIds) >= hub.maxConnectionsPerIP {
			socketLog("IP连接数超限: %s, 当前连接数: %d", ip, len(conn.clientIds))
			hub.recordIPLimitHit(ip)
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
