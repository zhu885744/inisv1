/**
 * 浏览器标签资源：网站标题 + 网站描述/关键词 + 网站图标（favicon）
 *
 * 站点配置存在 config 表的 Mellow_functions 记录里（后台「系统设置 → 网站设置」），
 * index.html 里的 <title> / <meta> / <link rel="icon" href="/favicon.ico"> 只是
 * 「未配置前的兜底」，运行时按配置覆盖：
 *
 * - 有配置就覆盖，配置为空（没填或后台被清空）时保持 index.html 的默认值，
 *   不会把标题/图标变成空白；
 * - 在 App.vue 里随站点配置加载应用一次，前台、登录页、后台、用户中心/创作中心
 *   所有布局共用（它们都挂在同一个 App 下），后台改完刷新一次即可看到效果。
 */

import { setSiteName } from './pageTitle'

// index.html 的默认值：配置为空时用它兜底
const DEFAULT_ICON = '/favicon.ico'

// 扩展名 → 图标 MIME：拿不准就不写 type，交给浏览器判断
const ICON_TYPES = {
  ico: 'image/x-icon',
  png: 'image/png',
  svg: 'image/svg+xml',
  webp: 'image/webp',
  jpg: 'image/jpeg',
  jpeg: 'image/jpeg',
  gif: 'image/gif'
}

function iconType(url) {
  const ext = String(url).split('?')[0].split('#')[0].split('.').pop().toLowerCase()
  return ICON_TYPES[ext] || ''
}

/** 更新（或创建）head 里的 meta[name] 标签，value 为空时保持原样 */
function applyMeta(name, value) {
  const text = String(value || '').trim()
  if (!text) return

  let el = document.head.querySelector(`meta[name="${name}"]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('name', name)
    document.head.appendChild(el)
  }
  el.setAttribute('content', text)
}

/**
 * 应用网站图标（favicon）
 *
 * 更新 head 里已有的 <link rel="icon">（index.html 那条），没有就创建一条；
 * url 为空时回退 index.html 的默认图标 /favicon.ico，避免把标签图标清成空白。
 */
export function applyFavicon(url) {
  if (typeof document === 'undefined') return

  const href = String(url || '').trim() || DEFAULT_ICON
  const type = iconType(href)

  let link = document.head.querySelector('link[rel~="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.setAttribute('rel', 'icon')
    document.head.appendChild(link)
  }

  link.setAttribute('href', href)

  if (type) {
    link.setAttribute('type', type)
  } else {
    link.removeAttribute('type')
  }

  // sizes="any" 是给可缩放图标（svg）用的，其它格式交回浏览器自己判断
  if (type === ICON_TYPES.svg) {
    link.setAttribute('sizes', 'any')
  } else {
    link.removeAttribute('sizes')
  }
}

/**
 * 应用站点信息到浏览器标签（标题 / 描述 / 关键词 / 图标）
 *
 * @param {object} config 站点配置（Mellow_functions 的 json）
 */
export function applySiteAssets(config = {}) {
  if (typeof document === 'undefined') return

  // 标签标题 =「页面标题 - 站点名」：这里只提供站点名，
  // 拼装与页面标题（路由 meta.title）由 utils/pageTitle.js 统一处理
  setSiteName(config.title)

  applyMeta('description', config.description)
  applyMeta('keywords', config.keyword)

  applyFavicon(config.favicon)
}
