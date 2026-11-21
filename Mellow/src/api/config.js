import { call } from './request'

// 获取单个配置项（按 key）
// config 透传给 axios（如 { silent: true } 表示失败时不弹全局提示）
export const getConfig = (key, config) =>
  call('config', 'one', { method: 'GET', params: { key }, config })

// 获取站点功能配置（Mellow_functions）
export const getSiteFunctions = (config) =>
  call('config', 'one', { method: 'GET', params: { key: 'Mellow_functions' }, config })

// 批量获取配置（后台系统配置页）
//
// 不要用 N 个 getConfig 并发拼一个 Promise.all：config/one 是同一个 path+method，
// 十几次并发会直接压满单接口 QPS 阈值（进而触发自动封禁），
// 而且缓存还没写入时全部穿透到数据库，每个请求都要几百毫秒。
export const getAllConfigs = (config) =>
  call('config', 'all', {
    method: 'GET',
    params: { limit: 200, page: 1, order: 'id asc' },
    config
  })

// 保存配置
export const saveConfig = (key, json, value = '') =>
  call('config', 'save', { method: 'POST', data: { key, json, value } })

/**
 * 保存系统配置项（自定义字段版）
 *
 * config 表结构是 key / value / json / text，不同配置项用的字段不同：
 * - SYSTEM_API_KEY、SYSTEM_API_KEY_LOCAL（开启密钥时是否放行本站来源）、ALLOW_REGISTER 只用 value（"0"/"1"）
 *   ALLOW_REGISTER 额外用 text 存「新用户默认权限组」ID 列表（如 "1,2"）
 * - SYSTEM_QPS、SYSTEM_QPS_BLOCK 用 value + json
 * 这里只提交调用方显式传入的字段，避免把不需要的字段写成空值。
 *
 * @param {{key: string, value?: string, json?: any, text?: string}} data
 */
export const saveSystemConfig = (data) =>
  call('config', 'save', { method: 'POST', data })
