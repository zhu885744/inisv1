import { call } from './request'

/**
 * @ 提及（@用户）
 *
 * 候选范围**仅限「我关注的人」** —— 与后端口径完全一致（见 app/model/mention.go 的注释）：
 * 未关注的人不出现在候选里，保存时也不会被解析成提及（既不转链接也不发通知）。
 *
 * 一次取全量（后端上限 500），组件里按关键词在本地过滤：
 * @ 面板会随输入实时刷新，每敲一个字都打接口既没必要也容易被限流。
 */
let cache = null
let pending = null

/**
 * 取候选列表
 * @param {boolean} force 强制刷新（忽略缓存，如刚关注/取关后）
 * @returns {Promise<Array<{id:number,nickname:string,avatar:string}>>}
 */
export const getMentionCandidates = async (force = false) => {
  if (!force && cache) return cache

  if (!pending) {
    pending = call('mention', 'candidates', { method: 'GET', params: { limit: 500 } })
      .then((res) => {
        cache = res?.code === 200 && Array.isArray(res.data) ? res.data : []
        return cache
      })
      .catch(() => [])
      .finally(() => {
        pending = null
      })
  }

  return pending
}

/** 清缓存（关注关系变化后调用） */
export const clearMentionCache = () => {
  cache = null
}
