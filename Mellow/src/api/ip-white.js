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

export const IP_WHITE_FIELD = 'id,ip,remark,create_time,update_time,delete_time'

// 白名单列表（传 onlyTrashed=true 看回收站）
export const listIpWhite = (params = {}) =>
  call('ip-white', 'all', {
    method: 'GET',
    params: { page: 1, limit: 20, order: 'id desc', field: IP_WHITE_FIELD, ...params }
  })

// 数量（count 支持 onlyTrashed）
export const countIpWhite = (params = {}) =>
  call('ip-white', 'count', { method: 'GET', params })

/**
 * 保存白名单：无 id 为新增，有 id 为更新
 * 允许字段只有 ip / remark；BeforeCreate 会校验 IP 不重复（重复返回错误）
 */
export const saveIpWhite = (data) =>
  call('ip-white', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeIpWhite = (ids) =>
  call('ip-white', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteIpWhite = (ids) =>
  call('ip-white', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreIpWhite = (ids) =>
  call('ip-white', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearIpWhiteRecycle = () =>
  call('ip-white', 'clear', { method: 'DELETE' })
