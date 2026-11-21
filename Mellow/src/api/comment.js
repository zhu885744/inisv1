import { call } from './request'
import { uploadAttachments } from './attachment'

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

// 评论树结构
export const getCommentTree = (bind_id, bind_type = 'article', params = {}) =>
  call('comment', 'flat', {
    method: 'GET',
    params: { bind_id, bind_type, ...params }
  })

// 单条评论（通知跳转用：先拿 bind_type / bind_id 才能定位评论所属内容）
export const getComment = (id, field = 'id,bind_type,bind_id') =>
  call('comment', 'one', { method: 'GET', params: { id, field } })

// 创建评论（用 create 而非 save）
export const createComment = (data) =>
  call('comment', 'create', { method: 'POST', data })

// 删除评论（软删除，移入回收站）
export const removeComment = (ids) =>
  call('comment', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 上传评论图片（统一走 uploadAttachments，含 WebP 压缩处理）
// 返回 res.data.results：数组项含 status 与 full_url
export const uploadCommentImages = (formData) => uploadAttachments(formData)

// ===== 后台（管理员）=====

// 评论列表（管理员视角：IP 不脱敏；field/where/order/like 由 utils/comment.js 统一拼装）
export const listComments = (params = {}) =>
  call('comment', 'all', {
    method: 'GET',
    params: { page: 1, limit: 15, order: 'create_time desc', ...params }
  })

// 评论数量统计（后端 comment/count 支持 onlyTrashed）
export const countComments = (params = {}) =>
  call('comment', 'count', { method: 'GET', params })

/**
 * 批量评论数（列表页专用）
 *
 * 一次拿到整页内容的评论数，替代「每条动态各调一次 comment/count」。
 * 口径与 count 一致：只统计审核通过的评论（含二级及更深的回复）。
 *
 * 返回：{ data: { counts: { [bind_id]: n } } }
 */
export const countCommentsBatch = (bind_type, bind_ids) => {
  const ids = Array.isArray(bind_ids) ? bind_ids : [bind_ids]
  return call('comment', 'counts', {
    method: 'GET',
    params: { bind_type, bind_ids: ids.join(',') }
  })
}

// 更新评论：管理员可改任意评论的 content / images，普通用户仅限自己的
export const updateComment = (data) =>
  call('comment', 'update', { method: 'PUT', data })

// 软删除（移入回收站），批量
export const removeComments = (ids) =>
  call('comment', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteComments = (ids) =>
  call('comment', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreComments = (ids) =>
  call('comment', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearCommentRecycle = () =>
  call('comment', 'clear', { method: 'DELETE' })
