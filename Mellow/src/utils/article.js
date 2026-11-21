/**
 * 文章（article）领域公共逻辑
 *
 * 后台统计卡片（views/admin/Articles.vue）与列表筛选（views/admin/ArticleList.vue）
 * 共用同一份筛选定义，避免两处各自维护 where 条件导致统计与列表口径不一致
 * （与 utils/user.js、utils/link.js 保持同样的组织方式）。
 */

// 回收站 key：不属于状态标签，需要单独的查询参数（onlyTrashed）
export const ARTICLE_TRASH_KEY = 'trash'

// 文章状态与审核状态（与后端 model.Article 对应）
export const ARTICLE_STATUS_DRAFT = 0   // 草稿
export const ARTICLE_STATUS_PUBLISH = 1 // 已发布
export const ARTICLE_AUDIT_PENDING = 0  // 待审核
export const ARTICLE_AUDIT_PASS = 1     // 审核通过
export const ARTICLE_AUDIT_REJECT = 2   // 审核驳回

/**
 * 筛选定义
 * - key         筛选标识，统计卡片与列表标签共用（也是父级联动卡片的 activeKey）
 * - label       统计卡片文案
 * - tabLabel    列表标签文案
 * - where       后端筛选条件（where 内多个字段以 AND 拼接）
 * - onlyTrashed 是否查询回收站数据
 */
export const ARTICLE_FILTERS = [
  {
    key: 'all',
    label: '全部文章',
    tabLabel: '全部',
    icon: 'bi bi-file-earmark-text',
    color: 'var(--primary)',
    where: {}
  },
  {
    key: 'pub',
    label: '已发布',
    tabLabel: '已发布',
    icon: 'bi bi-check2-circle',
    color: 'var(--success)',
    where: { status: ARTICLE_STATUS_PUBLISH, audit: ARTICLE_AUDIT_PASS }
  },
  {
    key: 'audit',
    label: '待审核',
    tabLabel: '待审核',
    icon: 'bi bi-hourglass-split',
    color: 'var(--warning)',
    where: { status: ARTICLE_STATUS_PUBLISH, audit: ARTICLE_AUDIT_PENDING }
  },
  {
    key: 'reject',
    label: '未通过',
    tabLabel: '未通过',
    icon: 'bi bi-slash-circle',
    color: 'var(--danger)',
    where: { status: ARTICLE_STATUS_PUBLISH, audit: ARTICLE_AUDIT_REJECT }
  },
  {
    key: 'draft',
    label: '草稿',
    tabLabel: '草稿',
    icon: 'bi bi-pencil',
    color: 'var(--text-muted)',
    where: { status: ARTICLE_STATUS_DRAFT }
  },
  {
    key: 'top',
    label: '已置顶',
    tabLabel: '已置顶',
    icon: 'bi bi-pin-angle-fill',
    color: '#0ea5e9',
    where: { top: 1 }
  },
  {
    key: ARTICLE_TRASH_KEY,
    label: '回收站',
    tabLabel: '回收站',
    icon: 'bi bi-trash3',
    color: 'var(--text-muted)',
    where: {},
    onlyTrashed: true
  }
]

// 列表状态标签（回收站由独立按钮触发，不进入标签行）
export const ARTICLE_TABS = ARTICLE_FILTERS
  .filter((f) => f.key !== ARTICLE_TRASH_KEY)
  .map(({ key, tabLabel, where }) => ({ key, label: tabLabel, where }))

/** 归一化筛选 key：未知值回落到「全部」，避免父级与子级参数对不上 */
export function resolveArticleFilterKey(key) {
  return ARTICLE_FILTERS.some((f) => f.key === key) ? key : 'all'
}

/** 取筛选定义（key 非法时返回「全部」） */
export function articleFilterOf(key) {
  const safeKey = resolveArticleFilterKey(key)
  return ARTICLE_FILTERS.find((f) => f.key === safeKey)
}

/**
 * 序列化筛选条件：无条件时返回 null（调用方不要传 where）
 *
 * 传 "{}" 给后端属于无意义往返：IWhere 对字符串会走 JSON 解码分支，
 * 空对象与「不传」语义相同，统一省略可减少歧义。
 */
export function articleWhereJSON(key) {
  const where = articleFilterOf(key)?.where || {}
  return Object.keys(where).length ? JSON.stringify(where) : null
}

/** 构造统计（count）接口参数（article/count 支持 onlyTrashed） */
export function articleCountParams(key) {
  const filter = articleFilterOf(key)
  const params = {}
  const where = articleWhereJSON(filter.key)
  if (where) params.where = where
  if (filter.onlyTrashed) params.onlyTrashed = true
  return params
}
