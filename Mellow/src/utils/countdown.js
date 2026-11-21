/**
 * 倒计时工具（积分商城兑换时间窗口等）
 *
 * 不依赖构建别名，便于单独测试。
 */

const pad = (n) => String(n).padStart(2, '0')

/** 秒 -> 拆分（天/时/分/秒） */
export function splitDuration(seconds) {
  const total = Math.max(0, Math.floor(Number(seconds) || 0))
  return {
    total,
    days: Math.floor(total / 86400),
    hours: Math.floor((total % 86400) / 3600),
    minutes: Math.floor((total % 3600) / 60),
    seconds: total % 60
  }
}

/**
 * 倒计时文案
 *  - ≥ 1 天：`3天 05:12:30`
 *  - < 1 天：`05:12:30`（始终保留到秒，保证精确）
 *  - 已结束：`00:00:00`
 */
export function formatCountdown(seconds) {
  const { days, hours, minutes, seconds: secs } = splitDuration(seconds)
  const clock = `${pad(hours)}:${pad(minutes)}:${pad(secs)}`
  return days > 0 ? `${days}天 ${clock}` : clock
}

/**
 * 距离目标时间戳（秒）还剩几秒
 * @param {number|string} targetTs 目标时间戳（秒）
 * @param {number} nowMs 当前时间（毫秒），默认 Date.now()
 */
export function remainSeconds(targetTs, nowMs = Date.now()) {
  const target = Number(targetTs) || 0
  if (!target) return 0
  return Math.max(0, Math.floor(target - nowMs / 1000))
}
