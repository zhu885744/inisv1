import axios from 'axios'
import utils from '{src}/utils/utils'
import cache from '{src}/utils/cache'

// 设置超时
axios.defaults.timeout = 60 * 1000
axios.defaults.baseURL = !utils.is.empty(globalThis?.inis?.api?.uri) ? globalThis?.inis?.api?.uri : ''

// ==================== 401 统一拦截 ====================
// 由页面自行处理登录态失败的接口，拦截器不干预（避免覆盖登录错误提示或重复跳转）
const skipAuthRedirectUrls = [
    '/api/comm/login',
    '/api/comm/register',
    '/api/comm/check-token',
    '/api/comm/logout',
]

// 无登录态的页面路由，已在这些页面时不重复跳转
const authFreeRoutes = ['/', '/register', '/reset-password']

// 并发请求同时 401 时只跳转一次
let authRedirecting = false

// 清除本地登录态并跳转登录页
const handleUnauthorized = () => {
    if (authRedirecting) return
    authRedirecting = true

    // 清除 token cookie、用户信息缓存与本地会话存储
    utils.clear.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN')
    cache.del('user-info')
    try {
        window.sessionStorage?.clear()
    } catch (e) { /* 忽略存储不可用的异常 */ }

    const currentPath = (window.location.hash || '').replace(/^#/, '') || '/'
    if (authFreeRoutes.includes(currentPath)) {
        authRedirecting = false
        return
    }

    // 整页跳转至登录页（彻底重置 SPA 内存状态，避免残留脏状态）
    window.location.href = '/'
}


// 请求拦截
//   所有的网络请求都会先走这个方法
axios.interceptors.request.use(
    config => {
        if (!utils.is.empty(globalThis?.inis?.api?.key)) {
            config.headers['i-api-key'] = globalThis?.inis?.api?.key
        }
        let TOKEN_NAME = !utils.is.empty(globalThis?.inis?.token_name) ? globalThis?.inis?.token_name : 'INIS_LOGIN_TOKEN'
        if (utils.has.cookie(TOKEN_NAME)) {
            let token = utils.get.cookie(TOKEN_NAME)
            if (!utils.is.empty(token)) {
                config.headers.Authorization = token
            }
        }
        return config
    },
    error  => Promise.reject(error)
)

// 响应拦截
//   所有的网络请求返回数据之后都会先执行这个方法
axios.interceptors.response.use(
    response => {
        // 401 = token 失效（过期/签名无效等），自动清理本地登录态并跳转登录页，
        // 避免前端携带失效 token 持续请求导致所有接口报错
        if (response?.data?.code === 401) {
            const url = response?.config?.url || ''
            if (!skipAuthRedirectUrls.some(item => url.includes(item))) {
                handleUnauthorized()
            }
        }
        return response.data
    },
    error => Promise.reject(error)
)

export default {
    // all
    all: async array => await axios.all(array),

    // GET请求
    get: async (url, params = {}, config = {}) => await axios.get(url, { params, ...config }),

    // DELETE请求
    del: async (url, params = {}, config = {}) => await axios.delete(url, { params, ...config }),

    // PUT请求
    put: async (url, data = {}, config = {}) => await axios.put(url, data, config),

    // POST请求
    post: async (url, data = {}, config = {}) => await axios.post(url, data, config),
}