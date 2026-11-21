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

// 全量话题（供文章编辑选择话题时使用）
export const listAllTags = (params = {}) =>
  call('tags', 'all', {
    method: 'GET',
    params: { page: 1, limit: 100, order: 'create_time desc', ...params }
  })

// 新建话题
export const createTag = (name) =>
  call('tags', 'create', { method: 'POST', data: { name } })

// ===== 后台（管理员）=====

/**
 * 后台话题列表
 * 后端 tags/all 会额外注入 article_count（该话题被已审核文章引用的数量），
 * 传 field 时必须把它带上，否则会被字段过滤掉。
 */
export const listTagsAdmin = (params = {}) =>
  call('tags', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'create_time desc',
      // moderator_uid / recommend 是话题扩展字段（主持人 / 精选），字段过滤时必须带上
      field:
        'id,name,description,avatar,moderator_uid,recommend,article_count,create_time,update_time,delete_time',
      ...params
    }
  })

// 保存话题：无 id 为新增，有 id 为更新（后端 save 自动分流）
export const saveTag = (data) =>
  call('tags', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeTags = (ids) =>
  call('tags', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteTags = (ids) =>
  call('tags', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreTags = (ids) =>
  call('tags', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearTagRecycle = () =>
  call('tags', 'clear', { method: 'DELETE' })

// 点赞
export const like = (target_type, target_id) =>
  call('user-likes', 'like', { method: 'POST', data: { target_type, target_id } })

export const unlike = (target_type, target_id) =>
  call('user-likes', 'unlike', { method: 'PUT', data: { target_type, target_id } })

export const isLiked = (target_type, target_id) =>
  call('user-likes', 'is-liked', { method: 'GET', params: { target_type, target_id } })

/**
 * 批量查询点赞状态 + 数量（列表页专用）
 *
 * 一次拿到整页内容的 { is_liked, count }，替代「每张卡片各调一次 is-liked」。
 * 未登录时 is_liked 全为 false，count 照常返回。
 *
 * 返回：{ data: { states: { [id]: { is_liked, count } } } }
 */
export const likeStates = (target_type, target_ids) => {
  const ids = Array.isArray(target_ids) ? target_ids : [target_ids]
  return call('user-likes', 'states', {
    method: 'GET',
    params: { target_type, target_ids: ids.join(',') }
  })
}

// 批量查询点赞数（参考 i-comment.vue：user-likes/counts 用 target_ids 数组）
export const likesCount = (target_type, target_ids) => {
  const ids = Array.isArray(target_ids) ? target_ids : [target_ids]
  return call('user-likes', 'counts', {
    method: 'GET',
    params: { target_type, target_ids: ids.join(',') }
  })
}

// 收藏
export const collect = (target_type, target_id) =>
  call('user-collects', 'collect', { method: 'POST', data: { target_type, target_id } })

export const uncollect = (target_type, target_id) =>
  call('user-collects', 'uncollect', { method: 'PUT', data: { target_type, target_id } })

export const isCollected = (target_type, target_id) =>
  call('user-collects', 'is-collected', { method: 'GET', params: { target_type, target_id } })

export const collectsCount = (target_type, target_ids) => {
  // 支持传入单个 id 或 id 数组，统一拼接为逗号分隔的 target_ids
  const ids = Array.isArray(target_ids) ? target_ids : [target_ids]
  return call('user-collects', 'counts', {
    method: 'GET',
    params: { target_type, target_ids: ids.join(',') }
  })
}

export const myCollects = (params = {}) =>
  call('user-collects', 'collects', { method: 'GET', params })

// 获取点赞列表（支持 where: { login } 查询他人点赞）
export const myLikes = (params = {}) =>
  call('user-likes', 'likes', { method: 'GET', params })

// 关注相关接口（follow / unfollow / is-following / relation / suggest / stats）由
// views/user/Author.vue 直接调 call('user-follows', ...)，这里不再另做一层封装。

// 通知
export const listNotifications = (params = {}) =>
  call('notification', 'list', { method: 'GET', params })

export const unreadCount = () =>
  call('notification', 'unread-count', { method: 'GET' })

export const readNotification = (id) =>
  call('notification', 'read', { method: 'PUT', data: { id } })

export const readAllNotifications = () =>
  call('notification', 'read-all', { method: 'PUT' })

// 清空通知：支持 { type }（按类型）、{ is_read: 1 }（仅已读），清空后进入回收站
export const removeAllNotifications = (params = {}) =>
  call('notification', 'remove-all', { method: 'DELETE', params })