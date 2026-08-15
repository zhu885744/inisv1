<template>
  <!-- 顶部导航栏 -->
  <nav class="navbar navbar-expand-lg bg-body-tertiary px-md-5 shadow-sm">
    <div class="container d-flex justify-content-between">
      <!-- 移动端侧边栏触发按钮 -->
      <button 
        class="navbar-toggler d-lg-none me-3 border-0" 
        type="button" 
        @click="toggleSidebar"
        aria-label="Toggle navigation"
      >
        <span class="navbar-toggler-icon"></span>
      </button>

      <!-- 网站标题：跳首页 -->
      <router-link class="navbar-brand flex-grow-1 text-center d-lg-block d-flex justify-content-start align-items-center" to="/">
        <img 
          v-if="store.comm.siteInfo?.avatar" 
          :src="store.comm.siteInfo.avatar" 
          alt="网站LOGO" 
          style="width: 45px;border-radius: 50%;"
        >
        <span v-else class="fw-bold">{{ store.comm.siteInfo?.title || '未设置网站名' }}</span>
      </router-link>

      <!-- 移动端右侧搜索按钮 -->
      <div class="d-flex align-items-center ms-3">
        <button class="btn d-lg-none border-0 bg-transparent" type="button" @click="method.showSearch()">
          <i class="bi bi-search"></i>
        </button>
      </div>

      <!-- PC端导航内容 -->
      <div class="collapse navbar-collapse d-none d-lg-flex" id="navbarSupportedContent">
        <ul class="navbar-nav me-auto mb-2 mb-lg-0">
          <!-- 动态 -->
          <li class="nav-item">
            <router-link class="nav-link" to="/" active-class="active" exact-active-class="active">
              动态
            </router-link>
          </li>

          <!-- 首页 -->
          <li class="nav-item">
            <router-link class="nav-link" aria-current="page" to="/blog" active-class="active" exact-active-class="active">
              文章
            </router-link>
          </li>

          <!-- 分类下拉列表 -->
          <li class="nav-item dropdown" ref="pcDropdownRef">
            <a class="nav-link dropdown-toggle" role="button" aria-expanded="false" data-bs-toggle="dropdown" data-bs-display="static">
              分类
            </a>
            <ul class="dropdown-menu dropdown-menu-start">
              <li v-for="category in categories" :key="category.id">
                <router-link class="dropdown-item" :to="`/category/${category.key}`">
                  {{ category.name }}
                </router-link>
              </li>
              <li v-if="categories.length === 0" class="dropdown-item text-muted">
                暂无分类数据
              </li>
            </ul>
          </li>

          <!-- 动态渲染的导航项 -->
          <li v-for="item in navItems" :key="item.key" class="nav-item">
            <router-link class="nav-link" :to="`/${item.key}`" active-class="active" exact-active-class="active">
              {{ item.title }}
            </router-link>
          </li>

          <!-- 自定义导航链接下拉菜单 -->
          <li v-if="customNavLinks.length > 0" class="nav-item dropdown">
            <a class="nav-link dropdown-toggle" role="button" aria-expanded="false" data-bs-toggle="dropdown" data-bs-display="static">
              推荐
            </a>
            <ul class="dropdown-menu dropdown-menu-start">
              <li v-for="(link, index) in customNavLinks" :key="'custom-' + index">
                <a class="dropdown-item" :href="link.url" target="_blank" rel="noopener noreferrer">
                  {{ link.text }}
                </a>
              </li>
            </ul>
          </li>
        </ul>
        
        <!-- 右侧功能区域 -->
        <div class="d-flex align-items-center">
          <!-- 搜索按钮 -->
          <button class="btn btn-outline-secondary me-2" type="button" @click="method.showSearch()">
            <i class="bi bi-search"></i>
          </button>
          
          <!-- 深色/浅色模式切换按钮 -->
          <button 
            class="btn btn-outline-secondary me-2" 
            type="button" 
            @click="toggleDarkMode"
            :title="isDarkMode ? '切换到浅色模式' : '切换到深色模式'"
          >
            <i :class="darkModeIcon"></i>
          </button>
          
          <!-- 签到按钮 -->
          <button 
            v-if="store.comm.isLoggedIn"
            class="btn btn-outline-secondary me-2" 
            type="button" 
            @click="doSign"
            :disabled="signLoading"
            :title="hasSigned ? '查看签到记录' : '每日签到'"
          >
            <i v-if="!signLoading" :class="hasSigned ? 'bi bi-check-circle' : 'bi bi-calendar-check'"></i>     
          </button>

          <!-- 通知按钮（含未读角标），仅登录后显示 -->
          <button 
            v-if="store.comm.isLoggedIn"
            class="btn btn-outline-secondary me-2 position-relative" 
            type="button" 
            @click="method.showNotification()"
            title="消息中心"
          >
            <i class="bi bi-bell"></i>
            <span 
              v-if="socketStore.unreadNotificationCount > 0" 
              class="position-absolute top-0 start-100 translate-middle badge rounded-pill bg-danger"
              style="font-size: 0.65rem;"
            >
              {{ socketStore.unreadNotificationCount > 99 ? '99+' : socketStore.unreadNotificationCount }}
            </span>
          </button>
          
          <!-- 用户相关功能 -->
          <div class="d-flex align-items-center" v-if="store.comm.isLoggedIn">
            <!-- 已登录用户信息 -->
            <div class="dropdown" ref="userDropdownRef">
              <button 
                class="btn btn-outline-secondary dropdown-toggle d-flex align-items-center" 
                type="button" 
                id="userDropdownMenu"
                data-bs-toggle="dropdown" 
                data-bs-display="static"
                aria-expanded="false"
              >
                <i class="bi bi-person-circle me-1"></i>
                <span class="text-truncate" style="max-width: 100px;">
                  {{ store.comm.login.user.nickname || store.comm.login.user.account }}
                </span>
              </button>
              <ul class="dropdown-menu dropdown-menu-start" aria-labelledby="userDropdownMenu">
                <li>
                  <router-link class="dropdown-item" to="/article-write">
                    <i class="bi bi-pencil-square me-1"></i>发布文章
                  </router-link>
                </li>
                <li>
                  <router-link class="dropdown-item" to="/attachments">
                    <i class="bi bi-folder2-open me-1"></i>附件管理
                  </router-link>
                </li>
                <li>
                  <router-link class="dropdown-item" :to="`/author/${store.comm.login.user.id}`">
                    <i class="bi bi-person me-1"></i>用户主页
                  </router-link>
                </li>
                <li>
                  <router-link class="dropdown-item" to="/user">
                    <i class="bi bi-gear me-1"></i>用户设置
                  </router-link>
                </li>
                <li v-if="isAdmin">
                  <router-link class="dropdown-item" to="/functions">
                    <i class="bi bi-palette me-1"></i>站点配置
                  </router-link>
                </li>
                <li v-if="isAdmin">
                  <router-link class="dropdown-item" to="/status">
                    <i class="bi bi-activity me-1"></i>服务器状态
                  </router-link>
                </li>
                <li>
                  <button class="dropdown-item" @click="clearCache()">
                    <i class="bi bi-trash3 me-1"></i>清除缓存
                  </button>
                </li>
                <li><hr class="dropdown-divider"></li>
                <li>
                  <button class="dropdown-item text-danger" @click="method.logout()">
                    <i class="bi bi-box-arrow-right me-1"></i>退出登录
                  </button>
                </li>
              </ul>
            </div>
          </div>
          <div class="d-flex" v-else>
            <!-- 未登录状态 -->
            <button class="btn btn-outline-secondary me-2" @click="method.showLogin()">
              登录
            </button>
            <button 
              v-if="parseInt(store.config.getAllowRegister?.value) === 1" 
              class="btn btn-outline-secondary"
              @click="method.showRegister()"
            >
              注册
            </button>
          </div>
        </div>
      </div>
    </div>
  </nav>

  <!-- 移动端侧边栏 -->
  <div 
    class="offcanvas offcanvas-start" 
    tabindex="-1" 
    id="mobileSidebar"
    aria-labelledby="mobileSidebarLabel"
  >
    <div class="offcanvas-header border-bottom">
      <h5 class="offcanvas-title d-flex align-items-center" id="mobileSidebarLabel">
        <img 
          v-if="store.comm.siteInfo?.avatar" 
          :src="store.comm.siteInfo.avatar" 
          alt="网站图标" 
          class="nav-logo"
        >
        <span v-else class="fw-bold">{{ store.comm.siteInfo?.title || '朱某的生活印记' }}</span>
      </h5>
      <button 
        type="button" 
        class="btn-close" 
        data-bs-dismiss="offcanvas" 
        aria-label="Close"
        @click="closeSidebar"
      ></button>
    </div>
    <div class="offcanvas-body d-flex flex-column">
      <!-- 移动端导航菜单 -->
      <ul class="navbar-nav flex-grow-1">
        <!-- 首页 -->
        <li class="nav-item mb-2">
          <router-link 
            class="nav-link" 
            to="/" 
            active-class="active" 
            exact-active-class="active"
            @click="closeSidebar"
          >
            动态
          </router-link>
        </li>

        <!-- 动态 -->
        <li class="nav-item mb-2">
          <router-link 
            class="nav-link" 
            to="/blog" 
            active-class="active" 
            exact-active-class="active"
            @click="closeSidebar"
          >
            文章
          </router-link>
        </li>

        <!-- 分类菜单 -->
        <li class="nav-item mb-2">
          <a 
            class="nav-link d-flex justify-content-between align-items-center" 
            href="javascript:;" 
            @click="toggleCategoryDropdown"
          >
            分类
            <i :class="categoryDropdownOpen ? 'bi bi-chevron-up' : 'bi bi-chevron-down'"></i>
          </a>
          <!-- 分类子菜单 -->
          <ul class="navbar-nav ms-3 mt-2" v-show="categoryDropdownOpen">
            <li v-for="category in categories" :key="category.id" class="nav-item mb-1">
              <router-link 
                class="nav-link" 
                :to="`/category/${category.key}`"
                @click="closeSidebar"
              >
                {{ category.name }}
              </router-link>
            </li>
            <li v-if="categories.length === 0" class="nav-item text-muted small">
              暂无分类数据
            </li>
          </ul>
        </li>

        <!-- 动态导航项 -->
        <li v-for="item in navItems" :key="item.key" class="nav-item mb-2">
          <router-link 
            class="nav-link" 
            :to="`/${item.key}`" 
            active-class="active" 
            exact-active-class="active"
            @click="closeSidebar"
          >
            {{ item.title }}
          </router-link>
        </li>

        <!-- 自定义导航链接 -->
        <li v-for="(link, index) in customNavLinks" :key="'custom-mobile-' + index" class="nav-item mb-2">
          <a 
            class="nav-link" 
            :href="link.url" 
            target="_blank" 
            rel="noopener noreferrer"
            @click="closeSidebar"
          >
            {{ link.text }}
          </a>
        </li>
      </ul>

      <!-- 移动端功能按钮组 -->
      <div class="mt-auto pt-3 border-top">
        <!-- 未登录状态 -->
        <div v-if="utils.is.empty(store.comm.login.user)" class="mb-4">
          <div class="d-grid grid-cols-2 gap-3">
            <button class="btn btn-outline-secondary text-center" type="button" @click="method.showLogin()">
              <i class="bi bi-person-circle me-1"></i>登录
            </button>
            <button 
              v-if="parseInt(store.config.getAllowRegister?.value) === 1" 
              class="btn btn-outline-secondary text-center" 
              type="button" 
              @click="method.showRegister()"
            >
              <i class="bi bi-person-plus me-1"></i>注册
            </button>
          </div>
        </div>
        
        <!-- 已登录状态 -->
        <div v-else-if="!utils.is.empty(store.comm.login.user)" class="mb-4">
          <div class="text-center mb-3">
            <i class="bi bi-person-circle fs-4 text-secondary"></i>
            <div class="mt-1">{{ store.comm.login.user.nickname }}</div>
            <small class="text-muted">{{ store.comm.login.user.email }}</small>
          </div>
          <button 
            class="btn btn-secondary w-100 mb-3" 
            type="button" 
            @click="doSign"
            :disabled="signLoading"
          >
            <i v-if="!signLoading" :class="hasSigned ? 'bi bi-check-circle' : 'bi bi-calendar-check'" class="me-1"></i>
            <i v-else class="bi bi-arrow-clockwise animate-spin me-1"></i>
            {{ hasSigned ? '签到记录' : '每日签到' }}
            <span v-if="signDays > 0" class="ms-1 text-muted">({{ signDays }}天)</span>
          </button>
          <div class="d-grid grid-cols-2 gap-3">
            <router-link class="btn btn-outline-secondary text-center" to="/article-write" @click="closeSidebar">
              <i class="bi bi-pencil-square me-1"></i>发布文章
            </router-link>
            <router-link class="btn btn-outline-secondary text-center" to="/messages" @click="closeSidebar">
              <i class="bi bi-bell me-1"></i>消息中心
            </router-link>
            <router-link class="btn btn-outline-secondary text-center" to="/attachments" @click="closeSidebar">
              <i class="bi bi-folder2-open me-1"></i>附件管理
            </router-link>
            <router-link class="btn btn-outline-secondary text-center" :to="`/author/${store.comm.login.user.id}`" @click="closeSidebar">
              <i class="bi bi-person me-1"></i>用户主页
            </router-link>
            <router-link class="btn btn-outline-secondary text-center" to="/user" @click="closeSidebar">
              <i class="bi bi-gear me-1"></i>用户设置
            </router-link>
            <router-link v-if="isAdmin" class="btn btn-outline-secondary text-center" to="/functions" @click="closeSidebar">
              <i class="bi bi-palette me-1"></i>主题设置
            </router-link>
            <router-link v-if="isAdmin" class="btn btn-outline-secondary text-center" to="/status" @click="closeSidebar">
              <i class="bi bi-activity me-1"></i>服务器状态
            </router-link>
            <button class="btn btn-outline-secondary text-center" type="button" @click="clearCache()">
              <i class="bi bi-trash3 me-1"></i>清除缓存
            </button>
            <button class="btn btn-outline-secondary text-center" type="button" @click="method.logout()">
              <i class="bi bi-box-arrow-right me-1"></i>退出登录
            </button>
          </div>
        </div>
        
        <!-- 其他功能按钮 -->
        <div class="d-flex gap-2 mt-3">
          <button class="btn btn-outline-secondary flex-grow-1" type="button" @click="method.showSearch()">
            <i class="bi bi-search me-1"></i>搜索
          </button>
          <button 
            class="btn btn-outline-secondary flex-grow-1" 
            type="button" 
            @click="toggleDarkMode"
          >
            <i :class="[darkModeIcon, 'me-1']"></i>{{ isDarkMode ? '浅色' : '深色' }}
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- 引入搜索对话框组件 -->
  <DialogSearch ref="searchDialog" />

  <!-- 引入认证对话框组件 -->
  <DialogAuth ref="authDialog" @finish="method.onAuthFinish" />

  <!-- 引入签到对话框组件 -->
  <DialogCheckin ref="checkinDialog" />

  <!-- 封禁提醒弹窗 -->
  <BanNotice ref="banNoticeDialog" @appeal="method.openAppealDialog" />

  <!-- 封禁申诉弹窗 -->
  <DialogAppeal ref="appealDialog" />
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, nextTick, reactive, watch } from 'vue'
import { request } from '@/utils/network'
import utils from '@/utils/utils'
import { toast } from '@/utils/app'
import { cache } from '@/utils/network'
import { useRouter } from 'vue-router'
import { useCommStore } from '@/store/comm'
import { useConfigStore } from '@/store/config'
import { useSocketStore } from '@/store/socket'
import { STORAGE_KEYS } from '@/constants'

// 引入对话框组件
import DialogAuth from '@/comps/index/dialog/auth.vue'
import DialogSearch from '@/comps/index/dialog/search.vue'
import DialogCheckin from '@/comps/index/dialog/checkin.vue'
import BanNotice from '@/comps/index/dialog/ban-notice.vue'
import DialogAppeal from '@/comps/index/dialog/appeal.vue'

// 初始化router
const router = useRouter()

// 导航项数据
const navItems = ref([])
const categories = ref([])
const sidebarOpen = ref(false)
const categoryDropdownOpen = ref(false)
const customNavLinks = ref([])

// 签到相关数据
const hasSigned = ref(false)
const signDays = ref(0)
const signLoading = ref(false)

// 组件引用
const authDialog = ref(null)
const searchDialog = ref(null)
const checkinDialog = ref(null)
const banNoticeDialog = ref(null)
const appealDialog = ref(null)

// 存储
const store = {
  comm: useCommStore(),
  config: useConfigStore()
}
const socketStore = useSocketStore()

// 清除缓存方法
const clearCache = () => {
  if (confirm('确定要清除所有缓存吗？此操作不会影响您的登录状态。')) {
    const userInfo = cache.get('user-info')
    const uid = localStorage.getItem(STORAGE_KEYS.UID)
    
    cache.clear()
    localStorage.removeItem(STORAGE_KEYS.SEARCH_HISTORY)
    localStorage.removeItem(STORAGE_KEYS.LAST_COMMENT_TIME)
    
    if (userInfo) {
      cache.set('user-info', userInfo, 15 * 24 * 60)
    }
    if (uid) {
      localStorage.setItem(STORAGE_KEYS.UID, uid)
    }
    
    toast.success('缓存已清除')
    setTimeout(() => {
      window.location.reload()
    }, 1500)
  }
}

// 状态管理
const state = reactive({
  theme: 'white',
  drawer: {
    show: false,
    menu: true
  },
  nav: {
    name: 'index',
    color: {
      active: 'rgb(var(--assist-color))',
      inactive: 'rgb(var(--vice-font-color))',
    }
  },
})

// 方法定义
const method = {
  // 显示搜索弹窗
  showSearch: () => {
    closeSidebar()
    if (searchDialog.value && searchDialog.value.show) {
      searchDialog.value.show()
    } else {
      setTimeout(() => {
        if (searchDialog.value && searchDialog.value.show) {
          searchDialog.value.show()
        }
      }, 100)
    }
  },

  // 显示对话框
  showLogin: () => {
    // 先关闭侧边栏
    closeSidebar()
    
    if (authDialog.value && authDialog.value.show) {
      authDialog.value.show('login')
    } else {
      // 延迟一下，确保对话框引用被正确初始化
      setTimeout(() => {
        if (authDialog.value && authDialog.value.show) {
          authDialog.value.show('login')
        }
      }, 100)
    }
  },
  
  showRegister: () => {
    // 先关闭侧边栏
    closeSidebar()
    
    if (authDialog.value && authDialog.value.show) {
      authDialog.value.show('register')
    } else {
      // 延迟一下，确保对话框引用被正确初始化
      setTimeout(() => {
        if (authDialog.value && authDialog.value.show) {
          authDialog.value.show('register')
        }
      }, 100)
    }
  },
  
  showResetPassword: () => {
    // 先关闭侧边栏
    closeSidebar()
    
    if (authDialog.value && authDialog.value.show) {
      authDialog.value.show('reset')
    } else {
      // 延迟一下，确保对话框引用被正确初始化
      setTimeout(() => {
        if (authDialog.value && authDialog.value.show) {
          authDialog.value.show('reset')
        }
      }, 100)
    }
  },
  
  // 事件处理
  onAuthFinish: (user) => {
    // 登录或注册成功
    if (user) {
      // console.log('登录/注册成功:', user)
      store.comm.login.finish = true
      store.comm.login.user = user
      closeSidebar()
      // 登录成功提示
      toast.success('登录成功')
      // 重新初始化下拉菜单（用户登录后）
      nextTick(() => {
        initDropdowns()
      })
    } else {
      // 密码重置成功
      // console.log('密码重置成功')
      // 密码重置成功提示
      toast.success('密码重置成功，请登录')
      method.showLogin()
    }
  },
  
  // 登出（核心修改：适配后端 DELETE /api/comm/logout 接口）
  logout: async () => {
    try {
      // 1. 调用后端退出登录接口（POST 请求）
      const response = await request.post('/api/comm/logout')
      
      // 2. 处理接口响应（根据后端返回状态码判断）
      if (response.code === 200) {
        // 接口调用成功 - 清理前端状态
        utils.set.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', '', -1)
        store.comm.login.finish = false
        store.comm.login.user = null
        
        // 清理本地缓存的用户信息
        if (utils.cache?.del) {
          utils.cache.del('user-info')
        } else if (localStorage) {
          localStorage.removeItem('user-info')
        }
        
        // 成功提示
        toast.success('已退出登录')
      } else {
        // 接口返回错误
        toast.error(response.msg || '退出登录失败')
      }
    } catch (error) {
      // 网络错误/接口调用失败
      console.error('退出登录接口调用失败:', error)
      
      // 兜底清理前端状态（即使接口失败，也清理本地Token）
      utils.set.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN', '', -1)
      store.comm.login.finish = false
      store.comm.login.user = null
      
      // 错误提示
      toast.error('网络异常，已本地退出登录')
    } finally {
      // 无论成功失败，都关闭侧边栏
      closeSidebar()
      // 可选：登出后跳转到首页
      router.push('/')
    }
  },
  
  // 获取当前主题
  async getTheme() {
    let theme = document.querySelector('body').getAttribute('inis-theme')
    if (!utils.is.empty(theme)) {
      if (theme.indexOf('white') !== -1) state.theme = 'white'
      else state.theme = 'dark'
    }
  },
  
  // 路由跳转 - 关闭抽屉
  push: (params = {}) => {
    router.push(params)
    state.drawer.show = false
  },

  // 打开申诉弹窗
  openAppealDialog() {
    if (appealDialog.value && appealDialog.value.show) {
      appealDialog.value.show()
    } else {
      setTimeout(() => {
        if (appealDialog.value && appealDialog.value.show) {
          appealDialog.value.show()
        }
      }, 100)
    }
  },

  // 跳转至消息中心
  showNotification() {
    closeSidebar()
    router.push('/messages')
  },

  // 获取未读通知数量
  async fetchUnreadCount() {
    if (!store.comm.login.finish || !store.comm.login.user) return
    try {
      const res = await request.get('/api/notification/unread-count')
      if (res.code === 200 && res.data != null) {
        const count = typeof res.data === 'object' ? (res.data.count || 0) : res.data
        socketStore.setUnreadCount(Number(count) || 0)
      }
    } catch {
      // 静默处理
    }
  },
}

// 暴露方法给父组件
defineExpose({
  method,
  store
})

// 初始化下拉菜单 - 由于使用了 data-bs-toggle 属性，Bootstrap 会自动初始化下拉菜单
// 此函数保留以备将来需要自定义配置时使用
const initDropdowns = () => {
  // 等待 DOM 更新完成
  nextTick(() => {
    // 下拉菜单已通过 data-bs-toggle 属性自动初始化
    // 如需自定义配置，可在此添加
  })
}

// 计算属性
const isDarkMode = computed(() => store.comm.isDarkMode)

const darkModeIcon = computed(() => {
  return isDarkMode.value ? 'bi bi-brightness-high-fill' : 'bi bi-brightness-high'
})

// 判断用户是否为管理员
const isAdmin = computed(() => {
  const user = store.comm.login.user
  if (!user || !user.result || !user.result.auth) return false
  
  // 检查是否有all权限
  if (user.result.auth.all) return true
  
  // 检查权限组中是否有admin
  const groups = user.result.auth.group?.list || []
  return groups.some(group => group.key === 'admin')
})

// 切换深色/浅色模式
const toggleDarkMode = () => {
  store.comm.toggleDarkMode()
}

// 初始化主题
const initTheme = () => {
  // 应用主题
  const htmlElement = document.documentElement
  htmlElement.setAttribute('data-bs-theme', isDarkMode.value ? 'dark' : 'light')
}

// 移除系统主题变化的监听，只支持手动切换
const setupSystemThemeListener = () => {
  // 不再监听系统主题变化
  return () => {}
}

// 侧边栏实例
let offcanvasInstance = null

// 初始化侧边栏
const initOffcanvas = () => {
  if (window.bootstrap && !offcanvasInstance) {
    const sidebar = document.getElementById('mobileSidebar')
    if (sidebar) {
      offcanvasInstance = new window.bootstrap.Offcanvas(sidebar, {
        backdrop: true,
        keyboard: true
      })
      
      // 监听关闭事件
      sidebar.addEventListener('hidden.bs.offcanvas', () => {
        sidebarOpen.value = false
        categoryDropdownOpen.value = false
      })
      
      // 监听显示事件
      sidebar.addEventListener('shown.bs.offcanvas', () => {
        sidebarOpen.value = true
      })
    }
  }
}

// 切换移动端侧边栏
const toggleSidebar = () => {
  if (!offcanvasInstance) {
    initOffcanvas()
  }
  
  if (!offcanvasInstance) {
    console.warn('[Nav] 侧边栏实例初始化失败')
    return
  }
  
  if (sidebarOpen.value) {
    offcanvasInstance.hide()
  } else {
    offcanvasInstance.show()
  }
}

// 关闭侧边栏
const closeSidebar = () => {
  if (offcanvasInstance) {
    offcanvasInstance.hide()
  }
  sidebarOpen.value = false
  categoryDropdownOpen.value = false
}

// 切换移动端分类下拉
const toggleCategoryDropdown = () => {
  categoryDropdownOpen.value = !categoryDropdownOpen.value
}

// 从响应中提取列表数据，兼容 data / data.data 两种结构
const pickList = (response) => {
  const payload = response?.data
  if (Array.isArray(payload)) return payload
  if (Array.isArray(payload?.data)) return payload.data
  return []
}

// 从API获取导航数据（走统一缓存，后台静默刷新）
const fetchNavData = async () => {
  try {
    const response = await request.cached('/api/pages/all', {
      field: 'key,title'
    }, { minutes: 60, swr: true })

    navItems.value = pickList(response)
  } catch {
    navItems.value = []
  }
}

// 从store获取站点信息
const fetchSiteInfo = async () => {
  try {
    await store.comm.fetchSiteInfo()
    // 解析自定义导航链接
    parseCustomNavLinks()
  } catch (error) {
    // console.error('获取站点信息失败:', error)
  }
}

// 解析自定义导航链接
// 格式：每行一个，跳转文字 || 跳转链接
const parseCustomNavLinks = () => {
  const linksStr = store.comm.siteInfo?.custom_nav_links || ''
  if (!linksStr || typeof linksStr !== 'string') {
    customNavLinks.value = []
    return
  }
  const lines = linksStr.split(/\r?\n/).map(line => line.trim()).filter(line => line)
  const links = []
  for (const line of lines) {
    const parts = line.split('||').map(p => p.trim())
    if (parts.length === 2 && parts[0] && parts[1]) {
      links.push({
        text: parts[0],
        url: parts[1]
      })
    }
  }
  customNavLinks.value = links
}

// 从API获取分类数据（走统一缓存，后台静默刷新）
const fetchCategories = async () => {
  try {
    const response = await request.cached('/api/article-group/all', {
      field: 'id,key,name'
    }, { minutes: 60, swr: true })

    categories.value = pickList(response)
  } catch {
    categories.value = []
  }
}

// 签到相关方法
const checkSignStatus = async () => {
  if (!store.comm.login.finish || !store.comm.login.user) return
  
  try {
    const response = await request.get('/api/exp/check-in')
    if (response.code === 200) {
      hasSigned.value = response.data?.checked || false
      signDays.value = response.data?.streak_days || response.data?.current_streak || 0
    }
  } catch (error) {
    // 静默处理
  }
}

const doSign = () => {
  closeSidebar()
  
  if (checkinDialog.value && checkinDialog.value.show) {
    checkinDialog.value.show()
  } else {
    setTimeout(() => {
      if (checkinDialog.value && checkinDialog.value.show) {
        checkinDialog.value.show()
      }
    }, 100)
  }
}

// 事件处理器保持引用，便于卸载时移除，避免内存泄漏
const onUnreadChange = () => method.fetchUnreadCount()

const onResize = () => {
  if (window.innerWidth >= 992) {
    closeSidebar()
  }
}

const onKeydown = (e) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault()
    method.showSearch()
  }
}

// 组件挂载时获取数据并初始化
onMounted(() => {
  // 导航、分类、站点信息三者互不依赖，并行请求，避免瀑布流等待
  Promise.all([
    fetchNavData(),
    fetchCategories(),
    fetchSiteInfo()
  ])

  initTheme()
  setupSystemThemeListener()
  method.getTheme()

  // 注册开关配置：命中缓存时不会发请求
  store.config.loadAllowRegister()

  // 登录态相关数据仅在已登录时请求
  if (store.comm.login?.finish && store.comm.login?.user) {
    checkSignStatus()
    method.fetchUnreadCount()
  }

  // 监听消息中心操作后刷新未读角标（消息通知通过 API 拉取，无需 socket 推送）
  window.addEventListener('notification-unread-change', onUnreadChange)
  window.addEventListener('resize', onResize, { passive: true })
  window.addEventListener('keydown', onKeydown)

  // 初始化 Bootstrap 组件
  if (window.bootstrap) {
    initDropdowns()
    initOffcanvas()
  } else {
    // 如果Bootstrap未加载，等待一下再尝试
    setTimeout(() => {
      if (window.bootstrap) {
        initDropdowns()
        initOffcanvas()
      }
    }, 100)
  }
})

onUnmounted(() => {
  window.removeEventListener('notification-unread-change', onUnreadChange)
  window.removeEventListener('resize', onResize)
  window.removeEventListener('keydown', onKeydown)
})

// 当用户登录状态真正切换时才重新初始化下拉菜单并拉取未读数。
// 这里比较用户 id，避免对象引用变化引起的无效触发。
watch(() => store.comm.login.user?.id, (id, prevId) => {
  if (id === prevId) return
  nextTick(() => initDropdowns())
  if (id) {
    method.fetchUnreadCount()
    checkSignStatus()
  }
})

// 监听当前路由 name 改变
watch(() => router.currentRoute.value.name, (val) => {
  const map = {
    'index-themes-list': 'themes',
    'index-articles-list': 'articles',
  }
  state.nav.name = map[val] || val
}, { immediate: true })

watch(() => state.drawer.show, (val) => {
  // 显示抽屉时禁止滚动
  document.querySelector('body').style.overflow = val ? 'hidden' : 'auto'
})

// 监听 auth 状态变化，自动显示对应的弹窗
watch(
  () => store.comm.auth,
  (newAuth) => {
    if (newAuth.login) {
      method.showLogin()
    } else if (newAuth.register) {
      method.showRegister()
    } else if (newAuth.reset) {
      method.showResetPassword()
    }
  },
  { deep: true }
)

// 监听用户登录状态，检测封禁并弹窗提醒
let banDialogShown = false
watch(
  () => store.comm.login.user,
  (user) => {
    if (!user || utils.is.empty(user)) {
      banDialogShown = false
      return
    }
    // 检查用户是否被封禁
    const ban = user.result?.ban
    if (ban && ban.is_banned && ban.record && !banDialogShown) {
      banDialogShown = true
      setTimeout(() => {
        if (banNoticeDialog.value && banNoticeDialog.value.show) {
          banNoticeDialog.value.show(ban.record)
        }
      }, 800)
    } else if (ban && !ban.is_banned) {
      banDialogShown = false
    }
  },
  { deep: true }
)
</script>

<style scoped>
/* 简洁现代导航栏 */
:deep(.navbar) {
  background: var(--bs-body-bg) !important;
  border-bottom: 1px solid var(--bs-border-color);
}

/* 导航链接 */
:deep(.nav-link) {
  font-weight: 500;
  color: var(--bs-body-color);
  padding: 0.5rem 0.875rem !important;
  border-radius: 6px;
  transition: color 0.2s, background-color 0.2s;
}

:deep(.nav-link:hover) {
  color: var(--bs-primary);
  background-color: var(--bs-tertiary-bg);
}

:deep(.nav-link.active) {
  color: var(--bs-primary) !important;
  font-weight: 600;
}

:deep(.dropdown-item) {
  border-radius: 6px;
  padding: 0.5rem 0.875rem;
  font-weight: 500;
  color: var(--bs-body-color);
  transition: background-color 0.15s, color 0.15s;
}

:deep(.dropdown-item:hover) {
  background-color: var(--bs-tertiary-bg);
  color: var(--bs-primary);
}

:deep(.dropdown-item:active) {
  background-color: var(--bs-primary);
  color: white;
}

/* LOGO */
.nav-logo {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  transition: opacity 0.2s;
}

.nav-logo:hover {
  opacity: 0.85;
}

@media (max-width: 768px) {
  .nav-logo {
    width: 32px;
    height: 32px;
  }
}

/* 移动端侧边栏 */
:deep(.offcanvas) {
  background: var(--bs-body-bg);
}

:deep(.offcanvas-header) {
  background: var(--bs-tertiary-bg);
  border-bottom: 1px solid var(--bs-border-color);
  padding: 1rem 1.25rem;
}

:deep(.offcanvas-title) {
  font-weight: 600;
  color: var(--bs-body-color);
}

:deep(.offcanvas-body .nav-link) {
  padding: 0.625rem 0.875rem !important;
  border-radius: 6px;
  margin-bottom: 2px;
}

:deep(.offcanvas-body .nav-link.active) {
  background-color: var(--bs-primary);
  color: white !important;
}

/* 网格布局 */
:deep(.grid-cols-2) {
  grid-template-columns: repeat(2, 1fr);
}

:deep(.d-grid) {
  display: grid;
}

/* 响应式 */
@media (max-width: 991.98px) {
  :deep(.navbar) {
    padding: 0.625rem 1rem;
  }
  
  :deep(.navbar-brand) {
    text-align: center !important;
    justify-content: center !important;
  }
}

@media (min-width: 992px) {
  :deep(.navbar-brand) {
    text-align: left !important;
    justify-content: flex-start !important;
    flex-grow: 0 !important;
  }
  
  :deep(.navbar-nav .nav-item) {
    margin-right: 0.25rem;
  }
}

/* PC端下拉悬停 */
@media (min-width: 992px) {
  :deep(.nav-item.dropdown) {
    position: relative;
  }
  
  :deep(.nav-item.dropdown .dropdown-menu) {
    z-index: 9999 !important;
    opacity: 0;
    visibility: hidden;
    transform: translateY(8px);
    pointer-events: none;
    transition: opacity 0.15s, transform 0.15s, visibility 0.15s;
  }
  
  :deep(.nav-item.dropdown:hover .dropdown-menu) {
    display: block !important;
    opacity: 1;
    visibility: visible;
    transform: translateY(0);
    pointer-events: auto;
  }
  
  :deep(.nav-item.dropdown .dropdown-toggle)[data-bs-toggle="dropdown"] {
    pointer-events: none;
  }
  
  /* 用户下拉菜单 */
  :deep(.dropdown) {
    position: relative;
  }
  
  :deep(.dropdown .dropdown-menu) {
    z-index: 10000 !important;
    opacity: 0;
    visibility: hidden;
    transform: translateY(8px);
    pointer-events: none;
    transition: opacity 0.15s, transform 0.15s, visibility 0.15s;
  }
  
  :deep(.dropdown:hover .dropdown-menu) {
    display: block !important;
    opacity: 1;
    visibility: visible;
    transform: translateY(0);
    pointer-events: auto;
  }
  
  :deep(.dropdown .dropdown-toggle)[data-bs-toggle="dropdown"] {
    pointer-events: none;
  }
  
  :deep(.dropdown-menu.show) {
    display: block !important;
    opacity: 1;
    visibility: visible;
    transform: translateY(0);
    pointer-events: auto;
  }
}
</style>
