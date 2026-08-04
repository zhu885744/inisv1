// 配置状态管理
import { defineStore } from 'pinia'
import { cache } from '@/utils/network'
import utils from '@/utils/utils'
import { request as axios } from '@/utils/network'
import { route } from '@/utils/app'

// ==================== 新增：配置相关工具函数 ====================
// 获取指定配置
const one = (state = {}, key = '') => {
  if (utils.is.empty(key)) return {}

  axios.get('/api/config/one', { key }).then(({ code, data }) => {
    if (code !== 200) return (state[key] = {})

    // 缓存数据
    cache.set(`config[${key}]`, data, globalThis?.inis?.cache || 3600)
    state[key] = data
  })

  return state[key]
}

// 消除异步污染
const infect = (cacheName, promise) => {
  let cacheData = cache.get(cacheName) || {
    status: 'wait',
    value: null,
  }

  // 成功 - 读取缓存
  if (cacheData?.status === 'success') {
    return cacheData.value
  }
  // 失败 - 抛出异常
  else if (cacheData?.status === 'error') {
    throw cacheData.value
  }

  // 发送真实请求 - 抛出错误
  throw promise
    .then(({ code, data }) => {
      if (code !== 200) {
        cacheData = { status: 'error', value: null }
        // 缓存数据
        cache.set(cacheName, cacheData, globalThis?.inis?.cache || 3600)
        return cacheData
      }

      cacheData = { status: 'success', value: data }
      // 缓存数据
      cache.set(cacheName, cacheData, globalThis?.inis?.cache || 3600)
      return cacheData
    })
    .catch((error) => {
      cacheData = { status: 'error', value: error }
      cache.set(cacheName, cacheData, globalThis?.inis?.cache || 3600)
      throw error
    })
}

// ==================== 原有：登录相关工具函数 ====================
// 校验token
const checkToken = (state = {}) => {
  const cacheName = 'user-info'
  // 缓存中存在用户信息
  if (cache.has(cacheName)) return (state.login.finish = true)

  axios.post('/api/comm/check-token').then(({ code, msg, data }) => {
    if (code === 412) return
    if (code === 401) return logout(state)
    if (code !== 200) return notyf.error(msg)

    state.login.user = data.user
    state.login.finish = true
    cache.set(cacheName, data.user, 10)
  })
}

// 登出
const logout = (state = {}, path = null) => {
  // 清除登录信息
  state.login.user = {}
  state.login.finish = false
  cache.del('user-info')
  utils.clear.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN')

  axios.del('/api/comm/logout').then((res) => res)

  // 返回首页
  if (!utils.is.empty(path))
    setTimeout(() => {
      route.push(path)
    }, 300)
}

// ==================== 新增：异常捕获执行函数 ====================
const run = (fn) => {
  try {
    fn()
  } catch (err) {
    if (err instanceof Promise) {
      err.then(fn, fn)
    }
  }
}

// ==================== 合并后的 Store 定义 ====================
// 1. 配置管理 Store（保留原有 config store 逻辑）
export const useConfigStore = defineStore('config', {
  state: () => ({
    ALLOW_REGISTER: cache.get('config[ALLOW_REGISTER]'),
  }),
  getters: {
    getAllowRegister(state = {}) {
      const cacheName = 'config[ALLOW_REGISTER]'
      if (cache.has(cacheName)) return (state.ALLOW_REGISTER = cache.get(cacheName))
      return one(this, 'ALLOW_REGISTER')
    },
  },
  actions: {
    // 获取指定配置（测试用）
    test() {
      const promise = axios.get('/api/config/one', { key: 'ALLOW_REGISTER' })
      return infect('config[TEST]', promise)
    },
    // 暴露 run 方法（可选）
    run(fn) {
      run(fn)
    },
  },
})

// 2. 通用状态 Store（保留原有 comm store 逻辑）
export const useCommStore = defineStore('comm', {
  state: () => ({
    // 登录注册重置密码的状态
    auth: {
      login: false,
      reset: false,
      register: false,
    },
    login: {
      // 登录状态 - 是否登录完成
      finish: false,
      // 当前登录的用户信息
      user: cache.get('user-info'),
    },
    progress: false,
    nav: {
      title: '',
    },
    // 新增：配置相关状态（可选，如需在 comm store 中管理配置可保留）
    ALLOW_REGISTER: cache.get('config[ALLOW_REGISTER]'),
  }),
  getters: {
    // 获取登录信息
    getLogin: (state = {}) => {
      // 校验token
      checkToken(state)
      // 返回登录信息
      return state.login
    },
    // 新增：配置获取 getter（如需在 comm store 中获取配置可保留）
    getAllowRegister(state = {}) {
      const cacheName = 'config[ALLOW_REGISTER]'
      if (cache.has(cacheName)) return (state.ALLOW_REGISTER = cache.get(cacheName))
      return one(this, 'ALLOW_REGISTER')
    },
  },
  actions: {
    // 切换登录注册重置密码的状态
    switchAuth(name = 'login', bool = true) {
      for (const key in this.auth) {
        this.auth[key] = key === name ? bool : false
      }
    },
    // 登出
    logout(path = null) {
      logout(this, path)
    },
    // 新增：配置测试方法（可选）
    configTest() {
      const promise = axios.get('/api/config/one', { key: 'ALLOW_REGISTER' })
      return infect('config[TEST]', promise)
    },
    // 新增：异常捕获执行方法（可选）
    run(fn) {
      run(fn)
    },
  },
})