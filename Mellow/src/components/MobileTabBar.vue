<template>
  <!-- 移动端底部 Tab Bar（仅手机端显示，见样式里的媒体查询） -->
  <nav class="mobile-tabbar" aria-label="主导航">
    <button
      v-for="tab in tabs"
      :key="tab.key"
      type="button"
      class="tab-item"
      :class="{
        'is-active': isActive(tab),
        highlight: tab.highlight,
        'is-open': tab.action === 'publish' && publishOpen
      }"
      :aria-label="tab.ariaLabel || tab.label"
      :aria-current="isActive(tab) ? 'page' : undefined"
      :aria-expanded="tab.action === 'publish' ? publishOpen : undefined"
      @click="onTabClick(tab)"
    >
      <span v-if="tab.badge" class="tab-badge">{{ tab.badge }}</span>
      <i class="bi tab-icon" :class="tab.icon" />
      <!-- label 为空时整行不渲染（发布 tab 只留圆形按钮，见 ariaLabel） -->
      <span v-if="tab.label" class="tab-label">{{ tab.label }}</span>
    </button>
  </nav>

  <!-- 发布面板遮罩：点空白收起（z-index 低于 Tab Bar，方便直接切其它 tab） -->
  <Transition name="publish-fade">
    <div v-if="publishOpen" class="publish-mask" @click="publishOpen = false" />
  </Transition>

  <!-- 发布面板：中间「发布」按钮展开，发文章 / 发动态 -->
  <Transition name="publish-pop">
    <div v-if="publishOpen" class="publish-panel" role="menu" aria-label="发布">
      <button
        v-for="item in PUBLISH_ACTIONS"
        :key="item.key"
        type="button"
        class="publish-item"
        role="menuitem"
        @click="goPublish(item)"
      >
        <span class="publish-item-icon"><i :class="item.icon" aria-hidden="true" /></span>
        <span class="publish-item-text">
          <strong>{{ item.label }}</strong>
          <small>{{ item.desc }}</small>
        </span>
      </button>
    </div>
  </Transition>

  <!-- 发动态：动态列表页已下线，改用首页同一个发布弹窗（components/MomentComposeDialog.vue） -->
  <MomentComposeDialog v-model:visible="composeOpen" />
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import MomentComposeDialog from './MomentComposeDialog.vue'

/**
 * 移动端底部 Tab Bar
 *
 * 为什么抽成独立组件：用户中心（views/user/*）挂在 views/user/Layout.vue 下，
 * 与 MainLayout 是两套壳。底栏原先只写在 MainLayout 里，导致点进「我的」后底栏消失、
 * 只能靠抽屉返回。抽出来后两套壳共用同一份底栏，行为完全一致。
 *
 * 未读角标与消息轮询也放在这里：底栏是角标唯一的展示位置，
 * 组件一挂载就保证有轮询（startPolling 内部会先清掉旧定时器，重复调用是安全的）。
 */
const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const notif = useNotificationStore()
const { count: notifCount } = storeToRefs(notif)

// 消息未读角标：未登录不显示；超过 99 显示 99+
const unreadBadge = computed(() => {
  if (!userStore.isLogged) return ''
  const total = Number(notifCount.value) || 0
  if (total <= 0) return ''
  return total > 99 ? '99+' : String(total)
})

// 底部 Tab 配置
const tabs = computed(() => [
  { key: 'home', to: '/', icon: 'bi-house-door-fill', label: '首页', exact: true },
  { key: 'goods', to: '/goods', icon: 'bi-shop', label: '商城' },
  // 发布按钮：不直接跳转，点击展开「发文章 / 发动态」面板
  { key: 'publish', to: '/manage/posts/write', icon: 'bi-plus-circle-fill', label: '', ariaLabel: '发布', needLogin: true, highlight: true, action: 'publish' },
  // badge 响应式取消息未读数
  { key: 'notif', to: '/notifications', icon: 'bi-bell-fill', label: '消息', needLogin: true, badge: unreadBadge.value },
  // 「我的」进用户中心聚合页；个人主页从页内昵称进入（/author/:id）
  { key: 'mine', to: '/user', icon: 'bi-person-fill', label: '我的', needLogin: true }
])

// 当前选中的 tab（唯一）：取「匹配得最具体」的那个。
//
// 不能只判前缀：「我的」的 to 是 /user，用户中心各子页（/user/integral 等）都落在它下面，
// 需要保持「我的」高亮；而「消息」现在是独立页面 /notifications，只按自身命中。
// 规则：先看精确命中（带 exact 的 tab，如首页），否则在命中的 tab 里取 to 最长的一个：
//   /notifications → 消息
//   /user/integral → 我的
const activeKey = computed(() => {
  const p = route.path
  let best = null
  for (const tab of tabs.value) {
    if (tab.exact) {
      if (p === tab.to) return tab.key
      continue
    }
    if (p !== tab.to && !p.startsWith(tab.to + '/')) continue
    if (!best || tab.to.length > best.to.length) best = tab
  }
  return best ? best.key : null
})

function isActive(tab) {
  return activeKey.value === tab.key
}

function goTab(tab) {
  if (tab.needLogin && !userStore.isLogged) {
    router.push('/auth/login')
    return
  }
  router.push(tab.to)
}

// 中间「发布」按钮：展开面板；其余 tab 正常跳转
function onTabClick(tab) {
  if (tab.action === 'publish') {
    togglePublish()
    return
  }
  goTab(tab)
}

// 发布面板：发文章 / 发动态（两个入口都需要登录）
const publishOpen = ref(false)
// 「发动态」不再跳页面（动态列表页已下线），就地打开发布弹窗
const composeOpen = ref(false)
const PUBLISH_ACTIONS = [
  { key: 'post', label: '发文章', desc: '写一篇图文文章', icon: 'bi-pencil-square', to: '/manage/posts/write' },
  { key: 'moment', label: '发动态', desc: '分享此刻的想法', icon: 'bi-chat-square-dots', action: 'compose' }
]

function togglePublish() {
  // 与其它 tab 的 needLogin 口径一致：未登录直接去登录页
  if (!userStore.isLogged) {
    router.push('/auth/login')
    return
  }
  publishOpen.value = !publishOpen.value
}

function goPublish(item) {
  publishOpen.value = false
  if (item.action === 'compose') {
    composeOpen.value = true
    return
  }
  router.push(item.to)
}

// 跳走（含点面板里的选项、切其它 tab）后自动收起
watch(
  () => route.path,
  () => {
    publishOpen.value = false
  }
)

// 未读角标依赖消息轮询：挂载即保证在跑
onMounted(() => {
  if (userStore.isLogged) notif.startPolling()
})

// 登录 / 登出时启停轮询
watch(
  () => userStore.isLogged,
  (v) => {
    if (v) notif.startPolling()
    else notif.stopPolling()
  }
)

// 切换页面时补一次刷新：轮询间隔 60 秒，光靠定时器新消息最久要等一分钟才出角标。
// 节流 30 秒，避免连续点导航时每次切页都发一轮请求。
let lastNotifRefresh = 0
watch(
  () => route.path,
  () => {
    if (!userStore.isLogged) return
    const now = Date.now()
    if (now - lastNotifRefresh < 30 * 1000) return
    lastNotifRefresh = now
    notif.refresh()
  }
)
</script>

<style scoped>
/* 默认隐藏（宽屏不显示） */
.mobile-tabbar {
  display: none;
}
.tab-item {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  padding: 0 2px;
  color: var(--text-muted);
  font-family: inherit;
  cursor: pointer;
  transition: color var(--dur) var(--ease);
  -webkit-tap-highlight-color: transparent;
  user-select: none;
}
.tab-icon {
  font-size: 20px;
  line-height: 1;
  /* 发布面板展开时「+」会转成「×」，这里给个过渡 */
  transition: transform var(--dur) var(--ease);
}
.tab-label {
  font-size: 10px;
  line-height: 1;
  letter-spacing: 0.06em;
}
.tab-badge {
  position: absolute;
  top: 2px;
  left: calc(50% + 6px);
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  background: var(--danger);
  color: #fff;
  font-size: 10px;
  line-height: 16px;
  text-align: center;
  font-weight: 600;
  border: 1.5px solid var(--bg-card);
  box-sizing: border-box;
}
.tab-item.is-active {
  color: var(--primary);
}
.tab-item.is-active .tab-icon {
  transform: translateY(-1px);
}
.tab-item.is-active .tab-label {
  font-weight: 600;
}
/* 中间「发布」按钮：圆形大按钮（无文字标签，在格子内垂直居中） */
.tab-item.highlight {
  justify-content: center;
  /* 去掉格子的上下内边距：内容区正好是 56px 底栏高度，46px 圆钮能完整放下。
     否则内容区只剩 40px（8+8 内边距），而 column 方向上 height 是主轴尺寸，
     默认 flex-shrink: 1 会把圆钮压成 46×40 的椭圆（宽度方向有 align-items: center
     不参与收缩，所以只扁了高度）。 */
  padding: 0;
}
.tab-item.highlight .tab-icon {
  width: 46px;
  height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  /* 原来靠 margin: -22px 0 4px 把圆钮凸出栏外并给下方标签留位，
     现在没有标签了，负边距只会让它贴着格子上沿 —— 去掉，交给 justify-content 居中 */
  margin: 0;
  /* 双保险：主轴方向禁止收缩，任何内边距/高度变化下都保持 46×46 的正圆 */
  flex-shrink: 0;
  font-size: 26px;
  color: #fff;
  background: linear-gradient(135deg, var(--primary), var(--primary-deep));
  border-radius: 50%;
  border: 3px solid var(--bg-card);
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.18);
}
/* 发布面板展开时：中间按钮的「+」转 45° 变「×」 */
.tab-item.is-open .tab-icon {
  transform: rotate(45deg);
}

/* ---------- 发布面板（发文章 / 发动态） ---------- */
.publish-mask {
  position: fixed;
  inset: 0;
  z-index: 150;
  background: rgba(0, 0, 0, 0.28);
}
.publish-panel {
  position: fixed;
  left: 12px;
  right: 12px;
  /* Dock 顶边（10 + 58）+ 凸出的圆钮（10）+ 间隙，落在圆钮上方 */
  bottom: calc(86px + env(safe-area-inset-bottom));
  z-index: 202;
  /* 两个入口一行两个（原来 flex column 各占一行，手机上显得很空）。
     用 minmax(0, 1fr) 而不是 1fr：避免长文案把列撑宽。 */
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  padding: 6px;
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  background: var(--bg-card);
  box-shadow: var(--shadow-md);
}
.publish-item {
  /* 半宽格子里改为竖排：图标在上、文字在下居中。
     横排（图标 + 两行文字）在 ~160px 宽时会把「写一篇图文文章」这类描述挤掉。 */
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 12px 8px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--text);
  font-family: inherit;
  text-align: center;
  cursor: pointer;
  transition: background-color 0.16s;
}
.publish-item:hover,
.publish-item:active {
  background: var(--bg-muted);
}
.publish-item-icon {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 4px;
  background: var(--accent-soft);
  color: var(--primary-deep);
  font-size: 18px;
}
.publish-item-text {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  min-width: 0;
}
.publish-item-text strong {
  font-size: 14px;
  font-weight: 600;
}
.publish-item-text small {
  font-size: 12px;
  color: var(--text-muted);
}
/* 入场 / 收起动效 */
.publish-pop-enter-active,
.publish-pop-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}
.publish-pop-enter-from,
.publish-pop-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
.publish-fade-enter-active,
.publish-fade-leave-active {
  transition: opacity 0.18s ease;
}
.publish-fade-enter-from,
.publish-fade-leave-to {
  opacity: 0;
}
/* 宽屏（Tab Bar 不显示）时面板与遮罩一并隐藏：避免从手机端拉宽后残留 */
@media (min-width: 641px) {
  .publish-mask,
  .publish-panel {
    display: none;
  }
}

/* ---------- 手机端：悬浮 Dock ----------
 * 胶囊形 + 半透明毛玻璃，浮在内容之上（不再贴屏幕底边）；
 * 中间「发布」是独立圆钮，向上凸出 Dock 上沿。
 * 页面底部让位：MainLayout / user/Layout 的页脚、Profile 的保存条都按
 * 「Dock 高 58 + 下边距 10 + 间隙」预留，改动尺寸时一并调整。
 */
@media (max-width: 640px) {
  .mobile-tabbar {
    position: fixed;
    left: 50%;
    bottom: calc(10px + env(safe-area-inset-bottom));
    z-index: 200;
    display: flex;
    align-items: center;
    width: calc(100% - 24px);
    max-width: 460px;
    height: 58px;
    padding: 0 6px;
    transform: translateX(-50%);
    background: var(--bg-overlay);
    border: 1px solid var(--border-soft);
    border-radius: 999px;
    box-shadow: 0 10px 28px rgba(40, 34, 26, 0.16), 0 2px 6px rgba(40, 34, 26, 0.05);
    backdrop-filter: blur(18px) saturate(160%);
    -webkit-backdrop-filter: blur(18px) saturate(160%);
  }
  .tab-item {
    height: 100%;
  }
  /* 中间发布键：凸出上沿 12px，白圈把圆钮和毛玻璃 Dock 分开 */
  .tab-item.highlight {
    flex: 0 0 auto;
    width: 54px;
    height: 54px;
    margin: 0 2px;
    padding: 0;
    transform: translateY(-12px);
  }
  .tab-item.highlight .tab-icon {
    width: 54px;
    height: 54px;
    margin: 0;
    font-size: 25px;
    border-radius: 50%;
    border: 3px solid var(--bg-card);
    box-shadow: 0 6px 16px var(--accent-glow);
  }
  /* 角标贴着图标右上角，位置随 Dock 高度收紧 */
  .tab-badge {
    top: 9px;
    left: calc(50% + 8px);
  }
}
</style>
