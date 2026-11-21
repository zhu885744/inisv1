<template>
  <div class="manage" :class="{ 'sidebar-collapsed': collapsed }">
    <!-- 移动端遮罩：点击空白处收起侧边栏 -->
    <div v-if="collapsed" class="manage-mask" @click="collapsed = false" />
    <aside class="manage-sidebar">
      <div class="manage-brand">
        <i class="bi bi-pen" />
        <span class="manage-brand-text">创作中心</span>
      </div>

      <nav class="manage-nav">
        <RouterLink
          to="/manage"
          class="manage-nav-item"
          :class="{ 'is-active': isOverview }"
        >
          <i class="bi bi-grid-1x2" />
          <span class="manage-nav-text">概览</span>
        </RouterLink>

        <div class="manage-nav-group">内容</div>

        <RouterLink
          to="/manage/posts/write"
          class="manage-nav-item"
          :class="{ 'is-active': isWrite }"
        >
          <i class="bi bi-pencil-square" />
          <span class="manage-nav-text">写文章</span>
        </RouterLink>
        <RouterLink
          to="/manage/posts"
          class="manage-nav-item"
          :class="{ 'is-active': isPosts }"
        >
          <i class="bi bi-file-earmark-text" />
          <span class="manage-nav-text">我的文章</span>
        </RouterLink>
        <RouterLink
          to="/manage/moments"
          class="manage-nav-item"
          :class="{ 'is-active': isMoments }"
        >
          <i class="bi bi-chat-square-dots" />
          <span class="manage-nav-text">我的动态</span>
        </RouterLink>
        <RouterLink
          to="/manage/links"
          class="manage-nav-item"
          :class="{ 'is-active': isLinks }"
        >
          <i class="bi bi-link-45deg" />
          <span class="manage-nav-text">友链管理</span>
        </RouterLink>
      </nav>

      <div class="manage-sidebar-footer">
        <RouterLink to="/user" class="manage-nav-item">
          <i class="bi bi-person" />
          <span class="manage-nav-text">用户中心</span>
        </RouterLink>
        <RouterLink v-if="isAdmin" to="/admin" class="manage-nav-item">
          <i class="bi bi-shield-lock" />
          <span class="manage-nav-text">管理后台</span>
        </RouterLink>
        <RouterLink to="/" class="manage-nav-item">
          <i class="bi bi-box-arrow-left" />
          <span class="manage-nav-text">返回前台</span>
        </RouterLink>
      </div>
    </aside>

    <div class="manage-main">
      <header class="manage-topbar">
        <button class="manage-toggle" type="button" title="菜单" @click="collapsed = !collapsed">
          <i class="bi bi-list" />
        </button>
        <div class="manage-topbar-title">{{ currentTitle }}</div>
        <div class="manage-topbar-right">
          <!-- 用户菜单：头像 + 昵称收进下拉（个人资料 / 返回前台 / 退出登录） -->
          <div class="manage-menu" ref="userMenuRef">
            <button
              class="manage-menu-btn"
              :class="{ 'is-open': userMenuOpen }"
              type="button"
              aria-haspopup="menu"
              :aria-expanded="userMenuOpen"
              @click="userMenuOpen = !userMenuOpen"
            >
              <img :src="user?.avatar || defaultAvatar" class="manage-avatar" alt="avatar" loading="lazy" />
              <span class="manage-user-name">{{ user?.nickname || '未登录' }}</span>
              <i class="bi bi-chevron-down manage-menu-caret" />
            </button>

            <div v-if="userMenuOpen" class="manage-menu-panel" role="menu">
              <RouterLink to="/user/profile" class="manage-menu-item" @click="userMenuOpen = false">
                <i class="bi bi-person" />
                <span>个人资料</span>
              </RouterLink>
              <RouterLink to="/" class="manage-menu-item" @click="userMenuOpen = false">
                <i class="bi bi-house" />
                <span>返回前台</span>
              </RouterLink>
              <div class="manage-menu-divider" />
              <button
                class="manage-menu-item is-danger"
                type="button"
                :disabled="loggingOut"
                @click="onLogout"
              >
                <i class="bi bi-box-arrow-right" />
                <span>{{ loggingOut ? '退出中…' : '退出登录' }}</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      <main class="manage-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/stores/user'
import { canEnterAdmin as helperCanEnterAdmin } from '@/utils/helper'

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const route = useRoute()
const userStore = useUserStore()
const { user } = storeToRefs(userStore)

const collapsed = ref(false)

// 有任意后台页面权限（或超管 / 组 key=admin）时显示后台入口
const isAdmin = computed(() => helperCanEnterAdmin(user.value))

// 精确判断高亮：写文章/编辑 与 我的文章 路径有前缀重叠，不能用 active-class
const isWrite = computed(() => /^\/manage\/posts\/(write|edit)/.test(route.path))
const isPosts = computed(() => route.path === '/manage/posts')
const isMoments = computed(() => route.path.startsWith('/manage/moments'))
const isLinks = computed(() => route.path.startsWith('/manage/links'))
const isOverview = computed(() => route.path === '/manage' || route.path === '/manage/')

const currentTitle = computed(() => {
  if (isWrite.value) return '写文章'
  if (isPosts.value) return '我的文章'
  if (isMoments.value) return '我的动态'
  if (isLinks.value) return '友链管理'
  return '创作中心'
})

// ---------- 顶栏用户下拉 ----------
const userMenuOpen = ref(false)
const userMenuRef = ref(null)
const loggingOut = ref(false)

// 移动端：路由变化后自动收起抽屉（桌面端保持用户选择的折叠状态不变）
watch(() => route.path, () => {
  userMenuOpen.value = false
  if (window.innerWidth <= 768) collapsed.value = false
})

// 点击菜单外部 / 按 Esc 关闭下拉
function onDocClick(e) {
  if (!userMenuOpen.value) return
  if (userMenuRef.value && !userMenuRef.value.contains(e.target)) userMenuOpen.value = false
}
function onDocKeydown(e) {
  if (e.key === 'Escape') userMenuOpen.value = false
}

// 退出登录：清登录态 + 清 token + 回首页都在 store 里完成（与用户中心、后台同一入口）
async function onLogout() {
  userMenuOpen.value = false
  if (loggingOut.value) return
  loggingOut.value = true
  try {
    await userStore.logout()
  } finally {
    loggingOut.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onDocKeydown)
})

onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocKeydown)
})
</script>

<style scoped>
.manage {
  display: flex;
  min-height: 100vh;
  background: var(--bg);
}
.manage-sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--bg-card);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  position: sticky;
  top: 0;
  height: 100vh;
  transition: width 0.2s;
}
.manage-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 18px;
  border-bottom: 1px solid var(--border-soft);
  font-size: 16px;
  font-weight: 700;
  color: var(--text);
  white-space: nowrap;
  overflow: hidden;
}
/* 品牌图标做成小胶囊：与导航图标拉开层级，折叠时仍是一个完整的视觉块 */
.manage-brand i {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  border-radius: var(--radius-sm);
  background: var(--accent-soft);
  color: var(--primary);
  font-size: 15px;
}
.manage-nav {
  flex: 1;
  padding: 10px 10px 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
}
.manage-nav-group {
  padding: 12px 12px 6px;
  font-size: 11px;
  letter-spacing: 0.08em;
  color: var(--text-muted);
}
.manage-nav-item {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: var(--radius-sm);
  color: var(--text-soft);
  text-decoration: none;
  font-size: 14px;
  white-space: nowrap;
  transition: background-color var(--dur) var(--ease), color var(--dur) var(--ease);
}
/* 图标统一宽度：不然 bi 图标宽窄不一，文字左边缘会参差 */
.manage-nav-item i {
  width: 18px;
  flex-shrink: 0;
  text-align: center;
  font-size: 15px;
}
.manage-nav-item:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.manage-nav-item.is-active {
  background: var(--accent-soft);
  color: var(--primary);
  font-weight: 600;
}
/* 当前位置指示条：比只靠底色更明确，也避免折叠态看不出选中 */
.manage-nav-item.is-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  width: 3px;
  height: 16px;
  margin-top: -8px;
  border-radius: 0 2px 2px 0;
  background: var(--primary);
}
.manage-sidebar-footer {
  padding: 12px;
  border-top: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.manage-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.manage-topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 14px;
  height: 56px;
  padding: 0 20px;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border);
}
.manage-toggle {
  display: none;
  border: none;
  background: transparent;
  font-size: 20px;
  color: var(--text);
  cursor: pointer;
}
.manage-topbar-title {
  font-size: 16px;
  font-weight: 600;
}
.manage-topbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}
.manage-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  object-fit: cover;
}
.manage-user-name {
  font-size: 14px;
  color: var(--text-soft);
}

/* ---------- 顶栏用户下拉（与用户中心、后台同一套观感，圆角统一 4px） ---------- */
.manage-menu {
  position: relative;
}
.manage-menu-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 3px 8px;
  border: 1px solid transparent;
  border-radius: 4px;
  background: transparent;
  color: var(--text-soft);
  font-size: 14px;
  font-family: inherit;
  cursor: pointer;
  transition: color 0.18s, border-color 0.18s, background 0.18s;
}
.manage-menu-btn:hover,
.manage-menu-btn.is-open {
  color: var(--text);
  border-color: var(--border);
  background: var(--bg-muted);
}
.manage-menu-caret {
  font-size: 11px;
  color: var(--text-light);
  transition: transform 0.18s;
}
.manage-menu-btn.is-open .manage-menu-caret {
  transform: rotate(180deg);
}
.manage-menu-panel {
  position: absolute;
  right: 0;
  top: calc(100% + 8px);
  z-index: 20;
  min-width: 172px;
  padding: 4px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 4px;
  box-shadow: var(--shadow-md);
}
.manage-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 10px;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--text-soft);
  font-size: 14px;
  font-family: inherit;
  line-height: 1.4;
  text-align: left;
  text-decoration: none;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.manage-menu-item i {
  font-size: 15px;
  color: var(--text-light);
  transition: color 0.15s;
}
.manage-menu-item:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.manage-menu-item:hover i {
  color: var(--primary);
}
.manage-menu-item.is-danger {
  color: var(--danger);
}
.manage-menu-item.is-danger i {
  color: var(--danger);
}
.manage-menu-item.is-danger:hover {
  background: rgba(217, 84, 77, 0.1);
  color: var(--danger);
}
.manage-menu-item.is-danger:hover i {
  color: var(--danger);
}
.manage-menu-item:disabled {
  opacity: 0.6;
  cursor: default;
}
.manage-menu-divider {
  height: 1px;
  margin: 4px 0;
  background: var(--border-soft);
}
.manage-content {
  /* 大屏下别把卡片拉成一整条：限宽居中，与前台阅读宽度观感一致 */
  width: 100%;
  max-width: 1280px;
  margin: 0 auto;
  padding: 22px;
  flex: 1;
}

/* 移动端遮罩：桌面端不显示，仅窄屏通过媒体查询启用 */
.manage-mask {
  display: none;
}
.sidebar-collapsed .manage-sidebar {
  width: 64px;
}
.sidebar-collapsed .manage-brand-text,
.sidebar-collapsed .manage-nav-text,
.sidebar-collapsed .manage-nav-group {
  display: none;
}
/* 折叠态只留图标，品牌区要与导航图标的中心线对齐 */
.sidebar-collapsed .manage-brand {
  justify-content: center;
  gap: 0;
  padding: 16px 0;
}
.sidebar-collapsed .manage-nav-item {
  justify-content: center;
  padding: 10px;
}

@media (max-width: 768px) {
  .manage-mask {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 25;
    background: rgba(0, 0, 0, 0.4);
  }
  .manage-sidebar {
    position: fixed;
    z-index: 30;
    height: 100vh;
    transform: translateX(-100%);
    transition: transform 0.25s;
  }
  .sidebar-collapsed .manage-sidebar {
    transform: translateX(0);
    width: 220px;
  }
  .sidebar-collapsed .manage-brand-text,
  .sidebar-collapsed .manage-nav-text,
  .sidebar-collapsed .manage-nav-group {
    display: inline;
  }
  .sidebar-collapsed .manage-nav-item {
    justify-content: flex-start;
    padding: 10px 12px;
  }
  .manage-toggle {
    display: inline-flex;
  }
  .manage-content {
    padding: 14px;
  }
  .manage-topbar {
    padding: 0 14px;
    gap: 10px;
  }
}

/* 超窄屏（多数手机）：顶栏只留头像，避免昵称把标题挤成两行 */
@media (max-width: 560px) {
  .manage-user-name,
  .manage-menu-caret {
    display: none;
  }
  .manage-menu-btn {
    padding: 3px;
  }
}
</style>
