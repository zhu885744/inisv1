import { call } from './request'

// ids 归一化：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔字符串
const toIdsString = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)
    .join(',')

// PUT/POST 走 JSON body，直接传数组即可（utils.Unity.Ids 会解析）
const toIdsArray = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)

// 列表字段：带上 result 才能拿到触发者（from_user）
export const NOTIFICATION_FIELD =
  'id,uid,from_uid,type,title,content,bind_id,bind_type,is_read,create_time,update_time,delete_time,result'

/**
 * 批量操作单次上限（与后端 controller.notificationBatchLimit 保持一致）
 * - read-batch：超过该数量后端返回 400
 * - remove-all：单次最多处理该数量，返回 { cleared, remaining }，需按 remaining 继续清理
 */
export const NOTIFICATION_BATCH_LIMIT = 200

/**
 * 消息列表（GET notification/all）
 *
 * 作用域由后端决定：普通用户只看自己的；root 可同时看到广播（uid=0）。
 * 显式传 uid 会精确过滤（uid=0 即只看广播；传自己的 id 即只看个人消息）。
 * 支持 where / like（「字段名|值」）/ order / onlyTrashed。
 */
export const listNotifications = (params = {}) =>
  call('notification', 'all', {
    method: 'GET',
    params: { page: 1, limit: 15, order: 'create_time desc', field: NOTIFICATION_FIELD, ...params }
  })

/**
 * 全体可见的短消息列表（群发短消息，uid = 0，只存一条记录）
 * 仅 root 可见（非 root 传 uid=0 会被后端与自身 uid 求交集，结果为空）
 */
export const listBroadcasts = (params = {}) =>
  listNotifications({ ...params, uid: 0 })

/**
 * 管理端视角的通知列表（scope=admin，仅 root 生效）
 *
 * 不加 uid 限制，可列出全站通知：广播（uid=0）、发给指定用户的消息，
 * 以及评论 / 点赞 / 收藏 / 关注等自动通知。
 * 可按 where 过滤（如 { type: 'system' } / { is_read: 0 }）、like、onlyTrashed。
 */
export const listManagedNotifications = (params = {}) =>
  listNotifications({ ...params, scope: 'admin' })

// 消息数量（后端固定按当前登录用户过滤，因此不能用于统计广播）
export const countNotifications = (params = {}) =>
  call('notification', 'count', { method: 'GET', params })

// 当前用户未读消息数
export const getUnreadCount = () =>
  call('notification', 'unread-count', { method: 'GET' })

/**
 * 发送系统消息（POST notification/send-system，仅 root）
 * target_type: all（广播，仅创建一条 uid=0 记录）/ partial / single
 * user_ids: 指定用户 ID 数组（target_type 非 all 时必填）
 * send_email: 是否同时发送邮件（仅指定用户生效）
 * as_system: 是否以「系统」身份发送（标题加【短消息】前缀）
 */
export const sendSystemMessage = (data) =>
  call('notification', 'send-system', { method: 'POST', data })

// 更新消息（PUT notification/update，需 id；允许字段见后端 allowFields）
export const updateNotification = (data) =>
  call('notification', 'update', { method: 'PUT', data })

// 标记单条已读
export const readNotification = (id) =>
  call('notification', 'read', { method: 'PUT', data: { id: Number(id) } })

// 全部标记已读（当前登录用户）
export const readAllNotifications = () =>
  call('notification', 'read-all', { method: 'PUT' })

// 批量标记已读（广播与个人通知分别处理）
export const readBatchNotifications = (ids) =>
  call('notification', 'read-batch', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 软删除（root 删除 uid=0 的短消息即「撤回」，全体不可见）
export const removeNotifications = (ids, params = {}) =>
  call('notification', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids), ...params } })

// 彻底删除（不可恢复）
export const forceDeleteNotifications = (ids, params = {}) =>
  call('notification', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids), ...params } })

// 从回收站恢复
export const restoreNotifications = (ids, params = {}) =>
  call('notification', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids), ...params } })

// 清空回收站（root 同时清理广播）
export const clearNotificationRecycle = () =>
  call('notification', 'clear', { method: 'DELETE' })

/**
 * 清空当前用户的消息（可按 type 过滤；is_read=1 时只清空已读）
 *
 * 返回 { ids, cleared, remaining }：单次最多清理 NOTIFICATION_BATCH_LIMIT 条，
 * remaining > 0 时表示还有剩余，需要再次调用（封装见 clearAllNotifications）。
 */
export const removeAllNotifications = (params = {}) =>
  call('notification', 'remove-all', { method: 'DELETE', params })

/**
 * 反复调用 remove-all 直到清空（分页上限保护，最多 rounds 轮）
 * @returns {Promise<{cleared:number, remaining:number, rounds:number}>}
 */
export async function clearAllNotifications(params = {}, rounds = 20) {
  let cleared = 0
  let remaining = 0
  let done = 0

  for (let i = 0; i < rounds; i++) {
    const res = await removeAllNotifications(params)
    // 204：没有可清理的数据
    if (res?.code === 204) {
      remaining = 0
      break
    }

    done++
    cleared += Number(res?.data?.cleared || 0)
    remaining = Number(res?.data?.remaining || 0)

    if (remaining <= 0) break
  }

  return { cleared, remaining, rounds: done }
}
