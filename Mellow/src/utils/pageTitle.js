/**
 * 浏览器标签标题（document.title）的统一管理
 *
 * 标题由两段拼成：**页面标题 - 站点名**
 *   - 页面标题：路由 `meta.title`（字符串，或 `(route) => string` 用于动态标题）。
 *     这个字段本来就是「页面名称」的公共约定 —— 后台各页早已用它（见 views/admin/Layout.vue
 *     的 currentTitle）与后台菜单名一致，所以后台页天然就有各自的标签标题；
 *   - 站点名：站点配置里的 title（异步加载，见 utils/siteAssets.js）。
 *
 * 为什么收口在这里：以前是单个页面（如用户协议）自己在 onMounted 里写 document.title，
 * 离开页面时没人还原，从「用户协议」跳到别的页面标签标题就一直是「用户协议 - 站点名」。
 * 现在只有两个入口，每次都会重算一次，谁先到位都不影响：
 *   - applyRouteTitle(route)：路由切换后调用（router.afterEach）
 *   - setSiteName(name)：站点配置加载 / 变化后调用（applySiteAssets）
 */

// index.html 里写死的默认标题：两段都算不出来时兜底用它，保证标题不会被清空
const fallbackTitle = typeof document !== 'undefined' ? document.title : ''

let siteName = ''
let routeTitle = ''

function sync() {
  if (typeof document === 'undefined') return

  const parts = [routeTitle, siteName].filter(Boolean)
  document.title = parts.length ? parts.join(' - ') : fallbackTitle
}

/**
 * 站点名（站点配置的 title），配置为空时传空串即可
 */
export function setSiteName(name) {
  const next = String(name || '').trim()
  if (next === siteName) return

  siteName = next
  sync()
}

/**
 * 按路由 `meta.title` 设置页面标题；没有 meta.title 的路由会回退成「只有站点名」
 *
 * @param {import('vue-router').RouteLocationNormalized} route
 */
export function applyRouteTitle(route) {
  const meta = (route && route.meta) || {}
  const title = typeof meta.title === 'function' ? meta.title(route) : meta.title

  routeTitle = String(title || '').trim()
  sync()
}
