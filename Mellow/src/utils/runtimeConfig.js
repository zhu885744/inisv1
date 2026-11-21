/**
 * 运行时后端地址解析
 *
 * 取值优先级（从高到低）：
 * 1. window.__INIS_CONFIG__ —— 由 public/runtime-config.js 提供，
 *    部署到后端 public 目录后可直接修改，无需重新打包；
 * 2. 打包时的 Vite 环境变量（VITE_API_URI / VITE_SOCKET）；
 * 3. 同源（空字符串）—— 主题由后端托管时，/api 就是后端自身的地址。
 *
 * 约定：runtime-config.js 中显式写 '' 表示「强制同源」；
 *       不写该字段（undefined）才回落到打包时的 .env。
 */
const RUNTIME = (typeof window !== 'undefined' && window.__INIS_CONFIG__) || {}

/** 取值：运行时优先，未声明该字段时回落到打包时的值 */
const pick = (key, buildTime) => {
  const value = Object.prototype.hasOwnProperty.call(RUNTIME, key) ? RUNTIME[key] : buildTime
  return typeof value === 'string' ? value.trim() : ''
}

/** 去掉结尾多余的 / */
const trimEnd = (url) => url.replace(/\/+$/, '')

/** 后端根地址，空字符串表示与当前站点同源 */
export const API_URI = trimEnd(pick('apiUri', import.meta.env.VITE_API_URI))

/** 接口基址（统一 /api 前缀），空 API_URI 时退化为同源相对路径 */
export const API_BASE_URL = API_URI ? `${API_URI}/api` : '/api'

/** 不带 /api 前缀的接口基址（如 /dev/info/time） */
export const API_ROOT_URL = API_URI

/** socket 地址，空字符串表示同源 */
export const SOCKET_URI = pick('socketUri', import.meta.env.VITE_SOCKET)

/**
 * 把后端返回的相对地址补全为可直接访问的完整地址
 * （表情包、附件等静态资源用）
 */
export const getFullUrl = (url) => {
  if (!url) return ''
  if (/^https?:\/\//.test(url)) return url
  if (url.startsWith('//')) return `${typeof location !== 'undefined' ? location.protocol : 'https:'}${url}`
  return `${API_URI}${url.startsWith('/') ? url : `/${url}`}`
}
