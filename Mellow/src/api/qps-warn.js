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

export const QPS_WARN_FIELD = 'id,ip,agent,path,method,create_time,update_time,delete_time'

/**
 * QPS 告警列表（传 onlyTrashed=true 看回收站）
 *
 * 说明：告警由系统按请求频率自动写入，允许字段只有 ip，
 * 因此后台以查看与清理为主，不提供新建。
 */
export const listQpsWarn = (params = {}) =>
  call('qps-warn', 'all', {
    method: 'GET',
    params: { page: 1, limit: 20, order: 'create_time desc', field: QPS_WARN_FIELD, ...params }
  })

// 数量（count 支持 onlyTrashed）
export const countQpsWarn = (params = {}) =>
  call('qps-warn', 'count', { method: 'GET', params })

// 软删除（移入回收站）
export const removeQpsWarn = (ids) =>
  call('qps-warn', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteQpsWarn = (ids) =>
  call('qps-warn', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreQpsWarn = (ids) =>
  call('qps-warn', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearQpsWarnRecycle = () =>
  call('qps-warn', 'clear', { method: 'DELETE' })
