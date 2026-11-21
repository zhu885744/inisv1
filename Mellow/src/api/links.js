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

// 友链分组
export const listLinkGroups = (params = {}) =>
  call('links-group', 'all', { method: 'GET', params })

// 友链列表
export const listLinks = (params = {}) =>
  call('links', 'all', { method: 'GET', params })

// 申请友链
export const createLink = (data) =>
  call('links', 'create', { method: 'POST', data })

// 更新友链
export const updateLink = (data) =>
  call('links', 'update', { method: 'PUT', data })

// 删除友链
export const removeLink = (ids) =>
  call('links', 'remove', { method: 'DELETE', params: { ids } })

// ===== 后台（管理员）=====

/**
 * 友链列表（管理员）：默认含待审核与已通过的全部数据；传 onlyTrashed=true 看回收站
 * 说明：非管理员只能查到 audit=1 的友链，后台列表依赖管理员身份，
 * field 需显式带上 result 才能拿到分组信息。
 */
export const listLinksAdmin = (params = {}) =>
  call('links', 'all', { method: 'GET', params })

/**
 * 友链数量统计
 * 注意：后端 links/count 未处理 onlyTrashed，回收站数量请用 listLinksAdmin 的 count。
 */
export const countLinks = (params = {}) =>
  call('links', 'count', { method: 'GET', params })

// 软删除（移入回收站）
export const removeLinks = (ids) =>
  call('links', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteLinks = (ids) =>
  call('links', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreLinks = (ids) =>
  call('links', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearLinkRecycle = () =>
  call('links', 'clear', { method: 'DELETE' })

// ===== 友链分组（后台）=====

// 分组列表（管理员：传 onlyTrashed=true 看回收站）
export const listLinkGroupsAdmin = (params = {}) =>
  call('links-group', 'all', { method: 'GET', params })

// 分组数量（links-group/count 同样不支持 onlyTrashed）
export const countLinkGroups = (params = {}) =>
  call('links-group', 'count', { method: 'GET', params })

// 保存分组：无 id 为新增，有 id 为更新（后端 save 自动分流）
export const saveLinkGroup = (data) =>
  call('links-group', 'save', { method: 'POST', data })

// 软删除分组
export const removeLinkGroups = (ids) =>
  call('links-group', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除分组（不可恢复）
export const forceDeleteLinkGroups = (ids) =>
  call('links-group', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复分组
export const restoreLinkGroups = (ids) =>
  call('links-group', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空分组回收站
export const clearLinkGroupRecycle = () =>
  call('links-group', 'clear', { method: 'DELETE' })