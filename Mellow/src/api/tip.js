import { call } from './request'

// ===== 积分打赏 =====
//
// 打赏对象用 bind_type + bind_id 描述：
//   article / moments / comment / page —— 内容作者收款（后端按内容反查作者，前端传的 to_uid 只做校验）
//   user —— 用户主页打赏，bind_id 即接收者 uid

/**
 * 打赏概览（GET tip/summary）
 * 返回 { count, integral, balance, min, max, message_max }
 * - count / integral：该内容累计被打赏次数与积分
 * - balance：当前登录用户余额（未登录为 0）
 */
export const getTipSummary = (bindType, bindId) =>
  call('tip', 'summary', { method: 'GET', params: { bind_type: bindType, bind_id: bindId } })

/**
 * 用户收到的打赏统计（GET tip/stats）
 * 返回 { uid, count, integral, tippers, recent: [...] }
 */
export const getTipStats = (uid) =>
  call('tip', 'stats', { method: 'GET', params: { uid: Number(uid) || 0 } })

/**
 * 打赏记录（GET tip/list）
 * - { bind_type, bind_id }：某条内容收到的打赏（公开）
 * - { box: 'received' | 'sent' }：我收到 / 我发出的打赏（需登录）
 */
export const listTips = (params = {}) =>
  call('tip', 'list', { method: 'GET', params: { limit: 20, ...params } })

/**
 * 打赏（POST tip/send）
 * data: { bind_type, bind_id, to_uid, value, message }
 * 返回 { id, value, balance, to_balance, ... }
 */
export const sendTip = (data) => call('tip', 'send', { method: 'POST', data })
