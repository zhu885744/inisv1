/**
 * 运行时主题样式
 *
 * 背景：配色以前靠 <link href="/theme-custom.css"> 引入，但静态 .css 很容易被
 * CDN / 反向代理强缓存（保存后长期拿旧样式，甚至缓存了上线前的 400）。
 * 现在改成：
 *   1. 页面启动时请求 /api/theme/active 拿到 { version, css }（接口响应 no-store，必回源）；
 *   2. 把 CSS 注入 <style id="theme-runtime">，覆盖 styles.css 里的 :root 变量；
 *   3. 同时写一份到 localStorage，下次首屏由 index.html 的内联脚本同步应用，避免闪烁。
 *
 * 注意：生成的 CSS 选择器是 :root:root / :root:root[data-theme='dark']，
 * 特异性高于 styles.css 的 :root，所以注入顺序不影响覆盖结果。
 */
import { API_BASE_URL } from '@/utils/runtimeConfig'

const STYLE_ID = 'theme-runtime'
const STORAGE_KEY = 'inis_theme_css'

let version = ''

/** 取出（或创建）承载主题变量的 style 元素 */
function getStyleElement() {
  let style = document.getElementById(STYLE_ID)
  if (!style) {
    style = document.createElement('style')
    style.id = STYLE_ID
    document.head.appendChild(style)
  }
  return style
}

function applyCss(css) {
  if (typeof document === 'undefined') return
  getStyleElement().textContent = css
}

function readCache() {
  try {
    const text = window.localStorage.getItem(STORAGE_KEY)
    if (!text) return null
    const cached = JSON.parse(text)
    if (cached && typeof cached.css === 'string' && cached.css) return cached
  } catch {
    /* 隐私模式 / 存储被禁用时忽略 */
  }
  return null
}

function writeCache(nextVersion, css) {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify({ version: nextVersion, css }))
  } catch {
    /* 写不进去不影响本次生效 */
  }
}

/** 同步应用上次缓存的样式（在 main.js 最早调用，减少首屏闪烁） */
export function applyCachedTheme() {
  const cached = readCache()
  if (!cached) return
  version = cached.version || ''
  applyCss(cached.css)
}

/**
 * 拉取当前主题样式
 * @param {{ force?: boolean }} options force=true 时忽略 version 比对（保存后刷新用）
 */
export async function fetchTheme(options = {}) {
  const { force = false } = options
  const url = `${API_BASE_URL}/theme/active${force ? `?t=${Date.now()}` : ''}`

  const response = await fetch(url, {
    method: 'GET',
    credentials: 'include',
    cache: 'no-store',
    headers: { Accept: 'application/json' }
  })

  if (!response.ok) return

  const body = await response.json()
  const data = body?.data
  if (!data || typeof data.css !== 'string' || !data.css) return

  // 版本没变就不重复写 <style>，避免无谓的样式重算
  if (!force && data.version && data.version === version && readCache()) return

  version = data.version || ''
  applyCss(data.css)
  writeCache(version, data.css)
}

/** 启动入口：先应用缓存，再异步校准为服务端最新值 */
export function initTheme() {
  applyCachedTheme()
  fetchTheme().catch(() => {
    /* 网络失败时保留缓存 / 默认样式 */
  })
}

/** 保存配色后强制刷新（后台「主题配色」页调用） */
export function refreshTheme() {
  return fetchTheme({ force: true })
}

export function currentThemeVersion() {
  return version
}
