import { call } from './request'

/**
 * 第三方登录（QQ / GitHub / Gitee）接口封装
 *
 * 后端见 app/api/controller/OAuth.go，同一个入口按条件自动分流：
 *   - 已绑定 + 未登录 → 发 token 登录
 *   - 未绑定 + 已登录 → 绑定到当前账号
 *   - 未绑定 + 未登录 → 返回 need_choice + ticket，由回调页让用户二选一：
 *       创建新账号（oauthRegisterByTicket） / 绑定已有账号（oauthBindAccount）
 */

// 第三方登录配置（公开）：前端据此拼授权地址
// cache=false：避免命中接口结果缓存，后台改完配置前端立刻拿到新值
export const getOauthConfig = (config) =>
  call('oauth', 'config', { method: 'GET', params: { cache: false }, config: { silent: true, ...config } })

// 用授权码登录 / 注册 / 绑定（服务端按是否已登录自动分流）
export const oauthSignIn = (platform, code, state = '') =>
  call('oauth', platform, { method: 'POST', data: { code, state }, config: { dedupe: false } })

/**
 * 未绑定时：用票据创建新账号并登录（回调页「创建新账号」）
 * 受两个开关约束：站点注册总开关 + 后台「允许第三方账号创建新账号」。
 * 建号必须带邮箱与密码：邮箱要验证码验过（发码走 comm/register），密码按注册规则校验强度。
 */
export const oauthRegisterByTicket = ({ key, email, code, password }) =>
  call('oauth', 'register', {
    method: 'POST',
    data: { key, email, code, password },
    config: { dedupe: false }
  })

/**
 * 未绑定时：用「已有账号 + 密码」绑定该第三方账号并登录（回调页「绑定已有账号」）
 * 账号可为 账号 / 邮箱 / 手机号；密码校验与风控口径同密码登录。
 */
export const oauthBindAccount = ({ key, account, password }) =>
  call('oauth', 'bind-account', { method: 'POST', data: { key, account, password }, config: { dedupe: false } })

/**
 * 绑定第三方账号（需登录）
 * - { key }：先登录再绑定（ticket 来自未登录时的第三方登录响应）
 * - { platform, code }：已登录状态下重新发起授权
 */
export const oauthBind = (data) =>
  call('oauth', 'bind', { method: 'POST', data, config: { dedupe: false } })

// 解绑第三方账号（需登录）
export const oauthUnbind = (platform) =>
  call('oauth', 'unbind', { method: 'POST', data: { platform } })

// 我绑定的第三方账号 + 还能绑哪些（需登录）
// 管理员可传 { uid } 查询指定用户的绑定（后台「用户详情」弹窗用）
export const getMyOauth = (params = {}, config = {}) =>
  call('oauth', 'mine', { method: 'GET', params, config: { silent: true, ...config } })
