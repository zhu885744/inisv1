import { call } from './request'
import { uploadAttachments } from './attachment'

export const getUser = (id, field = '') =>
  call('users', 'one', { method: 'GET', params: { id, field } })

/**
 * 批量获取用户（一次最多 50 个）
 *
 * 列表里要多个用户信息时用它，不要对 getUser 循环 —— 每次 one 都要跑一遍
 * 权限组 / 等级 / 封禁查询，N 次就是 N 倍开销。
 * 返回顺序与传入 ids 一致，查不到的 id 直接缺席。
 */
export const getUsersBatch = (ids = [], field = '') =>
  call('users', 'batch', { method: 'GET', params: { ids: ids.join(','), field } })

export const updateUser = (data) =>
  call('users', 'update', { method: 'PUT', data })

// 头像上传统一走 uploadAttachments（含「图片压缩为 WebP」处理）
export const uploadAvatar = (formData) => uploadAttachments(formData)

// 修改邮箱（发验证码 / 提交验证码）
export const updateEmail = (data) =>
  call('users', 'email', { method: 'PUT', data })

// 修改手机号（发验证码 / 提交验证码）
export const updatePhone = (data) =>
  call('users', 'phone', { method: 'PUT', data })

/**
 * 修改密码（登录态 + 身份验证）
 *
 * - data.old_password：当前密码（与 email_code / phone_code 三选一）
 * - data.email_code / data.phone_code：发到已绑定邮箱 / 手机号的验证码
 *   从未设过密码的账号（第三方登录注册）没有旧密码可用，走验证码这条路
 * - data.password：新密码（后端会做长度 / 弱密码 / 非纯数字校验）
 */
export const changePassword = (data) =>
  call('users', 'change-password', { method: 'PUT', data })

// 发送身份验证码：发给当前账号已绑定的邮箱 / 手机号（mode 传 'email' 或 'sms'，不传则优先邮箱）
export const sendIdentityCode = (mode = '') =>
  call('users', 'verify-code', { method: 'PUT', data: mode ? { mode } : {} })

/**
 * 账号修改规则（登录用户）
 *
 * 返回后台配置（是否开放改账号 / 冷却天数 / 是否要求绑定联系方式与身份验证）
 * 以及当前用户自身的冷却状态：
 *   { enabled, cooldown_days, require_contact, require_verify, has_contact,
 *     can_change, next_time, remaining_days, min_length, max_length, message }
 * 前台据此决定是否显示「修改」入口，并直接用 message 展示原因。
 */
export const getAccountRule = () =>
  call('users', 'account-rule', { method: 'GET' })

/**
 * 修改账号（用户名）
 *
 * - data.account：新账号（4-20 位字母、数字、下划线）
 * - data.password / data.email_code / data.phone_code：身份验证三选一（后台可关）
 * 冷却期（默认改完 30 天内不可再改）与「必须绑定手机号或邮箱」由后端按配置强制校验。
 */
export const changeAccount = (data) =>
  call('users', 'change-account', { method: 'PUT', data })

// 注销账户（发验证码）
export const destroySendCode = () =>
  call('users', 'destroy', { method: 'DELETE' })

// 注销账户（提交验证码 + 密码二次验证）
// code/password 为必填，source 为可选注销原因（默认 default）
export const destroy = (code, password, source = 'default') =>
  call('users', 'destroy', { method: 'DELETE', params: { code, password, source } })

// ===== 小黑屋（封禁公示） =====
// 后端已对封禁用户昵称脱敏；默认只查生效中的封禁（status=0）
export const getBlackroom = (params = {}) =>
  call('users', 'blackroom', {
    method: 'GET',
    params: { page: 1, limit: 20, order: 'create_time desc', ...params }
  })

// 提交封禁申诉（需登录，record_id 为当前生效的封禁记录 ID）
export const submitAppeal = (recordId, content) =>
  call('users', 'appeal', { method: 'POST', data: { record_id: recordId, content } })

// ===== 签到 =====
// 签到已从经验模块独立出去（后端 /api/checkin/*，配置键 SYSTEM_CHECKIN_RULES），
// 相关请求统一放在 @/api/checkin（getCheckinStatus / getCheckinCalendar / getCheckinRank /
// signIn / makeupSign …）。这里不再保留 exp/check-in* 的旧封装，避免两套入口混用。

export const expActive = (params = {}) =>
  call('exp', 'active', { method: 'GET', params })

// 经验任务规则
export const getExpRules = () =>
  call('exp', 'rules', { method: 'GET' })

// 经验明细（登录用户，where 传 { uid } 的 JSON 字符串）
export const getExpLogs = (params = {}) =>
  call('exp', 'all', { method: 'GET', params: { page: 1, limit: 20, order: 'create_time desc', ...params } })

// 等级体系列表
export const getLevels = (params = {}) =>
  call('level', 'all', { method: 'GET', params: { page: 1, limit: 100, order: 'exp asc', ...params } })

// ===== 后台（管理员）=====

// 批量按 id 拉取用户基础信息（后台列表展示作者昵称/头像/账号）
// where 使用 $in，后端会转换为 `id` IN (...)
// 仅后台页面使用（文章 / 动态 / 页面 / 友链列表），account 用于 hover 提示申请人身份
export const listUsersByIds = (ids = []) => {
  const list = (Array.isArray(ids) ? ids : [ids]).map((i) => Number(i)).filter((i) => i > 0)
  if (!list.length) return Promise.resolve({ code: 204, data: null })
  return call('users', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 100,
      field: 'id,nickname,avatar,account',
      where: JSON.stringify({ id: { $in: list } })
    }
  })
}

// 用户列表（管理员视角，后端不脱敏；field/where/order/like 由 utils/user.js 统一拼装）
export const listUsers = (params = {}) =>
  call('users', 'all', {
    method: 'GET',
    params: { page: 1, limit: 15, order: 'id desc', ...params }
  })

// 用户数量统计（注意：后端 users/count 不支持 onlyTrashed，回收站计数请用 listUsers 的 count）
export const countUsers = (params = {}) =>
  call('users', 'count', { method: 'GET', params })

// 新建用户（后端要求邮箱非空；password 为空时账号无法用密码登录）
export const createUser = (data) =>
  call('users', 'create', { method: 'POST', data })

// 修改用户状态（0 正常 / 1 冻结），后端禁止操作系统管理员
export const setUserStatus = (id, status) =>
  call('users', 'status', { method: 'PUT', data: { id: Number(id), status: Number(status) } })

// 封禁用户（管理员）
// duration 传字符串：'0' 表示永久封禁；不传 duration 并置 auto_gradient=true 时走后端自动梯度
export const banUser = (data) =>
  call('users', 'ban', { method: 'PUT', data })

// 解封用户（传 uid 或 record_id 其一）
export const unbanUser = (data) =>
  call('users', 'unban', { method: 'PUT', data })

/**
 * 处理封禁申诉（管理员）
 *
 * - action = approve：通过，立即解封（清空 restrictions / current_ban_id，记录状态=4）
 * - action = reject：驳回，封禁继续生效（记录状态=5），reply 必填（驳回理由）
 * 两种结果都会给用户发站内消息（强制发送）。
 */
export const handleUserAppeal = (data) =>
  call('users', 'appeal-handle', { method: 'PUT', data })

/**
 * 清空用户封禁信息（管理员）
 *
 * 物理删除该用户的全部封禁记录（含回收站里的），并把 ban_count / current_ban_id /
 * last_ban_at / restrictions 一起归零；unfreeze=1 时若账号处于「冻结」状态一并恢复为正常。
 * 与 unban 的区别：unban 只撤销当前记录、保留历史；clear-ban 是把封禁痕迹彻底抹掉。
 */
export const clearUserBan = (data) =>
  call('users', 'clear-ban', { method: 'PUT', data })

// 软删除用户（移入回收站），禁止包含系统管理员与自己
export const removeUsers = (ids) =>
  call('users', 'remove', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 彻底删除用户（不可恢复）
export const forceDeleteUsers = (ids) =>
  call('users', 'delete', {
    method: 'DELETE',
    params: { ids: (Array.isArray(ids) ? ids : [ids]).join(',') }
  })

// 从回收站恢复用户
export const restoreUsers = (ids) =>
  call('users', 'restore', {
    method: 'PUT',
    data: { ids: Array.isArray(ids) ? ids : [ids] }
  })

// 清空用户回收站
export const clearUserRecycle = () =>
  call('users', 'clear', { method: 'DELETE' })
