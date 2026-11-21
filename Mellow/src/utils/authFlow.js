/**
 * 登录 / 退出流程里的跨模块标记
 *
 * 背景：退出登录与「401 全局登出」是两条独立路径 ——
 *   - 用户主动退出：stores/user.js#logout() → 清登录态 → 跳回首页 /
 *   - 拦截器 401：api/request.js#handleLogout() → 清登录态 → 当前页需要登录时跳登录页
 *
 * 在 /admin/*、/user/* 这类 meta.auth 页面点「退出登录」时，页面上仍在飞的后台接口
 * 会紧接着返回 401，拦截器那条路径会把路由抢到 /auth/login，用户就停在登录页、
 * 而不是回到首页（表现为「退出登录后不会自动跳转到 /」）。
 *
 * 这个标记在用户主动退出期间置位，让拦截器那条路径「让路」——只清登录态，
 * 不做跳转，把路由交给 logout() 独占处理。
 *
 * 放在独立模块（而不是 store 或 request.js 里）是为了避开
 * stores/user.js ↔ api/request.js 的循环依赖。
 */
export const logoutState = {
  /** 是否正在执行「用户主动退出登录」 */
  active: false
}

export default logoutState
