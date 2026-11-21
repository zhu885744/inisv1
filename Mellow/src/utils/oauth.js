import { getOauthConfig, oauthBind } from '@/api/oauth'
import { toast } from '@/utils/toast'
import { useUserStore } from '@/stores/user'

/**
 * 第三方登录的前端流程工具
 *
 * 流程（服务端不校验 state，由前端自己校验，这是 SPA 里的标准做法）：
 *   1. 用户点按钮 → startOauth(platform) 生成 state 存 sessionStorage → 跳第三方授权页；
 *   2. 第三方带 code/state 回调到 `/auth/oauth`（回调地址必须与第三方后台登记的一致，
 *      所以这里不给 redirect_uri 附加任何参数）；
 *   3. 回调页校验 state → 调 /api/oauth/{platform} → 按返回结果收尾（登录 / 引导绑定 / 待审核）。
 *
 * 为什么 platform 不放在 redirect_uri 的 query：
 * QQ / Gitee 对 redirect_uri 是「全字符串精确匹配」，多一个 query 就会报 redirect_uri 不一致，
 * 因此三个平台共用同一个回调地址，平台信息由前端自己记住。
 *
 * 注意：本地存储不能作为唯一依据 —— 手机端（尤其从 QQ 应用内发起）回调页常常
 * 在新标签 / 内置浏览器里打开，sessionStorage 是空的，只靠它会误报「未找到发起平台」。
 * 所以 state 用「平台名 + 随机串」，回调页可以从 state 前缀还原平台（platformFromState）。
 */

// 平台图标（Bootstrap Icons v1.13+ 自带品牌图标）
const PLATFORM_ICONS = {
  qq: 'bi bi-tencent-qq',
  github: 'bi bi-github',
  gitee: 'bi bi-git'
}

/** 平台图标类名（未知平台给个通用登录图标兜底） */
export function platformIcon(platform) {
  return PLATFORM_ICONS[platform] || 'bi bi-box-arrow-in-right'
}

const STATE_KEY = 'mellow_oauth_state'
const PLATFORM_KEY = 'mellow_oauth_platform'
const MODE_KEY = 'mellow_oauth_mode'
const REDIRECT_KEY = 'mellow_oauth_redirect'
// 未绑定时后端下发的「先登录再绑定」票据：登录成功后由 finishPendingBind 消费
const TICKET_KEY = 'mellow_oauth_ticket'

// 已支持的第三方平台（与后端 oauth 平台一致）：state 前缀按此还原
const PLATFORMS = ['github', 'gitee', 'qq']

/** 从 state 前缀还原平台（回调页拿不到本地存储时的兜底） */
export function platformFromState(state) {
  const value = String(state || '').trim().toLowerCase()
  if (!value) return ''
  return PLATFORMS.find((item) => value.startsWith(item)) || ''
}

const store = {
  get(key) {
    try {
      return sessionStorage.getItem(key) || ''
    } catch {
      return ''
    }
  },
  set(key, value) {
    try {
      sessionStorage.setItem(key, value)
    } catch {
      /* 隐私模式下 sessionStorage 可能不可用，忽略 */
    }
  },
  del(key) {
    try {
      sessionStorage.removeItem(key)
    } catch {
      /* 忽略 */
    }
  }
}

// 配置缓存：登录页 / 注册页 / 账号安全页都会用到，避免重复请求
let configPromise = null

/**
 * 读取第三方登录配置
 * @param {boolean} force 强制刷新（后台刚改完配置时用）
 * @returns {Promise<{platforms: Array, autoRegister: boolean}>}
 */
export async function loadOauthConfig(force = false) {
  if (force || !configPromise) {
    configPromise = getOauthConfig()
      .then((res) => normalizeConfig(res?.data))
      .catch(() => ({ platforms: [], autoRegister: false }))
  }
  return configPromise
}

/** 只取「已开启且参数齐全」的平台（后端已经用 enable 标明） */
export async function enabledPlatforms(force = false) {
  const config = await loadOauthConfig(force)
  return config.platforms
}

// 归一化后端返回：既兼容 { platforms: [...] } 也兼容「平台字段平铺在顶层」的老结构
function normalizeConfig(data = {}) {
  const list = Array.isArray(data.platforms)
    ? data.platforms
    : Object.entries(data)
        .filter(([, value]) => value && typeof value === 'object')
        .map(([platform, value]) => ({ platform, ...value }))

  return {
    autoRegister: Number(data.auto_register) === 1,
    platforms: list
      .filter((item) => item && item.platform && Number(item.enable) === 1)
      .map((item) => ({
        platform: item.platform,
        name: item.name || item.platform,
        appId: item.app_id || '',
        redirect: item.redirect || '',
        authorize: item.authorize || '',
        scope: item.scope || ''
      }))
  }
}

/** 回调地址：优先用后台配置（必须与第三方登记的一致），没配则回退当前站点 */
export function callbackUrl(platform) {
  return platform?.redirect || `${window.location.origin}/auth/oauth`
}

/** 拼授权地址（各平台参数名略有差异，统一在这里处理） */
export function authorizeUrl(item) {
  const redirect = callbackUrl(item)
  const params = new URLSearchParams()

  if (item.platform === 'github') {
    params.set('client_id', item.appId)
    params.set('redirect_uri', redirect)
    params.set('scope', item.scope || 'read:user')
  } else if (item.platform === 'gitee') {
    params.set('client_id', item.appId)
    params.set('redirect_uri', redirect)
    params.set('response_type', 'code')
    params.set('scope', item.scope || 'user_info')
  } else {
    // QQ
    params.set('response_type', 'code')
    params.set('client_id', item.appId)
    params.set('redirect_uri', redirect)
    params.set('scope', item.scope || 'get_user_info')
  }

  params.set('state', store.get(STATE_KEY))
  return `${item.authorize}?${params.toString()}`
}

/**
 * 发起第三方登录 / 绑定
 * @param {object} item     loadOauthConfig() 里的平台项
 * @param {object} options  { mode: 'login' | 'bind', redirect: 成功后的落地页 }
 */
export function startOauth(item, options = {}) {
  if (!item || !item.platform || !item.authorize || !item.appId) {
    toast.error('该登录方式未配置，请联系管理员')
    return
  }

  // state：防 CSRF，回调时比对；前缀带上平台，回调页在本地存储缺失时也能还原
  // （全是字母数字，兼容各平台对 state 的透传）
  const state = `${item.platform}${Date.now().toString(36)}${Math.random().toString(36).slice(2, 10)}`
  store.set(STATE_KEY, state)
  store.set(PLATFORM_KEY, item.platform)
  store.set(MODE_KEY, options.mode || 'login')
  store.set(REDIRECT_KEY, options.redirect || '/')

  window.location.href = authorizeUrl(item)
}

/** 带走回调信息（校验完即焚，避免刷新重复提交 code） */
export function takeCallbackContext() {
  const context = {
    state: store.get(STATE_KEY),
    platform: store.get(PLATFORM_KEY),
    mode: store.get(MODE_KEY) || 'login',
    redirect: store.get(REDIRECT_KEY) || '/'
  }
  store.del(STATE_KEY)
  store.del(PLATFORM_KEY)
  store.del(MODE_KEY)
  store.del(REDIRECT_KEY)
  return context
}

/** 未绑定时把票据存起来，登录成功后自动完成绑定 */
export function saveBindTicket(ticket) {
  if (ticket) store.set(TICKET_KEY, ticket)
}

/** 清除待绑定票据（回调页已自行完成绑定/建号时调用，避免登录流程再试一次） */
export function clearBindTicket() {
  store.del(TICKET_KEY)
}

export function hasBindTicket() {
  return !!store.get(TICKET_KEY)
}

/**
 * 登录成功后消费待绑定票据：把这次第三方登录的账号绑到刚登录的账号上
 * @returns {Promise<boolean>} 是否真正完成了绑定
 */
export async function finishPendingBind() {
  const ticket = store.get(TICKET_KEY)
  if (!ticket) return false

  const userStore = useUserStore()
  if (!userStore.isLogged) return false

  store.del(TICKET_KEY)
  try {
    await oauthBind({ key: ticket })
    toast.success('第三方账号已绑定')
    return true
  } catch {
    // 票据过期 / 已被别的账号绑定：不打断登录流程，只提示
    toast.warning('第三方账号绑定失败，可在「账号安全」中重新绑定')
    return false
  }
}
