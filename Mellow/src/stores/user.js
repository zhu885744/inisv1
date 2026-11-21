import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { checkToken, logout as apiLogout } from '@/api/comm'
import { cache } from '@/utils/cache'
import { toast } from '@/utils/toast'
import { getCookie, setCookie, clearCookie } from '@/utils/cookie'
import { TOKEN_NAME } from '@/api/request'
import { useSiteStore } from '@/stores/site'
import { useNotificationStore } from '@/stores/notification'
import { logoutState } from '@/utils/authFlow'
import router from '@/router'

const TOKEN_KEY = 'blog_user'
const TOKEN_VALID = 'blog_token_valid'
const USER_CACHE = 'user-info'
const TOKEN_LS = 'mellow_token' // token 的 localStorage 兜底（与 request.js readToken 对应）

// 统一持久化 token：cookie（INIS_LOGIN_TOKEN，供同域兜底）+ localStorage（跨域/代理环境兜底）
function persistToken(token, valid) {
  if (!token) return
  try {
    setCookie(TOKEN_NAME, token, valid)
  } catch {}
  try {
    localStorage.setItem(TOKEN_LS, token)
  } catch {}
}

function clearToken() {
  try {
    clearCookie(TOKEN_NAME)
  } catch {}
  try {
    localStorage.removeItem(TOKEN_LS)
  } catch {}
}

// 并发保护
let checkingToken = false
let checkTokenPromise = null

// 最近一次向后端确认登录态的时间（ms）：ensureLogin 在启动链路上会被调用两次
// （main.js + MainLayout），用它做短时去重，避免每次打开页面发两遍 check-token
let lastCheckedAt = 0
const RECHECK_INTERVAL = 5000

// 规范化用户对象：确保 json 字段是对象（后端可能返回字符串）
function normalizeUser(u) {
  if (!u || typeof u !== 'object') return u || {}
  const user = { ...u }
  if (user.json && typeof user.json === 'string') {
    try {
      user.json = JSON.parse(user.json)
    } catch {
      user.json = {}
    }
  }
  if (!user.json) user.json = {}
  return user
}

export const useUserStore = defineStore('user', () => {
  const user = ref(null)
  const tokenValid = ref(0)
  // 兼容参考主题的 login 结构：finish 表示登录态校验是否完成
  const login = ref({ finish: false, user: {} })
  // 站点信息（从 config/one?key=Mellow_functions 获取）
  const siteInfo = ref({})

  const isLogged = computed(() => !!user.value)
  const isLoggedIn = computed(() => login.value.finish && Object.keys(login.value.user || {}).length > 0)

  // ===== 基础持久化 =====
  function setUser(u, valid = 1296000) {
    const nu = normalizeUser(u)
    user.value = nu
    tokenValid.value = valid
    login.value = { finish: true, user: nu || {} }
    save()
    cache.set(USER_CACHE, nu || {}, Math.ceil(valid / 60))
  }

  function clear() {
    user.value = null
    tokenValid.value = 0
    login.value = { finish: true, user: {} }
    save()
    cache.del(USER_CACHE)
    // 必须同步清除 token 本体（cookie + localStorage）：
    // 否则 request.js 的 readToken() 会再次读出失效 token 并附到后续所有请求上，
    // 造成 401 → clear → 再 401 的死循环（表现为所有接口持续报错、页面无数据）
    clearToken()
  }

  function save() {
    try {
      localStorage.setItem(TOKEN_KEY, JSON.stringify(user.value || null))
      localStorage.setItem(TOKEN_VALID, String(tokenValid.value || 0))
    } catch {}
  }

  function restore() {
    try {
      const u = localStorage.getItem(TOKEN_KEY)
      const v = localStorage.getItem(TOKEN_VALID)
      user.value = u && u !== 'null' ? JSON.parse(u) : null
      tokenValid.value = v ? parseInt(v) : 0
      if (user.value) {
        login.value = { finish: true, user: user.value }
      }
    } catch {}
  }

  // ===== 登录态校验（参考主题 checkToken）=====
  async function checkLoginState() {
    if (checkingToken && checkTokenPromise) {
      await checkTokenPromise
      return login.value
    }
    checkingToken = true
    // 记录「最近一次真正问过后端的时间」，供 ensureLogin 做短时去重：
    // 启动链路里 main.js 和 MainLayout 都会调 ensureLogin，不该各发一次请求
    lastCheckedAt = Date.now()
    checkTokenPromise = (async () => {
      login.value.finish = false
      try {
        const cachedUser = cache.get(USER_CACHE)
        if (cachedUser) {
          user.value = cachedUser
          login.value = { finish: true, user: cachedUser }
        }
        const res = await checkToken(false)
        if (res.code === 200) {
          const u = normalizeUser(res.data?.user || res.data || {})
          user.value = u
          login.value = { finish: true, user: u }
          const valid = Number(res.data?.valid_time) > 0 ? Number(res.data.valid_time) : 2 * 60 * 60
          tokenValid.value = valid
          // 同步 token 到 cookie + localStorage（刷新/续期）
          if (res.data?.token) {
            persistToken(res.data.token, valid)
          }
          cache.set(USER_CACHE, u, Math.ceil(valid / 60))
          save()
        } else if (res.code === 401 || res.code === 412) {
          clear()
        } else {
          login.value.finish = true
        }
      } catch {
        if (cache.get(USER_CACHE)) {
          login.value.finish = true
        }
      } finally {
        checkingToken = false
        checkTokenPromise = null
      }
    })()
    await checkTokenPromise
    return login.value
  }

  // 应用启动时校验（仅当存在 token 或缓存用户时才发请求）
  async function ensureLogin() {
    const hasToken = !!getCookie(TOKEN_NAME)
    const hasCachedUser = user.value || cache.get(USER_CACHE)

    if (!hasToken && !hasCachedUser) {
      login.value.finish = true
      return login.value
    }

    if (hasToken) {
      // 有 token 就必须回后端确认一次。
      //
      // 本地缓存的 user（localStorage 的 blog_user / user-info）只是「先让界面不闪」的快照，
      // 可能是很久以前的：权限组、封禁状态、头像框的 json.frame_hole 等都可能已在后台改过。
      // 旧逻辑写成 `if (!login.finish || !hasCachedUser)`，而 restore() 已经把两者都置好了，
      // 于是刷新页面完全跳过请求 —— 后台改了装扮，前台一直不生效，最长要等缓存过期（可达数天）。
      //
      // 5 秒内重复调用直接跳过：启动链路上 main.js 与 MainLayout 各会调一次，没必要发两遍。
      if (Date.now() - lastCheckedAt > RECHECK_INTERVAL) {
        await checkLoginState()
      }
      return login.value
    }

    // 没有 token 但有缓存用户：沿用上次的资料，不发请求
    login.value.finish = true
    return login.value
  }

  async function verifyToken(silent = false) {
    if (!user.value) return false
    try {
      const res = await checkToken(false)
      if (res.code === 200) {
        const u = normalizeUser(res.data?.user || res.data || {})
        user.value = u
        login.value = { finish: true, user: u }
        if (res.data?.valid_time) tokenValid.value = res.data.valid_time
        if (res.data?.token) {
          persistToken(res.data.token, Number(res.data.valid_time) > 0 ? Number(res.data.valid_time) : 2 * 60 * 60)
        }
        save()
        return true
      }
      clear()
      if (!silent) toast.error('登录已过期，请重新登录')
      return false
    } catch {
      return false
    }
  }

  // ===== 站点信息 =====
  // 统一委托给 site store（并发去重 + 缓存），避免与布局/侧栏重复请求同一配置
  async function fetchSiteInfo(force = false) {
    const siteStore = useSiteStore()
    const info = await siteStore.load(force)
    if (info && typeof info === 'object') {
      siteInfo.value = info
    }
    return siteInfo.value
  }

  // 防止重复触发退出（按钮连点 / 多处入口同时调用）
  let loggingOut = false

  /**
   * 回到站点首页，保证「退出登录 → 首页」这个结果稳定：
   * 1. 先按前端路由跳（保留 SPA 体验，不整页刷新）；
   * 2. 跳转未落地（被并发跳转改道 / 守卫抛错）则再试一次；
   * 3. 仍然没回到首页时兜底整页跳转 —— 退出登录必须回首页，不能停在后台/用户中心。
   */
  async function gotoHome() {
    try {
      await router.replace('/')
    } catch {}
    if (router.currentRoute.value.path === '/') return

    try {
      await router.replace('/')
    } catch {}
    if (router.currentRoute.value.path === '/') return

    window.location.assign('/')
  }

  async function logout() {
    if (loggingOut) return
    loggingOut = true
    // 告诉请求拦截器：本次退出期间的路由跳转由这里独占（见 utils/authFlow.js）
    logoutState.active = true
    try {
      // 服务端登出「只发不等」：接口慢或异常都不该拖住本地退出流程
      // （接口本身是幂等的，且前端清理 cookie/token 与它无关）
      apiLogout().catch(() => {})

      clear()
      // 清除 token（cookie + localStorage）
      clearToken()
      toast.success('已退出登录')

      // 停掉消息轮询并清掉未读数：侧栏（MainLayout）才会在登出时停它，
      // 从用户中心退出时没有侧栏，定时器会残留并持续发请求；
      // 不清零的话，登出后底栏 / 侧栏的未读角标会一直挂着上次的数字
      try {
        const notif = useNotificationStore()
        notif.stopPolling()
        notif.clearCount()
      } catch {}

      await gotoHome()
    } finally {
      loggingOut = false
      // 留一点余量窗口：尚未结束的请求可能还会返回 401，继续让路给「回首页」
      setTimeout(() => {
        logoutState.active = false
      }, 1000)
    }
  }

  return {
    user,
    tokenValid,
    login,
    siteInfo,
    isLogged,
    isLoggedIn,
    setUser,
    clear,
    save,
    restore,
    verifyToken,
    checkLoginState,
    ensureLogin,
    fetchSiteInfo,
    logout,
    persistToken,
    clearToken
  }
})
