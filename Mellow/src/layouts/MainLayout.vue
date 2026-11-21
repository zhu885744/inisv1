<template>
  <div class="layout">
    <!-- 顶部导航栏（吸顶）：品牌 + 横排导航 + 快捷操作
         布局约定：导航不再放左侧栏，正文单列铺满；右侧栏（SidebarRight）保留在内容右侧 -->
    <header class="site-header">
      <div class="header-inner">
        <!-- 品牌区：配了 LOGO 就只显示 LOGO（标题/描述已在图片里），否则退回「标题 + 描述」 -->
        <router-link to="/" class="site-brand" title="返回首页">
          <img
            v-if="site.avatar && !logoFailed"
            class="brand-logo"
            :src="site.avatar"
            :alt="site.title"
            loading="eager"
            @error="logoFailed = true"
          />
          <template v-else>
            <span class="brand-title">{{ site.title }}</span>
          </template>
        </router-link>

        <!-- 横排导航：手机端隐藏（导航由底部 Tab Bar 接管） -->
        <nav class="header-nav" aria-label="主导航">
          <!-- 未配置自定义菜单：显示默认导航 -->
          <template v-if="!customMenus.length">
            <router-link to="/" exact-active-class="is-active" class="nav-item">
              <span class="nav-zh">发现</span>
            </router-link>
            <!-- 聚合流（关注 / 推荐 / 热门）：按关注关系聚合，未登录由路由守卫引导登录 -->
            <router-link to="/feed" active-class="is-active" class="nav-item">
              <span class="nav-zh">关注</span>
            </router-link>
            <!-- 话题广场：热度榜 + 我关注的（聚合页见 /topic/:key） -->
            <router-link to="/topics" active-class="is-active" class="nav-item">
              <span class="nav-zh">话题</span>
            </router-link>
            <!-- 勋章馆：公开的荣誉墙 / 佩戴榜 -->
            <router-link to="/badges" active-class="is-active" class="nav-item">
              <span class="nav-zh">勋章</span>
            </router-link>
            <!-- 文档站：独立布局（/docs） -->
            <router-link to="/docs" active-class="is-active" class="nav-item">
              <span class="nav-zh">文档</span>
            </router-link>
            <!-- 动态导航页（来自 /api/pages/all） -->
            <router-link
              v-for="p in pages"
              :key="p.id"
              :to="navTarget(p)"
              active-class="is-active"
              class="nav-item"
            >
              <span class="nav-zh">{{ p.title }}</span>
            </router-link>
            <!-- 积分商城 -->
            <router-link to="/goods" active-class="is-active" class="nav-item">
              <span class="nav-zh">商城</span>
            </router-link>
          </template>

          <!-- 已配置自定义菜单：只显示这里的菜单项（站内用 router-link，站外用 a 新窗口） -->
          <template v-else>
            <template v-for="(item, i) in customMenus" :key="'menu-' + i">
              <a
                v-if="item.external"
                :href="safeUrl(item.url) || undefined"
                target="_blank"
                rel="noopener noreferrer"
                class="nav-item"
              >
                <i v-if="item.icon" :class="item.icon" aria-hidden="true" />
                <span class="nav-zh">{{ item.name }}</span>
              </a>
              <router-link
                v-else
                :to="item.url"
                :active-class="isMenuRoot(item.url) ? 'router-link-active' : 'is-active'"
                :exact-active-class="isMenuRoot(item.url) ? 'is-active' : 'router-link-exact-active'"
                class="nav-item"
              >
                <i v-if="item.icon" :class="item.icon" aria-hidden="true" />
                <span class="nav-zh">{{ item.name }}</span>
              </router-link>
            </template>
          </template>
        </nav>

        <div class="header-actions">
          <!-- 搜索：独立页面 /search，Ctrl/⌘ + K 直达 -->
          <button
            type="button"
            class="header-icon"
            aria-label="搜索"
            title="搜索（Ctrl K）"
            @click="openSearch"
          >
            <i class="bi bi-search" aria-hidden="true" />
          </button>
          <!-- 签到：独立页面 /checkin -->
          <button
            type="button"
            class="header-icon"
            aria-label="每日签到"
            title="每日签到"
            @click="onCheckinClick"
          >
            <i class="bi bi-calendar-check" aria-hidden="true" />
          </button>
          <router-link
            v-if="!userStore.isLogged"
            to="/auth/login"
            class="btn btn-sm btn-primary"
          >登录</router-link>
          <!-- 已登录：头像点开下拉（我的主页 / 修改资料 / 创作中心 / 管理后台 / 退出登录） -->
          <div v-else ref="userMenuRef" class="user-menu">
            <button
              type="button"
              class="header-avatar"
              :class="{ 'is-open': userMenuOpen }"
              :title="userStore.user?.nickname || '我的'"
              aria-haspopup="menu"
              :aria-expanded="userMenuOpen"
              @click="toggleUserMenu"
            >
              <img
                v-if="userStore.user?.avatar"
                :src="userStore.user.avatar"
                class="header-avatar-img"
                alt="头像"
                loading="lazy"
              />
              <span v-else class="header-avatar-fallback">我的</span>
            </button>

            <Transition name="user-menu-pop">
              <div v-if="userMenuOpen" class="user-menu-panel" role="menu">
                <!-- 头部只放昵称与等级，菜单项才是操作入口 -->
                <div class="user-menu-head">
                  <span class="user-menu-name">{{ userStore.user?.nickname || '我的账号' }}</span>
                  <span v-if="userLevel" class="user-menu-hint">Lv.{{ userLevel }}</span>
                </div>

                <router-link :to="profileLink" class="user-menu-item" role="menuitem" @click="closeUserMenu">
                  <i class="bi bi-person" />
                  <span>我的主页</span>
                </router-link>
                <router-link to="/user/profile" class="user-menu-item" role="menuitem" @click="closeUserMenu">
                  <i class="bi bi-pencil-square" />
                  <span>修改资料</span>
                </router-link>
                <router-link to="/manage" class="user-menu-item" role="menuitem" @click="closeUserMenu">
                  <i class="bi bi-grid-1x2" />
                  <span>创作中心</span>
                </router-link>
                <router-link
                  v-if="canEnterAdmin(userStore.user)"
                  to="/admin"
                  class="user-menu-item"
                  role="menuitem"
                  @click="closeUserMenu"
                >
                  <i class="bi bi-gear-wide-connected" />
                  <span>管理后台</span>
                </router-link>

                <div class="user-menu-divider" />
                <button
                  type="button"
                  class="user-menu-item is-danger"
                  role="menuitem"
                  :disabled="loggingOut"
                  @click="onLogout"
                >
                  <i class="bi bi-box-arrow-right" />
                  <span>{{ loggingOut ? '退出中…' : '退出登录' }}</span>
                </button>
              </div>
            </Transition>
          </div>
        </div>
      </div>
    </header>

    <!-- is-no-right：右侧栏关闭时不再为它留一列空白，正文直接吃掉那部分宽度 -->
    <div class="layout-body" :class="{ 'is-no-right': !hasSidebar }">
      <!-- 正文 - 内容（单列，铺满「内容 + 右侧栏」之间剩下的宽度） -->
      <main class="layout-main">
        <router-view />
      </main>

      <!-- 右栏 - 个人信息卡片（关闭侧边栏或没有模块时不渲染，避免占出一列空白） -->
      <aside v-if="hasSidebar" class="layout-right">
        <SidebarRight />
      </aside>
    </div>

    <!-- 全局页脚 -->
    <!-- is-no-record：未配置任何备案号（ICP / 公安备案）——手机端只展示备案号，此时不必留一张空卡片 -->
    <footer class="layout-footer" :class="{ 'is-no-record': !site.copyCode && !site.policeCode }">
      <div class="footer-card">
        <!-- 品牌 + 快捷链接 -->
        <div class="footer-top">
          <div class="footer-brand">
            <span class="footer-brand-name">{{ site.title }}</span>
            <p v-if="site.description" class="footer-brand-desc">{{ site.description }}</p>
          </div>

          <nav class="footer-links" aria-label="页脚导航">
            <!-- 用户协议 -->
            <router-link class="footer-link" to="/agreement" active-class="is-active">
              <i class="bi bi-file-earmark-text" aria-hidden="true" />
              <span>用户协议</span>
            </router-link>
            <!-- 隐私协议 -->
            <router-link class="footer-link" to="/privacy" active-class="is-active">
              <i class="bi bi-shield-check" aria-hidden="true" />
              <span>隐私协议</span>
            </router-link>
            <!-- 小黑屋（封禁公示） -->
            <router-link class="footer-link" to="/blackroom" active-class="is-active">
              <i class="bi bi-door-closed" aria-hidden="true" />
              <span>小黑屋</span>
            </router-link>
          </nav>
        </div>

        <div class="footer-divider" aria-hidden="true" />

        <!-- 版权 / 备案 / 技术信息 -->
        <div class="footer-bottom">
          <p class="footer-copy">
            <i class="bi bi-c-circle" aria-hidden="true" />
            <span>{{ year }} {{ site.title }} · 保留所有权利</span>
          </p>

          <div class="footer-meta">
            <span v-if="site.copyCode" class="footer-record">
              <i class="bi bi-patch-check" aria-hidden="true" />
              <a
                v-if="site.copyLink"
                :href="safeUrl(site.copyLink) || undefined"
                target="_blank"
                rel="noopener noreferrer"
              >{{ site.copyCode }}</a>
              <template v-else>{{ site.copyCode }}</template>
            </span>
            <span v-if="site.policeCode" class="footer-record">
              <i class="bi bi-shield-check" aria-hidden="true" />
              <a
                v-if="site.policeLink"
                :href="safeUrl(site.policeLink) || undefined"
                target="_blank"
                rel="noopener noreferrer"
              >{{ site.policeCode }}</a>
              <template v-else>{{ site.policeCode }}</template>
            </span>
            <span class="footer-record footer-record-powered">
              <i class="bi bi-brush" aria-hidden="true" />
              <span>
                Powered by
                <a href="https://github.com/zhu885744/inisv1" target="_blank" rel="noopener noreferrer">inisv1</a>
                ·
                <a href="https://github.com/zhu885744/Mellow" target="_blank" rel="noopener noreferrer">Mellow</a>
              </span>
            </span>
          </div>
        </div>
      </div>
    </footer>

    <!-- 移动端底部 Tab Bar + 发布面板（含未读角标与消息轮询，见组件内注释） -->
    <MobileTabBar />

    <!-- 全局封禁申诉弹窗（当前用户被封禁时自动弹出） -->
    <BanAppealDialog ref="banDialogRef" />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import SidebarRight from '@/components/SidebarRight.vue'
import BanAppealDialog from '@/components/BanAppealDialog.vue'
import MobileTabBar from '@/components/MobileTabBar.vue'
import { call } from '@/api/request'
import { useUserStore } from '@/stores/user'
import { useSiteStore } from '@/stores/site'
import { normalizeSidebarConfig } from '@/utils/sidebar'
import { canEnterAdmin, pickUserLevel } from '@/utils/helper'
import { safeUrl } from '@/utils/url'
import { useRouter, useRoute } from 'vue-router'

const userStore = useUserStore()
const siteStore = useSiteStore()
const router = useRouter()
const route = useRoute()

// 签到（独立页面 /checkin）：未登录时不跳登录页，由页面自身给出登录引导
function onCheckinClick() {
  if (route.path !== '/checkin') router.push('/checkin')
}

// 全局搜索（独立页面 /search）：Ctrl/⌘ + K 直达
function openSearch() {
  if (route.path !== '/search') router.push('/search')
}
function onGlobalKey(e) {
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    openSearch()
  }
  // Esc 收起头像下拉（与 CardActionMenu 的交互习惯保持一致）
  if (e.key === 'Escape') closeUserMenu()
}

// ---------- 头像下拉菜单 ----------
// 头像不再直跳个人主页：默认主页在「我的主页」项里，其余是常用入口 + 退出登录。
// 关闭时机三处：点面板外（pointerdown）、按 Esc、路由变化（点了菜单项）。
const userMenuOpen = ref(false)
const userMenuRef = ref(null)
const loggingOut = ref(false)

// 我的主页：有 uid 就去作者主页，取不到就退回用户中心
const profileLink = computed(() => {
  const id = userStore.user?.id
  return id ? `/author/${id}` : '/user'
})

// 等级展示用统一取值（result.level.current / users_rating 两种结构都兼容）
const userLevel = computed(() => pickUserLevel(userStore.user))

function toggleUserMenu() {
  userMenuOpen.value = !userMenuOpen.value
}
function closeUserMenu() {
  userMenuOpen.value = false
}

function onDocPointerDown(e) {
  if (!userMenuOpen.value) return
  if (userMenuRef.value && !userMenuRef.value.contains(e.target)) closeUserMenu()
}

// 退出登录：清登录态 / token / 跳首页都在 store 里完成，这里只管防连点
async function onLogout() {
  closeUserMenu()
  if (loggingOut.value) return
  loggingOut.value = true
  try {
    await userStore.logout()
  } finally {
    loggingOut.value = false
  }
}

watch(() => route.path, closeUserMenu)

onMounted(() => {
  window.addEventListener('keydown', onGlobalKey)
  document.addEventListener('pointerdown', onDocPointerDown)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKey)
  document.removeEventListener('pointerdown', onDocPointerDown)
})

// 站点信息（来自 /api/config/one?key=Mellow_functions，由 site store 统一去重缓存）
const pages = ref([])
const year = new Date().getFullYear()

const site = computed(() => {
  const c = siteStore.config || {}
  return {
    title: c.title || 'Mellow',
    description: c.description || '我从虚空中惊醒',
    // 网站 LOGO（后台「网站设置 → 网站 LOGO」）：左栏顶部与移动端顶栏展示
    avatar: c.avatar || '',
    copyCode: c.copy?.code || '',
    copyLink: c.copy?.link || '',
    policeCode: c.police?.code || '',
    recordCode: c.police?.code || '',
    policeLink: c.police?.link || ''
  }
})

// 右侧栏是否真的要渲染：与 SidebarRight 内部判定保持一致
// （后台关掉侧边栏、或一个模块都没配时，都不该在右侧留出一列空白）
const hasSidebar = computed(() => {
  const sidebar = normalizeSidebarConfig(siteStore.config?.sidebar)
  return sidebar.enabled && sidebar.modules.length > 0
})

// 网站 LOGO 地址失效时不再展示，避免左栏顶部出现破图（换了地址自动重试）
const logoFailed = ref(false)
watch(() => site.value.avatar, () => {
  logoFailed.value = false
})

// 获取导航页（/api/pages/all）
async function loadPages() {
  try {
    const res = await call('pages', 'all', {
      method: 'GET',
      params: {
        page: 1,
        limit: 20,
        field: 'id,key,title',
        order: 'create_time asc'
      }
    })
    pages.value = res.data?.data || []
  } catch {
    pages.value = []
  }
}

// 根据页面 key 映射到专属路由
function navTarget(p) {
  const key = p.key
  if (key === 'archive') return '/archives'
  if (key === 'links') return '/links'
  if (key === 'about') return '/about'
  return `/${p.key || p.id}`
}

// 根路径（/）必须用精确匹配：否则 router-link 的前缀匹配会让「首页」在所有页面常亮
function isMenuRoot(url) {
  return String(url || '').split(/[?#]/)[0] === '/'
}

// 自定义菜单（Mellow_functions.custom_menus，JSON 数组）：
// 每项 { name, icon, url, target: 'internal' | 'external' }
// 非空时替换默认导航；为空时前台继续显示默认导航。
// 同时兼容旧版 custom_nav_links（"名称 || 链接" 一行一个），老数据无需手动迁移即可继续展示。
const customMenus = computed(() => {
  const list = siteStore.config?.custom_menus
  if (Array.isArray(list)) {
    return list
      .filter((item) => item && item.name && item.url)
      .map((item) => ({
        name: item.name,
        icon: item.icon || '',
        url: item.url,
        external: item.target === 'external'
      }))
  }

  const raw = siteStore.config?.custom_nav_links || ''
  return raw
    .split('\n')
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line) => {
      const [name, url] = line.split('||').map((s) => s.trim())
      if (!name || !url) return null
      return { name, icon: '', url, external: /^https?:\/\//i.test(url) }
    })
    .filter(Boolean)
})

// 封禁检测：当前用户被封禁时自动弹出申诉弹窗（同一会话只弹一次）
const banDialogRef = ref(null)
const banShown = ref(false)

function checkBan() {
  const banned = userStore.user?.result?.ban?.is_banned
  if (banned && !banShown.value) {
    banShown.value = true
    banDialogRef.value?.show()
  }
  if (!banned) banShown.value = false
}

// 登录态刷新后（登录/解封/申诉）同步检测
watch(() => userStore.user?.result?.ban?.is_banned, checkBan)

onMounted(() => {
  siteStore.load()
  loadPages()
  // 先同步登录态（含封禁状态）再检测，避免读到未就绪的 user
  userStore.ensureLogin().then(checkBan)
})
</script>

<style scoped>
.layout {
  min-height: 100vh;
  background: var(--bg);
}

/* ==================== 顶部导航栏 ==================== */
/* 吸顶：滚动时导航一直可见；半透明 + 高斯模糊，正文从下面穿过时不糊住视线 */
.site-header {
  position: sticky;
  top: 0;
  z-index: 200;
  background: var(--bg-card);
  border-bottom: 1px solid var(--border-soft);
  backdrop-filter: blur(10px);
  -webkit-backdrop-filter: blur(10px);
}
.header-inner {
  display: flex;
  align-items: center;
  gap: 20px;
  max-width: var(--content-max);
  margin: 0 auto;
  padding: 0 24px;
  height: var(--header-h);
}

/* 品牌区：LOGO 或「标题 + 描述」，整块可点回首页 */
.site-brand {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
  flex-shrink: 0;
  color: inherit;
  text-decoration: none;
  transition: opacity 0.2s;
}
.site-brand:hover {
  opacity: 0.82;
}
.brand-logo {
  display: block;
  height: 32px;
  width: auto;
  max-width: 160px;
  object-fit: contain;
  align-self: center;
}
.brand-title {
  font-family: var(--font-serif);
  font-size: 20px;
  font-weight: 600;
  letter-spacing: 1px;
  color: var(--text);
  white-space: nowrap;
}
.brand-subtitle {
  font-size: 12px;
  color: var(--text-muted);
  letter-spacing: 0.5px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 横排导航：超出宽度时横向滚动（不换行，免得吸顶栏高度跳动）
   align-self: stretch 让导航条撑满顶栏高度，当前页下划线正好贴在顶栏底边 */
.header-nav {
  display: flex;
  align-self: stretch;
  align-items: stretch;
  gap: 2px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}
.header-nav::-webkit-scrollbar {
  width: 0;
  height: 0;
  display: none;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
  flex-shrink: 0;
}
/* 顶栏圆形图标按钮（搜索 / 签到）：30×30，与右侧头像同尺寸 */
.header-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  flex-shrink: 0;
  font-size: 15px;
  border: 1px solid var(--border-soft);
  border-radius: 50%;
  background: var(--bg-soft);
  color: var(--text-soft);
  cursor: pointer;
  -webkit-tap-highlight-color: transparent;
  transition: color 0.15s, border-color 0.15s;
}
.header-icon:hover,
.header-icon:active,
.header-icon.is-on {
  color: var(--primary);
  border-color: var(--primary);
}
.header-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  flex-shrink: 0;
  border-radius: 50%;
  overflow: hidden;
  border: 1px solid var(--border);
  background: var(--bg-soft);
  color: var(--text-soft);
  font-size: 12px;
  text-decoration: none;
  cursor: pointer;
  transition: border-color var(--dur) var(--ease);
}
.header-avatar:hover,
.header-avatar.is-open {
  border-color: var(--primary);
}
.header-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

/* ---------- 头像下拉菜单 ---------- */
.user-menu {
  position: relative;
  /* display:flex 不是多余的：里面的按钮是 inline-flex，直接塞进块级 div 会按基线排版，
     容器底部多出一截行高（约 3px），导致头像比左右两个图标按钮高半个行距、看着没对齐 */
  display: flex;
  align-items: center;
  flex-shrink: 0;
}
.user-menu-panel {
  position: absolute;
  top: calc(100% + 10px);
  right: 0;
  /* 顶栏 z-index 是 200，面板要压住正文与侧栏 */
  z-index: 220;
  min-width: 172px;
  padding: 6px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  box-shadow: var(--shadow);
}
/* 昵称 + 等级：只做身份标识，不可点 */
.user-menu-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px 8px;
  margin-bottom: 4px;
  border-bottom: 1px solid var(--border-soft);
}
.user-menu-name {
  min-width: 0;
  overflow: hidden;
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  white-space: nowrap;
  text-overflow: ellipsis;
}
.user-menu-hint {
  flex-shrink: 0;
  font-size: 11px;
  color: var(--text-light);
  font-family: var(--font-mono);
}
.user-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 36px;
  padding: 0 10px;
  font-size: 13px;
  color: var(--text-soft);
  text-align: left;
  white-space: nowrap;
  text-decoration: none;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease);
}
.user-menu-item:hover {
  color: var(--text);
  background: var(--bg-muted);
}
.user-menu-item i {
  font-size: 14px;
  opacity: 0.9;
}
.user-menu-item.is-danger {
  color: var(--danger);
}
.user-menu-item.is-danger:hover {
  color: var(--danger);
  background: var(--accent-soft);
}
.user-menu-item:disabled {
  opacity: 0.6;
  cursor: default;
}
.user-menu-divider {
  height: 1px;
  margin: 6px 4px;
  background: var(--border-soft);
}
/* 展开动画：轻微下滑 + 淡入 */
.user-menu-pop-enter-active,
.user-menu-pop-leave-active {
  transition: opacity 0.14s ease, transform 0.14s ease;
}
.user-menu-pop-enter-from,
.user-menu-pop-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* 正文区：内容单列 + 右侧栏；右侧栏关闭时内容吃掉那一列 */
.layout-body {
  max-width: var(--content-max);
  margin: 0 auto;
  padding: 28px 24px 64px;
  display: grid;
  grid-template-columns: minmax(0, 1fr) var(--right-width);
  gap: var(--gap);
  align-items: start;
}
.layout-body.is-no-right {
  grid-template-columns: minmax(0, 1fr);
}

/* 顶栏导航项：撑满顶栏高度，当前页用朱砂下划线标记（与页脚链接同一套高亮语言） */
.nav-item {
  position: relative;
  display: inline-flex;
  align-items: center;
  padding: 0 10px;
  flex-shrink: 0;
  color: var(--text-soft);
  font-size: 14px;
  white-space: nowrap;
  transition: color 0.2s, background 0.2s;
}
.nav-item:hover {
  color: var(--primary);
  background: var(--bg-muted);
}
.nav-item.is-active {
  color: var(--primary);
  font-weight: 600;
}
.nav-item.is-active::after {
  content: '';
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: 0;
  height: 2px;
  background: var(--primary);
  border-radius: 2px;
}
/* 自定义菜单项的前置图标（bootstrap-icons） */
.nav-item .bi {
  margin-right: 6px;
}

.layout-main {
  min-width: 0;
}

/* ==================== 全局页脚 ==================== */
.layout-footer {
  max-width: var(--content-max);
  margin: 20px auto 0;
  padding: 0 24px 32px;
  font-size: 12px;
  color: var(--text-muted);
}
/* 页脚本身做成一张「纸」：与内容卡同色系，靠边线区分层级（主题约定：阴影只给浮层） */
.footer-card {
  position: relative;
  padding: 20px 24px 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-soft);
  border-radius: var(--radius-lg);
  overflow: hidden;
}
/* 顶部朱砂细线：呼应站点主色，中间亮两端淡 */
.footer-card::before {
  content: '';
  position: absolute;
  inset: 0 0 auto;
  height: 2px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    var(--primary-soft) 42%,
    var(--accent-glow) 50%,
    var(--primary-soft) 58%,
    transparent 100%
  );
  opacity: 0.85;
}

.footer-top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px 24px;
  flex-wrap: wrap;
}
.footer-brand {
  min-width: 0;
}
.footer-brand-name {
  display: inline-block;
  font-family: var(--font-serif);
  font-size: 15px;
  font-weight: 700;
  letter-spacing: 0.5px;
  color: var(--text);
}
.footer-brand-desc {
  margin: 5px 0 0;
  max-width: 46ch;
  font-size: 12px;
  line-height: 1.7;
  color: var(--text-light);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 快捷链接：胶囊状，hover / 当前页有淡朱砂底 */
.footer-links {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.footer-link {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 10px;
  border-radius: 999px;
  color: var(--text-muted);
  transition: color 0.2s, background-color 0.2s;
}
.footer-link i {
  font-size: 12px;
  color: var(--text-light);
  transition: color 0.2s;
}
.footer-link:hover {
  color: var(--primary-deep);
  background: var(--accent-wash);
}
.footer-link:hover i {
  color: var(--primary);
}
.footer-link.is-active {
  color: var(--primary-deep);
  background: var(--accent-soft);
}
.footer-link.is-active i {
  color: var(--primary);
}

.footer-divider {
  height: 1px;
  margin: 16px 0 12px;
  background: linear-gradient(90deg, var(--border-soft), transparent 85%);
}

.footer-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px 20px;
  flex-wrap: wrap;
  color: var(--text-light);
}
.footer-copy,
.footer-meta,
.footer-record {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
}
.footer-meta {
  gap: 14px;
  flex-wrap: wrap;
}
.footer-copy i,
.footer-record i {
  font-size: 12px;
  color: var(--text-light);
}
.footer-record a,
.layout-footer a {
  color: var(--text-muted);
  transition: color 0.2s;
}
.footer-record a:hover,
.layout-footer a:hover {
  color: var(--primary);
}

/* 窄屏：品牌 / 链接 / 版权居中对齐（< 640px 时页脚只保留备案号，见「响应式 - 手机」） */
@media (max-width: 900px) {
  .footer-top,
  .footer-bottom {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }
  .footer-brand-desc {
    max-width: none;
  }
  .footer-links {
    justify-content: center;
  }
}
.layout-right {
  position: sticky;
  /* 吸顶导航的高度 + 一点呼吸距离，滚动时不会被顶栏压住 */
  top: calc(var(--header-h) + 20px);
}

/* 响应式 - 平板：收起右侧栏，正文单列 */
@media (max-width: 1024px) {
  .layout-body {
    grid-template-columns: minmax(0, 1fr);
  }
  .layout-right {
    display: none;
  }
  /* 导航项更容易挤，收紧间距；描述先让位给导航 */
  .header-inner {
    gap: 14px;
  }
  .brand-subtitle {
    display: none;
  }
}

/* 响应式 - 手机 */
@media (max-width: 640px) {
  /* 手机端导航交给底部 Tab Bar，顶栏只留品牌 + 操作 */
  .header-nav {
    display: none;
  }
  .header-inner {
    padding: 0 16px;
  }
  .brand-title {
    font-size: 17px;
  }
  /* 极窄屏：品牌让位给操作按钮，避免「签到 / 登录」被挤出屏幕 */
  .site-brand {
    flex-shrink: 1;
    overflow: hidden;
  }
  .brand-logo {
    max-width: min(140px, 42vw);
  }
  /* 手机端页脚：保留备案号一行（品牌 / 快捷链接 / 版权 / Powered by 收起），
     底部留出悬浮 Dock 的高度（58 + 下边距 10 + 间隙 16），避免备案号被挡住 */
  .layout-footer {
    margin-top: 12px;
    padding: 0 16px calc(84px + env(safe-area-inset-bottom));
  }
  .footer-card {
    padding: 10px 14px;
  }
  .footer-top,
  .footer-divider,
  .footer-copy,
  .footer-record-powered {
    display: none;
  }
  .footer-bottom {
    justify-content: center;
    text-align: center;
    gap: 4px 12px;
  }
  .footer-meta {
    justify-content: center;
    gap: 6px 16px;
  }
  /* 没有配置备案号时不渲染卡片（页脚仍保留底部让位，供固定 Tab Bar 使用） */
  .layout-footer.is-no-record .footer-card {
    display: none;
  }
  .layout-body {
    grid-template-columns: minmax(0, 1fr);
    /* 底部让位交给页脚（手机端页脚始终渲染，且位于最后一个元素）：
       这里只留一点间距，避免内容与备案号之间出现大段空白 */
    padding: 16px 16px calc(16px + env(safe-area-inset-bottom));
  }
  /* 底部 Tab Bar 的定位样式已随组件移到 components/MobileTabBar.vue */
}
</style>
