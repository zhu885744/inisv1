import { call } from '@/api/request'

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

export const AUTH_PAGE_FIELD = 'id,name,path,icon,svg,size,hash,remark,create_time,update_time,delete_time'

// ===== 后台管理（/admin/security/pages）=====

// 页面列表（传 onlyTrashed=true 看回收站）
export const listAuthPages = (params = {}) =>
  call('auth-pages', 'all', {
    method: 'GET',
    params: { page: 1, limit: 50, order: 'id asc', field: AUTH_PAGE_FIELD, ...params }
  })

// 页面数量（count 支持 onlyTrashed）
export const countAuthPages = (params = {}) =>
  call('auth-pages', 'count', { method: 'GET', params })

/**
 * 保存页面：无 id 为新增，有 id 为更新
 * 允许字段 name/path/icon/svg/size/remark（name 与 path 为必填，见 validator）；
 * hash 由后端生成且不在允许字段里，前端不要提交。
 */
export const saveAuthPage = (data) =>
  call('auth-pages', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeAuthPages = (ids) =>
  call('auth-pages', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteAuthPages = (ids) =>
  call('auth-pages', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreAuthPages = (ids) =>
  call('auth-pages', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearAuthPageRecycle = () =>
  call('auth-pages', 'clear', { method: 'DELETE' })

/**
 * 获取全部后台权限页面（扁平列表）
 * 接口：GET /api/auth-pages/all
 * 返回结构：res.data = { data: [...], count, page }
 * 单页 limit 给大一点，确保一次拉全（INIS 默认单页上限通常 10/20，这里用 1000）
 */
export const getAuthPagesFlat = () =>
  call('auth-pages', 'all', {
    method: 'GET',
    params: { page: 1, limit: 1000, order: 'create_time asc' }
  })
