/**
 * 评论（comment）领域公共逻辑
 *
 * 后台统计卡片（views/admin/Comments.vue）与列表筛选（views/admin/CommentList.vue）
 * 共用同一份筛选定义，避免统计与列表口径不一致（与 utils/moment.js、utils/user.js 一致）。
 *
 * 后端约束（app/api/controller/comment.go）：
 * - 评论表有 audit 字段（0 待审核 1 通过 2 未通过）；是否开启审核由 COMMENT 配置的 json.audit
 *   决定（「系统设置 → 评论配置 → 内容审核」）：关闭审核时提交即通过，开启时进待审核；
 *   非管理员在接口层只能看到 audit=1 的评论，管理端可筛选「待审核」并做通过 / 驳回；
 * - 可筛选维度为 bind_type（绑定类型）、audit（待审核）、pid（层级）、回收站；
 * - comment/count 已支持 onlyTrashed，回收站数量可直接用 count 统计；
 * - IP / UA 仅管理员可见原文，普通用户返回脱敏值；
 * - result 内已附带 author / article / page / moments，无需额外请求即可展示来源。
 */

// 实况图（图片::视频）的解析/序列化，见 utils/livePhoto.js
import { parseLiveList } from './livePhoto'

// 列表查询字段：不拉取 json/text/agent 等大字段，但需要 result（内含作者与来源）
// 注：前台评论树走 comment/flat 且不传 field（返回全字段），所以能拿到 agent 用来
// 展示「来自什么设备」（见 components/CommentItem.vue + utils/ua.js）；
export const COMMENT_LIST_FIELD =
  'id,pid,uid,content,images,ip,bind_id,bind_type,editor,audit,reason,scope,create_time,update_time,delete_time,result'

// 回收站 key：不属于状态标签，需要单独的查询参数（onlyTrashed）
export const COMMENT_TRASH_KEY = 'trash'

// 绑定类型（与后端 comment.bind_type 取值一致）
export const BIND_TYPE_ARTICLE = 'article'
export const BIND_TYPE_PAGE = 'page'
export const BIND_TYPE_MOMENTS = 'moments'
export const BIND_TYPE_DOCS = 'docs'

export const BIND_TYPE_LABELS = {
  [BIND_TYPE_ARTICLE]: '文章',
  [BIND_TYPE_PAGE]: '页面',
  [BIND_TYPE_MOMENTS]: '动态',
  [BIND_TYPE_DOCS]: '文档'
}

/**
 * 筛选定义（按绑定类型划分）
 * - key         筛选标识，统计卡片与列表标签共用（也是父级联动卡片的 activeKey）
 * - label       统计卡片文案
 * - tabLabel    列表标签文案
 * - where       后端筛选条件
 * - onlyTrashed 是否查询回收站数据
 */
export const COMMENT_FILTERS = [
  {
    key: 'all',
    label: '全部评论',
    tabLabel: '全部',
    icon: 'bi bi-chat-square-text',
    color: 'var(--primary)',
    where: {}
  },
  {
    key: BIND_TYPE_ARTICLE,
    label: '文章评论',
    tabLabel: '文章',
    icon: 'bi bi-file-earmark-text',
    color: 'var(--primary)',
    where: { bind_type: BIND_TYPE_ARTICLE }
  },
  {
    key: BIND_TYPE_PAGE,
    label: '页面评论',
    tabLabel: '页面',
    icon: 'bi bi-file-earmark',
    color: '#0ea5e9',
    where: { bind_type: BIND_TYPE_PAGE }
  },
  {
    key: BIND_TYPE_MOMENTS,
    label: '动态评论',
    tabLabel: '动态',
    icon: 'bi bi-lightning-charge',
    color: '#f59e0b',
    where: { bind_type: BIND_TYPE_MOMENTS }
  },
  {
    key: BIND_TYPE_DOCS,
    label: '文档评论',
    tabLabel: '文档',
    icon: 'bi bi-journal-bookmark',
    color: '#0d9488',
    where: { bind_type: BIND_TYPE_DOCS }
  },
  {
    // 待审核：开关在「系统设置 → 评论配置 → 内容审核」，开启后前台新评论落在这里
    key: 'pending',
    label: '待审核评论',
    tabLabel: '待审核',
    icon: 'bi bi-hourglass-split',
    color: '#f59e0b',
    where: { audit: 0 }
  },
  {
    key: COMMENT_TRASH_KEY,
    label: '回收站',
    tabLabel: '回收站',
    icon: 'bi bi-trash3',
    color: 'var(--text-muted)',
    where: {},
    onlyTrashed: true
  }
]

// 列表状态标签（回收站由独立按钮触发，不进入标签行）
export const COMMENT_TABS = COMMENT_FILTERS
  .filter((f) => f.key !== COMMENT_TRASH_KEY)
  .map(({ key, tabLabel, where }) => ({ key, label: tabLabel, where }))

/**
 * 层级筛选（与绑定类型并列的第二维度，由筛选栏的下拉选择）
 * 顶层评论 pid = 0，回复 pid > 0
 */
export const COMMENT_LEVEL_OPTIONS = [
  { value: '', label: '全部层级' },
  { value: 'top', label: '仅顶层评论' },
  { value: 'reply', label: '仅回复' }
]

/** 层级 -> where 条件 */
export function commentLevelWhere(level) {
  if (level === 'top') return { pid: 0 }
  if (level === 'reply') return { pid: { $gt: 0 } }
  return {}
}

// 排序白名单：后端 order 未做白名单校验，必须由前端限定取值
export const COMMENT_SORT_OPTIONS = [
  { value: 'create_time desc', label: '最新评论' },
  { value: 'create_time asc', label: '最早评论' },
  { value: 'update_time desc', label: '最近编辑' },
  { value: 'id desc', label: 'ID 倒序' }
]

export const COMMENT_DEFAULT_ORDER = COMMENT_SORT_OPTIONS[0].value
export const COMMENT_TRASH_ORDER = 'delete_time desc'

/**
 * 关键词搜索字段白名单（后端 like 参数只支持「字段名|值」单字段匹配）
 * 内容走 content，IP 走 ip（后端 ip 为真实字段，管理员视角未被脱敏）
 */
export const COMMENT_SEARCH_FIELDS = [
  { value: 'content', label: '内容' },
  { value: 'ip', label: 'IP' }
]

/** 归一化筛选 key：未知值回落到「全部」 */
export function resolveCommentFilterKey(key) {
  return COMMENT_FILTERS.some((f) => f.key === key) ? key : 'all'
}

/** 取筛选定义（key 非法时返回「全部」） */
export function commentFilterOf(key) {
  const safeKey = resolveCommentFilterKey(key)
  return COMMENT_FILTERS.find((f) => f.key === safeKey)
}

/**
 * 序列化筛选条件（绑定类型 + 层级），无条件时返回 null（调用方不要传 where）
 *
 * 传 "{}" 给后端属于无意义往返：IWhere 对字符串会走 JSON 解码分支，
 * 空对象与「不传」语义相同，统一省略可减少歧义。
 */
export function commentWhereJSON(key, level = '') {
  const where = {
    ...(commentFilterOf(key)?.where || {}),
    ...commentLevelWhere(level)
  }
  return Object.keys(where).length ? JSON.stringify(where) : null
}

/**
 * 构造统计（count）接口参数
 * 后端 comment/count 已支持 onlyTrashed，回收站数量可直接统计
 */
export function commentCountParams(key) {
  const filter = commentFilterOf(key)
  const params = {}
  const where = commentWhereJSON(filter.key)
  if (where) params.where = where
  if (filter.onlyTrashed) params.onlyTrashed = true
  return params
}

/**
 * 构造列表接口参数（字段、排序、分页、关键词一次拼全）
 * @param {object} options
 * @param {string} options.key 筛选 key（绑定类型 / 回收站）
 * @param {number} options.page 页码
 * @param {number} options.limit 每页条数
 * @param {string} options.keyword 关键词
 * @param {string} options.searchField 关键词匹配字段（见 COMMENT_SEARCH_FIELDS）
 * @param {string} options.level 层级（'' / top / reply）
 * @param {string} [options.order] 覆盖默认排序（回收站固定按删除时间倒序）
 */
export function commentListParams({ key, page = 1, limit = 15, keyword = '', searchField = 'content', level = '', order } = {}) {
  const filter = commentFilterOf(key)
  const params = {
    page,
    limit,
    field: COMMENT_LIST_FIELD,
    order: order || (filter.onlyTrashed ? COMMENT_TRASH_ORDER : COMMENT_DEFAULT_ORDER)
  }

  const where = commentWhereJSON(filter.key, level)
  if (where) params.where = where
  if (filter.onlyTrashed) params.onlyTrashed = true

  // 关键词走后端 like 的「字段名|值」格式
  const kw = sanitizeCommentKeyword(keyword)
  if (kw) {
    const field = COMMENT_SEARCH_FIELDS.some((f) => f.value === searchField) ? searchField : 'content'
    params.like = `${field}|${kw}`
  }

  return params
}

/** 搜索关键字清理：剔除会干扰后端 LIKE「字段名|值」格式的字符 */
export function sanitizeCommentKeyword(keyword) {
  return String(keyword ?? '')
    .replace(/['"\\%_|]/g, '')
    .trim()
}

// ---------- 展示辅助 ----------

/** 是否为回复（pid > 0） */
export function isCommentReply(item) {
  return Number(item?.pid || 0) > 0
}

/** 绑定类型文案 */
export function commentBindLabel(item) {
  return BIND_TYPE_LABELS[item?.bind_type] || '未知'
}

/** 来源标题（文章标题 / 页面标题 / 动态内容） */
export function commentBindTitle(item) {
  const result = item?.result || {}
  switch (item?.bind_type) {
    case BIND_TYPE_ARTICLE:
      return result.article?.title || ''
    case BIND_TYPE_PAGE:
      return result.page?.title || result.page?.key || ''
    case BIND_TYPE_MOMENTS:
      return result.moments?.content || ''
    case BIND_TYPE_DOCS:
      return result.docs?.title || ''
    default:
      return ''
  }
}

/**
 * 前台来源链接（不存在可跳转目标时返回空串，调用方据此禁用入口）
 * 文章 /archives/:id、动态 /moments/:id、页面 /:key、文档 /docs/:项目/:目录/:页面
 */
export function commentBindLink(item) {
  const result = item?.result || {}
  switch (item?.bind_type) {
    case BIND_TYPE_ARTICLE:
      return result.article?.id ? `/archives/${result.article.id}` : ''
    case BIND_TYPE_MOMENTS:
      return result.moments?.id ? `/moments/${result.moments.id}` : ''
    case BIND_TYPE_PAGE:
      return result.page?.key ? `/${result.page.key}` : ''
    case BIND_TYPE_DOCS:
      // result.docs 只有 {id, version, slug, title}，目录段拿不到，
      // 统一跳到项目概览，避免生成死链
      return result.docs?.version ? `/docs/${result.docs.version}` : ''
    default:
      return ''
  }
}

/**
 * 解析配图的原始项（保留「图片::视频」写法）：返回 string[]
 *
 * 用于后台编辑表单回填 / 合并 / 保存 —— 只取封面会把实况图降级成静态图。
 */
export function parseCommentRawList(raw) {
  if (!raw) return []
  const arr = Array.isArray(raw) ? raw : String(raw).split(/[,|]/)
  return arr.map((s) => String(s).trim()).filter(Boolean)
}

/**
 * 解析配图（含实况视频）：返回 { image, video }[]
 *
 * 与动态同一口径：实况图在库里写成 `图片URL::视频URL`（见 utils/livePhoto.js）。
 * 只要图片地址的地方（后台缩略图、举报图等）用 parseCommentImages；
 * 需要实况播放 / 灯箱「播放实况」的地方用本函数。
 */
export function parseCommentMedia(raw) {
  return parseLiveList(raw)
}

/**
 * 解析配图地址：返回图片 URL 数组（实况图只取封面）
 *
 * 注意：实况图一整项是 `图片::视频`，直接当 img src 会破图，
 * 所以所有「只展示图片」的地方都必须经过这里。
 */
export function parseCommentImages(raw) {
  return parseCommentMedia(raw).map((item) => item.image)
}
