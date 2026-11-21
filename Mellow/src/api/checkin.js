import { call } from './request'

/**
 * 每日签到（独立模块）
 *
 * 后端：app/api/controller/checkin.go + app/model/checkin.go + app/model/reward.go
 * 签到原先挂在 /api/exp/check-in*（现已废弃但保留兼容），规则也从经验规则里独立出来，
 * 存 config 表的 SYSTEM_CHECKIN_RULES 键，由后台「签到管理」页维护。
 */

/** 签到配置在配置表中的键（model.CheckinCacheKey） */
export const CHECKIN_RULES_KEY = 'SYSTEM_CHECKIN_RULES'

// ===== 用户端 =====

/**
 * 签到状态（需登录）
 * 返回：开关 / 今日是否已签 / 连签天数 / 周期进度 / 今日可得（含概率与区间）/
 *       里程碑 / 月累计 / 可补签日期 / 资产清单
 */
export const getCheckinStatus = () => call('checkin', 'status', { method: 'GET' })

/** 签到日历（需登录）：params = { year, month }，不传为当前月 */
export const getCheckinCalendar = (params = {}) =>
  call('checkin', 'calendar', { method: 'GET', params })

/** 签到排行榜（公开）：params = { start, end, limit } */
export const getCheckinRank = (params = {}) =>
  call('checkin', 'rank', { method: 'GET', params: { limit: 20, ...params } })

/** 签到规则（公开）：开关 / 周期奖励表 / 里程碑 / 月度奖励 / 资产清单 / 文案池 */
export const getCheckinRules = () => call('checkin', 'rules', { method: 'GET' })

/** 签到（需登录） */
export const signIn = () => call('checkin', 'sign', { method: 'POST' })

/**
 * 补签（需登录）
 * date 为签到日键（yyyymmdd），不传则补昨天；消耗多少积分由后台「签到管理」配置
 */
export const makeupSign = (date) =>
  call('checkin', 'makeup', { method: 'POST', data: { date } })

// ===== 后台配置（config 表） =====

/** 原始签到配置（json 已由后端解码为对象；未配置时返回 null） */
export const getCheckinConfig = () =>
  call('config', 'one', { method: 'GET', params: { key: CHECKIN_RULES_KEY } })

/** 保存签到配置（后端会顺带清除签到配置缓存） */
export const saveCheckinConfig = (json) =>
  call('config', 'save', { method: 'POST', data: { key: CHECKIN_RULES_KEY, json } })

/**
 * 卡密库存（管理员）：传该奖励项当前填写的卡密内容（数组或一行一个的文本）
 *
 * 返回 { total, issued, remain }：total = 填写的张数、issued = 已发出、remain = 还剩几张。
 * 后端会顺带把新填的卡密补进库存（幂等），所以后台不用等发放就能看到库存。
 */
export const getRewardCardStock = (codes) =>
  call('checkin', 'card-stock', { method: 'POST', data: { codes } })
