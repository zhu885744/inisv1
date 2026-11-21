/**
 * 权限分组（auth-group）领域公共逻辑
 *
 * 后端用特殊字符串格式存储成员与授权内容（见 model.AuthGroup）：
 * - uids   "|1|2|" 竖线包裹的用户 ID
 * - rules  规则 hash 逗号分隔，字面量 "all" 表示全部规则
 * - pages  页面 hash 逗号分隔，字面量 "all" 表示全部页面
 * 解析与序列化统一在这里维护，避免列表与编辑弹窗各写一份导致格式不一致。
 */

/** 解析 "|1|2|" -> [1, 2] */
export function parsePipeIds(text) {
  return String(text ?? '')
    .split('|')
    .map((s) => Number(String(s).trim()))
    .filter((n) => Number.isInteger(n) && n > 0)
}

/** [1, 2] -> "|1|2|"（去重，空数组返回空串） */
export function toPipeIds(ids) {
  const list = (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((n) => Number.isInteger(n) && n > 0)
  if (!list.length) return ''
  return `|${[...new Set(list)].join('|')}|`
}

/** 输入框文本（逗号 / 空格 / 竖线分隔）-> ID 数组 */
export function parseIdInput(text) {
  return String(text ?? '')
    .split(/[,，\s|]+/)
    .map((s) => Number(s.trim()))
    .filter((n) => Number.isInteger(n) && n > 0)
}

/**
 * 解析 hash 列表
 * @returns {{ all: boolean, list: string[] }} all 为真表示「全部」
 */
export function parseHashList(text) {
  const raw = String(text ?? '').trim()
  if (!raw) return { all: false, list: [] }
  const list = raw
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
  return { all: list.includes('all'), list: list.filter((v) => v !== 'all') }
}

/**
 * 受保护分组：id=1（超级管理员）与 default=1（系统内置）
 * 后端 remove / delete / clear 都会拒绝这两类分组
 */
export function isProtectedGroup(item) {
  return Number(item?.id) === 1 || Number(item?.default) === 1
}

/** 受保护原因（用于 tooltip） */
export function protectedGroupReason(item) {
  if (Number(item?.id) === 1) return '超级管理员分组受保护'
  if (Number(item?.default) === 1) return '系统内置分组不可删除'
  return ''
}

/** 组内成员（后端在 result.users 中解析好；未取 result 时返回空数组） */
export function groupMembers(item) {
  const list = item?.result?.users
  return Array.isArray(list) ? list : []
}
