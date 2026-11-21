import { call } from './request'

// ===== 独立页面（pages）=====

// 页面列表（后台：不限作者，可查回收站）
export const listPages = (params = {}) =>
  call('pages', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'create_time desc',
      field: 'id,uid,key,title,remark,tags,audit,reason,views,create_time,update_time,publish_time,delete_time',
      ...params
    }
  })

// 页面详情（编辑用，含 content）
export const getPageForEdit = (id) =>
  call('pages', 'one', {
    method: 'GET',
    params: {
      id,
      field: 'id,uid,key,title,content,remark,tags,editor,audit,reason,json,scope,scope_integral,create_time,update_time,publish_time'
    }
  })

// 前台页面详情（按 key 或 id 查询，仅返回已审核内容）
export const getPage = (params = {}) =>
  call('pages', 'one', { method: 'GET', params })

// 创建页面（POST pages/save）
export const createPage = (data) =>
  call('pages', 'save', { method: 'POST', data })

// 更新页面（PUT pages/update）
export const updatePage = (data) =>
  call('pages', 'update', { method: 'PUT', data })

// 软删除（移入回收站）
// 注意：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔字符串
export const removePage = (ids) =>
  call('pages', 'remove', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 从回收站恢复
export const restorePage = (ids) =>
  call('pages', 'restore', {
    method: 'PUT',
    data: { ids: Array.isArray(ids) ? ids : [ids] }
  })

// 彻底删除（不可恢复）
export const forceDeletePage = (ids) =>
  call('pages', 'delete', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 清空回收站
export const clearPageRecycle = () =>
  call('pages', 'clear', { method: 'DELETE' })
