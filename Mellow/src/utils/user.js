/**
 * 用户（users）领域公共逻辑
 *
 * 后台统计卡片（views/admin/Users.vue）与列表筛选（views/admin/UserList.vue）
 * 共用同一份筛选定义，避免两处各自维护 where 条件导致统计与列表口径不一致
 * （与 utils/moment.js 保持同样的组织方式）。
 */

// 列表查询字段：不拉取 json/text 等大字段，但需要 result（内含 ban / auth / level 解析结果）
export const USER_LIST_FIELD =
  'id,account,nickname,email,phone,avatar,description,title,gender,exp,integral,source,remark,status,ban_count,current_ban_id,last_ban_at,restrictions,login_time,create_time,update_time,delete_time,result'

// 回收站 key：不属于状态标签，需要单独的查询参数（onlyTrashed）
export const USER_TRASH_KEY = 'trash'

// 用户状态（与后端 model.UserStatusNormal / UserStatusFrozen / UserStatusAudit 对应）
export const USER_STATUS_NORMAL = 0
export const USER_STATUS_FROZEN = 1
// 待审核：注册验证方式为「人工审核」时写入，管理员在用户列表改为「正常」即通过
export const USER_STATUS_AUDIT = 2

/**
 * 筛选定义
 * - key         筛选标识，统计卡片与列表标签共用（也是父级联动卡片的 activeKey）
 * - label       统计卡片文案
 * - tabLabel    列表标签文案
 * - where       后端筛选条件（where 内多个字段以 AND 拼接）
 * - onlyTrashed 是否查询回收站数据
 */
export const USER_FILTERS = [
  {
    key: 'all',
    label: '全部用户',
    tabLabel: '全部',
    icon: 'bi bi-people',
    color: 'var(--primary)',
    where: {}
  },
  {
    key: 'normal',
    label: '正常',
    tabLabel: '正常',
    icon: 'bi bi-person-check',
    color: 'var(--success)',
    where: { status: USER_STATUS_NORMAL }
  },
  {
    key: 'audit',
    label: '待审核',
    tabLabel: '待审核',
    icon: 'bi bi-hourglass-split',
    color: '#0ea5e9',
    where: { status: USER_STATUS_AUDIT }
  },
  {
    key: 'frozen',
    label: '已冻结',
    tabLabel: '已冻结',
    icon: 'bi bi-snow',
    color: 'var(--warning)',
    where: { status: USER_STATUS_FROZEN }
  },
  {
    key: 'banned',
    label: '封禁中',
    tabLabel: '封禁中',
    icon: 'bi bi-shield-x',
    color: 'var(--danger)',
    // 当前生效的封禁记录 ID > 0；解封后后端会重置为 0，口径与列表展示一致
    where: { current_ban_id: { $gt: 0 } }
  },
  {
    key: USER_TRASH_KEY,
    label: '回收站',
    tabLabel: '回收站',
    icon: 'bi bi-trash3',
    color: 'var(--text-muted)',
    where: {},
    onlyTrashed: true
  }
]

// 列表状态标签（回收站由独立按钮触发，不进入标签行）
export const USER_TABS = USER_FILTERS
  .filter((f) => f.key !== USER_TRASH_KEY)
  .map(({ key, tabLabel, where }) => ({ key, label: tabLabel, where }))

// 排序白名单：后端 order 未做白名单校验，必须由前端限定取值
export const USER_SORT_OPTIONS = [
  { value: 'id desc', label: '最新注册' },
  { value: 'id asc', label: '最早注册' },
  { value: 'login_time desc', label: '最近登录' },
  { value: 'exp desc', label: '经验值最高' },
  { value: 'integral desc', label: '积分最多' },
  { value: 'ban_count desc', label: '封禁次数最多' }
]

export const USER_DEFAULT_ORDER = USER_SORT_OPTIONS[0].value
export const USER_TRASH_ORDER = 'delete_time desc'

/**
 * 关键词搜索字段白名单（后端 like 参数只支持「字段名|值」单字段匹配）
 */
export const USER_SEARCH_FIELDS = [
  { value: 'nickname', label: '昵称' },
  { value: 'account', label: '账号' },
  { value: 'email', label: '邮箱' },
  { value: 'phone', label: '手机号' }
]

/** 归一化筛选 key：未知值回落到「全部」，避免父级与子级参数对不上 */
export function resolveUserFilterKey(key) {
  return USER_FILTERS.some((f) => f.key === key) ? key : 'all'
}

/** 取筛选定义（key 非法时返回「全部」） */
export function userFilterOf(key) {
  const safeKey = resolveUserFilterKey(key)
  return USER_FILTERS.find((f) => f.key === safeKey)
}

/**
 * 序列化筛选条件：无条件时返回 null（调用方不要传 where）
 *
 * 传 "{}" 给后端属于无意义往返：IWhere 对字符串会走 JSON 解码分支，
 * 空对象与「不传」语义相同，统一省略可减少歧义。
 */
export function userWhereJSON(key) {
  const where = userFilterOf(key)?.where || {}
  return Object.keys(where).length ? JSON.stringify(where) : null
}

/**
 * 构造统计（count）接口参数
 *
 * 注意：后端 users/count 未处理 onlyTrashed，回收站数量请改用
 * listUsers({ page: 1, limit: 1, onlyTrashed: true, field: 'id' }) 的 count 字段。
 */
export function userCountParams(key) {
  const filter = userFilterOf(key)
  const params = {}
  const where = userWhereJSON(filter.key)
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
 * @param {string} options.searchField 关键词匹配字段（见 USER_SEARCH_FIELDS）
 * @param {string} [options.order] 覆盖默认排序（回收站固定按删除时间倒序）
 */
export function userListParams({ key, page = 1, limit = 15, keyword = '', searchField = 'nickname', order } = {}) {
  const filter = userFilterOf(key)
  const params = {
    page,
    limit,
    field: USER_LIST_FIELD,
    order: order || (filter.onlyTrashed ? USER_TRASH_ORDER : USER_DEFAULT_ORDER)
  }

  const where = userWhereJSON(filter.key)
  if (where) params.where = where
  if (filter.onlyTrashed) params.onlyTrashed = true

  // 关键词走后端 like 的「字段名|值」格式
  const kw = sanitizeUserKeyword(keyword)
  if (kw) {
    const field = USER_SEARCH_FIELDS.some((f) => f.value === searchField) ? searchField : 'nickname'
    params.like = `${field}|${kw}`
  }

  return params
}

/**
 * 搜索关键字清理：剔除会干扰后端 LIKE「字段名|值」格式的字符
 */
export function sanitizeUserKeyword(keyword) {
  return String(keyword ?? '')
    .replace(/['"\\%_|]/g, '')
    .trim()
}

// ---------- 封禁（ban） ----------

// 封禁类型位掩码（与后端 model.BanType* 常量对应）
export const BAN_TYPES = [
  { bit: 1, label: '限制登录', icon: 'bi bi-box-arrow-in-right' },
  { bit: 2, label: '限制发表内容', icon: 'bi bi-file-earmark-text' },
  { bit: 4, label: '限制评论', icon: 'bi bi-chat-square-text' },
  { bit: 8, label: '限制上传', icon: 'bi bi-cloud-arrow-up' },
  { bit: 16, label: '限制互动', icon: 'bi bi-hand-thumbs-up' }
]

// 全面封禁 = 全部权限位
export const BAN_TYPE_ALL = BAN_TYPES.reduce((sum, t) => sum | t.bit, 0)

// 封禁时长：auto 走后端自动梯度（首 1 天 / 7 / 15 / 30 / 五次以上永久）
export const BAN_DURATION_OPTIONS = [
  { value: 'auto', label: '自动梯度（首次 1 天，逐次加重）' },
  { value: 1, label: '1 天' },
  { value: 3, label: '3 天' },
  { value: 7, label: '7 天' },
  { value: 15, label: '15 天' },
  { value: 30, label: '30 天' },
  { value: 0, label: '永久封禁' }
]

/** 位掩码 -> 文字（如「限制登录、限制评论」） */
export function banTypeText(mask) {
  const value = Number(mask) || 0
  if (value === BAN_TYPE_ALL) return '全面封禁'
  const names = BAN_TYPES.filter((t) => value & t.bit).map((t) => t.label)
  return names.length ? names.join('、') : '无限制'
}

/** 封禁时长文案：0 = 永久 */
export function banDurationText(days) {
  return Number(days) > 0 ? `${Number(days)} 天` : '永久'
}

// ---------- 展示辅助 ----------

/** 用户状态文案（0 正常 / 1 冻结 / 2 待审核） */
export function userStatusLabel(item) {
  const status = Number(item?.status)
  if (status === USER_STATUS_FROZEN) return '已冻结'
  if (status === USER_STATUS_AUDIT) return '待审核'
  return '正常'
}

/** 是否处于封禁中（优先用后端解析好的 result.ban.is_banned） */
export function isUserBanned(item) {
  const banned = item?.result?.ban?.is_banned
  if (typeof banned === 'boolean') return banned
  return Number(item?.current_ban_id || 0) > 0
}

/** 当前生效的封禁记录（无则返回 null） */
export function banRecordOf(item) {
  return item?.result?.ban?.record || null
}

/**
 * 是否存在封禁痕迹（生效中的封禁、累计封禁次数 > 0、或有过封禁时间）
 * 用于决定「清空封禁信息」入口是否展示
 */
export function hasUserBanInfo(item) {
  if (isUserBanned(item)) return true
  const count = Number(item?.result?.ban?.ban_count ?? item?.ban_count ?? 0)
  if (count > 0) return true
  return Number(item?.last_ban_at || 0) > 0
}

/**
 * 封禁记录状态：0 封禁中 / 3 申诉中 / 5 申诉驳回（未封禁时 0）
 *
 * 注意：申诉中(3) 与申诉驳回(5) 期间封禁仍然生效（后端 BanStatusRestricted 口径），
 * 所以这两个状态在列表里也要显示封禁标识，直到真正解封。
 */
export function userBanStatus(item) {
  const info = item?.result?.ban
  if (!info || info.is_banned === false) return 0
  return Number(info.status ?? info.record?.status ?? 0) || 0
}

/** 是否申诉审核中（status=3） */
export function isUserAppealing(item) {
  return userBanStatus(item) === 3
}

/** 后台列表用的封禁状态文案：封禁中 / 申诉中 / 申诉驳回 */
export function userBanStatusLabel(item) {
  const status = userBanStatus(item)
  if (status === 3) return '申诉中'
  if (status === 5) return '申诉驳回'
  return '封禁中'
}

/** 用户所属权限组列表（后端 result.auth.group.list） */
export function userGroupsOf(item) {
  const list = item?.result?.auth?.group?.list
  return Array.isArray(list) ? list : []
}

/**
 * 受保护用户：系统管理员（id=1）与当前登录账号
 * 后端禁止删除/封禁系统管理员、禁止删除自己，前端同步禁用相关操作
 */
export function isProtectedUser(item, currentUid) {
  const id = Number(item?.id || 0)
  if (!id) return false
  if (id === 1) return true
  return Number(currentUid || 0) === id
}

/** 受保护原因（用于 tooltip） */
export function protectedReason(item, currentUid) {
  const id = Number(item?.id || 0)
  if (id === 1) return '系统管理员账号受保护'
  if (Number(currentUid || 0) === id) return '不能对当前登录账号执行该操作'
  return ''
}
