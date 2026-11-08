package model

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"inis/app/facade"

	"github.com/spf13/cast"
)

// PushChatRevoke - 机审撤回消息后的实时推送钩子
//
// model 不能直接 import socket（socket 已经 import model，会形成循环依赖），
// 由 app/socket/controller 在 init 里注入 socket.PushPrivate。
// 未注入时（例如离线任务）只改库，对方下次拉历史就能看到「已撤回」。
var PushChatRevoke func(targetUid int, payload map[string]any)

/**
 * 云端内容审核（异步兜底）
 *
 * 分工：
 *   - 本地词库（model/sensitive.go）在请求链路里同步判定，命中 block 直接拒绝，
 *     命中 review 把内容置为待审核；
 *   - 这里接第三方机审（百度文本审核），**只做收紧**：
 *       block  → audit = 2（驳回，前台立即不可见）
 *       review → audit = 0（进人工审核队列）
 *       pass   → 不改动任何状态
 *
 * 为什么 pass 不改状态：内容可能因为本地词库命中 review 而处于待审核，
 * 机审放行并不能替代人工复核；自动改成「通过」会绕过人工审核这道关。
 *
 * 为什么异步：第三方审核单次要 100~800ms，评论 / 私信是高频写，
 * 同步调用会把发布接口拖慢并放大第三方抖动。任务进内存队列，
 * 由固定数量的协程消费，队列满了直接丢弃（本地词库已做过同步快筛）。
 */

const (
	// 内容表 audit 字段取值（与 api/controller/audit.go 的判定保持一致）
	AuditStatusPending = 0 // 待审核
	AuditStatusPassed  = 1 // 审核通过
	AuditStatusReject  = 2 // 审核未通过
)

// cloudAuditTask - 一次待审任务
type cloudAuditTask struct {
	Scene    string
	BindType string
	BindId   int
	Uid      int
	// ToUid 私信接收者（聊天室为 0），写违规留证时用
	ToUid int
	Text  string
	// ImageURL / SavePath 图像审核用（BindType = image）
	ImageURL string
	SavePath string
}

var (
	cloudAuditQueue chan cloudAuditTask
	cloudAuditOnce  sync.Once
)

// SubmitCloudAudit - 提交一次异步机审
//
// scene 用 SceneArticle / SceneMoment / SceneComment；
// bindType 是内容表类型（article / moment / comment / pages），用于回写 audit。
func SubmitCloudAudit(scene string, uid int, bindType string, bindId int, text string) {

	if !facade.AuditEnabled() || bindId <= 0 {
		return
	}
	if strings.TrimSpace(text) == "" {
		return
	}

	cloudAuditOnce.Do(startCloudAuditWorkers)

	task := cloudAuditTask{
		Scene:    scene,
		BindType: bindType,
		BindId:   bindId,
		Uid:      uid,
		Text:     text,
	}

	select {
	case cloudAuditQueue <- task:
	default:
		// 队列满说明第三方变慢或流量突增：丢弃本次，本地词库已经拦过一遍
		facade.Log.Error(map[string]any{
			"scene":     scene,
			"bind_type": bindType,
			"bind_id":   bindId,
		}, "云端审核队列已满，本次内容未送审")
	}
}

func startCloudAuditWorkers() {

	setting := facade.AuditBaiduSetting()
	cloudAuditQueue = make(chan cloudAuditTask, setting.QueueSize)

	for i := 0; i < setting.Workers; i++ {
		go cloudAuditWorker()
	}
}

func cloudAuditWorker() {
	for task := range cloudAuditQueue {
		runCloudAudit(task)
	}
}

func runCloudAudit(task cloudAuditTask) {

	defer func() {
		if r := recover(); r != nil {
			facade.Log.Error(map[string]any{
				"panic":     r,
				"bind_type": task.BindType,
				"bind_id":   task.BindId,
			}, "云端审核任务执行异常")
		}
	}()

	// 图像审核走独立分支：不写敏感词命中，改的是附件记录
	if task.BindType == "image" {
		runCloudImageAudit(task)
		return
	}

	result, err := facade.BaiduTextAudit(task.Text)
	if err != nil {
		// 第三方不可用 / 熔断 / 未配置：不影响已发布的内容，只记日志
		facade.Log.Error(map[string]any{
			"error":     err.Error(),
			"bind_type": task.BindType,
			"bind_id":   task.BindId,
		}, "云端审核失败")
		return
	}

	switch result.Conclusion {

	case facade.BaiduAuditBlock:
		decision := SensitiveDecision{
			Action:       ActionBlock,
			Word:         truncateRunes(strings.Join(result.Words, ","), 60),
			Level:        LevelHigh,
			Match:        AuditProviderBaiduMatch,
			Scene:        task.Scene,
			Version:      "baidu",
			AuditType:    result.AuditType,
			AuditSubType: result.AuditSubType,
			AuditMsg:     firstAuditMessage(result.Messages),
		}
		if isChatMessageBind(task.BindType) {
			// 私信 / 聊天室：撤回本条 + 写违规留证（后台「聊天审核」复核）
			revokeChatByCloud(task.Scene, task.BindId)
			NoteChatModeration(task.Scene, task.Uid, task.ToUid, task.Text, decision, task.BindId)
		} else {
			// 驳回原因写进 reason：后台审核列表与作者自己的列表都靠它提示
			reason := cloudAuditReason(facade.BaiduAuditBlock, result)
			auditApplyStatus(task.BindType, task.BindId, AuditStatusReject, reason)
			notifyCloudAudit(task.BindType, task.BindId, AuditStatusReject, reason)
		}
		recordSensitiveHit(task.Scene, task.Uid, task.Text, decision)

	case facade.BaiduAuditReview:
		decision := SensitiveDecision{
			Action:       ActionReview,
			Word:         truncateRunes(strings.Join(result.Words, ","), 60),
			Level:        LevelMedium,
			Match:        AuditProviderBaiduMatch,
			Scene:        task.Scene,
			Version:      "baidu",
			AuditType:    result.AuditType,
			AuditSubType: result.AuditSubType,
			AuditMsg:     firstAuditMessage(result.Messages),
		}
		if isChatMessageBind(task.BindType) {
			// 疑似违规不撤回（私信隐私敏感，与本地词库的放行策略一致），只留证
			NoteChatModeration(task.Scene, task.Uid, task.ToUid, task.Text, decision, task.BindId)
		} else {
			// 待审核同样要留原因：人工复核时能看到「为什么被机审挑出来」
			reason := cloudAuditReason(facade.BaiduAuditReview, result)
			auditApplyStatus(task.BindType, task.BindId, AuditStatusPending, reason)
			notifyCloudAudit(task.BindType, task.BindId, AuditStatusPending, reason)
		}
		recordSensitiveHit(task.Scene, task.Uid, task.Text, decision)

	case facade.BaiduAuditFailed:
		// 百度侧「审核失败」（conclusionType = 4）：服务异常，不是内容判定。
		// 这里不改 audit、不发通知 —— 否则第三方一抖动，正常内容就会被批量打进待审
		facade.Log.Error(map[string]any{
			"scene":     task.Scene,
			"bind_type": task.BindType,
			"bind_id":   task.BindId,
		}, "云端审核返回审核失败，本次不做处置")
	}
}

// isChatMessageBind - 该任务对应的是私信 / 聊天室消息（而不是内容表）
func isChatMessageBind(bindType string) bool {
	return bindType == "dm-message" || bindType == "chat-message"
}

// AuditProviderBaiduMatch - 审计里标记来源，便于区分「本地词库」与「云端机审」
const AuditProviderBaiduMatch = "cloud"

// auditApplyStatus - 回写内容表的 audit 状态与驳回原因
//
// 只处理同时有 audit / reason 字段的内容表；私信 / 聊天室的消息表没有这两个字段，
// 由聊天的违规留证（ChatModeration）单独处理，不在本函数范围内。
//
// 通过（AuditStatusPassed）时不写 reason：人工通过时后端会清空原因，
// 机审放行若覆盖成空串也没意义，直接不动更安全。
func auditApplyStatus(bindType string, id int, status int, reason string) {

	if id <= 0 {
		return
	}

	var table any
	switch bindType {
	case "article":
		table = &Article{}
	case "moment":
		table = &Moments{}
	case "comment":
		table = &Comment{}
	case "pages":
		table = &Pages{}
	default:
		return
	}

	updates := map[string]any{"audit": status}
	if status != AuditStatusPassed && reason != "" {
		updates["reason"] = truncateRunes(reason, 200)
	}

	if err := facade.DB.Drive().Model(table).Where("id = ?", id).Updates(updates).Error; err != nil {
		facade.Log.Error(map[string]any{
			"error":     err.Error(),
			"bind_type": bindType,
			"bind_id":   id,
			"audit":     status,
		}, "回写云端审核结果失败")
	}
}

// cloudAuditReason - 把机审结论转成给作者 / 审核员看的原因
//
// 文案要能直接显示在列表里，所以控制在 200 字以内：
//   - 驳回：说清是机审判定，附最多 3 个命中词，作者知道从哪改；
//   - 疑似：说明还在等人工复核，避免作者以为已经定案。
func cloudAuditReason(conclusion string, result facade.BaiduAuditResult) string {

	prefix := "AI审核驳回："
	if conclusion == facade.BaiduAuditReview {
		prefix = "AI审核疑似违规："
	}

	// 1. 百度给的判定说明最贴近实际情况
	//    （如「存在女上露点不合规」「存在第一代领导人姓名及职务不合规」），优先用它
	if len(result.Messages) > 0 {
		return truncateRunes(prefix+strings.Join(result.Messages, "；"), 200)
	}

	// 2. 只有命中词时退回到命中词的写法
	hits := make([]string, 0, 3)
	for _, word := range result.Words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		hits = append(hits, truncateRunes(word, 20))
		if len(hits) >= 3 {
			break
		}
	}
	if len(hits) > 0 {
		return truncateRunes(prefix+"命中违规内容："+strings.Join(hits, "、"), 200)
	}

	// 3. 两者都没有：模型判了违规但没给具体维度，只说明结论
	if conclusion == facade.BaiduAuditReview {
		return "AI审核疑似违规：已转人工复核"
	}

	return "AI审核驳回：内容含有违规信息，已下架"
}

// firstAuditMessage - 取第一条判定说明（审计表只需要一句）
func firstAuditMessage(messages []string) string {
	for _, message := range messages {
		message = strings.TrimSpace(message)
		if message != "" {
			return truncateRunes(message, 200)
		}
	}
	return ""
}

// imageAuditText - 图像审核结果转成附件的审计文案
//
// 附件表只有 audit_result 一个摘要字段（后台附件库展示），
// 同样优先用百度给的 msg（「存在女上露点不合规」），比类型码直观。
func imageAuditText(result facade.BaiduAuditResult, prefix string) string {

	if len(result.Messages) > 0 {
		return truncateRunes(prefix+"："+strings.Join(result.Messages, "；"), 240)
	}
	if len(result.Words) > 0 {
		return truncateRunes(prefix+"："+strings.Join(result.Words, "、"), 240)
	}

	return prefix
}

// notifyCloudAudit - 机审驳回 / 转人工时给作者发系统消息
//
// 与人工审核（api/controller/mail-notify.go 的 notifyAuditChange）口径一致：
// 说清是哪条内容、为什么、发生在什么时候。
//
// 只发站内信、不发邮件：机审触发量大（尤其 review），邮件会变成骚扰；
// 需要邮件的话仍由人工审核那一步补齐，用户不会漏掉最终结论。
func notifyCloudAudit(bindType string, id int, status int, reason string) {

	if id <= 0 {
		return
	}

	var (
		uid   int
		title string
		label = "内容"
		typ   = NotificationTypeSystem
		bind  = bindType
	)

	// 查一次内容，拿作者与标题；bind_type 用消息中心的路由口径
	// （article / moments / page / comment，见前端 utils/notification.js）
	switch bindType {
	case "article":
		row, _ := facade.DB.Model(&Article{}).Find(id)
		uid = cast.ToInt(row["uid"])
		title = truncateRunes(cast.ToString(row["title"]), 40)
		label, typ, bind = "文章", NotificationTypeArticle, "article"
	case "moment":
		row, _ := facade.DB.Model(&Moments{}).Find(id)
		uid = cast.ToInt(row["uid"])
		title = truncateRunes(cast.ToString(row["content"]), 40)
		label, typ, bind = "动态", NotificationTypeMoments, "moments"
	case "comment":
		row, _ := facade.DB.Model(&Comment{}).Find(id)
		uid = cast.ToInt(row["uid"])
		title = truncateRunes(cast.ToString(row["content"]), 40)
		label, typ, bind = "评论", NotificationTypeSystem, "comment"
	case "pages":
		row, _ := facade.DB.Model(&Pages{}).Find(id)
		uid = cast.ToInt(row["uid"])
		title = truncateRunes(cast.ToString(row["title"]), 40)
		label, typ, bind = "独立页面", NotificationTypePage, "page"
	default:
		return
	}

	if uid <= 0 {
		return
	}

	nowText := MailNotifyTime()

	// 驳回：原因必须放最前面（消息列表内容区只有两行高度）
	if status == AuditStatusReject {
		titleText := "您的" + label + "未通过审核"
		content := "驳回原因：" + reason + " · 时间：" + nowText
		if title != "" {
			content = "内容：" + title + " · " + content
		}
		CreateUserNotify(uid, 0, typ, titleText, content, bind, id)
		return
	}

	// 转人工复核：说明还在处理中，避免作者以为已经定案
	titleText := "您的" + label + "待人工复核"
	content := reason + " · 时间：" + nowText
	if title != "" {
		content = "内容：" + title + " · " + content
	}
	CreateUserNotify(uid, 0, typ, titleText, content, bind, id)
}

// SubmitCloudChatAudit - 私信 / 聊天室的云端审核（异步）
//
// 消息表没有 audit 字段，处置方式与内容表不同：
//   - block：撤回本条消息（内容清空、保留行）+ 写违规留证，后台「聊天审核」可复核；
//   - review：只留证，不撤回 —— 私信隐私敏感，与本地词库对 review 的放行策略一致。
//
// 为什么异步：聊天是实时链路，第三方 200~800ms 的延迟不能加到发送上；
// 代价是「已发出后 1~2 秒才撤回」，本地词库仍然同步拦截明显违规。
func SubmitCloudChatAudit(scene string, uid, toUid, msgId int, content string) {

	if scene != SceneDM && scene != SceneChat {
		return
	}
	if !facade.AuditEnabled() || msgId <= 0 || strings.TrimSpace(content) == "" {
		return
	}

	cloudAuditOnce.Do(startCloudAuditWorkers)

	bindType := "dm-message"
	if scene == SceneChat {
		bindType = "chat-message"
	}

	task := cloudAuditTask{
		Scene:    scene,
		BindType: bindType,
		BindId:   msgId,
		Uid:      uid,
		ToUid:    toUid,
		Text:     content,
	}

	select {
	case cloudAuditQueue <- task:
	default:
		facade.Log.Error(map[string]any{
			"scene":  scene,
			"msg_id": msgId,
		}, "云端审核队列已满，本次消息未送审")
	}
}

// revokeChatByCloud - 云端审核命中 block 时撤回消息
//
// 不复用 ChatMessages.Revoke：那个方法要求「本人操作 + 2 分钟内」，
// 机审是系统行为且可能延迟到达，直接按 msg_id 更新更符合语义。
func revokeChatByCloud(scene string, msgId int) {

	if msgId <= 0 {
		return
	}

	now := time.Now().Unix()

	if scene == SceneDM {

		var message ChatMessages
		if err := facade.DB.Drive().Where("id = ?", msgId).First(&message).Error; err != nil || message.Id <= 0 {
			return
		}

		if err := facade.DB.Drive().Model(&ChatMessages{}).Where("id = ?", msgId).Updates(map[string]any{
			"is_revoked":  1,
			"content":     "",
			"revoke_time": now,
		}).Error; err != nil {
			facade.Log.Error(map[string]any{"error": err.Error(), "msg_id": msgId}, "云端审核撤回私信失败")
			return
		}

		// 撤回的是最后一条时，会话摘要要退回到上一条还在的消息
		(&ChatMessages{}).refreshConversationSummary(message.ConversationId)

		// 双方正在窗口里的就地替换成「已撤回」，不用刷新
		if PushChatRevoke != nil {
			var fresh ChatMessages
			if err := facade.DB.Drive().Where("id = ?", msgId).First(&fresh).Error; err == nil && fresh.Id > 0 {
				payload := chatMessageMap(&fresh)
				for _, targetUid := range []int{fresh.SendUid, fresh.ReceiveUid} {
					if targetUid <= 0 {
						continue
					}
					PushChatRevoke(targetUid, map[string]any{
						"type": "chat_revoke",
						"to":   fmt.Sprintf("user_%d", targetUid),
						"from": "system",
						"data": payload,
					})
				}
			}
		}

		return
	}

	if err := facade.DB.Drive().Model(&ChatRoomMessages{}).Where("id = ?", msgId).Updates(map[string]any{
		"is_revoked":  1,
		"content":     "",
		"revoke_uid":  0,
		"revoke_time": now,
	}).Error; err != nil {
		facade.Log.Error(map[string]any{"error": err.Error(), "msg_id": msgId}, "云端审核撤回聊天室消息失败")
	}
}

// cloudAuditImageExt - 需要送图像审核的扩展名
var cloudAuditImageExt = map[string]bool{
	"jpg":  true,
	"jpeg": true,
	"png":  true,
	"gif":  true,
	"webp": true,
	"bmp":  true,
}

// SubmitCloudImageAudit - 附件图像审核（异步）
//
// 与文本审核不同，图片审核不做同步拦截：百度拉图 / 解码要 200~800ms，
// 同步会让上传明显变慢。上传成功先返回地址，命中 block 后再删文件并软删记录。
func SubmitCloudImageAudit(attachId uint, imgURL string, savePath string) {

	if !facade.AuditEnabled() || attachId == 0 {
		return
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(savePath), "."))
	if !cloudAuditImageExt[ext] {
		return
	}

	cloudAuditOnce.Do(startCloudAuditWorkers)

	task := cloudAuditTask{
		Scene:    "image",
		BindType: "image",
		BindId:   int(attachId),
		ImageURL: imgURL,
		SavePath: savePath,
	}

	select {
	case cloudAuditQueue <- task:
	default:
		facade.Log.Error(map[string]any{"attachment_id": attachId}, "云端审核队列已满，本次图片未送审")
	}
}

// runCloudImageAudit - 图像审核结果处理
func runCloudImageAudit(task cloudAuditTask) {

	result, err := facade.BaiduImageAudit(task.ImageURL, task.SavePath)
	if err != nil {
		// 审核服务不可用：保持默认「通过」，不影响图片正常使用
		facade.Log.Error(map[string]any{
			"error":         err.Error(),
			"attachment_id": task.BindId,
		}, "图像审核失败")
		return
	}

	switch result.Conclusion {

	case facade.BaiduAuditBlock:
		auditApplyImage(task.BindId, 2, imageAuditText(result, "云端审核驳回"))
		// 先落审核状态再删文件：软删后记录仍在后台可查（按 audit=2 筛）
		deleteAttachmentFile(task.BindId)

	case facade.BaiduAuditReview:
		auditApplyImage(task.BindId, 0, imageAuditText(result, "云端审核疑似违规"))

	case facade.BaiduAuditPass:
		auditApplyImage(task.BindId, 1, "")

	case facade.BaiduAuditFailed:
		// 审核失败：保持附件默认状态，不删文件、不置待审
		facade.Log.Error(map[string]any{
			"attachment_id": task.BindId,
		}, "图像审核返回审核失败，本次不做处置")
	}
}

// auditApplyImage - 回写附件的审核状态
func auditApplyImage(attachId int, status int, result string) {

	if attachId <= 0 {
		return
	}

	if err := facade.DB.Drive().Model(&Attachment{}).Where("id = ?", attachId).Updates(map[string]any{
		"audit":        status,
		"audit_result": truncateRunes(result, 240),
		"audit_time":   time.Now().Unix(),
	}).Error; err != nil {
		facade.Log.Error(map[string]any{
			"error":         err.Error(),
			"attachment_id": attachId,
		}, "回写图像审核结果失败")
	}
}

// deleteAttachmentFile - 删除被驳回的图片
//
// 两层处理缺一不可：
//   - 删存储里的文件：图片 URL 已经写进内容正文，不删就还能直接访问；
//   - 软删附件记录：附件库 / 列表不再展示，后台仍能按 audit=2 查到这次处置。
func deleteAttachmentFile(attachId int) {

	var row Attachment
	if err := facade.DB.Drive().Where("id = ?", attachId).First(&row).Error; err != nil || row.Id == 0 {
		return
	}

	if row.SavePath != "" {
		if err := facade.Storage.Delete(row.SavePath); err != nil {
			facade.Log.Error(map[string]any{
				"error":         err.Error(),
				"attachment_id": attachId,
				"save_path":     row.SavePath,
			}, "删除违规图片文件失败")
		}
	}

	if err := facade.DB.Drive().Delete(&Attachment{}, attachId).Error; err != nil {
		facade.Log.Error(map[string]any{
			"error":         err.Error(),
			"attachment_id": attachId,
		}, "软删违规附件记录失败")
	}
}
