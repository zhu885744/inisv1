<template>
  <div class="my">
    <!--
      两栏结构（.my-main / .my-side）只在 PC 生效：
      手机端这两个容器是 display: contents（见样式），内部卡片直接参与外层单列栅格，
      因此视觉顺序完全等于 DOM 顺序，与 PC 端适配前的手机端表现一致。
    -->
    <div class="my-main">
      <!-- 资料卡：昵称可进个人主页；右侧签到 -->
      <section class="my-profile">
        <div class="my-profile-top">
          <span class="my-avatar-wrap">
            <img class="my-avatar" :src="avatar" :alt="nickname" @error="onAvatarError" loading="lazy" />
            <!-- 认证标识统一挂在头像右下角，与侧边栏 / 作者主页保持一处 -->
            <VerifiedBadge v-if="user?.cert?.title" class="my-avatar-cert" :cert="user.cert" variant="dot" />
          </span>
          <div class="my-profile-text">
            <RouterLink v-if="uid" :to="`/author/${uid}`" class="my-nickname" title="进入我的主页">
              <span class="my-nickname-text">{{ nickname }}</span>
              <i class="bi bi-chevron-right" aria-hidden="true" />
            </RouterLink>
            <span v-else class="my-nickname">
              <span class="my-nickname-text">{{ nickname }}</span>
            </span>
            <p class="my-sub">{{ subTitle }}</p>
          </div>
          <button
            v-if="checkin.enabled"
            type="button"
            class="btn btn-sm"
            :class="checkin.checked ? 'btn-ghost' : 'btn-primary'"
            :disabled="checkin.checked || signing"
            @click="onSignIn"
          >
            {{ checkin.checked ? '已签到' : signing ? '签到中…' : '签到' }}
          </button>
        </div>

        <!-- 三格概览：金币 / 经验 / 今日任务 -->
        <div class="my-stats">
          <RouterLink to="/user/integral" class="my-stat">
            <span class="my-stat-label">积分</span>
            <strong class="my-stat-value">{{ integral }}</strong>
          </RouterLink>
          <RouterLink to="/user/exp" class="my-stat">
            <span class="my-stat-label">经验</span>
            <strong class="my-stat-value">{{ exp }}</strong>
          </RouterLink>
          <RouterLink to="/user/integral" class="my-stat">
            <span class="my-stat-label">今日任务</span>
            <strong class="my-stat-value">{{ tasks.done_count }}/{{ tasks.total_count }}</strong>
          </RouterLink>
        </div>
      </section>

      <!-- 未读消息 -->
      <RouterLink to="/notifications" class="my-notice">
        <span class="my-notice-icon">
          <i class="bi bi-bell-fill" aria-hidden="true" />
          <em v-if="notifCount > 0" class="my-notice-dot" />
        </span>
        <span class="my-notice-text">
          <template v-if="notifCount > 0">{{ notifCount }} 条未读消息</template>
          <template v-else>暂无未读消息</template>
        </span>
        <i class="bi bi-chevron-right my-arrow" aria-hidden="true" />
      </RouterLink>

      <!-- 常用入口（可编辑：隐藏 / 恢复，记在本地） -->
      <section class="my-card">
        <header class="my-card-head">
          <h3 class="my-card-title">常用</h3>
          <button type="button" class="my-link" @click="editing = !editing">
            {{ editing ? '完成' : '编辑' }}
          </button>
        </header>

        <div class="my-grid">
          <button
            v-for="item in visibleShortcuts"
            :key="item.key"
            type="button"
            class="my-grid-item"
            :class="{ 'is-editing': editing }"
            @click="onShortcutTap(item)"
          >
            <span class="my-grid-icon"><i :class="item.icon" aria-hidden="true" /></span>
            <span class="my-grid-label">{{ item.name }}</span>
            <span v-if="editing" class="my-grid-badge"><i class="bi bi-dash" /></span>
          </button>
        </div>

        <template v-if="editing && hiddenShortcuts.length">
          <p class="my-tip">已隐藏（点一下加回来）</p>
          <div class="my-grid">
            <button
              v-for="item in hiddenShortcuts"
              :key="item.key"
              type="button"
              class="my-grid-item is-hidden"
              @click="toggleShortcut(item)"
            >
              <span class="my-grid-icon"><i :class="item.icon" aria-hidden="true" /></span>
              <span class="my-grid-label">{{ item.name }}</span>
              <span class="my-grid-badge is-add"><i class="bi bi-plus" /></span>
            </button>
          </div>
        </template>
      </section>

      <!-- 最近使用（点过的入口按时间倒序，最多 5 个） -->
      <section v-if="recent.length" class="my-card">
        <header class="my-card-head">
          <h3 class="my-card-title">
            最近使用
            <small class="my-card-sub">最近打开过的功能</small>
          </h3>
        </header>
        <div class="my-grid">
          <button
            v-for="item in recent"
            :key="'recent-' + item.key"
            type="button"
            class="my-grid-item"
            @click="onShortcutTap(item)"
          >
            <span class="my-grid-icon"><i :class="item.icon" aria-hidden="true" /></span>
            <span class="my-grid-label">{{ item.name }}</span>
          </button>
        </div>
      </section>
    </div>

    <!-- 侧栏（PC 右列：等级 / 账号安全；手机端平铺到列表里，位置不变） -->
    <aside class="my-side">
      <div class="my-cards">
        <RouterLink to="/user/exp" class="my-card my-mini">
          <header class="my-mini-head"><i class="bi bi-stars" /> 等级</header>
          <strong class="my-mini-value">{{ levelText }}</strong>
          <div class="my-bar"><span :style="{ width: expPercent + '%' }" /></div>
          <p class="my-mini-hint">{{ levelHint }}</p>
        </RouterLink>

        <RouterLink to="/user/security" class="my-card my-mini">
          <header class="my-mini-head"><i class="bi bi-shield-lock" /> 账号安全</header>
          <strong class="my-mini-value">{{ securityValue }}</strong>
          <p class="my-mini-hint">{{ securityHint }}</p>
        </RouterLink>
      </div>
    </aside>

    <!-- 今日任务（取前 3 条，更多进「我的积分」）
         PC 端放在左列第二行（见样式里的栅格定位），手机端保持在列表末尾 -->
    <section v-if="taskList.length" class="my-card my-tail">
      <header class="my-card-head">
        <h3 class="my-card-title">
          今日任务
          <small class="my-card-sub">已完成 {{ tasks.done_count }}/{{ tasks.total_count }}</small>
        </h3>
        <RouterLink to="/user/integral" class="my-link">更多</RouterLink>
      </header>
      <ul class="my-tasks">
        <li v-for="task in taskList.slice(0, 3)" :key="task.type" class="my-task" :class="{ 'is-done': task.done }">
          <i class="bi" :class="task.done ? 'bi-check-circle-fill' : 'bi-circle'" aria-hidden="true" />
          <span class="my-task-name">{{ task.name }}</span>
          <span class="my-task-value">{{ task.done ? '已完成' : task.remain > 0 ? `+${task.remain}` : '可继续' }}</span>
        </li>
      </ul>
    </section>

    <!-- 协议入口（今日任务下方）
         路由见 router/index.js：/agreement（用户协议）、/privacy（隐私协议），
         与本项目登录/注册弹窗（components/AuthAgreement.vue）读的是同一份站点配置。 -->
    <nav class="my-links" aria-label="用户协议与隐私协议">
      <RouterLink to="/agreement" class="my-link-item">
        <i class="bi bi-file-earmark-text" aria-hidden="true" />
        <span class="my-link-item-text">用户协议</span>
        <i class="bi bi-chevron-right my-arrow" aria-hidden="true" />
      </RouterLink>
      <RouterLink to="/privacy" class="my-link-item">
        <i class="bi bi-shield-check" aria-hidden="true" />
        <span class="my-link-item-text">隐私协议</span>
        <i class="bi bi-chevron-right my-arrow" aria-hidden="true" />
      </RouterLink>
    </nav>

    <!-- 发布动态：动态列表页已下线，与首页共用同一个发布弹窗 -->
    <MomentComposeDialog v-model:visible="composeOpen" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { getIntegral, getIntegralTasks } from '@/api/goods'
import { getCheckinStatus, signIn } from '@/api/checkin'
import { toast } from '@/utils/toast'
import MomentComposeDialog from '@/components/MomentComposeDialog.vue'
import VerifiedBadge from '@/components/VerifiedBadge.vue'

/**
 * 「我的」聚合页（/user）
 *
 * 定位：手机端底栏「我的」落地页 —— 以**导航入口 + 信息展示**为主，
 * 不做复杂表单（具体设置在各子页里）。
 *
 * 所有入口都对应本项目真实存在的路由，没有死链：
 *   - 个人主页（/author/:id）由头像旁的昵称进入
 *   - 等级 / 积分 / 装扮 / 商城 / 创作中心 / 消息 / 安全设置等均为既有页面
 *
 * 「常用」的显示项与「最近使用」都存 localStorage，属于纯前端偏好，不落库。
 */
const SHORTCUTS_STORE = 'mellow_my_shortcuts'
const RECENT_STORE = 'mellow_my_recent'
const RECENT_MAX = 5
const DEFAULT_VISIBLE = 8

// 全部可放进「常用」的入口（名称 / 图标与侧栏、底栏口径保持一致）
const CATALOGUE = [
  // 发布动态没有独立页面（动态列表页已下线）：点击直接打开发布弹窗，见 onShortcutTap
  { key: 'moment-new', name: '发布动态', icon: 'bi bi-chat-square-text', action: 'compose' },
  { key: 'manage', name: '创作中心', icon: 'bi bi-pencil-square', to: '/manage' },
  { key: 'manage-posts', name: '我的文章', icon: 'bi bi-journal-text', to: '/manage/posts' },
  { key: 'integral', name: '我的积分', icon: 'bi bi-coin', to: '/user/integral' },
  { key: 'exp', name: '等级经验', icon: 'bi bi-stars', to: '/user/exp' },
  { key: 'decorations', name: '我的装扮', icon: 'bi bi-gem', to: '/user/decorations' },
  { key: 'badges', name: '我的勋章', icon: 'bi bi-award', to: '/user/badges' },
  { key: 'verification', name: '我的认证', icon: 'bi bi-patch-check', to: '/user/verification' },
  { key: 'goods', name: '积分商城', icon: 'bi bi-shop', to: '/goods' },
  { key: 'notifications', name: '消息通知', icon: 'bi bi-bell', to: '/notifications' },
  { key: 'blocks', name: '拉黑与举报', icon: 'bi bi-shield-slash', to: '/user/blocks' },
  { key: 'checkin', name: '每日签到', icon: 'bi bi-calendar-check', to: '/checkin' },
  { key: 'write', name: '写新文章', icon: 'bi bi-file-earmark-plus', to: '/manage/posts/write' },
  { key: 'decoration-shop', name: '装扮商城', icon: 'bi bi-palette', to: '/goods?tab=decoration' },
  { key: 'profile', name: '个人资料', icon: 'bi bi-person', to: '/user/profile' },
  { key: 'settings', name: '隐私设置', icon: 'bi bi-gear', to: '/user/settings' },
  { key: 'security', name: '账号安全', icon: 'bi bi-shield-lock', to: '/user/security' }
]

const router = useRouter()
const userStore = useUserStore()
const notif = useNotificationStore()
const { user } = storeToRefs(userStore)
const { count: notifCount } = storeToRefs(notif)

const defaultAvatar =
  'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="34" fill="%238a8a82" font-family="serif">用</text></svg>'

const uid = computed(() => user.value?.id || 0)
const nickname = computed(() => user.value?.nickname || '未登录')

// 头像：user 是异步就绪的（ensureLogin），所以用 computed 跟随；
// 加载失败时退化为占位图，避免破图
const avatarFailed = ref(false)
const avatar = computed(() => (avatarFailed.value ? defaultAvatar : user.value?.avatar || defaultAvatar))

function onAvatarError() {
  avatarFailed.value = true
}

// ---------- 数据 ----------
const summary = ref({ integral: 0 })
const tasks = ref({ list: [], done_count: 0, total_count: 0, streak: 0 })
const checkin = ref({ enabled: false, checked: false, streak: 0 })
const signing = ref(false)

const taskList = computed(() => tasks.value.list || [])
const integral = computed(() => Number(summary.value.integral) || 0)
const exp = computed(() => Number(user.value?.exp) || 0)

// 等级：Lv.N · 已连续签到 N 天
const levelInfo = computed(() => user.value?.result?.level || null)
const levelValue = computed(() => Number(levelInfo.value?.current?.value) || 0)
const levelText = computed(() => (levelValue.value ? `Lv.${levelValue.value}` : '—'))

const expPercent = computed(() => {
  const cur = Number(levelInfo.value?.current?.exp) || 0
  const next = Number(levelInfo.value?.next?.exp) || 0
  if (!next || next <= cur) return 100
  const p = ((exp.value - cur) / (next - cur)) * 100
  return Math.max(0, Math.min(100, Math.round(p)))
})

const levelHint = computed(() => {
  const next = Number(levelInfo.value?.next?.exp) || 0
  if (!next) return '已是最高等级'
  const gap = Math.max(0, next - exp.value)
  return gap > 0 ? `还差 ${gap} 经验升级` : '可以升级了'
})

const subTitle = computed(() => {
  const parts = []
  if (levelValue.value) parts.push(`Lv.${levelValue.value}`)
  if (Number(checkin.value.streak) > 0) parts.push(`已连续签到 ${checkin.value.streak} 天`)
  return parts.join(' · ') || '欢迎回来'
})

const securityValue = computed(() => (user.value?.email ? '已绑定' : '未绑定'))
const securityHint = computed(() => {
  const bound = []
  if (user.value?.email) bound.push('邮箱')
  if (user.value?.phone) bound.push('手机号')
  return bound.length ? `已绑定 ${bound.join(' / ')}` : '建议绑定邮箱与手机号'
})

// ---------- 常用入口（本地偏好） ----------
const visibleKeys = ref([])

function loadShortcutKeys() {
  try {
    const raw = JSON.parse(localStorage.getItem(SHORTCUTS_STORE) || 'null')
    if (Array.isArray(raw) && raw.length) {
      // 过滤掉目录里已不存在的 key（例如以后下线某个入口）
      const valid = raw.filter((key) => CATALOGUE.some((item) => item.key === key))
      if (valid.length) {
        visibleKeys.value = valid
        return
      }
    }
  } catch {
    // 解析失败按默认展示
  }
  visibleKeys.value = CATALOGUE.slice(0, DEFAULT_VISIBLE).map((item) => item.key)
}

const visibleShortcuts = computed(() =>
  visibleKeys.value.map((key) => CATALOGUE.find((item) => item.key === key)).filter(Boolean)
)
const hiddenShortcuts = computed(() => CATALOGUE.filter((item) => !visibleKeys.value.includes(item.key)))

const editing = ref(false)

function saveShortcutKeys() {
  try {
    localStorage.setItem(SHORTCUTS_STORE, JSON.stringify(visibleKeys.value))
  } catch {
    // 隐私模式下写入失败也不影响本次会话使用
  }
}

// 编辑态：点「常用」里的项 = 隐藏；点「已隐藏」里的项 = 恢复
function toggleShortcut(item) {
  if (visibleKeys.value.includes(item.key)) {
    visibleKeys.value = visibleKeys.value.filter((key) => key !== item.key)
  } else {
    visibleKeys.value = [...visibleKeys.value, item.key]
  }
  saveShortcutKeys()
}

// ---------- 最近使用（本地偏好） ----------
const recentKeys = ref([])

function loadRecent() {
  try {
    const raw = JSON.parse(localStorage.getItem(RECENT_STORE) || 'null')
    if (Array.isArray(raw)) {
      recentKeys.value = raw.filter((key) => CATALOGUE.some((item) => item.key === key)).slice(0, RECENT_MAX)
    }
  } catch {
    recentKeys.value = []
  }
}

const recent = computed(() =>
  recentKeys.value.map((key) => CATALOGUE.find((item) => item.key === key)).filter(Boolean)
)

function pushRecent(item) {
  const next = [item.key, ...recentKeys.value.filter((key) => key !== item.key)].slice(0, RECENT_MAX)
  recentKeys.value = next
  try {
    localStorage.setItem(RECENT_STORE, JSON.stringify(next))
  } catch {
    // 同上，忽略
  }
}

// 发布动态弹窗（CATALOGUE 里 action === 'compose' 的入口用）
const composeOpen = ref(false)

// 编辑态下点「常用」是增删；非编辑态才跳转并记入最近使用
function onShortcutTap(item) {
  if (editing.value) {
    toggleShortcut(item)
    return
  }
  pushRecent(item)
  // 「发布动态」没有目标页面：就地打开发布弹窗，不做路由跳转
  if (item.action === 'compose') {
    composeOpen.value = true
    return
  }
  router.push(item.to)
}

// ---------- 签到 ----------
async function onSignIn() {
  if (signing.value || checkin.value.checked) return
  signing.value = true
  try {
    await signIn()
    toast.success('签到成功')
    await Promise.all([loadCheckin(), loadSummary(), loadTasks()])
  } catch {
    // request 拦截器已提示
  } finally {
    signing.value = false
  }
}

// ---------- 加载 ----------
async function loadSummary() {
  try {
    const res = await getIntegral()
    summary.value = res.data || summary.value
  } catch {
    // 忽略：保持默认值
  }
}

async function loadTasks() {
  try {
    const res = await getIntegralTasks()
    tasks.value = res.data || tasks.value
  } catch {
    // 忽略
  }
}

async function loadCheckin() {
  try {
    const res = await getCheckinStatus()
    checkin.value = res.data || checkin.value
  } catch {
    // 忽略
  }
}

onMounted(() => {
  loadShortcutKeys()
  loadRecent()
  // 手机端底栏挂载即会启动消息轮询，这里补一次让角标立刻准确
  notif.refresh()
  loadSummary()
  loadTasks()
  loadCheckin()
})
</script>

<style scoped>
.my {
  /* 手机端单列；≥1100px 换成「内容 + 侧栏」两栏（见文件末尾的 PC 媒体查询） */
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
  max-width: 720px;
  /* 平板/窄屏限宽但居中：否则 720px 的内容会贴左，右边留一条空白 */
  margin: 0 auto;
}

/* 手机端：两栏容器透明化，内部卡片直接成为外层单列栅格的项目，
   视觉顺序与 DOM 顺序一致（资料 → 消息 → 常用 → 最近 → 等级/安全 → 今日任务） */
.my-main,
.my-side {
  display: contents;
}

/* 搜索入口 */
.my-search {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border: 1px solid var(--border-soft);
  border-radius: 999px;
  background: var(--bg-card);
  color: var(--text-muted);
  font-size: 13px;
}
.my-search:hover {
  border-color: var(--primary);
  color: var(--text);
}

/* 资料卡 */
.my-profile {
  padding: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}
.my-profile-top {
  display: flex;
  align-items: center;
  gap: 12px;
}
.my-avatar-wrap {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
}
.my-avatar {
  width: 52px;
  height: 52px;
  flex-shrink: 0;
  border-radius: 50%;
  object-fit: cover;
  background: var(--bg-muted);
}
/* 头像右下角的认证角标（VerifiedBadge 的 dot 形态，尺寸随这里放大） */
.my-avatar-cert {
  position: absolute;
  right: 0;
  bottom: 0;
  z-index: 2;
  font-size: 15px;
}
.my-profile-text {
  flex: 1;
  min-width: 0;
}
.my-nickname {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  max-width: 100%;
  font-size: 17px;
  font-weight: 600;
  color: var(--text);
}
.my-nickname .bi {
  font-size: 13px;
  color: var(--text-light, var(--text-muted));
}
.my-nickname-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.my-sub {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}

.my-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--border-soft);
}
.my-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 0 4px;
  text-align: center;
  border-right: 1px solid var(--border-soft);
}
.my-stat:last-child {
  border-right: none;
}
.my-stat-label {
  font-size: 12px;
  color: var(--text-muted);
}
.my-stat-value {
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
}

/* 未读消息条 */
.my-notice {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  color: var(--text-soft);
  font-size: 13px;
}
.my-notice:hover {
  border-color: var(--primary);
}
.my-notice-icon {
  position: relative;
  font-size: 16px;
  color: var(--primary);
}
.my-notice-dot {
  position: absolute;
  top: -1px;
  right: -3px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--danger);
}
.my-notice-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.my-arrow {
  font-size: 12px;
  color: var(--text-light, var(--text-muted));
}

/* 通用卡片 */
.my-card {
  padding: 14px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
}
.my-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 12px;
}
.my-card-title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--text);
}
.my-card-sub {
  margin-left: 6px;
  font-size: 11px;
  font-weight: 400;
  color: var(--text-muted);
}
.my-link {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 12px;
  cursor: pointer;
}
.my-link:hover {
  color: var(--primary);
}

/* 常用 / 最近使用 的图标宫格 */
.my-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 4px 0;
}
.my-grid-item {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 8px 2px;
  border: none;
  background: transparent;
  color: var(--text-soft);
  font-family: inherit;
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
}
.my-grid-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: var(--bg-muted);
  color: var(--text-soft);
  font-size: 18px;
  transition: color 0.15s, background 0.15s;
}
.my-grid-item:hover .my-grid-icon {
  color: var(--primary);
  background: var(--accent-soft);
}
.my-grid-label {
  font-size: 12px;
  line-height: 1.2;
  text-align: center;
}
.my-grid-badge {
  position: absolute;
  top: 4px;
  right: calc(50% - 26px);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--danger);
  color: #fff;
  font-size: 11px;
  line-height: 1;
}
.my-grid-badge.is-add {
  background: var(--primary);
}
.my-grid-item.is-hidden {
  opacity: 0.55;
}
.my-tip {
  margin: 10px 0 6px;
  font-size: 12px;
  color: var(--text-muted);
}

/* 等级 / 账号安全：手机端固定并排两列
   （原来用 minmax(200px, 1fr)，手机可用宽度只有 ~350px，2×200 放不下 → 实际会竖排成两行） */
.my-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.my-mini {
  display: block;
  color: var(--text);
}
.my-mini-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-muted);
}
.my-mini-value {
  display: block;
  margin-top: 6px;
  font-size: 18px;
  font-weight: 700;
}
.my-mini-hint {
  margin: 8px 0 0;
  font-size: 12px;
  color: var(--text-muted);
}
.my-bar {
  height: 6px;
  margin-top: 10px;
  border-radius: 3px;
  background: var(--bg-muted);
  overflow: hidden;
}
.my-bar span {
  display: block;
  height: 100%;
  border-radius: 3px;
  background: linear-gradient(90deg, var(--primary), var(--primary-deep));
  transition: width 0.3s;
}

/* 今日任务 */
.my-tasks {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.my-task {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--text-soft);
}
.my-task .bi {
  font-size: 14px;
  color: var(--text-light, var(--text-muted));
}
.my-task.is-done {
  color: var(--text-muted);
}
.my-task.is-done .bi {
  color: var(--success, #16a34a);
}
.my-task-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.my-task-value {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--primary);
}

/* 协议入口（今日任务下方）：与「未读消息」同一套行式导航，两条合并成一组卡片 */
.my-links {
  display: grid;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  /* 让行 hover 底色被圆角裁掉，不溢出卡片外 */
  overflow: hidden;
}
.my-link-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 13px 16px;
  font-size: 14px;
  color: var(--text);
  text-decoration: none;
  transition: background 0.15s, color 0.15s;
}
/* 两条之间的分隔线（只给后一条加上边框，避免首条顶部多一条线） */
.my-link-item + .my-link-item {
  border-top: 1px solid var(--border-soft);
}
.my-link-item > .bi:first-child {
  font-size: 15px;
  color: var(--primary);
}
.my-link-item-text {
  flex: 1;
  min-width: 0;
}
.my-link-item:hover {
  background: var(--bg-muted);
  color: var(--primary);
}

/* 手机端：宫格间距略收紧，卡片内边距加大一点更好点 */
@media (max-width: 640px) {
  .my {
    max-width: none;
    gap: 10px;
  }
  .my-profile {
    padding: 14px;
  }
  .my-grid-icon {
    width: 42px;
    height: 42px;
  }
  .my-link-item {
    padding: 12px 14px;
  }
}

/* ---------- PC（≥1100px）：两栏布局 ----------
   左列：资料 / 未读消息 / 常用 / 最近使用 / 今日任务
   右列：等级、账号安全（吸顶跟随）

   宽度策略：外层 .user-content 没有宽度上限（flex: 1 铺满），所以这里必须**铺满内容区**，
   否则超出的宽度会全部堆在页面右侧（看起来像「页面没占满」）。
   上限只用来兜住 4K/带鱼屏：超过 1400px 时整页居中，左右对称留白，而不是把空白堆到一边。
   断点 1100px：此时内容区 836px，右列 320px 后左列仍有 ~500px，宫格还能排成 4 列不挤。 */
@media (min-width: 1100px) {
  .my {
    grid-template-columns: minmax(0, 1fr) 320px;
    align-items: start;
    gap: 20px;
    max-width: 1400px;
    margin: 0 auto;
  }
  .my-main {
    display: flex;
    flex-direction: column;
    gap: 18px;
    grid-column: 1;
    grid-row: 1;
    min-width: 0;
  }
  /* 右列吸顶：用户中心顶栏 position: sticky / height: 56px，留 20px 间距。
     跨 3 行是为了让吸顶范围覆盖左列全部内容（资料→…→今日任务→协议入口）。 */
  .my-side {
    display: flex;
    flex-direction: column;
    gap: 18px;
    grid-column: 2;
    grid-row: 1 / span 3;
    position: sticky;
    top: 76px;
  }
  /* 今日任务排在左列第二行 */
  .my-tail {
    grid-column: 1;
    grid-row: 2;
  }
  /* 协议入口（.my-links）不写死行列，交给栅格自动流：
     它是唯一的自动摆放项，会落到左列已有内容之后（今日任务在则第 3 行，
     今日任务为空时自然补到第 2 行）——写死 grid-row: 3 会在没任务时多留一个空行的间距。 */

  /* 鼠标端给整格 hover / 键盘 focus 反馈（触摸端没有 hover，所以只在 PC 生效） */
  .my-grid {
    /* 左列变宽后自动加列（每格不小于 120px），避免 4 列被拉成每格 250px 的空旷样；
       用 auto-fit 让「最近使用」这类少于整行数量的卡片把整行撑满，而不是右侧留空 */
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 8px 4px;
  }
  .my-grid-item {
    padding: 10px 6px;
    border-radius: 10px;
    transition: background 0.15s;
  }
  .my-grid-item:hover {
    background: var(--bg-muted);
  }
  .my-grid-item:focus-visible {
    outline: 2px solid var(--primary-soft);
    outline-offset: -2px;
  }
  .my-grid-icon {
    width: 44px;
    height: 44px;
    font-size: 19px;
  }

  /* PC 上内边距与字号略放大：左列更宽，保持原尺寸会显得空 */
  .my-profile {
    padding: 20px;
  }
  .my-avatar {
    width: 56px;
    height: 56px;
  }
  .my-nickname {
    font-size: 19px;
  }
  .my-card {
    padding: 16px;
  }
  .my-notice {
    padding: 14px 16px;
  }
  .my-mini-value {
    font-size: 20px;
  }
  /* 右栏只有 300px：两张小卡改为上下排列 */
  .my-cards {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
