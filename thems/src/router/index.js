/**
 * 路由配置与权限管理
 * 
 * 主要功能：
 * 1. 定义应用所有路由规则，按模块分组管理
 * 2. 实现维护模式检查与权限控制
 * 3. 配置全局路由守卫（前置/后置/错误处理）
 * 4. 支持 hash/history 两种路由模式
 */

import { createRouter, createWebHashHistory, createWebHistory } from 'vue-router'
import { cache } from '@/utils/network'
import utils from '@/utils/utils'
import { useCommStore } from '@/store/comm'

const setupRouteTitle = (router) => {
  router.beforeEach((to, from, next) => {
    const siteTitle = import.meta.env.VITE_TITLE || 'Xiao-INIS'
    const pageTitle = to.meta.title || to.name || '未知页面'
    document.title = `${pageTitle} - ${siteTitle}`
    next()
  })
}

const getRouterBase = () => {
  return import.meta.env.VITE_BASE_URL || '/'
}

const getRouterMode = () => {
  return import.meta.env.VITE_ROUTER_MODE || 'hash'
}

/**
 * 路由配置数组
 * 按功能模块分组：重定向、核心页面、用户相关、文章相关、分类标签、独立页面、管理路由、404兜底
 * 
 * 路由meta字段说明：
 * - title: 页面标题
 * - requiresAuth: 是否需要登录
 * - isAdmin: 是否需要管理员权限
 * - keepAlive: 是否缓存组件
 */
const routes = [
  // ========== 重定向路由 ==========
  {
    path: '/index',
    redirect: '/'
  },
  {
    path: '/moments',
    redirect: '/'
  },

  // ========== 核心页面路由 ==========
  {
    path: '/',
    name: '动态',
    component: () => import('@/views/index/pages/index.vue'),
    meta: { title: '动态', requiresAuth: false, keepAlive: true }
  },
  // ========== 文章相关路由 ==========
  {
    path: '/blog',
    name: '文章',
    component: () => import('@/views/index/pages/blog.vue'),
    meta: { title: '文章', requiresAuth: false }
  },
  {
    path: '/archives/:id',
    name: '文章详情',
    component: () => import('@/views/index/pages/archives.vue'),
    meta: { title: '文章详情', requiresAuth: false },
    props: true
  },
  {
    path: '/article-write/:id?',
    name: '撰写文章',
    component: () => import('@/views/index/pages/article-write[id].vue'),
    meta: { title: '撰写文章', requiresAuth: true },
    props: true
  },
  // ========== 分类相关路由 ==========
  {
    path: '/category/:id',
    name: '分类页面',
    component: () => import('@/views/index/pages/category.vue'),
    meta: { title: '分类页面', requiresAuth: false },
    props: true
  },
  // ========== 标签相关路由 ==========
  {
    path: '/tags',
    name: '标签页面',
    component: () => import('@/views/index/pages/tags.vue'),
    meta: { title: '标签', requiresAuth: false }
  },
  {
    path: '/tag/:id',
    name: '单个标签页面',
    component: () => import('@/views/index/pages/tags.vue'),
    meta: { title: '单个标签页面', requiresAuth: false },
    props: true
  },

  // ========== 独立页面路由 ==========
  {
    path: '/archive',
    name: '归档页面',
    component: () => import('@/views/index/pages/page.vue'),
    meta: { title: '网站统计', requiresAuth: false },
    props: { pageKey: 'archive' }
  },
  {
    path: '/links',
    name: '友链页面',
    component: () => import('@/views/index/pages/page.vue'),
    meta: { title: '友链', requiresAuth: false },
    props: { pageKey: 'links' }
  },
  // ========== 动态详情路由（必须在独立页 /:key 之前，避免路径冲突） ==========
  {
    path: '/moments/:id',
    name: '动态详情',
    component: () => import('@/views/index/pages/index.vue'),
    meta: { title: '动态详情', requiresAuth: false },
    props: true
  },
  {
    path: '/:key',
    name: '独立页面',
    component: () => import('@/views/index/pages/page.vue'),
    meta: { title: '独立页面', requiresAuth: false },
    props: true,
    beforeEnter: (to, from, next) => {
      const currentKey = (to.params.key || '').trim()
      next(currentKey ? undefined : '/404')
    }
  },
  // ========== 用户相关路由 ==========
  {
    path: '/messages',
    name: '消息中心',
    component: () => import('@/views/index/pages/messages.vue'),
    meta: { title: '消息中心', requiresAuth: true }
  },
  {
    path: '/user',
    name: '用户设置',
    component: () => import('@/views/index/pages/user.vue'),
    meta: { title: '用户设置', requiresAuth: true }
  },
  {
    path: '/author/:id',
    name: '用户主页',
    component: () => import('@/views/index/pages/author.vue'),
    meta: { title: '用户主页', requiresAuth: false },
    props: true
  },
  {
    path: '/users',
    name: '用户列表',
    component: () => import('@/views/index/pages/users.vue'),
    meta: { title: '用户列表', requiresAuth: false }
  },
  // ========== 小黑屋路由 ==========
  {
    path: '/blackroom',
    name: '小黑屋',
    component: () => import('@/views/index/pages/blackroom.vue'),
    meta: { title: '小黑屋', requiresAuth: false }
  },
  // ========== 附件管理路由 ==========
  {
    path: '/attachments',
    name: '附件管理',
    component: () => import('@/views/index/pages/attachments.vue'),
    meta: { title: '附件管理', requiresAuth: true }
  },
  // ========== 主题功能路由 ==========
  {
    path: '/functions',
    name: '主题设置',
    component: () => import('@/views/index/pages/functions.vue'),
    meta: { title: '主题设置', requiresAuth: true, isAdmin: true }
  },
  // ========== 服务器状态路由 ==========
  {
    path: '/status',
    name: '服务器状态',
    component: () => import('@/views/index/pages/status.vue'),
    meta: { title: '服务器状态', requiresAuth: true, isAdmin: true }
  },
  // ========== 404兜底路由（必须放在最后） ==========
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error.vue'),
    meta: { title: '页面不存在', requiresAuth: false }
  }
]

/**
 * 创建路由历史对象
 * 根据配置动态选择 hash 或 history 模式
 * 
 * @returns {RouterHistory} 路由历史对象
 */
const createRouterHistory = () => {
  const currentMode = getRouterMode()
  const baseUrl = getRouterBase()
  return currentMode === 'history'
    ? createWebHistory(baseUrl)
    : createWebHashHistory(baseUrl)
}

/**
 * 创建路由实例
 * 配置路由历史、路由表和滚动行为
 */
const router = createRouter({
  history: createRouterHistory(),
  routes,
  /**
   * 滚动行为配置
   * - 有保存的滚动位置则恢复
   * - 路由路径变化时滚动到顶部
   */
  scrollBehavior: (to, from, savedPosition) => {
    if (savedPosition) {
      return savedPosition
    }
    return to.path !== from.path ? { top: 0, left: 0 } : undefined
  }
})

/**
 * 检查用户是否具有管理员权限
 * 
 * @param {Object} userInfo - 用户信息对象
 * @returns {boolean} 是否为管理员
 */
const checkAdminPermission = (userInfo) => {
  if (!userInfo) return false
  
  // 兼容多种用户信息结构
  const userAuth = userInfo.result?.auth || userInfo?.auth
  const userGroups = userAuth?.group?.list || userInfo?.group?.list || []
  
  // 检查是否有全部权限或属于admin组
  return userAuth?.all || userGroups.some(group => group.key === 'admin')
}

/**
 * 全局前置守卫
 * 处理权限校验
 */
router.beforeEach(async (to, from, next) => {
  if (to.meta.requiresAuth) {
    const commStore = useCommStore()
    
    try {
      await commStore.checkLoginState()
    } catch (error) {
      console.error('检查登录状态失败:', error)
    }
    
    const loginState = commStore.getLogin
    const isLogin = loginState.finish && !utils.is.empty(loginState.user)

    if (!isLogin) {
      next('/')
      return
    }

    if (to.meta.isAdmin) {
      const userInfo = loginState.user
      const cacheKey = `admin_status_${userInfo.id || 'unknown'}`
      let isAdmin = cache.get(cacheKey)
      
      if (isAdmin === null) {
        isAdmin = checkAdminPermission(userInfo)
        cache.set(cacheKey, isAdmin, 60)
      }
      
      if (!isAdmin) {
        next('/')
        return
      }
    }
  }

  next()
})

/**
 * 全局后置守卫
 * 缓存路由历史记录（最多保存10条，有效期7天）
 */
router.afterEach((to) => {
  const historyKey = 'router_history'
  const history = cache.get(historyKey) || []
  
  // 移除重复的路由记录
  const filteredHistory = history.filter(item => item.path !== to.path)
  
  // 添加当前路由到历史记录开头
  filteredHistory.unshift({
    path: to.path,
    name: to.name,
    meta: to.meta,
    timestamp: Date.now()
  })
  
  // 只保留最近10条记录，缓存7天
  cache.set(historyKey, filteredHistory.slice(0, 10), 24 * 7)
})

/**
 * 全局错误处理
 * 捕获路由加载和跳转过程中的错误
 */
router.onError((error) => {
  console.error('[路由错误]', error)
  
  // 处理组件加载失败（网络问题或文件不存在）
  if (error.message?.includes('Failed to fetch dynamically imported module')) {
    router.push('/404').catch(() => {})
  }
})

/**
 * 设置路由标题管理
 * 自动更新页面标题
 */
setupRouteTitle(router)

/**
 * 导出路由实例
 * @type {Router}
 */
export default router