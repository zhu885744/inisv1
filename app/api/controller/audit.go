package controller

import (
	"inis/app/facade"
	"inis/app/model"

	"github.com/spf13/cast"
	"github.com/unti-io/go-utils/utils"
)

/**
 * 内容审核（文章 / 动态 / 独立页面共用）
 *
 * 「是否开启审核」由各内容模块自己的配置决定：config 表的 ARTICLE / MOMENTS / PAGE 记录里的
 * json.audit（1 开启 / 0 关闭），入口是「系统设置 → 文章配置 / 动态配置 / 独立页面配置 → 内容审核」。
 * 审核结果写在内容表的 audit 字段上：
 *
 *   0 = 待审核，1 = 审核通过，2 = 审核未通过（后台列表的「通过 / 驳回」按钮）
 *
 * 统一口径（此前三个控制器各写一份判定，改一处容易漏掉另外两处）：
 *
 *   新建（create）
 *     - 存草稿（status = 0）：记为「通过」—— 草稿不对外展示，无需审核
 *     - 发布：关闭审核 → 直接「通过」；开启审核 → 「待审核」
 *
 *   编辑（update）
 *     - 存草稿：记为「通过」
 *     - 关闭审核：一律「通过」
 *     - 开启审核：只有「首次发布」进入「待审核」（原文是草稿，或原文本就没有审核结果），
 *       其余情况沿用原审核状态 —— 已有内容二次编辑保存不会被打回待审核
 *
 * 管理员仍可在编辑页 / 列表里显式指定 audit：payload 里的 audit 在写库前覆盖这里的判定
 * （allowFields 只对 root 开放 audit，普通作者提交的 audit 会被忽略）。
 *
 * 草稿概念：文章（article）与动态（moments）有 status（0 草稿 / 1 发布），
 * 独立页面（pages）没有 status 字段，因此 pages 的 draft 恒为 false。
 */

// 审核状态取值（内容表 audit 字段）
const (
	AuditPending = 0 // 待审核
	AuditPassed  = 1 // 审核通过
	AuditReject  = 2 // 审核未通过
)

// contentAuditSwitch 读取某个内容模块的审核开关（config 表 json.audit）
//
// key 传 "ARTICLE" / "MOMENTS" / "PAGE" / "COMMENT" / "LINKS"。
//
// 缺省返回 true（开启审核）：老站点的配置里可能还没有 audit 字段，
// 「缺配置就当成需要审核」比「缺配置就放开」安全，也与四份种子里的默认值（audit = 1）一致。
func contentAuditSwitch(key string) bool {
	item, _ := facade.DB.Model(&model.Config{}).Where("key", key).Find()
	if utils.Is.Empty(item) {
		return true
	}
	if value, exist := cast.ToStringMap(item["json"])["audit"]; exist {
		return cast.ToBool(value)
	}
	return true
}

// auditForCreate 新建内容时应写入的审核状态
//
// auditSwitch - 该内容模块是否开启审核
// draft       - 本次是否存草稿
func auditForCreate(auditSwitch bool, draft bool) int {

	// 草稿不对外展示，无需审核；未开启审核时一律直接通过
	if draft || !auditSwitch {
		return AuditPassed
	}

	return AuditPending
}

// auditForUpdate 编辑已存在内容时应写入的审核状态
//
// 返回 (audit, ok)：ok 为 false 表示本次不写 audit 字段，沿用数据库里的原审核状态。
//
// auditSwitch - 该内容模块是否开启审核
// draft       - 本次是否存草稿
// prevDraft   - 原文是否为草稿（pages 没有草稿概念，传 false）
// prevAudit   - 原文审核状态
func auditForUpdate(auditSwitch bool, draft bool, prevDraft bool, prevAudit int) (int, bool) {

	// 草稿：不对外展示，跳过审核
	if draft {
		return AuditPassed, true
	}

	// 关闭审核：一律通过
	if !auditSwitch {
		return AuditPassed, true
	}

	// 首次发布（原文是草稿）：进入待审核
	if prevDraft {
		return AuditPending, true
	}

	// 已有内容二次编辑：原本就是待审核的显式写回，其余（已通过 / 未通过）沿用原状态
	return AuditPending, prevAudit == AuditPending
}
