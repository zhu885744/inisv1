<template>
  <div class="admin" :class="{ 'sidebar-collapsed': collapsed }">
    <!-- 移动端遮罩：点击空白处收起侧边栏 -->
    <div v-if="collapsed" class="admin-mask" @click="collapsed = false" />
    <aside class="admin-sidebar">
      <div class="admin-brand">
        <i class="bi bi-shield-lock" />
        <span class="admin-brand-text">Mellow 管理后台</span>
      </div>

      <nav class="admin-nav" v-if="menus.length || authReady">
        <!-- 概览 + 置顶入口（/admin/stats 等）：不进创作 / 管理 / 安全分组 -->
        <div class="admin-nav-pinned">
          <RouterLink
            to="/admin/dashboard"
            class="admin-nav-item"
            active-class="is-active"
          >
            <i class="bi bi-speedometer2" />
            <span class="admin-nav-text">概览</span>
          </RouterLink>
          <RouterLink
            v-for="item in pinnedItems"
            :key="item.path"
            :to="item.path"
            class="admin-nav-item"
            active-class="is-active"
          >
            <span class="admin-nav-icon" v-if="item.isSvg" v-html="item.svg" />
            <i v-else :class="item.icon" />
            <span class="admin-nav-text">{{ item.name }}</span>
          </RouterLink>
        </div>
        <template v-for="group in menus" :key="group.name">
          <div class="admin-nav-group">{{ group.label }}</div>
          <RouterLink
            v-for="child in group.children"
            :key="child.path"
            :to="child.path"
            class="admin-nav-item"
            active-class="is-active"
          >
            <span class="admin-nav-icon" v-if="child.isSvg" v-html="child.icon" />
            <i v-else :class="child.icon || 'bi bi-circle'" />
            <span class="admin-nav-text">{{ child.name }}</span>
          </RouterLink>
        </template>
      </nav>
      <div class="admin-nav-empty" v-else-if="!authReady">加载菜单中…</div>

      <div class="admin-sidebar-footer">
        <RouterLink to="/" class="admin-nav-item">
          <i class="bi bi-box-arrow-left" />
          <span class="admin-nav-text">返回前台</span>
        </RouterLink>
      </div>
    </aside>

    <div class="admin-main">
      <header class="admin-topbar">
        <button class="admin-toggle" type="button" @click="collapsed = !collapsed">
          <i class="bi bi-list" />
        </button>
        <div class="admin-topbar-title">{{ currentTitle }}</div>
        <div class="admin-topbar-right">
          <!-- 浅色 / 深色模式切换：与前台悬浮按钮共用 theme store（localStorage 持久化，刷新后保持） -->
          <button
            class="admin-icon-btn"
            type="button"
            :title="isDark ? '切换为浅色模式' : '切换为深色模式'"
            :aria-label="isDark ? '切换为浅色模式' : '切换为深色模式'"
            @click="toggleTheme"
          >
            <i :class="isDark ? 'bi bi-sun' : 'bi bi-moon-stars'" />
          </button>

          <!-- 用户菜单：头像 + 昵称收进下拉（个人资料 / 返回前台 / 退出登录） -->
          <div class="admin-user" ref="userMenuRef">
            <button
              class="admin-user-btn"
              :class="{ 'is-open': userMenuOpen }"
              type="button"
              aria-haspopup="menu"
              :aria-expanded="userMenuOpen"
              @click="userMenuOpen = !userMenuOpen"
            >
              <img :src="user?.avatar || defaultAvatar" class="admin-avatar" alt="avatar" loading="lazy" />
              <span class="admin-user-name">{{ user?.nickname || '未登录' }}</span>
              <i class="bi bi-chevron-down admin-user-caret" />
            </button>

            <div v-if="userMenuOpen" class="admin-user-menu" role="menu">
              <RouterLink to="/user/profile" class="admin-menu-item" @click="userMenuOpen = false">
                <i class="bi bi-person" />
                <span>个人资料</span>
              </RouterLink>
              <RouterLink to="/" class="admin-menu-item" @click="userMenuOpen = false">
                <i class="bi bi-house" />
                <span>返回前台</span>
              </RouterLink>
              <div class="admin-menu-divider" />
              <button
                class="admin-menu-item is-danger"
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

      <main class="admin-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { storeToRefs } from 'pinia'
import { useUserStore } from '@/stores/user'
import { useAuthPagesStore } from '@/stores/auth-pages'
import { useThemeStore } from '@/stores/theme'
import { canEnterAdmin } from '@/utils/helper'
import { buildAdminMenus, buildAdminPinnedItems, adminMenuOfPath, adminPathAllowed, rememberAdminPath } from '@/utils/admin-menu'

const defaultAvatar = 'data:image/svg+xml;utf8,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 80 80"><circle cx="40" cy="40" r="40" fill="%23e8e6dd"/><text x="50%25" y="55%25" text-anchor="middle" font-size="36" fill="%238a8a82" font-family="serif">用</text></svg>'

const route = useRoute()
const router = useRouter()
const userStore = useUserStore()
const { user } = storeToRefs(userStore)
const authPagesStore = useAuthPagesStore()
const themeStore = useThemeStore()

const collapsed = ref(false)
const authReady = ref(false)

// ---------- 顶栏：主题切换 + 用户下拉 ----------
const isDark = computed(() => themeStore.isDark)

// 后台只要两态（浅色 ⇄ 深色）：点一次即固定下来，不再走前台的 auto 循环
function toggleTheme() {
  themeStore.set(isDark.value ? 'light' : 'dark')
}

const userMenuOpen = ref(false)
const userMenuRef = ref(null)
const loggingOut = ref(false)

// 点击菜单外部 / 按 Esc 关闭下拉
function onDocClick(e) {
  if (!userMenuOpen.value) return
  if (userMenuRef.value && !userMenuRef.value.contains(e.target)) userMenuOpen.value = false
}
function onDocKeydown(e) {
  if (e.key === 'Escape') userMenuOpen.value = false
}

// 退出登录：清登录态 + 清 token + 回首页都在 store 里完成
// （「退出后必须落在 / 而不是被 401 抢去登录页」的修复见 stores/user.js 与 utils/authFlow.js）
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

// 菜单：分组顺序 / 权限过滤 / 图标解析统一在 utils/admin-menu.js（概览页快捷跳转共用同一份）
const menus = computed(() => buildAdminMenus(user.value, authPagesStore.getFlat || []))

// 置顶入口：固定在「概览」下方，不参与分组（同样按权限过滤）
const pinnedItems = computed(() => buildAdminPinnedItems(user.value, authPagesStore.getFlat || []))

const currentTitle = computed(() => route.meta?.title || '后台管理')

// 权限守卫：
// - 「能不能进后台」由 router 的 meta.requiresAdmin 拦截（会先回后端确认一次权限），
//   这里不再用 setup 时的一次性判断 —— 那时 user 可能还没拉回来，判断没有意义；
// - 这里只兜「已经进入后台、权限又被实时撤销」的情况：check-token 刷新后 user 变化，
//   没有后台页面权限就退回首页。
// （判定见 utils/helper.js 的 canEnterAdmin：超管 / 组 key=admin / 勾了任意后台页面）
watch(
  user,
  (u) => {
    if (u && !canEnterAdmin(u)) router.replace('/')
  },
  { immediate: true }
)

// 记录「最近访问」：把 /admin/article/edit/3、/admin/badges 这类子路由归一到一级菜单
// （/admin/article、/admin/users），供概览页快捷跳转的「最近访问」使用
function rememberVisited(path = route.path) {
  const menu = adminMenuOfPath(path)
  if (menu) rememberAdminPath(menu.path)
}

/**
 * 路由级页面校验：手输 URL 打开没有权限的后台页面时，跳到第一个可见页面
 * （/admin/dashboard 是概览，不在页面权限清单里，能进后台即可访问）
 *
 * 权限按「一级模块」判定：模块下任意一个页面被勾选即可访问该模块的所有标签页
 * （见 utils/admin-menu.js 的 adminPathAllowed）
 */
function guardRoute(path = route.path) {
  if (!path.startsWith('/admin/') || path === '/admin/dashboard') return

  const flat = authPagesStore.getFlat || []
  if (!flat.length) return

  if (adminPathAllowed(user.value, flat, path)) return

  // 兜底落地页：第一个可见分组 → 置顶入口 → 概览
  const first = menus.value[0]?.children?.[0]?.path || pinnedItems.value[0]?.path || '/admin/dashboard'
  router.replace(first)
}

onMounted(async () => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onDocKeydown)
  await authPagesStore.ensureLoaded()
  authReady.value = true
  guardRoute()
  rememberVisited()
})

onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onDocKeydown)
})

// 路由变化：校验页面权限；记录最近访问；关闭用户下拉；移动端顺带自动收起抽屉（桌面端保持用户选择的折叠状态）
watch(() => route.path, (path) => {
  guardRoute(path)
  rememberVisited(path)
  userMenuOpen.value = false
  if (window.innerWidth <= 768) collapsed.value = false
})
</script>

<style scoped>
.admin {
  display: flex;
  min-height: 100vh;
  background: var(--bg);
}
.admin-sidebar {
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
.admin-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 18px 20px;
  font-size: 17px;
  font-weight: 700;
  color: var(--primary);
  white-space: nowrap;
  overflow: hidden;
}
.admin-nav {
  flex: 1;
  padding: 6px 12px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
}
.admin-nav-group {
  padding: 14px 12px 6px;
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.admin-nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  color: var(--text-soft);
  text-decoration: none;
  font-size: 14px;
  white-space: nowrap;
  transition: all 0.15s;
}
.admin-nav-item:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.admin-nav-item.is-active {
  background: var(--accent-soft);
  color: var(--primary);
  font-weight: 600;
}
/* 概览 + 置顶入口：紧贴在一起（方角），与下方分组拉开间距 */
.admin-nav-pinned {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-bottom: 6px;
}
.admin-nav-pinned .admin-nav-item {
  border-radius: 0;
}
.admin-nav-icon {
  width: 16px;
  height: 16px;
  display: inline-flex;
}
.admin-nav-icon :deep(svg) {
  width: 16px;
  height: 16px;
}
.admin-nav-empty {
  padding: 20px 12px;
  color: var(--text-muted);
  font-size: 13px;
}
.admin-sidebar-footer {
  padding: 12px;
  border-top: 1px solid var(--border);
}
.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.admin-topbar {
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
.admin-toggle {
  display: none;
  border: none;
  background: transparent;
  font-size: 20px;
  color: var(--text);
  cursor: pointer;
}
.admin-topbar-title {
  font-size: 16px;
  font-weight: 600;
}
.admin-topbar-right {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}
.admin-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  object-fit: cover;
}
.admin-user-name {
  font-size: 14px;
  color: var(--text-soft);
}

/* ---------- 顶栏图标按钮（主题切换） ---------- */
.admin-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--bg-card);
  color: var(--text-soft);
  font-size: 15px;
  line-height: 1;
  cursor: pointer;
  transition: color 0.18s, border-color 0.18s, background 0.18s;
}
.admin-icon-btn:hover {
  color: var(--primary);
  border-color: var(--primary-soft);
  background: var(--bg-muted);
}
.admin-icon-btn:focus-visible {
  outline: 2px solid var(--primary-soft);
  outline-offset: 2px;
}

/* ---------- 顶栏用户下拉 ---------- */
.admin-user {
  position: relative;
}
.admin-user-btn {
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
.admin-user-btn:hover,
.admin-user-btn.is-open {
  color: var(--text);
  border-color: var(--border);
  background: var(--bg-muted);
}
.admin-user-caret {
  font-size: 11px;
  color: var(--text-light);
  transition: transform 0.18s;
}
.admin-user-btn.is-open .admin-user-caret {
  transform: rotate(180deg);
}
.admin-user-menu {
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
.admin-menu-item {
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
.admin-menu-item i {
  font-size: 15px;
  color: var(--text-light);
  transition: color 0.15s;
}
.admin-menu-item:hover {
  background: var(--bg-muted);
  color: var(--text);
}
.admin-menu-item:hover i {
  color: var(--primary);
}
.admin-menu-item.is-danger {
  color: var(--danger);
}
.admin-menu-item.is-danger i {
  color: var(--danger);
}
.admin-menu-item.is-danger:hover {
  background: rgba(217, 84, 77, 0.1);
  color: var(--danger);
}
.admin-menu-item.is-danger:hover i {
  color: var(--danger);
}
.admin-menu-item:disabled {
  opacity: 0.6;
  cursor: default;
}
.admin-menu-divider {
  height: 1px;
  margin: 4px 0;
  background: var(--border-soft);
}
.admin-content {
  padding: 22px;
  flex: 1;
}

/* 移动端遮罩：桌面端不显示，仅窄屏通过媒体查询启用 */
.admin-mask {
  display: none;
}
.sidebar-collapsed .admin-sidebar {
  width: 64px;
}
.sidebar-collapsed .admin-brand-text,
.sidebar-collapsed .admin-nav-text,
.sidebar-collapsed .admin-nav-group {
  display: none;
}
.sidebar-collapsed .admin-nav-item {
  justify-content: center;
  padding: 10px;
}

@media (max-width: 768px) {
  .admin-mask {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 25;
    background: rgba(0, 0, 0, 0.4);
  }
  .admin-sidebar {
    position: fixed;
    z-index: 30;
    height: 100vh;
    transform: translateX(-100%);
    transition: transform 0.25s;
  }
  .sidebar-collapsed .admin-sidebar {
    transform: translateX(0);
    width: 220px;
  }
  .sidebar-collapsed .admin-brand-text,
  .sidebar-collapsed .admin-nav-text,
  .sidebar-collapsed .admin-nav-group {
    display: inline;
  }
  .sidebar-collapsed .admin-nav-item {
    justify-content: flex-start;
    padding: 10px 12px;
  }
  .admin-toggle {
    display: inline-flex;
  }
}
</style>
