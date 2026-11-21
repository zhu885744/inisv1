import { call } from './request'
import { uploadAttachments } from './attachment'

// 获取动态列表（默认审核通过 + 已发布）
export const listMoments = (params = {}) =>
  call('moments', 'all', {
    method: 'GET',
    params: {
      where: { audit: 1, status: 1 },
      order: 'top desc, create_time desc',
      ...params
    }
  })

// 获取单条动态
export const getMoment = (id) =>
  call('moments', 'one', { method: 'GET', params: { id } })

// 获取动态评论
export const getMomentComments = (bind_id, params = {}) =>
  call('moments', 'comment', {
    method: 'GET',
    params: { bind_id, ...params }
  })

// 创建动态（用 create）
export const createMoment = (data) =>
  call('moments', 'create', { method: 'POST', data })

// 上传动态图片（统一走 uploadAttachments，含 WebP 压缩处理）
export const uploadMomentImages = (formData) => uploadAttachments(formData)

// 更新动态（可改 content/images/location/status 等）
export const updateMoment = (data) =>
  call('moments', 'update', { method: 'PUT', data })

// 删除动态（软删除，移入回收站）
// 注意：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔字符串
export const removeMoment = (ids) =>
  call('moments', 'remove', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// ===== 后台（管理员）=====

// 后台动态列表：不附加 audit/status 默认条件（管理员可查看全部状态）
export const listMomentsAdmin = (params = {}) =>
  call('moments', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'top desc, create_time desc',
      field:
        'id,uid,content,images,location,top,views,audit,reason,status,scope,scope_integral,create_time,update_time,publish_time,delete_time',
      ...params
    }
  })

// 设置 / 取消置顶（管理员专用，支持批量；top: 1 置顶 / 0 取消）
export const setMomentTop = (ids, top = 1) =>
  call('moments', 'set_top', {
    method: 'PUT',
    data: { ids: Array.isArray(ids) ? ids : [ids], top: Number(top) === 1 ? 1 : 0 }
  })

// 从回收站恢复
export const restoreMoment = (ids) =>
  call('moments', 'restore', {
    method: 'PUT',
    data: { ids: Array.isArray(ids) ? ids : [ids] }
  })

// 彻底删除（不可恢复）
export const forceDeleteMoment = (ids) =>
  call('moments', 'delete', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 清空回收站（管理员清空全站，普通用户仅清空自己的）
export const clearMomentRecycle = () =>
  call('moments', 'clear', { method: 'DELETE' })