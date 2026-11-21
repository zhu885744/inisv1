import { call } from './request'
import { uploadAttachments } from './attachment'

// 获取文章列表
export const listArticles = (params) =>
  call('article', 'all', { method: 'GET', params })

// 获取单篇文章
export const getArticle = (id) =>
  call('article', 'one', { method: 'GET', params: { id } })

// 文章分组（扁平列表，用 pid 表达层级）
export const getArticleGroups = (params = {}) =>
  call('article-group', 'all', {
    method: 'GET',
    params: { page: 1, limit: 100, field: 'id,pid,name,description,article_count', ...params }
  })

// 统计某分组下的文章数量（后端 article-group 的 article_count 字段未维护，需自行统计）
// group 字段在文章里以 |id| 形式存储，用 like 模糊匹配
export const countArticlesByGroup = (groupId) =>
  call('article', 'count', {
    method: 'GET',
    params: { like: `group|%7C${groupId}%7C`, where: JSON.stringify({ audit: 1, status: 1 }) }
  })

// 统计某作者发布的文章数量
export const countArticlesByAuthor = (uid) =>
  call('article', 'count', {
    method: 'GET',
    params: { where: JSON.stringify({ uid, audit: 1, status: 1 }) }
  })

// 统计某个分类下的文章数（后台用：管理员查询不限审核状态）
export const countArticlesInGroup = (groupId) =>
  call('article', 'count', {
    method: 'GET',
    params: { where: JSON.stringify({ group: { $like: `%|${groupId}|%` } }) }
  })

// 获取某作者发布的文章列表（分页）
export const getAuthorArticles = (uid, params = {}) =>
  call('article', 'all', {
    method: 'GET',
    params: {
      where: JSON.stringify({ uid, audit: 1, status: 1 }),
      field: 'id,title,abstract,views,create_time,group,login',
      order: 'create_time desc',
      page: 1,
      limit: 10,
      ...params
    }
  })

// 上传文章图片（封面 / 正文配图），统一走 uploadAttachments（含 WebP 压缩处理）
export const uploadArticleImage = (formData) => uploadAttachments(formData)

// 我的文章列表（分页；status 传 0 草稿 / 1 已发布 / 不传则全部）
export const getMyArticles = (uid, params = {}) =>
  call('article', 'all', {
    method: 'GET',
    params: {
      where: JSON.stringify({ uid }),
      order: 'create_time desc',
      page: 1,
      limit: 10,
      field: 'id,uid,title,abstract,views,status,audit,reason,covers,group,tags,scope,scope_integral,create_time,update_time,publish_time',
      ...params
    }
  })

// 统计我的文章数量（count 接口不附加 audit 条件，可用于统计待审核数量）
export const countMyArticles = (uid, extra = {}) =>
  call('article', 'count', {
    method: 'GET',
    params: { where: JSON.stringify({ uid, ...extra }) }
  })

// 获取文章详情（编辑用，含 content）
export const getArticleForEdit = (id) =>
  call('article', 'one', {
    method: 'GET',
    params: {
      id,
      field: 'id,uid,title,abstract,content,covers,group,tags,status,audit,reason,editor,json,scope,scope_integral,create_time,update_time,publish_time'
    }
  })

// 创建文章
export const createArticle = (data) =>
  call('article', 'save', { method: 'POST', data })

// 更新文章
export const updateArticle = (data) =>
  call('article', 'update', { method: 'PUT', data })

// 删除文章（软删除，移入回收站）
// 注意：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，故统一转逗号分隔字符串
export const removeArticle = (ids) =>
  call('article', 'remove', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 随机文章
export const randArticles = (limit = 4) =>
  call('article', 'rand', { method: 'GET', params: { limit, field: 'id,title,abstract,views,create_time,group' } })

// ===== 后台（管理员）=====

// 后台文章列表：不限定作者，可查询全部状态；传 onlyTrashed 查回收站
export const getArticles = (params = {}) =>
  call('article', 'all', {
    method: 'GET',
    params: { page: 1, limit: 15, order: 'create_time desc', ...params }
  })

/**
 * 后台文章数量统计
 * 支持 where（含 $like）与 onlyTrashed——后端 article/count 会处理 withTrashOptions，
 * 因此回收站数量可以直接用该接口统计。
 */
export const countArticles = (params = {}) =>
  call('article', 'count', { method: 'GET', params })

// 恢复回收站文章（管理员可恢复任意文章）
export const restoreArticle = (ids) =>
  call('article', 'restore', {
    method: 'PUT',
    data: { ids: Array.isArray(ids) ? ids : [ids] }
  })

// 彻底删除文章（不可恢复）
// 注意：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，故用逗号分隔字符串
export const forceDeleteArticle = (ids) =>
  call('article', 'delete', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 清空回收站（管理员清空全站，普通作者仅清空自己的）
export const clearArticleRecycle = () =>
  call('article', 'clear', { method: 'DELETE' })