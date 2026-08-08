<template>
  <div class="card mt-3">
    <div class="card-body">
    <!-- 页面头部 -->
    <div class="d-flex flex-wrap align-items-center justify-content-between gap-3 mb-3">
      <div>
        <h1 class="h4 mb-0 d-flex align-items-center gap-2">
          <i class="bi bi-bell-fill text-primary"></i>消息通知
          <span class="badge bg-primary rounded-pill ms-2">{{ unreadCount }}</span>
        </h1>
        <p class="text-muted small mb-0 mt-1">实时接收系统通知、互动消息和平台公告</p>
      </div>
      <div class="d-flex gap-2">
        <button class="btn btn-outline-secondary btn-sm" @click="markAllRead">
          <i class="bi bi-check2-all me-1"></i>全部已读
        </button>
        <button class="btn btn-outline-danger btn-sm" @click="clearAll">
          <i class="bi bi-trash3 me-1"></i>清空
        </button>
      </div>
    </div>

    <!-- 分类筛选 Tabs -->
    <ul class="nav nav-tabs border-bottom mb-3" role="tablist">
      <li class="nav-item" role="presentation">
        <button
          class="nav-link active"
          :class="{ active: activeTab === 'all' }"
          @click="activeTab = 'all'"
        >
          全部 <span class="badge bg-secondary rounded-pill ms-1">{{ messages.length }}</span>
        </button>
      </li>
      <li class="nav-item" role="presentation">
        <button
          class="nav-link"
          :class="{ active: activeTab === 'unread' }"
          @click="activeTab = 'unread'"
        >
          未读 <span class="badge bg-primary rounded-pill ms-1">{{ unreadCount }}</span>
        </button>
      </li>
      <li class="nav-item" role="presentation">
        <button
          class="nav-link"
          :class="{ active: activeTab === 'system' }"
          @click="activeTab = 'system'"
        >
          <i class="bi bi-gear me-1"></i>系统
        </button>
      </li>
      <li class="nav-item" role="presentation">
        <button
          class="nav-link"
          :class="{ active: activeTab === 'interaction' }"
          @click="activeTab = 'interaction'"
        >
          <i class="bi bi-chat me-1"></i>互动
        </button>
      </li>
      <li class="nav-item" role="presentation">
        <button
          class="nav-link"
          :class="{ active: activeTab === 'announcement' }"
          @click="activeTab = 'announcement'"
        >
          <i class="bi bi-megaphone me-1"></i>公告
        </button>
      </li>
    </ul>

    <!-- 消息列表 -->
    <div v-if="filteredMessages.length > 0" class="wx-message-list">
      <div
        v-for="msg in filteredMessages"
        :key="msg.id"
        class="wx-message-item border rounded mb-2 p-3"
        :class="{
          'border-start border-start-4 border-primary': msg.unread,
          'bg-light-subtle': msg.unread,
          'opacity-75': !msg.unread
        }"
        @click="markRead(msg.id)"
      >
        <div class="d-flex gap-3">
          <!-- 头像 / 图标 -->
          <div class="flex-shrink-0">
            <div
              class="wx-avatar d-flex align-items-center justify-content-center rounded-circle"
              :class="getAvatarClass(msg.type)"
              style="width: 44px; height: 44px; font-size: 1.2rem;"
            >
              <i :class="getIcon(msg.type)"></i>
            </div>
          </div>

          <!-- 消息内容 -->
          <div class="flex-grow-1 min-w-0">
            <div class="d-flex flex-wrap align-items-start justify-content-between gap-2">
              <div>
                <span class="fw-semibold">{{ msg.title }}</span>
                <span v-if="msg.unread" class="badge bg-primary rounded-pill ms-2 small">未读</span>
                <span class="badge bg-light text-secondary border ms-1 small">{{ msg.typeLabel }}</span>
              </div>
              <small class="text-muted flex-shrink-0">{{ msg.time }}</small>
            </div>

            <p class="mb-1 text-body-secondary small">{{ msg.content }}</p>

            <!-- 操作按钮 -->
            <div class="d-flex gap-3 mt-2">
              <button
                v-if="msg.unread"
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
                v-if="msg.actionLink"
                class="btn btn-link btn-sm p-0 text-decoration-none"
                @click.stop="handleAction(msg.actionLink)"
              >
                <i class="bi bi-box-arrow-up-right me-1"></i>查看详情
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else class="wx-card">
      <div class="wx-empty-state py-5">
        <i class="bi bi-inbox display-4 d-block mb-3 opacity-25"></i>
        <p class="h5 mb-1">暂无消息</p>
        <p class="text-muted small">当前分类下没有消息，稍后再来看看吧</p>
        <button class="btn btn-outline-primary btn-sm mt-2" @click="activeTab = 'all'">
          <i class="bi bi-arrow-left me-1"></i>返回全部
        </button>
      </div>
    </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

// ---------- 状态 ----------
const activeTab = ref('all')
const messages = ref([
  {
    id: 1,
    type: 'system',
    typeLabel: '系统',
    title: '系统维护通知',
    content: '服务器将于 2026-08-10 02:00-04:00 进行例行维护，期间服务可能短暂不可用，请提前做好准备。',
    time: '2026-08-08 14:30',
    unread: true,
    actionLink: '/notice/system-maintenance'
  },
  {
    id: 2,
    type: 'interaction',
    typeLabel: '互动',
    title: '用户 @张三 评论了你的文章',
    content: '「这篇文章写得太棒了！尤其是关于缓存策略的部分，学到了很多。」',
    time: '2026-08-08 10:15',
    unread: true,
    actionLink: '/article/123#comment-456'
  },
  {
    id: 3,
    type: 'announcement',
    typeLabel: '公告',
    title: '🎉 新版编辑器上线预告',
    content: '全新 Markdown 编辑器即将于下周发布，支持实时预览、代码高亮和 AI 辅助写作，敬请期待！',
    time: '2026-08-07 18:00',
    unread: false,
    actionLink: '/announcement/new-editor'
  },
  {
    id: 4,
    type: 'system',
    typeLabel: '系统',
    title: '账号安全提醒',
    content: '检测到您在新设备 (Chrome 浏览器, 广东深圳) 上登录。如非本人操作，请立即修改密码。',
    time: '2026-08-07 08:22',
    unread: false,
    actionLink: '/security/account'
  },
  {
    id: 5,
    type: 'interaction',
    typeLabel: '互动',
    title: '用户 @李四 赞了你的动态',
    content: '「分享的代码片段太实用了，已收藏！」',
    time: '2026-08-06 22:10',
    unread: false,
    actionLink: '/moment/567'
  },
  {
    id: 6,
    type: 'announcement',
    typeLabel: '公告',
    title: '📢 社区运营新规',
    content: '为营造更良好的社区氛围，即日起实施新版内容规范，详见公告链接。',
    time: '2026-08-06 16:45',
    unread: false,
    actionLink: '/announcement/community-rules'
  },
  {
    id: 7,
    type: 'interaction',
    typeLabel: '互动',
    title: '用户 @王五 回复了你的评论',
    content: '「@张三 是的，我试过了，确实可以这样用，感谢分享！」',
    time: '2026-08-06 13:20',
    unread: false,
    actionLink: '/article/456#comment-789'
  }
])

// ---------- 计算属性 ----------
const unreadCount = computed(() => messages.value.filter(m => m.unread).length)

const filteredMessages = computed(() => {
  switch (activeTab.value) {
    case 'unread':
      return messages.value.filter(m => m.unread)
    case 'system':
      return messages.value.filter(m => m.type === 'system')
    case 'interaction':
      return messages.value.filter(m => m.type === 'interaction')
    case 'announcement':
      return messages.value.filter(m => m.type === 'announcement')
    default:
      return messages.value
  }
})

// ---------- 方法 ----------
const getIcon = (type) => {
  const map = {
    system: 'bi-gear',
    interaction: 'bi-chat-dots',
    announcement: 'bi-megaphone'
  }
  return map[type] || 'bi-bell'
}

const getAvatarClass = (type) => {
  const map = {
    system: 'bg-primary-subtle text-primary',
    interaction: 'bg-success-subtle text-success',
    announcement: 'bg-warning-subtle text-warning'
  }
  return map[type] || 'bg-secondary-subtle text-secondary'
}

const markRead = (id) => {
  const msg = messages.value.find(m => m.id === id)
  if (msg) msg.unread = false
}

const markAllRead = () => {
  messages.value.forEach(m => m.unread = false)
}

const removeMessage = (id) => {
  messages.value = messages.value.filter(m => m.id !== id)
}

const clearAll = () => {
  if (confirm('确定要清空所有消息吗？此操作不可恢复。')) {
    messages.value = []
  }
}

const handleAction = (link) => {
  // 跳转或执行其他操作
  console.log('跳转到:', link)
  // 实际使用 router.push(link)
}
</script>

<style scoped>
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
.bg-warning-subtle {
  background-color: rgba(255, 193, 7, 0.12) !important;
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
[data-bs-theme="dark"] .bg-warning-subtle {
  background-color: rgba(255, 193, 7, 0.25) !important;
}
[data-bs-theme="dark"] .bg-secondary-subtle {
  background-color: rgba(108, 117, 125, 0.25) !important;
}

/* ----- 空状态微调 ----- */
.wx-empty-state i {
  opacity: 0.25;
}

/* ----- 响应式 Tabs 滚动 ----- */
@media (max-width: 576px) {
  .nav-tabs {
    flex-wrap: nowrap;
    overflow-x: auto;
    gap: 0.25rem;
    padding-bottom: 0.25rem;
  }
  .nav-tabs .nav-link {
    white-space: nowrap;
    font-size: 0.8rem;
    padding: 0.4rem 0.6rem;
  }
}
</style>