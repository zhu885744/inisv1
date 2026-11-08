package model

import "sort"

/**
 * 百度内容审核的维度（type）映射
 *
 * 文本与图像的 type 编号是两套体系，但有重叠（30~39 是文本，0~47 与 1230~1239 是图像/图文）。
 * 这里合并成一张表用于后台展示与筛选：
 *   - 后台「敏感词命中」按维度统计时，直接把编号翻译成中文；
 *   - 筛选下拉用 AuditTypeOptions()，避免前端再维护一份硬编码。
 *
 * 编号来源：百度内容审核 v2 的响应说明（文本 type 11~39、图像 type 0~47、图文 1230~1239）。
 */

// auditTypeLabels - 维度编号 → 中文名称
var auditTypeLabels = map[int]string{
	// 0 同时是本地词库（我们自填）与图像的「百度违禁图库」，
	// 用来源字段（match = cloud / 本地匹配方式）可区分，这里统一按「本地词库/其他」展示
	0: "本地词库/其他",

	// 文本
	11: "百度违禁词库",
	12: "文本反作弊",
	13: "自定义黑名单",
	14: "自定义白名单",
	30: "涉政",
	31: "暴恐",
	32: "违禁",
	33: "涉黄",
	34: "辱骂",
	35: "价值观",
	36: "广告",
	37: "广告法",
	38: "隐私信息",
	39: "低质灌水",

	// 图像（新版）
	42: "涉政（图）",
	43: "涉黄（图）",
	44: "暴恐（图）",
	45: "违禁（图）",
	47: "广告法（图）",

	// 图像（旧版与其它场景）
	1:  "涉黄（旧版）",
	2:  "暴恐（旧版）",
	3:  "恶心图像",
	4:  "广告（图）",
	5:  "涉政人物",
	6:  "图像质量",
	7:  "自定义图像黑库",
	8:  "自定义图像白库",
	10: "用户头像",
	15: "EasyDL 模型",
	16: "敏感旗帜标志",
	21: "违禁（旧版）",
	24: "直播场景",
	40: "官方增强场景库",
	41: "自定义场景库",

	// 图文
	1230: "图文-涉政",
	1231: "图文-暴恐",
	1232: "图文-违禁",
	1233: "图文-涉黄",
	1234: "图文-辱骂",
	1235: "图文-价值观",
	1236: "图文-广告",
	1237: "图文-广告法",
	1238: "图文-隐私信息",
	1239: "图文-低质灌水",
}

// AuditTypeLabel - 维度编号转中文（未知编号原样显示编号）
func AuditTypeLabel(auditType int) string {
	if label, ok := auditTypeLabels[auditType]; ok {
		return label
	}
	if auditType <= 0 {
		return "本地词库/其他"
	}
	return "其他维度"
}

// AuditTypeOptions - 后台筛选项（按编号升序，0 表示全部）
//
// 只列常用维度：把 30~45 与图文 1230~1239 全列出来会太长，
// 低频旧版编号（1/2/5/16/21/38）仍能通过「全部」结果里的标签看到。
func AuditTypeOptions() []map[string]any {

	common := []int{0, 11, 12, 13, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 42, 43, 44, 45, 47}

	result := make([]map[string]any, 0, len(common))
	for _, item := range common {
		result = append(result, map[string]any{
			"type":  item,
			"label": AuditTypeLabel(item),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i]["type"].(int) < result[j]["type"].(int)
	})

	return result
}
