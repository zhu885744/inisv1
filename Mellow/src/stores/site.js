import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getConfig } from '@/api/config'
import { cache } from '@/utils/cache'

/**
 * 站点配置（Mellow_functions）全局单例
 *
 * 布局、悬浮按钮、侧栏、协议弹窗等多处都需要同一份配置，
 * 统一由本 store 负责：并发去重 + 本地缓存（30 分钟），一次性消除重复请求。
 */
const CONFIG_KEY = 'Mellow_functions'
const CACHE_MINUTES = 30

// 进行中的请求（模块级，保证跨组件/跨 store 调用共享同一 Promise）
let pending = null

export const useSiteStore = defineStore('site', () => {
  const config = ref({})
  const loaded = ref(false)

  /**
   * 读取站点配置
   * @param {boolean} force 为 true 时忽略缓存强制刷新
   */
  function load(force = false) {
    if (pending) return pending
    if (!force) {
      const cached = cache.get(CONFIG_KEY)
      if (cached && typeof cached === 'object') {
        config.value = cached
        loaded.value = true
        return Promise.resolve(cached)
      }
    }
    pending = (async () => {
      try {
        // silent：配置读取失败时不打扰用户，各调用方均有默认值兜底
        const res = await getConfig(CONFIG_KEY, { silent: true })
        const info = res?.data?.json ?? res?.data ?? {}
        const data = info && typeof info === 'object' ? info : {}
        config.value = data
        cache.set(CONFIG_KEY, data, CACHE_MINUTES)
        return data
      } catch {
        // 请求失败：保留已有配置（可能是内存中的旧值）
        return config.value
      } finally {
        loaded.value = true
        pending = null
      }
    })()
    return pending
  }

  /** 保存后使缓存失效并刷新（供后台配置页调用） */
  function invalidate() {
    cache.del(CONFIG_KEY)
    return load(true)
  }

  return { config, loaded, load, invalidate }
})
