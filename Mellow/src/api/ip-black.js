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

export const IP_BLACK_FIELD =
  'id,ip,level,duration,expire_time,is_permanent,violation_count,agent,cause,remark,create_time,update_time,delete_time'

// 黑名单列表（传 onlyTrashed=true 看回收站）
export const listIpBlack = (params = {}) =>
  call('ip-black', 'all', {
    method: 'GET',
    params: { page: 1, limit: 20, order: 'id desc', field: IP_BLACK_FIELD, ...params }
  })

// 数量（count 支持 onlyTrashed；非 root 看到的 IP 会被后端脱敏）
export const countIpBlack = (params = {}) =>
  call('ip-black', 'count', { method: 'GET', params })

/**
 * 保存黑名单：无 id 为新增，有 id 为更新
 *
 * 封禁参数由后端按优先级计算：
 * is_permanent=true 时忽略时长（duration/expire_time 置 0）；
 * 否则优先用显式 duration（小时），未传则按 level 取默认时长（1/24/168 小时），
 * 再由 duration 推出 expire_time。
 */
export const saveIpBlack = (data) =>
  call('ip-black', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeIpBlack = (ids) =>
  call('ip-black', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteIpBlack = (ids) =>
  call('ip-black', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreIpBlack = (ids) =>
  call('ip-black', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearIpBlackRecycle = () =>
  call('ip-black', 'clear', { method: 'DELETE' })
