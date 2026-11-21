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

/** 经验规则在配置表中的键（model.ExpCacheKey） */
export const EXP_RULES_KEY = 'SYSTEM_EXP_RULES'

// ===== 经验流水（exp）=====

// 经验流水列表（需要 result 才能拿到作者信息）
export const listExps = (params = {}) =>
  call('exp', 'all', {
    method: 'GET',
    params: {
      page: 1,
      limit: 15,
      order: 'create_time desc',
      field: 'id,uid,value,type,bind_type,bind_id,state,description,create_time,update_time,delete_time,result',
      ...params
    }
  })

// 经验流水数量（注意：后端 exp/count 不支持 onlyTrashed，回收站数量请用列表的 count）
export const countExps = (params = {}) =>
  call('exp', 'count', { method: 'GET', params })

// 经验流水聚合合计（field 必填，后端返回 { value: 合计 }）
export const sumExp = (params = {}) =>
  call('exp', 'sum', { method: 'GET', params: { field: 'value', ...params } })

// 软删除（移入回收站）
export const removeExps = (ids) =>
  call('exp', 'remove', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 彻底删除（不可恢复）
export const forceDeleteExps = (ids) =>
  call('exp', 'delete', { method: 'DELETE', params: { ids: toIdsString(ids) } })

// 从回收站恢复
export const restoreExps = (ids) =>
  call('exp', 'restore', { method: 'PUT', data: { ids: toIdsArray(ids) } })

// 清空回收站
export const clearExpRecycle = () =>
  call('exp', 'clear', { method: 'DELETE' })

/**
 * 管理员发放 / 扣除经验（POST exp/give，仅管理员）
 * value 为正数发放、负数扣除（不能为 0）
 */
export const giveExp = (data) =>
  call('exp', 'give', { method: 'POST', data })

// ===== 经验规则（config: SYSTEM_EXP_RULES）=====

/**
 * 经验规则列表（GET exp/rules）
 * 返回 [{ type, name, value, daily_limit }]；
 * 注意：check-in 的连续签到加成（streak_bonus）与里程碑（milestones）不会在这里返回，
 * 修改规则时必须以 config/one 的原始 json 为准，避免把这些字段写丢。
 */
export const getExpRules = () =>
  call('exp', 'rules', { method: 'GET' })

// 原始经验配置（config/one，json 已由后端解码为对象；未配置时返回 null）
export const getExpConfig = () =>
  call('config', 'one', { method: 'GET', params: { key: EXP_RULES_KEY } })

// 保存经验配置（config/save；后端会顺带清除经验配置缓存）
export const saveExpConfig = (json) =>
  call('config', 'save', { method: 'POST', data: { key: EXP_RULES_KEY, json } })
