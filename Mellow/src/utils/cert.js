/**
 * 认证标识配色
 *
 * 唯一的取色入口：昵称旁的盾牌（VerifiedBadge）与简介下的认证行（Author）都走这里，
 * 避免两处各写一套规则，导致同一个认证在页面上出现两种颜色。
 *
 * 取值优先级：
 *   1. 类型配置里的 color（后台「认证类型」可自定义，如 #e8a020）；
 *   2. 内置类型的兜底色：职业=金 / 兴趣=蓝 / 机构=绿；
 *   3. 其它（自定义类型没配色）→ 品牌蓝。
 */

// 与后端默认播种的颜色保持一致；只在前端兜底，改了后端配置就以配置为准
const FALLBACK_COLORS = {
  1: '#e8a020',
  2: '#2f8fe8',
  3: '#2fa96b'
}

const DEFAULT_COLOR = '#2f8fe8'

// 3 / 6 / 8 位十六进制，与后端 NormalizeCertColor 同一口径
const HEX_COLOR = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$/

/** 是否为合法十六进制颜色（颜色会直接进 style，不能让任意字符串通过） */
export function isHexColor(value) {
  return HEX_COLOR.test(String(value || '').trim())
}

/**
 * 认证标识颜色
 * @param {Object|number} cert CertSummary 对象，或直接传 cert_type
 * @returns {string} 形如 #e8a020
 */
export function certColor(cert) {
  const item = typeof cert === 'object' && cert !== null ? cert : { cert_type: cert }

  const color = String(item.color || '').trim()
  if (isHexColor(color)) return color

  return FALLBACK_COLORS[Number(item.cert_type)] || DEFAULT_COLOR
}
