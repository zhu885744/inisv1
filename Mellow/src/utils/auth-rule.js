/**
 * 权限规则（auth-rules）领域公共逻辑
 *
 * 规则类型与请求方法的取值、展示文案、筛选条件统一在这里维护，
 * 列表与统计共用同一份定义，避免口径分叉。
 */

// 列表字段：hash 用于与权限分组的授权内容对照
export const AUTH_RULE_FIELD =
  'id,name,method,route,type,hash,cost,remark,create_time,update_time,delete_time'

/**
 * 规则类型：共三种（见 app/api/middleware/rule.go 的判定分支）
 * - common  公共接口：中间件直接放行，不校验登录态
 * - login   需登录：已登录即放行，不校验权限点
 * - default 默认：需具备对应权限点 [METHOD][route]，否则返回「无权限！」
 *
 * 说明：早期后端初始化数据误用过 type=root（exp/give、积分卡密、商品管理等），
 * 中间件从未对它做特殊判定，行为与 default 完全一致，属于错误的类型标注，
 * 种子数据已统一改为 default，老库由 InitAuthRules 的 normalizeAuthRuleTypes 纠正。
 * 为兼容尚未纠正的库，这里把 root 作为 default 的遗留别名处理（展示与配色同 default）。
 */
export const AUTH_RULE_TYPES = [
  { value: 'default', label: '默认（需权限点）', color: 'var(--text-muted)' },
  { value: 'common', label: '公共（免登录）', color: 'var(--success)' },
  { value: 'login', label: '需登录', color: '#0ea5e9' }
]

/** 遗留类型别名：root → default */
const LEGACY_TYPE_ALIAS = { root: 'default' }

/** 归一化类型值：空值按 default，root 视作 default */
export function authRuleTypeKey(item) {
  const type = String(item?.type || 'default').trim() || 'default'
  return LEGACY_TYPE_ALIAS[type] || type
}

// 请求方法（后端会把 method 统一转大写后参与 hash 计算）
export const AUTH_RULE_METHODS = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH']

/** 类型下拉 */
export const AUTH_RULE_TYPE_OPTIONS = [
  { value: '', label: '全部类型' },
  ...AUTH_RULE_TYPES.map(({ value, label }) => ({ value, label }))
]

/** 方法下拉 */
export const AUTH_RULE_METHOD_OPTIONS = [
  { value: '', label: '全部方法' },
  ...AUTH_RULE_METHODS.map((m) => ({ value: m, label: m }))
]

/** 搜索字段白名单（后端 like 参数只支持「字段名|值」单字段匹配） */
export const AUTH_RULE_SEARCH_FIELDS = [
  { value: 'name', label: '规则名称' },
  { value: 'route', label: '路由' },
  { value: 'remark', label: '备注' }
]

/**
 * 类型文案
 * 未知类型原样返回，避免把真实数据展示成别的类型（root 等遗留别名按 default 展示）
 */
export function authRuleTypeLabel(item) {
  const key = authRuleTypeKey(item)
  return AUTH_RULE_TYPES.find((t) => t.value === key)?.label || key
}

/** 方法统一转大写展示 */
export function authRuleMethodText(item) {
  return String(item?.method || 'GET').toUpperCase()
}

/**
 * 构造筛选条件（JSON 字符串）
 * 类型与方法可叠加；都为空时返回 null（调用方不要传 where）
 */
export function authRuleWhereJSON(type, method) {
  const where = {}
  if (type) where.type = type
  if (method) where.method = method
  return Object.keys(where).length ? JSON.stringify(where) : null
}

/** 搜索关键字清理：剔除会干扰后端 LIKE「字段名|值」格式的字符 */
export function sanitizeAuthRuleKeyword(keyword) {
  return String(keyword ?? '')
    .replace(/['"\\%_|]/g, '')
    .trim()
}

/* ============================================================
 * 权限点归类（权限分组页的「傻瓜式」勾选用）
 *
 * 后端每条规则 = [/api/<模块>/<动作>] + 请求方法，模块名与
 * app/model/auth-rules.go 的 names 映射一一对应。
 * 这里只是「展示层的归类」，不改数据库、不改 rules 的存储格式
 * （仍然是一串 hash），所以对已有权限组完全兼容。
 * ============================================================ */

/**
 * 模块 → 中文名 + 图标
 * 未登记的模块会自动兜底（直接用模块名），后端新增接口不会因此漏显示
 */
export const AUTH_RULE_MODULES = {
  article: { label: '文章', icon: 'bi bi-journal-text' },
  'article-group': { label: '文章分类', icon: 'bi bi-folder2' },
  comment: { label: '评论', icon: 'bi bi-chat-left-text' },
  moments: { label: '动态', icon: 'bi bi-chat-square-text' },
  pages: { label: '独立页面', icon: 'bi bi-file-earmark-text' },
  banner: { label: '轮播', icon: 'bi bi-images' },
  placard: { label: '公告', icon: 'bi bi-megaphone' },
  links: { label: '友链', icon: 'bi bi-link-45deg' },
  'links-group': { label: '友链分类', icon: 'bi bi-folder2-open' },
  tags: { label: '话题（管理）', icon: 'bi bi-hash' },
  topics: { label: '话题（互动）', icon: 'bi bi-chat-square-quote' },
  users: { label: '用户', icon: 'bi bi-people' },
  'user-follows': { label: '关注关系', icon: 'bi bi-person-plus' },
  'user-likes': { label: '点赞', icon: 'bi bi-hand-thumbs-up' },
  'user-collects': { label: '收藏', icon: 'bi bi-bookmark' },
  'user-blocks': { label: '拉黑', icon: 'bi bi-person-slash' },
  'auth-group': { label: '权限分组', icon: 'bi bi-shield-lock' },
  'auth-rules': { label: '权限规则', icon: 'bi bi-shield-check' },
  'auth-pages': { label: '页面权限', icon: 'bi bi-window-stack' },
  'api-keys': { label: '接口密钥', icon: 'bi bi-key' },
  toml: { label: '服务配置', icon: 'bi bi-sliders' },
  config: { label: '站点配置', icon: 'bi bi-gear' },
  attachment: { label: '附件', icon: 'bi bi-paperclip' },
  notification: { label: '消息通知', icon: 'bi bi-bell' },
  chat: { label: '私信', icon: 'bi bi-chat-dots' },
  'chat-admin': { label: '聊天管理', icon: 'bi bi-shield-exclamation' },
  'chat-moderation': { label: '聊天审核', icon: 'bi bi-shield-exclamation' },
  room: { label: '公共群聊', icon: 'bi bi-people' },
  integral: { label: '积分', icon: 'bi bi-coin' },
  goods: { label: '积分商城', icon: 'bi bi-shop' },
  decoration: { label: '装扮', icon: 'bi bi-gem' },
  checkin: { label: '签到', icon: 'bi bi-calendar-check' },
  exp: { label: '经验值', icon: 'bi bi-stars' },
  level: { label: '等级', icon: 'bi bi-bar-chart-steps' },
  docs: { label: '文档', icon: 'bi bi-book' },
  'docs-group': { label: '文档目录', icon: 'bi bi-folder' },
  'docs-project': { label: '文档项目', icon: 'bi bi-collection' },
  log: { label: '日志解析', icon: 'bi bi-file-earmark-code' },
  stats: { label: '数据统计', icon: 'bi bi-graph-up' },
  tip: { label: '积分打赏', icon: 'bi bi-cash-coin' },
  report: { label: '举报', icon: 'bi bi-flag' },
  search: { label: '搜索', icon: 'bi bi-search' },
  rss: { label: 'RSS 订阅', icon: 'bi bi-rss' },
  mention: { label: '@ 提及', icon: 'bi bi-at' },
  feed: { label: '聚合流', icon: 'bi bi-diagram-3' },
  unlock: { label: '内容解锁', icon: 'bi bi-lock' },
  comm: { label: '登录注册', icon: 'bi bi-person-check' },
  oauth: { label: '第三方登录', icon: 'bi bi-box-arrow-in-right' },
  proxy: { label: '代理接口', icon: 'bi bi-arrow-left-right' },
  'ip-black': { label: 'IP 黑名单', icon: 'bi bi-slash-circle' },
  'ip-white': { label: 'IP 白名单', icon: 'bi bi-check-circle' },
  'qps-warn': { label: 'QPS 预警', icon: 'bi bi-speedometer2' }
}

/** 模块展示顺序：常用/内容类在前，系统类在后；未登记的排在最后 */
export const AUTH_RULE_MODULE_ORDER = [
  'article', 'article-group', 'pages', 'comment', 'moments', 'tags', 'topics',
  'users', 'user-follows', 'user-likes', 'user-collects', 'user-blocks',
  'integral', 'goods', 'decoration', 'checkin', 'exp', 'level',
  'docs', 'docs-group', 'docs-project',
  'attachment', 'notification', 'chat', 'chat-admin', 'chat-moderation', 'room', 'report', 'banner', 'placard', 'links', 'links-group',
  'config', 'toml', 'api-keys', 'log', 'stats', 'qps-warn', 'ip-black', 'ip-white',
  'auth-group', 'auth-rules', 'auth-pages',
  'comm', 'oauth', 'search', 'rss', 'feed', 'mention', 'tip', 'unlock', 'proxy'
]

const MODULE_ORDER_INDEX = new Map(AUTH_RULE_MODULE_ORDER.map((key, index) => [key, index]))

/** 规则所属模块：/api/article/all → article；非 /api 前缀时取第一段兜底 */
export function authRuleModuleKey(item) {
  const route = String(item?.route || '')
  const matched = /^\/api\/([^/?#]+)/.exec(route)
  if (matched) return matched[1]
  const seg = route.split('/').filter(Boolean)[0]
  return seg || 'other'
}

/** 模块展示信息（未登记模块兜底为「模块名 + 通用图标」） */
export function authRuleModuleMeta(key) {
  return AUTH_RULE_MODULES[key] || { label: key, icon: 'bi bi-box' }
}

/** 模块排序权重（越小越靠前） */
export function authRuleModuleOrder(key) {
  return MODULE_ORDER_INDEX.has(key) ? MODULE_ORDER_INDEX.get(key) : 999
}

/** 规则标题：去掉后端加的【模块 API】前缀，模块名由分组卡片承担 */
export function authRuleTitle(item) {
  const name = String(item?.name || '').replace(/^【[^】]*】/, '').trim()
  return name || String(item?.route || '')
}

const METHOD_RANK = { GET: 0, POST: 1, PUT: 2, PATCH: 3, DELETE: 4 }

/** 组内排序：查看 → 新增 → 修改 → 删除 */
export function authRuleMethodRank(item) {
  const method = String(item?.method || 'GET').toUpperCase()
  return METHOD_RANK[method] ?? 9
}

/**
 * 快速套用模板（只勾 type=default 的规则 —— 另外两类根本不校验权限点）
 * modules 为空表示「所有模块」；methods 为空表示「所有方法」
 */
export const AUTH_RULE_PRESETS = [
  {
    key: 'ordinary',
    label: '普通用户默认',
    icon: 'bi bi-person-badge',
    desc: '发布 / 编辑 / 删除自己的文章与动态（控制器按 uid 限定了范围，只能动自己的数据）',
    // 与 app/model/auth-group.go 的 ordinaryAuthPermits 保持一致：
    // 评论、友链、附件上传的规则是 type=login，登录即放行，不需要权限点
    permits: [
      'POST /api/article/save',
      'POST /api/article/create',
      'PUT /api/article/update',
      'PUT /api/article/restore',
      'DELETE /api/article/remove',
      'DELETE /api/article/delete',
      'POST /api/moments/save',
      'POST /api/moments/create',
      'PUT /api/moments/update',
      'PUT /api/moments/restore',
      'DELETE /api/moments/remove',
      'DELETE /api/moments/delete'
    ]
  },
  {
    key: 'content',
    label: '内容编辑',
    icon: 'bi bi-pencil-square',
    desc: '文章 / 分类 / 评论 / 动态 / 页面 / 轮播 / 公告 / 话题',
    modules: ['article', 'article-group', 'comment', 'moments', 'pages', 'banner', 'placard', 'tags', 'topics']
  },
  {
    key: 'content-read',
    label: '只读查看',
    icon: 'bi bi-eye',
    desc: '所有模块的「查看类」权限（GET），适合审稿 / 客服',
    methods: ['GET']
  },
  {
    key: 'community',
    label: '社区运营',
    icon: 'bi bi-people',
    desc: '评论 / 举报 / 拉黑 / 用户 / 动态',
    modules: ['comment', 'report', 'user-blocks', 'users', 'moments', 'tags', 'topics']
  },
  {
    key: 'docs',
    label: '文档维护',
    icon: 'bi bi-book',
    desc: '文档库 / 目录 / 项目',
    modules: ['docs', 'docs-group', 'docs-project']
  },
  {
    key: 'shop',
    label: '积分商城',
    icon: 'bi bi-shop',
    desc: '积分 / 商品 / 装扮 / 签到 / 经验 / 等级',
    modules: ['integral', 'goods', 'decoration', 'checkin', 'exp', 'level']
  },
  {
    key: 'system',
    label: '站点配置',
    icon: 'bi bi-sliders',
    desc: '站点配置 / 服务配置 / 附件 / 日志 / 统计 / 安全',
    modules: ['config', 'toml', 'api-keys', 'attachment', 'log', 'stats', 'qps-warn', 'ip-black', 'ip-white']
  }
]
