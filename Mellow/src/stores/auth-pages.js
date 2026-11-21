import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getAuthPagesFlat } from '@/api/auth-pages'
import { cache } from '@/utils/cache'

const CACHE_KEY = 'admin-auth-pages'

export const useAuthPagesStore = defineStore('auth-pages', () => {
  const list = ref([])
  const loading = ref(false)
  const loaded = ref(false)

  // 扁平化列表（参考主题 useAuthPagesStore().getFlat）
  const getFlat = computed(() => list.value)

  async function ensureLoaded() {
    // 优先读缓存
    if (!list.value.length) {
      const cached = cache.get(CACHE_KEY)
      if (cached && Array.isArray(cached)) {
        list.value = cached
        loaded.value = true
      }
    }
    if (loaded.value && list.value.length) return list.value

    if (loading.value) return list.value
    loading.value = true
    try {
      const res = await getAuthPagesFlat()
      const data = res?.data?.data || res?.data || []
      const flat = Array.isArray(data) ? data : []
      list.value = flat
      loaded.value = true
      cache.set(CACHE_KEY, flat, 30)
    } catch {
      list.value = []
    } finally {
      loading.value = false
    }
    return list.value
  }

  return { list, loading, loaded, getFlat, ensureLoaded }
})
