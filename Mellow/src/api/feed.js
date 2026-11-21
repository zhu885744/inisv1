import { call } from './request'

/**
 * 聚合流（关注 / 推荐 / 热门）
 *
 * 后端：GET /api/feed/all?type=&page=&limit=（app/api/controller/feed.go）
 *   - 必须登录（权限种子 path=all&type=login），未登录会 401，由拦截器引导登录
 *   - 只返回对外公开的动态（audit=1 且 status=1），草稿 / 待审核不会出现在流里
 *   - limit 后端钳到 20、page 钳到 30（前端也别传大值，那属于抓取行为）
 *   - 响应结构：res.data = { data, count, page, type }
 *
 * @param {'follow'|'recommend'|'hot'} type follow=关注  recommend=推荐  hot=热门
 * @param {{page?: number, limit?: number}} params
 */
export const getFeed = (type = 'follow', params = {}) =>
  call('feed', 'all', {
    method: 'GET',
    params: {
      type,
      page: 1,
      limit: 15,
      ...params
    }
  })
