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

// ===== 公告（placard）=====

// 公告列表（首页公告轮播的数据源）
export const listPlacards = (params = {}) =>
  call('placard', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'create_time desc',
      field: 'id,title,content,type,url,target,create_time,update_time,delete_time',
      ...params
    }
  })

/**
 * 公告数量统计
 * 注意：后端 placard/count 未处理 onlyTrashed，回收站数量请用 listPlacards 的 count 字段
 */
export const countPlacards = (params = {}) =>
  call('placard', 'count', { method: 'GET', params })

// 保存公告：无 id 为新增，有 id 为更新（后端 save 自动分流）
export const savePlacard = (data) =>
  call('placard', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removePlacards = (ids) =>
  call('placard', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeletePlacards = (ids) =>
  call('placard', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restorePlacards = (ids) =>
  call('placard', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearPlacardRecycle = () =>
  call('placard', 'clear', { method: 'DELETE' })
