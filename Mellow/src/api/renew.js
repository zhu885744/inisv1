import { call } from './request'

/**
 * 续期下单（勋章 / 装扮）
 *
 * 续期档位不是全站统一的：每个勋章、每件装扮在自己的 renew 字段里配「多少天 / 多少积分」，
 * 随「我的勋章」「我的装扮」一起下发（renew_options），前台不单独请求价格表。
 *
 * 下单只传天数，价格由服务端按该商品自己的档位取，前端改不了价。
 */

/** 勋章续期（返回 expire_after / integral 余额） */
export const renewBadge = (badgeId, days) =>
  call('renew', 'badge', {
    method: 'POST',
    data: { badge_id: Number(badgeId), days: Number(days) }
  })

/** 装扮续期 */
export const renewDecoration = (decorationId, days) =>
  call('renew', 'decoration', {
    method: 'POST',
    data: { decoration_id: Number(decorationId), days: Number(days) }
  })
