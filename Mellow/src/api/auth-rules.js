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

/**
 * 规则列表（GET auth-rules/all）
 * 支持 where / like（「字段名|值」）/ order / onlyTrashed；count 接口同样支持 onlyTrashed。
 */
export const listAuthRules = (params = {}) =>
  call('auth-rules', 'all', { method: 'GET', params })

// 规则数量
export const countAuthRules = (params = {}) =>
  call('auth-rules', 'count', { method: 'GET', params })

/**
 * 保存规则：无 id 为新增，有 id 为更新（后端 save 自动分流）
 *
 * 注意：hash 由后端按 [METHOD]route 自动计算，前端不要传；
 * 修改 route / method 会让 hash 变化，已授权分组需重新配置。
 */
export const saveAuthRule = (data) =>
  call('auth-rules', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeAuthRules = (ids) =>
  call('auth-rules', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteAuthRules = (ids) =>
  call('auth-rules', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreAuthRules = (ids) =>
  call('auth-rules', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearAuthRuleRecycle = () =>
  call('auth-rules', 'clear', { method: 'DELETE' })
