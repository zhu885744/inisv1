import { call } from './request'

// ids 归一化：DELETE 走 query 传参，数组会被序列化成 ids[]= 导致后端取不到，统一转逗号分隔字符串
const toIdsString = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)
    .join(',')

// PUT/POST 走 JSON body，直接传数组即可（utils.Unity.Ids 会解析）
const toIdsArray = (ids) =>
  (Array.isArray(ids) ? ids : [ids])
    .map((i) => Number(i))
    .filter((i) => i > 0)

// 稀有度：只影响展示（描边 / 光效 / 标签），不参与发放判定
export const BADGE_RARITY_TEXT = {
  common: '普通',
  rare: '稀有',
  epic: '史诗',
  legendary: '传说'
}

// 条件中文名：后端 badge/conditions 会返回权威清单，这里只是首屏渲染的兜底文案
// 与 app/model/badge.go 的 BadgeConditionList 保持一致
export const BADGE_CONDITION_TEXT = {
  manual: '手动发放',
  // 内容产出
  article_count: '发布文章',
  article_views: '文章阅读量',
  moments_count: '发布动态',
  comment_count: '发表评论',
  docs_count: '发布文档',
  // 内容认可
  like_received: '获赞',
  collect_received: '被收藏',
  fans_count: '粉丝数',
  // 活跃度
  register_days: '注册满',
  checkin_days: '累计签到',
  checkin_streak: '连续签到',
  exp: '经验值达到',
  integral: '积分达到',
  // 收集 / 消费
  badge_count: '获得勋章',
  decoration_count: '拥有装扮',
  goods_count: '兑换商品',
  tip_received: '收到打赏'
}

// ===== 勋章分类（后台）=====

export const BADGE_CATEGORY_FIELD =
  'id,name,description,icon,color,cover,sort,status,json,create_time,update_time,delete_time'

// all 会额外返回 counts（category_id → 勋章数量，key=0 是未分类）
export const listBadgeCategories = (params = {}) =>
  call('badge-category', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 100,
      order: 'sort desc, id asc',
      field: BADGE_CATEGORY_FIELD,
      ...params
    }
  })

// 无 id 新增 / 有 id 更新
export const saveBadgeCategory = (data) => call('badge-category', 'save', { method: 'POST', data })

// force=true：分类下还有勋章时，把它们退化为「未分类」（不删勋章）
export const removeBadgeCategory = (ids, force = false) =>
  call('badge-category', 'remove', {
    method: 'DELETE',
    params: { ids: toIdsString(ids), force: force ? 1 : 0 }
  })

// 把一批勋章挪到指定分类（category_id = 0 表示未分类）
export const moveBadgesToCategory = (ids, categoryId) =>
  call('badge-category', 'move', {
    method: 'POST',
    data: { ids: toIdsArray(ids), category_id: Number(categoryId) || 0 }
  })

// ===== 勋章定义（后台）=====

export const listBadges = (params = {}) =>
  call('badge', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 50,
      order: 'sort desc, id asc',
      field:
        'id,name,description,icon,image,color,condition,condition_value,category_id,rarity,start_time,end_time,expire_days,renew,sort,status,remark,result,create_time,update_time,delete_time',
      ...params
    }
  })

export const countBadges = (params = {}) => call('badge', 'count', { method: 'GET', params })

// 条件清单（后台表单下拉）
export const getBadgeConditions = () => call('badge', 'conditions', { method: 'GET' })

// 无 id 新增 / 有 id 更新
export const saveBadge = (data) => call('badge', 'save', { method: 'POST', data })

export const removeBadges = (ids) =>
  call('badge', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

export const forceDeleteBadges = (ids) =>
  call('badge', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

export const restoreBadges = (ids) =>
  call('badge', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

export const clearBadgeRecycle = () => call('badge', 'clear', { method: 'DELETE' })

// ===== 发放 / 收回 =====

export const grantBadges = (data) =>
  call('badge', 'grant', {
    method: 'PUT',
    data: { ...data, uids: toIdsArray(data.uids), badge_ids: toIdsArray(data.badge_ids) }
  })

export const revokeBadges = (data) =>
  call('badge', 'revoke', {
    method: 'PUT',
    data: { ...data, uids: toIdsArray(data.uids), badge_ids: toIdsArray(data.badge_ids) }
  })

// 持有记录（后台）
export const listBadgeRecords = (params = {}) =>
  call('badge', 'records', {
    method: 'GET',
    params: { page: 1, limit: 20, order: 'create_time desc', ...params }
  })

// 手动触发条件重算：uid 只算一个人，不传则全站扫一遍
export const scanBadges = (data = {}) => call('badge', 'scan', { method: 'POST', data })

// ===== 前台 =====

// 我的勋章（登录）：{ data: [{ held, current, target, ... }], held, count, categories, wearing }
export const getMyBadges = () => call('badge', 'mine', { method: 'GET' })

// 勋章馆 / 佩戴榜（公开；登录时会带「我是否已获得」）
export const getBadgeWall = () => call('badge', 'wall', { method: 'GET' })

// 佩戴 / 卸下（同时只佩戴一枚；佩戴信息由后端写入 users.json.wearing_badge）
export const wearBadge = (badgeId) =>
  call('badge', 'wear', { method: 'POST', data: { badge_id: Number(badgeId) } })

export const unwearBadge = () => call('badge', 'unwear', { method: 'POST' })

// 指定用户已获得的勋章（公开）
export const getUserBadges = (uid) => call('badge', 'held', { method: 'GET', params: { uid } })
