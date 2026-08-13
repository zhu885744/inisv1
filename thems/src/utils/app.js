import * as bootstrap from 'bootstrap/dist/js/bootstrap.bundle.min.js'
import { computed, watch, ref } from 'vue'
import { useRoute } from 'vue-router'
import utils from '@/utils/utils'
import router from '@/router/index.js'
import { request, cache, uploadImage } from '@/utils/network'
import { useCommStore } from '@/store/comm'

const DEV = import.meta.env.DEV || false
const CONFIG_PREFIX = 'inis_theme_config_'
const CONFIG_EXPIRE_DAYS = 7

let cachedConfig = null
let configFetched = false

const _readConfigValue = (key, defaultValue) => {
  if (typeof key !== 'string' || !key.trim()) {
    return defaultValue
  }
  
  const localKey = `${CONFIG_PREFIX}${key}`
  const localValue = localStorage.getItem(localKey)
  
  if (localValue !== null) {
    try {
      const parsed = JSON.parse(localValue)
      if (parsed && typeof parsed === 'object') {
        if (parsed.expire && Date.now() > parsed.expire) {
          localStorage.removeItem(localKey)
        } else {
          return parsed.value !== undefined ? parsed.value : parsed
        }
      } else {
        return parsed
      }
    } catch {
      return localValue
    }
  }
  
  const envKey = `VITE_${key.toUpperCase()}`
  if (import.meta.env[envKey] !== undefined) {
    return import.meta.env[envKey]
  }

  if (cachedConfig && cachedConfig[key] !== undefined) {
    return cachedConfig[key]
  }

  return defaultValue
}

const getConfigSync = (key, defaultValue = null) => {
  return _readConfigValue(key, defaultValue)
}

const getConfig = async (key, defaultValue = null) => {
  if (!configFetched) {
    try {
      await syncConfigFromAPI()
    } catch (error) {
      console.error('[Config] 从接口同步配置失败:', error)
    }
  }
  return _readConfigValue(key, defaultValue)
}

const setConfig = (key, value, expireDays = CONFIG_EXPIRE_DAYS) => {
  if (typeof key !== 'string' || !key.trim()) {
    return
  }
  
  const localKey = `${CONFIG_PREFIX}${key}`
  
  if (value === null || value === undefined) {
    localStorage.removeItem(localKey)
  } else {
    const dataToStore = {
      value,
      expire: expireDays > 0 ? Date.now() + expireDays * 24 * 60 * 60 * 1000 : null,
      timestamp: Date.now()
    }
    localStorage.setItem(localKey, JSON.stringify(dataToStore))
  }
  
  if (cachedConfig) {
    cachedConfig[key] = value
  }
}

const initConfig = () => {
  cachedConfig = {
    title: import.meta.env.VITE_TITLE || '',
    api_uri: import.meta.env.VITE_API_URI || '',
    router_mode: import.meta.env.VITE_ROUTER_MODE || 'hash',
    base_url: import.meta.env.VITE_BASE_URL || '/',
    api_key: import.meta.env.VITE_API_KEY || '',
    token_name: import.meta.env.VITE_TOKEN_NAME || '',
    lazy_time: parseInt(import.meta.env.VITE_LAZY_TIME) || 500
  }
}

const syncConfigFromAPI = async () => {
  try {
    const response = await request.get('/api/config/all')
    if (response.code === 200 && response.data) {
      cachedConfig = { ...cachedConfig, ...response.data }
      Object.entries(response.data).forEach(([key, value]) => {
        setConfig(key, value)
      })
    }
  } catch (error) {
    console.error('[Config] 从接口同步配置失败:', error)
  } finally {
    configFetched = true
  }
}

initConfig()

const getAllConfig = () => {
  return cachedConfig || {}
}

const config = {
  get: getConfig,
  getSync: getConfigSync,
  set: setConfig,
  getAll: getAllConfig,
  sync: syncConfigFromAPI
}

class Channel {
  constructor(name = 'default') {
    this.channelName = this.normalizeChannelName(name)
    this.listeners = new Map()
    this.onceListeners = new Map()
    this.isClosed = false
    this.messageQueue = []
    this.unloadRegistered = false
    this.initChannel()
    this.bindUnloadHandler()
  }

  normalizeChannelName(name) {
    if (typeof name === 'string') {
      const trimmed = name.trim()
      return trimmed || 'default'
    }
    return 'default'
  }

  initChannel() {
    try {
      if (!window.BroadcastChannel) {
        this.bc = null
        this.isClosed = true
        return
      }
      
      this.bc = new BroadcastChannel(this.channelName)
      this.bc.onmessage = this.handleMessage.bind(this)
      this.bc.onmessageerror = this.handleError.bind(this)
      
      this.flushMessageQueue()
    } catch (error) {
      console.error('[Channel] 初始化失败:', error)
      this.bc = null
      this.isClosed = true
    }
  }

  bindUnloadHandler() {
    if (typeof window !== 'undefined' && !this.unloadRegistered) {
      window.addEventListener('beforeunload', this.handleUnload.bind(this))
      this.unloadRegistered = true
    }
  }

  handleUnload() {
    this.close()
  }

  handleMessage(event) {
    if (!event?.data || this.isClosed) return
    
    const { type = 'message', data, timestamp = Date.now() } = event.data
    
    const listeners = this.listeners.get(type) || []
    const onceListeners = [...(this.onceListeners.get(type) || [])]
    
    listeners.forEach(callback => {
      try {
        callback(data, { type, timestamp, channel: this.channelName })
      } catch (error) {
        console.error('[Channel] 监听器执行失败:', error)
      }
    })
    
    onceListeners.forEach(callback => {
      try {
        callback(data, { type, timestamp, channel: this.channelName })
      } catch (error) {
        console.error('[Channel] 一次性监听器执行失败:', error)
      }
    })
    
    if (onceListeners.length > 0) {
      this.onceListeners.delete(type)
    }
  }

  handleError(event) {
    console.error('[Channel] 消息错误:', event)
  }

  flushMessageQueue() {
    if (!this.bc || this.isClosed) return
    
    while (this.messageQueue.length > 0) {
      const item = this.messageQueue.shift()
      try {
        this.bc.postMessage(item)
      } catch (error) {
        console.error('[Channel] 发送队列消息失败:', error)
      }
    }
  }

  on(typeOrCallback, callback) {
    if (this.isClosed) {
      return () => {}
    }
    
    let type, handler
    
    if (typeof typeOrCallback === 'function') {
      type = 'message'
      handler = typeOrCallback
    } else {
      type = typeOrCallback || 'message'
      handler = callback
    }
    
    if (typeof handler !== 'function') {
      return () => {}
    }
    
    if (!this.listeners.has(type)) {
      this.listeners.set(type, [])
    }
    this.listeners.get(type).push(handler)
    
    return () => this.off(type, handler)
  }

  once(typeOrCallback, callback) {
    if (this.isClosed) return () => {}
    
    let type, handler
    
    if (typeof typeOrCallback === 'function') {
      type = 'message'
      handler = typeOrCallback
    } else {
      type = typeOrCallback || 'message'
      handler = callback
    }
    
    if (typeof handler !== 'function') return () => {}
    
    if (!this.onceListeners.has(type)) {
      this.onceListeners.set(type, [])
    }
    this.onceListeners.get(type).push(handler)
    
    return () => {
      const listeners = this.onceListeners.get(type)
      if (listeners) {
        const index = listeners.indexOf(handler)
        if (index > -1) listeners.splice(index, 1)
      }
    }
  }

  off(typeOrCallback, callback) {
    let type, handler
    
    if (typeof typeOrCallback === 'function') {
      type = 'message'
      handler = typeOrCallback
    } else {
      type = typeOrCallback || 'message'
      handler = callback
    }
    
    if (!handler) {
      this.listeners.delete(type)
      this.onceListeners.delete(type)
      return true
    }
    
    const listeners = this.listeners.get(type)
    if (listeners) {
      const index = listeners.indexOf(handler)
      if (index > -1) {
        listeners.splice(index, 1)
        if (listeners.length === 0) {
          this.listeners.delete(type)
        }
        return true
      }
    }
    
    const onceListeners = this.onceListeners.get(type)
    if (onceListeners) {
      const index = onceListeners.indexOf(handler)
      if (index > -1) {
        onceListeners.splice(index, 1)
        if (onceListeners.length === 0) {
          this.onceListeners.delete(type)
        }
        return true
      }
    }
    
    return false
  }

  emit(data, type = 'message') {
    if (this.isClosed) {
      return false
    }
    
    if (!this.bc) {
      this.messageQueue.push({ 
        type, 
        data, 
        timestamp: Date.now(),
        channel: this.channelName,
        source: 'channel'
      })
      return false
    }
    
    try {
      const message = {
        type: type || 'message',
        data,
        timestamp: Date.now(),
        channel: this.channelName,
        source: 'channel'
      }
      
      this.bc.postMessage(message)
      return true
    } catch (error) {
      console.error('[Channel] 发送消息失败:', error)
      return false
    }
  }

  close() {
    if (this.isClosed) return
    
    this.isClosed = true
    this.listeners.clear()
    this.onceListeners.clear()
    this.messageQueue = []
    
    if (this.bc) {
      try {
        this.bc.close()
      } catch (error) {
        console.error('[Channel] 关闭失败:', error)
      }
      this.bc = null
    }
    
    if (typeof window !== 'undefined' && this.unloadRegistered) {
      window.removeEventListener('beforeunload', this.handleUnload.bind(this))
      this.unloadRegistered = false
    }
  }

  static channelInstances = new Map()

  static create(name = 'default') {
    const normalizedName = typeof name === 'string' ? name.trim() || 'default' : 'default'
    
    if (Channel.channelInstances.has(normalizedName)) {
      return Channel.channelInstances.get(normalizedName)
    }
    
    const instance = new Channel(normalizedName)
    Channel.channelInstances.set(normalizedName, instance)
    return instance
  }

  static isSupported() {
    return typeof window !== 'undefined' && !!window.BroadcastChannel
  }

  static destroy(name) {
    const instance = Channel.channelInstances.get(name)
    if (instance) {
      instance.close()
      Channel.channelInstances.delete(name)
    }
  }
}

const defaultChannel = Channel.create('default')
const channel = defaultChannel

const push = (options) => {
  if (utils.is.empty(options)) {
    return Promise.reject(new Error('路由跳转参数不能为空'))
  }
  
  return new Promise((resolve, reject) => {
    router.push(options)
      .then(() => resolve())
      .catch((err) => {
        if (err.message?.includes('Avoided redundant navigation to current location')) {
          resolve()
        } else {
          reject(err)
        }
      })
  })
}

const replace = (options) => {
  if (utils.is.empty(options)) {
    return Promise.reject(new Error('路由跳转参数不能为空'))
  }
  
  return new Promise((resolve, reject) => {
    router.replace(options)
      .then(() => resolve())
      .catch((err) => {
        if (err.message?.includes('Avoided redundant navigation to current location')) {
          resolve()
        } else {
          reject(err)
        }
      })
  })
}

const goBack = (step = 1) => {
  return new Promise((resolve) => {
    const currentRoute = router.currentRoute.value
    if (currentRoute.matched.length <= 1) {
      push('/').then(() => resolve())
    } else {
      router.go(-step)
      setTimeout(resolve, 100)
    }
  })
}

const getCurrentRoute = () => {
  return router.currentRoute.value
}

const getRouteParams = () => {
  return getCurrentRoute().params || {}
}

const getRouteQuery = () => {
  return getCurrentRoute().query || {}
}

const getRouteMeta = () => {
  return getCurrentRoute().meta || {}
}

const isRouteActive = (path) => {
  return router.currentRoute.value.path === path
}

const redirectWithQuery = (path, query = {}) => {
  return push({ path, query })
}

const redirectWithParams = (name, params = {}) => {
  return push({ name, params })
}

const route = {
  push,
  replace,
  goBack,
  getCurrentRoute,
  getRouteParams,
  getRouteQuery,
  getRouteMeta,
  isRouteActive,
  redirectWithQuery,
  redirectWithParams
}

const MESSAGE_TYPES = {
  SUCCESS: 'success',
  ERROR: 'error',
  WARNING: 'warning',
  INFO: 'info'
}

const TYPE_CONFIG = {
  [MESSAGE_TYPES.INFO]: {
    bgClass: '',
    iconClass: 'bi-info-circle text-info'
  },
  [MESSAGE_TYPES.SUCCESS]: {
    bgClass: '',
    iconClass: 'bi-check-circle text-success'
  },
  [MESSAGE_TYPES.WARNING]: {
    bgClass: '',
    iconClass: 'bi-exclamation-triangle text-warning'
  },
  [MESSAGE_TYPES.ERROR]: {
    bgClass: '',
    iconClass: 'bi-x-circle text-danger'
  }
}

class ToastManager {
  constructor() {
    this.container = null
    this.toasts = new Map()
    this.idCounter = 0
    this.unloadRegistered = false
    this.bindUnloadHandler()
  }

  bindUnloadHandler() {
    if (typeof window !== 'undefined' && !this.unloadRegistered) {
      window.addEventListener('beforeunload', this.cleanup.bind(this))
      window.addEventListener('pagehide', this.cleanup.bind(this))
      this.unloadRegistered = true
    }
  }

  ensureContainer() {
    if (this.container) return this.container
    
    this.container = document.createElement('div')
    this.container.className = 'toast-container position-fixed bottom-0 end-0 p-3'
    this.container.style.zIndex = '9999'
    document.body.appendChild(this.container)
    
    return this.container
  }

  getIconByType(type) {
    return TYPE_CONFIG[type]?.iconClass || TYPE_CONFIG.info.iconClass
  }

  getTitleByType(type) {
    const titles = {
      [MESSAGE_TYPES.INFO]: '信息',
      [MESSAGE_TYPES.SUCCESS]: '成功',
      [MESSAGE_TYPES.WARNING]: '警告',
      [MESSAGE_TYPES.ERROR]: '错误'
    }
    return titles[type] || '通知'
  }

  show({ message = '', type = MESSAGE_TYPES.INFO, title = '', duration = 3000, closable = true, position = 'bottom-right' }) {
    if (!message) return { id: null, close: () => {} }

    const container = this.ensureContainer()
    const id = `toast_${++this.idCounter}`
    
    const iconClass = this.getIconByType(type)
    const defaultTitle = title || this.getTitleByType(type)
    
    const toastHtml = `
      <div class="toast" id="${id}" role="alert" aria-live="assertive" aria-atomic="true" data-bs-autohide="true" data-bs-delay="${duration}">
        <div class="toast-header">
          <i class="${iconClass} me-2"></i>
          <strong class="me-auto">${defaultTitle}</strong>
          ${closable ? '<button type="button" class="btn-close" data-bs-dismiss="toast" aria-label="关闭"></button>' : ''}
        </div>
        <div class="toast-body">${message}</div>
      </div>
    `
    
    container.insertAdjacentHTML('beforeend', toastHtml)
    
    const toastElement = document.getElementById(id)
    const toast = new bootstrap.Toast(toastElement)
    
    toastElement.addEventListener('hidden.bs.toast', () => {
      toastElement.remove()
      this.toasts.delete(id)
    })
    
    toast.show()
    this.toasts.set(id, { element: toastElement, toast })
    
    return {
      id,
      close: () => {
        if (this.toasts.has(id)) {
          toast.hide()
        }
      }
    }
  }

  info(message, options = {}) {
    return this.show({ ...options, message, type: MESSAGE_TYPES.INFO })
  }

  success(message, options = {}) {
    return this.show({ ...options, message, type: MESSAGE_TYPES.SUCCESS })
  }

  warning(message, options = {}) {
    return this.show({ ...options, message, type: MESSAGE_TYPES.WARNING })
  }

  error(message, options = {}) {
    return this.show({ ...options, message, type: MESSAGE_TYPES.ERROR })
  }

  cleanup() {
    this.toasts.forEach(({ toast }) => {
      try {
        toast.hide()
      } catch (error) {
        console.error('[ToastManager] 清理 toast 失败:', error)
      }
    })
    
    if (this.container && this.container.parentNode) {
      this.container.parentNode.removeChild(this.container)
    }
    
    this.toasts.clear()
    this.container = null
    
    if (typeof window !== 'undefined' && this.unloadRegistered) {
      window.removeEventListener('beforeunload', this.cleanup.bind(this))
      window.removeEventListener('pagehide', this.cleanup.bind(this))
      this.unloadRegistered = false
    }
  }
}

const toast = new ToastManager()

const formatDate = (date, format = 'YYYY-MM-DD HH:mm:ss') => {
  if (!date) return ''
  
  let timestamp = parseInt(date)
  if (!isNaN(timestamp) && timestamp < 10000000000) {
    timestamp *= 1000
  }
  
  const d = new Date(timestamp)
  if (isNaN(d.getTime())) return ''
  
  const year = d.getFullYear()
  const month = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const hours = String(d.getHours()).padStart(2, '0')
  const minutes = String(d.getMinutes()).padStart(2, '0')
  const seconds = String(d.getSeconds()).padStart(2, '0')
  
  return format
    .replace('YYYY', year)
    .replace('MM', month)
    .replace('DD', day)
    .replace('HH', hours)
    .replace('mm', minutes)
    .replace('ss', seconds)
}

const formatRelativeTime = (timestamp) => {
  if (!timestamp) return ''
  
  const now = Date.now()
  let time = parseInt(timestamp)
  
  if (!isNaN(time) && time < 10000000000) {
    time *= 1000
  }
  
  if (isNaN(time)) return ''
  
  const diff = now - time
  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)
  const weeks = Math.floor(days / 7)
  const months = Math.floor(days / 30)
  const years = Math.floor(days / 365)
  
  if (seconds < 60) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`
  if (weeks < 4) return `${weeks}周前`
  if (months < 12) return `${months}月前`
  return `${years}年前`
}

const formatNumber = (num) => {
  if (num === null || num === undefined || isNaN(num)) return '0'
  
  const n = Number(num)
  if (n >= 100000000) return (n / 100000000).toFixed(1) + '亿'
  if (n >= 10000) return (n / 10000).toFixed(1) + '万'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n.toString()
}

const truncateText = (text, maxLength = 100, suffix = '...') => {
  if (!text) return ''
  if (text.length <= maxLength) return text
  return text.substring(0, maxLength) + suffix
}

const escapeHtml = (text) => {
  if (!text) return ''
  const map = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;'
  }
  return text.replace(/[&<>"']/g, m => map[m])
}

const stripHtml = (html) => {
  if (!html) return ''
  return html.replace(/<[^>]*>/g, '')
}

const formatters = {
  formatDate,
  formatRelativeTime,
  formatNumber,
  truncateText,
  escapeHtml,
  stripHtml
}

const validateComment = (content, options = {}) => {
  const { minLength = 1, maxLength = 1000 } = options
  
  if (!content || typeof content !== 'string' || !content.trim()) {
    return { valid: false, message: '评论内容不能为空' }
  }
  
  const trimmed = content.trim()
  
  if (trimmed.length < minLength) {
    return { valid: false, message: `评论内容不能少于 ${minLength} 个字符` }
  }
  
  if (trimmed.length > maxLength) {
    return { valid: false, message: `评论内容不能超过 ${maxLength} 个字符` }
  }
  
  return { valid: true, message: '' }
}

const validateUsername = (username) => {
  if (!username || typeof username !== 'string' || !username.trim()) {
    return { valid: false, message: '用户名不能为空' }
  }
  
  if (username.length < 2) {
    return { valid: false, message: '用户名至少需要2个字符' }
  }
  
  if (username.length > 20) {
    return { valid: false, message: '用户名不能超过20个字符' }
  }
  
  const usernameRegex = /^[\u4e00-\u9fa5a-zA-Z0-9_]+$/
  if (!usernameRegex.test(username)) {
    return { valid: false, message: '用户名只能包含中文、字母、数字和下划线' }
  }
  
  return { valid: true, message: '' }
}

const validatePassword = (password) => {
  if (!password || typeof password !== 'string') {
    return { valid: false, message: '密码不能为空' }
  }
  
  if (password.length < 6) {
    return { valid: false, message: '密码至少需要6个字符' }
  }
  
  if (password.length > 32) {
    return { valid: false, message: '密码不能超过32个字符' }
  }
  
  return { valid: true, message: '' }
}

const validateEmail = (email) => {
  if (!email || typeof email !== 'string') {
    return { valid: false, message: '邮箱不能为空' }
  }
  
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  if (!emailRegex.test(email)) {
    return { valid: false, message: '请输入有效的邮箱地址' }
  }
  
  return { valid: true, message: '' }
}

const rateLimitCache = new Map()

const checkRateLimit = (key, limitSeconds = 30) => {
  if (!key || typeof key !== 'string') {
    return { allowed: true, remaining: 0 }
  }
  
  const lastTime = rateLimitCache.get(key)
  
  if (!lastTime) return { allowed: true, remaining: 0 }
  
  const now = Date.now()
  const diff = (now - lastTime) / 1000
  
  if (diff < limitSeconds) {
    return { allowed: false, remaining: Math.ceil(limitSeconds - diff) }
  }
  
  rateLimitCache.delete(key)
  return { allowed: true, remaining: 0 }
}

const setRateLimit = (key) => {
  if (!key || typeof key !== 'string') return
  
  rateLimitCache.set(key, Date.now())
}

const validators = {
  validateComment,
  validateUsername,
  validatePassword,
  validateEmail,
  checkRateLimit,
  setRateLimit
}

const getSiteTitle = () => {
  try {
    const store = useCommStore()
    return store.siteInfo?.title || getConfigSync('title') || 'Xiao-INIS'
  } catch (error) {
    return getConfigSync('title') || 'Xiao-INIS'
  }
}

const setupRouteTitle = (routerInstance) => {
  routerInstance.beforeEach((to, from, next) => {
    const siteTitle = getSiteTitle()
    const pageTitle = to.meta.title || to.name || '未知页面'
    document.title = `${pageTitle} - ${siteTitle}`
    next()
  })
}

// 根据当前路由重新计算并写入页面标题（单一数据源，避免多实例冲突导致标题错乱）
const applyRouteTitle = () => {
  const siteTitle = getSiteTitle()
  const route = router.currentRoute.value
  const pageTitle = route?.meta?.title || route?.name || '未知页面'
  document.title = `${pageTitle} - ${siteTitle}`
}

const usePageTitle = (options = {}) => {
  // 兼容旧调用方式：usePageTitle('小黑屋')
  const opts = typeof options === 'string' ? { staticTitle: options, defaultTitle: options } : options
  const {
    staticTitle = '',
    defaultTitle = '未知页面'
  } = opts

  const route = useRoute()

  const baseTitle = computed(() => getSiteTitle())

  const fullTitle = computed(() => {
    const pageTitle = route.meta.title || route.name || defaultTitle
    return `${pageTitle} - ${baseTitle.value}`
  })

  // 直接写入 document.title，不再依赖 @vueuse/useTitle 的多个实例（会造成标题串页）
  const setTitle = (newTitle) => {
    document.title = newTitle
  }

  const setDynamicTitle = (dynamicTitle, appendBase = true) => {
    if (appendBase) {
      document.title = `${dynamicTitle} - ${baseTitle.value}`
    } else {
      document.title = dynamicTitle
    }
  }

  const setLoadingTitle = (customLoadingText = '加载中...') => {
    const displayTitle = staticTitle || route.meta.title || route.name || '加载中'
    document.title = `${displayTitle} - ${customLoadingText}`
  }

  const setErrorTitle = (customErrorText = '获取失败') => {
    const displayTitle = staticTitle || route.meta.title || route.name || '错误'
    document.title = `${displayTitle} - ${customErrorText}`
  }

  // 还原为当前路由对应的标准标题（如 文章详情 - 网站标题）
  const resetTitle = () => {
    document.title = fullTitle.value
  }

  // 路由变化（含 keep-alive 切换）时，始终以当前路由 meta 为准，避免停留在旧页面标题
  watch(
    () => route.fullPath,
    () => {
      resetTitle()
    },
    { immediate: true }
  )

  return {
    title: fullTitle,
    fullTitle,
    baseTitle,
    setTitle,
    setDynamicTitle,
    setLoadingTitle,
    setErrorTitle,
    resetTitle
  }
}

const usePageTitleUtil = {
  usePageTitle,
  setupRouteTitle
}

const log = (...args) => {
  if (DEV) {
    console.log('[App]', new Date().toLocaleTimeString(), ...args)
  }
}

const logError = (...args) => {
  console.error('[App Error]', new Date().toLocaleTimeString(), ...args)
  
  if (typeof window !== 'undefined' && window.__inis_log_report) {
    try {
      window.__inis_log_report({
        level: 'error',
        message: args.map(arg => 
          typeof arg === 'object' ? JSON.stringify(arg) : String(arg)
        ).join(' '),
        timestamp: Date.now()
      })
    } catch (reportError) {
      console.error('[App] 日志上报失败:', reportError)
    }
  }
}

const registerLogReporter = (reporter) => {
  if (typeof window !== 'undefined' && typeof reporter === 'function') {
    window.__inis_log_report = reporter
  }
}

let initPromise = null

const init = async (app, options = {}) => {
  if (initPromise) {
    return initPromise
  }
  
  const { 
    bootstrap: loadBootstrap = true, 
    fancybox: loadFancybox = true, 
    api: loadApi = true,
    onProgress 
  } = options
  
  initPromise = (async () => {
    try {
      const modules = []
      
      if (loadBootstrap) {
        modules.push(import('bootstrap/dist/js/bootstrap.bundle.min.js').then(({ default: bs }) => ({ name: 'bootstrap', value: bs })))
      }
      
      if (loadFancybox) {
        modules.push(import('@fancyapps/ui/dist/fancybox/').then(({ Fancybox }) => ({ name: 'fancybox', value: Fancybox })))
      }
      
      if (loadApi) {
        modules.push(import('@/api').then(({ default: API }) => ({ name: 'api', value: API })))
      }
      
      const results = await Promise.all(modules)
      
      results.forEach(({ name, value }) => {
        if (onProgress) {
          onProgress(name, 'loaded')
        }
        
        switch (name) {
          case 'bootstrap':
            if (value && typeof window !== 'undefined') {
              window.bootstrap = value
            }
            break
          case 'api':
            if (value) {
              app.config.globalProperties.$api = value
            }
            break
          case 'fancybox':
            if (value && typeof window !== 'undefined') {
              window.Fancybox = value
              setTimeout(() => {
                value.bind("[data-fancybox]", {
                  Hash: false,
                  Thumbs: { autoStart: false }
                })
              }, 100)
            }
            break
        }
      })
      
      return {
        bootstrap: loadBootstrap ? bootstrap : null,
        Fancybox: loadFancybox ? (results.find(r => r.name === 'fancybox')?.value || null) : null,
        API: loadApi ? (results.find(r => r.name === 'api')?.value || null) : null
      }
    } catch (error) {
      logError('初始化失败:', error)
      return {}
    }
  })()
  
  return initPromise
}

const setupSiteInfo = async (commStore) => {
  try {
    await commStore.fetchSiteInfo()
    
    if (commStore.siteInfo?.favicon && typeof window !== 'undefined') {
      const favicon = document.querySelector('link[rel="icon"]')
      if (favicon) {
        favicon.href = commStore.siteInfo.favicon
      }
    }
    
    const htmlElement = document.documentElement
    htmlElement.setAttribute('data-bs-theme', commStore.isDarkMode ? 'dark' : 'light')
    
    return true
  } catch (error) {
    logError('设置站点信息失败:', error)
    return false
  }
}

const app = {
  config,
  channel: defaultChannel,
  Channel,
  route,
  toast,
  formatters,
  validators,
  usePageTitle: usePageTitleUtil,
  init,
  setupSiteInfo,
  MESSAGE_TYPES,
  registerLogReporter
}

// 头衔颜色映射 - 修仙体系
const TITLE_COLOR_MAP = {
  '掌门': 'title-zhangmen',
  '长老': 'title-zhanglao',
  '护法': 'title-hufa',
  '内门弟子': 'title-neimen',
  '外门弟子': 'title-waimen',
  '炼气修士': 'title-lianqi',
  '筑基修士': 'title-zhuji',
  '结丹修士': 'title-jiedan',
  '元婴老祖': 'title-yuanying',
  '化神大能': 'title-huashen'
}

const getTitleColorClass = (title) => {
  if (!title || !String(title).trim()) return ''
  return TITLE_COLOR_MAP[String(title).trim()] || 'title-default'
}

export {
  config,
  getConfig as get,
  getConfigSync as getSync,
  channel,
  Channel,
  route,
  toast,
  formatters,
  validators,
  usePageTitle,
  setupRouteTitle,
  applyRouteTitle,
  init,
  setupSiteInfo,
  MESSAGE_TYPES,
  log,
  logError,
  registerLogReporter,
  checkRateLimit,
  setRateLimit,
  validateComment,
  validateUsername,
  validatePassword,
  validateEmail,
  getTitleColorClass
}

export default app