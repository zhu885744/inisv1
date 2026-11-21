import { call, devService } from './request'

/**
 * 获取服务器 unix 时间戳（用于加密盐派生）
 * 接口路径为 /dev/info/time（不带 /api 前缀，属于 dev 类接口）
 * 优先请求后端时间接口，失败则回退本地时间（参考 Cardify-inis 实现）
 */
async function getUnix() {
  try {
    const { code, data } = await devService.get('/dev/info/time')
    if (code === 200 && data && typeof data.unix === 'number') {
      return data.unix
    }
  } catch (e) {
    // 忽略，使用本地时间
  }
  return Math.round(Date.now() / 1000)
}

/**
 * 生成加密所需的 AES 实例与请求头（参考 Cardify-inis 登录加密实现）
 * 返回 { AES, headers }
 */
async function buildEncryptedRequest(account, password) {
  // crypto-js 只有登录 / 注册的加密链路需要：改成按需加载，
  // 避免它（全量 AES/MD5/SHA 等，gzip 后几十 KB）被打进首屏入口 chunk
  const { default: cryptoUtil } = await import('@/utils/crypto')

  const unix = await getUnix()

  // 基于时间派生 key / iv（各 16 位）
  const iv = cryptoUtil.token
    ? cryptoUtil.token(`iv-${unix}`, 16, 'aes')
    : cryptoUtil.random(16)
  const key = cryptoUtil.token
    ? cryptoUtil.token(`key-${unix}`, 16, 'aes')
    : cryptoUtil.random(16)

  const safeIv = iv.length >= 16 ? iv.substring(0, 16) : iv.padEnd(16, '0')
  const safeKey = key.length >= 16 ? key.substring(0, 16) : key.padEnd(16, '0')

  const AES = cryptoUtil.createAES
    ? cryptoUtil.createAES(safeKey, safeIv)
    : new cryptoUtil.AES(safeKey, safeIv)

  const headers = {
    'X-Khronos': String(unix),
    'X-Gorgon': `${safeKey} ${safeIv}`,
    'X-Argus': AES.encrypt(
      JSON.stringify({ unix, account, password })
    )
  }

  return { AES, headers }
}

// 登录（加密传输）
export const login = async (account, password) => {
  const { AES, headers } = await buildEncryptedRequest(account, password)
  return call('comm', 'login', {
    method: 'POST',
    data: {
      account: AES.encrypt(account),
      password: AES.encrypt(password),
      source: 'default'
    },
    config: { headers, timeout: 10000 }
  })
}

/**
 * 验证码登录 - 发送验证码
 * 后端 /api/comm/sign-code：code 为空时只发码（返回 201），不会登录
 */
export const signCodeSend = (social) =>
  call('comm', 'sign-code', {
    method: 'POST',
    data: { social }
  })

/**
 * 验证码登录 - 校验并登录
 * 返回结构与密码登录一致（data: { user, token, valid_time }）
 */
export const signCode = (social, code) =>
  call('comm', 'sign-code', {
    method: 'POST',
    data: { social, code }
  })

// 注册（发送验证码）
export const registerSendCode = (social) =>
  call('comm', 'register', {
    method: 'POST',
    data: { social }
  })

/**
 * 注册来源（写入 users.source）
 * 用于区分「通过 Mellow 主题注册」的账号，后台用户列表可直接看到该标记。
 */
export const REGISTER_SOURCE = 'mellow'

// 注册（提交，加密传输）
export const register = async (social, code, password, account = '', nickname = '') => {
  // 注册接口使用 social 作为登录账号，password 需加密
  const { AES, headers } = await buildEncryptedRequest(social, password)
  return call('comm', 'register', {
    method: 'POST',
    data: {
      social,
      code,
      source: REGISTER_SOURCE,
      account: account ? AES.encrypt(account) : '',
      nickname: nickname ? AES.encrypt(nickname) : '',
      password: AES.encrypt(password)
    },
    config: { headers, timeout: 10000 }
  })
}

// 重置密码 - 发送验证码
export const resetPasswordSendCode = (social) =>
  call('comm', 'reset-password', {
    method: 'POST',
    data: { social }
  })

// 重置密码 - 提交
export const resetPassword = (social, code, password) =>
  call('comm', 'reset-password', {
    method: 'POST',
    data: { social, code, password }
  })

// 校验 Token
// skipAuthLogout: 让 401/412 原样返回，由调用方（user store 的 checkLoginState）统一处理本地登录态清理，
// 避免全局拦截直接 reject 导致误判为"保持登录"或误清有效登录态（参考 Cardify-inis 实现）
export const checkToken = (renew = false) =>
  call('comm', 'check-token', {
    method: 'POST',
    params: { renew },
    config: { skipAuthLogout: true }
  })

// 退出登录
export const logout = () =>
  call('comm', 'logout', { method: 'DELETE' })

/**
 * 滑块验证题（公开接口）
 * 正常流程中题目随 428 响应一起下发；这里用于「题目过期 / 校验失败次数超限作废」后换一题
 */
export const getSliderCaptcha = () =>
  call('comm', 'captcha', { method: 'GET', config: { silent: true, dedupe: false } })

// ============================== 后台管理会话 ==============================
//
// 后台登录态与前台登录态是两条独立的会话：
//   - 前台：cookie INIS_LOGIN_TOKEN（15 天，长期挂在浏览器上）
//   - 后台：cookie INIS_ADMIN_TOKEN（默认 2 小时，HttpOnly，脚本读不到）
// 后台会话必须由「有后台页面权限」的账号重新输入一次密码才能签发，
// 因此前台 token 泄露（XSS / 共享电脑 / 代理日志）并不能直接进后台。

/**
 * 签发后台管理会话（需要已登录 + 重新输入密码）
 * 成功后服务端写入 HttpOnly 的后台 cookie，前端不需要（也读不到）token 本体
 */
export const adminLogin = (password) =>
  call('comm', 'admin-login', {
    method: 'POST',
    data: { password },
    config: { silent: true, dedupe: false }
  })

/** 校验后台管理会话是否有效（失败返回 401 + auth:'admin'） */
export const adminCheck = () =>
  call('comm', 'admin-check', {
    method: 'POST',
    config: { silent: true, dedupe: false }
  })

/** 结束后台管理会话（只退后台，不影响前台登录态） */
export const adminLogout = () =>
  call('comm', 'admin-logout', { method: 'DELETE', config: { silent: true } })
