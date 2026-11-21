/**
 * 动态（moments）领域公共逻辑
 *
 * 后台统计卡片（views/admin/Moments.vue）与列表筛选（views/admin/MomentList.vue）
 * 共用同一份筛选定义，避免两处各自维护 where 条件导致统计与列表口径不一致。
 */

// 实况图（图片::视频）的解析/序列化，见 utils/livePhoto.js
import { parseLiveList, serializeLiveList } from './livePhoto'

// 列表查询字段：不拉取 json/text 等大字段
// 注意：reason（驳回原因）必须在这里，否则后台列表刷新后拿不到原因、行内的原因标签会消失
export const MOMENT_LIST_FIELD =
  'id,uid,content,images,location,top,views,audit,reason,status,create_time,update_time,publish_time,delete_time'

// 回收站 key：不属于状态标签，需要单独的查询参数（onlyTrashed）
export const MOMENT_TRASH_KEY = 'trash'

/**
 * 状态筛选定义
 * - key      筛选标识，统计卡片与列表标签共用（也是父级联动卡片的 activeKey）
 * - label    统计卡片文案
 * - tabLabel 列表标签文案
 * - where    后端筛选条件（where 中的字段会以 AND 方式拼接）
 * - onlyTrashed 是否查询回收站数据
 */
export const MOMENT_FILTERS = [
  {
    key: 'all',
    label: '全部动态',
    tabLabel: '全部',
    icon: 'bi bi-lightning-charge',
    color: 'var(--primary)',
    where: {}
  },
  {
    key: 'pub',
    label: '已发布',
    tabLabel: '已发布',
    icon: 'bi bi-check2-circle',
    color: 'var(--success)',
    where: { status: 1, audit: 1 }
  },
  {
    key: 'pending',
    label: '待审核',
    tabLabel: '待审核',
    icon: 'bi bi-hourglass-split',
    color: 'var(--warning)',
    // 待审核 = 已提交发布但未通过审核（与文章/页面后台口径一致：status=1 且 audit=0），
    // 只写 audit=0 会把「草稿」也算进来
    where: { status: 1, audit: 0 }
  },
  {
    key: 'reject',
    label: '未通过',
    tabLabel: '未通过',
    icon: 'bi bi-slash-circle',
    color: 'var(--danger)',
    // 驳回：已提交发布但审核未通过（与文章/页面后台一致：status=1 且 audit=2）
    where: { status: 1, audit: 2 }
  },
  {
    key: 'draft',
    label: '草稿',
    tabLabel: '草稿',
    icon: 'bi bi-file-earmark',
    color: 'var(--text-muted)',
    where: { status: 0 }
  },
  {
    key: MOMENT_TRASH_KEY,
    label: '回收站',
    tabLabel: '回收站',
    icon: 'bi bi-trash3',
    color: 'var(--danger)',
    where: {},
    onlyTrashed: true
  }
]

// 列表状态标签（回收站由独立按钮触发，不进入标签行）
export const MOMENT_TABS = MOMENT_FILTERS
  .filter((f) => f.key !== MOMENT_TRASH_KEY)
  .map(({ key, tabLabel, where }) => ({ key, label: tabLabel, where }))

// 排序白名单：后端 order 未做白名单校验，必须由前端限定取值
export const MOMENT_SORT_OPTIONS = [
  { value: 'top desc, create_time desc', label: '置顶优先' },
  { value: 'create_time desc', label: '最新创建' },
  { value: 'create_time asc', label: '最早创建' },
  { value: 'publish_time desc', label: '最新发布' },
  { value: 'views desc', label: '浏览量最多' }
]

export const MOMENT_DEFAULT_ORDER = MOMENT_SORT_OPTIONS[0].value
export const MOMENT_TRASH_ORDER = 'delete_time desc'

/**
 * 归一化筛选 key：未知值回落到「全部」，避免父级与子级参数对不上
 */
export function resolveMomentFilterKey(key) {
  return MOMENT_FILTERS.some((f) => f.key === key) ? key : 'all'
}

/** 取筛选定义（key 非法时返回「全部」） */
export function momentFilterOf(key) {
  const safeKey = resolveMomentFilterKey(key)
  return MOMENT_FILTERS.find((f) => f.key === safeKey)
}

/**
 * 序列化筛选条件：无条件时返回 null（调用方不要传 where）
 *
 * 传 "{}" 给后端属于无意义往返：IWhere 对字符串会走 JSON 解码分支，
 * 空对象与「不传」语义相同，统一省略可减少歧义。
 */
export function momentWhereJSON(key) {
  const where = momentFilterOf(key)?.where || {}
  return Object.keys(where).length ? JSON.stringify(where) : null
}

/** 构造统计（count）接口参数 */
export function momentCountParams(key) {
  const filter = momentFilterOf(key)
  const params = {}
  const where = momentWhereJSON(filter.key)
  if (where) params.where = where
  if (filter.onlyTrashed) params.onlyTrashed = true
  return params
}

/**
 * 构造列表接口参数（字段、排序、分页、关键词一次拼全）
 * @param {object} options
 * @param {string} options.key 筛选 key
 * @param {number} options.page 页码
 * @param {number} options.limit 每页条数
 * @param {string} options.keyword 内容关键词
 * @param {string} [options.order] 覆盖默认排序（回收站固定按删除时间倒序）
 */
export function momentListParams({ key, page = 1, limit = 15, keyword = '', order } = {}) {
  const filter = momentFilterOf(key)
  const params = {
    page,
    limit,
    field: MOMENT_LIST_FIELD,
    order: order || (filter.onlyTrashed ? MOMENT_TRASH_ORDER : MOMENT_DEFAULT_ORDER)
  }

  const where = momentWhereJSON(filter.key)
  if (where) params.where = where
  if (filter.onlyTrashed) params.onlyTrashed = true

  // 关键词走后端 like 的「字段名|值」格式
  const kw = sanitizeMomentKeyword(keyword)
  if (kw) params.like = `content|${kw}`

  return params
}

/** 是否可在前台公开展示（已发布且审核通过） */
export function isMomentPublic(item) {
  return Number(item?.status) === 1 && Number(item?.audit) === 1
}

/**
 * 解析配图：后端以逗号分隔字符串存储，兼容数组与 | 分隔写法
 *
 * 实况图在字符串里是「图片::视频」（见 utils/livePhoto.js），
 * 这里只返回图片地址 —— 所有「只要图」的地方（后台缩略图、回收站、举报图等）
 * 无需关心实况图；需要视频的用 parseMomentMedia。
 *
 * @returns {string[]}
 */
export function parseMomentImages(raw) {
  return parseLiveList(raw).map((item) => item.image)
}

/**
 * 解析配图（含实况视频）
 * @returns {{ image: string, video: string }[]}
 */
export function parseMomentMedia(raw) {
  return parseLiveList(raw)
}

/** 序列化配图（发布 / 编辑动态时用）：带视频的项写成 图片::视频 */
export function serializeMomentImages(items) {
  return serializeLiveList(items)
}

// 状态文案：草稿 / 待审核 / 未通过 / 已发布
export function momentStatusLabel(item) {
  if (Number(item?.status) === 0) return '草稿'
  const audit = Number(item?.audit)
  if (audit === 0) return '待审核'
  if (audit === 2) return '未通过'
  return '已发布'
}

// 状态配色类名，与 .post-status.is-* 对应
export function momentStatusClass(item) {
  if (Number(item?.status) === 0) return 'is-draft'
  const audit = Number(item?.audit)
  if (audit === 0) return 'is-audit'
  if (audit === 2) return 'is-reject'
  return 'is-pub'
}

/**
 * 搜索关键字清理：剔除会干扰后端 LIKE「字段名|值」格式的字符
 */
export function sanitizeMomentKeyword(keyword) {
  return String(keyword ?? '')
    .replace(/['"\\%_|]/g, '')
    .trim()
}
