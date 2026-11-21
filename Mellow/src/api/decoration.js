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

// ===== 公开 =====

// 装扮类型元数据 + 装扮配置（前端据此动态渲染分区与编辑表单）
export const getDecorationTypes = () => call('decoration', 'types', { method: 'GET' })

// 装扮配置（是否开启、头衔长度限制等）
export const getDecorationConfig = () => call('decoration', 'config', { method: 'GET' })

// 商城装扮列表（公开，仅上架，附带 goods_id / price / stock / on_sale）
// type 可选：avatar_frame 头像框 / title 头衔
// price_type 可选：free 只看免费 / paid 只看付费（= 非免费）；不传为全部
export const getDecorationShop = (params = {}) =>
  call('decoration', 'shop', { method: 'GET', params })

// 装扮详情
export const getDecorationOne = (id) =>
  call('decoration', 'one', { method: 'GET', params: { id } })

// 查看指定用户拥有的装扮（管理员）：返回 { user, types, wearing, list }
// 结构同 mine，便于后台排查「这个用户有哪些装扮 / 是否已过期」
export const getUserDecorations = (uid) =>
  call('decoration', 'user-decorations', { method: 'GET', params: { uid } })

// ===== 登录用户 =====

// 我的装扮（已拥有 + 系统默认），返回 { types, wearing, list }
export const getMyDecorations = () => call('decoration', 'mine', { method: 'GET' })

// 我当前佩戴的装扮
export const getWearingDecorations = () => call('decoration', 'wearing', { method: 'GET' })

// 佩戴装扮（text：纯样式类头衔允许自填文字）
export const wearDecoration = (decorationId, text = '') =>
  call('decoration', 'wear', { method: 'POST', data: { decoration_id: decorationId, text } })

// 兑换装扮（登录用户）：直接扣积分并发放
// 与积分商城解耦 —— 不经过商品 / 订单，价格、库存、经验门槛都取自装扮自身
// （稀缺性只用库存控制，每人限购已移除）
export const buyDecoration = (decorationId) =>
  call('decoration', 'buy', { method: 'POST', data: { decoration_id: decorationId } })

// 卸下某类型的装扮（type：avatar_frame / title）
export const unwearDecoration = (type) =>
  call('decoration', 'unwear', { method: 'POST', data: { type } })

// ===== 后台（管理员）=====

export const DECORATION_ADMIN_FIELD =
  'id,type,name,description,preview,payload,price_type,price,rarity,duration,renew,unlock,category,status,sort,sold,is_default,create_time,update_time,delete_time'

// 装扮列表（管理员：默认含下架；传 onlyTrashed=true 看回收站）
// 可用筛选：type 类型 / status 0下架 1上架 / price_type free 免费 · paid 付费 / keyword 名称模糊
export const listDecorationsAdmin = (params = {}) =>
  call('decoration', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'sort desc, id desc',
      field: DECORATION_ADMIN_FIELD,
      ...params
    }
  })

// 装扮统计（管理员）：总数 / 上架数 / 用户拥有数 / 各类型数量
export const countDecorations = () => call('decoration', 'count', { method: 'GET' })

// 保存装扮：无 id 为新增，有 id 为更新（后端 save 自动分流）
// payload 必须是 JSON 字符串（后端按 utils.Json.Decode 解析）
// 同时可传 stock / min_exp（装扮自带售卖字段；每人限购已废弃、不再生效）
export const saveDecoration = (data) =>
  call('decoration', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeDecorations = (ids) =>
  call('decoration', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteDecorations = (ids) =>
  call('decoration', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreDecorations = (ids) =>
  call('decoration', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearDecorationRecycle = () => call('decoration', 'clear', { method: 'DELETE' })

// 发放装扮（管理员）：批量用户 × 批量装扮
// duration：有效期秒数，0 或不传表示用装扮自身配置（两者都为 0 即永久）
export const grantDecorations = (data) =>
  call('decoration', 'grant', { method: 'PUT', data: { ...data, uids: toIdsArray(data.uids), decoration_ids: toIdsArray(data.decoration_ids) } })

// 回收装扮（管理员）：批量用户 × 批量装扮
export const revokeDecorations = (data) =>
  call('decoration', 'revoke', { method: 'PUT', data: { ...data, uids: toIdsArray(data.uids), decoration_ids: toIdsArray(data.decoration_ids) } })
