import { call } from './request'

/**
 * 拉黑（黑名单）
 *
 * 后端：app/api/controller/user-blocks.go
 *   GET  user-blocks/mine     我的拉黑列表（含被拉黑用户的公开信息）
 *   GET  user-blocks/status   我是否已拉黑某人
 *   POST user-blocks/block    拉黑（同时解除双方的关注关系）
 *   PUT  user-blocks/unblock  解除拉黑
 *
 * 语义是双向的：拉黑后彼此的内容都不再出现在对方的聚合流里，且不能互相私信。
 * 被拉黑方**不会**收到任何提示，也看不到「被谁拉黑」。
 */

/** 我的拉黑列表（分页） */
export const myBlocks = (params = {}) =>
  call('user-blocks', 'mine', { method: 'GET', params: { page: 1, limit: 20, ...params } })

/** 我是否已拉黑某人 */
export const blockStatus = (blockUid) =>
  call('user-blocks', 'status', { method: 'GET', params: { block_uid: blockUid } })

/** 拉黑（reason 可选，仅自己可见） */
export const blockUser = (blockUid, reason = '') =>
  call('user-blocks', 'block', { method: 'POST', data: { block_uid: blockUid, reason } })

/** 解除拉黑 */
export const unblockUser = (blockUid) =>
  call('user-blocks', 'unblock', { method: 'PUT', data: { block_uid: blockUid } })
