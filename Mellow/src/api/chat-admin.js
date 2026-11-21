import { call } from './request'

/**
 * 私信管理（后台）
 *
 * 后端：app/api/controller/chat-admin.go、app/model/chat-admin.go
 *
 * 与前台 `api/chat.js` 的区别：那组接口都围绕「我」，只能看自己的会话；
 * 这组接口给后台用，能看任意两人的会话与消息内容，撤回也不限 2 分钟。
 *
 * 权限：三个接口在 auth_rules 里登记为独立权限点（模块 chat-admin），
 * 没被授权的账号调用会被中间件拦掉（403），不会因为「登录了」就能看。
 */

/**
 * 全部私信会话（按最后一条消息时间倒序）
 * @param {object} [params] page / limit / uid（只看某人参与的会话）/ keyword（匹配最后一条摘要）
 */
export const listAdminChatConversations = (params = {}) =>
  call('chat-admin', 'conversations', {
    method: 'GET',
    params: { page: 1, limit: 20, ...params }
  })

/**
 * 某会话的消息内容（倒序取一页、正序返回；before_id 往前翻）
 * @param {number} conversationId 会话 ID（不是对方 uid）
 * @param {object} [params] before_id / limit
 */
export const listAdminChatMessages = (conversationId, params = {}) =>
  call('chat-admin', 'messages', {
    method: 'GET',
    params: { conversation_id: Number(conversationId), limit: 30, ...params }
  })

/**
 * 撤回一条消息（不限时间 / 不限发送者）
 *
 * scene=dm（默认）撤回私信，scene=chat 撤回群聊消息（后者走 room_revoke 广播，
 * 所有人立刻看到「已撤回」）。撤回只清内容并置标记，不做物理删除。
 */
export const revokeAdminChatMessage = (msgId, scene = 'dm') =>
  call('chat-admin', 'revoke', { method: 'POST', data: { msg_id: Number(msgId), scene } })

/**
 * 禁言用户（minutes=0 永久）
 *
 * 复用现有禁言体系：一次同时写「私信」与「群聊」两条记录，
 * 两处一起失效，解除也从同一个「禁言名单」里操作。
 */
export const muteChatUser = (uid, minutes, reason = '') =>
  call('chat-admin', 'mute', {
    method: 'POST',
    data: { uid: Number(uid), minutes: Number(minutes) || 0, reason }
  })

/**
 * 生效中的禁言名单（私信 + 群聊）
 * @param {object} [params] scope：'' 全部 / dm 私信 / room 群聊
 */
export const listChatMutes = (params = {}) =>
  call('chat-admin', 'mutes', { method: 'GET', params })

/**
 * 解除禁言
 * @param {number} uid
 * @param {string} [scope] dm / room；不传则两个作用域一起解除
 */
export const unmuteChatUser = (uid, scope = '') =>
  call('chat-admin', 'unmute', { method: 'POST', data: { uid: Number(uid), scope } })
