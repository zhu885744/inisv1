<template>
  <div class="room-messages-admin">
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">群聊消息</h2>
          <p class="block-desc">
            公共群聊的最近消息（全员同一条时间线）；发现违规可直接撤回，撤回后所有人都会看到「已撤回」
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm" :disabled="loading" @click="reload">
            <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': loading }" /> 刷新
          </button>
        </div>
      </header>

      <div class="list-filter">
        <div class="search-box">
          <i class="bi bi-search" aria-hidden="true" />
          <FormInput
            v-model="keyword"
            class="search-input"
            type="search"
            placeholder="搜索内容或昵称…"
            aria-label="搜索消息内容或发送者昵称" />
        </div>
        <span class="filter-tip">
          已加载 {{ list.length }} 条<template v-if="hasMore">，可继续加载更早的消息</template>
        </span>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!filteredList.length" class="empty-row">
        <EmptyState :icon="keyword ? 'bi bi-search' : 'bi bi-chat-dots'" :text="keyword ? '没有匹配的消息' : '群聊还没有消息'" />
      </div>

      <ul v-else class="msg-list">
        <li v-for="item in filteredList" :key="item.id" class="msg-row" :class="{ 'is-revoked': Number(item.is_revoked) === 1 }">
          <div class="msg-main">
            <div class="msg-head">
              <img class="msg-avatar" :src="item.user?.avatar || defaultAvatar" alt="" loading="lazy" />
              <span class="msg-name">{{ item.user?.nickname || `用户 #${item.uid}` }}</span>
              <span class="meta-text">#{{ item.uid }}</span>
              <span v-if="item.quote" class="quote-chip" :title="item.quote.content">
                引用 {{ item.quote.uid === item.uid ? '自己' : `#${item.quote.uid}` }}：{{ quotePreview(item.quote) }}
              </span>
              <span class="msg-time">{{ formatTime(item.create_time) }}</span>
            </div>

            <div class="msg-content">
              <template v-if="Number(item.is_revoked) === 1">
                <span class="revoked-text">该消息已撤回</span>
              </template>
              <template v-else-if="item.msg_type === 'image'">
                <a class="msg-image-link" :href="fullUrl(item.content)" target="_blank" rel="noopener noreferrer">
                  <img class="msg-image" :src="fullUrl(item.content)" alt="图片消息" loading="lazy" />
                </a>
              </template>
              <template v-else>
                <span class="msg-text">{{ item.content }}</span>
              </template>
            </div>
          </div>

          <div class="msg-actions">
            <button
              v-if="Number(item.is_revoked) !== 1"
              class="btn btn-ghost btn-sm danger"
              title="撤回消息"
              aria-label="撤回消息"
              :disabled="busy"
              @click="askRevoke(item)"
            >
              <i class="bi bi-arrow-counterclockwise" /> 撤回
            </button>
          </div>
        </li>
      </ul>

      <div v-if="!loading && hasMore && !keyword" class="load-more">
        <button class="btn btn-sm" :disabled="loadingMore" @click="loadMore">
          <i class="bi bi-clock-history" /> {{ loadingMore ? '加载中…' : '加载更早的消息' }}
        </button>
      </div>
    </section>

    <ConfirmDialog
      v-model:visible="confirm.visible"
      :title="confirm.title"
      :message="confirm.message"
      :confirm-text="confirm.confirmText"
      :danger="confirm.danger"
      :loading="confirm.loading"
      @confirm="runConfirm"
    />
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
/**
 * 群聊消息（/admin/chat/room）
 *
 * 后端：
 * - GET  room/messages：最近消息，before_id 往前翻页（返回正序，任何登录用户可读）；
 * - POST chat-admin/revoke（scene=chat）：撤回走后台权限点，不要求超级管理员，
 *   与「聊天审核」里的撤回是同一个口径。
 * - room/delete 只是「对自己隐藏」，对管理没有意义，因此不提供。
 */
import { ref, computed, reactive, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import { listRoomMessages } from '@/api/room'
import { revokeAdminChatMessage } from '@/api/chat-admin'
import { getFullUrl } from '@/utils/emoji'
import { formatTime } from '@/utils/time'
import { toast } from '@/utils/toast'

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const PAGE_SIZE = 50

// 新 → 旧（后端返回正序，展示时反过来，最新消息在最上面）
const list = ref([])
const hasMore = ref(false)
const loading = ref(false)
const loadingMore = ref(false)
const busy = ref(false)
const keyword = ref('')

const confirm = reactive({
  visible: false,
  title: '撤回消息',
  message: '',
  confirmText: '撤回',
  danger: true,
  loading: false,
  action: null
})

const filteredList = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return list.value
  return list.value.filter((item) => {
    const name = String(item.user?.nickname || '').toLowerCase()
    const text = item.msg_type === 'image' ? '[图片]' : String(item.content || '').toLowerCase()
    return name.includes(kw) || text.includes(kw)
  })
})

function fullUrl(url) {
  return getFullUrl(url || '')
}

function quotePreview(quote) {
  if (!quote) return ''
  if (quote.msg_type === 'image') return '[图片]'
  const text = String(quote.content || '')
  return text.length > 20 ? `${text.slice(0, 20)}…` : text
}

async function load() {
  loading.value = true
  try {
    // 后台这个页只审公共大厅（id=1）；自建群的消息在「群聊管理」里逐个查看
    const res = await listRoomMessages(1, { limit: PAGE_SIZE })
    const rows = res.data?.data || []
    // 接口是正序（旧 → 新），后台按「最新在上」展示
    list.value = rows.slice().reverse()
    hasMore.value = !!res.data?.has_more
  } catch {
    list.value = []
    hasMore.value = false
  } finally {
    loading.value = false
  }
}

function reload() {
  keyword.value = ''
  load()
}

async function loadMore() {
  const last = list.value[list.value.length - 1]
  if (!last || loadingMore.value) return

  loadingMore.value = true
  try {
    const res = await listRoomMessages(1, { limit: PAGE_SIZE, before_id: last.id })
    const rows = (res.data?.data || []).slice().reverse()
    const seen = new Set(list.value.map((item) => Number(item.id)))
    list.value.push(...rows.filter((item) => !seen.has(Number(item.id))))
    hasMore.value = !!res.data?.has_more
  } catch {
    /* 拦截器已提示 */
  } finally {
    loadingMore.value = false
  }
}

function openConfirm({ title, message, confirmText = '确定', danger = false, action }) {
  confirm.title = title
  confirm.message = message
  confirm.confirmText = confirmText
  confirm.danger = danger
  confirm.action = action
  confirm.visible = true
}

async function runConfirm() {
  if (typeof confirm.action !== 'function') {
    confirm.visible = false
    return
  }
  confirm.loading = true
  try {
    const ok = await confirm.action()
    if (ok !== false) confirm.visible = false
  } catch {
    /* 拦截器已提示，保持弹窗 */
  } finally {
    confirm.loading = false
  }
}

function askRevoke(item) {
  const preview = item.msg_type === 'image' ? '[图片]' : String(item.content || '').slice(0, 40)
  openConfirm({
    title: '撤回消息',
    message: `确定撤回「${item.user?.nickname || `用户 #${item.uid}`}」的这条消息吗？撤回后所有人都会看到「已撤回」。\n\n${preview}`,
    confirmText: '撤回',
    danger: true,
    action: async () => {
      busy.value = true
      try {
        await revokeAdminChatMessage(item.id, 'chat')
        item.is_revoked = 1
        item.content = ''
        toast.success('已撤回')
      } finally {
        busy.value = false
      }
    }
  })
}

onMounted(load)
</script>

<style scoped>
.room-messages-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.block-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}
.block-desc {
  margin: 4px 0 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-muted);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

/* ---------- 筛选 ---------- */
.list-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.search-box {
  position: relative;
  width: 220px;
}
.search-box .bi {
  position: absolute;
  left: 10px;
  top: 50%;
  transform: translateY(-50%);
  font-size: 13px;
  color: var(--text-light);
}
.search-input {
  width: 100%;
  height: var(--control-h-sm);
  padding: 0 12px 0 30px;
  font-size: 13px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-card);
  color: var(--text);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.search-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.filter-tip {
  margin-left: auto;
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 列表 ---------- */
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}

.msg-list {
  margin: 0;
  padding: 0;
  list-style: none;
}
.msg-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 8px;
  border-bottom: 1px dashed var(--border-soft);
}
.msg-row:last-child {
  border-bottom: none;
}
.msg-row:hover {
  background: var(--bg-muted);
}
.msg-row.is-revoked .msg-text {
  color: var(--text-light);
  text-decoration: line-through;
}
.msg-main {
  flex: 1;
  min-width: 0;
}
.msg-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 6px;
}
.msg-avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.msg-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
}
.msg-time {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-light);
}
.meta-text {
  font-size: 12px;
  color: var(--text-muted);
}
.quote-chip {
  max-width: 260px;
  padding: 1px 6px;
  font-size: 11px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-left: 2px solid var(--primary);
  border-radius: 3px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.msg-content {
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  word-break: break-word;
}
.msg-text {
  white-space: pre-wrap;
}
.revoked-text {
  font-size: 12px;
  color: var(--text-light);
}
.msg-image-link {
  display: inline-block;
}
.msg-image {
  max-width: 160px;
  max-height: 160px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-soft);
}
.msg-actions {
  flex-shrink: 0;
}
.msg-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}

.load-more {
  display: flex;
  justify-content: center;
  padding-top: 14px;
}

.is-spin {
  animation: rm-spin 0.9s linear infinite;
}
@keyframes rm-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 768px) {
  .search-box {
    width: 100%;
  }
  .filter-tip {
    margin-left: 0;
  }
}
</style>
