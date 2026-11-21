<template>
  <div class="chat-notice-stack" aria-live="polite">
    <TransitionGroup name="chat-notice">
      <div
        v-for="item in list"
        :key="item.key"
        class="chat-notice"
        role="button"
        tabindex="0"
        @click="openChat(item)"
        @keydown.enter="openChat(item)"
        @mouseenter="pause(item)"
        @mouseleave="schedule(item)"
      >
        <span class="chat-notice-avatar">
          <img v-if="item.avatar" :src="item.avatar" alt="" loading="lazy" />
          <i v-else class="bi bi-person-fill" aria-hidden="true" />
          <i class="chat-notice-dot" aria-hidden="true" />
        </span>

        <div class="chat-notice-body">
          <div class="chat-notice-top">
            <span v-if="item.kind === 'room'" class="chat-notice-tag">群聊</span>
            <span class="chat-notice-name">{{ item.nickname || `用户 #${item.peerUid}` }}</span>
            <span v-if="item.count > 1" class="chat-notice-count">{{ item.count }} 条</span>
          </div>
          <p class="chat-notice-text">{{ item.preview }}</p>
        </div>

        <button type="button" class="chat-notice-close" title="关闭" @click.stop="remove(item.key)">
          <i class="bi bi-x-lg" aria-hidden="true" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { getUser } from '@/api/users'
import { unlockNoticeSound } from '@/utils/noticeSound'
import { showDesktopNotice } from '@/utils/desktopNotify'
import { getFullUrl } from '@/utils/emoji'

/**
 * 新消息悬浮通知（右下角）
 *
 * 两条通道，由 store 判断该走哪条（看 alert.toast / alert.desktop 与页面可见性）：
 *   - 前台：右下角浮层（本组件绘制的卡片）；
 *   - 后台：系统级桌面通知（Notification API，见 utils/desktopNotify.js），
 *     窗口最小化 / 切到别的标签页也能弹。
 *
 * 浮层这边做三件事：
 *   1. 同一个人连发多条时合并成一条并累计条数，避免刷屏；
 *   2. 最多同时 3 条，6 秒自动消失，鼠标悬停时暂停倒计时；
 *   3. 点击直接跳到 /messages?to=<uid>。
 */
const router = useRouter()
const chatStore = useChatStore()

const MAX_NOTICES = 3
const DURATION = 6000

const list = ref([])
const timers = new Map()
const userCache = new Map()
// 每个会话的系统通知累计条数（系统通知没有「条数」概念，自己记在正文里）
const desktopState = new Map()
// 超过这个间隔就重新计数，避免用户下午看通知时还显示「上午的 5 条」
const DESKTOP_COUNT_WINDOW = 60 * 1000

let seq = 0
let unsubscribe = null

/**
 * 通知内容：私信按消息类型取摘要；群聊 @ 只带一小段正文预览
 *
 * 两者共用同一套浮层 / 桌面通知，靠 payload.kind 区分（默认就是私信）。
 */
function previewOf(message, kind = 'dm', preview = '') {
  if (kind === 'room') {
    const text = String(preview || '').replace(/\s+/g, ' ').trim()
    return `@你：${text || '在群聊里提到了你'}`
  }
  if (!message) return ''
  if (message.msg_type === 'image') return '[图片]'
  if (message.msg_type === 'pat') return '拍了拍你'

  const text = String(message.content || '')
    .replace(/\[emoji:[^\]]+\]/g, '[表情]')
    .replace(/\s+/g, ' ')
    .trim()

  if (!text) return '发来一条消息'
  return text.length > 42 ? `${text.slice(0, 42)}…` : text
}

function clearTimer(key) {
  const timer = timers.get(key)
  if (timer) {
    clearTimeout(timer)
    timers.delete(key)
  }
}

function remove(key) {
  clearTimer(key)
  list.value = list.value.filter((item) => item.key !== key)
}

/** 重置某条通知的自动消失倒计时 */
function schedule(item) {
  if (!item) return
  clearTimer(item.key)
  timers.set(item.key, setTimeout(() => remove(item.key), DURATION))
}

function pause(item) {
  if (item) clearTimer(item.key)
}

/** 取用户信息并缓存（消息体里只有 from_uid，没有昵称头像；后台通知也要用） */
async function ensureUser(uid) {
  const hit = userCache.get(uid)
  if (hit) return hit

  try {
    const res = await getUser(uid, 'id,nickname,avatar')
    const user = res?.data || null
    if (user) userCache.set(uid, user)
    return user
  } catch {
    return null
  }
}

/** 补昵称与头像，失败就先用「用户 #id」顶着 */
async function fillUser(uid, kind) {
  const user = await ensureUser(uid)
  if (!user) return

  const item = list.value.find((row) => row.peerUid === uid && row.kind === kind)
  if (!item) return
  item.nickname = user.nickname || ''
  item.avatar = user.avatar || ''
}

function pushNotice({ peerUid, message, kind = 'dm', preview = '' } = {}) {
  const uid = Number(peerUid) || 0
  if (!uid) return
  // 私信必须有消息体；群聊 @ 只有一段正文预览
  if (kind !== 'room' && !message) return

  const text = previewOf(message, kind, preview)

  // 同一个人还在屏幕上：合并成一条，累加条数并重置倒计时
  const exist = list.value.find((item) => item.peerUid === uid && item.kind === kind)
  if (exist) {
    exist.count += 1
    exist.preview = text
    schedule(exist)
    return
  }

  const item = {
    key: `notice-${Date.now()}-${(seq += 1)}`,
    kind,
    peerUid: uid,
    nickname: '',
    avatar: '',
    preview: text,
    count: 1
  }

  list.value.push(item)

  // 超出上限：把最早的一条挤掉（连同它的 timer）
  while (list.value.length > MAX_NOTICES) {
    remove(list.value[0].key)
  }

  schedule(item)
  fillUser(uid, kind)
}

/**
 * 系统级桌面通知（页面在后台时走这里）
 *
 * 同一个人用固定 tag：后一条会替换前一条，不会在系统通知中心堆成一列。
 * 头像要绝对地址，站内相对路径先补全域名，否则系统拿不到图。
 */
async function notifyDesktop({ peerUid, message, kind = 'dm', preview = '', roomId = 1 } = {}) {
  const uid = Number(peerUid) || 0
  if (!uid) return
  if (kind !== 'room' && !message) return

  const user = await ensureUser(uid)
  const name = user?.nickname || `用户 #${uid}`
  const avatar = user?.avatar ? getFullUrl(user.avatar) : undefined

  // 同一会话连续来消息：系统通知只保留最新一条（靠 tag 替换），
  // 条数自己数，写进正文，用户才知道「不止这一条」
  const now = Date.now()
  const key = `${kind}-${uid}`
  const prev = desktopState.get(key)
  const count = prev && now - prev.at < DESKTOP_COUNT_WINDOW ? prev.count + 1 : 1
  desktopState.set(key, { count, at: now })

  const text = previewOf(message, kind, preview)
  const body = count > 1 ? `${text}（${count} 条）` : text

  showDesktopNotice({
    title: kind === 'room' ? `群聊 · ${name}` : `私信 · ${name}`,
    body,
    icon: avatar,
    tag: `mellow-${kind}-${uid}`,
    // 用户关掉提示音时让系统通知也静音：系统侧的声音不受 Web Audio 开关控制
    silent: !chatStore.alert.sound,
    onClick: () => {
      desktopState.delete(key)
      if (kind === 'room') {
        router.push(`/chat/room/${Number(roomId) || 1}`).catch(() => {})
        return
      }
      router.push(`/chat/${uid}`).catch(() => {})
    }
  })
}

/** 通知分发：后台 → 系统通知；前台 → 右下角浮层 */
function handleNotice(payload) {
  if (!payload) return
  if (payload.background) {
    notifyDesktop(payload)
    return
  }
  pushNotice(payload)
}

function openChat(item) {
  const uid = Number(item?.peerUid) || 0
  const kind = item?.kind || 'dm'
  remove(item?.key)
  if (!uid) return

  // 群聊 @ 直接回「被提到的那个群」；私信跳到对应会话
  if (kind === 'room') {
    router.push(`/chat/room/${Number(item?.roomId) || 1}`).catch(() => {})
    return
  }

  // 已经在「和这个人的会话里」就什么都不用做（push 到同一位置会被 vue-router 忽略）
  router.push(`/chat/${uid}`).catch(() => {})
}

onMounted(() => {
  unsubscribe = chatStore.onNotice(handleNotice)
  // 浏览器要求用户手势后才能播放音频：任意一次点击/触摸时解锁一次
  document.addEventListener('pointerdown', unlockNoticeSound, { once: true })
})

onBeforeUnmount(() => {
  if (unsubscribe) unsubscribe()
  document.removeEventListener('pointerdown', unlockNoticeSound)
  timers.forEach((timer) => clearTimeout(timer))
  timers.clear()
})
</script>

<style scoped>
/* 右下角通知栈
   right 留出 76px：站点全局的悬浮按钮（主题切换 / 返回顶部）固定在右侧 18px，
   通知卡不能压住它；窄屏则改为贴底并整体上移，避开手机底栏。 */
.chat-notice-stack {
  position: fixed;
  right: 76px;
  bottom: 24px;
  z-index: 3600;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 320px;
  max-width: calc(100vw - 32px);
  pointer-events: none;
}

.chat-notice {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  cursor: pointer;
  pointer-events: auto;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-left: 3px solid var(--primary);
  border-radius: var(--radius);
  box-shadow: var(--shadow-md);
  transition: transform var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}
.chat-notice:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-lg);
}

.chat-notice-avatar {
  position: relative;
  flex-shrink: 0;
  width: 38px;
  height: 38px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  font-size: 17px;
  color: var(--text-light);
  background: var(--bg-muted);
  border-radius: 50%;
}
.chat-notice-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.chat-notice-dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 9px;
  height: 9px;
  background: var(--primary);
  border: 2px solid var(--bg-card);
  border-radius: 50%;
}

.chat-notice-body {
  flex: 1;
  min-width: 0;
}
.chat-notice-top {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-right: 18px;
}
.chat-notice-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
/* 「群聊」小标签：和私信通知区分开，点进去是房间而不是会话 */
.chat-notice-tag {
  flex-shrink: 0;
  padding: 1px 6px;
  font-size: 10px;
  color: var(--text-soft);
  background: var(--bg-muted);
  border-radius: 999px;
}
.chat-notice-count {
  flex-shrink: 0;
  padding: 1px 6px;
  font-size: 11px;
  color: var(--primary);
  background: var(--accent-wash);
  border-radius: 999px;
}
.chat-notice-text {
  margin: 3px 0 0;
  padding-right: 4px;
  font-size: 12.5px;
  line-height: 1.65;
  color: var(--text-muted);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}

.chat-notice-close {
  position: absolute;
  top: 8px;
  right: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  font-size: 10px;
  color: var(--text-light);
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
  opacity: 0;
  transition: opacity var(--dur-fast) var(--ease), background-color var(--dur-fast) var(--ease);
}
.chat-notice:hover .chat-notice-close {
  opacity: 1;
}
.chat-notice-close:hover {
  color: var(--text);
  background: var(--bg-muted);
}

/* 进出场：从右侧滑入，整体高度平滑收拢 */
.chat-notice-enter-active,
.chat-notice-leave-active,
.chat-notice-move {
  transition: opacity var(--dur) var(--ease), transform var(--dur) var(--ease);
}
.chat-notice-enter-from,
.chat-notice-leave-to {
  opacity: 0;
  transform: translateX(16px);
}
.chat-notice-leave-active {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 100%;
}

@media (max-width: 640px) {
  .chat-notice-stack {
    left: 12px;
    right: 12px;
    bottom: calc(84px + env(safe-area-inset-bottom));
    width: auto;
    max-width: none;
  }
  /* 触屏没有 hover：关闭按钮常显 */
  .chat-notice-close {
    opacity: 1;
  }
  .chat-notice-leave-active {
    right: 12px;
  }
}
</style>
