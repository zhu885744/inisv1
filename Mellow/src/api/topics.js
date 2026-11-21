import { call } from './request'

/**
 * 话题（Hashtag）接口
 *
 * 与后台「话题管理」共用 tags 表，但走独立的 topics 控制器：
 * 管理侧增删改仍在 api/tags.js，这里只放面向用户的能力。
 */

/**
 * 话题榜（按热度：文章×3 + 动态×2 + 关注×2）
 * @param {object} params { keyword, recommend, limit }
 */
export const listTopics = (params = {}) =>
  call('topics', 'board', { method: 'GET', params })

// 话题详情：id 与 key（话题名）二选一，与前台 /topic/:key 一致
export const getTopic = (params = {}) =>
  call('topics', 'detail', { method: 'GET', params })

/**
 * 话题内的内容聚合
 * @param {object} params { id, key, type: 'moments' | 'article', sort: 'new' | 'hot' | 'top', page, limit }
 */
export const getTopicContent = (params = {}) =>
  call('topics', 'content', { method: 'GET', params })

// 编辑器里输入 # 时的话题联想候选（需登录；一次取回后由浮层本地过滤）
//
// 页面上可能同时存在多个输入框（每条评论一个），所以这里做 60 秒的模块级缓存：
// 既避免同一页面重复请求，也不至于新建话题后长时间看不到自己刚建的话题。
let topicCandidatesCache = null
let topicCandidatesExpire = 0

export const getTopicCandidates = async (params = {}) => {
  const withKeyword = Object.keys(params).some((key) => key !== 'limit')

  if (!withKeyword && topicCandidatesCache && Date.now() < topicCandidatesExpire) {
    return { code: 200, data: topicCandidatesCache }
  }

  const res = await call('topics', 'candidates', {
    method: 'GET',
    params: { limit: 200, ...params }
  })

  if (!withKeyword) {
    topicCandidatesCache = res?.data || []
    topicCandidatesExpire = Date.now() + 60 * 1000
  }

  return res
}

// 我关注的话题
export const listMyTopics = (params = {}) =>
  call('topics', 'mine', { method: 'GET', params })

// 关注 / 取消关注（取消关注后端 PUT / POST 都支持，这里与 user-follows 一致走 POST）
export const followTopic = (id) =>
  call('topics', 'follow', { method: 'POST', data: { id } })

export const unfollowTopic = (id) =>
  call('topics', 'unfollow', { method: 'POST', data: { id } })

// 主持人编辑自己的话题资料（简介 / 封面）
export const updateTopicProfile = (data) =>
  call('topics', 'moderator-update', { method: 'POST', data })

// ===== 主持人能力（话题内置顶 / 移出话题 / 删除内容）=====
// data: { id: 话题ID, type: 'moments' | 'article', target_id: 内容ID }

export const pinTopicContent = (data) =>
  call('topics', 'pin', { method: 'POST', data })

export const unpinTopicContent = (data) =>
  call('topics', 'unpin', { method: 'POST', data })

// 移出话题：只解除关联，不删除内容本身
export const removeTopicContent = (data) =>
  call('topics', 'remove-content', { method: 'POST', data })

// 删除内容：软删除（可到后台回收站恢复），仅主持人 / 管理员
export const deleteTopicContent = (data) =>
  call('topics', 'delete-content', { method: 'POST', data })
