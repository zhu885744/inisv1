<template>
  <div class="message-admin">
    <!-- 概览：统计随当前视图切换口径 -->
    <section class="card card-pad panel">
      <header class="panel-head">
        <div>
          <h2 class="block-title">消息管理</h2>
          <p class="block-desc">
            {{ currentView.desc }}
            <RouterLink to="/notifications" class="panel-link">
              个人消息请到「用户中心 → 消息通知」查看
            </RouterLink>
          </p>
        </div>
        <div class="head-actions">
          <button class="btn btn-primary btn-sm" @click="openSend()">
            <i class="bi bi-send" /> 发送短消息
          </button>
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
          :class="{ 'is-link': !!s.click, 'is-active': !!s.active }"
          :aria-pressed="s.click ? !!s.active : undefined"
          :title="s.click ? `查看${s.label}` : undefined"
          @click="s.click && s.click()"
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
      <!-- ============ 视图切换 ============
           群发短消息：uid=0 的广播（全站只存一条记录，撤回对全体生效）
           全部通知：全站所有记录（含发给指定用户的消息 + 评论/点赞/收藏/关注等自动通知）
           文案（icon / label）取自 VIEWS；点击后重置筛选（状态、类型、已读、关键词）并
           重新拉取列表 + 统计，详见 script 里的 switchView() -->
      <div class="view-tabs">
        <button
          v-for="v in VIEWS"
          :key="v.key"
          type="button"
          class="status-tab"
          :class="{ 'is-active': view === v.key }"
          @click="switchView(v.key)"
        >
          <i :class="v.icon" /> {{ v.label }}
        </button>
      </div>

      <!-- 当前视图的标题与说明：标题与说明文案统一在 VIEWS 里维护（label / listDesc ...），
           模板只做展示；total 为当前筛选条件下的总条数（后端返回的 count） -->
      <header class="list-head">
        <div>
          <h2 class="block-title">{{ currentView.label }}</h2>
          <p class="block-desc">
            {{ currentView.listDesc }}
            <template v-if="total > 0"> · 共 {{ total }} 条</template>
          </p>
        </div>
      </header>

      <div class="list-filter">
        <div class="status-tabs">
          <button
            type="button"
            class="status-tab"
            :class="{ 'is-active': !trash }"
            @click="setTrash(false)"
          >{{ currentView.stateNormal }}</button>
          <button
            type="button"
            class="status-tab trash-tab"
            :class="{ 'is-active': trash }"
            @click="setTrash(true)"
          >
            <i class="bi bi-trash3" /> {{ currentView.stateTrash }}
          </button>
        </div>

        <div class="filter-right">
          <FormSelect variant="pill"
            v-if="view === 'all'"
            v-model="typeFilter"
            :options="typeOptions"
            icon="bi bi-tags"
            placeholder="全部类型"
            @change="reload" />

          <div class="search-box">
            <i class="bi bi-search" aria-hidden="true" />
            <FormInput
              v-model="keyword"
              class="search-input"
              type="search"
              placeholder="搜索标题…"
              aria-label="搜索标题"
              @input="doSearch" />
          </div>

          <button class="btn btn-sm" :disabled="loading" @click="loadList()">
            <i class="bi bi-arrow-clockwise" /> 刷新
          </button>
        </div>
      </div>

      <!-- 批量操作（回收站之外） -->
      <div v-if="selectedIds.length && !trash" class="batch-bar">
        <span class="batch-count">已选 <strong>{{ selectedIds.length }}</strong> 条</span>
        <div class="batch-actions">
          <button class="btn btn-sm" :disabled="busy" @click="askBatchRemove()">
            <i class="bi bi-eye-slash" /> 批量撤回
          </button>
          <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
            <i class="bi bi-x-octagon" /> 彻底删除
          </button>
          <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
        </div>
      </div>

      <!-- 回收站：提示与批量操作合并成一条，避免与批量条叠成两条 -->
      <div v-if="trash" class="trash-bar">
        <span class="batch-count">
          <i class="bi bi-trash3" />
          <template v-if="selectedIds.length">已选 <strong>{{ selectedIds.length }}</strong> 条</template>
          <template v-else>{{ currentView.trashHint }}</template>
        </span>
        <div class="batch-actions">
          <template v-if="selectedIds.length">
            <button class="btn btn-sm" :disabled="busy" @click="batchRestore()">
              <i class="bi bi-arrow-counterclockwise" /> 批量恢复
            </button>
            <button class="btn btn-sm btn-danger" :disabled="busy" @click="askBatchForceDelete()">
              <i class="bi bi-x-octagon" /> 彻底删除
            </button>
            <button class="btn btn-sm btn-ghost" :disabled="busy" @click="clearSelection">取消选择</button>
          </template>
          <!-- 清空回收站是「自己 + 广播」范围，只在群发短消息视图提供；全站记录请用批量彻底删除 -->
          <button
            v-else-if="view === 'broadcast'"
            class="btn btn-sm btn-danger"
            :disabled="busy || !list.length"
            @click="askClearRecycle()"
          >
            清空回收站
          </button>
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
              :disabled="busy || !list.length"
              aria-label="全选本页"
              @change="toggleSelectAll" />
          </label>
          <span class="list-head-text">本页 {{ list.length }} 条</span>
        </div>

        <ul class="notice-list">
          <li v-for="item in list" :key="item.id" class="notice-row" :class="{ 'is-selected': isSelected(item.id) }">
            <label class="pick" :title="isSelected(item.id) ? '取消选择' : '选择'">
              <FormCheckbox
                :model-value="isSelected(item.id)"
                :disabled="busy"
                :aria-label="`选择消息 ${item.title}`"
                @change="toggleSelect(item.id)" />
            </label>

            <span class="type-chip" :class="typeClass(item)">{{ typeLabel(item) }}</span>

            <div class="notice-main">
              <div class="notice-title-row">
                <span v-if="isUnread(item)" class="unread-dot" title="未读" />
                <span class="notice-title" :class="{ 'is-unread': isUnread(item) }">{{ item.title }}</span>
                <span class="id-chip">#{{ item.id }}</span>
                <span class="id-chip is-to">{{ recipientText(item) }}</span>
              </div>
              <p class="notice-content">{{ item.content || '—' }}</p>
              <div class="notice-meta">
                <span class="meta-text"><i class="bi bi-clock" /> {{ timeText(item) }}</span>
                <span v-if="item.result?.from_user?.nickname" class="meta-text">
                  <i class="bi bi-person" /> 来自 {{ item.result.from_user.nickname }}
                </span>
                <span v-if="isRead(item)" class="meta-text">
                  <i class="bi bi-check2-all" /> 接收用户已读
                </span>
                <span v-if="item.bind_type" class="meta-text">
                  {{ item.bind_type }}<template v-if="Number(item.bind_id) > 0"> #{{ item.bind_id }}</template>
                </span>
              </div>
            </div>

            <div class="notice-actions">
              <template v-if="trash">
                <button class="btn btn-ghost btn-sm" title="恢复" aria-label="恢复" :disabled="busy" @click="restore(item)">
                  <i class="bi bi-arrow-counterclockwise" />
                </button>
                <button class="btn btn-ghost btn-sm danger" title="彻底删除" aria-label="彻底删除" :disabled="busy" @click="askForceDelete(item)">
                  <i class="bi bi-x-octagon" />
                </button>
              </template>
              <template v-else>
                <button
                  v-if="editable(item)"
                  class="btn btn-ghost btn-sm"
                  title="编辑"
                  aria-label="编辑"
                  :disabled="busy"
                  @click="openEdit(item)"
                >
                  <i class="bi bi-pencil" />
                </button>
                <button
                  class="btn btn-ghost btn-sm danger"
                  :title="removeTitle(item)"
                  :aria-label="removeTitle(item)"
                  :disabled="busy"
                  @click="askRemove(item)"
                >
                  <i class="bi bi-eye-slash" />
                </button>
              </template>
            </div>
          </li>
        </ul>
      </template>

      <Pagination
        v-if="!loading && total > 0"
        :current="page"
        :total="total"
        :page-size="pageSize"
        :sizes="PAGE_SIZE_OPTIONS"
        @update:page-size="(size) => { pageSize = size; page = 1; loadList() }"
        @update:current="changePage"
      />
    </div>

    <!-- 发送短消息 -->
    <AdminFormDialog
      v-model:visible="send.visible"
      :title="send.title"
      icon="bi bi-send"
      :loading="send.loading"
      :confirm-text="send.target === 'all' ? '发送给全体' : '发送'"
      @confirm="submitSend"
    >
      <div class="form-item">
        <label class="form-label">发送目标</label>
        <FormSelect v-model="send.target" :options="targetOptions" />
        <p class="form-hint">
          全体用户采用广播模式（仅一条记录、全体可见）；指定用户会为每人单独创建记录
        </p>
      </div>

      <div v-if="send.target !== 'all'" class="form-item">
        <label class="form-label">用户 ID</label>
        <FormInput v-model="send.userIds" type="text" placeholder="多个用英文逗号分隔，如 1,2,3" />
        <p class="form-hint">请输入接收消息的用户 ID（可在「用户管理」中查看）</p>
      </div>

      <div class="form-item">
        <label class="form-label">通知标题</label>
        <FormInput v-model="send.titleText" type="text" maxlength="128" placeholder="必填" />
      </div>

      <div class="form-item">
        <label class="form-label">通知内容</label>
        <FormTextarea v-model="send.content" rows="4" placeholder="必填" />
      </div>

      <div class="form-item">
        <label class="check-line">
          <FormCheckbox v-model="send.asSystem" />
          <span>以系统身份发送（标题自动加「【短消息】」前缀）</span>
        </label>
        <label class="check-line" :class="{ 'is-disabled': send.target === 'all' }">
          <FormCheckbox v-model="send.sendEmail" :disabled="send.target === 'all'" />
          <span>同时发送邮件通知（使用「用户消息通知」邮件模板，仅对已填写邮箱的用户生效）</span>
        </label>
        <p v-if="send.target === 'all'" class="form-hint">广播模式不支持邮件通知</p>
      </div>
    </AdminFormDialog>

    <!-- 编辑通知（仅短消息可编辑） -->
    <AdminFormDialog
      v-model:visible="edit.visible"
      :title="`编辑通知 #${edit.id}`"
      icon="bi bi-pencil-square"
      :loading="edit.loading"
      confirm-text="保存"
      @confirm="saveEdit"
    >
      <div class="form-item">
        <label class="form-label">通知标题</label>
        <FormInput v-model="edit.title" type="text" maxlength="128" placeholder="必填" />
      </div>

      <div class="form-item">
        <label class="form-label">通知内容</label>
        <FormTextarea v-model="edit.content" rows="4" placeholder="必填" />
      </div>
    </AdminFormDialog>

    <!-- 操作确认 -->
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
import FormCheckbox from '@/components/form/FormCheckbox.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import { PAGE_SIZE_DEFAULT, PAGE_SIZE_OPTIONS } from '@/utils/page-size'
/**
 * 消息管理（/admin/message）—— 管理端专用
 *
 * 两个视图（顶部切换）：
 *  1. 群发短消息：发给全体用户的短消息（uid=0，全站只存一条记录）：发送、编辑、撤回、恢复、彻底删除、清空回收站
 *  2. 全部通知：全站所有通知（群发短消息 + 发给指定用户的消息 + 评论/点赞/收藏/关注等自动通知）：
 *     查看接收用户与已读状态，按类型 / 标题筛选，编辑（短消息）、撤回、恢复、彻底删除
 * 个人消息（当前账号收到的通知）不在此页处理，请到用户中心「消息通知」页（/notifications）。
 *
 * 交互约定：
 * - 统计卡可点击，等价于点对应的筛选标签（active 高亮当前筛选）
 * - 列表支持「全选本页 + 批量操作」，切换视图 / 状态 / 筛选、刷新、翻页时自动清空选择
 * - 回收站状态下只显示一条操作条（提示 + 批量恢复/彻底删除 [+ 清空回收站]），不与批量条叠加
 *
 * 后端约束（app/api/controller/notification.go、app/model/notification.go）：
 * - 可见范围（one / all / count / rand / column / sum|min|max 统一走 applyScope）：
 *   普通用户仅自己的通知；root 为自己 + 广播（uid=0）；root 且 scope=admin 时不限制 uid
 *   ——「全部通知」视图即用该视角，写入侧（update / remove / delete / restore）同样支持；
 * - 广播（uid=0）只存一条记录，用户的「已读 / 隐藏」状态在 notification_read 表：
 *   后台删除广播 = 全体撤回（软删除），普通用户删除 = 仅对自己隐藏；
 *   行上的 is_read 只对「发给指定用户的记录」有意义（它是接收用户的状态）；
 *   广播的已读状态按用户存在状态表里，行上的 is_read 不参与本页展示（因此已读/未读统计只算 uid != 0）；
 * - send-system 仅 root：target_type=all 发给全体（一条 uid=0 记录、不支持邮件），
 *   partial / single 逐个创建记录，send_email 仅在指定用户时生效；
 *   as_system=false 时正文前缀取管理员昵称，昵称为空回退「管理员」；
 * - 批量上限 notificationBatchLimit=200（remove / read-batch），超量请分批；
 * - clear（清空回收站）范围是「自己 + 广播」，因此只在群发短消息视图提供。
 */
import { ref, reactive, computed, watchEffect, onMounted } from 'vue'
import EmptyState from '@/components/EmptyState.vue'
import Pagination from '@/components/Pagination.vue'
import ConfirmDialog from '@/components/ConfirmDialog.vue'
import AdminFormDialog from '@/components/admin/AdminFormDialog.vue'
import {
  listBroadcasts,
  listManagedNotifications,
  countNotifications,
  sendSystemMessage,
  updateNotification,
  removeNotifications,
  forceDeleteNotifications,
  restoreNotifications,
  clearNotificationRecycle
} from '@/api/notification'
import { listUsersByIds } from '@/api/users'
import { fromNow } from '@/utils/time'
import { debounce } from '@/utils/helper'
import { toast } from '@/utils/toast'

const pageSize = ref(PAGE_SIZE_DEFAULT)

// 管理端视角：读 / 写操作都带上（后端仅对 root 生效，其他账号传了会被忽略）
const ADMIN = { scope: 'admin' }

// 视图定义：文案统一在这里维护，模板里只做展示
const VIEWS = [
  {
    key: 'broadcast',
    label: '短消息',
    icon: 'bi bi-megaphone',
    stateNormal: '已发布',
    stateTrash: '已撤回',
    desc: '发给全体用户的短消息（全站只存一条记录），支持发送、编辑、撤回与恢复',
    listDesc: '群发给全体用户的短消息只存一条记录（uid=0）；撤回后对所有用户立即不可见',
    trashHint: '撤回的群发短消息不再对任何用户展示，可恢复或彻底删除',
    empty: '还没有群发短消息，发送一条吧',
    emptyTrash: '没有已撤回的群发短消息',
    noun: '短消息'
  },
  {
    key: 'all',
    label: '全部通知',
    icon: 'bi bi-inboxes',
    stateNormal: '全部',
    stateTrash: '回收站',
    desc: '全部通知：群发短消息、发给指定用户的消息，以及评论 / 点赞 / 收藏 / 关注等自动通知',
    listDesc: '全部通知记录（含发给指定用户的消息与自动通知）；撤回后仅接收用户不可见，可恢复或彻底删除',
    trashHint: '回收站内的通知可恢复或彻底删除',
    empty: '暂无通知记录',
    emptyTrash: '回收站是空的',
    noun: '通知'
  }
]

// 通知类型（与后端 NotificationType* 常量对应）
const typeOptions = [
  { value: '', label: '全部类型' },
  { value: 'system', label: '短消息' },
  { value: 'comment', label: '评论/回复' },
  { value: 'like', label: '点赞' },
  { value: 'collect', label: '收藏' },
  { value: 'follow', label: '关注' },
  { value: 'article', label: '文章' },
  { value: 'page', label: '页面' },
  { value: 'links', label: '友链' },
  { value: 'moments', label: '动态' },
  // 内容被兑换（付费可见）：兑换者消耗积分后发给内容作者
  { value: 'unlock', label: '兑换' }
]

const targetOptions = [
  { value: 'all', label: '全体用户（广播）' },
  { value: 'partial', label: '指定用户' }
]

// ===== 列表状态（两个视图共用一套）=====
const view = ref('broadcast')
const list = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const trash = ref(false)
const keyword = ref('')
const searchKey = ref('')
const typeFilter = ref('')
const readFilter = ref('all')
const selectedIds = ref([])
const busy = ref(false)

// ===== 统计 =====
const loadingStats = ref(false)
const stats = reactive({
  // 群发短消息视图
  broadcast: 0,
  broadcastTrash: 0,
  latest: 0,
  // 全部通知视图
  all: 0,
  unread: 0,
  read: 0,
  trashed: 0
})

const currentView = computed(() => VIEWS.find((v) => v.key === view.value) || VIEWS[0])

// 统计卡与筛选联动：点击卡片 = 点对应的筛选标签
const statCards = computed(() => {
  if (view.value === 'broadcast') {
    return [
      {
        label: '群发短消息',
        value: stats.broadcast,
        icon: 'bi bi-megaphone',
        color: 'var(--primary)',
        active: !trash.value,
        click: () => setTrash(false)
      },
      {
        label: '已撤回',
        value: stats.broadcastTrash,
        icon: 'bi bi-eye-slash',
        color: 'var(--text-muted)',
        active: trash.value,
        click: () => setTrash(true)
      },
      {
        label: '最近发送',
        value: stats.latest ? fromNow(stats.latest) : '—',
        icon: 'bi bi-clock',
        color: 'var(--success)'
      }
    ]
  }

  return [
    {
      label: '通知总数',
      value: stats.all,
      icon: 'bi bi-bell',
      color: 'var(--primary)',
      active: !trash.value && readFilter.value === 'all',
      click: () => setRead('all')
    },
    {
      label: '未读',
      value: stats.unread,
      icon: 'bi bi-envelope',
      color: 'var(--warning)',
      active: !trash.value && readFilter.value === 'unread',
      click: () => setRead('unread')
    },
    {
      label: '已读',
      value: stats.read,
      icon: 'bi bi-check2-circle',
      color: 'var(--success)',
      active: !trash.value && readFilter.value === 'read',
      click: () => setRead('read')
    },
    {
      label: '已删除',
      value: stats.trashed,
      icon: 'bi bi-trash3',
      color: 'var(--text-muted)',
      active: trash.value,
      click: () => setTrash(true)
    }
  ]
})

// 接收用户昵称（uid -> 昵称），仅「全部通知」视图按当前页按需拉取
const userMap = reactive({})

// ===== 弹窗 =====
const send = reactive({
  visible: false,
  loading: false,
  target: 'all',
  title: '发送短消息',
  titleText: '',
  content: '',
  userIds: '',
  asSystem: true,
  sendEmail: false
})

const edit = reactive({
  visible: false,
  loading: false,
  id: 0,
  title: '',
  content: ''
})

const confirm = reactive({
  visible: false,
  title: '操作确认',
  message: '',
  confirmText: '确定',
  danger: false,
  loading: false,
  action: null
})

// ---------- 展示辅助 ----------
const TYPE_LABEL = {
  comment: '评论/回复',
  like: '点赞',
  collect: '收藏',
  follow: '关注',
  article: '文章',
  page: '页面',
  links: '友链',
  moments: '动态',
  unlock: '兑换',
  system: '短消息'
}

function typeLabel(item) {
  const type = String(item?.type || '')
  return TYPE_LABEL[type] || type || '通知'
}

function typeClass(item) {
  return `is-${String(item?.type || 'other')}`
}

function recipientName(item) {
  const uid = Number(item?.uid || 0)
  if (uid <= 0) return '全体用户'
  return userMap[uid] || `用户 #${uid}`
}

function recipientText(item) {
  return Number(item?.uid || 0) <= 0 ? '全体用户' : `→ ${recipientName(item)}`
}

// 已读状态只对「发给指定用户的记录」有意义：
// 广播（uid=0）的已读状态按用户在 notification_read 表中，行上的 is_read 不是某个用户的状态
function isUnread(item) {
  return Number(item?.uid || 0) > 0 && Number(item?.is_read) === 0
}

function isRead(item) {
  return Number(item?.uid || 0) > 0 && Number(item?.is_read) === 1
}

// 只有短消息（群发 / 管理员推送）才提供编辑
function editable(item) {
  return String(item?.type || '') === 'system'
}

function removeTitle(item) {
  return Number(item?.uid || 0) === 0 ? '撤回（全体不可见）' : '撤回（接收用户不可见）'
}

function timeText(item) {
  if (trash.value) return `删除于 ${fromNow(item.delete_time)}`
  return fromNow(item.create_time)
}

const emptyText = computed(() => {
  if (searchKey.value) return '没有匹配的通知'
  if (trash.value) return currentView.value.emptyTrash
  return currentView.value.empty
})

const emptyIcon = computed(() => {
  if (trash.value) return 'bi bi-trash3'
  return view.value === 'broadcast' ? 'bi bi-megaphone' : 'bi bi-inboxes'
})

// ---------- 统计加载 ----------
async function loadStats() {
  loadingStats.value = true
  try {
    if (view.value === 'broadcast') await loadBroadcastStats()
    else await loadAllStats()
  } finally {
    loadingStats.value = false
  }
}

async function loadBroadcastStats() {
  // count 接口默认按当前用户过滤，全体可见的短消息数量只能用列表接口的 count
  const [normalRes, trashRes] = await Promise.all([
    listBroadcasts({ page: 1, limit: 1, field: 'id,create_time' }),
    listBroadcasts({ page: 1, limit: 1, field: 'id', onlyTrashed: true })
  ])
  stats.broadcast = Number(normalRes?.data?.count || 0)
  stats.broadcastTrash = Number(trashRes?.data?.count || 0)
  stats.latest = Number(normalRes?.data?.data?.[0]?.create_time || 0)
}

async function loadAllStats() {
  // 管理端视角统计全站通知（与列表筛选无关，卡片本身就是筛选入口）
  // 已读 / 未读只统计「发给指定用户的记录」（uid != 0）：广播的已读状态按用户在状态表里，行上的 is_read 无意义
  const [totalRes, unreadRes, readRes, trashRes] = await Promise.all([
    countNotifications({ ...ADMIN }),
    countNotifications({ ...ADMIN, where: JSON.stringify({ is_read: 0, uid: { $ne: 0 } }) }),
    countNotifications({ ...ADMIN, where: JSON.stringify({ is_read: 1, uid: { $ne: 0 } }) }),
    countNotifications({ ...ADMIN, onlyTrashed: true })
  ])
  stats.all = Number(totalRes?.data || 0)
  stats.unread = Number(unreadRes?.data || 0)
  stats.read = Number(readRes?.data || 0)
  stats.trashed = Number(trashRes?.data || 0)
}

// ---------- 列表加载 ----------
async function loadList() {
  loading.value = true
  try {
    const params = { page: page.value, limit: pageSize.value }
    if (trash.value) params.onlyTrashed = true

    const kw = searchKey.value.replace(/['"\\%_|]/g, '').trim()
    if (kw) params.like = `title|${kw}`

    let res
    if (view.value === 'broadcast') {
      res = await listBroadcasts(params)
    } else {
      const where = {}
      if (typeFilter.value) where.type = typeFilter.value
      // 已读筛选同样只看「发给指定用户的记录」（广播的已读状态不在这张表上）
      if (readFilter.value === 'unread') Object.assign(where, { is_read: 0, uid: { $ne: 0 } })
      if (readFilter.value === 'read') Object.assign(where, { is_read: 1, uid: { $ne: 0 } })
      if (Object.keys(where).length) params.where = JSON.stringify(where)

      res = await listManagedNotifications(params)
    }

    list.value = res.data?.data || []
    total.value = res.data?.count || 0
    clearSelection()

    if (view.value === 'all') loadRecipients()
  } catch {
    list.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 按当前页的接收用户补昵称（拉不到时回落显示「用户 #id」）
async function loadRecipients() {
  const uids = [
    ...new Set(list.value.map((i) => Number(i.uid)).filter((uid) => uid > 0 && !(uid in userMap)))
  ]
  if (!uids.length) return

  try {
    const res = await listUsersByIds(uids)
    for (const user of res?.data?.data || []) {
      userMap[Number(user.id)] = user.nickname || ''
    }
  } catch {
    /* 失败时保留「用户 #id」展示 */
  }
}

function reload() {
  page.value = 1
  loadList()
}

const doSearch = debounce(() => {
  searchKey.value = keyword.value
  reload()
}, 350)

// 视图切换：重置筛选，避免把上一视图的条件带过去
async function switchView(key) {
  view.value = key
  trash.value = false
  typeFilter.value = ''
  readFilter.value = 'all'
  keyword.value = ''
  searchKey.value = ''
  page.value = 1
  clearSelection()
  await Promise.all([loadList(), loadStats()])
}

// 统一的筛选入口（状态 + 已读），变化时才重新拉列表
async function applyFilter({ trashed, read }) {
  if (trash.value === trashed && readFilter.value === read) return
  trash.value = trashed
  readFilter.value = read
  page.value = 1
  clearSelection()
  await loadList()
}

// 切换「未删除 / 回收站」：进入回收站时把已读筛选重置为全部
function setTrash(value) {
  return applyFilter({ trashed: value, read: value ? 'all' : readFilter.value })
}

// 统计卡：按已读状态筛选（并回到未删除状态）
function setRead(key) {
  return applyFilter({ trashed: false, read: key })
}

function changePage(p) {
  if (p < 1 || p === page.value) return
  page.value = p
  loadList()
}

// ---------- 选择 ----------
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

// ---------- 发送短消息 ----------
// 默认发给全体（本页的常用场景是群发），对话框里可切换为指定用户
function openSend(target = 'all') {
  send.target = target
  send.title = '发送短消息'
  send.titleText = ''
  send.content = ''
  send.userIds = ''
  send.asSystem = true
  send.sendEmail = false
  send.visible = true
}

async function submitSend() {
  const title = send.titleText.trim()
  const content = send.content.trim()
  if (!title) {
    toast.warning('请输入通知标题')
    return
  }
  if (!content) {
    toast.warning('请输入通知内容')
    return
  }

  const payload = { title, content, as_system: send.asSystem }
  if (send.target === 'all') {
    payload.target_type = 'all'
  } else {
    const ids = String(send.userIds || '')
      .split(',')
      .map((s) => Number(s.trim()))
      .filter((n) => Number.isInteger(n) && n > 0)
    if (!ids.length) {
      toast.warning('请输入有效的用户 ID')
      return
    }
    payload.target_type = ids.length === 1 ? 'single' : 'partial'
    payload.user_ids = ids
    payload.send_email = send.sendEmail
  }

  send.loading = true
  try {
    const res = await sendSystemMessage(payload)
    toast.success(send.target === 'all' ? '短消息已发送，全体用户可见' : `推送完成：成功 ${res?.data?.success || 0} 个`)
    send.visible = false
    // 发送后切到对应视图，方便立刻核对刚发出的记录
    await switchView(send.target === 'all' ? 'broadcast' : 'all')
  } catch {
    // 失败提示由请求拦截器统一给出（如无权限）
  } finally {
    send.loading = false
  }
}

// ---------- 编辑 ----------
function openEdit(item) {
  edit.id = Number(item.id)
  edit.title = item.title || ''
  edit.content = item.content || ''
  edit.visible = true
}

async function saveEdit() {
  const title = edit.title.trim()
  const content = edit.content.trim()
  if (!title) {
    toast.warning('请输入通知标题')
    return
  }
  if (!content) {
    toast.warning('请输入通知内容')
    return
  }

  edit.loading = true
  try {
    await updateNotification({ id: edit.id, title, content, ...ADMIN })
    toast.success('通知已更新')
    edit.visible = false
    await Promise.all([loadList(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    edit.loading = false
  }
}

// ---------- 撤回 / 恢复 / 彻底删除 ----------
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
    // 请求失败时保持弹窗打开，错误提示已由请求拦截器统一给出
  } finally {
    confirm.loading = false
  }
}

/** 撤回 = 软删除：广播（uid=0）对全体不可见；指定用户的消息仅该用户不可见 */
function askRemove(item) {
  const isBroadcast = Number(item?.uid || 0) === 0
  openConfirm({
    title: isBroadcast ? '撤回群发短消息' : '撤回通知',
    message: isBroadcast
      ? `确定撤回群发短消息「${item.title}」吗？撤回后对所有用户立即不可见，可在「已撤回」中恢复。`
      : `确定撤回发给「${recipientName(item)}」的通知「${item.title}」吗？撤回后该用户不可见，可在「回收站」中恢复。`,
    confirmText: '撤回',
    danger: true,
    action: async () => {
      await removeNotifications([item.id], ADMIN)
      toast.success('已撤回')
      await Promise.all([loadList(), loadStats()])
    }
  })
}

async function restore(item) {
  busy.value = true
  try {
    await restoreNotifications([item.id], ADMIN)
    toast.success('已恢复')
    await Promise.all([loadList(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askForceDelete(item) {
  openConfirm({
    title: '彻底删除',
    message: `确定彻底删除「${item.title}」吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteNotifications([item.id], ADMIN)
      toast.success('已彻底删除')
      await Promise.all([loadList(), loadStats()])
    }
  })
}

function askBatchRemove() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量撤回',
    message: `确定撤回选中的 ${ids.length} 条${currentView.value.noun}吗？撤回后接收用户不可见，可在回收站中恢复。`,
    confirmText: '撤回',
    danger: true,
    action: async () => {
      await removeNotifications(ids, ADMIN)
      toast.success(`已撤回 ${ids.length} 条`)
      clearSelection()
      await Promise.all([loadList(), loadStats()])
    }
  })
}

async function batchRestore() {
  const ids = [...selectedIds.value]
  busy.value = true
  try {
    await restoreNotifications(ids, ADMIN)
    toast.success(`已恢复 ${ids.length} 条`)
    clearSelection()
    await Promise.all([loadList(), loadStats()])
  } catch {
    /* 拦截器已提示 */
  } finally {
    busy.value = false
  }
}

function askBatchForceDelete() {
  const ids = [...selectedIds.value]
  openConfirm({
    title: '批量彻底删除',
    message: `确定彻底删除选中的 ${ids.length} 条${currentView.value.noun}吗？此操作不可恢复！`,
    confirmText: '彻底删除',
    danger: true,
    action: async () => {
      await forceDeleteNotifications(ids, ADMIN)
      toast.success(`已彻底删除 ${ids.length} 条`)
      clearSelection()
      await Promise.all([loadList(), loadStats()])
    }
  })
}

// 清空回收站（后端范围 = 自己 + 广播，只在群发短消息视图使用）
function askClearRecycle() {
  openConfirm({
    title: '清空回收站',
    message: '确定清空回收站吗？其中所有群发短消息将被彻底删除，不可恢复！',
    confirmText: '清空',
    danger: true,
    action: async () => {
      await clearNotificationRecycle()
      toast.success('回收站已清空')
      page.value = 1
      clearSelection()
      await Promise.all([loadList(), loadStats()])
    }
  })
}

onMounted(async () => {
  await Promise.all([loadList(), loadStats()])
})
</script>

<style scoped>
.message-admin {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ---------- 概览 ---------- */
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
  color: var(--text-muted);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
/* 概览描述里的跳转链接（去用户中心看个人消息） */
.panel-link {
  margin-left: 4px;
  color: var(--primary);
}
.panel-link:hover {
  text-decoration: underline;
}

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
  border: 1px solid transparent;
  border-radius: var(--radius);
  transition: background 0.15s, border-color 0.15s;
}
/* 可点击的统计卡（点击等价于点对应的筛选标签） */
.stat-card.is-link {
  cursor: pointer;
}
.stat-card.is-link:hover {
  background: var(--bg-card);
  border-color: var(--border);
}
.stat-card.is-link.is-active {
  background: var(--accent-wash);
  border-color: var(--primary);
}
.stat-card.is-link.is-active .stat-label {
  color: var(--primary-deep);
}
.stat-card.is-link:focus-visible {
  outline: 2px solid var(--primary-soft);
  outline-offset: 2px;
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

/* ---------- 视图切换（群发短消息 / 全部通知） ---------- */
.view-tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-soft);
  /* 与下方「模块标题（.list-head）」之间的间距：不加会紧贴在一起，视觉上像同一块 */
  margin-bottom: 16px;
}

/* ---------- 状态标签（未删除 / 回收站） ---------- */
.status-tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  /* 与下拉 / 搜索框 / 按钮等高 */
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

/* ---------- 批量 / 回收站 ---------- */
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

.trash-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
  padding: 10px 12px;
  font-size: 12px;
  background: var(--gold-wash);
  border: 1px solid var(--gold-line);
  border-radius: var(--radius);
}
/* 回收站条里复用 .batch-count 时保持警示色 */
.trash-bar .batch-count {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--warning);
}
.trash-bar .batch-count strong {
  color: var(--primary-deep);
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

.notice-list {
  display: flex;
  flex-direction: column;
}
.notice-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 6px;
  border-bottom: 1px dashed var(--border-soft);
  border-radius: var(--radius-sm);
  transition: background 0.15s;
}
.notice-row:last-child {
  border-bottom: none;
}
.notice-row:hover {
  background: var(--bg-muted);
}
.notice-row.is-selected {
  background: var(--accent-wash);
}

.pick {
  display: flex;
  align-items: center;
  height: 22px;
  cursor: pointer;
}
.pick input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}

/* 类型标签：颜色只做区分，不承载额外语义 */
.type-chip {
  flex-shrink: 0;
  padding: 2px 8px;
  font-size: 11px;
  border-radius: 3px;
  color: var(--text-muted);
  background: var(--bg-muted);
}
.type-chip.is-system {
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.type-chip.is-comment {
  color: #0ea5e9;
  background: rgba(14, 165, 233, 0.12);
}
.type-chip.is-like {
  color: var(--danger);
  background: rgba(239, 68, 68, 0.12);
}
.type-chip.is-collect {
  color: var(--warning);
  background: var(--gold-wash);
}
.type-chip.is-follow {
  color: var(--success);
  background: rgba(16, 185, 129, 0.12);
}

.notice-main {
  flex: 1;
  min-width: 0;
}
.notice-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.notice-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.notice-title.is-unread {
  color: var(--primary-deep);
}
.unread-dot {
  width: 7px;
  height: 7px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--danger);
}
.id-chip {
  padding: 1px 6px;
  font-size: 11px;
  color: var(--text-muted);
  background: var(--bg-muted);
  border-radius: 3px;
}
/* 接收用户标签（uid=0 显示「全体用户」） */
.id-chip.is-to {
  color: var(--text-soft);
  background: var(--accent-wash);
}
.notice-content {
  margin: 4px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--text-soft);
  /* 内容较长时最多显示两行 */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.notice-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-muted);
}
.meta-text {
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.notice-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.notice-actions .danger:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}
.notice-actions .btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ---------- 弹窗 ---------- */
.form-item {
  margin-bottom: 14px;
}
.check-line {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
  color: var(--text-soft);
  cursor: pointer;
}
.check-line input {
  width: 15px;
  height: 15px;
  accent-color: var(--primary);
  cursor: pointer;
}
.check-line.is-disabled {
  color: var(--text-muted);
  cursor: not-allowed;
}
.form-hint {
  margin: 6px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: var(--text-muted);
}

@media (max-width: 768px) {
  .list-filter {
    gap: 8px;
  }
  .filter-right,
  .search-box {
    width: 100%;
    margin-left: 0;
  }
  .notice-row {
    flex-wrap: wrap;
  }
  .notice-main {
    flex: 1 1 60%;
  }
  .notice-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .stat-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
