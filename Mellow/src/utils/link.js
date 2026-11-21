/**
 * 友链（links）领域公共逻辑
 *
 * 后台统计卡片（views/admin/Links.vue）与列表筛选（views/admin/LinkList.vue）
 * 共用同一份筛选定义，避免两处各自维护 where 条件导致统计与列表口径不一致
 * （与 utils/user.js、utils/moment.js 保持同样的组织方式）。
 */

// 列表查询字段：需要 result（内含分组信息），但不拉取 json/text 等大字段
export const LINK_LIST_FIELD =
  'id,uid,nickname,description,url,avatar,target,audit,reason,remark,group,create_time,update_time,delete_time,result'

// 分组查询字段
export const LINK_GROUP_FIELD = 'id,name,description,avatar,create_time,update_time,delete_time'

// 审核状态（与后端 model.Links.Audit 对应，非管理员只能看到 audit=1 的友链）
// 与文章 / 动态一致：0 待审核、1 已通过、2 未通过（驳回时管理员要填原因，作者能看到）
export const LINK_AUDIT_PENDING = 0
export const LINK_AUDIT_PASS = 1
export const LINK_AUDIT_REJECT = 2

// 回收站 key：不属于状态标签，需要单独的查询参数（onlyTrashed）
export const LINK_TRASH_KEY = 'trash'

/**
 * 筛选定义
 * - key         筛选标识，统计卡片与列表标签共用（也是父级联动卡片的 activeKey）
 * - label       统计卡片文案
 * - tabLabel    列表标签文案
 * - where       后端筛选条件（where 内多个字段以 AND 拼接）
 * - onlyTrashed 是否查询回收站数据
 */
export const LINK_FILTERS = [
  {
    key: 'all',
    label: '全部友链',
    tabLabel: '全部',
    icon: 'bi bi-link-45deg',
    color: 'var(--primary)',
    where: {}
  },
  {
    key: 'pass',
    label: '已通过',
    tabLabel: '已通过',
    icon: 'bi bi-patch-check',
    color: 'var(--success)',
    where: { audit: LINK_AUDIT_PASS }
  },
  {
    key: 'pending',
    label: '待审核',
    tabLabel: '待审核',
    icon: 'bi bi-hourglass-split',
    color: 'var(--warning)',
    where: { audit: LINK_AUDIT_PENDING }
  },
  {
    key: 'reject',
    label: '未通过',
    tabLabel: '未通过',
    icon: 'bi bi-slash-circle',
    color: 'var(--danger)',
    where: { audit: LINK_AUDIT_REJECT }
  },
  {
    key: LINK_TRASH_KEY,
    label: '回收站',
    tabLabel: '回收站',
    icon: 'bi bi-trash3',
    color: 'var(--text-muted)',
    where: {},
    onlyTrashed: true
  }
]

// 列表状态标签（回收站由独立按钮触发，不进入标签行）
export const LINK_TABS = LINK_FILTERS
  .filter((f) => f.key !== LINK_TRASH_KEY)
  .map(({ key, tabLabel, where }) => ({ key, label: tabLabel, where }))

// 排序白名单：后端 links/all 的 order 未做校验，必须由前端限定取值
export const LINK_SORT_OPTIONS = [
  { value: 'id desc', label: '最新添加' },
  { value: 'id asc', label: '最早添加' },
  { value: 'create_time desc', label: '创建时间倒序' },
  { value: 'create_time asc', label: '创建时间正序' }
]

export const LINK_DEFAULT_ORDER = LINK_SORT_OPTIONS[0].value
export const LINK_TRASH_ORDER = 'delete_time desc'

/** 关键词搜索字段白名单（后端 like 参数只支持「字段名|值」单字段匹配） */
export const LINK_SEARCH_FIELDS = [
  { value: 'nickname', label: '名称' },
  { value: 'url', label: '链接' },
  { value: 'description', label: '描述' }
]

/** 打开方式 */
export const LINK_TARGET_OPTIONS = [
  { value: '_blank', label: '新窗口打开' },
  { value: '_self', label: '当前窗口打开' }
]

/** 审核状态文案：已通过 / 待审核 / 未通过（未知数值按待审核处理） */
export function linkAuditLabel(item) {
  const audit = Number(item?.audit)
  if (audit === LINK_AUDIT_PASS) return '已通过'
  if (audit === LINK_AUDIT_REJECT) return '未通过'
  return '待审核'
}

/** 审核状态配色 class（列表状态芯片用） */
export function linkAuditClass(item) {
  const audit = Number(item?.audit)
  if (audit === LINK_AUDIT_PASS) return 'is-pass'
  if (audit === LINK_AUDIT_REJECT) return 'is-reject'
  return 'is-pending'
}

/** 是否已通过审核 */
export function isLinkAudited(item) {
  return Number(item?.audit) === LINK_AUDIT_PASS
}

/** 友链所属分组（后端在 result.group 中解析好，未分组时为「默认分组」） */
export function linkGroupOf(item) {
  return item?.result?.group || null
}

/** 归一化筛选 key：未知值回落到「全部」，避免父级与子级参数对不上 */
export function resolveLinkFilterKey(key) {
  return LINK_FILTERS.some((f) => f.key === key) ? key : 'all'
}

/** 取筛选定义（key 非法时返回「全部」） */
export function linkFilterOf(key) {
  const safeKey = resolveLinkFilterKey(key)
  return LINK_FILTERS.find((f) => f.key === safeKey)
}

/**
 * 序列化筛选条件：无条件时返回 null（调用方不要传 where）
 *
 * 传 "{}" 给后端属于无意义往返：IWhere 对字符串会走 JSON 解码分支，
 * 空对象与「不传」语义相同，统一省略可减少歧义。
 */
export function linkWhereJSON(key) {
  const where = linkFilterOf(key)?.where || {}
  return Object.keys(where).length ? JSON.stringify(where) : null
}

/** 构造统计（count）接口参数（回收站数量请改用列表接口的 count 字段） */
export function linkCountParams(key) {
  const filter = linkFilterOf(key)
  const params = {}
  const where = linkWhereJSON(filter.key)
  if (where) params.where = where
  return params
}

/**
 * 构造列表接口参数（字段、排序、分页、关键词一次拼全）
 * @param {object} options
 * @param {string} options.key 筛选 key
 * @param {number} options.page 页码
 * @param {number} options.limit 每页条数
 * @param {string} options.keyword 关键词
 * @param {string} options.searchField 关键词匹配字段（见 LINK_SEARCH_FIELDS）
 * @param {number} [options.group] 分组 ID（>0 时生效）
 * @param {string} [options.order] 覆盖默认排序（回收站固定按删除时间倒序）
 */
export function linkListParams({
  key,
  page = 1,
  limit = 15,
  keyword = '',
  searchField = 'nickname',
  group = 0,
  order
} = {}) {
  const filter = linkFilterOf(key)
  const params = {
    page,
    limit,
    field: LINK_LIST_FIELD,
    order: order || (filter.onlyTrashed ? LINK_TRASH_ORDER : LINK_DEFAULT_ORDER)
  }

  const where = linkWhereJSON(filter.key)
  if (where) params.where = where
  if (filter.onlyTrashed) params.onlyTrashed = true

  // 分组筛选与状态筛选合并到同一个 where（分组为 0 表示「未分组」，不传表示全部）
  const groupId = Number(group || 0)
  if (groupId > 0) {
    const merged = { ...(linkFilterOf(key)?.where || {}) }
    merged.group = groupId
    params.where = JSON.stringify(merged)
  }

  // 关键词走后端 like 的「字段名|值」格式
  const kw = sanitizeLinkKeyword(keyword)
  if (kw) {
    const field = LINK_SEARCH_FIELDS.some((f) => f.value === searchField) ? searchField : 'nickname'
    params.like = `${field}|${kw}`
  }

  return params
}

/** 搜索关键字清理：剔除会干扰后端 LIKE「字段名|值」格式的字符 */
export function sanitizeLinkKeyword(keyword) {
  return String(keyword ?? '')
    .replace(/['"\\%_|]/g, '')
    .trim()
}
