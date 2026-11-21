<template>
  <div class="chat-app" :class="{ 'is-active': hasActive }">
    <!-- ==================== 最左：功能栏（QQ NT 窄条） ==================== -->
    <nav class="chat-rail">
      <RouterLink to="/user" class="chat-rail-avatar" :title="me ? '个人中心' : '登录'">
        <img v-if="avatar" :src="avatar" alt="" loading="lazy" />
        <i v-else class="bi bi-person-fill" aria-hidden="true" />
        <span v-if="totalUnread" class="chat-rail-dot" />
      </RouterLink>

      <RouterLink to="/chat" class="chat-rail-btn" :class="{ 'is-active': panel === 'chats' }" title="消息">
        <i class="bi bi-chat-dots-fill" aria-hidden="true" />
        <span v-if="totalUnread" class="chat-rail-badge">{{ totalUnread > 99 ? '99+' : totalUnread }}</span>
      </RouterLink>

      <!-- 联系人：互关好友 + 已加入的群聊，点谁就开谁的聊天窗口（同 QQ 的联系人页） -->
      <RouterLink
        :to="{ path: '/chat', query: { panel: 'contacts' } }"
        class="chat-rail-btn"
        :class="{ 'is-active': panel === 'contacts' }"
        title="联系人"
      >
        <i class="bi bi-people-fill" aria-hidden="true" />
      </RouterLink>

      <RouterLink to="/" class="chat-rail-btn" title="返回站点">
        <i class="bi bi-compass" aria-hidden="true" />
      </RouterLink>

      <button type="button" class="chat-rail-btn" title="消息提醒" @click="alertOpen = !alertOpen">
        <i
          :class="chatStore.alert.sound || chatStore.alert.toast || chatStore.alert.desktop ? 'bi bi-bell' : 'bi bi-bell-slash'"
          aria-hidden="true"
        />
      </button>

      <span class="chat-rail-grow" />

      <RouterLink to="/user" class="chat-rail-btn" title="设置">
        <i class="bi bi-gear" aria-hidden="true" />
      </RouterLink>
    </nav>

    <!-- ==================== 会话列表 ==================== -->
    <aside class="chat-side">
      <div class="chat-search-row">
        <label class="chat-search">
          <i class="bi bi-search" aria-hidden="true" />
          <FormInput
            v-model="keyword"
            type="search"
            :placeholder="panel === 'contacts' ? '搜索好友 / 群聊' : '搜索'"
            :aria-label="panel === 'contacts' ? '搜索好友或群聊' : '搜索会话'" />
        </label>
        <button v-if="panel === 'chats'" type="button" class="chat-add" title="新建群聊" @click="openCreate">
          <i class="bi bi-plus-lg" aria-hidden="true" />
        </button>
      </div>

      <!-- 窄屏没有最左功能栏：给一个面板切换，否则联系人进不去 -->
      <div class="chat-panel-tabs">
        <RouterLink to="/chat" class="chat-panel-tab" :class="{ 'is-active': panel === 'chats' }">
          <i class="bi bi-chat-dots" aria-hidden="true" />
          <span>消息</span>
        </RouterLink>
        <RouterLink
          :to="{ path: '/chat', query: { panel: 'contacts' } }"
          class="chat-panel-tab"
          :class="{ 'is-active': panel === 'contacts' }"
        >
          <i class="bi bi-people" aria-hidden="true" />
          <span>联系人</span>
        </RouterLink>
        <!-- 窄屏把 rail 藏了，「返回站点」也跟着消失，这里补一个，手机上才有回前台的路 -->
        <RouterLink to="/" class="chat-panel-home" title="返回站点" aria-label="返回站点">
          <i class="bi bi-compass" aria-hidden="true" />
        </RouterLink>
      </div>

      <ul class="chat-list">
        <!-- ==================== 消息：会话列表 ==================== -->
        <template v-if="panel === 'chats'">
        <!-- 公共大厅：固定置顶 -->
        <li v-if="hallVisible">
          <RouterLink to="/chat/room/1" class="chat-item" :class="{ 'is-active': isRoomActive(1) }">
            <span class="chat-item-avatar is-room">
              <img v-if="publicRoom.avatar" :src="fullUrl(publicRoom.avatar)" alt="" loading="lazy" />
              <i v-else class="bi bi-people-fill" aria-hidden="true" />
            </span>
            <span class="chat-item-body">
              <span class="chat-item-top">
                <strong>{{ publicRoom.name || '公共群聊' }}</strong>
                <em class="chat-item-tag">大厅</em>
              </span>
              <span class="chat-item-bottom">
                <span class="chat-item-desc">{{ publicRoom.description || '全站用户都在这里' }}</span>
                <span v-if="chatStore.roomMentions" class="chat-item-unread">
                  {{ chatStore.roomMentions > 99 ? '99+' : chatStore.roomMentions }}
                </span>
              </span>
            </span>
          </RouterLink>
        </li>

        <li v-for="row in rows" :key="row.key">
          <RouterLink :to="row.to" class="chat-item" :class="{ 'is-active': row.active }">
            <span class="chat-item-avatar" :class="{ 'is-room': row.type === 'room' }">
              <img v-if="row.avatar" :src="fullUrl(row.avatar)" alt="" loading="lazy" />
              <i v-else :class="row.type === 'room' ? 'bi bi-people-fill' : 'bi bi-person-fill'" aria-hidden="true" />
              <i v-if="row.online" class="chat-item-online" title="在线" />
            </span>

            <span class="chat-item-body">
              <span class="chat-item-top">
                <i v-if="row.pinned" class="bi bi-pin-angle-fill chat-item-flag" title="已置顶" aria-hidden="true" />
                <strong>{{ row.name }}</strong>
                <time v-if="row.time">{{ shortTime(row.time) }}</time>
              </span>
              <span class="chat-item-bottom">
                <span class="chat-item-desc">{{ row.preview }}</span>
                <!-- 未读数：与摘要同一行、贴最右（对齐 QQ NT 的会话列表） -->
                <span
                  v-if="row.unread"
                  class="chat-item-unread"
                  :class="{ 'is-muted': row.muted }"
                >{{ row.unread > 99 ? '99+' : row.unread }}</span>
                <i v-if="row.muted" class="bi bi-bell-slash chat-item-muted" title="免打扰" aria-hidden="true" />
              </span>
            </span>
          </RouterLink>
        </li>

        <li v-if="!loading && !rows.length && !hallVisible" class="chat-empty">
          <i :class="keyword ? 'bi bi-search' : 'bi bi-chat-square-dots'" aria-hidden="true" />
          <p>{{ keyword ? '没有匹配的会话' : '还没有聊过天' }}</p>
          <span v-if="!keyword">互相关注后即可私信，也可以新建一个群聊</span>
        </li>
        </template>

        <!-- ==================== 联系人：互关好友 + 群聊 ==================== -->
        <template v-else>
          <li class="chat-group-head">
            <strong>我的好友</strong>
            <em>{{ onlineFriendCount }}/{{ friends.length }}</em>
          </li>

          <li v-for="item in filteredFriends" :key="`friend-${item.uid}`">
            <RouterLink
              :to="{ path: `/chat/${item.uid}`, query: { panel: 'contacts' } }"
              class="chat-item"
              :class="{ 'is-active': Number(route.params.uid) === Number(item.uid) }"
            >
              <span class="chat-item-avatar">
                <img v-if="item.user?.avatar" :src="fullUrl(item.user.avatar)" alt="" loading="lazy" />
                <i v-else class="bi bi-person-fill" aria-hidden="true" />
                <i v-if="item.online" class="chat-item-online" title="在线" />
              </span>
              <span class="chat-item-body">
                <span class="chat-item-top">
                  <strong>{{ item.user?.nickname || `用户 #${item.uid}` }}</strong>
                </span>
                <span class="chat-item-bottom">
                  <span class="chat-item-status" :class="{ 'is-online': item.online }">
                    {{ item.online ? '在线' : '离线' }}
                  </span>
                  <span v-if="item.user?.title || item.user?.description" class="chat-item-desc">
                    {{ item.user.title || item.user.description }}
                  </span>
                </span>
              </span>
            </RouterLink>
          </li>

          <li v-if="!loadingFriends && !filteredFriends.length" class="chat-empty is-inline">
            <p>{{ keyword ? '没有匹配的好友' : '还没有互关好友' }}</p>
            <span v-if="!keyword">互相关注后，对方会出现在这里</span>
          </li>

          <li class="chat-group-head">
            <strong>群聊</strong>
            <em>{{ filteredRooms.length }}</em>
          </li>

          <li v-for="item in filteredRooms" :key="`contact-room-${item.id}`">
            <RouterLink
              :to="{ path: `/chat/room/${item.id}`, query: { panel: 'contacts' } }"
              class="chat-item"
              :class="{ 'is-active': isRoomActive(item.id) }"
            >
              <span class="chat-item-avatar is-room">
                <img v-if="item.avatar" :src="fullUrl(item.avatar)" alt="" loading="lazy" />
                <i v-else class="bi bi-people-fill" aria-hidden="true" />
              </span>
              <span class="chat-item-body">
                <span class="chat-item-top">
                  <strong>{{ item.name }}</strong>
                </span>
                <span class="chat-item-bottom">
                  <span class="chat-item-desc">{{ item.description || roomPreview(item) }}</span>
                </span>
              </span>
            </RouterLink>
          </li>

          <li v-if="!loading && !filteredRooms.length" class="chat-empty is-inline">
            <p>{{ keyword ? '没有匹配的群聊' : '还没有加入群聊' }}</p>
            <span v-if="!keyword">可以从「消息」页右上角新建一个群</span>
          </li>
        </template>
      </ul>
    </aside>

    <!-- ==================== 聊天区 ==================== -->
    <RouterView v-slot="{ Component }">
      <component :is="Component" v-if="Component" />
      <div v-else class="chat-blank">
        <i :class="panel === 'contacts' ? 'bi bi-person-vcard' : 'bi bi-chat-square-dots'" aria-hidden="true" />
        <p>{{ panel === 'contacts' ? '选择联系人查看资料并开始聊天' : '选择一个会话开始聊天' }}</p>
        <RouterLink v-if="panel === 'chats'" to="/chat/room/1" class="chat-blank-btn">进入公共大厅</RouterLink>
      </div>
    </RouterView>

    <!-- ==================== 提醒设置 ==================== -->
    <div v-if="alertOpen" class="chat-alert-mask" @click="alertOpen = false" />
    <div v-if="alertOpen" class="chat-alert-panel">
      <p class="chat-alert-title">消息提醒</p>

      <div class="chat-alert-row">
        <span class="chat-alert-copy">
          <strong>提示音</strong>
          <em>页面在前台时收到消息响一声</em>
        </span>
        <button
          type="button"
          class="chat-switch"
          :class="{ 'is-on': chatStore.alert.sound }"
          role="switch"
          :aria-checked="chatStore.alert.sound"
          @click="chatStore.setAlert('sound', !chatStore.alert.sound)"
        >
          <span class="chat-switch-knob" />
        </button>
      </div>

      <div class="chat-alert-row">
        <span class="chat-alert-copy">
          <strong>右下角浮层</strong>
          <em>在别的页面收到消息时弹出提示卡</em>
        </span>
        <button
          type="button"
          class="chat-switch"
          :class="{ 'is-on': chatStore.alert.toast }"
          role="switch"
          :aria-checked="chatStore.alert.toast"
          @click="chatStore.setAlert('toast', !chatStore.alert.toast)"
        >
          <span class="chat-switch-knob" />
        </button>
      </div>

      <div class="chat-alert-row">
        <span class="chat-alert-copy">
          <strong>浏览器通知</strong>
          <em>{{ desktopHint }}</em>
        </span>
        <button
          type="button"
          class="chat-switch"
          :class="{ 'is-on': desktopEnabled }"
          role="switch"
          :aria-checked="desktopEnabled"
          :disabled="!desktopSupported"
          @click="toggleDesktop"
        >
          <span class="chat-switch-knob" />
        </button>
      </div>

      <p class="chat-alert-tip">设置只保存在本机浏览器，不影响未读角标</p>
    </div>

    <!-- ==================== 新建群聊 ==================== -->
    <div v-if="createOpen" class="chat-modal">
      <div class="chat-modal-mask" @click="createOpen = false" />
      <div class="chat-modal-card">
        <header class="chat-modal-head">
          <span>新建群聊</span>
          <button type="button" class="chat-icon-btn" title="关闭" @click="createOpen = false">
            <i class="bi bi-x-lg" aria-hidden="true" />
          </button>
        </header>

        <div v-if="!canCreate.allowed" class="chat-limit">
          <i class="bi bi-info-circle" aria-hidden="true" />
          <span>{{ canCreate.reason }}</span>
        </div>

        <template v-else>
          <label class="chat-field">
            <span>名称</span>
            <FormInput v-model="form.name" maxlength="30" placeholder="2 ~ 30 个字" />
          </label>

          <label class="chat-field">
            <span>简介</span>
            <FormTextarea v-model="form.description" rows="3" maxlength="200" placeholder="一句话介绍这个群（可选）" />
          </label>

          <label class="chat-field">
            <span>入群方式</span>
            <FormSelect v-model="form.join_type" :options="joinTypeOptions" placeholder="请选择入群方式" />
          </label>

          <p class="chat-tip">每人最多创建 {{ canCreate.max_owned || 3 }} 个群，已创建 {{ canCreate.owned || 0 }} 个。</p>

          <div class="chat-modal-actions">
            <button type="button" class="chat-btn" @click="createOpen = false">取消</button>
            <button type="button" class="chat-btn chat-btn-primary" :disabled="creating" @click="submit">
              {{ creating ? '创建中…' : '创建' }}
            </button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import FormTextarea from '@/components/form/FormTextarea.vue'
import FormInput from '@/components/form/FormInput.vue'
import FormSelect from '@/components/form/FormSelect.vue'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { listChatConversations, listChatFriends } from '@/api/chat'
import { createRoom, listRooms } from '@/api/room'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { useDesktopNotice } from '@/composables/useDesktopNotice'
import { toast } from '@/utils/toast'
import { getFullUrl } from '@/utils/emoji'

/**
 * 消息外壳：私信 + 群聊共用一份列表（/chat）
 *
 * 布局仿 QQ NT：最左功能窄条 + 会话列表 + 聊天区，
 * 私信与群聊不再是两个页面，点哪个会话右栏就切到哪个（路由子页面）。
 *
 * 颜色：聊天页局部使用 --chat-* 变量（QQ 风格的蓝/浅灰），
 * 不改站点主色 —— 其它页面仍是原来的主题。
 */

const route = useRoute()
const router = useRouter()
const chatStore = useChatStore()
const userStore = useUserStore()

const hasActive = computed(() => !!(route.params.uid || route.params.id))

// 左栏面板：消息（会话列表）/ 联系人（互关好友 + 群聊）
//
// 用地址栏 query 而不是组件内 ref：刷新、前进后退、分享链接都能停在同一面板，
// 而且切换面板不会影响到右栏正在聊的会话（子路由不变）。
const panel = computed(() => (route.query.panel === 'contacts' ? 'contacts' : 'chats'))

const keyword = ref('')
const loading = ref(false)
const conversations = ref([])
const rooms = ref([])
const publicRoom = ref({})
// 互关好友（联系人页）：会话列表只含「聊过的人」，这里补上没聊过的
const friends = ref([])
const loadingFriends = ref(false)

const alertOpen = ref(false)

// 系统级桌面通知：与私信页共用同一个 composable（权限申请必须在点击栈里，
// 偏好仍落 chat store 的 alert.desktop，存本机）
const {
  supported: desktopSupported,
  enabled: desktopEnabled,
  blocked: desktopBlocked,
  toggle: toggleDesktop
} = useDesktopNotice()

const desktopHint = computed(() => {
  if (!desktopSupported) return '当前环境不支持（浏览器要求 HTTPS）'
  if (desktopBlocked.value) return '浏览器已拒绝，请在站点设置里允许通知'
  if (desktopEnabled.value) return '切到其他标签页也能收到'
  return '窗口在后台时也能收到系统通知'
})
const canCreate = ref({ allowed: false, reason: '', owned: 0, max_owned: 3 })
const createOpen = ref(false)
const creating = ref(false)
const form = ref({ name: '', description: '', join_type: 0 })

const joinTypeOptions = [
  { value: 0, label: '自由加入' },
  { value: 1, label: '需要审批' },
  { value: 2, label: '仅限邀请' }
]

const me = computed(() => Number(userStore.user?.id || 0))
const avatar = computed(() => (userStore.user?.avatar ? getFullUrl(userStore.user.avatar) : ''))

function fullUrl(url) {
  return getFullUrl(url || '')
}

function shortTime(value) {
  const time = Number(value) || 0
  if (!time) return ''

  const date = new Date(time * 1000)
  const now = new Date()
  if (date.toDateString() === now.toDateString()) {
    return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  }

  const days = Math.floor((now - date) / 86400000)
  if (days < 7) return `${days} 天前`
  return `${date.getMonth() + 1}-${date.getDate()}`
}

function roomPreview(item) {
  const last = item.last_message
  if (!last) return item.description || '还没有人说话'
  if (Number(last.is_revoked) === 1) return '[已撤回]'
  return last.msg_type === 'image' ? '[图片]' : String(last.content || '')
}

const rows = computed(() => {
  const list = []

  conversations.value.forEach((item) => {
    list.push({
      key: `dm-${item.other_uid}`,
      type: 'dm',
      id: Number(item.other_uid),
      to: `/chat/${item.other_uid}`,
      name: item.user?.nickname || `用户 #${item.other_uid}`,
      avatar: item.user?.avatar || '',
      preview: item.last_text || '还没有聊过',
      time: Number(item.last_time) || 0,
      unread: Number(item.unread) || 0,
      pinned: !!item.pinned,
      muted: !!item.muted,
      online: !!item.online,
      active: Number(route.params.uid) === Number(item.other_uid)
    })
  })

  rooms.value.forEach((item) => {
    list.push({
      key: `room-${item.id}`,
      type: 'room',
      id: Number(item.id),
      to: `/chat/room/${item.id}`,
      name: item.name,
      avatar: item.avatar || '',
      preview: roomPreview(item),
      time: Number(item.last_message?.create_time) || 0,
      unread: Number(item.unread) || 0,
      pinned: false,
      muted: false,
      online: false,
      active: isRoomActive(item.id)
    })
  })

  list.sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
    return b.time - a.time
  })

  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return list
  return list.filter((item) => String(item.name || '').toLowerCase().includes(kw))
})

const hallVisible = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return true
  return String(publicRoom.value.name || '公共群聊').toLowerCase().includes(kw)
})

const totalUnread = computed(() => {
  const dm = conversations.value.reduce((sum, item) => sum + (Number(item.unread) || 0), 0)
  const room = rooms.value.reduce((sum, item) => sum + (Number(item.unread) || 0), 0)
  return dm + room
})

// ---------- 联系人面板 ----------

const onlineFriendCount = computed(() => friends.value.filter((item) => item.online).length)

const filteredFriends = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return friends.value
  return friends.value.filter((item) => {
    const name = String(item.user?.nickname || '').toLowerCase()
    const intro = String(item.user?.title || item.user?.description || '').toLowerCase()
    return name.includes(kw) || intro.includes(kw)
  })
})

const filteredRooms = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return rooms.value
  return rooms.value.filter((item) => String(item.name || '').toLowerCase().includes(kw))
})

function isRoomActive(id) {
  return String(route.path).startsWith('/chat/room') && Number(route.params.id) === Number(id)
}

async function loadConversations() {
  if (!userStore.isLogged) return
  try {
    const res = await listChatConversations()
    if (res?.code !== 200) return
    conversations.value = res.data?.data || []
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

async function loadRooms() {
  try {
    const res = await listRooms({ scope: 'mine' })
    if (res?.code !== 200) return
    rooms.value = res.data?.data || []
    canCreate.value = res.data?.can_create || canCreate.value
    if (res.data?.public_room) publicRoom.value = res.data.public_room
  } catch {
    /* 错误已由 request.js 提示 */
  }
}

async function loadAll() {
  loading.value = true
  await Promise.all([loadConversations(), loadRooms()])
  loading.value = false
}

/**
 * 互关好友（联系人面板用）
 *
 * 单独拉、不并进 loadAll：好友关系很少变，没必要跟着会话列表一起重拉
 * （loadAll 在收到消息、兜底轮询时都会跑，多一个请求是纯浪费）。
 * 在线状态会变，所以每次切到联系人面板时补刷一次。
 */
async function loadFriends() {
  if (!userStore.isLogged) return
  loadingFriends.value = true
  try {
    const res = await listChatFriends()
    if (res?.code === 200) friends.value = res.data?.data || []
  } catch {
    /* 好友列表失败只影响联系人面板，静默降级（会话列表照常可用） */
  } finally {
    loadingFriends.value = false
  }
}

/** 游客点了需要登录的动作：提示一句并带上回跳地址去登录页 */
function requireLogin(message = '请先登录') {
  if (userStore.isLogged) return false
  toast.warning(message)
  router.push({ name: 'login', query: { redirect: route.fullPath } })
  return true
}

function openCreate() {
  if (requireLogin('请先登录后再创建群聊')) return

  if (!canCreate.value.allowed) {
    toast.warning(canCreate.value.reason || '暂时不能创建群聊')
    return
  }
  form.value = { name: '', description: '', join_type: 0 }
  createOpen.value = true
}

async function submit() {
  if (creating.value) return

  const name = form.value.name.trim()
  if (name.length < 2) {
    toast.warning('名称至少 2 个字')
    return
  }

  creating.value = true
  try {
    const res = await createRoom({
      name,
      description: form.value.description.trim(),
      join_type: Number(form.value.join_type) || 0
    })
    if (res?.code !== 200) return

    createOpen.value = false
    toast.success('创建成功')
    router.push(`/chat/room/${res.data?.id}`)
  } catch {
    /* 错误已由 request.js 提示 */
  } finally {
    creating.value = false
  }
}

let unwatchEvent = null
let refreshTimer = null
let incomingTimer = null
let fallbackTimer = null

// 兜底轮询间隔（毫秒）
//
// 实时通道正常时完全不需要它；但长连接会因为代理、切网、休眠而断掉，
// 断线期间如果只依赖推送，列表就彻底「冻住」，表现正是
// 「不刷新就看不到新消息」。这里只求能自愈，不追求秒级。
const FALLBACK_INTERVAL = 20000

/** 没连上长连接时，定时把会话列表重拉一遍 */
function startFallback() {
  if (fallbackTimer || !userStore.isLogged) return
  fallbackTimer = setInterval(() => {
    if (document.visibilityState !== 'visible') return
    loadAll()
  }, FALLBACK_INTERVAL)
}

function stopFallback() {
  clearInterval(fallbackTimer)
  fallbackTimer = null
}

/** 按当前连接状态决定兜底轮询的启停 */
function syncFallback() {
  if (!userStore.isLogged || chatStore.connected) {
    stopFallback()
    return
  }
  startFallback()
}

/** 页面回到前台：先补连一次，再把列表对齐（断线期间可能积了几条） */
function onVisible() {
  if (document.visibilityState !== 'visible') return
  if (!chatStore.connected) chatStore.reconnect()
  loadAll()
}

// 切到联系人面板时补刷一次：在线状态、新加入的群可能在别的页面变过
watch(
  () => panel.value,
  (value) => {
    if (value !== 'contacts') return
    loadFriends()
    loadRooms()
  }
)

onMounted(() => {
  chatStore.start()
  loadAll()
  loadFriends()
  syncFallback()

  unwatchEvent = router.afterEach((to, from) => {
    if (String(to.path).startsWith('/chat') && String(from.path).startsWith('/chat')) {
      clearTimeout(refreshTimer)
      refreshTimer = setTimeout(loadAll, 200)
    }
  })

  document.addEventListener('visibilitychange', onVisible)
})

// 掉线后主动重连，并让兜底轮询接管，避免连不上时列表一直不动
watch(
  () => chatStore.connected,
  (open) => {
    if (!open) chatStore.reconnect()
    syncFallback()
  }
)

watch(
  () => chatStore.incoming,
  (value) => {
    if (!value) return
    const peer =
      Number(value.from_uid) === Number(userStore.user?.id)
        ? Number(value.to_uid)
        : Number(value.from_uid)
    const row = conversations.value.find((item) => Number(item.other_uid) === peer)

    if (row) {
      row.last_text = value.msg_type === 'image' ? '[图片]' : String(value.content || '')
      row.last_time = Number(value.create_time) || row.last_time
    }

    // 陌生人发来的第一条消息在本地没有对应行，只打补丁永远补不出来 ——
    // 这就是「新会话必须刷新才出现」的原因。统一做一次防抖后的全量对齐，
    // 顺带把未读数、免打扰与置顶顺序一起摆正。
    clearTimeout(incomingTimer)
    incomingTimer = setTimeout(loadAll, 600)
  }
)

onBeforeUnmount(() => {
  if (unwatchEvent) unwatchEvent()
  clearTimeout(refreshTimer)
  clearTimeout(incomingTimer)
  stopFallback()
  document.removeEventListener('visibilitychange', onVisible)
})
</script>

<style scoped>
/* ==================== QQ 风局部配色 ==================== */
.chat-app {
  /* 全部映射到站点语义 token（styles.css 的设计变量），换主题 / 暗色自动跟随 */
  --chat-blue: var(--primary);
  --chat-blue-deep: var(--primary-deep);
  --chat-panel: var(--bg);
  --chat-side: var(--bg-card);
  --chat-bubble: var(--bg-card);
  --chat-line: var(--border-soft);
  --chat-text-weak: var(--text-light);

  display: flex;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
  color: var(--text);
  background: var(--bg);
}

/* ==================== 最左功能栏 ==================== */
.chat-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  width: 56px;
  flex-shrink: 0;
  padding: 14px 0 12px;
  background: var(--chat-side);
  border-right: 1px solid var(--chat-line);
}
.chat-rail-avatar {
  position: relative;
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  margin-bottom: 8px;
  overflow: hidden;
  font-size: 16px;
  color: #fff;
  background: var(--chat-blue);
  border-radius: 50%;
}
.chat-rail-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.chat-rail-dot {
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 9px;
  height: 9px;
  background: var(--danger);
  border: 2px solid var(--chat-side);
  border-radius: 50%;
}
.chat-rail-btn {
  position: relative;
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  font-size: 17px;
  color: var(--text-soft);
  border-radius: 8px;
  transition: background 0.15s, color 0.15s;
}
.chat-rail-btn:hover {
  color: var(--chat-blue);
  background: var(--accent-wash);
}
.chat-rail-btn.is-active {
  color: var(--chat-blue);
  background: var(--accent-soft);
}
.chat-rail-badge {
  position: absolute;
  top: 1px;
  right: -2px;
  min-width: 15px;
  padding: 0 4px;
  font-size: 10px;
  font-weight: 600;
  line-height: 15px;
  color: #fff;
  text-align: center;
  background: var(--danger);
  border-radius: 8px;
}
.chat-rail-grow {
  flex: 1;
}

/* ==================== 会话列表 ==================== */
.chat-side {
  display: flex;
  flex-direction: column;
  width: 276px;
  flex-shrink: 0;
  min-height: 0;
  background: var(--chat-side);
  border-right: 1px solid var(--chat-line);
}
.chat-search-row {
  display: flex;
  gap: 6px;
  padding: 14px 12px 10px;
}
.chat-search {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 0 10px;
  height: 30px;
  font-size: 12px;
  color: var(--chat-text-weak);
  background: var(--bg-muted);
  border-radius: 6px;
}
.chat-search input {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
  color: var(--text);
  background: transparent;
  border: 0;
  outline: none;
}
.chat-add {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  font-size: 15px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 6px;
}
.chat-add:hover {
  color: var(--chat-blue);
  background: var(--accent-soft);
}
.chat-list {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 0 6px 8px;
  overflow-y: auto;
  list-style: none;
  scrollbar-width: thin;
}
.chat-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 8px 8px;
  border-radius: 6px;
  transition: background 0.12s;
}
.chat-item:hover {
  background: var(--bg-muted);
}
.chat-item.is-active {
  background: var(--accent-soft);
}
.chat-item-avatar {
  position: relative;
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  overflow: hidden;
  font-size: 16px;
  color: var(--chat-text-weak);
  background: var(--bg-muted);
  border-radius: 50%;
}
.chat-item-avatar.is-room {
  border-radius: 8px;
}
.chat-item-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.chat-item-online {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 9px;
  height: 9px;
  background: var(--success);
  border: 2px solid var(--chat-side);
  border-radius: 50%;
}
.chat-item-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}
.chat-item-top {
  display: flex;
  align-items: baseline;
  gap: 5px;
}
.chat-item-top strong {
  flex: 1;
  min-width: 0;
  font-size: 13.5px;
  font-weight: 500;
  color: var(--text);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.chat-item-top time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--chat-text-weak);
}
.chat-item-tag {
  flex-shrink: 0;
  padding: 0 4px;
  font-size: 10px;
  font-style: normal;
  color: var(--chat-blue);
  background: var(--accent-soft);
  border-radius: 3px;
}
.chat-item-flag {
  flex-shrink: 0;
  font-size: 10px;
  color: var(--chat-text-weak);
}
.chat-item-bottom {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 2px;
}
.chat-item-desc {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--chat-text-weak);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
/* 未读数：跟在摘要后面、贴最右（QQ NT 的位置）
 *
 * 早前是 absolute 钉在右上角，正好压在「最后一条时间」上，
 * 两个数字叠在一起；现在它是第二行的普通子项，与时间各占一行，
 * 摘要过长时由 .chat-item-desc 省略，不会把角标挤出去。 */
.chat-item-unread {
  flex-shrink: 0;
  min-width: 15px;
  padding: 0 4px;
  font-size: 10px;
  font-weight: 600;
  line-height: 15px;
  color: #fff;
  text-align: center;
  background: var(--danger);
  border-radius: 7.5px;
}
/* 免打扰会话：角标转灰，与红色的「需要处理」区分开（同 QQ） */
.chat-item-unread.is-muted {
  color: var(--bg-card);
  background: var(--text-light);
}
.chat-item-muted {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--chat-text-weak);
}

/* ==================== 联系人面板（互关好友 / 群聊） ==================== */
/* 分组标题：仿 QQ 的「我的好友 / 群聊」，右侧带「在线数/总数」 */
.chat-group-head {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 8px 4px;
  font-size: 11px;
  color: var(--chat-text-weak);
}
.chat-group-head strong {
  font-size: 11.5px;
  font-weight: 600;
  color: var(--text-soft);
}
.chat-group-head em {
  margin-left: auto;
  font-style: normal;
  font-size: 10.5px;
}
.chat-item-status {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--chat-text-weak);
}
.chat-item-status.is-online {
  color: var(--success);
}
/* 分组内的空态：比整列表空态矮一些，避免两段空态把侧栏撑满 */
.chat-empty.is-inline {
  padding: 14px 10px 18px;
}
.chat-empty.is-inline p {
  font-size: 12px;
}
.chat-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 40px 12px;
  font-size: 12px;
  color: var(--chat-text-weak);
  text-align: center;
}
.chat-empty i {
  font-size: 26px;
  opacity: 0.6;
}
.chat-empty p {
  margin: 0;
  font-size: 13px;
}

/* ==================== 空态 ==================== */
.chat-blank {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  min-width: 0;
  color: var(--chat-text-weak);
  background: var(--chat-panel);
}
.chat-blank i {
  font-size: 34px;
  opacity: 0.5;
}
.chat-blank p {
  margin: 0;
  font-size: 13px;
}
.chat-blank-btn {
  padding: 6px 14px;
  font-size: 12.5px;
  color: #fff;
  background: var(--chat-blue);
  border-radius: 6px;
}

/* ==================== 提醒设置 ==================== */
.chat-alert-mask {
  position: fixed;
  inset: 0;
  z-index: 20;
}
.chat-alert-panel {
  position: fixed;
  top: 180px;
  left: 62px;
  z-index: 21;
  width: 260px;
  padding: 14px;
  background: var(--bg-card);
  border: 1px solid var(--chat-line);
  border-radius: 8px;
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.14);
}
.chat-alert-title {
  margin: 0 0 8px;
  font-size: 13px;
  font-weight: 600;
}
.chat-alert-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 0;
}
.chat-alert-copy {
  flex: 1;
  min-width: 0;
}
.chat-alert-copy strong {
  display: block;
  font-size: 12.5px;
  font-weight: 500;
}
.chat-alert-copy em {
  display: block;
  margin-top: 2px;
  font-size: 11px;
  font-style: normal;
  line-height: 1.5;
  color: var(--chat-text-weak);
}
.chat-switch {
  position: relative;
  width: 34px;
  height: 19px;
  flex-shrink: 0;
  background: var(--border);
  border-radius: 10px;
  transition: background 0.15s;
}
.chat-switch.is-on {
  background: var(--chat-blue);
}
.chat-switch-knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 15px;
  height: 15px;
  background: var(--bg-card);
  border-radius: 50%;
  transition: transform 0.15s;
}
.chat-switch.is-on .chat-switch-knob {
  transform: translateX(15px);
}
.chat-switch:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.chat-alert-tip {
  margin: 8px 0 0;
  font-size: 11px;
  line-height: 1.5;
  color: var(--chat-text-weak);
}

/* ==================== 新建群聊 ==================== */
.chat-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  font-size: 13px;
  color: var(--chat-text-weak);
  border-radius: 6px;
}
.chat-icon-btn:hover {
  color: var(--text);
  background: var(--chat-panel);
}
.chat-modal {
  position: fixed;
  inset: 0;
  z-index: 3200;
  display: grid;
  place-items: center;
}
.chat-modal-mask {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.32);
}
.chat-modal-card {
  position: relative;
  width: min(420px, calc(100vw - 32px));
  padding: 16px 18px 18px;
  background: var(--bg-card);
  border-radius: 10px;
  box-shadow: 0 18px 50px rgba(0, 0, 0, 0.2);
}
.chat-modal-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
}
.chat-field {
  display: block;
  margin-bottom: 12px;
}
.chat-field > span {
  display: block;
  margin-bottom: 5px;
  font-size: 12px;
  color: var(--chat-text-weak);
}
.chat-field input,
.chat-field textarea {
  width: 100%;
  padding: 8px 10px;
  font-family: inherit;
  font-size: 13px;
  color: var(--text);
  background: var(--bg-muted);
  border: 1px solid var(--chat-line);
  border-radius: 6px;
  outline: none;
  resize: vertical;
}
.chat-field input:focus,
.chat-field textarea:focus {
  border-color: var(--chat-blue);
}
.chat-tip {
  margin: 0 0 12px;
  font-size: 11px;
  color: var(--chat-text-weak);
}
.chat-limit {
  display: flex;
  gap: 8px;
  padding: 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--chat-text-weak);
  background: var(--bg-muted);
  border-radius: 6px;
}
.chat-modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.chat-btn {
  padding: 7px 16px;
  font-size: 12.5px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 6px;
}
.chat-btn-primary {
  color: #fff;
  background: var(--chat-blue);
}
.chat-btn-primary:hover {
  background: var(--chat-blue-deep);
}
.chat-btn:disabled {
  opacity: 0.6;
}

/* ==================== 响应式 ==================== */
/* 面板切换只在窄屏出现（宽屏用最左功能栏，信息密度更接近 QQ NT） */
.chat-panel-tabs {
  display: none;
  gap: 4px;
  padding: 8px 8px 0;
}
.chat-panel-tab {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  gap: 5px;
  height: 30px;
  font-size: 12.5px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 6px;
  transition: background 0.12s, color 0.12s;
}
.chat-panel-tab.is-active {
  color: var(--chat-blue);
  background: var(--accent-soft);
}
/* 返回站点：宽度固定，不跟两个 tab 抢空间 */
.chat-panel-home {
  display: flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 30px;
  font-size: 14px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 6px;
  transition: background 0.12s, color 0.12s;
}
.chat-panel-home:hover,
.chat-panel-home:active {
  color: var(--chat-blue);
  background: var(--accent-soft);
}

@media (max-width: 860px) {
  .chat-rail {
    display: none;
  }
  .chat-panel-tabs {
    display: flex;
  }
  .chat-side {
    width: 100%;
    border-right: 0;
  }
  .chat-app.is-active .chat-side {
    display: none;
  }
  .chat-app:not(.is-active) .chat-blank {
    display: none;
  }
  .chat-alert-panel {
    top: auto;
    right: 12px;
    bottom: 12px;
    left: 12px;
    width: auto;
  }
}
</style>
