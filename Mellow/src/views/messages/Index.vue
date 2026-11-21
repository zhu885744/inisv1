<template>
  <div class="msg-app is-embedded" :class="{ 'is-chat': !!activeUid }">
    <!-- ==================== 会话列表 ==================== -->
    <!-- 私信与群聊的列表已上移到 /chat 的统一左栏（views/chat/Layout.vue），
         这里只为「单独复用本组件」保留结构；在 /chat 路由下 embedded=true，整块不渲染 -->
    <aside v-if="!embedded" class="msg-sidebar">
      <header class="msg-sidebar-head">
        <div class="msg-sidebar-title">
          <span>私信</span>
        </div>
        <div class="msg-sidebar-actions">
          <!-- 提醒设置：提示音 / 右下角悬浮通知，偏好只存本机 -->
          <div class="msg-alert-wrap">
            <button
              type="button"
              class="msg-icon-btn"
              :title="alertOn ? '消息提醒已开启' : '消息提醒已关闭'"
              @click="alertOpen = !alertOpen"
            >
              <i :class="alertOn ? 'bi bi-bell' : 'bi bi-bell-slash'" aria-hidden="true" />
            </button>

            <div v-if="alertOpen" class="msg-alert-mask" @click="alertOpen = false" />
            <div v-if="alertOpen" class="msg-alert-panel">
              <p class="msg-alert-title">消息提醒</p>

              <div class="msg-alert-row">
                <span class="msg-alert-copy">
                  <strong>提示音</strong>
                  <em>页面在前台时收到私信响一声</em>
                </span>
                <button
                  type="button"
                  class="msg-switch"
                  :class="{ 'is-on': chatStore.alert.sound }"
                  role="switch"
                  :aria-checked="chatStore.alert.sound"
                  @click="chatStore.setAlert('sound', !chatStore.alert.sound)"
                >
                  <span class="msg-switch-knob" />
                </button>
              </div>

              <div class="msg-alert-row">
                <span class="msg-alert-copy">
                  <strong>右下角悬浮通知</strong>
                  <em>其他页面收到私信时弹出提示卡</em>
                </span>
                <button
                  type="button"
                  class="msg-switch"
                  :class="{ 'is-on': chatStore.alert.toast }"
                  role="switch"
                  :aria-checked="chatStore.alert.toast"
                  @click="chatStore.setAlert('toast', !chatStore.alert.toast)"
                >
                  <span class="msg-switch-knob" />
                </button>
              </div>

              <div class="msg-alert-row">
                <span class="msg-alert-copy">
                  <strong>桌面通知</strong>
                  <em>{{ desktopHint }}</em>
                </span>
                <button
                  type="button"
                  class="msg-switch"
                  :class="{ 'is-on': desktopEnabled }"
                  role="switch"
                  :aria-checked="desktopEnabled"
                  :disabled="!desktopSupported"
                  @click="toggleDesktop"
                >
                  <span class="msg-switch-knob" />
                </button>
              </div>

              <p class="msg-alert-tip">设置只保存在本机浏览器，不影响未读角标</p>
            </div>
          </div>

          <button type="button" class="msg-icon-btn" title="刷新列表" @click="refreshList">
            <i class="bi bi-arrow-clockwise" :class="{ 'is-spin': loadingList }" aria-hidden="true" />
          </button>
          <RouterLink to="/" class="msg-icon-btn" title="返回站点">
            <i class="bi bi-box-arrow-up-left" aria-hidden="true" />
          </RouterLink>
        </div>
      </header>

      <label class="msg-search">
        <i class="bi bi-search" aria-hidden="true" />
        <FormInput v-model="keyword" type="search" placeholder="搜索昵称" aria-label="搜索会话" />
        <button v-if="keyword" type="button" class="msg-search-clear" title="清空" @click="keyword = ''">
          <i class="bi bi-x-circle-fill" aria-hidden="true" />
        </button>
      </label>

      <ul class="msg-conversations">
        <!-- 群聊入口：公共大厅 + 用户自建的群，不参与搜索过滤，始终置顶 -->
        <li class="msg-room-entry">
          <RouterLink to="/room" class="msg-room-card">
            <span class="msg-room-avatar">
              <i class="bi bi-people-fill" aria-hidden="true" />
            </span>
            <span class="msg-room-body">
              <span class="msg-room-name">群聊</span>
              <span class="msg-room-desc">公共大厅与大家建的群</span>
            </span>
            <!-- 有人在群聊 @ 我：进房间前先在入口上标出来，否则这条提醒就飘走了 -->
            <span v-if="chatStore.roomMentions" class="msg-room-badge">
              {{ chatStore.roomMentions > 99 ? '99+' : chatStore.roomMentions }}
            </span>
            <i class="bi bi-chevron-right msg-room-arrow" aria-hidden="true" />
          </RouterLink>
        </li>

        <li v-for="item in filteredConversations" :key="item.id">
          <button
            type="button"
            class="msg-conv"
            :class="{ 'is-active': Number(item.other_uid) === Number(activeUid) }"
            @click="openChat(item.other_uid)"
            @contextmenu.prevent="openConvMenu($event, item.other_uid)"
          >
            <span class="msg-conv-avatar">
              <img
                :src="item.user?.avatar || defaultAvatar"
                :alt="item.user?.nickname || '用户头像'"
                loading="lazy"
              />
              <i v-if="item.online" class="msg-presence" title="在线" />
            </span>
            <span class="msg-conv-body">
              <span class="msg-conv-top">
                <i v-if="item.pinned" class="bi bi-pin-angle-fill msg-conv-flag" title="已置顶" aria-hidden="true" />
                <span class="msg-conv-name">{{ item.user?.nickname || `用户 #${item.other_uid}` }}</span>
                <i v-if="item.muted" class="bi bi-bell-slash msg-conv-mute" title="免打扰" aria-hidden="true" />
                <time class="msg-conv-time">{{ shortTime(item.last_time) }}</time>
              </span>
              <span class="msg-conv-bottom">
                <span class="msg-conv-text">{{ item.last_text || '还没有聊过' }}</span>
                <span v-if="item.unread" class="msg-conv-badge">
                  {{ item.unread > 99 ? '99+' : item.unread }}
                </span>
              </span>
            </span>
          </button>
        </li>

        <!-- 互关好友：还没聊过的人也直接列出来，点一下就能开聊 -->
        <li v-for="item in filteredFriends" :key="`friend-${item.uid}`">
          <button
            type="button"
            class="msg-conv"
            :class="{ 'is-active': Number(item.uid) === Number(activeUid) }"
            @click="openChat(item.uid)"
          >
            <span class="msg-conv-avatar">
              <img
                :src="item.user?.avatar || defaultAvatar"
                :alt="item.user?.nickname || '用户头像'"
                loading="lazy"
              />
              <i v-if="item.online" class="msg-presence" title="在线" />
            </span>
            <span class="msg-conv-body">
              <span class="msg-conv-top">
                <span class="msg-conv-name">{{ item.user?.nickname || `用户 #${item.uid}` }}</span>
                <span class="msg-conv-tag">互关</span>
              </span>
              <span class="msg-conv-bottom">
                <span class="msg-conv-text">互关好友，点一下就能开聊</span>
              </span>
            </span>
          </button>
        </li>

        <li v-if="!loadingList && !filteredConversations.length && !filteredFriends.length" class="msg-list-empty">
          <i :class="keyword ? 'bi bi-search' : 'bi bi-chat-square-dots'" aria-hidden="true" />
          <p>{{ keyword ? '没有匹配的会话' : '还没有可以聊天的人' }}</p>
          <span v-if="!keyword">互相关注后，好友会直接出现在这里</span>
        </li>
      </ul>
    </aside>

    <!-- ==================== 聊天窗口 ==================== -->
    <section class="msg-chat">
      <!-- 游客：私信要登录后才能看。这里给明确引导，而不是把人直接甩去登录页 -->
      <div v-if="isGuest" class="msg-placeholder">
        <div class="msg-placeholder-mark"><i class="bi bi-person-lock" aria-hidden="true" /></div>
        <h2>登录后即可查看私信</h2>
        <p>私信只对互相关注的好友开放，未登录也可以先逛公共群聊</p>
        <div class="msg-guest-actions">
          <button type="button" class="msg-guest-btn is-primary" @click="requireLogin()">去登录</button>
          <RouterLink to="/chat/room/1" class="msg-guest-btn">进入公共大厅</RouterLink>
        </div>
      </div>

      <template v-else-if="activeUid">
        <header class="msg-chat-head">
          <button type="button" class="msg-icon-btn msg-back" title="返回会话列表" @click="closeChat">
            <i class="bi bi-chevron-left" aria-hidden="true" />
          </button>

          <RouterLink v-if="target?.id" :to="`/author/${target.id}`" class="msg-peer" title="查看 TA 的主页">
            <img class="msg-peer-avatar" :src="target.avatar || defaultAvatar" alt="" loading="lazy" />
            <span class="msg-peer-name">{{ target.nickname || `用户 #${activeUid}` }}</span>
          </RouterLink>
          <span v-else class="msg-peer-name">用户 #{{ activeUid }}</span>

          <span class="msg-peer-status" :class="{ 'is-online': online || peerTyping }">
            <i aria-hidden="true" />{{ peerTyping ? '正在输入…' : online ? '在线' : '离线' }}
          </span>

          <div class="msg-chat-actions">
            <button
              type="button"
              class="msg-icon-btn"
              title="拍一拍"
              :disabled="!canSend"
              @click="sendPat"
            >
              <i class="bi bi-hand-index-thumb" aria-hidden="true" />
            </button>

            <!-- 会话设置：置顶 / 免打扰（只影响自己） -->
            <button
              type="button"
              class="msg-icon-btn"
              title="会话设置"
              @click="openConvMenuFromHeader($event)"
            >
              <i class="bi bi-three-dots" aria-hidden="true" />
            </button>
          </div>
        </header>

        <div ref="bodyRef" class="msg-body" :class="{ 'is-patted': patting }">
          <div v-if="loadingHistory && !messages.length" class="msg-body-tip">加载中…</div>
          <button
            v-else-if="hasMore"
            type="button"
            class="msg-more"
            :disabled="loadingHistory"
            @click="loadMessages(messages[0]?.id)"
          >
            <i class="bi bi-clock-history" aria-hidden="true" />查看更早的消息
          </button>

          <template v-for="row in rows" :key="row.key">
            <div v-if="row.type === 'day'" class="msg-day"><span>{{ row.label }}</span></div>

            <div v-else-if="row.type === 'system'" class="msg-system">{{ systemText(row.msg) }}</div>

            <div
              v-else
              class="msg-row"
              :class="{ 'is-mine': isMine(row.msg) }"
              @contextmenu.prevent="openMessageMenu($event, row.msg)"
            >
              <img
                v-if="!isMine(row.msg)"
                class="msg-row-avatar"
                :src="target?.avatar || defaultAvatar"
                alt=""
                title="双击拍一拍 TA"
                loading="lazy"
                @dblclick="sendPat"
              />
              <div class="msg-row-main">
                <div class="msg-bubble">
                  <!-- 引用回复：展示被引用消息的快照（原消息撤回后依然可读） -->
                  <div v-if="row.msg.quote" class="msg-quote" :title="quoteTitle(row.msg.quote)">
                    <span class="msg-quote-name">{{ quoteName(row.msg.quote) }}</span>
                    <span
                      class="msg-quote-text"
                      v-html="renderEmojiWithBreaks(quotePreview(row.msg.quote), { size: 16 })"
                    />
                  </div>
                  <img
                    v-if="row.msg.msg_type === 'image'"
                    class="msg-bubble-image"
                    :src="fullUrl(row.msg.content)"
                    alt="图片消息"
                    loading="lazy"
                    @click="openLightbox([fullUrl(row.msg.content)])"
                  />
                  <!-- 文本消息：表情是 [emoji:url] 标记，必须走 renderEmojiWithBreaks（先转义再替换） -->
                  <span
                    v-else
                    class="msg-text"
                    v-html="renderEmojiWithBreaks(row.msg.content, { size: 22 })"
                  />
                </div>
                <div class="msg-meta">
                  <time>{{ shortTime(row.msg.create_time) }}</time>
                  <button
                    v-if="canRevoke(row.msg)"
                    type="button"
                    class="msg-revoke"
                    @click="revokeMessage(row.msg)"
                  >撤回</button>
                  <!-- 已读状态只标在「我发出的最后一条」上，避免每行都挂一个 -->
                  <span
                    v-if="showReadState(row.msg)"
                    class="msg-read"
                    :class="{ 'is-read': row.msg.is_read }"
                  >{{ row.msg.is_read ? '已读' : '未读' }}</span>
                </div>
              </div>
            </div>
          </template>

          <p v-if="!loadingHistory && !messages.length" class="msg-body-empty">
            还没有聊天记录，打个招呼吧
          </p>
        </div>

        <footer class="msg-composer">
          <p v-if="!canSend" class="msg-blocked">
            <i class="bi bi-slash-circle" aria-hidden="true" />
            {{ blockedReason || '暂时无法发送私信' }}
          </p>

          <form v-else class="msg-composer-box" @submit.prevent="send">
            <!-- 引用态：显示被引用消息的摘要，发送后自动清除 -->
            <div v-if="quoteDraft" class="msg-quote-draft">
              <div class="msg-quote-draft-body">
                <span class="msg-quote-name">{{ quoteName(quoteDraft) }}</span>
                <span
                  class="msg-quote-text"
                  v-html="renderEmojiWithBreaks(quotePreview(quoteDraft), { size: 16 })"
                />
              </div>
              <button type="button" class="msg-icon-btn" title="取消引用" @click="clearQuote">
                <i class="bi bi-x-lg" aria-hidden="true" />
              </button>
            </div>

            <div class="msg-tools">
              <!-- 表情：复用评论/动态同一个表情面板，插入的是 [emoji:url] 标记 -->
              <div class="msg-emoji-wrap">
                <button
                  type="button"
                  class="msg-icon-btn"
                  title="插入表情"
                  @click="emojiOpen = !emojiOpen"
                >
                  <i class="bi bi-emoji-smile" aria-hidden="true" />
                </button>
                <EmojiPicker v-model="emojiOpen" @select="insertEmoji" />
              </div>

              <!-- 图片：直接走附件上传（与评论图片同一条链路，含 WebP 压缩） -->
              <button
                type="button"
                class="msg-icon-btn"
                :title="uploading ? '图片上传中…' : '发送图片'"
                :disabled="uploading"
                @click="pickImage"
              >
                <i :class="uploading ? 'bi bi-arrow-repeat is-spin' : 'bi bi-image'" aria-hidden="true" />
              </button>
              <input
                ref="fileRef"
                class="msg-file"
                type="file"
                accept="image/*"
                multiple
                @change="onFilesPicked"
              />
            </div>

            <!-- contenteditable：表情在输入框里直接显示成图片，存储仍是 [emoji:url] 纯文本 -->
            <ChatEmojiInput
              ref="inputRef"
              v-model="draft"
              class="msg-input"
              :maxlength="2000"
              placeholder="发消息…"
              @input="onDraftInput"
              @send="send"
              @files="uploadAndSendImage"
            />

            <button
              type="submit"
              class="msg-send"
              title="发送（Enter）"
              :disabled="sending || !draft.trim()"
            >
              <i v-if="sending" class="bi bi-arrow-repeat is-spin" aria-hidden="true" />
              <template v-else>发送</template>
            </button>
          </form>
        </footer>
      </template>

      <div v-else class="msg-placeholder">
        <div class="msg-placeholder-mark"><i class="bi bi-chat-dots" aria-hidden="true" /></div>
        <h2>选择一段对话</h2>
        <p>从左侧挑一个会话或互关好友，直接开聊</p>
        <RouterLink to="/" class="msg-placeholder-link">
          返回站点<i class="bi bi-arrow-right-short" aria-hidden="true" />
        </RouterLink>
      </div>
    </section>

    <!-- ==================== 会话设置菜单（置顶 / 免打扰） ==================== -->
    <div
      v-if="convMenu.open"
      class="msg-menu-mask"
      @click="closeConvMenu"
      @contextmenu.prevent="closeConvMenu"
    />
    <div
      v-if="convMenu.open"
      class="msg-menu"
      role="menu"
      :style="{ left: `${convMenu.x}px`, top: `${convMenu.y}px` }"
    >
      <button type="button" class="msg-menu-item" role="menuitem" @click="toggleConvPref('pinned')">
        <i :class="convMenuRow?.pinned ? 'bi bi-pin-angle' : 'bi bi-pin-angle-fill'" aria-hidden="true" />
        {{ convMenuRow?.pinned ? '取消置顶' : '置顶会话' }}
      </button>
      <button type="button" class="msg-menu-item" role="menuitem" @click="toggleConvPref('muted')">
        <i :class="convMenuRow?.muted ? 'bi bi-bell' : 'bi bi-bell-slash'" aria-hidden="true" />
        {{ convMenuRow?.muted ? '开启提醒' : '消息免打扰' }}
      </button>
    </div>

    <!-- ==================== 消息右键菜单 ==================== -->
    <div
      v-if="messageMenu.open"
      class="msg-menu-mask"
      @click="closeMessageMenu"
      @contextmenu.prevent="closeMessageMenu"
    />
    <div
      v-if="messageMenu.open"
      class="msg-menu"
      role="menu"
      :style="{ left: `${messageMenu.x}px`, top: `${messageMenu.y}px` }"
    >
      <button type="button" class="msg-menu-item" role="menuitem" @click="quoteMessage(messageMenu.msg)">
        <i class="bi bi-quote" aria-hidden="true" />引用消息
      </button>
      <button
        v-if="canRevoke(messageMenu.msg)"
        type="button"
        class="msg-menu-item"
        role="menuitem"
        @click="revokeFromMenu(messageMenu.msg)"
      >
        <i class="bi bi-arrow-counterclockwise" aria-hidden="true" />撤回消息
      </button>
      <button
        type="button"
        class="msg-menu-item is-danger"
        role="menuitem"
        @click="deleteFromMenu(messageMenu.msg)"
      >
        <i class="bi bi-trash3" aria-hidden="true" />删除消息
      </button>
      <button type="button" class="msg-menu-item" role="menuitem" @click="copyMessage(messageMenu.msg)">
        <i class="bi bi-clipboard" aria-hidden="true" />复制
      </button>
    </div>
  </div>
</template>

<script setup>
import FormInput from '@/components/form/FormInput.vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  deleteChatMessage,
  getChatStatus,
  listChatConversations,
  listChatFriends,
  listChatMessages,
  patChatUser,
  readChatMessages,
  revokeChatMessage,
  sendChatMessage,
  setChatPref,
  syncChatMessages
} from '@/api/chat'
import { useChatStore } from '@/stores/chat'
import { useUserStore } from '@/stores/user'
import { toast } from '@/utils/toast'
import { fromNow, formatTime } from '@/utils/time'
import { openLightbox } from '@/utils/lightbox'
import { getFullUrl, renderEmojiWithBreaks } from '@/utils/emoji'
import { uploadAttachments } from '@/api/attachment'
import { useDesktopNotice } from '@/composables/useDesktopNotice'
import EmojiPicker from '@/components/EmojiPicker.vue'
import ChatEmojiInput from '@/components/ChatEmojiInput.vue'

/**
 * 私信（独立页面，路由 /messages）
 *
 * 为什么独立：聊天需要整屏高度与自己的左右分栏，塞进用户中心会被侧边栏 + 顶栏
 * 挤成一条窄缝；这里不带任何布局组件（无站点侧栏 / 页脚 / 手机底栏），
 * 页面自己撑满视口，只在左上留一个「返回站点」的出口。
 * 旧地址 /user/messages 在路由里保留为重定向，书签与站内旧链接不会失效。
 *
 * 数据流：
 *   - 会话列表 / 历史记录走 HTTP（chat/conversations、chat/messages）；
 *   - 新消息走 WebSocket（stores/chat.js 订阅的 private 消息）；
 *   - 发送走 HTTP（拿到落库后的 id 直接上屏），服务端会同时推给在线的对方。
 *
 * 窄屏：一次只显示一屏（列表 or 聊天窗），用 activeUid 切换；宽屏左右分栏。
 */
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const chatStore = useChatStore()

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const conversations = ref([])
// 互关好友（私信只对互关好友开放）：补上「还没聊过」的人，点一下就能开聊
const friends = ref([])
const loadingList = ref(false)
const keyword = ref('')

// 在 /chat 的统一布局里只渲染聊天区，左栏由父级承担
const embedded = true

// 游客：/chat/:uid 不再强制登录，页面里给一条引导（去登录 / 先进公共大厅）
const isGuest = computed(() => !userStore.isLogged)

/** 游客点了「去登录」：带上回跳地址，登录后还能回到这段会话 */
function requireLogin() {
  if (userStore.isLogged) return
  router.push({ name: 'login', query: { redirect: route.fullPath } })
}

// 当前会话由路由参数决定：/chat/:uid（刷新、分享、前进后退都能落到同一段对话）
// 这里保持 0，初始化与路由变化都走 openChat()，避免出现「有 uid 但没加载消息」
const activeUid = ref(0)
const target = ref(null)
const messages = ref([])
const hasMore = ref(false)
const loadingHistory = ref(false)
const online = ref(false)
const canSend = ref(true)
// 不能发送的原因（拉黑 / 未互关），由服务端返回，直接显示在输入区上方
const blockedReason = ref('')
// 当前引用中的消息（null = 不在引用态）
const quoteDraft = ref(null)

const draft = ref('')
const sending = ref(false)
const uploading = ref(false)
const bodyRef = ref(null)
const inputRef = ref(null)
const fileRef = ref(null)
const emojiOpen = ref(false)
// 提醒设置面板（提示音 / 悬浮通知 / 桌面通知），偏好由 chat store 统一管理并持久化到本机
const alertOpen = ref(false)
const alertOn = computed(
  () => chatStore.alert.sound || chatStore.alert.toast || chatStore.alert.desktop
)

// 桌面通知需要浏览器授权：申请必须发生在点击里，统一交给 composable
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
// 被拍一拍的抖动动画标记（600ms 后自动复位）
const patting = ref(false)

// 消息右键菜单：open / 坐标 / 目标消息
const messageMenu = ref({ open: false, x: 0, y: 0, msg: null })

// 会话设置菜单（置顶 / 免打扰）：open / 坐标 / 目标会话的对方 uid
const convMenu = ref({ open: false, x: 0, y: 0, uid: 0 })
const convMenuRow = computed(
  () => conversations.value.find((item) => Number(item.other_uid) === Number(convMenu.value.uid)) || null
)

// 对方正在输入（收到 chat_typing 后 4 秒自动消失，不需要服务端再发「停止」）
const peerTyping = ref(false)
let typingTimer = null

const myUid = computed(() => Number(userStore.user?.id || 0))

/** 附件 / 表情地址可能是站内相对路径，展示前统一补全域名 */
function fullUrl(url) {
  return getFullUrl(url || '')
}

const filteredConversations = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return conversations.value
  return conversations.value.filter((item) => {
    const name = String(item.user?.nickname || '').toLowerCase()
    return name.includes(kw) || String(item.other_uid).includes(kw)
  })
})

// 还没聊过的互关好友：已经在会话列表里的不再重复显示（会话行有摘要 / 未读，信息更全）
const friendOnlyRows = computed(() => {
  const chatted = new Set(conversations.value.map((item) => Number(item.other_uid)))
  return friends.value.filter((item) => !chatted.has(Number(item.uid)))
})

const filteredFriends = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) return friendOnlyRows.value
  return friendOnlyRows.value.filter((item) => {
    const name = String(item.user?.nickname || '').toLowerCase()
    return name.includes(kw) || String(item.uid).includes(kw)
  })
})

function isMine(msg) {
  return Number(msg.from_uid) === myUid.value
}

// 我发出的最后一条消息（已读状态只挂在这一条上，避免整屏都是「已读」）
const lastMineId = computed(() => {
  for (let i = messages.value.length - 1; i >= 0; i -= 1) {
    const item = messages.value[i]
    if (isMine(item) && !item.is_revoked && item.msg_type !== 'pat') return item.id
  }
  return 0
})

function showReadState(msg) {
  return Number(msg.id) === Number(lastMineId.value)
}

// 与后端 chatRevokeWindow 保持一致（2 分钟）：这里只决定「撤回」按钮要不要露出来，
// 真正的时间校验在服务端，改了客户端也绕不过去
const REVOKE_WINDOW = 120

function canRevoke(msg) {
  if (!msg || !isMine(msg) || msg.is_revoked || msg.msg_type === 'pat') return false
  return Date.now() / 1000 - Number(msg.create_time || 0) <= REVOKE_WINDOW
}

// ---------- 引用回复 ----------

/**
 * 被引用消息的展示名：自己 / 对方
 *
 * 兼容两种入参：消息里的 quote 快照（uid）与引用态里的整条消息（from_uid）。
 */
function quoteName(quote) {
  if (!quote) return ''
  const uid = Number(quote.uid ?? quote.from_uid ?? 0)
  return uid === myUid.value
    ? '我'
    : target.value?.nickname || `用户 #${activeUid.value}`
}

/** 被引用消息的摘要：服务端已落快照，这里只做类型兜底 */
function quotePreview(quote) {
  if (!quote) return ''
  if (quote.msg_type === 'image') return '[图片]'
  if (quote.msg_type === 'pat') return '[拍一拍]'
  return String(quote.content || '')
}

function quoteTitle(quote) {
  return quotePreview(quote).replace(/\[emoji:[^\]]+\]/g, '[表情]')
}

function clearQuote() {
  quoteDraft.value = null
}

// ---------- 消息右键菜单 ----------

/** 打开右键菜单：坐标按视口夹紧，避免贴边时菜单跑出屏幕 */
function openMessageMenu(event, msg) {
  if (!msg?.id) return

  const width = 148
  const height = canRevoke(msg) ? 172 : 138
  const x = Math.min(Number(event.clientX) || 0, window.innerWidth - width - 8)
  const y = Math.min(Number(event.clientY) || 0, window.innerHeight - height - 8)

  messageMenu.value = { open: true, x: Math.max(8, x), y: Math.max(8, y), msg }
}

function closeMessageMenu() {
  if (!messageMenu.value.open) return
  messageMenu.value = { ...messageMenu.value, open: false, msg: null }
}

function quoteMessage(msg) {
  closeMessageMenu()
  if (!msg?.id) return
  quoteDraft.value = msg
  nextTick(() => inputRef.value?.focus())
}

function revokeFromMenu(msg) {
  closeMessageMenu()
  revokeMessage(msg)
}

async function deleteFromMenu(msg) {
  closeMessageMenu()
  if (!msg?.id) return

  try {
    const res = await deleteChatMessage(msg.id)
    if (res?.code !== 200) return

    // 只删自己这一侧：本地直接移除，未读数由服务端下次校准时对齐
    messages.value = messages.value.filter((item) => Number(item.id) !== Number(msg.id))
    // 正在引用的就是这条：顺手取消引用，避免发出一个指向已删消息的引用
    if (Number(quoteDraft.value?.id) === Number(msg.id)) quoteDraft.value = null
    toast.success('已删除')
  } catch {
    /* 业务错误已由 request.js 统一 toast */
  }
}

/** 复制：文本去掉 [emoji:url] 标记（否则复制出来是一串链接），图片复制地址 */
async function copyMessage(msg) {
  closeMessageMenu()
  if (!msg) return

  let text = ''
  if (msg.is_revoked || msg.msg_type === 'pat') {
    text = systemText(msg)
  } else if (msg.msg_type === 'image') {
    text = fullUrl(msg.content)
  } else {
    text = String(msg.content || '').replace(/\[emoji:[^\]]+\]/g, '[表情]')
  }

  if (!text) return

  try {
    await navigator.clipboard.writeText(text)
    toast.success('已复制')
  } catch {
    // 非 HTTPS / 无剪贴板权限：退回 textarea + execCommand
    const el = document.createElement('textarea')
    el.value = text
    el.setAttribute('readonly', '')
    el.style.position = 'fixed'
    el.style.opacity = '0'
    document.body.appendChild(el)
    el.select()
    try {
      document.execCommand('copy')
      toast.success('已复制')
    } catch {
      toast.warning('复制失败，请手动选择文字')
    }
    document.body.removeChild(el)
  }
}

// 系统提示（撤回 / 拍一拍）：不渲染气泡，居中灰字
function systemText(msg) {
  if (msg.is_revoked) {
    return isMine(msg) ? '你撤回了一条消息' : '对方撤回了一条消息'
  }
  if (msg.msg_type === 'pat') {
    const name = target.value?.nickname || `用户 #${activeUid.value}`
    const suffix = msg.content ? ` ${msg.content}` : ''
    return isMine(msg) ? `你拍了拍 ${name}${suffix}` : `${name} 拍了拍你${suffix}`
  }
  return ''
}

/**
 * 消息按天分段
 *
 * 聊天记录一多就分不清「这是今天还是上周」，这里在跨天处插一条日期分隔，
 * 顺带把撤回 / 拍一拍这类系统提示单独列出来（它们不占气泡）。
 */
const rows = computed(() => {
  const list = []
  let lastDay = ''

  messages.value.forEach((msg) => {
    const day = dayKey(msg.create_time)
    if (day && day !== lastDay) {
      list.push({ type: 'day', key: `day-${day}-${msg.id}`, label: dayLabel(msg.create_time) })
      lastDay = day
    }
    list.push({
      type: msg.is_revoked || msg.msg_type === 'pat' ? 'system' : 'msg',
      key: `msg-${msg.id}`,
      msg
    })
  })

  return list
})

// 会话列表用相对时间（刚刚 / 3 分钟前），气泡上显示到分钟，更好定位上下文
function shortTime(ts) {
  if (!ts) return ''
  const diff = Date.now() / 1000 - Number(ts)
  if (diff < 60) return '刚刚'
  if (diff < 86400) return fromNow(ts)
  return formatTime(ts).slice(5, 16)
}

/** 日期分段键（同一天的消息归到一组） */
function dayKey(ts) {
  const value = Number(ts) || 0
  if (value <= 0) return ''
  const date = new Date(value * 1000)
  return `${date.getFullYear()}-${date.getMonth() + 1}-${date.getDate()}`
}

/** 日期分隔上的文案：今天 / 昨天 / 8 月 12 日 / 2025 年 8 月 12 日 */
function dayLabel(ts) {
  const value = Number(ts) || 0
  if (value <= 0) return ''

  const date = new Date(value * 1000)
  const now = new Date()
  const startOf = (item) => new Date(item.getFullYear(), item.getMonth(), item.getDate()).getTime()
  const diffDays = Math.round((startOf(now) - startOf(date)) / 86400000)

  if (diffDays === 0) return '今天'
  if (diffDays === 1) return '昨天'
  if (date.getFullYear() === now.getFullYear()) {
    return `${date.getMonth() + 1} 月 ${date.getDate()} 日`
  }
  return `${date.getFullYear()} 年 ${date.getMonth() + 1} 月 ${date.getDate()} 日`
}

async function loadConversations() {
  if (!userStore.isLogged) return
  loadingList.value = true
  try {
    const res = await listChatConversations()
    if (res?.code === 200) {
      // 正在看的会话一律按 0 显示：标已读请求可能还在路上，
      // 直接用服务端旧值会把刚清掉的数字又贴回列表上
      conversations.value = (res.data?.data || []).map((item) =>
        Number(item.other_uid) === Number(activeUid.value) ? { ...item, unread: 0 } : item
      )
      // 免打扰名单要跟着列表刷新：换设备 / 多端修改后，本机的提醒策略才对得上
      syncMutedPeers()
    }
  } catch {
    // 错误提示已由 request.js 统一 toast，这里只保证界面不卡在 loading
  } finally {
    loadingList.value = false
  }
}

/**
 * 互关好友（私信只对互关好友开放，见后端 model.ChatPermission）
 *
 * 会话列表只包含「聊过的人」，所以这里单独拉一份互关好友：
 * 已经在会话里的由会话行展示（有摘要、未读、时间），剩下的作为新入口列出来。
 */
async function loadFriends() {
  if (!userStore.isLogged) return
  try {
    const res = await listChatFriends()
    if (res?.code === 200) friends.value = res.data?.data || []
  } catch {
    // 好友列表失败不影响会话列表（只是少了「还没聊过」的入口），静默降级
  }
}

/** 手动刷新：会话列表 + 互关好友一起更新 */
async function refreshList() {
  await Promise.all([loadConversations(), loadFriends()])
}

// ---------- 会话设置（置顶 / 免打扰） ----------

/** 会话排序：置顶优先，其余按最后消息时间倒序（与服务端一致，改完不必重拉整个列表） */
function sortConversations() {
  conversations.value = [...conversations.value].sort((a, b) => {
    if (Boolean(a.pinned) !== Boolean(b.pinned)) return a.pinned ? -1 : 1
    return Number(b.last_time || 0) - Number(a.last_time || 0)
  })
}

/** 把免打扰名单同步给 chat store：响铃 / 悬浮通知 / 桌面通知据此抑制 */
function syncMutedPeers() {
  chatStore.setMutedPeers(
    conversations.value.filter((item) => item.muted).map((item) => item.other_uid)
  )
}

function openConvMenu(event, uid) {
  const targetUid = Number(uid) || 0
  if (targetUid <= 0) return

  const width = 180
  const height = 96
  const x = Math.min(Number(event.clientX) || 0, window.innerWidth - width - 8)
  const y = Math.min(Number(event.clientY) || 0, window.innerHeight - height - 8)

  convMenu.value = { open: true, x: Math.max(8, x), y: Math.max(8, y), uid: targetUid }
}

/** 顶栏的「⋯」：菜单挂在按钮正下方，比鼠标坐标更可预期 */
function openConvMenuFromHeader(event) {
  const rect = event.currentTarget.getBoundingClientRect()
  openConvMenu({ clientX: rect.right - 16, clientY: rect.bottom + 6 }, activeUid.value)
}

function closeConvMenu() {
  if (!convMenu.value.open) return
  convMenu.value = { ...convMenu.value, open: false }
}

async function toggleConvPref(key) {
  const uid = Number(convMenu.value.uid) || 0
  const row = conversations.value.find((item) => Number(item.other_uid) === uid)
  closeConvMenu()
  if (!row) return

  const next = !row[key]

  try {
    const res = await setChatPref({ target_uid: uid, [key]: next })
    if (res?.code !== 200) return

    // 以服务端返回为准（可能因为并发点击与本地预期不同）
    row[key] = typeof res.data?.[key] === 'boolean' ? res.data[key] : next
    sortConversations()
    syncMutedPeers()

    if (key === 'pinned') {
      toast.success(row[key] ? '已置顶' : '已取消置顶')
    } else {
      toast.success(row[key] ? '已开启免打扰' : '已开启提醒')
    }
  } catch {
    /* 业务错误已由 request.js 提示 */
  }
}

/** 输入时上报「正在输入」（store 内部已按 3 秒节流，服务端再兜一层） */
function onDraftInput() {
  if (!activeUid.value || !canSend.value) return
  chatStore.sendTyping(activeUid.value)
}

function scrollToBottom() {
  nextTick(() => {
    const el = bodyRef.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

/** 一组消息里最大的 seq（seq 即消息 id，作为补拉游标用） */
function maxSeqOf(list) {
  let max = 0
  ;(list || []).forEach((item) => {
    const seq = Number(item.seq || item.id) || 0
    if (seq > max) max = seq
  })
  return max
}

/**
 * 断线重连补拉
 *
 * 断线期间的实时推送是直接丢掉的，靠这里按「本地最大 seq」把缺的消息捞回来：
 * 服务端只返回 seq 更大的那些（id 升序），与在屏消息按 id 去重即可。
 * 断得久可能积压很多，按 has_more 最多翻 5 页；再多就交给刷新页面。
 * 补拉失败不打扰用户 —— 再断一次或刷新页面还会重来。
 */
async function syncMessages(uid) {
  uid = Number(uid) || 0
  if (!uid || loadingHistory.value) return

  try {
    let cursor = Math.max(chatStore.lastSeq(uid), maxSeqOf(messages.value))
    const collected = []

    for (let page = 0; page < 5; page += 1) {
      const res = await syncChatMessages(uid, { after_seq: cursor, limit: 50 })
      if (res?.code !== 200) return

      // 请求期间切了会话：立即停手，避免把 A 的消息贴到 B 的窗口
      if (uid !== activeUid.value) return

      collected.push(...(res.data?.data || []))
      cursor = Number(res.data?.last_seq) || cursor
      if (!res.data?.has_more) break
    }

    if (!collected.length) return

    const seen = new Set(messages.value.map((item) => Number(item.id)))
    const added = collected.filter((item) => !seen.has(Number(item.id)))
    if (!added.length) return

    messages.value = [...messages.value, ...added].sort(
      (a, b) => Number(a.seq || a.id) - Number(b.seq || b.id)
    )
    chatStore.ack(uid, maxSeqOf(messages.value))
    scrollToBottom()
    markRead(uid)
  } catch {
    /* 网络抖动：静默失败 */
  }
}

/**
 * 加载与当前会话对象的历史消息
 * @param {number} [beforeId] 传 0/空表示重新加载最新一页
 */
async function loadMessages(beforeId = 0) {
  const uid = activeUid.value
  if (!uid || loadingHistory.value) return

  loadingHistory.value = true
  try {
    const res = await listChatMessages(uid, beforeId ? { before_id: beforeId } : {})
    if (res?.code !== 200) return

    // 请求期间切了会话：丢弃这次结果，避免把 A 的消息贴到 B 的窗口里
    if (uid !== activeUid.value) return

    const list = res.data?.data || []
    if (beforeId) {
      // 翻历史：与已在屏上的消息按 id 去重（期间 socket 可能刚好推来同一条）
      const seen = new Set(messages.value.map((item) => item.id))
      messages.value = [...list.filter((item) => !seen.has(item.id)), ...messages.value]
    } else {
      messages.value = list
    }
    hasMore.value = !!res.data?.has_more
    target.value = res.data?.target || target.value
    online.value = !!res.data?.online
    canSend.value = res.data?.can_send !== false
    // 不能发的原因（拉黑 / 未互关）由服务端给出，避免前端各写一套文案
    blockedReason.value = canSend.value ? '' : res.data?.can_send_tip || '暂时无法发送私信'

    if (beforeId) {
      // 翻历史时保持当前位置（新内容在顶部，不跳到底部）
      const el = bodyRef.value
      if (el) el.scrollTop = 0
    } else {
      scrollToBottom()
      // 最新一页已经到本地：把送达水位报上去（服务端 ACK 只增不减）
      chatStore.ack(uid, maxSeqOf(list))
      markRead(uid)
    }
  } catch {
    // 业务错误已由 request.js 统一 toast
  } finally {
    loadingHistory.value = false
  }
}

async function openChat(uid) {
  uid = Number(uid) || 0
  if (uid <= 0) return
  if (uid === activeUid.value) return

  // 切走前先把上一段对话的已读落地（防抖窗口里那次不能被丢掉），
  // 再告诉 store「我现在看的是这段对话」——这期间它的新消息不再计入未读角标
  flushRead()
  chatStore.setViewing(uid)
  // 点开即视为已读：数字当场消失，已读请求随后照常上报
  clearRowUnread(uid)

  activeUid.value = uid
  target.value = null
  messages.value = []
  hasMore.value = false
  canSend.value = true
  blockedReason.value = ''
  quoteDraft.value = null
  resetTyping()
  closeMessageMenu()

  // 同步到地址栏：/chat/:uid，刷新 / 分享都能直接落到这段对话
  if (String(route.params.uid || '') !== String(uid)) {
    router.replace(`/chat/${uid}`)
  }

  await loadMessages(0)

  // 会话可能还不存在（从作者主页点「私信」进来）：先问一句能不能发
  try {
    const res = await getChatStatus(uid)
    if (res?.code === 200) {
      canSend.value = !!res.data?.can_send
      blockedReason.value = canSend.value ? '' : res.data?.reason || '暂时无法发送私信'
      online.value = !!res.data?.online
    }
  } catch {
    /* 状态查询失败不阻塞聊天，发送时后端还会再校验一次 */
  }
}

function closeChat() {
  // 关掉会话也算「看过了」：把待上报的已读立刻发出，否则列表/角标会残留一个未读
  flushRead()
  chatStore.setViewing(0)
  resetTyping()
  closeConvMenu()

  activeUid.value = 0
  target.value = null
  messages.value = []
  // 回到统一列表（窄屏下就是左栏；宽屏右栏显示空态）
  //
  // 保留 panel：从「联系人」里点进来的，返回时应当回到联系人面板而不是消息面板。
  // ?to=<uid> 这类一次性的直达参数不保留，避免回去又被弹进这段对话。
  if (String(route.path) !== '/chat') {
    router.replace({ path: '/chat', query: route.query.panel ? { panel: route.query.panel } : {} })
  }
}

let readTimer = null
// 防抖期间等待上报的会话：切走 / 离开页面时要把它立刻落地，否则这次已读就丢了
let pendingReadUid = 0

/**
 * 标记已读（防抖 300ms，避免连续收到多条消息时打一片请求）
 *
 * 数字必须在发请求之前就从本地清掉：用户已经看到这段对话了，
 * 再等「标已读 + 重新拉未读总数」两个往返才消失，观感上就是「数字不消失」。
 */
function markRead(uid) {
  uid = Number(uid) || 0
  if (uid <= 0) return

  clearTimeout(readTimer)
  pendingReadUid = uid
  readTimer = setTimeout(() => {
    readTimer = null
    const target = pendingReadUid
    pendingReadUid = 0
    saveRead(target)
  }, 300)
}

/** 把还没上报的已读立刻发出去（切换会话 / 离开页面时调用，别让防抖把未读吞掉） */
function flushRead() {
  if (!readTimer) return
  clearTimeout(readTimer)
  readTimer = null
  const target = pendingReadUid
  pendingReadUid = 0
  saveRead(target)
}

/** 把某一行的未读抹掉，并同步扣减顶部角标（重复调用无害：第二次待扣已经是 0） */
function clearRowUnread(uid) {
  const row = conversations.value.find((item) => Number(item.other_uid) === Number(uid))
  if (!row || !row.unread) return
  const pending = Number(row.unread) || 0
  row.unread = 0
  chatStore.consumeUnread(pending)
}

// 已读同步失败的提示节流：同一个原因没必要每次进会话都弹一遍
let readFailAt = 0

/** 真正落地已读：先本地清账，再让服务端确认 */
async function saveRead(uid) {
  uid = Number(uid) || 0
  if (uid <= 0) return

  clearRowUnread(uid)

  let ok = false
  let hasCount = false

  try {
    const res = await readChatMessages(uid)
    ok = res?.code === 200

    const next = ok ? res.data?.unread : undefined
    // 标已读接口顺带返回了「标记之后的未读总数」：直接覆盖角标。
    // 比「再发一次 GET /chat/unread」更可靠：少一次往返，也不会被并发请求的旧结果盖回来。
    if (typeof next === 'number' || typeof next === 'string') {
      chatStore.setUnread(next)
      hasCount = true
    }
  } catch {
    ok = false
  }

  // 没拿到服务端总数（旧接口 / 请求失败）：用 GET 兜底校准，
  // 避免本地把角标清成假值，或让 socket 累加的虚高值一直挂着
  if (!hasCount) chatStore.refresh()

  // 标记本身失败：给一次可见提示，否则用户只会看到「数字怎么都清不掉」
  if (!ok && Date.now() - readFailAt > 10000) {
    readFailAt = Date.now()
    toast.warning('已读状态同步失败，请检查网络后重试')
  }
}

/**
 * 发送一条消息（文本 / 图片共用）
 *
 * 发送统一走 HTTP：落库后再上屏，能拿到真实 id 用于去重，也能直接获得
 * 拉黑 / 超长等业务错误；服务端落库后会顺带通过 socket 推给在线的对方。
 */
async function postMessage(content, msgType = 'text', quoteId = 0) {
  const res = await sendChatMessage({
    target_uid: activeUid.value,
    content,
    msg_type: msgType,
    // 引用回复：只传被引用消息 id，摘要由服务端回查后落快照
    quote_id: Number(quoteId) || 0,
    // 幂等键：网络抖动 / 连点重试时后端不会再插一条
    client_msg_id: `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
  })

  if (res?.code !== 200) return null

  const message = res.data
  appendMessage(message)

  // 更新会话列表摘要：图片显示「[图片]」，文本截断展示
  const preview = msgType === 'image'
    ? '[图片]'
    : content.length > 60 ? `${content.slice(0, 60)}…` : content

  updateConversationRow(preview, message?.create_time)

  return message
}

/** 消息上屏（按 id 去重：接口返回与 socket 推送可能先后到达同一条） */
function appendMessage(message) {
  if (!message?.id) return
  if (messages.value.some((item) => Number(item.id) === Number(message.id))) return
  messages.value.push(message)
  scrollToBottom()
}

/** 更新当前会话在列表里的摘要；还没有这一行说明是新会话，直接重拉列表 */
function updateConversationRow(preview, createTime) {
  const row = conversations.value.find((item) => Number(item.other_uid) === Number(activeUid.value))
  if (row) {
    row.last_text = preview || ''
    row.last_time = Number(createTime) || Math.floor(Date.now() / 1000)
    row.last_uid = myUid.value
  } else {
    loadConversations()
  }
}

async function send() {
  const content = draft.value.trim()
  if (!content || sending.value || !activeUid.value || !canSend.value) return

  sending.value = true
  try {
    const message = await postMessage(content, 'text', quoteDraft.value?.id)
    // 发送成功才清空输入：失败时保留内容（含引用态），改完能直接重发
    if (message) {
      draft.value = ''
      clearQuote()
    }
  } catch (err) {
    // 业务错误已由 request.js 统一 toast；403 = 拉黑关系，顺手禁用输入区
    if (err?.code === 403) canSend.value = false
  } finally {
    sending.value = false
  }
}

/** 撤回：成功后气泡变成「你撤回了一条消息」，会话摘要按服务端重算结果刷新 */
async function revokeMessage(msg) {
  if (!canRevoke(msg)) return

  try {
    const res = await revokeChatMessage(msg.id)
    if (res?.code !== 200) return

    applyRevoke(res.data || { id: msg.id })
    toast.success('已撤回')

    // 撤回的如果是最后一条，摘要会退回到上一条，重拉列表最省心
    clearTimeout(listTimer)
    listTimer = setTimeout(loadConversations, 300)
  } catch {
    // 超过 2 分钟等业务错误已由 request.js 统一 toast
  }
}

/** 就地替换成撤回后的消息（自己撤回与收到 chat_revoke 共用） */
function applyRevoke(message) {
  if (!message?.id) return
  const index = messages.value.findIndex((item) => Number(item.id) === Number(message.id))
  if (index < 0) return
  messages.value[index] = { ...messages.value[index], ...message, is_revoked: 1, content: '' }
}

/**
 * 拍一拍
 *
 * 前端 3 秒节流只是省点请求，真正的频率限制在后端（5 秒 / 同一对用户）；
 * 拍一拍会作为一条系统消息落库，双方下次拉历史也看得到。
 */
let lastPatAt = 0

async function sendPat() {
  if (!activeUid.value || !canSend.value) return

  const now = Date.now()
  if (now - lastPatAt < 3000) return
  lastPatAt = now

  try {
    const res = await patChatUser(activeUid.value)
    if (res?.code !== 200) return

    appendMessage(res.data)
    updateConversationRow('[拍一拍]', res.data?.create_time)

    patting.value = true
    clearTimeout(patTimer)
    patTimer = setTimeout(() => {
      patting.value = false
    }, 600)
  } catch {
    // 拍得太快 / 拉黑等错误已由 request.js toast，放开节流让用户能重试
    lastPatAt = 0
  }
}

/** 已读回执：把 id <= lastMsgId 的「我发出的消息」标成已读 */
function applyReadReceipt(byUid, lastMsgId) {
  if (Number(byUid) !== Number(activeUid.value) || !lastMsgId) return
  messages.value.forEach((item) => {
    if (Number(item.from_uid) === myUid.value && Number(item.id) <= Number(lastMsgId)) {
      item.is_read = 1
    }
  })
}

/**
 * 表情回填
 *
 * 插入到光标处而不是末尾：连续点几个表情、或者在句子中间补一个表情时才顺手。
 * 表情以 [emoji:url] 标记存进正文（与评论/动态同一套渲染），
 * 服务端按普通文本存储，不改表结构。
 */
function insertEmoji(text) {
  const editor = inputRef.value
  const value = String(text || '')

  if (!editor) {
    draft.value += value
    return
  }

  // 表情要在输入框里显示成图片，走编辑器的 insertEmoji；颜文字等纯文本按普通文字插入
  if (value.startsWith('[emoji:')) {
    editor.insertEmoji(value.slice(7, -1))
    return
  }
  editor.insertText(value)
}

function pickImage() {
  if (!canSend.value || uploading.value) return
  fileRef.value?.click()
}

const MAX_IMAGE_SIZE = 5 * 1024 * 1024

/** 上传并逐张发送（多选时按顺序各发一条图片消息，语义比合并成一堆清楚） */
async function uploadAndSendImage(files) {

  if (!activeUid.value || !canSend.value) return

  const list = Array.from(files || []).filter((file) => file && String(file.type).startsWith('image/'))
  // 选到非图片（或在部分浏览器里 FileList 被提前清空）时给个提示，别让点击看起来没反应
  if (!list.length) {
    toast.warning('请选择图片文件')
    return
  }

  const oversize = list.find((file) => file.size > MAX_IMAGE_SIZE)
  if (oversize) {
    toast.warning('图片不能超过 5MB')
    return
  }

  uploading.value = true
  try {
    const form = new FormData()
    list.forEach((file) => form.append('files', file))

    const res = await uploadAttachments(form)
    const urls = (res?.data?.results || [])
      .filter((row) => row.status !== 'fail')
      .map((row) => row.full_url || row.url || '')
      .filter(Boolean)

    if (!urls.length) {
      toast.warning('图片上传失败，请稍后重试')
      return
    }

    for (const url of urls) {
      // 引用只挂在第一张上：成功发出后即消费掉，后面的图片按普通消息发送
      const message = await postMessage(url, 'image', quoteDraft.value?.id)
      // 发送失败（业务错误已由 request.js toast）：不再继续发剩下的，避免连点一屏报错
      if (!message) break
      clearQuote()
    }
  } catch {
    // 业务错误（超限 / 未登录等）已由 request.js toast；网络异常时这里补一条，
    // 否则用户只会看到「转了一下什么都没发生」
    toast.error('图片发送失败，请稍后重试')
  } finally {
    uploading.value = false
  }
}

function onFilesPicked(event) {
  const input = event.target
  // 必须先拷一份再清空：input.value = '' 会把 input.files 一起清掉（FileList 与 input 联动），
  // 之后再读 files 就是空的 —— 现象就是「选完图片没有任何反应」。
  const files = Array.from(input.files || [])
  // 清空 value：同一个文件再选一次也能触发 change
  input.value = ''
  uploadAndSendImage(files)
}

// 直接 Ctrl+V 粘贴截图：由 ChatEmojiInput 识别后抛 @files（那里同时负责纯文本粘贴）

// ---------- 实时消息 ----------
let unsubscribe = null
let unsubscribeEvent = null
let listTimer = null
let patTimer = null

function onPrivateMessage(message) {
  if (!message || !message.id) return

  const withUid = Number(message.from_uid) === Number(activeUid.value)
    || Number(message.to_uid) === Number(activeUid.value)

  if (activeUid.value && withUid) {
    appendMessage(message)

    // 被拍一拍：抖一下聊天窗，给个即时反馈
    if (message.msg_type === 'pat' && Number(message.to_uid) === myUid.value) {
      patting.value = true
      clearTimeout(patTimer)
      patTimer = setTimeout(() => {
        patting.value = false
      }, 600)
    }

    // 正在看这段对话：立刻回执已读，未读数不挂在我的角标上
    if (Number(message.to_uid) === myUid.value) markRead(activeUid.value)
    return
  }

  // 不在当前对话：刷新会话列表（摘要 / 未读角标），做个短防抖
  clearTimeout(listTimer)
  listTimer = setTimeout(loadConversations, 400)
}

/** 清掉「正在输入」提示（切会话 / 关页面时必须清，否则会挂在下一段对话上） */
function resetTyping() {
  clearTimeout(typingTimer)
  typingTimer = null
  peerTyping.value = false
}

/**
 * 已读 / 撤回 / 输入状态等事件（由服务端通过 socket 推来）
 *
 * 和普通消息分开：这些不改未读数，只改当前会话里的气泡 / 提示状态。
 */
function onChatEvent(event) {
  if (!event) return

  // 断线重连：当前会话按本地最大 seq 补拉，顺带刷新未读与会话列表
  if (event.type === 'reconnect') {
    syncMessages(activeUid.value)
    chatStore.refresh()
    clearTimeout(listTimer)
    listTimer = setTimeout(loadConversations, 300)
    return
  }

  // 对方正在输入：只认当前会话，4 秒没有新上报就自动消失
  if (event.type === 'typing') {
    if (Number(event.fromUid) !== Number(activeUid.value)) return

    peerTyping.value = true
    clearTimeout(typingTimer)
    typingTimer = setTimeout(() => {
      typingTimer = null
      peerTyping.value = false
    }, 4000)
    return
  }

  if (event.type === 'read') {
    applyReadReceipt(event.byUid, event.lastMsgId)
    return
  }

  if (event.type === 'revoke') {
    const message = event.message
    if (!message?.id) return

    const withUid = Number(message.from_uid) === Number(activeUid.value)
      || Number(message.to_uid) === Number(activeUid.value)
    if (!withUid) return

    applyRevoke(message)
    // 撤回的若是最后一条，会话摘要要跟着回退
    clearTimeout(listTimer)
    listTimer = setTimeout(loadConversations, 400)
    return
  }

  // 同一账号在其它设备上删除了消息：本机也把它从当前会话移除
  if (event.type === 'delete') {
    if (!event.id) return
    messages.value = messages.value.filter((item) => Number(item.id) !== Number(event.id))
    if (Number(quoteDraft.value?.id) === Number(event.id)) quoteDraft.value = null
  }
}

// ---------- 断线兜底 ----------
//
// 实时消息全靠 /socket 推送：长连接一旦断掉（代理重启、切网、休眠唤醒），
// 页面就再也不会自己更新，只能刷新 —— 这正是「不刷新就不显示」的根源。
// 这里在「未连上」时按固定间隔补拉，连上后立刻停掉，不白白耗请求。
const FALLBACK_INTERVAL = 15000
let fallbackTimer = null

function startFallback() {
  if (fallbackTimer || !userStore.isLogged) return
  fallbackTimer = setInterval(() => {
    // 页面切到后台时不拉：既省流量，也避免回来时一堆并发
    if (document.visibilityState !== 'visible') return
    loadConversations()
    // 当前会话按本地最大 seq 增量补拉，只取缺的那几条
    if (activeUid.value) syncMessages(activeUid.value)
    chatStore.refresh()
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

/** 回到前台：先补连一次，再把消息与列表对齐（离开期间可能积了几条） */
function onVisible() {
  if (document.visibilityState !== 'visible') return
  if (!chatStore.connected) chatStore.reconnect()
  loadConversations()
  if (activeUid.value) syncMessages(activeUid.value)
}

// 掉线后主动重连，并交给兜底轮询接管
watch(
  () => chatStore.connected,
  (open) => {
    if (!open) chatStore.reconnect()
    syncFallback()
  }
)

/** 菜单打开时按 Esc / 滚动页面 / 切走会话都要收起，避免浮在错误的位置 */
function onMenuKeydown(e) {
  if (e.key !== 'Escape') return
  closeMessageMenu()
  closeConvMenu()
}

function onMenuScroll() {
  closeMessageMenu()
  closeConvMenu()
}

// 路由变化即切换会话：从列表点另一条（/chat/2）走这里；
// 回到 /chat 时清空右栏（窄屏就是退回列表）
watch(
  () => route.params.uid,
  (value) => {
    // 游客没有会话可开：页面显示的是登录引导，不必发请求
    if (isGuest.value) return

    const uid = Number(value) || 0
    if (!uid) {
      if (activeUid.value) closeChat()
      return
    }
    if (uid !== activeUid.value) openChat(uid)
  }
)

onMounted(async () => {
  // 游客：只渲染登录引导，跳过所有需要登录的初始化请求
  if (isGuest.value) return

  chatStore.start()
  await Promise.all([loadConversations(), loadFriends()])
  // 角标以服务端为准：socket 的 +1 累加在断线重连 / 重复推送后可能偏高
  chatStore.refresh()

  // 会话由路由决定：/chat/:uid。`/chat` 时右栏保持空态，
  // 不再「自动打开第一条」——列表里点哪个才开哪个，避免刚进来就误标记已读。
  const uid = Number(route.params.uid || 0)
  if (uid > 0) openChat(uid)

  unsubscribe = chatStore.onMessage(onPrivateMessage)
  unsubscribeEvent = chatStore.onEvent(onChatEvent)

  // 进页面时如果长连接还没就绪，先用轮询顶上，连上后自动停
  syncFallback()

  document.addEventListener('keydown', onMenuKeydown)
  document.addEventListener('visibilitychange', onVisible)
  window.addEventListener('scroll', onMenuScroll, true)
})

onBeforeUnmount(() => {
  // 离开页面：防抖里的已读不能丢（否则会话列表与角标会残留未读），
  // 同时解除「正在查看」标记，避免其它页面的新消息被误判为已读而不计未读
  flushRead()
  chatStore.setViewing(0)
  clearTimeout(listTimer)
  clearTimeout(patTimer)
  clearTimeout(typingTimer)
  if (unsubscribe) unsubscribe()
  if (unsubscribeEvent) unsubscribeEvent()
  stopFallback()
  document.removeEventListener('keydown', onMenuKeydown)
  document.removeEventListener('visibilitychange', onVisible)
  window.removeEventListener('scroll', onMenuScroll, true)
})

// 地址栏变化（浏览器前进 / 后退、从作者主页再点一次「私信」）时同步当前会话
watch(
  () => route.query.to,
  (value) => {
    if (isGuest.value) return

    const uid = Number(value || 0)
    if (uid > 0 && uid !== Number(activeUid.value)) openChat(uid)
    if (!uid) closeChat()
  }
)
</script>

<style scoped>
/* ==================== 页面骨架：撑满视口，不带任何布局组件 ==================== */
.msg-app {
  /* 与 /chat 布局、群聊页共用同一套 QQ 风配色（局部变量，不影响站点其它页面） */
  --chat-blue: var(--primary);
  --chat-panel: var(--bg);
  --chat-line: var(--border-soft);

  display: flex;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
  color: var(--text);
  background: var(--bg);
}

/* 嵌在 /chat 布局里：不要自己占满视口，交给父级的 flex 决定宽度 */
.msg-app.is-embedded {
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
}

/* ==================== 左栏：会话列表 ==================== */
.msg-sidebar {
  width: 320px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--bg-card);
  border-right: 1px solid var(--border);
}
.msg-sidebar-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 18px 16px 10px;
}
.msg-sidebar-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 0.02em;
  color: var(--text);
}
.msg-total-badge {
  min-width: 18px;
  height: 18px;
  padding: 0 6px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  color: #fff;
  background: var(--primary);
  border-radius: 9px;
}
.msg-sidebar-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}

/* 提醒设置：铃铛 + 下拉面板（不用原生 select，项目里也禁用 datalist） */
.msg-alert-wrap {
  position: relative;
}
.msg-alert-mask {
  position: fixed;
  inset: 0;
  z-index: 20;
}
.msg-alert-panel {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 21;
  width: 236px;
  padding: 12px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
}
.msg-alert-title {
  margin: 0 0 10px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-soft);
  letter-spacing: 0.02em;
}
.msg-alert-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 6px 0;
}
.msg-alert-copy {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.msg-alert-copy strong {
  font-size: 12.5px;
  font-weight: 500;
  color: var(--text);
}
.msg-alert-copy em {
  font-size: 11px;
  font-style: normal;
  line-height: 1.5;
  color: var(--text-light);
}
.msg-alert-tip {
  margin: 8px 0 0;
  padding-top: 8px;
  font-size: 11px;
  line-height: 1.6;
  color: var(--text-light);
  border-top: 1px solid var(--border-soft);
}

/* 开关：与用户中心设置页同一套尺寸与配色 */
.msg-switch {
  position: relative;
  flex-shrink: 0;
  width: 40px;
  height: 22px;
  padding: 0;
  background: var(--border);
  border: none;
  border-radius: 999px;
  cursor: pointer;
  transition: background-color var(--dur-fast) var(--ease);
}
.msg-switch.is-on {
  background: var(--primary);
}
.msg-switch:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.msg-switch-knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  background: var(--switch-knob, #fff);
  border-radius: 50%;
  transition: transform var(--dur-fast) var(--ease);
}
.msg-switch.is-on .msg-switch-knob {
  transform: translateX(18px);
}

/* 图标按钮：自绘而非全局 .btn，聊天页里的按钮密度远高于普通页面 */
.msg-icon-btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  font-size: 15px;
  color: var(--text-light);
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), background-color var(--dur-fast) var(--ease);
}
.msg-icon-btn:hover:not(:disabled) {
  color: var(--primary);
  background: var(--bg-muted);
}
.msg-icon-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

/* 搜索：胶囊输入，聚焦时描边变主题色 */
.msg-search {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 16px 8px;
  padding: 0 12px;
  height: 36px;
  color: var(--text-light);
  background: var(--bg-muted);
  border: 1px solid transparent;
  border-radius: 999px;
  transition: border-color var(--dur-fast) var(--ease), background-color var(--dur-fast) var(--ease);
}
.msg-search:focus-within {
  background: var(--bg-card);
  border-color: var(--primary);
}
.msg-search input {
  flex: 1;
  min-width: 0;
  height: 100%;
  font-family: inherit;
  font-size: 13px;
  color: var(--text);
  background: transparent;
  border: 0;
  outline: none;
}
.msg-search-clear {
  display: inline-flex;
  padding: 0;
  font-size: 13px;
  color: var(--text-light);
  background: transparent;
  border: 0;
  cursor: pointer;
}
.msg-search-clear:hover {
  color: var(--text-soft);
}

/* 会话行 */
.msg-conversations {
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 4px 8px 12px;
  list-style: none;
  overflow-y: auto;
  scrollbar-width: thin;
}
/* 公共群聊入口：置顶一条，与下面的私信会话区分开 */
.msg-room-entry {
  margin-bottom: 6px;
  padding-bottom: 6px;
  border-bottom: 1px solid var(--border-soft);
}
.msg-room-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  color: inherit;
  text-decoration: none;
  border-radius: var(--radius);
  transition: background-color var(--dur-fast) var(--ease);
}
.msg-room-card:hover {
  background: var(--bg-muted);
}
.msg-room-avatar {
  flex-shrink: 0;
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 19px;
  color: var(--primary);
  background: var(--accent-wash);
  border: 1px solid var(--primary-soft);
  border-radius: 50%;
}
.msg-room-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.msg-room-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.msg-room-desc {
  font-size: 12px;
  color: var(--text-light);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.msg-room-arrow {
  flex-shrink: 0;
  font-size: 13px;
  color: var(--text-light);
}

.msg-conv {
  display: flex;
  gap: 10px;
  width: 100%;
  padding: 10px;
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius);
  cursor: pointer;
  transition: background-color var(--dur-fast) var(--ease);
}
.msg-conv:hover {
  background: var(--bg-muted);
}
.msg-conv.is-active {
  background: var(--accent-soft);
}
/* 置顶 / 免打扰标记：只做视觉提示，不占额外一行 */
.msg-conv-flag {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--primary);
}
.msg-conv-mute {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
}
.msg-room-badge {
  flex-shrink: 0;
  min-width: 18px;
  padding: 0 5px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  color: #fff;
  background: var(--danger, #e5484d);
  border-radius: 999px;
}
.msg-conv-avatar {
  position: relative;
  flex-shrink: 0;
}
.msg-conv-avatar img {
  display: block;
  width: 42px;
  height: 42px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.msg-presence {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 10px;
  height: 10px;
  background: var(--success);
  border: 2px solid var(--bg-card);
  border-radius: 50%;
}
.msg-conv-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.msg-conv-top,
.msg-conv-bottom {
  display: flex;
  align-items: center;
  gap: 8px;
}
.msg-conv-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.msg-conv.is-active .msg-conv-name {
  color: var(--primary);
}
/* 互关标记：还没聊过的好友行右侧挂一个小标签，和真实会话区分开 */
.msg-conv-tag {
  flex-shrink: 0;
  padding: 0 6px;
  font-size: 10px;
  font-weight: 500;
  line-height: 16px;
  color: var(--primary);
  background: var(--accent-soft);
  border-radius: 999px;
}
.msg-conv-time {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
}
.msg-conv-text {
  flex: 1;
  min-width: 0;
  font-size: 12.5px;
  color: var(--text-muted);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.msg-conv-badge {
  flex-shrink: 0;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  color: #fff;
  background: var(--danger);
  border-radius: 9px;
}
.msg-list-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 48px 24px;
  font-size: 12px;
  line-height: 1.7;
  text-align: center;
  color: var(--text-light);
}
.msg-list-empty i {
  font-size: 26px;
  opacity: 0.7;
}
.msg-list-empty p {
  margin: 0;
  font-size: 13px;
  color: var(--text-soft);
}

/* ==================== 右栏：聊天窗口 ==================== */
.msg-chat {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--chat-panel);
}
.msg-chat-head {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 60px;
  flex-shrink: 0;
  padding: 0 20px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--chat-line);
}
.msg-back {
  display: none;
}
.msg-peer {
  display: inline-flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  color: var(--text);
  text-decoration: none;
}
.msg-peer-avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
.msg-peer-name {
  font-size: 15px;
  font-weight: 600;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.msg-peer:hover .msg-peer-name {
  color: var(--primary);
}
.msg-peer-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11px;
  color: var(--text-light);
}
.msg-peer-status i {
  width: 6px;
  height: 6px;
  background: var(--text-light);
  border-radius: 50%;
}
.msg-peer-status.is-online {
  color: var(--success);
}
.msg-peer-status.is-online i {
  background: var(--success);
  box-shadow: 0 0 0 3px rgba(90, 160, 120, 0.16);
}
.msg-chat-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 2px;
}

/* 消息区 */
.msg-body {
  flex: 1;
  min-height: 0;
  padding: 20px 24px 8px;
  overflow-y: auto;
  scrollbar-width: thin;
}
.msg-body.is-patted {
  animation: msg-patted 0.5s var(--ease);
}
@keyframes msg-patted {
  0%,
  100% {
    transform: translateX(0);
  }
  25% {
    transform: translateX(-4px);
  }
  55% {
    transform: translateX(4px);
  }
  80% {
    transform: translateX(-2px);
  }
}
.msg-body-tip,
.msg-body-empty {
  padding: 32px 0;
  font-size: 12px;
  text-align: center;
  color: var(--text-light);
}
.msg-more {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 0 auto 16px;
  padding: 5px 14px;
  font-family: inherit;
  font-size: 12px;
  color: var(--text-soft);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}
.msg-more:hover:not(:disabled) {
  color: var(--primary);
  border-color: var(--primary-soft);
}

/* 日期分隔 */
.msg-day {
  display: flex;
  justify-content: center;
  margin: 18px 0 14px;
  font-size: 11px;
  letter-spacing: 0.04em;
  color: var(--text-light);
}
.msg-day span {
  padding: 2px 10px;
  background: var(--bg-muted);
  border-radius: 999px;
}

/* 系统提示（撤回 / 拍一拍）：居中灰字，不占气泡 */
.msg-system {
  margin: 10px 0;
  font-size: 12px;
  line-height: 1.6;
  text-align: center;
  color: var(--text-light);
}

/* 气泡 */
.msg-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 16px;
}
.msg-row.is-mine {
  flex-direction: row-reverse;
}
.msg-row-avatar {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
  cursor: pointer;
}
.msg-row-main {
  display: flex;
  flex-direction: column;
  max-width: 66%;
  min-width: 0;
}
.msg-row.is-mine .msg-row-main {
  align-items: flex-end;
}
.msg-bubble {
  padding: 8px 12px;
  font-size: 14px;
  line-height: 1.7;
  color: var(--text);
  background: var(--bg-card);
  border: 0;
  border-radius: 8px;
  border-top-left-radius: 2px;
  box-shadow: var(--shadow-sm);
  word-break: break-word;
  white-space: pre-wrap;
}
.msg-row.is-mine .msg-bubble {
  color: #fff;
  background: var(--chat-blue);
  border-radius: 8px;
  border-top-right-radius: 2px;
  box-shadow: none;
}
.msg-bubble-image {
  display: block;
  max-width: 240px;
  max-height: 240px;
  border-radius: var(--radius-sm);
  cursor: zoom-in;
}
.msg-text {
  display: inline-block;
  vertical-align: top;
  white-space: pre-wrap;
}

/* 引用块：气泡内的浅色摘要，左侧竖线区分「这是被引用的内容」 */
.msg-quote {
  display: flex;
  flex-direction: column;
  gap: 1px;
  margin-bottom: 6px;
  padding: 5px 9px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-left: 3px solid var(--primary);
  border-radius: var(--radius-xs);
  white-space: normal;
}
.msg-quote-name {
  font-weight: 600;
  color: var(--text-light);
}
.msg-quote-text {
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  line-clamp: 2;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  word-break: break-word;
}
/* 自己的气泡是主色底：引用块改成半透明白，避免浅灰在深色上糊成一片 */
.msg-row.is-mine .msg-quote {
  color: rgba(255, 255, 255, 0.82);
  background: rgba(255, 255, 255, 0.14);
  border-left-color: rgba(255, 255, 255, 0.65);
}
.msg-row.is-mine .msg-quote-name {
  color: #fff;
}

/* 时间 / 撤回 / 已读：气泡下方的小字行 */
.msg-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-light);
  min-height: 15px;
}
.msg-revoke {
  padding: 0;
  font-family: inherit;
  font-size: 11px;
  color: var(--text-light);
  background: transparent;
  border: 0;
  cursor: pointer;
  opacity: 0;
  transition: opacity var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}
.msg-row:hover .msg-revoke {
  opacity: 1;
}
.msg-revoke:hover {
  color: var(--danger);
}
.msg-read.is-read {
  color: var(--primary);
}
.msg-row.is-mine .msg-read.is-read {
  color: var(--primary-deep);
}

/* ==================== 输入区 ==================== */
.msg-composer {
  flex-shrink: 0;
  padding: 10px 20px calc(12px + env(safe-area-inset-bottom));
  background: var(--bg-card);
  border-top: 1px solid var(--chat-line);
}
.msg-composer-box {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 6px;
  padding: 6px 6px 6px 8px;
  background: var(--bg-card);
  border: 1px solid var(--chat-line);
  border-radius: 8px;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}
.msg-composer-box:focus-within {
  border-color: var(--chat-blue);
  box-shadow: 0 0 0 3px var(--accent-ring);
}
.msg-tools {
  display: flex;
  align-items: center;
  gap: 0;
  flex-shrink: 0;
  margin-bottom: 2px;
}
.msg-emoji-wrap {
  position: relative;
}
/* 表情面板统一向上弹：输入框永远在底部，向下弹会被聊天区裁掉 */
.msg-emoji-wrap :deep(.emoji-picker-panel) {
  top: auto;
  bottom: calc(100% + 8px);
  z-index: 3200;
}
/* 窄屏：面板按视口宽度铺开，从左侧补回偏移
   断点与 EmojiPicker 内部的手机端样式（768px）保持一致，
   否则 641~768 区间面板会退化成「只有按钮那么宽」的一条竖条 */
@media (max-width: 768px) {
  .msg-emoji-wrap :deep(.emoji-picker-panel) {
    left: -8px;
    right: auto;
    width: calc(100vw - 32px);
    min-width: 0;
    max-width: calc(100vw - 32px);
  }
}
.msg-input {
  flex: 1;
  min-width: 0;
  min-height: 36px;
  max-height: 132px;
  padding: 8px 2px;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.6;
  color: var(--text);
  background: transparent;
  border: 0;
  outline: none;
  resize: none;
}
.msg-input::placeholder {
  color: var(--text-light);
}
.msg-send {
  min-width: 64px;
  height: 34px;
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 18px;
  font-family: inherit;
  font-size: 13px;
  color: #fff;
  background: var(--chat-blue);
  border: 0;
  border-radius: 6px;
  cursor: pointer;
  transition: filter var(--dur-fast) var(--ease), opacity var(--dur-fast) var(--ease);
}
.msg-send:hover:not(:disabled) {
  filter: brightness(1.06);
}
.msg-send:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
/* 不用 display:none：部分浏览器 / 内嵌 WebView 对不可见元素的 click() 不弹选择框。
   用「fixed + 移出视口 + 透明」隐藏，既不影响布局，点击行为也和真实元素一致。 */
.msg-file {
  position: fixed;
  left: -9999px;
  top: -9999px;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}
.msg-blocked {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  margin: 0;
  padding: 10px 0;
  font-size: 12px;
  color: var(--danger);
  background: var(--accent-wash);
  border-radius: var(--radius);
}

/* 未选中会话时的占位 */
.msg-placeholder {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 24px;
  text-align: center;
  color: var(--text-light);
}
.msg-placeholder-mark {
  width: 64px;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 6px;
  font-size: 28px;
  color: var(--primary);
  background: var(--accent-wash);
  border: 1px solid var(--primary-soft);
  border-radius: 50%;
}
.msg-placeholder h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
}
.msg-placeholder p {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.8;
}
.msg-placeholder-link {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-top: 10px;
  font-size: 12.5px;
  color: var(--primary);
  text-decoration: none;
}
.msg-placeholder-link:hover {
  color: var(--primary-deep);
}

/* 游客引导：两个出口并排（去登录 / 进公共大厅） */
.msg-guest-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  margin-top: 10px;
}
.msg-guest-btn {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  height: 32px;
  padding: 0 16px;
  font-family: inherit;
  font-size: 12.5px;
  color: var(--text-soft);
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  text-decoration: none;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease);
}
.msg-guest-btn:hover {
  color: var(--primary);
  border-color: var(--primary-soft);
}
.msg-guest-btn.is-primary {
  color: #fff;
  background: var(--primary);
  border-color: var(--primary);
}
.msg-guest-btn.is-primary:hover {
  color: #fff;
  background: var(--primary-deep);
  border-color: var(--primary-deep);
}

/* 旋转（加载中） */
.is-spin {
  animation: msg-spin 0.9s linear infinite;
}
@keyframes msg-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ==================== 窄屏：列表 / 聊天窗二选一 ==================== */
@media (max-width: 860px) {
  .msg-sidebar {
    width: 100%;
    border-right: 0;
  }
  .msg-app.is-chat .msg-sidebar {
    display: none;
  }
  .msg-app:not(.is-chat) .msg-chat {
    display: none;
  }
  .msg-back {
    display: inline-flex;
  }
  .msg-body {
    padding: 16px 14px 6px;
  }
  .msg-row-main {
    max-width: 80%;
  }
  .msg-chat-head {
    padding: 0 12px;
  }
  .msg-composer {
    padding: 8px 12px calc(10px + env(safe-area-inset-bottom));
  }
  /* 触屏没有 hover：撤回按钮常显，否则永远点不到 */
  .msg-revoke {
    opacity: 1;
  }
  .msg-bubble-image {
    max-width: 200px;
    max-height: 200px;
  }
}

/* ==================== 引用态（输入框上方） ==================== */
.msg-quote-draft {
  flex-basis: 100%;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  margin-bottom: 2px;
  background: var(--bg-muted);
  border-left: 3px solid var(--primary);
  border-radius: var(--radius-xs);
}
.msg-quote-draft-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
  font-size: 12px;
  line-height: 1.5;
}

/* ==================== 消息右键菜单 ==================== */
.msg-menu-mask {
  position: fixed;
  inset: 0;
  z-index: 60;
}
.msg-menu {
  position: fixed;
  z-index: 61;
  min-width: 148px;
  padding: 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
}
.msg-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 34px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--text-soft);
  text-align: left;
  white-space: nowrap;
  background: none;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), background-color var(--dur-fast) var(--ease);
}
.msg-menu-item:hover {
  color: var(--text);
  background: var(--bg-muted);
}
.msg-menu-item i {
  font-size: 14px;
  opacity: 0.9;
}
.msg-menu-item.is-danger {
  color: var(--danger);
}
.msg-menu-item.is-danger:hover {
  color: var(--danger);
  background: var(--accent-soft);
}
</style>
