import { call } from './request'

/**
 * toml 配置读写（后端 app/api/controller/toml.go，路由前缀 /api/toml/:method）
 *
 * 约定：
 * 1. 读取：GET /api/toml/{name}，可选 ?name=xxx 只取某个分组；
 *    **storage 不带 name 时后端不脱敏**，因此存储相关一律按分组读取（见 getStorageGroup）。
 * 2. 保存：PUT /api/toml/{method}，各分项方法只更新自己那段，未提交的分组保持原值。
 * 3. 测试：POST /api/toml/test-xxx，用于连通性验证（Redis / 腾讯云 COS / 短信）。
 * 4. 脱敏：后端对**非管理员**会把密钥类字段（password / access_key_secret / secret_key /
 *    access_key / secret_id / key）脱敏成 `****` 占位串；管理员（后台）返回明文。
 *    若拿到的仍是占位串，原样回传即可表示不修改，后端 restoreSecretParams 会还原真实值。
 */

// 读取配置（name 为空时取整份）
export const getToml = (method, params = {}) =>
  call('toml', method, { method: 'GET', params })

// 保存配置
export const saveToml = (method, data = {}) =>
  call('toml', method, { method: 'PUT', data })

// 连通性测试
export const testToml = (method, data = {}) =>
  call('toml', method, { method: 'POST', data })

// ===== 语义化封装 =====

/** 日志配置（只读：后端未提供保存方法） */
export const getTomlLog = () => getToml('log')

/** 通知配置（消息保留天数） */
export const getTomlNotification = () => getToml('notification')

/**
 * 全站敏感词库（独立 config/sensitive.toml）
 *
 * 返回 { enabled, default_action, version, rules, whitelist, content }：
 * - rules / whitelist 是解析后的结构化结果，可视化表单直接用；
 * - content 是文件原文，源码模式编辑用（表单保存会按模板重写，注释不保留）。
 */
export const getTomlSensitive = () => getToml('sensitive')

/**
 * 保存敏感词库（同一个接口，两种载荷）
 *
 * - { content: string }：TOML 原文，保留手写注释（「TOML 源码」模式）；
 * - { enabled, default_action, rules, whitelist }：结构化数据，服务端按模板生成规范 TOML
 *   （「可视化配置」模式，会重写文件、注释不保留）。
 *
 * 两条路径共用同一套校验 / 备份 / 版本自增 / 热更新，所以只占一个权限点。
 */
export const saveTomlSensitive = (data) => saveToml('sensitive', data)

/** 回滚到上一次保存前的词库 */
export const rollbackTomlSensitive = () => testToml('sensitive-rollback')

/** 用当前词库试跑一条文本：{ scene, text } → { decision } */
export const testTomlSensitive = (data) => testToml('sensitive-test', data)

/**
 * 命中统计：{ days, type, items: [{ scene, action, total }],
 * dimensions: [{ type, label, total }], options: [{ type, label }] }
 *
 * @param {{ days?: number, type?: number }} params
 *   days 统计天数（默认 7），type 机审维度（百度的 type，0 = 全部）
 */
export const getTomlSensitiveStats = (params = {}) => getToml('sensitive-stats', params)

/** 短信配置（不带 name 时返回 drive + 各服务商，且已脱敏） */
export const getTomlSms = () => getToml('sms')

/**
 * 发件队列（config/sms.toml 的 [email] 段：分批 + 重试参数）
 *
 * - 读取：getTomlSms() 的 data.email 里就带着这些字段（与邮件服务配置同表）；
 * - 保存：saveToml('sms-email-queue', { batch_size, batch_interval, retry_delay,
 *   max_attempts, send_timeout, verify_wait, queue_size })，只提交要改的字段，
 *   未提交的保持原值，越界（见后端 facade.MailQueueLimits）返回 400。
 */

/** 缓存配置（返回 open / default / redis / file / ram） */
export const getTomlCache = () => getToml('cache')

/** 登录会话配置（返回 { session: { expire } }；JWT 已移除，老配置会由后端映射） */
export const getTomlSession = () => getToml('crypt')

/**
 * 存储配置：必须按分组读取，避免拿到明文密钥
 * @param {string} name local / cos / attachment
 */
export const getStorageGroup = (name) => getToml('storage', { name })

/**
 * 云端内容审核（独立 config/audit.toml，AK/SK 存这里）
 *
 * 返回 { enabled, active, provider, baidu: { api_key, secret_key, text_strategy_id,
 * image_strategy_id, timeout, workers, queue_size } }：
 * - enabled 是配置里的开关，active 才是「凭据齐全、真正会生效」；
 * - 凭据对管理员原样返回，表单直接回填即可。
 */
export const getTomlAudit = () => getToml('audit')

/** 保存云端审核配置（服务端按模板重写 audit.toml 并热更新） */
export const saveTomlAudit = (data) => saveToml('audit', data)

/** 试跑一条文本：{ text } → { conclusion, words }（真实调用，会计费） */
export const testTomlAudit = (data) => testToml('audit-test', data)
