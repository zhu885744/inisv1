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

// ===== 等级（level）=====

// 等级列表（默认按等级值升序，便于按梯度查看）
export const listLevels = (params = {}) =>
  call('level', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 20,
      order: 'value asc',
      field: 'id,name,value,description,exp,remark,create_time,update_time,delete_time',
      ...params
    }
  })

// 等级数量统计（注意：后端 level/count 不支持 onlyTrashed，回收站数量请用列表的 count）
export const countLevels = (params = {}) =>
  call('level', 'count', { method: 'GET', params })

// 保存等级：无 id 为新增，有 id 为更新（后端 save 自动分流）
export const saveLevel = (data) =>
  call('level', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeLevels = (ids) =>
  call('level', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteLevels = (ids) =>
  call('level', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreLevels = (ids) =>
  call('level', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearLevelRecycle = () =>
  call('level', 'clear', { method: 'DELETE' })
