/**
 * 后台管理会话（scope=admin）
 *
 * 为什么需要它：
 *   前台登录态（INIS_LOGIN_TOKEN）是长期挂在浏览器上的（默认 15 天），
 *   一旦被 XSS / 共享电脑 / 代理日志拿到，就等于拿到了后台入口。
 *   后台因此另建一条会话：独立 cookie（INIS_ADMIN_TOKEN，HttpOnly、默认 2 小时），
 *   只有「有后台页面权限」的账号**重新输入一次密码**才能签发。
 *
 * 交互约定：
 *   - 后台（/admin）发出的接口请求失败时统一返回 { code: 401, auth: 'admin' }
 *     （见 app/api/middleware/rule.go），由 api/request.js 唤起这里的密码弹窗，
 *     通过后自动重放原请求；
 *   - 弹窗本体是 AdminUnlockDialog.vue（挂在 App.vue，与滑块验证同一套「全局单例」写法），
 *     本模块只做状态与流程，不 import 任何组件，避免与 request.js 形成循环依赖。
 */

import { adminCheck } from '@/api/comm'

// 最近一次确认通过的时间（ms）：后台内部切页不必每次都问一遍后端
let lastVerifiedAt = 0
const VERIFY_INTERVAL = 30 * 1000

// 并发保护：同时有多个请求拿到 401 时只弹一次密码框
let unlockPromise = null

// 弹窗实现（由 AdminUnlockDialog 挂载时注册）
let unlockHandler = null

/** 注册密码弹窗实现（AdminUnlockDialog 挂载时调用） */
export function registerAdminUnlockHandler(handler) {
  unlockHandler = handler
}

/** 注销（组件卸载时调用，避免持有已销毁组件的引用） */
export function unregisterAdminUnlockHandler() {
  unlockHandler = null
}

/**
 * 弹出密码框换一条后台会话
 * @param {string} message 后端返回的原因（如「进入后台需要重新验证密码！」）
 * @returns {Promise<boolean>} 是否已拿到有效的后台会话（取消返回 false）
 */
export function unlockAdminSession(message) {
  if (unlockPromise) return unlockPromise
  if (typeof unlockHandler !== 'function') return Promise.resolve(false)

  unlockPromise = Promise.resolve(unlockHandler(message))
    .then((ok) => {
      if (ok) lastVerifiedAt = Date.now()
      return !!ok
    })
    .catch(() => false)
    .finally(() => {
      unlockPromise = null
    })

  return unlockPromise
}

/**
 * 进入 /admin 前的门槛：确认后台会话可用（失效则弹密码框）
 * 注意：401 的弹窗与重放由 api/request.js 统一处理，这里只负责判断结果。
 */
export async function ensureAdminSession() {
  if (lastVerifiedAt && Date.now() - lastVerifiedAt < VERIFY_INTERVAL) return true
  try {
    await adminCheck()
    lastVerifiedAt = Date.now()
    return true
  } catch {
    return false
  }
}

/** 后台会话被服务端确认失效 / 主动退出后调用，清掉本地短时缓存 */
export function resetAdminSession() {
  lastVerifiedAt = 0
}

export default { ensureAdminSession, unlockAdminSession, resetAdminSession }
