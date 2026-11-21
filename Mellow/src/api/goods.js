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

// ===== 商品 =====

// 商品列表（公开，默认仅上架；默认按排序权重展示）
// 支持：category 分类、keyword 关键词、order 排序（sort desc, id desc / price asc / sold desc ...）
export const getGoods = (params = {}) =>
  call('goods', 'all', { method: 'GET', params: { page: 1, limit: 20, order: 'sort desc, id desc', ...params } })

// 商品详情（返回附带 can_buy / buy_reason / limit_remain / my_bought）
export const getGoodsOne = (id) =>
  call('goods', 'one', { method: 'GET', params: { id } })

// 商品分类聚合（公开）
export const getGoodsCategories = () =>
  call('goods', 'categories', { method: 'GET' })

// 购买商品（登录，address 为收货地址 JSON 字符串，实物商品必填）
export const buyGoods = (goodsId, address) =>
  call('goods', 'buy', { method: 'POST', data: { goods_id: goodsId, address } })

// 我的订单（登录，支持 status / goods_id 筛选）
export const getOrders = (params = {}) =>
  call('goods', 'orders', { method: 'GET', params: { page: 1, limit: 20, order: 'create_time desc', ...params } })

// 订单详情（登录，仅本人或管理员）
export const getOrderOne = (id) =>
  call('goods', 'order-one', { method: 'GET', params: { id } })

// 取消订单（登录，仅「待发货」可取消，积分原路退还）
export const cancelOrder = (id) =>
  call('goods', 'cancel-order', { method: 'PUT', data: { id } })

// 确认收货（登录，已发货 → 已完成）
export const receiveOrder = (id) =>
  call('goods', 'receive', { method: 'PUT', data: { id } })

// 我的兑换统计（登录）
export const getMyGoodsStats = () =>
  call('goods', 'my-stats', { method: 'GET' })

// 商城统计（管理员）
export const getGoodsStats = () =>
  call('goods', 'stats', { method: 'GET' })

// ===== 积分 =====

// 积分概览（登录）：余额 + 累计收支 + 今日收支
export const getIntegral = () =>
  call('integral', 'status', { method: 'GET' })

// 积分流水（登录）
// 支持：type 类型、direction（income/expense）、start/end 时间戳、keyword 关键词
export const getIntegralLogs = (params = {}) =>
  call('integral', 'all', { method: 'GET', params: { page: 1, limit: 20, order: 'create_time desc', ...params } })

// 积分任务规则（公开）
export const getIntegralRules = () =>
  call('integral', 'rules', { method: 'GET' })

// 今日积分任务进度（登录）
export const getIntegralTasks = () =>
  call('integral', 'tasks', { method: 'GET' })

// 积分排行榜（公开）：by = earned（累计获得，默认）/ balance（当前余额）
export const getIntegralRank = (params = {}) =>
  call('integral', 'rank', { method: 'GET', params: { by: 'earned', limit: 20, ...params } })

// 卡密兑换积分（登录）：输入卡密即可兑换，成功后积分立即到账
export const redeemIntegralCard = (card) =>
  call('integral', 'card-redeem', { method: 'POST', data: { card } })

// ===== 后台（管理员）=====

// 后台商品列表字段：管理员需要 cards / deliver_content（非管理员会被后端剔除）
export const GOODS_ADMIN_FIELD =
  'id,title,description,cover,price,stock,status,type,deliver_type,deliver_content,cards,decoration_id,category,limit_per_user,min_exp,start_time,end_time,sort,sold,create_time,update_time,delete_time'

/**
 * 商品列表（管理员：默认含下架商品；传 onlyTrashed=true 看回收站）
 * 支持：status（0/1）、category、keyword（商品名模糊）、order（见后端白名单）
 */
export const listGoodsAdmin = (params = {}) =>
  call('goods', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'sort desc, id desc',
      field: GOODS_ADMIN_FIELD,
      ...params
    }
  })

// 商品数量统计
export const countGoods = (params = {}) =>
  call('goods', 'count', { method: 'GET', params })

// 保存商品：无 id 为新增，有 id 为更新（后端 save 自动分流）
// 注意：cards 必须传 JSON 字符串（后端按 utils.Json.Decode 解析），
// 传数组会被 processFieldValue 用逗号拼接，导致卡密池无法解析。
export const saveGoods = (data) =>
  call('goods', 'save', { method: 'POST', data })

// 软删除（移入回收站）
export const removeGoods = (ids) =>
  call('goods', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteGoods = (ids) =>
  call('goods', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreGoods = (ids) =>
  call('goods', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearGoodsRecycle = () =>
  call('goods', 'clear', { method: 'DELETE' })

// 全部订单（管理员，支持 uid / status / goods_id 过滤）
export const listOrdersAdmin = (params = {}) =>
  call('goods', 'orders-all', {
    method: 'GET',
    params: { page: 1, limit: 15, order: 'create_time desc', ...params }
  })

// 订单状态流转（管理员）：status 0待发货 1已发货 2已完成 3已取消（取消会退还积分并回滚库存）
export const setOrderStatus = (data) =>
  call('goods', 'order-status', { method: 'PUT', data })
