package controller

/**
 * 审核状态变化的邮件通知（文章 / 独立页面 / 友链共用）
 *
 * 三个控制器的审核规则是一致的，审核状态字段 audit 的取值也一致：
 *   0 = 待审核，1 = 审核通过，2 = 审核未通过（后台列表的「通过 / 驳回」按钮）
 *
 * 通知规则：
 *   - 首次（重新）进入待审核 → 通知管理员去审核（article.pending / page.pending / links.pending）
 *   - 变为通过 → 通知作者（*.passed）
 *   - 变为未通过 → 通知作者（*.rejected）
 * 只在状态真正变化时发送，普通编辑保存（audit 未变）不会打扰任何人。
 *
 * 开关与收件人见「系统设置 → 邮件通知」（config 表 SYSTEM_MAIL_NOTIFY），
 * 统一由 model.MailNotifyAdmin / MailNotifyUser 判断与投递（走邮箱队列，非阻塞）。
 */

import (
	"inis/app/model"

	"github.com/spf13/cast"
)

// notifyAuditChange 审核状态变化时发邮件
//
// uid      内容作者（收件人，未通过 / 通过时通知他）
// kind     内容类型：article / page / links（同时作为场景 key 前缀）
// title    内容标题（友链传名称）
// prevAudit 变更前的审核状态
// raw      本次提交里的 audit 值（payload["audit"]，为空表示本次没有改审核状态）
func notifyAuditChange(uid int, kind, title string, prevAudit int, raw any) {
	if uid <= 0 || raw == nil {
		return
	}

	now := cast.ToInt(raw)
	if now == prevAudit {
		return
	}

	label := "内容"
	switch kind {
	case "article":
		label = "文章"
	case "page":
		label = "独立页面"
	case "links":
		label = "友链"
	}

	lines := []string{
		label + "：" + title,
		"时间：" + model.MailNotifyTime(),
	}

	switch now {
	case 0:
		// 发给管理员：带上作者的身份信息（账号 / 昵称），方便核对是谁提交的
		model.MailNotifyAdmin(kind+".pending", "有"+label+"重新提交待审核", append(
			model.MailNotifyUserInfo(uid),
			lines...,
		)...)
	case 1:
		model.MailNotifyUser(uid, kind+".passed", "您的"+label+"已通过审核", lines...)
	case 2:
		model.MailNotifyUser(uid, kind+".rejected", "您的"+label+"未通过审核", lines...)
	}
}
