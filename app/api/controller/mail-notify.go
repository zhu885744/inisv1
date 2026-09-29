package controller

/**
 * 审核状态变化的通知（文章 / 独立页面 / 友链共用，邮件 + 站内信双通道）
 *
 * 这几个控制器的审核规则是一致的，审核状态字段 audit 的取值也一致：
 *   0 = 待审核，1 = 审核通过，2 = 审核未通过（后台列表的「通过 / 驳回」按钮）
 *
 * 通知规则：
 *   - 首次（重新）进入待审核 → 通知管理员去审核（article.pending / page.pending / links.pending）
 *   - 变为通过 → 通知作者（*.passed）
 *   - 变为未通过 → 通知作者（*.rejected），并带上管理员填写的驳回原因
 * 只在状态真正变化时发送，普通编辑保存（audit 未变）不会打扰任何人。
 *
 * 两个通道：
 *   - 邮件：model.MailNotifyAdmin / model.MailNotifyUser，开关见「系统设置 → 邮件通知」
 *     （config 表 SYSTEM_MAIL_NOTIFY，走邮箱队列，非阻塞）；
 *   - 站内信：model.NotifyAdmins / model.CreateUserNotify，类型为内容类型
 *     （article / page / links），不受邮件开关控制，保证作者能在「消息」中心看到驳回原因。
 *
 * 动态（moments）的审核文案要带正文摘要（动态没有标题），因此单独实现在
 * controller/moments.go 的 notifyMomentsAudit 里，复用同一套场景开关与站内信 helper。
 */

import (
	"inis/app/model"
	"strings"

	"github.com/spf13/cast"
)

// auditNotifyInfo 一次「审核状态变化」通知所需的上下文
type auditNotifyInfo struct {
	Uid       int    // 内容作者（未通过 / 通过时通知他）
	Kind      string // 内容类型：article / page / links（同时作为邮件场景 key 前缀与站内信类型）
	Title     string // 内容标题（友链传名称）
	BindId    int    // 内容 id（站内信点击跳转用）
	PrevAudit int    // 变更前的审核状态
	Audit     any    // 本次提交里的 audit 值（为 nil 表示本次没有改审核状态）
	Reason    string // 本次提交里的驳回原因（audit = 2 时展示给作者）
}

// notifyAuditChange 审核状态变化时发通知（邮件 + 站内信）
func notifyAuditChange(info auditNotifyInfo) {

	if info.Uid <= 0 || info.Audit == nil {
		return
	}

	now := cast.ToInt(info.Audit)
	if now == info.PrevAudit {
		return
	}

	label := "内容"
	switch info.Kind {
	case "article":
		label = "文章"
	case "page":
		label = "独立页面"
	case "links":
		label = "友链"
	}

	nowText := model.MailNotifyTime()
	reason := strings.TrimSpace(info.Reason)

	// 邮件正文：与其它场景保持一致的「字段名：值」风格
	lines := []string{
		label + "：" + info.Title,
		"时间：" + nowText,
	}
	if now == 2 && reason != "" {
		lines = append(lines, "驳回原因："+reason)
	}

	// 站内信正文：驳回原因放最前面（消息列表内容区只有 2 行高度，原因比时间更需要被看到）
	notifyContent := "时间：" + nowText
	if now == 2 && reason != "" {
		notifyContent = "驳回原因：" + reason + " · " + notifyContent
	}

	notifyType := notifyTypeOf(info.Kind)

	switch now {
	case 0:
		// 发给管理员：带上作者的身份信息（账号 / 昵称），方便核对是谁提交的
		title := "有" + label + "重新提交待审核"
		account, nickname := model.MailNotifyUserIdentity(info.Uid)
		model.MailNotifyAdmin(info.Kind+".pending", title, append(model.MailNotifyUserInfo(info.Uid), lines...)...)
		model.NotifyAdmins(info.Uid, notifyType, title,
			"作者："+nickname+"（"+account+"） · "+notifyContent, info.Kind, info.BindId)
	case 1:
		title := "您的" + label + "已通过审核"
		model.MailNotifyUser(info.Uid, info.Kind+".passed", title, lines...)
		model.CreateUserNotify(info.Uid, 0, notifyType, title, notifyContent, info.Kind, info.BindId)
	case 2:
		title := "您的" + label + "未通过审核"
		model.MailNotifyUser(info.Uid, info.Kind+".rejected", title, lines...)
		model.CreateUserNotify(info.Uid, 0, notifyType, title, notifyContent, info.Kind, info.BindId)
	}
}

// notifyTypeOf 内容类型 → 站内信通知类型（与前端「消息」中心的类型筛选一一对应）
func notifyTypeOf(kind string) string {
	switch kind {
	case "article":
		return model.NotificationTypeArticle
	case "page":
		return model.NotificationTypePage
	case "links":
		return model.NotificationTypeLinks
	default:
		return model.NotificationTypeSystem
	}
}
