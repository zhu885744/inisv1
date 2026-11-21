import { call } from './request'

/**
 * 用户私聊（私信）
 *
 * 后端：app/api/controller/chat.go（模块 chat，全部 type=login），
 * 实时送达复用现有 WebSocket（/socket 的 private 消息），见 stores/chat.js。
 *
 * 为什么发送走 HTTP 而不是直接用 socket 发：
 *   - HTTP 有明确的返回码与消息体（拉黑 / 内容为空 / 超长都能直接提示），
 *     幂等键（client_msg_id）重试也不会产生重复消息；
 *   - 服务端落库后会通过 socket 推给在线的对方，实时性一样。
 * socket 只在「收」的方向上用。
 */

// 我的会话列表（对方信息 + 最后一条摘要 + 未读数）
export const listChatConversations = (params = {}) =>
  call('chat', 'conversations', { method: 'GET', params: { limit: 50, ...params } })

/**
 * 互关好友列表（私信只对互关好友开放）
 *
 * 会话列表只有「聊过的人」，这里补上还没聊过的互关好友，
 * 消息页合并展示后点一下就能开聊，不必再去对方主页点「私信」。
 * 返回项：{ uid, user: { id, nickname, avatar, ... }, online }
 * @param {object} [params] limit：最多返回人数（服务端上限 200）
 */
export const listChatFriends = (params = {}) =>
  call('chat', 'friends', { method: 'GET', params: { limit: 200, ...params } })

/**
 * 与某人的历史消息（游标分页）
 * @param {number} targetUid 对方用户 ID
 * @param {object} [params] before_id：取这个 id 之前的一页；limit：条数（默认 20，最大 50）
 */
export const listChatMessages = (targetUid, params = {}) =>
  call('chat', 'messages', {
    method: 'GET',
    params: { target_uid: Number(targetUid), limit: 20, ...params }
  })

/**
 * 断线重连补拉：after_seq 之后的消息（正序）
 *
 * seq 就是消息 id（自增主键，全局单调），客户端只要记着「本地已有的最大 seq」，
 * 重连时带上来，服务端返回更大的那些即可，不会漏也不会重。
 * @param {number} targetUid 对方用户 ID
 * @param {object} [params] after_seq：本地已有的最大 seq；limit：条数
 */
export const syncChatMessages = (targetUid, params = {}) =>
  call('chat', 'sync', {
    method: 'GET',
    params: { target_uid: Number(targetUid), ...params }
  })

/**
 * 批量确认「已收到 seq 及之前的消息」（水位只增不减）
 *
 * 正常走 socket（{type:'ack'}）更省一次往返，这里作为连不上时的兜底。
 */
export const ackChatMessages = (targetUid, seq) =>
  call('chat', 'ack', {
    method: 'PUT',
    data: { target_uid: Number(targetUid), seq: Number(seq) }
  })

// 我的私信未读总数
export const getChatUnread = () => call('chat', 'unread', { method: 'GET' })

/**
 * 会话偏好：置顶 / 免打扰（只影响自己那一侧的视图）
 *
 * - pinned：会话列表排到最前；
 * - muted ：新消息照常计未读，但不响提示音、不弹悬浮 / 桌面通知。
 * 两个字段可只传一个，未传的保持原值。
 * @param {object} data { target_uid, pinned?: boolean, muted?: boolean }
 */
export const setChatPref = (data) => call('chat', 'pref', { method: 'PUT', data })

// 能否与某人私聊（作者主页的「发私信」按钮状态）
export const getChatStatus = (targetUid) =>
  call('chat', 'status', { method: 'GET', params: { target_uid: Number(targetUid) } })



/**
 * 发送私信
 * @param {object} data { target_uid, content, msg_type?: 'text'|'image', client_msg_id?: string, quote_id?: number }
 */
export const sendChatMessage = (data) => call('chat', 'send', { method: 'POST', data })

/**
 * 删除一条消息（只对自己生效）
 *
 * 与撤回的区别：撤回会清空内容并通知对方，双方都看到「撤回了一条消息」；
 * 删除只是把这条从自己的历史 / 未读数里藏起来，对方完全无感。
 */
export const deleteChatMessage = (msgId) =>
  call('chat', 'delete', { method: 'POST', data: { msg_id: Number(msgId) } })

// 把某人的消息标记为已读（服务端会给对方推 chat_read 回执）
export const readChatMessages = (targetUid) =>
  call('chat', 'read', { method: 'PUT', data: { target_uid: Number(targetUid) } })

// 撤回自己发的消息（仅 2 分钟内，服务端校验）
export const revokeChatMessage = (msgId) =>
  call('chat', 'revoke', { method: 'POST', data: { msg_id: Number(msgId) } })

/**
 * 拍一拍
 * @param {number} targetUid 对方用户 ID
 * @param {string} [suffix] 后缀（如「的后脑勺」），留空就是「拍了拍你」
 */
export const patChatUser = (targetUid, suffix = '') =>
  call('chat', 'pat', {
    method: 'POST',
    data: { target_uid: Number(targetUid), suffix }
  })
