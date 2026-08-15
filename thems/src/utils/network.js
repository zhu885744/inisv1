import axios from 'axios'
import utils from '@/utils/utils'
import { getSync } from '@/utils/app'

const DEV = import.meta.env.DEV
const DEFAULT_TIMEOUT = 60 * 1000
const MAX_RETRY = 2
const MAX_CACHE_SIZE = 500
const API_WHITELIST = ['/api/', '/dev/']

const axiosInstance = axios.create({
  timeout: DEFAULT_TIMEOUT,
  withCredentials: false,
  headers: {
    'X-Requested-With': 'XMLHttpRequest',
    'Accept': 'application/json',
    'Content-Type': 'application/json; charset=utf-8'
  }
})

class Cache {
  constructor() {
    this.testKey = '__cache_test__'
    this.maxSize = MAX_CACHE_SIZE
    this._available = null
    // 内存镜像：避免同一次渲染内重复 JSON.parse
    this._memo = new Map()
  }

  // localStorage 可用性只探测一次并缓存结果，
  // 避免每次 get/set/has 都执行一次写入+删除
  isAvailable() {
    if (this._available !== null) {
      return this._available
    }
    try {
      localStorage.setItem(this.testKey, 'test')
      localStorage.removeItem(this.testKey)
      this._available = true
    } catch {
      this._available = false
    }
    return this._available
  }

  isValidKey(key) {
    return typeof key === 'string' && key.trim() !== ''
  }

  has(key) {
    if (!this.isValidKey(key) || !this.isAvailable()) {
      return false
    }
    const rawValue = localStorage.getItem(key)
    if (rawValue === null || rawValue === '') {
      return false
    }
    try {
      const parsed = JSON.parse(rawValue)
      if (this.isExpiredCache(parsed)) {
        return false
      }
      return true
    } catch {
      return true
    }
  }

  get(key) {
    if (!this.isValidKey(key) || !this.isAvailable()) {
      return null
    }

    // 命中内存镜像，省去一次 JSON.parse
    if (this._memo.has(key)) {
      const hit = this._memo.get(key)
      if (hit.expire && Date.now() > hit.expire) {
        this.del(key)
        return null
      }
      return hit.data
    }

    const rawValue = localStorage.getItem(key)
    if (rawValue === null || rawValue === '') {
      return null
    }

    try {
      const parsed = JSON.parse(rawValue)
      
      if (this.isExpiredCache(parsed)) {
        this.del(key)
        return null
      }

      const data = parsed && parsed.data !== undefined ? parsed.data : parsed
      this._memo.set(key, { data, expire: parsed?.expire || 0 })
      return data
    } catch (error) {
      console.warn(`[Cache] 解析缓存失败: ${key}`, error)
      return rawValue || null
    }
  }

  isExpiredCache(cacheData) {
    return cacheData && 
           typeof cacheData === 'object' && 
           cacheData.expire && 
           Date.now() > cacheData.expire
  }

  set(key, value, minutes = 0) {
    if (!this.isValidKey(key) || !this.isAvailable()) {
      return
    }

    const storeValue = value === undefined ? null : value
    
    let dataToStore = storeValue
    const expireMinutes = Number(minutes)
    
    if (!isNaN(expireMinutes) && expireMinutes > 0) {
      dataToStore = {
        data: storeValue,
        expire: Date.now() + expireMinutes * 60 * 1000,
        timestamp: Date.now()
      }
    } else {
      dataToStore = {
        data: storeValue,
        timestamp: Date.now()
      }
    }

    this._memo.set(key, {
      data: storeValue,
      expire: dataToStore.expire || 0
    })

    try {
      localStorage.setItem(key, JSON.stringify(dataToStore))
    } catch (error) {
      // 容量超限：先淘汰最旧的数据再重试一次
      this.cleanupOldest()
      try {
        localStorage.setItem(key, JSON.stringify(dataToStore))
      } catch {
        console.warn(`[Cache] 存储失败，已降级为内存缓存: ${key}`)
      }
    }
  }

  // 淘汰最旧的 20% 缓存项。
  // 之前只统计带 CACHE_KEY_PREFIX 前缀的键，而业务侧写入的都是裸键，
  // 导致淘汰逻辑永远选不出任何数据，这里改为识别本缓存写入的结构。
  cleanupOldest() {
    if (!this.isAvailable()) return

    const cacheItems = []

    this.keys().forEach(key => {
      try {
        const parsed = JSON.parse(localStorage.getItem(key))
        if (parsed && typeof parsed === 'object' && parsed.timestamp) {
          cacheItems.push({ key, timestamp: parsed.timestamp })
        }
      } catch {
      }
    })

    cacheItems.sort((a, b) => a.timestamp - b.timestamp)

    const removeCount = Math.max(1, Math.floor(cacheItems.length * 0.2))
    cacheItems.slice(0, removeCount).forEach(item => this.del(item.key))
  }

  setItem(key, value, minutes = 0) {
    return this.set(key, value, minutes)
  }

  getItem(key) {
    return this.get(key)
  }

  del(key) {
    if (!this.isValidKey(key) || !this.isAvailable()) {
      return
    }
    this._memo.delete(key)
    localStorage.removeItem(key)
  }

  removeItem(key) {
    return this.del(key)
  }

  // 清空本缓存写入的所有数据（不误删站点其它 localStorage 数据，
  // 例如登录 token、主题偏好等非本缓存结构的键）
  clear() {
    if (!this.isAvailable()) {
      return
    }
    this.keys().forEach(key => {
      if (this.isOwnedKey(key)) {
        this.del(key)
      }
    })
  }

  // 判断某个键是否由本缓存写入（结构含 timestamp）
  isOwnedKey(key) {
    if (key === this.testKey) return false
    try {
      const parsed = JSON.parse(localStorage.getItem(key))
      return !!(parsed && typeof parsed === 'object' && parsed.timestamp)
    } catch {
      return false
    }
  }

  setMultiple(items) {
    if (!Array.isArray(items) || !this.isAvailable()) {
      return false
    }

    let successCount = 0
    items.forEach(({ key, value, minutes = 0 }) => {
      try {
        this.set(key, value, minutes)
        successCount++
      } catch (error) {
        console.error(`[Cache] 批量存储失败: ${key}`, error)
      }
    })
    
    return successCount === items.length
  }

  getMultiple(keys) {
    if (!Array.isArray(keys) || !this.isAvailable()) {
      return {}
    }

    const result = {}
    keys.forEach(key => {
      if (this.isValidKey(key)) {
        result[key] = this.get(key)
      }
    })
    
    return result
  }

  delMultiple(keys) {
    if (!Array.isArray(keys) || !this.isAvailable()) {
      return false
    }

    keys.forEach(key => {
      if (this.isValidKey(key)) {
        this.del(key)
      }
    })
    
    return true
  }

  keys() {
    if (!this.isAvailable()) {
      return []
    }
    return Object.keys(localStorage)
  }

  size() {
    if (!this.isAvailable()) {
      return 0
    }
    return this.keys().filter(k => this.isOwnedKey(k)).length
  }

  clearExpired() {
    if (!this.isAvailable()) {
      return 0
    }

    let clearedCount = 0
    const keys = this.keys()
    
    keys.forEach(key => {
      const value = localStorage.getItem(key)
      if (!value) return
      try {
        const parsed = JSON.parse(value)
        if (this.isExpiredCache(parsed)) {
          this.del(key)
          clearedCount++
        }
      } catch {
        // 非 JSON 数据（如登录 token、主题偏好）不属于本缓存，跳过
      }
    })
    
    return clearedCount
  }

  replace(key, value, minutes = 0) {
    const oldValue = this.get(key)
    this.set(key, value, minutes)
    return oldValue
  }

  memoize(key, callback, minutes = 0) {
    const cached = this.get(key)
    if (cached !== null) {
      return cached
    }
    
    const result = callback()
    this.set(key, result, minutes)
    return result
  }

  async memoizeAsync(key, callback, minutes = 0) {
    const cached = this.get(key)
    if (cached !== null) {
      return cached
    }
    
    const result = await callback()
    this.set(key, result, minutes)
    return result
  }
}

const cache = new Cache()

let baseURL = import.meta.env.VITE_API_URI || ''
let baseURLPromise = null
let baseURLResolved = !!baseURL

const initBaseURL = async () => {
  if (baseURLResolved) return
  
  if (baseURLPromise) {
    return baseURLPromise
  }

  baseURLPromise = (async () => {
    try {
      const api_uri = getSync('api_uri') || import.meta.env.VITE_API_URI
      if (api_uri) {
        baseURL = api_uri
        axiosInstance.defaults.baseURL = api_uri
      }
    } catch (error) {
      console.error('[Network] 初始化 baseURL 失败:', error)
    } finally {
      baseURLResolved = true
    }
    return baseURL
  })()

  return baseURLPromise
}

initBaseURL()

const waitingQueue = []

const waitForBaseURL = async () => {
  if (baseURLResolved && baseURL) {
    return baseURL
  }
  
  if (baseURLPromise) {
    return baseURLPromise
  }
  
  return new Promise((resolve) => {
    waitingQueue.push(resolve)
  })
}

const resolveWaitingQueue = () => {
  waitingQueue.forEach(resolve => resolve(baseURL))
  waitingQueue.length = 0
}

const logRequest = (method, url, data) => {
  if (DEV) {
    console.log(`[Request] ${method.toUpperCase()} ${url}`, data || '')
  }
}

const logResponse = (method, url, data, status) => {
  if (DEV) {
    console.log(`[Response] ${method.toUpperCase()} ${url} [${status}]`, data || '')
  }
}

const logError = (method, url, error) => {
  console.error(`[Request Error] ${method.toUpperCase()} ${url}`, error)
}

const pendingRequests = new Map()

const buildRequestKey = (method, url, data) => {
  return `${method}:${url}:${JSON.stringify(data)}`
}

const handleError = (error) => {
  const response = error.response
  const message = response?.data?.msg || response?.statusText || error.message || '请求失败'
  
  const errorInfo = {
    code: response?.status || -1,
    message,
    data: response?.data,
    url: error.config?.url
  }
  
  return Promise.reject(errorInfo)
}

const isRetryable = (error, method) => {
  if (method.toLowerCase() === 'post') {
    return false
  }
  
  if (error?.response) {
    const status = error.response.status
    return status >= 500 || status === 429
  }
  return error?.code === 'ECONNABORTED' || !error?.response
}

let isLoggingOut = false

const handleLogout = async () => {
  if (isLoggingOut) return
  isLoggingOut = true

  try {
    const TOKEN_NAME = getSync('token_name') || 'INIS_LOGIN_TOKEN'
    cache.del('user-info')
    utils.clear.cookie(TOKEN_NAME)
    
    try {
      await axios.post('/api/comm/logout', {}, { withCredentials: true })
    } catch (err) {
      console.error('登出接口调用失败：', err)
    }
    
    // 只在当前页面需要登录时才跳转到登录页
    // 如果当前页面不需要登录，清空登录状态即可，不需要强制跳转
    const router = await import('@/router')
    const currentRoute = router.default.currentRoute.value
    if (currentRoute.meta?.requiresAuth) {
      setTimeout(() => {
        window.location.href = '/login'
      }, 1500)
    }
  } finally {
    setTimeout(() => {
      isLoggingOut = false
    }, 3000)
  }
}

const requestWithRetry = async (method, url, dataOrParams, options = {}) => {
  await waitForBaseURL()
  
  if (!DEV && !baseURL) {
    throw new Error('请在配置文件中设置后端API地址（api_uri）')
  }

  const { skipRetry = false, skipToken = false, silentError = false, skipAuthLogout = false } = options
  
  const requestKey = buildRequestKey(method, url, dataOrParams)
  if (!options.skipDuplicate && pendingRequests.has(requestKey)) {
    return pendingRequests.get(requestKey)
  }

  const controller = new AbortController()
  const requestConfig = {
    baseURL: options.baseURL || baseURL,
    signal: controller.signal,
    ...options
  }

  const requestPromise = (async () => {
    let attempts = 0
    let lastError = null
    
    while (attempts <= (skipRetry ? 0 : MAX_RETRY)) {
      try {
        logRequest(method, url, dataOrParams)
        
        let response
        switch (method.toLowerCase()) {
          case 'get':
          case 'delete':
            response = await axiosInstance[method](url, { params: dataOrParams, ...requestConfig })
            break
          case 'post':
          case 'put':
          case 'patch':
            response = await axiosInstance[method](url, dataOrParams, requestConfig)
            break
          default:
            throw new Error(`不支持的请求方法: ${method}`)
        }

        logResponse(method, url, response, response.status)
        
        const responseData = response.data
        
        if (responseData?.code === 401) {
          // skipAuthLogout：由调用方自行处理 401（如 check-token 需要拿到原始 401 码做本地状态清理），
          // 这里不再触发全局登出，而是把 401 响应原样返回给调用方
          if (skipAuthLogout) {
            return responseData
          }
          handleLogout()
          return Promise.reject({
            code: 401,
            message: responseData?.msg || '登录已过期，请重新登录！',
            data: responseData?.data,
            url: response.config?.url
          })
        }

        if (responseData?.code !== 200 && !silentError) {
          console.warn(`[Business Error] ${method.toUpperCase()} ${url}`, responseData)
        }
        
        return responseData
        
      } catch (error) {
        lastError = error
        attempts++
        
        if (error?.code === 'ERR_CANCELED' || error?.name === 'AbortError') {
          throw error
        }

        if (response?.status === 401 || response?.data?.code === 401) {
          handleLogout()
          return Promise.reject({
            code: 401,
            message: response?.data?.msg || '登录已过期，请重新登录！',
            data: response?.data,
            url: error.config?.url
          })
        }

        if (attempts > (skipRetry ? 0 : MAX_RETRY) || !isRetryable(error, method)) {
          if (!silentError) {
            logError(method, url, error)
          }
          return handleError(error)
        }

        const delay = Math.pow(2, attempts) * 1000
        await new Promise(resolve => setTimeout(resolve, delay))
      }
    }
    
    return handleError(lastError)
  })()

  pendingRequests.set(requestKey, requestPromise)
  
  try {
    const result = await requestPromise
    return result
  } finally {
    pendingRequests.delete(requestKey)
  }
}

axiosInstance.interceptors.request.use(
  axiosConfig => {
    const { skipToken = false } = axiosConfig

    if (!skipToken) {
      const apiKey = getSync('api_key')
      if (!utils.is.empty(apiKey)) {
        axiosConfig.headers['i-api-key'] = apiKey
      }

      const TOKEN_NAME = getSync('token_name') || 'INIS_LOGIN_TOKEN'
      if (utils.has.cookie(TOKEN_NAME)) {
        const token = utils.get.cookie(TOKEN_NAME)
        if (!utils.is.empty(token)) {
          axiosConfig.headers.Authorization = token
        }
      }
    }

    axiosConfig.headers['X-CSRF-Token'] = utils.get.cookie('csrf_token') || ''

    const isValidUrl = API_WHITELIST.some(prefix => axiosConfig.url?.startsWith(prefix))
    if (!axiosConfig.url || !isValidUrl) {
      console.warn(`[Security] 请求路径不合法: ${axiosConfig.url}`)
    }

    if (axiosConfig.data instanceof FormData) {
      delete axiosConfig.headers['Content-Type']
    }

    return axiosConfig
  },
  error => Promise.reject(error)
)

axiosInstance.interceptors.response.use(
  response => {
    return response
  },
  error => {
    const response = error.response
    if (response?.status === 401 || response?.data?.code === 401) {
      handleLogout()
      return Promise.reject({
        code: 401,
        message: response?.data?.msg || '登录已过期，请重新登录！',
        data: response?.data,
        url: error.config?.url
      })
    }
    return Promise.reject(error)
  }
)

// ============ 带缓存的 GET ============
// 统一的「缓存 + 并发去重 + 后台刷新(SWR)」读取入口。
// 各页面不再各自手写 cache.get / cache.set 样板代码。
//
//   const res = await request.cached('/api/article/all', params, { minutes: 10 })
//
// 选项：
//   minutes    缓存有效期（分钟），默认 5
//   key        自定义缓存键，默认由 url + params 生成
//   swr        true 时命中缓存也在后台静默刷新，下次访问即为最新数据
//   force      true 时跳过缓存强制请求（用于下拉刷新等场景）
const CACHED_PREFIX = '__inis_cache_'
const inflight = new Map()

const buildCacheKey = (url, params) => {
  const keys = Object.keys(params || {}).sort()
  const normalized = keys.map(k => `${k}=${params[k]}`).join('&')
  return `${CACHED_PREFIX}${url}${normalized ? `?${normalized}` : ''}`
}

const cachedGet = async (url, params = {}, options = {}) => {
  const {
    minutes = 5,
    key,
    swr = false,
    force = false,
    ...requestOptions
  } = options

  const cacheKey = key || buildCacheKey(url, params)

  if (!force) {
    const hit = cache.get(cacheKey)
    if (hit !== null && hit !== undefined) {
      // 后台静默刷新，不阻塞当前渲染
      if (swr && !inflight.has(cacheKey)) {
        const task = requestWithRetry('get', url, params, { silentError: true, ...requestOptions })
          .then(fresh => {
            if (fresh?.code === 200) cache.set(cacheKey, fresh, minutes)
            return fresh
          })
          .catch(() => null)
          .finally(() => inflight.delete(cacheKey))
        inflight.set(cacheKey, task)
      }
      return hit
    }
  }

  // 并发去重：同一时刻的相同请求共享一个 Promise
  if (inflight.has(cacheKey)) {
    return inflight.get(cacheKey)
  }

  const task = requestWithRetry('get', url, params, requestOptions)
    .then(res => {
      // 只缓存成功响应，避免把错误态写进缓存
      if (res?.code === 200) cache.set(cacheKey, res, minutes)
      return res
    })
    .finally(() => inflight.delete(cacheKey))

  inflight.set(cacheKey, task)
  return task
}

const request = {
  get: async (url, params = {}, options = {}) => {
    return requestWithRetry('get', url, params, options)
  },

  cached: cachedGet,

  // 主动失效缓存：传 url 前缀或完整缓存键
  invalidate: (urlOrKey, params) => {
    if (params) {
      cache.del(buildCacheKey(urlOrKey, params))
      return
    }
    const target = urlOrKey.startsWith(CACHED_PREFIX)
      ? urlOrKey
      : `${CACHED_PREFIX}${urlOrKey}`
    cache.keys().forEach(k => {
      if (k === target || k.startsWith(target)) cache.del(k)
    })
  },

  delete: async (url, params = {}, options = {}) => {
    return requestWithRetry('delete', url, params, options)
  },

  put: async (url, data = {}, options = {}) => {
    return requestWithRetry('put', url, data, options)
  },

  post: async (url, data = {}, options = {}) => {
    return requestWithRetry('post', url, data, options)
  },

  patch: async (url, data = {}, options = {}) => {
    return requestWithRetry('patch', url, data, options)
  },

  all: async (array) => {
    await waitForBaseURL()
    
    if (!DEV && !baseURL) {
      return Promise.reject(new Error('请在配置文件中设置后端API地址（api_uri）'))
    }
    
    return axios.all(array.map(req => {
      if (req && typeof req.then === 'function') {
        return req
      }
      return Promise.reject(new Error('request.all 需要传入 Promise 数组'))
    }))
  },

  createAbortController: () => new AbortController(),

  getBaseURL: () => baseURL,

  setBaseURL: (url) => {
    baseURL = url
    axiosInstance.defaults.baseURL = url
    baseURLResolved = true
    resolveWaitingQueue()
  },

  axios: axiosInstance
}

const uploadImage = async (options = {}) => {
  const { 
    maxSize = 5 * 1024 * 1024, 
    accept = 'image/*',
    onSuccess, 
    onError 
  } = options

  return new Promise((resolve, reject) => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = accept

    let settled = false

    const cleanup = () => {
      input.removeEventListener('change', handleChange)
      input.removeEventListener('cancel', handleCancel)
      if (input.parentNode) {
        document.body.removeChild(input)
      }
    }

    const handleCancel = () => {
      if (settled) return
      settled = true
      cleanup()
      reject(new Error('已取消选择'))
    }

    const handleChange = async () => {
      if (settled) return
      settled = true
      try {
        if (!input.files || input.files.length === 0) {
          cleanup()
          reject(new Error('未选择文件'))
          return
        }

        const file = input.files[0]

        if (file.size > maxSize) {
          cleanup()
          const error = new Error(`文件大小超过限制（最大 ${maxSize / 1024 / 1024}MB）`)
          if (onError) onError(error)
          reject(error)
          return
        }

        const validTypes = ['image/jpeg', 'image/png', 'image/gif', 'image/webp']
        if (!validTypes.includes(file.type)) {
          cleanup()
          const error = new Error('只支持 JPG、PNG、GIF、WebP 格式图片')
          if (onError) onError(error)
          reject(error)
          return
        }

        await checkFileType([file.name])

        const params = new FormData()
        params.append('file', file)

        const result = await request.post('/api/attachment/batch', params)

        if (result.code !== 200) {
          const error = new Error(result.msg || '上传失败')
          if (onError) onError(error)
          reject(error)
          return
        }

        const fullUrl = result.data.results?.[0]?.full_url
        if (!fullUrl) {
          const error = new Error('上传失败，未返回文件链接')
          if (onError) onError(error)
          reject(error)
          return
        }

        if (onSuccess) onSuccess(fullUrl)
        resolve(fullUrl)
      } catch (error) {
        if (onError) onError(error)
        reject(error)
      } finally {
        cleanup()
      }
    }

    input.addEventListener('change', handleChange)
    input.addEventListener('cancel', handleCancel)
    document.body.appendChild(input)
    input.click()
  })
}

const checkFileType = async (fileNames) => {
  try {
    const result = await request.post('/api/attachment/checkType', {
      file_names: fileNames
    })
    if (result.code === 200 && result.data) {
      const disallowedFiles = result.data.results?.filter(item => !item.is_allowed) || []
      if (disallowedFiles.length > 0) {
        const messages = disallowedFiles.map(item => `${item.file_name}: ${item.message}`)
        throw new Error(messages.join('；'))
      }
      return result.data
    }
    throw new Error(result.msg || '文件类型检查失败')
  } catch (error) {
    throw error
  }
}

export { cache, request, uploadImage, checkFileType }

export default { cache, request, uploadImage, checkFileType }