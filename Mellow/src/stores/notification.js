import { defineStore } from 'pinia'
import { ref } from 'vue'
import { unreadCount, listNotifications, readAllNotifications } from '@/api/tags'

const POLL_INTERVAL = 60 * 1000

export const useNotificationStore = defineStore('notification', () => {
  const count = ref(0)
  // 最新一条通知（优先取未读的最新一条，无未读时取已读的最新一条）
  const latest = ref(null)
  let timer = null
  let visibilityBound = false

  const pickFirst = (res) => {
    const payload = res?.data?.data
    const list = Array.isArray(payload) ? payload : payload?.data || []
    return list[0] || null
  }

  // unread-count 的 data 兼容：直接是数字 / { count } / { total } / 嵌套 { data: { count } }
  const pickCount = (res) => {
    const raw = res?.data
    if (typeof raw === 'number') return raw
    return Number(raw?.count ?? raw?.total ?? raw?.data?.count) || 0
  }

  async function refresh() {
    try {
      const res = await unreadCount()
      // 后端返回结构: { code, data: { count } }
      count.value = pickCount(res)
      // 只拉取一条：有未读优先取未读最新，否则取全部最新（比原来两次列表请求少一次）
      const params = { page: 1, limit: 1, order: 'create_time desc' }
      if (count.value > 0) params.is_read = 0
      const list = await listNotifications(params)
      const item = pickFirst(list)
      // 请求成功但无数据时保留旧值，避免误显示“暂无消息”
      if (item) latest.value = item
    } catch {
      // 失败时保留旧值
    }
  }

  // 页面不可见时静默跳过，可见时立即补一次刷新
  function onVisibilityChange() {
    if (!timer) return
    if (!document.hidden) refresh()
  }

  function bindVisibility() {
    if (visibilityBound || typeof document === 'undefined') return
    visibilityBound = true
    document.addEventListener('visibilitychange', onVisibilityChange)
  }

  function unbindVisibility() {
    if (!visibilityBound || typeof document === 'undefined') return
    visibilityBound = false
    document.removeEventListener('visibilitychange', onVisibilityChange)
  }

  function startPolling() {
    // 侧栏与移动端底栏都会在挂载时调用：已经在轮询就不要重启，
    // 否则同一轮挂载会立刻多发一次 unread-count + 列表请求（移动端侧栏只是被 CSS 隐藏，组件仍然挂载）
    if (timer) return
    refresh()
    timer = setInterval(() => {
      // 后台标签页不轮询，避免无效请求
      if (typeof document !== 'undefined' && document.hidden) return
      refresh()
    }, POLL_INTERVAL)
    bindVisibility()
  }

  function stopPolling() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    unbindVisibility()
  }

  function clearCount() {
    count.value = 0
  }

  async function readAll() {
    await readAllNotifications()
    count.value = 0
    await refresh()
  }

  return { count, latest, refresh, startPolling, stopPolling, clearCount, readAll }
})
