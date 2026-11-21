/**
 * 后台菜单（侧边栏 / 概览页快捷跳转）共享口径
 *
 * 菜单合并（37 → 15）之后，侧边栏只保留一级模块入口，子功能由
 * views/admin/AdminHub.vue 的标签页承载。这里负责「分组顺序 + 权限过滤 + 图标解析」：
 * - 侧边栏（views/admin/Layout.vue）与概览页快捷跳转（views/admin/Dashboard.vue）
 *   共用同一份结果，避免出现「侧边栏有、快捷入口没有」或权限过滤不一致；
 * - 新增一级模块时只需在 ADMIN_MENU_GROUPS 里补一条（想固定在「概览」下方则加到 ADMIN_PINNED）。
 *
 * 每个菜单项的字段：
 *   path   菜单跳转地址（新的一级路径，旧地址在 router 里统一 redirect 过来）
 *   pages  该模块包含的所有页面路径（含合并前的旧路径）
 *          → 权限判定：只要勾选了其中任意一个页面，整个入口就可见（旧库已勾选的权限不会失效）
 *          → 存在性判定：只要有一条 auth-pages 记录存在，入口就显示
 *   name / icon  前端兜底文案与图标；若库里有同 path 的页面记录，则以库里的为准（保留管理员的改名 / 换图标）
 */
import { adminAllowedPaths } from '@/utils/helper'
import { cache } from '@/utils/cache'

// 分组定义：与参考主题一致（创作 / 管理 / 安全）
export const ADMIN_MENU_GROUPS = [
  {
    name: 'create',
    label: '创作',
    items: [
      {
        path: '/admin/article',
        name: '文章管理',
        icon: 'bi bi-file-earmark-text',
        pages: ['/admin/article', '/admin/article/write', '/admin/article/group']
      },
      {
        path: '/admin/pages',
        name: '独立页面',
        icon: 'bi bi-window',
        pages: ['/admin/pages', '/admin/pages/write']
      },
      {
        path: '/admin/docs',
        name: '文档库',
        icon: 'bi bi-journal-bookmark',
        pages: ['/admin/docs']
      },
      {
        path: '/admin/content',
        name: '内容管理',
        icon: 'bi bi-collection-play',
        pages: ['/admin/moments', '/admin/placard', '/admin/banner', '/admin/tags']
      }
    ]
  },
  {
    name: 'manage',
    label: '管理',
    items: [
      {
        path: '/admin/users',
        name: '用户管理',
        icon: 'bi bi-people',
        pages: ['/admin/users', '/admin/certifications', '/admin/cert-types', '/admin/level', '/admin/exp', '/admin/badges']
      },
      {
        path: '/admin/comment',
        name: '评论管理',
        icon: 'bi bi-chat-square-text',
        pages: ['/admin/comment']
      },
      {
        path: '/admin/goods',
        name: '商城管理',
        icon: 'bi bi-bag',
        pages: ['/admin/goods', '/admin/integral', '/admin/decoration', '/admin/goods/renew', '/admin/checkin']
      },
      {
        path: '/admin/message',
        name: '消息通知',
        icon: 'bi bi-bell',
        pages: ['/admin/message']
      },
      {
        // 聊天管理：私信 / 公共群聊的命中留证审核 + 群聊消息、禁言、资料
        path: '/admin/chat',
        name: '聊天管理',
        icon: 'bi bi-chat-square-dots',
        pages: ['/admin/chat']
      },
      {
        path: '/admin/reports',
        name: '举报管理',
        icon: 'bi bi-flag',
        pages: ['/admin/reports']
      },
      {
        path: '/admin/links',
        name: '友链管理',
        icon: 'bi bi-link-45deg',
        pages: ['/admin/links', '/admin/links/group']
      },
      {
        path: '/admin/attachment',
        name: '附件管理',
        icon: 'bi bi-folder2-open',
        pages: ['/admin/attachment']
      },
      {
        path: '/admin/system',
        name: '系统配置',
        icon: 'bi bi-gear',
        pages: ['/admin/system', '/admin/theme']
      },
      {
        path: '/admin/logs',
        name: '查看日志',
        icon: 'bi bi-journal-text',
        pages: ['/admin/logs']
      }
    ]
  },
  {
    name: 'security',
    label: '安全',
    items: [
      {
        path: '/admin/security',
        name: '安全中心',
        icon: 'bi bi-shield-lock',
        pages: [
          '/admin/security',
          // 内容审核（敏感词库 + 云端机审）、安全与限流（密钥 / QPS / 滑块）
          '/admin/security/content', '/admin/security/policy',
          '/admin/auth/rules', '/admin/auth/group', '/admin/api/keys',
          '/admin/ip/black', '/admin/ip/white', '/admin/qps/warn',
          // 后台页面管理：维护菜单/页面权限点
          '/admin/auth/pages'
        ]
      }
    ]
  }
]

/**
 * 置顶入口：固定显示在侧边栏「概览」正下方，**不参与**创作 / 管理 / 安全分组
 * （侧边栏不显示它的分组标题；概览页快捷跳转里以 label 作为分组标题）
 * 数据统计是实时资源 / 数据库 / 在线用户，属于「看板」性质，与概览同级更顺手
 */
export const ADMIN_PINNED = {
  name: 'pinned',
  label: '置顶',
  items: [
    { path: '/admin/stats', name: '数据统计', icon: 'bi bi-graph-up-arrow', pages: ['/admin/stats'] }
  ]
}

// 置顶路径：从分组里排除，避免同一个页面上两处出现
const PINNED_PATHS = new Set(ADMIN_PINNED.items.map((item) => item.path))
const ALL_MENU_ITEMS = [...ADMIN_PINNED.items, ...ADMIN_MENU_GROUPS.flatMap((group) => group.items)]

// icon 名称 → Bootstrap Icons（仅用于展示；auth-pages 自带 svg 优先）
const FALLBACK_ICONS = {
  article: 'bi bi-file-earmark-text',
  group: 'bi bi-collection',
  user: 'bi bi-people',
  comment: 'bi bi-chat-square-text',
  bell: 'bi bi-bell',
  banner: 'bi bi-images',
  tag: 'bi bi-tag',
  level: 'bi bi-bar-chart',
  link: 'bi bi-link-45deg',
  system: 'bi bi-gear',
  open: 'bi bi-box-arrow-up-right',
  rule: 'bi bi-shield-check',
  key: 'bi bi-key',
  qps: 'bi bi-speedometer2',
  black: 'bi bi-shield-x',
  white: 'bi bi-shield-plus',
  file: 'bi bi-paperclip'
}

/**
 * 解析菜单图标：
 * 1. 后端已是 Bootstrap Icons 类名（"bi bi-bag" 或 "bi-bag"）→ 直接使用；
 * 2. 旧版短名（article / group …）→ 查 FALLBACK_ICONS 映射；
 * 3. 其余 → 兜底圆圈。
 * 这样新老数据库（auth-pages.icon 两种形态）都能正确展示。
 */
export function resolveAdminIcon(name) {
  if (typeof name !== 'string' || !name.trim()) return 'bi bi-circle'
  const value = name.trim()
  if (value.includes('bi-')) return value.startsWith('bi ') ? value : `bi ${value}`
  return FALLBACK_ICONS[value] || 'bi bi-circle'
}

/**
 * 菜单上下文：页面字典 + 允许访问的 path 集合（null = 拥有全部页面）
 */
function menuContext(user, flat) {
  const list = Array.isArray(flat) ? flat : []
  if (!list.length) return null

  return {
    allowedPaths: adminAllowedPaths(user, list),
    byPath: Object.fromEntries(list.map((p) => [p.path, p]))
  }
}

// 菜单项覆盖的页面路径（新一级 path + 合并前的旧 path）
function menuPages(menu) {
  const pages = Array.isArray(menu.pages) && menu.pages.length ? menu.pages : [menu.path]
  return pages.includes(menu.path) ? pages : [menu.path, ...pages]
}

/**
 * 判断某个菜单项是否对当前账号可见
 *
 * 存在性：库里至少有一条该模块的页面记录（老库只有旧 path，新库可能有新一级 path）
 * 权限：  勾选了该模块下任意一个页面即可见 —— 权限仍按 auth-pages 的细粒度记录存，
 *        界面按一级模块展示，避免老库已勾选的权限在菜单合并后失效
 */
function menuVisible(menu, allowedPaths, byPath) {
  const pages = menuPages(menu)
  if (!pages.some((path) => byPath[path])) return false
  return !allowedPaths || pages.some((path) => allowedPaths.has(path))
}

/**
 * 取出一批菜单项对应的展示数据（仅保留系统实际存在 + 当前账号有权限的）
 *
 * 名称 / 图标优先用库里的记录（管理员可在「后台页面管理」改名换图标），
 * 一级 path 尚未入库时回退到配置里的 name / icon。
 */
function pickMenuItems(items, allowedPaths, byPath) {
  return items
    .filter((item) => menuVisible(item, allowedPaths, byPath))
    .map((item) => {
      const page = byPath[item.path] || {}
      const svg = page.svg || (typeof page.icon === 'string' && page.icon.includes('<svg') ? page.icon : '')
      return {
        path: item.path,
        name: page.name || item.name || item.path,
        icon: svg ? '' : resolveAdminIcon(page.icon || item.icon),
        isSvg: !!svg,
        svg
      }
    })
}

/**
 * 构建后台菜单（已按当前账号权限过滤，仅保留系统实际存在的页面）
 *
 * 置顶页面（ADMIN_PINNED）会从分组里排除，它由 buildAdminPinnedItems 单独提供，
 * 避免同一个页面在侧边栏出现两次。
 *
 * @param {object} user 当前登录用户（权限取自 user.result.auth / user.auth）
 * @param {Array}  flat auth-pages 扁平列表
 * @returns {Array<{name:string,label:string,children:Array}>}
 */
export function buildAdminMenus(user, flat) {
  const ctx = menuContext(user, flat)
  if (!ctx) return []

  return ADMIN_MENU_GROUPS
    .map((group) => ({
      name: group.name,
      label: group.label,
      children: pickMenuItems(group.items.filter((item) => !PINNED_PATHS.has(item.path)), ctx.allowedPaths, ctx.byPath)
    }))
    .filter((group) => group.children.length)
}

/**
 * 构建置顶入口（固定显示在「概览」下方，与普通菜单同一套权限过滤 / 图标解析）
 */
export function buildAdminPinnedItems(user, flat) {
  const ctx = menuContext(user, flat)
  if (!ctx) return []

  return pickMenuItems(ADMIN_PINNED.items, ctx.allowedPaths, ctx.byPath)
}

/**
 * 路由 path → 所属的一级菜单项（配置，不是接口数据）
 *
 * 例：/admin/badges（旧地址）、/admin/users/badges（新地址）都归到「用户管理」。
 * 用于「最近访问」归一化与手输 URL 的权限校验。
 */
export function adminMenuOfPath(path) {
  if (typeof path !== 'string') return null
  let matched = null
  let matchedLen = -1
  for (const menu of ALL_MENU_ITEMS) {
    for (const prefix of menuPages(menu)) {
      if ((path === prefix || path.startsWith(`${prefix}/`)) && prefix.length > matchedLen) {
        matched = menu
        matchedLen = prefix.length
      }
    }
  }
  return matched
}

/**
 * 当前账号能否访问某个后台路径（按所属一级模块判定）
 *
 * @returns {boolean} 不属于任何一级模块的路径（如 /admin/dashboard）一律放行
 */
export function adminPathAllowed(user, flat, path) {
  const menu = adminMenuOfPath(path)
  if (!menu) return true

  const ctx = menuContext(user, flat)
  if (!ctx) return true

  return menuVisible(menu, ctx.allowedPaths, ctx.byPath)
}

// ---------- 最近访问（概览页快捷跳转专用，存 localStorage） ----------
const RECENT_KEY = 'admin-recent-paths'
const RECENT_LIMIT = 6

/**
 * 记录一次后台页面访问（只记菜单级 path，编辑页等子路由会被归一到菜单页）
 */
export function rememberAdminPath(path) {
  if (typeof path !== 'string' || !path.startsWith('/admin/') || path === '/admin/dashboard') return

  const history = getRecentAdminPaths().filter((item) => item !== path)
  history.unshift(path)
  cache.set(RECENT_KEY, history.slice(0, RECENT_LIMIT))
}

/**
 * 读取最近访问列表（最新在前）
 */
export function getRecentAdminPaths() {
  const value = cache.get(RECENT_KEY)
  if (!Array.isArray(value)) return []
  return value.filter((item) => typeof item === 'string' && item.startsWith('/admin/')).slice(0, RECENT_LIMIT)
}
