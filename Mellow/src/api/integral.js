import { call } from './request'

// ids 归一化：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔字符串
const toIdsString = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)
    .join(',')

/** 积分规则在配置表中的键（model.IntegralCacheKey） */
export const INTEGRAL_RULES_KEY = 'SYSTEM_INTEGRAL_RULES'

// ===== 积分规则 =====

/**
 * 积分任务规则（GET integral/rules）
 * 返回 [{ type, name, value, daily_limit, icon }]
 */
export const getIntegralRules = () =>
  call('integral', 'rules', { method: 'GET' })

/**
 * 积分排行榜（GET integral/rank）
 * by: earned（累计获得，默认）/ balance（当前余额）；limit: 1~100
 * 返回 { list: [...], by, my_rank, my_value }
 */
export const getIntegralRank = (params = {}) =>
  call('integral', 'rank', { method: 'GET', params: { by: 'earned', limit: 20, ...params } })

// 原始积分规则配置（config/one，json 已由后端解码为对象；未配置时返回 null）
export const getIntegralConfig = () =>
  call('config', 'one', { method: 'GET', params: { key: INTEGRAL_RULES_KEY } })

// 保存积分规则（config/save，key=SYSTEM_INTEGRAL_RULES）
export const saveIntegralConfig = (json) =>
  call('config', 'save', { method: 'POST', data: { key: INTEGRAL_RULES_KEY, json } })

// ===== 管理员调整积分 =====

/**
 * 管理员发放 / 扣除积分（POST integral/give）
 * 支持 uid（单个）或 uids（数组批量），value 为正数发放、负数扣除（不能为 0）
 */
export const giveIntegral = (data) =>
  call('integral', 'give', { method: 'POST', data })

// ===== 积分卡密 =====

/**
 * 卡密列表（管理员）
 * 支持 status（0未使用 1已使用）、batch、uid、value、keyword（卡密模糊）、
 * expired（1已过期 / 0未过期含永久）
 */
export const listIntegralCards = (params = {}) =>
  call('integral', 'card-all', {
    method: 'GET',
    params: { page: 1, limit: 20, order: 'id desc', ...params }
  })

// 卡密统计（管理员）：总数 / 未使用 / 已发放（待兑换）/ 已使用 / 已过期 / 累计发放 / 已兑换
export const getIntegralCardStats = () =>
  call('integral', 'card-stats', { method: 'GET' })

// 我的待兑换卡密（登录用户）：签到等活动发到账号里的卡密，在这里找回并兑换
export const getMyIntegralCards = (params = {}) =>
  call('integral', 'card-mine', { method: 'GET', params: { limit: 20, ...params } })

// 生成卡密（管理员）：value 面额、count 数量、length 长度、expire 日期或 expire_time 时间戳、remark 备注
export const generateIntegralCards = (data) =>
  call('integral', 'card-generate', { method: 'POST', data })

/**
 * 导入自定义卡密（管理员）：codes 卡密内容（一行一个的文本，或已拆好的数组）、value 面额、
 * expire 日期或 expire_time 时间戳、remark 备注
 *
 * 与「生成卡密」的区别：卡密内容由管理员自己填写，不再随机生成。
 * 返回 data.cards 为实际写入的卡密明文，data.report 说明被跳过的卡密与原因
 * （total / success / skipped / skipped_count / truncated）。
 */
export const importIntegralCards = (data) =>
  call('integral', 'card-import', { method: 'POST', data })

// 导出未使用卡密（管理员，单次上限由后端控制，超出时返回 truncated=true）
/**
 * 手动设置卡密的兑换人（管理员）
 *
 * @param ids 卡密 id 数组（单张也传数组）
 * @param uid 目标用户 id；传 0 表示解除绑定（已发放的卡密退回「未使用」）
 * 语义：未使用的卡密 → 已发放并绑定该用户；已发放的卡密 → 改绑；已兑换的卡密会被后端跳过
 */
export const bindIntegralCards = (ids, uid) =>
  call('integral', 'card-bind', {
    method: 'POST',
    data: {
      ids: (Array.isArray(ids) ? ids : [ids]).map((i) => Number(i)).filter((i) => i > 0),
      uid: Number(uid) || 0
    }
  })

export const exportIntegralCards = (params = {}) =>
  call('integral', 'card-export', { method: 'GET', params })

// 软删除卡密
export const removeIntegralCards = (ids) =>
  call('integral', 'card-remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除卡密（不可恢复）
export const forceDeleteIntegralCards = (ids) =>
  call('integral', 'card-delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })
