import axios from 'axios'
import { toast } from '@/utils/toast'
import { useUserStore } from '@/stores/user'
import router from '@/router'
import { cache } from '@/utils/cache'
import { getCookie } from '@/utils/cookie'
import { showAuthDialog, clearLocalAuthData } from '@/utils/authDialog'
import { logoutState } from '@/utils/authFlow'
import { requestSliderCaptcha } from '@/utils/captchaGate'
import { API_BASE_URL, API_ROOT_URL } from '@/utils/runtimeConfig'

// 与 stores/user.js、Login.vue、Register.vue 保持一致的 token cookie 名
export const TOKEN_NAME = 'INIS_LOGIN_TOKEN'

// 约定：token 同时存于 cookie（INIS_LOGIN_TOKEN）和 localStorage（mellow_token）两处，
// 便于跨域/代理环境下稳定读取。优先读 cookie，回退读 localStorage。
function readToken() {
  const fromCookie = getCookie(TOKEN_NAME)
  if (fromCookie) return fromCookie
  try {
    const fromLs = localStorage.getItem('mellow_token')
    if (fromLs) return fromLs
  } catch (e) {
    // localStorage 不可用时忽略
  }
  return ''
}

// baseURL 由 @/utils/runtimeConfig 统一解析：
// 运行时配置（public/runtime-config.js）> 打包时 .env 的 VITE_API_URI > 同源 '/api'
const baseURL = API_BASE_URL

const service = axios.create({
  baseURL,
  timeout: 30000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json'
  }
})

// dev 类接口（如 /dev/info/time、/dev/info/version）不走 /api 前缀，单独导出实例供调用
export const devService = axios.create({
  baseURL: API_ROOT_URL,
  timeout: 10000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json'
  }
})

// dev 接口的响应体结构与 /api 相同（{ code, msg, data }，见 app/dev/controller/base.go 的 json()），
// 这里同样剥离 axios 包装，调用方可直接解构 { code, data }。
// 注意：漏配这个拦截器会让所有 dev 接口的返回值变成 axios response，解构出来全是 undefined
// （表现为「服务器时间/版本号静默取不到、回退到本地值」）。
devService.interceptors.response.use(
  (res) => res.data,
  (err) => {
    if (err.code === 'ERR_CANCELED') return Promise.reject(err)
    return Promise.reject({
      code: err.response?.status || 0,
      msg: err.response?.data?.msg || err.message || '请求失败'
    })
  }
)

// INIS 鉴权说明（参考 Cardify-inis 实现，实测 /api/comm/check-token 返回 "Authorization 不能为空！"）：
// 后端校验的是请求头 Authorization: <token>（裸 JWT，不要加 "Bearer " 前缀）。
// 登录/注册等匿名接口不带 token；check-token 等鉴权接口需要带 token（由拦截器统一注入）。
// 不需要带 token 的接口：登录 / 验证码登录 / 注册 / 校验登录态 / 退出 / 找回密码
//
// 这些接口**绝对不能**附带残留的旧 token：一旦带上失效 token，后端会先按「token 无效」
// 把请求拦成 401（登录接口自己都进不去），表现就是「输入账号密码点登录毫无反应」。
const AUTH_ENDPOINTS = [
  '/comm/login',
  '/comm/sign-code',
  '/comm/register',
  '/comm/check-token',
  '/comm/logout',
  '/comm/reset-password'
]

const isAuthEndpoint = (url = '') => AUTH_ENDPOINTS.some((item) => String(url).includes(item))

// 后台管理界面标记：后端（controller/scope.go）只有在收到该标记时才对管理员豁免「可见范围」，
// 管理员在前台浏览时同样会看到「登录可见 / 付费可见」的真实效果，便于验证付费墙。
// 复用 CORS 白名单里已有的 X-Requested-With，避免新增自定义头导致跨域预检被拒。
const ADMIN_PANEL_FLAG = 'inis-admin'
const isAdminPanelPage = () => /(^|\/)admin(\/|$)/.test(window.location.pathname)

service.interceptors.request.use((config) => {
  // 已确认「未安装」：/api 请求一律不再发出（后端也全会拦成 412，发出去只是噪音）
  if (installGate) {
    return Promise.reject({ code: 'ERR_CANCELED', message: '安装引导未完成' })
  }

  const token = readToken()
  if (token && !isAuthEndpoint(config.url)) {
    config.headers.Authorization = token
  }
  // 后台管理页发出的请求：带标记，后端据此对管理员豁免可见范围
  if (isAdminPanelPage()) {
    config.headers['X-Requested-With'] = ADMIN_PANEL_FLAG
  }
  // 上传文件：手动设置的 "multipart/form-data" 不带 boundary，Go 后端无法解析
  // （报错：request Content-Type isn't multipart/form-data）。
  // 这里统一移除该请求头，改由浏览器自动生成带 boundary 的 multipart/form-data。
  if (config.data instanceof FormData) {
    const headers = config.headers
    if (headers) {
      if (typeof headers.delete === 'function') headers.delete('Content-Type')
      else delete headers['Content-Type']
    }
  }
  return config
})

// ---------- 安装引导 ----------
// 未完成安装时后端会把所有 /api 请求拦下（{code:412, gate:'install'}）。
// 这种情况不是登录失效，不能走 401 那套弹窗 —— 否则未安装的站点每发一个请求就弹一次
// 「登录状态异常」（外壳挂载时会连发好几个，看起来就是「一直弹」）。
// 处理方式：静默置标志 + 跳转安装向导，并让后续 /api 请求直接取消（连发都不再发出）。
let installGate = false

/** 是否已进入「安装引导未完成」状态（供布局等外部逻辑判断） */
export const isInstallGate = () => installGate

const enterInstallGate = () => {
  if (installGate) return
  installGate = true
  if (router.currentRoute.value?.name !== 'install') {
    // 跳转失败（如与当前导航冲突）不影响后续判断，忽略即可
    Promise.resolve(router.replace({ name: 'install' })).catch(() => {})
  }
}

// 是否正在登出中，防止重复触发
let isLoggingOut = false

const handleLogout = () => {
  if (isLoggingOut) return
  isLoggingOut = true
  const userStore = useUserStore()
  userStore.clear()
  // 用户主动退出登录期间：路由由 stores/user.js#logout() 独占（它负责跳回首页），
  // 这里只清登录态、不抢跳转 —— 否则后台这类 meta.auth 页面里退出时，
  // 页面上仍在飞的接口返回的 401 会把路由抢到登录页，导致「退出后回不到首页」
  if (!logoutState.active) {
    // 仅当前页面需要登录时才跳转
    const current = router.currentRoute.value
    if (current.meta?.auth) {
      router.push({ name: 'login', query: { redirect: current.fullPath } })
    }
  }
  setTimeout(() => { isLoggingOut = false }, 2000)
}

// 401 弹窗的确认动作：彻底清除本地登录信息（全部 cookie + localStorage + sessionStorage）并跳转登录页
const confirmClearAndRelogin = () => {
  // 先重置内存中的用户状态（store 的内存值不会因存储清空而自动重置）
  useUserStore().clear()
  // 再彻底清空当前域的 cookie 与本地/会话存储
  clearLocalAuthData()
  const current = router.currentRoute.value
  router.push({ name: 'login', query: { redirect: current.fullPath } })
}

// 401 统一处理：自动清理登录态 + 弹窗提示（用户点击可彻底清除并重新登录）
const handleUnauthorized = (msg) => {
  handleLogout()
  showAuthDialog(msg, confirmClearAndRelogin)
}

// ---------- 「没登录」与「登录态异常」的区分 ----------
// 两者都是 401/412，但含义完全不同：
//   - 没登录：匿名请求了需要登录的接口（刚装完还没登录就打开后台、旧 token 指向的用户已不存在、
//     /api/comm/check-token 没带 Authorization…）——这属于正常状态，静默清理即可；
//   - 登录态异常：带了 token 但服务端判为无效/过期/签名不符——才需要提示用户重新登录。
// 以前两者都会弹「登录状态异常（需要您自行手动清除）」的弹窗，装完站第一次打开就中招。
const GUEST_MESSAGES = ['请先登录！', 'Authorization 不能为空！', '禁止非法操作！']

/** 请求是否带了登录凭证（AxiosHeaders 与普通对象都兼容） */
function sentCredential(config) {
  const headers = config?.headers
  if (!headers) return false
  if (typeof headers.get === 'function') return Boolean(headers.get('Authorization'))
  return Boolean(headers.Authorization)
}

/** 是否是「没登录」（而不是登录态异常） */
function isGuestAuth(data, config) {
  // 后端在「未登录」分支会带 auth:'guest'（见 app/api/middleware/rule.go）
  if (data?.auth === 'guest') return true
  // 请求压根没带 Authorization，服务端自然只能说你没登录
  if (!sentCredential(config)) return true
  // 其余按后端文案兜底（各控制器自己返回的 401「请先登录！」也走这里）
  const message = String(data?.msg || '')
  return GUEST_MESSAGES.some((item) => message.includes(item))
}

/**
 * 是否是「后台管理会话失效」（而不是前台登录态异常）
 *
 * 后端约定 auth:'admin' 放在响应体顶层（middleware/rule.go）。
 * 这里兼容 data.auth：控制器若把 auth 塞进 data（base.json 的固定结构），
 * 也要能被识别 —— 否则会被当成普通 401，把用户整体登出。
 */
function isAdminAuthRequired(data) {
  return data?.auth === 'admin' || data?.data?.auth === 'admin'
}

service.interceptors.response.use(
  async (res) => {
    const data = res.data
    if (res.config.responseType === 'blob') return res

    if (data && typeof data === 'object' && 'code' in data) {
      if (data.code === 200 || data.code === 201) {
        return data
      }
      // 204 无数据（如 config 查询不存在的 key），静默返回
      if (data.code === 204) {
        return { code: 204, data: null, msg: data.msg }
      }
      // 发送验证码防刷：后端要求先过滑块（428 + need_captcha）
      // 统一在这里弹滑块，通过后自动重放原请求 —— 业务调用方完全无感知。
      // __captchaRetried 用于防止「重放后仍被判频繁」时无限弹窗
      if (data.code === 428 && data.data?.need_captcha && !res.config?.__captchaRetried) {
        return retryWithSlider(res)
      }
      // 后台管理会话失效（scope=admin，见 app/api/middleware/rule.go）：
      // 后台页面不共用前台登录态，这里弹密码框重新换一条管理会话，通过后重放原请求。
      // 不走下面 401 那套「登出 + 弹窗」：前台登录态仍然有效，不该被一起清掉。
      if (data.code === 401 && isAdminAuthRequired(data)) {
        // 已经验证过一次还是 401：多半是 cookie 没存下来（浏览器禁用 / 跨域被拦），
        // 再弹一次密码框也没用，直接说明原因
        if (res.config?.__adminRetried) {
          toast.error('后台验证未生效，请确认浏览器允许本站保存 cookie 后重试')
          return Promise.reject({ ...data, code: data.code })
        }
        const { unlockAdminSession } = await import('@/utils/adminSession')
        const unlocked = await unlockAdminSession(data.msg)
        if (!unlocked) {
          return Promise.reject({ ...data, code: data.code })
        }
        return service.request({ ...res.config, __adminRetried: true })
      }
      // 401/412 未登录或 token 失效：清理本地状态并弹窗提示
      // 若调用方传了 skipAuthLogout（如 check-token 需要拿到原始码做本地状态清理），
      // 则不触发全局登出，原样返回给调用方处理（参考 Cardify-inis 实现）
      if (data.code === 401 || data.code === 412) {
        // 安装引导未完成（后端带 gate:'install'）：静默去安装向导，不是登录问题
        if (data.gate === 'install') {
          enterInstallGate()
          return Promise.reject({ ...data, code: data.code })
        }
        // 用户主动退出登录期间：页面上尚未结束的请求必然拿到 401/412，
        // 这里只静默清理，既不弹「登录态异常」框，也不抢路由跳转（由 logout() 统一回首页）
        if (logoutState.active) {
          useUserStore().clear()
          return Promise.reject({ ...data, code: data.code })
        }
        // 没登录（不是登录态异常）：静默清掉本地登录态即可，别弹「需要您自行手动清除」的框
        if (isGuestAuth(data, res.config)) {
          if (!res.config?.skipAuthLogout) {
            handleLogout()
          }
          return Promise.reject({ ...data, code: data.code })
        }
        if (!res.config?.skipAuthLogout) {
          handleUnauthorized(data.msg)
        }
        return Promise.reject({ ...data, code: data.code })
      }
      // config.silent 为 true 时不弹全局提示，交调用方自行处理（如列表页局部错误态）
      if (res.config?.silent) {
        return Promise.reject(data)
      }
      if (data.code === 403) {
        toast.error(data.msg || '没有权限')
        return Promise.reject(data)
      }
      toast.error(data.msg || '请求失败')
      return Promise.reject(data)
    }
    return data
  },
  (err) => {
    // 已主动取消的请求（如组件卸载/切换筛选）不提示
    if (err.code === 'ERR_CANCELED') {
      return Promise.reject(err)
    }
    if (err.config?.silent) {
      return Promise.reject(err)
    }
    if (err.response) {
      if (err.response.status === 401) {
        handleUnauthorized(err.response.data?.msg)
      } else {
        toast.error(err.response.data?.msg || `请求异常 (${err.response.status})`)
      }
    } else if (err.request) {
      toast.error('网络异常，请检查连接')
    } else {
      toast.error(err.message || '请求失败')
    }
    return Promise.reject(err)
  }
)

/**
 * 滑块重试：把「需要滑块」的 428 响应换成「弹滑块 → 重放原请求」
 *
 * 失败 / 用户取消时按普通错误 reject，调用方拿到的仍是后端原始 body
 * （取消是用户主动行为，不再弹全局错误提示，避免与滑块弹窗重复打扰）。
 */
async function retryWithSlider(res) {
  const body = res.data
  const ticket = await requestSliderCaptcha(body?.data?.captcha)

  if (!ticket?.token) {
    return Promise.reject({ ...body, code: body.code })
  }

  try {
    return await service.request(buildSliderReplayConfig(res.config, ticket))
  } catch (err) {
    toast.error(err?.msg || '滑块验证未通过，请重试')
    return Promise.reject({ ...body, code: body.code })
  }
}

/**
 * 重放请求的参数：票据按 HTTP 方法放进 query（GET/DELETE）或 body（POST/PUT）
 * 说明：axios 的 transformRequest 可能已把 config.data 序列化成 JSON 字符串，这里先还原再合并
 */
function buildSliderReplayConfig(config, ticket) {
  const next = { ...config, __captchaRetried: true }
  const method = String(config.method || 'get').toLowerCase()

  if (['get', 'delete', 'head'].includes(method)) {
    next.params = { ...(config.params || {}), captcha_token: ticket.token, captcha_value: ticket.value }
    return next
  }

  let body = config.data
  if (typeof body === 'string') {
    try {
      body = JSON.parse(body)
    } catch {
      body = {}
    }
  }

  // 上传类请求（FormData）不能整体替换，只能追加字段
  if (typeof FormData !== 'undefined' && body instanceof FormData) {
    body.append('captcha_token', ticket.token)
    body.append('captcha_value', String(ticket.value))
    next.data = body
    return next
  }

  next.data = {
    ...(body || {}),
    captcha_token: ticket.token,
    captcha_value: ticket.value
  }
  return next
}

export default service
export { cache }

/**
 * 并发去重：同一时刻发起的「完全相同」的 GET 请求复用同一个 Promise，
 * 避免侧栏/布局中多个组件同时拉取同一接口造成重复请求。
 * 仅对进行中的请求去重（请求结束后即释放），因此不会出现数据陈旧问题。
 */
const inflight = new Map()

/** 稳定序列化查询参数（对象按 key 排序，保证同等参数生成同一 key） */
function serializeParams(params) {
  if (!params) return ''
  return Object.keys(params)
    .sort()
    .map((k) => {
      const v = params[k]
      const val = v !== null && typeof v === 'object' ? JSON.stringify(v) : String(v ?? '')
      return `${k}=${val}`
    })
    .join('&')
}

function dedupeRequest(key, factory) {
  const running = inflight.get(key)
  if (running) return running
  const promise = factory().finally(() => {
    // 仅当仍是自己时删除，避免覆盖后到的同 key 请求
    if (inflight.get(key) === promise) inflight.delete(key)
  })
  inflight.set(key, promise)
  return promise
}

/**
 * INIS 控制器通用调用方法
 * @param {string} controller 控制器名,如 'article'
 * @param {string} method 方法名,如 'all'
 * @param {object} options
 * @param {string} options.method HTTP 方法,默认 GET
 * @param {object} options.params GET 参数（where 可直接传对象）
 * @param {object} options.data POST/PUT body
 * @param {object} options.config 额外 axios 配置（silent: 静默错误；dedupe: false 关闭去重）
 */
export const call = (controller, method, options = {}) => {
  const { method: httpMethod = 'GET', params, data, config } = options
  const url = `/${controller}/${method}`
  const requestConfig = {
    url,
    method: httpMethod,
    params: httpMethod === 'GET' || httpMethod === 'DELETE' ? params : undefined,
    data: httpMethod !== 'GET' && httpMethod !== 'DELETE' ? data : undefined,
    ...config
  }
  // GET 且未显式关闭去重时启用
  if (httpMethod === 'GET' && requestConfig.dedupe !== false) {
    const key = `${baseURL}${url}?${serializeParams(requestConfig.params)}`
    return dedupeRequest(key, () => service.request(requestConfig))
  }
  return service.request(requestConfig)
}
