import { createRouter, createWebHistory } from 'vue-router'
import { canEnterAdmin } from '@/utils/helper'
import { toast } from '@/utils/toast'
import { progressStart, progressDone } from '@/utils/progress'
import { applyRouteTitle } from '@/utils/pageTitle'
import { closeLightbox } from '@/utils/lightbox'

const routes = [
  // 安装向导：未安装时后端 Install 中间件会把首页 302 到这里（见 app/middleware/install.go）。
  // 独立于所有布局，且不依赖任何 /api 数据（未安装时 /api 会被拦住）。
  {
    path: '/install',
    name: 'install',
    component: () => import('@/views/install/Index.vue'),
    meta: { title: '安装向导' }
  },
  // 文档站：独立布局（独立侧边栏 + 独立内容区，与 /user 同级）
  // 必须放在 `/` 之前：MainLayout 下最后有 `:key` 兜底路由，否则 /docs 会被它吞掉
  //路径规则：/docs 为项目列表，/docs/:项目 为项目概览，/docs/:项目/:目录/:页面 为正文
    {
      path: '/docs',
      component: () => import('@/views/docs/Layout.vue'),
      children: [
        { path: '', name: 'docs', component: () => import('@/views/docs/Projects.vue') },
        {
          path: ':version',
          name: 'docs-version',
          component: () => import('@/views/docs/Home.vue'),
          props: true
        },
      {
        path: ':version/:group/:slug',
        name: 'docs-detail',
        component: () => import('@/views/docs/Page.vue'),
        props: true
      }
    ]
  },
  // 消息中心：私信 + 群聊共用一份左栏列表（/chat）
  // 右栏是子路由：/chat/:uid 私信，/chat/room/:id 群聊；/chat 无选中时显示空态。
  // 窄屏下变成单栏（有选中只看聊天，没有只看列表）。
  //
  // 游客可进：列表里只展示公共大厅，群聊页只读（输入框换成「请登录」提示）。
  // 只读接口 room/list、room/info、room/messages 是 type=common（见 app/model/auth-rules.go），
  // 私信子路由 /chat/:uid 仍然要登录。
  {
    path: '/chat',
    component: () => import('@/views/chat/Layout.vue'),
    meta: { title: '消息' },
    children: [
      {
        path: 'room/:id(\\d+)',
        name: 'room',
        component: () => import('@/views/room/Index.vue'),
        meta: { title: '群聊' }
      },
      {
        // 私信不强制登录：游客直接访问 /chat/2 时，页面里显示「登录后即可查看私信」
        // 的引导（去登录 / 先进公共大厅），而不是被守卫一把甩到登录页
        path: ':uid(\\d+)',
        name: 'chat-dm',
        component: () => import('@/views/messages/Index.vue'),
        meta: { title: '私信' }
      }
    ]
  },
  // 群邀请落地页：链接与二维码都指向 /join/<code>
  // 未登录由守卫带到登录页，登录后带 redirect 回到这里
  {
    path: '/join/:code([A-Za-z0-9]+)',
    name: 'room-join',
    component: () => import('@/views/room/Join.vue'),
    meta: { auth: true, title: '加入群聊' }
  },
  // 旧地址兼容：/messages?to=1、/room/1、/room 都跳到 /chat 下的对应页面
  {
    path: '/messages',
    redirect: (to) => (to.query.to ? `/chat/${to.query.to}` : '/chat')
  },
  {
    path: '/room',
    redirect: '/chat'
  },
  {
    path: '/room/:id(\\d+)',
    redirect: (to) => `/chat/room/${to.params.id}`
  },
  {
    path: '/',
    component: () => import('@/layouts/MainLayout.vue'),
    children: [
      { path: '', name: 'home', component: () => import('@/views/home/Index.vue') },
      { path: 'articles', name: 'articles', component: () => import('@/views/article/List.vue') },
      { path: 'archives/:id', name: 'article-detail', component: () => import('@/views/article/Detail.vue') },
      { path: 'archives', name: 'archives', component: () => import('@/views/article/Archives.vue') },
      // 独立的话题云页已并入话题广场（/topics：热度榜 + 搜索 + 我关注的，侧栏的「话题广场」入口也指向它）
      // 旧地址保留重定向，避免老书签 / 站内自定义导航链接失效
      { path: 'tags', name: 'tags', redirect: '/topics' },
      // 话题（Hashtag）：广场 + 聚合页
      // 旧的 /tag/:key 统一重定向到话题页（话题页同时含动态与文章，是话题页的超集）
      { path: 'topics', name: 'topics', component: () => import('@/views/topic/Index.vue') },
      { path: 'topic/:key', name: 'topic', component: () => import('@/views/topic/Detail.vue'), props: true },
      { path: 'tag/:key', name: 'tag', redirect: (to) => ({ path: `/topic/${to.params.key}` }) },
      { path: 'categories', name: 'categories', component: () => import('@/views/article/CategoryList.vue') },
      { path: 'category/:key', name: 'category', component: () => import('@/views/article/CategoryDetail.vue'), props: true },
      // 动态列表页已下线（views/moments 只保留详情页）：列表场景由首页「发现」与 /feed 聚合流承担。
      // 旧地址 /moments 原本是「发动态」页，所以重定向到首页并带 compose=1，落地直接打开发布弹窗
      // （首页 onMounted / watch 处理，见 views/home/Index.vue 的 openComposeFromQuery）
      { path: 'moments', name: 'moments', redirect: { name: 'home', query: { compose: '1' } } },
      { path: 'moments/:id', name: 'moment-detail', component: () => import('@/views/moments/Detail.vue'), props: true },
      // 聚合流（关注 / 推荐 / 热门）：按关注关系算的个性化数据，必须登录
      // （后端 feed 接口同样是 path=all&type=login，见 app/model/auth-rules.go）
      { path: 'feed', name: 'feed', component: () => import('@/views/feed/Index.vue'), meta: { auth: true } },
      { path: 'links', name: 'links', component: () => import('@/views/links/Index.vue') },
      // 搜索 / 签到：原先是弹窗，现改为独立页面（支持 /search?q=xxx&scope=article 直达）
      { path: 'search', name: 'search', component: () => import('@/views/Search.vue') },
      { path: 'checkin', name: 'checkin', component: () => import('@/views/Checkin.vue') },
      // 消息通知：从用户中心独立出来，不再套 /user 的侧边导航，改为挂在 MainLayout 下
      // —— 这样站点的顶部导航与手机底部 Tab Bar 都会正常显示。
      // 旧地址 /user/notifications 保留重定向。
      {
        path: 'notifications',
        name: 'notifications',
        component: () => import('@/views/notifications/Index.vue'),
        meta: { auth: true, title: '消息通知' }
      },
      // 勋章馆 / 佩戴榜：公开的荣誉墙（按稀有度分组，带持有人数与正在佩戴的用户）
      { path: 'badges', name: 'badges', component: () => import('@/views/badge/Index.vue') },
      { path: 'goods', name: 'goods', component: () => import('@/views/goods/Index.vue') },
      { path: 'about', name: 'about', component: () => import('@/views/page/Index.vue'), props: { pageKey: 'about' } },
      // 用户协议 / 隐私协议：内容取自站点配置 Mellow_functions.auth_dialog_agreement
      // （与登录弹窗里的协议弹窗同一份内容；必须注册在下面的 /:key 兜底路由之前）
      {
        path: 'agreement',
        name: 'agreement',
        component: () => import('@/views/Agreement.vue'),
        props: { type: 'user' },
        // 标签标题会显示成「用户协议 - 站点名」（与页面 H1 一致）
        meta: { title: '用户协议' }
      },
      {
        path: 'privacy',
        name: 'privacy',
        component: () => import('@/views/Agreement.vue'),
        props: { type: 'privacy' },
        meta: { title: '隐私协议' }
      },
      // 用户主页 /author/:id（必须在 /:key 之前，避免被兜底路由拦截）
      { path: 'author/:id', name: 'author', component: () => import('@/views/user/Author.vue'), props: true },
      // 独立页面 /page/:id：按 id 直达（作者页 / 评论通知 / 打赏通知都用 id 形式）
      { path: 'page/:id', name: 'page-detail', component: () => import('@/views/page/Index.vue'), props: true },
      // 独立页面 /:key（必须放在最后，避免与其他固定路径冲突）
      { path: ':key', name: 'page', component: () => import('@/views/page/Index.vue'), props: (route) => ({ pageKey: route.params.key }) },
    ]
  },
  // 用户中心：与 /admin 一样是独立于前台 MainLayout 的顶级路由，拥有自己的完整布局
  {
    path: '/user',
    component: () => import('@/views/user/Layout.vue'),
    meta: { auth: true },
    children: [
      // 「我的」聚合页：手机端底栏「我的」的落地页（导航入口 + 信息展示，不承载具体的设置表单）
    { path: '', name: 'user-home', component: () => import('@/views/user/Index.vue'), meta: { title: '我的' } },
      { path: 'profile', name: 'user-profile', component: () => import('@/views/user/Profile.vue') },
      { path: 'settings', name: 'user-settings', component: () => import('@/views/user/Settings.vue') },
      // 「联系方式」页已下线：邮箱 / 手机号改绑并入「账号安全」，个人网站设置并入「个人资料」；
      // 旧地址保留为重定向，避免书签 / 外部链接 404
      { path: 'contact', name: 'user-contact', redirect: '/user/profile' },
      { path: 'security', name: 'user-security', component: () => import('@/views/user/Security.vue') },
      { path: 'exp', name: 'user-exp', component: () => import('@/views/user/Exp.vue') },
      { path: 'integral', name: 'user-integral', component: () => import('@/views/user/Integral.vue') },
      // 我的装扮：查看/佩戴已拥有的头像框与头衔（商城只负责兑换，管理集中在这里）
      { path: 'decorations', name: 'user-decorations', component: () => import('@/views/user/Decorations.vue') },
      // 我的勋章：已获得的勋章 + 未完成的任务条件与进度
      { path: 'badges', name: 'user-badges', component: () => import('@/views/user/Badges.vue') },
      // 我的认证：按后台配置的认证类型提交申请、查看审核进度与已通过的认证信息
      { path: 'verification', name: 'user-verification', component: () => import('@/views/user/Verification.vue') },
      // 消息通知已独立为 /notifications（挂在站点布局下，不带用户中心侧栏），旧地址保留重定向
      { path: 'notifications', redirect: '/notifications' },
      // 消息已合并为 /chat：旧地址保留重定向，
      // 书签、站内旧链接、?to=<uid> 直达参数都不会失效
      {
        path: 'messages',
        redirect: (to) => (to.query.to ? `/chat/${to.query.to}` : '/chat')
      },
      // 拉黑与举报：我的拉黑列表（可解除）+ 我提交的举报与处理进展（可撤回）
      { path: 'blocks', name: 'user-blocks', component: () => import('@/views/user/Blocks.vue') },
      // 站点配置（管理员）：已迁至后台 /admin/system 的「网站设置」模块，旧地址保留为重定向
      {
        path: 'site',
        name: 'user-site',
        redirect: { path: '/admin/system', query: { tab: 'site' } },
        meta: { auth: true }
      }
    ]
  },
  // 创作中心：与 /admin 一样是独立于前台 MainLayout 的顶级路由，拥有自己的完整布局
  {
    path: '/manage',
    component: () => import('@/views/manage/Layout.vue'),
    meta: { auth: true },
    children: [
      { path: '', name: 'manage', component: () => import('@/views/manage/Index.vue') },
      { path: 'moments', name: 'manage-moments', component: () => import('@/views/manage/Moments.vue') },
      { path: 'posts', name: 'manage-posts', component: () => import('@/views/manage/Posts.vue') },
      { path: 'posts/write', name: 'manage-post-write', component: () => import('@/views/manage/PostWrite.vue') },
      { path: 'posts/edit/:id', name: 'manage-post-edit', component: () => import('@/views/manage/PostWrite.vue') },
      { path: 'links', name: 'manage-links', component: () => import('@/views/manage/Links.vue') }
    ]
  },
  {
    path: '/admin',
    component: () => import('@/views/admin/Layout.vue'),
    // requiresAdmin：进入后台前必须回后端确认一次「有没有后台页面权限」，
    // 只靠组件内的 setup 检查不够（localStorage 里的 user 可能是旧快照，也可能被手工改过）
    meta: { auth: true, requiresAdmin: true },
    children: [
      { path: '', redirect: '/admin/dashboard' },
      { path: 'dashboard', name: 'admin-dashboard', component: () => import('@/views/admin/Dashboard.vue'), meta: { title: '概览' } },
      // ==================== 文章管理 ====================
      // AdminHub 负责标签页，子路由承载具体功能；列表页外层 Articles.vue 还带统计卡片
      {
        path: 'article',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'article' },
        meta: { title: '文章管理' },
        children: [
          {
            path: '',
            component: () => import('@/views/admin/Articles.vue'),
            children: [
              { path: '', name: 'admin-article', component: () => import('@/views/admin/ArticleList.vue'), meta: { title: '文章管理' } }
            ]
          },
          { path: 'group', name: 'admin-article-group', component: () => import('@/views/admin/ArticleGroup.vue'), meta: { title: '文章分类' } },
          { path: 'write', name: 'admin-article-write', component: () => import('@/views/admin/ArticleWrite.vue'), meta: { title: '撰写文章' } },
          { path: 'edit/:id', name: 'admin-article-edit', component: () => import('@/views/admin/ArticleWrite.vue'), meta: { title: '编辑文章' } }
        ]
      },
      // ==================== 用户管理 ====================
      {
        path: 'users',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'users' },
        meta: { title: '用户管理' },
        children: [
          {
            path: '',
            component: () => import('@/views/admin/Users.vue'),
            children: [
              { path: '', name: 'admin-users', component: () => import('@/views/admin/UserList.vue'), meta: { title: '用户管理' } }
            ]
          },
          { path: 'certifications', name: 'admin-certifications', component: () => import('@/views/admin/Certifications.vue'), meta: { title: '用户认证' } },
          { path: 'cert-types', name: 'admin-cert-types', component: () => import('@/views/admin/CertTypes.vue'), meta: { title: '认证类型' } },
          { path: 'level', name: 'admin-level', component: () => import('@/views/admin/Level.vue'), meta: { title: '等级管理' } },
          { path: 'exp', name: 'admin-exp', component: () => import('@/views/admin/Exp.vue'), meta: { title: '经验管理' } },
          { path: 'badges', name: 'admin-badges', component: () => import('@/views/admin/Badges.vue'), meta: { title: '勋章管理' } }
        ]
      },
      // ==================== 评论管理 ====================
      {
        path: 'comment',
        component: () => import('@/views/admin/Comments.vue'),
        meta: { title: '评论管理' },
        children: [
          { path: '', name: 'admin-comment', component: () => import('@/views/admin/CommentList.vue'), meta: { title: '评论管理' } }
        ]
      },
      // ==================== 内容管理 ====================
      {
        path: 'content',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'content' },
        meta: { title: '内容管理' },
        children: [
          { path: '', redirect: '/admin/content/moments' },
          {
            path: 'moments',
            component: () => import('@/views/admin/Moments.vue'),
            children: [
              { path: '', name: 'admin-moments', component: () => import('@/views/admin/MomentList.vue'), meta: { title: '动态管理' } }
            ]
          },
          { path: 'placard', name: 'admin-placard', component: () => import('@/views/admin/Placard.vue'), meta: { title: '公告管理' } },
          { path: 'banner', name: 'admin-banner', component: () => import('@/views/admin/Banner.vue'), meta: { title: '轮播管理' } },
          { path: 'tags', name: 'admin-tags', component: () => import('@/views/admin/TagList.vue'), meta: { title: '话题管理' } }
        ]
      },
      // ==================== 独立页面 ====================
      {
        path: 'pages',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'pages' },
        meta: { title: '独立页面' },
        children: [
          { path: '', name: 'admin-pages', component: () => import('@/views/admin/PageList.vue'), meta: { title: '独立页面' } },
          { path: 'write', name: 'admin-pages-write', component: () => import('@/views/admin/PageWrite.vue'), meta: { title: '撰写独立页面' } },
          { path: 'edit/:id', name: 'admin-pages-edit', component: () => import('@/views/admin/PageWrite.vue'), meta: { title: '编辑独立页面' } }
        ]
      },
      // ==================== 文档库 ====================
      // 文档站管理：文档库（项目列表） → 项目内目录/页面 → 独立整页编辑器
      // 三层平级路由：adminMenuOfPath() 会把它们都归到「文档库」，共用一个权限点
      { path: 'docs', name: 'admin-docs', component: () => import('@/views/admin/DocsLibrary.vue'), meta: { title: '文档库' } },
      { path: 'docs/:slug', name: 'admin-docs-project', component: () => import('@/views/admin/Docs.vue'), meta: { title: '文档管理' } },
      { path: 'docs/:slug/write', name: 'admin-docs-write', component: () => import('@/views/admin/DocsWrite.vue'), meta: { title: '新建文档' } },
      { path: 'docs/:slug/:id/edit', name: 'admin-docs-edit', component: () => import('@/views/admin/DocsWrite.vue'), meta: { title: '编辑文档' } },
      // ==================== 商城管理 ====================
      {
        path: 'goods',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'goods' },
        meta: { title: '商城管理' },
        children: [
          { path: '', name: 'admin-goods', component: () => import('@/views/admin/Goods.vue'), meta: { title: '商品管理' } },
          { path: 'integral', name: 'admin-integral', component: () => import('@/views/admin/Integral.vue'), meta: { title: '积分管理' } },
          { path: 'decoration', name: 'admin-decoration', component: () => import('@/views/admin/Decoration.vue'), meta: { title: '装扮管理' } },
          { path: 'checkin', name: 'admin-checkin', component: () => import('@/views/admin/Checkin.vue'), meta: { title: '签到管理' } }
        ]
      },
      // ==================== 消息通知 ====================
      { path: 'message', name: 'admin-message', component: () => import('@/views/admin/Message.vue'), meta: { title: '消息管理' } },
      // ==================== 聊天管理 ====================
      // 私信 / 公共群聊：命中词库的留证审核（chat-moderation）+ 群聊运营（room）
      {
        path: 'chat',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'chat' },
        meta: { title: '聊天管理' },
        children: [
          { path: '', redirect: '/admin/chat/conversations' },
          { path: 'conversations', name: 'admin-chat-conversations', component: () => import('@/views/admin/ChatConversations.vue'), meta: { title: '私信会话' } },
          { path: 'moderation', name: 'admin-chat-moderation', component: () => import('@/views/admin/ChatModeration.vue'), meta: { title: '聊天审核' } },
          { path: 'room', name: 'admin-chat-room', component: () => import('@/views/admin/RoomMessages.vue'), meta: { title: '群聊消息' } },
          { path: 'groups', name: 'admin-chat-groups', component: () => import('@/views/admin/RoomGroups.vue'), meta: { title: '群聊管理' } },
          { path: 'mutes', name: 'admin-chat-mutes', component: () => import('@/views/admin/RoomMutes.vue'), meta: { title: '禁言名单' } },
          { path: 'settings', name: 'admin-chat-settings', component: () => import('@/views/admin/RoomSettings.vue'), meta: { title: '群聊设置' } }
        ]
      },
      // ==================== 举报管理 ====================
      { path: 'reports', name: 'admin-reports', component: () => import('@/views/admin/Reports.vue'), meta: { title: '举报管理' } },
      // ==================== 友链管理 ====================
      {
        path: 'links',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'links' },
        meta: { title: '友链管理' },
        children: [
          {
            path: '',
            component: () => import('@/views/admin/Links.vue'),
            children: [
              { path: '', name: 'admin-links', component: () => import('@/views/admin/LinkList.vue'), meta: { title: '友链管理' } }
            ]
          },
          { path: 'group', name: 'admin-links-group', component: () => import('@/views/admin/LinkGroup.vue'), meta: { title: '友链分组' } }
        ]
      },
      // ==================== 附件管理 ====================
      { path: 'attachment', name: 'admin-attachment', component: () => import('@/views/admin/Attachment.vue'), meta: { title: '附件管理' } },
      // ==================== 数据统计 ====================
      { path: 'stats', name: 'admin-stats', component: () => import('@/views/admin/Stats.vue'), meta: { title: '数据统计' } },
      // ==================== 系统配置 ====================
      {
        path: 'system',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'system' },
        meta: { title: '系统配置' },
        children: [
          { path: '', name: 'admin-system', component: () => import('@/views/admin/SystemConfig.vue'), meta: { title: '系统设置' } },
          { path: 'theme', name: 'admin-theme', component: () => import('@/views/admin/Theme.vue'), meta: { title: '主题配色' } }
        ]
      },
      // ==================== 安全中心 ====================
      {
        path: 'security',
        component: () => import('@/views/admin/AdminHub.vue'),
        props: { section: 'security' },
        meta: { title: '安全中心' },
        children: [
          // 默认落在「内容审核」：敏感词库与云端机审是安全中心最常看的日常页面
          { path: '', redirect: '/admin/security/content' },
          // 内容审核：敏感词库 + 云端机审（原在「系统配置 → 互动配置」下，合并到安全域）
          {
            path: 'content',
            name: 'admin-security-content',
            component: () => import('@/views/admin/security/ContentSecurity.vue'),
            meta: { title: '内容审核' }
          },
          // 安全与限流：密钥校验 / QPS / 自动封禁 / 滑块（原在「系统配置 → 账号与登录」下）
          {
            path: 'policy',
            name: 'admin-security-policy',
            component: () => import('@/views/admin/security/SecurityPolicy.vue'),
            meta: { title: '安全与限流' }
          },
          { path: 'rules', name: 'admin-auth-rules', component: () => import('@/views/admin/AuthRules.vue'), meta: { title: '权限规则' } },
          { path: 'group', name: 'admin-auth-group', component: () => import('@/views/admin/AuthGroup.vue'), meta: { title: '权限分组' } },
          { path: 'keys', name: 'admin-api-keys', component: () => import('@/views/admin/ApiKeys.vue'), meta: { title: '接口密钥' } },
          { path: 'ip-black', name: 'admin-ip-black', component: () => import('@/views/admin/IpBlack.vue'), meta: { title: 'IP 黑名单' } },
          { path: 'ip-white', name: 'admin-ip-white', component: () => import('@/views/admin/IpWhite.vue'), meta: { title: 'IP 白名单' } },
          { path: 'qps', name: 'admin-qps-warn', component: () => import('@/views/admin/QpsWarn.vue'), meta: { title: 'QPS 预警' } },
          // 后台页面管理：维护菜单/页面权限点（原在「系统配置」下，与权限规则同属安全域）
          { path: 'pages', name: 'admin-auth-pages', component: () => import('@/views/admin/AuthPages.vue'), meta: { title: '后台页面管理' } }
        ]
      },
      // ==================== 查看日志 ====================
      // 日志解析：只读查看 runtime/ 下的系统日志与通知日志（后端 app/api/controller/log.go）
      { path: 'logs', name: 'admin-logs', component: () => import('@/views/admin/Logs.vue'), meta: { title: '日志' } },

      // ==================== 旧地址兼容 ====================
      // 菜单合并（37 → 15）后，老书签 / 站内历史链接仍按原路径访问，这里统一重定向到新位置
      { path: 'placard', redirect: '/admin/content/placard' },
      { path: 'banner', redirect: '/admin/content/banner' },
      { path: 'tags', redirect: '/admin/content/tags' },
      { path: 'moments', redirect: '/admin/content/moments' },
      { path: 'certifications', redirect: '/admin/users/certifications' },
      { path: 'cert-types', redirect: '/admin/users/cert-types' },
      { path: 'level', redirect: '/admin/users/level' },
      { path: 'exp', redirect: '/admin/users/exp' },
      { path: 'badges', redirect: '/admin/users/badges' },
      { path: 'integral', redirect: '/admin/goods/integral' },
      { path: 'decoration', redirect: '/admin/goods/decoration' },
      { path: 'checkin', redirect: '/admin/goods/checkin' },
      { path: 'theme', redirect: '/admin/system/theme' },
      { path: 'system/pages', redirect: '/admin/security/pages' },
      { path: 'auth/pages', redirect: '/admin/security/pages' },
      { path: 'auth/rules', redirect: '/admin/security/rules' },
      { path: 'auth/group', redirect: '/admin/security/group' },
      { path: 'api/keys', redirect: '/admin/security/keys' },
      { path: 'ip/black', redirect: '/admin/security/ip-black' },
      { path: 'ip/white', redirect: '/admin/security/ip-white' },
      { path: 'qps/warn', redirect: '/admin/security/qps' }
    ]
  },
  {
    path: '/auth',
    component: () => import('@/layouts/AuthLayout.vue'),
    children: [
      { path: 'login', name: 'login', component: () => import('@/views/auth/Login.vue') },
      { path: 'register', name: 'register', component: () => import('@/views/auth/Register.vue') },
      { path: 'forgot', name: 'forgot', component: () => import('@/views/auth/Forgot.vue') },
      // 第三方登录回调页（QQ / GitHub / Gitee 授权后跳回，见 views/auth/OAuth.vue）
      // 必须与其他 auth 子页同一个回调地址，因此不带 query（平台信息记在 sessionStorage）
      { path: 'oauth', name: 'auth-oauth', component: () => import('@/views/auth/OAuth.vue') }
    ]
  },
  // 小黑屋（封禁公示）：独立于前台 MainLayout 的顶级路由，页面自己撑满视口，
  // 不带左右栏 / 顶栏 / 页脚等任何布局与导航组件（与 /404 同级，路径仍是 /blackroom）
  { path: '/blackroom', name: 'blackroom', component: () => import('@/views/Blackroom.vue'), meta: { title: '小黑屋' } },
  { path: '/:pathMatch(.*)*', name: 'not-found', component: () => import('@/views/NotFound.vue') }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(_to, _from, saved) {
    if (saved) return saved
    return { top: 0 }
  }
})

router.beforeEach(async (to, from, next) => {
  progressStart()

  // 需要登录：本地没有登录态就直接去登录页（后端仍会再校验 token）
  if (to.matched.some((r) => r.meta?.auth)) {
    const userStr = localStorage.getItem('blog_user')
    if (!userStr || userStr === 'null') {
      next({ name: 'login', query: { redirect: to.fullPath } })
      return
    }
  }

  // 需要后台权限：必须以服务端返回的用户权限为准
  //
  // 之前只在 views/admin/Layout.vue 的 setup 里用 localStorage 里的 user 判断，
  // 普通用户（权限组 pages 为空）能直接输 /admin 进入后台外壳：
  // 缓存里的 user 可能是旧快照（曾被授权过），也可能被手工改过 result.auth.pages.hash。
  if (to.matched.some((r) => r.meta?.requiresAdmin)) {
    const entering = !from.matched.some((r) => r.meta?.requiresAdmin)

    let allowed = false
    try {
      const { useUserStore } = await import('@/stores/user')
      const userStore = useUserStore()

      // 从后台外部进来时刷新一次权限；后台内部切页不重复请求
      if (entering || !userStore.user) {
        await userStore.checkLoginState().catch(() => null)
      }
      allowed = canEnterAdmin(userStore.user)
    } catch {
      // 校验链路本身出错时保守拒绝：没确认过的权限不放进后台
      allowed = false
    }

    if (!allowed) {
      toast.warning('当前账号没有后台权限')
      next({ path: '/', replace: true })
      return
    }

    // 有后台页面权限 ≠ 后台入口已解锁：后台是独立会话（cookie INIS_ADMIN_TOKEN），
    // 必须重新输入一次密码才签发。会话失效时这里会弹密码框（见 utils/adminSession.js），
    // 用户取消或密码不对则不进后台。
    const { ensureAdminSession } = await import('@/utils/adminSession')
    const unlocked = await ensureAdminSession()
    if (!unlocked) {
      toast.warning('未通过后台验证')
      next({ path: '/', replace: true })
      return
    }
  }

  next()
})

router.afterEach((to) => {
  progressDone()
  // 浏览器标签标题：meta.title（后台各页已用它作为页面名）- 站点名
  // 每次切换路由都会重算，因此不会出现「停在上一页标题」的情况，见 utils/pageTitle.js
  applyRouteTitle(to)
  // 灯箱是全局单例状态（utils/lightbox），跨布局切换时可能残留：
  // 例如在后台点了轮播图 / 举报证据图，当时没有灯箱组件接管，状态一直停在 visible，
  // 切回前台布局后就会「凭空」弹出来。这里每次路由切换都强制关闭。
  closeLightbox()
})

export default router