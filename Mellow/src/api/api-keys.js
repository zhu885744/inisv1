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

// 含审计字段：status（0停用 1启用）/ expire_time（unix 秒，0=永久）/ last_used_at / use_count
export const API_KEY_FIELD =
  'id,value,remark,status,expire_time,last_used_at,use_count,create_time,update_time,delete_time'

// 密钥列表（传 onlyTrashed=true 看回收站）
export const listApiKeys = (params = {}) =>
  call('api-keys', 'all', {
    method: 'GET',
    params: { page: 1, limit: 20, order: 'id desc', field: API_KEY_FIELD, ...params }
  })

// 密钥数量（count 支持 onlyTrashed）
export const countApiKeys = (params = {}) =>
  call('api-keys', 'count', { method: 'GET', params })

/**
 * 保存密钥：无 id 为新增，有 id 为更新（后端 save 自动分流）
 *
 * 允许字段：value / remark / status / expire_time；value 会被后端统一转大写。
 * ⚠️ 更新时若 **不传 value 或传空**，后端会**重新生成密钥**（等于重置，原密钥立即失效）——
 * 所以只想改状态 / 有效期 / 备注时，必须把当前 value 一起提交（见 toggleStatus）。
 */
export const saveApiKey = (data) =>
  call('api-keys', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeApiKeys = (ids) =>
  call('api-keys', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteApiKeys = (ids) =>
  call('api-keys', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreApiKeys = (ids) =>
  call('api-keys', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearApiKeyRecycle = () =>
  call('api-keys', 'clear', { method: 'DELETE' })
