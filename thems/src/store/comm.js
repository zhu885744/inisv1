// 通用状态管理
import { defineStore } from 'pinia'
import { cache } from '@/utils/network'
import utils from '@/utils/utils'
import { request as axios } from '@/utils/network'
import { route, applyRouteTitle } from '@/utils/app'
import { STORAGE_KEYS } from '@/constants'

// 定义Token名称（和后端配置一致）
const TOKEN_NAME = globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN'

// 防止并发调用的标志位和等待队列
let checkingToken = false
let checkTokenPromise = null
let fetchingSiteInfo = false
let fetchSiteInfoPromise = null

/**
 * 校验token（完善版）
 * @param {Object} state - 状态对象
 * @returns {Promise<void>}
 */
const checkToken = async (state = {}) => {
    // 防止并发调用：已有进行中的校验时复用同一个 Promise，
    // 让并发调用方等待同一次校验结果，而不是各自再发一次请求
    if (checkingToken && checkTokenPromise) {
        await checkTokenPromise
        return
    }
    
    checkingToken = true
    
    // 保存当前Promise供后续调用等待
    checkTokenPromise = (async () => {
    
    const cacheName = 'user-info'
    state.login.finish = false // 先置为未完成，避免缓存欺骗
    try {
        // 1. 优先从缓存取用户信息（提升体验）
        const cacheUser = cache.get(cacheName)
        if (cacheUser) {
            state.login.user = cacheUser
            state.login.finish = true // 先设置为完成，提升用户体验
        }

        // 2. 强制校验后端Token有效性（异步）
        // skipAuthLogout: 让 401 原样返回，由下面的 switch 统一处理本地登录态清理，
        // 避免 network.js 的全局 401 拦截直接 reject，导致 catch 分支误判为“保持登录”
        const { code, msg, data } = await axios.post('/api/comm/check-token', {}, {
            // 携带Cookie（适配后端从Cookie取Token）
            withCredentials: true,
            // 请求头携带Token（适配后端从Authorization取Token）
            headers: {
                Authorization: utils.get.cookie(TOKEN_NAME) || ''
            },
            skipAuthLogout: true
        })

        switch (code) {
            case 200: {
                // 校验成功 → 更新状态和缓存
                state.login.user = data.user
                state.login.finish = true
                // cache.set 第三个参数单位为「分钟」；优先使用后端返回的 token 剩余有效期
                const validSeconds = Number(data.valid_time) > 0 ? Number(data.valid_time) : 2 * 60 * 60
                cache.set(cacheName, data.user, Math.ceil(validSeconds / 60))
                break;
            }
            case 401: // Token过期/无效
            case 412: // Token格式错误
                // token 已失效，仅清除本地登录状态，不再发起 logout 请求（避免失效 token 触发 401/405）
                clearLogin(state)
                break;
            default:
                console.error('Token校验失败：', msg)
                // 不登出，保持缓存中的用户信息
                state.login.finish = true
        }
        } catch (err) {
        // 网络错误/接口异常 → 保持缓存中的用户信息
        console.error('Token校验失败：', err)
        // 不登出，保持缓存中的用户信息
        if (cache.get('user-info')) {
            state.login.finish = true
        }
    } finally {
        // 重置标志位
        checkingToken = false
        checkTokenPromise = null
    }
    })()
    
    // 等待校验完成
    await checkTokenPromise
}

/**
 * 仅清除本地登录状态（不发起网络请求）
 * @param {Object} state - 状态对象
 */
const clearLogin = (state = {}) => {
    state.login.user = {}
    state.login.finish = false
    cache.del('user-info')
    utils.clear.cookie(TOKEN_NAME)
}

/**
 * 登出（完善版）
 * @param {Object} state - 状态对象
 * @param {string|null} path - 跳转路径
 * @returns {Promise<void>}
 */
const logout = async (state = {}, path = null) => {
    // 1. 清除前端状态
    clearLogin(state)

    // 2. 调用后端登出接口（兼容失败场景）
    try {
        await axios.post('/api/comm/logout', {}, { withCredentials: true })
    } catch (err) {
        console.error('登出接口调用失败：', err)
    }

    // 3. 跳转指定页面（默认首页）
    if (path || path === '') {
        setTimeout(() => route.push(path || '/'), 300)
    }
}

/**
 * 获取站点信息
 * @param {Object} state - 状态对象
 * @param {boolean} force - 是否强制刷新
 * @returns {Promise<Object|null>}
 */
const fetchSiteInfo = async (state = {}, force = false) => {
    // 已有进行中的请求则复用，避免多个组件同时挂载时重复请求。
    // 强制刷新需要跳过复用，否则拿到的仍是旧请求的结果。
    if (!force && fetchingSiteInfo && fetchSiteInfoPromise) {
        return fetchSiteInfoPromise
    }
    
    // 防止并发调用
    fetchingSiteInfo = true
    fetchSiteInfoPromise = (async () => {
        try {
            // 缓存键
            const cacheName = 'cardify_functions'
            const cacheExpire = 30 // 缓存30分钟
            
            // 优先从缓存获取（除非强制刷新）
            if (!force) {
                const cachedSiteInfo = cache.get(cacheName)
                if (cachedSiteInfo && typeof cachedSiteInfo === 'object') {
                    state.siteInfo = cachedSiteInfo
                    fetchingSiteInfo = false
                    fetchSiteInfoPromise = null
                    return cachedSiteInfo
                }
            }
            
            // 直接从API获取站点信息
            const response = await axios.get(`/api/config/one?key=cardify_functions`)

            // 检查响应结构
            if (response.code === 200 && response.data) {
                let siteInfo
                
                // 尝试不同的响应结构
                if (response.data.data && response.data.data.json) {
                    siteInfo = response.data.data.json
                } else if (response.data.json) {
                    siteInfo = response.data.json
                } else {
                    siteInfo = response.data
                }
                
                // 处理反引号问题和XSS防护
                if (siteInfo && typeof siteInfo === 'object') {
                    // 递归处理对象中的字符串
                    const sanitizeObject = (obj, parentKey = '', grandParentKey = '') => {
                    if (!obj || typeof obj !== 'object') return obj
                    
                    Object.keys(obj).forEach(key => {
                        // 不对 float_buttons.buttons 中的 content 字段进行转义，允许使用 HTML
                        // 不对 reward 配置中的收款码图片链接进行转义
                        if (typeof obj[key] === 'string' && 
                            !(grandParentKey === 'float_buttons' && parentKey === 'buttons' && key === 'content') &&
                            !(parentKey === 'reward' && (key === 'wechat' || key === 'alipay'))) {
                            // 移除反引号并进行基本的XSS防护
                            obj[key] = obj[key].replace(/`/g, '').replace(/</g, '&lt;').replace(/>/g, '&gt;')
                        } else if (typeof obj[key] === 'object' && obj[key] !== null) {
                            sanitizeObject(obj[key], key, parentKey)
                        }
                    })
                    return obj
                }
                    
                    siteInfo = sanitizeObject(siteInfo)
                    
                    // 确保enable_custom_style字段存在，默认为false
                    if (siteInfo.enable_custom_style === undefined) {
                        siteInfo.enable_custom_style = false
                    }
                    
                    // 缓存站点信息
                    cache.set(cacheName, siteInfo, cacheExpire)
                    
                    state.siteInfo = siteInfo
                    
                    // 站点信息加载完成后，按当前路由重新拼接完整标题（页面名 - 站点名）
                    // 避免用裸站点名覆盖掉路由已设置好的页面标题
                    if (siteInfo.title) {
                        const pageTitle = route.currentRoute?.value?.meta?.title ||
                            route.currentRoute?.value?.name || '未知页面'
                        document.title = `${pageTitle} - ${siteInfo.title}`
                    }
                    // 兜底：若路由实例未就绪，则交由 router.beforeEach 在下次导航时修正
                    applyRouteTitle?.()
                    
                    return siteInfo
                } else {
                    console.error('站点信息格式错误:', siteInfo)
                }
            } else {
                console.error('API响应不符合预期格式:', response)
            }
        } catch (error) {
            console.error('获取站点信息失败:', error)
        } finally {
            // 重置标志位（此前该赋值被误并入注释，导致标志位永远为 true，
            // 后续包括 force 强制刷新在内的调用都会被并发保护挡掉）
            fetchingSiteInfo = false
            fetchSiteInfoPromise = null
        }
    })()
    
    return fetchSiteInfoPromise
}

export const useCommStore = defineStore('comm', {
    state: () => {
        const cachedUser = cache.get('user-info') || {}
        const hasUser = !utils.is.empty(cachedUser)
        const cachedSiteInfo = cache.get('cardify_functions') || {}
        const cachedDarkMode = localStorage.getItem(STORAGE_KEYS.DARK_MODE) === 'true'
        
        return {
            auth: {
                login: false,
                reset: false,
                register: false,
            },
            login: {
                finish: hasUser,
                user: cachedUser
            },
            progress: false,
            nav: {
                title: ''
            },
            siteInfo: cachedSiteInfo,
            darkMode: cachedDarkMode
        }
    },
    actions: {
        /**
         * 切换认证状态
         * @param {string} name - 认证类型
         * @param {boolean} bool - 状态
         */
        switchAuth(name = 'login', bool = true) {
            for (const key in this.auth) {
                this.auth[key] = key === name ? bool : false
            }
        },
        
        /**
         * 登出
         * @param {string|null} path - 跳转路径
         * @returns {Promise<void>}
         */
        logout(path = null) {
            return logout(this, path)
        },
        
        /**
         * 手动触发登录态校验（比如登录后页面刷新后）
         * @returns {Promise<Object>} 登录状态对象
         */
        async checkLoginState() {
            await checkToken(this)
            return this.login
        },
        
        /**
         * 确保登录态已校验（应用启动时调用一次）
         * 仅在存在 Token 且尚未校验完成时才发请求，避免游客态的无效请求
         * @returns {Promise<Object>} 登录状态对象
         */
        async ensureLogin() {
            const hasToken = !!utils.get.cookie(TOKEN_NAME)
            const hasCachedUser = !utils.is.empty(this.login.user)
            
            if (!hasToken && !hasCachedUser) {
                this.login.finish = true
                return this.login
            }
            
            if (!this.login.finish || !hasCachedUser) {
                await checkToken(this)
            }
            
            return this.login
        },
        
        /**
         * 获取站点信息
         * @param {boolean} force - 是否强制刷新
         * @returns {Promise<Object|null>}
         */
        async fetchSiteInfo(force = false) {
            return await fetchSiteInfo(this, force)
        },
        
        /**
         * 清除站点信息缓存
         */
        clearSiteInfoCache() {
            const cacheName = 'cardify_functions'
            cache.del(cacheName)
            this.siteInfo = {}
        },
        
        /**
         * 设置导航标题
         * @param {string} title - 标题
         */
        setNavTitle(title) {
            this.nav.title = title
        },
        
        /**
         * 设置加载状态
         * @param {boolean} state - 加载状态
         */
        setProgress(state) {
            this.progress = state
        },
        
        /**
         * 切换暗黑模式
         */
        toggleDarkMode() {
            if (this.darkMode === undefined) {
                this.darkMode = localStorage.getItem(STORAGE_KEYS.DARK_MODE) === 'true'
            }
            
            this.darkMode = !this.darkMode
            
            const htmlElement = document.documentElement
            htmlElement.setAttribute('data-bs-theme', this.darkMode ? 'dark' : 'light')
            
            localStorage.setItem(STORAGE_KEYS.DARK_MODE, this.darkMode.toString())
        }
    },
    getters: {
        /**
         * 获取登录状态
         * @param {Object} state - 状态对象
         * @returns {Object} 登录状态
         */
        // 保持为纯读取：getter 内发起请求会在多个组件读取时反复触发校验，
        // 且它修改的 state.login 正是自身依赖，容易形成额外的无效请求。
        // Token 校验统一由 ensureLogin() 在应用启动时触发一次。
        getLogin: (state) => state.login,
        
        /**
         * 获取站点信息
         * @param {Object} state - 状态对象
         * @returns {Object} 站点信息
         */
        getSiteInfo: (state) => {
            // 直接返回站点信息，不再自动获取
            return state.siteInfo
        },
        
        /**
         * 获取是否已登录
         * @param {Object} state - 状态对象
         * @returns {boolean} 是否已登录
         */
        isLoggedIn: (state) => {
            return state.login.finish && !utils.is.empty(state.login.user)
        },
        
        /**
         * 获取当前用户
         * @param {Object} state - 状态对象
         * @returns {Object} 当前用户
         */
        currentUser: (state) => {
            return state.login.user || {}
        },
        
        /**
         * 获取暗黑模式状态
         * @param {Object} state - 状态对象
         * @returns {boolean} 是否为暗黑模式
         */
        isDarkMode: (state) => {
            return state.darkMode
        }
    }
})