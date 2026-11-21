import { call } from './request'

/**
 * 聊天审核（私信 + 公共群聊）
 *
 * 后端：app/api/controller/chat-moderation.go、app/model/chat_moderation.go
 *
 * 数据来源：消息命中敏感词库时由服务端自动留证（只存命中词前后各 40 字的片段，
 * 不含整条消息）。私信 / 群聊对 review / warn / log 是放行的，这条记录是后台
 * 判断「到底算不算违规」的唯一依据。
 *
 * 只提供查看 / 标记处理 / 删除：撤回在「群聊消息」页，禁言在「禁言名单」页，
 * 账号处罚在「用户管理」页，避免同一动作有多套口径。
 */

// ids 归一化：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔
const toIdsString = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)
    .join(',')

// PUT 走 JSON body，直接传数组即可（后端 utils.Unity.Ids 会解析）
const toIdsArray = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)

/**
 * 留证列表
 * @param {object} [params] page / limit / scene（dm|chat）/ action / handled（0|1）/ uid / word
 */
export const listChatModeration = (params = {}) =>
  call('chat-moderation', 'all', {
    method: 'GET',
    params: { page: 1, limit: 20, ...params }
  })

/** 数量（参数与列表一致，用于顶部统计 / 待处理角标） */
export const countChatModeration = (params = {}) =>
  call('chat-moderation', 'count', { method: 'GET', params })

/**
 * 标记已处理（handled=1）/ 恢复待处理（handled=0）
 * @param {number|number[]} ids
 * @param {number} [handled]
 * @param {string} [remark] 处理备注（最多 255 字）
 */
export const handleChatModeration = (ids, handled = 1, remark = '') =>
  call('chat-moderation', 'handle', {
    method: 'PUT',
    data: { ids: toIdsArray(ids), handled: Number(handled) ? 1 : 0, remark }
  })

// 删除记录（证据过期或确认误伤）
export const removeChatModeration = (ids) =>
  call('chat-moderation', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })
