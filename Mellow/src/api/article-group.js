import { call } from './request'

// ===== 文章分类（article-group）=====

// 分类列表（全量，用于树形展示与父级选择）
export const listArticleGroups = (params = {}) =>
  call('article-group', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 100,
      order: 'id asc',
      field: 'id,pid,key,name,description,avatar,create_time',
      ...params
    }
  })

// 树形结构（含 children，后端 pid 递归查询）
export const treeArticleGroups = (params = {}) =>
  call('article-group', 'tree', {
    method: 'GET',
    params: { order: 'id asc', ...params }
  })

// 保存分类：无 id 为新增，有 id 为更新（后端 save 自动分流）
export const saveArticleGroup = (data) =>
  call('article-group', 'save', { method: 'POST', data })

// 软删除（移入回收站）
// 注意：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔字符串
export const removeArticleGroup = (ids) =>
  call('article-group', 'remove', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 从回收站恢复
export const restoreArticleGroup = (ids) =>
  call('article-group', 'restore', {
    method: 'PUT',
    data: { ids: Array.isArray(ids) ? ids : [ids] }
  })

// 彻底删除（不可恢复）
export const forceDeleteArticleGroup = (ids) =>
  call('article-group', 'delete', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 清空回收站
export const clearArticleGroupRecycle = () =>
  call('article-group', 'clear', { method: 'DELETE' })
