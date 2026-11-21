/**
 * 右侧边栏的模块化配置（Mellow_functions.sidebar）
 *
 * 设计目标：让「右侧栏有哪些卡片、什么顺序、叫什么、显示几条」由站点配置决定，
 * 而不是写死在 components/SidebarRight.vue 里。前台与后台共用本文件的注册表，
 * 新增一种模块只需要在这里加一条 meta + 在 SidebarRight 里加一个分支。
 *
 * 配置结构：
 * {
 *   enabled: true,
 *   modules: [
 *     {
 *       id: 1,                       // 内部唯一 id（仅用于排序 / v-for key）
 *       type: 'articleRank',         // 见 SIDEBAR_TYPES
 *       title: '随机文章',            // 留空则用类型默认标题
 *       icon: 'bi bi-shuffle',       // 留空则用类型默认图标
 *       visible: true,               // false 时前台不渲染（配置仍保留）
 *       collapsible: false,          // 允许用户折叠
 *       collapsed: false,            // 默认是否折叠
 *       limit: 6,                    // 列表类模块的条数
 *       moreText: '', moreUrl: '',   // 右上角「更多」入口
 *       content: '',                 // html 模块的正文
 *       items: []                    // links 模块的链接
 *     }
 *   ]
 * }
 */

/** 模块类型注册表：前台渲染分支、后台表单字段都读这里 */
export const SIDEBAR_TYPES = [
  {
    value: 'user',
    name: '用户卡片',
    icon: 'bi-person-circle',
    desc: '头像、昵称、快捷入口与签到（未登录时显示登录引导）',
    fields: ['title', 'icon', 'collapsible'],
    defaults: { title: '', icon: '', collapsible: false }
  },
  {
    value: 'notify',
    name: '未读通知',
    icon: 'bi-bell',
    desc: '最新一条未读消息预览（需登录）',
    fields: ['title', 'icon', 'moreUrl', 'collapsible'],
    defaults: { title: '消息通知', icon: 'bi-bell', moreUrl: '/notifications', collapsible: false }
  },
  {
    value: 'articleRank',
    name: '文章列表',
    icon: 'bi-shuffle',
    desc: '随机推荐文章，条数可调',
    fields: ['title', 'icon', 'limit', 'moreText', 'moreUrl', 'collapsible'],
    defaults: { title: '随机文章', icon: 'bi-shuffle', limit: 6, moreText: '', moreUrl: '', collapsible: false }
  },
  {
    value: 'comments',
    name: '最新评论',
    icon: 'bi-chat-dots',
    desc: '全站最新评论，条数可调',
    fields: ['title', 'icon', 'limit', 'moreText', 'moreUrl', 'collapsible'],
    defaults: { title: '最新评论', icon: 'bi-chat-dots', limit: 6, moreText: '', moreUrl: '', collapsible: false }
  },
  {
    value: 'categories',
    name: '分类导航',
    icon: 'bi-folder2',
    desc: '文章分类标签云',
    fields: ['title', 'icon', 'limit', 'moreText', 'moreUrl', 'collapsible'],
    defaults: { title: '分类导航', icon: 'bi-folder2', limit: 20, moreText: '查看全部', moreUrl: '/categories', collapsible: false }
  },
  {
    value: 'topics',
    name: '话题云',
    icon: 'bi-hash',
    desc: '热门话题标签云',
    fields: ['title', 'icon', 'limit', 'moreText', 'moreUrl', 'collapsible'],
    defaults: { title: '话题云', icon: 'bi-hash', limit: 20, moreText: '话题广场', moreUrl: '/topics', collapsible: false }
  },
  {
    value: 'archive',
    name: '文章归档',
    icon: 'bi-archive',
    desc: '全部文章数与归档入口',
    fields: ['title', 'icon', 'moreText', 'moreUrl', 'collapsible'],
    defaults: { title: '文章归档', icon: 'bi-archive', moreText: '', moreUrl: '', collapsible: false }
  },
  {
    value: 'stats',
    name: '站点信息',
    icon: 'bi-bar-chart',
    desc: '文章 / 动态 / 话题 / 友链数量与运行天数',
    fields: ['title', 'icon', 'collapsible'],
    defaults: { title: '站点信息', icon: 'bi-bar-chart', collapsible: false }
  },
  {
    value: 'html',
    name: '自定义内容',
    icon: 'bi-code-slash',
    desc: '任意 HTML：公告、友链、广告位、备案信息都可以',
    fields: ['title', 'icon', 'content', 'collapsible'],
    defaults: { title: '自定义模块', icon: 'bi-code-slash', content: '', collapsible: false }
  },
  {
    value: 'links',
    name: '自定义链接',
    icon: 'bi-link-45deg',
    desc: '图标 + 名称 + 链接的列表，可跳站内或站外',
    fields: ['title', 'icon', 'items', 'collapsible'],
    defaults: { title: '友情链接', icon: 'bi-link-45deg', items: [], collapsible: false }
  }
]

/** 默认侧边栏：与改造前的顺序 / 文案保持一致，站点没配过时行为不变 */
export const SIDEBAR_DEFAULT_MODULES = [
  { id: 1, type: 'user' },
  { id: 2, type: 'notify' },
  { id: 3, type: 'articleRank' },
  { id: 4, type: 'comments' },
  { id: 5, type: 'categories' },
  { id: 6, type: 'topics' },
  { id: 7, type: 'archive' },
  { id: 8, type: 'stats' }
]

/** 取类型定义（未知类型回退到自定义内容，保证配置坏了也不会整栏消失） */
export function sidebarTypeMeta(type) {
  return SIDEBAR_TYPES.find((item) => item.value === type) || SIDEBAR_TYPES[SIDEBAR_TYPES.length - 1]
}

/** 该类型是否需要拉列表数据（前台按需请求，避免全量打接口） */
export function sidebarNeeds(type, key) {
  const meta = SIDEBAR_TYPES.find((item) => item.value === type)
  return Boolean(meta && meta.fields.includes(key))
}

/** bootstrap-icons 需要 `bi` 基础类 + `bi-xxx` 图标类，这里统一补齐前缀 */
function normalizeIcon(icon) {
  const raw = String(icon || '').trim()
  if (!raw) return ''
  return /(^|\s)bi(\s|$)/.test(raw) ? raw : `bi ${raw}`
}

function toInt(value, fallback) {
  const num = Number(value)
  if (!Number.isFinite(num) || num <= 0) return fallback
  return Math.min(Math.floor(num), 100)
}

function normalizeLinks(items) {
  if (!Array.isArray(items)) return []
  return items
    .map((item, index) => {
      const name = String(item?.name || '').trim()
      const url = String(item?.url || '').trim()
      if (!name || !url) return null
      return {
        id: Number(item?.id) || index + 1,
        name,
        icon: String(item?.icon || '').trim(),
        url,
        target: item?.target === 'external' ? 'external' : 'internal'
      }
    })
    .filter(Boolean)
}

/** 归一化单个模块（供前台渲染与后台表单共用） */
export function normalizeSidebarModule(raw, index = 0) {
  if (!raw || typeof raw !== 'object') return null
  const meta = sidebarTypeMeta(raw.type)
  const defaults = meta.defaults || {}
  const type = SIDEBAR_TYPES.some((item) => item.value === raw.type) ? raw.type : 'html'

  return {
    id: Number(raw.id) || index + 1,
    type,
    /* 标题 / 图标：留空时由前台按类型取默认值 */
    title: String(raw.title ?? defaults.title ?? '').trim(),
    icon: normalizeIcon(raw.icon ?? defaults.icon ?? ''),
    visible: raw.visible !== false,
    collapsible: raw.collapsible === true,
    collapsed: raw.collapsed === true,
    limit: toInt(raw.limit ?? defaults.limit, defaults.limit || 0),
    moreText: String(raw.moreText ?? defaults.moreText ?? '').trim(),
    moreUrl: String(raw.moreUrl ?? defaults.moreUrl ?? '').trim(),
    content: String(raw.content ?? defaults.content ?? ''),
    items: normalizeLinks(raw.items ?? defaults.items)
  }
}

/** 归一化整份侧边栏配置 */
export function normalizeSidebarConfig(raw) {
  const src = raw && typeof raw === 'object' ? raw : {}
  const list = Array.isArray(src.modules) && src.modules.length ? src.modules : SIDEBAR_DEFAULT_MODULES
  const modules = list.map((item, index) => normalizeSidebarModule(item, index)).filter(Boolean)

  // 去重 id：老配置里可能有重复 id，会让 v-for key 冲突
  const used = new Set()
  modules.forEach((module, index) => {
    let id = module.id
    while (used.has(id)) id += 1
    used.add(id)
    module.id = id
    module.order = index
  })

  return {
    enabled: src.enabled !== false,
    modules
  }
}

/** 后台「重置为默认」用：一份可直接编辑的默认配置 */
export function defaultSidebarConfig() {
  return normalizeSidebarConfig({ enabled: true, modules: SIDEBAR_DEFAULT_MODULES })
}

/** 新建一个模块（id 取当前最大值 +1） */
export function createSidebarModule(type, modules = []) {
  const meta = sidebarTypeMeta(type)
  const id = modules.reduce((max, item) => Math.max(max, Number(item?.id) || 0), 0) + 1
  return normalizeSidebarModule({ id, type, ...(meta.defaults || {}) }, modules.length)
}

/** 模块展示用的标题 / 图标（配置留空时回退到类型默认值） */
export function sidebarModuleTitle(module) {
  const meta = sidebarTypeMeta(module?.type)
  return String(module?.title || meta.defaults?.title || meta.name || '')
}

export function sidebarModuleIcon(module) {
  const meta = sidebarTypeMeta(module?.type)
  return String(module?.icon || meta.defaults?.icon || meta.icon || 'bi bi-grid')
}

/** 是否需要在没有标题时仍然渲染外壳（用户卡片/站点信息等自带标题栏的模块） */
export function sidebarHasHeader(module) {
  return String(sidebarModuleTitle(module) || '').length > 0
}
