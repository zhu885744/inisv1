/**
 * Cookie 工具
 */
import { API_URI } from '@/utils/runtimeConfig'

/**
 * 计算 cookie 的默认属性后缀
 *
 * 为什么要显式写：token 同时存在 cookie 与 localStorage（见 api/request.js），
 * 而不带属性的 cookie 在部分浏览器里等同 SameSite=None —— 任何第三方站点发起的
 * 跨站请求都会把它带上，配合后台「用 cookie 判定管理会话」的接口就是 CSRF 面。
 *
 * 但不能一律写 Lax：主题支持前后端分离部署（前端一个域、API 另一个域），
 * 这种跨站场景里 Lax 会让浏览器在 XHR / fetch 上直接丢弃 cookie，
 * 只能显式 None + Secure。所以这里按「是否同站」分流：
 *   - 同站（同源，或同一注册域下的子域）→ Lax，XHR 不受影响，且挡住跨站携带；
 *   - 跨站 → None + Secure（浏览器只接受这一种组合）；
 *   - http 页面不加 Secure，否则本地开发时 cookie 会被直接丢弃。
 */
function cookieSuffix() {
  const loc = typeof location !== 'undefined' ? location : null
  if (!loc) return '; SameSite=Lax'

  const secure = loc.protocol === 'https:' ? '; Secure' : ''

  // 未配置 API 地址 = 同源部署，必定同站
  if (!API_URI) return `${secure}; SameSite=Lax`

  try {
    const api = new URL(API_URI, loc.origin)
    const sameSite =
      api.hostname === loc.hostname ||
      api.hostname.endsWith(`.${loc.hostname}`) ||
      loc.hostname.endsWith(`.${api.hostname}`)
    return sameSite ? `${secure}; SameSite=Lax` : '; SameSite=None; Secure'
  } catch {
    // 地址解析失败按同站处理：Lax 更安全，且不会让同源部署失效
    return `${secure}; SameSite=Lax`
  }
}

export const getCookie = (name) => {
  if (!name || !document.cookie) return ''
  // 用边界匹配，避免 name 命中其它 cookie 名的子串（如 token 匹配 token_backup）
  const match = document.cookie.match(new RegExp('(?:^|;\\s*)' + name + '=([^;]*)'))
  if (!match) return ''
  try {
    return decodeURIComponent(match[1])
  } catch {
    return match[1]
  }
}

export const setCookie = (name, value, seconds = 3600) => {
  const time = new Date()
  time.setTime(time.getTime() + seconds * 1000)
  document.cookie = `${name}=${value}; expires=${time.toUTCString()}; path=/${cookieSuffix()}`
}

export const hasCookie = (name) => {
  if (!name) return false
  return document.cookie.indexOf(name + '=') !== -1
}

export const clearCookie = (name) => {
  if (!name) return
  // 删除时属性尽量与写入一致，否则个别浏览器会留下同名残留
  document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/${cookieSuffix()}`
}
