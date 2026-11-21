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

// ===== 轮播（banner）=====

// 轮播列表（不拉取 json/text 等大字段）
export const listBanners = (params = {}) =>
  call('banner', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'create_time desc',
      field: 'id,uid,title,content,url,image,target,start_time,end_time,remark,create_time,update_time,delete_time',
      ...params
    }
  })

// 轮播数量统计（后端 banner/count 支持 onlyTrashed）
export const countBanners = (params = {}) =>
  call('banner', 'count', { method: 'GET', params })

// 保存轮播：无 id 为新增，有 id 为更新（后端 save 自动分流）
export const saveBanner = (data) =>
  call('banner', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeBanners = (ids) =>
  call('banner', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteBanners = (ids) =>
  call('banner', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreBanners = (ids) =>
  call('banner', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearBannerRecycle = () =>
  call('banner', 'clear', { method: 'DELETE' })
