/**
 * 续期配置（renew）的读写辅助
 *
 * 结构与后端一致（见 app/model/renew-plan.go）：
 *   { enabled: true, options: [{ days: 30, price: 100, origin_price: 150, tag: '推荐' }] }
 *
 * 后台列表接口返回的是库里的原始 JSON 字符串，所以读写都要过一次 parseRenewConfig。
 */

/** 空配置（新建商品时的初值） */
export const emptyRenewConfig = () => ({ enabled: false, options: [] })

function normalizeOptions(options) {
  if (!Array.isArray(options)) return []
  return options
    .map((item) => ({
      days: Number(item?.days) || 0,
      price: Number(item?.price) || 0,
      origin_price: Number(item?.origin_price) || 0,
      tag: item?.tag || ''
    }))
    .sort((a, b) => a.days - b.days)
}

/** 任意来源（对象 / JSON 字符串 / null）→ 规范的续期配置 */
export function parseRenewConfig(raw) {
  if (!raw) return emptyRenewConfig()

  let data = raw
  if (typeof raw === 'string') {
    try {
      data = JSON.parse(raw)
    } catch {
      return emptyRenewConfig()
    }
  }

  return {
    enabled: !!data?.enabled,
    options: normalizeOptions(data?.options)
  }
}

/** 天数文案（30 → 30 天，180 → 6 个月，365 → 1 年），与后端 RenewDurationText 一致 */
export function renewDaysText(days) {
  const value = Number(days) || 0
  if (value <= 0) return ''
  if (value % 365 === 0) return `${value / 365} 年`
  if (value % 30 === 0 && value >= 60) return `${value / 30} 个月`
  return `${value} 天`
}
