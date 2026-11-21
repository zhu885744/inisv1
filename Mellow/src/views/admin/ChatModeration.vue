<template>
  <div class="chat-moderation-admin">
    <!-- 概览 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">聊天审核</h2>
          <p class="block-desc">
            私信 / 公共群聊命中词库时自动留证（只存命中片段，不含整条消息）；确认违规后到「私信会话」/「群聊消息」
            撤回，或到「用户管理」处理账号
          </p>
        </div>
        <div class="head-actions">
          <button
            class="btn btn-ghost btn-sm"
            :disabled="loadingStats"
            title="刷新统计"
            aria-label="刷新统计"
            @click="loadStats"
          >
            <i class="bi bi-arrow-clockwise" />
          </button>
        </div>
      </header>

      <div class="stat-grid" :aria-busy="loadingStats">
        <div
          v-for="s in statCards"
          :key="s.label"
          class="stat-card"
          :class="{ 'is-link': !!s.pick }"
          @click="s.pick && s.pick()"
        >
          <span class="stat-icon" :style="{ color: s.color }"><i :class="s.icon" /></span>
          <span class="stat-body">
            <span class="stat-value">{{ loadingStats ? '···' : s.value }}</span>
            <span class="stat-label">{{ s.label }}</span>
          </span>
        </div>
      </div>
    </section>

    <div class="card card-pad">
      <header class="list-head">
        <div>
          <h2 class="block-title">命中记录</h2>
          <p class="block-desc">
            命中片段由服务端截取（前后各 40 字），图片消息不参与判定
            <template v-if="total > 0"> · 共 {{ total }} 条</template>
          </p>
        </div>
      </header>

      <!-- 工具栏 -->
      <div class="list-filter">
        <div class="status-tabs">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            class="status-tab"
            :class="{ 'is-active': handled === tab.key }"
            @click="switchHandled(tab.key)"
          >{{ tab.label }}</button>
        </div>

        <div class="filter-right">
          <FormSelect variant="pill"
            v-model="sceneFilter"
            :options="sceneOptions"
            icon="bi bi-chat-square-dots"
            placeholder="全部场景"
            @change="reload" />

          <FormSelect variant="pill"
            v-model="actionFilter"
            :options="actionOptions"
            icon="bi bi-sliders"
            placeholder="全部动作"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索命中词…"
              aria-label="搜索命中词"
              @input="doSearch" />
          </div>

          <div class="search-box is-narrow">
            <i class="bi bi-person" aria-hidden="true" />
            <FormInput
              v-model="uidText"
              class="search-input"
              type="number"
              min="1"
              placeholder="发送者 UID"
              aria-label="按发送者 UID 筛选"
              @input="doSearch" />
          </div>

          <button class="btn btn-sm" :disabled="loading" @click="load()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 批量操作 -->
      <div v-if="selectedIds.length" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 条</span>
        <div class="batch-actions">
          <button class="btn btn-sm btn-primary" :disabled="busy" @click="askHandle(selectedIds, 1)">
            <i class="bi bi-check2-circle" /> 标记已处理
          </button>
          <button class="btn btn-sm" :disabled="busy" @click="askHandle(selectedIds, 0)">
            <i class="bi bi-arrow-counterclockwise" /> 恢复待处理
          </button>
          <button class="btn btn-sm btn-danger" :disabled="busy" @click="askRemove(selectedIds)">
            <i class="bi bi-trash" /> 删除
          </button>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <div v-if="loading" class="is-loading"><span class="spinner" /> 加载中...</div>

      <div v-else-if="!list.length" class="empty-row">
        <EmptyState :icon="emptyIcon" :text="emptyText" />
      </div>

      <template v-else>
        <div class="list-head-row">
          <label class="pick" title="全选本页">
            <FormCheckbox
              ref="selectAllRef"
              :model-value="pageAllSelected"
              :disabled="busy"
              aria-label="全选本页"
              @change="toggleSelectAll" />
          </label>
          <span class="list-head-text">本页 {{ list.length }} 条</span>
        </div>

        <table class="moderation-table">
          <thead>
            <tr>
              <th class="cell-pick"></th>
              <th class="cell-scene">场景</th>
              <th>发送者</th>
              <th class="cell-word">命中</th>
              <th>片段</th>
              <th class="cell-time">时间</th>
              <th class="cell-state">状态</th>
              <th class="cell-act">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in list" :key="item.id" :class="{ 'is-selected': isSelected(item.id) }">
              <td class="cell-pick">
                <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
                  <FormCheckbox
                    :model-value="isSelected(item.id)"
                    :disabled="busy"
                    :aria-label="`选择记录 ${item.id}`"
                    @change="toggleSelect(item.id)" />
                </label>
              </td>

              <td class="cell-scene">
                <span class="scene-chip" :class="`is-${item.scene || 'dm'}`">
                  <i :class="item.scene === 'chat' ? 'bi bi-people-fill' : 'bi bi-envelope-fill'" aria-hidden="true" />
                  {{ sceneLabel(item.scene) }}
                </span>
              </td>

              <td>
                <div class="user-cell">
                  <img class="user-avatar" :src="item.avatar || defaultAvatar" alt="" loading="lazy" />
                  <span class="user-body">
                    <span class="user-name">{{ item.nickname || `用户 #${item.uid}` }}</span>
                    <span class="meta-text">
                      #{{ item.uid }}
                      <template v-if="Number(item.to_uid) > 0"> → {{ item.to_nickname || `#${item.to_uid}` }}</template>
                    </span>
                  </span>
                </div>
              </td>

              <td class="cell-word">
                <span class="word-chip">{{ item.word || '—' }}</span>
                <span class="meta-text">
                  {{ actionLabel(item.action) }}<template v-if="item.level"> · {{ levelLabel(item.level) }}</template>
                </span>
              </td>

              <td>
                <p class="excerpt" :title="item.excerpt">{{ item.excerpt || '—' }}</p>
              </td>

              <td class="cell-time">
                <span class="meta-text">{{ formatTime(item.create_time) }}</span>
                <span class="meta-text">#{{ item.msg_id || '—' }}</span>
              </td>

              <td class="cell-state">
                <span v-if="Number(item.handled) === 1" class="state-chip is-handled">已处理</span>
                <span v-else class="state-chip is-pending">待处理</span>
                <span v-if="item.remark" class="meta-text remark-text" :title="item.remark">{{ item.remark }}</span>
              </td>

              <td class="cell-act">
                <div class="row-actions">
                  <button
                    class="btn btn-ghost btn-sm is-punish"
                    title="处置：撤回 / 禁言 / 封禁"
                    aria-label="处置"
                    :disabled="busy"
                    @click="askPunish(item)"
                  >
                    <i class="bi bi-shield-exclamation" />
                  </button>
                  <button
                    v-if="Number(item.handled) === 1"
                    class="btn btn-ghost btn-sm"
                    title="恢复待处理"
                    aria-label="恢复待处理"
                    :disabled="busy"
                    @click="askHandle([item.id], 0)"
                  >
                    <i class="bi bi-arrow-counterclockwise" />
                  </button>
                  <button
                    v-else
                    class="btn btn-ghost btn-sm"
                    title="标记已处理"
                    aria-label="标记已处理"
                    :disabled="busy"
                    @click="askHandle([item.id], 1)"
                  >
                    <i class="bi bi-check2-circle" />
                  </button>
                  <button
                    class="btn btn-ghost btn-sm danger"
                    title="删除"
                    aria-label="删除"
                    :disabled="busy"
                    @click="askRemove([item.id])"
                  >
                    <i class="bi bi-trash" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </template>

      <Pagination
        v-if="!loading && total > 0"
        :current="page"
        :total="total"
        :page-size="pageSize"
        :sizes="PAGE_SIZE_OPTIONS"
        @update:page-size="(size) => { pageSize = size; page = 1; load() }"
        @update:current="changePage"
      />
    </div>

    <!-- 操作确认（标记处理时可选填备注） -->
    <ConfirmDialog
      v-model:visible="confirm.visible"
      :title="confirm.title"
      :message="confirm.message"
      :confirm-text="confirm.confirmText"
      :danger="confirm.danger"
      :loading="confirm.loading"
      @confirm="runConfirm"
    >
      <label v-if="confirm.withRemark" class="remark-field">
        <span class="remark-label">处理备注（可选，最多 255 字）</span>
        <FormTextarea
          v-model="confirm.remark"
          class="remark-input"
          rows="2"
          maxlength="255"
          placeholder="例如：已撤回并警告 / 疑似误伤" />
      </label>
    </ConfirmDialog>

    <!-- 处置：撤回 / 禁言 / 封禁 一次做完 -->
    <ConfirmDialog
      v-model:visible="punish.visible"
      title="处置违规用户"
      :message="punishTip"
      confirm-text="执行处置"
      :loading="punish.loading"
      @confirm="runPunish"
    >
      <div class="punish-form">
        <label class="punish-check">
          <FormCheckbox v-model="punish.revoke" :disabled="!punish.record?.msg_id" />
          <span>
            撤回这条消息
            <em v-if="!punish.record?.msg_id">（这条记录没有消息 ID，无法撤回）</em>
          </span>
        </label>

        <label class="punish-field">
          <span>禁言（私信 + 群聊同时生效）</span>
          <FormSelect variant="pill" v-model="punish.muteMinutes" :options="punishOptions" />
        </label>

        <label class="punish-field">
          <span>原因 / 备注</span>
          <FormInput
            v-model="punish.reason"
            class="remark-input"
            type="text"
            maxlength="120"
            placeholder="会写进处理备注与禁言原因" />
        </label>

        <p class="punish-hint">
          禁言后该用户不能发私信、也不能在群聊发言；到期自动解除，也可到「禁言名单」提前解除。
        </p>
      </div>
    </ConfirmDialog>
  </div>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 聊天审核（/admin/chat/moderation）
 *
 * 后端约束（app/api/controller/chat-moderation.go、app/model/chat_moderation.go）：
 * - 记录由消息命中词库时自动写入，不支持手动新增；
 * - 只存命中片段（命中词前后各 40 字），图片消息不参与判定；
 * - 页面只做「查看 + 标记处理 + 删除」，撤回 / 禁言 / 封号各自走对应页面。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import {
  listChatModeration,
  countChatModeration,
  handleChatModeration,
  removeChatModeration
} from '@/api/chat-moderation'
import { muteChatUser, revokeAdminChatMessage } from '@/api/chat-admin'
import { formatTime } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// handled 筛选：'' 全部 / '0' 待处理 / '1' 已处理
const tabs = [
  { key: '0', label: '待处理' },
  { key: '1', label: '已处理' },
  { key: '', label: '全部' }
]

const sceneOptions = [
  { value: '', label: '全部场景' },
  { value: 'dm', label: '私信' },
  { value: 'chat', label: '公共群聊' }
]

const actionOptions = [
  { value: '', label: '全部动作' },
  { value: 'block', label: '拒绝发送' },
  { value: 'limit', label: '限流拒绝' },
  { value: 'review', label: '放行待审核' },
  { value: 'warn', label: '放行并提示' },
  { value: 'log', label: '仅记录' }
]

// ===== 列表 =====
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const busy = ref(false)
const handled = ref('0')
const sceneFilter = ref('')
const actionFilter = ref('')
const keyword = ref('')
const uidText = ref('')
const searchKey = ref('')
const searchUid = ref(0)
const selectedIds = ref([])

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({ pending: 0, dm: 0, chat: 0, handled: 0 })

const confirm = reactive({
  visible: false,
  title: '操作确认',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  withRemark: false,
  remark: '',
  action: null
})

const statCards = computed(() => [
  { label: '待处理', value: stats.pending, icon: 'bi bi-exclamation-circle', color: 'var(--danger)', pick: () => switchHandled('0') },
  { label: '私信命中', value: stats.dm, icon: 'bi bi-envelope-fill', color: 'var(--primary)', pick: () => pickScene('dm') },
  { label: '群聊命中', value: stats.chat, icon: 'bi bi-people-fill', color: 'var(--warning)', pick: () => pickScene('chat') },
  { label: '已处理', value: stats.handled, icon: 'bi bi-check2-circle', color: 'var(--success)', pick: () => switchHandled('1') }
])

const emptyIcon = computed(() => {
  if (searchKey.value || searchUid.value) return 'bi bi-search'
  if (handled.value === '0') return 'bi bi-check2-circle'
  return 'bi bi-chat-square-dots'
})

const emptyText = computed(() => {
  if (searchKey.value || searchUid.value) return '没有匹配的命中记录'
  if (handled.value === '0') return '没有待处理的记录'
  if (handled.value === '1') return '还没有处理过的记录'
  return '暂无聊天命中记录'
})

// ---------- 展示辅助 ----------
function sceneLabel(scene) {
  return scene === 'chat' ? '群聊' : '私信'
}

function actionLabel(action) {
  const map = {
    block: '拒绝发送',
    limit: '限流拒绝',
    review: '放行待审核',
    warn: '放行并提示',
    log: '仅记录',
    allow: '未命中'
  }
  return map[action] || action || '—'
}

function levelLabel(level) {
  const map = { high: '高风险', medium: '中风险', low: '低风险' }
  return map[level] || level
}

// ---------- 筛选 ----------
function buildParams() {
  const params = {}
  if (handled.value !== '') params.handled = Number(handled.value)
  if (sceneFilter.value) params.scene = sceneFilter.value
  if (actionFilter.value) params.action = actionFilter.value
  if (searchUid.value) params.uid = searchUid.value
  if (searchKey.value) params.word = searchKey.value
  return params
}

// ---------- 统计 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    const [pending, dm, chat, done] = await Promise.all([
      countChatModeration({ handled: 0 }),
      countChatModeration({ scene: 'dm' }),
      countChatModeration({ scene: 'chat' }),
      countChatModeration({ handled: 1 })
    ])
    stats.pending = Number(pending?.data || 0)
    stats.dm = Number(dm?.data || 0)
    stats.chat = Number(chat?.data || 0)
    stats.handled = Number(done?.data || 0)
  } catch {
    /* 统计失败不阻塞列表 */
  } finally {
    loadingStats.value = false
  }
}

// ---------- 列表 ----------
async function load() {
  loading.value = true
  try {
    const res = await listChatModeration({
      page: page.value,
      limit: pageSize.value,
      ...buildParams()
    })
    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

function reload() {
  page.value = 1
  load()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value.trim()
  searchUid.value = Number(uidText.value) > 0 ? Number(uidText.value) : 0
  reload()
}, 350)

function switchHandled(key) {
  if (handled.value === key) return
  handled.value = key
  reload()
}

function pickScene(scene) {
  sceneFilter.value = sceneFilter.value === scene ? '' : scene
  reload()
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  load()
}

// ---------- 选中 ----------
const selectedSet = computed(() => new Set(selectedIds.value))
const pageAllSelected = computed(
  () => list.value.length > 0 && list.value.every((i) => selectedSet.value.has(i.id))
)
const someSelected = computed(() => selectedIds.value.length > 0 && !pageAllSelected.value)
const selectAllRef = ref(null)

watchEffect(() => {
  if (selectAllRef.value) selectAllRef.value.setIndeterminate(someSelected.value)
})

function isSelected(id) {
  return selectedSet.value.has(id)
}

function toggleSelect(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = [...next]
}

function toggleSelectAll() {
  selectedIds.value = pageAllSelected.value ? [] : list.value.map((i) => i.id)
}

function clearSelection() {
  selectedIds.value = []
}

async function afterMutation(removedCount = 0) {
  if (removedCount > 0 && list.value.length <= removedCount && page.value > 1) {
    page.value -= 1
  }
  await load()
  loadStats()
}

// ---------- 确认框 ----------
function openConfirm({ title, message, confirmText = '确定', danger = false, withRemark = false, action }) {
  confirm.title = title
  confirm.message = message
  confirm.confirmText = confirmText
  confirm.danger = danger
  confirm.withRemark = withRemark
  confirm.remark = ''
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
    const ok = await confirm.action(confirm.remark)
    if (ok !== false) confirm.visible = false
  } catch {
    // 请求失败保持弹窗打开，错误提示已由请求拦截器统一给出
  } finally {
    confirm.loading = false
  }
}

// ---------- 标记处理 / 删除 ----------
function askHandle(ids, nextHandled) {
  const targets = [...ids]
  const marking = Number(nextHandled) === 1
  openConfirm({
    title: marking ? '标记已处理' : '恢复待处理',
    message: marking
      ? `确定把选中的 ${targets.length} 条记录标记为已处理吗？`
      : `确定把选中的 ${targets.length} 条记录恢复为待处理吗？处理痕迹会被清空。`,
    confirmText: marking ? '标记已处理' : '恢复待处理',
    withRemark: marking,
    action: async (remark) => {
      busy.value = true
      try {
        await handleChatModeration(targets, nextHandled, remark || '')
        toast.success(marking ? '已标记处理' : '已恢复待处理')
        clearSelection()
        await afterMutation(0)
      } finally {
        busy.value = false
      }
    }
  })
}

function askRemove(ids) {
  const targets = [...ids]
  openConfirm({
    title: '删除记录',
    message: `确定删除选中的 ${targets.length} 条记录吗？删除后不可恢复（消息本身不受影响）。`,
    confirmText: '删除',
    danger: true,
    action: async () => {
      busy.value = true
      try {
        await removeChatModeration(targets)
        toast.success('已删除')
        clearSelection()
        await afterMutation(targets.length)
      } finally {
        busy.value = false
      }
    }
  })
}

// ---------- 处置（撤回 / 禁言） ----------
//
// 两个动作各自独立，失败不应互相阻断：比如消息已被本人撤回（撤回失败），
// 该禁言的还是要禁。这里逐个 attempt，最后汇总提示，并把这条记录标记为已处理。
const punishOptions = [
  { value: -1, label: '不处理' },
  { value: 10, label: '10 分钟' },
  { value: 60, label: '1 小时' },
  { value: 1440, label: '1 天' },
  { value: 10080, label: '7 天' },
  { value: 43200, label: '30 天' },
  { value: 0, label: '永久' }
]

const punish = reactive({
  visible: false,
  loading: false,
  record: null,
  revoke: true,
  muteMinutes: -1,
  reason: ''
})

const punishTip = computed(() => {
  const rec = punish.record
  if (!rec) return ''
  const who = rec.nickname || `用户 #${rec.uid}`
  return `正在处置「${who}」在${sceneLabel(rec.scene)}里命中的记录，勾选要执行的动作后确认（禁言会同时作用于私信与群聊）。`
})

function askPunish(item) {
  punish.record = item
  // 没有消息 ID（留证时消息尚未落库）就没什么可撤回的，默认不勾
  punish.revoke = Number(item.msg_id) > 0
  punish.muteMinutes = -1
  punish.reason = ''
  punish.visible = true
}

function minutesText(minutes) {
  const value = Number(minutes)
  if (value === 0) return '永久'
  if (value < 60) return `${value} 分钟`
  if (value < 1440) return `${Math.round(value / 60)} 小时`
  return `${Math.round(value / 1440)} 天`
}

async function runPunish() {
  const rec = punish.record
  if (!rec) return

  const reason = punish.reason.trim()
  const fallback = '聊天审核处置'
  const done = []
  const failed = []

  const attempt = async (label, fn) => {
    try {
      await fn()
      done.push(label)
    } catch {
      failed.push(label)
    }
  }

  punish.loading = true
  try {
    if (punish.revoke && Number(rec.msg_id) > 0) {
      await attempt('撤回消息', () =>
        revokeAdminChatMessage(rec.msg_id, rec.scene === 'chat' ? 'chat' : 'dm')
      )
    }

    if (Number(punish.muteMinutes) >= 0) {
      await attempt(`禁言 ${minutesText(punish.muteMinutes)}`, () =>
        muteChatUser(rec.uid, punish.muteMinutes, reason || fallback)
      )
    }

    // 处置完顺手把这条记录标记已处理，备注写这次做了什么
    await attempt('标记处理', () =>
      handleChatModeration([rec.id], 1, reason || done.join('、') || fallback)
    )
  } finally {
    punish.loading = false
    punish.visible = false
  }

  if (failed.length) toast.warning(`部分操作失败：${failed.join('、')}`)
  else toast.success(done.length ? `已处置：${done.join('、')}` : '已标记处理')

  clearSelection()
  await afterMutation(0)
}

onMounted(() => {
  load()
  loadStats()
})
</script>

<style scoped>
.chat-moderation-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-head,
.list-head {
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

/* ---------- 统计 ---------- */
.stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 12px;
}
.stat-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: var(--bg-muted);
  border-radius: var(--radius);
}
.stat-card.is-link {
  cursor: pointer;
  transition: box-shadow 0.15s;
}
.stat-card.is-link:hover {
  box-shadow: 0 0 0 1px var(--primary) inset;
}
.stat-icon {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  background: var(--bg-card);
  font-size: 18px;
}
.stat-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.stat-value {
  font-size: 18px;
  font-weight: 700;
  line-height: 1.2;
}
.stat-label {
  font-size: 12px;
  color: var(--text-muted);
}

/* ---------- 工具栏 ---------- */
.list-filter {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
}
.status-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}
.status-tab {
  height: var(--control-h-sm);
  padding: 0 14px;
  font-size: 13px;
  border-radius: var(--radius-sm);
  border: 1px solid transparent;
  background: var(--bg-muted);
  color: var(--text-soft);
  cursor: pointer;
  transition: all 0.15s;
}
.status-tab:hover {
  color: var(--primary);
}
.status-tab.is-active {
  background: var(--primary);
  color: #fff;
}
.filter-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-left: auto;
}
.search-box {
  position: relative;
  width: 170px;
}
.search-box.is-narrow {
  width: 130px;
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

/* ---------- 批量 ---------- */
.batch-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--accent-wash);
  border: 1px solid var(--accent-soft);
  border-radius: var(--radius);
}
.batch-count {
  font-size: 13px;
  color: var(--text-soft);
}
.batch-count strong {
  color: var(--primary-deep);
}
.batch-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
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
.list-head-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 8px 4px;
}
.list-head-text {
  font-size: 12px;
  color: var(--text-muted);
}

.moderation-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}
.moderation-table th {
  padding: 8px 10px;
  font-weight: 500;
  text-align: left;
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.moderation-table td {
  padding: 10px;
  border-bottom: 1px dashed var(--border-soft);
  color: var(--text-soft);
  vertical-align: middle;
}
.moderation-table tr:last-child td {
  border-bottom: none;
}
.moderation-table tbody tr:hover td {
  background: var(--bg-muted);
}
.moderation-table tbody tr.is-selected td {
  background: var(--accent-wash);
}

.cell-pick {
  width: 36px;
}
.cell-scene {
  width: 96px;
}
.cell-word {
  width: 150px;
}
.cell-time {
  width: 130px;
}
.cell-state {
  width: 120px;
}
.cell-act {
  width: 90px;
  text-align: right;
}

.pick {
  display: flex;
  align-items: center;
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}

.scene-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: 11px;
  font-weight: 600;
  border-radius: 999px;
}
.scene-chip.is-dm {
  color: var(--primary);
  background: var(--accent-soft);
}
.scene-chip.is-chat {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.user-avatar {
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.user-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.user-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--text-muted);
}

.word-chip {
  display: inline-block;
  max-width: 140px;
  padding: 1px 6px;
  margin-bottom: 2px;
  font-size: 12px;
  color: var(--danger);
  background: var(--accent-soft);
  border-radius: 3px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  vertical-align: middle;
}

.excerpt {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.6;
  color: var(--text-soft);
  display: -webkit-box;
  line-clamp: 2;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.state-chip {
  display: inline-block;
  padding: 1px 8px;
  font-size: 11px;
  border-radius: 999px;
}
.state-chip.is-pending {
  color: var(--danger);
  background: var(--accent-soft);
}
.state-chip.is-handled {
  color: var(--success);
  background: rgba(108, 154, 77, 0.12);
}
.remark-text {
  display: block;
  max-width: 120px;
  margin-top: 3px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.row-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}
.row-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.row-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 处置表单 ---------- */
.punish-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.punish-check {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text);
  cursor: pointer;
}
.punish-check input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}
.punish-check em {
  font-style: normal;
  font-size: 12px;
  color: var(--text-light);
}
.punish-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;
  color: var(--text-soft);
}
.punish-field > span {
  flex-shrink: 0;
}
.punish-field .remark-input {
  flex: 1;
  max-width: 220px;
}
.punish-hint {
  margin: 2px 0 0;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-light);
}

/* ---------- 处理备注 ---------- */
.remark-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.remark-label {
  font-size: 12px;
  color: var(--text-muted);
}
.remark-input {
  width: 100%;
  padding: 8px 10px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text);
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: none;
  resize: vertical;
}
.remark-input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--accent-ring);
}

@media (max-width: 768px) {
  .list-filter,
  .filter-right,
  .search-box,
  .search-box.is-narrow {
    width: 100%;
    margin-left: 0;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .moderation-table {
    display: block;
    overflow-x: auto;
  }
}
</style>
