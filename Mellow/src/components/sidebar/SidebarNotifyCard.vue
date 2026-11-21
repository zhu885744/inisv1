<template>
  <SidebarCard
    :title="title"
    :icon="icon"
    :more-text="moreText"
    :more-url="moreUrl"
    :collapsible="collapsible"
    :default-collapsed="defaultCollapsed"
    :store-key="storeKey"
  >
    <template #badge>
      <span v-if="notif.count > 0" class="notification-badge">{{ notif.count > 99 ? '99+' : notif.count }}</span>
    </template>

    <button
      v-if="notif.latest"
      type="button"
      class="notification-preview"
      :class="{ unread: !notif.latest.is_read }"
      @click="openLatest"
    >
      <span class="notification-preview-icon" :class="`type-${notif.latest.type}`">
        <i :class="notifIcon(notif.latest.type)" />
      </span>
      <span class="notification-preview-body">
        <span class="notification-preview-content">{{ notif.latest.content || notif.latest.title }}</span>
        <span class="notification-preview-meta">
          <span>{{ typeLabel(notif.latest.type) }}</span>
          <span>·</span>
          <span>{{ fromNow(notif.latest.create_time) }}</span>
        </span>
      </span>
    </button>
    <p v-else class="notification-empty">暂无消息</p>
  </SidebarCard>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useNotificationStore } from '@/stores/notification'
import { readNotification } from '@/api/tags'
import { notificationLink } from '@/utils/notification'
import { fromNow } from '@/utils/time'
import SidebarCard from './SidebarCard.vue'

/** 侧边栏「未读通知」模块：卡片外壳 + 最新一条消息预览 */
defineProps({
  title: { type: String, default: '消息通知' },
  icon: { type: String, default: 'bi-bell' },
  moreText: { type: String, default: '' },
  moreUrl: { type: String, default: '/notifications' },
  collapsible: { type: Boolean, default: false },
  defaultCollapsed: { type: Boolean, default: false },
  storeKey: { type: [String, Number], default: '' }
})

const notif = useNotificationStore()
const router = useRouter()

const typeMap = {
  comment: { icon: 'bi bi-chat-dots', label: '评论' },
  like: { icon: 'bi bi-hand-thumbs-up', label: '点赞' },
  follow: { icon: 'bi bi-people', label: '关注' },
  collect: { icon: 'bi bi-star', label: '收藏' },
  // 内容被兑换（付费可见）
  unlock: { icon: 'bi bi-coin', label: '兑换' },
  system: { icon: 'bi bi-megaphone', label: '系统' }
}

function typeLabel(t) {
  return typeMap[t]?.label || '通知'
}

function notifIcon(t) {
  return typeMap[t]?.icon || 'bi bi-bell'
}

async function openLatest() {
  const n = notif.latest
  if (!n) return
  // 未读则先标记已读，并刷新侧边栏的最新消息/角标
  if (!n.is_read) {
    try {
      await readNotification(n.id)
      n.is_read = 1
      await notif.refresh()
    } catch {}
  }
  // 跳转关联内容：与消息中心共用同一套规则，没有目标时回落到消息中心
  const link = await notificationLink(n)
  router.push(link || '/notifications')
}
</script>

<style scoped>
.notification-badge {
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: 9px;
  background: var(--danger);
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  box-shadow: 0 0 0 2px var(--bg-card);
  animation: notifPulse 2s ease-in-out infinite;
}
@keyframes notifPulse {
  0%,
  100% {
    box-shadow: 0 0 0 2px var(--bg-card), 0 0 0 0 rgba(217, 84, 77, 0.35);
  }
  50% {
    box-shadow: 0 0 0 2px var(--bg-card), 0 0 0 5px rgba(217, 84, 77, 0);
  }
}
.notification-preview {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 100%;
  padding: 10px;
  border: 1px solid var(--border-soft);
  border-radius: 8px;
  background: var(--bg-muted);
  text-align: left;
  cursor: pointer;
  font-family: inherit;
  transition: border-color 0.15s, background 0.15s;
}
.notification-preview:hover {
  border-color: var(--primary);
  background: var(--bg-card);
}
.notification-preview.unread {
  background: var(--accent-wash);
}
.notification-preview-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  background: var(--bg-muted);
}
.notification-preview-icon.type-comment {
  background: var(--accent-soft);
  color: var(--primary-deep);
}
.notification-preview-icon.type-like {
  background: rgba(217, 84, 77, 0.12);
  color: var(--danger);
}
.notification-preview-icon.type-follow {
  background: rgba(108, 154, 77, 0.12);
  color: var(--success);
}
.notification-preview-icon.type-collect {
  background: var(--gold-wash);
  color: #c7902f;
}
.notification-preview-icon.type-system {
  background: rgba(74, 144, 226, 0.12);
  color: #4a90e2;
}
.notification-preview-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.notification-preview-content {
  font-size: 12px;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.notification-preview.unread .notification-preview-content {
  font-weight: 600;
}
.notification-preview-meta {
  display: flex;
  gap: 4px;
  font-size: 11px;
  color: var(--text-muted);
}
.notification-empty {
  font-size: 12px;
  color: var(--text-muted);
  padding: 8px 4px;
}
</style>
