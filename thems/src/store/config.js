// 配置状态管理
import { defineStore } from 'pinia'
import { cache } from '@/utils/network'
import utils from '@/utils/utils'
import { request as axios } from '@/utils/network'

// ==================== 配置相关工具函数 ====================
// 记录进行中的配置请求，避免同一 key 被并发重复请求
const configInflight = new Map()

// 获取指定配置（异步写入 state，命中缓存则不发请求）
const one = (state = {}, key = '') => {
  if (utils.is.empty(key)) return {}

  const cacheName = `config[${key}]`

  const cached = cache.get(cacheName)
  if (cached) {
    state[key] = cached
    return state[key]
  }

  if (!configInflight.has(key)) {
    const task = axios
      .get('/api/config/one', { key })
      .then(({ code, data }) => {
        if (code !== 200) {
          state[key] = {}
          return
        }
        // 缓存数据（第三个参数单位为分钟）
        cache.set(cacheName, data, globalThis?.inis?.cache || 60)
        state[key] = data
      })
      .catch(() => {
        state[key] = {}
      })
      .finally(() => configInflight.delete(key))

    configInflight.set(key, task)
  }

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

// ==================== 异常捕获执行函数 ====================
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
    // 纯读取：模板中的 v-if 会在每次渲染时求值，
    // 因此这里不发请求，仅返回已加载的配置（由 loadAllowRegister 负责拉取）
    getAllowRegister: (state) => state.ALLOW_REGISTER,
  },
  actions: {
    // 加载注册开关配置（命中缓存不发请求，并发调用自动合并）
    loadAllowRegister() {
      return one(this, 'ALLOW_REGISTER')
    },
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

// 说明：此文件曾额外定义过一个 id 同为 'comm' 的 store，
// 与 store/comm.js 冲突（Pinia 以 id 注册，先导入者生效），且无人使用，已移除。
// 通用状态请统一从 store/comm.js 引入 useCommStore。