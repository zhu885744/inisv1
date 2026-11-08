package model

import (
	"strings"

	"inis/app/facade"
)

/**
 * 聊天违规留证（私信 / 公共聊天室）
 *
 * 为什么需要：私信与聊天室对 review / warn / log 是**放行**的（隐私敏感，不塞人工审核队列），
 * 若不留证据，后台只剩一条统计数字，管理员无法判断「这句话到底算不算违规」。
 * 命中词库时留一条可复核的记录，就是后台「聊天审核」的数据来源。
 *
 * 隐私边界（最小必要）：
 *   - 只记录命中词库的消息，正常聊天完全不入库；
 *   - 只存命中词前后各 40 字的片段（sensitiveExcerpt），不存整条消息；
 *   - 图片消息不参与判定（地址没有语义），也就不会入队。
 *
 * 与 SensitiveHits 的分工：那张表做全场景统计（只有哈希），这张表是人工复核队列
 * （带片段、消息 ID 与处理状态），互不重复。
 */

// ChatModeration 命中片段的前后留白 / 总长度（rune）
const (
	chatModerationPadding = 40
	chatModerationSize    = 120
)

type ChatModeration struct {
	Id      int    `gorm:"type:int(32); comment:主键;" json:"id"`
	Scene   string `gorm:"size:16; index:idx_chat_moderation_scene; comment:场景（dm 私信 / chat 聊天室）;" json:"scene"`
	MsgId   int    `gorm:"type:int(32); comment:关联消息ID;" json:"msg_id"`
	Uid     int    `gorm:"type:int(32); index:idx_chat_moderation_uid; comment:发送者;" json:"uid"`
	ToUid   int    `gorm:"type:int(32); comment:私信接收者（聊天室为 0）;" json:"to_uid"`
	Action  string `gorm:"size:16; comment:命中规则的处置动作;" json:"action"`
	Word    string `gorm:"size:64; comment:命中的词;" json:"word"`
	Level   string `gorm:"size:16; comment:风险等级;" json:"level"`
	Excerpt string `gorm:"size:255; comment:命中片段（前后各 40 字）;" json:"excerpt"`

	Handled     int    `gorm:"type:int(32); default:0; index:idx_chat_moderation_handled; comment:是否已处理;" json:"handled"`
	HandledUid  int    `gorm:"type:int(32); default:0; comment:处理人;" json:"handled_uid"`
	HandledTime int64  `gorm:"comment:处理时间; default:0;" json:"handled_time"`
	Remark      string `gorm:"size:255; comment:处理备注;" json:"remark"`

	CreateTime int64 `gorm:"autoCreateTime; index:idx_chat_moderation_time; comment:创建时间;" json:"create_time"`
}

func InitChatModeration() {
	if err := facade.DB.Drive().AutoMigrate(&ChatModeration{}); err != nil {
		facade.Log.Error(map[string]any{"error": err}, "ChatModeration表迁移失败")
	}
}

// NoteChatModeration - 命中词库时留证
//
// 同步写入：调用方紧接着就要回写 msg_id（消息入库后才拿得到 ID），
// 一条 insert 相对消息本身的写入可忽略；失败只记日志，绝不影响消息发送。
func NoteChatModeration(scene string, uid, toUid int, content string, decision SensitiveDecision, msgId int) {

	if facade.DB == nil || uid <= 0 {
		return
	}
	if scene != SceneDM && scene != SceneChat {
		return
	}
	// 未命中（allow / 空动作）不入队
	if decision.Action == "" || decision.Action == ActionAllow {
		return
	}

	row := ChatModeration{
		Scene:   scene,
		MsgId:   msgId,
		Uid:     uid,
		ToUid:   toUid,
		Action:  decision.Action,
		Word:    decision.Word,
		Level:   decision.Level,
		Excerpt: sensitiveExcerpt(content, decision.Word),
	}

	if _, err := facade.DB.Model(&row).Create(&row); err != nil {
		facade.Log.Error(map[string]any{"error": err, "scene": scene, "uid": uid}, "写入聊天违规留证失败")
	}
}

// sensitiveExcerpt - 取命中词前后各 chatModerationPadding 字的片段
//
// 长消息里命中词可能出现在很靠后的位置，只截前 N 字会让管理员看不到「到底哪句话违规」。
// 归一化后命中、原文里找不到词（如「广 告」插了空格）时退化为截前 120 字。
func sensitiveExcerpt(text, word string) string {

	plain := strings.TrimSpace(sensitivePlainText(text))
	if plain == "" {
		return ""
	}

	runes := []rune(plain)
	start := 0

	if word != "" {
		if byteIndex := strings.Index(plain, word); byteIndex >= 0 {
			start = len([]rune(plain[:byteIndex])) - chatModerationPadding
			if start < 0 {
				start = 0
			}
		}
	}

	end := start + chatModerationSize
	if end > len(runes) {
		end = len(runes)
		start = end - chatModerationSize
		if start < 0 {
			start = 0
		}
	}

	excerpt := string(runes[start:end])
	if start > 0 {
		excerpt = "…" + excerpt
	}
	if end < len(runes) {
		excerpt += "…"
	}

	return excerpt
}
