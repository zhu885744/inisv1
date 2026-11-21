/**
 * 轻量缓存工具（localStorage + 内存镜像）
 * 参考主题实现方式：set 第三个参数单位为「分钟」，写入结构 { data, expire, timestamp }
 */
const MAX_CACHE_SIZE = 500

class Cache {
  constructor() {
    this.testKey = '__cache_test__'
    this.maxSize = MAX_CACHE_SIZE
    this._available = null
    this._memo = new Map()
  }

  isAvailable() {
    if (this._available !== null) return this._available
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

  isExpired(cacheData) {
    return cacheData && typeof cacheData === 'object' && cacheData.expire && Date.now() > cacheData.expire
  }

  get(key) {
    if (!this.isValidKey(key) || !this.isAvailable()) return null
    if (this._memo.has(key)) {
      const hit = this._memo.get(key)
      if (hit.expire && Date.now() > hit.expire) {
        this.del(key)
        return null
      }
      return hit.data
    }
    const raw = localStorage.getItem(key)
    if (raw === null || raw === '') return null
    try {
      const parsed = JSON.parse(raw)
      if (this.isExpired(parsed)) {
        this.del(key)
        return null
      }
      const data = parsed && parsed.data !== undefined ? parsed.data : parsed
      this._memo.set(key, { data, expire: parsed?.expire || 0 })
      return data
    } catch {
      return raw || null
    }
  }

  set(key, value, minutes = 0) {
    if (!this.isValidKey(key) || !this.isAvailable()) return
    const dataToStore = { data: value, timestamp: Date.now() }
    if (!isNaN(Number(minutes)) && Number(minutes) > 0) {
      dataToStore.expire = Date.now() + Number(minutes) * 60 * 1000
    }
    this._memo.set(key, { data: value, expire: dataToStore.expire || 0 })
    try {
      localStorage.setItem(key, JSON.stringify(dataToStore))
    } catch {
      this.cleanupOldest()
      try {
        localStorage.setItem(key, JSON.stringify(dataToStore))
      } catch {
        // 降级为内存缓存
      }
    }
  }

  cleanupOldest() {
    if (!this.isAvailable()) return
    const items = []
    this.keys().forEach((key) => {
      try {
        const parsed = JSON.parse(localStorage.getItem(key))
        if (parsed && typeof parsed === 'object' && parsed.timestamp) {
          items.push({ key, timestamp: parsed.timestamp })
        }
      } catch {}
    })
    items.sort((a, b) => a.timestamp - b.timestamp)
    const removeCount = Math.max(1, Math.floor(items.length * 0.2))
    items.slice(0, removeCount).forEach((i) => this.del(i.key))
  }

  del(key) {
    if (!this.isValidKey(key) || !this.isAvailable()) return
    this._memo.delete(key)
    localStorage.removeItem(key)
  }

  delMultiple(keys) {
    if (!Array.isArray(keys)) return false
    keys.forEach((k) => this.del(k))
    return true
  }

  keys() {
    if (!this.isAvailable()) return []
    return Object.keys(localStorage)
  }

  isOwnedKey(key) {
    if (key === this.testKey) return false
    try {
      const parsed = JSON.parse(localStorage.getItem(key))
      return !!(parsed && typeof parsed === 'object' && parsed.timestamp)
    } catch {
      return false
    }
  }

  clear() {
    if (!this.isAvailable()) return
    this.keys().forEach((k) => {
      if (this.isOwnedKey(k)) this.del(k)
    })
  }
}

export const cache = new Cache()
export default cache
