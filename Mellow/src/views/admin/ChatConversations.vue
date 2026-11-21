<template>
  <div class="chat-admin" :class="{ 'is-chat': !!activeConv }">
    <section class="card card-pad">
      <header class="panel-head">
        <div>
          <h2 class="block-title">私信会话</h2>
          <p class="block-desc">
            按会话维度查看任意两位用户的私信内容；发现违规可直接撤回（双方都会看到「撤回了一条消息」），不做物理删除
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-sm" :disabled="loadingConv || loadingMsg" @click="refresh">
            <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': loadingConv }" /> 刷新
          </button>
        </div>
      </header>

      <div class="chat-layout">
        <!-- 左：会话列表 -->
        <aside class="conv-side">
          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="用户 UID 或消息关键词…"
              aria-label="按 UID 或消息关键词搜索会话"
              @input="doSearch" />
          </div>

          <div v-if="loadingConv" class="is-loading"><span class="spinner" /> 加载中...</div>

          <div v-else-if="!conversations.length" class="empty-row">
            <EmptyState :icon="keyword ? 'bi bi-search' : 'bi bi-chat-dots'" :text="keyword ? '没有匹配的会话' : '还没有私信会话'" />
          </div>

          <ul v-else class="conv-list">
            <li v-for="item in conversations" :key="item.id">
              <button
                type="button"
                class="conv-row"
                :class="{ 'is-active': item.id === activeConv?.id }"
                @click="openConversation(item)"
              >
                <span class="conv-avatars">
                  <img class="conv-avatar" :src="avatarOf(item.user)" alt="" loading="lazy" />
                  <img class="conv-avatar" :src="avatarOf(item.target)" alt="" loading="lazy" />
                </span>
                <span class="conv-body">
                  <span class="conv-top">
                    <span class="conv-name">{{ nameOf(item.user) }} ↔ {{ nameOf(item.target) }}</span>
                    <time class="conv-time">{{ shortTime(item.last_time) }}</time>
                  </span>
                  <span class="conv-bottom">
                    <span class="conv-uids">UID {{ item.uid }} · {{ item.target_uid }}</span>
                    <span class="conv-text">{{ item.last_text || '（暂无消息）' }}</span>
                  </span>
                </span>
              </button>
            </li>
          </ul>

          <div v-if="!loadingConv && conversations.length < convTotal" class="load-more">
            <button class="btn btn-sm" :disabled="loadingMoreConv" @click="loadMoreConversations">
              {{ loadingMoreConv ? '加载中…' : `加载更多会话（${conversations.length}/${convTotal}）` }}
            </button>
          </div>
        </aside>

        <!-- 右：消息内容 -->
        <section class="conv-main">
          <template v-if="activeConv">
            <header class="conv-head">
              <button type="button" class="btn btn-ghost btn-sm conv-back" title="返回会话列表" @click="closeConversation">
                <i class="bi bi-chevron-left" />
              </button>
              <img class="conv-avatar is-head" :src="avatarOf(activeConv.user)" alt="" loading="lazy" />
              <img class="conv-avatar is-head" :src="avatarOf(activeConv.target)" alt="" loading="lazy" />
              <span class="conv-head-name">
                {{ nameOf(activeConv.user) }} ↔ {{ nameOf(activeConv.target) }}
              </span>
              <span class="meta-text">UID {{ activeConv.uid }} · {{ activeConv.target_uid }} · 已加载 {{ messages.length }} 条</span>
            </header>

            <div class="msg-scroll">
              <div v-if="loadingMsg" class="is-loading"><span class="spinner" /> 加载中...</div>

              <div v-else-if="!messages.length" class="empty-row">
                <EmptyState icon="bi bi-chat-square-dots" text="这段会话还没有消息" />
              </div>

              <template v-else>
                <div
                  v-for="item in messages"
                  :key="item.id"
                  class="dm-row"
                  :class="{ 'is-revoked': Number(item.is_revoked) === 1 }"
                >
                  <img class="dm-avatar" :src="avatarOf(senderOf(item))" alt="" loading="lazy" />
                  <div class="dm-main">
                    <div class="dm-head">
                      <span class="dm-name">{{ nameOf(senderOf(item)) }}</span>
                      <span class="meta-text">{{ formatTime(item.create_time) }}</span>
                      <span v-if="item.quote" class="quote-chip" :title="item.quote.content">
                        引用：{{ quotePreview(item.quote) }}
                      </span>
                    </div>

                    <div class="dm-content">
                      <template v-if="Number(item.is_revoked) === 1">
                        <span class="revoked-text">该消息已撤回</span>
                      </template>
                      <template v-else-if="item.msg_type === 'image'">
                        <a :href="fullUrl(item.content)" target="_blank" rel="noopener noreferrer">
                          <img class="dm-image" :src="fullUrl(item.content)" alt="图片消息" loading="lazy" />
                        </a>
                      </template>
                      <template v-else-if="item.msg_type === 'pat'">
                        <span class="revoked-text">拍一拍{{ item.content ? ` ${item.content}` : '' }}</span>
                      </template>
                      <template v-else>
                        <span class="dm-text">{{ item.content }}</span>
                      </template>
                    </div>
                  </div>

                  <button
                    v-if="Number(item.is_revoked) !== 1"
                    type="button"
                    class="btn btn-ghost btn-sm danger dm-revoke"
                    title="撤回这条消息"
                    aria-label="撤回这条消息"
                    :disabled="busy"
                    @click="askRevoke(item)"
                  >
                    <i class="bi bi-arrow-counterclockwise" /> 撤回
                  </button>
                </div>

                <div v-if="hasMore" class="load-more">
                  <button class="btn btn-sm" :disabled="loadingOlder" @click="loadOlder">
                    <i class="bi bi-clock-history" /> {{ loadingOlder ? '加载中…' : '加载更早的消息' }}
                  </button>
                </div>
              </template>
            </div>
          </template>

          <div v-else class="placeholder">
            <div class="placeholder-mark"><i class="bi bi-envelope-paper" aria-hidden="true" /></div>
            <p>从左侧选择一个会话，查看这两位用户的私信内容</p>
          </div>
        </section>
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
 * 私信会话（/admin/chat/conversations）
 *
 * 后端（app/api/controller/chat-admin.go、app/model/chat-admin.go）：
 * - GET  chat-admin/conversations：全部会话（uid 过滤 / 摘要关键词 / 分页）；
 * - GET  chat-admin/messages：某会话的消息（before_id 往前翻，倒序取一页、正序返回）；
 * - POST chat-admin/revoke：撤回（不限 2 分钟、不限发送者），只清内容不物理删除。
 *
 * 权限：三个接口是独立权限点（模块 chat-admin），没被授权的账号调用会 403 ——
 * 后台页面权限与接口权限分开授予，站点可以只给「能看不能改」的运营账号开列表。
 */
import { ref, reactive, computed, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import {
  listAdminChatConversations,
  listAdminChatMessages,
  revokeAdminChatMessage
} from '@/api/chat-admin'
import { formatTime, fromNow } from '@/utils/time'
import { getFullUrl } from '@/utils/emoji'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

// ===== 会话列表 =====
const conversations = ref([])
const convTotal = ref(0)
const convPage = ref(1)
const loadingConv = ref(false)
const loadingMoreConv = ref(false)
const keyword = ref('')

// ===== 当前会话 =====
const activeConv = ref(null)
const messages = ref([]) // 最新在上
const hasMore = ref(false)
const loadingMsg = ref(false)
const loadingOlder = ref(false)
const busy = ref(false)

const confirm = reactive({
  visible: false,
  title: '撤回消息',
  message: '',
  confirmText: '撤回',
  danger: true,
  loading: false,
  action: null
})

// ---------- 展示辅助 ----------
function avatarOf(user) {
  return user?.avatar || defaultAvatar
}

function nameOf(user) {
  return user?.nickname || (user?.id ? `用户 #${user.id}` : '已注销用户')
}

/** 消息发送者在本次会话里的资料（左侧是 uid、右侧是 target_uid） */
function senderOf(msg) {
  const from = Number(msg?.from_uid || 0)
  const conv = activeConv.value
  if (!conv) return null
  return Number(conv.uid) === from ? conv.user : conv.target
}

function shortTime(ts) {
  const value = Number(ts) || 0
  if (value <= 0) return ''
  const diff = Date.now() / 1000 - value
  if (diff < 60) return '刚刚'
  if (diff < 86400) return fromNow(value)
  return formatTime(value).slice(5, 16)
}

function fullUrl(url) {
  return getFullUrl(url || '')
}

function quotePreview(quote) {
  if (!quote) return ''
  if (quote.msg_type === 'image') return '[图片]'
  const text = String(quote.content || '')
  return text.length > 20 ? `${text.slice(0, 20)}…` : text
}

const searchParams = computed(() => {
  const kw = keyword.value.trim()
  if (!kw) return {}
  // 纯数字按 UID 查（「这个用户跟谁聊过」），其它按最后一条摘要模糊匹配
  return /^\d+$/.test(kw) ? { uid: Number(kw) } : { keyword: kw }
})

// ---------- 会话列表 ----------
async function loadConversations(reset = true) {
  if (reset) {
    convPage.value = 1
    loadingConv.value = true
  } else {
    loadingMoreConv.value = true
  }

  try {
    const res = await listAdminChatConversations({
      page: convPage.value,
      limit: 20,
      ...searchParams.value
    })
    const rows = res.data?.data || []
    conversations.value = reset ? rows : [...conversations.value, ...rows]
    convTotal.value = Number(res.data?.count) || 0
  } catch {
    if (reset) {
      conversations.value = []
      convTotal.value = 0
    }
  } finally {
    loadingConv.value = false
    loadingMoreConv.value = false
  }
}

const doSearch = debounce(() => {
  loadConversations(true)
}, 350)

function loadMoreConversations() {
  if (loadingMoreConv.value) return
  convPage.value += 1
  loadConversations(false)
}

async function refresh() {
  await loadConversations(true)
  if (activeConv.value) await loadMessages(activeConv.value.id, 0)
}

// ---------- 消息 ----------
async function loadMessages(conversationId, beforeId = 0) {
  if (!conversationId) return

  if (beforeId) loadingOlder.value = true
  else {
    loadingMsg.value = true
    messages.value = []
    hasMore.value = false
  }

  try {
    const res = await listAdminChatMessages(conversationId, beforeId ? { before_id: beforeId } : {})
    // 请求期间切了会话：丢弃结果，避免把 A 的消息贴到 B 的面板
    if (Number(activeConv.value?.id) !== Number(conversationId)) return

    // 接口返回正序（旧 → 新），面板按「最新在上」展示
    const rows = (res.data?.data || []).slice().reverse()
    messages.value = beforeId ? [...messages.value, ...rows] : rows
    hasMore.value = !!res.data?.has_more

    // 会话资料（含双方头像昵称）以这次返回为准，列表里的可能是旧快照
    const conv = res.data?.conversation
    if (conv && Number(conv.id) === Number(conversationId)) {
      activeConv.value = { ...activeConv.value, ...conv }
    }
  } catch {
    /* 拦截器已提示 */
  } finally {
    loadingMsg.value = false
    loadingOlder.value = false
  }
}

async function openConversation(item) {
  if (Number(activeConv.value?.id) === Number(item.id)) return
  activeConv.value = item
  messages.value = []
  hasMore.value = false
  await loadMessages(item.id, 0)
}

function closeConversation() {
  activeConv.value = null
  messages.value = []
  hasMore.value = false
}

function loadOlder() {
  const last = messages.value[messages.value.length - 1]
  if (!last || !activeConv.value || loadingOlder.value) return
  loadMessages(activeConv.value.id, last.id)
}

// ---------- 撤回 ----------
function askRevoke(item) {
  const preview = item.msg_type === 'image' ? '[图片]' : String(item.content || '').slice(0, 40)
  confirm.message = `确定撤回这条私信吗？撤回后双方都会看到「撤回了一条消息」，内容不可恢复。\n\n${preview}`
  confirm.action = async () => {
    busy.value = true
    try {
      await revokeAdminChatMessage(item.id)
      item.is_revoked = 1
      item.content = ''
      toast.success('已撤回')
      // 会话摘要可能因此回退，顺手刷一次列表
      loadConversations(true)
    } finally {
      busy.value = false
    }
  }
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

onMounted(() => loadConversations(true))
</script>

<style scoped>
.chat-admin {
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

/* ---------- 左右分栏 ---------- */
.chat-layout {
  display: flex;
  gap: 16px;
  min-height: 520px;
}
.conv-side {
  width: 320px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding-right: 16px;
  border-right: 1px solid var(--border-soft);
}
.conv-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* ---------- 搜索 ---------- */
.search-box {
  position: relative;
  margin-bottom: 10px;
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

/* ---------- 会话行 ---------- */
.conv-list {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 0;
  list-style: none;
  overflow-y: auto;
  scrollbar-width: thin;
}
.conv-row {
  display: flex;
  gap: 10px;
  width: 100%;
  padding: 10px 8px;
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius);
  cursor: pointer;
  transition: background-color 0.15s;
}
.conv-row:hover {
  background: var(--bg-muted);
}
.conv-row.is-active {
  background: var(--accent-soft);
}
.conv-avatars {
  display: flex;
  flex-shrink: 0;
  padding-top: 2px;
}
.conv-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
  border: 2px solid var(--bg-card);
}
.conv-avatars .conv-avatar + .conv-avatar {
  margin-left: -10px;
}
.conv-avatar.is-head {
  width: 26px;
  height: 26px;
  border-width: 1px;
}
.conv-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.conv-top {
  display: flex;
  align-items: center;
  gap: 8px;
}
.conv-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.conv-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
}
.conv-bottom {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.conv-uids {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
}
.conv-text {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--text-muted);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

/* ---------- 消息面板 ---------- */
.conv-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-soft);
}
.conv-head-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.conv-back {
  display: none;
}
.msg-scroll {
  flex: 1;
  min-height: 0;
  max-height: 520px;
  overflow-y: auto;
  scrollbar-width: thin;
  padding-top: 12px;
}

.dm-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 6px;
  border-bottom: 1px dashed var(--border-soft);
}
.dm-row:last-child {
  border-bottom: none;
}
.dm-row:hover {
  background: var(--bg-muted);
}
.dm-avatar {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.dm-main {
  flex: 1;
  min-width: 0;
}
.dm-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 4px;
}
.dm-name {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--text);
}
.meta-text {
  font-size: 12px;
  color: var(--text-muted);
}
.quote-chip {
  max-width: 240px;
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
.dm-content {
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  word-break: break-word;
}
.dm-text {
  white-space: pre-wrap;
}
.dm-row.is-revoked .dm-text {
  color: var(--text-light);
  text-decoration: line-through;
}
.revoked-text {
  font-size: 12px;
  color: var(--text-light);
}
.dm-image {
  max-width: 160px;
  max-height: 160px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-soft);
}
.dm-revoke {
  flex-shrink: 0;
}
.dm-revoke.danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}

/* ---------- 其他 ---------- */
.is-loading {
  padding: 40px;
  text-align: center;
  color: var(--text-muted);
}
.empty-row {
  padding: 12px 0;
}
.load-more {
  display: flex;
  justify-content: center;
  padding-top: 12px;
}
.placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 40px 20px;
  text-align: center;
  color: var(--text-light);
}
.placeholder-mark {
  width: 58px;
  height: 58px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: var(--primary);
  background: var(--accent-wash);
  border: 1px solid var(--primary-soft);
  border-radius: 50%;
}
.placeholder p {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.8;
}

.is-spin {
  animation: ca-spin 0.9s linear infinite;
}
@keyframes ca-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ---------- 窄屏：列表 / 消息二选一 ---------- */
@media (max-width: 900px) {
  .chat-layout {
    display: block;
    min-height: 0;
  }
  .conv-side {
    width: 100%;
    padding-right: 0;
    border-right: 0;
  }
  .chat-admin.is-chat .conv-side {
    display: none;
  }
  .chat-admin:not(.is-chat) .conv-main {
    display: none;
  }
  .conv-back {
    display: inline-flex;
  }
  .msg-scroll {
    max-height: none;
  }
}
</style>
