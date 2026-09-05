import { createRouter, createWebHistory, createWebHashHistory } from 'vue-router'
import cache from '{src}/utils/cache'
import utils from '{src}/utils/utils'
import axios from '{src}/utils/request'
import { useCommStore } from '{src}/store/comm'

// 登录、注册、找回密码路由（无需登录）
const auth = {
    path: '/',
    children: [{
        path: '/',
        name: 'login',
        meta: { title: '登录' },
        component: () => import('{src}/views/admin/pages/login.vue'),
    },{
        path: '/register',
        name: 'register',
        meta: { title: '注册' },
        component: () => import('{src}/views/admin/pages/register.vue'),
    },{
        path: '/reset-password',
        name: 'reset-password',
        meta: { title: '找回密码' },
        component: () => import('{src}/views/admin/pages/reset-password.vue'),
    }],
}

// 后台路由
const admin = {
    name: 'admin',
    path: '/admin',
    component: () => import('{src}/views/admin/layout/base.vue'),
    children: [{
        path: '',
        name: 'admin-home',
        meta: { title: '控制台', auth: false },
        component: () => import('{src}/views/admin/pages/index.vue'),
    },{
        path: 'profile',
        name: 'admin-profile',
        meta: { title: '个人中心' },
        component: () => import('{src}/views/admin/pages/profile.vue'),
    },{
        path: 'users',
        name: 'admin-users',
        meta: { title: '用户管理' },
        component: () => import('{src}/views/admin/pages/users.vue'),
    },{
        path: 'banner',
        name: 'admin-banner',
        meta: { title: '轮播管理' },
        component: () => import('{src}/views/admin/pages/banner.vue'),
    },{
        path: 'links',
        name: 'admin-links',
        meta: { title: '友链管理' },
        component: () => import('{src}/views/admin/pages/links.vue'),
    },{
        path: 'tags',
        name: 'admin-tags',
        meta: { title: '标签管理' },
        component: () => import('{src}/views/admin/pages/tags.vue'),
    },{
        path: 'placard',
        name: 'admin-placard',
        meta: { title: '公告管理' },
        component: () => import('{src}/views/admin/pages/placard.vue'),
    },{
        path: 'level',
        name: 'admin-level',
        meta: { title: '等级管理' },
        component: () => import('{src}/views/admin/pages/level.vue'),
    },{
        path: 'comment',
        name: 'admin-comment',
        meta: { title: '评论管理' },
        component: () => import('{src}/views/admin/pages/comment.vue'),
    },{
        path: 'article',
        name: 'admin-article',
        meta: { title: '文章列表' },
        component: () => import('{src}/views/admin/pages/article.vue'),
    },{
        path: 'article/write/:id(\\d+)?',
        name: 'admin-article-write',
        meta: { title: '撰写文章' },
        component: () => import('{src}/views/admin/pages/article-write[id].vue'),
    },{
        path: 'article/group',
        name: 'admin-article-group',
        meta: { title: '文章分组' },
        component: () => import('{src}/views/admin/pages/article-group.vue'),
    },{
        path: 'pages',
        name: 'admin-pages',
        meta: { title: '页面列表' },
        component: () => import('{src}/views/admin/pages/pages.vue'),
    },{
        path: 'pages/write/:id(\\d+)?',
        name: 'admin-pages-write',
        meta: { title: '撰写页面' },
        component: () => import('{src}/views/admin/pages/pages-write[id].vue'),
    },{
        path: 'links/group',
        name: 'admin-links-group',
        meta: { title: '友链分组' },
        component: () => import('{src}/views/admin/pages/links-group.vue'),
    },{
        path: 'auth/rules',
        name: 'admin-auth-rules',
        meta: { title: '权限规则' },
        component: () => import('{src}/views/admin/pages/auth-rules.vue'),
    },{
        path: 'auth/group',
        name: 'admin-auth-group',
        meta: { title: '权限分组' },
        component: () => import('{src}/views/admin/pages/auth-group.vue'),
    },{
        path: 'auth/pages',
        name: 'admin-auth-pages',
        meta: { title: '后台页面管理' },
        component: () => import('{src}/views/admin/pages/auth-pages.vue'),
    },{
        path: 'api/keys',
        name: 'admin-api-keys',
        meta: { title: '接口密钥' },
        component: () => import('{src}/views/admin/pages/api-keys.vue'),
    },{
        path: 'system',
        name: 'admin-system',
        meta: { title: '系统配置' },
        component: () => import('{src}/views/admin/pages/system.vue'),
    },{
        path: 'ip/black',
        name: 'admin-ip-black',
        meta: { title: 'IP黑名单' },
        component: () => import('{src}/views/admin/pages/ip-black.vue'),
    },{
        path: 'ip/white',
        name: 'admin-ip-white',
        meta: { title: 'IP白名单' },
        component: () => import('{src}/views/admin/pages/ip-white.vue'),
    },{
        path: 'qps/warn',
        name: 'admin-qps-warn',
        meta: { title: 'QPS预警' },
        component: () => import('{src}/views/admin/pages/qps-warn.vue'),
    },{
        path: 'moments',
        name: 'admin-moments',
        meta: { title: '动态管理' },
        component: () => import('{src}/views/admin/pages/moments.vue'),
    },{
        path: 'attachment',
        name: 'admin-attachment',
        meta: { title: '附件管理' },
        component: () => import('{src}/views/admin/pages/attachment.vue'),
    },{
        path: 'exp',
        name: 'admin-exp',
        meta: { title: '经验管理' },
        component: () => import('{src}/views/admin/pages/exp.vue'),
    },{
        path: 'goods',
        name: 'admin-goods',
        meta: { title: '商品管理' },
        component: () => import('{src}/views/admin/pages/goods.vue'),
    },{
        path: 'integral',
        name: 'admin-integral',
        meta: { title: '积分管理' },
        component: () => import('{src}/views/admin/pages/integral.vue'),
    },{
        path: 'message',
        name: 'admin-message',
        meta: { title: '消息通知' },
        component: () => import('{src}/views/admin/pages/message.vue'),
    }],
}

const routes = [ auth, admin, {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('{src}/views/error.vue'),
}]

const base = '/'
const mode = 'hash'
const route = createRouter({
    routes,
    history: mode === 'history' ? createWebHistory(base) : createWebHashHistory(base)
})

// 不需要登录验证的路由
const noAuthRoutes = ['login', 'register', 'reset-password', 'not-found']

// 路由守卫
route.beforeEach(async (to, from, next) => {
    // 设置页面标题
    if (to.meta?.title) document.title = to.meta.title

    // 登录状态无效处理
    const invalid = (params = { path: '/' }) => {
        cache.del('user-info')
        utils.clear.cookie(globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN')
        // 注意：token 失效时后端 Jwt 中间件已自动清除 cookie，
        // 这里无需再发 logout 请求，避免 405 报错及与刷新时序冲突导致的“退出又恢复”问题
        next(params)
    }

    // 登录、注册、找回密码路由直接放行
    if (noAuthRoutes.includes(to.name)) {
        // 根路径 / 在已登录状态下重定向到 /admin
        if (to.name === 'login' && to.path === '/') {
            const TOKEN_NAME = globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN'
            if (utils.has.cookie(TOKEN_NAME)) {
                next('/admin')
                return
            }
        }
        next()
        return
    }

    // 后台路由校验 - 未登录跳转到登录页
    if (to.path.indexOf('/admin') === 0) {
        const TOKEN_NAME = globalThis?.inis?.token_name || 'INIS_LOGIN_TOKEN'

        if (!utils.has.cookie(TOKEN_NAME)) return invalid()

        if (cache.has('user-info')) {
            useCommStore().nav.title = to.meta.title
            next()
            return
        }

        const { code } = await axios.post('/api/comm/check-token')
        if (code !== 200) return invalid()

        useCommStore().nav.title = to.meta.title
        next()
        return
    }

    // 其他路由跳转到登录页
    next('/')
})

export default route