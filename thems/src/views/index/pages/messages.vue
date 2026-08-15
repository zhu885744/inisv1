<template>
  <!-- 顶部头部卡片：标题 + 操作按钮 -->
  <div class="card mt-2">
    <div class="card-body">
      <div class="d-flex flex-wrap align-items-center justify-content-between gap-3">
        <div>
          <h1 class="h4 mb-0 d-flex align-items-center gap-2">
            <i class="bi bi-bell-fill text-primary"></i>消息通知
          </h1>
        </div>
        <div class="d-flex gap-2">
          <button class="btn btn-outline-secondary btn-sm" @click="markAllRead" :disabled="loading || unreadCount === 0">
            <i class="bi bi-check2-all me-1"></i>全部已读
          </button>
          <button class="btn btn-outline-danger btn-sm" @click="clearAll" :disabled="loading || filteredMessages.length === 0">
            <i class="bi bi-trash3 me-1"></i>清空当前通知
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- 下面网格：左侧导航card | 右侧内容card，两个独立卡片 -->
  <div class="row g-2 mt-1">
    <!-- 左侧：侧边栏导航卡片 -->
    <div class="col-12 col-md-3">
      <div class="card h-100">
        <div class="card-body">
          <div class="nav-sidebar">
            <div class="nav flex-column nav-pills">
              <button
                class="nav-link text-start d-flex align-items-center justify-content-between"
                :class="{ active: activeTab === 'all' }"
                @click="activeTab = 'all'"
              >
                <span><i class="bi bi-inbox me-2"></i>全部</span>
              </button>
              <button
                class="nav-link text-start d-flex align-items-center justify-content-between"
                :class="{ active: activeTab === 'unread' }"
                @click="activeTab = 'unread'"
              >
                <span><i class="bi bi-envelope me-2"></i>未读</span>
              </button>
              <button
                class="nav-link text-start d-flex align-items-center justify-content-between"
                :class="{ active: activeTab === 'comment' }"
                @click="activeTab = 'comment'"
              >
                <span><i class="bi bi-chat-dots me-2"></i>回复</span>
              </button>
              <button
                class="nav-link text-start d-flex align-items-center justify-content-between"
                :class="{ active: activeTab === 'like' }"
                @click="activeTab = 'like'"
              >
                <span><i class="bi bi-heart me-2"></i>点赞</span>
              </button>
              <button
                class="nav-link text-start d-flex align-items-center justify-content-between"
                :class="{ active: activeTab === 'collection' }"
                @click="activeTab = 'collection'"
              >
                <span><i class="bi bi-bookmark me-2"></i>收藏</span>
              </button>
              <button
                class="nav-link text-start d-flex align-items-center justify-content-between"
                :class="{ active: activeTab === 'follow' }"
                @click="activeTab = 'follow'"
              >
                <span><i class="bi bi-person-plus me-2"></i>关注</span>
              </button>
              <button
                class="nav-link text-start d-flex align-items-center justify-content-between"
                :class="{ active: activeTab === 'system' }"
                @click="activeTab = 'system'"
              >
                <span><i class="bi bi-gear me-2"></i>系统</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧：消息内容卡片 -->
    <div class="col-12 col-md-9">
      <div class="card h-100">
        <div class="card-body">
          <!-- 加载中 -->
          <div v-if="loading" class="text-center py-5">
            <div class="spinner-border text-primary" role="status">
              <span class="visually-hidden">加载中...</span>
            </div>
            <p class="mt-2 text-muted small">加载消息中...</p>
          </div>

          <!-- 消息列表 -->
          <div v-else-if="filteredMessages.length > 0" class="wx-message-list">
            <div
              v-for="msg in filteredMessages"
              :key="msg.id"
              class="wx-message-item border rounded mb-2 p-3"
              :class="{
                'border-start border-start-4 border-primary': !msg.is_read && msg.is_read !== undefined,
                'bg-light-subtle': !msg.is_read && msg.is_read !== undefined,
                'opacity-75': msg.is_read === 1
              }"
              @click="handleItemClick(msg)"
            >
              <div class="d-flex gap-3">
                <!-- 头像 / 图标 -->
                <div class="flex-shrink-0">
                  <div
                    class="wx-avatar d-flex align-items-center justify-content-center rounded-circle"
                    :class="getAvatarClass(msg.type)"
                    style="width: 44px; height: 44px; font-size: 1.2rem;"
                  >
                    <img v-if="msg.result?.from_user?.avatar" :src="msg.result?.from_user?.avatar" alt="" style="width:44px;height:44px;border-radius:50%;object-fit:cover" />
                    <i v-else :class="getIcon(msg.type)"></i>
                  </div>
                </div>

                <!-- 消息内容 -->
                <div class="flex-grow-1 min-w-0">
                  <div class="d-flex flex-wrap align-items-start justify-content-between gap-2">
                    <div>
                      <span class="fw-semibold">{{ msg.title }}</span>
                      <span v-if="!msg.is_read && msg.is_read !== undefined" class="badge bg-primary ms-2 small">未读</span>
                      <span class="badge bg-light text-secondary border ms-1 small">{{ getTypeLabel(msg.type, msg.uid) }}</span>
                    </div>
                    <small class="text-muted flex-shrink-0">{{ formatTime(msg.create_time) }}</small>
                  </div>

                  <p class="mb-1 text-body-secondary small" v-html="formatContent(msg.content)"></p>

                  <!-- 操作按钮 -->
                  <div class="d-flex gap-3 mt-2">
                    <button
                      v-if="!msg.is_read && msg.is_read !== undefined"
                      class="btn btn-link btn-sm p-0 text-primary text-decoration-none"
                      @click.stop="markRead(msg.id)"
                    >
                      <i class="bi bi-check2 me-1"></i>标记已读
                    </button>
                    <button
                      class="btn btn-link btn-sm p-0 text-danger text-decoration-none"
                      @click.stop="removeMessage(msg.id)"
                    >
                      <i class="bi bi-trash3 me-1"></i>删除
                    </button>
                    <button
                      v-if="getActionLink(msg)"
                      class="btn btn-link btn-sm p-0 text-decoration-none"
                      @click.stop="handleAction(msg)"
                    >
                      <i class="bi bi-box-arrow-up-right me-1"></i>查看详情
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <!-- 分页 -->
            <div v-if="totalPages > 1" class="d-flex justify-content-center mt-3">
              <nav>
                <ul class="pagination pagination-sm">
                  <li class="page-item" :class="{ disabled: currentPage <= 1 }">
                    <button class="page-link" @click="changePage(currentPage - 1)">上一页</button>
                  </li>
                  <li class="page-item disabled">
                    <span class="page-link">{{ currentPage }} / {{ totalPages }}</span>
                  </li>
                  <li class="page-item" :class="{ disabled: currentPage >= totalPages }">
                    <button class="page-link" @click="changePage(currentPage + 1)">下一页</button>
                  </li>
                </ul>
              </nav>
            </div>
          </div>

          <!-- 空状态 -->
          <div v-else class="text-center py-5">
            <i class="bi bi-inbox display-4 d-block mb-3 opacity-25"></i>
            <p class="h5 mb-1">暂无消息</p>
            <p class="text-muted small">当前分类下没有消息，稍后再来看看吧</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { request } from '@/utils/network'

const router = useRouter()

// ---------- 状态 ----------
const activeTab = ref('all')
const messages = ref([])
const loading = ref(false)
const currentPage = ref(1)
const totalCount = ref(0)
const totalPages = ref(1)
const unreadCount = ref(0)
const pageSize = 20

// ---------- API 请求 ----------
const fetchNotifications = async () => {
  loading.value = true
  try {
    let params = {
      page: currentPage.value,
      size: pageSize,
      order: 'create_time desc',
    }

    if (activeTab.value === 'unread') {
      params.is_read = 0
    } else if (activeTab.value !== 'all') {
      // collection 映射为 collect
      params.type = activeTab.value === 'collection' ? 'collect' : activeTab.value
    }

    const res = await request.get('/api/notification/list', params)
    const result = res?.data?.data || res?.data

    if (result?.data) {
      messages.value = Array.isArray(result.data) ? result.data : []
      totalCount.value = result.count || 0
      totalPages.value = result.page || 1
    } else if (Array.isArray(result)) {
      messages.value = result
      totalCount.value = result.length
      totalPages.value = 1
    } else {
      messages.value = []
      totalCount.value = 0
      totalPages.value = 1
    }
  } catch (err) {
    console.error('获取通知列表失败:', err)
    messages.value = []
  } finally {
    loading.value = false
  }
}

const fetchUnreadCount = async () => {
  try {
    const res = await request.get('/api/notification/unread-count')
    const data = res.data?.data || res.data
    unreadCount.value = data?.count || 0
  } catch (err) {
    console.error('获取未读通知数失败:', err)
  }
}

// 通知导航栏刷新未读角标（通过事件联动）
const notifyUnreadChange = () => {
  window.dispatchEvent(new CustomEvent('notification-unread-change'))
}

const markRead = async (id) => {
  try {
    await request.put('/api/notification/read', { id })
    const msg = messages.value.find(m => m.id === id)
    if (msg && msg.is_read !== undefined) {
      msg.is_read = 1
      unreadCount.value = Math.max(0, unreadCount.value - 1)
    }
    notifyUnreadChange()
  } catch (err) {
    console.error('标记已读失败:', err)
  }
}

const markAllRead = async () => {
  try {
    await request.put('/api/notification/read-all')
    messages.value.forEach(m => {
      if (m.is_read !== undefined) m.is_read = 1
    })
    unreadCount.value = 0
    notifyUnreadChange()
  } catch (err) {
    console.error('全部标记已读失败:', err)
  }
}

const removeMessage = async (id) => {
  try {
    await request.delete('/api/notification/remove', { ids: String(id) })
    messages.value = messages.value.filter(m => m.id !== id)
    totalCount.value = Math.max(0, totalCount.value - 1)
    await fetchUnreadCount()
    notifyUnreadChange()
  } catch (err) {
    console.error('删除消息失败:', err)
  }
}

const clearAll = async () => {
  if (!confirm('确定要清空当前分类下的所有消息吗？此操作不可恢复。')) return

  try {
    let params = {}
    if (activeTab.value !== 'all' && activeTab.value !== 'unread') {
      params.type = activeTab.value === 'collection' ? 'collect' : activeTab.value
    }
    await request.delete('/api/notification/remove-all', params)
    messages.value = []
    totalCount.value = 0
    await fetchUnreadCount()
    notifyUnreadChange()
  } catch (err) {
    console.error('清空消息失败:', err)
  }
}

// ---------- 计算属性 ----------
const filteredMessages = computed(() => messages.value)

// ---------- 辅助方法 ----------
const getIcon = (type) => {
  const map = {
    comment: 'bi-chat-dots',
    like: 'bi-heart',
    collect: 'bi-bookmark',
    follow: 'bi-person-plus',
    system: 'bi-gear',
  }
  return map[type] || 'bi-bell'
}

const getTypeLabel = (type, uid) => {
  // 广播通知（uid=0）显示为"系统公告"
  if (uid === 0) return '系统'
  const map = {
    comment: '回复',
    like: '点赞',
    collect: '收藏',
    follow: '关注',
    system: '系统',
  }
  return map[type] || type
}

const getAvatarClass = (type) => {
  const map = {
    comment: 'bg-info-subtle text-info',
    like: 'bg-danger-subtle text-danger',
    collect: 'bg-warning-subtle text-warning',
    follow: 'bg-success-subtle text-success',
    system: 'bg-primary-subtle text-primary',
  }
  return map[type] || 'bg-secondary-subtle text-secondary'
}

const formatTime = (ts) => {
  if (!ts) return ''
  const date = new Date(ts * 1000)
  const now = new Date()
  const diff = now - date
  const mins = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins}分钟前`
  if (hours < 24) return `${hours}小时前`
  if (days < 7) return `${days}天前`

  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

const getActionLink = (msg) => {
  if (!msg.bind_id || !msg.bind_type) return null

  switch (msg.bind_type) {
    case 'article':
      return `/archives/${msg.bind_id}`
    case 'page':
      return `/${msg.bind_id}`
    case 'moments':
      return `/moments/${msg.bind_id}`
    case 'comment':
      return null
    case 'user':
      return `/author/${msg.bind_id}`
    default:
      return null
  }
}

const handleAction = (msg) => {
  const link = getActionLink(msg)
  if (link) {
    markRead(msg.id).finally(() => {
      if (msg.bind_type === 'article') {
        router.push({ name: '文章详情', params: { id: String(msg.bind_id) } })
      } else if (msg.bind_type === 'moments') {
        router.push({ name: '动态详情', params: { id: String(msg.bind_id) } })
      } else if (msg.bind_type === 'user') {
        router.push({ name: '用户主页', params: { id: String(msg.bind_id) } })
      } else {
        router.push(link)
      }
    })
  }
}

const handleItemClick = (msg) => {
  if (!msg.is_read && msg.is_read !== undefined) {
    markRead(msg.id)
  }
}

const changePage = (page) => {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  fetchNotifications()
}

// ---------- ★ 新增：HTML 转义和 Emoji 解析 ----------
// 转义 HTML 特殊字符，防止 XSS
const escapeHtml = (text) => {
  if (!text) return ''
  const map = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#039;'
  }
  return text.replace(/[&<>"']/g, (m) => map[m])
}

// 将 [emoji:图片链接] 格式替换为 img 标签
const formatContent = (content) => {
  if (!content) return ''
  // 先转义，再替换 emoji
  const escaped = escapeHtml(content)
  const withBr = escaped.replace(/\n/g, '<br>')
  return withBr.replace(
    /\[emoji:([^\]]+)\]/g,
    (match, url) => {
      // 对 URL 做简单过滤，只允许 http/https，防止 javascript: 等危险协议
      const sanitizedUrl = url.trim()
      if (!/^https?:\/\/./i.test(sanitizedUrl)) {
        return match // 不安全的 URL 保留原样
      }
      // 再次转义 URL 中的特殊字符（虽然已经转义过，但为了安全再处理）
      const safeUrl = escapeHtml(sanitizedUrl)
      return `<img src="${safeUrl}" alt="emoji" class="emoji-inline" />`
    }
  )
}

// ---------- 生命周期 ----------
onMounted(async () => {
  await fetchNotifications()
  await fetchUnreadCount()
})

// 监听Tab切换
watch(activeTab, () => {
  currentPage.value = 1
  fetchNotifications()
})
</script>

<style scoped>
/* 左侧侧边栏导航 */
.nav-sidebar .nav-pills .nav-link {
  margin-bottom: 0.3rem;
  border-radius: 0.4rem;
}

/* ----- 消息列表项 ----- */
.wx-message-item {
  transition: all 0.2s ease;
  cursor: pointer;
}

.wx-message-item:hover {
  background-color: var(--bs-tertiary-bg);
  border-color: var(--bs-primary) !important;
}

.wx-message-item.border-start-4 {
  border-left-width: 4px !important;
}

/* ----- 头像/图标容器（适配深色模式） ----- */
.bg-primary-subtle {
  background-color: rgba(13, 110, 253, 0.12) !important;
}
.bg-success-subtle {
  background-color: rgba(25, 135, 84, 0.12) !important;
}
.bg-info-subtle {
  background-color: rgba(13, 202, 240, 0.12) !important;
}
.bg-danger-subtle {
  background-color: rgba(220, 53, 69, 0.12) !important;
}
.bg-secondary-subtle {
  background-color: rgba(108, 117, 125, 0.12) !important;
}

[data-bs-theme="dark"] .bg-primary-subtle {
  background-color: rgba(13, 110, 253, 0.25) !important;
}
[data-bs-theme="dark"] .bg-success-subtle {
  background-color: rgba(25, 135, 84, 0.25) !important;
}
[data-bs-theme="dark"] .bg-info-subtle {
  background-color: rgba(13, 202, 240, 0.25) !important;
}
[data-bs-theme="dark"] .bg-danger-subtle {
  background-color: rgba(220, 53, 69, 0.25) !important;
}
[data-bs-theme="dark"] .bg-secondary-subtle {
  background-color: rgba(108, 117, 125, 0.25) !important;
}

/* ----- ★ 表情图片样式 ----- */
.emoji-inline {
  height: 1.2em;
  width: auto;
  vertical-align: middle;
  display: inline-block;
  object-fit: contain;
}
</style>
