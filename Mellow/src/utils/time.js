import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

export const formatTime = (timestamp) => {
  if (!timestamp) return ''
  const t = typeof timestamp === 'string' ? parseInt(timestamp) * 1000 : timestamp * 1000
  return dayjs(t).format('YYYY-MM-DD HH:mm')
}

export const formatDate = (timestamp) => {
  if (!timestamp) return ''
  const t = typeof timestamp === 'string' ? parseInt(timestamp) * 1000 : timestamp * 1000
  return dayjs(t).format('YYYY-MM-DD')
}

/**
 * unix 秒 -> 完整时间（YYYY-MM-DD HH:mm:ss）
 *
 * 与 formatTime 的区别：多了秒。用于「同一天可能操作多次」的场景（文档历史记录等），
 * 只到分的话短时间内连续保存会显示成一模一样的时间。
 */
export const formatDateTime = (timestamp) => {
  if (!timestamp) return ''
  const t = typeof timestamp === 'string' ? parseInt(timestamp) * 1000 : timestamp * 1000
  return dayjs(t).format('YYYY-MM-DD HH:mm:ss')
}

export const fromNow = (timestamp) => {
  if (!timestamp) return ''
  const t = typeof timestamp === 'string' ? parseInt(timestamp) * 1000 : timestamp * 1000
  return dayjs(t).fromNow()
}

/**
 * 有效期（秒）-> 可读文本
 * 0 / 空值 / 负数表示永久有效（装扮、虚拟商品等「获得后可用多久」的展示统一走这里）
 */
export const formatDuration = (seconds) => {
  const s = Number(seconds) || 0
  if (s <= 0) return '永久'
  const days = s / 86400
  if (days >= 1) return `${Math.round(days)} 天`
  const hours = s / 3600
  if (hours >= 1) return `${Math.round(hours)} 小时`
  return `${Math.max(1, Math.round(s / 60))} 分钟`
}

/**
 * unix 秒 -> <input type="datetime-local"> 需要的本地时间字符串（YYYY-MM-DDTHH:mm）
 * 无有效时间戳时返回空串（输入框显示为空）
 */
export const toLocalInput = (timestamp) => {
  const ts = Number(timestamp)
  if (!ts) return ''
  const d = new Date(ts * 1000)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/**
 * <input type="datetime-local"> 的值 -> unix 秒
 * 空值返回 0（业务上 0 表示「未设置 / 不限制」）
 */
export const fromLocalInput = (value) => {
  if (!value) return 0
  const ts = Math.floor(new Date(value).getTime() / 1000)
  return Number.isFinite(ts) ? ts : 0
}