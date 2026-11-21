<template>
  <div class="notice">
    <!-- 页头：标题 + 未读概览 + 批量操作 -->
    <header class="notice-head">
      <div class="notice-head-main">
        <h1 class="notice-title">
          消息通知
          <em v-if="unreadCount > 0" class="notice-title-dot" aria-hidden="true" />
        </h1>
        <p class="notice-sub">
          <template v-if="unreadCount > 0">
            <strong>{{ unreadCount }}</strong> 条未读 ·
          </template>
          共 {{ total }} 条
        </p>
      </div>

      <div class="notice-head-actions">
        <button
          type="button"
          class="notice-act"
          :disabled="!canReadAll || readAllLoading"
          :title="readAllLoading ? '处理中' : '全部已读'"
          aria-label="全部已读"
          @click="readAll"
        >
          <i :class="readAllLoading ? 'bi bi-hourglass-split' : 'bi bi-check2-all'" aria-hidden="true" />
          <span>全部已读</span>
        </button>
        <button
          type="button"
          class="notice-act is-ghost"
          :disabled="loading || clearReadLoading"
          title="清空已读"
          aria-label="清空已读"
          @click="showClearConfirm = true"
        >
          <i class="bi bi-trash3" aria-hidden="true" />
        </button>
      </div>
    </header>

    <!-- 聊天入口：私聊 / 群聊，带未读角标 -->
    <nav class="notice-hub" aria-label="聊天入口">
      <RouterLink to="/chat" class="notice-hub-card">
        <span class="notice-hub-icon is-dm"><i class="bi bi-chat-dots-fill" aria-hidden="true" /></span>
        <span class="notice-hub-text">
          <strong>私聊</strong>
          <small>{{ chatUnread > 0 ? `${chatUnread} 条未读` : '和好友单独聊' }}</small>
        </span>
        <em v-if="chatUnread > 0" class="notice-hub-badge">{{ chatUnread > 99 ? '99+' : chatUnread }}</em>
      </RouterLink>

      <RouterLink to="/chat/room/1" class="notice-hub-card">
        <span class="notice-hub-icon is-room"><i class="bi bi-people-fill" aria-hidden="true" /></span>
        <span class="notice-hub-text">
          <strong>群聊</strong>
          <small>{{ roomMentions > 0 ? `${roomMentions} 条 @ 我` : '进公共群聊看看' }}</small>
        </span>
        <em v-if="roomMentions > 0" class="notice-hub-badge">{{ roomMentions > 99 ? '99+' : roomMentions }}</em>
      </RouterLink>
    </nav>

    <!-- 筛选：读取状态分段 + 类型下拉 -->
    <div class="notice-filter">
      <div class="notice-seg" role="group" aria-label="读取状态">
        <button
          v-for="opt in readOptions"
          :key="opt.value"
          type="button"
          class="notice-seg-item"
          :class="{ 'is-active': filter.is_read === opt.value }"
          :aria-pressed="filter.is_read === opt.value"
          @click="setRead(opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>

      <FormSelect variant="pill"
        v-model="filter.type"
        :options="typeOptions"
        icon="bi bi-funnel"
        placeholder="全部类型"
        @change="reload" />
    </div>

    <!-- 骨架 -->
    <div v-if="loading" class="notice-group">
      <div class="notice-list">
        <div v-for="i in 5" :key="i" class="notice-sk">
          <span class="notice-sk-icon" />
          <div class="notice-sk-lines">
            <span class="notice-sk-line w55" />
            <span class="notice-sk-line w85" />
          </div>
        </div>
      </div>
    </div>

    <EmptyState v-else-if="!items.length" icon="bi bi-bell-slash" text="暂无通知" />

    <!-- 按日期分组的时间线 -->
    <template v-else>
      <section v-for="group in groups" :key="group.key" class="notice-group">
        <h2 class="notice-group-title">
          {{ group.label }}
          <em v-if="group.unread > 0">{{ group.unread }}</em>
        </h2>

        <ul class="notice-list">
          <li
            v-for="n in group.items"
            :key="n.id"
            class="notice-item"
            :class="{ 'is-unread': !n.is_read }"
            role="button"
            tabindex="0"
            @click="onClick(n)"
            @keydown.enter.prevent="onClick(n)"
            @keydown.space.prevent="onClick(n)"
          >
            <span class="notice-item-icon" :class="`tone-${toneOf(n.type)}`">
              <i :class="iconOf(n.type)" aria-hidden="true" />
            </span>

            <div class="notice-item-main">
              <p class="notice-item-title">{{ n.title }}</p>
              <p class="notice-item-desc">{{ n.content || typeLabel(n.type) }}</p>
            </div>

            <div class="notice-item-side">
              <time class="notice-item-time">{{ fromNow(n.create_time) }}</time>
              <span v-if="!n.is_read" class="notice-item-dot" aria-label="未读" />
            </div>
          </li>
        </ul>
      </section>
    </template>

    <Pagination
      :current="page"
      :total="total"
      :page-size="pageSize"
      @update:current="(p) => { page = p; load() }"
    />

    <!-- 清空已读确认弹窗 -->
    <ConfirmDialog
      v-model:visible="showClearConfirm"
      title="清空已读消息"
      message="确定清空所有已读消息吗？清空后将移入回收站，可稍后恢复。"
      confirm-text="清空"
      loading-text="清空中..."
      danger
      :loading="clearReadLoading"
      @confirm="clearRead"
    />
  </div>
</template>

<script setup>
import FormSelect from '@/components/form/FormSelect.vue'
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { listNotifications, readNotification } from '@/api/tags'
// remove-all 单次有数量上限，用 @/api/notification 的封装循环清理到清空
import { clearAllNotifications } from '@/api/notification'
import { useNotificationStore } from '@/stores/notification'
import { useChatStore } from '@/stores/chat'
import { useRouter } from 'vue-router'
import { fromNow } from '@/utils/time'
import { notificationLink } from '@/utils/notification'
import { toast } from '@/utils/toast'

const router = useRouter()
const notif = useNotificationStore()
const chat = useChatStore()

const items = ref([])
const total = ref(0)
const loading = ref(false)
const readAllLoading = ref(false)
const clearReadLoading = ref(false)
const showClearConfirm = ref(false)
const page = ref(1)
const pageSize = 15

const filter = ref({
  type: '',
  is_read: ''
})

const typeOptions = [
  { value: '', label: '全部类型' },
  { value: 'comment', label: '评论' },
  { value: 'mention', label: '@我的' },
  { value: 'like', label: '点赞' },
  { value: 'follow', label: '关注' },
  { value: 'collect', label: '收藏' },
  { value: 'article', label: '文章' },
  { value: 'page', label: '页面' },
  { value: 'links', label: '友链' },
  { value: 'moments', label: '动态' },
  { value: 'decoration', label: '装扮' },
  // 勋章：限定勋章即将到期提醒
  { value: 'badge', label: '勋章' },
  // 内容被兑换：别人消耗积分兑换了你设置的付费内容
  { value: 'unlock', label: '兑换' },
  // 收到积分打赏
  { value: 'tip', label: '打赏' },
  { value: 'system', label: '系统' }
]

const readOptions = [
  { value: '', label: '全部' },
  { value: '0', label: '未读' },
  { value: '1', label: '已读' }
]

const unreadCount = computed(() => Number(notif.count) || 0)

// 当前列表里是否还有未读（全局未读数依赖轮询，可能滞后或接口异常）
const unreadInView = computed(() => items.value.some((n) => !n.is_read))

// 满足任一条件即可一键已读：全局有未读，或当前列表存在未读
const canReadAll = computed(() => unreadCount.value > 0 || unreadInView.value)

// 聊天入口角标：与 /chat 侧栏同一份数据（私信总数 / 群聊里被 @ 的次数）
const chatUnread = computed(() => Number(chat.unread) || 0)
const roomMentions = computed(() => Number(chat.roomMentions) || 0)

const typeMap = {
  comment: '评论',
  // @ 提及：文章 / 动态 / 独立页 / 评论里被 @ 到（见 app/model/mention.go）
  mention: '@我的',
  like: '点赞',
  follow: '关注',
  collect: '收藏',
  article: '文章',
  page: '页面',
  links: '友链',
  moments: '动态',
  // 装扮：获得 / 续期 / 被回收 / 即将过期 / 已过期
  decoration: '装扮',
  // 勋章：限定勋章即将到期（见 model.BadgeExpireWarn）
  badge: '勋章',
  // 兑换：别人消耗积分兑换了你的付费内容（见 model.SendUnlockNotify）
  unlock: '兑换',
  // 打赏：别人给你送了积分（见 model.SendTipNotify）
  tip: '打赏',
  system: '系统'
}

/**
 * 按「今天 / 昨天 / 近 7 天 / 具体日期」分组
 *
 * 接口按时间倒序返回，这里顺着顺序收集即可，不用再排一次。
 * 组内未读数顺手统计，用来在分组标题上打个小角标。
 */
const groups = computed(() => {
  const out = []
  const seen = new Map()

  for (const item of items.value) {
    const label = dayLabel(item.create_time)
    if (!seen.has(label)) {
      const group = { key: label, label, items: [], unread: 0 }
      seen.set(label, group)
      out.push(group)
    }
    const group = seen.get(label)
    group.items.push(item)
    if (!item.is_read) group.unread += 1
  }

  return out
})

/** 时间戳兼容秒 / 毫秒，取本地日期分组名 */
function dayLabel(ts) {
  let value = Number(ts) || 0
  if (value > 1e12) value /= 1000
  if (!value) return '更早'

  const date = new Date(value * 1000)
  const today = new Date()
  today.setHours(0, 0, 0, 0)

  const target = new Date(date)
  target.setHours(0, 0, 0, 0)

  const diff = Math.round((today.getTime() - target.getTime()) / 86400000)
  if (diff <= 0) return '今天'
  if (diff === 1) return '昨天'
  if (diff < 7) return '近 7 天'
  if (date.getFullYear() === today.getFullYear()) return `${date.getMonth() + 1} 月 ${date.getDate()} 日`
  return `${date.getFullYear()} 年 ${date.getMonth() + 1} 月 ${date.getDate()} 日`
}

async function load() {
  loading.value = true
  try {
    const params = { page: page.value }
    if (filter.value.type) params.type = filter.value.type
    if (filter.value.is_read) params.is_read = filter.value.is_read
    const res = await listNotifications(params)
    // 后端返回结构: { code, data: { data: [...], count, page } }
    const payload = res.data?.data
    items.value = Array.isArray(payload) ? payload : (payload?.data || [])
    total.value = payload?.count || res.data?.count || 0
    notif.refresh()
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

function setRead(v) {
  if (filter.value.is_read === v) return
  filter.value.is_read = v
  reload()
}

function typeLabel(t) {
  return typeMap[t] || '通知'
}

function iconOf(t) {
  return ({
    comment: 'bi bi-chat-dots',
    mention: 'bi bi-at',
    like: 'bi bi-hand-thumbs-up',
    follow: 'bi bi-people',
    collect: 'bi bi-star',
    article: 'bi bi-file-text',
    page: 'bi bi-file-earmark',
    links: 'bi bi-link-45deg',
    moments: 'bi bi-chat-square-text',
    decoration: 'bi bi-gem',
    // 勋章：到期提醒
    badge: 'bi bi-award',
    // 兑换：积分消费类，用金币图标
    unlock: 'bi bi-coin',
    // 打赏：收到的积分，用礼物图标区分于「兑换」
    tip: 'bi bi-gift',
    system: 'bi bi-megaphone'
  })[t] || 'bi bi-bell'
}

/** 图标底色：按通知类型分组，视觉上区分「互动 / 内容 / 虚拟物品 / 系统」 */
function toneOf(t) {
  return ({
    comment: 'reply',
    mention: 'reply',
    moments: 'reply',
    like: 'like',
    follow: 'follow',
    collect: 'mark',
    article: 'doc',
    page: 'doc',
    links: 'doc',
    decoration: 'gift',
    badge: 'gift',
    unlock: 'gift',
    tip: 'gift',
    system: 'system'
  })[t] || 'system'
}

async function readAll() {
  if (readAllLoading.value || !canReadAll.value) return
  readAllLoading.value = true
  try {
    await notif.readAll()
    toast.success('已全部标记为已读')
    await load()
  } catch {
    toast.error('操作失败，请重试')
  } finally {
    readAllLoading.value = false
  }
}

// 清空已读消息：只软删除已读通知（移入回收站，可恢复），未读消息不受影响
// 由主题内置的 <ConfirmDialog> 二次确认后触发，处理中弹窗保持打开
//
// 注意：后端 remove-all 单次最多清理 200 条（NOTIFICATION_BATCH_LIMIT），
// 返回 { cleared, remaining }，因此这里循环调用直到清空（最多 20 轮，防御性上限）。
async function clearRead() {
  if (clearReadLoading.value) return
  clearReadLoading.value = true
  try {
    const { cleared, remaining } = await clearAllNotifications({ is_read: 1 })

    if (cleared === 0) {
      toast.info('暂无已读消息')
    } else if (remaining > 0) {
      toast.success(`已清空 ${cleared} 条已读消息，剩余 ${remaining} 条请再次点击`)
    } else {
      toast.success('已清空全部已读消息')
    }

    // 清空后第一页可能为空，回到首页重新加载
    if (page.value > 1) page.value = 1
    await load()
  } catch {
    toast.error('清空失败，请重试')
  } finally {
    clearReadLoading.value = false
    showClearConfirm.value = false
  }
}

async function onClick(n) {
  if (!n.is_read) {
    try {
      await readNotification(n.id)
      n.is_read = 1
      notif.count = Math.max(0, notif.count - 1)
    } catch {}
  }
  // 跳转：bind_type → 路由的规则统一放在 utils/notification.js
  // （article/moments/page 进详情、comment 先查所属内容、user 进作者页、其余为固定页面）
  const link = await notificationLink(n)
  if (link) router.push(link)
}

// 通知不走长连接（只靠查询接口，见 app/model/notification.go），
// 所以页面停留期间必须自己再拉一次，否则新通知要刷新页面才出现。
// 只在第一页、且页面可见时轮询：翻页看历史时不该把用户拽回第一页。
const POLL_INTERVAL = 30000
let pollTimer = null

function startPolling() {
  if (pollTimer) return
  pollTimer = setInterval(() => {
    if (document.visibilityState !== 'visible') return
    if (page.value !== 1 || loading.value) return
    load()
  }, POLL_INTERVAL)
}

function stopPolling() {
  clearInterval(pollTimer)
  pollTimer = null
}

/** 回到前台立刻补一次（后台期间的定时器通常被浏览器限流甚至暂停） */
function onVisible() {
  if (document.visibilityState !== 'visible') return
  if (page.value === 1) load()
  else notif.refresh()
  // 在其它标签页读完私信后回来，角标也要跟着对齐
  chat.refresh()
}

onMounted(() => {
  // 私信角标归 chat store 管，进本页先校准一次：
  // 否则展示的是「上次在 /chat」留下的旧数字（读完了这里还挂着几十条未读）
  chat.refresh()
  load()
  startPolling()
  document.addEventListener('visibilitychange', onVisible)
})

onBeforeUnmount(() => {
  stopPolling()
  document.removeEventListener('visibilitychange', onVisible)
})
</script>

<style scoped>
/* 挂在 MainLayout 下：只负责内容区纵向间距，宽度交给布局的主列（撑满即可） */
/* 不要在这里加 max-width + margin: 0 auto —— 主列比它宽时，整块会被挤在中间，两侧留出大片空白 */
.notice {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

/* ---------- 页头 ---------- */
.notice-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.notice-head-main {
  min-width: 0;
}
.notice-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 19px;
  font-weight: 600;
  letter-spacing: 0.2px;
  color: var(--text);
}
.notice-title-dot {
  width: 7px;
  height: 7px;
  background: var(--danger);
  border-radius: 50%;
}
.notice-sub {
  margin: 3px 0 0;
  font-size: 12.5px;
  color: var(--text-muted);
}
.notice-sub strong {
  color: var(--primary-deep);
  font-weight: 600;
}
.notice-head-actions {
  display: flex;
  align-items: center;
  gap: 6px;
}
.notice-act {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 30px;
  padding: 0 12px;
  font-family: inherit;
  font-size: 12.5px;
  color: var(--primary-deep);
  background: var(--accent-soft);
  border: 0;
  border-radius: 8px;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), opacity var(--dur-fast) var(--ease);
}
.notice-act:hover:not(:disabled) {
  background: var(--accent-wash);
}
.notice-act.is-ghost {
  width: 30px;
  padding: 0;
  justify-content: center;
  font-size: 14px;
  color: var(--text-soft);
  background: var(--bg-muted);
}
.notice-act.is-ghost:hover:not(:disabled) {
  color: var(--danger);
  background: var(--bg-muted);
}
.notice-act:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* ---------- 聊天入口 ---------- */
.notice-hub {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.notice-hub-card {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease),
    transform var(--dur-fast) var(--ease);
}
.notice-hub-card:hover {
  border-color: var(--primary-soft);
  box-shadow: var(--shadow-md);
  transform: translateY(-1px);
}
.notice-hub-icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  font-size: 15px;
  border-radius: 10px;
}
.notice-hub-icon.is-dm {
  color: #12a594;
  background: rgba(18, 165, 148, 0.12);
}
.notice-hub-icon.is-room {
  color: #4a90e2;
  background: rgba(74, 144, 226, 0.12);
}
.notice-hub-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.notice-hub-text strong {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.notice-hub-text small {
  overflow: hidden;
  font-size: 11.5px;
  color: var(--text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notice-hub-badge {
  display: grid;
  place-items: center;
  min-width: 18px;
  height: 18px;
  margin-left: auto;
  padding: 0 5px;
  font-size: 11px;
  font-style: normal;
  font-weight: 600;
  color: #fff;
  background: var(--danger);
  border-radius: 9px;
}

/* ---------- 筛选 ---------- */
.notice-filter {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
/* 分段控件：与 chips 区分，表达「互斥的三选一」 */
.notice-seg {
  display: inline-flex;
  gap: 2px;
  padding: 3px;
  background: var(--bg-muted);
  border-radius: 9px;
}
.notice-seg-item {
  height: 26px;
  padding: 0 12px;
  font-family: inherit;
  font-size: 12.5px;
  color: var(--text-soft);
  background: transparent;
  border: 0;
  border-radius: 7px;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), background var(--dur-fast) var(--ease),
    box-shadow var(--dur-fast) var(--ease);
}
.notice-seg-item:hover {
  color: var(--text);
}
.notice-seg-item.is-active {
  color: var(--text);
  background: var(--bg-card);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
  font-weight: 600;
}

/* ---------- 分组列表 ---------- */
.notice-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.notice-group-title {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 0 2px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.3px;
}
.notice-group-title em {
  display: grid;
  place-items: center;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  font-size: 10.5px;
  font-style: normal;
  color: #fff;
  background: var(--danger);
  border-radius: 8px;
}
.notice-list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  overflow: hidden;
  list-style: none;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
}
.notice-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 13px;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}
.notice-item + .notice-item {
  border-top: 1px solid var(--border-soft);
}
.notice-item:hover {
  background: var(--bg-muted);
}
.notice-item:focus-visible {
  outline: 2px solid var(--primary-soft);
  outline-offset: -2px;
}
.notice-item-icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  font-size: 15px;
  border-radius: 10px;
}
.notice-item-main {
  flex: 1;
  min-width: 0;
}
.notice-item-title {
  margin: 0;
  overflow: hidden;
  font-size: 13.5px;
  font-weight: 500;
  color: var(--text-soft);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notice-item.is-unread .notice-item-title {
  font-weight: 600;
  color: var(--text);
}
.notice-item-desc {
  margin: 2px 0 0;
  overflow: hidden;
  font-size: 12px;
  line-height: 1.4;
  color: var(--text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notice-item-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 4px;
  flex-shrink: 0;
}
.notice-item-time {
  font-size: 11px;
  color: var(--text-light);
}
.notice-item-dot {
  width: 7px;
  height: 7px;
  background: var(--danger);
  border-radius: 50%;
}

/* 类型配色：互动（青绿 / 粉 / 蓝）、内容（紫）、虚拟物品（玫红）、系统（灰） */
.notice-item-icon.tone-reply {
  color: #12a594;
  background: rgba(18, 165, 148, 0.1);
}
.notice-item-icon.tone-like {
  color: #e8618c;
  background: rgba(232, 97, 140, 0.1);
}
.notice-item-icon.tone-follow {
  color: #4a90e2;
  background: rgba(74, 144, 226, 0.1);
}
.notice-item-icon.tone-mark {
  color: #eda13a;
  background: rgba(237, 161, 58, 0.14);
}
.notice-item-icon.tone-doc {
  color: #7c6cf0;
  background: rgba(124, 108, 240, 0.12);
}
.notice-item-icon.tone-gift {
  color: #e2557a;
  background: rgba(226, 85, 122, 0.12);
}
.notice-item-icon.tone-system {
  color: var(--text-soft);
  background: var(--bg-muted);
}

/* ---------- 骨架 ---------- */
.notice-sk {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 13px;
}
.notice-sk + .notice-sk {
  border-top: 1px solid var(--border-soft);
}
.notice-sk-icon {
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  background: var(--bg-muted);
  border-radius: 10px;
}
.notice-sk-lines {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 7px;
}
.notice-sk-line {
  height: 10px;
  background: var(--bg-muted);
  border-radius: 6px;
}
.notice-sk-line.w55 {
  width: 55%;
}
.notice-sk-line.w85 {
  width: 85%;
}

/* ---------- 窄屏 ---------- */
@media (max-width: 520px) {
  .notice {
    gap: 11px;
  }
  .notice-title {
    font-size: 17px;
  }
  .notice-act span {
    display: none;
  }
  .notice-act {
    width: 30px;
    padding: 0;
    justify-content: center;
    font-size: 14px;
  }
  .notice-hub-card {
    padding: 9px 11px;
    gap: 9px;
  }
  .notice-hub-icon {
    width: 32px;
    height: 32px;
    font-size: 14px;
    border-radius: 9px;
  }
  .notice-hub-text strong {
    font-size: 13px;
  }
  .notice-item {
    gap: 9px;
    padding: 9px 11px;
  }
  .notice-item-icon {
    width: 32px;
    height: 32px;
    font-size: 14px;
    border-radius: 9px;
  }
}
</style>
