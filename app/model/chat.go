package model

import (
	"errors"
	"fmt"
	"regexp"
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
 * 用户私聊（私信）
 *
 * 会话与消息分两张表：
 *   - chat_conversations：两人一条，key 是「小 ID_大 ID」（唯一），只存最后一条消息的摘要，
 *     会话列表按 last_time 倒序取即可，不必扫消息表；
 *   - chat_messages：消息本体，is_read 由接收方标记（进入会话即置 1）。
 *
 * 为什么未读数不落冗余字段：收发都是低频写，而「未读」必须和 is_read 强一致，
 * 冗余计数一旦漏更新（并发、补发、后台改数据）就再也对不上。这里改成
 * 「按会话一次 group by 聚合」，会话列表最多 50 行，一次 SQL 就够。
 *
 * 与 socket 的关系（见 app/socket/controller/index.go）：
 *   - 落库只走本文件的 Send —— socket 收发的私聊与 HTTP 兜底发送共用同一条写入路径，
 *     这样无论消息从哪来，历史记录、未读数、会话列表行为都一致；
 *   - socket 只负责「实时送达」，离线期间的消息由本表兜底（进页面拉历史）。
 */

// ChatConversations - 会话表
type ChatConversations struct {
	Id        int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Key       string `gorm:"size:48; uniqueIndex:uk_chat_conversations_key; comment:会话键（小UID_大UID）;" json:"key"`
	Uid       int    `gorm:"type:int(32); index:idx_chat_conversations_uid; comment:参与者A（较小的用户ID）;" json:"uid"`
	TargetUid int    `gorm:"type:int(32); index:idx_chat_conversations_target; comment:参与者B（较大的用户ID）;" json:"target_uid"`
	LastMsgId int    `gorm:"type:int(32); comment:最后一条消息ID;" json:"last_msg_id"`
	LastText  string `gorm:"size:180; comment:最后一条消息摘要;" json:"last_text"`
	LastUid   int    `gorm:"type:int(32); comment:最后一条消息的发送者;" json:"last_uid"`
	LastTime  int64  `gorm:"index:idx_chat_conversations_last_time; comment:最后一条消息时间;" json:"last_time"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// ChatMessages - 消息表
type ChatMessages struct {
	Id             int    `gorm:"type:int(32); comment:主键;" json:"id"`
	ConversationId int    `gorm:"type:int(32); index:idx_chat_messages_conversation; comment:会话ID;" json:"conversation_id"`
	SendUid        int    `gorm:"type:int(32); index:idx_chat_messages_send; comment:发送者;" json:"send_uid"`
	ReceiveUid     int    `gorm:"type:int(32); index:idx_chat_messages_receive; comment:接收者;" json:"receive_uid"`
	MsgType        string `gorm:"size:16; default:'text'; comment:消息类型 text/image/pat;" json:"msg_type"`
	Content        string `gorm:"size:2000; comment:消息内容（pat 时为拍一拍后缀）;" json:"content"`
	IsRead         int    `gorm:"type:tinyint(1); default:0; index:idx_chat_messages_unread; comment:是否已读;" json:"is_read"`
	ReadTime       int64  `gorm:"comment:已读时间;" json:"read_time"`
	// IsRevoked - 撤回标记：内容就地清空，保留这一行，让对方界面能显示「撤回了一条消息」
	IsRevoked  int   `gorm:"type:tinyint(1); default:0; comment:是否已撤回;" json:"is_revoked"`
	RevokeTime int64 `gorm:"comment:撤回时间;" json:"revoke_time"`
	// ClientMsgId - 客户端生成的幂等键：socket 重发 / 用户连点 / 网络重试都靠它去重
	ClientMsgId string `gorm:"size:64; default:''; comment:客户端消息ID（幂等）;" json:"client_msg_id"`
	// 引用回复：只存被引用消息的快照（id / 发送者 / 类型 / 摘要）。
	//
	// 为什么快照而不只存 quote_id：被引用的消息随时可能撤回（内容会被清空），
	// 只存 id 的话，引用气泡会在对方撤回后变成空白；快照让「引用」在历史里始终可读。
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

// ChatMessageDeletes - 「按人删除」的私信消息
//
// 私信是双方共同的记录，删除只对操作者生效：这里记一条 (uid, msg_id)，
// 历史记录 / 增量补拉 / 未读数都把它过滤掉，对方与其它设备不受影响。
// 用独立表而不是在 messages 上加字段：一条消息可能被两个人各删一次，
// 也可以被删除后又因为重新进入会话而「恢复」（删掉这行即可）。
type ChatMessageDeletes struct {
	Id         int   `gorm:"type:int(32); comment:主键;" json:"id"`
	Uid        int   `gorm:"type:int(32); uniqueIndex:uk_chat_message_deletes,priority:1; comment:删除者;" json:"uid"`
	MsgId      int   `gorm:"type:int(32); uniqueIndex:uk_chat_message_deletes,priority:2; comment:消息ID;" json:"msg_id"`
	CreateTime int64 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
}

// ChatStates - 每个用户在每个会话里的同步水位
//
// 一张表管两个位置，避免「逐条已读」表随消息量膨胀：
//   - LastAckSeq：客户端确认收到的最大 seq（批量 ACK，见 PUT /chat/ack）；
//   - LastReadSeq：用户真正读到（打开会话）的最大 seq（PUT /chat/read）。
//
// 消息的 seq 直接复用自增主键（见 chatMessageMap 的 seq 字段）：
// 自增值全局单调，会话内自然单调，客户端只需记「我收到的最大 seq」，
// 重连时用 after_seq 补拉即可，不需要额外维护一套会话内序号生成器。
type ChatStates struct {
	Id             int   `gorm:"type:int(32); comment:主键;" json:"id"`
	ConversationId int   `gorm:"type:int(32); uniqueIndex:uk_chat_states,priority:1; comment:会话ID;" json:"conversation_id"`
	Uid            int   `gorm:"type:int(32); uniqueIndex:uk_chat_states,priority:2; comment:用户ID;" json:"uid"`
	LastAckSeq     int64 `gorm:"comment:已确认收到的最大 seq;" json:"last_ack_seq"`
	LastReadSeq    int64 `gorm:"comment:已读到的最大 seq;" json:"last_read_seq"`
	// 以下为公共字段
	Json       any                   `gorm:"type:longtext; comment:用于存储JSON数据;" json:"json"`
	Text       any                   `gorm:"type:longtext; comment:用于存储文本数据;" json:"text"`
	Result     any                   `gorm:"type:varchar(256); comment:不存储数据，用于封装返回结果;" json:"result"`
	CreateTime int64                 `gorm:"autoCreateTime; comment:创建时间;" json:"create_time"`
	UpdateTime int64                 `gorm:"autoUpdateTime; comment:更新时间;" json:"update_time"`
	DeleteTime soft_delete.DeletedAt `gorm:"comment:删除时间; default:0;" json:"delete_time"`
}

// AfterFind - 查询Hook
func (this *ChatMessages) AfterFind(tx *gorm.DB) (err error) {
	this.Text = cast.ToString(this.Text)
	this.Json = utils.Json.Decode(this.Json)
	return
}

func InitChatConversations() {
	if err := facade.DB.Drive().AutoMigrate(&ChatConversations{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatConversations表迁移失败")
	}
}

func InitChatMessages() {
	if err := facade.DB.Drive().AutoMigrate(&ChatMessages{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatMessages表迁移失败")
	}
}

func InitChatStates() {
	if err := facade.DB.Drive().AutoMigrate(&ChatStates{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatStates表迁移失败")
	}
}

func InitChatMessageDeletes() {
	if err := facade.DB.Drive().AutoMigrate(&ChatMessageDeletes{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatMessageDeletes表迁移失败")
	}
}

// chatHiddenQuery - 我删除过的消息 id 子查询
//
// 删除只对自己生效，所以所有「我的消息视图」都必须带上这个条件：
// 历史 / 增量补拉 / 未读总数 / 会话未读数。漏掉任何一处，删掉的消息都会
// 在刷新或未读数里重新冒出来。
func chatHiddenQuery(uid int) *gorm.DB {
	return facade.DB.Drive().Model(&ChatMessageDeletes{}).Select("msg_id").Where("uid = ?", uid)
}

// chatConversationId - 按「双方 UID」取会话 ID
//
// 没有会话（还没发过消息）时返回 0：没消息就没有水位要记，
// 调用方据此直接跳过写库，不必先 GetOrCreate 造一条空会话。
func chatConversationId(uid, targetUid int) int {
	_, _, key := ChatPair(uid, targetUid)

	var conversation ChatConversations
	if err := facade.DB.Drive().Select("id").Where("`key` = ?", key).First(&conversation).Error; err != nil {
		return 0
	}

	return conversation.Id
}

// chatStateAdvance - 推进某个水位（只增不减），返回推进后的值
//
// 为什么用 ON CONFLICT：ACK 是高频写，先查再插会有并发窗口；
// 唯一索引 uk_chat_states 冲突时直接 UPDATE，一条 SQL 搞定。
// CASE WHEN 保证乱序到达（比如客户端重连后补发的旧 ACK）不会把水位拨回去。
func chatStateAdvance(uid, targetUid int, seq int64, read bool) int64 {

	if uid <= 0 || targetUid <= 0 || uid == targetUid || seq <= 0 {
		return 0
	}

	conversationId := chatConversationId(uid, targetUid)
	if conversationId <= 0 {
		return 0
	}

	column := "last_ack_seq"
	state := ChatStates{ConversationId: conversationId, Uid: uid}
	if read {
		column = "last_read_seq"
		state.LastReadSeq = seq
	} else {
		state.LastAckSeq = seq
	}

	advance := gorm.Expr("CASE WHEN `"+column+"` < ? THEN ? ELSE `"+column+"` END", seq, seq)

	if err := facade.DB.Drive().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "conversation_id"}, {Name: "uid"}},
		DoUpdates: clause.Assignments(map[string]any{column: advance, "update_time": time.Now().Unix()}),
	}).Create(&state).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": uid, "target_uid": targetUid, "seq": seq}, "更新私信同步水位失败")
		return 0
	}

	lastAck, lastRead := chatStateOf(conversationId, uid)
	if read {
		return lastRead
	}
	return lastAck
}

// chatStateOf - 直接按「会话 ID + 用户」读水位（调用方已经有会话 ID，省一次查询）
func chatStateOf(conversationId, uid int) (lastAck, lastRead int64) {

	if conversationId <= 0 || uid <= 0 {
		return 0, 0
	}

	var state ChatStates
	if err := facade.DB.Drive().
		Where("conversation_id = ? AND uid = ?", conversationId, uid).
		First(&state).Error; err != nil {
		return 0, 0
	}

	return state.LastAckSeq, state.LastReadSeq
}

// Ack - 客户端批量确认「已收到 seq 及之前的消息」
func (this *ChatStates) Ack(uid, targetUid int, seq int64) int64 {
	return chatStateAdvance(uid, targetUid, seq, false)
}

// Read - 用户已读到 seq（打开会话 / 滚到最新时上报）
func (this *ChatStates) Read(uid, targetUid int, seq int64) int64 {
	return chatStateAdvance(uid, targetUid, seq, true)
}

// State - 我在某个会话里的两个水位（没有记录时都是 0）
func (this *ChatStates) State(uid, targetUid int) (lastAck, lastRead int64) {

	if uid <= 0 || targetUid <= 0 || uid == targetUid {
		return 0, 0
	}

	conversationId := chatConversationId(uid, targetUid)
	if conversationId <= 0 {
		return 0, 0
	}

	return chatStateOf(conversationId, uid)
}

// ChatPair - 归一化会话双方：始终 小ID 在前，保证两人只有一条会话
func ChatPair(a, b int) (small int, large int, key string) {
	if a > b {
		a, b = b, a
	}
	return a, b, fmt.Sprintf("%d_%d", a, b)
}

// chatEmojiTag - 正文里的表情标记 [emoji:url]（与前端 @/utils/emoji 同一套协议）
var chatEmojiTag = regexp.MustCompile(`\[emoji:[^\]]+\]`)

// ChatPreview - 会话列表里的消息摘要（列表宽度只有一行的容量）
//
// 表情是 [emoji:url] 标记，直接塞进摘要会变成一长串链接，统一换成 [表情]。
func ChatPreview(msgType, content string) string {
	if msgType == "image" {
		return "[图片]"
	}
	if msgType == "pat" {
		return "[拍一拍]"
	}

	text := chatEmojiTag.ReplaceAllString(content, "[表情]")
	text = strings.TrimSpace(text)

	runes := []rune(text)
	if len(runes) > 60 {
		return string(runes[:60]) + "…"
	}
	return text
}

// isChatImageUrl - 私信图片地址白名单
//
// 与前端 @/utils/emoji 的 safeEmojiUrl 同一口径：
// 只接受 http(s):// 或站内 / 开头，且不含空白、引号、尖括号（防止属性注入）。
func isChatImageUrl(url string) bool {
	value := strings.TrimSpace(url)
	if value == "" || len(value) > 1024 {
		return false
	}
	if !strings.HasPrefix(value, "http://") &&
		!strings.HasPrefix(value, "https://") &&
		!strings.HasPrefix(value, "/") {
		return false
	}
	return !strings.ContainsAny(value, " \t\r\n\"'<>")
}

// chatMessageMap - 消息的统一输出结构（HTTP 与 socket 推送共用，前端只认这一份）
func chatMessageMap(item *ChatMessages) map[string]any {
	return map[string]any{
		"id": item.Id,
		// seq 即自增主键：客户端用它做去重和断线补拉的游标（after_seq）
		"seq":             item.Id,
		"conversation_id": item.ConversationId,
		"from_uid":        item.SendUid,
		"to_uid":          item.ReceiveUid,
		"msg_type":        utils.Default(item.MsgType, "text"),
		"content":         item.Content,
		"is_read":         item.IsRead,
		"is_revoked":      item.IsRevoked,
		"revoke_time":     item.RevokeTime,
		"client_msg_id":   item.ClientMsgId,
		"quote":           chatQuoteMap(item),
		"create_time":     item.CreateTime,
	}
}

// chatQuoteMap - 引用消息的输出结构（没有引用时返回 nil，前端据此不渲染引用块）
func chatQuoteMap(item *ChatMessages) any {
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

// chatQuoteText - 被引用消息的摘要快照
//
// 与 ChatPreview 的区别：这里保留 [emoji:url] 标记（前端能直接渲染成表情），
// 所以不能直接把预览文案当快照。图片 / 拍一拍本来就只适合显示类型标签。
func chatQuoteText(msgType, content string) string {
	if msgType == "image" {
		return "[图片]"
	}
	if msgType == "pat" {
		return "[拍一拍]"
	}

	runes := []rune(strings.TrimSpace(content))
	if len(runes) > 80 {
		return string(runes[:80]) + "…"
	}
	return string(runes)
}

// GetOrCreate - 取（或建）两人之间的会话
//
// 并发下两个人同时给对方发第一条消息会同时 INSERT：key 上有唯一索引，
// 冲突时 OnConflict DoNothing 后重新读一次即可。
func (this *ChatConversations) GetOrCreate(uid, targetUid int) (*ChatConversations, error) {
	_, _, key := ChatPair(uid, targetUid)

	db := facade.DB.Drive()

	var table ChatConversations
	err := db.Where("`key` = ?", key).First(&table).Error
	if err == nil && table.Id > 0 {
		return &table, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	small, large, _ := ChatPair(uid, targetUid)
	table = ChatConversations{
		Key:       key,
		Uid:       small,
		TargetUid: large,
	}
	if createErr := db.Create(&table).Error; createErr != nil {
		// 唯一索引冲突（并发创建）：读一次已有的
		if findErr := db.Where("`key` = ?", key).First(&table).Error; findErr == nil {
			return &table, nil
		}
		return nil, createErr
	}

	return &table, nil
}

// 私信业务常量
const (
	// chatRevokeWindow - 允许撤回的时间窗（秒）：超过就只能当历史记录留着
	chatRevokeWindow int64 = 120
	// chatPatInterval - 同一对用户两次拍一拍的最小间隔（秒），防连点刷屏
	chatPatInterval int64 = 5
	// chatPatSuffixMax - 拍一拍后缀最大字符数（如「的后脑勺」）
	chatPatSuffixMax = 20
)

// ChatPermission - 私信发送权限（唯一的判定入口）
//
// 私信默认只对「互相关注」的好友开放：
//   - 自己 → 拒绝；
//   - 任一方向拉黑 → 拒绝（拉黑优先于互关，提示也更具体）；
//   - 未互关 → 拒绝。
//
// 放在 model 层而不是 HTTP 控制器：socket 私聊同样走 Send，
// 只在 HTTP 里拦等于给 socket 留了一条绕过通道。
func ChatPermission(uid, targetUid int) error {
	if uid <= 0 || targetUid <= 0 {
		return errors.New("参数不完整！")
	}
	if uid == targetUid {
		return errors.New("不能给自己发私信！")
	}
	// 私信禁言优先于关系判断：被后台禁言的人，跟谁的关系再好也不能发
	// （与聊天室禁言同一张表，room_id = ChatMuteDM 表示私信作用域）
	if muted, until := (&ChatRoomMutes{}).Active(ChatMuteDM, uid); muted {
		if until > 0 {
			return fmt.Errorf("你的私信功能已被禁言，%s 后自动解除！", time.Unix(until, 0).Format("01-02 15:04"))
		}
		return errors.New("你的私信功能已被禁言！")
	}
	if !(&UserBlocks{}).CanInteract(uid, targetUid) {
		return errors.New("你们之间存在拉黑关系，无法私聊！")
	}
	if !(&UserFollows{}).IsMutual(uid, targetUid) {
		return errors.New("仅互相关注的好友可以私信！")
	}
	return nil
}

// Send - 写入一条私聊消息，并同步会话的最后一条摘要
//
// 参数：
//   - msgType：text（默认）/ image（内容为图片地址）/ pat（拍一拍，内容为后缀）
//   - clientMsgId：客户端幂等键，重复提交直接返回已有消息，不重复入库
//   - quoteId：引用的消息 ID（0 = 不引用）；服务端回查原消息并落快照，
//     不信任客户端传来的引用内容
//
// 注意：pat 只应由 Pat() 调用（那里有频率限制）；HTTP 与 socket 发送入口都会把
// msg_type 限制为 text/image，避免有人用 /send 绕过拍一拍的节流。
func (this *ChatMessages) Send(sendUid, receiveUid int, msgType, content, clientMsgId string, quoteId int) (map[string]any, error) {

	if sendUid <= 0 || receiveUid <= 0 {
		return nil, errors.New("参数不完整！")
	}
	if sendUid == receiveUid {
		return nil, errors.New("不能给自己发私信！")
	}
	if err := ChatPermission(sendUid, receiveUid); err != nil {
		return nil, err
	}

	msgType = strings.TrimSpace(msgType)
	if msgType == "" {
		msgType = "text"
	}
	if !utils.InArray(msgType, []string{"text", "image", "pat"}) {
		return nil, errors.New("不支持的消息类型！")
	}

	content = strings.TrimSpace(content)
	// 拍一拍可以不带后缀（等于「拍了拍你」），其它类型不允许空内容
	if msgType != "pat" && content == "" {
		return nil, errors.New("消息内容不能为空！")
	}
	if len([]rune(content)) > 2000 {
		return nil, errors.New("消息内容太长了！")
	}
	if msgType == "pat" && len([]rune(content)) > chatPatSuffixMax {
		return nil, errors.New("拍一拍文案太长了！")
	}

	// 图片消息的内容是地址：只放行 http(s) 与站内相对路径，且不允许空白 / 引号 / 尖括号。
	// 前端渲染时虽然只塞进 <img src>，但聊天内容还会进会话摘要、日志、后台列表，
	// 存进来之前就把非法值挡掉，比事后每处都转义更可靠。
	if msgType == "image" && !isChatImageUrl(content) {
		return nil, errors.New("图片地址不合法！")
	}

	// 敏感词：文本消息与拍一拍后缀都要过全站词库（图片地址不检查）。
	// 私信场景只按 block / limit 拦截；review / warn / log 放行 —— 放行的消息
	// 在入库后留一条「命中片段」作为证据（见 NoteChatModeration），后台据此人工判断违规。
	sensitiveHit := SensitiveDecision{}
	if msgType != "image" {
		sensitiveHit = (&Sensitive{}).Decide(SceneDM, sendUid, content)
		if sensitiveHit.Blocked() {
			return nil, errors.New(sensitiveHit.UserMessage())
		}
	}

	// 幂等：同一发送者 + 同一 client_msg_id 只入库一次
	clientMsgId = strings.TrimSpace(clientMsgId)
	if clientMsgId != "" {
		var exist ChatMessages
		if err := facade.DB.Drive().Where("send_uid = ? AND client_msg_id = ?", sendUid, clientMsgId).First(&exist).Error; err == nil && exist.Id > 0 {
			return chatMessageMap(&exist), nil
		}
	}

	// 引用回复：服务端回查原消息并落快照，客户端只传 quote_id。
	//
	// 校验三件事：存在、未撤回、属于同一段会话 —— 否则就能跨会话引用，
	// 甚至引用一条自己根本看不到的消息。
	var quote ChatMessages
	if quoteId > 0 {
		if err := facade.DB.Drive().Where("id = ?", quoteId).First(&quote).Error; err != nil || quote.Id <= 0 {
			return nil, errors.New("引用的消息不存在！")
		}
		if quote.IsRevoked == 1 {
			return nil, errors.New("引用的消息已撤回！")
		}
		samePair := (quote.SendUid == sendUid && quote.ReceiveUid == receiveUid) ||
			(quote.SendUid == receiveUid && quote.ReceiveUid == sendUid)
		if !samePair {
			return nil, errors.New("只能引用当前会话里的消息！")
		}
	}

	conversation, err := (&ChatConversations{}).GetOrCreate(sendUid, receiveUid)
	if err != nil {
		facade.Log.Error(map[string]any{"error": err, "send_uid": sendUid, "receive_uid": receiveUid}, "获取会话失败")
		return nil, errors.New("发送失败，请稍后重试！")
	}

	message := ChatMessages{
		ConversationId: conversation.Id,
		SendUid:        sendUid,
		ReceiveUid:     receiveUid,
		MsgType:        msgType,
		Content:        content,
		IsRead:         0,
		ClientMsgId:    clientMsgId,
	}

	if quote.Id > 0 {
		message.QuoteId = quote.Id
		message.QuoteUid = quote.SendUid
		message.QuoteType = utils.Default(quote.MsgType, "text")
		message.QuoteText = chatQuoteText(quote.MsgType, quote.Content)
	}

	db := facade.DB.Drive()
	tx := db.Begin()
	if tx.Error != nil {
		return nil, errors.New("发送失败，请稍后重试！")
	}

	if err := tx.Create(&message).Error; err != nil {
		tx.Rollback()
		facade.Log.Error(map[string]any{"error": err}, "私信写入失败")
		return nil, errors.New("发送失败，请稍后重试！")
	}

	now := time.Now().Unix()
	if message.CreateTime <= 0 {
		message.CreateTime = now
	}

	if err := tx.Model(&ChatConversations{}).Where("id = ?", conversation.Id).Updates(map[string]any{
		"last_msg_id": message.Id,
		"last_text":   ChatPreview(msgType, content),
		"last_uid":    sendUid,
		"last_time":   message.CreateTime,
	}).Error; err != nil {
		tx.Rollback()
		facade.Log.Error(map[string]any{"error": err}, "更新会话摘要失败")
		return nil, errors.New("发送失败，请稍后重试！")
	}

	if err := tx.Commit().Error; err != nil {
		return nil, errors.New("发送失败，请稍后重试！")
	}

	// 命中词库留证：只存命中片段，不存整条消息（后台「聊天审核」据此判断违规）
	if sensitiveHit.Action != "" && sensitiveHit.Action != ActionAllow {
		NoteChatModeration(SceneDM, sendUid, receiveUid, content, sensitiveHit, message.Id)
	}

	return chatMessageMap(&message), nil
}

// History - 与某人的历史消息（游标分页：取 beforeId 之前的一页，返回时按时间正序）
//
// 用 id 游标而不是 page：聊天记录会不断新增，page 翻页在「期间来了新消息」时会错位。
func (this *ChatMessages) History(uid, targetUid, beforeId, limit int) (list []map[string]any, hasMore bool) {

	if uid <= 0 || targetUid <= 0 {
		return []map[string]any{}, false
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	query := facade.DB.Drive().Model(&ChatMessages{}).
		Where("(send_uid = ? AND receive_uid = ?) OR (send_uid = ? AND receive_uid = ?)",
			uid, targetUid, targetUid, uid).
		// 我删除过的消息不再出现在我的历史里（删除只对自己生效）
		Where("id NOT IN (?)", chatHiddenQuery(uid))

	if beforeId > 0 {
		query = query.Where("id < ?", beforeId)
	}

	// 多取一条用于判断是否还有更早的记录
	var rows []ChatMessages
	if err := query.Order("id desc").Limit(limit + 1).Find(&rows).Error; err != nil {
		return []map[string]any{}, false
	}

	hasMore = len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	list = make([]map[string]any, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		list = append(list, chatMessageMap(&rows[i]))
	}

	return list, hasMore
}

// Sync - 断线重连补拉：取 seq（即消息 id）大于 afterSeq 的消息，按时间正序返回
//
// 与 History 的 beforeId 正好相反：补拉只关心「本地已有的最大 seq 之后还有哪些」，
// 所以用 id 升序 + limit。返回值：
//   - list：这一批消息（可能为空，说明没漏）；
//   - lastSeq：这批里最大的 seq（没有新消息时原样返回 afterSeq），
//     客户端仍可以用它继续往后拉；
//   - hasMore：还有更多（用 lastSeq 当新的 after_seq 再来一次）。
//
// 只靠 id 游标就够：消息 id 全局单调，同一会话内取 id > afterSeq 不会漏也不会重。
func (this *ChatMessages) Sync(uid, targetUid int, afterSeq int64, limit int) (list []map[string]any, lastSeq int64, hasMore bool) {

	if uid <= 0 || targetUid <= 0 || uid == targetUid {
		return []map[string]any{}, afterSeq, false
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if afterSeq < 0 {
		afterSeq = 0
	}

	var rows []ChatMessages
	err := facade.DB.Drive().Model(&ChatMessages{}).
		Where("(send_uid = ? AND receive_uid = ?) OR (send_uid = ? AND receive_uid = ?)",
			uid, targetUid, targetUid, uid).
		Where("id NOT IN (?)", chatHiddenQuery(uid)).
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
		list = append(list, chatMessageMap(&rows[i]))
	}

	lastSeq = afterSeq
	if len(rows) > 0 {
		lastSeq = int64(rows[len(rows)-1].Id)
	}

	return list, lastSeq, hasMore
}

// MarkRead - 把「某人发给我的未读消息」全部标记为已读
//
// 返回值：
//   - rows：本次标记的条数；
//   - lastId：这次读到的最后一条消息 id。
//     它用来给对方发已读回执（对方把 id <= lastId 的「我发出的消息」标成已读），
//     比逐条回执省事，也不会因为消息多而刷屏。
func (this *ChatMessages) MarkRead(uid, targetUid int) (rows int64, lastId int) {

	if uid <= 0 || targetUid <= 0 {
		return 0, 0
	}

	db := facade.DB.Drive()

	// 先取「对方发来的最后一条」：一条都没有就不必写库
	// （不按 is_read 过滤：别的设备已经读过时，本地水位仍要往前推）
	var last ChatMessages
	if err := db.Where("receive_uid = ? AND send_uid = ?", uid, targetUid).
		Order("id desc").First(&last).Error; err != nil || last.Id <= 0 {
		return 0, 0
	}

	tx := db.Model(&ChatMessages{}).
		Where("receive_uid = ? AND send_uid = ? AND is_read = 0", uid, targetUid).
		Updates(map[string]any{"is_read": 1, "read_time": time.Now().Unix()})

	if tx.Error != nil {
		return 0, 0
	}

	// 读到的位置同时记进同步水位：客户端重连时不必再逐条比对 is_read，
	// 只需比较自己的 seq 与 last_read_seq 就知道有没有新东西
	(&ChatStates{}).Read(uid, targetUid, int64(last.Id))

	return tx.RowsAffected, last.Id
}

// Revoke - 撤回一条自己发的消息
//
// 撤回不清行、只清内容并打标记：
//   - 对方要看到「XX 撤回了一条消息」，历史记录也不能凭空少一条；
//   - 未读数仍按原样统计（撤回不产生新未读，也不消别人的未读）。
//
// 返回撤回后的消息结构（调用方据此推送 chat_revoke）。
func (this *ChatMessages) Revoke(uid, msgId int) (map[string]any, error) {

	if uid <= 0 || msgId <= 0 {
		return nil, errors.New("参数不完整！")
	}

	var message ChatMessages
	if err := facade.DB.Drive().Where("id = ?", msgId).First(&message).Error; err != nil || message.Id <= 0 {
		return nil, errors.New("消息不存在！")
	}
	if message.SendUid != uid {
		return nil, errors.New("只能撤回自己的消息！")
	}
	if message.IsRevoked == 1 {
		return nil, errors.New("消息已经撤回了！")
	}
	if time.Now().Unix()-message.CreateTime > chatRevokeWindow {
		return nil, errors.New("超过 2 分钟的消息不能撤回！")
	}

	now := time.Now().Unix()
	if err := facade.DB.Drive().Model(&ChatMessages{}).Where("id = ?", message.Id).Updates(map[string]any{
		"is_revoked":  1,
		"content":     "",
		"revoke_time": now,
	}).Error; err != nil {
		return nil, errors.New("撤回失败，请稍后重试！")
	}

	message.IsRevoked = 1
	message.Content = ""
	message.RevokeTime = now

	// 撤回的如果是会话最后一条，摘要要退回到上一条还在的消息，
	// 否则会话列表会一直挂着一条已经看不到的内容
	this.refreshConversationSummary(message.ConversationId)

	return chatMessageMap(&message), nil
}

// Delete - 删除一条消息（只对自己生效）
//
// 与 Revoke 的区别：
//   - 撤回：清空内容并广播给对方，双方都看到「撤回了一条消息」；
//   - 删除：只把这条消息从「我的」历史里藏起来，对方的记录完全不动。
//
// 因此这里不删消息行、也不改内容，只写一条 (uid, msg_id) 记录；
// 历史 / 补拉 / 未读数都会带上过滤条件（见 chatHiddenQuery）。
func (this *ChatMessages) Delete(uid, msgId int) error {

	if uid <= 0 || msgId <= 0 {
		return errors.New("参数不完整！")
	}

	var message ChatMessages
	if err := facade.DB.Drive().Where("id = ?", msgId).First(&message).Error; err != nil || message.Id <= 0 {
		return errors.New("消息不存在！")
	}
	if message.SendUid != uid && message.ReceiveUid != uid {
		return errors.New("无权删除该消息！")
	}

	row := ChatMessageDeletes{Uid: uid, MsgId: msgId}
	if err := facade.DB.Drive().Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err, "uid": uid, "msg_id": msgId}, "删除私信消息失败")
		return errors.New("删除失败，请稍后重试！")
	}

	return nil
}

// 不做真实推送弹窗，只作为聊天记录里的一行灰色提示，
// 这样「谁拍了谁、拍了什么」双方都能在历史里回看。
func (this *ChatMessages) Pat(fromUid, toUid int, suffix string) (map[string]any, error) {

	if fromUid <= 0 || toUid <= 0 {
		return nil, errors.New("参数不完整！")
	}
	if fromUid == toUid {
		return nil, errors.New("不能拍自己！")
	}

	suffix = strings.TrimSpace(suffix)
	if len([]rune(suffix)) > chatPatSuffixMax {
		return nil, errors.New("拍一拍文案太长了！")
	}

	// 节流：同一对用户短时间连点只算一次，避免刷屏
	var last ChatMessages
	_ = facade.DB.Drive().
		Where("send_uid = ? AND receive_uid = ? AND msg_type = ?", fromUid, toUid, "pat").
		Order("id desc").First(&last).Error
	if last.Id > 0 && time.Now().Unix()-last.CreateTime < chatPatInterval {
		return nil, errors.New("拍得太快啦，歇一下再拍！")
	}

	return this.Send(fromUid, toUid, "pat", suffix, "", 0)
}

// refreshConversationSummary - 重新计算会话的最后一条消息摘要
//
// 只用于撤回（消息还在追加时，Send 里已经顺手更新过了）。
func (this *ChatMessages) refreshConversationSummary(conversationId int) {

	if conversationId <= 0 {
		return
	}

	db := facade.DB.Drive()

	var conversation ChatConversations
	if err := db.Where("id = ?", conversationId).First(&conversation).Error; err != nil || conversation.Id <= 0 {
		return
	}

	updates := map[string]any{
		"last_msg_id": 0,
		"last_text":   "",
		"last_uid":    0,
	}

	var last ChatMessages
	if err := db.Where("conversation_id = ? AND is_revoked = 0", conversationId).
		Order("id desc").First(&last).Error; err == nil && last.Id > 0 {
		updates["last_msg_id"] = last.Id
		updates["last_text"] = ChatPreview(last.MsgType, last.Content)
		updates["last_uid"] = last.SendUid
		updates["last_time"] = last.CreateTime
	}

	_ = db.Model(&ChatConversations{}).Where("id = ?", conversationId).Updates(updates).Error
}

// UnreadTotal - 我的私信未读总数（左栏 / 用户中心角标用）
func (this *ChatMessages) UnreadTotal(uid int) int64 {
	if uid <= 0 {
		return 0
	}
	count, _ := facade.DB.Model(&ChatMessages{}).
		Where("receive_uid = ? AND is_read = 0", uid).
		Where("id NOT IN (?)", chatHiddenQuery(uid)).
		Count()
	return count
}

// unreadByConversation - 我各个会话的未读数（一次 group by 取回，避免 N+1）
func (this *ChatMessages) unreadByConversation(uid int, ids []int) map[int]int64 {

	result := make(map[int]int64)
	if uid <= 0 || len(ids) == 0 {
		return result
	}

	var rows []struct {
		ConversationId int
		Total          int64
	}

	err := facade.DB.Drive().Model(&ChatMessages{}).
		Select("conversation_id, COUNT(*) AS total").
		Where("receive_uid = ? AND is_read = 0 AND conversation_id IN ?", uid, ids).
		Where("id NOT IN (?)", chatHiddenQuery(uid)).
		Group("conversation_id").
		Scan(&rows).Error
	if err != nil {
		return result
	}

	for _, row := range rows {
		result[row.ConversationId] = row.Total
	}

	return result
}

// ListMine - 我的会话列表（按最后消息时间倒序）
//
// 返回结构（other_uid 已换算成「对方」，前端不用再关心表里谁是 uid 谁是 target_uid）：
//
//	{ id, other_uid, last_msg_id, last_text, last_uid, last_time, unread }
func (this *ChatConversations) ListMine(uid, limit int) []map[string]any {

	if uid <= 0 {
		return []map[string]any{}
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows []ChatConversations
	_ = facade.DB.Drive().
		Where("uid = ? OR target_uid = ?", uid, uid).
		Order("last_time desc").Limit(limit).Find(&rows).Error

	ids := make([]int, 0, len(rows))
	for _, item := range rows {
		ids = append(ids, item.Id)
	}

	unread := (&ChatMessages{}).unreadByConversation(uid, ids)

	// 批量取「各会话最后一条是否被我删除」：一次 IN 查询，避免会话列表 N+1
	hiddenLast := make(map[int]bool)
	lastIds := make([]int, 0, len(rows))
	for _, item := range rows {
		if item.LastMsgId > 0 {
			lastIds = append(lastIds, item.LastMsgId)
		}
	}
	if len(lastIds) > 0 {
		var hidden []struct{ MsgId int }
		_ = facade.DB.Drive().Model(&ChatMessageDeletes{}).
			Select("msg_id").
			Where("uid = ? AND msg_id IN ?", uid, lastIds).
			Scan(&hidden).Error
		for _, item := range hidden {
			hiddenLast[item.MsgId] = true
		}
	}

	list := make([]map[string]any, 0, len(rows))
	for _, item := range rows {
		other := item.TargetUid
		if item.Uid != uid {
			other = item.Uid
		}

		lastMsgId, lastText := item.LastMsgId, item.LastText
		lastUid, lastTime := item.LastUid, item.LastTime

		// 会话摘要存在会话表里、是双方共用的：如果最后一条正好被我删了，
		// 我这一侧的列表不能继续显示那条内容（否则删了还在），
		// 但又不能改表（会把对方的摘要一起改掉），所以只在本人的列表里回退到
		// 「我还能看到的上一条」。对方看到的摘要不受影响。
		if lastMsgId > 0 && hiddenLast[lastMsgId] {
			lastMsgId, lastText, lastUid, lastTime = 0, "", 0, 0
			var last ChatMessages
			if err := facade.DB.Drive().
				Where("conversation_id = ?", item.Id).
				Where("id NOT IN (?)", chatHiddenQuery(uid)).
				Order("id desc").First(&last).Error; err == nil && last.Id > 0 {
				lastMsgId = last.Id
				lastText = ChatPreview(last.MsgType, last.Content)
				lastUid = last.SendUid
				lastTime = last.CreateTime
			}
		}

		list = append(list, map[string]any{
			"id":          item.Id,
			"other_uid":   other,
			"last_msg_id": lastMsgId,
			"last_text":   lastText,
			"last_uid":    lastUid,
			"last_time":   lastTime,
			"unread":      unread[item.Id],
		})
	}

	return list
}

// ConversationUnread - 我在某个会话里的未读数（进入会话后不回 0 时用于提示）
func (this *ChatMessages) ConversationUnread(uid, targetUid int) int64 {
	if uid <= 0 || targetUid <= 0 {
		return 0
	}
	count, _ := facade.DB.Model(&ChatMessages{}).
		Where("receive_uid = ? AND send_uid = ? AND is_read = 0", uid, targetUid).
		Where("id NOT IN (?)", chatHiddenQuery(uid)).
		Count()
	return count
}
