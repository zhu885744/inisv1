import { call } from './request'

/**
 * 群聊（公共大厅 id=1 + 用户自建群聊）
 *
 * 后端：app/api/controller/room.go（模块 room，全部 type=login）。
 * 实时消息走同一个 WebSocket：服务端按房间扇出
 *   { type:'room' | 'room_revoke' | 'room_delete' | 'room_mute' | 'room_unmute'
 *         | 'room_update' | 'room_clear' | 'room_announce' | 'room_dissolve'
 *         | 'room_member_join' | 'room_member_leave' | 'room_member_remove'
 *         | 'room_member_role' | 'room_join_request', data }，
 * 由 stores/chat.js 转成 { type:'room', event } 事件给房间页订阅。
 *
 * roomId 省略时后端按公共大厅（id=1）处理，老调用点无需改动。
 * 为什么发言也走 HTTP：与私信一致 —— 能拿到明确的业务错误（禁言 / 非成员 / 超长），
 * 幂等键 client_msg_id 让连点与重试都不会产生重复消息；落库后服务端会广播给成员。
 */

const rid = (roomId) => ({ room_id: Number(roomId) || 1 })

/** 群列表：scope=mine 我加入的 / discover 可发现并加入的 */
export const listRooms = (params = {}) =>
  call('room', 'list', { method: 'GET', params: { scope: 'mine', ...params } })

/** 创建群聊：{ name, description, avatar, join_type }（0 自由 / 1 审批 / 2 仅邀请） */
export const createRoom = (data) => call('room', 'create', { method: 'POST', data })

/** 房间资料 + 我的状态（角色 / 是否禁言 / 是否可管理） */
export const getRoomInfo = (roomId) =>
  call('room', 'info', { method: 'GET', params: rid(roomId) })

/**
 * 历史消息
 * @param {number} roomId
 * @param {object} [params] before_id：往前翻页；after_seq：断线补拉；limit：条数
 */
export const listRoomMessages = (roomId, params = {}) =>
  call('room', 'messages', { method: 'GET', params: { ...rid(roomId), limit: 30, ...params } })

export const sendRoomMessage = (roomId, data) =>
  call('room', 'send', { method: 'POST', data: { ...rid(roomId), ...data } })

// 撤回：本人 2 分钟内，群管理 / 管理员不限
export const revokeRoomMessage = (roomId, msgId) =>
  call('room', 'revoke', { method: 'POST', data: { ...rid(roomId), msg_id: Number(msgId) } })

/**
 * 删除一条消息（只对自己生效）
 *
 * 与撤回不同：撤回会清空内容并广播给所有人；删除只是把自己视图里的这条藏起来，
 * 对方 / 其他人完全无感。
 */
export const deleteRoomMessage = (roomId, msgId) =>
  call('room', 'delete', { method: 'POST', data: { ...rid(roomId), msg_id: Number(msgId) } })

/**
 * 禁言（群主 / 群管理员；公共大厅仅超级管理员）
 * @param {number} uid 目标用户
 * @param {number} [minutes] 禁言分钟数，0 = 永久
 */
export const muteRoomUser = (roomId, uid, minutes = 0, reason = '') =>
  call('room', 'mute', {
    method: 'POST',
    data: { ...rid(roomId), uid: Number(uid), minutes: Number(minutes), reason }
  })

export const unmuteRoomUser = (roomId, uid) =>
  call('room', 'unmute', { method: 'POST', data: { ...rid(roomId), uid: Number(uid) } })

// 生效中的禁言名单（群主 / 群管理员）
export const listRoomMutes = (roomId) =>
  call('room', 'mutes', { method: 'GET', params: rid(roomId) })

/**
 * 修改房间资料（群主 / 公共大厅为超级管理员）
 * @param {object} data { name, description, avatar, join_type }
 */
export const updateRoom = (roomId, data) =>
  call('room', 'update', { method: 'PUT', data: { ...rid(roomId), ...data } })

/** 发布 / 清除群公告（notice 传空串即清除） */
export const announceRoom = (roomId, notice) =>
  call('room', 'announce', { method: 'POST', data: { ...rid(roomId), notice } })

/** 入群：自由加入直接成功，需审批的群返回 pending=true */
export const joinRoom = (roomId, message = '') =>
  call('room', 'join', { method: 'POST', data: { ...rid(roomId), message } })

/** 退群（群主需先解散） */
export const leaveRoom = (roomId) =>
  call('room', 'leave', { method: 'POST', data: rid(roomId) })

/** 解散群聊（群主 / 超级管理员） */
export const dissolveRoom = (roomId) =>
  call('room', 'dissolve', { method: 'POST', data: rid(roomId) })

/** 成员列表（公共大厅返回空，前端用在线成员面板） */
export const listRoomMembers = (roomId) =>
  call('room', 'members', { method: 'GET', params: rid(roomId) })

/** 设置 / 撤销群管理员：role 0 成员 / 1 管理员 */
export const setRoomMemberRole = (roomId, uid, role) =>
  call('room', 'member-role', {
    method: 'POST',
    data: { ...rid(roomId), uid: Number(uid), role: Number(role) }
  })

/** 邀请成员（群主 / 管理员；「仅邀请」的群只能这样进人） */
export const addRoomMember = (roomId, uid) =>
  call('room', 'member-add', { method: 'POST', data: { ...rid(roomId), uid: Number(uid) } })

/**
 * 生成邀请（链接 / 二维码共用）
 *
 * 返回的 code 拼成 <站点>/join/<code> 就是邀请链接，二维码画的是同一个地址。
 * 群邀请一律按「永久 + 不限次」生成（days=0 / max_uses=0）：
 * 链接要能长期贴在主页、群公告或二维码海报上，带时限会变成死链。
 * @param {number} days 有效期天数，0 = 永久（默认）
 * @param {number} maxUses 可用次数，0 = 不限
 */
export const createRoomInvite = (roomId, { days = 0, maxUses = 0 } = {}) =>
  call('room', 'invite-create', {
    method: 'POST',
    data: { ...rid(roomId), days: Number(days) || 0, max_uses: Number(maxUses) || 0 }
  })

/** 邀请落地页信息：群资料 + 我是否已在群里 + 是否过期 */
export const getRoomInvite = (code) =>
  call('room', 'invite', { method: 'GET', params: { code: String(code || '') } })

/** 凭邀请码加入群聊（不受入群方式限制） */
export const joinRoomByCode = (code) =>
  call('room', 'join-by-code', { method: 'POST', data: { code: String(code || '') } })

/** 已生成的邀请列表（群主 / 管理员） */
export const listRoomInvites = (roomId) =>
  call('room', 'invites', { method: 'GET', params: rid(roomId) })

/** 移出成员（群主 / 管理员） */
export const removeRoomMember = (roomId, uid) =>
  call('room', 'member-remove', { method: 'POST', data: { ...rid(roomId), uid: Number(uid) } })

/** 待审的入群申请（群主 / 管理员） */
export const listRoomRequests = (roomId) =>
  call('room', 'requests', { method: 'GET', params: rid(roomId) })

/** 审批入群申请 */
export const handleRoomRequest = (roomId, requestId, approve) =>
  call('room', 'request-handle', {
    method: 'POST',
    data: { ...rid(roomId), request_id: Number(requestId), approve: !!approve }
  })

/** 上报已读水位（进群 / 拉到最新消息后调用，用于列表页未读数） */
export const markRoomRead = (roomId, lastId) =>
  call('room', 'read', { method: 'POST', data: { ...rid(roomId), last_id: Number(lastId) || 0 } })

/**
 * 自动清空配置（仅超级管理员，仅公共大厅）
 *
 * 返回 { days, options, last_time, last_count, next_time }：
 * 清空本身由服务端定时任务执行（app/timer/room.go），前端只负责改间隔。
 */
export const getRoomAutoClear = () => call('room', 'auto-clear', { method: 'GET' })

/**
 * 保存自动清空间隔（仅超级管理员）
 * @param {number} days 间隔天数，只接受后端提供的选项
 */
export const saveRoomAutoClear = (days) =>
  call('room', 'auto-clear', { method: 'PUT', data: { days: Number(days) } })
